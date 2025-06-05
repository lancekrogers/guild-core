package prompts

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// GuildLayeredManager implements LayeredManager interface for Guild prompt management
type GuildLayeredManager struct {
	baseManager Manager                // Existing prompt manager
	assembler   *LayeredPromptAssembler // Prompt assembler
	registry    LayeredRegistry         // Layered registry
	store       LayeredStore            // Guild Archives storage with layered support
}

// NewGuildLayeredManager creates a new layered prompt manager for Guild
func NewGuildLayeredManager(
	baseManager Manager,
	store LayeredStore,
	baseRegistry Registry,
	ragRetriever RAGRetriever,
	tokenBudget int,
) *GuildLayeredManager {
	// Create layered registry
	layeredReg := NewGuildLayeredRegistry(baseRegistry, store)
	
	// Create layered assembler
	assembler := NewLayeredPromptAssembler(
		baseManager,
		baseManager.(Formatter), // Assuming base manager implements Formatter
		store,
		ragRetriever,
		tokenBudget,
	)
	
	return &GuildLayeredManager{
		baseManager: baseManager,
		assembler:   assembler,
		registry:    layeredReg,
		store:       store,
	}
}

// Legacy Manager interface methods (delegate to base manager)

func (glm *GuildLayeredManager) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	return glm.baseManager.GetSystemPrompt(ctx, role, domain)
}

func (glm *GuildLayeredManager) GetTemplate(ctx context.Context, templateName string) (string, error) {
	return glm.baseManager.GetTemplate(ctx, templateName)
}

func (glm *GuildLayeredManager) FormatContext(ctx context.Context, context Context) (string, error) {
	return glm.baseManager.FormatContext(ctx, context)
}

func (glm *GuildLayeredManager) ListRoles(ctx context.Context) ([]string, error) {
	return glm.baseManager.ListRoles(ctx)
}

func (glm *GuildLayeredManager) ListDomains(ctx context.Context, role string) ([]string, error) {
	return glm.baseManager.ListDomains(ctx, role)
}

// LayeredManager interface methods

// BuildLayeredPrompt assembles a complete layered prompt for a Guild artisan
func (glm *GuildLayeredManager) BuildLayeredPrompt(
	ctx context.Context,
	artisanID, sessionID string,
	turnCtx TurnContext,
) (*LayeredPrompt, error) {
	return glm.assembler.BuildPrompt(ctx, artisanID, sessionID, turnCtx)
}

// GetPromptLayer retrieves a specific prompt layer
func (glm *GuildLayeredManager) GetPromptLayer(
	ctx context.Context,
	layer PromptLayer,
	artisanID, sessionID string,
) (*SystemPrompt, error) {
	// Determine the appropriate identifier based on layer type
	identifier := glm.getLayerIdentifier(layer, artisanID, sessionID)
	
	return glm.registry.GetLayeredPrompt(layer, identifier)
}

// SetPromptLayer sets or updates a specific prompt layer
func (glm *GuildLayeredManager) SetPromptLayer(ctx context.Context, prompt SystemPrompt) error {
	// Validate the prompt
	if err := glm.validatePrompt(prompt); err != nil {
		return fmt.Errorf("invalid prompt: %w", err)
	}
	
	// Determine the appropriate identifier
	identifier := glm.getLayerIdentifier(prompt.Layer, prompt.ArtisanID, prompt.SessionID)
	
	// Store the prompt
	if err := glm.registry.RegisterLayeredPrompt(prompt.Layer, identifier, prompt); err != nil {
		return fmt.Errorf("failed to set prompt layer: %w", err)
	}
	
	// Invalidate relevant caches
	return glm.invalidateRelevantCaches(ctx, prompt.Layer, prompt.ArtisanID, prompt.SessionID)
}

