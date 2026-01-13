package cursor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMCPConfigPath(t *testing.T) {
	workDir := "/home/user/project"
	expected := "/home/user/project/.cursor/mcp.json"
	got := MCPConfigPath(workDir)
	if got != expected {
		t.Errorf("MCPConfigPath(%q) = %q, want %q", workDir, got, expected)
	}
}

func TestLoadMCPConfig_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "mcp.json")

	config, err := LoadMCPConfig(path)
	if err != nil {
		t.Fatalf("LoadMCPConfig failed: %v", err)
	}

	if config.McpServers == nil {
		t.Error("McpServers should not be nil")
	}
	if len(config.McpServers) != 0 {
		t.Errorf("McpServers should be empty, got %d", len(config.McpServers))
	}
}

func TestLoadMCPConfig_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "mcp.json")

	content := []byte(`{
		"mcpServers": {
			"test-server": {
				"url": "https://example.com/mcp"
			}
		}
	}`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadMCPConfig(path)
	if err != nil {
		t.Fatalf("LoadMCPConfig failed: %v", err)
	}

	if len(config.McpServers) != 1 {
		t.Errorf("expected 1 server, got %d", len(config.McpServers))
	}

	server, exists := config.McpServers["test-server"]
	if !exists {
		t.Fatal("test-server not found")
	}
	if server.URL != "https://example.com/mcp" {
		t.Errorf("URL = %q, want %q", server.URL, "https://example.com/mcp")
	}
}

func TestSaveMCPConfig(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".cursor", "mcp.json")

	config := &MCPConfig{
		McpServers: map[string]MCPServer{
			"my-server": {
				Command: "my-mcp",
				Args:    []string{"--port", "8080"},
				Env: map[string]string{
					"API_KEY": "secret",
				},
			},
		},
	}

	if err := SaveMCPConfig(path, config); err != nil {
		t.Fatalf("SaveMCPConfig failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("mcp.json was not created")
	}

	// Reload and verify
	loaded, err := LoadMCPConfig(path)
	if err != nil {
		t.Fatalf("LoadMCPConfig failed: %v", err)
	}

	server, exists := loaded.McpServers["my-server"]
	if !exists {
		t.Fatal("my-server not found after reload")
	}
	if server.Command != "my-mcp" {
		t.Errorf("Command = %q, want %q", server.Command, "my-mcp")
	}
	if len(server.Args) != 2 || server.Args[0] != "--port" || server.Args[1] != "8080" {
		t.Errorf("Args = %v, want [--port 8080]", server.Args)
	}
	if server.Env["API_KEY"] != "secret" {
		t.Errorf("Env[API_KEY] = %q, want %q", server.Env["API_KEY"], "secret")
	}
}

func TestAddMCPServer(t *testing.T) {
	tmpDir := t.TempDir()

	// Add first server
	err := AddMCPServer(tmpDir, "server1", MCPServer{
		URL: "https://server1.com/mcp",
	})
	if err != nil {
		t.Fatalf("AddMCPServer failed: %v", err)
	}

	// Add second server
	err = AddMCPServer(tmpDir, "server2", MCPServer{
		Command: "server2-cli",
	})
	if err != nil {
		t.Fatalf("AddMCPServer failed: %v", err)
	}

	// Verify both exist
	config, err := LoadMCPConfig(MCPConfigPath(tmpDir))
	if err != nil {
		t.Fatal(err)
	}

	if len(config.McpServers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(config.McpServers))
	}
	if config.McpServers["server1"].URL != "https://server1.com/mcp" {
		t.Error("server1 URL mismatch")
	}
	if config.McpServers["server2"].Command != "server2-cli" {
		t.Error("server2 Command mismatch")
	}
}

func TestRemoveMCPServer(t *testing.T) {
	tmpDir := t.TempDir()

	// Add servers
	_ = AddMCPServer(tmpDir, "keep", MCPServer{URL: "https://keep.com"})
	_ = AddMCPServer(tmpDir, "remove", MCPServer{URL: "https://remove.com"})

	// Remove one
	err := RemoveMCPServer(tmpDir, "remove")
	if err != nil {
		t.Fatalf("RemoveMCPServer failed: %v", err)
	}

	// Verify
	config, err := LoadMCPConfig(MCPConfigPath(tmpDir))
	if err != nil {
		t.Fatal(err)
	}

	if len(config.McpServers) != 1 {
		t.Errorf("expected 1 server, got %d", len(config.McpServers))
	}
	if _, exists := config.McpServers["keep"]; !exists {
		t.Error("'keep' server should still exist")
	}
	if _, exists := config.McpServers["remove"]; exists {
		t.Error("'remove' server should not exist")
	}
}

