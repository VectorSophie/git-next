package rules

import (
	"github.com/VectorSophie/git-next/pkg/model"
)

// InformationalRules returns rules for priority <10: Informational trivia
func InformationalRules() []RuleDef {
	return []RuleDef{
		{
			ID:          "R056",
			Check:       R056,
			Command:     "git gc --aggressive",
			Description: "Repo size growing unusually fast - just so you're aware",
			Priority:    9,
		},
		{
			ID:          "R057",
			Check:       R057,
			Command:     "git branch -d <branch>",
			Description: "Inactive branches detected - archaeology opportunity",
			Priority:    8,
		},
		{
			ID:          "R058",
			Check:       R058,
			Command:     "git checkout <branch>",
			Description: "Detached HEAD but clean - nothing wrong, just vibes",
			Priority:    5,
		},
	}
}

// Rule check functions for informational trivia (<10)

// R056 - Repo size growing fast
func R056(state model.RepoState) bool {
	return state.RepoSizeGrowingFast
}

// R057 - Inactive branches detected
func R057(state model.RepoState) bool {
	return len(state.InactiveBranches) > 0
}

// R058 - Detached HEAD but clean
func R058(state model.RepoState) bool {
	return state.OnDetachedHeadClean
}
