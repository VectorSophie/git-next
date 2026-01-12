package repo

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/VectorSophie/git-next/pkg/model"
)

// collectRepoIntegrity detects repo integrity issues (R042-R046)
func collectRepoIntegrity(state *model.RepoState) error {
	// R042: Conflicted files staged
	if err := detectConflictedStaged(state); err != nil {
		return err
	}

	// R043: Binary files without LFS
	if err := detectLargeBinaries(state); err != nil {
		return err
	}

	// R044: Line ending conflicts
	if err := detectLineEndingConflict(state); err != nil {
		return err
	}

	// R045: Submodule detached HEAD
	if err := detectSubmoduleDetached(state); err != nil {
		return err
	}

	// R046: Shallow clone doing history ops
	if err := detectShallowCloneHistoryOps(state); err != nil {
		return err
	}

	return nil
}

// detectConflictedStaged checks for conflict markers in staged files
func detectConflictedStaged(state *model.RepoState) error {
	// Get staged files
	staged, err := gitOutput("git", "diff", "--cached", "--name-only")
	if err != nil || strings.TrimSpace(staged) == "" {
		return nil
	}

	files := strings.Split(strings.TrimSpace(staged), "\n")
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		// Check file for conflict markers
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		contentStr := string(content)
		if strings.Contains(contentStr, "<<<<<<<") ||
		   strings.Contains(contentStr, "=======") ||
		   strings.Contains(contentStr, ">>>>>>>") {
			state.ConflictedFilesStaged = true
			state.ConflictedFiles = append(state.ConflictedFiles, file)
		}
	}

	return nil
}

// detectLargeBinaries checks for large binary files without LFS
func detectLargeBinaries(state *model.RepoState) error {
	// Check if LFS is installed
	_, lfsErr := gitOutput("git", "lfs", "version")
	hasLFS := lfsErr == nil

	// Get recently added/modified files
	recent, err := gitOutput("git", "diff", "--cached", "--name-only", "--diff-filter=AM")
	if err != nil || strings.TrimSpace(recent) == "" {
		return nil
	}

	files := strings.Split(strings.TrimSpace(recent), "\n")
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		// Check file size
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		// If file > 1MB and binary and not tracked by LFS
		if info.Size() > 1024*1024 {
			// Check if binary
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}

			// Simple binary check: look for null bytes
			isBinary := false
			for _, b := range content[:min(512, len(content))] {
				if b == 0 {
					isBinary = true
					break
				}
			}

			if isBinary {
				// Check if tracked by LFS
				if hasLFS {
					lfsCheck, _ := gitOutput("git", "lfs", "ls-files", file)
					if strings.TrimSpace(lfsCheck) == "" {
						state.LargeBinariesWithoutLFS = true
						state.LargeBinaryFiles = append(state.LargeBinaryFiles, file)
					}
				} else {
					state.LargeBinariesWithoutLFS = true
					state.LargeBinaryFiles = append(state.LargeBinaryFiles, file)
				}
			}
		}
	}

	return nil
}

// detectLineEndingConflict checks for CRLF/LF inconsistencies
func detectLineEndingConflict(state *model.RepoState) error {
	// Check git config for core.autocrlf
	autocrlf, _ := gitOutput("git", "config", "core.autocrlf")
	autocrlf = strings.TrimSpace(autocrlf)

	// Check recent warnings about line endings
	status, err := gitOutput("git", "status")
	if err != nil {
		return nil
	}

	if strings.Contains(status, "CRLF") || strings.Contains(status, "LF") {
		state.LineEndingConflict = true
	}

	return nil
}

// detectSubmoduleDetached checks if submodules are in detached HEAD
func detectSubmoduleDetached(state *model.RepoState) error {
	// Check if repo has submodules
	submodules, err := gitOutput("git", "submodule", "status")
	if err != nil || strings.TrimSpace(submodules) == "" {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(submodules), "\n")
	for _, line := range lines {
		// Detached HEAD submodules start with a space or -
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "-") {
			// Extract submodule name
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				state.SubmoduleDetachedHead = true
				state.SubmoduleName = parts[1]
				break
			}
		}
	}

	return nil
}

// detectShallowCloneHistoryOps checks for history operations on shallow clone
func detectShallowCloneHistoryOps(state *model.RepoState) error {
	// Check if this is a shallow clone
	gitDir, err := gitOutput("git", "rev-parse", "--git-dir")
	if err != nil {
		return nil
	}
	gitDir = strings.TrimSpace(gitDir)

	shallowFile := filepath.Join(gitDir, "shallow")
	if _, err := os.Stat(shallowFile); err != nil {
		return nil // Not shallow
	}

	// Check reflog for history operations
	reflog, err := gitOutput("git", "reflog", "-5", "--format=%gs")
	if err != nil {
		return nil
	}

	historyOps := []string{"rebase", "blame", "bisect", "log --all"}
	for _, line := range strings.Split(strings.TrimSpace(reflog), "\n") {
		for _, op := range historyOps {
			if strings.Contains(line, op) {
				state.ShallowCloneHistoryOps = true
				return nil
			}
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
