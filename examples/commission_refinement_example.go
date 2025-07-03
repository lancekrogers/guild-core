// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build example
// +build example

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lancekrogers/guild/pkg/agents/core/manager"
	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/orchestrator"
	"github.com/lancekrogers/guild/pkg/providers/anthropic"
	"github.com/lancekrogers/guild/pkg/providers/openai"
	"github.com/lancekrogers/guild/pkg/registry"
)

// Example of using the complete commission refinement pipeline
func main() {
	ctx := context.Background()

	// Step 1: Initialize the registry
	reg := registry.NewComponentRegistry()

	// Step 2: Set up providers
	if err := setupProviders(reg); err != nil {
		log.Fatalf("Failed to setup providers: %v", err)
	}

	// Step 3: Set up memory and storage
	if err := setupMemory(reg); err != nil {
		log.Fatalf("Failed to setup memory: %v", err)
	}

	// Step 4: Set up prompts
	if err := setupPrompts(reg); err != nil {
		log.Fatalf("Failed to setup prompts: %v", err)
	}

	// Step 5: Create the commission integration service
	integrationService, err := orchestrator.NewCommissionIntegrationService(reg)
	if err != nil {
		log.Fatalf("Failed to create integration service: %v", err)
	}

	// Step 6: Load guild configuration
	guildConfig := &config.GuildConfig{
		Name:        "Example Guild",
		Description: "A guild for demonstrating commission refinement",
		Agents: []config.AgentConfig{
			{
				ID:           "backend-artisan",
				Name:         "Backend Artisan",
				Type:         "specialist",
				Description:  "Specializes in backend development",
				Capabilities: []string{"backend", "api", "database"},
			},
			{
				ID:           "frontend-artisan",
				Name:         "Frontend Artisan",
				Type:         "specialist",
				Description:  "Specializes in frontend development",
				Capabilities: []string{"frontend", "ui", "react"},
			},
			{
				ID:           "test-artisan",
				Name:         "Test Artisan",
				Type:         "specialist",
				Description:  "Specializes in testing",
				Capabilities: []string{"testing", "qa", "automation"},
			},
		},
	}

	// Step 7: Create a commission
	commission := manager.Commission{
		ID:    "example-001",
		Title: "Build a Task Management System",
		Description: `Create a web-based task management system with the following features:
		- User authentication and authorization
		- Create, read, update, and delete tasks
		- Assign tasks to team members
		- Set due dates and priorities
		- Filter and search tasks
		- Real-time updates using WebSockets
		- RESTful API for third-party integrations
		- Mobile-responsive design`,
		Domain: "web-app",
		Context: map[string]interface{}{
			"tech_stack":     "React, Node.js, PostgreSQL",
			"timeline":       "4 weeks",
			"team_size":      "3 developers",
			"deployment":     "AWS",
			"priority_focus": "user experience and performance",
		},
	}

	// Step 8: Process the commission
	fmt.Println("Processing commission:", commission.Title)
	fmt.Println("Domain:", commission.Domain)
	fmt.Println()

	// Add output directory to context if you want files written
	ctx = context.WithValue(ctx, "output_dir", ".guild/commissions/refined/example-001")

	result, err := integrationService.ProcessCommissionToTasks(ctx, commission, guildConfig)
	if err != nil {
		log.Fatalf("Failed to process commission: %v", err)
	}

	// Step 9: Display results
	fmt.Println("Commission processing complete!")
	fmt.Printf("Created %d tasks\n", result.GetTaskCount())
	fmt.Printf("Assigned to %d artisans\n", result.GetAssignedArtisanCount())
	fmt.Println()

	// Display tasks by status
	todoTasks := result.GetTasksByStatus("todo")
	fmt.Printf("Todo tasks: %d\n", len(todoTasks))
	for _, task := range todoTasks {
		fmt.Printf("  - [%s] %s (Priority: %s)\n", task.ID, task.Title, task.Priority)
		if task.AssignedTo != "" {
			fmt.Printf("    Assigned to: %s\n", task.AssignedTo)
		}
	}

	// Step 10: Use the task bridge directly for additional operations
	taskBridge := integrationService.GetTaskBridge()
	if taskBridge != nil {
		// Get all tasks for this commission
		commissionTasks, err := taskBridge.GetTasksForCommission(ctx, commission.ID)
		if err == nil {
			fmt.Printf("\nTotal tasks for commission %s: %d\n", commission.ID, len(commissionTasks))
		}
	}

	// Step 11: Demonstrate using the refiner directly
	fmt.Println("\n--- Direct Refiner Usage ---")
	factory := integrationService.GetGuildMasterFactory()
	if factory != nil {
		refiner, err := factory.CreateCommissionRefinerWithDefaults()
		if err == nil {
			// Use the simple interface
			refinedContent, err := refiner.(*manager.GuildMasterRefiner).RefineCommissionSimple(
				ctx,
				"Create a CLI tool for managing Docker containers",
				"cli-tool",
			)
			if err == nil {
				fmt.Println("Refined content preview:")
				if len(refinedContent) > 500 {
					fmt.Println(refinedContent[:500] + "...")
				} else {
					fmt.Println(refinedContent)
				}
			}
		}
	}
}

// setupProviders configures AI providers
func setupProviders(reg registry.ComponentRegistry) error {
	providerReg := reg.Providers()

	// Try to register Anthropic if API key is available
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		provider := anthropic.NewClient(apiKey)
		if err := providerReg.RegisterProvider("anthropic", provider); err != nil {
			return err
		}
		log.Println("Registered Anthropic provider")
	}

	// Try to register OpenAI if API key is available
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		provider := openai.NewClient(apiKey)
		if err := providerReg.RegisterProvider("openai", provider); err != nil {
			return err
		}
		log.Println("Registered OpenAI provider")
	}

	// Set default provider
	if _, err := providerReg.GetProvider("anthropic"); err == nil {
		providerReg.SetDefaultProvider("anthropic")
	} else if _, err := providerReg.GetProvider("openai"); err == nil {
		providerReg.SetDefaultProvider("openai")
	} else {
		return gerror.New(gerror.ErrCodeMissingRequired, "no AI providers available - set ANTHROPIC_API_KEY or OPENAI_API_KEY", nil).
			WithComponent("commission_example").
			WithOperation("setupProviders")
	}

	return nil
}

// setupMemory configures the memory system
func setupMemory(reg registry.ComponentRegistry) error {
	// Create BoltDB store
	store, err := boltdb.NewStore(".guild/memory.db")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create BoltDB store").
			WithComponent("commission_example").
			WithOperation("setupMemory")
	}

	// Register with memory registry
	memReg := reg.Memory()
	return memReg.RegisterMemoryStore("default", store)
}

// setupPrompts configures the prompt system
func setupPrompts(reg registry.ComponentRegistry) error {
	// The prompt registry and integration service handle prompts internally
	// No additional setup needed for this example
	return nil
}
