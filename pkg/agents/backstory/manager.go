// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package backstory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/prompts/layered"
)

// BackstoryManager manages agent personalities and integrates them with the layered prompt system
type BackstoryManager struct {
	agents         map[string]*EnhancedAgent
	promptRegistry layered.LayeredRegistry
}

// EnhancedAgent represents an agent with full personality and context
type EnhancedAgent struct {
	Config   *config.AgentConfig
	Context  *AgentContext
	Memory   *AgentMemory
	LastSeen time.Time
}

// AgentContext holds current situational information for an agent
type AgentContext struct {
	CurrentTask     string
	CurrentProject  string
	TeamMembers     []string
	ProjectContext  map[string]interface{}
	RecentDecisions []Decision
	WorkingState    string // "focused", "collaborating", "teaching", "debugging"
	Mood            string // "determined", "patient", "excited", "concerned"
}

// AgentMemory stores learned patterns and preferences
type AgentMemory struct {
	Interactions         []Interaction
	LearnedPatterns      map[string]Pattern
	Preferences          map[string]interface{}
	SuccessfulApproaches map[string]int // Track what works
}

// Decision represents a decision made by the agent
type Decision struct {
	Type      string    `json:"type"`
	Summary   string    `json:"summary"`
	Reasoning string    `json:"reasoning"`
	Outcome   string    `json:"outcome"`
	Timestamp time.Time `json:"timestamp"`
}

// Interaction represents a past interaction
type Interaction struct {
	Type       string    `json:"type"`
	Summary    string    `json:"summary"`
	Outcome    string    `json:"outcome"`
	UserRating int       `json:"user_rating"` // 1-10
	Timestamp  time.Time `json:"timestamp"`
}

// Pattern represents a learned behavior pattern
type Pattern struct {
	Description string    `json:"description"`
	Confidence  float64   `json:"confidence"`
	UseCount    int       `json:"use_count"`
	LastUsed    time.Time `json:"last_used"`
}

// NewBackstoryManager creates a new backstory manager
func NewBackstoryManager(promptRegistry layered.LayeredRegistry) *BackstoryManager {
	return &BackstoryManager{
		agents:         make(map[string]*EnhancedAgent),
		promptRegistry: promptRegistry,
	}
}

// RegisterAgent registers an agent with the backstory system
func (m *BackstoryManager) RegisterAgent(agentConfig *config.AgentConfig) error {
	agent := &EnhancedAgent{
		Config: agentConfig,
		Context: &AgentContext{
			ProjectContext:  make(map[string]interface{}),
			RecentDecisions: make([]Decision, 0),
			WorkingState:    "ready",
			Mood:            m.determineInitialMood(agentConfig),
		},
		Memory: &AgentMemory{
			Interactions:         make([]Interaction, 0),
			LearnedPatterns:      make(map[string]Pattern),
			Preferences:          make(map[string]interface{}),
			SuccessfulApproaches: make(map[string]int),
		},
		LastSeen: time.Now(),
	}

	m.agents[agentConfig.ID] = agent

	// Register personality layers in the prompt system
	if err := m.registerPersonalityLayers(agent); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register personality layers").
			WithComponent("BackstoryManager").
			WithOperation("RegisterAgent")
	}

	return nil
}

// BuildPersonalityPrompt builds a personality-enhanced prompt for an agent
func (m *BackstoryManager) BuildPersonalityPrompt(
	ctx context.Context,
	agentID string,
	basePrompt string,
	turnContext *layered.TurnContext,
) (string, error) {
	agent, exists := m.agents[agentID]
	if !exists {
		// Return base prompt if no personality configured
		return basePrompt, nil
	}

	agent.LastSeen = time.Now()

	// Update agent context based on current situation
	m.updateAgentContext(agent, turnContext)

	// Build personality layers
	var promptParts []string

	// 1. Identity and Background Layer
	if identityLayer := m.buildIdentityLayer(agent); identityLayer != "" {
		promptParts = append(promptParts, identityLayer)
	}

	// 2. Expertise and Specialization Layer
	if expertiseLayer := m.buildExpertiseLayer(agent); expertiseLayer != "" {
		promptParts = append(promptParts, expertiseLayer)
	}

	// 3. Current Context and Mood Layer
	if contextLayer := m.buildContextLayer(agent); contextLayer != "" {
		promptParts = append(promptParts, contextLayer)
	}

	// 4. Memory and Experience Layer
	if memoryLayer := m.buildMemoryLayer(agent, turnContext); memoryLayer != "" {
		promptParts = append(promptParts, memoryLayer)
	}

	// 5. Communication Style Layer
	if styleLayer := m.buildCommunicationStyleLayer(agent); styleLayer != "" {
		promptParts = append(promptParts, styleLayer)
	}

	// Combine all layers with base prompt
	if len(promptParts) > 0 {
		promptParts = append(promptParts, basePrompt)
		return strings.Join(promptParts, "\n\n"), nil
	}

	return basePrompt, nil
}

