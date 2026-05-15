package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/internal/engine"
	"github.com/VectorSophie/git-next/internal/repo"
	"github.com/VectorSophie/git-next/pkg/model"
)

// severityIncludes returns true when the rule's severity meets or exceeds the threshold.
var failOnSeverities = map[string]map[string]bool{
	"critical": {"critical": true},
	"high":     {"critical": true, "high": true},
	"medium":   {"critical": true, "high": true, "medium": true},
	"low":      {"critical": true, "high": true, "medium": true, "low": true},
	"all":      {"critical": true, "high": true, "medium": true, "low": true, "info": true},
}

// githubLevel maps severity to GitHub Actions annotation level.
var githubLevel = map[string]string{
	"critical": "error",
	"high":     "error",
	"medium":   "warning",
	"low":      "notice",
	"info":     "notice",
}

func runCI(args []string) {
	fs := flag.NewFlagSet("ci", flag.ExitOnError)
	failOn := fs.String("fail-on", "all", "Minimum severity that causes exit 1 (critical|high|medium|low|all)")
	format := fs.String("format", "human", "Output format (human|github)")
	networkEnabled := fs.Bool("network", false, "Enable network calls (unpushed-tags check via git ls-remote)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `git-next ci [--fail-on <tier>] [--format <format>] [--network]

Run git-next as a CI gate. Exits 1 when rules matching --fail-on fire.

Examples:
  git-next ci --fail-on high
  git-next ci --fail-on critical --format github
  git-next ci --fail-on high --network`)
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	thresholds, ok := failOnSeverities[*failOn]
	if !ok {
		fmt.Fprintf(os.Stderr, "invalid --fail-on value %q; use: critical, high, medium, low, all\n", *failOn)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.Defaults()
	}
	if *networkEnabled {
		cfg.Network = true
	}

	state, err := repo.CollectState(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	advice := engine.Evaluate(state, cfg)

	active := []model.Advice{}
	for _, a := range advice {
		if !a.Suppressed {
			active = append(active, a)
		}
	}

	switch *format {
	case "github":
		printCIGitHub(active)
	default:
		printCIHuman(active, *failOn)
	}

	// Exit 1 if any active rule meets the threshold.
	for _, a := range active {
		if thresholds[a.Severity] {
			os.Exit(1)
		}
	}
}

func printCIGitHub(active []model.Advice) {
	for _, a := range active {
		level := githubLevel[a.Severity]
		if level == "" {
			level = "notice"
		}
		fmt.Printf("::%s::git-next %s: %s\n", level, a.RuleID, a.Description)
	}
}

func printCIHuman(active []model.Advice, failOn string) {
	if len(active) == 0 {
		fmt.Println("✓ Repository is clean.")
		return
	}
	fmt.Printf("git-next ci (fail-on: %s)\n", failOn)
	fmt.Println("───────────────────────────────")
	for _, a := range active {
		fmt.Printf("[%s] [%s] %s\n", a.Severity, a.RuleID, a.Description)
	}
}
