// Package cursor provides Cursor CLI configuration management.
package cursor

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed config/*.mdc
var configFS embed.FS

// RoleType indicates whether a role is autonomous or interactive.
type RoleType string

const (
	// Autonomous roles (polecat, witness, refinery) need initialization commands
	// at session start because they may be triggered externally.
	Autonomous RoleType = "autonomous"

	// Interactive roles (mayor, crew) wait for user input.
	Interactive RoleType = "interactive"
)

// RoleTypeFor returns the RoleType for a given role name.
func RoleTypeFor(role string) RoleType {
	switch role {
	case "polecat", "witness", "refinery", "deacon":
		return Autonomous
	default:
		return Interactive
	}
}

// EnsureSettings ensures .cursor/rules directory exists with Gas Town rules,
// and installs Gas Town hooks for Cursor CLI.
// For worktrees, we use sparse checkout to exclude source repo's .cursor/ directory,
// so our rules are the only ones Cursor sees.
func EnsureSettings(workDir string, roleType RoleType) error {
	cursorDir := filepath.Join(workDir, ".cursor", "rules")
	rulesFile := filepath.Join(cursorDir, "gastown.mdc")

	// Create .cursor/rules directory if needed
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		return fmt.Errorf("creating .cursor/rules directory: %w", err)
	}

	// Install rules file if it doesn't exist
	if _, err := os.Stat(rulesFile); os.IsNotExist(err) {
		// Select template based on role type
		var templateName string
		switch roleType {
		case Autonomous:
			templateName = "config/rules-autonomous.mdc"
		default:
			templateName = "config/rules-interactive.mdc"
		}

		// Read template
		content, err := configFS.ReadFile(templateName)
		if err != nil {
			return fmt.Errorf("reading template %s: %w", templateName, err)
		}

		// Write rules file
		if err := os.WriteFile(rulesFile, content, 0600); err != nil {
			return fmt.Errorf("writing rules: %w", err)
		}
	}

	// Install Gas Town hooks for Cursor CLI
	if err := EnsureHooks(workDir); err != nil {
		return fmt.Errorf("installing hooks: %w", err)
	}

	return nil
}

// EnsureSettingsForRole is a convenience function that combines RoleTypeFor and EnsureSettings.
func EnsureSettingsForRole(workDir, role string) error {
	return EnsureSettings(workDir, RoleTypeFor(role))
}
