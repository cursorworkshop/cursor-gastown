// Package cursor provides Cursor CLI configuration management.
package cursor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cursorworkshop/cursor-gastown/internal/config"
)

// Adapter translates Gas Town operations to Cursor CLI commands.
// This is the primary interface for interacting with cursor-agent.
type Adapter struct {
	// WorkDir is the workspace directory.
	WorkDir string

	// Model is the model to use (e.g., "sonnet-4.5", "gpt-5.2").
	Model string

	// ForceMode enables YOLO/force mode (-f flag).
	ForceMode bool

	// PrintMode enables non-interactive/print mode (-p flag).
	PrintMode bool

	// OutputFormat specifies output format ("text" or "json").
	OutputFormat string

	// SessionID is an optional session ID for resume.
	SessionID string

	// ApproveAll auto-approves MCP servers and other prompts.
	ApproveAll bool

	// AdditionalArgs are extra arguments to pass to cursor-agent.
	AdditionalArgs []string
}

// DefaultAdapter returns an adapter with sensible defaults for Gas Town.
func DefaultAdapter(workDir string) *Adapter {
	return &Adapter{
		WorkDir:   workDir,
		ForceMode: true,  // Gas Town agents need autonomy
		ApproveAll: true, // Auto-approve for autonomous operation
	}
}

// AdapterForRole returns an adapter configured for a specific Gas Town role.
func AdapterForRole(workDir, role string) *Adapter {
	adapter := DefaultAdapter(workDir)

	// Role-specific configurations
	switch role {
	case "mayor":
		// Mayor uses the best model for coordination
		adapter.Model = "opus-4.5-thinking"
	case "refinery":
		// Refinery uses a different model family for code review diversity
		adapter.Model = "gpt-5.2-high"
	case "witness":
		// Witness uses fast, cheap model for monitoring
		adapter.Model = "gemini-3-flash"
	case "polecat":
		// Polecats use good coding model by default
		adapter.Model = "sonnet-4.5"
	case "crew":
		// Crew uses auto (user preference)
		adapter.Model = "auto"
	default:
		adapter.Model = "auto"
	}

	return adapter
}

// BuildCommand builds the cursor-agent command with all configured options.
func (a *Adapter) BuildCommand(prompt string) *exec.Cmd {
	args := a.BuildArgs(prompt)
	cmd := exec.Command("cursor-agent", args...)
	cmd.Dir = a.WorkDir
	return cmd
}

// BuildArgs builds the command-line arguments for cursor-agent.
func (a *Adapter) BuildArgs(prompt string) []string {
	var args []string

	// Session resume takes precedence
	if a.SessionID != "" {
		args = append(args, "--resume", a.SessionID)
	}

	// Model selection
	if a.Model != "" && a.Model != "auto" {
		args = append(args, "--model", a.Model)
	}

	// Force mode (YOLO equivalent)
	if a.ForceMode {
		args = append(args, "-f")
	}

	// Print mode for non-interactive
	if a.PrintMode {
		args = append(args, "-p")
	}

	// Output format
	if a.OutputFormat != "" {
		args = append(args, "--output-format", a.OutputFormat)
	}

	// MCP approval
	if a.ApproveAll {
		args = append(args, "--approve-mcps")
	}

	// Workspace
	if a.WorkDir != "" {
		args = append(args, "--workspace", a.WorkDir)
	}

	// Additional args
	args = append(args, a.AdditionalArgs...)

	// Prompt (must be last if provided)
	if prompt != "" {
		args = append(args, prompt)
	}

	return args
}

// BuildCommandString returns the full command as a string (for tmux SendKeys).
func (a *Adapter) BuildCommandString(prompt string) string {
	args := a.BuildArgs(prompt)
	return "cursor-agent " + strings.Join(args, " ")
}

// Run executes cursor-agent and returns the output.
// For non-interactive use; use BuildCommand for interactive sessions.
func (a *Adapter) Run(prompt string) (string, error) {
	a.PrintMode = true
	cmd := a.BuildCommand(prompt)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), fmt.Errorf("cursor-agent failed: %s\n%s", exitErr.Error(), string(exitErr.Stderr))
		}
		return "", fmt.Errorf("running cursor-agent: %w", err)
	}

	return string(output), nil
}

// RunJSON executes cursor-agent and returns JSON output.
func (a *Adapter) RunJSON(prompt string) ([]byte, error) {
	a.PrintMode = true
	a.OutputFormat = "json"
	cmd := a.BuildCommand(prompt)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running cursor-agent: %w", err)
	}

	return output, nil
}

