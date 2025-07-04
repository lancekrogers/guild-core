// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package providers

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/providers"
)

// CostOptimizer manages cost optimization for provider usage
type CostOptimizer struct {
	manager         *ProviderManager
	budgetManager   *BudgetManager
	costPredictor   *CostPredictor
	usageTracker    *UsageTracker
	optimizations   []CostOptimization
	running         bool
	mu              sync.RWMutex
}

// BudgetManager manages budget constraints and enforcement
type BudgetManager struct {
	budgets         map[string]*Budget
	alerts          []*BudgetAlert
	enforcements    []*BudgetEnforcement
	mu              sync.RWMutex
}

// Budget represents a budget constraint
type Budget struct {
	ID              string
	Name            string
	Limit           float64
	Period          time.Duration
	Current         float64
	StartTime       time.Time
	AlertThresholds []float64
	HardLimit       bool
	Priority        BudgetPriority
}

// BudgetPriority represents budget priority levels
type BudgetPriority int

const (
	BudgetPriorityLow BudgetPriority = iota
	BudgetPriorityMedium
	BudgetPriorityHigh
	BudgetPriorityCritical
)

// BudgetAlert represents a budget alert
type BudgetAlert struct {
	ID        string
	BudgetID  string
	Threshold float64
	Triggered bool
	Time      time.Time
	Message   string
}

// BudgetEnforcement represents budget enforcement action
type BudgetEnforcement struct {
	ID        string
	BudgetID  string
	Action    EnforcementAction
	Triggered bool
	Time      time.Time
	Reason    string
}

// EnforcementAction represents enforcement actions
type EnforcementAction int

const (
	EnforcementActionNone EnforcementAction = iota
	EnforcementActionThrottle
	EnforcementActionBlock
	EnforcementActionDowngrade
)

// CostPredictor predicts costs for provider operations
type CostPredictor struct {
	historicalData map[providers.ProviderType]*CostHistory
	models         map[providers.ProviderType]*CostModel
	mu             sync.RWMutex
}

// CostHistory tracks historical cost data
type CostHistory struct {
	Entries     []CostEntry
	LastUpdated time.Time
}

// CostEntry represents a cost data point
type CostEntry struct {
	Timestamp   time.Time
	Provider    providers.ProviderType
	Model       string
	TokensIn    int
	TokensOut   int
	Cost        float64
	Quality     float64
	Latency     time.Duration
}

// CostModel represents a cost prediction model
type CostModel struct {
	Provider      providers.ProviderType
	InputCostPer1M  float64
	OutputCostPer1M float64
	FixedCost     float64
	QualityFactor float64
	LastUpdated   time.Time
}

// UsageTracker tracks provider usage and costs
type UsageTracker struct {
	usage      map[providers.ProviderType]*ProviderUsage
	sessions   map[string]*UsageSession
	mu         sync.RWMutex
}

// ProviderUsage tracks usage for a specific provider
type ProviderUsage struct {
	Provider        providers.ProviderType
	TotalRequests   int64
	TotalTokensIn   int64
	TotalTokensOut  int64
	TotalCost       float64
	AverageCost     float64
	CostPerToken    float64
	LastUsed        time.Time
	PeakUsageHour   time.Time
	PeakCostHour    time.Time
}

// UsageSession tracks usage for a specific session
type UsageSession struct {
	ID            string
	StartTime     time.Time
	EndTime       time.Time
	Requests      []UsageRequest
	TotalCost     float64
	BudgetID      string
}

// UsageRequest tracks a single request's usage
type UsageRequest struct {
	Timestamp time.Time
	Provider  providers.ProviderType
	Model     string
	TokensIn  int
	TokensOut int
	Cost      float64
	Latency   time.Duration
}

// CostOptimization represents a cost optimization recommendation
type CostOptimization struct {
	ID          string
	Type        OptimizationType
	Provider    providers.ProviderType
	Impact      OptimizationImpact
	Confidence  float64
	Description string
	Savings     float64
	Timestamp   time.Time
}

// OptimizationType represents types of cost optimizations
type OptimizationType int

