// Package council provides multi-model orchestration for Gas Town.
package council

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Pattern represents an orchestration pattern for multi-model execution.
type Pattern string

const (
	// PatternSingle uses a single model for the task.
	PatternSingle Pattern = "single"

	// PatternChain passes output through a sequence of models.
	PatternChain Pattern = "chain"

	// PatternEnsemble runs multiple models in parallel and votes on output.
	PatternEnsemble Pattern = "ensemble"

	// PatternFallback tries models in sequence until one succeeds.
	PatternFallback Pattern = "fallback"

	// PatternSpecialist routes to specialized models based on task type.
	PatternSpecialist Pattern = "specialist"
)

// ChainConfig configures a chain-of-models pattern.
type ChainConfig struct {
	// Steps defines the sequence of models to use.
	Steps []ChainStep `json:"steps" toml:"steps"`

	// PassContext determines if each step receives the full conversation history.
	PassContext bool `json:"pass_context" toml:"pass_context"`

	// StopOnError halts the chain if any step fails.
	StopOnError bool `json:"stop_on_error" toml:"stop_on_error"`
}

// ChainStep represents a single step in a chain.
type ChainStep struct {
	// Name identifies this step.
	Name string `json:"name" toml:"name"`

	// Model to use for this step.
	Model string `json:"model" toml:"model"`

	// Role to apply (e.g., "refinery" for code review).
	Role string `json:"role" toml:"role"`

	// Prompt template or instruction for this step.
	Prompt string `json:"prompt" toml:"prompt"`

	// TransformOutput applies a transformation to the output before passing to next step.
	TransformOutput string `json:"transform_output" toml:"transform_output"`
}

// EnsembleConfig configures an ensemble voting pattern.
type EnsembleConfig struct {
	// Models to run in parallel.
	Models []string `json:"models" toml:"models"`

	// VotingStrategy determines how to combine outputs.
	VotingStrategy VotingStrategy `json:"voting_strategy" toml:"voting_strategy"`

	// Threshold for consensus (0-1, percentage of models that must agree).
	Threshold float64 `json:"threshold" toml:"threshold"`

	// Timeout for waiting on all models.
	Timeout time.Duration `json:"timeout" toml:"timeout"`

	// MinResponses is the minimum number of responses required before voting.
	MinResponses int `json:"min_responses" toml:"min_responses"`
}

// VotingStrategy determines how ensemble outputs are combined.
type VotingStrategy string

const (
	// VoteMajority takes the most common response.
	VoteMajority VotingStrategy = "majority"

	// VoteConsensus requires all models to agree.
	VoteConsensus VotingStrategy = "consensus"

	// VoteWeighted weights votes by model confidence/quality scores.
	VoteWeighted VotingStrategy = "weighted"

	// VoteBest selects the best response based on quality metrics.
	VoteBest VotingStrategy = "best"
)

// ModelResponse represents a response from a single model.
type ModelResponse struct {
	Model      string        `json:"model"`
	Output     string        `json:"output"`
	Duration   time.Duration `json:"duration"`
	Tokens     int64         `json:"tokens"`
	Cost       float64       `json:"cost"`
	Success    bool          `json:"success"`
	Error      string        `json:"error,omitempty"`
	Confidence float64       `json:"confidence"` // 0-1, model's confidence in response
}

