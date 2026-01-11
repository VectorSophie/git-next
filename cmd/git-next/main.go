package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yourusername/git-next/internal/action"
	"github.com/yourusername/git-next/internal/engine"
	"github.com/yourusername/git-next/internal/output"
	"github.com/yourusername/git-next/internal/repo"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		showVersion    bool
		showAll        bool
		formatJSON     bool
		formatCompact  bool
		showDebug      bool
		interactiveAction bool
	)

	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information (shorthand)")
	flag.BoolVar(&showAll, "all", false, "Show suppressed advice")
	flag.BoolVar(&showAll, "a", false, "Show suppressed advice (shorthand)")
	flag.BoolVar(&formatJSON, "json", false, "Output in JSON format")
	flag.BoolVar(&formatCompact, "compact", false, "Output compact one-line summary")
	flag.BoolVar(&showDebug, "debug", false, "Show debug information (repo state)")
	flag.BoolVar(&interactiveAction, "action", false, "Interactive mode to execute suggested actions")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `git-next - Git advice that doesn't lie

Usage:
  git-next [options]

Options:
  -v, --version     Show version information
  -a, --all         Show suppressed advice
  --json            Output in JSON format
  --compact         Output compact one-line summary
  --debug           Show debug information (repo state)
  --action          Interactive mode to execute suggested actions

Examples:
  git-next                    # Show current advice
  git-next --all              # Show all advice including suppressed
  git-next --json             # Output as JSON
  git-next --compact          # Show compact summary
  git-next --action           # Interactive mode to execute actions

The tool never lies. It analyzes your repository state and suggests
the least harmful move based on who has the history.

`)
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("git-next version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("built: %s\n", date)
		os.Exit(0)
	}

	// Collect repository state
	state, err := repo.CollectState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Show debug info if requested
	if showDebug {
		fmt.Printf("Repository State:\n")
		fmt.Printf("  Dirty: %v\n", state.Dirty)
		fmt.Printf("  StagedFiles: %d\n", state.StagedFiles)
		fmt.Printf("  ModifiedFiles: %d\n", state.ModifiedFiles)
		fmt.Printf("  UntrackedFiles: %d\n", state.UntrackedFiles)
		fmt.Printf("  Ahead: %d\n", state.Ahead)
		fmt.Printf("  Behind: %d\n", state.Behind)
		fmt.Printf("  HasStash: %v\n", state.HasStash)
		fmt.Printf("  OnDetachedHead: %v\n", state.OnDetachedHead)
		fmt.Printf("  LastCommitPushed: %v\n", state.LastCommitPushed)
		fmt.Printf("  CommitCountSincePush: %d\n", state.CommitCountSincePush)
		fmt.Printf("  OnProtectedBranch: %v\n", state.OnProtectedBranch)
		fmt.Printf("  HasMergeCommits: %v\n", state.HasMergeCommits)
		fmt.Printf("  MergeInProgress: %v\n", state.MergeInProgress)
		fmt.Printf("  RebaseInProgress: %v\n", state.RebaseInProgress)
		fmt.Printf("  CherryPickInProgress: %v\n", state.CherryPickInProgress)
		fmt.Printf("  NoUpstream: %v\n", state.NoUpstream)
		fmt.Printf("  MergedBranches: %v\n", state.MergedBranches)
		fmt.Printf("  GoneBranches: %v\n\n", state.GoneBranches)
	}

	// Evaluate rules
	advice := engine.Evaluate(state)

	// Interactive action mode
	if interactiveAction {
		if err := action.Execute(advice); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Format output
	var outputStr string
	if formatJSON {
		var err error
		outputStr, err = output.FormatJSON(advice)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			os.Exit(1)
		}
	} else if formatCompact {
		outputStr = output.FormatCompact(advice)
	} else {
		outputStr = output.FormatHuman(advice, showAll)
	}

	fmt.Print(outputStr)

	// Exit with status 1 if there are active suggestions
	activeCount := 0
	for _, a := range advice {
		if !a.Suppressed {
			activeCount++
		}
	}

	if activeCount > 0 {
		os.Exit(1)
	}
}
