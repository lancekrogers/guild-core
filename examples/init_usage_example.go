// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package examples demonstrates how to use the enhanced guild init functionality
package examples

import (
	"fmt"
	"path/filepath"
)

// ExampleCampaignStructure shows the complete campaign directory structure created by guild init
func ExampleCampaignStructure() {
	fmt.Println("After running 'guild init', the following structure is created:")
	fmt.Println()
	fmt.Println("project-root/")
	fmt.Println("├── .campaign/                    # Campaign-specific configuration")
	fmt.Println("│   ├── .hash                    # Unique campaign identifier (16 chars)")
	fmt.Println("│   ├── campaign.yaml            # Campaign configuration")
	fmt.Println("│   ├── socket-registry.yaml     # Daemon socket information")
	fmt.Println("│   ├── agents/                  # Agent configurations")
	fmt.Println("│   │   ├── elena-guild-master.yaml")
	fmt.Println("│   │   ├── marcus-developer.yaml")
	fmt.Println("│   │   └── vera-tester.yaml")
	fmt.Println("│   ├── guilds/                  # Guild configurations")
	fmt.Println("│   │   └── default-guild.yaml")
	fmt.Println("│   ├── memory.db                # SQLite database")
	fmt.Println("│   ├── prompts/                 # Custom prompt templates")
	fmt.Println("│   ├── tools/                   # Project-specific tools")
	fmt.Println("│   └── workspaces/              # Agent workspaces")
	fmt.Println("├── commissions/                 # User-facing commission files")
	fmt.Println("│   └── refined/                 # AI-refined commissions")
	fmt.Println("├── corpus/                      # Project documentation")
	fmt.Println("│   └── index/                   # Vector store indices")
	fmt.Println("└── kanban/                      # Task tracking")
}

// ExampleCampaignYAML shows the structure of campaign.yaml
func ExampleCampaignYAML() {
	fmt.Println("Example campaign.yaml:")
	fmt.Println()
	fmt.Println(`campaign:
  hash: "a1b2c3d4e5f6g7h8"  # Unique 16-character identifier
  name: "guild-demo"
  project_name: "my-project"
  project_type: "go"
  created_at: "2025-01-26T10:30:00Z"
  version: "1.0.0"

daemon:
  socket_path: "/tmp/guild-a1b2c3d4e5f6g7h8.sock"
  log_level: "info"

storage:
  database: "memory.db"
  backend: "sqlite"

settings:
  auto_start_daemon: true
  session_timeout: "24h"
  max_agents: 10`)
}

// ExampleAgentConfiguration shows an agent YAML structure
func ExampleAgentConfiguration() {
	fmt.Println("Example elena-guild-master.yaml:")
	fmt.Println()
	fmt.Println(`id: "elena-guild-master"
name: "Elena"
type: "manager"
description: "Guild Master who orchestrates the team and manages projects"
provider: "anthropic"
model: "claude-3-opus-20240229"

capabilities:
  - task_decomposition
  - agent_coordination
  - strategic_planning
  - progress_monitoring
  - commission_refinement

tools:
  - task_planner
  - agent_coordinator
  - commission_refiner

max_tokens: 4000
temperature: 0.1

backstory:
  experience: "20 years leading guilds and managing complex projects"
  expertise: "Project management, team coordination, strategic planning"
  philosophy: "Success comes from clear communication and empowering team members"
  guild_rank: "Master"

system_prompt: |
  You are Elena, the Guild Master. You coordinate the team of AI agents
  and ensure projects succeed through careful planning and delegation.
  You break down complex tasks, assign work to the right agents, and
  monitor progress. Maintain a professional yet supportive leadership style.`)
}

// ExampleGuildConfiguration shows the default guild configuration
func ExampleGuildConfiguration() {
	fmt.Println("Example default-guild.yaml:")
	fmt.Println()
	fmt.Println(`guild:
  name: "my-project"
  description: "my-project Guild - Orchestrating AI agents for development"
  version: "1.0.0"
  created_at: "2025-01-26T10:30:00Z"

manager:
  default: "elena-guild-master"

agents:
  - elena-guild-master
  - marcus-developer
  - vera-tester

workflows:
  default: "collaborative"
  available:
    - collaborative
    - sequential
    - parallel

cost_optimization:
  enabled: true
  max_cost: 100.0
  alert_at: 80.0
  currency: "USD"`)
}

// ExampleProjectTypeAdaptation shows how agents adapt to different project types
func ExampleProjectTypeAdaptation() {
	fmt.Println("Agent adaptations by project type:")
	fmt.Println()
	
	projectTypes := []struct {
		Language    string
		Tools       []string
		Capabilities []string
		Expertise   string
	}{
		{
			Language:     "go",
			Tools:        []string{"go_test", "go_build"},
			Capabilities: []string{"goroutines", "channels"},
			Expertise:    "Go, concurrency patterns, error handling, testing",
		},
		{
			Language:     "python",
			Tools:        []string{"pytest", "pip"},
			Capabilities: []string{"data_analysis", "machine_learning"},
			Expertise:    "Python, Django/Flask, async programming, data science libraries",
		},
		{
			Language:     "javascript",
			Tools:        []string{"npm", "webpack"},
			Capabilities: []string{"frontend", "backend"},
			Expertise:    "JavaScript/TypeScript, React/Vue/Angular, Node.js, modern web development",
		},
		{
			Language:     "rust",
			Tools:        []string{"cargo", "rustfmt"},
			Capabilities: []string{"memory_safety", "zero_cost_abstractions"},
			Expertise:    "Rust, memory safety, performance optimization, systems programming",
		},
	}

	for _, pt := range projectTypes {
		fmt.Printf("%s Project:\n", pt.Language)
		fmt.Printf("  Additional Tools: %v\n", pt.Tools)
		fmt.Printf("  Additional Capabilities: %v\n", pt.Capabilities)
		fmt.Printf("  Marcus's Expertise: %s\n", pt.Expertise)
		fmt.Println()
	}
}

// ExampleInitWorkflow shows the complete initialization workflow
func ExampleInitWorkflow() {
	fmt.Println("Guild Init Workflow:")
	fmt.Println()
	fmt.Println("1. Create campaign structure (.campaign/ and user directories)")
	fmt.Println("2. Detect project type (Go, Python, JS, Rust, etc.)")
	fmt.Println("3. Auto-detect AI providers (Anthropic, OpenAI, Ollama, etc.)")
	fmt.Println("4. Generate campaign.yaml with unique hash")
	fmt.Println("5. Create agent configurations (Elena, Marcus, Vera)")
	fmt.Println("6. Adapt agents to detected project type")
	fmt.Println("7. Create guild configuration")
	fmt.Println("8. Initialize SQLite database")
	fmt.Println("9. Set up socket registry for daemon")
	fmt.Println()
	fmt.Println("Result: Complete campaign ready for 'guild chat'")
}

// GetCampaignPath returns the path to campaign directory
func GetCampaignPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".campaign")
}

// GetAgentPath returns the path to a specific agent configuration
func GetAgentPath(projectRoot, agentID string) string {
	return filepath.Join(projectRoot, ".campaign", "agents", agentID+".yaml")
}

// GetGuildPath returns the path to guild configuration
func GetGuildPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".campaign", "guilds", "default-guild.yaml")
}