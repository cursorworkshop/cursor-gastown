package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSettingsForRole_Claude(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureSettingsForRole(tmpDir, "polecat", "claude")
	if err != nil {
		t.Fatalf("EnsureSettingsForRole failed: %v", err)
	}

	// Check Claude settings were created
	claudeSettings := filepath.Join(tmpDir, ".claude", "settings.json")
	if _, err := os.Stat(claudeSettings); os.IsNotExist(err) {
		t.Error("Claude settings.json not created")
	}
}

func TestEnsureSettingsForRole_Cursor(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureSettingsForRole(tmpDir, "polecat", "cursor")
	if err != nil {
		t.Fatalf("EnsureSettingsForRole failed: %v", err)
	}

	// Check Cursor rules were created
	cursorRules := filepath.Join(tmpDir, ".cursor", "rules", "gastown.mdc")
	if _, err := os.Stat(cursorRules); os.IsNotExist(err) {
		t.Error("Cursor rules not created")
	}
}

func TestEnsureSettingsForRole_DefaultsToClaude(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty agent name should default to claude
	err := EnsureSettingsForRole(tmpDir, "polecat", "")
	if err != nil {
		t.Fatalf("EnsureSettingsForRole failed: %v", err)
	}

	claudeSettings := filepath.Join(tmpDir, ".claude", "settings.json")
	if _, err := os.Stat(claudeSettings); os.IsNotExist(err) {
		t.Error("Claude settings.json not created for empty agent name")
	}
}

func TestEnsureSettingsForRole_UnknownAgent(t *testing.T) {
	tmpDir := t.TempDir()

	// Unknown agent should fall back to claude
	err := EnsureSettingsForRole(tmpDir, "polecat", "unknown-agent")
	if err != nil {
		t.Fatalf("EnsureSettingsForRole failed: %v", err)
	}

	claudeSettings := filepath.Join(tmpDir, ".claude", "settings.json")
	if _, err := os.Stat(claudeSettings); os.IsNotExist(err) {
		t.Error("Claude settings.json not created for unknown agent")
	}
}

func TestEnsureSettingsForRole_Gemini(t *testing.T) {
	tmpDir := t.TempDir()

	// Gemini doesn't have settings yet, should be a no-op
	err := EnsureSettingsForRole(tmpDir, "polecat", "gemini")
	if err != nil {
		t.Fatalf("EnsureSettingsForRole failed: %v", err)
	}

	// Neither Claude nor Cursor settings should be created for Gemini
	claudeSettings := filepath.Join(tmpDir, ".claude", "settings.json")
	cursorRules := filepath.Join(tmpDir, ".cursor", "rules", "gastown.mdc")

	if _, err := os.Stat(claudeSettings); !os.IsNotExist(err) {
		t.Error("Claude settings.json should not be created for Gemini")
	}
	if _, err := os.Stat(cursorRules); !os.IsNotExist(err) {
		t.Error("Cursor rules should not be created for Gemini")
	}
}

func TestEnsureSettingsForAllAgents(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureSettingsForAllAgents(tmpDir, "polecat")
	if err != nil {
		t.Fatalf("EnsureSettingsForAllAgents failed: %v", err)
	}

	// Both Claude and Cursor settings should be created
	claudeSettings := filepath.Join(tmpDir, ".claude", "settings.json")
	if _, err := os.Stat(claudeSettings); os.IsNotExist(err) {
		t.Error("Claude settings.json not created")
	}

	cursorRules := filepath.Join(tmpDir, ".cursor", "rules", "gastown.mdc")
	if _, err := os.Stat(cursorRules); os.IsNotExist(err) {
		t.Error("Cursor rules not created")
	}
}
