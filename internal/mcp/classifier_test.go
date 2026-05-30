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
