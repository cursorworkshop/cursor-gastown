// Package council provides multi-model orchestration for Gas Town.
package council

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// MetricsStore stores and retrieves model performance metrics.
type MetricsStore struct {
	mu      sync.RWMutex
	path    string
	metrics *Metrics
}

// Metrics contains all collected metrics.
type Metrics struct {
	Version     int                      `json:"version"`
	UpdatedAt   time.Time                `json:"updated_at"`
	ByRole      map[string]*RoleMetrics  `json:"by_role"`
	ByModel     map[string]*ModelMetrics `json:"by_model"`
	ByProvider  map[string]*ProviderMetrics `json:"by_provider"`
	TaskHistory []TaskMetric             `json:"task_history,omitempty"`
}

// RoleMetrics contains metrics for a specific Gas Town role.
type RoleMetrics struct {
	Role           string             `json:"role"`
	TotalTasks     int                `json:"total_tasks"`
	CompletedTasks int                `json:"completed_tasks"`
	FailedTasks    int                `json:"failed_tasks"`
	TotalDuration  time.Duration      `json:"total_duration_ms"`
	TotalTokens    int64              `json:"total_tokens"`
	TotalCost      float64            `json:"total_cost"`
	ModelUsage     map[string]int     `json:"model_usage"` // model -> count
	AvgDuration    time.Duration      `json:"avg_duration_ms"`
	SuccessRate    float64            `json:"success_rate"`
}

// ModelMetrics contains metrics for a specific model.
type ModelMetrics struct {
	Model          string        `json:"model"`
	Provider       string        `json:"provider"`
	TotalTasks     int           `json:"total_tasks"`
	CompletedTasks int           `json:"completed_tasks"`
	FailedTasks    int           `json:"failed_tasks"`
	TotalDuration  time.Duration `json:"total_duration_ms"`
	TotalTokens    int64         `json:"total_tokens"`
	TotalCost      float64       `json:"total_cost"`
	AvgDuration    time.Duration `json:"avg_duration_ms"`
	SuccessRate    float64       `json:"success_rate"`
	RoleUsage      map[string]int `json:"role_usage"` // role -> count
}

// ProviderMetrics contains metrics for a provider.
type ProviderMetrics struct {
	Provider       string        `json:"provider"`
	TotalTasks     int           `json:"total_tasks"`
	CompletedTasks int           `json:"completed_tasks"`
	FailedTasks    int           `json:"failed_tasks"`
	TotalCost      float64       `json:"total_cost"`
	RateLimitHits  int           `json:"rate_limit_hits"`
	AvgLatency     time.Duration `json:"avg_latency_ms"`
	Availability   float64       `json:"availability"` // 0-1
}

