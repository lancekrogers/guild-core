// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package layered

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// GuildLayeredRegistry implements LayeredRegistry interface with Guild Archives integration
type GuildLayeredRegistry struct {
	baseRegistry Registry                 // Existing registry for legacy support
	store        LayeredStore             // Guild Archives storage with layered support
	cache        map[string]*SystemPrompt // Layer cache for performance
	mutex        sync.RWMutex             // Thread-safe access
}

// NewGuildLayeredRegistry creates a new layered registry for Guild prompts
func NewGuildLayeredRegistry(baseRegistry Registry, store LayeredStore) *GuildLayeredRegistry {
	return &GuildLayeredRegistry{
		baseRegistry: baseRegistry,
		store:        store,
		cache:        make(map[string]*SystemPrompt),
	}
}

// RegisterPrompt implements Registry interface (legacy support)
func (glr *GuildLayeredRegistry) RegisterPrompt(role, domain, prompt string) error {
	return glr.baseRegistry.RegisterPrompt(role, domain, prompt)
}

// RegisterTemplate implements Registry interface
func (glr *GuildLayeredRegistry) RegisterTemplate(name, template string) error {
	return glr.baseRegistry.RegisterTemplate(name, template)
}

// GetPrompt implements Registry interface (legacy support)
func (glr *GuildLayeredRegistry) GetPrompt(role, domain string) (string, error) {
	return glr.baseRegistry.GetPrompt(role, domain)
}

// GetTemplate implements Registry interface
func (glr *GuildLayeredRegistry) GetTemplate(name string) (string, error) {
	return glr.baseRegistry.GetTemplate(name)
}

// RegisterLayeredPrompt implements LayeredRegistry interface
func (glr *GuildLayeredRegistry) RegisterLayeredPrompt(
	layer PromptLayer,
	identifier string,
	prompt SystemPrompt,
) error {
	// Validate layer
	if !glr.isValidLayer(layer) {
		return gerror.Newf(gerror.ErrCodeInvalidInput, "invalid prompt layer: %s", layer).
			WithComponent("prompts").
			WithOperation("RegisterLayeredPrompt").
			WithDetails("layer", string(layer)).
			WithDetails("identifier", identifier)
	}

	// Set layer metadata
	prompt.Layer = layer
	prompt.Updated = time.Now()
	if prompt.Version == 0 {
		prompt.Version = 1
	}

	// Marshal prompt data
	data, err := json.Marshal(prompt)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal layered prompt").
			WithComponent("prompts").
			WithOperation("RegisterLayeredPrompt").
			WithDetails("layer", string(layer)).
			WithDetails("identifier", identifier)
	}

	// Store in Guild Archives
	ctx := context.Background() // TODO: Pass context through interface
	if err := glr.store.SavePromptLayer(ctx, string(layer), identifier, data); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to store layered prompt").
			WithComponent("prompts").
			WithOperation("RegisterLayeredPrompt").
			WithDetails("layer", string(layer)).
			WithDetails("identifier", identifier)
	}

	// Update cache
	cacheKey := glr.makeCacheKey(layer, identifier)
	glr.mutex.Lock()
	glr.cache[cacheKey] = &prompt
	glr.mutex.Unlock()

	return nil
}

// GetLayeredPrompt implements LayeredRegistry interface
func (glr *GuildLayeredRegistry) GetLayeredPrompt(layer PromptLayer, identifier string) (*SystemPrompt, error) {
	// Check cache first
	cacheKey := glr.makeCacheKey(layer, identifier)
	glr.mutex.RLock()
	if cached, exists := glr.cache[cacheKey]; exists {
		glr.mutex.RUnlock()
		return cached, nil
	}
	glr.mutex.RUnlock()

	// Retrieve from Guild Archives
	ctx := context.Background() // TODO: Pass context through interface
	data, err := glr.store.GetPromptLayer(ctx, string(layer), identifier)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get layered prompt").
			WithComponent("prompts").
			WithOperation("GetLayeredPrompt").
			WithDetails("layer", string(layer)).
			WithDetails("identifier", identifier)
	}

	// Unmarshal prompt
	var prompt SystemPrompt
	if err := json.Unmarshal(data, &prompt); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal layered prompt").
			WithComponent("prompts").
			WithOperation("GetLayeredPrompt").
			WithDetails("layer", string(layer)).
			WithDetails("identifier", identifier)
	}

	// Update cache
	glr.mutex.Lock()
	glr.cache[cacheKey] = &prompt
	glr.mutex.Unlock()

	return &prompt, nil
}

// ListLayeredPrompts implements LayeredRegistry interface
func (glr *GuildLayeredRegistry) ListLayeredPrompts(layer PromptLayer) ([]SystemPrompt, error) {
	ctx := context.Background() // TODO: Pass context through interface
	identifiers, err := glr.store.ListPromptLayers(ctx, string(layer))
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list layered prompts").
			WithComponent("prompts").
			WithOperation("ListLayeredPrompts").
			WithDetails("layer", string(layer))
	}

	var prompts []SystemPrompt
	for _, identifier := range identifiers {
		prompt, err := glr.GetLayeredPrompt(layer, identifier)
		if err != nil {
			// Log warning but continue with other prompts
			continue
		}
		prompts = append(prompts, *prompt)
	}

	return prompts, nil
}