const (
	OptimizationTypeProviderSwitch OptimizationType = iota
	OptimizationTypeModelDowngrade
	OptimizationTypeBatching
	OptimizationTypeCaching
	OptimizationTypeScheduling
)

// OptimizationImpact represents the impact of an optimization
type OptimizationImpact struct {
	CostSavings      float64
	QualityImpact    float64
	LatencyImpact    time.Duration
	AvailabilityRisk float64
}

// NewCostOptimizer creates a new cost optimizer
func NewCostOptimizer(manager *ProviderManager) (*CostOptimizer, error) {
	optimizer := &CostOptimizer{
		manager:       manager,
		optimizations: make([]CostOptimization, 0),
	}

	// Initialize budget manager
	budgetManager, err := NewBudgetManager()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create budget manager")
	}
	optimizer.budgetManager = budgetManager

	// Initialize cost predictor
	costPredictor, err := NewCostPredictor()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create cost predictor")
	}
	optimizer.costPredictor = costPredictor

	// Initialize usage tracker
	usageTracker, err := NewUsageTracker()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create usage tracker")
	}
	optimizer.usageTracker = usageTracker

	return optimizer, nil
}

// NewBudgetManager creates a new budget manager
func NewBudgetManager() (*BudgetManager, error) {
	return &BudgetManager{
		budgets:      make(map[string]*Budget),
		alerts:       make([]*BudgetAlert, 0),
		enforcements: make([]*BudgetEnforcement, 0),
	}, nil
}

// NewCostPredictor creates a new cost predictor
func NewCostPredictor() (*CostPredictor, error) {
	predictor := &CostPredictor{
		historicalData: make(map[providers.ProviderType]*CostHistory),
		models:         make(map[providers.ProviderType]*CostModel),
	}

	// Initialize with default cost models
	predictor.initializeDefaultModels()

	return predictor, nil
}

// initializeDefaultModels initializes default cost models for providers
func (p *CostPredictor) initializeDefaultModels() {
	// OpenAI pricing (example rates)
	p.models[providers.ProviderOpenAI] = &CostModel{
		Provider:        providers.ProviderOpenAI,
		InputCostPer1M:  5.0,  // $5 per 1M input tokens
		OutputCostPer1M: 15.0, // $15 per 1M output tokens
		QualityFactor:   0.95,
		LastUpdated:     time.Now(),
	}

	// Anthropic pricing
	p.models[providers.ProviderAnthropic] = &CostModel{
		Provider:        providers.ProviderAnthropic,
		InputCostPer1M:  3.0,  // $3 per 1M input tokens
		OutputCostPer1M: 15.0, // $15 per 1M output tokens
		QualityFactor:   0.97,
		LastUpdated:     time.Now(),
	}

	// Ollama (local) pricing
	p.models[providers.ProviderOllama] = &CostModel{
		Provider:        providers.ProviderOllama,
		InputCostPer1M:  0.0, // Free local usage
		OutputCostPer1M: 0.0,
		QualityFactor:   0.85,
		LastUpdated:     time.Now(),
	}

	// DeepSeek pricing
	p.models[providers.ProviderDeepSeek] = &CostModel{
		Provider:        providers.ProviderDeepSeek,
		InputCostPer1M:  0.14, // $0.14 per 1M input tokens
		OutputCostPer1M: 0.28, // $0.28 per 1M output tokens
		QualityFactor:   0.90,
		LastUpdated:     time.Now(),
	}

	// Ora pricing
	p.models[providers.ProviderOra] = &CostModel{
		Provider:        providers.ProviderOra,
		InputCostPer1M:  1.0,  // $1 per 1M input tokens
		OutputCostPer1M: 2.0,  // $2 per 1M output tokens
		QualityFactor:   0.88,
		LastUpdated:     time.Now(),
	}
}

// NewUsageTracker creates a new usage tracker
func NewUsageTracker() (*UsageTracker, error) {
	return &UsageTracker{
		usage:    make(map[providers.ProviderType]*ProviderUsage),
		sessions: make(map[string]*UsageSession),
	}, nil
}

