package rules

import "github.com/yourusername/git-next/pkg/model"

// Rule represents a function that evaluates repository state
type Rule func(state model.RepoState) bool

// RuleDef defines a rule with its metadata
type RuleDef struct {
	ID          string
	Check       Rule
	Command     string
	Description string
	Priority    int
}

// AllRules returns all defined rules sorted by priority
func AllRules() []RuleDef {
	return []RuleDef{
		// 100-90: YOU ARE ABOUT TO DO SOMETHING DANGEROUS
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

		// 89-60: REPO INTEGRITY ISSUES
		{
			ID:          "R006",
			Check:       R006,
			Command:     "git rebase origin/<branch> OR git merge origin/<branch>",
			Description: "Branch has diverged - need to sync",
			Priority:    80,
		},
		{
			ID:          "R034",
			Check:       R034,
			Command:     "git branch --set-upstream-to=origin/<branch>",
			Description: "No upstream configured for current branch",
			Priority:    75,
		},
		{
			ID:          "R031",
			Check:       R031,
			Command:     "git rebase origin/<branch>",
			Description: "Feature branch diverged - rebase to keep linear history",
			Priority:    70,
		},
		{
			ID:          "R035",
			Check:       R035,
			Command:     "git branch -d <branch>",
			Description: "Merged branches ready for cleanup",
			Priority:    65,
		},
		{
			ID:          "R036",
			Check:       R036,
			Command:     "git branch -d <branch>",
			Description: "Gone remote branches - local cleanup needed",
			Priority:    62,
		},
		{
			ID:          "R033",
			Check:       R033,
			Command:     "git merge origin/<branch>",
			Description: "Existing merge commits detected - continue with merge",
			Priority:    60,
		},

		// 59-30: WORKFLOW HYGIENE
		{
			ID:          "R005",
			Check:       R005,
			Command:     "git pull",
			Description: "Behind remote and clean - pull updates",
			Priority:    55,
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
			ID:          "R020",
			Check:       R020,
			Command:     "git reset --soft HEAD~N",
			Description: "Local commits (≤3) can be soft reset",
			Priority:    45,
		},
		{
			ID:          "R022",
			Check:       R022,
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

		// 29-10: MILD SUGGESTIONS
		{
			ID:          "R007",
			Check:       R007,
			Command:     "git add <files>",
			Description: "Untracked files present",
			Priority:    20,
		},
		{
			ID:          "R008",
			Check:       R008,
			Command:     "git stash pop",
			Description: "Stash exists - consider applying",
			Priority:    15,
		},
	}
}

// 100-90: YOU ARE ABOUT TO DO SOMETHING DANGEROUS

// R001 - Detached HEAD
func R001(state model.RepoState) bool {
	return state.OnDetachedHead
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

// R021 - Revert Public Commit (beats reset)
func R021(state model.RepoState) bool {
	return state.LastCommitPushed
}

// R032 - Merge on Protected Branch
func R032(state model.RepoState) bool {
	return state.Ahead > 0 &&
		state.Behind > 0 &&
		state.OnProtectedBranch
}

// 89-60: REPO INTEGRITY ISSUES

// R006 - Diverged Branch
func R006(state model.RepoState) bool {
	return state.Ahead > 0 && state.Behind > 0
}

// R031 - Rebase Feature Branch
func R031(state model.RepoState) bool {
	return state.Ahead > 0 &&
		state.Behind > 0 &&
		!state.OnProtectedBranch
}

// R033 - Existing Merge History
func R033(state model.RepoState) bool {
	return state.HasMergeCommits &&
		state.Behind > 0
}

// R034 - No Upstream Configured
func R034(state model.RepoState) bool {
	return state.NoUpstream && !state.OnDetachedHead
}

// R035 - Merged Branches Ready for Cleanup
func R035(state model.RepoState) bool {
	return len(state.MergedBranches) > 0
}

// R036 - Gone Remote Branches
func R036(state model.RepoState) bool {
	return len(state.GoneBranches) > 0
}

// 59-30: WORKFLOW HYGIENE

// R020 - Soft Reset Local Commits
func R020(state model.RepoState) bool {
	return !state.LastCommitPushed &&
		state.CommitCountSincePush > 0 &&
		state.CommitCountSincePush <= 3
}

// R022 - Too Many Commits to Reset
func R022(state model.RepoState) bool {
	return !state.LastCommitPushed &&
		state.CommitCountSincePush > 3
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

// R003 - Staged but Not Committed
func R003(state model.RepoState) bool {
	return state.StagedFiles > 0
}

// R002 - Dirty Working Tree
func R002(state model.RepoState) bool {
	return state.ModifiedFiles > 0 &&
		state.StagedFiles == 0
}

// 29-10: MILD SUGGESTIONS

// R007 - Untracked Files
func R007(state model.RepoState) bool {
	return state.UntrackedFiles > 0
}

// R008 - Stash Exists
func R008(state model.RepoState) bool {
	return state.HasStash
}
