# git-next MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `git-next mcp` subcommand that exposes 7 git-safety tools over MCP stdio, letting AI agents inspect repo state, guard dangerous commands, and execute git with policy-enforced guardrails.

**Architecture:** New `internal/mcp/` package (classifier, executor, tool handlers, JSON-RPC server) + thin `cmd/git-next/cmd_mcp.go` subcommand. Existing packages untouched. Transport is newline-delimited JSON-RPC 2.0 over stdio.

**Tech Stack:** Go stdlib only (`os/exec`, `bufio`, `encoding/json`). Zero new dependencies.

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `internal/mcp/classifier.go` | Create | `Tier` type, `Classify()` — SAFE/GUARDED/BLOCKED |
| `internal/mcp/classifier_test.go` | Create | Table tests for all classification paths |
| `internal/mcp/executor.go` | Create | `RunResult`, `Run()`, `execute()` |
| `internal/mcp/tools.go` | Create | `Tool` type, `registry`, all 7 `Handle*` handlers |
| `internal/mcp/tools_test.go` | Create | Unit tests for each handler using testutil.NewRepo |
| `internal/mcp/server.go` | Create | `Serve()`, JSON-RPC 2.0 dispatch loop |
| `cmd/git-next/cmd_mcp.go` | Create | `runMCP()` subcommand entry |
| `cmd/git-next/main.go` | Modify | Add `"mcp"` case to dispatch switch |
| `.mcp.json` | Create | Example Claude Code config |
| `CLAUDE.md` | Modify | Add MCP section |

---

## Task 1: Command Classifier

**Files:**
- Create: `internal/mcp/classifier.go`
- Create: `internal/mcp/classifier_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/mcp/classifier_test.go`:

```go
package mcp_test

import (
	"testing"

	"github.com/VectorSophie/git-next/internal/mcp"
	"github.com/VectorSophie/git-next/pkg/model"
)

func TestClassify(t *testing.T) {
	clean := model.RepoState{}
	protected := model.RepoState{OnProtectedBranch: true, LastCommitPushed: true}
	hasUntracked := model.RepoState{UntrackedFiles: 3}

	tests := []struct {
		command string
		state   model.RepoState
		want    mcp.Tier
	}{
		// Non-git: always blocked
		{"bash -c 'rm -rf /'", clean, mcp.TierBlocked},
		{"rm -rf /", clean, mcp.TierBlocked},
		{"python script.py", clean, mcp.TierBlocked},

		// Safe
		{"git status", clean, mcp.TierSafe},
		{"git status --porcelain", clean, mcp.TierSafe},
		{"git log -n 5", clean, mcp.TierSafe},
		{"git diff", clean, mcp.TierSafe},
		{"git diff --staged", clean, mcp.TierSafe},
		{"git add README.md", clean, mcp.TierSafe},
		{"git add -p", clean, mcp.TierSafe},
		{"git fetch", clean, mcp.TierSafe},
		{"git fetch origin", clean, mcp.TierSafe},
		{"git branch", clean, mcp.TierSafe},
		{"git branch --list", clean, mcp.TierSafe},
		{"git stash", clean, mcp.TierSafe},
		{"git stash list", clean, mcp.TierSafe},
		{"git stash show", clean, mcp.TierSafe},
		{"git tag --list", clean, mcp.TierSafe},
		{"git tag -l", clean, mcp.TierSafe},
		{"git commit -m message", clean, mcp.TierSafe},
		{"git show HEAD", clean, mcp.TierSafe},

		// Guarded
		{"git push", clean, mcp.TierGuarded},
		{"git push origin main", clean, mcp.TierGuarded},
		{"git merge feature", clean, mcp.TierGuarded},
		{"git rebase main", clean, mcp.TierGuarded},
		{"git commit --amend", clean, mcp.TierGuarded},
		{"git commit --amend -m new", clean, mcp.TierGuarded},
		{"git stash pop", clean, mcp.TierGuarded},
		{"git stash drop", clean, mcp.TierGuarded},
		{"git stash clear", clean, mcp.TierGuarded},
		{"git branch -d old-feature", clean, mcp.TierGuarded},
		{"git tag v1.0", clean, mcp.TierGuarded},
		{"git tag -a v1.0 -m release", clean, mcp.TierGuarded},

		// Blocked by policy
		{"git push --force", protected, mcp.TierBlocked},
		{"git push -f", protected, mcp.TierBlocked},
		{"git reset --hard", protected, mcp.TierBlocked},
		{"git filter-branch", clean, mcp.TierBlocked},
		{"git branch -D old", clean, mcp.TierBlocked},
		{"git tag -f v1.0", clean, mcp.TierBlocked},
		{"git clean -f", hasUntracked, mcp.TierBlocked},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := mcp.Classify(tt.command, tt.state)
			if got != tt.want {
				t.Errorf("Classify(%q) = %s, want %s", tt.command, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to confirm it fails**

```
go test ./internal/mcp/... -run TestClassify -v
```

Expected: `cannot find package "github.com/VectorSophie/git-next/internal/mcp"`

- [ ] **Step 3: Write implementation**

Create `internal/mcp/classifier.go`:

```go
package mcp

