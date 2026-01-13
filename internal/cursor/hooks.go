package cursor

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed config/hooks.json config/gastown-prompt.sh config/gastown-stop.sh config/gastown-audit.sh
var hooksFS embed.FS

// HooksConfig represents the structure of Cursor's hooks.json
type HooksConfig struct {
	Version int                    `json:"version"`
	Hooks   map[string][]HookEntry `json:"hooks"`
}

// HookEntry represents a single hook configuration
type HookEntry struct {
	Command string `json:"command"`
}

// EnsureHooks ensures Gas Town hooks are installed in the workspace.
// This creates .cursor/hooks.json and .cursor/hooks/ directory with hook scripts.
func EnsureHooks(workDir string) error {
	cursorDir := filepath.Join(workDir, ".cursor")
	hooksDir := filepath.Join(cursorDir, "hooks")

	// Create .cursor/hooks directory
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}

	// Install hooks.json if it doesn't exist
	hooksJsonPath := filepath.Join(cursorDir, "hooks.json")
	if _, err := os.Stat(hooksJsonPath); os.IsNotExist(err) {
		content, err := hooksFS.ReadFile("config/hooks.json")
		if err != nil {
			return fmt.Errorf("reading hooks.json template: %w", err)
		}
		if err := os.WriteFile(hooksJsonPath, content, 0644); err != nil {
			return fmt.Errorf("writing hooks.json: %w", err)
		}
	}

	// Install hook scripts
	hookScripts := []string{
		"gastown-prompt.sh",
		"gastown-stop.sh",
		"gastown-audit.sh",
	}

	for _, script := range hookScripts {
		scriptPath := filepath.Join(hooksDir, script)
		
		// Always overwrite hook scripts to ensure latest version
		content, err := hooksFS.ReadFile("config/" + script)
		if err != nil {
			return fmt.Errorf("reading %s template: %w", script, err)
		}
		if err := os.WriteFile(scriptPath, content, 0755); err != nil {
			return fmt.Errorf("writing %s: %w", script, err)
		}
	}

	return nil
}

// HooksInstalled checks if Gas Town hooks are installed in the workspace.
func HooksInstalled(workDir string) bool {
	hooksJsonPath := filepath.Join(workDir, ".cursor", "hooks.json")
	_, err := os.Stat(hooksJsonPath)
	return err == nil
}

// RemoveHooks removes Gas Town hooks from the workspace.
func RemoveHooks(workDir string) error {
	hooksDir := filepath.Join(workDir, ".cursor", "hooks")
	hooksJsonPath := filepath.Join(workDir, ".cursor", "hooks.json")

	// Remove hooks directory
	if err := os.RemoveAll(hooksDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing hooks directory: %w", err)
	}

	// Remove hooks.json
	if err := os.Remove(hooksJsonPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing hooks.json: %w", err)
	}

	return nil
}
