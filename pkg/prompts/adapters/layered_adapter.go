// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package adapters

import (
	"context"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/prompts"
	"github.com/lancekrogers/guild/pkg/prompts/layered"
)

// LayeredAdapter adapts a layered.LayeredManager to implement prompts.Manager
type LayeredAdapter struct {
	layeredMgr layered.LayeredManager
}

// NewLayeredAdapter creates a new adapter for layered prompt manager
func NewLayeredAdapter(mgr layered.LayeredManager) prompts.Manager {
	return &LayeredAdapter{
		layeredMgr: mgr,
	}
}

// RenderPrompt renders a prompt template with the given data
func (a *LayeredAdapter) RenderPrompt(ctx context.Context, name string, data interface{}) (string, error) {
	// For compatibility, render as a template from the layered system
	template, err := a.layeredMgr.GetTemplate(ctx, name)
	if err != nil {
		return "", err
	}

	// TODO: Implement template rendering with data
	// For now, just return the template
	return template, nil
}

// LoadTemplate loads a template from the filesystem
func (a *LayeredAdapter) LoadTemplate(ctx context.Context, path string) error {
	// The layered system doesn't load from filesystem in the same way
	// This would need to be implemented based on requirements
	return gerror.New(gerror.ErrCodeNotImplemented, "LoadTemplate not implemented for layered system", nil).
		WithComponent("prompts").
		WithOperation("LoadTemplate")
}

// GetType returns the type of prompt manager
func (a *LayeredAdapter) GetType() string {
	return "layered"
}

// AsLayeredManager returns the underlying LayeredManager if this is a layered adapter
func (a *LayeredAdapter) AsLayeredManager() (prompts.LayeredManager, bool) {
	// We need to create another adapter that goes the other way
	return &LayeredManagerAdapter{mgr: a.layeredMgr}, true
}

// LayeredManagerAdapter adapts internal layered manager to public interface
type LayeredManagerAdapter struct {
	mgr layered.LayeredManager
}

// RenderPrompt implements prompts.Manager
func (l *LayeredManagerAdapter) RenderPrompt(ctx context.Context, name string, data interface{}) (string, error) {
	return l.mgr.GetTemplate(ctx, name)
}

// LoadTemplate implements prompts.Manager
func (l *LayeredManagerAdapter) LoadTemplate(ctx context.Context, path string) error {
	return gerror.New(gerror.ErrCodeNotImplemented, "LoadTemplate not implemented for layered system", nil).
		WithComponent("prompts").
		WithOperation("LoadTemplate")
}

// GetType implements prompts.Manager
func (l *LayeredManagerAdapter) GetType() string {
	return "layered"
}

// SetLayer implements prompts.LayeredManager
func (l *LayeredManagerAdapter) SetLayer(layer prompts.PromptLayer, content string) error {
	prompt := layered.SystemPrompt{
		Layer:   layered.PromptLayer(layer),
		Content: content,
	}
	return l.mgr.SetPromptLayer(context.Background(), prompt)
}

// GetCompiledPrompt implements prompts.LayeredManager
func (l *LayeredManagerAdapter) GetCompiledPrompt(ctx context.Context, config prompts.LayerConfig) (string, error) {
	// Convert LayerConfig to TurnContext for the internal system
	turnCtx := layered.TurnContext{
		// Map fields as needed
	}

	layeredPrompt, err := l.mgr.BuildLayeredPrompt(ctx, config.AgentID, config.SessionID, turnCtx)
	if err != nil {
		return "", err
	}

	return layeredPrompt.Compiled, nil
}

// ClearLayer implements prompts.LayeredManager
func (l *LayeredManagerAdapter) ClearLayer(layer prompts.PromptLayer) error {
	// Delete the layer for the default artisan/session
	return l.mgr.DeletePromptLayer(context.Background(), layered.PromptLayer(layer), "", "")
}
