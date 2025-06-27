// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// DefaultSuggestionManager implements SuggestionManager with multiple providers
type DefaultSuggestionManager struct {
	providers   []SuggestionProvider
	history     []SuggestionHistory
	suggestions map[string]Suggestion // Cache of suggestions by ID
	mu          sync.RWMutex
}

// NewSuggestionManager creates a new suggestion manager
func NewSuggestionManager() *DefaultSuggestionManager {
	return &DefaultSuggestionManager{
		providers:   make([]SuggestionProvider, 0),
		history:     make([]SuggestionHistory, 0),
		suggestions: make(map[string]Suggestion),
	}
}

// NewSuggestionManagerWithProviders creates a manager with pre-registered providers
func NewSuggestionManagerWithProviders(providers ...SuggestionProvider) *DefaultSuggestionManager {
	manager := NewSuggestionManager()
	for _, provider := range providers {
		_ = manager.RegisterProvider(provider)
	}
	return manager
}

// RegisterProvider registers a new suggestion provider
func (m *DefaultSuggestionManager) RegisterProvider(provider SuggestionProvider) error {
	if provider == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "provider cannot be nil", nil).
			WithComponent("suggestions").
			WithOperation("RegisterProvider")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate providers
	metadata := provider.GetMetadata()
	for _, p := range m.providers {
		if p.GetMetadata().Name == metadata.Name {
			return gerror.New(gerror.ErrCodeAlreadyExists, "provider already registered", nil).
				WithComponent("suggestions").
				WithOperation("RegisterProvider").
				WithDetails("provider_name", metadata.Name)
		}
	}

	m.providers = append(m.providers, provider)
	return nil
}

// GetSuggestions retrieves suggestions from all registered providers
func (m *DefaultSuggestionManager) GetSuggestions(ctx context.Context, context SuggestionContext, filter *SuggestionFilter) ([]Suggestion, error) {
	m.mu.RLock()
	providers := make([]SuggestionProvider, len(m.providers))
	copy(providers, m.providers)
	m.mu.RUnlock()

	if len(providers) == 0 {
		return []Suggestion{}, nil
	}

	// Collect suggestions from all providers concurrently
	type result struct {
		suggestions []Suggestion
		err         error
		source      string
	}

	results := make(chan result, len(providers))

	for _, provider := range providers {
		go func(p SuggestionProvider) {
			suggestions, err := p.GetSuggestions(ctx, context)
			results <- result{
				suggestions: suggestions,
				err:         err,
				source:      p.GetMetadata().Name,
			}
		}(provider)
	}

	// Collect results
	allSuggestions := make([]Suggestion, 0)
	errors := make([]error, 0)

	for i := 0; i < len(providers); i++ {
		res := <-results
		if res.err != nil {
			errors = append(errors, res.err)
			continue
		}

		// Add source to each suggestion and generate IDs if needed
		for _, suggestion := range res.suggestions {
			if suggestion.ID == "" {
				suggestion.ID = uuid.New().String()
			}
			suggestion.Source = res.source
			suggestion.CreatedAt = time.Now()
			allSuggestions = append(allSuggestions, suggestion)
		}
	}

	// Apply filters
	filtered := m.applyFilters(allSuggestions, filter)

	// Sort by priority and confidence
	sort.Slice(filtered, func(i, j int) bool {
		// First by priority (higher is better)
		if filtered[i].Priority != filtered[j].Priority {
			return filtered[i].Priority > filtered[j].Priority
		}
		// Then by confidence (higher is better)
		return filtered[i].Confidence > filtered[j].Confidence
	})

	// Cache suggestions for later reference
	m.mu.Lock()
	for _, s := range filtered {
		m.suggestions[s.ID] = s
	}
	m.mu.Unlock()

	// Return errors if all providers failed
	if len(errors) == len(providers) && len(errors) > 0 {
		return nil, gerror.New(gerror.ErrCodeInternal, "all providers failed", errors[0]).
			WithComponent("suggestions").
			WithOperation("GetSuggestions")
	}

	return filtered, nil
}

// RecordUsage records whether a suggestion was accepted or rejected
func (m *DefaultSuggestionManager) RecordUsage(ctx context.Context, suggestionID string, accepted bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	suggestion, exists := m.suggestions[suggestionID]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "suggestion not found", nil).
			WithComponent("suggestions").
			WithOperation("RecordUsage").
			WithDetails("suggestion_id", suggestionID)
	}

	history := SuggestionHistory{
		ID:           uuid.New().String(),
		SuggestionID: suggestionID,
		Type:         suggestion.Type,
		Content:      suggestion.Content,
		Accepted:     accepted,
		UsedAt:       time.Now(),
		Metadata:     suggestion.Metadata,
	}

	m.history = append(m.history, history)

	return nil
}

// GetHistory retrieves suggestion history for a session
func (m *DefaultSuggestionManager) GetHistory(ctx context.Context, sessionID string, limit int) ([]SuggestionHistory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Filter by session if provided
	filtered := make([]SuggestionHistory, 0)
	for _, h := range m.history {
		if sessionID == "" || h.Context.SessionID == sessionID {
			filtered = append(filtered, h)
		}
	}

	// Sort by most recent first
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].UsedAt.After(filtered[j].UsedAt)
	})

	// Apply limit
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

