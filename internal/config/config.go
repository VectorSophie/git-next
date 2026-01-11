package config

// Config represents the git-next configuration
type Config struct {
	ProtectedBranches []string          `yaml:"protected_branches"`
	Rules             RuleConfig        `yaml:"rules"`
	Suppression       SuppressionConfig `yaml:"suppression"`
}

// RuleConfig contains rule-specific configuration
type RuleConfig struct {
	Disabled   []string                       `yaml:"disabled"`
	Parameters map[string]map[string]interface{} `yaml:"parameters"`
}

// SuppressionConfig contains custom suppression rules
type SuppressionConfig struct {
	Custom map[string][]string `yaml:"custom"`
}

// Defaults returns the default configuration
func Defaults() *Config {
	return &Config{
		ProtectedBranches: []string{"main", "master", "develop", "production"},
		Rules: RuleConfig{
			Disabled: []string{},
			Parameters: map[string]map[string]interface{}{
				"R020": {"max_commits": 3},
				"R022": {"min_commits": 4},
			},
		},
		Suppression: SuppressionConfig{
			Custom: make(map[string][]string),
		},
	}
}

// IsRuleDisabled checks if a rule is disabled in the configuration
func (c *Config) IsRuleDisabled(ruleID string) bool {
	for _, disabled := range c.Rules.Disabled {
		if disabled == ruleID {
			return true
		}
	}
	return false
}

// GetIntParam retrieves an integer parameter for a rule with a default fallback
func (c *Config) GetIntParam(ruleID, param string, defaultVal int) int {
	if c.Rules.Parameters[ruleID] != nil {
		if val, ok := c.Rules.Parameters[ruleID][param]; ok {
			switch v := val.(type) {
			case int:
				return v
			case float64:
				return int(v)
			}
		}
	}
	return defaultVal
}

// GetStringParam retrieves a string parameter for a rule with a default fallback
func (c *Config) GetStringParam(ruleID, param string, defaultVal string) string {
	if c.Rules.Parameters[ruleID] != nil {
		if val, ok := c.Rules.Parameters[ruleID][param]; ok {
			if v, ok := val.(string); ok {
				return v
			}
		}
	}
	return defaultVal
}

// MergeWithDefaults merges the config with defaults
func (c *Config) MergeWithDefaults() {
	defaults := Defaults()

	// Merge protected branches if empty
	if len(c.ProtectedBranches) == 0 {
		c.ProtectedBranches = defaults.ProtectedBranches
	}

	// Ensure Rules exists
	if c.Rules.Disabled == nil {
		c.Rules.Disabled = []string{}
	}
	if c.Rules.Parameters == nil {
		c.Rules.Parameters = defaults.Rules.Parameters
	}

	// Ensure Suppression exists
	if c.Suppression.Custom == nil {
		c.Suppression.Custom = make(map[string][]string)
	}
}
