package rules

import (
	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/pkg/model"
)

// DangerousRules returns rules for priority 100-90: "Put the keyboard down"
func DangerousRules(cfg *config.Config) []RuleDef {
	return []RuleDef{
		{
			ID:          "R037",
			Check:       R037,
			Command:     "# DO NOT git push --force on shared branches!",
			Description: "Force-push to shared branch - this is how trust dies",
			Priority:    100,
		},
		{
			ID:          "R038",
			Check:       R038,
			Command:     "# DO NOT rewrite published tags!",
			Description: "Rewrite published tags - releases are now folklore",
			Priority:    100,
		},
		{
			ID:          "R039",
			Check:       R039,
			Command:     "# DO NOT reset on protected branches!",
			Description: "Reset on protected branch - muscle memory is not a justification",
			Priority:    100,
		},
		{
			ID:          "R040",
			Check:       R040,
			Command:     "git submodule update --remote",
			Description: "Submodule pointer rewrite without update - builds will fail creatively",
			Priority:    100,
		},
		{
			ID:          "R041",
			Check:       R041,
			Command:     "# Accidental history rewrite detected - you don't get to pretend this was fine",
			Description: "Rebase or filter-branch after commits pulled by others",
			Priority:    100,
		},
		{
			ID:          "R021",
			Check:       R021,
			Command:     "git revert HEAD",
			Description: "Last commit was pushed - use revert instead of reset",
			Priority:    100,
		},
		{
			ID:          "R009",
			Check:       R009,
			Command:     "git merge --continue OR git merge --abort",
			Description: "Merge in progress - complete or abort",
			Priority:    98,
		},
		{
			ID:          "R010",
			Check:       R010,
			Command:     "git rebase --continue OR git rebase --abort",
			Description: "Rebase in progress - complete or abort",
			Priority:    97,
		},
		{
			ID:          "R011",
			Check:       R011,
			Command:     "git cherry-pick --continue OR git cherry-pick --abort",
			Description: "Cherry-pick in progress - complete or abort",
			Priority:    96,
		},
		{
			ID:          "R001",
			Check:       R001,
			Command:     "git checkout <branch>",
			Description: "Detached HEAD detected - checkout a branch",
			Priority:    95,
		},
		{
			ID:          "R032",
			Check:       R032,
			Command:     "git merge origin/<branch>",
			Description: "Diverged on protected branch - merge instead of rebase",
			Priority:    90,
		},
	}
}

// Rule check functions for dangerous operations (100-90)

// R037 - Force-push to shared branch
func R037(state model.RepoState) bool {
	return state.ForcePushToShared
}

// R038 - Rewritten published tags
func R038(state model.RepoState) bool {
	return state.RewrittenPublishedTags
}

// R039 - Reset on protected branch
func R039(state model.RepoState) bool {
	return state.ResetOnProtectedBranch
}

// R040 - Submodule pointer rewrite without update
func R040(state model.RepoState) bool {
	return state.SubmoduleRewriteNoUpdate
}

// R041 - Accidental history rewrite
func R041(state model.RepoState) bool {
	return state.AccidentalHistoryRewrite
}

// R021 - Revert Public Commit (beats reset)
func R021(state model.RepoState) bool {
	return state.LastCommitPushed
}

// R009 - Merge in Progress
func R009(state model.RepoState) bool {
	return state.MergeInProgress
}

// R010 - Rebase in Progress
func R010(state model.RepoState) bool {
	return state.RebaseInProgress
}

// R011 - Cherry-pick in Progress
func R011(state model.RepoState) bool {
	return state.CherryPickInProgress
}

// R001 - Detached HEAD
func R001(state model.RepoState) bool {
	return state.OnDetachedHead
}

// R032 - Merge on Protected Branch
func R032(state model.RepoState) bool {
	return state.Ahead > 0 &&
		state.Behind > 0 &&
		state.OnProtectedBranch
}
