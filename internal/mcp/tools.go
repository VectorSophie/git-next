package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/internal/engine"
	"github.com/VectorSophie/git-next/internal/explain"
	"github.com/VectorSophie/git-next/internal/policy"
	"github.com/VectorSophie/git-next/internal/repo"
	"github.com/VectorSophie/git-next/pkg/model"
)

// Tool is a single MCP tool entry.
type Tool struct {
	Name        string
	Description string
	InputSchema string
	Handler     func(ctx context.Context, params map[string]any, repoPath string) (any, error)
}

// Registry is the ordered list of all MCP tools.
var Registry = []Tool{
	{
		Name:        "git_status",
		Description: "Inspect the current git repository state and get prioritized advice. Call this first to understand what needs attention before running any git commands.",
		InputSchema: `{"type":"object","properties":{"repo_path":{"type":"string","description":"Path to the git repository. Defaults to the server launch directory."}}}`,
		Handler:     HandleGitStatus,
	},
	{
		Name:        "git_guard",
		Description: "Check whether a git command is safe to run. Returns allowed/blocked with reason and a safer alternative. Never executes the command.",
		InputSchema: `{"type":"object","required":["command"],"properties":{"command":{"type":"string","description":"The full git command to evaluate, e.g. 'git push --force'"},"repo_path":{"type":"string"}}}`,
		Handler:     HandleGitGuard,
	},
	{
		Name:        "git_run",
		Description: "Run a git command with safety classification. Safe commands execute immediately. Guarded commands (push, merge, rebase, commit --amend, stash pop) require a second call with confirmed=true. Blocked commands are never executed even with confirmed=true. Note: arguments are split on whitespace; avoid shell quoting for arguments with spaces.",
		InputSchema: `{"type":"object","required":["command"],"properties":{"command":{"type":"string","description":"The git command to run, e.g. 'git add .' or 'git push'"},"confirmed":{"type":"boolean","description":"Set to true to confirm a guarded command after reviewing the risk","default":false},"repo_path":{"type":"string"}}}`,
		Handler:     HandleGitRun,
	},
	{
		Name:        "git_explain",
		Description: "Get a detailed explanation of a git-next rule ID (e.g. R037). Returns the title, why the rule exists, and what to do instead.",
		InputSchema: `{"type":"object","required":["rule_id"],"properties":{"rule_id":{"type":"string","description":"The rule ID to explain, e.g. 'R037'"}}}`,
		Handler:     HandleGitExplain,
	},
	{
		Name:        "git_diff",
		Description: "Get the git diff of the working tree or staging area.",
		InputSchema: `{"type":"object","properties":{"staged":{"type":"boolean","description":"Show staged (index) diff instead of unstaged","default":false},"file":{"type":"string","description":"Limit diff to a specific file path"},"repo_path":{"type":"string"}}}`,
		Handler:     HandleGitDiff,
	},
	{
		Name:        "git_log",
		Description: "Get recent commit history.",
		InputSchema: `{"type":"object","properties":{"n":{"type":"integer","description":"Number of commits to return","default":10},"repo_path":{"type":"string"}}}`,
		Handler:     HandleGitLog,
	},
	{
		Name:        "git_remote_status",
		Description: "Check remote branch status (ahead/behind). Makes a network call via git fetch --dry-run. Use explicitly when you need fresh remote data, not on every status check.",
		InputSchema: `{"type":"object","properties":{"repo_path":{"type":"string"}}}`,
		Handler:     HandleGitRemoteStatus,
	},
}

// chdirMu guards os.Chdir calls in runInDir.
var chdirMu sync.Mutex

// runInDir temporarily changes the working directory to dir, calls fn, then restores.
// If dir is empty, fn is called in the current directory.
func runInDir(dir string, fn func() (model.RepoState, error)) (model.RepoState, error) {
	if dir == "" {
		return fn()
	}
	chdirMu.Lock()
	defer chdirMu.Unlock()
	original, err := os.Getwd()
	if err != nil {
		return model.RepoState{}, err
	}
	if err := os.Chdir(dir); err != nil {
		return model.RepoState{}, fmt.Errorf("invalid repo_path %q: %w", dir, err)
	}
	defer os.Chdir(original) //nolint:errcheck
	return fn()
}

func loadCfg() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		return config.Defaults()
	}
	return cfg
}

func overallRisk(advice []model.Advice) string {
	for _, sev := range []string{"critical", "high", "medium", "low", "info"} {
		for _, a := range advice {
			if !a.Suppressed && a.Severity == sev {
				return sev
			}
		}
	}
	return "safe"
}

