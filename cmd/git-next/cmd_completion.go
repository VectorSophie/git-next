package main

import (
	"fmt"
	"os"

	"github.com/VectorSophie/git-next/internal/completion"
)

func runCompletion(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: git-next completion <bash|zsh|fish>")
		os.Exit(1)
	}

	switch args[0] {
	case "bash":
		fmt.Print(completion.Bash())
	case "zsh":
		fmt.Print(completion.Zsh())
	case "fish":
		fmt.Print(completion.Fish())
	default:
		fmt.Fprintf(os.Stderr, "unknown shell %q; use: bash, zsh, or fish\n", args[0])
		os.Exit(1)
	}
}