// buildIdentityLayer creates the identity and background portion of the prompt
func (m *BackstoryManager) buildIdentityLayer(agent *EnhancedAgent) string {
	backstory := agent.Config.Backstory
	if backstory == nil {
		return ""
	}

	// Check if we have any meaningful content
	hasContent := backstory.GuildRank != "" ||
		backstory.Experience != "" ||
		len(backstory.PreviousRoles) > 0 ||
		backstory.Expertise != "" ||
		backstory.Philosophy != "" ||
		len(backstory.Specialties) > 0

	if !hasContent {
		return ""
	}

	var layer strings.Builder

	layer.WriteString("# Your Identity and Background\n\n")

	// Medieval guild identity
	if backstory.GuildRank != "" {
		layer.WriteString(fmt.Sprintf("You are %s, a %s in the Guild of Digital Artisans.\n",
			agent.Config.Name, backstory.GuildRank))
	} else {
		layer.WriteString(fmt.Sprintf("You are %s, a skilled artisan in the Guild of Digital Artisans.\n",
			agent.Config.Name))
	}

	// Professional background
	if backstory.Experience != "" {
		layer.WriteString(fmt.Sprintf("Your experience: %s\n", backstory.Experience))
	}

	if len(backstory.PreviousRoles) > 0 {
		layer.WriteString(fmt.Sprintf("Your previous positions: %s\n",
			strings.Join(backstory.PreviousRoles, ", ")))
	}

	// Core expertise
	if backstory.Expertise != "" {
		layer.WriteString(fmt.Sprintf("\n## Your Mastery\n%s\n", backstory.Expertise))
	}

	// Philosophy
	if backstory.Philosophy != "" {
		layer.WriteString(fmt.Sprintf("\n## Your Philosophy\n%s\n", backstory.Philosophy))
	}

	// Guild specialties
	if len(backstory.Specialties) > 0 {
		layer.WriteString(fmt.Sprintf("\n## Your Guild Specialties\n%s\n",
			strings.Join(backstory.Specialties, ", ")))
	}

	return layer.String()
}

// buildExpertiseLayer creates the expertise and technical focus portion
func (m *BackstoryManager) buildExpertiseLayer(agent *EnhancedAgent) string {
	spec := agent.Config.Specialization
	if spec == nil {
		return ""
	}

	// Check if we have any meaningful content
	hasContent := spec.Domain != "" ||
		spec.ExpertiseLevel != "" ||
		spec.Craft != "" ||
		len(spec.CoreKnowledge) > 0 ||
		len(spec.Technologies) > 0 ||
		len(spec.Methodologies) > 0 ||
		len(spec.Principles) > 0

	if !hasContent {
		return ""
	}

	var layer strings.Builder

	layer.WriteString("# Your Craft and Expertise\n\n")

	// Domain specialization
	if spec.Domain != "" && spec.ExpertiseLevel != "" {
		layer.WriteString(fmt.Sprintf("You are a %s-level practitioner in the craft of %s.\n",
			spec.ExpertiseLevel, spec.Domain))
	} else if spec.Domain != "" {
		layer.WriteString(fmt.Sprintf("You specialize in the craft of %s.\n", spec.Domain))
	}

	if spec.Craft != "" {
		layer.WriteString(fmt.Sprintf("Your primary craft: %s\n", spec.Craft))
	}

	// Core knowledge areas
	if len(spec.CoreKnowledge) > 0 {
		layer.WriteString("\n## Deep Mastery:\n")
		for _, knowledge := range spec.CoreKnowledge {
			layer.WriteString(fmt.Sprintf("- %s\n", knowledge))
		}
	}

	// Technical preferences
	if len(spec.Technologies) > 0 {
		layer.WriteString(fmt.Sprintf("\n## Preferred Tools: %s\n",
			strings.Join(spec.Technologies, ", ")))
	}

	if len(spec.Methodologies) > 0 {
		layer.WriteString(fmt.Sprintf("## Favored Methods: %s\n",
			strings.Join(spec.Methodologies, ", ")))
	}

	if len(spec.Principles) > 0 {
		layer.WriteString(fmt.Sprintf("## Guiding Principles: %s\n",
			strings.Join(spec.Principles, ", ")))
	}

	return layer.String()
}

