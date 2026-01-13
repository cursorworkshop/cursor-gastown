package cursor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MCPConfig represents the structure of a Cursor mcp.json file.
// See: https://cursor.com/docs/context/mcp
type MCPConfig struct {
	// McpServers maps server names to their configurations.
	McpServers map[string]MCPServer `json:"mcpServers"`
}

// MCPServer represents an MCP server configuration.
// Supports both stdio (local command) and HTTP-based (remote URL) servers.
type MCPServer struct {
	// Type indicates the server transport type: "stdio" for local commands.
	// If URL is set and Command is empty, the type is implied to be HTTP/SSE.
	Type string `json:"type,omitempty"`

	// URL is the endpoint URL for the MCP server (for HTTP-based servers).
	URL string `json:"url,omitempty"`

	// Command is the command to run for stdio-based MCP servers.
	Command string `json:"command,omitempty"`

	// Args are command-line arguments for stdio-based servers.
	Args []string `json:"args,omitempty"`

	// Env contains environment variables for the server process.
	// Supports interpolation: ${env:NAME}, ${workspaceFolder}, ${userHome}
	Env map[string]string `json:"env,omitempty"`

	// EnvFile is the path to an environment file to load additional variables.
	// Supports interpolation: ${workspaceFolder}/.env
	EnvFile string `json:"envFile,omitempty"`

	// Headers contains HTTP headers for HTTP-based servers.
	// Supports interpolation: ${env:MY_TOKEN}
	Headers map[string]string `json:"headers,omitempty"`

	// Auth contains OAuth configuration for remote servers.
	Auth *MCPAuth `json:"auth,omitempty"`
}

// MCPAuth contains OAuth configuration for remote MCP servers.
type MCPAuth struct {
	// ClientID is the OAuth 2.0 Client ID from the MCP provider.
	ClientID string `json:"CLIENT_ID,omitempty"`

	// ClientSecret is the OAuth 2.0 Client Secret (for confidential clients).
	ClientSecret string `json:"CLIENT_SECRET,omitempty"`

	// Scopes are the OAuth scopes to request.
	Scopes []string `json:"scopes,omitempty"`
}

// MCPConfigPath returns the path to the workspace-level mcp.json.
// This is located at .cursor/mcp.json in the workspace root.
func MCPConfigPath(workDir string) string {
	return filepath.Join(workDir, ".cursor", "mcp.json")
}

// LoadMCPConfig loads an MCP configuration from the given path.
// Returns an empty config if the file doesn't exist.
func LoadMCPConfig(path string) (*MCPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &MCPConfig{
				McpServers: make(map[string]MCPServer),
			}, nil
		}
		return nil, fmt.Errorf("reading mcp.json: %w", err)
	}

	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing mcp.json: %w", err)
	}

	if config.McpServers == nil {
		config.McpServers = make(map[string]MCPServer)
	}

	return &config, nil
}

// SaveMCPConfig writes an MCP configuration to the given path.
func SaveMCPConfig(path string, config *MCPConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing mcp.json: %w", err)
	}

	return nil
}

// AddMCPServer adds or updates an MCP server in the workspace configuration.
func AddMCPServer(workDir, name string, server MCPServer) error {
	path := MCPConfigPath(workDir)

	config, err := LoadMCPConfig(path)
	if err != nil {
		return err
	}

	config.McpServers[name] = server

	return SaveMCPConfig(path, config)
}

// RemoveMCPServer removes an MCP server from the workspace configuration.
func RemoveMCPServer(workDir, name string) error {
	path := MCPConfigPath(workDir)

	config, err := LoadMCPConfig(path)
	if err != nil {
		return err
	}

	delete(config.McpServers, name)

	return SaveMCPConfig(path, config)
}

