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
	// Switch to a non-protected branch so no protected-branch rules fire.
	r.Git("checkout", "-b", "feature/test")

	result, err := mcp.HandleGitStatus(context.Background(), map[string]any{}, r.Dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]any)
	if m["risk_level"] == nil {
		t.Error("risk_level field missing")
	}
	if m["advice"] == nil {
		t.Error("advice field missing")
	}
	if m["summary"] == nil {
		t.Error("summary field missing")
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
