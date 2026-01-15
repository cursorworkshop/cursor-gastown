// Package cmd provides CLI commands for the gt tool.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/council"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/templates"
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

var councilTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Show available role templates",
	Long: `Show available role templates for each provider.

Different model providers respond better to different prompt styles.
This command shows which provider-optimized templates are available
for each Gas Town role.

Examples:
  gt council templates`,
	RunE: runCouncilTemplates,
}

var councilStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show model performance statistics",
	Long: `Show performance statistics for models across roles.

Displays metrics including task counts, success rates, costs,
and model comparisons.

Examples:
  gt council stats
  gt council stats --json`,
	RunE: runCouncilStats,
}

var councilCompareCmd = &cobra.Command{
	Use:   "compare <model1> <model2>",
	Short: "Compare two models",
	Long: `Compare performance metrics between two models.

Shows differences in success rate, average duration, and cost
between the specified models.

Examples:
  gt council compare sonnet-4.5 gpt-5.2
  gt council compare opus-4.5-thinking sonnet-4.5`,
	Args: cobra.ExactArgs(2),
	RunE: runCouncilCompare,
}

var councilChainsCmd = &cobra.Command{
	Use:   "chains",
	Short: "List available chain patterns",
	Long: `Show predefined chain-of-models patterns.

Chains pass output through a sequence of models, where each model
refines or transforms the previous output. This is useful for
complex tasks that benefit from multiple perspectives.

Examples:
  gt council chains
  gt council chains --json`,
	RunE: runCouncilChains,
}

var councilEnsemblesCmd = &cobra.Command{
	Use:   "ensembles",
	Short: "List available ensemble patterns",
	Long: `Show predefined ensemble voting patterns.

Ensembles run multiple models in parallel and combine their outputs
through voting. This provides higher confidence for critical decisions.

Examples:
  gt council ensembles
  gt council ensembles --json`,
	RunE: runCouncilEnsembles,
}

var councilPatternCmd = &cobra.Command{
	Use:   "pattern <name>",
	Short: "Show details of a specific pattern",
	Long: `Show detailed configuration for a chain or ensemble pattern.

Examples:
  gt council pattern code-review
  gt council pattern critical-decision`,
	Args: cobra.ExactArgs(1),
	RunE: runCouncilPattern,
}

