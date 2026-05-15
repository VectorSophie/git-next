package rules

import (
	"testing"

	"github.com/VectorSophie/git-next/internal/config"
	"github.com/VectorSophie/git-next/pkg/model"
)

// withDefaults wraps a config-taking rule function for table tests.
func withDefaults(fn func(model.RepoState, *config.Config) bool) func(model.RepoState) bool {
	return func(s model.RepoState) bool { return fn(s, config.Defaults()) }
}

type ruleCase struct {
	name   string
	check  func(model.RepoState) bool
	fires  model.RepoState
	noFire model.RepoState
}

func runCases(t *testing.T, cases []ruleCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name+"/fires", func(t *testing.T) {
			if !tc.check(tc.fires) {
				t.Error("expected rule to fire")
			}
		})
		t.Run(tc.name+"/silent", func(t *testing.T) {
			if tc.check(tc.noFire) {
				t.Error("expected rule not to fire")
			}
		})
	}
}

func TestDangerousRuleChecks(t *testing.T) {
	runCases(t, []ruleCase{
		{
			name:   "R001 detached HEAD",
			check:  R001,
			fires:  model.RepoState{OnDetachedHead: true},
			noFire: model.RepoState{OnDetachedHead: false},
		},
		{
			name:   "R009 merge in progress",
			check:  R009,
			fires:  model.RepoState{MergeInProgress: true},
			noFire: model.RepoState{MergeInProgress: false},
		},
		{
			name:   "R010 rebase in progress",
			check:  R010,
			fires:  model.RepoState{RebaseInProgress: true},
			noFire: model.RepoState{RebaseInProgress: false},
		},
		{
			name:   "R011 cherry-pick in progress",
			check:  R011,
			fires:  model.RepoState{CherryPickInProgress: true},
			noFire: model.RepoState{CherryPickInProgress: false},
		},
		{
			name:   "R032 diverged on protected branch",
			check:  R032,
			fires:  model.RepoState{Ahead: 1, Behind: 1, OnProtectedBranch: true},
			noFire: model.RepoState{Ahead: 1, Behind: 0, OnProtectedBranch: true},
		},
		{
			name:   "R037 force push to shared",
			check:  R037,
			fires:  model.RepoState{ForcePushToShared: true},
			noFire: model.RepoState{ForcePushToShared: false},
		},
		{
			name:   "R038 rewritten published tags",
			check:  R038,
			fires:  model.RepoState{RewrittenPublishedTags: true},
			noFire: model.RepoState{RewrittenPublishedTags: false},
		},
		{
			name:   "R039 reset on protected branch",
			check:  R039,
			fires:  model.RepoState{ResetOnProtectedBranch: true},
			noFire: model.RepoState{ResetOnProtectedBranch: false},
		},
		{
			name:   "R040 submodule rewrite without update",
			check:  R040,
			fires:  model.RepoState{SubmoduleRewriteNoUpdate: true},
			noFire: model.RepoState{SubmoduleRewriteNoUpdate: false},
		},
		{
			name:   "R041 accidental history rewrite",
			check:  R041,
			fires:  model.RepoState{AccidentalHistoryRewrite: true},
			noFire: model.RepoState{AccidentalHistoryRewrite: false},
		},
	})
}

