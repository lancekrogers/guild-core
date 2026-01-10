// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package cost provides cost tracking and optimization for MCP
package cost

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/mcp/protocol"
)

// Observer monitors and records costs
type Observer interface {
	// RecordCost records cost metrics for an operation
	RecordCost(ctx context.Context, operationID string, cost protocol.CostReport)

	// GetCosts retrieves costs based on filter
	GetCosts(ctx context.Context, filter CostFilter) ([]protocol.CostReport, error)

	// GetAverageCost calculates average cost based on filter
	GetAverageCost(ctx context.Context, filter CostFilter) (protocol.CostReport, error)

	// GetTotalCost calculates total cost based on filter
	GetTotalCost(ctx context.Context, filter CostFilter) (protocol.CostReport, error)

	// Analyze performs cost analysis
	Analyze(ctx context.Context, query protocol.CostQuery) (*protocol.CostAnalysis, error)
}

// CostFilter defines filtering criteria for cost queries
type CostFilter struct {
	OperationIDs []string
	StartTime    time.Time
	EndTime      time.Time
	MinCost      float64
	MaxCost      float64
	ToolIDs      []string
	UserIDs      []string
}

// MemoryObserver implements an in-memory cost observer
type MemoryObserver struct {
	costs      []protocol.CostReport
	byOp       map[string][]protocol.CostReport
	byTool     map[string][]protocol.CostReport
	byUser     map[string][]protocol.CostReport
	mu         sync.RWMutex
	maxRecords int
}

// NewMemoryObserver creates a new in-memory cost observer
func NewMemoryObserver(maxRecords int) *MemoryObserver {
	if maxRecords <= 0 {
		maxRecords = 10000
	}

	return &MemoryObserver{
		costs:      make([]protocol.CostReport, 0, maxRecords),
		byOp:       make(map[string][]protocol.CostReport),
		byTool:     make(map[string][]protocol.CostReport),
		byUser:     make(map[string][]protocol.CostReport),
		maxRecords: maxRecords,
	}
}

// RecordCost records a cost report
func (o *MemoryObserver) RecordCost(ctx context.Context, operationID string, cost protocol.CostReport) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Set operation ID if not set
	if cost.OperationID == "" {
		cost.OperationID = operationID
	}

	// Add to main list
	o.costs = append(o.costs, cost)

	// Maintain max records limit
	if len(o.costs) > o.maxRecords {
		// Remove oldest records
		o.costs = o.costs[len(o.costs)-o.maxRecords:]
		// Rebuild indices
		o.rebuildIndices()
	} else {
		// Update indices
		o.byOp[cost.OperationID] = append(o.byOp[cost.OperationID], cost)

		// Extract metadata for indexing
		if toolID := ctx.Value("tool_id"); toolID != nil {
			o.byTool[fmt.Sprintf("%v", toolID)] = append(o.byTool[fmt.Sprintf("%v", toolID)], cost)
		}
		if userID := ctx.Value("user_id"); userID != nil {
			o.byUser[fmt.Sprintf("%v", userID)] = append(o.byUser[fmt.Sprintf("%v", userID)], cost)
		}
	}
}

// GetCosts retrieves costs matching the filter
func (o *MemoryObserver) GetCosts(ctx context.Context, filter CostFilter) ([]protocol.CostReport, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var results []protocol.CostReport

	// If specific operation IDs requested, use index
	if len(filter.OperationIDs) > 0 {
		for _, opID := range filter.OperationIDs {
			if costs, exists := o.byOp[opID]; exists {
				results = append(results, costs...)
			}
		}
	} else {
		// Start with all costs
		results = make([]protocol.CostReport, len(o.costs))
		copy(results, o.costs)
	}

	// Apply filters
	filtered := results[:0]
	for _, cost := range results {
		// Time filter
		if !filter.StartTime.IsZero() && cost.StartTime.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && cost.EndTime.After(filter.EndTime) {
			continue
		}

		// Cost filter
		if filter.MinCost > 0 && cost.FinancialCost < filter.MinCost {
			continue
		}
		if filter.MaxCost > 0 && cost.FinancialCost > filter.MaxCost {
			continue
		}

		filtered = append(filtered, cost)
	}

	return filtered, nil
}

