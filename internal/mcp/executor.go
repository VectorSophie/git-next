package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/VectorSophie/git-next/internal/policy"
	"github.com/VectorSophie/git-next/pkg/model"
)

// RunResult is the response shape for git_run.
type RunResult struct {
	Stdout       string `json:"stdout,omitempty"`
	Stderr       string `json:"stderr,omitempty"`
	ExitCode     int    `json:"exit_code,omitempty"`
	Blocked      bool   `json:"blocked,omitempty"`
	BlockReason  string `json:"block_reason,omitempty"`
	Alternative  string `json:"alternative,omitempty"`
	NeedsConfirm bool   `json:"requires_confirmation,omitempty"`
	RiskLevel    string `json:"risk_level,omitempty"`
	ConfirmMsg   string `json:"confirm_message,omitempty"`
}

// Run classifies and executes a git command.
// repoPath is passed as cmd.Dir; empty string uses the process cwd.
func Run(ctx context.Context, command, repoPath string, confirmed bool, state model.RepoState) RunResult {
	switch Classify(command, state) {
	case TierBlocked:
		p := policy.Evaluate(command, state)
		res := RunResult{Blocked: true, RiskLevel: "danger"}
		if p != nil {
			res.BlockReason = p.Reason
			res.Alternative = p.Alternative
		} else {
			res.BlockReason = "not a git command or unconditionally blocked"
		}
		return res

	case TierGuarded:
		if !confirmed {
			return RunResult{
				NeedsConfirm: true,
				RiskLevel:    "caution",
				ConfirmMsg:   fmt.Sprintf("%q requires confirmation; call git_run again with confirmed=true to proceed", command),
			}
		}
		return execute(ctx, command, repoPath)

	default: // TierSafe
		return execute(ctx, command, repoPath)
	}
}

func execute(ctx context.Context, command, repoPath string) RunResult {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return RunResult{Blocked: true, BlockReason: "empty command"}
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	if repoPath != "" {
		cmd.Dir = repoPath
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := RunResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = 1
		}
	}
	return res
}
