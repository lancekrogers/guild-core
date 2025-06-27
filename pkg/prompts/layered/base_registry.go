// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package layered

import (
	"fmt"
	"sync"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// MemoryRegistry is an in-memory implementation of the Registry interface
type MemoryRegistry struct {
	prompts   map[string]string
	templates map[string]string
	mu        sync.RWMutex
}

// NewMemoryRegistry creates a new in-memory registry
func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		prompts:   make(map[string]string),
		templates: make(map[string]string),
	}
}

// RegisterPrompt registers a system prompt
func (r *MemoryRegistry) RegisterPrompt(role, domain, prompt string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if role == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "role cannot be empty", nil).
			WithComponent("prompts").
			WithOperation("RegisterPrompt")
	}
	if domain == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "domain cannot be empty", nil).
			WithComponent("prompts").
			WithOperation("RegisterPrompt").
			WithDetails("role", role)
	}
	if prompt == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "prompt cannot be empty", nil).
			WithComponent("prompts").
			WithOperation("RegisterPrompt").
			WithDetails("role", role).
			WithDetails("domain", domain)
	}

	key := r.promptKey(role, domain)
	r.prompts[key] = prompt
	return nil
}

// RegisterTemplate registers a template
func (r *MemoryRegistry) RegisterTemplate(name, template string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "template name cannot be empty", nil).
			WithComponent("prompts").
			WithOperation("RegisterTemplate")
	}
	if template == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "template cannot be empty", nil).
			WithComponent("prompts").
			WithOperation("RegisterTemplate").
			WithDetails("template_name", name)
	}

	r.templates[name] = template
	return nil
}

// GetPrompt retrieves a registered prompt
func (r *MemoryRegistry) GetPrompt(role, domain string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := r.promptKey(role, domain)
	if prompt, ok := r.prompts[key]; ok {
		return prompt, nil
	}
	return "", gerror.New(gerror.ErrCodeNotFound, "prompt not found", nil).
		WithComponent("prompts").
		WithOperation("GetPrompt").
		WithDetails("role", role).
		WithDetails("domain", domain)
}

// GetTemplate retrieves a registered template
func (r *MemoryRegistry) GetTemplate(name string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if template, ok := r.templates[name]; ok {
		return template, nil
	}
	return "", gerror.New(gerror.ErrCodeNotFound, "template not found", nil).
		WithComponent("prompts").
		WithOperation("GetTemplate").
		WithDetails("template_name", name)
}

// ListPrompts returns all registered prompt keys (for debugging/testing)
func (r *MemoryRegistry) ListPrompts() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.prompts))
	for key := range r.prompts {
		keys = append(keys, key)
	}
	return keys
}

// ListTemplates returns all registered template names (for debugging/testing)
func (r *MemoryRegistry) ListTemplates() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}

// Clear removes all registered prompts and templates
func (r *MemoryRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.prompts = make(map[string]string)
	r.templates = make(map[string]string)
}

// promptKey generates a key for prompt storage
func (r *MemoryRegistry) promptKey(role, domain string) string {
	return fmt.Sprintf("%s:%s", role, domain)
}
