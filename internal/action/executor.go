package action

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/yourusername/git-next/pkg/model"
)

// Execute runs the interactive action selector
func Execute(advice []model.Advice) error {
	// Filter out suppressed advice
	activeAdvice := []model.Advice{}
	for _, a := range advice {
		if !a.Suppressed {
			activeAdvice = append(activeAdvice, a)
		}
	}

	if len(activeAdvice) == 0 {
		fmt.Println("✓ Repository is clean. No actions to execute.")
		return nil
	}

	// Display menu
	fmt.Println("Git Next - Interactive Action Mode")
	fmt.Println("═══════════════════════════════════")
	fmt.Println()

	for i, a := range activeAdvice {
		fmt.Printf("%d. [%s] %s\n", i+1, a.RuleID, a.Description)
		fmt.Printf("   Command: %s\n", a.Command)
		fmt.Printf("   Priority: %d\n\n", a.Priority)
	}

	// Prompt for selection
	fmt.Print("Select action to execute (1-")
	fmt.Print(len(activeAdvice))
	fmt.Print(", or 'q' to quit): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)

	// Handle quit
	if input == "q" || input == "Q" {
		fmt.Println("Cancelled.")
		return nil
	}

	// Parse selection
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(activeAdvice) {
		return fmt.Errorf("invalid selection: %s", input)
	}

	selectedAdvice := activeAdvice[selection-1]

	// Prepare command
	cmd := selectedAdvice.Command
	cmd, err = resolveCommand(cmd, reader)
	if err != nil {
		return err
	}

	// Confirm execution
	fmt.Printf("\nAbout to execute: %s\n", cmd)
	fmt.Print("Proceed? (y/N): ")

	confirm, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	// Execute command
	return executeGitCommand(cmd)
}

// resolveCommand handles placeholders in commands
func resolveCommand(cmd string, reader *bufio.Reader) (string, error) {
	// Handle <branch> placeholder
	if strings.Contains(cmd, "<branch>") {
		// Check if this is a branch cleanup command
		if strings.Contains(cmd, "git branch -d") {
			// For branch cleanup, we need to prompt for which branch(es)
			fmt.Print("\nEnter branch name(s) to delete (space-separated): ")
			branches, err := reader.ReadString('\n')
			if err != nil {
				return "", fmt.Errorf("failed to read branch names: %w", err)
			}
			branches = strings.TrimSpace(branches)
			if branches == "" {
				return "", fmt.Errorf("no branches specified")
			}
			cmd = strings.ReplaceAll(cmd, "<branch>", branches)
		} else {
			// For other commands, use current branch
			branch, err := getCurrentBranch()
			if err != nil {
				return "", err
			}
			cmd = strings.ReplaceAll(cmd, "<branch>", branch)
		}
	}

	// Handle <files> placeholder
	if strings.Contains(cmd, "<files>") {
		fmt.Print("\nEnter file pattern (e.g., '.' for all, or specific files): ")
		files, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read files: %w", err)
		}
		files = strings.TrimSpace(files)
		if files == "" {
			files = "."
		}
		cmd = strings.ReplaceAll(cmd, "<files>", files)
	}

	// Handle HEAD~N placeholder
	if strings.Contains(cmd, "HEAD~N") {
		fmt.Print("\nEnter number of commits: ")
		num, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read commit count: %w", err)
		}
		num = strings.TrimSpace(num)
		if num == "" {
			num = "1"
		}
		cmd = strings.ReplaceAll(cmd, "HEAD~N", "HEAD~"+num)
	}

	// Handle OR commands - prompt user to choose
	if strings.Contains(cmd, " OR ") {
		parts := strings.Split(cmd, " OR ")
		fmt.Println("\nMultiple options available:")
		for i, part := range parts {
			fmt.Printf("%d. %s\n", i+1, strings.TrimSpace(part))
		}
		fmt.Printf("Select option (1-%d): ", len(parts))

		choice, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read choice: %w", err)
		}

		choiceNum, err := strconv.Atoi(strings.TrimSpace(choice))
		if err != nil || choiceNum < 1 || choiceNum > len(parts) {
			return "", fmt.Errorf("invalid choice: %s", choice)
		}

		cmd = strings.TrimSpace(parts[choiceNum-1])
	}

	return cmd, nil
}

// getCurrentBranch gets the current git branch name
func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// executeGitCommand runs a git command and streams output
func executeGitCommand(cmdStr string) error {
	fmt.Println("\n───────────────────────────────")
	fmt.Println("Executing...")
	fmt.Println()

	// Parse command into parts
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Handle compound commands with &&
	if strings.Contains(cmdStr, "&&") {
		commands := strings.Split(cmdStr, "&&")
		for _, c := range commands {
			c = strings.TrimSpace(c)
			if err := runSingleCommand(c); err != nil {
				return err
			}
		}
		return nil
	}

	return runSingleCommand(cmdStr)
}

// runSingleCommand executes a single git command
func runSingleCommand(cmdStr string) error {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Command completed successfully")
	return nil
}
