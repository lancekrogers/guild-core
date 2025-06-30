// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cost

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// CostTracker is the main cost tracking system that orchestrates
// multiple cost providers and aggregates real-time cost data
type CostTracker struct {
	providers  map[string]CostProvider
	storage    *CostStorage
	aggregator *CostAggregator
	calculator *CostCalculator
	config     *TrackerConfig
	mu         sync.RWMutex
}

// CostProvider defines the interface for provider-specific cost tracking
type CostProvider interface {
	GetRates(ctx context.Context) (*RateCard, error)
	TrackUsage(ctx context.Context, usage Usage) error
	GetCosts(ctx context.Context, period TimePeriod) (*Cost, error)
	GetProviderName() string
}

// Usage represents a single usage event to be tracked
type Usage struct {
	AgentID   string                 `json:"agent_id"`
	SessionID string                 `json:"session_id,omitempty"`
	Provider  string                 `json:"provider"`
	Resource  string                 `json:"resource"`
	Quantity  float64                `json:"quantity"`
	Unit      string                 `json:"unit"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Cost represents calculated cost information
type Cost struct {
	Provider  string     `json:"provider"`
	Resource  string     `json:"resource"`
	Quantity  float64    `json:"quantity"`
	UnitPrice float64    `json:"unit_price"`
	Total     float64    `json:"total"`
	Currency  string     `json:"currency"`
	Period    TimePeriod `json:"period"`
	Breakdown []CostItem `json:"breakdown,omitempty"`
}

// CostItem represents individual cost components
type CostItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Total       float64 `json:"total"`
}

// TimePeriod represents a time range for cost analysis
type TimePeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// RateCard contains pricing information for a provider
type RateCard struct {
	Provider    string                        `json:"provider"`
	Currency    string                        `json:"currency"`
	LastUpdated time.Time                     `json:"last_updated"`
	Rates       map[string]map[string]float64 `json:"rates"` // model -> token_type -> rate
}

// TrackerConfig contains configuration for the cost tracker
type TrackerConfig struct {
	UpdateInterval       time.Duration      `json:"update_interval"`
	RetentionPeriod      time.Duration      `json:"retention_period"`
	AggregationWindow    time.Duration      `json:"aggregation_window"`
	EnableRealTimeAlerts bool               `json:"enable_real_time_alerts"`
	BudgetLimits         map[string]float64 `json:"budget_limits"` // period -> limit
}

// NewCostTracker creates a new cost tracking system
func NewCostTracker(ctx context.Context, config *TrackerConfig) (*CostTracker, error) {
	ctx = observability.WithComponent(ctx, "cost.tracker")
	ctx = observability.WithOperation(ctx, "NewCostTracker")

	if config == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "config cannot be nil", nil).
			WithComponent("cost.tracker").
			WithOperation("NewCostTracker")
	}

	storage, err := NewCostStorage(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create cost storage").
			WithComponent("cost.tracker").
			WithOperation("NewCostTracker")
	}

	calculator, err := NewCostCalculator(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create cost calculator").
			WithComponent("cost.tracker").
			WithOperation("NewCostTracker")
	}

	tracker := &CostTracker{
		providers:  make(map[string]CostProvider),
		storage:    storage,
		calculator: calculator,
		config:     config,
	}

	aggregator, err := NewCostAggregator(ctx, tracker, config.AggregationWindow)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create cost aggregator").
			WithComponent("cost.tracker").
			WithOperation("NewCostTracker")
	}
	tracker.aggregator = aggregator

	return tracker, nil
}

// RegisterProvider registers a new cost provider
func (ct *CostTracker) RegisterProvider(ctx context.Context, provider CostProvider) error {
	ctx = observability.WithComponent(ctx, "cost.tracker")
	ctx = observability.WithOperation(ctx, "RegisterProvider")

	if provider == nil {
		return gerror.New(gerror.ErrCodeValidation, "provider cannot be nil", nil).
			WithComponent("cost.tracker").
			WithOperation("RegisterProvider")
	}

	ct.mu.Lock()
	defer ct.mu.Unlock()

	providerName := provider.GetProviderName()
	if providerName == "" {
		return gerror.New(gerror.ErrCodeValidation, "provider name cannot be empty", nil).
			WithComponent("cost.tracker").
			WithOperation("RegisterProvider")
	}

	ct.providers[providerName] = provider
	return nil
}

// TrackUsage records a usage event
func (ct *CostTracker) TrackUsage(ctx context.Context, usage Usage) error {
	ctx = observability.WithComponent(ctx, "cost.tracker")
	ctx = observability.WithOperation(ctx, "TrackUsage")

	if err := ct.validateUsage(usage); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid usage").
			WithComponent("cost.tracker").
			WithOperation("TrackUsage").
			WithDetails("agent_id", usage.AgentID).
			WithDetails("provider", usage.Provider)
	}

	ct.mu.RLock()
	provider, exists := ct.providers[usage.Provider]
	ct.mu.RUnlock()

	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "provider not found", nil).
			WithComponent("cost.tracker").
			WithOperation("TrackUsage").
			WithDetails("provider", usage.Provider)
	}

	// Track usage with provider
	if err := provider.TrackUsage(ctx, usage); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeProvider, "provider failed to track usage").
			WithComponent("cost.tracker").
			WithOperation("TrackUsage").
			WithDetails("provider", usage.Provider)
	}

	// Store usage record
	if err := ct.storage.StoreUsage(ctx, usage); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to store usage").
			WithComponent("cost.tracker").
			WithOperation("TrackUsage")
	}

	return nil
}

// GetCurrentCosts returns current cost summary
func (ct *CostTracker) GetCurrentCosts(ctx context.Context) (*CostSummary, error) {
	ctx = observability.WithComponent(ctx, "cost.tracker")
	ctx = observability.WithOperation(ctx, "GetCurrentCosts")

	return ct.aggregator.GetCurrentCosts(ctx)
}

// GetHistoricalCosts returns historical cost data
func (ct *CostTracker) GetHistoricalCosts(ctx context.Context, period TimePeriod) (*HistoricalCosts, error) {
	ctx = observability.WithComponent(ctx, "cost.tracker")
	ctx = observability.WithOperation(ctx, "GetHistoricalCosts")

	return ct.storage.GetHistoricalCosts(ctx, period)
}

// GetCostsByAgent returns cost breakdown by agent
func (ct *CostTracker) GetCostsByAgent(ctx context.Context, period TimePeriod) (map[string]float64, error) {
	ctx = observability.WithComponent(ctx, "cost.tracker")
	ctx = observability.WithOperation(ctx, "GetCostsByAgent")

	return ct.storage.GetCostsByAgent(ctx, period)
}

// GetCostsByProvider returns cost breakdown by provider
func (ct *CostTracker) GetCostsByProvider(ctx context.Context, period TimePeriod) (map[string]float64, error) {
	ctx = observability.WithComponent(ctx, "cost.tracker")
	ctx = observability.WithOperation(ctx, "GetCostsByProvider")

	return ct.storage.GetCostsByProvider(ctx, period)
}

// Start begins the cost tracking system
func (ct *CostTracker) Start(ctx context.Context) error {
	ctx = observability.WithComponent(ctx, "cost.tracker")
	ctx = observability.WithOperation(ctx, "Start")

	// Start aggregator
	go ct.aggregator.Start(ctx)

	return nil
}

// Stop gracefully shuts down the cost tracking system
func (ct *CostTracker) Stop(ctx context.Context) error {
	ctx = observability.WithComponent(ctx, "cost.tracker")
	ctx = observability.WithOperation(ctx, "Stop")

	return ct.aggregator.Stop(ctx)
}

// validateUsage validates a usage record
func (ct *CostTracker) validateUsage(usage Usage) error {
	if usage.AgentID == "" {
		return gerror.New(gerror.ErrCodeValidation, "agent_id cannot be empty", nil)
	}

	if usage.Provider == "" {
		return gerror.New(gerror.ErrCodeValidation, "provider cannot be empty", nil)
	}

	if usage.Resource == "" {
		return gerror.New(gerror.ErrCodeValidation, "resource cannot be empty", nil)
	}

	if usage.Quantity <= 0 {
		return gerror.New(gerror.ErrCodeValidation, "quantity must be positive", nil)
	}

	if usage.Unit == "" {
		return gerror.New(gerror.ErrCodeValidation, "unit cannot be empty", nil)
	}

	if usage.Timestamp.IsZero() {
		return gerror.New(gerror.ErrCodeValidation, "timestamp cannot be zero", nil)
	}

	return nil
}

// CostSummary provides a summary of current costs
type CostSummary struct {
	TotalCost       float64            `json:"total_cost"`
	CostByAgent     map[string]float64 `json:"cost_by_agent"`
	CostByProvider  map[string]float64 `json:"cost_by_provider"`
	CostByModel     map[string]float64 `json:"cost_by_model"`
	Period          TimePeriod         `json:"period"`
	HourlyRate      float64            `json:"hourly_rate"`
	DailyProjection float64            `json:"daily_projection"`
	Currency        string             `json:"currency"`
}

// HistoricalCosts contains historical cost data
type HistoricalCosts struct {
	Period     TimePeriod      `json:"period"`
	DataPoints []CostDataPoint `json:"data_points"`
	TotalCost  float64         `json:"total_cost"`
	Currency   string          `json:"currency"`
}

// CostDataPoint represents a single point in cost history
type CostDataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Cost      float64                `json:"cost"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// OpenAICostProvider implements CostProvider for OpenAI
type OpenAICostProvider struct {
	apiKey      string
	rateCard    *RateCard
	usageBuffer chan Usage
	calculator  *CostCalculator
	mu          sync.RWMutex
}

// NewOpenAICostProvider creates a new OpenAI cost provider
func NewOpenAICostProvider(ctx context.Context, apiKey string) (*OpenAICostProvider, error) {
	ctx = observability.WithComponent(ctx, "cost.openai_provider")
	ctx = observability.WithOperation(ctx, "NewOpenAICostProvider")

	if apiKey == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "api key cannot be empty", nil).
			WithComponent("cost.openai_provider").
			WithOperation("NewOpenAICostProvider")
	}

	calculator, err := NewCostCalculator(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create cost calculator").
			WithComponent("cost.openai_provider").
			WithOperation("NewOpenAICostProvider")
	}

	provider := &OpenAICostProvider{
		apiKey:      apiKey,
		usageBuffer: make(chan Usage, 1000),
		calculator:  calculator,
	}

	// Initialize rate card
	if err := provider.updateRateCard(ctx); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeProvider, "failed to initialize rate card").
			WithComponent("cost.openai_provider").
			WithOperation("NewOpenAICostProvider")
	}

	return provider, nil
}

