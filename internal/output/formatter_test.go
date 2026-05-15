package output_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/internal/output"
	"github.com/VectorSophie/git-next/pkg/model"
)

func makeAdvice(id, cmd, desc string, priority int, suppressed bool, severity string) model.Advice {
	return model.Advice{
		RuleID:      id,
		Command:     cmd,
		Description: desc,
		Priority:    priority,
		Suppressed:  suppressed,
		Severity:    severity,
	}
}

func TestFormatHumanCleanRepo(t *testing.T) {
	result := output.FormatHuman(nil, false, config.Defaults(), false, "info")
	if !strings.Contains(result, "clean") {
		t.Errorf("expected 'clean' in output for no advice, got: %q", result)
	}
}

func TestFormatHumanShowsActiveAdvice(t *testing.T) {
	advice := []model.Advice{
		makeAdvice("R003", "git commit", "Staged files waiting for commit", 38, false, "medium"),
	}
	result := output.FormatHuman(advice, false, config.Defaults(), false, "info")
	if !strings.Contains(result, "R003") {
		t.Errorf("expected R003 in output, got: %q", result)
	}
	if !strings.Contains(result, "git commit") {
		t.Errorf("expected command in output, got: %q", result)
	}
}

func TestFormatHumanHidesSuppressedByDefault(t *testing.T) {
	advice := []model.Advice{
		makeAdvice("R003", "git commit", "Staged files", 38, false, "medium"),
		makeAdvice("R004", "git push", "Push commits", 50, true, "medium"),
	}
	result := output.FormatHuman(advice, false, config.Defaults(), false, "info")
	if strings.Contains(result, "R004") {
		t.Error("suppressed advice should be hidden by default")
	}
}

func TestFormatHumanShowsSuppressedWithFlag(t *testing.T) {
	advice := []model.Advice{
		makeAdvice("R003", "git commit", "Staged files", 38, false, "medium"),
		makeAdvice("R004", "git push", "Push commits", 50, true, "medium"),
	}
	result := output.FormatHuman(advice, true, config.Defaults(), false, "info")
	if !strings.Contains(result, "R004") {
		t.Error("suppressed advice should be visible with showSuppressed=true")
	}
}

func TestFormatHumanHidesLowBelowMinSeverity(t *testing.T) {
	advice := []model.Advice{
		makeAdvice("R003", "git commit", "Staged files", 38, false, "medium"),
		makeAdvice("R007", "git add", "Untracked files", 20, false, "low"),
	}
	result := output.FormatHuman(advice, false, config.Defaults(), false, "medium")
	if !strings.Contains(result, "R003") {
		t.Error("expected medium-severity R003 to be visible")
	}
	if strings.Contains(result, "R007") {
		t.Error("low-severity R007 should be hidden when minSeverity=medium")
	}
	if !strings.Contains(result, "hint") {
		t.Error("expected hidden-hint footer when low rules are filtered")
	}
}

func TestFormatCompactCleanRepo(t *testing.T) {
	result := output.FormatCompact(nil)
	if !strings.Contains(result, "clean") {
		t.Errorf("expected 'clean' for empty advice, got: %q", result)
	}
}

func TestFormatCompactListsActiveRules(t *testing.T) {
	advice := []model.Advice{
		makeAdvice("R003", "git commit", "Staged files", 38, false, "medium"),
		makeAdvice("R004", "git push", "Push commits", 50, false, "medium"),
		makeAdvice("R007", "git add", "Untracked", 20, true, "low"), // suppressed, should be excluded
	}
	result := output.FormatCompact(advice)
	if !strings.Contains(result, "R003") {
		t.Errorf("expected R003 in compact output, got: %q", result)
	}
	if !strings.Contains(result, "R004") {
		t.Errorf("expected R004 in compact output, got: %q", result)
	}
	if strings.Contains(result, "R007") {
		t.Error("suppressed R007 should not appear in compact output")
	}
}

func TestFormatJSONStructure(t *testing.T) {
	advice := []model.Advice{
		makeAdvice("R003", "git commit", "Staged files", 38, false, "medium"),
	}
	result, err := output.FormatJSON(advice)
	if err != nil {
		t.Fatalf("FormatJSON returned error: %v", err)
	}

	var parsed struct {
		Advice []model.Advice `json:"advice"`
		Stats  struct {
			Total      int `json:"total"`
			Active     int `json:"active"`
			Suppressed int `json:"suppressed"`
		} `json:"stats"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, result)
	}

	if parsed.Stats.Total != 1 {
		t.Errorf("expected total=1, got %d", parsed.Stats.Total)
	}
	if parsed.Stats.Active != 1 {
		t.Errorf("expected active=1, got %d", parsed.Stats.Active)
	}
	if len(parsed.Advice) == 0 || parsed.Advice[0].RuleID != "R003" {
		t.Error("expected R003 in JSON advice array")
	}
}

func TestFormatJSONIncludesSeverity(t *testing.T) {
	advice := []model.Advice{
		makeAdvice("R037", "# warning", "Force push", 100, false, "critical"),
	}
	result, err := output.FormatJSON(advice)
	if err != nil {
		t.Fatalf("FormatJSON returned error: %v", err)
	}
	if !strings.Contains(result, `"critical"`) {
		t.Errorf("expected severity in JSON output, got: %s", result)
	}
}

func TestFormatJSONEmptyAdvice(t *testing.T) {
	result, err := output.FormatJSON(nil)
	if err != nil {
		t.Fatalf("FormatJSON with nil advice returned error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON for empty advice: %v", err)
	}
}
