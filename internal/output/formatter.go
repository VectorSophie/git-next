package output

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/internal/explain"
	"github.com/VectorSophie/git-next/pkg/model"
)

// severityAtLeast returns true when a is at least as severe as min.
// Delegates to the severityRank function already defined in agent.go.
func severityAtLeast(a, min string) bool {
	return severityRank(a) >= severityRank(min)
}

// FormatHuman returns human-readable output.
//
// showSuppressed: include suppressed rules in the listing.
// showExplain: print a "why" line under each active rule.
// minSeverity: hide rules below this level (show count in footer instead).
//   - "medium" is the default (hides low/info noise)
//   - "info" shows everything
func FormatHuman(advice []model.Advice, showSuppressed bool, cfg *config.Config, showExplain bool, minSeverity string) string {
	if minSeverity == "" {
		minSeverity = "medium"
	}

	var sb strings.Builder

	if len(advice) == 0 {
		sb.WriteString("✓ Repository is clean. No actions needed.\n")
		return sb.String()
	}

	sb.WriteString("Git Next - Suggested Actions\n")
	sb.WriteString("═══════════════════════════════\n\n")

	activeCount := 0
	suppressedCount := 0
	hiddenCount := 0 // active but below minSeverity

	for _, a := range advice {
		if a.Suppressed {
			suppressedCount++
			if !showSuppressed {
				continue
			}
		} else {
			activeCount++
			if !severityAtLeast(a.Severity, minSeverity) {
				hiddenCount++
				if !showSuppressed {
					continue
				}
			}
		}

		if a.Suppressed {
			sb.WriteString(fmt.Sprintf("  [%s] (suppressed)\n", a.RuleID))
			sb.WriteString(fmt.Sprintf("  %s\n", a.Description))
			sb.WriteString(fmt.Sprintf("  Reason: %s\n\n", a.Reason))
		} else {
			sb.WriteString(fmt.Sprintf("→ [%s] %s\n", a.RuleID, a.Description))

			if a.RuleID == "R035" || a.RuleID == "R036" {
				branches := getBranchList(a.RuleID, cfg)
				if len(branches) > 0 {
					sb.WriteString(fmt.Sprintf("  Branches: %s\n", strings.Join(branches, ", ")))
				}
			}

			sb.WriteString(fmt.Sprintf("  Command: %s\n", a.Command))

			if showExplain {
				if ex, ok := explain.Lookup(a.RuleID); ok {
					why := strings.SplitN(strings.TrimSpace(ex.Why), "\n", 2)[0]
					sb.WriteString(fmt.Sprintf("  Why: %s\n", why))
				}
			}

			sb.WriteString("\n")
		}
	}

	sb.WriteString("───────────────────────────────\n")

	visibleActive := activeCount - hiddenCount
	if suppressedCount > 0 && showSuppressed {
		sb.WriteString(fmt.Sprintf("Active: %d  Suppressed: %d\n", visibleActive, suppressedCount))
	} else if suppressedCount > 0 {
		sb.WriteString(fmt.Sprintf("Active: %d  Suppressed: %d\n", visibleActive, suppressedCount))
		sb.WriteString("(Use --all to show suppressed advice)\n")
	} else {
		sb.WriteString(fmt.Sprintf("Total: %d action(s)\n", visibleActive))
	}

	if hiddenCount > 0 && !showSuppressed {
		sb.WriteString(fmt.Sprintf("+ %d low-severity hint(s) hidden  (use --verbose to see)\n", hiddenCount))
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
		cmd := exec.Command("git", "branch", "--merged")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			currentBranch := getCurrentBranch()

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

func getCurrentBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
