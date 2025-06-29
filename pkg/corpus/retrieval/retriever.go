// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package retrieval

import (
	"context"
	"sync"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// RetrieverImpl implements multi-strategy document retrieval with sophisticated ranking
type RetrieverImpl struct {
	vectorStore   VectorStore
	metadataStore MetadataStore
	strategies    []RetrievalStrategy
	ranker        *ResultRanker
	eventBus      EventBus
}

// NewRetriever creates a new retriever with the specified components
func NewRetriever(ctx context.Context, vectorStore VectorStore, metadataStore MetadataStore, ranker *ResultRanker) (Retriever, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("Retriever").
		WithOperation("NewRetriever")

	if vectorStore == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "vectorStore cannot be nil", nil).
			WithComponent("Retriever").
			WithOperation("NewRetriever")
	}

	if metadataStore == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "metadataStore cannot be nil", nil).
			WithComponent("Retriever").
			WithOperation("NewRetriever")
	}

	if ranker == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "ranker cannot be nil", nil).
			WithComponent("Retriever").
			WithOperation("NewRetriever")
	}

	retriever := &RetrieverImpl{
		vectorStore:   vectorStore,
		metadataStore: metadataStore,
		strategies:    make([]RetrievalStrategy, 0),
		ranker:        ranker,
	}

	logger.Info("Retriever created successfully")
	return retriever, nil
}

// AddStrategy adds a retrieval strategy to the retriever
func (r *RetrieverImpl) AddStrategy(strategy RetrievalStrategy) error {
	if strategy == nil {
		return gerror.New(gerror.ErrCodeValidation, "strategy cannot be nil", nil).
			WithComponent("Retriever").
			WithOperation("AddStrategy")
	}

	r.strategies = append(r.strategies, strategy)
	return nil
}

// SetEventBus sets the event bus for publishing retrieval events
func (r *RetrieverImpl) SetEventBus(eventBus EventBus) {
	r.eventBus = eventBus
}

// Retrieve performs multi-strategy retrieval and ranking
func (r *RetrieverImpl) Retrieve(ctx context.Context, query Query) ([]RankedDocument, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("Retriever").
		WithOperation("Retrieve")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("Retriever").
			WithOperation("Retrieve")
	}

	// Validate query
	if query.Text == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "query text cannot be empty", nil).
			WithComponent("Retriever").
			WithOperation("Retrieve")
	}

	if len(r.strategies) == 0 {
		return nil, gerror.New(gerror.ErrCodeInternal, "no retrieval strategies configured", nil).
			WithComponent("Retriever").
			WithOperation("Retrieve")
	}

	// Collect results from all strategies
	allResults := make([][]Document, len(r.strategies))
	weights := make([]float64, len(r.strategies))
	
	// Parallel retrieval
	var wg sync.WaitGroup
	resultsChan := make(chan strategyResult, len(r.strategies))
	
	for i, strategy := range r.strategies {
		wg.Add(1)
		go func(idx int, strat RetrievalStrategy) {
			defer wg.Done()
			
			strategyCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			
			strategyLogger := logger.WithComponent(strat.Name())
			strategyLogger.Debug("Starting strategy retrieval")
			
			results, err := strat.Retrieve(strategyCtx, query)
			if err != nil {
				strategyLogger.WithError(err).Warn("Strategy retrieval failed")
				resultsChan <- strategyResult{index: idx, error: err}
				return
			}
			
			strategyLogger.Debug("Strategy retrieval completed", "results_count", len(results))
			resultsChan <- strategyResult{
				index:   idx,
				results: results,
				weight:  strat.Weight(),
			}
		}(i, strategy)
	}
	
	// Wait for all strategies to complete
	wg.Wait()
	close(resultsChan)
	
	// Collect results
	for result := range resultsChan {
		if result.error != nil {
			logger.WithError(result.error).Warn("Strategy failed", "index", result.index)
			continue
		}
		allResults[result.index] = result.results
		weights[result.index] = result.weight
	}
	
	// Merge and rank results
	merged := r.mergeResults(allResults, weights)
	logger.Debug("Results merged", "merged_count", len(merged))
	
	ranked := r.ranker.Rank(merged, query)
	logger.Debug("Results ranked", "ranked_count", len(ranked))
	
	// Apply score threshold
	filtered := r.filterByScore(ranked, query.MinScore)
	logger.Debug("Results filtered by score", "filtered_count", len(filtered))
	
	// Limit results
	if query.MaxResults > 0 && len(filtered) > query.MaxResults {
		filtered = filtered[:query.MaxResults]
	}
	
	// Publish retrieval event
	if r.eventBus != nil {
		event := Event{
			Type:   "retrieval.completed",
			Source: "corpus.retriever",
			Data: map[string]interface{}{
				"query":         query.Text,
				"results_count": len(filtered),
				"strategies":    len(r.strategies),
			},
		}
		if err := r.eventBus.Publish(ctx, event); err != nil {
			logger.WithError(err).Warn("Failed to publish retrieval event")
		}
	}
	
	logger.Info("Retrieval completed successfully", "final_count", len(filtered))
	return filtered, nil
}

// strategyResult holds the result of a strategy execution
type strategyResult struct {
	index   int
	results []Document
	weight  float64
	error   error
}

// mergeResults combines results from multiple strategies with weighted scoring
func (r *RetrieverImpl) mergeResults(allResults [][]Document, weights []float64) []Document {
	documentMap := make(map[string]*Document)
	
	for i, results := range allResults {
		if results == nil {
			continue
		}
		
		weight := weights[i]
		for _, doc := range results {
			if existing, exists := documentMap[doc.ID]; exists {
				// Combine scores with weights
				existing.Score = (existing.Score + doc.Score*weight) / 2.0
				// Merge metadata
				if existing.Metadata == nil {
					existing.Metadata = make(map[string]interface{})
				}
				for k, v := range doc.Metadata {
					existing.Metadata[k] = v
				}
			} else {
				// Create new document with weighted score
				newDoc := doc
				newDoc.Score *= weight
				documentMap[doc.ID] = &newDoc
			}
		}
	}
	
	// Convert back to slice
	merged := make([]Document, 0, len(documentMap))
	for _, doc := range documentMap {
		merged = append(merged, *doc)
	}
	
	return merged
}

// filterByScore removes documents below the minimum score threshold
func (r *RetrieverImpl) filterByScore(ranked []RankedDocument, minScore float64) []RankedDocument {
	if minScore <= 0 {
		return ranked
	}
	
	filtered := make([]RankedDocument, 0, len(ranked))
	for _, doc := range ranked {
		if doc.FinalScore >= minScore {
			filtered = append(filtered, doc)
		}
	}
	
	return filtered
}

// GetStrategies returns the current retrieval strategies
func (r *RetrieverImpl) GetStrategies() []RetrievalStrategy {
	return r.strategies
}

// Close releases resources used by the retriever
func (r *RetrieverImpl) Close() error {
	// Close any resources if needed
	return nil
}