package prompts

import (
	"context"
	"fmt"
	"sync"
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

// GetSystemPrompt retrieves a system prompt for a specific role and domain
func (m *DefaultManager) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.registry == nil {
		return "", fmt.Errorf("registry not initialized")
	}

	prompt, err := m.registry.GetPrompt(role, domain)
	if err != nil {
		// Try to get a default prompt for the role
		defaultPrompt, defaultErr := m.registry.GetPrompt(role, "default")
		if defaultErr != nil {
			return "", fmt.Errorf("prompt not found for role=%s, domain=%s: %w", role, domain, err)
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
		return "", fmt.Errorf("registry not initialized")
	}

	return m.registry.GetTemplate(templateName)
}

// FormatContext formats a context object into a string representation
func (m *DefaultManager) FormatContext(ctx context.Context, context Context) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.formatter == nil {
		return "", fmt.Errorf("formatter not initialized")
	}

	// Default to XML formatting for efficiency
	return m.formatter.FormatAsXML(context)
}

// ListRoles returns all available roles
func (m *DefaultManager) ListRoles(ctx context.Context) ([]string, error) {
	// For now, return predefined roles
	// In the future, this could be dynamic based on registry content
	return []string{
		"manager",      // Guild Master
		"developer",    // Code Artisan
		"reviewer",     // Quality Inspector
		"architect",    // Architecture Sage
		"tester",       // Test Crafter
		"documenter",   // Scribe
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