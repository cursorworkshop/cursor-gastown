// Package council provides multi-model orchestration for Gas Town.
package council

import (
	"fmt"
	"sync"
)

// Router selects the optimal model for a given task based on role and complexity.
type Router struct {
	config *Config
	mu     sync.RWMutex

	// providerStatus tracks provider availability.
	providerStatus map[string]bool
}

// NewRouter creates a new model router with the given configuration.
func NewRouter(config *Config) *Router {
	if config == nil {
		config = DefaultCouncilConfig()
	}

	r := &Router{
		config:         config,
		providerStatus: make(map[string]bool),
	}

	// Initialize providers based on config availability
	for provider, pc := range config.Providers {
		available := true
		if pc != nil {
			available = pc.Enabled
		}
		r.providerStatus[provider] = available
	}

	return r
}

// RouteRequest represents a request for model routing.
type RouteRequest struct {
	// Role is the Gas Town role making the request.
	Role string

	// Task describes the task (optional, for complexity analysis).
	Task *TaskInfo

	// PreferredModel is an optional model override.
	PreferredModel string

	// ExcludeProviders lists providers to exclude (e.g., due to rate limits).
	ExcludeProviders []string
}

// TaskInfo provides information about the task for complexity analysis.
type TaskInfo struct {
	// FilesAffected is the number of files the task will touch.
	FilesAffected int

	// LinesChanged is the estimated lines of code changed.
	LinesChanged int

	// IsArchitectural indicates if the change affects architecture.
	IsArchitectural bool

	// HasTests indicates if tests need to be written.
	HasTests bool

	// Description is a text description of the task.
	Description string
}

// RouteResult contains the routing decision.
type RouteResult struct {
	// Model is the selected model.
	Model string

	// Provider is the provider for the model.
	Provider string

	// Rationale explains why this model was selected.
	Rationale string

	// Complexity is the assessed task complexity.
	Complexity ComplexityLevel

	// Fallback indicates if this is a fallback selection.
	Fallback bool

	// FallbackReason explains why fallback was needed.
	FallbackReason string
}

// Route selects the optimal model for a request.
func (r *Router) Route(req *RouteRequest) (*RouteResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := &RouteResult{}

	// Check for preferred model override
	if req.PreferredModel != "" && req.PreferredModel != "auto" {
		if r.isModelAvailable(req.PreferredModel, req.ExcludeProviders) {
			result.Model = req.PreferredModel
			result.Provider = ModelProvider(req.PreferredModel)
			result.Rationale = "User-specified model preference"
			return result, nil
		}
		result.FallbackReason = fmt.Sprintf("Preferred model %s unavailable", req.PreferredModel)
		result.Fallback = true
	}

	// Determine complexity
	result.Complexity = r.assessComplexity(req.Task)

	// Get role-specific model
	var model string
	if r.config.SupportsComplexityRouting(req.Role) {
		model = r.config.GetModelForComplexity(req.Role, result.Complexity)
		result.Rationale = fmt.Sprintf("Complexity-based routing: %s task", result.Complexity)
	} else {
		model = r.config.GetModelForRole(req.Role)
		result.Rationale = r.config.GetRationale(req.Role)
		if result.Rationale == "" {
			result.Rationale = "Role-based model selection"
		}
	}

	// Check availability and apply fallbacks
	if r.isModelAvailable(model, req.ExcludeProviders) {
		result.Model = model
		result.Provider = ModelProvider(model)
		return result, nil
	}

	// Try fallback chain
	fallbacks := r.config.GetFallbackChain(req.Role)
	for _, fb := range fallbacks {
		if r.isModelAvailable(fb, req.ExcludeProviders) {
			result.Model = fb
			result.Provider = ModelProvider(fb)
			result.Fallback = true
			if result.FallbackReason == "" {
				result.FallbackReason = fmt.Sprintf("Primary model %s unavailable", model)
			}
			return result, nil
		}
	}

	// Last resort: any available model
	for provider, pc := range r.config.Providers {
		if !r.providerStatus[provider] || contains(req.ExcludeProviders, provider) {
			continue
		}
		for _, m := range pc.Models {
			result.Model = m
			result.Provider = provider
			result.Fallback = true
			result.FallbackReason = "All preferred models unavailable, using emergency fallback"
			return result, nil
		}
	}

	return nil, fmt.Errorf("no available models for role %s", req.Role)
}