// Flags
var (
	councilShowJSON     bool
	councilRouteComplex string
	councilInitForce    bool
	councilStatsJSON    bool
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

func runCouncilTemplates(cmd *cobra.Command, args []string) error {
	tmpl, err := templates.New()
	if err != nil {
		return fmt.Errorf("loading templates: %w", err)
	}

	fmt.Printf("%s\n\n", style.Bold.Render("Provider-Optimized Role Templates"))

	// Get provider templates
	providerTemplates := tmpl.ProviderTemplateNames()

	// Show all roles and their available templates
	roles := tmpl.RoleNames()
	for _, role := range roles {
		providers := providerTemplates[role]
		if len(providers) > 0 {
			fmt.Printf("  %s: default, %s\n", style.Bold.Render(role), strings.Join(providers, ", "))
		} else {
			fmt.Printf("  %s: default only\n", style.Bold.Render(role))
		}
	}

	fmt.Printf("\n%s\n", style.Dim.Render("Provider templates are auto-selected based on council configuration."))
	fmt.Printf("%s\n", style.Dim.Render("OpenAI templates use structured formats; Google templates use explicit grounding."))

	return nil
}

func runCouncilStats(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("finding town root: %w", err)
	}

	store, err := council.NewMetricsStore(townRoot)
	if err != nil {
		return fmt.Errorf("loading metrics: %w", err)
	}

	metrics := store.GetMetrics()
	summary := store.GetSummary()

	if councilStatsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"summary": summary,
			"metrics": metrics,
		})
	}

	// Summary
	fmt.Printf("%s\n\n", style.Bold.Render("Gas Town Council Statistics"))

	fmt.Printf("%s\n", style.Bold.Render("Summary:"))
	fmt.Printf("  Total Tasks:     %d\n", summary.TotalTasks)
	fmt.Printf("  Completed:       %d\n", summary.CompletedTasks)
	fmt.Printf("  Success Rate:    %.1f%%\n", summary.AvgSuccessRate*100)
	fmt.Printf("  Total Cost:      $%.2f\n", summary.TotalCost)
	if summary.CostSavings > 0 {
		fmt.Printf("  Cost Savings:    %.1f%% %s\n", summary.CostSavings, style.Dim.Render("(vs Opus for all)"))
	}
	if summary.TopModel != "" {
		fmt.Printf("  Top Model:       %s\n", summary.TopModel)
	}

	// By Role
	if len(metrics.ByRole) > 0 {
		fmt.Printf("\n%s\n", style.Bold.Render("By Role:"))
		for role, rm := range metrics.ByRole {
			fmt.Printf("  %s: %d tasks, %.1f%% success, $%.2f\n",
				style.Bold.Render(role),
				rm.TotalTasks,
				rm.SuccessRate*100,
				rm.TotalCost)
		}
	}

	// By Model
	if len(metrics.ByModel) > 0 {
		fmt.Printf("\n%s\n", style.Bold.Render("By Model:"))
		for model, mm := range metrics.ByModel {
			fmt.Printf("  %s: %d tasks, %.1f%% success, avg %v\n",
				style.Bold.Render(model),
				mm.TotalTasks,
				mm.SuccessRate*100,
				mm.AvgDuration.Round(time.Second))
		}
	}

	// By Provider
	if len(metrics.ByProvider) > 0 {
		fmt.Printf("\n%s\n", style.Bold.Render("By Provider:"))
		for provider, pm := range metrics.ByProvider {
			status := style.Success.Render("healthy")
			if pm.RateLimitHits > 5 {
				status = style.Warning.Render("rate limited")
			}
			fmt.Printf("  %s: %d tasks, $%.2f, %s\n",
				style.Bold.Render(provider),
				pm.TotalTasks,
				pm.TotalCost,
				status)
		}
	}

	if summary.TotalTasks == 0 {
		fmt.Printf("\n%s\n", style.Dim.Render("No metrics recorded yet. Run tasks to collect data."))
	}

	return nil
}

func runCouncilCompare(cmd *cobra.Command, args []string) error {
	model1, model2 := args[0], args[1]

	townRoot, err := workspace.FindFromCwd()
	if err != nil {
		return fmt.Errorf("finding town root: %w", err)
	}

	store, err := council.NewMetricsStore(townRoot)
	if err != nil {
		return fmt.Errorf("loading metrics: %w", err)
	}

	comparison := store.CompareModels(model1, model2)
	if comparison == nil {
		return fmt.Errorf("insufficient data for comparison (need metrics for both %s and %s)", model1, model2)
	}

	mm1 := store.GetModelMetrics(model1)
	mm2 := store.GetModelMetrics(model2)

	fmt.Printf("%s\n\n", style.Bold.Render("Model Comparison: "+model1+" vs "+model2))

	// Side by side comparison
	fmt.Printf("%-20s %15s %15s %15s\n", "", style.Bold.Render(model1), style.Bold.Render(model2), style.Bold.Render("Diff"))
	fmt.Printf("%-20s %15d %15d %+15d\n", "Total Tasks", mm1.TotalTasks, mm2.TotalTasks, comparison.TaskDiff)
	fmt.Printf("%-20s %14.1f%% %14.1f%% %+14.1f%%\n", "Success Rate", mm1.SuccessRate*100, mm2.SuccessRate*100, comparison.SuccessDiff*100)
	fmt.Printf("%-20s %15s %15s %15s\n", "Avg Duration",
		mm1.AvgDuration.Round(time.Second).String(),
		mm2.AvgDuration.Round(time.Second).String(),
		comparison.DurationDiff.Round(time.Second).String())
	fmt.Printf("%-20s $%14.2f $%14.2f $%+14.2f\n", "Total Cost", mm1.TotalCost, mm2.TotalCost, comparison.CostDiff)

	// Recommendation
	fmt.Printf("\n%s ", style.Bold.Render("Recommendation:"))
	if mm1.SuccessRate > mm2.SuccessRate && mm1.TotalCost <= mm2.TotalCost {
		fmt.Printf("%s has better success rate at equal or lower cost\n", style.Success.Render(model1))
	} else if mm2.SuccessRate > mm1.SuccessRate && mm2.TotalCost <= mm1.TotalCost {
		fmt.Printf("%s has better success rate at equal or lower cost\n", style.Success.Render(model2))
	} else if mm1.TotalCost < mm2.TotalCost && mm1.SuccessRate >= mm2.SuccessRate*0.95 {
		fmt.Printf("%s is more cost-effective with similar success rate\n", style.Success.Render(model1))
	} else if mm2.TotalCost < mm1.TotalCost && mm2.SuccessRate >= mm1.SuccessRate*0.95 {
		fmt.Printf("%s is more cost-effective with similar success rate\n", style.Success.Render(model2))
	} else {
		fmt.Printf("Trade-off depends on priorities (cost vs success rate)\n")
	}

	return nil
}

