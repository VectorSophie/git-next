package engine

import (
	"sort"
	"strings"

	"github.com/VectorSophie/git-next/internal/rules"
	"github.com/VectorSophie/git-next/pkg/model"
)

// Suppresses maps commands to what they suppress
// These aren't "rules", they're laws.
var Suppresses = map[string][]string{
	// Active operations suppress almost everything
	"merge --continue":      {"merge", "rebase", "reset", "commit", "pull", "push", "checkout"},
	"merge --abort":         {"merge", "rebase", "reset", "commit", "pull", "push", "checkout"},
	"rebase --continue":     {"merge", "rebase", "reset", "commit", "pull", "push", "checkout"},
	"rebase --abort":        {"merge", "rebase", "reset", "commit", "pull", "push", "checkout"},
	"cherry-pick --continue": {"merge", "rebase", "reset", "commit", "pull", "push", "checkout"},
	"cherry-pick --abort":    {"merge", "rebase", "reset", "commit", "pull", "push", "checkout"},

	// Original suppression rules
	"revert": {"reset"},
	"merge":  {"rebase"},
	"rebase": {"pull"},
	"reset":  {"commit"},
}

// Evaluate runs all rules against the repo state and returns advice
func Evaluate(state model.RepoState) []model.Advice {
	allRules := rules.AllRules()
	var advice []model.Advice

	// Evaluate all rules
	for _, ruleDef := range allRules {
		if ruleDef.Check(state) {
			advice = append(advice, model.Advice{
				RuleID:      ruleDef.ID,
				Command:     ruleDef.Command,
				Description: ruleDef.Description,
				Priority:    ruleDef.Priority,
				Suppressed:  false,
				Reason:      "",
			})
		}
	}

	// Sort by priority (highest first)
	sort.Sort(model.ByPriority(advice))

	// Apply suppression rules
	advice = applySuppression(advice)

	return advice
}

// applySuppression applies suppression rules to advice list
func applySuppression(advice []model.Advice) []model.Advice {
	// Track which commands are active (not suppressed)
	activeCommands := make(map[string]bool)

	// First pass: mark active commands
	for i := range advice {
		if !advice[i].Suppressed {
			cmd := extractCommand(advice[i].Command)
			activeCommands[cmd] = true
		}
	}

	// Second pass: suppress based on active commands
	for i := range advice {
		if advice[i].Suppressed {
			continue
		}

		cmd := extractCommand(advice[i].Command)

		// Check if this command suppresses any others
		if suppressedCmds, exists := Suppresses[cmd]; exists {
			// This command is active, so suppress lower-priority instances of suppressed commands
			for j := i + 1; j < len(advice); j++ {
				targetCmd := extractCommand(advice[j].Command)
				for _, suppressedCmd := range suppressedCmds {
					if targetCmd == suppressedCmd {
						advice[j].Suppressed = true
						advice[j].Reason = "Suppressed by " + cmd + " (higher priority)"
						break
					}
				}
			}
		}
	}

	return advice
}

// extractCommand extracts the primary git command from a command string
func extractCommand(cmd string) string {
	// Handle compound commands (e.g., "git add <files> && git commit")
	if strings.Contains(cmd, "&&") {
		parts := strings.Split(cmd, "&&")
		cmd = strings.TrimSpace(parts[len(parts)-1])
	}

	// Handle OR commands (e.g., "git rebase ... OR git merge ...")
	if strings.Contains(cmd, " OR ") {
		parts := strings.Split(cmd, " OR ")
		cmd = strings.TrimSpace(parts[0])
	}

	// Extract the git command and key flags
	// (e.g., "git reset --soft HEAD~N" -> "reset")
	// (e.g., "git merge --continue" -> "merge --continue")
	parts := strings.Fields(cmd)
	if len(parts) >= 2 && parts[0] == "git" {
		command := parts[1]

		// For certain commands, include the flag as part of the command identifier
		if len(parts) >= 3 {
			flag := parts[2]
			if flag == "--continue" || flag == "--abort" {
				return command + " " + flag
			}
		}

		return command
	}

	return ""
}