// TaskMetric records a single task execution.
type TaskMetric struct {
	ID          string        `json:"id"`
	Role        string        `json:"role"`
	Model       string        `json:"model"`
	Provider    string        `json:"provider"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at,omitempty"`
	Duration    time.Duration `json:"duration_ms"`
	Tokens      int64         `json:"tokens,omitempty"`
	Cost        float64       `json:"cost,omitempty"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
	Complexity  string        `json:"complexity,omitempty"`
	Fallback    bool          `json:"fallback"`
}

// CurrentMetricsVersion is the current schema version.
const CurrentMetricsVersion = 1

// MetricsFileName is the default filename for metrics storage.
const MetricsFileName = "council-metrics.json"

// MaxTaskHistory is the maximum number of tasks to keep in history.
const MaxTaskHistory = 1000

// NewMetricsStore creates a new metrics store.
func NewMetricsStore(townRoot string) (*MetricsStore, error) {
	path := filepath.Join(townRoot, ".beads", MetricsFileName)

	store := &MetricsStore{
		path: path,
		metrics: &Metrics{
			Version:    CurrentMetricsVersion,
			ByRole:     make(map[string]*RoleMetrics),
			ByModel:    make(map[string]*ModelMetrics),
			ByProvider: make(map[string]*ProviderMetrics),
		},
	}

	// Load existing metrics if available
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading metrics: %w", err)
	}

	return store, nil
}

// load reads metrics from disk.
func (s *MetricsStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	var metrics Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return fmt.Errorf("parsing metrics: %w", err)
	}

	s.mu.Lock()
	s.metrics = &metrics
	s.mu.Unlock()

	return nil
}

// save writes metrics to disk.
func (s *MetricsStore) save() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.metrics, "", "  ")
	s.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("marshaling metrics: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("creating metrics directory: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("writing metrics: %w", err)
	}

	return nil
}

// RecordTask records a task execution.
func (s *MetricsStore) RecordTask(task TaskMetric) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure maps are initialized
	if s.metrics.ByRole == nil {
		s.metrics.ByRole = make(map[string]*RoleMetrics)
	}
	if s.metrics.ByModel == nil {
		s.metrics.ByModel = make(map[string]*ModelMetrics)
	}
	if s.metrics.ByProvider == nil {
		s.metrics.ByProvider = make(map[string]*ProviderMetrics)
	}

	// Update role metrics
	rm := s.metrics.ByRole[task.Role]
	if rm == nil {
		rm = &RoleMetrics{
			Role:       task.Role,
			ModelUsage: make(map[string]int),
		}
		s.metrics.ByRole[task.Role] = rm
	}
	rm.TotalTasks++
	if task.Success {
		rm.CompletedTasks++
	} else {
		rm.FailedTasks++
	}
	rm.TotalDuration += task.Duration
	rm.TotalTokens += task.Tokens
	rm.TotalCost += task.Cost
	rm.ModelUsage[task.Model]++
	rm.AvgDuration = rm.TotalDuration / time.Duration(rm.TotalTasks)
	if rm.TotalTasks > 0 {
		rm.SuccessRate = float64(rm.CompletedTasks) / float64(rm.TotalTasks)
	}

	// Update model metrics
	mm := s.metrics.ByModel[task.Model]
	if mm == nil {
		mm = &ModelMetrics{
			Model:     task.Model,
			Provider:  task.Provider,
			RoleUsage: make(map[string]int),
		}
		s.metrics.ByModel[task.Model] = mm
	}
	mm.TotalTasks++
	if task.Success {
		mm.CompletedTasks++
	} else {
		mm.FailedTasks++
	}
	mm.TotalDuration += task.Duration
	mm.TotalTokens += task.Tokens
	mm.TotalCost += task.Cost
	mm.RoleUsage[task.Role]++
	mm.AvgDuration = mm.TotalDuration / time.Duration(mm.TotalTasks)
	if mm.TotalTasks > 0 {
		mm.SuccessRate = float64(mm.CompletedTasks) / float64(mm.TotalTasks)
	}

	// Update provider metrics
	pm := s.metrics.ByProvider[task.Provider]
	if pm == nil {
		pm = &ProviderMetrics{
			Provider: task.Provider,
		}
		s.metrics.ByProvider[task.Provider] = pm
	}
	pm.TotalTasks++
	if task.Success {
		pm.CompletedTasks++
	} else {
		pm.FailedTasks++
	}
	pm.TotalCost += task.Cost
	if pm.TotalTasks > 0 {
		pm.Availability = float64(pm.CompletedTasks) / float64(pm.TotalTasks)
	}

	// Add to history
	s.metrics.TaskHistory = append(s.metrics.TaskHistory, task)
	if len(s.metrics.TaskHistory) > MaxTaskHistory {
		s.metrics.TaskHistory = s.metrics.TaskHistory[len(s.metrics.TaskHistory)-MaxTaskHistory:]
	}

	s.metrics.UpdatedAt = time.Now()

	// Save to disk
	s.mu.Unlock()
	err := s.save()
	s.mu.Lock()
	return err
}

// RecordRateLimit records a rate limit hit for a provider.
func (s *MetricsStore) RecordRateLimit(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.metrics.ByProvider == nil {
		s.metrics.ByProvider = make(map[string]*ProviderMetrics)
	}

	pm := s.metrics.ByProvider[provider]
	if pm == nil {
		pm = &ProviderMetrics{Provider: provider}
		s.metrics.ByProvider[provider] = pm
	}
	pm.RateLimitHits++

	s.metrics.UpdatedAt = time.Now()

	s.mu.Unlock()
	err := s.save()
	s.mu.Lock()
	return err
}

// GetMetrics returns a copy of all metrics.
func (s *MetricsStore) GetMetrics() *Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Deep copy
	data, err := json.Marshal(s.metrics)
	if err != nil {
		return s.metrics
	}

	var copy Metrics
	if err := json.Unmarshal(data, &copy); err != nil {
		return s.metrics
	}
	return &copy
}

// GetRoleMetrics returns metrics for a specific role.
func (s *MetricsStore) GetRoleMetrics(role string) *RoleMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metrics.ByRole[role]
}

// GetModelMetrics returns metrics for a specific model.
func (s *MetricsStore) GetModelMetrics(model string) *ModelMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metrics.ByModel[model]
}

// GetProviderMetrics returns metrics for a specific provider.
func (s *MetricsStore) GetProviderMetrics(provider string) *ProviderMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metrics.ByProvider[provider]
}

// GetRecentTasks returns the N most recent tasks.
func (s *MetricsStore) GetRecentTasks(n int) []TaskMetric {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := s.metrics.TaskHistory
	if len(history) <= n {
		result := make([]TaskMetric, len(history))
		copy(result, history)
		return result
	}

	result := make([]TaskMetric, n)
	copy(result, history[len(history)-n:])
	return result
}

// Summary returns a summary of all metrics.
type Summary struct {
	TotalTasks     int     `json:"total_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	TotalCost      float64 `json:"total_cost"`
	AvgSuccessRate float64 `json:"avg_success_rate"`
	TopModel       string  `json:"top_model"`
	TopProvider    string  `json:"top_provider"`
	CostSavings    float64 `json:"cost_savings_percent"`
}