// DeletePromptLayer removes a specific prompt layer
func (glm *GuildLayeredManager) DeletePromptLayer(
	ctx context.Context,
	layer PromptLayer,
	artisanID, sessionID string,
) error {
	identifier := glm.getLayerIdentifier(layer, artisanID, sessionID)
	
	if err := glm.registry.DeleteLayeredPrompt(layer, identifier); err != nil {
		return fmt.Errorf("failed to delete prompt layer: %w", err)
	}
	
	// Invalidate relevant caches
	return glm.invalidateRelevantCaches(ctx, layer, artisanID, sessionID)
}

// ListPromptLayers returns all layers for an artisan/session
func (glm *GuildLayeredManager) ListPromptLayers(
	ctx context.Context,
	artisanID, sessionID string,
) ([]SystemPrompt, error) {
	var allPrompts []SystemPrompt
	
	// Define layer order for consistency
	layers := []PromptLayer{
		LayerPlatform,
		LayerGuild,
		LayerRole,
		LayerDomain,
		LayerSession,
		LayerTurn,
	}
	
	for _, layer := range layers {
		identifier := glm.getLayerIdentifier(layer, artisanID, sessionID)
		
		prompt, err := glm.registry.GetLayeredPrompt(layer, identifier)
		if err != nil {
			// Layer may not exist - this is OK for optional layers
			continue
		}
		
		allPrompts = append(allPrompts, *prompt)
	}
	
	return allPrompts, nil
}

// InvalidateCache clears the layered prompt cache
func (glm *GuildLayeredManager) InvalidateCache(ctx context.Context, artisanID, sessionID string) error {
	// Clear in-memory cache in assembler
	glm.assembler.clearCache(artisanID, sessionID)
	
	// Clear persistent cache
	pattern := fmt.Sprintf("artisan:%s:session:%s", artisanID, sessionID)
	return glm.store.InvalidatePromptCache(ctx, pattern)
}

// Helper methods

// getLayerIdentifier determines the correct identifier for a layer
func (glm *GuildLayeredManager) getLayerIdentifier(layer PromptLayer, artisanID, sessionID string) string {
	switch layer {
	case LayerPlatform:
		return "default" // Platform prompts are global
	case LayerGuild:
		// TODO: Get actual guild ID from artisan config
		return "default" // For now, use default guild
	case LayerRole:
		return extractRole(artisanID) // Extract role from artisan ID
	case LayerDomain:
		return fmt.Sprintf("%s:%s", extractRole(artisanID), extractDomain(artisanID))
	case LayerSession:
		return sessionID // Session prompts are per-session
	case LayerTurn:
		return fmt.Sprintf("%s:%s", artisanID, sessionID) // Turn prompts are per-artisan-session
	default:
		return "default"
	}
}

// validatePrompt ensures the prompt is valid before storing
func (glm *GuildLayeredManager) validatePrompt(prompt SystemPrompt) error {
	if prompt.Layer == "" {
		return fmt.Errorf("prompt layer is required")
	}
	
	if prompt.Content == "" {
		return fmt.Errorf("prompt content is required")
	}
	
	// Validate layer-specific requirements
	switch prompt.Layer {
	case LayerSession:
		if prompt.SessionID == "" {
			return fmt.Errorf("session ID is required for session layer prompts")
		}
	case LayerTurn:
		if prompt.ArtisanID == "" || prompt.SessionID == "" {
			return fmt.Errorf("artisan ID and session ID are required for turn layer prompts")
		}
	}
	
	return nil
}

// invalidateRelevantCaches clears caches that might be affected by a prompt change
func (glm *GuildLayeredManager) invalidateRelevantCaches(
	ctx context.Context,
	layer PromptLayer,
	artisanID, sessionID string,
) error {
	// Clear specific cache entries
	if artisanID != "" && sessionID != "" {
		if err := glm.InvalidateCache(ctx, artisanID, sessionID); err != nil {
			return err
		}
	}
	
	// For global layers, clear broader caches
	switch layer {
	case LayerPlatform, LayerGuild:
		// These affect all artisans, so clear more broadly
		return glm.store.InvalidatePromptCache(ctx, "artisan:")
	case LayerRole:
		// Clear cache for all artisans with this role
		role := extractRole(artisanID)
		pattern := fmt.Sprintf("artisan:%s", role)
		return glm.store.InvalidatePromptCache(ctx, pattern)
	}
	
	return nil
}

