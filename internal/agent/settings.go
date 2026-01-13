// Package agent provides unified agent settings management.
package agent

import (
	"github.com/steveyegge/gastown/internal/claude"
	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/cursor"
)

// EnsureSettingsForRole ensures agent settings exist for the given agent preset and role.
// This is a unified function that delegates to the appropriate agent-specific implementation.
//
// For Claude: Creates .claude/settings.json with hooks
// For Cursor: Creates .cursor/rules/gastown.mdc with rules
// For other agents: Currently no-op (may be extended in future)
func EnsureSettingsForRole(workDir, role string, agentName string) error {
	// If no agent specified, default to claude for backwards compatibility
	if agentName == "" {
		agentName = "claude"
	}

	preset := config.GetAgentPresetByName(agentName)
	if preset == nil {
		// Unknown agent, try claude as fallback
		return claude.EnsureSettingsForRole(workDir, role)
	}

	switch preset.Name {
	case config.AgentClaude:
		return claude.EnsureSettingsForRole(workDir, role)
	case config.AgentCursor:
		return cursor.EnsureSettingsForRole(workDir, role)
	case config.AgentGemini, config.AgentCodex, config.AgentAuggie, config.AgentAmp:
		// These agents don't have a similar settings/rules mechanism yet
		// They may read AGENTS.md or have their own config
		return nil
	default:
		// Unknown preset, default to claude for backwards compatibility
		return claude.EnsureSettingsForRole(workDir, role)
	}
}

// EnsureSettingsForAllAgents ensures settings exist for all supported agents.
// This is useful during installation to prepare the workspace for any agent.
func EnsureSettingsForAllAgents(workDir, role string) error {
	// Ensure Claude settings (always, for backwards compatibility)
	if err := claude.EnsureSettingsForRole(workDir, role); err != nil {
		return err
	}

	// Ensure Cursor rules
	if err := cursor.EnsureSettingsForRole(workDir, role); err != nil {
		return err
	}

	return nil
}
