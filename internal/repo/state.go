package repo

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/pkg/model"
)

// CollectState gathers the current repository state
func CollectState(cfg *config.Config) (model.RepoState, error) {
	state := model.RepoState{}

	// Check if we're in a git repo
	if err := runGitCommand("git", "rev-parse", "--git-dir"); err != nil {
		return state, fmt.Errorf("not a git repository")
	}

	// Get working tree status
	if err := collectWorkingTreeStatus(&state); err != nil {
		return state, err
	}

	// Get branch status
	if err := collectBranchStatus(&state); err != nil {
		return state, err
	}

	// Get stash status
	if err := collectStashStatus(&state); err != nil {
		return state, err
	}

	// Get detached HEAD status
	if err := collectDetachedHeadStatus(&state); err != nil {
		return state, err
	}

	// Get protected branch status
	if err := collectProtectedBranchStatus(&state, cfg); err != nil {
		return state, err
	}

	// Get push status
	if err := collectPushStatus(&state); err != nil {
		return state, err
	}

	// Get merge commit status
	if err := collectMergeCommitStatus(&state); err != nil {
		return state, err
	}

	// Get active operation status (R9-R11)
	if err := collectActiveOperations(&state); err != nil {
		return state, err
	}

	// Get branch health status (R34-R36)
	if err := collectBranchHealth(&state, cfg); err != nil {
		return state, err
	}

	// Get dangerous operation status (R037-R041)
	if err := collectDangerousOperations(&state, cfg); err != nil {
		return state, err
	}

	// Get repo integrity status (R042-R046)
	if err := collectRepoIntegrity(&state); err != nil {
		return state, err
	}

	// Get workflow hygiene status (R047-R051)
	if err := collectWorkflowHygiene(&state, cfg); err != nil {
		return state, err
	}

	// Get mild suggestion status (R052-R055)
	if err := collectMildSuggestions(&state); err != nil {
		return state, err
	}

	// Get informational status (R056-R058)
	if err := collectInformational(&state); err != nil {
		return state, err
	}

	return state, nil
}

func collectWorkingTreeStatus(state *model.RepoState) error {
	output, err := gitOutput("git", "status", "--porcelain")
	if err != nil {
		return err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}

	for _, line := range lines {
		if len(line) < 2 {
			continue
		}

		indexStatus := line[0]
		workTreeStatus := line[1]

		// Staged files (index modified)
		if indexStatus != ' ' && indexStatus != '?' {
			state.StagedFiles++
		}

		// Modified files (work tree modified)
		if workTreeStatus == 'M' {
			state.ModifiedFiles++
		}

		// Untracked files
		if indexStatus == '?' && workTreeStatus == '?' {
			state.UntrackedFiles++
		}
	}

	state.Dirty = state.StagedFiles > 0 || state.ModifiedFiles > 0 || state.UntrackedFiles > 0

	return nil
}

func collectBranchStatus(state *model.RepoState) error {
	output, err := gitOutput("git", "status", "--branch", "--porcelain")
	if err != nil {
		return err
	}

	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return nil
	}

	branchLine := lines[0]
	if !strings.HasPrefix(branchLine, "## ") {
		return nil
	}

	// Parse ahead/behind counts
	if strings.Contains(branchLine, "[") {
		parts := strings.Split(branchLine, "[")
		if len(parts) > 1 {
			tracking := strings.TrimSuffix(parts[1], "]")

			// Parse "ahead N, behind M" or "ahead N" or "behind M"
			if strings.Contains(tracking, "ahead") {
				aheadParts := strings.Split(tracking, "ahead ")
				if len(aheadParts) > 1 {
					numStr := strings.Fields(aheadParts[1])[0]
					numStr = strings.TrimSuffix(numStr, ",")
					if ahead, err := strconv.Atoi(numStr); err == nil {
						state.Ahead = ahead
					}
				}
			}

			if strings.Contains(tracking, "behind") {
				behindParts := strings.Split(tracking, "behind ")
				if len(behindParts) > 1 {
					numStr := strings.Fields(behindParts[1])[0]
					if behind, err := strconv.Atoi(numStr); err == nil {
						state.Behind = behind
					}
				}
			}
		}
	}

	return nil
}

func collectStashStatus(state *model.RepoState) error {
	output, err := gitOutput("git", "stash", "list")
	if err != nil {
		return err
	}

	state.HasStash = strings.TrimSpace(output) != ""
	return nil
}

func collectDetachedHeadStatus(state *model.RepoState) error {
	_, err := gitOutput("git", "symbolic-ref", "HEAD")
	state.OnDetachedHead = err != nil
	return nil
}

