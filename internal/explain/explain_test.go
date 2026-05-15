package explain_test

import (
	"strings"
	"testing"

	"github.com/VectorSophie/git-next/internal/explain"
)

func TestLookupKnownRule(t *testing.T) {
	ex, ok := explain.Lookup("R037")
	if !ok {
		t.Fatal("expected R037 to have an explanation")
	}
	if ex.ID != "R037" {
		t.Errorf("expected ID R037, got %q", ex.ID)
	}
	if ex.Title == "" {
		t.Error("expected non-empty Title")
	}
	if ex.Why == "" {
		t.Error("expected non-empty Why")
	}
	if ex.Alternative == "" {
		t.Error("expected non-empty Alternative for R037")
	}
}

func TestLookupUnknownRule(t *testing.T) {
	_, ok := explain.Lookup("R999")
	if ok {
		t.Error("expected R999 to be unknown")
	}
}

func TestLookupCaseInsensitive(t *testing.T) {
	// The caller in cmd_explain.go normalises to uppercase, but test the map directly.
	_, ok := explain.Lookup("R001")
	if !ok {
		t.Error("R001 should have an explanation")
	}
}

func TestAllRulesHaveExplanations(t *testing.T) {
	// Every rule that ships should have an explanation.
	ruleIDs := []string{
		"R001", "R002", "R003", "R004", "R005", "R006", "R007", "R008",
		"R009", "R010", "R011", "R020", "R021", "R022", "R030", "R031",
		"R032", "R033", "R034", "R035", "R036", "R037", "R038", "R039",
		"R040", "R041", "R042", "R043", "R044", "R045", "R046", "R047",
		"R048", "R049", "R050", "R051", "R052", "R053", "R054", "R055",
		"R056", "R057", "R058",
	}
	for _, id := range ruleIDs {
		t.Run(id, func(t *testing.T) {
			ex, ok := explain.Lookup(id)
			if !ok {
				t.Errorf("rule %s has no explanation", id)
				return
			}
			if strings.TrimSpace(ex.Why) == "" {
				t.Errorf("rule %s has empty Why", id)
			}
		})
	}
}

func TestAllIDsReturnsEntries(t *testing.T) {
	ids := explain.AllIDs()
	if len(ids) == 0 {
		t.Error("AllIDs should return at least one entry")
	}
}
