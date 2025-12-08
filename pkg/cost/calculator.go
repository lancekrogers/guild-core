// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cost

import (
	"context"
	"math"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// CostCalculator provides cost calculation utilities
type CostCalculator struct {
	// Internal state for calculations
}

// NewCostCalculator creates a new cost calculator
func NewCostCalculator(ctx context.Context) (*CostCalculator, error) {
	ctx = observability.WithComponent(ctx, "cost.calculator")
	ctx = observability.WithOperation(ctx, "NewCostCalculator")

	return &CostCalculator{}, nil
}

// CalculateTokenCost calculates cost for token usage
func (cc *CostCalculator) CalculateTokenCost(ctx context.Context, tokens int, ratePerMillion float64) (float64, error) {
	ctx = observability.WithComponent(ctx, "cost.calculator")
	ctx = observability.WithOperation(ctx, "CalculateTokenCost")

	if tokens < 0 {
		return 0, gerror.New(gerror.ErrCodeValidation, "tokens cannot be negative", nil).
			WithComponent("cost.calculator").
			WithOperation("CalculateTokenCost").
			WithDetails("tokens", tokens)
	}

	if ratePerMillion < 0 {
		return 0, gerror.New(gerror.ErrCodeValidation, "rate cannot be negative", nil).
			WithComponent("cost.calculator").
			WithOperation("CalculateTokenCost").
			WithDetails("rate_per_million", ratePerMillion)
	}

	// Convert rate per million to actual cost
	cost := float64(tokens) / 1000000.0 * ratePerMillion
	return cost, nil
}

// CalculateProjectedCost projects future cost based on current usage patterns
func (cc *CostCalculator) CalculateProjectedCost(ctx context.Context, historicalUsage []Usage, projectionPeriod time.Duration) (*CostProjection, error) {
	ctx = observability.WithComponent(ctx, "cost.calculator")
	ctx = observability.WithOperation(ctx, "CalculateProjectedCost")

	if len(historicalUsage) == 0 {
		return &CostProjection{
			HourlyRate:      0,
			DailyEstimate:   0,
			MonthlyEstimate: 0,
			BudgetRemaining: 0,
			DaysUntilLimit:  0,
			Confidence:      0,
			LastUpdated:     time.Now(),
		}, nil
	}

	// Calculate time span of historical data
	oldest := historicalUsage[0].Timestamp
	newest := historicalUsage[0].Timestamp
	totalCost := 0.0

	for _, usage := range historicalUsage {
		if usage.Timestamp.Before(oldest) {
			oldest = usage.Timestamp
		}
		if usage.Timestamp.After(newest) {
			newest = usage.Timestamp
		}

		// Extract total cost from metadata if available
		if cost, ok := usage.Metadata["total_cost"]; ok {
			if costFloat, ok := cost.(float64); ok {
				totalCost += costFloat
			}
		}
	}

	timeSpan := newest.Sub(oldest)
	if timeSpan <= 0 {
		timeSpan = time.Hour // Minimum time span
	}

	// Calculate hourly rate
	hourlyRate := totalCost / timeSpan.Hours()

	// Calculate confidence based on data quality
	confidence := cc.calculateProjectionConfidence(historicalUsage, timeSpan)

	return &CostProjection{
		HourlyRate:      hourlyRate,
		DailyEstimate:   hourlyRate * 24,
		MonthlyEstimate: hourlyRate * 24 * 30,
		BudgetRemaining: 0, // This would come from budget configuration
		DaysUntilLimit:  0, // This would be calculated based on budget
		Confidence:      confidence,
		LastUpdated:     time.Now(),
	}, nil
}

// calculateProjectionConfidence calculates confidence level for projections
func (cc *CostCalculator) calculateProjectionConfidence(usage []Usage, timeSpan time.Duration) float64 {
	// Base confidence on data volume and time span
	dataPoints := float64(len(usage))
	hours := timeSpan.Hours()

	// More data points and longer time span = higher confidence
	dataConfidence := math.Min(dataPoints/100.0, 1.0) // Max confidence at 100+ data points
	timeConfidence := math.Min(hours/168.0, 1.0)      // Max confidence at 1 week+ data

	// Calculate variance in costs to assess stability
	costs := make([]float64, 0, len(usage))
	for _, u := range usage {
		if cost, ok := u.Metadata["total_cost"]; ok {
			if costFloat, ok := cost.(float64); ok {
				costs = append(costs, costFloat)
			}
		}
	}

	varianceConfidence := cc.calculateVarianceConfidence(costs)

	// Weighted average of confidence factors
	confidence := (dataConfidence*0.3 + timeConfidence*0.4 + varianceConfidence*0.3)
	return math.Max(0.1, math.Min(1.0, confidence)) // Keep between 0.1 and 1.0
}

// calculateVarianceConfidence calculates confidence based on cost variance
func (cc *CostCalculator) calculateVarianceConfidence(costs []float64) float64 {
	if len(costs) < 2 {
		return 0.5 // Medium confidence with insufficient data
	}

	// Calculate mean
	mean := 0.0
	for _, cost := range costs {
		mean += cost
	}
	mean /= float64(len(costs))

	// Calculate variance
	variance := 0.0
	for _, cost := range costs {
		diff := cost - mean
		variance += diff * diff
	}
	variance /= float64(len(costs))

	// Calculate coefficient of variation
	if mean == 0 {
		return 0.5
	}

	cv := math.Sqrt(variance) / mean

	// Lower variance = higher confidence
	// CV > 1.0 = very high variance, low confidence
	// CV < 0.1 = very low variance, high confidence
	confidence := 1.0 - math.Min(cv, 1.0)
	return confidence
}

// EstimateCostForTask estimates cost for a specific task
func (cc *CostCalculator) EstimateCostForTask(ctx context.Context, taskComplexity int, modelType string, estimatedTokens int) (*CostEstimate, error) {
	ctx = observability.WithComponent(ctx, "cost.calculator")
	ctx = observability.WithOperation(ctx, "EstimateCostForTask")

	// Get model rates (these should come from rate cards in production)
	rates := cc.getModelRates()

	modelRates, exists := rates[modelType]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "model rates not found", nil).
			WithComponent("cost.calculator").
			WithOperation("EstimateCostForTask").
			WithDetails("model_type", modelType)
	}

	// Estimate input/output token split based on task complexity
	// Higher complexity tasks typically have longer outputs
	outputRatio := 0.2 + (float64(taskComplexity)/10.0)*0.3 // 20-50% output tokens
	outputRatio = math.Min(outputRatio, 0.5)

	inputTokens := int(float64(estimatedTokens) * (1.0 - outputRatio))
	outputTokens := estimatedTokens - inputTokens

	// Calculate costs
	inputCost, err := cc.CalculateTokenCost(ctx, inputTokens, modelRates["input"])
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to calculate input cost").
			WithComponent("cost.calculator").
			WithOperation("EstimateCostForTask")
	}

	outputCost, err := cc.CalculateTokenCost(ctx, outputTokens, modelRates["output"])
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to calculate output cost").
			WithComponent("cost.calculator").
			WithOperation("EstimateCostForTask")
	}

	totalCost := inputCost + outputCost

	// Apply complexity multiplier for execution overhead
	complexityMultiplier := 1.0 + (float64(taskComplexity)/10.0)*0.2 // 0-20% overhead
	totalCost *= complexityMultiplier

	return &CostEstimate{
		ModelType:       modelType,
		EstimatedTokens: estimatedTokens,
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		InputCost:       inputCost,
		OutputCost:      outputCost,
		TotalCost:       totalCost,
		Confidence:      cc.calculateEstimateConfidence(taskComplexity, estimatedTokens),
		Currency:        "USD",
		EstimatedAt:     time.Now(),
	}, nil
}

