package prompt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
)

// Analyzer analyzes prompt chains
type Analyzer interface {
	// StartChain marks the start of a chain execution
	StartChain(ctx context.Context, input *Input) string

	// EndChain marks the end of a chain execution
	EndChain(ctx context.Context, chainID string)

	// RecordExchange records a prompt exchange
	RecordExchange(ctx context.Context, exchange *Exchange)

	// RecordError records an error in the chain
	RecordError(ctx context.Context, chainID string, err error)

	// RecordFailure records a complete failure
	RecordFailure(ctx context.Context, input *Input, err error, duration time.Duration)

	// GetChainAnalysis returns analysis for a chain
	GetChainAnalysis(chainID string) (*ChainAnalysis, error)

	// GetAggregateAnalysis returns aggregate analysis
	GetAggregateAnalysis() *AggregateAnalysis
}

// ChainAnalysis contains analysis for a single chain execution
type ChainAnalysis struct {
	ChainID       string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	Exchanges     []*Exchange
	TotalCost     protocol.CostReport
	Success       bool
	Error         error
	ProcessorPath []string
}

// AggregateAnalysis contains aggregate analysis across chains
type AggregateAnalysis struct {
	TotalChains      int
	SuccessfulChains int
	FailedChains     int
	AverageLatency   time.Duration
	TotalCost        protocol.CostReport
	ProcessorStats   map[string]*ProcessorStats
}

// ProcessorStats contains statistics for a processor
type ProcessorStats struct {
	Name           string
	Invocations    int
	Successes      int
	Failures       int
	AverageLatency time.Duration
	TotalCost      protocol.CostReport
}

// MemoryAnalyzer implements an in-memory analyzer
type MemoryAnalyzer struct {
	chains     map[string]*ChainAnalysis
	aggregate  *AggregateAnalysis
	mu         sync.RWMutex
	maxChains  int
	chainOrder []string // Track insertion order for cleanup
}

// NewAnalyzer creates a new memory analyzer
func NewAnalyzer() *MemoryAnalyzer {
	return &MemoryAnalyzer{
		chains: make(map[string]*ChainAnalysis),
		aggregate: &AggregateAnalysis{
			ProcessorStats: make(map[string]*ProcessorStats),
		},
		maxChains:  1000,
		chainOrder: make([]string, 0, 1000),
	}
}

// StartChain marks the start of a chain execution
func (a *MemoryAnalyzer) StartChain(ctx context.Context, input *Input) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	chainID := fmt.Sprintf("chain-%d-%s", time.Now().UnixNano(), randomString(8))

	analysis := &ChainAnalysis{
		ChainID:       chainID,
		StartTime:     time.Now(),
		Exchanges:     make([]*Exchange, 0),
		ProcessorPath: make([]string, 0),
		Success:       true, // Assume success until proven otherwise
	}

	a.chains[chainID] = analysis
	a.chainOrder = append(a.chainOrder, chainID)

	// Cleanup old chains if needed
	if len(a.chains) > a.maxChains {
		oldestID := a.chainOrder[0]
		delete(a.chains, oldestID)
		a.chainOrder = a.chainOrder[1:]
	}

	a.aggregate.TotalChains++

	return chainID
}

// EndChain marks the end of a chain execution
func (a *MemoryAnalyzer) EndChain(ctx context.Context, chainID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	analysis, exists := a.chains[chainID]
	if !exists {
		return
	}

	analysis.EndTime = time.Now()
	analysis.Duration = analysis.EndTime.Sub(analysis.StartTime)

	// Calculate total cost
	for _, exchange := range analysis.Exchanges {
		analysis.TotalCost.ComputeCost += exchange.Metrics.ComputeCost
		analysis.TotalCost.MemoryCost += exchange.Metrics.MemoryCost
		analysis.TotalCost.LatencyCost += exchange.Metrics.LatencyCost
		analysis.TotalCost.TokensCost += exchange.Metrics.TokensCost
		analysis.TotalCost.APICallsCost += exchange.Metrics.APICallsCost
		analysis.TotalCost.FinancialCost += exchange.Metrics.FinancialCost
	}

	// Update aggregate stats
	if analysis.Success {
		a.aggregate.SuccessfulChains++
	} else {
		a.aggregate.FailedChains++
	}

	// Update average latency
	totalLatency := a.aggregate.AverageLatency * time.Duration(a.aggregate.TotalChains-1)
	a.aggregate.AverageLatency = (totalLatency + analysis.Duration) / time.Duration(a.aggregate.TotalChains)

	// Update total cost
	a.aggregate.TotalCost.ComputeCost += analysis.TotalCost.ComputeCost
	a.aggregate.TotalCost.MemoryCost += analysis.TotalCost.MemoryCost
	a.aggregate.TotalCost.LatencyCost += analysis.TotalCost.LatencyCost
	a.aggregate.TotalCost.TokensCost += analysis.TotalCost.TokensCost
	a.aggregate.TotalCost.APICallsCost += analysis.TotalCost.APICallsCost
	a.aggregate.TotalCost.FinancialCost += analysis.TotalCost.FinancialCost
}

