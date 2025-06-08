package prompts

import (
	"bytes"
	"sync"
	"text/template"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/prompts/standard/templates/commission"
)

// PromptManager handles loading and rendering of system prompts
type PromptManager struct {
	templates map[string]*template.Template
	mu        sync.RWMutex
}

// NewPromptManager creates a new prompt manager with all available prompts
func NewPromptManager() (*PromptManager, error) {
	pm := &PromptManager{
		templates: make(map[string]*template.Template),
	}

	// Load commission prompts
	commissionTemplates, err := commission.LoadPrompts()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "error loading commission prompts").
			WithComponent("prompts").
			WithOperation("NewPromptManager")
	}

	// Add to template map
	for name, tmpl := range commissionTemplates {
		pm.templates[name] = tmpl
	}

	// In the future, add other prompt categories here
	// Example:
	// agentTemplates, err := agent.LoadPrompts()
	// if err != nil {
	//     return nil, fmt.Errorf("error loading agent prompts: %w", err)
	// }
	// for name, tmpl := range agentTemplates {
	//     pm.templates[name] = tmpl
	// }

	return pm, nil
}

// RenderPrompt renders a prompt template with the given data
func (pm *PromptManager) RenderPrompt(name string, data interface{}) (string, error) {
	pm.mu.RLock()
	tmpl, exists := pm.templates[name]
	pm.mu.RUnlock()

	if !exists {
		return "", gerror.New(gerror.ErrCodeNotFound, "prompt template not found", nil).
			WithComponent("prompts").
			WithOperation("RenderPrompt").
			WithDetails("template_name", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "error rendering prompt template").
			WithComponent("prompts").
			WithOperation("RenderPrompt").
			WithDetails("template_name", name)
	}

	return buf.String(), nil
}

// HasPrompt checks if a prompt with the given name exists
func (pm *PromptManager) HasPrompt(name string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	_, exists := pm.templates[name]
	return exists
}

// ListPrompts returns a list of all available prompt names
func (pm *PromptManager) ListPrompts() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	prompts := make([]string, 0, len(pm.templates))
	for name := range pm.templates {
		prompts = append(prompts, name)
	}
	return prompts
}

// RefreshPrompts reloads all prompts from their sources
func (pm *PromptManager) RefreshPrompts() error {
	// Load commission prompts
	commissionTemplates, err := commission.LoadPrompts()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "error refreshing commission prompts").
			WithComponent("prompts").
			WithOperation("RefreshPrompts")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Clear existing templates
	pm.templates = make(map[string]*template.Template)

	// Add commission templates
	for name, tmpl := range commissionTemplates {
		pm.templates[name] = tmpl
	}

	// In the future, refresh other prompt categories here

	return nil
}
