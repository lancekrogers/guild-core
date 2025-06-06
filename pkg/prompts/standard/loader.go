package standard

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
func (pm *PromptManager) RenderPrompt(name string, data interface{}) (string, error) {
	return pm.enhanced.RenderPrompt(name, data)
}