// GetProviderName returns the provider name
func (ocp *OpenAICostProvider) GetProviderName() string {
	return "openai"
}

// TrackUsage tracks OpenAI usage
func (ocp *OpenAICostProvider) TrackUsage(ctx context.Context, usage Usage) error {
	ctx = observability.WithComponent(ctx, "cost.openai_provider")
	ctx = observability.WithOperation(ctx, "TrackUsage")

	if usage.Resource == "completion" {
		return ocp.trackCompletionUsage(ctx, usage)
	}

	return gerror.New(gerror.ErrCodeValidation, "unsupported resource type", nil).
		WithComponent("cost.openai_provider").
		WithOperation("TrackUsage").
		WithDetails("resource", usage.Resource)
}

// trackCompletionUsage tracks completion API usage
func (ocp *OpenAICostProvider) trackCompletionUsage(ctx context.Context, usage Usage) error {
	metadata := usage.Metadata

	inputTokens, ok := metadata["input_tokens"].(int)
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "missing input_tokens in metadata", nil).
			WithComponent("cost.openai_provider").
			WithOperation("trackCompletionUsage")
	}

	outputTokens, ok := metadata["output_tokens"].(int)
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "missing output_tokens in metadata", nil).
			WithComponent("cost.openai_provider").
			WithOperation("trackCompletionUsage")
	}

	model, ok := metadata["model"].(string)
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "missing model in metadata", nil).
			WithComponent("cost.openai_provider").
			WithOperation("trackCompletionUsage")
	}

	// Calculate costs
	inputCost, err := ocp.calculateTokenCost(ctx, inputTokens, model, "input")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to calculate input cost").
			WithComponent("cost.openai_provider").
			WithOperation("trackCompletionUsage")
	}

	outputCost, err := ocp.calculateTokenCost(ctx, outputTokens, model, "output")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to calculate output cost").
			WithComponent("cost.openai_provider").
			WithOperation("trackCompletionUsage")
	}

	totalCost := inputCost + outputCost

	// Store cost metadata
	usage.Metadata["input_cost"] = inputCost
	usage.Metadata["output_cost"] = outputCost
	usage.Metadata["total_cost"] = totalCost

	select {
	case ocp.usageBuffer <- usage:
		return nil
	default:
		return gerror.New(gerror.ErrCodeResourceLimit, "usage buffer full", nil).
			WithComponent("cost.openai_provider").
			WithOperation("trackCompletionUsage")
	}
}