// calculateEstimateConfidence calculates confidence for cost estimates
func (cc *CostCalculator) calculateEstimateConfidence(taskComplexity, estimatedTokens int) float64 {
	// Higher confidence for:
	// - Lower complexity tasks (more predictable)
	// - Moderate token counts (not too small or too large)

	complexityConfidence := 1.0 - (float64(taskComplexity)/10.0)*0.3 // Lower confidence for complex tasks

	// Optimal token range is 500-5000 tokens
	tokenConfidence := 1.0
	if estimatedTokens < 100 {
		tokenConfidence = 0.6 // Low confidence for very small estimates
	} else if estimatedTokens > 10000 {
		tokenConfidence = 0.7 // Lower confidence for very large estimates
	}

	confidence := (complexityConfidence + tokenConfidence) / 2.0
	return math.Max(0.3, math.Min(1.0, confidence))
}

// getModelRates returns hardcoded model rates (in production, these would come from providers)
func (cc *CostCalculator) getModelRates() map[string]map[string]float64 {
	return map[string]map[string]float64{
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
	}
}

// CostEstimate represents a cost estimate for a task
type CostEstimate struct {
	ModelType       string    `json:"model_type"`
	EstimatedTokens int       `json:"estimated_tokens"`
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	InputCost       float64   `json:"input_cost"`
	OutputCost      float64   `json:"output_cost"`
	TotalCost       float64   `json:"total_cost"`
	Confidence      float64   `json:"confidence"`
	Currency        string    `json:"currency"`
	EstimatedAt     time.Time `json:"estimated_at"`
}

// CalculateSavings calculates potential savings between two cost estimates
func (cc *CostCalculator) CalculateSavings(ctx context.Context, currentCost, optimizedCost *CostEstimate) (*SavingsAnalysis, error) {
	ctx = observability.WithComponent(ctx, "cost.calculator")
	ctx = observability.WithOperation(ctx, "CalculateSavings")

	if currentCost == nil || optimizedCost == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "cost estimates cannot be nil", nil).
			WithComponent("cost.calculator").
			WithOperation("CalculateSavings")
	}

	absoluteSavings := currentCost.TotalCost - optimizedCost.TotalCost
	percentageSavings := 0.0
	if currentCost.TotalCost > 0 {
		percentageSavings = (absoluteSavings / currentCost.TotalCost) * 100
	}

	// Calculate confidence as minimum of both estimates
	confidence := math.Min(currentCost.Confidence, optimizedCost.Confidence)

	return &SavingsAnalysis{
		CurrentCost:       currentCost.TotalCost,
		OptimizedCost:     optimizedCost.TotalCost,
		AbsoluteSavings:   absoluteSavings,
		PercentageSavings: percentageSavings,
		Confidence:        confidence,
		Currency:          currentCost.Currency,
		AnalyzedAt:        time.Now(),
	}, nil
}

// SavingsAnalysis represents analysis of potential cost savings
type SavingsAnalysis struct {
	CurrentCost       float64   `json:"current_cost"`
	OptimizedCost     float64   `json:"optimized_cost"`
	AbsoluteSavings   float64   `json:"absolute_savings"`
	PercentageSavings float64   `json:"percentage_savings"`
	Confidence        float64   `json:"confidence"`
	Currency          string    `json:"currency"`
	AnalyzedAt        time.Time `json:"analyzed_at"`
}
