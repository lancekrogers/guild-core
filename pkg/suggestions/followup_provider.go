// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package suggestions

import (
	"context"
	"fmt"
	"strings"
)

// FollowUpSuggestionProvider suggests follow-up actions based on conversation flow
type FollowUpSuggestionProvider struct {
	patterns []FollowUpPattern
}

// FollowUpPattern defines a pattern for follow-up suggestions
type FollowUpPattern struct {
	TriggerRole    string   // Role that triggers this pattern (user/assistant)
	TriggerKeywords []string // Keywords in the trigger message
	Suggestions    []FollowUpSuggestion
}

// FollowUpSuggestion defines a specific follow-up suggestion
type FollowUpSuggestion struct {
	Text        string
	Description string
	Priority    int
	Confidence  float64
	Tags        []string
}

// NewFollowUpSuggestionProvider creates a new follow-up suggestion provider
func NewFollowUpSuggestionProvider() *FollowUpSuggestionProvider {
	return &FollowUpSuggestionProvider{
		patterns: getDefaultFollowUpPatterns(),
	}
}

// GetSuggestions returns follow-up suggestions based on conversation flow
func (p *FollowUpSuggestionProvider) GetSuggestions(ctx context.Context, context SuggestionContext) ([]Suggestion, error) {
	suggestions := make([]Suggestion, 0)
	
	// Need conversation history for follow-ups
	if len(context.ConversationHistory) == 0 {
		return suggestions, nil
	}
	
	// Get the last message
	lastMessage := context.ConversationHistory[len(context.ConversationHistory)-1]
	
	// Check patterns based on last message role and content
	for _, pattern := range p.patterns {
		if pattern.TriggerRole != "" && pattern.TriggerRole != lastMessage.Role {
			continue
		}
		
		// Check if keywords match
		if p.matchesKeywords(lastMessage.Content, pattern.TriggerKeywords) {
			// Add all suggestions from this pattern
			for _, followUp := range pattern.Suggestions {
				// Adjust confidence based on context
				confidence := p.adjustConfidence(followUp.Confidence, context)
				
				if confidence > 0.3 {
					suggestion := Suggestion{
						Type:        SuggestionTypeFollowUp,
						Content:     followUp.Text,
						Display:     fmt.Sprintf("➡️ %s", followUp.Text),
						Description: followUp.Description,
						Confidence:  confidence,
						Priority:    followUp.Priority,
						Action: SuggestionAction{
							Type:   ActionTypeInsert,
							Target: followUp.Text,
						},
						Tags: followUp.Tags,
						Metadata: map[string]interface{}{
							"pattern_type": pattern.TriggerRole,
						},
					}
					
					suggestions = append(suggestions, suggestion)
				}
			}
		}
	}
	
	// Add dynamic suggestions based on conversation analysis
	dynamicSuggestions := p.generateDynamicSuggestions(context)
	suggestions = append(suggestions, dynamicSuggestions...)
	
	return suggestions, nil
}

// UpdateContext updates the provider's context (no-op for stateless provider)
func (p *FollowUpSuggestionProvider) UpdateContext(ctx context.Context, context SuggestionContext) error {
	// This provider is stateless
	return nil
}

// SupportedTypes returns the suggestion types this provider handles
func (p *FollowUpSuggestionProvider) SupportedTypes() []SuggestionType {
	return []SuggestionType{SuggestionTypeFollowUp}
}

// GetMetadata returns provider metadata
func (p *FollowUpSuggestionProvider) GetMetadata() ProviderMetadata {
	return ProviderMetadata{
		Name:        "FollowUpSuggestionProvider",
		Version:     "1.0.0",
		Description: "Suggests logical follow-up actions based on conversation flow",
		Capabilities: []string{
			"conversation_analysis",
			"pattern_matching",
			"dynamic_generation",
		},
	}
}

