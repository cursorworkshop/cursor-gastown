// Package council provides multi-model orchestration for Gas Town.
package council

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Profile represents a shareable council configuration profile.
type Profile struct {
	// Metadata
	Name        string    `json:"name" toml:"name"`
	Description string    `json:"description" toml:"description"`
	Author      string    `json:"author" toml:"author"`
	Version     string    `json:"version" toml:"version"`
	CreatedAt   time.Time `json:"created_at" toml:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" toml:"updated_at"`

	// Tags for discovery
	Tags []string `json:"tags,omitempty" toml:"tags"`

	// Use case description
	UseCase string `json:"use_case,omitempty" toml:"use_case"`

	// The actual configuration
	Config *Config `json:"config" toml:"config"`

	// Performance metrics (optional)
	Metrics *ProfileMetrics `json:"metrics,omitempty" toml:"metrics"`
}

// ProfileMetrics contains performance data for a profile.
type ProfileMetrics struct {
	TotalTasks      int     `json:"total_tasks" toml:"total_tasks"`
	SuccessRate     float64 `json:"success_rate" toml:"success_rate"`
	AvgCostPerTask  float64 `json:"avg_cost_per_task" toml:"avg_cost_per_task"`
	CostSavings     float64 `json:"cost_savings_percent" toml:"cost_savings_percent"`
	ReportedIssues  int     `json:"reported_issues" toml:"reported_issues"`
	CommunityRating float64 `json:"community_rating" toml:"community_rating"`
}