import (
	"strings"

	"github.com/VectorSophie/git-next/internal/policy"
	"github.com/VectorSophie/git-next/pkg/model"
)

// Tier is the safety classification of a proposed git command.
type Tier int

const (
	TierSafe    Tier = iota // Execute immediately
	TierGuarded             // Require confirmed=true before executing
	TierBlocked             // Never execute
)

func (t Tier) String() string {
	switch t {
	case TierSafe:
		return "safe"
	case TierGuarded:
		return "guarded"
	default:
		return "blocked"
	}
}

// Classify returns the execution tier for a proposed command.
// Commands not starting with "git " are always TierBlocked.
// policy.Evaluate is checked first; per-verb rules apply only when policy passes.
func Classify(proposed string, state model.RepoState) Tier {
	normalized := strings.TrimSpace(proposed)
	if !strings.HasPrefix(normalized, "git ") {
		return TierBlocked
	}
	sub := strings.TrimPrefix(normalized, "git ")
	args := strings.Fields(sub)
	if len(args) == 0 {
		return TierBlocked
	}
	verb := args[0]

	if policy.Evaluate(proposed, state) != nil {
		return TierBlocked
	}

	switch verb {
	case "status", "log", "diff", "show", "fetch", "remote",
		"rev-list", "rev-parse", "ls-files", "shortlog",
		"describe", "cherry", "check-ignore":
		return TierSafe

	case "add":
		return TierSafe

	case "commit":
		for _, arg := range args[1:] {
			if arg == "--amend" {
				return TierGuarded
			}
		}
		return TierSafe

	case "branch":
		if len(args) > 1 && args[1] == "-d" {
			return TierGuarded
		}
		return TierSafe

	case "stash":
		if len(args) > 1 {
			switch args[1] {
			case "pop", "drop", "clear":
				return TierGuarded
			}
		}
		return TierSafe

	case "tag":
		if len(args) > 1 {
			switch args[1] {
			case "-l", "--list", "--contains", "--sort", "--format", "--points-at":
				return TierSafe
			}
		}
		return TierGuarded

	case "config":
		for _, arg := range args[1:] {
			if arg == "--global" || arg == "--system" || arg == "--local" {
				return TierGuarded
			}
		}
		return TierSafe

	case "push", "merge", "rebase":
		return TierGuarded
	}

	return TierGuarded
}
```

- [ ] **Step 4: Run test to confirm it passes**

```
go test ./internal/mcp/... -run TestClassify -v
```

Expected: all subtests PASS.

- [ ] **Step 5: Commit**

```
git add internal/mcp/classifier.go internal/mcp/classifier_test.go
git commit -m "feat(mcp): add command classifier (SAFE/GUARDED/BLOCKED)"
```

---

## Task 2: Executor

**Files:**
- Create: `internal/mcp/executor.go`

- [ ] **Step 1: Write implementation**

Create `internal/mcp/executor.go`:

```go
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

