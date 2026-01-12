package repo

import (
	"strconv"
	"strings"
	"time"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/pkg/model"
)

// collectWorkflowHygiene detects workflow hygiene issues (R047-R051)
func collectWorkflowHygiene(state *model.RepoState, cfg *config.Config) error {
	// R047: Work on main instead of feature branch
	if err := detectWorkOnMain(state, cfg); err != nil {
		return err
	}

	// R048: Long-lived feature branch
	if err := detectLongLivedBranch(state, cfg); err != nil {
		return err
	}

	// R049: Squash recommended
	if err := detectNoisyCommits(state); err != nil {
		return err
	}

	// R050: WIP commit on shared branch
	if err := detectWIPCommit(state, cfg); err != nil {
		return err
	}

	// R051: Rebase instead of merge recommended
	if err := detectRebaseInsteadOfMerge(state, cfg); err != nil {
		return err
	}

	return nil
}

// detectWorkOnMain checks if working directly on protected branches
func detectWorkOnMain(state *model.RepoState, cfg *config.Config) error {
	if !state.OnProtectedBranch {
		return nil
	}

	// Check if there are local commits that aren't merges
	if state.Ahead > 0 {
		// Check if latest commit is a merge
		lastCommit, err := gitOutput("git", "log", "-1", "--format=%s")
		if err != nil {
			return nil
		}

		// If not a merge commit, flag it
		if !strings.HasPrefix(strings.ToLower(lastCommit), "merge") {
			state.WorkOnMainNotFeature = true
		}
	}

	return nil
}

// detectLongLivedBranch checks for feature branches that are too old
func detectLongLivedBranch(state *model.RepoState, cfg *config.Config) error {
	// Skip if on protected branch or detached HEAD
	if state.OnProtectedBranch || state.OnDetachedHead {
		return nil
	}

	// Get branch creation date (first commit on this branch)
	currentBranch, err := gitOutput("git", "branch", "--show-current")
	if err != nil || strings.TrimSpace(currentBranch) == "" {
		return nil
	}

	// Get merge-base with main
	mergeBase, err := gitOutput("git", "merge-base", "HEAD", "origin/main")
	if err != nil {
		// Try master
		mergeBase, err = gitOutput("git", "merge-base", "HEAD", "origin/master")
		if err != nil {
			return nil
		}
	}

	if strings.TrimSpace(mergeBase) == "" {
		return nil
	}

	// Get date of merge-base (when branch diverged)
	dateStr, err := gitOutput("git", "log", "-1", "--format=%ct", strings.TrimSpace(mergeBase))
	if err != nil {
		return nil
	}

	timestamp, err := strconv.ParseInt(strings.TrimSpace(dateStr), 10, 64)
	if err != nil {
		return nil
	}

	branchAge := time.Since(time.Unix(timestamp, 0))
	ageDays := int(branchAge.Hours() / 24)

	// Configurable threshold (default 14 days)
	maxDays := cfg.GetIntParam("R048", "max_days", 14)

	if ageDays > maxDays && state.Behind > 0 {
		state.LongLivedFeatureBranch = true
		state.FeatureBranchAgeDays = ageDays
	}

	return nil
}

// detectNoisyCommits checks for many small "fix" commits
func detectNoisyCommits(state *model.RepoState) error {
	if state.Ahead == 0 {
		return nil
	}

	// Get unpushed commits
	commits, err := gitOutput("git", "log", "--format=%s", "@{u}..HEAD")
	if err != nil {
		return nil
	}

	if strings.TrimSpace(commits) == "" {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(commits), "\n")
	noisyCount := 0

	noisyPatterns := []string{
		"fix", "oops", "wip", "temp", "debug", "test",
		"typo", ".", "update", "change",
	}

	for _, line := range lines {
		lineLower := strings.ToLower(strings.TrimSpace(line))
		for _, pattern := range noisyPatterns {
			if lineLower == pattern || strings.HasPrefix(lineLower, pattern+" ") {
				noisyCount++
				break
			}
		}
	}

	// If > 30% of commits are noisy and there are > 3 commits
	if len(lines) > 3 && float64(noisyCount)/float64(len(lines)) > 0.3 {
		state.SquashRecommended = true
		state.NoisyCommitCount = noisyCount
	}

	return nil
}

// detectWIPCommit checks for WIP commits on shared branches
func detectWIPCommit(state *model.RepoState, cfg *config.Config) error {
	if !state.OnProtectedBranch {
		return nil
	}

	// Check last commit message
	lastMsg, err := gitOutput("git", "log", "-1", "--format=%s")
	if err != nil {
		return nil
	}

	msgLower := strings.ToLower(strings.TrimSpace(lastMsg))
	wipPatterns := []string{"wip", "temp", "debug", "todo", "fixme"}

	for _, pattern := range wipPatterns {
		if strings.Contains(msgLower, pattern) {
			state.WIPCommitOnShared = true
			state.WIPCommitMessage = strings.TrimSpace(lastMsg)
			break
		}
	}

	return nil
}

// detectRebaseInsteadOfMerge checks if rebase would be better than merge
func detectRebaseInsteadOfMerge(state *model.RepoState, cfg *config.Config) error {
	// Skip if on protected branch (protected branches should use merge)
	if state.OnProtectedBranch {
		return nil
	}

	// Check if latest commit is a merge commit
	mergeCheck, err := gitOutput("git", "log", "-1", "--format=%p")
	if err != nil {
		return nil
	}

	// If commit has multiple parents, it's a merge
	parents := strings.Fields(strings.TrimSpace(mergeCheck))
	if len(parents) > 1 {
		// Feature branch with merge commit - suggest rebase instead
		state.RebaseInsteadOfMerge = true
	}

	return nil
}
