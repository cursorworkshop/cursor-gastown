// Package council provides multi-model orchestration for Gas Town.
// The Council layer routes tasks to optimal models based on role and complexity.
package council

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the Gas Town Council configuration.
// This defines role-model mappings and routing rules.
type Config struct {
	// Version is the schema version.
	Version int `json:"version" toml:"version"`

	// Roles maps Gas Town roles to their model configurations.
	Roles map[string]*RoleConfig `json:"roles" toml:"roles"`

	// Defaults contains default settings.
	Defaults *DefaultConfig `json:"defaults,omitempty" toml:"defaults"`

	// Providers contains provider-specific settings.
	Providers map[string]*ProviderConfig `json:"providers,omitempty" toml:"providers"`
}

// RoleConfig defines the model configuration for a Gas Town role.
type RoleConfig struct {
	// Model is the primary model for this role.
	Model string `json:"model" toml:"model"`

	// Fallback is a list of fallback models if the primary is unavailable.
	Fallback []string `json:"fallback,omitempty" toml:"fallback"`

	// Rationale explains why this model was chosen for the role.
	Rationale string `json:"rationale,omitempty" toml:"rationale"`

	// ComplexityRouting enables routing based on task complexity.
	ComplexityRouting bool `json:"complexity_routing,omitempty" toml:"complexity_routing"`

	// Complexity defines model selection based on task complexity.
	Complexity *ComplexityConfig `json:"complexity,omitempty" toml:"complexity"`

	// Provider overrides the default provider detection.
	Provider string `json:"provider,omitempty" toml:"provider"`
}

// ComplexityConfig defines models for different complexity levels.
type ComplexityConfig struct {
	// High complexity tasks (multi-file, architectural changes).
	High string `json:"high" toml:"high"`

	// Medium complexity tasks (single file, moderate changes).
	Medium string `json:"medium" toml:"medium"`

	// Low complexity tasks (small changes, simple fixes).
	Low string `json:"low" toml:"low"`
}

// DefaultConfig contains default Council settings.
type DefaultConfig struct {
	// Model is the default model when no role-specific config exists.
	Model string `json:"model" toml:"model"`

	// Provider is the default provider.
	Provider string `json:"provider,omitempty" toml:"provider"`

	// Fallback is the default fallback chain.
	Fallback []string `json:"fallback,omitempty" toml:"fallback"`
}

// ProviderConfig contains provider-specific settings.
type ProviderConfig struct {
	// Enabled indicates if this provider is available.
	Enabled bool `json:"enabled" toml:"enabled"`

	// RateLimit is the rate limit in requests per minute.
	RateLimit int `json:"rate_limit,omitempty" toml:"rate_limit"`

	// Priority is used for fallback ordering (higher = preferred).
	Priority int `json:"priority,omitempty" toml:"priority"`

	// Models lists available models from this provider.
	Models []string `json:"models,omitempty" toml:"models"`
}

// CurrentConfigVersion is the current schema version.
const CurrentConfigVersion = 1

// DefaultCouncilConfig returns the default Gas Town Council configuration.
// This implements the role-model matrix from the Multi-Model Orchestration design.
func DefaultCouncilConfig() *Config {
	return &Config{
		Version: CurrentConfigVersion,
		Roles: map[string]*RoleConfig{
			"mayor": {
				Model:     "opus-4.5-thinking",
				Fallback:  []string{"sonnet-4.5", "gpt-5.2-high"},
				Rationale: "Strategic coordination requires sustained reasoning",
			},
			"polecat": {
				Model:             "sonnet-4.5",
				Fallback:          []string{"gpt-5.2", "gemini-3-flash"},
				Rationale:         "Best coding model for multi-file tasks",
				ComplexityRouting: true,
				Complexity: &ComplexityConfig{
					High:   "opus-4.5",
					Medium: "sonnet-4.5",
					Low:    "gemini-3-flash",
				},
			},
			"refinery": {
				Model:     "gpt-5.2-high",
				Fallback:  []string{"opus-4.5", "sonnet-4.5"},
				Rationale: "Different model family provides fresh perspective on code review",
			},
			"witness": {
				Model:     "gemini-3-flash",
				Fallback:  []string{"sonnet-4.5", "gpt-5.2"},
				Rationale: "Fast, cost-effective monitoring",
			},
			"deacon": {
				Model:     "gemini-3-flash",
				Fallback:  []string{"sonnet-4.5"},
				Rationale: "Lightweight lifecycle management",
			},
			"crew": {
				Model:     "auto",
				Rationale: "User preference for interactive work",
			},
		},
		Defaults: &DefaultConfig{
			Model:    "sonnet-4.5",
			Fallback: []string{"gpt-5.2", "gemini-3-flash"},
		},
		Providers: map[string]*ProviderConfig{
			"anthropic": {
				Enabled:   true,
				Priority:  100,
				RateLimit: 60,
				Models:    []string{"opus-4.5-thinking", "opus-4.5", "sonnet-4.5", "sonnet-4.5-thinking"},
			},
			"openai": {
				Enabled:   true,
				Priority:  90,
				RateLimit: 60,
				Models:    []string{"gpt-5.2", "gpt-5.2-high", "gpt-5.1-codex-max", "o4-mini"},
			},
			"google": {
				Enabled:   true,
				Priority:  80,
				RateLimit: 60,
				Models:    []string{"gemini-3-pro", "gemini-3-flash"},
			},
		},
	}
}

