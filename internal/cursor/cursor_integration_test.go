//go:build integration

package cursor

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCursorAgentAvailable verifies cursor-agent is installed and available.
func TestCursorAgentAvailable(t *testing.T) {
	path, err := exec.LookPath("cursor-agent")
	if err != nil {
		t.Skip("cursor-agent not installed, skipping cursor integration tests")
	}
	t.Logf("cursor-agent found at: %s", path)
}

// TestCursorConfigurationSetup verifies that Cursor configuration
// is correctly set up for Gas Town agents.
func TestCursorConfigurationSetup(t *testing.T) {
	tmpDir := t.TempDir()

	// Test autonomous settings
	t.Run("autonomous role setup", func(t *testing.T) {
		workDir := filepath.Join(tmpDir, "autonomous")
		if err := os.MkdirAll(workDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := EnsureSettings(workDir, Autonomous); err != nil {
			t.Fatalf("EnsureSettings failed: %v", err)
		}

		// Verify rules file
		rulesPath := filepath.Join(workDir, ".cursor", "rules", "gastown.mdc")
		if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
			t.Error("rules file not created")
		}

		// Verify hooks
		if !HooksInstalled(workDir) {
			t.Error("hooks not installed")
		}

		// Verify hooks.json
		hooksPath := filepath.Join(workDir, ".cursor", "hooks.json")
		if _, err := os.Stat(hooksPath); os.IsNotExist(err) {
			t.Error("hooks.json not created")
		}

		// Verify hook scripts
		scripts := []string{"gastown-prompt.sh", "gastown-stop.sh", "gastown-shell.sh"}
		for _, script := range scripts {
			scriptPath := filepath.Join(workDir, ".cursor", "hooks", script)
			info, err := os.Stat(scriptPath)
			if os.IsNotExist(err) {
				t.Errorf("hook script %s not created", script)
				continue
			}
			if info.Mode()&0111 == 0 {
				t.Errorf("hook script %s is not executable", script)
			}
		}
	})

	// Test interactive settings
	t.Run("interactive role setup", func(t *testing.T) {
		workDir := filepath.Join(tmpDir, "interactive")
		if err := os.MkdirAll(workDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := EnsureSettings(workDir, Interactive); err != nil {
			t.Fatalf("EnsureSettings failed: %v", err)
		}

		// Verify rules file
		rulesPath := filepath.Join(workDir, ".cursor", "rules", "gastown.mdc")
		content, err := os.ReadFile(rulesPath)
		if err != nil {
			t.Fatalf("failed to read rules: %v", err)
		}

		// Should contain interactive-specific content
		if len(content) == 0 {
			t.Error("rules file is empty")
		}
	})
}

// TestMCPConfigurationManagement verifies MCP configuration CRUD operations.
func TestMCPConfigurationManagement(t *testing.T) {
	tmpDir := t.TempDir()

	// Test full lifecycle
	t.Run("full lifecycle", func(t *testing.T) {
		// Start with no config
		names, err := ListMCPServers(tmpDir)
		if err != nil {
			t.Fatalf("ListMCPServers failed: %v", err)
		}
		if len(names) != 0 {
			t.Errorf("expected 0 servers initially, got %d", len(names))
		}

		// Add stdio server
		err = AddMCPServer(tmpDir, "local-tool", MCPServer{
			Type:    "stdio",
			Command: "my-mcp-tool",
			Args:    []string{"--port", "8080"},
			Env: map[string]string{
				"API_KEY": "${env:MY_API_KEY}",
			},
		})
		if err != nil {
			t.Fatalf("AddMCPServer failed: %v", err)
		}

		// Add HTTP server
		err = AddMCPServer(tmpDir, "remote-api", MCPServer{
			URL: "https://api.example.com/mcp",
			Headers: map[string]string{
				"Authorization": "Bearer ${env:API_TOKEN}",
			},
		})
		if err != nil {
			t.Fatalf("AddMCPServer failed: %v", err)
		}

		// Verify both exist
		names, err = ListMCPServers(tmpDir)
		if err != nil {
			t.Fatalf("ListMCPServers failed: %v", err)
		}
		if len(names) != 2 {
			t.Errorf("expected 2 servers, got %d", len(names))
		}

		// Get and verify stdio server
		server, err := GetMCPServer(tmpDir, "local-tool")
		if err != nil {
			t.Fatalf("GetMCPServer failed: %v", err)
		}
		if server == nil {
			t.Fatal("server should not be nil")
		}
		if server.MCPServerType() != "stdio" {
			t.Errorf("expected stdio type, got %s", server.MCPServerType())
		}
		if server.Command != "my-mcp-tool" {
			t.Errorf("expected command 'my-mcp-tool', got %s", server.Command)
		}

		// Get and verify HTTP server
		server, err = GetMCPServer(tmpDir, "remote-api")
		if err != nil {
			t.Fatalf("GetMCPServer failed: %v", err)
		}
		if server.MCPServerType() != "http" {
			t.Errorf("expected http type, got %s", server.MCPServerType())
		}

		// Remove one
		err = RemoveMCPServer(tmpDir, "local-tool")
		if err != nil {
			t.Fatalf("RemoveMCPServer failed: %v", err)
		}

		// Verify only one remains
		names, err = ListMCPServers(tmpDir)
		if err != nil {
			t.Fatalf("ListMCPServers failed: %v", err)
		}
		if len(names) != 1 {
			t.Errorf("expected 1 server after removal, got %d", len(names))
		}
		if names[0] != "remote-api" {
			t.Errorf("expected 'remote-api' to remain, got %s", names[0])
		}
	})
}

// TestMCPServerWithAuth verifies OAuth configuration.
func TestMCPServerWithAuth(t *testing.T) {
	tmpDir := t.TempDir()

	err := AddMCPServer(tmpDir, "oauth-server", MCPServer{
		URL: "https://api.provider.com/mcp",
		Auth: &MCPAuth{
			ClientID:     "${env:OAUTH_CLIENT_ID}",
			ClientSecret: "${env:OAUTH_CLIENT_SECRET}",
			Scopes:       []string{"read", "write"},
		},
	})
	if err != nil {
		t.Fatalf("AddMCPServer with auth failed: %v", err)
	}

	server, err := GetMCPServer(tmpDir, "oauth-server")
	if err != nil {
		t.Fatalf("GetMCPServer failed: %v", err)
	}
	if server.Auth == nil {
		t.Fatal("Auth should not be nil")
	}
	if server.Auth.ClientID != "${env:OAUTH_CLIENT_ID}" {
		t.Errorf("ClientID = %q, want interpolation pattern", server.Auth.ClientID)
	}
	if len(server.Auth.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(server.Auth.Scopes))
	}
}


