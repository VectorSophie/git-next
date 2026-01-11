package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load loads configuration with fallback chain:
// 1. .git-next.yaml in current directory
// 2. ~/.config/git-next/config.yaml
// 3. Defaults
func Load() (*Config, error) {
	// Try .git-next.yaml in current directory
	if configPath := ".git-next.yaml"; fileExists(configPath) {
		cfg, err := LoadFromFile(configPath)
		if err == nil {
			cfg.MergeWithDefaults()
			return cfg, nil
		}
		// If error, log but continue to fallbacks
		fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", configPath, err)
	}

	// Try ~/.config/git-next/config.yaml
	if homeDir, err := os.UserHomeDir(); err == nil {
		configPath := filepath.Join(homeDir, ".config", "git-next", "config.yaml")
		if fileExists(configPath) {
			cfg, err := LoadFromFile(configPath)
			if err == nil {
				cfg.MergeWithDefaults()
				return cfg, nil
			}
			fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", configPath, err)
		}
	}

	// Return defaults
	return Defaults(), nil
}

// LoadFromFile loads configuration from a specific YAML file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &cfg, nil
}

// LoadFromPath loads config from a custom path with fallback to defaults
func LoadFromPath(path string) (*Config, error) {
	if path == "" {
		return Load()
	}

	if !fileExists(path) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	cfg, err := LoadFromFile(path)
	if err != nil {
		return nil, err
	}

	cfg.MergeWithDefaults()
	return cfg, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
