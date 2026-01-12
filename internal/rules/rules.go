package rules

import (
	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/pkg/model"
)

// Rule represents a function that evaluates repository state
type Rule func(state model.RepoState) bool

// RuleDef defines a rule with its metadata
type RuleDef struct {
	ID          string
	Check       Rule
	Command     string
	Description string
	Priority    int
}

// AllRules returns all defined rules sorted by priority
// Rules are organized into modules by danger level for better maintainability:
//  - Dangerous (100-90): "Put the keyboard down" - operations that can break things
//  - Integrity (89-60): Repo integrity issues that need immediate attention
//  - Workflow (59-30): Workflow hygiene and best practices
//  - Suggestions (29-10): Mild suggestions and optimizations
//  - Informational (<10): Informational trivia, just FYI
func AllRules(cfg *config.Config) []RuleDef {
	var all []RuleDef

	// 100-90: "Put the keyboard down"
	all = append(all, DangerousRules(cfg)...)

	// 89-60: Repo integrity issues
	all = append(all, IntegrityRules()...)

	// 59-30: Workflow hygiene
	all = append(all, WorkflowRules(cfg)...)

	// 29-10: Mild suggestions
	all = append(all, SuggestionRules()...)

	// <10: Informational trivia
	all = append(all, InformationalRules()...)

	return all
}