// Start starts the cost optimizer
func (o *CostOptimizer) Start(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		return gerror.New(gerror.ErrCodeConflict, "cost optimizer already running", nil)
	}

	// Start optimization monitoring
	go o.monitorCostOptimization(ctx)

	// Start budget monitoring
	go o.budgetManager.monitorBudgets(ctx)

	o.running = true
	return nil
}

// Stop stops the cost optimizer
func (o *CostOptimizer) Stop() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.running = false
	return nil
}

// PredictCost predicts the cost for a given request
func (o *CostOptimizer) PredictCost(provider providers.ProviderType, tokensIn, tokensOut int) (float64, error) {
	return o.costPredictor.PredictCost(provider, tokensIn, tokensOut)
}

// PredictCost predicts cost for a provider request
func (p *CostPredictor) PredictCost(provider providers.ProviderType, tokensIn, tokensOut int) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	model, exists := p.models[provider]
	if !exists {
		return 0, gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("cost model for %s not found", provider), nil)
	}

	inputCost := (float64(tokensIn) / 1000000) * model.InputCostPer1M
	outputCost := (float64(tokensOut) / 1000000) * model.OutputCostPer1M
	totalCost := inputCost + outputCost + model.FixedCost

	return totalCost, nil
}

// OptimizeSelection optimizes provider selection for cost
func (o *CostOptimizer) OptimizeSelection(requirements TaskRequirements) (providers.ProviderType, float64, error) {
	availableProviders := o.manager.GetAvailableProviders()
	if len(availableProviders) == 0 {
		return "", 0, gerror.New(gerror.ErrCodeInternal, "no providers available", nil)
	}

	// Estimate token usage based on complexity
	estimatedTokensIn := o.estimateInputTokens(requirements.Complexity)
	estimatedTokensOut := o.estimateOutputTokens(requirements.Complexity)

	bestProvider := ""
	bestCost := math.Inf(1)
	var bestSavings float64

	for _, provider := range availableProviders {
		cost, err := o.PredictCost(provider, estimatedTokensIn, estimatedTokensOut)
		if err != nil {
			continue
		}

		// Apply quality penalty if below requirements
		performance, err := o.manager.GetProviderPerformance(provider)
		if err != nil {
			continue
		}

		if performance.QualityScore < requirements.QualityRequired {
			cost *= 1.5 // Penalty for lower quality
		}

		// Check if within budget
		if requirements.MaxCost > 0 && cost > requirements.MaxCost {
			continue
		}

		if cost < bestCost {
			bestCost = cost
			bestProvider = string(provider)
		}
	}

	if bestProvider == "" {
		return "", 0, gerror.New(gerror.ErrCodeInternal, "no cost-effective provider found", nil)
	}

	// Calculate potential savings vs most expensive option
	maxCost := 0.0
	for _, provider := range availableProviders {
		cost, err := o.PredictCost(provider, estimatedTokensIn, estimatedTokensOut)
		if err == nil && cost > maxCost {
			maxCost = cost
		}
	}

	if maxCost > 0 {
		bestSavings = maxCost - bestCost
	}

	return providers.ProviderType(bestProvider), bestSavings, nil
}

// estimateInputTokens estimates input tokens based on complexity
func (o *CostOptimizer) estimateInputTokens(complexity TaskComplexity) int {
	switch complexity {
	case TaskComplexitySimple:
		return 100
	case TaskComplexityModerate:
		return 500
	case TaskComplexityComplex:
		return 2000
	case TaskComplexityExpert:
		return 8000
	default:
		return 500
	}
}

// estimateOutputTokens estimates output tokens based on complexity
func (o *CostOptimizer) estimateOutputTokens(complexity TaskComplexity) int {
	switch complexity {
	case TaskComplexitySimple:
		return 50
	case TaskComplexityModerate:
		return 200
	case TaskComplexityComplex:
		return 800
	case TaskComplexityExpert:
		return 2000
	default:
		return 200
	}
}

