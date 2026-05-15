package main

import (
	"fmt"
	"os"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/internal/rules"
)

func runRules(args []string) {
	if len(args) == 0 || args[0] != "list" {
		fmt.Fprintln(os.Stderr, "usage: git-next rules list")
		os.Exit(1)
	}

	cfg := config.Defaults()
	all := rules.AllRules(cfg)

	fmt.Printf("%-6s  %-4s  %-8s  %-4s  %s\n", "ID", "PRI", "SEVERITY", "DEST", "DESCRIPTION")
	fmt.Println("──────────────────────────────────────────────────────────────────────────────")
	for _, r := range all {
		dest := "no"
		if r.Destructive {
			dest = "yes"
		}
		desc := r.Description
		if len(desc) > 55 {
			desc = desc[:52] + "..."
		}
		fmt.Printf("%-6s  %-4d  %-8s  %-4s  %s\n", r.ID, r.Priority, r.Severity, dest, desc)
	}
}