// RecordExchange records a prompt exchange
func (a *MemoryAnalyzer) RecordExchange(ctx context.Context, exchange *Exchange) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Get chain ID from context
	chainID, ok := ctx.Value("chain_id").(string)
	if !ok {
		return
	}

	analysis, exists := a.chains[chainID]
	if !exists {
		return
	}

	// Add exchange
	analysis.Exchanges = append(analysis.Exchanges, exchange)
	analysis.ProcessorPath = append(analysis.ProcessorPath, exchange.Processor)

	// Update processor stats
	stats, exists := a.aggregate.ProcessorStats[exchange.Processor]
	if !exists {
		stats = &ProcessorStats{
			Name: exchange.Processor,
		}
		a.aggregate.ProcessorStats[exchange.Processor] = stats
	}

	stats.Invocations++
	stats.Successes++ // Will be decremented if error recorded

	// Update processor latency
	processorLatency := exchange.EndTime.Sub(exchange.StartTime)
	totalLatency := stats.AverageLatency * time.Duration(stats.Invocations-1)
	stats.AverageLatency = (totalLatency + processorLatency) / time.Duration(stats.Invocations)

	// Update processor cost
	stats.TotalCost.ComputeCost += exchange.Metrics.ComputeCost
	stats.TotalCost.MemoryCost += exchange.Metrics.MemoryCost
	stats.TotalCost.LatencyCost += exchange.Metrics.LatencyCost
	stats.TotalCost.TokensCost += exchange.Metrics.TokensCost
	stats.TotalCost.APICallsCost += exchange.Metrics.APICallsCost
	stats.TotalCost.FinancialCost += exchange.Metrics.FinancialCost
}

// RecordError records an error in the chain
func (a *MemoryAnalyzer) RecordError(ctx context.Context, chainID string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	analysis, exists := a.chains[chainID]
	if !exists {
		return
	}

	analysis.Success = false
	analysis.Error = err

	// Update processor failure stats if we know which processor failed
	if len(analysis.ProcessorPath) > 0 {
		lastProcessor := analysis.ProcessorPath[len(analysis.ProcessorPath)-1]
		if stats, exists := a.aggregate.ProcessorStats[lastProcessor]; exists {
			stats.Successes--
			stats.Failures++
		}
	}
}

// RecordFailure records a complete failure
func (a *MemoryAnalyzer) RecordFailure(ctx context.Context, input *Input, err error, duration time.Duration) {
	chainID := a.StartChain(ctx, input)
	a.RecordError(ctx, chainID, err)

	a.mu.Lock()
	if analysis, exists := a.chains[chainID]; exists {
		analysis.Duration = duration
		analysis.EndTime = analysis.StartTime.Add(duration)
	}
	a.mu.Unlock()

	a.EndChain(ctx, chainID)
}

// GetChainAnalysis returns analysis for a specific chain
func (a *MemoryAnalyzer) GetChainAnalysis(chainID string) (*ChainAnalysis, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	analysis, exists := a.chains[chainID]
	if !exists {
		return nil, fmt.Errorf("chain %s not found", chainID)
	}

	// Return a copy to prevent modification
	result := *analysis
	result.Exchanges = make([]*Exchange, len(analysis.Exchanges))
	copy(result.Exchanges, analysis.Exchanges)
	result.ProcessorPath = make([]string, len(analysis.ProcessorPath))
	copy(result.ProcessorPath, analysis.ProcessorPath)

	return &result, nil
}

// GetAggregateAnalysis returns aggregate analysis
func (a *MemoryAnalyzer) GetAggregateAnalysis() *AggregateAnalysis {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Return a copy
	result := &AggregateAnalysis{
		TotalChains:      a.aggregate.TotalChains,
		SuccessfulChains: a.aggregate.SuccessfulChains,
		FailedChains:     a.aggregate.FailedChains,
		AverageLatency:   a.aggregate.AverageLatency,
		TotalCost:        a.aggregate.TotalCost,
		ProcessorStats:   make(map[string]*ProcessorStats),
	}

	// Copy processor stats
	for name, stats := range a.aggregate.ProcessorStats {
		statsCopy := *stats
		result.ProcessorStats[name] = &statsCopy
	}

	return result
}

// Helper function
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