func TestListMCPServers(t *testing.T) {
	tmpDir := t.TempDir()

	// Initially empty
	names, err := ListMCPServers(tmpDir)
	if err != nil {
		t.Fatalf("ListMCPServers failed: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0 servers, got %d", len(names))
	}

	// Add servers
	_ = AddMCPServer(tmpDir, "server1", MCPServer{URL: "https://s1.com"})
	_ = AddMCPServer(tmpDir, "server2", MCPServer{Command: "my-mcp"})

	names, err = ListMCPServers(tmpDir)
	if err != nil {
		t.Fatalf("ListMCPServers failed: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("expected 2 servers, got %d", len(names))
	}
}

func TestGetMCPServer(t *testing.T) {
	tmpDir := t.TempDir()

	// Add a server
	_ = AddMCPServer(tmpDir, "test", MCPServer{
		Command: "test-mcp",
		Args:    []string{"--flag"},
	})

	// Get existing server
	server, err := GetMCPServer(tmpDir, "test")
	if err != nil {
		t.Fatalf("GetMCPServer failed: %v", err)
	}
	if server == nil {
		t.Fatal("expected server, got nil")
	}
	if server.Command != "test-mcp" {
		t.Errorf("Command = %q, want %q", server.Command, "test-mcp")
	}

	// Get non-existent server
	server, err = GetMCPServer(tmpDir, "nonexistent")
	if err != nil {
		t.Fatalf("GetMCPServer failed: %v", err)
	}
	if server != nil {
		t.Errorf("expected nil for non-existent server, got %+v", server)
	}
}

func TestMCPServerType(t *testing.T) {
	tests := []struct {
		name     string
		server   MCPServer
		expected string
	}{
		{
			name:     "stdio server",
			server:   MCPServer{Command: "my-mcp"},
			expected: "stdio",
		},
		{
			name:     "http server",
			server:   MCPServer{URL: "https://example.com/mcp"},
			expected: "http",
		},
		{
			name:     "unknown server",
			server:   MCPServer{},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.server.MCPServerType()
			if got != tt.expected {
				t.Errorf("MCPServerType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMCPServerIsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		server   MCPServer
		expected bool
	}{
		{
			name:     "with command",
			server:   MCPServer{Command: "my-mcp"},
			expected: true,
		},
		{
			name:     "with url",
			server:   MCPServer{URL: "https://example.com/mcp"},
			expected: true,
		},
		{
			name:     "empty",
			server:   MCPServer{},
			expected: false,
		},
		{
			name:     "only env",
			server:   MCPServer{Env: map[string]string{"KEY": "value"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.server.IsConfigured()
			if got != tt.expected {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMergeMCPConfigs(t *testing.T) {
	config1 := &MCPConfig{
		McpServers: map[string]MCPServer{
			"server1": {URL: "https://s1.com"},
			"shared":  {URL: "https://original.com"},
		},
	}
	config2 := &MCPConfig{
		McpServers: map[string]MCPServer{
			"server2": {Command: "s2"},
			"shared":  {URL: "https://override.com"}, // Should override
		},
	}

	result := MergeMCPConfigs(config1, config2)

	if len(result.McpServers) != 3 {
		t.Errorf("expected 3 servers, got %d", len(result.McpServers))
	}
	if result.McpServers["server1"].URL != "https://s1.com" {
		t.Error("server1 should be preserved")
	}
	if result.McpServers["server2"].Command != "s2" {
		t.Error("server2 should be added")
	}
	if result.McpServers["shared"].URL != "https://override.com" {
		t.Error("shared should be overridden by later config")
	}
}

func TestMergeMCPConfigs_NilHandling(t *testing.T) {
	config1 := &MCPConfig{
		McpServers: map[string]MCPServer{
			"server1": {URL: "https://s1.com"},
		},
	}

	// Should handle nil configs
	result := MergeMCPConfigs(nil, config1, nil)
	if len(result.McpServers) != 1 {
		t.Errorf("expected 1 server, got %d", len(result.McpServers))
	}
}

func TestCleanOrphanClaudeConfig(t *testing.T) {
	t.Run("no claude dir", func(t *testing.T) {
		tmpDir := t.TempDir()

		cleaned, err := CleanOrphanClaudeConfig(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleaned {
			t.Error("should not report cleaned when no .claude dir")
		}
	})

	t.Run("claude dir but no cursor dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatal(err)
		}

		cleaned, err := CleanOrphanClaudeConfig(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleaned {
			t.Error("should not clean when no .cursor dir")
		}
	})

	t.Run("both dirs with only gastown files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .claude with only Gas Town files
		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create .cursor dir
		cursorDir := filepath.Join(tmpDir, ".cursor")
		if err := os.MkdirAll(cursorDir, 0755); err != nil {
			t.Fatal(err)
		}

		cleaned, err := CleanOrphanClaudeConfig(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cleaned {
			t.Error("should clean orphan .claude with only Gas Town files")
		}

		// Verify .claude is removed
		if _, err := os.Stat(claudeDir); !os.IsNotExist(err) {
			t.Error(".claude should be removed")
		}
	})

	t.Run("both dirs with user files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .claude with user files
		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		// Add a user file
		if err := os.WriteFile(filepath.Join(claudeDir, "my-custom-file.txt"), []byte("user data"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create .cursor dir
		cursorDir := filepath.Join(tmpDir, ".cursor")
		if err := os.MkdirAll(cursorDir, 0755); err != nil {
			t.Fatal(err)
		}

		cleaned, err := CleanOrphanClaudeConfig(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleaned {
			t.Error("should NOT clean .claude with user files")
		}

		// Verify .claude is preserved
		if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
			t.Error(".claude should be preserved")
		}
	})
}

func TestCleanOrphanCursorConfig(t *testing.T) {
	t.Run("no cursor dir", func(t *testing.T) {
		tmpDir := t.TempDir()

		cleaned, err := CleanOrphanCursorConfig(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleaned {
			t.Error("should not report cleaned when no .cursor dir")
		}
	})

	t.Run("cursor dir but no claude dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		cursorDir := filepath.Join(tmpDir, ".cursor")
		if err := os.MkdirAll(cursorDir, 0755); err != nil {
			t.Fatal(err)
		}

		cleaned, err := CleanOrphanCursorConfig(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleaned {
			t.Error("should not clean when no .claude dir")
		}
	})

	t.Run("both dirs with only gastown files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .cursor with only Gas Town files
		cursorDir := filepath.Join(tmpDir, ".cursor")
		rulesDir := filepath.Join(cursorDir, "rules")
		if err := os.MkdirAll(rulesDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(rulesDir, "gastown.mdc"), []byte("# rules"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cursorDir, "hooks.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create .claude dir
		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatal(err)
		}

		cleaned, err := CleanOrphanCursorConfig(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cleaned {
			t.Error("should clean orphan .cursor with only Gas Town files")
		}

		// Verify .cursor is removed
		if _, err := os.Stat(cursorDir); !os.IsNotExist(err) {
			t.Error(".cursor should be removed")
		}
	})

	t.Run("both dirs with user rules", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .cursor with user rules
		cursorDir := filepath.Join(tmpDir, ".cursor")
		rulesDir := filepath.Join(cursorDir, "rules")
		if err := os.MkdirAll(rulesDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(rulesDir, "gastown.mdc"), []byte("# rules"), 0644); err != nil {
			t.Fatal(err)
		}
		// Add a user rule
		if err := os.WriteFile(filepath.Join(rulesDir, "my-custom-rules.mdc"), []byte("# custom"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create .claude dir
		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatal(err)
		}

		cleaned, err := CleanOrphanCursorConfig(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleaned {
			t.Error("should NOT clean .cursor with user rules")
		}

		// Verify .cursor is preserved
		if _, err := os.Stat(cursorDir); os.IsNotExist(err) {
			t.Error(".cursor should be preserved")
		}
	})
}

func TestCleanOrphanAgentConfigs(t *testing.T) {
	t.Run("cursor agent cleans claude", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Set up orphan .claude
		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		// Set up active .cursor
		cursorDir := filepath.Join(tmpDir, ".cursor")
		if err := os.MkdirAll(cursorDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Clean for cursor agent
		if err := CleanOrphanAgentConfigs(tmpDir, "cursor"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// .claude should be removed
		if _, err := os.Stat(claudeDir); !os.IsNotExist(err) {
			t.Error(".claude should be cleaned for cursor agent")
		}
	})

	t.Run("claude agent cleans cursor", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Set up orphan .cursor
		cursorDir := filepath.Join(tmpDir, ".cursor")
		rulesDir := filepath.Join(cursorDir, "rules")
		if err := os.MkdirAll(rulesDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(rulesDir, "gastown.mdc"), []byte("# rules"), 0644); err != nil {
			t.Fatal(err)
		}

		// Set up active .claude
		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Clean for claude agent
		if err := CleanOrphanAgentConfigs(tmpDir, "claude"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// .cursor should be removed
		if _, err := os.Stat(cursorDir); !os.IsNotExist(err) {
			t.Error(".cursor should be cleaned for claude agent")
		}
	})
}
