// Package council provides multi-model orchestration for Gas Town.
package council

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// FallbackManager handles provider availability and automatic fallback.
type FallbackManager struct {
	router         *Router
	mu             sync.RWMutex
	healthChecks   map[string]time.Time
	checkInterval  time.Duration
	failureCounts  map[string]int
	failureWindow  map[string][]time.Time
	circuitBreaker map[string]*CircuitBreaker
}

// CircuitBreaker implements circuit breaker pattern for providers.
type CircuitBreaker struct {
	// State is "closed" (normal), "open" (failing), or "half-open" (testing)
	State string

	// FailureCount is consecutive failures in current window
	FailureCount int

	// LastFailure is the timestamp of the last failure
	LastFailure time.Time

	// LastSuccess is the timestamp of the last success
	LastSuccess time.Time

	// OpenedAt is when the circuit was opened
	OpenedAt time.Time

	// Threshold is failures before opening
	Threshold int

	// ResetTimeout is how long to wait before testing again
	ResetTimeout time.Duration
}

// ProviderHealth represents the health status of a provider.
type ProviderHealth struct {
	Provider      string        `json:"provider"`
	Available     bool          `json:"available"`
	LastChecked   time.Time     `json:"last_checked"`
	ResponseTime  time.Duration `json:"response_time_ms"`
	FailureCount  int           `json:"failure_count"`
	CircuitState  string        `json:"circuit_state"`
	RateLimitHits int           `json:"rate_limit_hits"`
}

// ProviderEndpoints maps providers to their health check endpoints.
var ProviderEndpoints = map[string]string{
	"anthropic": "https://api.anthropic.com/v1/messages", // Will return 401 without auth, but proves reachability
	"openai":    "https://api.openai.com/v1/models",
	"google":    "https://generativelanguage.googleapis.com/v1/models",
}

// NewFallbackManager creates a new fallback manager.
func NewFallbackManager(router *Router) *FallbackManager {
	fm := &FallbackManager{
		router:         router,
		healthChecks:   make(map[string]time.Time),
		checkInterval:  5 * time.Minute,
		failureCounts:  make(map[string]int),
		failureWindow:  make(map[string][]time.Time),
		circuitBreaker: make(map[string]*CircuitBreaker),
	}

	// Initialize circuit breakers for all providers
	for provider := range router.config.Providers {
		fm.circuitBreaker[provider] = &CircuitBreaker{
			State:        "closed",
			Threshold:    5,
			ResetTimeout: 30 * time.Second,
		}
	}

	return fm
}

