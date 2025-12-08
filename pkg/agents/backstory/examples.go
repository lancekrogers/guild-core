// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package backstory

import (
	"context"
	"fmt"
	"strings"

	"github.com/guild-framework/guild-core/pkg/agents/backstory/templates"
	"github.com/guild-framework/guild-core/pkg/prompts/layered"
)

// PersonalityDemo demonstrates how different agent personalities respond to the same scenario
type PersonalityDemo struct {
	manager *BackstoryManager
}

// NewPersonalityDemo creates a new personality demonstration
func NewPersonalityDemo() *PersonalityDemo {
	// Use the mock registry from tests
	registry := NewMockLayeredRegistry()
	manager := NewBackstoryManager(registry)

	return &PersonalityDemo{
		manager: manager,
	}
}

// DemoSamePromptDifferentPersonalities shows how the same prompt gets different personality treatments
func (demo *PersonalityDemo) DemoSamePromptDifferentPersonalities() {
	fmt.Println("🎭 GUILD AGENT PERSONALITY DEMONSTRATION")
	fmt.Println("=========================================")
	fmt.Println()

	// Register all specialist agents
	specialists := []struct {
		key  string
		role string
	}{
		{"security-sentinel", "Security Guardian"},
		{"performance-artisan", "Performance Optimizer"},
		{"frontend-artist", "UX Designer"},
		{"code-sage", "Code Architect"},
		{"data-mystic", "Data Scientist"},
	}

	for _, spec := range specialists {
		if template, exists := templates.SpecialistTemplates[spec.key]; exists {
			demo.manager.RegisterAgent(template)
		}
	}

	// Add guild master
	guildMaster := templates.CreateMedievalGuildMaster()
	demo.manager.RegisterAgent(guildMaster)

	scenarios := []struct {
		title  string
		prompt string
	}{
		{
			title:  "Authentication Implementation",
			prompt: "We need to add user authentication to our web application. What approach should we take?",
		},
		{
			title:  "Performance Issue",
			prompt: "Our application is running slowly. Users are complaining about long load times. How should we address this?",
		},
		{
			title:  "User Interface Design",
			prompt: "We need to design a dashboard for our analytics application. What should we consider?",
		},
		{
			title:  "Code Architecture Decision",
			prompt: "Should we refactor this legacy codebase or rewrite it from scratch?",
		},
	}

	for _, scenario := range scenarios {
		fmt.Printf("📝 SCENARIO: %s\n", scenario.title)
		fmt.Printf("Prompt: \"%s\"\n\n", scenario.prompt)

		// Show responses from different specialists
		for _, spec := range specialists {
			if template, exists := templates.SpecialistTemplates[spec.key]; exists {
				demo.showAgentResponse(template.ID, template.Name, spec.role, scenario.prompt)
			}
		}

		// Show guild master response
		demo.showAgentResponse(guildMaster.ID, guildMaster.Name, "Guild Master", scenario.prompt)

		fmt.Println("\n" + strings.Repeat("=", 80) + "\n")
	}
}

// DemoTeamDynamics shows how agents adapt when working in teams
func (demo *PersonalityDemo) DemoTeamDynamics() {
	fmt.Println("👥 TEAM COLLABORATION DEMONSTRATION")
	fmt.Println("====================================")
	fmt.Println()

	// Register agents
	securityAgent := templates.SpecialistTemplates["security-sentinel"]
	perfAgent := templates.SpecialistTemplates["performance-artisan"]
	uxAgent := templates.SpecialistTemplates["frontend-artist"]

	demo.manager.RegisterAgent(securityAgent)
	demo.manager.RegisterAgent(perfAgent)
	demo.manager.RegisterAgent(uxAgent)

	// Set up team collaboration
	demo.manager.UpdateTeamContext(securityAgent.ID, []string{perfAgent.ID, uxAgent.ID})
	demo.manager.UpdateTeamContext(perfAgent.ID, []string{securityAgent.ID, uxAgent.ID})
	demo.manager.UpdateTeamContext(uxAgent.ID, []string{securityAgent.ID, perfAgent.ID})

	prompt := "We're building a user login system. It needs to be secure, fast, and user-friendly. How should we approach this?"

	fmt.Printf("🎯 TEAM CHALLENGE: %s\n\n", prompt)

	fmt.Println("💬 TEAM RESPONSES (when working together):")
	fmt.Println()

	demo.showAgentResponse(securityAgent.ID, securityAgent.Name, "Security Lead", prompt)
	demo.showAgentResponse(perfAgent.ID, perfAgent.Name, "Performance Lead", prompt)
	demo.showAgentResponse(uxAgent.ID, uxAgent.Name, "UX Lead", prompt)
}