// ChainResult represents the result of a chain execution.
type ChainResult struct {
	Steps      []StepResult  `json:"steps"`
	FinalOutput string       `json:"final_output"`
	TotalDuration time.Duration `json:"total_duration"`
	TotalCost   float64       `json:"total_cost"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
}

// StepResult represents the result of a single chain step.
type StepResult struct {
	Name     string        `json:"name"`
	Model    string        `json:"model"`
	Input    string        `json:"input"`
	Output   string        `json:"output"`
	Duration time.Duration `json:"duration"`
	Success  bool          `json:"success"`
	Error    string        `json:"error,omitempty"`
}

// EnsembleResult represents the result of an ensemble execution.
type EnsembleResult struct {
	Responses    []ModelResponse `json:"responses"`
	Winner       string          `json:"winner"`
	WinnerOutput string          `json:"winner_output"`
	Votes        map[string]int  `json:"votes"`
	Agreement    float64         `json:"agreement"` // 0-1
	Duration     time.Duration   `json:"duration"`
	Success      bool            `json:"success"`
	Error        string          `json:"error,omitempty"`
}

// ModelExecutor executes prompts against models.
type ModelExecutor interface {
	Execute(ctx context.Context, model, prompt string) (*ModelResponse, error)
}

// ChainExecutor executes chain-of-models patterns.
type ChainExecutor struct {
	executor ModelExecutor
	config   *ChainConfig
}

// NewChainExecutor creates a new chain executor.
func NewChainExecutor(executor ModelExecutor, config *ChainConfig) *ChainExecutor {
	return &ChainExecutor{
		executor: executor,
		config:   config,
	}
}

// Execute runs the chain of models.
func (c *ChainExecutor) Execute(ctx context.Context, initialInput string) (*ChainResult, error) {
	result := &ChainResult{
		Steps: make([]StepResult, 0, len(c.config.Steps)),
	}

	startTime := time.Now()
	currentInput := initialInput

	for i, step := range c.config.Steps {
		stepResult := StepResult{
			Name:  step.Name,
			Model: step.Model,
			Input: currentInput,
		}

		// Build prompt for this step
		prompt := step.Prompt
		if prompt == "" {
			prompt = currentInput
		} else {
			// Replace {{input}} with current input
			prompt = strings.ReplaceAll(prompt, "{{input}}", currentInput)
		}

		// Execute step
		stepStart := time.Now()
		response, err := c.executor.Execute(ctx, step.Model, prompt)
		stepResult.Duration = time.Since(stepStart)

		if err != nil {
			stepResult.Success = false
			stepResult.Error = err.Error()
			result.Steps = append(result.Steps, stepResult)

			if c.config.StopOnError {
				result.Success = false
				result.Error = fmt.Sprintf("step %d (%s) failed: %s", i+1, step.Name, err.Error())
				result.TotalDuration = time.Since(startTime)
				return result, nil
			}
			continue
		}

		stepResult.Success = response.Success
		stepResult.Output = response.Output
		if !response.Success {
			stepResult.Error = response.Error
		}

		result.Steps = append(result.Steps, stepResult)
		result.TotalCost += response.Cost

		// Transform output if specified
		if step.TransformOutput != "" {
			currentInput = applyTransform(response.Output, step.TransformOutput)
		} else {
			currentInput = response.Output
		}
	}

	result.TotalDuration = time.Since(startTime)
	result.FinalOutput = currentInput
	result.Success = true

	// Check if any step failed
	for _, step := range result.Steps {
		if !step.Success {
			result.Success = false
			break
		}
	}

	return result, nil
}

// applyTransform applies a simple transformation to output.
func applyTransform(output, transform string) string {
	switch transform {
	case "extract_code":
		// Extract code blocks from markdown
		return extractCodeBlocks(output)
	case "first_line":
		// Return first line only
		if idx := strings.Index(output, "\n"); idx >= 0 {
			return output[:idx]
		}
		return output
	case "trim":
		return strings.TrimSpace(output)
	default:
		return output
	}
}

// extractCodeBlocks extracts code from markdown code blocks.
func extractCodeBlocks(s string) string {
	var blocks []string
	inBlock := false
	var current strings.Builder

	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "```") {
			if inBlock {
				blocks = append(blocks, current.String())
				current.Reset()
			}
			inBlock = !inBlock
			continue
		}
		if inBlock {
			if current.Len() > 0 {
				current.WriteString("\n")
			}
			current.WriteString(line)
		}
	}

	return strings.Join(blocks, "\n\n")
}

