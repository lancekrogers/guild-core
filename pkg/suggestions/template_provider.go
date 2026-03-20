// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions

import (
	"context"
	"fmt"
	"strings"

	"github.com/lancekrogers/guild-core/pkg/templates"
)

// TemplateSuggestionProvider suggests relevant templates based on context
type TemplateSuggestionProvider struct {
	templateManager templates.TemplateManager
}

// NewTemplateSuggestionProvider creates a new template suggestion provider
func NewTemplateSuggestionProvider(templateManager templates.TemplateManager) *TemplateSuggestionProvider {
	return &TemplateSuggestionProvider{
		templateManager: templateManager,
	}
}

// GetSuggestions returns template suggestions based on context
func (p *TemplateSuggestionProvider) GetSuggestions(ctx context.Context, context SuggestionContext) ([]Suggestion, error) {
	if p.templateManager == nil {
		return []Suggestion{}, nil
	}

	// Get contextual suggestions from template manager
	contextMap := map[string]interface{}{
		"ctx":             ctx,
		"current_message": context.CurrentMessage,
		"project_type":    context.ProjectContext.ProjectType,
		"language":        context.ProjectContext.Language,
		"recent_files":    context.ProjectContext.RecentFiles,
	}

	templates, err := p.templateManager.GetContextualSuggestions(contextMap)
	if err != nil {
		// Don't fail completely if template suggestions fail
		return []Suggestion{}, nil
	}

	suggestions := make([]Suggestion, 0)

	// Analyze context to determine which templates are most relevant
	currentMsg := strings.ToLower(context.CurrentMessage)
	recentContext := p.getRecentContext(context.ConversationHistory, 3)

	for _, template := range templates {
		confidence := p.calculateTemplateRelevance(template, currentMsg, recentContext, context)

		if confidence > 0.2 { // Lower threshold for templates
			suggestion := Suggestion{
				Type:        SuggestionTypeTemplate,
				Content:     template.Name,
				Display:     fmt.Sprintf("📝 %s", template.Name),
				Description: template.Description,
				Confidence:  confidence,
				Priority:    p.calculatePriority(template, context),
				Action: SuggestionAction{
					Type:   ActionTypeTemplate,
					Target: template.ID,
					Parameters: map[string]interface{}{
						"template_id": template.ID,
						"variables":   p.extractVariableDefaults(template),
					},
					Preview: p.generatePreview(template),
				},
				Tags: []string{template.Category},
				Metadata: map[string]interface{}{
					"category":    template.Category,
					"use_count":   template.UseCount,
					"is_built_in": template.IsBuiltIn,
				},
			}

			suggestions = append(suggestions, suggestion)
		}
	}

	// Also search for templates matching the current context
	if currentMsg != "" {
		searchResults, err := p.templateManager.SearchTemplates(currentMsg, 5)
		if err == nil {
			for _, result := range searchResults {
				// Check if we already have this template
				exists := false
				for _, s := range suggestions {
					if s.Action.Target == result.Template.ID {
						exists = true
						break
					}
				}

				if !exists && result.Relevance > 0.3 {
					suggestion := Suggestion{
						Type:        SuggestionTypeTemplate,
						Content:     result.Template.Name,
						Display:     fmt.Sprintf("🔍 %s", result.Template.Name),
						Description: result.Template.Description,
						Confidence:  result.Relevance,
						Priority:    3, // Lower priority for search results
						Action: SuggestionAction{
							Type:   ActionTypeTemplate,
							Target: result.Template.ID,
							Parameters: map[string]interface{}{
								"template_id": result.Template.ID,
								"variables":   p.extractVariableDefaults(result.Template),
							},
							Preview: p.generatePreview(result.Template),
						},
						Tags: append([]string{result.Template.Category}, result.Matches...),
						Metadata: map[string]interface{}{
							"search_result": true,
							"matches":       result.Matches,
						},
					}

					suggestions = append(suggestions, suggestion)
				}
			}
		}
	}

	return suggestions, nil
}

// UpdateContext updates the provider's context (no-op for stateless provider)
func (p *TemplateSuggestionProvider) UpdateContext(ctx context.Context, context SuggestionContext) error {
	// This provider is stateless
	return nil
}

// SupportedTypes returns the suggestion types this provider handles
func (p *TemplateSuggestionProvider) SupportedTypes() []SuggestionType {
	return []SuggestionType{SuggestionTypeTemplate}
}

// GetMetadata returns provider metadata
func (p *TemplateSuggestionProvider) GetMetadata() ProviderMetadata {
	return ProviderMetadata{
		Name:        "TemplateSuggestionProvider",
		Version:     "1.0.0",
		Description: "Suggests relevant templates based on conversation context",
		Capabilities: []string{
			"template_search",
			"context_matching",
			"usage_tracking",
		},
	}
}

