package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const managedHeader = "# Managed by git-next. Remove this line to stop git-next from managing this hook.\n"

// hookScript is the content written by Install.
const hookScript = managedHeader + `#!/bin/sh
git-next ci --fail-on high
`

// Install writes a git-next managed hook to the repository's .git/hooks directory.
func Install(hookName string) error {
	hookPath, err := hookFilePath(hookName)
	if err != nil {
		return err
	}

	// If hook already exists and is not managed by us, refuse to overwrite.
	existing, readErr := os.ReadFile(hookPath)
	if readErr == nil && !isManagedByUs(string(existing)) {
		return fmt.Errorf("hook %s already exists and is not managed by git-next\n"+
			"Remove it manually or add the following line at the top:\n  %s", hookName, strings.TrimRight(managedHeader, "\n"))
	}

	if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
		return fmt.Errorf("writing hook: %w", err)
	}
	fmt.Printf("Installed git-next hook: %s\n", hookPath)
	return nil
}

// Uninstall removes a git-next managed hook. It refuses to remove unmanaged hooks.
func Uninstall(hookName string) error {
	hookPath, err := hookFilePath(hookName)
	if err != nil {
		return err
	}

	existing, err := os.ReadFile(hookPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("no hook found at %s", hookPath)
	}
	if err != nil {
		return fmt.Errorf("reading hook: %w", err)
	}
	if !isManagedByUs(string(existing)) {
		return fmt.Errorf("hook %s was not installed by git-next; refusing to remove it", hookName)
	}

	if err := os.Remove(hookPath); err != nil {
		return fmt.Errorf("removing hook: %w", err)
	}
	fmt.Printf("Removed git-next hook: %s\n", hookPath)
	return nil
}

// List prints the status of known hooks in the current repository.
func List() error {
	gitDir, err := gitDir()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	knownHooks := []string{"pre-commit", "pre-push", "commit-msg", "post-merge"}

	fmt.Println("git-next hook status:")
	for _, name := range knownHooks {
		path := filepath.Join(hooksDir, name)
		content, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			fmt.Printf("  %-14s  not installed\n", name)
			continue
		}
		if isManagedByUs(string(content)) {
			fmt.Printf("  %-14s  managed by git-next\n", name)
		} else {
			fmt.Printf("  %-14s  exists (not managed by git-next)\n", name)
		}
	}
	return nil
}

func hookFilePath(hookName string) (string, error) {
	dir, err := gitDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hooks", hookName), nil
}

func gitDir() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--git-dir").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}

func isManagedByUs(content string) bool {
	return strings.Contains(content, "Managed by git-next")
}