// EnsureGasTownMCPServers ensures Gas Town MCP servers are configured.
// This is a no-op if the servers are already configured.
func EnsureGasTownMCPServers(workDir string) error {
	path := MCPConfigPath(workDir)

	config, err := LoadMCPConfig(path)
	if err != nil {
		return err
	}

	// Check if we need to add any servers
	modified := false

	// Add Gas Town MCP server if not present
	// Note: This is a placeholder - Gas Town doesn't have an MCP server yet,
	// but this is where we'd configure it when available.
	if _, exists := config.McpServers["gastown"]; !exists {
		// Uncomment when Gas Town has an MCP server:
		// config.McpServers["gastown"] = MCPServer{
		// 	Command: "gt",
		// 	Args:    []string{"mcp", "serve"},
		// }
		// modified = true
		_ = exists // silence unused warning
	}

	if modified {
		return SaveMCPConfig(path, config)
	}

	return nil
}

// ListMCPServers returns a list of configured MCP server names.
func ListMCPServers(workDir string) ([]string, error) {
	path := MCPConfigPath(workDir)
	config, err := LoadMCPConfig(path)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(config.McpServers))
	for name := range config.McpServers {
		names = append(names, name)
	}
	return names, nil
}

// GetMCPServer returns a specific MCP server configuration.
// Returns nil if the server doesn't exist.
func GetMCPServer(workDir, name string) (*MCPServer, error) {
	path := MCPConfigPath(workDir)
	config, err := LoadMCPConfig(path)
	if err != nil {
		return nil, err
	}

	server, exists := config.McpServers[name]
	if !exists {
		return nil, nil
	}
	return &server, nil
}

// MCPServerType returns the type of an MCP server ("stdio" or "http").
func (s *MCPServer) MCPServerType() string {
	if s.Command != "" {
		return "stdio"
	}
	if s.URL != "" {
		return "http"
	}
	return "unknown"
}

// IsConfigured checks if the MCP server has minimum required configuration.
func (s *MCPServer) IsConfigured() bool {
	return s.Command != "" || s.URL != ""
}

// GlobalMCPConfigPath returns the path to the global mcp.json.
// This is located at ~/.cursor/mcp.json for user-wide configuration.
func GlobalMCPConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".cursor", "mcp.json"), nil
}

// MergeMCPConfigs merges multiple MCP configurations.
// Later configs override earlier ones for servers with the same name.
func MergeMCPConfigs(configs ...*MCPConfig) *MCPConfig {
	result := &MCPConfig{
		McpServers: make(map[string]MCPServer),
	}

	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		for name, server := range cfg.McpServers {
			result.McpServers[name] = server
		}
	}

	return result
}

// CleanOrphanClaudeConfig removes .claude/ directory that may be left behind
// when switching from Claude to Cursor agent. This prevents confusion and
// potential conflicts between agent configurations.
//
// Only removes .claude/ if:
// - The directory exists
// - A .cursor/ directory also exists (indicates Cursor is being used)
// - The .claude/ directory contains only Gas Town-managed files
//
// Returns true if cleanup was performed, false otherwise.
func CleanOrphanClaudeConfig(workDir string) (bool, error) {
	claudeDir := filepath.Join(workDir, ".claude")
	cursorDir := filepath.Join(workDir, ".cursor")

	// Check if both directories exist
	claudeInfo, claudeErr := os.Stat(claudeDir)
	cursorInfo, cursorErr := os.Stat(cursorDir)

	if os.IsNotExist(claudeErr) {
		// No .claude directory, nothing to clean
		return false, nil
	}
	if claudeErr != nil {
		return false, fmt.Errorf("checking .claude directory: %w", claudeErr)
	}
	if !claudeInfo.IsDir() {
		return false, nil
	}

	if os.IsNotExist(cursorErr) {
		// No .cursor directory, don't clean (user might be using Claude)
		return false, nil
	}
	if cursorErr != nil {
		return false, fmt.Errorf("checking .cursor directory: %w", cursorErr)
	}
	if !cursorInfo.IsDir() {
		return false, nil
	}

	// Both directories exist - check if .claude/ only has Gas Town files
	if !isGasTownManagedClaudeDir(claudeDir) {
		// Has user files, don't touch it
		return false, nil
	}

	// Safe to remove - only Gas Town files
	if err := os.RemoveAll(claudeDir); err != nil {
		return false, fmt.Errorf("removing orphan .claude directory: %w", err)
	}

	return true, nil
}