// TrackUsage tracks provider usage and cost
func (o *CostOptimizer) TrackUsage(provider providers.ProviderType, tokensIn, tokensOut int, cost float64, sessionID string) {
	o.usageTracker.TrackUsage(provider, tokensIn, tokensOut, cost, sessionID)
}

// TrackUsage tracks provider usage
func (t *UsageTracker) TrackUsage(provider providers.ProviderType, tokensIn, tokensOut int, cost float64, sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Update provider usage
	usage, exists := t.usage[provider]
	if !exists {
		usage = &ProviderUsage{
			Provider: provider,
		}
		t.usage[provider] = usage
	}

	usage.TotalRequests++
	usage.TotalTokensIn += int64(tokensIn)
	usage.TotalTokensOut += int64(tokensOut)
	usage.TotalCost += cost
	usage.AverageCost = usage.TotalCost / float64(usage.TotalRequests)
	if usage.TotalTokensIn+usage.TotalTokensOut > 0 {
		usage.CostPerToken = usage.TotalCost / float64(usage.TotalTokensIn+usage.TotalTokensOut)
	}
	usage.LastUsed = time.Now()

	// Update session usage
	session, exists := t.sessions[sessionID]
	if !exists {
		session = &UsageSession{
			ID:        sessionID,
			StartTime: time.Now(),
			Requests:  make([]UsageRequest, 0),
		}
		t.sessions[sessionID] = session
	}

	request := UsageRequest{
		Timestamp: time.Now(),
		Provider:  provider,
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		Cost:      cost,
	}

	session.Requests = append(session.Requests, request)
	session.TotalCost += cost
	session.EndTime = time.Now()
}

// CreateBudget creates a new budget constraint
func (o *CostOptimizer) CreateBudget(name string, limit float64, period time.Duration, priority BudgetPriority) (*Budget, error) {
	return o.budgetManager.CreateBudget(name, limit, period, priority)
}

// CreateBudget creates a new budget
func (b *BudgetManager) CreateBudget(name string, limit float64, period time.Duration, priority BudgetPriority) (*Budget, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	budget := &Budget{
		ID:              fmt.Sprintf("budget-%d", time.Now().UnixNano()),
		Name:            name,
		Limit:           limit,
		Period:          period,
		Current:         0,
		StartTime:       time.Now(),
		AlertThresholds: []float64{0.7, 0.9, 0.95}, // 70%, 90%, 95%
		HardLimit:       priority == BudgetPriorityCritical,
		Priority:        priority,
	}

	b.budgets[budget.ID] = budget
	return budget, nil
}

// CheckBudgetCompliance checks if a cost is within budget
func (o *CostOptimizer) CheckBudgetCompliance(cost float64, budgetID string) (bool, *BudgetEnforcement, error) {
	return o.budgetManager.CheckCompliance(cost, budgetID)
}

// CheckCompliance checks budget compliance
func (b *BudgetManager) CheckCompliance(cost float64, budgetID string) (bool, *BudgetEnforcement, error) {
	b.mu.RLock()
	budget, exists := b.budgets[budgetID]
	b.mu.RUnlock()

	if !exists {
		return true, nil, nil // No budget constraint
	}

	projectedCost := budget.Current + cost
	utilizationRate := projectedCost / budget.Limit

	// Check for budget violations
	if utilizationRate > 1.0 {
		if budget.HardLimit {
			enforcement := &BudgetEnforcement{
				ID:        fmt.Sprintf("enforcement-%d", time.Now().UnixNano()),
				BudgetID:  budgetID,
				Action:    EnforcementActionBlock,
				Triggered: true,
				Time:      time.Now(),
				Reason:    fmt.Sprintf("Hard budget limit exceeded: %.2f%% utilization", utilizationRate*100),
			}

			b.mu.Lock()
			b.enforcements = append(b.enforcements, enforcement)
			b.mu.Unlock()

			return false, enforcement, nil
		} else {
			// Soft limit - allow but warn
			enforcement := &BudgetEnforcement{
				ID:        fmt.Sprintf("enforcement-%d", time.Now().UnixNano()),
				BudgetID:  budgetID,
				Action:    EnforcementActionThrottle,
				Triggered: true,
				Time:      time.Now(),
				Reason:    fmt.Sprintf("Soft budget limit exceeded: %.2f%% utilization", utilizationRate*100),
			}

			b.mu.Lock()
			b.enforcements = append(b.enforcements, enforcement)
			b.mu.Unlock()

			return true, enforcement, nil
		}
	}

	// Check alert thresholds
	for _, threshold := range budget.AlertThresholds {
		if utilizationRate >= threshold {
			alert := &BudgetAlert{
				ID:        fmt.Sprintf("alert-%d", time.Now().UnixNano()),
				BudgetID:  budgetID,
				Threshold: threshold,
				Triggered: true,
				Time:      time.Now(),
				Message:   fmt.Sprintf("Budget alert: %.1f%% of budget consumed", utilizationRate*100),
			}

			b.mu.Lock()
			b.alerts = append(b.alerts, alert)
			b.mu.Unlock()
			break
		}
	}

	return true, nil, nil
}

