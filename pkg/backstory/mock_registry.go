// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package backstory

import (
	"github.com/guild-ventures/guild-core/pkg/prompts/layered"
)

// MockLayeredRegistry for testing and demonstrations
type MockLayeredRegistry struct {
	prompts map[string]map[string]layered.SystemPrompt
}

// NewMockLayeredRegistry creates a new mock registry
func NewMockLayeredRegistry() *MockLayeredRegistry {
	return &MockLayeredRegistry{
		prompts: make(map[string]map[string]layered.SystemPrompt),
	}
}

func (m *MockLayeredRegistry) RegisterPrompt(role, domain, prompt string) error {
	return nil
}

func (m *MockLayeredRegistry) RegisterTemplate(name, template string) error {
	return nil
}

func (m *MockLayeredRegistry) GetPrompt(role, domain string) (string, error) {
	return "", layered.ErrPromptNotFound
}

func (m *MockLayeredRegistry) GetTemplate(name string) (string, error) {
	return "", layered.ErrTemplateNotFound
}

func (m *MockLayeredRegistry) RegisterLayeredPrompt(layer layered.PromptLayer, identifier string, prompt layered.SystemPrompt) error {
	layerKey := string(layer)
	if m.prompts[layerKey] == nil {
		m.prompts[layerKey] = make(map[string]layered.SystemPrompt)
	}
	m.prompts[layerKey][identifier] = prompt
	return nil
}

func (m *MockLayeredRegistry) GetLayeredPrompt(layer layered.PromptLayer, identifier string) (*layered.SystemPrompt, error) {
	layerKey := string(layer)
	if prompts, exists := m.prompts[layerKey]; exists {
		if prompt, found := prompts[identifier]; found {
			return &prompt, nil
		}
	}
	return nil, layered.ErrPromptNotFound
}

func (m *MockLayeredRegistry) ListLayeredPrompts(layer layered.PromptLayer) ([]layered.SystemPrompt, error) {
	layerKey := string(layer)
	if prompts, exists := m.prompts[layerKey]; exists {
		result := make([]layered.SystemPrompt, 0, len(prompts))
		for _, prompt := range prompts {
			result = append(result, prompt)
		}
		return result, nil
	}
	return nil, nil
}

func (m *MockLayeredRegistry) DeleteLayeredPrompt(layer layered.PromptLayer, identifier string) error {
	layerKey := string(layer)
	if prompts, exists := m.prompts[layerKey]; exists {
		delete(prompts, identifier)
	}
	return nil
}

func (m *MockLayeredRegistry) GetDefaultPrompts(layer layered.PromptLayer) ([]layered.SystemPrompt, error) {
	return nil, nil
}