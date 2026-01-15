// Package cmd provides CLI commands for the gt tool.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/council"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

var councilCmd = &cobra.Command{
	Use:     "council",
	GroupID: GroupConfig,
	Short:   "Multi-model orchestration configuration",
	Long: `Manage Gas Town Council configuration for multi-model orchestration.

The Council layer enables role-based model routing, allowing different
Gas Town roles (Mayor, Polecat, Refinery, etc.) to use models best
suited for their specific tasks.

Commands:
  gt council show                    Show current council configuration
  gt council role <role>             Show model configuration for a role
  gt council set <role> <model>      Set model for a role
  gt council fallback <role> <model> Add fallback model for a role
  gt council providers               List provider availability
  gt council route <role>            Test routing decision for a role`,
	RunE: requireSubcommand,
}

// Council subcommands

var councilShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show council configuration",
	Long: `Show the current Gas Town Council configuration.

Displays the role-model matrix, default settings, and provider configuration.

Examples:
  gt council show
  gt council show --json`,
	RunE: runCouncilShow,
}

var councilRoleCmd = &cobra.Command{
	Use:   "role <role>",
	Short: "Show model configuration for a role",
	Long: `Show the model configuration for a specific Gas Town role.

Displays the primary model, fallback chain, complexity routing settings,
and rationale for the role's model selection.

Examples:
  gt council role mayor
  gt council role polecat
  gt council role refinery`,
	Args: cobra.ExactArgs(1),
	RunE: runCouncilRole,
}

var councilSetCmd = &cobra.Command{
	Use:   "set <role> <model>",
	Short: "Set model for a role",
	Long: `Set the primary model for a Gas Town role.

This updates the council configuration to use the specified model
for the given role. The change takes effect for new sessions.

Available models:
  - opus-4.5-thinking, opus-4.5, sonnet-4.5, sonnet-4.5-thinking (Anthropic)
  - gpt-5.2, gpt-5.2-high, gpt-5.1-codex-max, o4-mini (OpenAI)
  - gemini-3-pro, gemini-3-flash (Google)
  - auto (use Cursor's default)

Examples:
  gt council set mayor opus-4.5-thinking
  gt council set polecat sonnet-4.5
  gt council set refinery gpt-5.2-high`,
	Args: cobra.ExactArgs(2),
	RunE: runCouncilSet,
}

var councilFallbackCmd = &cobra.Command{
	Use:   "fallback <role> <models...>",
	Short: "Set fallback models for a role",
	Long: `Set the fallback model chain for a Gas Town role.

When the primary model is unavailable (rate limited, provider down),
the council will try fallback models in order.

Examples:
  gt council fallback mayor sonnet-4.5 gpt-5.2-high
  gt council fallback polecat gpt-5.2 gemini-3-flash`,
	Args: cobra.MinimumNArgs(2),
	RunE: runCouncilFallback,
}

var councilProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List provider availability",
	Long: `Show the availability status of model providers.

Displays which providers are enabled, their priority for fallback,
and any rate limiting or availability issues.

Examples:
  gt council providers
  gt council providers --json`,
	RunE: runCouncilProviders,
}

var councilRouteCmd = &cobra.Command{
	Use:   "route <role>",
	Short: "Test routing decision",
	Long: `Test the model routing decision for a role.

Shows which model would be selected for the given role,
including any fallback decisions and the rationale.

Examples:
  gt council route mayor
  gt council route polecat --complexity high
  gt council route refinery`,
	Args: cobra.ExactArgs(1),
	RunE: runCouncilRoute,
}

var councilInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize council configuration",
	Long: `Initialize council configuration with defaults.

Creates the council.toml configuration file with the recommended
role-model matrix for Gas Town multi-model orchestration.

Examples:
  gt council init
  gt council init --force  # Overwrite existing config`,
	RunE: runCouncilInit,
}

// Flags
var (
	councilShowJSON     bool
	councilRouteComplex string
	councilInitForce    bool
)