// DemoMoodAndContext shows how agent mood affects responses
func (demo *PersonalityDemo) DemoMoodAndContext() {
	fmt.Println("😊 MOOD AND CONTEXT DEMONSTRATION")
	fmt.Println("==================================")
	fmt.Println()

	perfAgent := templates.SpecialistTemplates["performance-artisan"]
	demo.manager.RegisterAgent(perfAgent)

	contexts := []struct {
		description string
		message     string
		urgency     string
	}{
		{
			description: "Casual inquiry",
			message:     "I'm curious about how we might optimize this function",
			urgency:     "normal",
		},
		{
			description: "Critical emergency",
			message:     "URGENT: Our production system is down due to performance issues!",
			urgency:     "high",
		},
		{
			description: "Learning opportunity",
			message:     "Can you help me understand how this optimization technique works?",
			urgency:     "normal",
		},
		{
			description: "Creative project",
			message:     "Let's build something amazing - a new high-performance API from scratch",
			urgency:     "normal",
		},
	}

	for _, ctx := range contexts {
		fmt.Printf("🎭 CONTEXT: %s\n", ctx.description)
		fmt.Printf("Message: \"%s\"\n", ctx.message)

		turnContext := &layered.TurnContext{
			UserMessage: ctx.message,
			Urgency:     ctx.urgency,
		}

		enhancedPrompt, _ := demo.manager.BuildPersonalityPrompt(
			context.Background(),
			perfAgent.ID,
			"Respond to the user's request.",
			turnContext,
		)

		// Extract mood from the enhanced prompt
		if strings.Contains(enhancedPrompt, "Current mood:") {
			lines := strings.Split(enhancedPrompt, "\n")
			for _, line := range lines {
				if strings.Contains(line, "Current mood:") {
					fmt.Printf("Agent's mood: %s\n", strings.TrimSpace(strings.Split(line, ":")[1]))
					break
				}
			}
		}

		fmt.Println()
	}
}

// DemoLearningAndMemory shows how agents learn from interactions
func (demo *PersonalityDemo) DemoLearningAndMemory() {
	fmt.Println("🧠 LEARNING AND MEMORY DEMONSTRATION")
	fmt.Println("=====================================")
	fmt.Println()

	uxAgent := templates.SpecialistTemplates["frontend-artist"]
	demo.manager.RegisterAgent(uxAgent)

	// Record some past interactions
	interactions := []Interaction{
		{
			Type:       "design_feedback",
			Summary:    "User loved the accessible navigation design with clear visual hierarchy",
			Outcome:    "success",
			UserRating: 9,
		},
		{
			Type:       "usability_test",
			Summary:    "Users struggled with complex multi-step form - too many fields at once",
			Outcome:    "needs_improvement",
			UserRating: 4,
		},
		{
			Type:       "accessibility_review",
			Summary:    "Screen reader users praised the semantic markup and clear labels",
			Outcome:    "success",
			UserRating: 10,
		},
	}

	for _, interaction := range interactions {
		demo.manager.RecordInteraction(uxAgent.ID, interaction)
	}

	prompts := []string{
		"How should we design the navigation for this new application?",
		"We need to create a multi-step user registration form. What's the best approach?",
		"What accessibility considerations should we keep in mind for this interface?",
	}

	for _, prompt := range prompts {
		fmt.Printf("🤔 QUESTION: %s\n", prompt)

		turnContext := &layered.TurnContext{
			UserMessage: prompt,
		}

		enhancedPrompt, _ := demo.manager.BuildPersonalityPrompt(
			context.Background(),
			uxAgent.ID,
			"Provide design guidance.",
			turnContext,
		)

		// Check if relevant experience is included
		if strings.Contains(enhancedPrompt, "Relevant Experience") {
			fmt.Println("✅ Agent drew from relevant past experience")
		} else {
			fmt.Println("ℹ️  No directly relevant experience found")
		}

		fmt.Println()
	}
}

