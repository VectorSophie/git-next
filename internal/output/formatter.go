package output

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/pkg/model"
)

// FormatHuman returns human-readable output
func FormatHuman(advice []model.Advice, showSuppressed bool, cfg *config.Config) string {
	var sb strings.Builder

	if len(advice) == 0 {
		sb.WriteString("✓ Repository is clean. No actions needed.\n")
		return sb.String()
	}

	sb.WriteString("Git Next - Suggested Actions\n")
	sb.WriteString("═══════════════════════════════\n\n")

	activeCount := 0
	suppressedCount := 0

	for _, a := range advice {
		if a.Suppressed {
			suppressedCount++
			if !showSuppressed {
				continue
			}
		} else {
			activeCount++
		}

		if a.Suppressed {
			sb.WriteString(fmt.Sprintf("  [%s] (suppressed)\n", a.RuleID))
			sb.WriteString(fmt.Sprintf("  %s\n", a.Description))
			sb.WriteString(fmt.Sprintf("  Reason: %s\n\n", a.Reason))
		} else {
			sb.WriteString(fmt.Sprintf("→ [%s] %s\n", a.RuleID, a.Description))

			// Add extra details for branch cleanup rules
			if a.RuleID == "R035" || a.RuleID == "R036" {
				branches := getBranchList(a.RuleID, cfg)
				if len(branches) > 0 {
					sb.WriteString(fmt.Sprintf("  Branches: %s\n", strings.Join(branches, ", ")))
				}
			}

			sb.WriteString(fmt.Sprintf("  Command: %s\n\n", a.Command))
		}
	}

	sb.WriteString("───────────────────────────────\n")
	if suppressedCount > 0 {
		sb.WriteString(fmt.Sprintf("Active: %d  Suppressed: %d\n", activeCount, suppressedCount))
		if !showSuppressed {
			sb.WriteString("(Use --all to show suppressed advice)\n")
		}
	} else {
		sb.WriteString(fmt.Sprintf("Total: %d action(s)\n", activeCount))
	}

	return sb.String()
}

// FormatJSON returns JSON output
func FormatJSON(advice []model.Advice) (string, error) {
	type Output struct {
		Advice []model.Advice `json:"advice"`
		Stats  struct {
			Total      int `json:"total"`
			Active     int `json:"active"`
			Suppressed int `json:"suppressed"`
		} `json:"stats"`
	}

	output := Output{
		Advice: advice,
	}

	output.Stats.Total = len(advice)
	for _, a := range advice {
		if a.Suppressed {
			output.Stats.Suppressed++
		} else {
			output.Stats.Active++
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// FormatCompact returns a compact one-line summary
func FormatCompact(advice []model.Advice) string {
	active := []string{}
	for _, a := range advice {
		if !a.Suppressed {
			active = append(active, a.RuleID)
		}
	}

	if len(active) == 0 {
		return "✓ clean"
	}

	return fmt.Sprintf("→ %s", strings.Join(active, ", "))
}

// getBranchList retrieves branch information for branch cleanup rules
func getBranchList(ruleID string, cfg *config.Config) []string {
	var branches []string

	if ruleID == "R035" {
		// Get merged branches
		cmd := exec.Command("git", "branch", "--merged")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			currentBranch := getCurrentBranch()

			// Build protected branches map from config
			protectedBranches := make(map[string]bool)
			for _, branch := range cfg.ProtectedBranches {
				protectedBranches[branch] = true
			}

			for _, line := range lines {
				branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
				branch = strings.TrimSpace(branch)
				if branch != "" && branch != currentBranch && !protectedBranches[branch] {
					branches = append(branches, branch)
				}
			}
		}
	} else if ruleID == "R036" {
		// Get gone branches
		cmd := exec.Command("git", "branch", "-vv")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
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
							branches = append(branches, branch)
						}
					}
				}
			}
		}
	}

	return branches
}

// getCurrentBranch gets the current git branch name
func getCurrentBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