// GetAverageCost calculates average cost
func (o *MemoryObserver) GetAverageCost(ctx context.Context, filter CostFilter) (protocol.CostReport, error) {
	costs, err := o.GetCosts(ctx, filter)
	if err != nil {
		return protocol.CostReport{}, err
	}

	if len(costs) == 0 {
		return protocol.CostReport{}, nil
	}

	// Calculate averages
	var total protocol.CostReport
	for _, cost := range costs {
		total.ComputeCost += cost.ComputeCost
		total.MemoryCost += cost.MemoryCost
		total.LatencyCost += cost.LatencyCost
		total.TokensCost += cost.TokensCost
		total.APICallsCost += cost.APICallsCost
		total.FinancialCost += cost.FinancialCost
	}

	count := float64(len(costs))
	avg := protocol.CostReport{
		ComputeCost:   total.ComputeCost / count,
		MemoryCost:    int64(float64(total.MemoryCost) / count),
		LatencyCost:   time.Duration(float64(total.LatencyCost) / count),
		TokensCost:    int(float64(total.TokensCost) / count),
		APICallsCost:  int(float64(total.APICallsCost) / count),
		FinancialCost: total.FinancialCost / count,
	}

	return avg, nil
}

// GetTotalCost calculates total cost
func (o *MemoryObserver) GetTotalCost(ctx context.Context, filter CostFilter) (protocol.CostReport, error) {
	costs, err := o.GetCosts(ctx, filter)
	if err != nil {
		return protocol.CostReport{}, err
	}

	var total protocol.CostReport
	for _, cost := range costs {
		total.ComputeCost += cost.ComputeCost
		total.MemoryCost += cost.MemoryCost
		total.LatencyCost += cost.LatencyCost
		total.TokensCost += cost.TokensCost
		total.APICallsCost += cost.APICallsCost
		total.FinancialCost += cost.FinancialCost
	}

	return total, nil
}

