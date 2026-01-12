package repo

import (
	"strings"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/pkg/model"
)

// collectDangerousOperations detects dangerous git operations (R037-R041)
func collectDangerousOperations(state *model.RepoState, cfg *config.Config) error {
	// R037: Force-push to shared branch
	if err := detectForcePushToShared(state, cfg); err != nil {
		return err
	}

	// R038: Rewritten published tags
	if err := detectRewrittenTags(state); err != nil {
		return err
	}

	// R039: Reset on protected branch
	if err := detectResetOnProtected(state, cfg); err != nil {
		return err
	}

	// R040: Submodule pointer rewrite without update
	if err := detectSubmoduleRewrite(state); err != nil {
		return err
	}

	// R041: Accidental history rewrite
	if err := detectHistoryRewrite(state); err != nil {
		return err
	}

	return nil
}

// detectForcePushToShared checks if a force push is needed on a shared branch
func detectForcePushToShared(state *model.RepoState, cfg *config.Config) error {
	// Skip if not on a protected/shared branch
	if !state.OnProtectedBranch {
		return nil
	}

	// Check reflog for force-push indicators
	reflog, err := gitOutput("git", "reflog", "-1", "--format=%gs")
	if err != nil {
		return nil
	}

	// Look for rebase, amend, or filter-branch operations
	if strings.Contains(reflog, "rebase") ||
	   strings.Contains(reflog, "amend") ||
	   strings.Contains(reflog, "filter-branch") {
		// Check if HEAD exists on remote
		headHash, err := gitOutput("git", "rev-parse", "HEAD")
		if err != nil {
			return nil
		}
		headHash = strings.TrimSpace(headHash)

		// Check if HEAD is on remote
		_, err = gitOutput("git", "branch", "-r", "--contains", headHash)
		if err != nil {
			// HEAD not on remote = would need force push
			state.ForcePushToShared = true
		}
	}

	return nil
}

// detectRewrittenTags checks for tag rewrites
func detectRewrittenTags(state *model.RepoState) error {
	// Check for tags that were force-pushed
	tags, err := gitOutput("git", "tag")
	if err != nil {
		return nil
	}

	if strings.TrimSpace(tags) == "" {
		return nil
	}

	for _, tag := range strings.Split(strings.TrimSpace(tags), "\n") {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		// Check if local tag differs from remote
		localHash, err := gitOutput("git", "rev-parse", tag)
		if err != nil {
			continue
		}

		remoteHash, err := gitOutput("git", "rev-parse", "origin/"+tag)
		if err != nil {
			continue // Tag doesn't exist on remote
		}

		if strings.TrimSpace(localHash) != strings.TrimSpace(remoteHash) {
			state.RewrittenPublishedTags = true
			break
		}
	}

	return nil
}

// detectResetOnProtected checks for git reset on protected branches
func detectResetOnProtected(state *model.RepoState, cfg *config.Config) error {
	if !state.OnProtectedBranch {
		return nil
	}

	// Check reflog for reset operations
	reflog, err := gitOutput("git", "reflog", "-5", "--format=%gs")
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(reflog), "\n")
	for _, line := range lines {
		if strings.Contains(line, "reset:") {
			state.ResetOnProtectedBranch = true
			break
		}
	}

	return nil
}

// detectSubmoduleRewrite checks for submodule pointer changes without updates
func detectSubmoduleRewrite(state *model.RepoState) error {
	// Check if repo has submodules
	_, err := gitOutput("git", "config", "--file", ".gitmodules", "--get-regexp", "path")
	if err != nil {
		return nil // No submodules
	}

	// Check if any submodules were modified in the index
	status, err := gitOutput("git", "status", "--porcelain")
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(status), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "M ") && strings.Contains(line, "/") {
			// Check if this is a submodule
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Simple heuristic: if it's a directory in git status, might be submodule
				state.SubmoduleRewriteNoUpdate = true
				break
			}
		}
	}

	return nil
}

// detectHistoryRewrite detects accidental history rewrites
func detectHistoryRewrite(state *model.RepoState) error {
	// Check if recent commits were rebased after being pulled by others
	// This is detected by checking reflog for rebase after commits were pushed

	reflog, err := gitOutput("git", "reflog", "-10", "--format=%gs||%gd")
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(reflog), "\n")
	foundPush := false
	for _, line := range lines {
		parts := strings.Split(line, "||")
		if len(parts) < 1 {
			continue
		}

		message := parts[0]

		if strings.Contains(message, "push") {
			foundPush = true
		}

		if foundPush && (strings.Contains(message, "rebase") || strings.Contains(message, "filter-branch")) {
			state.AccidentalHistoryRewrite = true
			break
		}
	}

	return nil
}