// calculateTokenCost calculates cost for tokens
func (ocp *OpenAICostProvider) calculateTokenCost(ctx context.Context, tokens int, model, tokenType string) (float64, error) {
	ocp.mu.RLock()
	defer ocp.mu.RUnlock()

	if ocp.rateCard == nil {
		return 0, gerror.New(gerror.ErrCodeInternal, "rate card not initialized", nil).
			WithComponent("cost.openai_provider").
			WithOperation("calculateTokenCost")
	}

	modelRates, exists := ocp.rateCard.Rates[model]
	if !exists {
		return 0, gerror.New(gerror.ErrCodeNotFound, "model not found in rate card", nil).
			WithComponent("cost.openai_provider").
			WithOperation("calculateTokenCost").
			WithDetails("model", model)
	}

	rate, exists := modelRates[tokenType]
	if !exists {
		return 0, gerror.New(gerror.ErrCodeNotFound, "token type not found in rate card", nil).
			WithComponent("cost.openai_provider").
			WithOperation("calculateTokenCost").
			WithDetails("model", model).
			WithDetails("token_type", tokenType)
	}

	// Rate is per 1M tokens, convert to actual cost
	return float64(tokens) / 1000000.0 * rate, nil
}

// GetRates returns the current rate card
func (ocp *OpenAICostProvider) GetRates(ctx context.Context) (*RateCard, error) {
	ocp.mu.RLock()
	defer ocp.mu.RUnlock()

	if ocp.rateCard == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "rate card not initialized", nil).
			WithComponent("cost.openai_provider").
			WithOperation("GetRates")
	}

	return ocp.rateCard, nil
}

