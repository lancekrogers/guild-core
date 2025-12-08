// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cost

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// CostAggregator provides real-time cost aggregation and projection capabilities
type CostAggregator struct {
	tracker     *CostTracker
	window      time.Duration
	subscribers []chan CostUpdate
	projections *CostProjection
	mu          sync.RWMutex
	stopCh      chan struct{}
	started     bool
}

// CostUpdate represents a real-time cost update
type CostUpdate struct {
	Period         TimePeriod             `json:"period"`
	TotalCost      float64                `json:"total_cost"`
	CostByAgent    map[string]float64     `json:"cost_by_agent"`
	CostByModel    map[string]float64     `json:"cost_by_model"`
	CostByProvider map[string]float64     `json:"cost_by_provider"`
	Projections    *CostProjection        `json:"projections"`
	Timestamp      time.Time              `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// CostProjection provides cost forecasting
type CostProjection struct {
	HourlyRate      float64   `json:"hourly_rate"`
	DailyEstimate   float64   `json:"daily_estimate"`
	MonthlyEstimate float64   `json:"monthly_estimate"`
	BudgetRemaining float64   `json:"budget_remaining"`
	DaysUntilLimit  int       `json:"days_until_limit"`
	Confidence      float64   `json:"confidence"`
	LastUpdated     time.Time `json:"last_updated"`
}

// NewCostAggregator creates a new cost aggregator
func NewCostAggregator(ctx context.Context, tracker *CostTracker, window time.Duration) (*CostAggregator, error) {
	ctx = observability.WithComponent(ctx, "cost.aggregator")
	ctx = observability.WithOperation(ctx, "NewCostAggregator")

	if tracker == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "tracker cannot be nil", nil).
			WithComponent("cost.aggregator").
			WithOperation("NewCostAggregator")
	}

	if window <= 0 {
		window = 5 * time.Minute // Default aggregation window
	}

	return &CostAggregator{
		tracker:     tracker,
		window:      window,
		subscribers: make([]chan CostUpdate, 0),
		stopCh:      make(chan struct{}),
		projections: &CostProjection{},
	}, nil
}

// Start begins real-time cost aggregation
func (ca *CostAggregator) Start(ctx context.Context) {
	ctx = observability.WithComponent(ctx, "cost.aggregator")
	ctx = observability.WithOperation(ctx, "Start")

	ca.mu.Lock()
	if ca.started {
		ca.mu.Unlock()
		return
	}
	ca.started = true
	ca.mu.Unlock()

	ticker := time.NewTicker(ca.window)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ca.stopCh:
			return
		case <-ticker.C:
			if err := ca.aggregate(ctx); err != nil {
				// Log error but continue aggregation
				continue
			}
		}
	}
}

// Stop stops the cost aggregation
func (ca *CostAggregator) Stop(ctx context.Context) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	if !ca.started {
		return nil
	}

	ca.started = false
	close(ca.stopCh)
	return nil
}

// Subscribe adds a subscriber for cost updates
func (ca *CostAggregator) Subscribe(updateCh chan CostUpdate) {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	ca.subscribers = append(ca.subscribers, updateCh)
}

// GetCurrentCosts returns the current cost summary
func (ca *CostAggregator) GetCurrentCosts(ctx context.Context) (*CostSummary, error) {
	ctx = observability.WithComponent(ctx, "cost.aggregator")
	ctx = observability.WithOperation(ctx, "GetCurrentCosts")

	now := time.Now()
	period := TimePeriod{
		Start: now.Add(-ca.window),
		End:   now,
	}

	// Get costs by agent
	costByAgent, err := ca.tracker.GetCostsByAgent(ctx, period)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get costs by agent").
			WithComponent("cost.aggregator").
			WithOperation("GetCurrentCosts")
	}

	// Get costs by provider
	costByProvider, err := ca.tracker.GetCostsByProvider(ctx, period)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get costs by provider").
			WithComponent("cost.aggregator").
			WithOperation("GetCurrentCosts")
	}

	// Calculate total cost
	totalCost := 0.0
	for _, cost := range costByAgent {
		totalCost += cost
	}

	// Calculate hourly rate
	duration := period.End.Sub(period.Start)
	hourlyRate := totalCost / duration.Hours()

	return &CostSummary{
		TotalCost:       totalCost,
		CostByAgent:     costByAgent,
		CostByProvider:  costByProvider,
		CostByModel:     make(map[string]float64), // TODO: Implement model breakdown
		Period:          period,
		HourlyRate:      hourlyRate,
		DailyProjection: hourlyRate * 24,
		Currency:        "USD",
	}, nil
}

// aggregate performs cost aggregation and broadcasts updates
func (ca *CostAggregator) aggregate(ctx context.Context) error {
	ctx = observability.WithComponent(ctx, "cost.aggregator")
	ctx = observability.WithOperation(ctx, "aggregate")

	now := time.Now()
	period := TimePeriod{
		Start: now.Add(-ca.window),
		End:   now,
	}

	// Get costs from all providers
	totalCost := 0.0
	costByAgent := make(map[string]float64)
	costByModel := make(map[string]float64)
	costByProvider := make(map[string]float64)

	// Get cost breakdown by agent
	agentCosts, err := ca.tracker.GetCostsByAgent(ctx, period)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get agent costs").
			WithComponent("cost.aggregator").
			WithOperation("aggregate")
	}

	for agent, cost := range agentCosts {
		costByAgent[agent] = cost
		totalCost += cost
	}

	// Get cost breakdown by provider
	providerCosts, err := ca.tracker.GetCostsByProvider(ctx, period)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get provider costs").
			WithComponent("cost.aggregator").
			WithOperation("aggregate")
	}

	for provider, cost := range providerCosts {
		costByProvider[provider] = cost
	}

	// Calculate projections
	projections := ca.calculateProjections(ctx, totalCost, period)

	// Store projections
	ca.mu.Lock()
	ca.projections = projections
	ca.mu.Unlock()

	// Broadcast update
	update := CostUpdate{
		Period:         period,
		TotalCost:      totalCost,
		CostByAgent:    costByAgent,
		CostByModel:    costByModel,
		CostByProvider: costByProvider,
		Projections:    projections,
		Timestamp:      now,
	}

	ca.broadcastUpdate(update)

	return nil
}

// calculateProjections calculates cost projections
func (ca *CostAggregator) calculateProjections(ctx context.Context, currentCost float64, period TimePeriod) *CostProjection {
	duration := period.End.Sub(period.Start)
	hourlyRate := currentCost / duration.Hours()

	// Get budget limits from tracker config
	budgetRemaining := ca.getBudgetRemaining(ctx)

	// Calculate days until budget limit
	daysUntilLimit := 0
	if hourlyRate > 0 {
		dailyRate := hourlyRate * 24
		if dailyRate > 0 {
			daysUntilLimit = int(math.Ceil(budgetRemaining / dailyRate))
		}
	}

	// Calculate confidence based on data points
	confidence := ca.calculateConfidence(ctx, period)

	return &CostProjection{
		HourlyRate:      hourlyRate,
		DailyEstimate:   hourlyRate * 24,
		MonthlyEstimate: hourlyRate * 24 * 30,
		BudgetRemaining: budgetRemaining,
		DaysUntilLimit:  daysUntilLimit,
		Confidence:      confidence,
		LastUpdated:     time.Now(),
	}
}

// getBudgetRemaining returns remaining budget
func (ca *CostAggregator) getBudgetRemaining(ctx context.Context) float64 {
	// TODO: Implement budget tracking with config integration
	// For now, return a default value
	monthlyBudget := 1000.0

	// Get current month costs
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	period := TimePeriod{
		Start: monthStart,
		End:   now,
	}

	monthlySpent := 0.0
	if agentCosts, err := ca.tracker.GetCostsByAgent(ctx, period); err == nil {
		for _, cost := range agentCosts {
			monthlySpent += cost
		}
	}

	remaining := monthlyBudget - monthlySpent
	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// calculateConfidence calculates projection confidence
func (ca *CostAggregator) calculateConfidence(ctx context.Context, period TimePeriod) float64 {
	// Confidence decreases with shorter time periods and increases with more data points
	duration := period.End.Sub(period.Start)
	hours := duration.Hours()

	// Base confidence on time window
	baseConfidence := math.Min(hours/24.0, 1.0) // Max confidence at 24+ hours

	// TODO: Factor in data variance, historical accuracy, etc.
	// For now, return base confidence
	return baseConfidence
}

// broadcastUpdate sends update to all subscribers
func (ca *CostAggregator) broadcastUpdate(update CostUpdate) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	for _, ch := range ca.subscribers {
		select {
		case ch <- update:
			// Successfully sent
		default:
			// Channel is full, skip this subscriber
		}
	}
}

// GetProjections returns current cost projections
func (ca *CostAggregator) GetProjections(ctx context.Context) *CostProjection {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	projectionsCopy := *ca.projections
	return &projectionsCopy
}

// UpdateBudget updates budget limits for projections
func (ca *CostAggregator) UpdateBudget(ctx context.Context, monthlyLimit float64) error {
	ctx = observability.WithComponent(ctx, "cost.aggregator")
	ctx = observability.WithOperation(ctx, "UpdateBudget")

	if monthlyLimit < 0 {
		return gerror.New(gerror.ErrCodeValidation, "budget limit cannot be negative", nil).
			WithComponent("cost.aggregator").
			WithOperation("UpdateBudget").
			WithDetails("monthly_limit", monthlyLimit)
	}

	// TODO: Store budget in tracker config or database
	// For now, just trigger recalculation
	return ca.aggregate(ctx)
}
