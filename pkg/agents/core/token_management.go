// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// TokenCounter provides token counting for different providers
type TokenCounter interface {
	CountTokens(text string) (int, error)
	CountMessages(messages []ContextMessage) (int, error)
	GetModelLimits() ModelLimits
}

// ModelLimits defines token limits for a model
type ModelLimits struct {
	MaxContextTokens  int     `json:"max_context_tokens"`
	MaxResponseTokens int     `json:"max_response_tokens"`
	RecommendedBuffer int     `json:"recommended_buffer"`
	CostPer1KTokens   float64 `json:"cost_per_1k_tokens"`
	Provider          string  `json:"provider"`
	Model             string  `json:"model"`
}

// ContextWindow manages token usage within model limits
type ContextWindow struct {
	ID        string `json:"id"`
	AgentID   string `json:"agent_id"`
	SessionID string `json:"session_id"`

	// Limits
	ModelLimits    ModelLimits `json:"model_limits"`
	ReservedTokens int         `json:"reserved_tokens"` // For response
	SafetyMargin   float64     `json:"safety_margin"`   // 0.8 = 80% utilization

	// Usage tracking
	CurrentTokens   int64 `json:"current_tokens"`
	PeakTokens      int64 `json:"peak_tokens"`
	TotalTokensUsed int64 `json:"total_tokens_used"`

	// Content
	Messages     []WindowMessage `json:"messages"`
	SystemPrompt string          `json:"system_prompt"`

	// Compaction
	CompactionCount int        `json:"compaction_count"`
	LastCompaction  *time.Time `json:"last_compaction,omitempty"`

	// Metadata
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// WindowMessage represents a message in the context window
type WindowMessage struct {
	ID               string                 `json:"id"`
	Role             string                 `json:"role"`
	Content          string                 `json:"content"`
	TokenCount       int                    `json:"token_count"`
	Timestamp        time.Time              `json:"timestamp"`
	Compressible     bool                   `json:"compressible"`
	Priority         MessagePriority        `json:"priority"`
	ReasoningBlockID *string                `json:"reasoning_block_id,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// MessagePriority indicates importance for retention
type MessagePriority int

const (
	PriorityLow      MessagePriority = 0
	PriorityNormal   MessagePriority = 1
	PriorityHigh     MessagePriority = 2
	PriorityCritical MessagePriority = 3
)

// TokenManager provides comprehensive token management
type TokenManager struct {
	windows   sync.Map // map[string]*ContextWindow
	counters  map[string]TokenCounter
	compactor *ContextCompactor
	analyzer  *TokenAnalyzer
	predictor *TokenPredictor
	// metrics          *observability.Metrics // TODO: Update to use MetricsRegistry

	// Configuration
	config TokenConfig
	mu     sync.RWMutex
}

// TokenConfig configures token management
type TokenConfig struct {
	DefaultSafetyMargin   float64       `json:"default_safety_margin"`
	CompactionThreshold   float64       `json:"compaction_threshold"`
	EmergencyThreshold    float64       `json:"emergency_threshold"`
	EnablePrediction      bool          `json:"enable_prediction"`
	EnableAutoCompaction  bool          `json:"enable_auto_compaction"`
	MaxCompactionAttempts int           `json:"max_compaction_attempts"`
	TokenCountCache       bool          `json:"token_count_cache"`
	CacheTTL              time.Duration `json:"cache_ttl"`
}

// DefaultTokenConfig returns default configuration
func DefaultTokenConfig() TokenConfig {
	return TokenConfig{
		DefaultSafetyMargin:   0.85, // 85% utilization
		CompactionThreshold:   0.80, // Compact at 80%
		EmergencyThreshold:    0.95, // Emergency at 95%
		EnablePrediction:      true,
		EnableAutoCompaction:  true,
		MaxCompactionAttempts: 3,
		TokenCountCache:       true,
		CacheTTL:              5 * time.Minute,
	}
}

// NewTokenManager creates a new token manager
func NewTokenManager(config TokenConfig) *TokenManager {
	tm := &TokenManager{
		counters:  make(map[string]TokenCounter),
		compactor: NewContextCompactor(),
		analyzer:  NewTokenAnalyzer(),
		predictor: NewTokenPredictor(),
		config:    config,
	}

	// Register default counters
	tm.RegisterCounter("openai", NewOpenAITokenCounter())
	tm.RegisterCounter("anthropic", NewAnthropicTokenCounter())
	tm.RegisterCounter("ollama", NewOllamaTokenCounter())

	return tm
}

// RegisterCounter registers a token counter for a provider
func (tm *TokenManager) RegisterCounter(provider string, counter TokenCounter) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.counters[provider] = counter
}

// CreateWindow creates a new context window
func (tm *TokenManager) CreateWindow(ctx context.Context, agentID, sessionID, provider, model string) (*ContextWindow, error) {
	logger := observability.GetLogger(ctx)

	counter, exists := tm.counters[provider]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "token counter not found", nil).
			WithComponent("token_manager").
			WithDetails("provider", provider)
	}

	limits := counter.GetModelLimits()

	window := &ContextWindow{
		ID:             generateWindowID(),
		AgentID:        agentID,
		SessionID:      sessionID,
		ModelLimits:    limits,
		SafetyMargin:   tm.config.DefaultSafetyMargin,
		ReservedTokens: limits.RecommendedBuffer,
		Messages:       make([]WindowMessage, 0),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Metadata:       make(map[string]interface{}),
	}

	tm.windows.Store(window.ID, window)

	logger.InfoContext(ctx, "Created context window",
		"window_id", window.ID,
		"max_tokens", limits.MaxContextTokens,
		"provider", provider,
		"model", model)

	return window, nil
}

// AddMessage adds a message to the context window
func (tm *TokenManager) AddMessage(ctx context.Context, windowID string, message WindowMessage) error {
	windowI, exists := tm.windows.Load(windowID)
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "context window not found", nil).
			WithComponent("token_manager").
			WithDetails("window_id", windowID)
	}

	window := windowI.(*ContextWindow)

	// Count tokens if not provided
	if message.TokenCount == 0 {
		counter := tm.counters[window.ModelLimits.Provider]
		count, err := counter.CountTokens(message.Content)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to count tokens").
				WithComponent("token_manager")
		}
		message.TokenCount = count
	}

	// Check if addition would exceed limits
	newTotal := atomic.LoadInt64(&window.CurrentTokens) + int64(message.TokenCount)
	if err := tm.checkTokenSafety(window, newTotal); err != nil {
		// Try auto-compaction if enabled
		if tm.config.EnableAutoCompaction {
			if compactErr := tm.compactWindow(ctx, window); compactErr != nil {
				return gerror.Wrap(err, gerror.ErrCodeResourceLimit, "token limit exceeded and compaction failed").
					WithComponent("token_manager").
					WithDetails("compaction_error", compactErr.Error())
			}
			// Retry after compaction
			newTotal = atomic.LoadInt64(&window.CurrentTokens) + int64(message.TokenCount)
			if err := tm.checkTokenSafety(window, newTotal); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Add message
	window.Messages = append(window.Messages, message)
	atomic.AddInt64(&window.CurrentTokens, int64(message.TokenCount))
	atomic.AddInt64(&window.TotalTokensUsed, int64(message.TokenCount))

	// Update peak if necessary
	for {
		peak := atomic.LoadInt64(&window.PeakTokens)
		if newTotal <= peak || atomic.CompareAndSwapInt64(&window.PeakTokens, peak, newTotal) {
			break
		}
	}

	window.UpdatedAt = time.Now()

	// Record metrics
	tm.recordUsageMetrics(window)

	return nil
}

// checkTokenSafety verifies token usage is within safe limits
func (tm *TokenManager) checkTokenSafety(window *ContextWindow, newTotal int64) error {
	maxAllowed := int64(float64(window.ModelLimits.MaxContextTokens) * window.SafetyMargin)
	maxAllowed -= int64(window.ReservedTokens)

	if newTotal > maxAllowed {
		utilization := float64(newTotal) / float64(window.ModelLimits.MaxContextTokens)

		// Emergency threshold check
		if utilization >= tm.config.EmergencyThreshold {
			return gerror.New(gerror.ErrCodeResourceLimit, "emergency token limit exceeded", nil).
				WithComponent("token_manager").
				WithDetails("current_tokens", newTotal).
				WithDetails("max_allowed", maxAllowed).
				WithDetails("utilization", fmt.Sprintf("%.2f%%", utilization*100))
		}

		return gerror.New(gerror.ErrCodeResourceLimit, "token safety margin exceeded", nil).
			WithComponent("token_manager").
			WithDetails("current_tokens", newTotal).
			WithDetails("max_allowed", maxAllowed).
			WithDetails("utilization", fmt.Sprintf("%.2f%%", utilization*100))
	}

	return nil
}

// GetWindow retrieves a context window
func (tm *TokenManager) GetWindow(windowID string) (*ContextWindow, error) {
	windowI, exists := tm.windows.Load(windowID)
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "context window not found", nil).
			WithComponent("token_manager").
			WithDetails("window_id", windowID)
	}

	return windowI.(*ContextWindow), nil
}

// GetUtilization returns current token utilization
func (tm *TokenManager) GetUtilization(windowID string) (TokenUtilization, error) {
	window, err := tm.GetWindow(windowID)
	if err != nil {
		return TokenUtilization{}, err
	}

	current := atomic.LoadInt64(&window.CurrentTokens)
	max := int64(window.ModelLimits.MaxContextTokens)

	return TokenUtilization{
		WindowID:        windowID,
		CurrentTokens:   current,
		MaxTokens:       max,
		ReservedTokens:  int64(window.ReservedTokens),
		AvailableTokens: max - current - int64(window.ReservedTokens),
		Utilization:     float64(current) / float64(max),
		SafetyMargin:    window.SafetyMargin,
		NearLimit:       float64(current)/float64(max) > window.SafetyMargin,
		Cost:            tm.calculateCost(window),
	}, nil
}

// TokenUtilization represents current usage statistics
type TokenUtilization struct {
	WindowID        string  `json:"window_id"`
	CurrentTokens   int64   `json:"current_tokens"`
	MaxTokens       int64   `json:"max_tokens"`
	ReservedTokens  int64   `json:"reserved_tokens"`
	AvailableTokens int64   `json:"available_tokens"`
	Utilization     float64 `json:"utilization"`
	SafetyMargin    float64 `json:"safety_margin"`
	NearLimit       bool    `json:"near_limit"`
	Cost            float64 `json:"cost"`
}

// compactWindow performs context compaction
func (tm *TokenManager) compactWindow(ctx context.Context, window *ContextWindow) error {
	logger := observability.GetLogger(ctx)

	if window.CompactionCount >= tm.config.MaxCompactionAttempts {
		return gerror.New(gerror.ErrCodeResourceLimit, "max compaction attempts reached", nil).
			WithComponent("token_manager").
			WithDetails("attempts", window.CompactionCount)
	}

	startTokens := atomic.LoadInt64(&window.CurrentTokens)

	// Analyze messages for compaction
	analysis := tm.analyzer.AnalyzeWindow(window)

	// Perform compaction
	compacted, err := tm.compactor.Compact(ctx, window, analysis)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "compaction failed").
			WithComponent("token_manager")
	}

	// Update window with compacted messages
	window.Messages = compacted.Messages
	atomic.StoreInt64(&window.CurrentTokens, int64(compacted.TotalTokens))
	window.CompactionCount++
	now := time.Now()
	window.LastCompaction = &now

	savedTokens := startTokens - int64(compacted.TotalTokens)

	logger.InfoContext(ctx, "Context compaction completed",
		"window_id", window.ID,
		"saved_tokens", savedTokens,
		"compression_ratio", fmt.Sprintf("%.2f", float64(savedTokens)/float64(startTokens)),
		"remaining_tokens", compacted.TotalTokens)

	// TODO: Update to use MetricsRegistry
	// tm.metrics.RecordCounter("token_compaction_saved", float64(savedTokens))

	return nil
}

// PredictTokenUsage predicts future token usage
func (tm *TokenManager) PredictTokenUsage(windowID string, plannedMessages int) (TokenPrediction, error) {
	if !tm.config.EnablePrediction {
		return TokenPrediction{}, gerror.New(gerror.ErrCodeValidation, "prediction disabled", nil).
			WithComponent("token_manager")
	}

	window, err := tm.GetWindow(windowID)
	if err != nil {
		return TokenPrediction{}, err
	}

	prediction := tm.predictor.Predict(window, plannedMessages)

	// Check if predicted usage would exceed limits
	if prediction.WillExceedLimit {
		prediction.RecommendedAction = "compact_before_continuing"
		prediction.CompactionNeeded = true
	}

	return prediction, nil
}

// TokenPrediction represents predicted token usage
type TokenPrediction struct {
	CurrentTokens      int64   `json:"current_tokens"`
	PredictedTokens    int64   `json:"predicted_tokens"`
	AverageMessageSize int     `json:"average_message_size"`
	MessagesUntilLimit int     `json:"messages_until_limit"`
	WillExceedLimit    bool    `json:"will_exceed_limit"`
	CompactionNeeded   bool    `json:"compaction_needed"`
	RecommendedAction  string  `json:"recommended_action"`
	Confidence         float64 `json:"confidence"`
}

// calculateCost calculates the cost of tokens used
func (tm *TokenManager) calculateCost(window *ContextWindow) float64 {
	totalTokens := atomic.LoadInt64(&window.TotalTokensUsed)
	costPer1K := window.ModelLimits.CostPer1KTokens
	return float64(totalTokens) / 1000.0 * costPer1K
}

// recordUsageMetrics records token usage metrics
func (tm *TokenManager) recordUsageMetrics(window *ContextWindow) {
	// TODO: Update to use MetricsRegistry
	// current := atomic.LoadInt64(&window.CurrentTokens)
	// utilization := float64(current) / float64(window.ModelLimits.MaxContextTokens)

	// tm.metrics.RecordGauge("token_window_utilization", utilization,
	// 	"window_id", window.ID,
	// 	"provider", window.ModelLimits.Provider)

	// tm.metrics.RecordGauge("token_window_current", float64(current),
	// 	"window_id", window.ID)

	// tm.metrics.RecordCounter("token_total_used", float64(window.TotalTokensUsed),
	// 	"window_id", window.ID)
}

// ContextCompactor handles intelligent context compaction
type ContextCompactor struct {
	strategies []CompactionStrategy
	cache      *sync.Map
	// metrics    *observability.Metrics // TODO: Update to use MetricsRegistry
}

// CompactionStrategy defines a compaction approach
type CompactionStrategy interface {
	Name() string
	CanApply(window *ContextWindow, analysis *WindowAnalysis) bool
	Apply(ctx context.Context, window *ContextWindow, analysis *WindowAnalysis) (*CompactedWindow, error)
	Priority() int
}

// WindowAnalysis contains analysis results for compaction
type WindowAnalysis struct {
	TotalMessages        int
	CompressibleMessages int
	RedundantMessages    int
	LowPriorityMessages  int
	EstimatedSavings     int
	Recommendations      []string
}

// CompactedWindow represents the result of compaction
type CompactedWindow struct {
	Messages    []WindowMessage
	TotalTokens int
	TokensSaved int
	Strategy    string
	Summary     string
}

// NewContextCompactor creates a new compactor
func NewContextCompactor() *ContextCompactor {
	cc := &ContextCompactor{
		strategies: make([]CompactionStrategy, 0),
		cache:      &sync.Map{},
	}

	// Register default strategies
	cc.RegisterStrategy(&SummarizationStrategy{})
	cc.RegisterStrategy(&PriorityFilterStrategy{})
	cc.RegisterStrategy(&RedundancyRemovalStrategy{})
	cc.RegisterStrategy(&TemporalDecayStrategy{})

	return cc
}

// RegisterStrategy adds a compaction strategy
func (cc *ContextCompactor) RegisterStrategy(strategy CompactionStrategy) {
	cc.strategies = append(cc.strategies, strategy)
	// Sort by priority
	sort.Slice(cc.strategies, func(i, j int) bool {
		return cc.strategies[i].Priority() > cc.strategies[j].Priority()
	})
}

// Compact performs context compaction using the best strategy
func (cc *ContextCompactor) Compact(ctx context.Context, window *ContextWindow, analysis *WindowAnalysis) (*CompactedWindow, error) {
	logger := observability.GetLogger(ctx)

	startTime := time.Now()
	defer func() {
		// TODO: Update to use MetricsRegistry
		// cc.metrics.RecordDuration("context_compaction", time.Since(startTime))
		_ = startTime
	}()

	// Try strategies in priority order
	for _, strategy := range cc.strategies {
		if strategy.CanApply(window, analysis) {
			result, err := strategy.Apply(ctx, window, analysis)
			if err != nil {
				logger.WarnContext(ctx, "Compaction strategy failed",
					"strategy", strategy.Name(),
					"error", err)
				continue
			}

			logger.InfoContext(ctx, "Applied compaction strategy",
				"strategy", strategy.Name(),
				"tokens_saved", result.TokensSaved)

			return result, nil
		}
	}

	return nil, gerror.New(gerror.ErrCodeInternal, "no applicable compaction strategy", nil).
		WithComponent("context_compactor")
}

// TokenAnalyzer analyzes token usage patterns
type TokenAnalyzer struct {
	patterns map[string]func(*ContextWindow) float64
}

// NewTokenAnalyzer creates a new analyzer
func NewTokenAnalyzer() *TokenAnalyzer {
	ta := &TokenAnalyzer{
		patterns: make(map[string]func(*ContextWindow) float64),
	}

	// Register analysis patterns
	ta.patterns["redundancy"] = ta.analyzeRedundancy
	ta.patterns["temporal_decay"] = ta.analyzeTemporalDecay
	ta.patterns["priority_distribution"] = ta.analyzePriorityDistribution

	return ta
}

// AnalyzeWindow performs comprehensive window analysis
func (ta *TokenAnalyzer) AnalyzeWindow(window *ContextWindow) *WindowAnalysis {
	analysis := &WindowAnalysis{
		TotalMessages:   len(window.Messages),
		Recommendations: make([]string, 0),
	}

	// Count message types
	for _, msg := range window.Messages {
		if msg.Compressible {
			analysis.CompressibleMessages++
		}
		if msg.Priority == PriorityLow {
			analysis.LowPriorityMessages++
		}
	}

	// Estimate redundancy
	redundancyScore := ta.patterns["redundancy"](window)
	if redundancyScore > 0.3 {
		analysis.RedundantMessages = int(float64(len(window.Messages)) * redundancyScore)
		analysis.Recommendations = append(analysis.Recommendations, "High redundancy detected - consider deduplication")
	}

	// Estimate savings
	analysis.EstimatedSavings = ta.estimateSavings(window, analysis)

	return analysis
}

// analyzeRedundancy detects redundant content
func (ta *TokenAnalyzer) analyzeRedundancy(window *ContextWindow) float64 {
	// Simple similarity check (in production, use more sophisticated methods)
	contentMap := make(map[string]int)

	for _, msg := range window.Messages {
		// Hash content chunks
		words := strings.Fields(msg.Content)
		for i := 0; i < len(words)-5; i++ {
			chunk := strings.Join(words[i:i+5], " ")
			contentMap[chunk]++
		}
	}

	// Calculate redundancy score
	duplicates := 0
	for _, count := range contentMap {
		if count > 1 {
			duplicates += count - 1
		}
	}

	return float64(duplicates) / float64(len(window.Messages))
}

// analyzeTemporalDecay checks message age
func (ta *TokenAnalyzer) analyzeTemporalDecay(window *ContextWindow) float64 {
	if len(window.Messages) == 0 {
		return 0
	}

	now := time.Now()
	oldMessages := 0

	for _, msg := range window.Messages {
		age := now.Sub(msg.Timestamp)
		if age > 30*time.Minute {
			oldMessages++
		}
	}

	return float64(oldMessages) / float64(len(window.Messages))
}

// analyzePriorityDistribution checks priority balance
func (ta *TokenAnalyzer) analyzePriorityDistribution(window *ContextWindow) float64 {
	priorityCounts := make(map[MessagePriority]int)

	for _, msg := range window.Messages {
		priorityCounts[msg.Priority]++
	}

	// Return ratio of low priority messages
	return float64(priorityCounts[PriorityLow]) / float64(len(window.Messages))
}

// estimateSavings estimates potential token savings
func (ta *TokenAnalyzer) estimateSavings(window *ContextWindow, analysis *WindowAnalysis) int {
	savings := 0

	// Estimate savings from compressible messages
	for _, msg := range window.Messages {
		if msg.Compressible {
			savings += msg.TokenCount / 3 // Assume 3:1 compression
		}
		if msg.Priority == PriorityLow && len(window.Messages) > 20 {
			savings += msg.TokenCount // Can be removed
		}
	}

	return savings
}

// TokenPredictor predicts future token usage
type TokenPredictor struct {
	history *sync.Map // Historical patterns
}

// NewTokenPredictor creates a new predictor
func NewTokenPredictor() *TokenPredictor {
	return &TokenPredictor{
		history: &sync.Map{},
	}
}

// Predict predicts future token usage
func (tp *TokenPredictor) Predict(window *ContextWindow, plannedMessages int) TokenPrediction {
	// Calculate average message size
	totalTokens := 0
	for _, msg := range window.Messages {
		totalTokens += msg.TokenCount
	}

	avgSize := 150 // Default
	if len(window.Messages) > 0 {
		avgSize = totalTokens / len(window.Messages)
	}

	// Predict future usage
	currentTokens := atomic.LoadInt64(&window.CurrentTokens)
	predictedAdditional := int64(avgSize * plannedMessages)
	predictedTotal := currentTokens + predictedAdditional

	// Calculate messages until limit
	maxTokens := int64(window.ModelLimits.MaxContextTokens)
	availableTokens := maxTokens - currentTokens - int64(window.ReservedTokens)
	messagesUntilLimit := int(availableTokens / int64(avgSize))

	return TokenPrediction{
		CurrentTokens:      currentTokens,
		PredictedTokens:    predictedTotal,
		AverageMessageSize: avgSize,
		MessagesUntilLimit: messagesUntilLimit,
		WillExceedLimit:    predictedTotal > int64(float64(maxTokens)*window.SafetyMargin),
		Confidence:         0.75, // Simple prediction
	}
}

// Helper functions

func generateWindowID() string {
	return fmt.Sprintf("window_%d", time.Now().UnixNano())
}