// ProvideFeedback adds user feedback to a suggestion history item
func (m *DefaultSuggestionManager) ProvideFeedback(ctx context.Context, historyID string, feedback UserFeedback) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, h := range m.history {
		if h.ID == historyID {
			feedback.ReportedAt = time.Now()
			m.history[i].UserFeedback = &feedback
			return nil
		}
	}

	return gerror.New(gerror.ErrCodeNotFound, "history item not found", nil).
		WithComponent("suggestions").
		WithOperation("ProvideFeedback").
		WithDetails("history_id", historyID)
}

// GetAnalytics provides analytics on suggestion usage
func (m *DefaultSuggestionManager) GetAnalytics(ctx context.Context) (*SuggestionAnalytics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	analytics := &SuggestionAnalytics{
		TypeBreakdown:     make(map[SuggestionType]int64),
		ProviderBreakdown: make(map[string]int64),
		TopSuggestions:    make([]SuggestionStat, 0),
	}

	// Calculate basic metrics
	acceptedCount := int64(0)
	totalRating := 0
	ratingCount := 0

	// Track suggestion patterns
	patternStats := make(map[string]*SuggestionStat)

	for _, h := range m.history {
		analytics.TotalSuggestions++

		if h.Accepted {
			acceptedCount++
		}

		// Type breakdown
		analytics.TypeBreakdown[h.Type]++

		// Provider breakdown (from cached suggestion)
		if s, exists := m.suggestions[h.SuggestionID]; exists {
			analytics.ProviderBreakdown[s.Source]++
		}

		// User satisfaction
		if h.UserFeedback != nil && h.UserFeedback.Rating > 0 {
			totalRating += h.UserFeedback.Rating
			ratingCount++
		}

		// Pattern tracking
		pattern := fmt.Sprintf("%s:%s", h.Type, h.Content)
		if stat, exists := patternStats[pattern]; exists {
			stat.UsageCount++
			if h.Accepted {
				stat.AcceptanceRate = float64(stat.UsageCount) / float64(analytics.TotalSuggestions)
			}
			if h.UserFeedback != nil && h.UserFeedback.Rating > 0 {
				stat.AverageRating = (stat.AverageRating*float64(stat.UsageCount-1) + float64(h.UserFeedback.Rating)) / float64(stat.UsageCount)
			}
		} else {
			patternStats[pattern] = &SuggestionStat{
				Pattern:        pattern,
				Type:           h.Type,
				UsageCount:     1,
				AcceptanceRate: 0,
				AverageRating:  0,
			}
			if h.Accepted {
				patternStats[pattern].AcceptanceRate = 1.0
			}
			if h.UserFeedback != nil && h.UserFeedback.Rating > 0 {
				patternStats[pattern].AverageRating = float64(h.UserFeedback.Rating)
			}
		}
	}

	// Calculate rates
	if analytics.TotalSuggestions > 0 {
		analytics.AcceptedSuggestions = acceptedCount
		analytics.AcceptanceRate = float64(acceptedCount) / float64(analytics.TotalSuggestions)
	}

	if ratingCount > 0 {
		analytics.UserSatisfaction = float64(totalRating) / float64(ratingCount)
	}

	// Get top suggestions
	for _, stat := range patternStats {
		analytics.TopSuggestions = append(analytics.TopSuggestions, *stat)
	}

	// Sort by usage count
	sort.Slice(analytics.TopSuggestions, func(i, j int) bool {
		return analytics.TopSuggestions[i].UsageCount > analytics.TopSuggestions[j].UsageCount
	})

	// Limit to top 10
	if len(analytics.TopSuggestions) > 10 {
		analytics.TopSuggestions = analytics.TopSuggestions[:10]
	}

	return analytics, nil
}

// applyFilters applies the provided filter to suggestions
func (m *DefaultSuggestionManager) applyFilters(suggestions []Suggestion, filter *SuggestionFilter) []Suggestion {
	if filter == nil {
		return suggestions
	}

	filtered := make([]Suggestion, 0, len(suggestions))

	for _, s := range suggestions {
		// Type filter
		if len(filter.Types) > 0 {
			typeMatch := false
			for _, t := range filter.Types {
				if s.Type == t {
					typeMatch = true
					break
				}
			}
			if !typeMatch {
				continue
			}
		}

		// Confidence filter
		if filter.MinConfidence > 0 && s.Confidence < filter.MinConfidence {
			continue
		}

		// Source filter
		if len(filter.Sources) > 0 {
			sourceMatch := false
			for _, src := range filter.Sources {
				if s.Source == src {
					sourceMatch = true
					break
				}
			}
			if !sourceMatch {
				continue
			}
		}

		// Tag filter
		if len(filter.Tags) > 0 {
			tagMatch := false
			for _, filterTag := range filter.Tags {
				for _, sTag := range s.Tags {
					if sTag == filterTag {
						tagMatch = true
						break
					}
				}
				if tagMatch {
					break
				}
			}
			if !tagMatch {
				continue
			}
		}

		filtered = append(filtered, s)
	}

	// Apply max results limit
	if filter.MaxResults > 0 && len(filtered) > filter.MaxResults {
		filtered = filtered[:filter.MaxResults]
	}

	return filtered
}