// buildContextLayer creates the current context and mood portion
func (m *BackstoryManager) buildContextLayer(agent *EnhancedAgent) string {
	var layer strings.Builder

	layer.WriteString("# Current Context\n\n")

	// Working state and mood
	layer.WriteString(fmt.Sprintf("Current state: %s\n", agent.Context.WorkingState))
	layer.WriteString(fmt.Sprintf("Current mood: %s\n", agent.Context.Mood))

	// Current task
	if agent.Context.CurrentTask != "" {
		layer.WriteString(fmt.Sprintf("Current task: %s\n", agent.Context.CurrentTask))
	}

	// Project context
	if agent.Context.CurrentProject != "" {
		layer.WriteString(fmt.Sprintf("Current project: %s\n", agent.Context.CurrentProject))
	}

	// Team collaboration
	if len(agent.Context.TeamMembers) > 0 {
		layer.WriteString(fmt.Sprintf("Working alongside: %s\n",
			strings.Join(agent.Context.TeamMembers, ", ")))
	}

	// Recent decisions
	if len(agent.Context.RecentDecisions) > 0 {
		layer.WriteString("\n## Recent Decisions:\n")
		// Show last 3 decisions
		count := len(agent.Context.RecentDecisions)
		start := 0
		if count > 3 {
			start = count - 3
		}
		for i := start; i < count; i++ {
			decision := agent.Context.RecentDecisions[i]
			layer.WriteString(fmt.Sprintf("- %s: %s\n",
				decision.Type, decision.Summary))
		}
	}

	return layer.String()
}

// buildMemoryLayer creates the relevant history and learned patterns portion
func (m *BackstoryManager) buildMemoryLayer(agent *EnhancedAgent, turnContext *layered.TurnContext) string {
	memory := agent.Memory
	if memory == nil {
		return ""
	}

	var layer strings.Builder

	// Find relevant interactions
	relevantInteractions := m.findRelevantInteractions(memory, turnContext, 3)
	if len(relevantInteractions) > 0 {
		layer.WriteString("# Relevant Experience\n\n")
		for _, interaction := range relevantInteractions {
			layer.WriteString(fmt.Sprintf("- %s: %s (outcome: %s)\n",
				interaction.Type, interaction.Summary, interaction.Outcome))
		}
	}

	// Learned patterns with high confidence
	highConfidencePatterns := make([]Pattern, 0)
	for _, pattern := range memory.LearnedPatterns {
		if pattern.Confidence > 0.7 {
			highConfidencePatterns = append(highConfidencePatterns, pattern)
		}
	}

	if len(highConfidencePatterns) > 0 {
		if layer.Len() == 0 {
			layer.WriteString("# Learned Wisdom\n\n")
		} else {
			layer.WriteString("\n## Learned Wisdom:\n")
		}
		for _, pattern := range highConfidencePatterns {
			layer.WriteString(fmt.Sprintf("- %s\n", pattern.Description))
		}
	}

	return layer.String()
}

// buildCommunicationStyleLayer creates the communication style portion
func (m *BackstoryManager) buildCommunicationStyleLayer(agent *EnhancedAgent) string {
	backstory := agent.Config.Backstory
	personality := agent.Config.Personality

	if backstory == nil && personality == nil {
		return ""
	}

	var layer strings.Builder
	hasContent := false

	// Communication preferences from backstory
	if backstory != nil {
		if backstory.CommunicationStyle != "" {
			if !hasContent {
				layer.WriteString("# Your Communication Style\n\n")
				hasContent = true
			}
			layer.WriteString(fmt.Sprintf("Communication approach: %s\n", backstory.CommunicationStyle))
		}
		if backstory.TeachingStyle != "" {
			if !hasContent {
				layer.WriteString("# Your Communication Style\n\n")
				hasContent = true
			}
			layer.WriteString(fmt.Sprintf("Teaching style: %s\n", backstory.TeachingStyle))
		}
	}

	// Personality-driven communication
	if personality != nil {
		if personality.Formality != "" {
			if !hasContent {
				layer.WriteString("# Your Communication Style\n\n")
				hasContent = true
			}
			layer.WriteString(fmt.Sprintf("Formality: %s\n", personality.Formality))
		}
		if personality.DetailLevel != "" {
			layer.WriteString(fmt.Sprintf("Detail level: %s\n", personality.DetailLevel))
		}

		if personality.HumorLevel != "" && personality.HumorLevel != "none" {
			layer.WriteString(fmt.Sprintf("Humor: %s\n", personality.HumorLevel))
		}

		// Personality traits
		if len(personality.Traits) > 0 {
			layer.WriteString("\n## Key Traits:\n")
			for _, trait := range personality.Traits {
				if trait.Strength > 0.7 {
					layer.WriteString(fmt.Sprintf("- **%s** (%.0f%%): %s\n",
						trait.Name, trait.Strength*100, trait.Description))
				}
			}
		}

		// Medieval traits
		if personality.Honor > 7 {
			layer.WriteString("- Highly honorable - keeps promises and maintains integrity\n")
		}
		if personality.Wisdom > 7 {
			layer.WriteString("- Wise - provides thoughtful counsel and learns from experience\n")
		}
		if personality.Craftsmanship > 7 {
			layer.WriteString("- Master craftsman - takes pride in quality and attention to detail\n")
		}
	}

	// Return empty string if no content was added
	if !hasContent {
		return ""
	}

	return layer.String()
}