// Run classifies command and either executes it, returns a guard result, or blocks it.
// repoPath is passed as cmd.Dir; an empty string uses the process cwd.
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
```

- [ ] **Step 2: Confirm it compiles**

```
go build ./internal/mcp/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```
git add internal/mcp/executor.go
git commit -m "feat(mcp): add smart executor (Run/execute)"
```

---

## Task 3: Tool Handlers

**Files:**
- Create: `internal/mcp/tools.go`
- Create: `internal/mcp/tools_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/mcp/tools_test.go`:

```go
package mcp_test

import (
	"context"
	"testing"

	"github.com/VectorSophie/git-next/internal/mcp"
	"github.com/VectorSophie/git-next/internal/testutil"
)

func TestHandleGitStatus_CleanRepo(t *testing.T) {
	r := testutil.NewRepo(t)
	r.InitialCommit()

	result, err := mcp.HandleGitStatus(context.Background(), map[string]any{}, r.Dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["risk_level"] != "safe" {
		t.Errorf("expected risk_level 'safe', got %v", m["risk_level"])
	}
	if m["advice"] == nil {
		t.Error("advice field missing")
	}
}

func TestHandleGitStatus_DirtyRepo(t *testing.T) {
	r := testutil.NewRepo(t)
	r.InitialCommit()
	r.WriteFile("new.txt", "untracked")

	result, err := mcp.HandleGitStatus(context.Background(), map[string]any{}, r.Dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	items := m["advice"].([]map[string]any)
	if len(items) == 0 {
		t.Error("expected at least one advice item for untracked file")
	}
}

func TestHandleGitGuard_AllowsSafe(t *testing.T) {
	r := testutil.NewRepo(t)
	r.InitialCommit()

	result, err := mcp.HandleGitGuard(context.Background(), map[string]any{
		"command": "git status",
	}, r.Dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["allowed"] != true {
		t.Errorf("expected allowed=true, got %v", m["allowed"])
	}
}

func TestHandleGitGuard_BlocksFilterBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.InitialCommit()

	result, err := mcp.HandleGitGuard(context.Background(), map[string]any{
		"command": "git filter-branch",
	}, r.Dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["allowed"] != false {
		t.Errorf("expected allowed=false for git filter-branch")
	}
}

func TestHandleGitGuard_MissingCommand(t *testing.T) {
	_, err := mcp.HandleGitGuard(context.Background(), map[string]any{}, "")
	if err == nil {
		t.Error("expected error when command param is missing")
	}
}

func TestHandleGitRun_SafeExecutes(t *testing.T) {
	r := testutil.NewRepo(t)
	r.InitialCommit()

	result, err := mcp.HandleGitRun(context.Background(), map[string]any{
		"command": "git status",
	}, r.Dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rr := result.(mcp.RunResult)
	if rr.Blocked || rr.NeedsConfirm {
		t.Errorf("expected safe execution, got blocked=%v needsConfirm=%v", rr.Blocked, rr.NeedsConfirm)
	}
}

func TestHandleGitRun_GuardedRequiresConfirm(t *testing.T) {
	r := testutil.NewRepo(t)
	r.InitialCommit()

	result, err := mcp.HandleGitRun(context.Background(), map[string]any{
		"command":   "git push",
		"confirmed": false,
	}, r.Dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rr := result.(mcp.RunResult)
	if !rr.NeedsConfirm {
		t.Error("expected requires_confirmation=true for git push without confirmed")
	}
}

func TestHandleGitRun_BlockedNeverExecutes(t *testing.T) {
	r := testutil.NewRepo(t)
	r.InitialCommit()

	result, err := mcp.HandleGitRun(context.Background(), map[string]any{
		"command":   "git filter-branch",
		"confirmed": true,
	}, r.Dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rr := result.(mcp.RunResult)
	if !rr.Blocked {
		t.Error("expected blocked=true for git filter-branch even with confirmed=true")
	}
}

func TestHandleGitExplain_KnownRule(t *testing.T) {
	result, err := mcp.HandleGitExplain(context.Background(), map[string]any{
		"rule_id": "R037",
	}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["id"] != "R037" {
		t.Errorf("expected id=R037, got %v", m["id"])
	}
	if m["why"] == "" {
		t.Error("expected non-empty why field")
	}
}

func TestHandleGitExplain_UnknownRule(t *testing.T) {
	_, err := mcp.HandleGitExplain(context.Background(), map[string]any{
		"rule_id": "R999",
	}, "")
	if err == nil {
		t.Error("expected error for unknown rule ID")
	}
}

func TestHandleGitDiff_Clean(t *testing.T) {
	r := testutil.NewRepo(t)
	r.InitialCommit()

	result, err := mcp.HandleGitDiff(context.Background(), map[string]any{}, r.Dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["diff"] == nil {
		t.Error("diff field missing")
	}
	if m["files_changed"].(int) != 0 {
		t.Errorf("expected 0 files changed in clean repo, got %v", m["files_changed"])
	}
}

func TestHandleGitLog(t *testing.T) {
	r := testutil.NewRepo(t)
	r.InitialCommit()

	result, err := mcp.HandleGitLog(context.Background(), map[string]any{
		"n": float64(5),
	}, r.Dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	commits := m["commits"].([]map[string]any)
	if len(commits) == 0 {
		t.Error("expected at least one commit")
	}
	if commits[0]["hash"] == "" {
		t.Error("expected non-empty hash")
	}
}
```

