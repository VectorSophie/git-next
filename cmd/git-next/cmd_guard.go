package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/internal/policy"
	"github.com/VectorSophie/git-next/internal/repo"
	"github.com/VectorSophie/git-next/pkg/model"
)

func runGuard(args []string) {
	fs := flag.NewFlagSet("guard", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `git-next guard [--json] -- <git command>

Check whether a git command is safe to run in the current repository state.

Examples:
  git-next guard -- git push --force
  git-next guard -- git reset --hard HEAD~3
  git-next guard --json -- git rebase main`)
	}

	// Split on "--" to separate flags from the proposed command.
	var proposed []string
	flagArgs := args
	for i, a := range args {
		if a == "--" {
			proposed = args[i+1:]
			flagArgs = args[:i]
			break
		}
	}
	if err := fs.Parse(flagArgs); err != nil {
		os.Exit(1)
	}

	if len(proposed) == 0 {
		fmt.Fprintln(os.Stderr, "usage: git-next guard -- <git command>")
		os.Exit(1)
	}

	proposedStr := strings.Join(proposed, " ")

	cfg, err := config.Load()
	if err != nil {
		cfg = config.Defaults()
	}

	state, err := repo.CollectState(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	blocked := policy.Evaluate(proposedStr, state)

	if *jsonOut {
		printGuardJSON(proposedStr, blocked)
		return
	}
	printGuardHuman(proposedStr, blocked)
}

func printGuardHuman(proposed string, blocked *policy.CommandPolicy) {
	if blocked == nil {
		fmt.Printf("ALLOWED: %s\n", proposed)
		return
	}
	fmt.Printf("BLOCKED: %s\n", proposed)
	fmt.Printf("Reason:  %s\n", blocked.Reason)
	if blocked.Alternative != "" {
		fmt.Printf("Safer:   %s\n", blocked.Alternative)
	}
	os.Exit(1)
}

func printGuardJSON(proposed string, blocked *policy.CommandPolicy) {
	out := model.GuardOutput{
		Command:   proposed,
		Allowed:   blocked == nil,
		RiskLevel: "safe",
	}
	if blocked != nil {
		out.RiskLevel = blocked.RiskLevel
		out.Reason = blocked.Reason
		out.Alternative = blocked.Alternative
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
	if blocked != nil {
		os.Exit(1)
	}
}
