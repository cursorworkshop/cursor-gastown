package cursor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
// Currently a no-op as Gas Town does not yet have an MCP server.
func EnsureGasTownMCPServers(workDir string) error {
	// Future: Add Gas Town MCP server configuration here when available.
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