// updateAgentContext updates the agent's context based on current situation
func (m *BackstoryManager) updateAgentContext(agent *EnhancedAgent, turnContext *layered.TurnContext) {
	if turnContext == nil {
		return
	}

	// Update current task
	if turnContext.TaskID != "" {
		agent.Context.CurrentTask = turnContext.TaskID
	}

	// Update working state based on urgency and context
	if turnContext.Urgency == "high" {
		agent.Context.WorkingState = "focused"
		agent.Context.Mood = "determined"
	} else if len(agent.Context.TeamMembers) > 0 {
		agent.Context.WorkingState = "collaborating"
		agent.Context.Mood = "engaged"
	} else {
		agent.Context.WorkingState = "ready"
		agent.Context.Mood = m.determineContextualMood(agent, turnContext)
	}
}

// determineInitialMood determines the initial mood based on personality
func (m *BackstoryManager) determineInitialMood(agentConfig *config.AgentConfig) string {
	if agentConfig.Personality == nil {
		return "ready"
	}

	personality := agentConfig.Personality

	// Base mood on personality traits
	if personality.Patience > 8 {
		return "patient"
	}
	if personality.Assertiveness > 8 {
		return "confident"
	}
	if personality.Empathy > 8 {
		return "caring"
	}

	// Check for dominant traits (handle nil traits gracefully)
	if personality.Traits != nil {
		for _, trait := range personality.Traits {
			if trait.Strength > 0.8 {
				switch strings.ToLower(trait.Name) {
				case "excited", "enthusiastic":
					return "excited"
				case "analytical", "methodical":
					return "focused"
				case "creative":
					return "inspired"
				case "protective", "caring":
					return "protective"
				}
			}
		}
	}

	return "ready"
}

// determineContextualMood determines mood based on current context
func (m *BackstoryManager) determineContextualMood(agent *EnhancedAgent, turnContext *layered.TurnContext) string {
	// Check for error or problem keywords
	if turnContext.UserMessage != "" {
		message := strings.ToLower(turnContext.UserMessage)
		if strings.Contains(message, "error") || strings.Contains(message, "broken") ||
			strings.Contains(message, "problem") || strings.Contains(message, "failing") {
			return "concerned"
		}
		if strings.Contains(message, "help") || strings.Contains(message, "how") ||
			strings.Contains(message, "explain") {
			return "helpful"
		}
		if strings.Contains(message, "new") || strings.Contains(message, "create") ||
			strings.Contains(message, "build") {
			return "excited"
		}
	}

	return agent.Context.Mood // Keep current mood
}

// findRelevantInteractions finds interactions relevant to the current context
func (m *BackstoryManager) findRelevantInteractions(memory *AgentMemory, turnContext *layered.TurnContext, limit int) []Interaction {
	if turnContext == nil || len(memory.Interactions) == 0 {
		return nil
	}

	// Simple relevance scoring based on keywords
	type scoredInteraction struct {
		interaction Interaction
		score       int
	}

	var scored []scoredInteraction
	message := strings.ToLower(turnContext.UserMessage)

	for _, interaction := range memory.Interactions {
		score := 0
		interactionText := strings.ToLower(interaction.Summary)

		// Check for keyword matches
		words := strings.Fields(message)
		for _, word := range words {
			if len(word) > 3 && strings.Contains(interactionText, word) {
				score++
			}
		}

		// Boost recent interactions
		if time.Since(interaction.Timestamp) < 24*time.Hour {
			score += 2
		}

		// Boost successful interactions
		if interaction.UserRating > 7 {
			score++
		}

		if score > 0 {
			scored = append(scored, scoredInteraction{interaction, score})
		}
	}

	// Sort by score and return top results
	if len(scored) == 0 {
		return nil
	}

	// Simple sort by score (descending)
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Return top interactions
	result := make([]Interaction, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].interaction)
	}

	return result
}