// PredefinedProfiles contains built-in profile configurations.
var PredefinedProfiles = map[string]*Profile{
	"cost-optimized": {
		Name:        "cost-optimized",
		Description: "Minimize costs by using cheaper models where possible",
		Author:      "Gas Town",
		Version:     "1.0.0",
		Tags:        []string{"cost", "budget", "efficient"},
		UseCase:     "Teams on a budget who want to maximize output per dollar",
		Config: &Config{
			Version: 1,
			Roles: map[string]*RoleConfig{
				"mayor": {
					Model:     "sonnet-4.5",
					Fallback:  []string{"gpt-5.2", "gemini-3-flash"},
					Rationale: "Sonnet provides good coordination at lower cost than Opus",
				},
				"polecat": {
					Model:     "gemini-3-flash",
					Fallback:  []string{"gpt-5.2", "sonnet-4.5"},
					Rationale: "Flash handles routine coding tasks effectively",
					Complexity: &ComplexityConfig{
						High:   "sonnet-4.5",
						Medium: "gpt-5.2",
						Low:    "gemini-3-flash",
					},
				},
				"refinery": {
					Model:     "gpt-5.2",
					Fallback:  []string{"sonnet-4.5"},
					Rationale: "GPT provides solid code review at moderate cost",
				},
				"witness": {
					Model:     "gemini-3-flash",
					Fallback:  []string{"gpt-5.2"},
					Rationale: "Flash is extremely cost-effective for monitoring",
				},
			},
			Defaults: &DefaultConfig{
				Model:    "gemini-3-flash",
				Fallback: []string{"gpt-5.2", "sonnet-4.5"},
			},
		},
	},

	"quality-focused": {
		Name:        "quality-focused",
		Description: "Maximize output quality using flagship models",
		Author:      "Gas Town",
		Version:     "1.0.0",
		Tags:        []string{"quality", "enterprise", "flagship"},
		UseCase:     "Critical projects where quality matters more than cost",
		Config: &Config{
			Version: 1,
			Roles: map[string]*RoleConfig{
				"mayor": {
					Model:     "opus-4.5-thinking",
					Fallback:  []string{"gpt-5.2-high", "sonnet-4.5"},
					Rationale: "Extended thinking for complex strategic decisions",
				},
				"polecat": {
					Model:     "sonnet-4.5",
					Fallback:  []string{"opus-4.5", "gpt-5.2-high"},
					Rationale: "Best-in-class coding with flagship fallbacks",
					Complexity: &ComplexityConfig{
						High:   "opus-4.5-thinking",
						Medium: "sonnet-4.5",
						Low:    "sonnet-4.5",
					},
				},
				"refinery": {
					Model:     "opus-4.5",
					Fallback:  []string{"gpt-5.2-high", "sonnet-4.5"},
					Rationale: "Flagship model for thorough code review",
				},
				"witness": {
					Model:     "sonnet-4.5",
					Fallback:  []string{"gpt-5.2"},
					Rationale: "More capable monitoring for complex systems",
				},
			},
			Defaults: &DefaultConfig{
				Model:    "sonnet-4.5",
				Fallback: []string{"opus-4.5", "gpt-5.2-high"},
			},
		},
	},

	"balanced": {
		Name:        "balanced",
		Description: "Balance between cost and quality (recommended default)",
		Author:      "Gas Town",
		Version:     "1.0.0",
		Tags:        []string{"balanced", "default", "recommended"},
		UseCase:     "General purpose configuration suitable for most teams",
		Config: &Config{
			Version: 1,
			Roles: map[string]*RoleConfig{
				"mayor": {
					Model:     "opus-4.5-thinking",
					Fallback:  []string{"sonnet-4.5", "gpt-5.2-high"},
					Rationale: "Strategic coordination warrants extended thinking",
				},
				"polecat": {
					Model:     "sonnet-4.5",
					Fallback:  []string{"gpt-5.2", "gemini-3-flash"},
					Rationale: "Best coding model for primary work",
					Complexity: &ComplexityConfig{
						High:   "opus-4.5",
						Medium: "sonnet-4.5",
						Low:    "gemini-3-flash",
					},
				},
				"refinery": {
					Model:     "gpt-5.2-high",
					Fallback:  []string{"opus-4.5", "sonnet-4.5"},
					Rationale: "Different model family for diverse review perspective",
				},
				"witness": {
					Model:     "gemini-3-flash",
					Fallback:  []string{"sonnet-4.5"},
					Rationale: "Cost-effective monitoring with capable fallback",
				},
			},
			Defaults: &DefaultConfig{
				Model:    "sonnet-4.5",
				Fallback: []string{"gpt-5.2", "gemini-3-flash"},
			},
		},
	},

	"anthropic-only": {
		Name:        "anthropic-only",
		Description: "Use only Anthropic models (single-provider setup)",
		Author:      "Gas Town",
		Version:     "1.0.0",
		Tags:        []string{"anthropic", "single-provider", "claude"},
		UseCase:     "Teams with Anthropic API access only",
		Config: &Config{
			Version: 1,
			Roles: map[string]*RoleConfig{
				"mayor": {
					Model:     "opus-4.5-thinking",
					Fallback:  []string{"sonnet-4.5", "haiku-3.5"},
					Rationale: "Flagship Claude for coordination",
				},
				"polecat": {
					Model:     "sonnet-4.5",
					Fallback:  []string{"opus-4.5", "haiku-3.5"},
					Rationale: "Sonnet is best Anthropic model for coding",
					Complexity: &ComplexityConfig{
						High:   "opus-4.5",
						Medium: "sonnet-4.5",
						Low:    "haiku-3.5",
					},
				},
				"refinery": {
					Model:     "opus-4.5",
					Fallback:  []string{"sonnet-4.5"},
					Rationale: "Opus for thorough review",
				},
				"witness": {
					Model:     "haiku-3.5",
					Fallback:  []string{"sonnet-4.5"},
					Rationale: "Haiku is fast and cheap for monitoring",
				},
			},
			Defaults: &DefaultConfig{
				Model:    "sonnet-4.5",
				Fallback: []string{"opus-4.5", "haiku-3.5"},
			},
			Providers: map[string]*ProviderConfig{
				"anthropic": {Enabled: true, Priority: 100},
				"openai":    {Enabled: false},
				"google":    {Enabled: false},
			},
		},
	},

	"openai-only": {
		Name:        "openai-only",
		Description: "Use only OpenAI models (single-provider setup)",
		Author:      "Gas Town",
		Version:     "1.0.0",
		Tags:        []string{"openai", "single-provider", "gpt"},
		UseCase:     "Teams with OpenAI API access only",
		Config: &Config{
			Version: 1,
			Roles: map[string]*RoleConfig{
				"mayor": {
					Model:     "gpt-5.2-high",
					Fallback:  []string{"gpt-5.2", "gpt-4.1"},
					Rationale: "High-capacity GPT for coordination",
				},
				"polecat": {
					Model:     "gpt-5.2",
					Fallback:  []string{"gpt-5.2-high", "gpt-4.1"},
					Rationale: "GPT-5.2 handles coding well",
					Complexity: &ComplexityConfig{
						High:   "gpt-5.2-high",
						Medium: "gpt-5.2",
						Low:    "gpt-4.1",
					},
				},
				"refinery": {
					Model:     "gpt-5.2-high",
					Fallback:  []string{"gpt-5.2"},
					Rationale: "High-capacity for thorough review",
				},
				"witness": {
					Model:     "gpt-4.1",
					Fallback:  []string{"gpt-5.2"},
					Rationale: "Efficient monitoring with newer GPT",
				},
			},
			Defaults: &DefaultConfig{
				Model:    "gpt-5.2",
				Fallback: []string{"gpt-5.2-high", "gpt-4.1"},
			},
			Providers: map[string]*ProviderConfig{
				"anthropic": {Enabled: false},
				"openai":    {Enabled: true, Priority: 100},
				"google":    {Enabled: false},
			},
		},
	},

	"google-only": {
		Name:        "google-only",
		Description: "Use only Google models (single-provider setup)",
		Author:      "Gas Town",
		Version:     "1.0.0",
		Tags:        []string{"google", "single-provider", "gemini"},
		UseCase:     "Teams with Google AI access only",
		Config: &Config{
			Version: 1,
			Roles: map[string]*RoleConfig{
				"mayor": {
					Model:     "gemini-3-ultra",
					Fallback:  []string{"gemini-3-pro", "gemini-3-flash"},
					Rationale: "Ultra for strategic coordination",
				},
				"polecat": {
					Model:     "gemini-3-pro",
					Fallback:  []string{"gemini-3-ultra", "gemini-3-flash"},
					Rationale: "Pro balances capability and cost",
					Complexity: &ComplexityConfig{
						High:   "gemini-3-ultra",
						Medium: "gemini-3-pro",
						Low:    "gemini-3-flash",
					},
				},
				"refinery": {
					Model:     "gemini-3-pro",
					Fallback:  []string{"gemini-3-ultra"},
					Rationale: "Pro for code review",
				},
				"witness": {
					Model:     "gemini-3-flash",
					Fallback:  []string{"gemini-3-pro"},
					Rationale: "Flash is extremely fast and cheap",
				},
			},
			Defaults: &DefaultConfig{
				Model:    "gemini-3-flash",
				Fallback: []string{"gemini-3-pro", "gemini-3-ultra"},
			},
			Providers: map[string]*ProviderConfig{
				"anthropic": {Enabled: false},
				"openai":    {Enabled: false},
				"google":    {Enabled: true, Priority: 100},
			},
		},
	},
}