// GetSummary returns a high-level summary of metrics.
func (s *MetricsStore) GetSummary() *Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := &Summary{}

	// Aggregate totals
	for _, rm := range s.metrics.ByRole {
		summary.TotalTasks += rm.TotalTasks
		summary.CompletedTasks += rm.CompletedTasks
		summary.TotalCost += rm.TotalCost
	}

	if summary.TotalTasks > 0 {
		summary.AvgSuccessRate = float64(summary.CompletedTasks) / float64(summary.TotalTasks)
	}

	// Find top model by task count
	maxTasks := 0
	for model, mm := range s.metrics.ByModel {
		if mm.TotalTasks > maxTasks {
			maxTasks = mm.TotalTasks
			summary.TopModel = model
		}
	}

	// Find top provider by task count
	maxProviderTasks := 0
	for provider, pm := range s.metrics.ByProvider {
		if pm.TotalTasks > maxProviderTasks {
			maxProviderTasks = pm.TotalTasks
			summary.TopProvider = provider
		}
	}

	// Calculate cost savings (compared to using Opus for everything)
	opusRate := 0.075 // $75/1M tokens estimated
	var estimatedOpusCost float64
	for _, rm := range s.metrics.ByRole {
		estimatedOpusCost += float64(rm.TotalTokens) * opusRate / 1000000
	}
	if estimatedOpusCost > 0 {
		summary.CostSavings = (1 - summary.TotalCost/estimatedOpusCost) * 100
	}

	return summary
}

// Reset clears all metrics.
func (s *MetricsStore) Reset() error {
	s.mu.Lock()
	s.metrics = &Metrics{
		Version:    CurrentMetricsVersion,
		UpdatedAt:  time.Now(),
		ByRole:     make(map[string]*RoleMetrics),
		ByModel:    make(map[string]*ModelMetrics),
		ByProvider: make(map[string]*ProviderMetrics),
	}
	s.mu.Unlock()

	return s.save()
}

// CompareModels returns a comparison of two models.
type ModelComparison struct {
	Model1       string        `json:"model1"`
	Model2       string        `json:"model2"`
	TaskDiff     int           `json:"task_diff"`
	SuccessDiff  float64       `json:"success_diff"`
	DurationDiff time.Duration `json:"duration_diff_ms"`
	CostDiff     float64       `json:"cost_diff"`
}

// CompareModels compares two models.
func (s *MetricsStore) CompareModels(model1, model2 string) *ModelComparison {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mm1 := s.metrics.ByModel[model1]
	mm2 := s.metrics.ByModel[model2]

	if mm1 == nil || mm2 == nil {
		return nil
	}

	return &ModelComparison{
		Model1:       model1,
		Model2:       model2,
		TaskDiff:     mm1.TotalTasks - mm2.TotalTasks,
		SuccessDiff:  mm1.SuccessRate - mm2.SuccessRate,
		DurationDiff: mm1.AvgDuration - mm2.AvgDuration,
		CostDiff:     mm1.TotalCost - mm2.TotalCost,
	}
}

// GetModelRanking returns models ranked by success rate.
func (s *MetricsStore) GetModelRanking() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type modelScore struct {
		model string
		score float64
	}

	var scores []modelScore
	for model, mm := range s.metrics.ByModel {
		scores = append(scores, modelScore{
			model: model,
			score: mm.SuccessRate,
		})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	result := make([]string, len(scores))
	for i, s := range scores {
		result[i] = s.model
	}
	return result
}
