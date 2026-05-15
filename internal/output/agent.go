package output

import (
	"encoding/json"
	"strings"

	"github.com/VectorSophie/git-next/pkg/model"
)

// ruleBlockedCommands maps rule IDs to the git commands they prohibit.
var ruleBlockedCommands = map[string][]string{
	"R037": {"git push --force", "git push -f"},
	"R038": {"git tag -f"},
	"R039": {"git reset --hard", "git reset"},
	"R041": {"git push --force", "git push -f", "git rebase", "git filter-branch"},
	"R009": {"git push", "git pull", "git commit", "git rebase", "git checkout", "git switch", "git reset"},
	"R010": {"git push", "git pull", "git commit", "git merge", "git checkout", "git switch", "git reset"},
	"R011": {"git push", "git pull", "git commit", "git rebase", "git merge", "git checkout", "git switch"},
	"R001": {"git commit", "git push"},
	"R042": {"git commit", "git push"},
}

// baselineSafe commands that are always safe to run.
var baselineSafe = []string{
	"git status",
	"git log",
	"git log --oneline",
	"git diff",
	"git diff --staged",
	"git show",
	"git fetch",
	"git fetch --prune",
	"git remote -v",
	"git branch -a",
	"git tag -l",
	"git stash list",
	"git describe",
}

// baselineConditional commands that are safe when not blocked by an active rule.
var baselineConditional = []string{
	"git add",
	"git commit",
	"git push",
	"git pull",
	"git merge",
	"git rebase",
	"git checkout",
	"git switch",
	"git stash",
}

// severityToStatus maps the highest active severity to a status string.
var severityToStatus = map[string]string{
	"critical": "critical",
	"high":     "unsafe",
	"medium":   "caution",
	"low":      "caution",
	"info":     "caution",
}

// severityOrder for comparison (higher index = higher severity).
var severityOrder = []string{"info", "low", "medium", "high", "critical"}

func severityRank(s string) int {
	for i, v := range severityOrder {
		if v == s {
			return i
		}
	}
	return -1
}

// FormatAgent produces the machine-readable agent JSON output.
func FormatAgent(advice []model.Advice) (string, error) {
	active := filterActive(advice)

	highestSeverity := "safe"
	var triggered []model.AgentRule
	var summaryParts []string
	blocked := map[string]bool{}

	for _, a := range active {
		if severityRank(a.Severity) > severityRank(highestSeverity) {
			highestSeverity = a.Severity
		}
		triggered = append(triggered, model.AgentRule{
			ID:          a.RuleID,
			Severity:    a.Severity,
			Description: a.Description,
			Destructive: a.Destructive,
		})
		summaryParts = append(summaryParts, a.Description)

		for _, cmd := range ruleBlockedCommands[a.RuleID] {
			blocked[cmd] = true
		}
	}

	status := "safe"
	riskLevel := "safe"
	if highestSeverity != "safe" {
		status = severityToStatus[highestSeverity]
		riskLevel = highestSeverity
	}

	// Build allowed list: baseline safe + conditional minus blocked.
	allowedSet := map[string]bool{}
	for _, cmd := range baselineSafe {
		allowedSet[cmd] = true
	}
	for _, cmd := range baselineConditional {
		if !blocked[cmd] {
			allowedSet[cmd] = true
		}
	}
	allowed := sortedKeys(allowedSet)

	blockedList := sortedKeys(blocked)

	// Recommended next command: first active rule whose command isn't a comment.
	var recommended string
	for _, a := range active {
		if !strings.HasPrefix(a.Command, "#") && !strings.Contains(a.Command, " OR ") {
			recommended = a.Command
			break
		}
	}

	summary := strings.Join(summaryParts, "; ")
	if summary == "" {
		summary = "Repository is clean."
	}

	out := model.AgentOutput{
		SchemaVersion:          "1",
		Status:                 status,
		RiskLevel:              riskLevel,
		Summary:                summary,
		RecommendedNextCommand: recommended,
		AllowedCommands:        allowed,
		BlockedCommands:        blockedList,
		RulesTriggered:         triggered,
	}
	if out.RulesTriggered == nil {
		out.RulesTriggered = []model.AgentRule{}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func filterActive(advice []model.Advice) []model.Advice {
	var out []model.Advice
	for _, a := range advice {
		if !a.Suppressed {
			out = append(out, a)
		}
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// deterministic order
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