// showAgentResponse displays how an agent would respond with their personality
func (demo *PersonalityDemo) showAgentResponse(agentID, agentName, role, prompt string) {
	turnContext := &layered.TurnContext{
		UserMessage: prompt,
	}

	enhancedPrompt, err := demo.manager.BuildPersonalityPrompt(
		context.Background(),
		agentID,
		"Please provide your professional advice on this matter.",
		turnContext,
	)

	if err != nil {
		fmt.Printf("❌ Error getting response from %s: %v\n\n", agentName, err)
		return
	}

	fmt.Printf("🗣️  **%s** (%s):\n", agentName, role)

	// Extract key personality elements from the prompt
	if strings.Contains(enhancedPrompt, "Your Identity and Background") {
		// Show the personality is active
		fmt.Printf("   (Responding with %s personality)\n", extractPersonalityHint(enhancedPrompt))
	}

	// Show a simulated response based on the personality
	response := generateSimulatedResponse(agentName, role, prompt, enhancedPrompt)
	fmt.Printf("   💬 \"%s\"\n\n", response)
}

// extractPersonalityHint extracts personality indicators from the enhanced prompt
func extractPersonalityHint(enhancedPrompt string) string {
	if strings.Contains(enhancedPrompt, "vigilant") || strings.Contains(enhancedPrompt, "guardian") {
		return "security-focused"
	}
	if strings.Contains(enhancedPrompt, "velocity") || strings.Contains(enhancedPrompt, "optimization") {
		return "performance-oriented"
	}
	if strings.Contains(enhancedPrompt, "empathy") || strings.Contains(enhancedPrompt, "user") {
		return "user-centered"
	}
	if strings.Contains(enhancedPrompt, "wisdom") || strings.Contains(enhancedPrompt, "principle") {
		return "architecture-minded"
	}
	if strings.Contains(enhancedPrompt, "pattern") || strings.Contains(enhancedPrompt, "data") {
		return "analytical"
	}
	if strings.Contains(enhancedPrompt, "leadership") || strings.Contains(enhancedPrompt, "guild") {
		return "strategic"
	}
	return "professional"
}

// generateSimulatedResponse creates a personality-appropriate response
func generateSimulatedResponse(agentName, role, prompt, enhancedPrompt string) string {
	switch role {
	case "Security Guardian":
		return "I must insist on a fortress-like approach. We'll implement multi-layered authentication with bcrypt hashing, rate limiting, and 2FA. Security cannot be compromised - I've seen too many breaches from cutting corners."

	case "Performance Optimizer":
		return "Let's measure first! I'd benchmark our current auth flow, then implement JWT with Redis caching. We could reduce auth time from 200ms to 50ms with proper optimization. *excitement about the performance gains*"

	case "UX Designer":
		return "We need to design this from the user's perspective. A simple, accessible login form with clear error messages and social login options. Users shouldn't have to think - they should feel welcomed into our digital realm."

	case "Code Architect":
		return "This is an opportunity to apply proven patterns. I recommend the OAuth 2.0 standard with proper separation of concerns. We'll build something maintainable that future artisans can easily understand and extend."

	case "Data Scientist":
		return "Let's look at the patterns in our user data. We can implement adaptive authentication that learns from user behavior patterns while respecting privacy. The data will guide us to the most effective approach."

	case "Guild Master":
		return "This requires coordination between all our specialists. Security must lead the design, Performance will optimize it, UX will make it intuitive, and our Code Sage will ensure it's maintainable. Together, we'll craft something worthy of our guild."

	default:
		return "I'll approach this with my specialized expertise and the wisdom of our guild's traditions."
	}
}

// RunAllDemos runs all personality demonstrations
func RunAllDemos() {
	demo := NewPersonalityDemo()

	demo.DemoSamePromptDifferentPersonalities()
	fmt.Println()
	demo.DemoTeamDynamics()
	fmt.Println()
	demo.DemoMoodAndContext()
	fmt.Println()
	demo.DemoLearningAndMemory()

	fmt.Println("🎭 PERSONALITY DEMONSTRATION COMPLETE")
	fmt.Println("=====================================")
	fmt.Println("Each agent brings their unique perspective and medieval guild identity to every interaction.")
	fmt.Println("This creates a rich, immersive experience where users feel they're working with skilled")
	fmt.Println("artisans who have real expertise, personality, and history.")
}
