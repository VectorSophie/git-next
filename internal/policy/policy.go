package policy

import (
	"strings"

	"github.com/VectorSophie/git-next/pkg/model"
)

// CommandPolicy defines whether a proposed git command is safe in a given repo state.
type CommandPolicy struct {
	Pattern     string // matched against the command with "git " stripped
	BlockWhen   func(model.RepoState) bool
	Reason      string
	Alternative string
	RiskLevel   string // "caution" | "danger"
}

// Policies is the ordered list of command safety rules.
// First matching policy wins.
var Policies = []CommandPolicy{
	{
		Pattern:     "push --force",
		BlockWhen:   func(s model.RepoState) bool { return s.OnProtectedBranch || s.ForcePushToShared },
		Reason:      "force-pushing rewrites remote history and will break other developers' local branches",
		Alternative: "git revert HEAD   # create an inverse commit instead",
		RiskLevel:   "danger",
	},
	{
		Pattern:     "push -f",
		BlockWhen:   func(s model.RepoState) bool { return s.OnProtectedBranch || s.ForcePushToShared },
		Reason:      "force-pushing rewrites remote history and will break other developers' local branches",
		Alternative: "git revert HEAD   # create an inverse commit instead",
		RiskLevel:   "danger",
	},
	{
		Pattern:     "reset --hard",
		BlockWhen:   func(s model.RepoState) bool { return s.LastCommitPushed || s.OnProtectedBranch },
		Reason:      "commits have already been pushed; hard-resetting rewrites shared history",
		Alternative: "git revert HEAD   # safer: creates an undo commit rather than erasing history",
		RiskLevel:   "danger",
	},
	{
		Pattern:   "reset",
		BlockWhen: func(s model.RepoState) bool { return s.OnProtectedBranch && s.LastCommitPushed },
		Reason:    "resetting a protected branch after push would require a force-push to share the result",
		Alternative: "git revert HEAD   # preserve history instead",
		RiskLevel: "danger",
	},
	{
		Pattern:   "rebase",
		BlockWhen: func(s model.RepoState) bool { return s.OnProtectedBranch && s.LastCommitPushed },
		Reason:    "rebasing published commits on a protected branch rewrites shared history",
		Alternative: "git merge origin/<branch>   # merge preserves history on protected branches",
		RiskLevel: "danger",
	},
	{
		Pattern:     "filter-branch",
		BlockWhen:   func(s model.RepoState) bool { return true },
		Reason:      "filter-branch rewrites all affected commits and is effectively irreversible on shared branches",
		Alternative: "git filter-repo   # use git-filter-repo instead; it's safer and faster",
		RiskLevel:   "danger",
	},
	{
		Pattern:   "clean -f",
		BlockWhen: func(s model.RepoState) bool { return s.UntrackedFiles > 0 },
		Reason:    "git clean -f permanently deletes untracked files with no undo",
		Alternative: "git clean -n   # dry-run first to see what would be deleted",
		RiskLevel: "danger",
	},
	{
		Pattern:   "branch -D",
		BlockWhen: func(s model.RepoState) bool { return true },
		Reason:    "force-deleting a branch bypasses the safety check that the branch is fully merged",
		Alternative: "git branch -d <branch>   # use -d; it refuses to delete unmerged branches",
		RiskLevel: "caution",
	},
	{
		Pattern:   "tag -f",
		BlockWhen: func(s model.RepoState) bool { return true },
		Reason:    "force-tagging rewrites a published tag; anyone who fetched the old tag will have a mismatch",
		Alternative: "create a new tag with a new name instead",
		RiskLevel: "danger",
	},
	{
		Pattern:   "commit",
		BlockWhen: func(s model.RepoState) bool { return s.MergeInProgress || s.RebaseInProgress || s.CherryPickInProgress },
		Reason:    "an interactive operation is already in progress; resolve it first",
		Alternative: "git status   # check what operation is in progress and use --continue or --abort",
		RiskLevel: "caution",
	},
	{
		Pattern:   "push",
		BlockWhen: func(s model.RepoState) bool { return s.MergeInProgress || s.RebaseInProgress || s.CherryPickInProgress },
		Reason:    "an interactive operation is in progress; finish or abort it before pushing",
		Alternative: "git status   # check what operation is in progress",
		RiskLevel: "caution",
	},
}

// Evaluate checks the proposed command against all policies given the current repo state.
// Returns the first matching policy that blocks the command, or nil if allowed.
func Evaluate(proposed string, state model.RepoState) *CommandPolicy {
	normalized := strings.TrimSpace(proposed)
	normalized = strings.TrimPrefix(normalized, "git ")

	for i := range Policies {
		p := &Policies[i]
		if matchPattern(normalized, p.Pattern) && p.BlockWhen(state) {
			return p
		}
	}
	return nil
}

func matchPattern(normalized, pattern string) bool {
	return normalized == pattern ||
		strings.HasPrefix(normalized, pattern+" ") ||
		strings.HasPrefix(normalized, pattern+"\t")
}
