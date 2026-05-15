package policy_test

import (
	"testing"

	"github.com/VectorSophie/git-next/internal/policy"
	"github.com/VectorSophie/git-next/pkg/model"
)

func TestForcePushBlockedOnProtectedBranch(t *testing.T) {
	state := model.RepoState{OnProtectedBranch: true}
	if p := policy.Evaluate("git push --force", state); p == nil {
		t.Error("expected push --force to be blocked on protected branch")
	}
}

func TestForcePushAllowedOnFeatureBranch(t *testing.T) {
	state := model.RepoState{OnProtectedBranch: false, ForcePushToShared: false}
	if p := policy.Evaluate("git push --force", state); p != nil {
		t.Errorf("expected push --force to be allowed on feature branch, got: %s", p.Reason)
	}
}

func TestResetHardBlockedWhenCommitPushed(t *testing.T) {
	state := model.RepoState{LastCommitPushed: true}
	if p := policy.Evaluate("git reset --hard HEAD~1", state); p == nil {
		t.Error("expected reset --hard to be blocked when last commit is pushed")
	}
}

func TestResetHardAllowedWhenNotPushed(t *testing.T) {
	state := model.RepoState{LastCommitPushed: false, OnProtectedBranch: false}
	if p := policy.Evaluate("git reset --hard HEAD~1", state); p != nil {
		t.Errorf("expected reset --hard to be allowed on unpushed commit, got: %s", p.Reason)
	}
}

func TestFilterBranchAlwaysBlocked(t *testing.T) {
	state := model.RepoState{}
	if p := policy.Evaluate("git filter-branch --tree-filter 'rm -f passwords.txt'", state); p == nil {
		t.Error("expected filter-branch to always be blocked")
	}
}

func TestSafeCommandAllowed(t *testing.T) {
	state := model.RepoState{OnProtectedBranch: true, LastCommitPushed: true}
	safeCmds := []string{"git status", "git log", "git diff", "git fetch"}
	for _, cmd := range safeCmds {
		if p := policy.Evaluate(cmd, state); p != nil {
			t.Errorf("expected %q to be allowed, got blocked: %s", cmd, p.Reason)
		}
	}
}

func TestPushBlockedDuringMerge(t *testing.T) {
	state := model.RepoState{MergeInProgress: true}
	if p := policy.Evaluate("git push", state); p == nil {
		t.Error("expected push to be blocked during merge in progress")
	}
}

func TestShortFlagForcePush(t *testing.T) {
	state := model.RepoState{OnProtectedBranch: true}
	if p := policy.Evaluate("git push -f", state); p == nil {
		t.Error("expected push -f to be blocked on protected branch")
	}
}