// GetCosts returns costs for a time period
func (ocp *OpenAICostProvider) GetCosts(ctx context.Context, period TimePeriod) (*Cost, error) {
	ctx = observability.WithComponent(ctx, "cost.openai_provider")
	ctx = observability.WithOperation(ctx, "GetCosts")

	// This would query stored usage data and calculate costs
	// For now, return a basic structure
	return &Cost{
		Provider: "openai",
		Resource: "completion",
		Period:   period,
		Currency: "USD",
	}, nil
}

// updateRateCard updates the OpenAI rate card with current pricing
func (ocp *OpenAICostProvider) updateRateCard(ctx context.Context) error {
	// Use the same rates as the existing providers system
	rateCard := &RateCard{
		Provider:    "openai",
		Currency:    "USD",
		LastUpdated: time.Now(),
		Rates: map[string]map[string]float64{
			"gpt-4": {
				"input":  30.0, // $30 per 1M tokens
				"output": 60.0, // $60 per 1M tokens
			},
			"gpt-4-turbo": {
				"input":  10.0, // $10 per 1M tokens
				"output": 30.0, // $30 per 1M tokens
			},
			"gpt-3.5-turbo": {
				"input":  0.5, // $0.50 per 1M tokens
				"output": 1.5, // $1.50 per 1M tokens
			},
		},
	}

	ocp.mu.Lock()
	ocp.rateCard = rateCard
	ocp.mu.Unlock()

	return nil
}