// matchesKeywords checks if content contains any of the keywords
func (p *FollowUpSuggestionProvider) matchesKeywords(content string, keywords []string) bool {
	contentLower := strings.ToLower(content)
	
	for _, keyword := range keywords {
		if strings.Contains(contentLower, strings.ToLower(keyword)) {
			return true
		}
	}
	
	return false
}

// adjustConfidence adjusts confidence based on additional context
func (p *FollowUpSuggestionProvider) adjustConfidence(baseConfidence float64, context SuggestionContext) float64 {
	confidence := baseConfidence
	
	// Boost confidence if user preferences indicate they like follow-ups
	if context.UserPreferences.SuggestionFrequency == "always" {
		confidence *= 1.2
	} else if context.UserPreferences.SuggestionFrequency == "minimal" {
		confidence *= 0.8
	}
	
	// Adjust based on conversation length
	if len(context.ConversationHistory) > 10 {
		// In longer conversations, be more selective
		confidence *= 0.9
	}
	
	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}

// generateDynamicSuggestions generates context-specific follow-up suggestions
func (p *FollowUpSuggestionProvider) generateDynamicSuggestions(context SuggestionContext) []Suggestion {
	suggestions := make([]Suggestion, 0)
	
	if len(context.ConversationHistory) < 2 {
		return suggestions
	}
	
	lastMessage := context.ConversationHistory[len(context.ConversationHistory)-1]
	
	// If assistant just provided code, suggest testing or running it
	if lastMessage.Role == "assistant" && p.containsCode(lastMessage.Content) {
		suggestions = append(suggestions, Suggestion{
			Type:        SuggestionTypeFollowUp,
			Content:     "Can you help me test this code?",
			Display:     "➡️ Can you help me test this code?",
			Description: "Ask for help with testing the provided code",
			Confidence:  0.7,
			Priority:    6,
			Action: SuggestionAction{
				Type:   ActionTypeInsert,
				Target: "Can you help me test this code?",
			},
			Tags: []string{"testing", "code"},
		})
		
		suggestions = append(suggestions, Suggestion{
			Type:        SuggestionTypeFollowUp,
			Content:     "How do I run this?",
			Display:     "➡️ How do I run this?",
			Description: "Ask for instructions on running the code",
			Confidence:  0.6,
			Priority:    5,
			Action: SuggestionAction{
				Type:   ActionTypeInsert,
				Target: "How do I run this?",
			},
			Tags: []string{"execution", "code"},
		})
	}
	
	// If discussing an error, suggest debugging steps
	if p.discussingError(context) {
		suggestions = append(suggestions, Suggestion{
			Type:        SuggestionTypeFollowUp,
			Content:     "Can you help me debug this error?",
			Display:     "➡️ Can you help me debug this error?",
			Description: "Request debugging assistance",
			Confidence:  0.8,
			Priority:    7,
			Action: SuggestionAction{
				Type:   ActionTypeInsert,
				Target: "Can you help me debug this error?",
			},
			Tags: []string{"debugging", "error"},
		})
	}
	
	// If discussing implementation, suggest next steps
	if p.discussingImplementation(context) {
		suggestions = append(suggestions, Suggestion{
			Type:        SuggestionTypeFollowUp,
			Content:     "What should I implement next?",
			Display:     "➡️ What should I implement next?",
			Description: "Ask about next implementation steps",
			Confidence:  0.6,
			Priority:    5,
			Action: SuggestionAction{
				Type:   ActionTypeInsert,
				Target: "What should I implement next?",
			},
			Tags: []string{"planning", "implementation"},
		})
	}
	
	return suggestions
}

// containsCode checks if the message contains code
func (p *FollowUpSuggestionProvider) containsCode(content string) bool {
	codeIndicators := []string{
		"```",
		"func ",
		"def ",
		"class ",
		"import ",
		"package ",
		"return ",
		"{",
		"}",
		"();",
		"const ",
		"let ",
		"var ",
	}
	
	for _, indicator := range codeIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	
	return false
}