// EnsembleExecutor executes ensemble voting patterns.
type EnsembleExecutor struct {
	executor ModelExecutor
	config   *EnsembleConfig
}

// NewEnsembleExecutor creates a new ensemble executor.
func NewEnsembleExecutor(executor ModelExecutor, config *EnsembleConfig) *EnsembleExecutor {
	return &EnsembleExecutor{
		executor: executor,
		config:   config,
	}
}

// Execute runs models in parallel and votes on output.
func (e *EnsembleExecutor) Execute(ctx context.Context, prompt string) (*EnsembleResult, error) {
	result := &EnsembleResult{
		Responses: make([]ModelResponse, 0, len(e.config.Models)),
		Votes:     make(map[string]int),
	}

	startTime := time.Now()

	// Set up timeout
	timeout := e.config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute all models in parallel
	var wg sync.WaitGroup
	responseChan := make(chan ModelResponse, len(e.config.Models))

	for _, model := range e.config.Models {
		wg.Add(1)
		go func(m string) {
			defer wg.Done()

			response, err := e.executor.Execute(ctx, m, prompt)
			if err != nil {
				responseChan <- ModelResponse{
					Model:   m,
					Success: false,
					Error:   err.Error(),
				}
				return
			}
			response.Model = m
			responseChan <- *response
		}(model)
	}

	// Wait for all or timeout
	go func() {
		wg.Wait()
		close(responseChan)
	}()

	// Collect responses
	for response := range responseChan {
		result.Responses = append(result.Responses, response)
	}

	result.Duration = time.Since(startTime)

	// Check minimum responses
	successfulResponses := 0
	for _, r := range result.Responses {
		if r.Success {
			successfulResponses++
		}
	}

	minResponses := e.config.MinResponses
	if minResponses == 0 {
		minResponses = len(e.config.Models) / 2 + 1
	}

	if successfulResponses < minResponses {
		result.Success = false
		result.Error = fmt.Sprintf("insufficient responses: got %d, need %d", successfulResponses, minResponses)
		return result, nil
	}

	// Vote on output
	winner, agreement := e.vote(result.Responses)
	result.Winner = winner.Model
	result.WinnerOutput = winner.Output
	result.Agreement = agreement

	// Check threshold
	if agreement < e.config.Threshold {
		result.Success = false
		result.Error = fmt.Sprintf("agreement %.2f below threshold %.2f", agreement, e.config.Threshold)
		return result, nil
	}

	result.Success = true
	return result, nil
}

// vote determines the winning response based on voting strategy.
func (e *EnsembleExecutor) vote(responses []ModelResponse) (ModelResponse, float64) {
	switch e.config.VotingStrategy {
	case VoteConsensus:
		return e.voteConsensus(responses)
	case VoteWeighted:
		return e.voteWeighted(responses)
	case VoteBest:
		return e.voteBest(responses)
	default:
		return e.voteMajority(responses)
	}
}

// voteMajority selects the most common response.
func (e *EnsembleExecutor) voteMajority(responses []ModelResponse) (ModelResponse, float64) {
	// Normalize and count responses
	counts := make(map[string][]ModelResponse)
	for _, r := range responses {
		if !r.Success {
			continue
		}
		normalized := normalizeOutput(r.Output)
		counts[normalized] = append(counts[normalized], r)
	}

	// Find majority
	var maxCount int
	var maxKey string
	for key, resps := range counts {
		if len(resps) > maxCount {
			maxCount = len(resps)
			maxKey = key
		}
	}

	if maxKey == "" {
		return ModelResponse{}, 0
	}

	successCount := 0
	for _, r := range responses {
		if r.Success {
			successCount++
		}
	}

	agreement := float64(maxCount) / float64(successCount)
	return counts[maxKey][0], agreement
}