// HandleGitStatus collects repo state, evaluates rules, returns advice list.
func HandleGitStatus(ctx context.Context, params map[string]any, repoPath string) (any, error) {
	cfg := loadCfg()
	state, err := runInDir(repoPath, func() (model.RepoState, error) {
		return repo.CollectState(cfg)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to collect repo state: %w", err)
	}
	advice := engine.Evaluate(state, cfg)

	active := make([]map[string]any, 0)
	for _, a := range advice {
		if !a.Suppressed {
			active = append(active, map[string]any{
				"rule_id":     a.RuleID,
				"severity":    a.Severity,
				"description": a.Description,
				"command":     a.Command,
				"destructive": a.Destructive,
			})
		}
	}

	risk := overallRisk(advice)
	summary := "clean"
	if len(active) > 0 {
		summary = fmt.Sprintf("%d active issue(s), highest severity: %s", len(active), risk)
	}
	return map[string]any{
		"advice":     active,
		"risk_level": risk,
		"summary":    summary,
	}, nil
}

// HandleGitGuard evaluates a proposed command against policy. Never executes.
func HandleGitGuard(ctx context.Context, params map[string]any, repoPath string) (any, error) {
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return nil, fmt.Errorf("command parameter is required")
	}
	cfg := loadCfg()
	state, err := runInDir(repoPath, func() (model.RepoState, error) {
		return repo.CollectState(cfg)
	})
	if err != nil {
		return nil, err
	}
	blocked := policy.Evaluate(command, state)
	if blocked == nil {
		return map[string]any{"allowed": true, "risk_level": "safe"}, nil
	}
	return map[string]any{
		"allowed":     false,
		"risk_level":  blocked.RiskLevel,
		"reason":      blocked.Reason,
		"alternative": blocked.Alternative,
	}, nil
}

// HandleGitRun classifies and runs a git command with the SAFE/GUARDED/BLOCKED model.
func HandleGitRun(ctx context.Context, params map[string]any, repoPath string) (any, error) {
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return nil, fmt.Errorf("command parameter is required")
	}
	confirmed, _ := params["confirmed"].(bool)
	cfg := loadCfg()
	state, err := runInDir(repoPath, func() (model.RepoState, error) {
		return repo.CollectState(cfg)
	})
	if err != nil {
		return nil, err
	}
	return Run(ctx, command, repoPath, confirmed, state), nil
}

// HandleGitExplain returns a full explanation for a rule ID.
func HandleGitExplain(ctx context.Context, params map[string]any, repoPath string) (any, error) {
	ruleID, ok := params["rule_id"].(string)
	if !ok || ruleID == "" {
		return nil, fmt.Errorf("rule_id parameter is required")
	}
	ex, found := explain.Lookup(ruleID)
	if !found {
		return nil, fmt.Errorf("unknown rule ID: %s", ruleID)
	}
	return map[string]any{
		"id":          ex.ID,
		"title":       ex.Title,
		"why":         ex.Why,
		"alternative": ex.Alternative,
	}, nil
}

// HandleGitDiff returns staged or unstaged diff.
func HandleGitDiff(ctx context.Context, params map[string]any, repoPath string) (any, error) {
	staged, _ := params["staged"].(bool)
	file, _ := params["file"].(string)

	args := []string{"diff"}
	if staged {
		args = append(args, "--staged")
	}
	if file != "" {
		args = append(args, "--", file)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	if repoPath != "" {
		cmd.Dir = repoPath
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff: %s", stderr.String())
	}
	diff := stdout.String()
	filesChanged := strings.Count(diff, "\ndiff --git")
	if strings.HasPrefix(diff, "diff --git") {
		filesChanged++
	}
	return map[string]any{
		"diff":          diff,
		"files_changed": filesChanged,
	}, nil
}

// HandleGitLog returns recent commits in a structured format.
func HandleGitLog(ctx context.Context, params map[string]any, repoPath string) (any, error) {
	n := 10
	if nf, ok := params["n"].(float64); ok && nf > 0 {
		n = int(nf)
	}
	sep := "\x1f"
	format := fmt.Sprintf("--format=%%H%s%%s%s%%an%s%%ai", sep, sep, sep)
	cmd := exec.CommandContext(ctx, "git", "log", fmt.Sprintf("-n%d", n), format)
	if repoPath != "" {
		cmd.Dir = repoPath
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git log: %s", stderr.String())
	}
	commits := make([]map[string]any, 0)
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, sep)
		if len(parts) != 4 {
			continue
		}
		commits = append(commits, map[string]any{
			"hash":    parts[0],
			"subject": parts[1],
			"author":  parts[2],
			"date":    parts[3],
		})
	}
	return map[string]any{"commits": commits}, nil
}

// HandleGitRemoteStatus fetches ahead/behind counts via a network call.
func HandleGitRemoteStatus(ctx context.Context, params map[string]any, repoPath string) (any, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fetchCmd := exec.CommandContext(fetchCtx, "git", "fetch", "--dry-run")
	if repoPath != "" {
		fetchCmd.Dir = repoPath
	}
	fetchCmd.Run() //nolint:errcheck — best-effort refresh

	gitOut := func(args ...string) string {
		cmd := exec.CommandContext(ctx, "git", args...)
		if repoPath != "" {
			cmd.Dir = repoPath
		}
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Run() //nolint:errcheck
		return strings.TrimSpace(out.String())
	}

	ahead, _ := strconv.Atoi(gitOut("rev-list", "--count", "@{u}..HEAD"))
	behind, _ := strconv.Atoi(gitOut("rev-list", "--count", "HEAD..@{u}"))
	upstream := gitOut("rev-parse", "--abbrev-ref", "@{u}")

	return map[string]any{
		"ahead":         ahead,
		"behind":        behind,
		"remote_branch": upstream,
	}, nil
}