// discussingError checks if the conversation is about an error
func (p *FollowUpSuggestionProvider) discussingError(context SuggestionContext) bool {
	errorKeywords := []string{"error", "exception", "fail", "crash", "bug", "issue", "problem"}
	
	// Check last few messages
	checkCount := 3
	if len(context.ConversationHistory) < checkCount {
		checkCount = len(context.ConversationHistory)
	}
	
	for i := len(context.ConversationHistory) - checkCount; i < len(context.ConversationHistory); i++ {
		msg := context.ConversationHistory[i]
		for _, keyword := range errorKeywords {
			if strings.Contains(strings.ToLower(msg.Content), keyword) {
				return true
			}
		}
	}
	
	return false
}

// discussingImplementation checks if the conversation is about implementation
func (p *FollowUpSuggestionProvider) discussingImplementation(context SuggestionContext) bool {
	implKeywords := []string{"implement", "build", "create", "develop", "feature", "function", "method"}
	
	// Check last few messages
	checkCount := 2
	if len(context.ConversationHistory) < checkCount {
		checkCount = len(context.ConversationHistory)
	}
	
	for i := len(context.ConversationHistory) - checkCount; i < len(context.ConversationHistory); i++ {
		msg := context.ConversationHistory[i]
		for _, keyword := range implKeywords {
			if strings.Contains(strings.ToLower(msg.Content), keyword) {
				return true
			}
		}
	}
	
	return false
}

// getDefaultFollowUpPatterns returns default follow-up patterns
func getDefaultFollowUpPatterns() []FollowUpPattern {
	return []FollowUpPattern{
		// After assistant provides solution
		{
			TriggerRole:     "assistant",
			TriggerKeywords: []string{"here's how", "you can", "try this", "solution"},
			Suggestions: []FollowUpSuggestion{
				{
					Text:        "Can you explain this in more detail?",
					Description: "Request more detailed explanation",
					Priority:    5,
					Confidence:  0.6,
					Tags:        []string{"clarification"},
				},
				{
					Text:        "Are there any alternatives to this approach?",
					Description: "Explore alternative solutions",
					Priority:    4,
					Confidence:  0.5,
					Tags:        []string{"alternatives"},
				},
			},
		},
		// After user asks question
		{
			TriggerRole:     "user",
			TriggerKeywords: []string{"how do i", "can you", "what is", "why"},
			Suggestions: []FollowUpSuggestion{
				{
					Text:        "Can you provide an example?",
					Description: "Request a concrete example",
					Priority:    6,
					Confidence:  0.7,
					Tags:        []string{"example"},
				},
			},
		},
		// After error discussion
		{
			TriggerRole:     "",
			TriggerKeywords: []string{"error", "exception", "failed"},
			Suggestions: []FollowUpSuggestion{
				{
					Text:        "What's the full error message?",
					Description: "Request complete error details",
					Priority:    8,
					Confidence:  0.8,
					Tags:        []string{"debugging", "error"},
				},
				{
					Text:        "What have I tried so far?",
					Description: "Review attempted solutions",
					Priority:    6,
					Confidence:  0.6,
					Tags:        []string{"debugging", "review"},
				},
			},
		},
		// After implementation discussion
		{
			TriggerRole:     "",
			TriggerKeywords: []string{"implement", "build", "create"},
			Suggestions: []FollowUpSuggestion{
				{
					Text:        "What are the requirements for this?",
					Description: "Clarify implementation requirements",
					Priority:    7,
					Confidence:  0.7,
					Tags:        []string{"requirements", "planning"},
				},
				{
					Text:        "Can you help me break this down into steps?",
					Description: "Request step-by-step breakdown",
					Priority:    6,
					Confidence:  0.6,
					Tags:        []string{"planning", "breakdown"},
				},
			},
		},
	}
}