// voteConsensus requires all models to agree.
func (e *EnsembleExecutor) voteConsensus(responses []ModelResponse) (ModelResponse, float64) {
	var firstOutput string
	var firstResponse ModelResponse
	allAgree := true

	for _, r := range responses {
		if !r.Success {
			continue
		}

		normalized := normalizeOutput(r.Output)
		if firstOutput == "" {
			firstOutput = normalized
			firstResponse = r
		} else if normalized != firstOutput {
			allAgree = false
		}
	}

	if allAgree && firstOutput != "" {
		return firstResponse, 1.0
	}

	// Fall back to majority voting
	return e.voteMajority(responses)
}

// voteWeighted weights votes by confidence scores.
func (e *EnsembleExecutor) voteWeighted(responses []ModelResponse) (ModelResponse, float64) {
	// Group by normalized output
	weights := make(map[string]float64)
	groups := make(map[string][]ModelResponse)

	for _, r := range responses {
		if !r.Success {
			continue
		}
		normalized := normalizeOutput(r.Output)
		confidence := r.Confidence
		if confidence == 0 {
			confidence = 0.5 // Default confidence
		}
		weights[normalized] += confidence
		groups[normalized] = append(groups[normalized], r)
	}

	// Find highest weighted
	var maxWeight float64
	var maxKey string
	var totalWeight float64
	for key, weight := range weights {
		totalWeight += weight
		if weight > maxWeight {
			maxWeight = weight
			maxKey = key
		}
	}

	if maxKey == "" {
		return ModelResponse{}, 0
	}

	agreement := maxWeight / totalWeight
	return groups[maxKey][0], agreement
}

