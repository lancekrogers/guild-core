// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package layered

import (
	"context"
	"fmt"
	"sync"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// DefaultManager is the default implementation of the Manager interface
type DefaultManager struct {
	registry  Registry
	formatter Formatter
	mu        sync.RWMutex
}

// NewDefaultManager creates a new default prompt manager
func NewDefaultManager(registry Registry, formatter Formatter) *DefaultManager {
	return &DefaultManager{
		registry:  registry,
		formatter: formatter,
	}
}

// NewLayeredPromptManager creates a new layered prompt manager with default configuration
func NewLayeredPromptManager() *DefaultManager {
	// Create a default registry and formatter
	registry := NewMemoryRegistry()
	formatter := &simpleFormatter{}
	return NewDefaultManager(registry, formatter)
}

// simpleFormatter provides basic formatting functionality
type simpleFormatter struct{}

// FormatAsXML formats content as XML
func (f *simpleFormatter) FormatAsXML(ctx Context) (string, error) {
	return fmt.Sprintf("<content>%v</content>", ctx), nil
}

// FormatAsMarkdown formats content as Markdown
func (f *simpleFormatter) FormatAsMarkdown(ctx Context) (string, error) {
	return fmt.Sprintf("## Content\n\n%v", ctx), nil
}

// OptimizeForTokens optimizes content for token usage
func (f *simpleFormatter) OptimizeForTokens(content string, maxTokens int) (string, error) {
	// Simple truncation for now
	if len(content) > maxTokens*4 { // Rough estimate: 1 token ≈ 4 chars
		return content[:maxTokens*4] + "...", nil
	}
	return content, nil
}

// GetSystemPrompt retrieves a system prompt for a specific role and domain
func (m *DefaultManager) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.registry == nil {
		return "", gerror.New(gerror.ErrCodeInternal, "registry not initialized", nil).
			WithComponent("prompts").
			WithOperation("GetSystemPrompt")
	}

	prompt, err := m.registry.GetPrompt(role, domain)
	if err != nil {
		// Try to get a default prompt for the role
		defaultPrompt, defaultErr := m.registry.GetPrompt(role, "default")
		if defaultErr != nil {
			return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "prompt not found").
				WithComponent("prompts").
				WithOperation("GetSystemPrompt").
				WithDetails("role", role).
				WithDetails("domain", domain)
		}
		return defaultPrompt, nil
	}

	return prompt, nil
}

// GetTemplate retrieves a named template
func (m *DefaultManager) GetTemplate(ctx context.Context, templateName string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.registry == nil {
		return "", gerror.New(gerror.ErrCodeInternal, "registry not initialized", nil).
			WithComponent("prompts").
			WithOperation("GetTemplate")
	}

	return m.registry.GetTemplate(templateName)
}

// FormatContext formats a context object into a string representation
func (m *DefaultManager) FormatContext(ctx context.Context, context Context) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.formatter == nil {
		return "", gerror.New(gerror.ErrCodeInternal, "formatter not initialized", nil).
			WithComponent("prompts").
			WithOperation("FormatContext")
	}

	// Default to XML formatting for efficiency
	return m.formatter.FormatAsXML(context)
}

// ListRoles returns all available roles
func (m *DefaultManager) ListRoles(ctx context.Context) ([]string, error) {
	// For now, return predefined roles
	// In the future, this could be dynamic based on registry content
	return []string{
		"manager",    // Guild Master
		"developer",  // Code Artisan
		"reviewer",   // Quality Inspector
		"architect",  // Architecture Sage
		"tester",     // Test Crafter
		"documenter", // Scribe
	}, nil
}

// ListDomains returns all available domains for a role
func (m *DefaultManager) ListDomains(ctx context.Context, role string) ([]string, error) {
	// For now, return predefined domains based on role
	// In the future, this could be dynamic based on registry content
	switch role {
	case "manager":
		return []string{
			"default",
			"web-app",
			"cli-tool",
			"library",
			"microservice",
			"data-pipeline",
		}, nil
	case "developer":
		return []string{
			"default",
			"backend",
			"frontend",
			"fullstack",
			"infrastructure",
			"integration",
		}, nil
	case "reviewer":
		return []string{
			"default",
			"code-quality",
			"security",
			"performance",
			"architecture",
		}, nil
	default:
		return []string{"default"}, nil
	}
}

// SetRegistry updates the registry (useful for testing)
func (m *DefaultManager) SetRegistry(registry Registry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry = registry
}

// SetFormatter updates the formatter (useful for testing)
func (m *DefaultManager) SetFormatter(formatter Formatter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.formatter = formatter
}

// RenderPrompt renders a prompt template with the given data
func (m *DefaultManager) RenderPrompt(ctx context.Context, name string, data interface{}) (string, error) {
	// For layered prompts, we use templates from the registry
	return m.GetTemplate(ctx, name)
}

// LoadTemplate loads a template from the filesystem
func (m *DefaultManager) LoadTemplate(ctx context.Context, path string) error {
	// For layered prompts, templates are managed through the registry
	// This is a no-op for compatibility
	return nil
}

// GetType returns the type of this manager
func (m *DefaultManager) GetType() string {
	return "layered"
}

// SetLayer sets content for a specific layer
func (m *DefaultManager) SetLayer(layer interface{}, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.registry == nil {
		return gerror.New(gerror.ErrCodeInternal, "registry not initialized", nil).
			WithComponent("prompts").
			WithOperation("SetLayer")
	}

	// Store layer content in registry as a template
	templateName := fmt.Sprintf("layer_%v", layer)
	return m.registry.RegisterTemplate(templateName, content)
}

// GetCompiledPrompt compiles all layers into a final prompt
func (m *DefaultManager) GetCompiledPrompt(ctx context.Context, config interface{}) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// This would compile layers based on config
	// For now, return a basic implementation
	return "", gerror.New(gerror.ErrCodeInternal, "GetCompiledPrompt not yet implemented", nil).
		WithComponent("prompts").
		WithOperation("GetCompiledPrompt")
}

// ClearLayer removes content from a specific layer
func (m *DefaultManager) ClearLayer(layer interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.registry == nil {
		return gerror.New(gerror.ErrCodeInternal, "registry not initialized", nil).
			WithComponent("prompts").
			WithOperation("ClearLayer")
	}

	// Clear by overwriting with empty content
	templateName := fmt.Sprintf("layer_%v", layer)
	return m.registry.RegisterTemplate(templateName, "")
}