// extractRole extracts the role from an artisan ID
func extractRole(artisanID string) string {
	// Artisan IDs follow pattern: role-domain-instance (e.g., "backend-dev-001")
	// Extract the first part as the role
	parts := strings.Split(artisanID, "-")
	if len(parts) > 0 {
		return parts[0]
	}
	return "artisan" // Default role
}

// extractDomain extracts the domain from an artisan ID
func extractDomain(artisanID string) string {
	// Extract the second part as the domain
	parts := strings.Split(artisanID, "-")
	if len(parts) > 1 {
		return parts[1]
	}
	return "default" // Default domain
}

// Additional methods for the assembler

// clearCache clears in-memory cache for specific artisan/session
func (lpa *LayeredPromptAssembler) clearCache(artisanID, sessionID string) {
	lpa.mutex.Lock()
	defer lpa.mutex.Unlock()
	
	// Find and remove all cache entries for this artisan/session
	pattern := fmt.Sprintf("artisan:%s:session:%s", artisanID, sessionID)
	for key := range lpa.cache {
		if strings.HasPrefix(key, pattern) {
			delete(lpa.cache, key)
		}
	}
}

// GetMetrics returns performance metrics for the layered prompt system
func (glm *GuildLayeredManager) GetMetrics(ctx context.Context) (*PromptMetrics, error) {
	// TODO: Implement comprehensive metrics collection
	return &PromptMetrics{
		CacheHitRate:    0.85, // Placeholder
		AverageTokens:   1247, // Placeholder
		AssemblyTime:    time.Millisecond * 8, // Placeholder
		ActiveSessions:  42, // Placeholder
		LayerUsage: map[PromptLayer]int{
			LayerPlatform: 100,
			LayerGuild:    95,
			LayerRole:     87,
			LayerDomain:   65,
			LayerSession:  23,
			LayerTurn:     12,
		},
	}, nil
}

// PromptMetrics represents performance metrics for the layered prompt system
type PromptMetrics struct {
	CacheHitRate   float64                 `json:"cache_hit_rate"`
	AverageTokens  int                     `json:"average_tokens"`
	AssemblyTime   time.Duration           `json:"assembly_time"`
	ActiveSessions int                     `json:"active_sessions"`
	LayerUsage     map[PromptLayer]int     `json:"layer_usage"`
	LastUpdated    time.Time               `json:"last_updated"`
}

// GetLayerStats returns statistics for a specific layer
func (glm *GuildLayeredManager) GetLayerStats(ctx context.Context, layer PromptLayer) (*LayerStats, error) {
	prompts, err := glm.registry.ListLayeredPrompts(layer)
	if err != nil {
		return nil, err
	}
	
	stats := &LayerStats{
		Layer:        layer,
		PromptCount:  len(prompts),
		LastUpdated:  time.Now(),
	}
	
	// Calculate average tokens and find most recent update
	totalTokens := 0
	var mostRecent time.Time
	
	for _, prompt := range prompts {
		tokens := len(prompt.Content) / 4 // Rough estimate
		totalTokens += tokens
		
		if prompt.Updated.After(mostRecent) {
			mostRecent = prompt.Updated
		}
	}
	
	if len(prompts) > 0 {
		stats.AverageTokens = totalTokens / len(prompts)
		stats.LastUpdated = mostRecent
	}
	
	return stats, nil
}

// LayerStats represents statistics for a specific prompt layer
type LayerStats struct {
	Layer         PromptLayer `json:"layer"`
	PromptCount   int         `json:"prompt_count"`
	AverageTokens int         `json:"average_tokens"`
	LastUpdated   time.Time   `json:"last_updated"`
}