// calculateTemplateRelevance calculates how relevant a template is to the current context
func (p *TemplateSuggestionProvider) calculateTemplateRelevance(template *templates.Template, currentMsg string, recentContext string, context SuggestionContext) float64 {
	relevance := 0.0

	// Check template name and description
	nameLower := strings.ToLower(template.Name)
	descLower := strings.ToLower(template.Description)

	// Direct name match
	if strings.Contains(currentMsg, nameLower) {
		relevance = 0.9
	} else if strings.Contains(recentContext, nameLower) {
		relevance = 0.6
	}

	// Description keyword matching
	descWords := strings.Fields(descLower)
	msgWords := strings.Fields(currentMsg)
	contextWords := strings.Fields(recentContext)

	matchCount := 0
	for _, word := range descWords {
		if len(word) < 4 { // Skip short words
			continue
		}
		for _, msgWord := range msgWords {
			if word == msgWord {
				matchCount++
			}
		}
		for _, ctxWord := range contextWords {
			if word == ctxWord {
				matchCount++
			}
		}
	}

	if matchCount > 0 {
		wordRelevance := float64(matchCount) / float64(len(descWords))
		relevance = maxFloat(relevance, wordRelevance*0.7)
	}

	// Category matching for project context
	if p.isCategoryRelevant(template.Category, context) {
		relevance *= 1.3
		if relevance > 1.0 {
			relevance = 1.0
		}
	}

	// Boost based on usage frequency
	if template.UseCount > 10 {
		relevance *= 1.1
	} else if template.UseCount > 5 {
		relevance *= 1.05
	}

	// Cap at 1.0
	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}

// calculatePriority calculates the priority of a template suggestion
func (p *TemplateSuggestionProvider) calculatePriority(template *templates.Template, context SuggestionContext) int {
	priority := 5 // Base priority

	// Boost for frequently used templates
	if template.UseCount > 20 {
		priority += 3
	} else if template.UseCount > 10 {
		priority += 2
	} else if template.UseCount > 5 {
		priority += 1
	}

	// Boost for favorites (if in user preferences)
	if context.UserPreferences.FavoriteTemplates != nil {
		for _, fav := range context.UserPreferences.FavoriteTemplates {
			if fav == template.ID || fav == template.Name {
				priority += 5
				break
			}
		}
	}

	// Boost for category match
	if p.isCategoryRelevant(template.Category, context) {
		priority += 2
	}

	return priority
}

// isCategoryRelevant checks if a template category is relevant to the current context
func (p *TemplateSuggestionProvider) isCategoryRelevant(category string, context SuggestionContext) bool {
	categoryLower := strings.ToLower(category)

	// Check against project type
	if context.ProjectContext.ProjectType != "" {
		projectType := strings.ToLower(context.ProjectContext.ProjectType)
		if strings.Contains(categoryLower, projectType) || strings.Contains(projectType, categoryLower) {
			return true
		}
	}

	// Check against language
	if context.ProjectContext.Language != "" {
		language := strings.ToLower(context.ProjectContext.Language)
		if strings.Contains(categoryLower, language) {
			return true
		}
	}

	// Category-specific relevance
	switch categoryLower {
	case "code review":
		return p.hasKeywords(context, []string{"review", "check", "analyze", "quality"})
	case "debugging":
		return p.hasKeywords(context, []string{"bug", "error", "issue", "problem", "debug"})
	case "documentation":
		return p.hasKeywords(context, []string{"document", "readme", "docs", "explain"})
	case "testing":
		return p.hasKeywords(context, []string{"test", "unit", "integration", "coverage"})
	}

	return false
}

// hasKeywords checks if the context contains any of the given keywords
func (p *TemplateSuggestionProvider) hasKeywords(context SuggestionContext, keywords []string) bool {
	combined := strings.ToLower(context.CurrentMessage)
	if len(context.ConversationHistory) > 0 {
		recent := context.ConversationHistory[len(context.ConversationHistory)-1]
		combined += " " + strings.ToLower(recent.Content)
	}

	for _, keyword := range keywords {
		if strings.Contains(combined, keyword) {
			return true
		}
	}

	return false
}

// extractVariableDefaults extracts default values from template variables
func (p *TemplateSuggestionProvider) extractVariableDefaults(template *templates.Template) map[string]interface{} {
	defaults := make(map[string]interface{})

	for _, variable := range template.Variables {
		if variable.DefaultValue != "" {
			defaults[variable.Name] = variable.DefaultValue
		}
	}

	return defaults
}

// generatePreview generates a preview of the template
func (p *TemplateSuggestionProvider) generatePreview(template *templates.Template) string {
	preview := template.Content

	// Truncate if too long
	if len(preview) > 100 {
		preview = preview[:97] + "..."
	}

	// Replace newlines with spaces for single-line preview
	preview = strings.ReplaceAll(preview, "\n", " ")

	return preview
}

// getRecentContext extracts recent conversation context
func (p *TemplateSuggestionProvider) getRecentContext(history []ChatMessage, limit int) string {
	if len(history) == 0 {
		return ""
	}

	start := len(history) - limit
	if start < 0 {
		start = 0
	}

	var context strings.Builder
	for i := start; i < len(history); i++ {
		context.WriteString(strings.ToLower(history[i].Content))
		context.WriteString(" ")
	}

	return context.String()
}
