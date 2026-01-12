package model

// RepoState represents the current state of a Git repository
type RepoState struct {
	Dirty                bool
	StagedFiles          int
	ModifiedFiles        int
	UntrackedFiles       int
	Ahead                int
	Behind               int
	HasStash             bool
	OnDetachedHead       bool
	LastCommitPushed     bool
	CommitCountSincePush int
	OnProtectedBranch    bool
	HasMergeCommits      bool

	// Active operations (R9-R11)
	MergeInProgress      bool
	RebaseInProgress     bool
	CherryPickInProgress bool

	// Branch health (R34-R36)
	NoUpstream           bool
	MergedBranches       []string
	GoneBranches         []string

	// Dangerous operations (R037-R041)
	ForcePushToShared       bool
	RewrittenPublishedTags  bool
	ResetOnProtectedBranch  bool
	SubmoduleRewriteNoUpdate bool
	AccidentalHistoryRewrite bool

	// Repo integrity (R042-R046)
	ConflictedFilesStaged    bool
	ConflictedFiles          []string
	LargeBinariesWithoutLFS  bool
	LargeBinaryFiles         []string
	LineEndingConflict       bool
	SubmoduleDetachedHead    bool
	SubmoduleName            string
	ShallowCloneHistoryOps   bool

	// Workflow hygiene (R047-R051)
	WorkOnMainNotFeature     bool
	LongLivedFeatureBranch   bool
	FeatureBranchAgeDays     int
	SquashRecommended        bool
	NoisyCommitCount         int
	WIPCommitOnShared        bool
	WIPCommitMessage         string
	RebaseInsteadOfMerge     bool

	// Mild suggestions (R052-R055)
	PoorCommitMessage        bool
	LastCommitMessage        string
	AmendLastCommitSuggested bool
	UnpushedLocalTags        bool
	UnpushedTags             []string
	StashStackGrowing        bool
	StashCount               int
	OldestStashAgeDays       int

	// Informational (R056-R058)
	RepoSizeGrowingFast      bool
	RepoSizeMB               int
	InactiveBranches         []string
	OnDetachedHeadClean      bool
}

// Advice represents a single piece of actionable advice
type Advice struct {
	RuleID      string
	Command     string
	Description string
	Priority    int
	Suppressed  bool
	Reason      string
}

// ByPriority implements sort.Interface for []Advice based on Priority field
type ByPriority []Advice

func (a ByPriority) Len() int           { return len(a) }
func (a ByPriority) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPriority) Less(i, j int) bool { return a[i].Priority > a[j].Priority }