// CheckHealth performs a health check on a provider.
func (fm *FallbackManager) CheckHealth(ctx context.Context, provider string) (*ProviderHealth, error) {
	endpoint, ok := ProviderEndpoints[provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	health := &ProviderHealth{
		Provider:    provider,
		LastChecked: time.Now(),
	}

	// Create request with timeout
	req, err := http.NewRequestWithContext(ctx, "HEAD", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	health.ResponseTime = time.Since(start)

	if err != nil {
		health.Available = false
		fm.recordFailure(provider)
		return health, nil
	}
	defer resp.Body.Close()

	// 401/403 means the endpoint is reachable (auth failed, but API is up)
	// 429 means rate limited
	switch resp.StatusCode {
	case http.StatusOK, http.StatusUnauthorized, http.StatusForbidden:
		health.Available = true
		fm.recordSuccess(provider)
	case http.StatusTooManyRequests:
		health.Available = false
		health.RateLimitHits++
		fm.recordRateLimit(provider)
	default:
		health.Available = false
		fm.recordFailure(provider)
	}

	fm.mu.Lock()
	fm.healthChecks[provider] = time.Now()
	cb := fm.circuitBreaker[provider]
	health.CircuitState = cb.State
	health.FailureCount = cb.FailureCount
	fm.mu.Unlock()

	return health, nil
}

// recordFailure records a provider failure.
func (fm *FallbackManager) recordFailure(provider string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	cb, ok := fm.circuitBreaker[provider]
	if !ok {
		return
	}

	cb.FailureCount++
	cb.LastFailure = time.Now()

	// Check if we should open the circuit
	if cb.State == "closed" && cb.FailureCount >= cb.Threshold {
		cb.State = "open"
		cb.OpenedAt = time.Now()
		fm.router.SetProviderStatus(provider, false)
	}
}

// recordSuccess records a provider success.
func (fm *FallbackManager) recordSuccess(provider string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	cb, ok := fm.circuitBreaker[provider]
	if !ok {
		return
	}

	cb.LastSuccess = time.Now()

	// If half-open, close the circuit
	if cb.State == "half-open" {
		cb.State = "closed"
		cb.FailureCount = 0
		fm.router.SetProviderStatus(provider, true)
	} else if cb.State == "closed" {
		// Reset failure count on success
		cb.FailureCount = 0
	}
}

// recordRateLimit records a rate limit hit.
func (fm *FallbackManager) recordRateLimit(provider string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Add to failure window
	now := time.Now()
	fm.failureWindow[provider] = append(fm.failureWindow[provider], now)

	// Clean old entries (older than 1 minute)
	cutoff := now.Add(-time.Minute)
	var recent []time.Time
	for _, t := range fm.failureWindow[provider] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	fm.failureWindow[provider] = recent

	// If too many rate limits in window, temporarily disable
	if len(recent) >= 5 {
		cb := fm.circuitBreaker[provider]
		if cb != nil && cb.State == "closed" {
			cb.State = "open"
			cb.OpenedAt = now
			fm.router.SetProviderStatus(provider, false)
		}
	}
}

// MaybeRecover checks if open circuits should be tested.
func (fm *FallbackManager) MaybeRecover(ctx context.Context) {
	fm.mu.Lock()
	var toTest []string
	for provider, cb := range fm.circuitBreaker {
		if cb.State == "open" && time.Since(cb.OpenedAt) > cb.ResetTimeout {
			cb.State = "half-open"
			toTest = append(toTest, provider)
		}
	}
	fm.mu.Unlock()

	// Test half-open circuits
	for _, provider := range toTest {
		health, err := fm.CheckHealth(ctx, provider)
		if err != nil {
			continue
		}
		if !health.Available {
			// Re-open the circuit
			fm.mu.Lock()
			cb := fm.circuitBreaker[provider]
			cb.State = "open"
			cb.OpenedAt = time.Now()
			fm.mu.Unlock()
		}
	}
}

// GetAllHealth returns health status for all providers.
func (fm *FallbackManager) GetAllHealth(ctx context.Context) map[string]*ProviderHealth {
	result := make(map[string]*ProviderHealth)

	for provider := range fm.router.config.Providers {
		health, err := fm.CheckHealth(ctx, provider)
		if err != nil {
			health = &ProviderHealth{
				Provider:    provider,
				Available:   false,
				LastChecked: time.Now(),
			}
		}
		result[provider] = health
	}

	return result
}

// GetAvailableProviders returns a list of currently available providers.
func (fm *FallbackManager) GetAvailableProviders() []string {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	var available []string
	for provider, cb := range fm.circuitBreaker {
		if cb.State == "closed" || cb.State == "half-open" {
			available = append(available, provider)
		}
	}
	return available
}

// RouteWithFallback routes a request with automatic fallback handling.
func (fm *FallbackManager) RouteWithFallback(req *RouteRequest) (*RouteResult, error) {
	fm.mu.RLock()
	unavailable := make([]string, 0)
	for provider, cb := range fm.circuitBreaker {
		if cb.State == "open" {
			unavailable = append(unavailable, provider)
		}
	}
	fm.mu.RUnlock()

	// Add unavailable providers to exclude list
	req.ExcludeProviders = append(req.ExcludeProviders, unavailable...)

	return fm.router.Route(req)
}

// RecordRequestOutcome records the outcome of a request for circuit breaker.
func (fm *FallbackManager) RecordRequestOutcome(provider string, success bool, err error) {
	if success {
		fm.recordSuccess(provider)
	} else {
		// Check if this is a rate limit error
		if err != nil && isRateLimitError(err) {
			fm.recordRateLimit(provider)
		} else {
			fm.recordFailure(provider)
		}
	}
}

// isRateLimitError checks if an error indicates rate limiting.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains([]string{errStr}, "rate limit") ||
		contains([]string{errStr}, "429") ||
		contains([]string{errStr}, "too many requests")
}

// StartBackgroundRecovery starts a goroutine to periodically check for circuit recovery.
func (fm *FallbackManager) StartBackgroundRecovery(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(fm.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fm.MaybeRecover(ctx)
			}
		}
	}()
}

// Reset resets all circuit breakers to closed state.
func (fm *FallbackManager) Reset() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	for provider, cb := range fm.circuitBreaker {
		cb.State = "closed"
		cb.FailureCount = 0
		fm.router.SetProviderStatus(provider, true)
	}
	fm.failureWindow = make(map[string][]time.Time)
}