// registerPersonalityLayers registers personality-specific prompt layers
func (m *BackstoryManager) registerPersonalityLayers(agent *EnhancedAgent) error {
	// Skip if no backstory or personality is configured
	if agent.Config.Backstory == nil && agent.Config.Personality == nil {
		return nil
	}

	// Build role personality prompt
	rolePrompt := m.buildRolePersonalityPrompt(agent)

	// Only register if we have actual content
	if rolePrompt == "" {
		return nil
	}

	prompt := layered.SystemPrompt{
		Layer:     layered.LayerRole,
		ArtisanID: agent.Config.ID,
		Content:   rolePrompt,
		Version:   1,
		Priority:  100,
		Updated:   time.Now(),
		Metadata: map[string]interface{}{
			"backstory_enabled":   true,
			"personality_version": 1,
		},
	}

	if err := m.promptRegistry.RegisterLayeredPrompt(layered.LayerRole, agent.Config.ID, prompt); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register personality prompt layer").
			WithComponent("BackstoryManager").
			WithOperation("registerPersonalityLayers").
			WithDetails("agent_id", agent.Config.ID)
	}

	return nil
}

// buildRolePersonalityPrompt builds a role-specific prompt with personality
func (m *BackstoryManager) buildRolePersonalityPrompt(agent *EnhancedAgent) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("You are %s, acting in the role of %s.\n\n",
		agent.Config.Name, agent.Config.Type))

	// Add medieval guild context
	if agent.Config.Backstory != nil && agent.Config.Backstory.GuildRank != "" {
		prompt.WriteString(fmt.Sprintf("Your rank in the Guild: %s\n", agent.Config.Backstory.GuildRank))
	}

	// Add personality-driven role interpretation
	if agent.Config.Personality != nil {
		prompt.WriteString("\nYour approach to this role:\n")

		if agent.Config.Personality.ApproachStyle != "" {
			prompt.WriteString(fmt.Sprintf("- Work style: %s\n", agent.Config.Personality.ApproachStyle))
		}

		if agent.Config.Personality.DecisionMaking != "" {
			prompt.WriteString(fmt.Sprintf("- Decision making: %s\n", agent.Config.Personality.DecisionMaking))
		}

		if agent.Config.Personality.RiskTolerance != "" {
			prompt.WriteString(fmt.Sprintf("- Risk tolerance: %s\n", agent.Config.Personality.RiskTolerance))
		}
	}

	return prompt.String()
}

// RecordInteraction records an interaction for learning
func (m *BackstoryManager) RecordInteraction(agentID string, interaction Interaction) error {
	agent, exists := m.agents[agentID]
	if !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "agent %s not found", agentID).
			WithComponent("BackstoryManager").
			WithOperation("RecordInteraction")
	}

	interaction.Timestamp = time.Now()
	agent.Memory.Interactions = append(agent.Memory.Interactions, interaction)

	// Keep only recent interactions (last 100)
	if len(agent.Memory.Interactions) > 100 {
		agent.Memory.Interactions = agent.Memory.Interactions[len(agent.Memory.Interactions)-100:]
	}

	return nil
}

// UpdateTeamContext updates the team context for an agent
func (m *BackstoryManager) UpdateTeamContext(agentID string, teamMembers []string) error {
	agent, exists := m.agents[agentID]
	if !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "agent %s not found", agentID).
			WithComponent("BackstoryManager").
			WithOperation("UpdateTeamContext")
	}

	agent.Context.TeamMembers = teamMembers
	return nil
}

// GetAgentPersonality returns the personality information for an agent
func (m *BackstoryManager) GetAgentPersonality(agentID string) (*config.Personality, error) {
	agent, exists := m.agents[agentID]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "agent %s not found", agentID).
			WithComponent("BackstoryManager").
			WithOperation("GetAgentPersonality")
	}

	return agent.Config.Personality, nil
}

// GetAgentBackstory returns the backstory information for an agent
func (m *BackstoryManager) GetAgentBackstory(agentID string) (*config.Backstory, error) {
	agent, exists := m.agents[agentID]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "agent %s not found", agentID).
			WithComponent("BackstoryManager").
			WithOperation("GetAgentBackstory")
	}

	return agent.Config.Backstory, nil
}