// GetCostOptimizations returns cost optimization recommendations
func (o *CostOptimizer) GetCostOptimizations() []CostOptimization {
	o.mu.RLock()
	defer o.mu.RUnlock()

	optimizations := make([]CostOptimization, len(o.optimizations))
	copy(optimizations, o.optimizations)
	return optimizations
}

// GetUsageStats returns usage statistics
func (o *CostOptimizer) GetUsageStats() map[providers.ProviderType]*ProviderUsage {
	return o.usageTracker.GetUsageStats()
}

// GetUsageStats returns usage statistics
func (t *UsageTracker) GetUsageStats() map[providers.ProviderType]*ProviderUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := make(map[providers.ProviderType]*ProviderUsage)
	for k, v := range t.usage {
		// Return a copy to avoid race conditions
		stats[k] = &ProviderUsage{
			Provider:        v.Provider,
			TotalRequests:   v.TotalRequests,
			TotalTokensIn:   v.TotalTokensIn,
			TotalTokensOut:  v.TotalTokensOut,
			TotalCost:       v.TotalCost,
			AverageCost:     v.AverageCost,
			CostPerToken:    v.CostPerToken,
			LastUsed:        v.LastUsed,
			PeakUsageHour:   v.PeakUsageHour,
			PeakCostHour:    v.PeakCostHour,
		}
	}
	return stats
}

// GetBudgetStatus returns budget status
func (o *CostOptimizer) GetBudgetStatus() map[string]*Budget {
	return o.budgetManager.GetBudgetStatus()
}

// GetBudgetStatus returns budget status
func (b *BudgetManager) GetBudgetStatus() map[string]*Budget {
	b.mu.RLock()
	defer b.mu.RUnlock()

	status := make(map[string]*Budget)
	for k, v := range b.budgets {
		// Return a copy to avoid race conditions
		status[k] = &Budget{
			ID:              v.ID,
			Name:            v.Name,
			Limit:           v.Limit,
			Period:          v.Period,
			Current:         v.Current,
			StartTime:       v.StartTime,
			AlertThresholds: append([]float64(nil), v.AlertThresholds...),
			HardLimit:       v.HardLimit,
			Priority:        v.Priority,
		}
	}
	return status
}

// monitorCostOptimization continuously monitors for optimization opportunities
func (o *CostOptimizer) monitorCostOptimization(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !o.running {
				return
			}
			o.analyzeOptimizationOpportunities()
		}
	}
}

// analyzeOptimizationOpportunities analyzes for cost optimization opportunities
func (o *CostOptimizer) analyzeOptimizationOpportunities() {
	// Analyze usage patterns
	usageStats := o.usageTracker.GetUsageStats()

	o.mu.Lock()
	defer o.mu.Unlock()

	// Clear old optimizations
	o.optimizations = o.optimizations[:0]

	for provider, usage := range usageStats {
		// Check if switching to a cheaper provider would save money
		if usage.TotalCost > 10.0 { // Only analyze significant costs
			o.analyzeProviderSwitchOpportunity(provider, usage)
		}

		// Check for batching opportunities
		if usage.TotalRequests > 100 {
			o.analyzeBatchingOpportunity(provider, usage)
		}

		// Check for caching opportunities
		o.analyzeCachingOpportunity(provider, usage)
	}
}