// ConfigFileName is the default filename for council configuration.
const ConfigFileName = "council.toml"

// ConfigPath returns the path to the council configuration file.
// By default, it's stored in .beads/council.toml in the town root.
func ConfigPath(townRoot string) string {
	return filepath.Join(townRoot, ".beads", ConfigFileName)
}

// AlternateConfigPath returns an alternate config path in settings/.
func AlternateConfigPath(townRoot string) string {
	return filepath.Join(townRoot, "settings", ConfigFileName)
}

// LoadConfig loads council configuration from the given path.
// Supports both TOML and JSON formats based on file extension.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultCouncilConfig(), nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	config := &Config{}

	// Parse based on extension
	ext := filepath.Ext(path)
	switch ext {
	case ".toml":
		if _, err := toml.Decode(string(data), config); err != nil {
			return nil, fmt.Errorf("parsing TOML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("parsing JSON config: %w", err)
		}
	default:
		// Try TOML first, then JSON
		if _, err := toml.Decode(string(data), config); err != nil {
			if err := json.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("parsing config (tried TOML and JSON): %w", err)
			}
		}
	}

	// Apply defaults if missing
	if config.Version == 0 {
		config.Version = CurrentConfigVersion
	}
	if config.Roles == nil {
		config.Roles = make(map[string]*RoleConfig)
	}

	return config, nil
}

// SaveConfig saves council configuration to the given path.
// Saves as TOML for human readability.
func SaveConfig(path string, config *Config) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Encode as TOML
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	return nil
}

// LoadOrCreate loads config from the path, creating default if it doesn't exist.
func LoadOrCreate(townRoot string) (*Config, error) {
	path := ConfigPath(townRoot)

	// Check primary path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Check alternate path
		altPath := AlternateConfigPath(townRoot)
		if _, err := os.Stat(altPath); err == nil {
			return LoadConfig(altPath)
		}

		// Create default config
		config := DefaultCouncilConfig()
		if err := SaveConfig(path, config); err != nil {
			return nil, fmt.Errorf("saving default config: %w", err)
		}
		return config, nil
	}

	return LoadConfig(path)
}

// GetModelForRole returns the configured model for a role.
func (c *Config) GetModelForRole(role string) string {
	if rc, ok := c.Roles[role]; ok && rc.Model != "" {
		return rc.Model
	}
	if c.Defaults != nil && c.Defaults.Model != "" {
		return c.Defaults.Model
	}
	return "auto"
}

// GetFallbackChain returns the fallback models for a role.
func (c *Config) GetFallbackChain(role string) []string {
	if rc, ok := c.Roles[role]; ok && len(rc.Fallback) > 0 {
		return rc.Fallback
	}
	if c.Defaults != nil {
		return c.Defaults.Fallback
	}
	return nil
}

// GetRationale returns the rationale for a role's model selection.
func (c *Config) GetRationale(role string) string {
	if rc, ok := c.Roles[role]; ok {
		return rc.Rationale
	}
	return ""
}

// SupportsComplexityRouting checks if a role supports complexity-based routing.
func (c *Config) SupportsComplexityRouting(role string) bool {
	if rc, ok := c.Roles[role]; ok {
		return rc.ComplexityRouting && rc.Complexity != nil
	}
	return false
}

// GetModelForComplexity returns the model for a given complexity level.
func (c *Config) GetModelForComplexity(role string, complexity ComplexityLevel) string {
	rc, ok := c.Roles[role]
	if !ok || !rc.ComplexityRouting || rc.Complexity == nil {
		return c.GetModelForRole(role)
	}

	switch complexity {
	case ComplexityHigh:
		if rc.Complexity.High != "" {
			return rc.Complexity.High
		}
	case ComplexityMedium:
		if rc.Complexity.Medium != "" {
			return rc.Complexity.Medium
		}
	case ComplexityLow:
		if rc.Complexity.Low != "" {
			return rc.Complexity.Low
		}
	}

	return c.GetModelForRole(role)
}

// ComplexityLevel represents task complexity.
type ComplexityLevel int

const (
	ComplexityLow ComplexityLevel = iota
	ComplexityMedium
	ComplexityHigh
)

// String returns the string representation of a complexity level.
func (c ComplexityLevel) String() string {
	switch c {
	case ComplexityLow:
		return "low"
	case ComplexityMedium:
		return "medium"
	case ComplexityHigh:
		return "high"
	default:
		return "unknown"
	}
}

// ParseComplexity parses a complexity level from string.
func ParseComplexity(s string) ComplexityLevel {
	switch s {
	case "high":
		return ComplexityHigh
	case "medium":
		return ComplexityMedium
	case "low":
		return ComplexityLow
	default:
		return ComplexityMedium
	}
}
