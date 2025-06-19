// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/guild-ventures/guild-core/internal/setup"
)

func main() {
	// Get project path from command line or use current directory
	projectPath := "."
	if len(os.Args) > 1 {
		projectPath = os.Args[1]
	}

	generator := setup.NewAgentTemplateGenerator()

	// Example 1: List available templates
	fmt.Println("Available Templates:")
	fmt.Println("===================")
	for _, name := range generator.ListTemplates() {
		if template, exists := generator.GetTemplate(name); exists {
			fmt.Printf("- %s: %s (%s)\n", name, template.Description, template.Provider)
		}
	}
	fmt.Println()

	// Example 2: Quick setup for a provider
	fmt.Println("Quick Setup Example:")
	fmt.Println("===================")
	quickSetupExample(generator, projectPath)
	fmt.Println()

	// Example 3: Use built-in templates
	fmt.Println("Built-in Template Example:")
	fmt.Println("=========================")
	builtInTemplateExample(generator, projectPath)
	fmt.Println()

	// Example 4: Create custom minimal agents
	fmt.Println("Custom Minimal Agent Example:")
	fmt.Println("============================")
	customMinimalExample(generator, projectPath)
	fmt.Println()

	// Example 5: Create agent with optional backstory
	fmt.Println("Agent with Optional Backstory:")
	fmt.Println("==============================")
	optionalBackstoryExample(generator, projectPath)
	fmt.Println()

	// Example 6: Create provider-specific team
	fmt.Println("Provider-Specific Team Example:")
	fmt.Println("===============================")
	providerTeamExample(generator, projectPath)
}

func quickSetupExample(generator *setup.AgentTemplateGenerator, projectPath string) {
	// Quickest way to get started - creates manager and worker
	ctx := context.Background()
	err := generator.QuickSetup(ctx, projectPath, "openai", "gpt-4", "gpt-3.5-turbo")
	if err != nil {
		log.Printf("Quick setup failed: %v", err)
		return
	}
	
	fmt.Println("✓ Created manager.yml and worker-1.yml in .guild/agents/")
	fmt.Println("  These agents are ready to use with minimal configuration")
}

func builtInTemplateExample(generator *setup.AgentTemplateGenerator, projectPath string) {
	// Use a pre-configured Claude Code developer template
	if template, exists := generator.GetTemplate("claude-code-developer"); exists {
		ctx := context.Background()
		err := generator.GenerateAgentFile(ctx, projectPath, template)
		if err != nil {
			log.Printf("Failed to generate claude developer: %v", err)
			return
		}
		fmt.Println("✓ Created claude-developer.yml from built-in template")
		fmt.Printf("  Provider: %s, Model: %s\n", template.Provider, template.Model)
		fmt.Printf("  Capabilities: %v\n", template.Capabilities)
	}
}

func customMinimalExample(generator *setup.AgentTemplateGenerator, projectPath string) {
	// Create the most minimal agent possible
	minimal := generator.CreateCustomTemplate(
		"minimal-bot",
		"Minimal Bot",
		"worker",
		"ollama",
		"llama3:latest",
		"A minimal local agent",
		[]string{"general_tasks"},
	)
	
	ctx := context.Background()
	err := generator.GenerateAgentFile(ctx, projectPath, minimal)
	if err != nil {
		log.Printf("Failed to generate minimal bot: %v", err)
		return
	}
	
	fmt.Println("✓ Created minimal-bot.yml with just required fields")
	fmt.Println("  This agent has no backstory or personality - just functionality")
}

func optionalBackstoryExample(generator *setup.AgentTemplateGenerator, projectPath string) {
	// Create an agent with some personality but not full backstory
	seasoned := setup.AgentTemplate{
		ID:           "seasoned-dev",
		Name:         "Seasoned Developer",
		Type:         "specialist",
		Provider:     "anthropic",
		Model:        "claude-3-sonnet-20240229",
		Description:  "Experienced developer with practical wisdom",
		Capabilities: []string{"coding", "code_review", "mentoring", "architecture"},
		Tools:        []string{"code_executor", "git_tools", "test_runner"},
		
		// Optional fields - add just what makes sense
		Experience: "20 years across startups and Fortune 500",
		Philosophy: "Code is read more often than written - optimize for clarity",
		// Expertise left empty - not needed for this agent
	}
	
	ctx := context.Background()
	err := generator.GenerateAgentFile(ctx, projectPath, seasoned)
	if err != nil {
		log.Printf("Failed to generate seasoned developer: %v", err)
		return
	}
	
	fmt.Println("✓ Created seasoned-dev.yml with selective backstory elements")
	fmt.Println("  Only included experience and philosophy - no forced fields")
}

func providerTeamExample(generator *setup.AgentTemplateGenerator, projectPath string) {
	// Create a team optimized for a specific provider (Ollama for local/private work)
	ollamaTemplates := generator.ProviderTemplates("ollama")
	
	fmt.Printf("Creating local Ollama team with %d agents:\n", len(ollamaTemplates))
	
	for _, template := range ollamaTemplates {
		if template.Provider == "ollama" { // Skip generic templates
			ctx := context.Background()
		err := generator.GenerateAgentFile(ctx, projectPath, template)
			if err != nil {
				log.Printf("Failed to generate %s: %v", template.ID, err)
				continue
			}
			fmt.Printf("  ✓ %s - %s\n", template.ID, template.Description)
		}
	}
	
	// Add a custom local agent for the team
	customLocal := setup.AgentTemplate{
		ID:            "local-security",
		Name:          "Local Security Analyst",
		Type:          "specialist",
		Provider:      "ollama",
		Model:         "codellama:latest",
		Description:   "Privacy-focused security analysis",
		Capabilities:  []string{"security_review", "vulnerability_scan", "privacy_audit"},
		CostMagnitude: 0, // Free local model
		Temperature:   0.1, // Low temperature for consistent security analysis
	}
	
	ctx := context.Background()
	err := generator.GenerateAgentFile(ctx, projectPath, customLocal)
	if err != nil {
		log.Printf("Failed to generate local security analyst: %v", err)
		return
	}
	fmt.Printf("  ✓ %s - %s\n", customLocal.ID, customLocal.Description)
	
	fmt.Println("\nLocal team ready for privacy-sensitive development!")
}

// Example showing how to programmatically work with generated configs
func demonstrateConfigGeneration() {
	generator := setup.NewAgentTemplateGenerator()
	
	// Create a template
	template := setup.AgentTemplate{
		ID:           "example",
		Name:         "Example Agent",
		Type:         "worker",
		Provider:     "openai",
		Model:        "gpt-4",
		Description:  "Example for documentation",
		Capabilities: []string{"examples", "documentation"},
	}
	
	// Generate config (without writing file)
	config, err := generator.GenerateAgentConfig(template)
	if err != nil {
		log.Fatal(err)
	}
	
	// The config is now a full agent configuration
	fmt.Printf("Generated config:\n")
	fmt.Printf("  ID: %s\n", config.ID)
	fmt.Printf("  System Prompt: %s\n", config.SystemPrompt)
	fmt.Printf("  Max Tokens: %d (auto-set for %s)\n", config.MaxTokens, config.Type)
	fmt.Printf("  Temperature: %.1f (auto-set for %s)\n", config.Temperature, config.Type)
}