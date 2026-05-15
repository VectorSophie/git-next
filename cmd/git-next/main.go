package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/VectorSophie/git-next/internal/action"
	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/internal/engine"
	"github.com/VectorSophie/git-next/internal/output"
	"github.com/VectorSophie/git-next/internal/repo"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Dispatch subcommands before flag parsing so they can manage their own flags.
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		switch os.Args[1] {
		case "guard":
			runGuard(os.Args[2:])
			return
		case "hook":
			runHook(os.Args[2:])
			return
		case "ci":
			runCI(os.Args[2:])
			return
		case "explain":
			runExplain(os.Args[2:])
			return
		case "rules":
			runRules(os.Args[2:])
			return
		case "completion":
			runCompletion(os.Args[2:])
			return
		}
	}

	var (
		showVersion       bool
		showAll           bool
		formatJSON        bool
		formatCompact     bool
		agentMode         bool
		showDebug         bool
		showExplain       bool
		interactiveAction bool
		configPath        string
	)

	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information (shorthand)")
	flag.BoolVar(&showAll, "all", false, "Show suppressed advice")
	flag.BoolVar(&showAll, "a", false, "Show suppressed advice (shorthand)")
	flag.BoolVar(&formatJSON, "json", false, "Output in JSON format")
	flag.BoolVar(&formatCompact, "compact", false, "Output compact one-line summary")
	flag.BoolVar(&agentMode, "agent", false, "Output agent-compatible JSON for coding assistants and CI bots")
	flag.BoolVar(&showDebug, "debug", false, "Show debug information (repo state)")
	flag.BoolVar(&showExplain, "explain", false, "Show explanation for each active rule")
	flag.BoolVar(&interactiveAction, "action", false, "Interactive mode to execute suggested actions")
	flag.StringVar(&configPath, "config", "", "Path to config file (default: .git-next.yaml or ~/.config/git-next/config.yaml)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `git-next - Git advice that doesn't lie

Usage:
  git-next [options]
  git-next <subcommand> [args]

Options:
  -v, --version     Show version information
  -a, --all         Show suppressed advice
  --json            Output in JSON format
  --compact         Output compact one-line summary
  --agent           Machine-readable JSON for coding agents and CI bots
  --explain         Show explanation for each active rule
  --debug           Show debug information (repo state)
  --action          Interactive mode to execute suggested actions

Subcommands:
  guard -- <cmd>    Check whether a git command is safe to run
  hook              Install/uninstall/list git hooks (hook install --hook pre-push)
  ci                Run as a CI gate (--fail-on critical|high|medium|low)
  explain <rule>    Show full explanation for a rule (e.g. explain R037)
  rules list        List all rules with severity and metadata
  completion        Print shell completion script (bash|zsh|fish)

Examples:
  git-next                         # Show current advice
  git-next --agent                 # Machine-readable output for AI agents
  git-next guard -- git push -f    # Check whether a command is safe
  git-next ci --fail-on high       # CI gate, fail on high+ severity
  git-next hook install            # Install pre-push hook
  git-next explain R037            # Explain a specific rule

`)
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("git-next version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("built: %s\n", date)
		os.Exit(0)
	}

	// Load configuration
	var cfg *config.Config
	var err error
	if configPath != "" {
		cfg, err = config.LoadFromPath(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
	} else {
		cfg, err = config.Load()
		if err != nil {
			// Non-fatal: use defaults
			cfg = config.Defaults()
		}
	}

	// Collect repository state
	state, err := repo.CollectState(cfg)
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
	advice := engine.Evaluate(state, cfg)

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
	if agentMode {
		var err error
		outputStr, err = output.FormatAgent(advice)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting agent output: %v\n", err)
			os.Exit(1)
		}
	} else if formatJSON {
		var err error
		outputStr, err = output.FormatJSON(advice)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			os.Exit(1)
		}
	} else if formatCompact {
		outputStr = output.FormatCompact(advice)
	} else {
		outputStr = output.FormatHuman(advice, showAll, cfg, showExplain)
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