// AnthropicCostProvider implements CostProvider for Anthropic
type AnthropicCostProvider struct {
	apiKey      string
	rateCard    *RateCard
	usageBuffer chan Usage
	calculator  *CostCalculator
	mu          sync.RWMutex
}

// NewAnthropicCostProvider creates a new Anthropic cost provider
func NewAnthropicCostProvider(ctx context.Context, apiKey string) (*AnthropicCostProvider, error) {
	ctx = observability.WithComponent(ctx, "cost.anthropic_provider")
	ctx = observability.WithOperation(ctx, "NewAnthropicCostProvider")

	if apiKey == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "api key cannot be empty", nil).
			WithComponent("cost.anthropic_provider").
			WithOperation("NewAnthropicCostProvider")
	}

	calculator, err := NewCostCalculator(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create cost calculator").
			WithComponent("cost.anthropic_provider").
			WithOperation("NewAnthropicCostProvider")
	}

	provider := &AnthropicCostProvider{
		apiKey:      apiKey,
		usageBuffer: make(chan Usage, 1000),
		calculator:  calculator,
	}

	// Initialize rate card
	if err := provider.updateRateCard(ctx); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeProvider, "failed to initialize rate card").
			WithComponent("cost.anthropic_provider").
			WithOperation("NewAnthropicCostProvider")
	}

	return provider, nil
}

// GetProviderName returns the provider name
func (acp *AnthropicCostProvider) GetProviderName() string {
	return "anthropic"
}

// TrackUsage tracks Anthropic usage
func (acp *AnthropicCostProvider) TrackUsage(ctx context.Context, usage Usage) error {
	ctx = observability.WithComponent(ctx, "cost.anthropic_provider")
	ctx = observability.WithOperation(ctx, "TrackUsage")

	if usage.Resource == "completion" {
		return acp.trackCompletionUsage(ctx, usage)
	}

	return gerror.New(gerror.ErrCodeValidation, "unsupported resource type", nil).
		WithComponent("cost.anthropic_provider").
		WithOperation("TrackUsage").
		WithDetails("resource", usage.Resource)
}

// trackCompletionUsage tracks completion API usage
func (acp *AnthropicCostProvider) trackCompletionUsage(ctx context.Context, usage Usage) error {
	// Similar implementation to OpenAI with Anthropic-specific logic
	metadata := usage.Metadata

	inputTokens, ok := metadata["input_tokens"].(int)
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "missing input_tokens in metadata", nil).
			WithComponent("cost.anthropic_provider").
			WithOperation("trackCompletionUsage")
	}

	outputTokens, ok := metadata["output_tokens"].(int)
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "missing output_tokens in metadata", nil).
			WithComponent("cost.anthropic_provider").
			WithOperation("trackCompletionUsage")
	}

	model, ok := metadata["model"].(string)
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "missing model in metadata", nil).
			WithComponent("cost.anthropic_provider").
			WithOperation("trackCompletionUsage")
	}

	// Calculate costs
	inputCost, err := acp.calculateTokenCost(ctx, inputTokens, model, "input")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to calculate input cost").
			WithComponent("cost.anthropic_provider").
			WithOperation("trackCompletionUsage")
	}

	outputCost, err := acp.calculateTokenCost(ctx, outputTokens, model, "output")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to calculate output cost").
			WithComponent("cost.anthropic_provider").
			WithOperation("trackCompletionUsage")
	}

	totalCost := inputCost + outputCost

	// Store cost metadata
	usage.Metadata["input_cost"] = inputCost
	usage.Metadata["output_cost"] = outputCost
	usage.Metadata["total_cost"] = totalCost

	select {
	case acp.usageBuffer <- usage:
		return nil
	default:
		return gerror.New(gerror.ErrCodeResourceLimit, "usage buffer full", nil).
			WithComponent("cost.anthropic_provider").
			WithOperation("trackCompletionUsage")
	}
}