func runCouncilChains(cmd *cobra.Command, args []string) error {
	if councilShowJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(council.PredefinedChains)
	}

	fmt.Printf("%s\n\n", style.Bold.Render("Predefined Chain Patterns"))
	fmt.Printf("%s\n\n", style.Dim.Render("Chains pass output through a sequence of models"))

	for name, chain := range council.PredefinedChains {
		fmt.Printf("  %s\n", style.Bold.Render(name))
		fmt.Printf("    Steps: %d\n", len(chain.Steps))

		// Show step models
		var models []string
		for _, step := range chain.Steps {
			models = append(models, step.Model)
		}
		fmt.Printf("    Flow:  %s\n", strings.Join(models, " -> "))

		// Show options
		var opts []string
		if chain.PassContext {
			opts = append(opts, "pass-context")
		}
		if chain.StopOnError {
			opts = append(opts, "stop-on-error")
		}
		if len(opts) > 0 {
			fmt.Printf("    Opts:  %s\n", strings.Join(opts, ", "))
		}
		fmt.Println()
	}

	fmt.Printf("%s\n", style.Dim.Render("Use 'gt council pattern <name>' for full details"))

	return nil
}

func runCouncilEnsembles(cmd *cobra.Command, args []string) error {
	if councilShowJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(council.PredefinedEnsembles)
	}

	fmt.Printf("%s\n\n", style.Bold.Render("Predefined Ensemble Patterns"))
	fmt.Printf("%s\n\n", style.Dim.Render("Ensembles run models in parallel and vote on output"))

	for name, ensemble := range council.PredefinedEnsembles {
		fmt.Printf("  %s\n", style.Bold.Render(name))
		fmt.Printf("    Models:   %s\n", strings.Join(ensemble.Models, ", "))
		fmt.Printf("    Strategy: %s\n", ensemble.VotingStrategy)
		if ensemble.Threshold > 0 {
			fmt.Printf("    Threshold: %.0f%%\n", ensemble.Threshold*100)
		}
		fmt.Printf("    Timeout:  %s\n", ensemble.Timeout)
		fmt.Println()
	}

	fmt.Printf("%s\n", style.Dim.Render("Use 'gt council pattern <name>' for full details"))

	return nil
}

