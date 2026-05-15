package output_test

import (
	"encoding/json"
	"testing"

	"github.com/VectorSophie/git-next/internal/output"
	"github.com/VectorSophie/git-next/pkg/model"
)

func TestFormatAgentSafeRepo(t *testing.T) {
	result, err := output.FormatAgent(nil)
	if err != nil {
		t.Fatalf("FormatAgent returned error: %v", err)
	}

	var out model.AgentOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, result)
	}

	if out.Status != "safe" {
		t.Errorf("expected status=safe for no advice, got %q", out.Status)
	}
	if out.RiskLevel != "safe" {
		t.Errorf("expected risk_level=safe, got %q", out.RiskLevel)
	}
	if out.SchemaVersion != "1" {
		t.Errorf("expected schema_version=1, got %q", out.SchemaVersion)
	}
	if len(out.AllowedCommands) == 0 {
		t.Error("expected at least some allowed commands even for safe repo")
	}
}

func TestFormatAgentCriticalRule(t *testing.T) {
	advice := []model.Advice{
		{RuleID: "R037", Command: "# DO NOT push --force", Description: "Force-push to shared branch", Priority: 100, Severity: "critical", Destructive: true},
	}
	result, err := output.FormatAgent(advice)
	if err != nil {
		t.Fatalf("FormatAgent returned error: %v", err)
	}

	var out model.AgentOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if out.Status != "critical" {
		t.Errorf("expected status=critical, got %q", out.Status)
	}
	if out.RiskLevel != "critical" {
		t.Errorf("expected risk_level=critical, got %q", out.RiskLevel)
	}
	if len(out.RulesTriggered) != 1 || out.RulesTriggered[0].ID != "R037" {
		t.Error("expected R037 in rules_triggered")
	}
	if !out.RulesTriggered[0].Destructive {
		t.Error("expected R037 to be marked destructive")
	}
}

func TestFormatAgentSuppressedRuleNotCounted(t *testing.T) {
	advice := []model.Advice{
		{RuleID: "R003", Command: "git commit", Description: "Staged files", Priority: 38, Severity: "medium", Suppressed: false},
		{RuleID: "R004", Command: "git push", Description: "Push commits", Priority: 50, Severity: "medium", Suppressed: true},
	}
	result, err := output.FormatAgent(advice)
	if err != nil {
		t.Fatalf("FormatAgent returned error: %v", err)
	}

	var out model.AgentOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Only R003 should appear (R004 is suppressed)
	if len(out.RulesTriggered) != 1 {
		t.Errorf("expected 1 triggered rule, got %d", len(out.RulesTriggered))
	}
}

func TestFormatAgentBlockedCommandsFromR037(t *testing.T) {
	advice := []model.Advice{
		{RuleID: "R037", Command: "# warning", Description: "Force-push", Priority: 100, Severity: "critical", Destructive: true},
	}
	result, _ := output.FormatAgent(advice)

	var out model.AgentOutput
	json.Unmarshal([]byte(result), &out)

	found := false
	for _, cmd := range out.BlockedCommands {
		if cmd == "git push --force" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'git push --force' in blocked_commands, got: %v", out.BlockedCommands)
	}
}

func TestFormatAgentAlwaysContainsStatusField(t *testing.T) {
	result, _ := output.FormatAgent(nil)
	var raw map[string]interface{}
	json.Unmarshal([]byte(result), &raw)

	for _, field := range []string{"status", "risk_level", "summary", "allowed_commands", "blocked_commands", "rules_triggered", "schema_version"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("expected field %q in agent output", field)
		}
	}
}