// ExportProfile exports the current configuration as a shareable profile.
func ExportProfile(cfg *Config, name, description, author string) *Profile {
	return &Profile{
		Name:        name,
		Description: description,
		Author:      author,
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Config:      cfg,
	}
}

// ExportProfileToFile exports a profile to a JSON file.
func ExportProfileToFile(profile *Profile, path string) error {
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling profile: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing profile: %w", err)
	}

	return nil
}

// ImportProfileFromFile imports a profile from a JSON file.
func ImportProfileFromFile(path string) (*Profile, error) {
	data, err := readProfileBytes(path)
	if err != nil {
		return nil, err
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("parsing profile: %w", err)
	}

	return &profile, nil
}

func readProfileBytes(path string) ([]byte, error) {
	if isHTTPURL(path) {
		return fetchProfileFromURL(path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading profile: %w", err)
	}

	return data, nil
}

func isHTTPURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

func fetchProfileFromURL(url string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching profile: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Best-effort; response body is already consumed.
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching profile: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading profile response: %w", err)
	}

	return data, nil
}

// ApplyProfile applies a profile's configuration.
func ApplyProfile(profile *Profile, townRoot string) error {
	if profile.Config == nil {
		return fmt.Errorf("profile has no configuration")
	}

	// Save the configuration
	configPath := filepath.Join(townRoot, ".beads", ConfigFileName)
	return SaveConfig(configPath, profile.Config)
}

// GetProfile returns a predefined profile by name.
func GetProfile(name string) (*Profile, bool) {
	profile, ok := PredefinedProfiles[name]
	return profile, ok
}

// ListProfiles returns all available profile names.
func ListProfiles() []string {
	names := make([]string, 0, len(PredefinedProfiles))
	for name := range PredefinedProfiles {
		names = append(names, name)
	}
	return names
}

// SearchProfiles searches profiles by tag.
func SearchProfiles(tag string) []*Profile {
	tag = strings.ToLower(tag)
	var matches []*Profile

	for _, profile := range PredefinedProfiles {
		for _, t := range profile.Tags {
			if strings.Contains(strings.ToLower(t), tag) {
				matches = append(matches, profile)
				break
			}
		}
	}

	return matches
}

// ValidateProfile validates a profile configuration.
func ValidateProfile(profile *Profile) []string {
	var issues []string

	if profile.Name == "" {
		issues = append(issues, "profile name is required")
	}

	if profile.Config == nil {
		issues = append(issues, "profile configuration is required")
		return issues
	}

	// Validate each role configuration
	for role, cfg := range profile.Config.Roles {
		if cfg.Model == "" {
			issues = append(issues, fmt.Sprintf("role %q has no model specified", role))
		}
	}

	// Validate defaults
	if profile.Config.Defaults == nil {
		issues = append(issues, "profile should have default configuration")
	} else if profile.Config.Defaults.Model == "" {
		issues = append(issues, "default model is required")
	}

	return issues
}