func TestIntegrityRuleChecks(t *testing.T) {
	runCases(t, []ruleCase{
		{
			name:   "R006 branch diverged",
			check:  R006,
			fires:  model.RepoState{Ahead: 1, Behind: 1},
			noFire: model.RepoState{Ahead: 1, Behind: 0},
		},
		{
			name:   "R031 feature branch diverged",
			check:  R031,
			fires:  model.RepoState{Ahead: 1, Behind: 1, OnProtectedBranch: false},
			noFire: model.RepoState{Ahead: 1, Behind: 1, OnProtectedBranch: true},
		},
		{
			name:   "R033 merge history exists and behind",
			check:  R033,
			fires:  model.RepoState{HasMergeCommits: true, Behind: 1},
			noFire: model.RepoState{HasMergeCommits: true, Behind: 0},
		},
		{
			name:   "R034 no upstream skips detached HEAD",
			check:  R034,
			fires:  model.RepoState{NoUpstream: true, OnDetachedHead: false},
			noFire: model.RepoState{NoUpstream: true, OnDetachedHead: true},
		},
		{
			name:   "R035 merged branches present",
			check:  R035,
			fires:  model.RepoState{MergedBranches: []string{"old-feature"}},
			noFire: model.RepoState{MergedBranches: nil},
		},
		{
			name:   "R036 gone branches present",
			check:  R036,
			fires:  model.RepoState{GoneBranches: []string{"deleted-remote"}},
			noFire: model.RepoState{GoneBranches: nil},
		},
		{
			name:   "R042 conflicted files staged",
			check:  R042,
			fires:  model.RepoState{ConflictedFilesStaged: true},
			noFire: model.RepoState{ConflictedFilesStaged: false},
		},
		{
			name:   "R043 large binaries without LFS",
			check:  R043,
			fires:  model.RepoState{LargeBinariesWithoutLFS: true},
			noFire: model.RepoState{LargeBinariesWithoutLFS: false},
		},
		{
			name:   "R044 line ending conflict",
			check:  R044,
			fires:  model.RepoState{LineEndingConflict: true},
			noFire: model.RepoState{LineEndingConflict: false},
		},
		{
			name:   "R045 submodule detached HEAD",
			check:  R045,
			fires:  model.RepoState{SubmoduleDetachedHead: true},
			noFire: model.RepoState{SubmoduleDetachedHead: false},
		},
		{
			name:   "R046 shallow clone history ops",
			check:  R046,
			fires:  model.RepoState{ShallowCloneHistoryOps: true},
			noFire: model.RepoState{ShallowCloneHistoryOps: false},
		},
	})
}

func TestWorkflowRuleChecks(t *testing.T) {
	runCases(t, []ruleCase{
		{
			name:   "R002 modified not staged",
			check:  R002,
			fires:  model.RepoState{ModifiedFiles: 1, StagedFiles: 0},
			noFire: model.RepoState{ModifiedFiles: 1, StagedFiles: 1},
		},
		{
			name:   "R003 staged not committed",
			check:  R003,
			fires:  model.RepoState{StagedFiles: 1},
			noFire: model.RepoState{StagedFiles: 0},
		},
		{
			name:   "R004 push local commits",
			check:  R004,
			fires:  model.RepoState{Ahead: 1, Behind: 0, Dirty: false},
			noFire: model.RepoState{Ahead: 0, Behind: 0, Dirty: false},
		},
		{
			name:   "R004 blocked when dirty",
			check:  R004,
			fires:  model.RepoState{Ahead: 1, Behind: 0, Dirty: false},
			noFire: model.RepoState{Ahead: 1, Behind: 0, Dirty: true},
		},
		{
			name:   "R005 pull when behind and clean",
			check:  R005,
			fires:  model.RepoState{Behind: 1, Dirty: false},
			noFire: model.RepoState{Behind: 1, Dirty: true},
		},
		{
			name:   "R020 soft reset within limit",
			check:  withDefaults(R020),
			fires:  model.RepoState{LastCommitPushed: false, CommitCountSincePush: 2},
			noFire: model.RepoState{LastCommitPushed: false, CommitCountSincePush: 5},
		},
		{
			name:   "R020 skips pushed commits",
			check:  withDefaults(R020),
			fires:  model.RepoState{LastCommitPushed: false, CommitCountSincePush: 1},
			noFire: model.RepoState{LastCommitPushed: true, CommitCountSincePush: 1},
		},
		{
			name:   "R022 interactive rebase above threshold",
			check:  withDefaults(R022),
			fires:  model.RepoState{LastCommitPushed: false, CommitCountSincePush: 5},
			noFire: model.RepoState{LastCommitPushed: false, CommitCountSincePush: 2},
		},
		{
			name:   "R030 fast-forward only",
			check:  R030,
			fires:  model.RepoState{Behind: 1, Ahead: 0, Dirty: false},
			noFire: model.RepoState{Behind: 1, Ahead: 1, Dirty: false},
		},
		{
			name:   "R047 work on main",
			check:  R047,
			fires:  model.RepoState{WorkOnMainNotFeature: true},
			noFire: model.RepoState{WorkOnMainNotFeature: false},
		},
		{
			name:   "R048 long-lived feature branch",
			check:  R048,
			fires:  model.RepoState{LongLivedFeatureBranch: true},
			noFire: model.RepoState{LongLivedFeatureBranch: false},
		},
		{
			name:   "R049 squash recommended",
			check:  R049,
			fires:  model.RepoState{SquashRecommended: true},
			noFire: model.RepoState{SquashRecommended: false},
		},
		{
			name:   "R050 WIP commit on shared branch",
			check:  R050,
			fires:  model.RepoState{WIPCommitOnShared: true},
			noFire: model.RepoState{WIPCommitOnShared: false},
		},
		{
			name:   "R051 rebase instead of merge",
			check:  R051,
			fires:  model.RepoState{RebaseInsteadOfMerge: true},
			noFire: model.RepoState{RebaseInsteadOfMerge: false},
		},
	})
}