// assessComplexity determines the complexity level of a task.
func (r *Router) assessComplexity(task *TaskInfo) ComplexityLevel {
	if task == nil {
		return ComplexityMedium
	}

	score := 0

	// Files affected scoring
	switch {
	case task.FilesAffected >= 10:
		score += 3
	case task.FilesAffected >= 5:
		score += 2
	case task.FilesAffected >= 2:
		score += 1
	}

	// Lines changed scoring
	switch {
	case task.LinesChanged >= 500:
		score += 3
	case task.LinesChanged >= 200:
		score += 2
	case task.LinesChanged >= 50:
		score += 1
	}

	// Architectural changes
	if task.IsArchitectural {
		score += 3
	}

	// Tests required
	if task.HasTests {
		score += 1
	}

	// Map score to complexity
	switch {
	case score >= 6:
		return ComplexityHigh
	case score >= 3:
		return ComplexityMedium
	default:
		return ComplexityLow
	}
}

// isModelAvailable checks if a model is available.
func (r *Router) isModelAvailable(model string, excludeProviders []string) bool {
	provider := ModelProvider(model)

	// Check if provider is excluded
	if contains(excludeProviders, provider) {
		return false
	}

	// Check provider status
	if status, ok := r.providerStatus[provider]; ok && !status {
		return false
	}

	return true
}

// ModelProvider returns the provider for a model.
// Duplicated from cursor package to avoid circular imports.
func ModelProvider(model string) string {
	switch {
	case hasPrefix(model, "opus-", "sonnet-", "haiku-", "claude-"):
		return "anthropic"
	case hasPrefix(model, "gpt-", "o4-"):
		return "openai"
	case hasPrefix(model, "gemini-"):
		return "google"
	case model == "grok":
		return "xai"
	default:
		return "unknown"
	}
}

// SetProviderStatus updates a provider's availability status.
func (r *Router) SetProviderStatus(provider string, available bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providerStatus[provider] = available
}

// GetProviderStatus returns a provider's availability status.
func (r *Router) GetProviderStatus(provider string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providerStatus[provider]
}

// ReloadConfig reloads the router configuration.
func (r *Router) ReloadConfig(config *Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = config
}

// GetConfig returns the current configuration.
func (r *Router) GetConfig() *Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// hasPrefix checks if a string has any of the given prefixes.
func hasPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if len(s) >= len(p) && s[:len(p)] == p {
			return true
		}
	}
	return false
}

// QuickRoute is a convenience function for simple routing.
func QuickRoute(role string) (string, error) {
	config := DefaultCouncilConfig()
	router := NewRouter(config)
	result, err := router.Route(&RouteRequest{Role: role})
	if err != nil {
		return "", err
	}
	return result.Model, nil
}

// RouteWithComplexity routes with explicit complexity level.
func RouteWithComplexity(role string, complexity ComplexityLevel) (string, error) {
	config := DefaultCouncilConfig()
	router := NewRouter(config)
	task := &TaskInfo{}
	switch complexity {
	case ComplexityHigh:
		task.FilesAffected = 10
		task.LinesChanged = 600
		task.IsArchitectural = true
		task.HasTests = true
	case ComplexityLow:
		task.FilesAffected = 1
		task.LinesChanged = 10
	default:
		task.FilesAffected = 5
		task.LinesChanged = 200
		task.HasTests = true
	}
	result, err := router.Route(&RouteRequest{
		Role: role,
		Task: task,
	})
	if err != nil {
		return "", err
	}
	return result.Model, nil
}