// analyzeProviderSwitchOpportunity analyzes provider switch opportunities
func (o *CostOptimizer) analyzeProviderSwitchOpportunity(currentProvider providers.ProviderType, usage *ProviderUsage) {
	availableProviders := o.manager.GetAvailableProviders()

	for _, provider := range availableProviders {
		if provider == currentProvider {
			continue
		}

		// Estimate cost with alternative provider
		averageTokensIn := int(usage.TotalTokensIn / usage.TotalRequests)
		averageTokensOut := int(usage.TotalTokensOut / usage.TotalRequests)

		altCost, err := o.PredictCost(provider, averageTokensIn, averageTokensOut)
		if err != nil {
			continue
		}

		currentCost := usage.AverageCost
		if altCost < currentCost*0.8 { // 20% savings threshold
			savings := (currentCost - altCost) * float64(usage.TotalRequests)
			
			optimization := CostOptimization{
				ID:          fmt.Sprintf("switch-%d", time.Now().UnixNano()),
				Type:        OptimizationTypeProviderSwitch,
				Provider:    provider,
				Confidence:  0.8,
				Description: fmt.Sprintf("Switch from %s to %s for 20%% cost savings", currentProvider, provider),
				Savings:     savings,
				Timestamp:   time.Now(),
				Impact: OptimizationImpact{
					CostSavings: savings,
					QualityImpact: -0.05, // Assume small quality impact
				},
			}

			o.optimizations = append(o.optimizations, optimization)
		}
	}
}

// analyzeBatchingOpportunity analyzes batching opportunities
func (o *CostOptimizer) analyzeBatchingOpportunity(provider providers.ProviderType, usage *ProviderUsage) {
	// Estimate savings from batching requests
	if usage.TotalRequests > 100 {
		estimatedSavings := float64(usage.TotalRequests) * 0.1 // 10% savings from batching
		
		optimization := CostOptimization{
			ID:          fmt.Sprintf("batch-%d", time.Now().UnixNano()),
			Type:        OptimizationTypeBatching,
			Provider:    provider,
			Confidence:  0.7,
			Description: fmt.Sprintf("Batch requests to %s for efficiency gains", provider),
			Savings:     estimatedSavings,
			Timestamp:   time.Now(),
			Impact: OptimizationImpact{
				CostSavings:   estimatedSavings,
				LatencyImpact: time.Second * 2, // Slight latency increase
			},
		}

		o.optimizations = append(o.optimizations, optimization)
	}
}

// analyzeCachingOpportunity analyzes caching opportunities
func (o *CostOptimizer) analyzeCachingOpportunity(provider providers.ProviderType, usage *ProviderUsage) {
	// Estimate savings from caching repeated requests
	if usage.TotalRequests > 50 {
		estimatedCacheHit := 0.2 // Assume 20% cache hit rate
		estimatedSavings := usage.TotalCost * estimatedCacheHit
		
		optimization := CostOptimization{
			ID:          fmt.Sprintf("cache-%d", time.Now().UnixNano()),
			Type:        OptimizationTypeCaching,
			Provider:    provider,
			Confidence:  0.6,
			Description: fmt.Sprintf("Implement caching for %s requests", provider),
			Savings:     estimatedSavings,
			Timestamp:   time.Now(),
			Impact: OptimizationImpact{
				CostSavings:   estimatedSavings,
				LatencyImpact: -time.Millisecond * 500, // Faster response from cache
			},
		}

		o.optimizations = append(o.optimizations, optimization)
	}
}

// monitorBudgets continuously monitors budget status
func (b *BudgetManager) monitorBudgets(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.checkBudgetResets()
		}
	}
}

// checkBudgetResets checks for budget period resets
func (b *BudgetManager) checkBudgetResets() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	for _, budget := range b.budgets {
		if now.Sub(budget.StartTime) >= budget.Period {
			// Reset budget for new period
			budget.Current = 0
			budget.StartTime = now
		}
	}
}