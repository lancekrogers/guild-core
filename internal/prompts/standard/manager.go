package standard

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
)

//go:embed objective/markdown/*.md objective/markdown/lite/*.md
var promptFS embed.FS

// EnhancedPromptManager handles loading, rendering, and managing prompt templates with metadata
type EnhancedPromptManager struct {
	templates map[string]*template.Template
	metadata  map[string]*PromptMetadata
	mu        sync.RWMutex
}

// NewEnhancedPromptManager creates a new enhanced prompt manager
func NewEnhancedPromptManager() (*EnhancedPromptManager, error) {
	templates := make(map[string]*template.Template)
	metadata := make(map[string]*PromptMetadata)

	// Load prompts from both directories
	if err := loadPromptsFromDir(promptFS, "objective/markdown", templates, metadata, "objective"); err != nil {
		return nil, fmt.Errorf("error loading prompts: %w", err)
	}
	if err := loadPromptsFromDir(promptFS, "objective/markdown/lite", templates, metadata, "objective.lite"); err != nil {
		return nil, fmt.Errorf("error loading lite prompts: %w", err)
	}

	return &EnhancedPromptManager{
		templates: templates,
		metadata:  metadata,
	}, nil
}

// RenderPrompt renders a prompt template with the given data
func (pm *EnhancedPromptManager) RenderPrompt(name string, data interface{}) (string, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	tmpl, exists := pm.templates[name]
	if !exists {
		return "", fmt.Errorf("prompt template %s not found", name)
	}

	// Validate required variables if metadata exists
	if meta, hasMetadata := pm.metadata[name]; hasMetadata {
		if dataMap, ok := data.(map[string]interface{}); ok {
			if err := meta.HasRequiredVariables(dataMap); err != nil {
				return "", fmt.Errorf("validation error: %w", err)
			}
		}
	}

	// Add helper functions for conditional blocks and results
	funcMap := template.FuncMap{
		"has": func(key string) bool {
			if dataMap, ok := data.(map[string]interface{}); ok {
				val, exists := dataMap[key]
				return exists && val != nil && val != ""
			}
			return false
		},
	}

	// Clone template with functions
	tmplWithFuncs, err := tmpl.Clone()
	if err != nil {
		return "", fmt.Errorf("error cloning template: %w", err)
	}
	tmplWithFuncs = tmplWithFuncs.Funcs(funcMap)

	var buf bytes.Buffer
	if err := tmplWithFuncs.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	// Process conditional blocks
	rendered := buf.String()
	rendered = processConditionalBlocks(rendered, data)
	
	return rendered, nil
}

// GetMetadata returns metadata for a prompt
func (pm *EnhancedPromptManager) GetMetadata(name string) (*PromptMetadata, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	meta, exists := pm.metadata[name]
	if !exists {
		return nil, fmt.Errorf("metadata for prompt %s not found", name)
	}
	return meta, nil
}

// ValidatePrompt validates a prompt with given data without rendering
func (pm *EnhancedPromptManager) ValidatePrompt(name string, data map[string]interface{}) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	meta, exists := pm.metadata[name]
	if !exists {
		// No metadata means no validation required
		return nil
	}

	return meta.HasRequiredVariables(data)
}

// ListPrompts returns all available prompts with their metadata
func (pm *EnhancedPromptManager) ListPrompts() map[string]*PromptMetadata {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Create a copy to avoid external modifications
	result := make(map[string]*PromptMetadata)
	for k, v := range pm.metadata {
		result[k] = v
	}
	return result
}

// GetPromptsByCategory returns prompts filtered by category
func (pm *EnhancedPromptManager) GetPromptsByCategory(category string) map[string]*PromptMetadata {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*PromptMetadata)
	for name, meta := range pm.metadata {
		if meta.Category == category {
			result[name] = meta
		}
	}
	return result
}

// GetPromptsByTag returns prompts that have a specific tag
func (pm *EnhancedPromptManager) GetPromptsByTag(tag string) map[string]*PromptMetadata {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*PromptMetadata)
	for name, meta := range pm.metadata {
		for _, t := range meta.Tags {
			if strings.EqualFold(t, tag) {
				result[name] = meta
				break
			}
		}
	}
	return result
}

// IsModelCompatible checks if a prompt is compatible with a given model
func (pm *EnhancedPromptManager) IsModelCompatible(promptName, modelName string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	meta, exists := pm.metadata[promptName]
	if !exists {
		// No metadata means assume compatibility
		return true
	}

	return meta.IsCompatibleWithModel(modelName)
}

// loadPromptsFromDir loads prompts from a specific directory
func loadPromptsFromDir(fsys fs.FS, dir string, templates map[string]*template.Template, metadata map[string]*PromptMetadata, prefix string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Read the file
		content, err := fs.ReadFile(fsys, filepath.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", entry.Name(), err)
		}

		// Parse prompt with metadata
		prompt, err := ParsePromptWithMetadata(string(content))
		if err != nil {
			return fmt.Errorf("error parsing prompt %s: %w", entry.Name(), err)
		}

		// Generate template name
		baseName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		templateName := prefix + "." + baseName
		prompt.Name = templateName

		// Create template
		tmpl, err := template.New(templateName).Parse(prompt.Content)
		if err != nil {
			return fmt.Errorf("error creating template %s: %w", templateName, err)
		}

		templates[templateName] = tmpl
		if prompt.Metadata != nil {
			metadata[templateName] = prompt.Metadata
		}
	}

	return nil
}

// processConditionalBlocks processes <if_block> conditions in the rendered content
func processConditionalBlocks(content string, data interface{}) string {
	// Simple implementation - in production, use a proper parser
	lines := strings.Split(content, "\n")
	var result []string
	inBlock := false
	blockCondition := ""
	blockContent := []string{}

	for _, line := range lines {
		if strings.Contains(line, "<if_block condition=") {
			inBlock = true
			// Extract condition
			start := strings.Index(line, `"`) + 1
			end := strings.LastIndex(line, `"`)
			if start > 0 && end > start {
				blockCondition = line[start:end]
			}
			continue
		}

		if strings.Contains(line, "</if_block>") {
			inBlock = false
			// Evaluate condition
			if shouldIncludeBlock(blockCondition, data) {
				result = append(result, blockContent...)
			}
			blockContent = []string{}
			blockCondition = ""
			continue
		}

		if inBlock {
			blockContent = append(blockContent, line)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// shouldIncludeBlock evaluates if a conditional block should be included
func shouldIncludeBlock(condition string, data interface{}) bool {
	if condition == "" {
		return false
	}

	// Simple implementation for "has_" conditions
	if strings.HasPrefix(condition, "has_") {
		varName := strings.TrimPrefix(condition, "has_")
		varName = toCamelCase(varName)
		
		if dataMap, ok := data.(map[string]interface{}); ok {
			val, exists := dataMap[varName]
			return exists && val != nil && val != ""
		}
	}

	return false
}

// toCamelCase converts snake_case to CamelCase
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}