// Analyze performs cost analysis based on query
func (o *MemoryObserver) Analyze(ctx context.Context, query protocol.CostQuery) (*protocol.CostAnalysis, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Build filter from query
	filter := CostFilter{
		OperationIDs: query.OperationIDs,
		StartTime:    query.StartTime,
		EndTime:      query.EndTime,
	}

	// Get total cost
	total, err := o.GetTotalCost(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Build breakdown based on GroupBy
	var breakdown []protocol.CostReport
	var breakdownBy string

	switch query.GroupBy {
	case "operation":
		breakdownBy = "operation"
		// Group by operation
		for opID, costs := range o.byOp {
			var opTotal protocol.CostReport
			opTotal.OperationID = opID
			for _, cost := range costs {
				// Apply time filter
				if !query.StartTime.IsZero() && cost.StartTime.Before(query.StartTime) {
					continue
				}
				if !query.EndTime.IsZero() && cost.EndTime.After(query.EndTime) {
					continue
				}

				opTotal.ComputeCost += cost.ComputeCost
				opTotal.MemoryCost += cost.MemoryCost
				opTotal.LatencyCost += cost.LatencyCost
				opTotal.TokensCost += cost.TokensCost
				opTotal.APICallsCost += cost.APICallsCost
				opTotal.FinancialCost += cost.FinancialCost
			}
			if opTotal.FinancialCost > 0 {
				breakdown = append(breakdown, opTotal)
			}
		}

	case "tool":
		breakdownBy = "tool"
		// Group by tool
		for toolID, costs := range o.byTool {
			var toolTotal protocol.CostReport
			toolTotal.OperationID = toolID // Store tool ID in operation ID field
			for _, cost := range costs {
				// Apply time filter
				if !query.StartTime.IsZero() && cost.StartTime.Before(query.StartTime) {
					continue
				}
				if !query.EndTime.IsZero() && cost.EndTime.After(query.EndTime) {
					continue
				}

				toolTotal.ComputeCost += cost.ComputeCost
				toolTotal.MemoryCost += cost.MemoryCost
				toolTotal.LatencyCost += cost.LatencyCost
				toolTotal.TokensCost += cost.TokensCost
				toolTotal.APICallsCost += cost.APICallsCost
				toolTotal.FinancialCost += cost.FinancialCost
			}
			if toolTotal.FinancialCost > 0 {
				breakdown = append(breakdown, toolTotal)
			}
		}

	case "user":
		breakdownBy = "user"
		// Group by user
		for userID, costs := range o.byUser {
			var userTotal protocol.CostReport
			userTotal.OperationID = userID // Store user ID in operation ID field
			for _, cost := range costs {
				// Apply time filter
				if !query.StartTime.IsZero() && cost.StartTime.Before(query.StartTime) {
					continue
				}
				if !query.EndTime.IsZero() && cost.EndTime.After(query.EndTime) {
					continue
				}

				userTotal.ComputeCost += cost.ComputeCost
				userTotal.MemoryCost += cost.MemoryCost
				userTotal.LatencyCost += cost.LatencyCost
				userTotal.TokensCost += cost.TokensCost
				userTotal.APICallsCost += cost.APICallsCost
				userTotal.FinancialCost += cost.FinancialCost
			}
			if userTotal.FinancialCost > 0 {
				breakdown = append(breakdown, userTotal)
			}
		}

	default:
		// No grouping, just return filtered costs
		costs, _ := o.GetCosts(ctx, filter)
		breakdown = costs
	}

	// Apply limit if specified
	if query.Limit > 0 && len(breakdown) > query.Limit {
		breakdown = breakdown[:query.Limit]
	}

	// Generate recommendations
	recommendations := o.generateRecommendations(total, breakdown)

	return &protocol.CostAnalysis{
		TotalCost:       total,
		BreakdownBy:     breakdownBy,
		Breakdown:       breakdown,
		Recommendations: recommendations,
	}, nil
}

// rebuildIndices rebuilds the indices after pruning old records
func (o *MemoryObserver) rebuildIndices() {
	o.byOp = make(map[string][]protocol.CostReport)
	o.byTool = make(map[string][]protocol.CostReport)
	o.byUser = make(map[string][]protocol.CostReport)

	for _, cost := range o.costs {
		o.byOp[cost.OperationID] = append(o.byOp[cost.OperationID], cost)
		// Tool and user would need to be stored in the cost report metadata
	}
}

// generateRecommendations generates cost optimization recommendations
func (o *MemoryObserver) generateRecommendations(total protocol.CostReport, breakdown []protocol.CostReport) []string {
	var recommendations []string

	// High latency recommendation
	avgLatency := total.LatencyCost / time.Duration(len(breakdown))
	if avgLatency > 5*time.Second {
		recommendations = append(recommendations,
			"Consider using faster tools or caching results to reduce latency")
	}

	// High token usage recommendation
	if total.TokensCost > 100000 {
		recommendations = append(recommendations,
			"High token usage detected. Consider using more concise prompts or chunking large operations")
	}

	// Cost concentration recommendation
	if len(breakdown) > 0 {
		topCost := breakdown[0].FinancialCost
		if topCost > total.FinancialCost*0.5 {
			recommendations = append(recommendations,
				fmt.Sprintf("Over 50%% of costs come from a single source. Consider optimizing or finding alternatives"))
		}
	}

	// API calls recommendation
	if total.APICallsCost > 1000 {
		recommendations = append(recommendations,
			"High number of API calls. Consider batching requests or implementing caching")
	}

	return recommendations
}
