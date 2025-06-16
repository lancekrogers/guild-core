// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package standard

import (
	"context"
)

// PromptManager handles loading and rendering prompt templates (legacy compatibility)
type PromptManager struct {
	enhanced *EnhancedPromptManager
}

// NewPromptManager creates a new prompt manager with enhanced features
func NewPromptManager() (*PromptManager, error) {
	enhanced, err := NewEnhancedPromptManager()
	if err != nil {
		return nil, err
	}

	return &PromptManager{
		enhanced: enhanced,
	}, nil
}

// RenderPrompt renders a prompt template with the given data (delegates to enhanced manager)
func (pm *PromptManager) RenderPrompt(ctx context.Context, name string, data interface{}) (string, error) {
	return pm.enhanced.RenderPrompt(name, data)
}

// LoadTemplate loads a template from the filesystem
func (pm *PromptManager) LoadTemplate(ctx context.Context, path string) error {
	// For now, just return nil as templates are embedded
	// In the future, this could load external templates
	return nil
}

// GetType returns the type of this manager
func (pm *PromptManager) GetType() string {
	return "standard"
}
