// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package prompts provides prompt management systems for Guild agents
package prompts

import (
	"context"
	"errors"
)

// ErrPromptNotFound is returned when a requested prompt does not exist
var ErrPromptNotFound = errors.New("prompt not found")

// ErrTemplateNotFound is returned when a requested template does not exist
var ErrTemplateNotFound = errors.New("template not found")

// Manager is the base interface for all prompt management systems
type Manager interface {
	// RenderPrompt renders a prompt template with the given data
	RenderPrompt(ctx context.Context, name string, data interface{}) (string, error)

	// LoadTemplate loads a template from the filesystem
	LoadTemplate(ctx context.Context, path string) error

	// GetType returns the type of prompt manager (standard, layered, etc.)
	GetType() string
}

// StandardManager provides basic template-based prompt rendering
type StandardManager interface {
	Manager

	// LoadFromDirectory loads all templates from a directory
	LoadFromDirectory(ctx context.Context, dir string) error
}

// LayeredManager provides the advanced 6-layer prompt system
type LayeredManager interface {
	Manager

	// SetLayer sets content for a specific layer
	SetLayer(layer PromptLayer, content string) error

	// GetCompiledPrompt compiles all layers into a final prompt
	GetCompiledPrompt(ctx context.Context, config LayerConfig) (string, error)

	// ClearLayer removes content from a specific layer
	ClearLayer(layer PromptLayer) error
}

// PromptLayer represents the hierarchical layers of Guild prompts
type PromptLayer string

const (
	// LayerPlatform contains core Guild platform rules (terms of service, safety)
	LayerPlatform PromptLayer = "platform"

	// LayerGuild contains project-wide goals and style guidelines
	LayerGuild PromptLayer = "guild"

	// LayerRole contains artisan role definitions (Guild Master, Code Artisan, etc.)
	LayerRole PromptLayer = "role"

	// LayerDomain contains project type specializations (web-app, cli-tool, etc.)
	LayerDomain PromptLayer = "domain"

	// LayerSession contains user preferences and session-specific context
	LayerSession PromptLayer = "session"

	// LayerTurn contains ephemeral instructions for single interactions
	LayerTurn PromptLayer = "turn"
)

// LayerConfig provides configuration for compiling layered prompts
type LayerConfig struct {
	// AgentID is the ID of the agent requesting the prompt
	AgentID string

	// SessionID is the current session ID
	SessionID string

	// Role is the agent's role (e.g., "guild_master", "code_artisan")
	Role string

	// Domain is the project domain (e.g., "web-app", "cli-tool")
	Domain string

	// IncludeLayers specifies which layers to include
	IncludeLayers []PromptLayer

	// MaxTokens limits the compiled prompt size
	MaxTokens int
}
