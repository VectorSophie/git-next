package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/VectorSophie/git-next/internal/explain"
)

func runExplain(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: git-next explain <rule-id>  (e.g. git-next explain R037)")
		os.Exit(1)
	}

	id := strings.ToUpper(strings.TrimSpace(args[0]))
	ex, ok := explain.Lookup(id)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown rule %q\nRun 'git-next rules list' to see all rule IDs.\n", id)
		os.Exit(1)
	}

	fmt.Printf("%s: %s\n\n", ex.ID, ex.Title)
	fmt.Printf("Why it matters:\n%s\n\n", indent(ex.Why, "  "))
	if ex.Alternative != "" {
		fmt.Printf("Safer alternative:\n%s\n\n", indent(ex.Alternative, "  "))
	}
}

func indent(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
