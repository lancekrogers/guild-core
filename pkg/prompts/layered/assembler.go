// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package layered

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// LayeredPromptAssembler implements the core Guild layered prompt system
type LayeredPromptAssembler struct {
	manager      Manager                   // Existing prompt manager
	formatter    Formatter                 // Context formatter
	store        LayeredStore              // Guild Archives storage with layered support
	ragRetriever RAGRetriever              // Memory retrieval interface
	tokenBudget  int                       // Maximum tokens for assembled prompt
	cache        map[string]*LayeredPrompt // In-memory cache
	mutex        sync.RWMutex              // Thread-safe cache access
}

// RAGRetriever interface for memory chunk retrieval
type RAGRetriever interface {
	GetContextualMemory(ctx context.Context, sessionID, query string, maxTokens int, threshold float64) ([]MemoryChunk, error)
}

// MemoryChunk represents a piece of contextual memory
type MemoryChunk struct {
	Content   string                 `json:"content"`
	Score     float64                `json:"score"`
	Source    string                 `json:"source"`
	Tokens    int                    `json:"tokens"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewLayeredPromptAssembler creates a new Guild layered prompt assembler
func NewLayeredPromptAssembler(
	manager Manager,
	formatter Formatter,
	store LayeredStore,
	ragRetriever RAGRetriever,
	tokenBudget int,
) *LayeredPromptAssembler {
	return &LayeredPromptAssembler{
		manager:      manager,
		formatter:    formatter,
		store:        store,
		ragRetriever: ragRetriever,
		tokenBudget:  tokenBudget,
		cache:        make(map[string]*LayeredPrompt),
	}
}

// BuildPrompt assembles a complete layered prompt for a Guild artisan
func (lpa *LayeredPromptAssembler) BuildPrompt(
	ctx context.Context,
	artisanID, sessionID string,
	turnCtx TurnContext,
) (*LayeredPrompt, error) {
	// Generate cache key
	cacheKey := lpa.generateCacheKey(artisanID, sessionID, turnCtx)

	// Check cache first
	if cached := lpa.getCachedPrompt(cacheKey); cached != nil {
		return cached, nil
	}

	// Get artisan configuration from registry
	artisan, err := lpa.getArtisanConfig(ctx, artisanID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get artisan config").
			WithComponent("prompts").
			WithOperation("BuildPrompt").
			WithDetails("artisan_id", artisanID)
	}

	// Collect all prompt layers in priority order
	layers, err := lpa.collectPromptLayers(ctx, artisan, sessionID, turnCtx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to collect prompt layers").
			WithComponent("prompts").
			WithOperation("BuildPrompt").
			WithDetails("artisan_id", artisanID).
			WithDetails("session_id", sessionID)
	}

	// Retrieve and inject RAG memory if available
	memoryChunks, err := lpa.retrieveMemoryChunks(ctx, sessionID, turnCtx)
	if err != nil {
		// Log warning but don't fail - memory retrieval is optional
		// TODO: Add proper logging
	}

	// Apply token budget and intelligent truncation
	optimizedLayers, truncated, err := lpa.optimizeForTokenBudget(layers, memoryChunks)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to optimize for token budget").
			WithComponent("prompts").
			WithOperation("BuildPrompt").
			WithDetails("artisan_id", artisanID).
			WithDetails("token_budget", lpa.tokenBudget)
	}

	// Compile the final prompt
	compiled := lpa.compilePrompt(optimizedLayers, memoryChunks, turnCtx)

	// Create the layered prompt result
	layeredPrompt := &LayeredPrompt{
		Layers:      optimizedLayers,
		Compiled:    compiled,
		TokenCount:  lpa.estimateTokens(compiled),
		Truncated:   truncated,
		CacheKey:    cacheKey,
		ArtisanID:   artisanID,
		SessionID:   sessionID,
		AssembledAt: time.Now(),
		Metadata: map[string]interface{}{
			"layer_count":   len(optimizedLayers),
			"memory_chunks": len(memoryChunks),
			"token_budget":  lpa.tokenBudget,
			"turn_context":  turnCtx.UserMessage != "",
		},
	}

	// Cache the result
	lpa.cachePrompt(cacheKey, layeredPrompt)

	// Store in persistent cache if significant
	if lpa.isSignificantPrompt(layeredPrompt) {
		if err := lpa.storePersistentCache(ctx, cacheKey, layeredPrompt); err != nil {
			// Log warning but don't fail
		}
	}

	return layeredPrompt, nil
}

// collectPromptLayers gathers all relevant prompt layers in priority order
func (lpa *LayeredPromptAssembler) collectPromptLayers(
	ctx context.Context,
	artisan *ArtisanConfig,
	sessionID string,
	turnCtx TurnContext,
) ([]SystemPrompt, error) {
	var layers []SystemPrompt

	// Layer order by priority (lowest to highest)
	layerOrder := []PromptLayer{
		LayerPlatform,
		LayerGuild,
		LayerRole,
		LayerDomain,
		LayerSession,
		LayerTurn,
	}

	for priority, layer := range layerOrder {
		prompt, err := lpa.getLayerPrompt(ctx, layer, artisan, sessionID, turnCtx)
		if err != nil {
			// Some layers may not exist - this is OK
			continue
		}

		if prompt != nil {
			prompt.Priority = priority
			layers = append(layers, *prompt)
		}
	}

	return layers, nil
}

// getLayerPrompt retrieves a prompt for a specific layer
func (lpa *LayeredPromptAssembler) getLayerPrompt(
	ctx context.Context,
	layer PromptLayer,
	artisan *ArtisanConfig,
	sessionID string,
	turnCtx TurnContext,
) (*SystemPrompt, error) {
	switch layer {
	case LayerPlatform:
		return lpa.getPlatformPrompt(ctx)
	case LayerGuild:
		return lpa.getGuildPrompt(ctx, artisan.GuildID)
	case LayerRole:
		return lpa.getRolePrompt(ctx, artisan.Role)
	case LayerDomain:
		return lpa.getDomainPrompt(ctx, artisan.Role, artisan.Domain)
	case LayerSession:
		return lpa.getSessionPrompt(ctx, sessionID)
	case LayerTurn:
		return lpa.getTurnPrompt(ctx, turnCtx)
	default:
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "unknown prompt layer: %s", layer).
			WithComponent("prompts").
			WithOperation("getLayerPrompt").
			WithDetails("layer", string(layer))
	}
}

// getPlatformPrompt retrieves the Guild platform-level prompt
func (lpa *LayeredPromptAssembler) getPlatformPrompt(ctx context.Context) (*SystemPrompt, error) {
	// Try to get from storage first
	data, err := lpa.store.GetPromptLayer(ctx, string(LayerPlatform), "default")
	if err == nil {
		var prompt SystemPrompt
		if err := json.Unmarshal(data, &prompt); err == nil {
			return &prompt, nil
		}
	}

	// Fall back to default platform prompt
	content := `You are part of the Guild Framework, a high-performance AI agent orchestration system.

## Guild Core Principles
- Maintain medieval Guild terminology throughout interactions
- Follow the Workshop Board task management system
- Collaborate effectively with other Guild artisans
- Preserve context and traceability in all work
- Prioritize quality craftsmanship over speed

## Safety Guidelines
- Never execute harmful or malicious code
- Respect user privacy and data protection
- Follow ethical AI principles
- Report security concerns to Guild administration

## Communication Style
- Use Guild lore terminology (artisans, commissions, Workshop Board, etc.)
- Be professional yet approachable
- Provide clear explanations with examples
- Ask clarifying questions when requirements are unclear`

	return &SystemPrompt{
		Layer:   LayerPlatform,
		Content: content,
		Version: 1,
		Updated: time.Now(),
		Metadata: map[string]interface{}{
			"source": "default_platform_prompt",
		},
	}, nil
}

// getGuildPrompt retrieves the guild-specific prompt
func (lpa *LayeredPromptAssembler) getGuildPrompt(ctx context.Context, guildID string) (*SystemPrompt, error) {
	if guildID == "" {
		return nil, nil // No guild-specific prompt
	}

	// Try to get from storage
	data, err := lpa.store.GetPromptLayer(ctx, string(LayerGuild), guildID)
	if err != nil {
		return nil, nil // Guild prompt is optional
	}

	var prompt SystemPrompt
	if err := json.Unmarshal(data, &prompt); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal guild prompt").
			WithComponent("prompts").
			WithOperation("getGuildPrompt").
			WithDetails("guild_id", guildID)
	}

	return &prompt, nil
}

// getRolePrompt retrieves the role-specific prompt from the registry
func (lpa *LayeredPromptAssembler) getRolePrompt(ctx context.Context, role string) (*SystemPrompt, error) {
	// Use existing manager for role prompts
	content, err := lpa.manager.GetSystemPrompt(ctx, role, "default")
	if err != nil {
		return nil, err
	}

	return &SystemPrompt{
		Layer:   LayerRole,
		Content: content,
		Version: 1,
		Updated: time.Now(),
		Metadata: map[string]interface{}{
			"role":   role,
			"source": "prompt_manager",
		},
	}, nil
}

// getDomainPrompt retrieves the domain-specific prompt from the registry
func (lpa *LayeredPromptAssembler) getDomainPrompt(ctx context.Context, role, domain string) (*SystemPrompt, error) {
	if domain == "" || domain == "default" {
		return nil, nil // No domain specialization
	}

	// Use existing manager for domain prompts
	content, err := lpa.manager.GetSystemPrompt(ctx, role, domain)
	if err != nil {
		return nil, nil // Domain prompt is optional
	}

	return &SystemPrompt{
		Layer:   LayerDomain,
		Content: content,
		Version: 1,
		Updated: time.Now(),
		Metadata: map[string]interface{}{
			"role":   role,
			"domain": domain,
			"source": "prompt_manager",
		},
	}, nil
}

// getSessionPrompt retrieves session-specific preferences
func (lpa *LayeredPromptAssembler) getSessionPrompt(ctx context.Context, sessionID string) (*SystemPrompt, error) {
	if sessionID == "" {
		return nil, nil // No session context
	}

	// Try to get from storage
	data, err := lpa.store.GetPromptLayer(ctx, string(LayerSession), sessionID)
	if err != nil {
		return nil, nil // Session prompt is optional
	}

	var prompt SystemPrompt
	if err := json.Unmarshal(data, &prompt); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal session prompt").
			WithComponent("prompts").
			WithOperation("getSessionPrompt").
			WithDetails("session_id", sessionID)
	}

	return &prompt, nil
}

// getTurnPrompt creates ephemeral turn-specific instructions
func (lpa *LayeredPromptAssembler) getTurnPrompt(ctx context.Context, turnCtx TurnContext) (*SystemPrompt, error) {
	if turnCtx.UserMessage == "" && len(turnCtx.Instructions) == 0 {
		return nil, nil // No turn context
	}

	var content strings.Builder
	content.WriteString("## Current Turn Context\n\n")

	if turnCtx.UserMessage != "" {
		content.WriteString(fmt.Sprintf("**User Request**: %s\n\n", turnCtx.UserMessage))
	}

	if turnCtx.TaskID != "" {
		content.WriteString(fmt.Sprintf("**Active Task**: %s\n", turnCtx.TaskID))
	}

	if turnCtx.CommissionID != "" {
		content.WriteString(fmt.Sprintf("**Commission**: %s\n", turnCtx.CommissionID))
	}

	if turnCtx.Urgency != "" {
		content.WriteString(fmt.Sprintf("**Urgency**: %s\n", turnCtx.Urgency))
	}

	if len(turnCtx.Instructions) > 0 {
		content.WriteString("\n**Special Instructions**:\n")
		for _, instruction := range turnCtx.Instructions {
			content.WriteString(fmt.Sprintf("- %s\n", instruction))
		}
	}

	return &SystemPrompt{
		Layer:   LayerTurn,
		Content: content.String(),
		Version: 1,
		Updated: time.Now(),
		Metadata: map[string]interface{}{
			"ephemeral":      true,
			"turn_context":   true,
			"has_task":       turnCtx.TaskID != "",
			"has_commission": turnCtx.CommissionID != "",
		},
	}, nil
}

// ArtisanConfig represents artisan configuration from registry
type ArtisanConfig struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Domain  string `json:"domain"`
	GuildID string `json:"guild_id"`
}

// getArtisanConfig retrieves artisan configuration from the registry
func (lpa *LayeredPromptAssembler) getArtisanConfig(ctx context.Context, artisanID string) (*ArtisanConfig, error) {
	// TODO: Implement proper registry lookup when agent registry is available
	// For now, return a default config based on artisanID

	// Parse artisanID to extract role and domain hints
	parts := strings.Split(artisanID, "-")
	role := "artisan"
	domain := "default"

	if len(parts) >= 2 {
		role = parts[0]
		if len(parts) >= 3 {
			domain = parts[1]
		}
	}

	return &ArtisanConfig{
		ID:      artisanID,
		Role:    role,
		Domain:  domain,
		GuildID: "", // Will be set when guild registry is implemented
	}, nil
}

// Helper methods for the assembler
func (lpa *LayeredPromptAssembler) generateCacheKey(artisanID, sessionID string, turnCtx TurnContext) string {
	// Create deterministic cache key
	key := fmt.Sprintf("artisan:%s:session:%s", artisanID, sessionID)

	// Add turn context hash for ephemeral turns
	if turnCtx.UserMessage != "" || len(turnCtx.Instructions) > 0 {
		h := md5.New()
		h.Write([]byte(turnCtx.UserMessage))
		for _, inst := range turnCtx.Instructions {
			h.Write([]byte(inst))
		}
		key += fmt.Sprintf(":turn:%x", h.Sum(nil)[:8])
	}

	return key
}

func (lpa *LayeredPromptAssembler) getCachedPrompt(cacheKey string) *LayeredPrompt {
	lpa.mutex.RLock()
	defer lpa.mutex.RUnlock()

	if prompt, exists := lpa.cache[cacheKey]; exists {
		// Check if cache is still fresh (5 minutes)
		if time.Since(prompt.AssembledAt) < 5*time.Minute {
			return prompt
		}
		// Remove stale cache entry
		delete(lpa.cache, cacheKey)
	}
	return nil
}

func (lpa *LayeredPromptAssembler) cachePrompt(cacheKey string, prompt *LayeredPrompt) {
	lpa.mutex.Lock()
	defer lpa.mutex.Unlock()
	lpa.cache[cacheKey] = prompt
}

// Additional helper methods continue in next part...

// retrieveMemoryChunks gets relevant memory chunks if RAG is available
func (lpa *LayeredPromptAssembler) retrieveMemoryChunks(
	ctx context.Context,
	sessionID string,
	turnCtx TurnContext,
) ([]MemoryChunk, error) {
	if lpa.ragRetriever == nil {
		return nil, nil
	}

	query := turnCtx.UserMessage
	if query == "" {
		return nil, nil
	}

	// Reserve 20% of token budget for memory
	memoryTokenBudget := int(float64(lpa.tokenBudget) * 0.2)

	return lpa.ragRetriever.GetContextualMemory(ctx, sessionID, query, memoryTokenBudget, 0.7)
}

// optimizeForTokenBudget applies intelligent truncation based on layer priority
func (lpa *LayeredPromptAssembler) optimizeForTokenBudget(
	layers []SystemPrompt,
	memoryChunks []MemoryChunk,
) ([]SystemPrompt, bool, error) {
	// Calculate total tokens needed
	totalTokens := 0
	for _, layer := range layers {
		totalTokens += lpa.estimateTokens(layer.Content)
	}

	for _, chunk := range memoryChunks {
		totalTokens += chunk.Tokens
	}

	// If within budget, return as-is
	if totalTokens <= lpa.tokenBudget {
		return layers, false, nil
	}

	// Sort layers by priority (higher priority = more important)
	sort.Slice(layers, func(i, j int) bool {
		return layers[i].Priority > layers[j].Priority
	})

	// Apply truncation strategy
	optimizedLayers := make([]SystemPrompt, 0, len(layers))
	remainingBudget := lpa.tokenBudget

	// Reserve tokens for memory chunks (they're high priority)
	memoryTokens := 0
	for _, chunk := range memoryChunks {
		memoryTokens += chunk.Tokens
	}
	remainingBudget -= memoryTokens

	// Add layers in priority order until budget is exhausted
	for _, layer := range layers {
		layerTokens := lpa.estimateTokens(layer.Content)

		if layerTokens <= remainingBudget {
			optimizedLayers = append(optimizedLayers, layer)
			remainingBudget -= layerTokens
		} else if remainingBudget > 100 { // If we have some tokens left, truncate the layer
			truncated := lpa.truncateContent(layer.Content, remainingBudget)
			truncatedLayer := layer
			truncatedLayer.Content = truncated
			optimizedLayers = append(optimizedLayers, truncatedLayer)
			break
		}
	}

	return optimizedLayers, true, nil
}

// compilePrompt combines all layers into the final prompt
func (lpa *LayeredPromptAssembler) compilePrompt(
	layers []SystemPrompt,
	memoryChunks []MemoryChunk,
	turnCtx TurnContext,
) string {
	var compiled strings.Builder

	// Add layers in reverse priority order (platform first, turn last)
	sort.Slice(layers, func(i, j int) bool {
		return layers[i].Priority < layers[j].Priority
	})

	for i, layer := range layers {
		if i > 0 {
			compiled.WriteString("\n\n---\n\n")
		}
		compiled.WriteString(layer.Content)
	}

	// Add memory chunks if available
	if len(memoryChunks) > 0 {
		compiled.WriteString("\n\n## Relevant Guild Memory\n\n")
		for _, chunk := range memoryChunks {
			compiled.WriteString(fmt.Sprintf("[[MEMORY:%s]] %s\n\n", chunk.Source, chunk.Content))
		}
	}

	// Add turn context if available
	if turnCtx.Context != nil {
		if contextStr, err := lpa.formatter.FormatAsXML(turnCtx.Context); err == nil {
			compiled.WriteString("\n\n## Current Task Context\n\n")
			compiled.WriteString(contextStr)
		}
	}

	return compiled.String()
}

// estimateTokens provides a rough token count estimate (4 chars ≈ 1 token)
func (lpa *LayeredPromptAssembler) estimateTokens(content string) int {
	return len(content) / 4
}

// truncateContent intelligently truncates content to fit token budget
func (lpa *LayeredPromptAssembler) truncateContent(content string, maxTokens int) string {
	maxChars := maxTokens * 4
	if len(content) <= maxChars {
		return content
	}

	// Try to truncate at sentence boundaries
	truncated := content[:maxChars]
	lastPeriod := strings.LastIndex(truncated, ".")
	if lastPeriod > maxChars/2 { // If we find a reasonable sentence boundary
		truncated = truncated[:lastPeriod+1]
	}

	return truncated + "\n\n[Content truncated due to token limit]"
}

// isSignificantPrompt determines if a prompt should be cached persistently
func (lpa *LayeredPromptAssembler) isSignificantPrompt(prompt *LayeredPrompt) bool {
	return len(prompt.Layers) >= 3 || prompt.TokenCount > 1000
}

// storePersistentCache stores the prompt in persistent storage
func (lpa *LayeredPromptAssembler) storePersistentCache(
	ctx context.Context,
	cacheKey string,
	prompt *LayeredPrompt,
) error {
	data, err := json.Marshal(prompt)
	if err != nil {
		return err
	}

	return lpa.store.CacheCompiledPrompt(ctx, cacheKey, data)
}