func TestSuggestionRuleChecks(t *testing.T) {
	runCases(t, []ruleCase{
		{
			name:   "R007 untracked files",
			check:  R007,
			fires:  model.RepoState{UntrackedFiles: 1},
			noFire: model.RepoState{UntrackedFiles: 0},
		},
		{
			name:   "R008 stash exists",
			check:  R008,
			fires:  model.RepoState{HasStash: true},
			noFire: model.RepoState{HasStash: false},
		},
		{
			name:   "R052 poor commit message",
			check:  R052,
			fires:  model.RepoState{PoorCommitMessage: true},
			noFire: model.RepoState{PoorCommitMessage: false},
		},
		{
			name:   "R053 amend last commit",
			check:  R053,
			fires:  model.RepoState{AmendLastCommitSuggested: true},
			noFire: model.RepoState{AmendLastCommitSuggested: false},
		},
		{
			name:   "R054 unpushed local tags",
			check:  R054,
			fires:  model.RepoState{UnpushedLocalTags: true},
			noFire: model.RepoState{UnpushedLocalTags: false},
		},
		{
			name:   "R055 stash stack growing",
			check:  R055,
			fires:  model.RepoState{StashStackGrowing: true},
			noFire: model.RepoState{StashStackGrowing: false},
		},
	})
}

func TestInformationalRuleChecks(t *testing.T) {
	runCases(t, []ruleCase{
		{
			name:   "R056 repo size growing fast",
			check:  R056,
			fires:  model.RepoState{RepoSizeGrowingFast: true},
			noFire: model.RepoState{RepoSizeGrowingFast: false},
		},
		{
			name:   "R057 inactive branches",
			check:  R057,
			fires:  model.RepoState{InactiveBranches: []string{"old-branch"}},
			noFire: model.RepoState{InactiveBranches: nil},
		},
		{
			name:   "R058 detached HEAD clean",
			check:  R058,
			fires:  model.RepoState{OnDetachedHeadClean: true},
			noFire: model.RepoState{OnDetachedHeadClean: false},
		},
	})
}

func TestRuleMetadata(t *testing.T) {
	validSeverities := map[string]bool{
		"critical": true, "high": true, "medium": true, "low": true, "info": true,
	}

	for _, r := range AllRules(config.Defaults()) {
		t.Run(r.ID, func(t *testing.T) {
			if r.ID == "" {
				t.Error("rule has empty ID")
			}
			if r.Check == nil {
				t.Errorf("rule %s has nil Check function", r.ID)
			}
			if r.Severity == "" {
				t.Errorf("rule %s has empty Severity", r.ID)
			}
			if !validSeverities[r.Severity] {
				t.Errorf("rule %s has invalid Severity %q", r.ID, r.Severity)
			}
		})
	}
}