func collectProtectedBranchStatus(state *model.RepoState, cfg *config.Config) error {
	branch, err := gitOutput("git", "branch", "--show-current")
	if err != nil {
		return err
	}

	branch = strings.TrimSpace(branch)

	for _, protected := range cfg.ProtectedBranches {
		if branch == protected {
			state.OnProtectedBranch = true
			return nil
		}
	}

	return nil
}

func collectPushStatus(state *model.RepoState) error {
	// Check if HEAD has been pushed to remote
	_, err := gitOutput("git", "rev-parse", "@{u}")
	if err != nil {
		// No upstream configured
		state.LastCommitPushed = false
		state.CommitCountSincePush = 0
		return nil
	}

	// Check if HEAD exists on remote
	headHash, err := gitOutput("git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	headHash = strings.TrimSpace(headHash)

	remoteHash, err := gitOutput("git", "rev-parse", "@{u}")
	if err != nil {
		return err
	}
	remoteHash = strings.TrimSpace(remoteHash)

	// Count commits since last push
	countOutput, err := gitOutput("git", "rev-list", "--count", "@{u}..HEAD")
	if err == nil {
		count, _ := strconv.Atoi(strings.TrimSpace(countOutput))
		state.CommitCountSincePush = count
	}

	// Check if current HEAD is pushed
	_, err = gitOutput("git", "branch", "-r", "--contains", headHash)
	state.LastCommitPushed = err == nil

	return nil
}

func collectMergeCommitStatus(state *model.RepoState) error {
	// Check if there are any merge commits in recent history
	output, err := gitOutput("git", "log", "--merges", "--oneline", "-n", "10")
	if err != nil {
		return err
	}

	state.HasMergeCommits = strings.TrimSpace(output) != ""
	return nil
}

func gitOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func runGitCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// collectActiveOperations detects ongoing git operations (R9-R11)
func collectActiveOperations(state *model.RepoState) error {
	// Get git directory path
	gitDir, err := gitOutput("git", "rev-parse", "--git-dir")
	if err != nil {
		return err
	}
	gitDir = strings.TrimSpace(gitDir)

	// Check for merge in progress
	mergeHeadPath := filepath.Join(gitDir, "MERGE_HEAD")
	if fileExists(mergeHeadPath) {
		state.MergeInProgress = true
	}

	// Check for rebase in progress
	rebaseMergePath := filepath.Join(gitDir, "rebase-merge")
	rebaseApplyPath := filepath.Join(gitDir, "rebase-apply")
	if dirExists(rebaseMergePath) || dirExists(rebaseApplyPath) {
		state.RebaseInProgress = true
	}

	// Check for cherry-pick in progress
	cherryPickHeadPath := filepath.Join(gitDir, "CHERRY_PICK_HEAD")
	if fileExists(cherryPickHeadPath) {
		state.CherryPickInProgress = true
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// collectBranchHealth checks branch tracking and cleanup opportunities (R34-R36)
func collectBranchHealth(state *model.RepoState, cfg *config.Config) error {
	// Skip if on detached HEAD
	if state.OnDetachedHead {
		return nil
	}

	// R034: Check if current branch has upstream
	_, err := gitOutput("git", "rev-parse", "--abbrev-ref", "@{u}")
	state.NoUpstream = err != nil

	// R035: Find merged branches (exclude current and protected branches)
	currentBranch, err := gitOutput("git", "branch", "--show-current")
	if err != nil {
		return err
	}
	currentBranch = strings.TrimSpace(currentBranch)

	mergedOutput, err := gitOutput("git", "branch", "--merged")
	if err == nil {
		lines := strings.Split(strings.TrimSpace(mergedOutput), "\n")

		// Build protected branches map from config
		protectedBranches := make(map[string]bool)
		for _, branch := range cfg.ProtectedBranches {
			protectedBranches[branch] = true
		}

		for _, line := range lines {
			branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
			branch = strings.TrimSpace(branch)

			// Exclude current branch and protected branches
			if branch != "" && branch != currentBranch && !protectedBranches[branch] {
				state.MergedBranches = append(state.MergedBranches, branch)
			}
		}
	}

	// R036: Find gone branches (remote deleted but local remains)
	branchOutput, err := gitOutput("git", "branch", "-vv")
	if err == nil {
		lines := strings.Split(strings.TrimSpace(branchOutput), "\n")
		for _, line := range lines {
			if strings.Contains(line, ": gone]") {
				// Extract branch name (first field after optional *)
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					branch := parts[0]
					if branch == "*" && len(parts) >= 3 {
						branch = parts[1]
					}
					branch = strings.TrimPrefix(branch, "*")
					branch = strings.TrimSpace(branch)
					if branch != "" {
						state.GoneBranches = append(state.GoneBranches, branch)
					}
				}
			}
		}
	}

	return nil
}