- [ ] **Step 2: Run test to confirm failures**

```
go test ./internal/mcp/... -run "TestHandle" -v
```

Expected: compile errors (tools.go doesn't exist yet).

- [ ] **Step 3: Write implementation**

Create `internal/mcp/tools.go`:

```go
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
		Description: "Run a git command with safety classification. Safe commands execute immediately. Guarded commands (push, merge, rebase, commit --amend, stash pop) require a second call with confirmed=true. Blocked commands are never executed even with confirmed=true. Note: arguments with spaces are split on whitespace; avoid shell quoting.",
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

// chdirMu guards os.Chdir calls in runInDir. MCP stdio is sequential
// but Go tests may run handlers concurrently.
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

	type item struct {
		RuleID      string `json:"rule_id"`
		Severity    string `json:"severity"`
		Description string `json:"description"`
		Command     string `json:"command"`
		Destructive bool   `json:"destructive"`
	}
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
	_ = item{} // keep import

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

	gitDir := func(args ...string) string {
		cmd := exec.CommandContext(ctx, "git", args...)
		if repoPath != "" {
			cmd.Dir = repoPath
		}
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Run() //nolint:errcheck
		return strings.TrimSpace(out.String())
	}

	ahead, _ := strconv.Atoi(gitDir("rev-list", "--count", "@{u}..HEAD"))
	behind, _ := strconv.Atoi(gitDir("rev-list", "--count", "HEAD..@{u}"))
	upstream := gitDir("rev-parse", "--abbrev-ref", "@{u}")

	return map[string]any{
		"ahead":         ahead,
		"behind":        behind,
		"remote_branch": upstream,
	}, nil
}
```

- [ ] **Step 4: Fix unused import**

The `item` struct in `HandleGitStatus` was used to illustrate the shape but the actual code uses `map[string]any`. Remove the `item` type and the `_ = item{}` line:

```go
// Remove these two lines from HandleGitStatus:
type item struct { ... }
_ = item{}
```

The final `HandleGitStatus` builds the `active` slice directly using `map[string]any` — no helper struct needed.

- [ ] **Step 5: Run tests**

```
go test ./internal/mcp/... -v
```

Expected: all `TestHandle*` tests pass. `TestClassify` continues to pass.

- [ ] **Step 6: Commit**

```
git add internal/mcp/tools.go internal/mcp/tools_test.go
git commit -m "feat(mcp): add 7 tool handlers and registry"
```

---

## Task 4: MCP Server

**Files:**
- Create: `internal/mcp/server.go`

- [ ] **Step 1: Write implementation**

Create `internal/mcp/server.go`:

```go
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolCallResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Serve runs the MCP JSON-RPC 2.0 server over stdio until stdin closes.
// repoPath is the default repository directory for all tool calls.
func Serve(repoPath string) error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024) // 4 MB for large diffs
	enc := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			enc.Encode(rpcResponse{ //nolint:errcheck
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32700, Message: "parse error"},
			})
			continue
		}

		// Notifications (no ID) get no response.
		if req.ID == nil || string(req.ID) == "null" {
			continue
		}

		result, rpcErr := dispatch(context.Background(), req, repoPath)
		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
		if rpcErr != nil {
			resp.Error = rpcErr
		} else {
			resp.Result = result
		}
		enc.Encode(resp) //nolint:errcheck
	}
	return scanner.Err()
}

func dispatch(ctx context.Context, req rpcRequest, repoPath string) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "git-next", "version": "1.0"},
		}, nil

	case "tools/list":
		list := make([]map[string]any, 0, len(Registry))
		for _, t := range Registry {
			list = append(list, map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": json.RawMessage(t.InputSchema),
			})
		}
		return map[string]any{"tools": list}, nil

	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid params"}
		}
		var found *Tool
		for i := range Registry {
			if Registry[i].Name == p.Name {
				found = &Registry[i]
				break
			}
		}
		if found == nil {
			return nil, &rpcError{Code: -32601, Message: fmt.Sprintf("unknown tool: %s", p.Name)}
		}

		// repo_path param overrides server default
		resolvedPath := repoPath
		if rp, ok := p.Arguments["repo_path"].(string); ok && rp != "" {
			resolvedPath = rp
		}

		result, err := found.Handler(ctx, p.Arguments, resolvedPath)
		if err != nil {
			text, _ := json.Marshal(map[string]string{"error": err.Error()})
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: string(text)}},
				IsError: true,
			}, nil
		}
		text, _ := json.Marshal(result)
		return toolCallResult{
			Content: []toolContent{{Type: "text", Text: string(text)}},
		}, nil

	default:
		return nil, &rpcError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
	}
}
```

- [ ] **Step 2: Build**

```
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Run all tests**

```
go test ./...
```

Expected: all pass.

- [ ] **Step 4: Commit**

```
git add internal/mcp/server.go
git commit -m "feat(mcp): add JSON-RPC 2.0 stdio server"
```

---

## Task 5: Subcommand + Dispatch

**Files:**
- Create: `cmd/git-next/cmd_mcp.go`
- Modify: `cmd/git-next/main.go` (lines 26–44, add `"mcp"` case)

- [ ] **Step 1: Create the subcommand file**

Create `cmd/git-next/cmd_mcp.go`:

```go
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

Add to .mcp.json in your project:
  {
    "mcpServers": {
      "git-next": { "command": "git-next", "args": ["mcp"] }
    }
  }

Or install globally in ~/.claude/mcp.json.`)
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
```

- [ ] **Step 2: Add mcp to main.go dispatch**

In `cmd/git-next/main.go`, the dispatch switch starts at line 25. Add the `mcp` case before the closing brace:

```go
// existing cases:
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
// ADD THIS:
case "mcp":
    runMCP(os.Args[2:])
    return
```

- [ ] **Step 3: Build the binary**

```
go build -o git-next ./cmd/git-next/main.go
```

Expected: binary produced, no errors.

- [ ] **Step 4: Run all tests**

```
go test ./...
```

Expected: all pass.

- [ ] **Step 5: Commit**

```
git add cmd/git-next/cmd_mcp.go cmd/git-next/main.go
git commit -m "feat(mcp): add 'git-next mcp' subcommand"
```

---

## Task 6: Release Artifacts

**Files:**
- Create: `.mcp.json`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Create .mcp.json**

Create `.mcp.json` in the repo root:

```json
{
  "mcpServers": {
    "git-next": {
      "command": "git-next",
      "args": ["mcp"]
    }
  }
}
```

- [ ] **Step 2: Update CLAUDE.md**

Add a new `### MCP server` section under `### cmd/git-next`:

```markdown
### MCP server — `git-next mcp`

Starts a JSON-RPC 2.0 MCP server over stdio. Claude Code (and any MCP-compatible agent) spawns it via `.mcp.json`. Exposes 7 tools: `git_status`, `git_guard`, `git_run`, `git_explain`, `git_diff`, `git_log`, `git_remote_status`.

`internal/mcp/` package layout:
- `classifier.go` — `Classify(command, state) Tier` (SAFE/GUARDED/BLOCKED)
- `executor.go` — `Run()` / `RunResult` — smart executor with confirmation flow
- `tools.go` — `Registry []Tool` + all 7 `Handle*` functions
- `server.go` — `Serve(repoPath)` — request/response loop

To add a tool: append a `Tool{}` entry to `Registry` in `tools.go`. The server discovers it automatically.
```

- [ ] **Step 3: Commit**

```
git add .mcp.json CLAUDE.md
git commit -m "docs: add .mcp.json and MCP server docs to CLAUDE.md"
```

---

## Task 7: Integration Test

- [ ] **Step 1: Build the binary**

```
go build -o git-next ./cmd/git-next/main.go
```

- [ ] **Step 2: Test the initialize handshake**

```
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./git-next mcp
```

Expected output (one JSON line):
```json
{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{}},"protocolVersion":"2024-11-05","serverInfo":{"name":"git-next","version":"1.0"}}}
```

- [ ] **Step 3: Test tools/list**

```
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./git-next mcp
```

Expected: JSON with `result.tools` array containing 7 entries (`git_status`, `git_guard`, `git_run`, `git_explain`, `git_diff`, `git_log`, `git_remote_status`).

- [ ] **Step 4: Test git_status tool call**

```
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"git_status","arguments":{}}}' | ./git-next mcp
```

Expected: JSON with `result.content[0].text` containing a JSON object with `advice`, `risk_level`, and `summary` fields.

- [ ] **Step 5: Test git_run safe command**

```
echo '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"git_run","arguments":{"command":"git status"}}}' | ./git-next mcp
```

Expected: `result.content[0].text` contains `{"stdout":"...","exit_code":0}`.

- [ ] **Step 6: Test git_run guarded command (no confirmation)**

```
echo '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"git_run","arguments":{"command":"git push","confirmed":false}}}' | ./git-next mcp
```

Expected: `result.content[0].text` contains `{"requires_confirmation":true,"risk_level":"caution","confirm_message":"..."}`.

- [ ] **Step 7: Test with Claude Code**

Add `.mcp.json` to the project root (already done in Task 6). Start Claude Code in this repository. In a Claude Code session, run:

```
Use the git_status tool to check this repo.
```

Verify Claude Code calls `git_status` and reports advice correctly.

- [ ] **Step 8: Final test run**

```
go test ./...
```

Expected: all pass.

- [ ] **Step 9: Final commit**

```
git add -u
git commit -m "test(mcp): integration validated, MCP server complete"
```

---

## Self-Review Notes

- **Spec coverage:** All 7 tools implemented. SAFE/GUARDED/BLOCKED model matches spec. `repo_path` override present in all handlers. `git_remote_status` uses 30s timeout. ✓
- **Type consistency:** `RunResult` defined in `executor.go`, used in `tools_test.go` as `mcp.RunResult`. `Tool` and `Registry` defined in `tools.go`, used in `server.go`. ✓
- **Known limitation documented:** `execute()` uses `strings.Fields` (whitespace split) — commit messages with spaces must use pre-staged commits or `--file`. Documented in `git_run` tool description. ✓
- **No placeholders:** All steps have complete code. ✓
