package cursor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoleTypeFor(t *testing.T) {
	tests := []struct {
		role     string
		expected RoleType
	}{
		{"polecat", Autonomous},
		{"witness", Autonomous},
		{"refinery", Autonomous},
		{"deacon", Autonomous},
		{"mayor", Interactive},
		{"crew", Interactive},
		{"unknown", Interactive},
		{"", Interactive},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := RoleTypeFor(tt.role)
			if got != tt.expected {
				t.Errorf("RoleTypeFor(%q) = %v, want %v", tt.role, got, tt.expected)
			}
		})
	}
}

func TestEnsureSettings_Autonomous(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureSettings(tmpDir, Autonomous)
	if err != nil {
		t.Fatalf("EnsureSettings failed: %v", err)
	}

	rulesPath := filepath.Join(tmpDir, ".cursor", "rules", "gastown.mdc")
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		t.Errorf("rules file not created at %s", rulesPath)
	}

	content, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("failed to read rules file: %v", err)
	}

	// Check for autonomous-specific content
	if !strings.Contains(string(content), "autonomous agent") {
		t.Errorf("rules file should contain 'autonomous agent' for Autonomous role type")
	}
	if !strings.Contains(string(content), "gt mail check --inject") {
		t.Errorf("rules file should contain mail check instruction")
	}
}

func TestEnsureSettings_Interactive(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureSettings(tmpDir, Interactive)
	if err != nil {
		t.Fatalf("EnsureSettings failed: %v", err)
	}

	rulesPath := filepath.Join(tmpDir, ".cursor", "rules", "gastown.mdc")
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		t.Errorf("rules file not created at %s", rulesPath)
	}

	content, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("failed to read rules file: %v", err)
	}

	// Check for interactive-specific content
	if !strings.Contains(string(content), "interactive agent") {
		t.Errorf("rules file should contain 'interactive agent' for Interactive role type")
	}
}

func TestEnsureSettings_NoOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory and file with custom content
	rulesDir := filepath.Join(tmpDir, ".cursor", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}

	customContent := []byte("# Custom rules - do not overwrite")
	rulesPath := filepath.Join(rulesDir, "gastown.mdc")
	if err := os.WriteFile(rulesPath, customContent, 0600); err != nil {
		t.Fatal(err)
	}

	// Call EnsureSettings - should not overwrite
	err := EnsureSettings(tmpDir, Autonomous)
	if err != nil {
		t.Fatalf("EnsureSettings failed: %v", err)
	}

	// Check content wasn't overwritten
	content, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(customContent) {
		t.Errorf("rules file was overwritten, got %q", string(content))
	}
}

func TestEnsureSettingsForRole(t *testing.T) {
	tests := []struct {
		role         string
		expectsAuto  bool
	}{
		{"polecat", true},
		{"witness", true},
		{"refinery", true},
		{"deacon", true},
		{"mayor", false},
		{"crew", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			tmpDir := t.TempDir()

			err := EnsureSettingsForRole(tmpDir, tt.role)
			if err != nil {
				t.Fatalf("EnsureSettingsForRole failed: %v", err)
			}

			rulesPath := filepath.Join(tmpDir, ".cursor", "rules", "gastown.mdc")
			content, err := os.ReadFile(rulesPath)
			if err != nil {
				t.Fatal(err)
			}

			if tt.expectsAuto {
				if !strings.Contains(string(content), "autonomous") {
					t.Errorf("expected autonomous rules for role %s", tt.role)
				}
			} else {
				if !strings.Contains(string(content), "interactive") {
					t.Errorf("expected interactive rules for role %s", tt.role)
				}
			}
		})
	}
}
