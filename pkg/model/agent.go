package model

// AgentOutput is the machine-readable response for --agent mode.
type AgentOutput struct {
	SchemaVersion          string      `json:"schema_version"`
	Status                 string      `json:"status"`           // "safe" | "caution" | "unsafe" | "critical"
	RiskLevel              string      `json:"risk_level"`       // "safe" | "low" | "medium" | "high" | "critical"
	Summary                string      `json:"summary"`
	RecommendedNextCommand string      `json:"recommended_next_command,omitempty"`
	AllowedCommands        []string    `json:"allowed_commands"`
	BlockedCommands        []string    `json:"blocked_commands"`
	RulesTriggered         []AgentRule `json:"rules_triggered"`
}

// AgentRule is a single triggered rule entry in AgentOutput.
type AgentRule struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Destructive bool   `json:"destructive"`
}

// GuardOutput is the response for the guard subcommand.
type GuardOutput struct {
	Command     string   `json:"command"`
	Allowed     bool     `json:"allowed"`
	RiskLevel   string   `json:"risk_level"` // "safe" | "caution" | "danger"
	Reason      string   `json:"reason,omitempty"`
	Alternative string   `json:"alternative,omitempty"`
	ActiveRules []string `json:"active_rules,omitempty"`
}
