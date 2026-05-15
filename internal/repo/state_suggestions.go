package repo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/pkg/model"
)

// collectMildSuggestions detects mild suggestions (R052-R054).
// R055 (stash stack) is computed inside collectStashStatus to avoid
// a redundant "git stash list" call.
// R054 (unpushed tags) requires a network call and is skipped unless cfg.Network is true.
func collectMildSuggestions(state *model.RepoState, cfg *config.Config) error {
	// R052: Poor commit message
	if err := detectPoorCommitMessage(state); err != nil {
		return err
	}

	// R053: Amend last commit suggested
	if err := detectAmendSuggestion(state); err != nil {
		return err
	}

	// R054: Unpushed local tags (requires network — skip unless --network / network: true)
	if cfg.Network {
		if err := detectUnpushedTags(state); err != nil {
			return err
		}
	}

	return nil
}

// detectPoorCommitMessage checks commit message quality
func detectPoorCommitMessage(state *model.RepoState) error {
	// Only evaluate when commits are ahead and nothing is currently staged.
	// If staged files exist, the relevant advice is "git commit" (R003), not "amend the last message".
	if state.StagedFiles > 0 || state.Ahead == 0 {
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
	if len(lastMsg) < 5 ||
		lastMsg == "." ||
		lastMsg == ".." ||
		!startsWithVerb(lastMsg) {
		state.PoorCommitMessage = true
	}

	return nil
}

// startsWithVerb checks if message starts with a common git verb or conventional commit prefix.
func startsWithVerb(msg string) bool {
	verbs := []string{
		"add", "fix", "update", "remove", "delete", "create", "implement",
		"refactor", "improve", "enhance", "optimize", "clean", "bump",
		"merge", "revert", "upgrade", "downgrade", "move", "rename",
	}
	// Conventional commit types (feat:, fix:, docs:, feat(scope):, etc.)
	conventional := []string{
		"feat", "fix", "docs", "style", "refactor", "perf",
		"test", "build", "ci", "chore", "revert",
	}

	msgLower := strings.ToLower(msg)

	for _, verb := range verbs {
		if strings.HasPrefix(msgLower, verb+" ") || msgLower == verb {
			return true
		}
	}
	for _, prefix := range conventional {
		if strings.HasPrefix(msgLower, prefix+":") ||
			strings.HasPrefix(msgLower, prefix+"(") {
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

// detectUnpushedTags checks for local tags not on remote.
// Uses a single ls-remote call for all tags rather than one per tag.
func detectUnpushedTags(state *model.RepoState) error {
	localTagsOut, err := gitOutput("git", "tag")
	if err != nil || strings.TrimSpace(localTagsOut) == "" {
		return nil
	}

	// Fetch all remote tags in one call to avoid N network round-trips.
	remoteTags := make(map[string]bool)
	if remoteOut, err := gitOutput("git", "ls-remote", "--tags", "origin"); err == nil {
		for _, line := range strings.Split(remoteOut, "\n") {
			parts := strings.Fields(line)
			if len(parts) == 2 && strings.HasPrefix(parts[1], "refs/tags/") {
				name := strings.TrimPrefix(parts[1], "refs/tags/")
				name = strings.TrimSuffix(name, "^{}") // strip annotated-tag deref suffix
				remoteTags[name] = true
			}
		}
	}

	for _, tag := range strings.Split(strings.TrimSpace(localTagsOut), "\n") {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if !remoteTags[tag] {
			state.UnpushedLocalTags = true
			state.UnpushedTags = append(state.UnpushedTags, tag)
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
	gitDir := state.GitDir
	if gitDir == "" {
		return nil
	}

	// Get .git directory size
	var totalSize int64
	walkErr := filepath.Walk(gitDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if walkErr != nil {
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

// detectInactiveBranches finds branches with no recent commits.
// Uses a single for-each-ref call instead of one git-log per branch.
func detectInactiveBranches(state *model.RepoState) error {
	currentBranch := state.CurrentBranch

	// Fetch all branch names + last-committer timestamps in one shot.
	out, err := gitOutput("git", "for-each-ref",
		"--format=%(refname:short) %(committerdate:unix)", "refs/heads/")
	if err != nil || strings.TrimSpace(out) == "" {
		return nil
	}

	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		branch := parts[0]
		if branch == currentBranch {
			continue
		}

		timestamp, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}

		ageDays := int(time.Since(time.Unix(timestamp, 0)).Hours() / 24)
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
