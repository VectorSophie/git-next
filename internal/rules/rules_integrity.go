package rules

import (
	"github.com/VectorSophie/git-next/pkg/model"
)

// IntegrityRules returns rules for priority 89-60: Repo integrity issues
func IntegrityRules() []RuleDef {
	return []RuleDef{
		{
			ID:          "R042",
			Check:       R042,
			Command:     "# Remove conflict markers from files before committing",
			Description: "Conflicted files staged - if <<<<<<< is in the diff, stop pretending",
			Priority:    89,
		},
		{
			ID:          "R043",
			Check:       R043,
			Command:     "git lfs track <pattern> && git add .gitattributes",
			Description: "Binary files changed without LFS - Git is not a landfill",
			Priority:    85,
		},
		{
			ID:          "R044",
			Check:       R044,
			Command:     "git config core.autocrlf true (or false)",
			Description: "Line ending normalization conflict - someone's editor declared war",
			Priority:    82,
		},
		{
			ID:          "R045",
			Check:       R045,
			Command:     "cd <submodule> && git checkout <branch>",
			Description: "Submodule detached HEAD - time capsule mode engaged",
			Priority:    81,
		},
		{
			ID:          "R046",
			Check:       R046,
			Command:     "git fetch --unshallow",
			Description: "Shallow clone doing history ops - Git will lie to you politely",
			Priority:    80,
		},
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
	}
}

// Rule check functions for repo integrity (89-60)

// R042 - Conflicted files staged
func R042(state model.RepoState) bool {
	return state.ConflictedFilesStaged
}

// R043 - Large binaries without LFS
func R043(state model.RepoState) bool {
	return state.LargeBinariesWithoutLFS
}

// R044 - Line ending normalization conflict
func R044(state model.RepoState) bool {
	return state.LineEndingConflict
}

// R045 - Submodule detached HEAD
func R045(state model.RepoState) bool {
	return state.SubmoduleDetachedHead
}

// R046 - Shallow clone doing history ops
func R046(state model.RepoState) bool {
	return state.ShallowCloneHistoryOps
}

// R006 - Diverged Branch
func R006(state model.RepoState) bool {
	return state.Ahead > 0 && state.Behind > 0
}

// R034 - No Upstream Configured
func R034(state model.RepoState) bool {
	return state.NoUpstream && !state.OnDetachedHead
}

// R031 - Rebase Feature Branch
func R031(state model.RepoState) bool {
	return state.Ahead > 0 &&
		state.Behind > 0 &&
		!state.OnProtectedBranch
}

// R035 - Merged Branches Ready for Cleanup
func R035(state model.RepoState) bool {
	return len(state.MergedBranches) > 0
}

// R036 - Gone Remote Branches
func R036(state model.RepoState) bool {
	return len(state.GoneBranches) > 0
}

// R033 - Existing Merge History
func R033(state model.RepoState) bool {
	return state.HasMergeCommits &&
		state.Behind > 0
}