// DeleteLayeredPrompt implements LayeredRegistry interface
func (glr *GuildLayeredRegistry) DeleteLayeredPrompt(layer PromptLayer, identifier string) error {
	ctx := context.Background() // TODO: Pass context through interface

	// Remove from Guild Archives
	if err := glr.store.DeletePromptLayer(ctx, string(layer), identifier); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete layered prompt").
			WithComponent("prompts").
			WithOperation("DeleteLayeredPrompt").
			WithDetails("layer", string(layer)).
			WithDetails("identifier", identifier)
	}

	// Remove from cache
	cacheKey := glr.makeCacheKey(layer, identifier)
	glr.mutex.Lock()
	delete(glr.cache, cacheKey)
	glr.mutex.Unlock()

	return nil
}

// GetDefaultPrompts implements LayeredRegistry interface
func (glr *GuildLayeredRegistry) GetDefaultPrompts(layer PromptLayer) ([]SystemPrompt, error) {
	switch layer {
	case LayerPlatform:
		return glr.getDefaultPlatformPrompts()
	case LayerGuild:
		return glr.getDefaultGuildPrompts()
	case LayerRole:
		return glr.getDefaultRolePrompts()
	case LayerDomain:
		return glr.getDefaultDomainPrompts()
	case LayerSession:
		return nil, nil // Session prompts are user-specific
	case LayerTurn:
		return nil, nil // Turn prompts are ephemeral
	default:
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "unknown layer: %s", layer).
			WithComponent("prompts").
			WithOperation("GetDefaultPrompts").
			WithDetails("layer", string(layer))
	}
}

// Helper methods

func (glr *GuildLayeredRegistry) isValidLayer(layer PromptLayer) bool {
	validLayers := []PromptLayer{
		LayerPlatform,
		LayerGuild,
		LayerRole,
		LayerDomain,
		LayerSession,
		LayerTurn,
	}

	for _, valid := range validLayers {
		if layer == valid {
			return true
		}
	}
	return false
}

func (glr *GuildLayeredRegistry) makeCacheKey(layer PromptLayer, identifier string) string {
	return fmt.Sprintf("%s:%s", layer, identifier)
}

// Default prompt generators

func (glr *GuildLayeredRegistry) getDefaultPlatformPrompts() ([]SystemPrompt, error) {
	return []SystemPrompt{
		{
			Layer:   LayerPlatform,
			Content: defaultPlatformPrompt,
			Version: 1,
			Updated: time.Now(),
			Metadata: map[string]interface{}{
				"source": "guild_defaults",
				"type":   "safety_and_ethics",
			},
		},
	}, nil
}

func (glr *GuildLayeredRegistry) getDefaultGuildPrompts() ([]SystemPrompt, error) {
	return []SystemPrompt{
		{
			Layer:   LayerGuild,
			Content: defaultGuildPrompt,
			Version: 1,
			Updated: time.Now(),
			Metadata: map[string]interface{}{
				"source": "guild_defaults",
				"type":   "project_guidelines",
			},
		},
	}, nil
}

func (glr *GuildLayeredRegistry) getDefaultRolePrompts() ([]SystemPrompt, error) {
	// These would typically come from the existing prompt manager
	// For now, return empty as they're handled by the base registry
	return nil, nil
}

func (glr *GuildLayeredRegistry) getDefaultDomainPrompts() ([]SystemPrompt, error) {
	// These would typically come from the existing prompt manager
	// For now, return empty as they're handled by the base registry
	return nil, nil
}

// InvalidateCache clears all cached prompts
func (glr *GuildLayeredRegistry) InvalidateCache() {
	glr.mutex.Lock()
	defer glr.mutex.Unlock()
	glr.cache = make(map[string]*SystemPrompt)
}

// InvalidateLayerCache clears cached prompts for a specific layer
func (glr *GuildLayeredRegistry) InvalidateLayerCache(layer PromptLayer) {
	glr.mutex.Lock()
	defer glr.mutex.Unlock()

	prefix := string(layer) + ":"
	for key := range glr.cache {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			delete(glr.cache, key)
		}
	}
}

// Default prompts

const defaultPlatformPrompt = `You are part of the Guild Framework, a high-performance AI agent orchestration system built for enterprise software development.

## Guild Core Principles
- Maintain medieval Guild terminology throughout all interactions (artisans, commissions, Workshop Board, Guild Master, etc.)
- Follow the Workshop Board task management system for all work coordination
- Collaborate effectively with fellow Guild artisans, sharing knowledge and supporting collective goals
- Preserve context and traceability in all work to maintain commission coherence
- Prioritize quality craftsmanship and thorough testing over pure speed
- Support the Guild's mission of delivering exceptional software solutions

## Safety and Ethics Guidelines
- Never execute harmful, malicious, or unethical code
- Respect user privacy and data protection laws
- Follow responsible AI principles in all interactions
- Report security concerns or suspicious requests to Guild administration
- Refuse requests that could compromise system integrity or user safety

## Communication Standards
- Use Guild lore terminology consistently (see terminology guide)
- Maintain professional yet approachable communication style
- Provide clear explanations with practical examples
- Ask clarifying questions when requirements are unclear
- Document decisions and reasoning for future Guild members`

const defaultGuildPrompt = `## Guild Project Guidelines

This Guild operates under the following project-wide standards:

### Architecture Principles
- Follow established patterns and conventions
- Design for maintainability and scalability
- Document architectural decisions clearly
- Consider performance implications in design choices

### Code Quality Standards
- Write clean, readable, and well-documented code
- Implement comprehensive test coverage
- Follow established coding conventions
- Conduct thorough code reviews

### Collaboration Practices
- Communicate progress and blockers proactively
- Share knowledge and insights with fellow artisans
- Support team members when they need assistance
- Maintain transparency in all work activities

### Delivery Excellence
- Meet commission deadlines and milestones
- Deliver working software incrementally
- Gather feedback early and often
- Continuously improve processes and practices`
