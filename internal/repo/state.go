package repo

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/pkg/model"
)

// CollectState gathers the current repository state.
// Independent collectors run concurrently; a second wave runs after
// they complete because those depend on the first wave's output.
func CollectState(cfg *config.Config) (model.RepoState, error) {
	state := model.RepoState{}

	// Populate the two most-reused values before any goroutine starts.
	// Every other collector that previously called git for these can now
	// read directly from state.GitDir / state.CurrentBranch.
	gitDir, err := gitOutput("git", "rev-parse", "--git-dir")
	if err != nil {
		return state, fmt.Errorf("not a git repository")
	}
	state.GitDir = strings.TrimSpace(gitDir)

	currentBranch, _ := gitOutput("git", "branch", "--show-current")
	state.CurrentBranch = strings.TrimSpace(currentBranch)

	// Protected-branch status is a pure config lookup; do it now so
	// Phase 1 goroutines can read it without a mutex.
	for _, b := range cfg.ProtectedBranches {
		if state.CurrentBranch == b {
			state.OnProtectedBranch = true
			break
		}
	}

	// Phase 1: collectors with no inter-dependencies.
	if err := runConcurrent(&state, []func(*model.RepoState) error{
		collectWorkingTreeStatus,
		collectBranchStatus,
		collectStashStatus,
		collectDetachedHeadStatus,
		collectPushStatus,
		collectMergeCommitStatus,
		collectActiveOperations,
	}); err != nil {
		return state, err
	}

	// Phase 2: collectors that read Phase 1 outputs.
	if err := runConcurrent(&state, []func(*model.RepoState) error{
		func(s *model.RepoState) error { return collectBranchHealth(s, cfg) },
		func(s *model.RepoState) error { return collectDangerousOperations(s, cfg) },
		collectRepoIntegrity,
		func(s *model.RepoState) error { return collectWorkflowHygiene(s, cfg) },
		func(s *model.RepoState) error { return collectMildSuggestions(s, cfg) },
		collectInformational,
	}); err != nil {
		return state, err
	}

	return state, nil
}

// runConcurrent launches each fn in its own goroutine, waits for all to
// finish, and returns the first error encountered (if any).
// Each fn must write only to fields that no other concurrent fn touches.
func runConcurrent(state *model.RepoState, fns []func(*model.RepoState) error) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(fns))

	for _, fn := range fns {
		fn := fn
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(state); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)

	return <-errCh // nil if channel is empty
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

		if indexStatus != ' ' && indexStatus != '?' {
			state.StagedFiles++
		}

		if workTreeStatus == 'M' {
			state.ModifiedFiles++
		}

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

	if strings.Contains(branchLine, "[") {
		parts := strings.Split(branchLine, "[")
		if len(parts) > 1 {
			tracking := strings.TrimSuffix(parts[1], "]")

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

// collectStashStatus detects stash presence, count, and stack health (R055).
// Merging both concerns into one call avoids a redundant "git stash list".
func collectStashStatus(state *model.RepoState) error {
	output, err := gitOutput("git", "stash", "list")
	if err != nil {
		return err
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return nil
	}

	state.HasStash = true
	stashes := strings.Split(output, "\n")
	state.StashCount = len(stashes)

	if state.StashCount > 3 {
		stashDetails, err := gitOutput("git", "stash", "list", "--format=%ct", "--max-count=1")
		if err == nil {
			timestamp, err := strconv.ParseInt(strings.TrimSpace(stashDetails), 10, 64)
			if err == nil {
				ageDays := int(time.Since(time.Unix(timestamp, 0)).Hours() / 24)
				state.OldestStashAgeDays = ageDays
				if ageDays > 7 {
					state.StashStackGrowing = true
				}
			}
		} else {
			state.StashStackGrowing = true
		}
	}

	return nil
}

func collectDetachedHeadStatus(state *model.RepoState) error {
	_, err := gitOutput("git", "symbolic-ref", "HEAD")
	state.OnDetachedHead = err != nil
	return nil
}

func collectPushStatus(state *model.RepoState) error {
	_, err := gitOutput("git", "rev-parse", "@{u}")
	if err != nil {
		state.LastCommitPushed = false
		state.CommitCountSincePush = 0
		return nil
	}

	headHash, err := gitOutput("git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	headHash = strings.TrimSpace(headHash)

	countOutput, err := gitOutput("git", "rev-list", "--count", "@{u}..HEAD")
	if err == nil {
		count, _ := strconv.Atoi(strings.TrimSpace(countOutput))
		state.CommitCountSincePush = count
	}

	_, err = gitOutput("git", "branch", "-r", "--contains", headHash)
	state.LastCommitPushed = err == nil

	return nil
}

func collectMergeCommitStatus(state *model.RepoState) error {
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

// collectActiveOperations detects ongoing git operations (R9-R11).
// Uses state.GitDir to avoid a second "git rev-parse --git-dir" call.
func collectActiveOperations(state *model.RepoState) error {
	gitDir := state.GitDir
	if gitDir == "" {
		var err error
		gitDir, err = gitOutput("git", "rev-parse", "--git-dir")
		if err != nil {
			return err
		}
		gitDir = strings.TrimSpace(gitDir)
	}

	if fileExists(filepath.Join(gitDir, "MERGE_HEAD")) {
		state.MergeInProgress = true
	}

	if dirExists(filepath.Join(gitDir, "rebase-merge")) || dirExists(filepath.Join(gitDir, "rebase-apply")) {
		state.RebaseInProgress = true
	}

	if fileExists(filepath.Join(gitDir, "CHERRY_PICK_HEAD")) {
		state.CherryPickInProgress = true
	}

	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// collectBranchHealth checks branch tracking and cleanup opportunities (R34-R36).
// Uses state.CurrentBranch to avoid a second "git branch --show-current" call.
func collectBranchHealth(state *model.RepoState, cfg *config.Config) error {
	if state.OnDetachedHead {
		return nil
	}

	_, err := gitOutput("git", "rev-parse", "--abbrev-ref", "@{u}")
	state.NoUpstream = err != nil

	currentBranch := state.CurrentBranch

	mergedOutput, err := gitOutput("git", "branch", "--merged")
	if err == nil {
		protectedBranches := make(map[string]bool)
		for _, branch := range cfg.ProtectedBranches {
			protectedBranches[branch] = true
		}

		for _, line := range strings.Split(strings.TrimSpace(mergedOutput), "\n") {
			branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
			branch = strings.TrimSpace(branch)

			if branch != "" && branch != currentBranch && !protectedBranches[branch] {
				state.MergedBranches = append(state.MergedBranches, branch)
			}
		}
	}

	branchOutput, err := gitOutput("git", "branch", "-vv")
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(branchOutput), "\n") {
			if strings.Contains(line, ": gone]") {
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