// TestConfigMerging verifies MCP configuration merging.
func TestConfigMerging(t *testing.T) {
	// Simulate global + project config merge
	globalConfig := &MCPConfig{
		McpServers: map[string]MCPServer{
			"global-tool": {
				Command: "global-mcp",
				Args:    []string{"--global"},
			},
			"shared-tool": {
				Command: "shared-v1",
			},
		},
	}

	projectConfig := &MCPConfig{
		McpServers: map[string]MCPServer{
			"project-tool": {
				Command: "project-mcp",
			},
			"shared-tool": {
				Command: "shared-v2", // Override global
			},
		},
	}

	// Project config should override global
	merged := MergeMCPConfigs(globalConfig, projectConfig)

	if len(merged.McpServers) != 3 {
		t.Errorf("expected 3 servers after merge, got %d", len(merged.McpServers))
	}

	// Global tool should be present
	if _, ok := merged.McpServers["global-tool"]; !ok {
		t.Error("global-tool should be present")
	}

	// Project tool should be present
	if _, ok := merged.McpServers["project-tool"]; !ok {
		t.Error("project-tool should be present")
	}

	// Shared tool should use project version
	if shared := merged.McpServers["shared-tool"]; shared.Command != "shared-v2" {
		t.Errorf("shared-tool should use project version, got %s", shared.Command)
	}
}