// isGasTownManagedClaudeDir checks if a .claude/ directory only contains
// files that were created by Gas Town (safe to remove).
func isGasTownManagedClaudeDir(claudeDir string) bool {
	// Known Gas Town managed files
	gasTownFiles := map[string]bool{
		"settings.json": true,
		"commands":      true, // directory
	}

	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		name := entry.Name()
		if !gasTownFiles[name] {
			// Found a file we don't recognize - not safe to remove
			return false
		}
	}

	return true
}

// CleanOrphanCursorConfig removes .cursor/ directory that may be left behind
// when switching from Cursor to Claude agent. This is the inverse of
// CleanOrphanClaudeConfig.
//
// Only removes .cursor/ if:
// - The directory exists
// - A .claude/ directory also exists (indicates Claude is being used)
// - The .cursor/ directory contains only Gas Town-managed files
//
// Returns true if cleanup was performed, false otherwise.
func CleanOrphanCursorConfig(workDir string) (bool, error) {
	cursorDir := filepath.Join(workDir, ".cursor")
	claudeDir := filepath.Join(workDir, ".claude")

	// Check if both directories exist
	cursorInfo, cursorErr := os.Stat(cursorDir)
	claudeInfo, claudeErr := os.Stat(claudeDir)

	if os.IsNotExist(cursorErr) {
		// No .cursor directory, nothing to clean
		return false, nil
	}
	if cursorErr != nil {
		return false, fmt.Errorf("checking .cursor directory: %w", cursorErr)
	}
	if !cursorInfo.IsDir() {
		return false, nil
	}

	if os.IsNotExist(claudeErr) {
		// No .claude directory, don't clean (user might be using Cursor)
		return false, nil
	}
	if claudeErr != nil {
		return false, fmt.Errorf("checking .claude directory: %w", claudeErr)
	}
	if !claudeInfo.IsDir() {
		return false, nil
	}

	// Both directories exist - check if .cursor/ only has Gas Town files
	if !isGasTownManagedCursorDir(cursorDir) {
		// Has user files, don't touch it
		return false, nil
	}

	// Safe to remove - only Gas Town files
	if err := os.RemoveAll(cursorDir); err != nil {
		return false, fmt.Errorf("removing orphan .cursor directory: %w", err)
	}

	return true, nil
}

// isGasTownManagedCursorDir checks if a .cursor/ directory only contains
// files that were created by Gas Town (safe to remove).
func isGasTownManagedCursorDir(cursorDir string) bool {
	// Known Gas Town managed files/directories
	gasTownFiles := map[string]bool{
		"rules":      true, // directory with gastown.mdc
		"hooks":      true, // directory with hook scripts
		"hooks.json": true,
		"mcp.json":   true,
	}

	entries, err := os.ReadDir(cursorDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		name := entry.Name()
		if !gasTownFiles[name] {
			// Found a file we don't recognize - not safe to remove
			return false
		}
	}

	// Also check that rules/ only contains gastown.mdc
	rulesDir := filepath.Join(cursorDir, "rules")
	if entries, err := os.ReadDir(rulesDir); err == nil {
		for _, entry := range entries {
			if entry.Name() != "gastown.mdc" {
				return false
			}
		}
	}

	return true
}

// CleanOrphanAgentConfigs cleans up orphan agent configurations based on
// the active agent type. Call this when switching agents or during cleanup.
func CleanOrphanAgentConfigs(workDir, activeAgent string) error {
	switch strings.ToLower(activeAgent) {
	case "cursor":
		// Using Cursor, clean orphan Claude config
		if _, err := CleanOrphanClaudeConfig(workDir); err != nil {
			return err
		}
	case "claude":
		// Using Claude, clean orphan Cursor config
		if _, err := CleanOrphanCursorConfig(workDir); err != nil {
			return err
		}
	}
	return nil
}