// calculateTokenCost calculates cost for tokens
func (acp *AnthropicCostProvider) calculateTokenCost(ctx context.Context, tokens int, model, tokenType string) (float64, error) {
	acp.mu.RLock()
	defer acp.mu.RUnlock()

	if acp.rateCard == nil {
		return 0, gerror.New(gerror.ErrCodeInternal, "rate card not initialized", nil).
			WithComponent("cost.anthropic_provider").
			WithOperation("calculateTokenCost")
	}

	modelRates, exists := acp.rateCard.Rates[model]
	if !exists {
		return 0, gerror.New(gerror.ErrCodeNotFound, "model not found in rate card", nil).
			WithComponent("cost.anthropic_provider").
			WithOperation("calculateTokenCost").
			WithDetails("model", model)
	}

	rate, exists := modelRates[tokenType]
	if !exists {
		return 0, gerror.New(gerror.ErrCodeNotFound, "token type not found in rate card", nil).
			WithComponent("cost.anthropic_provider").
			WithOperation("calculateTokenCost").
			WithDetails("model", model).
			WithDetails("token_type", tokenType)
	}

	// Rate is per 1M tokens, convert to actual cost
	return float64(tokens) / 1000000.0 * rate, nil
}

// GetRates returns the current rate card
func (acp *AnthropicCostProvider) GetRates(ctx context.Context) (*RateCard, error) {
	acp.mu.RLock()
	defer acp.mu.RUnlock()

	if acp.rateCard == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "rate card not initialized", nil).
			WithComponent("cost.anthropic_provider").
			WithOperation("GetRates")
	}

	return acp.rateCard, nil
}

// GetCosts returns costs for a time period
func (acp *AnthropicCostProvider) GetCosts(ctx context.Context, period TimePeriod) (*Cost, error) {
	ctx = observability.WithComponent(ctx, "cost.anthropic_provider")
	ctx = observability.WithOperation(ctx, "GetCosts")

	// This would query stored usage data and calculate costs
	// For now, return a basic structure
	return &Cost{
		Provider: "anthropic",
		Resource: "completion",
		Period:   period,
		Currency: "USD",
	}, nil
}

// updateRateCard updates the Anthropic rate card with current pricing
func (acp *AnthropicCostProvider) updateRateCard(ctx context.Context) error {
	// Use the same rates as the existing providers system
	rateCard := &RateCard{
		Provider:    "anthropic",
		Currency:    "USD",
		LastUpdated: time.Now(),
		Rates: map[string]map[string]float64{
			"claude-3-opus-20240229": {
				"input":  15.0, // $15 per 1M tokens
				"output": 75.0, // $75 per 1M tokens
			},
			"claude-3-sonnet-20240229": {
				"input":  3.0,  // $3 per 1M tokens
				"output": 15.0, // $15 per 1M tokens
			},
			"claude-3-haiku-20240307": {
				"input":  0.25, // $0.25 per 1M tokens
				"output": 1.25, // $1.25 per 1M tokens
			},
		},
	}

	acp.mu.Lock()
	acp.rateCard = rateCard
	acp.mu.Unlock()

	return nil
}