func runCouncilShow(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("finding town root: %w", err)
	}

	config, err := council.LoadOrCreate(townRoot)
	if err != nil {
		return fmt.Errorf("loading council config: %w", err)
	}

	if councilShowJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(config)
	}

	// Text output
	fmt.Printf("%s\n\n", style.Bold.Render("Gas Town Council Configuration"))

	// Role-Model Matrix
	fmt.Printf("%s\n", style.Bold.Render("Role-Model Matrix:"))
	roles := make([]string, 0, len(config.Roles))
	for role := range config.Roles {
		roles = append(roles, role)
	}
	sort.Strings(roles)

	for _, role := range roles {
		rc := config.Roles[role]
		fmt.Printf("  %-10s %s", style.Bold.Render(role+":"), rc.Model)
		if len(rc.Fallback) > 0 {
			fmt.Printf(" %s", style.Dim.Render("(fallback: "+strings.Join(rc.Fallback, ", ")+")"))
		}
		fmt.Println()
		if rc.Rationale != "" {
			fmt.Printf("             %s\n", style.Dim.Render(rc.Rationale))
		}
	}

	// Defaults
	if config.Defaults != nil {
		fmt.Printf("\n%s\n", style.Bold.Render("Defaults:"))
		fmt.Printf("  Model:    %s\n", config.Defaults.Model)
		if len(config.Defaults.Fallback) > 0 {
			fmt.Printf("  Fallback: %s\n", strings.Join(config.Defaults.Fallback, ", "))
		}
	}

	// Providers
	if len(config.Providers) > 0 {
		fmt.Printf("\n%s\n", style.Bold.Render("Providers:"))
		for name, pc := range config.Providers {
		status := style.Success.Render("enabled")
		if !pc.Enabled {
			status = style.Error.Render("disabled")
		}
			fmt.Printf("  %-10s %s (priority: %d)\n", name+":", status, pc.Priority)
		}
	}

	return nil
}

func runCouncilRole(cmd *cobra.Command, args []string) error {
	role := args[0]

	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("finding town root: %w", err)
	}

	config, err := council.LoadOrCreate(townRoot)
	if err != nil {
		return fmt.Errorf("loading council config: %w", err)
	}

	rc, ok := config.Roles[role]
	if !ok {
		return fmt.Errorf("unknown role: %s (known roles: %s)",
			role, strings.Join(getKnownRoles(config), ", "))
	}

	fmt.Printf("%s\n\n", style.Bold.Render("Role: "+role))
	fmt.Printf("Model:     %s\n", rc.Model)

	if len(rc.Fallback) > 0 {
		fmt.Printf("Fallback:  %s\n", strings.Join(rc.Fallback, " -> "))
	}

	if rc.Rationale != "" {
		fmt.Printf("Rationale: %s\n", rc.Rationale)
	}

	if rc.ComplexityRouting && rc.Complexity != nil {
		fmt.Printf("\n%s\n", style.Bold.Render("Complexity Routing:"))
		fmt.Printf("  High:   %s\n", rc.Complexity.High)
		fmt.Printf("  Medium: %s\n", rc.Complexity.Medium)
		fmt.Printf("  Low:    %s\n", rc.Complexity.Low)
	}

	return nil
}

func runCouncilSet(cmd *cobra.Command, args []string) error {
	role := args[0]
	model := args[1]

	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("finding town root: %w", err)
	}

	config, err := council.LoadOrCreate(townRoot)
	if err != nil {
		return fmt.Errorf("loading council config: %w", err)
	}

	// Initialize role config if needed
	if config.Roles == nil {
		config.Roles = make(map[string]*council.RoleConfig)
	}
	if config.Roles[role] == nil {
		config.Roles[role] = &council.RoleConfig{}
	}

	// Set model
	config.Roles[role].Model = model

	// Save config
	configPath := council.ConfigPath(townRoot)
	if err := council.SaveConfig(configPath, config); err != nil {
		return fmt.Errorf("saving council config: %w", err)
	}

	fmt.Printf("Set %s model to %s\n", style.Bold.Render(role), style.Bold.Render(model))
	return nil
}

func runCouncilFallback(cmd *cobra.Command, args []string) error {
	role := args[0]
	fallbacks := args[1:]

	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("finding town root: %w", err)
	}

	config, err := council.LoadOrCreate(townRoot)
	if err != nil {
		return fmt.Errorf("loading council config: %w", err)
	}

	// Initialize role config if needed
	if config.Roles == nil {
		config.Roles = make(map[string]*council.RoleConfig)
	}
	if config.Roles[role] == nil {
		config.Roles[role] = &council.RoleConfig{}
	}

	// Set fallbacks
	config.Roles[role].Fallback = fallbacks

	// Save config
	configPath := council.ConfigPath(townRoot)
	if err := council.SaveConfig(configPath, config); err != nil {
		return fmt.Errorf("saving council config: %w", err)
	}

	fmt.Printf("Set %s fallback chain: %s\n", style.Bold.Render(role), strings.Join(fallbacks, " -> "))
	return nil
}