func runCouncilPattern(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check chains first
	if chain, ok := council.PredefinedChains[name]; ok {
		fmt.Printf("%s %s\n\n", style.Bold.Render("Chain:"), name)
		fmt.Printf("Type: Chain-of-Models\n")
		fmt.Printf("Pass Context: %v\n", chain.PassContext)
		fmt.Printf("Stop on Error: %v\n\n", chain.StopOnError)

		fmt.Printf("%s\n", style.Bold.Render("Steps:"))
		for i, step := range chain.Steps {
			fmt.Printf("\n  %d. %s\n", i+1, style.Bold.Render(step.Name))
			fmt.Printf("     Model: %s\n", step.Model)
			if step.Role != "" {
				fmt.Printf("     Role:  %s\n", step.Role)
			}
			if step.TransformOutput != "" {
				fmt.Printf("     Transform: %s\n", step.TransformOutput)
			}
			if step.Prompt != "" {
				// Truncate long prompts
				prompt := step.Prompt
				if len(prompt) > 60 {
					prompt = prompt[:57] + "..."
				}
				fmt.Printf("     Prompt: %s\n", style.Dim.Render(prompt))
			}
		}
		return nil
	}

	// Check ensembles
	if ensemble, ok := council.PredefinedEnsembles[name]; ok {
		fmt.Printf("%s %s\n\n", style.Bold.Render("Ensemble:"), name)
		fmt.Printf("Type: Ensemble Voting\n")
		fmt.Printf("Strategy: %s\n", ensemble.VotingStrategy)
		fmt.Printf("Timeout: %s\n", ensemble.Timeout)
		if ensemble.Threshold > 0 {
			fmt.Printf("Threshold: %.0f%%\n", ensemble.Threshold*100)
		}
		fmt.Printf("Min Responses: %d\n\n", ensemble.MinResponses)

		fmt.Printf("%s\n", style.Bold.Render("Models:"))
		for _, model := range ensemble.Models {
			fmt.Printf("  - %s\n", model)
		}

		// Explain voting strategy
		fmt.Printf("\n%s\n", style.Bold.Render("Voting Strategy:"))
		switch ensemble.VotingStrategy {
		case council.VoteMajority:
			fmt.Printf("  Selects the most common response among models.\n")
		case council.VoteConsensus:
			fmt.Printf("  Requires all models to agree. Falls back to majority if not.\n")
		case council.VoteWeighted:
			fmt.Printf("  Weights votes by model confidence scores.\n")
		case council.VoteBest:
			fmt.Printf("  Selects the highest quality response based on metrics.\n")
		}

		return nil
	}

	return fmt.Errorf("pattern %q not found (try 'gt council chains' or 'gt council ensembles')", name)
}

func init() {
	// Add flags
	councilShowCmd.Flags().BoolVar(&councilShowJSON, "json", false, "Output as JSON")
	councilProvidersCmd.Flags().BoolVar(&councilShowJSON, "json", false, "Output as JSON")
	councilRouteCmd.Flags().StringVar(&councilRouteComplex, "complexity", "", "Task complexity (low, medium, high)")
	councilInitCmd.Flags().BoolVar(&councilInitForce, "force", false, "Overwrite existing config")
	councilStatsCmd.Flags().BoolVar(&councilStatsJSON, "json", false, "Output as JSON")
	councilChainsCmd.Flags().BoolVar(&councilShowJSON, "json", false, "Output as JSON")
	councilEnsemblesCmd.Flags().BoolVar(&councilShowJSON, "json", false, "Output as JSON")

	// Add subcommands
	councilCmd.AddCommand(councilShowCmd)
	councilCmd.AddCommand(councilRoleCmd)
	councilCmd.AddCommand(councilSetCmd)
	councilCmd.AddCommand(councilFallbackCmd)
	councilCmd.AddCommand(councilProvidersCmd)
	councilCmd.AddCommand(councilRouteCmd)
	councilCmd.AddCommand(councilInitCmd)
	councilCmd.AddCommand(councilTemplatesCmd)
	councilCmd.AddCommand(councilStatsCmd)
	councilCmd.AddCommand(councilCompareCmd)
	councilCmd.AddCommand(councilChainsCmd)
	councilCmd.AddCommand(councilEnsemblesCmd)
	councilCmd.AddCommand(councilPatternCmd)

	// Register with root
	rootCmd.AddCommand(councilCmd)
}
