package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/VectorSophie/git-next/internal/mcp"
)

func runMCP(args []string) {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `git-next mcp

Start the git-next MCP server over stdio. Expose git safety tools to AI agents.

Add to .mcp.json in your project root:
  {
    "mcpServers": {
      "git-next": { "command": "git-next", "args": ["mcp"] }
    }
  }

Available tools: git_status, git_guard, git_run, git_explain, git_diff, git_log, git_remote_status`)
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	repoPath, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := mcp.Serve(repoPath); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}