func runCouncilProviders(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("finding town root: %w", err)
	}

	config, err := council.LoadOrCreate(townRoot)
	if err != nil {
		return fmt.Errorf("loading council config: %w", err)
	}

	if councilShowJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(config.Providers)
	}

	fmt.Printf("%s\n\n", style.Bold.Render("Model Providers"))

	// Sort by priority
	type providerInfo struct {
		name string
		cfg  *council.ProviderConfig
	}
	var providers []providerInfo
	for name, cfg := range config.Providers {
		providers = append(providers, providerInfo{name: name, cfg: cfg})
	}
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].cfg.Priority > providers[j].cfg.Priority
	})

	for _, p := range providers {
		status := style.Success.Render("enabled")
		if !p.cfg.Enabled {
			status = style.Error.Render("disabled")
		}

		fmt.Printf("  %s %s\n", style.Bold.Render(p.name+":"), status)
		fmt.Printf("    Priority:   %d\n", p.cfg.Priority)
		fmt.Printf("    Rate Limit: %d req/min\n", p.cfg.RateLimit)
		if len(p.cfg.Models) > 0 {
			fmt.Printf("    Models:     %s\n", strings.Join(p.cfg.Models, ", "))
		}
	}

	return nil
}

func runCouncilRoute(cmd *cobra.Command, args []string) error {
	role := args[0]

	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("finding town root: %w", err)
	}

	config, err := council.LoadOrCreate(townRoot)
	if err != nil {
		return fmt.Errorf("loading council config: %w", err)
	}

	router := council.NewRouter(config)

	// Build request
	req := &council.RouteRequest{Role: role}

	// Add complexity if specified
	if councilRouteComplex != "" {
		complexity := council.ParseComplexity(councilRouteComplex)
		req.Task = &council.TaskInfo{
			FilesAffected: int(complexity) * 5,
			LinesChanged:  int(complexity) * 200,
		}
	}

	result, err := router.Route(req)
	if err != nil {
		return fmt.Errorf("routing failed: %w", err)
	}

	fmt.Printf("%s\n\n", style.Bold.Render("Routing Decision"))
	fmt.Printf("Role:       %s\n", role)
	fmt.Printf("Model:      %s\n", style.Bold.Render(result.Model))
	fmt.Printf("Provider:   %s\n", result.Provider)
	fmt.Printf("Complexity: %s\n", result.Complexity)

	if result.Rationale != "" {
		fmt.Printf("Rationale:  %s\n", result.Rationale)
	}

	if result.Fallback {
		fmt.Printf("\n%s %s\n", style.Warning.Render("Fallback:"), result.FallbackReason)
	}

	return nil
}

func runCouncilInit(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("finding town root: %w", err)
	}

	configPath := council.ConfigPath(townRoot)

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil && !councilInitForce {
		return fmt.Errorf("council config already exists at %s (use --force to overwrite)", configPath)
	}

	// Create default config
	config := council.DefaultCouncilConfig()

	if err := council.SaveConfig(configPath, config); err != nil {
		return fmt.Errorf("saving council config: %w", err)
	}

	fmt.Printf("Created council configuration at %s\n", configPath)
	fmt.Printf("\nRun 'gt council show' to view the configuration.\n")

	return nil
}

func getKnownRoles(config *council.Config) []string {
	roles := make([]string, 0, len(config.Roles))
	for role := range config.Roles {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func init() {
	// Add flags
	councilShowCmd.Flags().BoolVar(&councilShowJSON, "json", false, "Output as JSON")
	councilProvidersCmd.Flags().BoolVar(&councilShowJSON, "json", false, "Output as JSON")
	councilRouteCmd.Flags().StringVar(&councilRouteComplex, "complexity", "", "Task complexity (low, medium, high)")
	councilInitCmd.Flags().BoolVar(&councilInitForce, "force", false, "Overwrite existing config")

	// Add subcommands
	councilCmd.AddCommand(councilShowCmd)
	councilCmd.AddCommand(councilRoleCmd)
	councilCmd.AddCommand(councilSetCmd)
	councilCmd.AddCommand(councilFallbackCmd)
	councilCmd.AddCommand(councilProvidersCmd)
	councilCmd.AddCommand(councilRouteCmd)
	councilCmd.AddCommand(councilInitCmd)

	// Register with root
	rootCmd.AddCommand(councilCmd)
}
