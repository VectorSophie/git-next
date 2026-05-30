package mcp

import (
	"strings"

	"github.com/VectorSophie/git-next/internal/policy"
	"github.com/VectorSophie/git-next/pkg/model"
)

// Tier is the safety classification of a proposed git command.
type Tier int

const (
	TierSafe    Tier = iota // Execute immediately
	TierGuarded             // Require confirmed=true before executing
	TierBlocked             // Never execute
)

func (t Tier) String() string {
	switch t {
	case TierSafe:
		return "safe"
	case TierGuarded:
		return "guarded"
	default:
		return "blocked"
	}
}

// Classify returns the execution tier for a proposed command.
// Commands not starting with "git " are always TierBlocked.
// policy.Evaluate is checked first; per-verb rules apply only when policy passes.
func Classify(proposed string, state model.RepoState) Tier {
	normalized := strings.TrimSpace(proposed)
	if !strings.HasPrefix(normalized, "git ") {
		return TierBlocked
	}
	sub := strings.TrimPrefix(normalized, "git ")
	args := strings.Fields(sub)
	if len(args) == 0 {
		return TierBlocked
	}
	verb := args[0]

	if policy.Evaluate(proposed, state) != nil {
		return TierBlocked
	}

	switch verb {
	case "status", "log", "diff", "show", "fetch", "remote",
		"rev-list", "rev-parse", "ls-files", "shortlog",
		"describe", "cherry", "check-ignore":
		return TierSafe

	case "add":
		return TierSafe

	case "commit":
		for _, arg := range args[1:] {
			if arg == "--amend" {
				return TierGuarded
			}
		}
		return TierSafe

	case "branch":
		if len(args) > 1 && args[1] == "-d" {
			return TierGuarded
		}
		return TierSafe

	case "stash":
		if len(args) > 1 {
			switch args[1] {
			case "pop", "drop", "clear":
				return TierGuarded
			}
		}
		return TierSafe

	case "tag":
		if len(args) > 1 {
			switch args[1] {
			case "-l", "--list", "--contains", "--sort", "--format", "--points-at":
				return TierSafe
			}
		}
		return TierGuarded

	case "config":
		for _, arg := range args[1:] {
			if arg == "--global" || arg == "--system" || arg == "--local" {
				return TierGuarded
			}
		}
		return TierSafe

	case "push", "merge", "rebase":
		return TierGuarded
	}

	// Unknown git commands default to guarded
	return TierGuarded
}
