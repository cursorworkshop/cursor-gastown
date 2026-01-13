package cursor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureHooks(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureHooks(tmpDir)
	if err != nil {
		t.Fatalf("EnsureHooks failed: %v", err)
	}

	// Check hooks.json was created
	hooksJsonPath := filepath.Join(tmpDir, ".cursor", "hooks.json")
	if _, err := os.Stat(hooksJsonPath); os.IsNotExist(err) {
		t.Error("hooks.json not created")
	}

	// Verify hooks.json is valid JSON
	content, err := os.ReadFile(hooksJsonPath)
	if err != nil {
		t.Fatalf("failed to read hooks.json: %v", err)
	}

	var config HooksConfig
	if err := json.Unmarshal(content, &config); err != nil {
		t.Errorf("hooks.json is not valid JSON: %v", err)
	}

	// Check version
	if config.Version != 1 {
		t.Errorf("expected version 1, got %d", config.Version)
	}

	// Check hooks are configured
	if len(config.Hooks) == 0 {
		t.Error("no hooks configured")
	}

	// Check specific hooks
	if _, ok := config.Hooks["beforeSubmitPrompt"]; !ok {
		t.Error("beforeSubmitPrompt hook not configured")
	}
	if _, ok := config.Hooks["stop"]; !ok {
		t.Error("stop hook not configured")
	}
}

func TestEnsureHooks_ScriptsCreated(t *testing.T) {
	tmpDir := t.TempDir()

	err := EnsureHooks(tmpDir)
	if err != nil {
		t.Fatalf("EnsureHooks failed: %v", err)
	}

	// Check hook scripts were created
	scripts := []string{
		"gastown-prompt.sh",
		"gastown-stop.sh",
		"gastown-audit.sh",
	}

	for _, script := range scripts {
		scriptPath := filepath.Join(tmpDir, ".cursor", "hooks", script)
		info, err := os.Stat(scriptPath)
		if os.IsNotExist(err) {
			t.Errorf("hook script %s not created", script)
			continue
		}

		// Check script is executable
		if info.Mode()&0111 == 0 {
			t.Errorf("hook script %s is not executable", script)
		}
	}
}

func TestEnsureHooks_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// Run twice
	for i := 0; i < 2; i++ {
		err := EnsureHooks(tmpDir)
		if err != nil {
			t.Fatalf("EnsureHooks iteration %d failed: %v", i+1, err)
		}
	}

	// Verify still works
	hooksJsonPath := filepath.Join(tmpDir, ".cursor", "hooks.json")
	if _, err := os.Stat(hooksJsonPath); os.IsNotExist(err) {
		t.Error("hooks.json should still exist")
	}
}

func TestHooksInstalled(t *testing.T) {
	tmpDir := t.TempDir()

	// Should be false initially
	if HooksInstalled(tmpDir) {
		t.Error("HooksInstalled should return false before installation")
	}

	// Install hooks
	if err := EnsureHooks(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Should be true now
	if !HooksInstalled(tmpDir) {
		t.Error("HooksInstalled should return true after installation")
	}
}

func TestRemoveHooks(t *testing.T) {
	tmpDir := t.TempDir()

	// Install hooks
	if err := EnsureHooks(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Remove hooks
	if err := RemoveHooks(tmpDir); err != nil {
		t.Fatalf("RemoveHooks failed: %v", err)
	}

	// Verify removed
	if HooksInstalled(tmpDir) {
		t.Error("hooks should be removed")
	}

	hooksDir := filepath.Join(tmpDir, ".cursor", "hooks")
	if _, err := os.Stat(hooksDir); !os.IsNotExist(err) {
		t.Error("hooks directory should be removed")
	}
}

func TestEnsureSettings_InstallsHooks(t *testing.T) {
	tmpDir := t.TempDir()

	// EnsureSettings should also install hooks
	err := EnsureSettings(tmpDir, Autonomous)
	if err != nil {
		t.Fatalf("EnsureSettings failed: %v", err)
	}

	// Verify hooks are installed
	if !HooksInstalled(tmpDir) {
		t.Error("EnsureSettings should install hooks")
	}

	// Verify rules are installed
	rulesPath := filepath.Join(tmpDir, ".cursor", "rules", "gastown.mdc")
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		t.Error("EnsureSettings should install rules")
	}
}