// Available checks if cursor-agent is available in PATH.
func Available() bool {
	_, err := exec.LookPath("cursor-agent")
	return err == nil
}

// Version returns the cursor-agent version.
func Version() (string, error) {
	cmd := exec.Command("cursor-agent", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting cursor-agent version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// SupportedModels returns the list of supported models.
// These are the models available via cursor-agent.
var SupportedModels = []string{
	"auto",
	"opus-4.5-thinking",
	"opus-4.5",
	"sonnet-4.5",
	"sonnet-4.5-thinking",
	"gpt-5.2",
	"gpt-5.2-high",
	"gpt-5.1-codex-max",
	"gemini-3-pro",
	"gemini-3-flash",
	"grok",
}

// IsValidModel checks if a model name is valid.
func IsValidModel(model string) bool {
	for _, m := range SupportedModels {
		if m == model {
			return true
		}
	}
	return false
}

// ModelProvider returns the provider for a given model.
func ModelProvider(model string) string {
	switch {
	case strings.HasPrefix(model, "opus-"), strings.HasPrefix(model, "sonnet-"), strings.HasPrefix(model, "haiku-"):
		return "anthropic"
	case strings.HasPrefix(model, "gpt-"), strings.HasPrefix(model, "o4-"):
		return "openai"
	case strings.HasPrefix(model, "gemini-"):
		return "google"
	case model == "grok":
		return "xai"
	default:
		return "unknown"
	}
}

// TranslateRuntimeConfig converts a Gas Town RuntimeConfig to an Adapter.
func TranslateRuntimeConfig(rc *config.RuntimeConfig, workDir string) *Adapter {
	adapter := DefaultAdapter(workDir)

	if rc == nil {
		return adapter
	}

	// Parse args to extract model and other flags
	for i := 0; i < len(rc.Args); i++ {
		arg := rc.Args[i]
		switch {
		case arg == "--model" && i+1 < len(rc.Args):
			adapter.Model = rc.Args[i+1]
			i++
		case arg == "-f" || arg == "--force":
			adapter.ForceMode = true
		case arg == "-p" || arg == "--print":
			adapter.PrintMode = true
		case arg == "--output-format" && i+1 < len(rc.Args):
			adapter.OutputFormat = rc.Args[i+1]
			i++
		case arg == "--approve-mcps":
			adapter.ApproveAll = true
		default:
			adapter.AdditionalArgs = append(adapter.AdditionalArgs, arg)
		}
	}

	return adapter
}

// CleanOrphanClaudeConfig removes any orphaned Claude-specific configuration files.
// Returns true if any files were cleaned, false otherwise.
// This is a no-op in the current implementation as we've moved to multi-model orchestration.
func CleanOrphanClaudeConfig(workDir string) (bool, error) {
	// In the future, this could check for old Claude-specific config files
	// and remove them. For now, it's a safe no-op.
	return false, nil
}

// EnsureWorkspaceReady ensures the workspace is ready for cursor-agent.
// This creates necessary directories and configuration files.
func EnsureWorkspaceReady(workDir, role string) error {
	// Ensure .cursor directory exists
	cursorDir := filepath.Join(workDir, ".cursor")
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		return fmt.Errorf("creating .cursor directory: %w", err)
	}

	// Ensure settings (rules) are installed
	if err := EnsureSettingsForRole(workDir, role); err != nil {
		return fmt.Errorf("ensuring settings: %w", err)
	}

	// Clean orphan Claude config if exists
	if _, err := CleanOrphanClaudeConfig(workDir); err != nil {
		return fmt.Errorf("cleaning orphan config: %w", err)
	}

	return nil
}

// GetModelForRole returns the recommended model for a Gas Town role.
// This implements the Council's role-model matrix.
func GetModelForRole(role string) string {
	switch role {
	case "mayor":
		return "opus-4.5-thinking"
	case "refinery":
		return "gpt-5.2-high"
	case "witness":
		return "gemini-3-flash"
	case "polecat":
		return "sonnet-4.5"
	case "crew":
		return "auto"
	case "deacon":
		return "gemini-3-flash"
	default:
		return "auto"
	}
}

// GetModelRationale returns the reasoning for a role's model choice.
func GetModelRationale(role string) string {
	switch role {
	case "mayor":
		return "Strategic coordination requires sustained reasoning"
	case "refinery":
		return "Different model family catches bugs Claude misses"
	case "witness":
		return "Fast, cheap monitoring with good reasoning"
	case "polecat":
		return "Best coding model for implementation tasks"
	case "crew":
		return "User preference for interactive work"
	case "deacon":
		return "Lightweight lifecycle management"
	default:
		return "Default selection"
	}
}
