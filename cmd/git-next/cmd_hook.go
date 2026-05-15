package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/VectorSophie/git-next/internal/hook"
)

func runHook(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: git-next hook <install|uninstall|list> [--hook <name>]")
		os.Exit(1)
	}

	subcommand := args[0]
	rest := args[1:]

	fs := flag.NewFlagSet("hook "+subcommand, flag.ExitOnError)
	hookName := fs.String("hook", "pre-push", "Hook name (pre-commit, pre-push, commit-msg, …)")
	if err := fs.Parse(rest); err != nil {
		os.Exit(1)
	}

	var err error
	switch subcommand {
	case "install":
		err = hook.Install(*hookName)
	case "uninstall":
		err = hook.Uninstall(*hookName)
	case "list":
		err = hook.List()
	default:
		fmt.Fprintf(os.Stderr, "unknown hook subcommand %q\n", subcommand)
		fmt.Fprintln(os.Stderr, "usage: git-next hook <install|uninstall|list>")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
