package rules

import (
	"github.com/VectorSophie/git-next/pkg/model"
)

// SuggestionRules returns rules for priority 29-10: Mild suggestions
func SuggestionRules() []RuleDef {
	return []RuleDef{
		{
			ID:          "R052",
			Check:       R052,
			Command:     "git commit --amend",
			Description: "Commit message quality warning - Git logs are for humans, allegedly",
			Priority:    25,
		},
		{
			ID:          "R053",
			Check:       R053,
			Command:     "git commit --amend",
			Description: "Amend last commit suggested - you knew this already",
			Priority:    23,
		},
		{
			ID:          "R054",
			Check:       R054,
			Command:     "git push --tags",
			Description: "Unpushed local tags - Schrödinger's release",
			Priority:    21,
		},
		{
			ID:          "R007",
			Check:       R007,
			Command:     "git add <files>",
			Description: "Untracked files present",
			Priority:    20,
		},
		{
			ID:          "R055",
			Check:       R055,
			Command:     "git stash pop OR git stash clear",
			Description: "Stash stack growing - you're hoarding unfinished thoughts",
			Priority:    18,
		},
		{
			ID:          "R008",
			Check:       R008,
			Command:     "git stash pop",
			Description: "Stash exists - consider applying",
			Priority:    15,
		},
	}
}

// Rule check functions for mild suggestions (29-10)

// R052 - Commit message quality warning
func R052(state model.RepoState) bool {
	return state.PoorCommitMessage
}

// R053 - Amend last commit suggested
func R053(state model.RepoState) bool {
	return state.AmendLastCommitSuggested
}

// R054 - Unpushed local tags
func R054(state model.RepoState) bool {
	return state.UnpushedLocalTags
}

// R007 - Untracked Files
func R007(state model.RepoState) bool {
	return state.UntrackedFiles > 0
}

// R055 - Stash stack growing
func R055(state model.RepoState) bool {
	return state.StashStackGrowing
}

// R008 - Stash Exists
func R008(state model.RepoState) bool {
	return state.HasStash
}
