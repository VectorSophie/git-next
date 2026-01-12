package rules

import (
	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/pkg/model"
)

// WorkflowRules returns rules for priority 59-30: Workflow hygiene
func WorkflowRules(cfg *config.Config) []RuleDef {
	return []RuleDef{
		{
			ID:          "R047",
			Check:       R047,
			Command:     "git checkout -b feature/<name>",
			Description: "Work on main instead of feature branch - you skipped the whole process part",
			Priority:    58,
		},
		{
			ID:          "R048",
			Check:       R048,
			Command:     "git merge main (or rebase)",
			Description: "Long-lived feature branch - merge debt accumulating interest",
			Priority:    56,
		},
		{
			ID:          "R005",
			Check:       R005,
			Command:     "git pull",
			Description: "Behind remote and clean - pull updates",
			Priority:    55,
		},
		{
			ID:          "R049",
			Check:       R049,
			Command:     "git rebase -i HEAD~N",
			Description: "Squash recommended before merge - many noisy commits",
			Priority:    52,
		},
		{
			ID:          "R050",
			Check:       R050,
			Command:     "git commit --amend",
			Description: "WIP commit on shared branch - this is not your personal notebook",
			Priority:    51,
		},
		{
			ID:          "R051",
			Check:       R051,
			Command:     "git rebase main",
			Description: "Rebase recommended instead of merge - keep linear history",
			Priority:    50,
		},
		{
			ID:          "R004",
			Check:       R004,
			Command:     "git push",
			Description: "Local commits ready to push",
			Priority:    50,
		},
		{
			ID:          "R030",
			Check:       R030,
			Command:     "git pull --ff-only",
			Description: "Can fast-forward - safe to pull",
			Priority:    48,
		},
		{
			ID:    "R020",
			Check: func(state model.RepoState) bool { return R020(state, cfg) },
			Command:     "git reset --soft HEAD~N",
			Description: "Local commits (≤3) can be soft reset",
			Priority:    45,
		},
		{
			ID:    "R022",
			Check: func(state model.RepoState) bool { return R022(state, cfg) },
			Command:     "git rebase -i HEAD~N",
			Description: "Too many local commits - use interactive rebase",
			Priority:    42,
		},
		{
			ID:          "R003",
			Check:       R003,
			Command:     "git commit",
			Description: "Staged files waiting for commit",
			Priority:    38,
		},
		{
			ID:          "R002",
			Check:       R002,
			Command:     "git add <files> && git commit",
			Description: "Modified files not staged",
			Priority:    35,
		},
	}
}

// Rule check functions for workflow hygiene (59-30)

// R047 - Work on main instead of feature branch
func R047(state model.RepoState) bool {
	return state.WorkOnMainNotFeature
}

// R048 - Long-lived feature branch
func R048(state model.RepoState) bool {
	return state.LongLivedFeatureBranch
}

// R049 - Squash recommended before merge
func R049(state model.RepoState) bool {
	return state.SquashRecommended
}

// R050 - WIP commit on shared branch
func R050(state model.RepoState) bool {
	return state.WIPCommitOnShared
}

// R051 - Rebase recommended instead of merge
func R051(state model.RepoState) bool {
	return state.RebaseInsteadOfMerge
}

// R005 - Pull When Behind and Clean
func R005(state model.RepoState) bool {
	return state.Behind > 0 &&
		!state.Dirty
}

// R030 - Fast-Forward Pull
func R030(state model.RepoState) bool {
	return state.Behind > 0 &&
		state.Ahead == 0 &&
		!state.Dirty
}

// R004 - Push Local Commits
func R004(state model.RepoState) bool {
	return state.Ahead > 0 &&
		state.Behind == 0 &&
		!state.Dirty
}

// R020 - Soft Reset Local Commits
func R020(state model.RepoState, cfg *config.Config) bool {
	maxCommits := cfg.GetIntParam("R020", "max_commits", 3)
	return !state.LastCommitPushed &&
		state.CommitCountSincePush > 0 &&
		state.CommitCountSincePush <= maxCommits
}

// R022 - Too Many Commits to Reset
func R022(state model.RepoState, cfg *config.Config) bool {
	minCommits := cfg.GetIntParam("R022", "min_commits", 4)
	return !state.LastCommitPushed &&
		state.CommitCountSincePush >= minCommits
}

// R003 - Staged but Not Committed
func R003(state model.RepoState) bool {
	return state.StagedFiles > 0
}

// R002 - Dirty Working Tree
func R002(state model.RepoState) bool {
	return state.ModifiedFiles > 0 &&
		state.StagedFiles == 0
}