// voteBest selects based on quality metrics.
func (e *EnsembleExecutor) voteBest(responses []ModelResponse) (ModelResponse, float64) {
	// Score each response
	type scoredResponse struct {
		response ModelResponse
		score    float64
	}

	var scored []scoredResponse
	for _, r := range responses {
		if !r.Success {
			continue
		}

		score := scoreResponse(r)
		scored = append(scored, scoredResponse{r, score})
	}

	if len(scored) == 0 {
		return ModelResponse{}, 0
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Agreement is based on score gap
	if len(scored) >= 2 {
		gap := scored[0].score - scored[1].score
		agreement := 0.5 + gap*0.5 // Larger gap = more confidence
		if agreement > 1 {
			agreement = 1
		}
		return scored[0].response, agreement
	}

	return scored[0].response, 1.0
}

// scoreResponse calculates a quality score for a response.
func scoreResponse(r ModelResponse) float64 {
	score := 0.0

	// Base score from confidence
	if r.Confidence > 0 {
		score += r.Confidence * 0.3
	}

	// Length score (prefer moderate length)
	length := len(r.Output)
	if length > 100 && length < 5000 {
		score += 0.2
	} else if length >= 50 && length <= 10000 {
		score += 0.1
	}

	// Speed score (faster is better)
	if r.Duration > 0 && r.Duration < 5*time.Second {
		score += 0.2
	} else if r.Duration < 15*time.Second {
		score += 0.1
	}

	// Cost efficiency
	if r.Cost > 0 && r.Cost < 0.01 {
		score += 0.2
	} else if r.Cost < 0.05 {
		score += 0.1
	}

	// Format quality
	if hasStructuredOutput(r.Output) {
		score += 0.1
	}

	return score
}

// normalizeOutput normalizes output for comparison.
func normalizeOutput(s string) string {
	// Lowercase
	s = strings.ToLower(s)

	// Remove whitespace variations
	s = strings.Join(strings.Fields(s), " ")

	// Remove common prefixes
	prefixes := []string{
		"here is", "here's", "the answer is", "i think",
		"based on", "let me", "sure,", "certainly,",
	}
	for _, prefix := range prefixes {
		s = strings.TrimPrefix(s, prefix)
	}

	return strings.TrimSpace(s)
}

// hasStructuredOutput checks if output has structured formatting.
func hasStructuredOutput(s string) bool {
	// Check for markdown headers, lists, or code blocks
	indicators := []string{"# ", "## ", "- ", "* ", "```", "1. "}
	for _, indicator := range indicators {
		if strings.Contains(s, indicator) {
			return true
		}
	}
	return false
}

// PredefinedChains contains common chain configurations.
var PredefinedChains = map[string]*ChainConfig{
	// Code review chain: Draft -> Review -> Refine
	"code-review": {
		PassContext: true,
		StopOnError: false,
		Steps: []ChainStep{
			{
				Name:   "initial-review",
				Model:  "sonnet-4.5",
				Role:   "refinery",
				Prompt: "Review this code for issues:\n\n{{input}}",
			},
			{
				Name:   "deep-analysis",
				Model:  "opus-4.5-thinking",
				Role:   "refinery",
				Prompt: "Based on this initial review, provide a detailed analysis:\n\n{{input}}",
			},
			{
				Name:   "final-summary",
				Model:  "gpt-5.2",
				Role:   "refinery",
				Prompt: "Summarize the code review findings concisely:\n\n{{input}}",
			},
		},
	},

	// Architecture decision chain
	"architecture": {
		PassContext: true,
		StopOnError: true,
		Steps: []ChainStep{
			{
				Name:   "gather-requirements",
				Model:  "gemini-3-flash",
				Prompt: "Extract the key requirements from:\n\n{{input}}",
			},
			{
				Name:   "design-options",
				Model:  "opus-4.5-thinking",
				Prompt: "Based on these requirements, propose 3 architecture options:\n\n{{input}}",
			},
			{
				Name:   "evaluate-tradeoffs",
				Model:  "sonnet-4.5",
				Prompt: "Evaluate the tradeoffs of each architecture option:\n\n{{input}}",
			},
			{
				Name:   "recommend",
				Model:  "gpt-5.2",
				Prompt: "Based on the analysis, recommend the best architecture:\n\n{{input}}",
			},
		},
	},

	// Bug fix chain
	"bug-fix": {
		PassContext: true,
		StopOnError: false,
		Steps: []ChainStep{
			{
				Name:   "diagnose",
				Model:  "sonnet-4.5",
				Role:   "polecat",
				Prompt: "Diagnose the root cause of this bug:\n\n{{input}}",
			},
			{
				Name:   "propose-fix",
				Model:  "gpt-5.2",
				Role:   "polecat",
				Prompt: "Based on this diagnosis, propose a fix:\n\n{{input}}",
				TransformOutput: "extract_code",
			},
			{
				Name:   "verify-fix",
				Model:  "gemini-3-flash",
				Role:   "witness",
				Prompt: "Verify this proposed fix addresses the bug:\n\n{{input}}",
			},
		},
	},
}

// PredefinedEnsembles contains common ensemble configurations.
var PredefinedEnsembles = map[string]*EnsembleConfig{
	// Critical decision ensemble
	"critical-decision": {
		Models:         []string{"opus-4.5-thinking", "gpt-5.2", "sonnet-4.5"},
		VotingStrategy: VoteConsensus,
		Threshold:      0.66,
		Timeout:        120 * time.Second,
		MinResponses:   2,
	},

	// Fast consensus ensemble
	"fast-consensus": {
		Models:         []string{"sonnet-4.5", "gpt-5.2", "gemini-3-flash"},
		VotingStrategy: VoteMajority,
		Threshold:      0.5,
		Timeout:        30 * time.Second,
		MinResponses:   2,
	},

	// Quality-focused ensemble
	"quality": {
		Models:         []string{"opus-4.5-thinking", "gpt-5.2"},
		VotingStrategy: VoteBest,
		Threshold:      0.0, // No threshold for best strategy
		Timeout:        90 * time.Second,
		MinResponses:   1,
	},
}
