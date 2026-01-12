package repo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/VectorSophie/git-next/pkg/model"
)

// collectMildSuggestions detects mild suggestions (R052-R055)
func collectMildSuggestions(state *model.RepoState) error {
	// R052: Poor commit message
	if err := detectPoorCommitMessage(state); err != nil {
		return err
	}

	// R053: Amend last commit suggested
	if err := detectAmendSuggestion(state); err != nil {
		return err
	}

	// R054: Unpushed local tags
	if err := detectUnpushedTags(state); err != nil {
		return err
	}

	// R055: Stash stack growing
	if err := detectStashStack(state); err != nil {
		return err
	}

	return nil
}

// detectPoorCommitMessage checks commit message quality
func detectPoorCommitMessage(state *model.RepoState) error {
	// Only check if there are staged files or recent commits
	if state.StagedFiles == 0 && state.Ahead == 0 {
		return nil
	}

	// Get last commit message
	lastMsg, err := gitOutput("git", "log", "-1", "--format=%s")
	if err != nil {
		return nil
	}

	lastMsg = strings.TrimSpace(lastMsg)
	state.LastCommitMessage = lastMsg

	// Check for poor quality indicators
	if len(lastMsg) < 5 || // Too short
	   lastMsg == "." ||
	   lastMsg == ".." ||
	   !startsWithVerb(lastMsg) {
		state.PoorCommitMessage = true
	}

	return nil
}

// startsWithVerb checks if message starts with a common git verb
func startsWithVerb(msg string) bool {
	verbs := []string{
		"add", "fix", "update", "remove", "delete", "create", "implement",
		"refactor", "improve", "enhance", "optimize", "clean", "bump",
		"merge", "revert", "upgrade", "downgrade", "move", "rename",
	}

	msgLower := strings.ToLower(msg)
	for _, verb := range verbs {
		if strings.HasPrefix(msgLower, verb+" ") || msgLower == verb {
			return true
		}
	}

	return false
}

// detectAmendSuggestion checks if last commit should be amended
func detectAmendSuggestion(state *model.RepoState) error {
	if state.Ahead < 2 || state.StagedFiles == 0 {
		return nil
	}

	// Get last two commit times
	times, err := gitOutput("git", "log", "-2", "--format=%ct")
	if err != nil {
		return nil
	}

	timeLines := strings.Split(strings.TrimSpace(times), "\n")
	if len(timeLines) < 2 {
		return nil
	}

	t1, err1 := strconv.ParseInt(strings.TrimSpace(timeLines[0]), 10, 64)
	t2, err2 := strconv.ParseInt(strings.TrimSpace(timeLines[1]), 10, 64)

	if err1 != nil || err2 != nil {
		return nil
	}

	// If commits are within 5 minutes of each other
	timeDiff := t1 - t2
	if timeDiff < 300 && timeDiff > 0 {
		state.AmendLastCommitSuggested = true
	}

	return nil
}

// detectUnpushedTags checks for local tags not on remote
func detectUnpushedTags(state *model.RepoState) error {
	// Get all local tags
	localTags, err := gitOutput("git", "tag")
	if err != nil || strings.TrimSpace(localTags) == "" {
		return nil
	}

	tags := strings.Split(strings.TrimSpace(localTags), "\n")
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		// Check if tag exists on remote
		_, err := gitOutput("git", "ls-remote", "--tags", "origin", tag)
		if err != nil {
			state.UnpushedLocalTags = true
			state.UnpushedTags = append(state.UnpushedTags, tag)
		}
	}

	return nil
}

// detectStashStack checks for growing stash
func detectStashStack(state *model.RepoState) error {
	// Get stash count
	stashList, err := gitOutput("git", "stash", "list")
	if err != nil || strings.TrimSpace(stashList) == "" {
		return nil
	}

	stashes := strings.Split(strings.TrimSpace(stashList), "\n")
	state.StashCount = len(stashes)

	if len(stashes) > 3 {
		// Get oldest stash age from newest stash entry
		stashDetails, err := gitOutput("git", "stash", "list", "--format=%ct", "--max-count=1")
		if err == nil {
			timestamp, err := strconv.ParseInt(strings.TrimSpace(stashDetails), 10, 64)
			if err == nil {
				age := time.Since(time.Unix(timestamp, 0))
				ageDays := int(age.Hours() / 24)
				state.OldestStashAgeDays = ageDays

				if ageDays > 7 {
					state.StashStackGrowing = true
				}
			}
		} else {
			// Fallback: just check stash count
			state.StashStackGrowing = true
		}
	}

	return nil
}

// collectInformational detects informational status (R056-R058)
func collectInformational(state *model.RepoState) error {
	// R056: Repo size growing fast
	if err := detectRepoSize(state); err != nil {
		return err
	}

	// R057: Inactive branches
	if err := detectInactiveBranches(state); err != nil {
		return err
	}

	// R058: Detached HEAD but clean
	if err := detectDetachedHeadClean(state); err != nil {
		return err
	}

	return nil
}

// detectRepoSize checks repository size
func detectRepoSize(state *model.RepoState) error {
	gitDir, err := gitOutput("git", "rev-parse", "--git-dir")
	if err != nil {
		return nil
	}
	gitDir = strings.TrimSpace(gitDir)

	// Get .git directory size
	var totalSize int64
	err = filepath.Walk(gitDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return nil
	}

	sizeMB := int(totalSize / 1024 / 1024)
	state.RepoSizeMB = sizeMB

	// Flag if > 100MB
	if sizeMB > 100 {
		state.RepoSizeGrowingFast = true
	}

	return nil
}

// detectInactiveBranches finds branches with no recent commits
func detectInactiveBranches(state *model.RepoState) error {
	// Get all local branches
	branches, err := gitOutput("git", "branch", "--format=%(refname:short)")
	if err != nil {
		return nil
	}

	if strings.TrimSpace(branches) == "" {
		return nil
	}

	currentBranch, _ := gitOutput("git", "branch", "--show-current")
	currentBranch = strings.TrimSpace(currentBranch)

	for _, branch := range strings.Split(strings.TrimSpace(branches), "\n") {
		branch = strings.TrimSpace(branch)
		if branch == "" || branch == currentBranch {
			continue
		}

		// Get last commit date on branch
		dateStr, err := gitOutput("git", "log", "-1", "--format=%ct", branch)
		if err != nil {
			continue
		}

		timestamp, err := strconv.ParseInt(strings.TrimSpace(dateStr), 10, 64)
		if err != nil {
			continue
		}

		age := time.Since(time.Unix(timestamp, 0))
		ageDays := int(age.Hours() / 24)

		// If > 90 days old
		if ageDays > 90 {
			state.InactiveBranches = append(state.InactiveBranches, branch)
		}
	}

	return nil
}

// detectDetachedHeadClean checks for detached HEAD with clean working tree
func detectDetachedHeadClean(state *model.RepoState) error {
	if state.OnDetachedHead && !state.Dirty {
		state.OnDetachedHeadClean = true
	}
	return nil
}
