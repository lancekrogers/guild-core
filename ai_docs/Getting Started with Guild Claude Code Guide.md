# Getting Started with Guild - Claude Code Guide

## Project Overview

Guild is a Go-based framework for orchestrating AI agents working together in "guilds" to complete complex tasks. It focuses on:

- **Simplicity**: Easy to configure and extend
- **Cost optimization**: Smart use of local vs. API models
- **Flexibility**: Multi-model support (OpenAI, Claude, Ollama, etc.)
- **Human-in-the-loop**: Kanban-based workflow with manual review points

## Development Environment

```bash
# Clone the repository
git clone https://github.com/your-username/guild.git
cd guild

# Install dependencies
go get -u github.com/spf13/cobra
go get -u github.com/boltdb/bolt
go get -u github.com/qdrant/go-client
go get -u github.com/zeromq/goczmq
```

## Project Structure

The Guild project follows this structure:

```
guild/
├── ai_docs/                    # Agent knowledge repository
│   ├── api_docs/               # External API documentation
│   ├── architecture/           # System design documentation
│   ├── integration_guides/     # Integration documentation
│   └── patterns/               # Go patterns and best practices
├── cmd/
│   └── guild/
│       ├── main.go             # Main entry point for the CLI
│       ├── init.go             # Init command implementation
│       ├── run.go              # Run command implementation
│       └── commands/           # Additional command implementations
├── pkg/
│   ├── agent/                  # Agent interfaces and implementations
│   ├── comms/                  # Communication interfaces
│   │   ├── events/             # Event definitions
│   │   ├── api/                # HTTP API (if needed)
│   │   └── transport/          # Transport implementations (ZeroMQ, etc.)
│   ├── config/                 # Configuration parsing
│   ├── kanban/                 # Task tracking system
│   ├── memory/                 # Memory systems
│   │   ├── boltdb/             # BoltDB implementation
│   │   ├── vector/             # Vector store implementations
│   │   └── interface.go        # Memory interface definitions
│   ├── objective/              # Objective parsing and management
│   │   ├── markdown/           # Markdown parser
│   │   └── template/           # Objective templates
│   ├── orchestrator/           # Orchestration and coordination
│   └── providers/              # LLM provider implementations
├── tools/                      # Tool implementations
│   ├── interfaces.go           # Tool interface definitions
│   ├── factory.go              # Tool registration and creation
│   └── implementations/        # Specific tool implementations
├── examples/                   # Example guilds and use cases
├── specs/                      # Design specifications
├── testdata/                   # Test fixtures
└── Makefile                    # Common development operations
```

## Core Interfaces

Let's refine the fundamental interfaces that define Guild's behavior, aligning with your existing structure.

### Agent Interface

```go
// pkg/agent/agent.go
package agent

import (
	"context"

	"github.com/your-username/guild/pkg/kanban"
	"github.com/your-username/guild/pkg/providers"
	"github.com/your-username/guild/tools"
)

// Agent represents a worker that can execute tasks using an LLM
type Agent interface {
	// ID returns the unique identifier for this agent
	ID() string

	// Name returns the display name of this agent
	Name() string

	// Execute runs a task and returns the result
	Execute(ctx context.Context, task kanban.Task) (kanban.Result, error)

	// Tools returns the list of tools available to this agent
	Tools() []tools.Tool

	// Provider returns the LLM provider for this agent
	Provider() providers.Provider

	// Board returns the agent's personal Kanban board
	Board() kanban.Board

	// GetPromptChain returns the current prompt chain for a task
	GetPromptChain(taskID string) (PromptChain, error)
}

// PromptChain represents a sequence of prompt-response pairs
type PromptChain struct {
	// TaskID is the associated task
	TaskID string

	// Entries are the prompt-response pairs
	Entries []PromptEntry
}

// PromptEntry represents a single prompt-response pair
type PromptEntry struct {
	// Prompt is the input to the LLM
	Prompt string

	// Response is the output from the LLM
	Response string

	// TokensUsed is the total tokens consumed
	TokensUsed int

	// ToolsUsed is the list of tools that were used
	ToolsUsed []string

	// Timestamp is when this entry was created
	Timestamp time.Time
}

// Config defines the configuration for an agent
type Config struct {
	// ID is the unique identifier
	ID string

	// Name is the display name
	Name string

	// ProviderConfig defines the LLM provider configuration
	ProviderConfig providers.ProviderConfig

	// ToolIDs are the names of tools this agent can use
	ToolIDs []string

	// Cost is the relative cost of using this agent
	Cost int
}
```

### Orchestrator Interface

Instead of a direct Guild interface, we'll use your existing orchestrator pattern:

```go
// pkg/orchestrator/orchestrator.go
package orchestrator

import (
	"context"

	"github.com/your-username/guild/pkg/agent"
	"github.com/your-username/guild/pkg/kanban"
	"github.com/your-username/guild/pkg/objective"
)

// Orchestrator coordinates multiple agents working toward objectives
type Orchestrator interface {
	// ID returns the unique identifier
	ID() string

	// Name returns the display name
	Name() string

	// Agents returns the list of agents
	Agents() []agent.Agent

	// Execute runs the orchestrator with the given objective
	Execute(ctx context.Context, obj objective.Objective) error

	// Board returns the orchestrator's master Kanban board
	Board() kanban.Board

	// EventBus returns the event bus for this orchestrator
	EventBus() EventBus

	// Manager returns the manager agent (if any)
	Manager() agent.Agent
}

// Config defines orchestrator configuration
type Config struct {
	// ID is the unique identifier
	ID string

	// Name is the display name
	Name string

	// AgentIDs are the IDs of agents in this orchestrator
	AgentIDs []string

	// ManagerID is the ID of the manager agent (optional)
	ManagerID string

	// CostConfig defines cost settings
	CostConfig CostConfig
}

// CostConfig defines cost settings
type CostConfig struct {
	// APIModels maps API model names to costs
	APIModels map[string]int

	// LocalModels maps local model names to costs
	LocalModels map[string]int

	// CLITools maps tool names to costs
	CLITools map[string]int
}
```

### Tool Interface

```go
// tools/interfaces.go
package tools

import (
	"context"
)

// Tool represents a capability that can be used by an agent
type Tool interface {
	// ID returns the unique identifier for this tool
	ID() string

	// Name returns the display name of this tool
	Name() string

	// Description returns a natural language description of when to use this tool
	Description() string

	// Execute runs the tool with the given input
	Execute(ctx context.Context, input string) (string, error)

	// Cost returns the relative cost of using this tool
	Cost() int
}

// Registry manages available tools
type Registry interface {
	// Register adds a tool to the registry
	Register(tool Tool) error

	// Get retrieves a tool by ID
	Get(id string) (Tool, error)

	// List returns all registered tools
	List() []Tool
}
```

### Event System

```go
// pkg/orchestrator/eventbus.go
package orchestrator

import (
	"context"
)

// EventType represents the type of event
type EventType string

const (
	EventTaskCreated  EventType = "task_created"
	EventTaskUpdated  EventType = "task_updated"
	EventTaskMoved    EventType = "task_moved"
	EventTaskComplete EventType = "task_completed"
	EventTaskBlocked  EventType = "task_blocked"
)

// Event represents a system event
type Event struct {
	// Type is the event type
	Type EventType

	// TaskID is the associated task
	TaskID string

	// AgentID is the associated agent
	AgentID string

	// Data contains event-specific details
	Data map[string]interface{}

	// Timestamp is when the event occurred
	Timestamp time.Time
}

// EventBus manages event publishing and subscription
type EventBus interface {
	// Publish sends an event to all subscribers
	Publish(ctx context.Context, event Event) error

	// Subscribe registers a callback for events
	Subscribe(eventType EventType, callback func(Event)) (string, error)

	// Unsubscribe removes a subscription
	Unsubscribe(subscriptionID string) error
}
```

### Kanban System

```go
// pkg/kanban/taskmodel.go
package kanban

import (
	"time"
)

// TaskStatus represents the state of a task
type TaskStatus string

const (
	StatusToDo       TaskStatus = "ToDo"
	StatusInProgress TaskStatus = "InProgress"
	StatusBlocked    TaskStatus = "Blocked"
	StatusDone       TaskStatus = "Done"
)

// Task represents a unit of work
type Task struct {
	// ID is the unique identifier
	ID string

	// Title is a short summary
	Title string

	// Description is the full specification
	Description string

	// Status is the current state
	Status TaskStatus

	// AgentID is the assigned agent
	AgentID string

	// ObjectiveID is the associated objective
	ObjectiveID string

	// CreatedAt is when the task was created
	CreatedAt time.Time

	// UpdatedAt is when the task was last updated
	UpdatedAt time.Time

	// Tags contains searchable tags
	Tags []string

	// Metadata contains additional information
	Metadata map[string]interface{}
}

// Board manages a collection of tasks
type Board interface {
	// Add creates a new task
	Add(task Task) error

	// Get retrieves a task by ID
	Get(id string) (Task, error)

	// Update modifies an existing task
	Update(task Task) error

	// List returns all tasks with optional filtering
	List(filter map[string]interface{}) ([]Task, error)

	// Move changes a task's status
	Move(id string, status TaskStatus) error

	// GetByAgent returns tasks assigned to an agent
	GetByAgent(agentID string) ([]Task, error)

	// GetByStatus returns tasks with a specific status
	GetByStatus(status TaskStatus) ([]Task, error)
}

// Result represents the output of a task execution
type Result struct {
	// TaskID is the associated task
	TaskID string

	// Success indicates if the task succeeded
	Success bool

	// Output is the task result
	Output string

	// Error is any error message
	Error string

	// CompletedAt is when the task finished
	CompletedAt time.Time

	// ToolsUsed lists the tools that were used
	ToolsUsed []string

	// Metadata contains additional information
	Metadata map[string]interface{}
}
```

## CLI Implementation

Following your established structure, let's enhance the CLI implementation with expanded commands and modular organization:

```go
// cmd/guild/main.go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/your-username/guild/cmd/guild/commands"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "guild",
		Short: "Guild - AI Agent Framework",
		Long:  `Guild is a framework for orchestrating AI agents working together in guilds.`,
	}

	// Add subcommands
	rootCmd.AddCommand(commands.InitCmd())
	rootCmd.AddCommand(commands.RunCmd())
	rootCmd.AddCommand(commands.AgentCmd())
	rootCmd.AddCommand(commands.ToolCmd())
	rootCmd.AddCommand(commands.ObjectiveCmd())
	rootCmd.AddCommand(commands.KanbanCmd())
	rootCmd.AddCommand(commands.DashboardCmd())
	rootCmd.AddCommand(commands.ConfigCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```

### Modular Command Implementation

```go
// cmd/guild/commands/init.go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/your-username/guild/pkg/config"
)

// InitCmd returns the init command
func InitCmd() *cobra.Command {
	var (
		templateName string
		offline      bool
	)

	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new Guild project",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]
			fmt.Printf("Initializing new Guild project: %s\n", projectName)

			// Create project scaffolding
			err := config.CreateProjectScaffolding(projectName, templateName, offline)
			if err != nil {
				fmt.Printf("Error creating project: %s\n", err)
				return
			}

			fmt.Printf("Successfully created project in ./%s\n", projectName)
			fmt.Println("Next steps:")
			fmt.Println("  1. cd", projectName)
			fmt.Println("  2. guild add-agent")
			fmt.Println("  3. guild add-objective")
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&templateName, "template", "t", "basic", "Template to use (basic, dev, content)")
	cmd.Flags().BoolVarP(&offline, "offline", "o", false, "Initialize for offline use only")

	return cmd
}
```

### Agent Command with Subcommands

```go
// cmd/guild/commands/agent.go
package commands

import (
	"github.com/spf13/cobra"
)

// AgentCmd returns the agent management command
func AgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agents",
		Long:  "Create, list, and configure agents for your Guild project",
	}

	// Add subcommands
	cmd.AddCommand(agentAddCmd())
	cmd.AddCommand(agentListCmd())
	cmd.AddCommand(agentShowCmd())
	cmd.AddCommand(agentConfigCmd())

	return cmd
}

// agentAddCmd returns the command to add a new agent
func agentAddCmd() *cobra.Command {
	var (
		model string
		tools []string
		cost  int
	)

	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new agent",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Implementation details...
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&model, "model", "m", "claude-3-sonnet", "LLM model to use")
	cmd.Flags().StringSliceVarP(&tools, "tools", "t", []string{}, "Tools this agent can use")
	cmd.Flags().IntVarP(&cost, "cost", "c", 0, "Relative cost of this agent")

	return cmd
}

// Additional agent subcommands...
```

### Interactive Commands

The Guild CLI should support interactive prompting when flags are not provided:

```go
// cmd/guild/commands/objective.go
package commands

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// ObjectiveCmd returns the objective management command
func ObjectiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "objective",
		Short: "Manage objectives",
		Long:  "Create, list, and configure objectives for your Guild project",
	}

	// Add subcommands
	cmd.AddCommand(objectiveAddCmd())
	cmd.AddCommand(objectiveListCmd())
	cmd.AddCommand(objectiveShowCmd())

	return cmd
}

// objectiveAddCmd returns the command to add a new objective
func objectiveAddCmd() *cobra.Command {
	var (
		title       string
		description string
		path        string
		template    string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new objective",
		Run: func(cmd *cobra.Command, args []string) {
			// Interactive mode if flags not provided
			if title == "" {
				prompt := promptui.Prompt{
					Label: "Objective Title",
				}
				result, err := prompt.Run()
				if err != nil {
					fmt.Printf("Error: %s\n", err)
					return
				}
				title = result
			}

			// Similar prompts for other missing fields...

			// Implementation details...
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&title, "title", "t", "", "Objective title")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Objective description")
	cmd.Flags().StringVarP(&path, "path", "p", "", "Path to save the objective")
	cmd.Flags().StringVarP(&template, "template", "m", "basic", "Template to use (basic, code, content)")

	return cmd
}

// Additional objective subcommands...
```

## Configuration System

Building on your existing `pkg/config/loader.go`, let's enhance the configuration system with additional features:

```go
// pkg/config/loader.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config represents the top-level configuration
type Config struct {
	// Agents is the list of agent configurations
	Agents []AgentConfig `yaml:"agents"`

	// Guilds is the list of guild configurations (orchestrators)
	Guilds []GuildConfig `yaml:"guilds"`

	// Costs defines relative costs for models and tools
	Costs CostConfig `yaml:"costs"`

	// APIKeys contains API keys for LLM providers
	APIKeys map[string]string `yaml:"api_keys"`

	// ProjectRoot is the root directory of the project
	ProjectRoot string `yaml:"-"`
}

// AgentConfig represents an agent configuration
type AgentConfig struct {
	// Name is the agent's display name
	Name string `yaml:"name"`

	// Provider is the LLM provider (openai, anthropic, ollama, etc.)
	Provider string `yaml:"provider"`

	// Model is the specific model name (gpt-4, claude-3-opus, llama3-8b, etc.)
	Model string `yaml:"model"`

	// Tools is the list of tool names
	Tools []string `yaml:"tools"`

	// Personality contains agent personality settings (optional)
	Personality *PersonalityConfig `yaml:"personality,omitempty"`
}

// PersonalityConfig defines agent personality characteristics
type PersonalityConfig struct {
	// Persona is a short description of the agent's role or character
	Persona string `yaml:"persona"`

	// Style defines the agent's communication style
	Style string `yaml:"style"`

	// Expertise defines the agent's areas of expertise
	Expertise []string `yaml:"expertise"`
}

// GuildConfig represents a guild (orchestrator) configuration
type GuildConfig struct {
	// Name is the guild's display name
	Name string `yaml:"name"`

	// Agents is the list of agent names
	Agents []string `yaml:"agents"`

	// Manager is the optional manager agent
	Manager string `yaml:"manager,omitempty"`

	// ObjectivesPath is the path to the objectives directory
	ObjectivesPath string `yaml:"objectives_path,omitempty"`
}

// CostConfig defines costs for various components
type CostConfig struct {
	// CLITools defines costs for command-line tools
	CLITools map[string]int `yaml:"cli_tools"`

	// LocalModels defines costs for local models
	LocalModels map[string]int `yaml:"local_models"`

	// APIModels defines costs for API models
	APIModels map[string]int `yaml:"api_models"`

	// Budget defines the optional maximum cost for a run
	Budget *int `yaml:"budget,omitempty"`
}

// ToolConfig defines a command-line tool configuration
type ToolConfig struct {
	// Name is the tool's display name
	Name string `yaml:"name"`

	// Command is the command to execute
	Command string `yaml:"cmd"`

	// Description is when to use this tool
	Description string `yaml:"context_description"`

	// WorkingDir is the optional working directory
	WorkingDir string `yaml:"working_dir,omitempty"`

	// Args are optional arguments with template support
	Args map[string]string `yaml:"args,omitempty"`
}

// LoadConfig reads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	// Try loading .env file if it exists
	_ = godotenv.Load()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set project root to the directory containing the config file
	config.ProjectRoot = filepath.Dir(path)

	// Process environment variables for API keys
	if config.APIKeys == nil {
		config.APIKeys = make(map[string]string)
	}

	// Add API keys from environment
	envKeyPrefixes := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "DEEPSEEK_API_KEY", "ORA_API_KEY"}
	for _, prefix := range envKeyPrefixes {
		if key := os.Getenv(prefix); key != "" {
			provider := strings.ToLower(strings.Split(prefix, "_")[0])
			config.APIKeys[provider] = key
		}
	}

	return &config, nil
}

// SaveConfig writes the configuration to a file
func SaveConfig(config *Config, path string) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Convert to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// LoadTools reads tool configurations from a directory
func LoadTools(dir string) ([]ToolConfig, error) {
	path := filepath.Join(dir, "tools.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read tools file: %w", err)
	}

	var tools []ToolConfig
	if err := yaml.Unmarshal(data, &tools); err != nil {
		return nil, fmt.Errorf("failed to parse tools: %w", err)
	}

	return tools, nil
}

// CreateProjectScaffolding creates a new Guild project
func CreateProjectScaffolding(name, template string, offline bool) error {
	// Create project directory
	if err := os.MkdirAll(name, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create subdirectories
	dirs := []string{
		"objectives",
		"ai_docs",
		"ai_docs/api_docs",
		"ai_docs/integration_guides",
		"ai_docs/patterns",
	}

	for _, dir := range dirs {
		path := filepath.Join(name, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create initial guild.yaml
	config := Config{
		Agents: []AgentConfig{},
		Guilds: []GuildConfig{
			{
				Name:          name,
				Agents:        []string{},
				ObjectivesPath: "objectives",
			},
		},
		Costs: CostConfig{
			CLITools:    map[string]int{"default": 0},
			LocalModels: map[string]int{},
			APIModels:   map[string]int{},
		},
		APIKeys: map[string]string{},
	}

	// Add specific configurations based on template
	switch template {
	case "dev":
		// Development team template
		config.Agents = append(config.Agents,
			AgentConfig{
				Name:     "planner",
				Provider: "anthropic",
				Model:    "claude-3-opus",
				Tools:    []string{"tree2scaffold"},
			},
			AgentConfig{
				Name:     "implementer",
				Provider: "ollama",
				Model:    "llama3-8b",
				Tools:    []string{"aider"},
			},
		)
		config.Guilds[0].Agents = []string{"planner", "implementer"}
		config.Guilds[0].Manager = "planner"

	case "content":
		// Content creation template
		config.Agents = append(config.Agents,
			AgentConfig{
				Name:     "writer",
				Provider: "anthropic",
				Model:    "claude-3-sonnet",
				Tools:    []string{"file-writer"},
			},
		)
		config.Guilds[0].Agents = []string{"writer"}

	case "offline":
		// Offline-only template
		config.Agents = append(config.Agents,
			AgentConfig{
				Name:     "assistant",
				Provider: "ollama",
				Model:    "llama3-70b",
				Tools:    []string{"file-reader", "file-writer"},
			},
		)
		config.Guilds[0].Agents = []string{"assistant"}
		config.Costs.APIModels["default"] = 99999 // Discourage API usage

	default:
		// Basic template
		config.Agents = append(config.Agents,
			AgentConfig{
				Name:     "assistant",
				Provider: "anthropic",
				Model:    "claude-3-sonnet",
				Tools:    []string{"file-reader", "file-writer"},
			},
		)
		config.Guilds[0].Agents = []string{"assistant"}
	}

	// Override API models if offline mode requested
	if offline {
		config.Costs.APIModels["default"] = 99999 // Discourage API usage
	}

	// Save config
	configPath := filepath.Join(name, "guild.yaml")
	if err := SaveConfig(&config, configPath); err != nil {
		return err
	}

	// Create README.md
	readmePath := filepath.Join(name, "README.md")
	readmeContent := fmt.Sprintf("# %s\n\nA Guild project for AI agent collaboration.\n", name)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	// Create sample objective
	objectivePath := filepath.Join(name, "objectives", "README.md")
	objectiveContent := `# 🧠 Goal

Define the main objective for this Guild project.

# 📂 Context

Describe the background and purpose of this project.

# 🔧 Requirements

- List key requirements
- Define success criteria
- Specify constraints

# 📌 Tags

- guild
- project

# 🔗 Related

- None yet
`
	if err := os.WriteFile(objectivePath, []byte(objectiveContent), 0644); err != nil {
		return fmt.Errorf("failed to create sample objective: %w", err)
	}

	return nil
}
```

## Memory System

Enhancing your existing memory implementation to include BoltDB and vector store support:

```go
// pkg/memory/interface.go
package memory

import (
	"context"
	"time"
)

// Store defines the interface for memory storage
type Store interface {
	// SavePromptChain persists a prompt chain
	SavePromptChain(ctx context.Context, chain PromptChain) error

	// GetPromptChain retrieves a prompt chain by ID
	GetPromptChain(ctx context.Context, id string) (PromptChain, error)

	// GetPromptChainsByTask retrieves prompt chains by task ID
	GetPromptChainsByTask(ctx context.Context, taskID string) ([]PromptChain, error)

	// GetPromptChainsByAgent retrieves prompt chains by agent ID
	GetPromptChainsByAgent(ctx context.Context, agentID string) ([]PromptChain, error)

	// SaveEmbedding stores a vector embedding
	SaveEmbedding(ctx context.Context, embedding Embedding) error

	// QueryEmbeddings performs a similarity search
	QueryEmbeddings(ctx context.Context, query string, limit int) ([]EmbeddingMatch, error)

	// Close closes the store
	Close() error
}

// PromptChain represents a sequence of prompt-response pairs
type PromptChain struct {
	// ID is the unique identifier
	ID string

	// TaskID is the associated task
	TaskID string

	// AgentID is the agent that generated this chain
	AgentID string

	// Entries is the list of prompt-response pairs
	Entries []PromptEntry

	// CreatedAt is when the chain was created
	CreatedAt time.Time

	// UpdatedAt is when the chain was last updated
	UpdatedAt time.Time

	// Tags are searchable labels
	Tags []string
}

// PromptEntry represents a single prompt-response pair
type PromptEntry struct {
	// ID is the unique identifier
	ID string

	// Prompt is the input to the LLM
	Prompt string

	// Response is the output from the LLM
	Response string

	// TokensUsed is the total tokens used
	TokensUsed int

	// ToolsUsed is the list of tools used
	ToolsUsed []string

	// Timestamp is when this entry was created
	Timestamp time.Time

	// Cost is the estimated cost of this interaction
	Cost float64

	// Metadata contains additional information
	Metadata map[string]interface{}
}

// Embedding represents a vector embedding
type Embedding struct {
	// ID is the unique identifier
	ID string

	// Text is the source text
	Text string

	// Vector is the embedding vector
	Vector []float32

	// Source is where this embedding came from
	Source string

	// Metadata contains additional information
	Metadata map[string]interface{}

	// Timestamp is when this embedding was created
	Timestamp time.Time
}

// EmbeddingMatch represents a similarity match
type EmbeddingMatch struct {
	// ID is the matched embedding ID
	ID string

	// Text is the matched text
	Text string

	// Score is the similarity score
	Score float32

	// Source is where this embedding came from
	Source string

	// Metadata contains additional information
	Metadata map[string]interface{}
}
```

### BoltDB Implementation

```go
// pkg/memory/boltdb/store.go
package boltdb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
	"github.com/your-username/guild/pkg/memory"
)

var (
	// Bucket names
	promptChainBucket = []byte("prompt_chains")
	taskBucket        = []byte("tasks")
	agentBucket       = []byte("agents")
	objectiveBucket   = []byte("objectives")
)

// Store implements memory.Store using BoltDB
type Store struct {
	db *bolt.DB
}

// NewStore creates a new BoltDB store
func NewStore(path string) (*Store, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open BoltDB: %w", err)
	}

	// Create buckets if they don't exist
	err = db.Update(func(tx *bolt.Tx) error {
		buckets := [][]byte{
			promptChainBucket,
			taskBucket,
			agentBucket,
			objectiveBucket,
		}

		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists(bucket)
			if err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

// SavePromptChain persists a prompt chain
func (s *Store) SavePromptChain(ctx context.Context, chain memory.PromptChain) error {
	// Generate ID if not provided
	if chain.ID == "" {
		chain.ID = uuid.New().String()
	}

	// Set timestamps if not provided
	now := time.Now()
	if chain.CreatedAt.IsZero() {
		chain.CreatedAt = now
	}
	chain.UpdatedAt = now

	// Marshal to JSON
	data, err := json.Marshal(chain)
	if err != nil {
		return fmt.Errorf("failed to marshal prompt chain: %w", err)
	}

	// Save to BoltDB
	err = s.db.Update(func(tx *bolt.Tx) error {
		// Save to prompt chain bucket
		b := tx.Bucket(promptChainBucket)
		err := b.Put([]byte(chain.ID), data)
		if err != nil {
			return err
		}

		// Index by task ID
		if chain.TaskID != "" {
			tb := tx.Bucket(taskBucket)
			taskKey := []byte(fmt.Sprintf("task:%s:chain:%s", chain.TaskID, chain.ID))
			err = tb.Put(taskKey, []byte(chain.ID))
			if err != nil {
				return err
			}
		}

		// Index by agent ID
		if chain.AgentID != "" {
			ab := tx.Bucket(agentBucket)
			agentKey := []byte(fmt.Sprintf("agent:%s:chain:%s", chain.AgentID, chain.ID))
			err = ab.Put(agentKey, []byte(chain.ID))
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

// GetPromptChain retrieves a prompt chain by ID
func (s *Store) GetPromptChain(ctx context.Context, id string) (memory.PromptChain, error) {
	var chain memory.PromptChain

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(promptChainBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("prompt chain not found: %s", id)
		}

		return json.Unmarshal(data, &chain)
	})

	return chain, err
}

// GetPromptChainsByTask retrieves prompt chains by task ID
func (s *Store) GetPromptChainsByTask(ctx context.Context, taskID string) ([]memory.PromptChain, error) {
	var chains []memory.PromptChain

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(taskBucket)
		prefix := []byte(fmt.Sprintf("task:%s:chain:", taskID))

		// Collect chain IDs
		var chainIDs []string
		c := b.Cursor()
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			chainIDs = append(chainIDs, string(v))
		}

		// Fetch chains
		cb := tx.Bucket(promptChainBucket)
		for _, id := range chainIDs {
			data := cb.Get([]byte(id))
			if data == nil {
				continue
			}

			var chain memory.PromptChain
			if err := json.Unmarshal(data, &chain); err != nil {
				return err
			}

			chains = append(chains, chain)
		}

		return nil
	})

	return chains, err
}

// GetPromptChainsByAgent retrieves prompt chains by agent ID
func (s *Store) GetPromptChainsByAgent(ctx context.Context, agentID string) ([]memory.PromptChain, error) {
	// Similar implementation to GetPromptChainsByTask
	// ...

	return nil, nil // Placeholder
}

// SaveEmbedding stores a vector embedding (not supported in BoltDB)
func (s *Store) SaveEmbedding(ctx context.Context, embedding memory.Embedding) error {
	return fmt.Errorf("vector embeddings not supported in BoltDB")
}

// QueryEmbeddings performs a similarity search (not supported in BoltDB)
func (s *Store) QueryEmbeddings(ctx context.Context, query string, limit int) ([]memory.EmbeddingMatch, error) {
	return nil, fmt.Errorf("vector embeddings not supported in BoltDB")
}

// Close closes the store
func (s *Store) Close() error {
	return s.db.Close()
}
```

### Vector Store Integration

```go
// pkg/memory/vector/qdrant.go
package vector

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	qdrant "github.com/qdrant/go-client/qdrant"
	"github.com/your-username/guild/pkg/memory"
)

// QdrantStore implements memory.Store for vector embeddings
type QdrantStore struct {
	client       *qdrant.QdrantClient
	collectionName string
	vectorSize  int
	embedder    Embedder
}

// Embedder generates embeddings from text
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// NewQdrantStore creates a new Qdrant vector store
func NewQdrantStore(address string, collectionName string, vectorSize int, embedder Embedder) (*QdrantStore, error) {
	// Create Qdrant client
	client, err := qdrant.NewQdrantClient(address, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	// Create collection if it doesn't exist
	_, err = client.CreateCollection(context.Background(), &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Size:     uint64(vectorSize),
			Distance: qdrant.Distance_Cosine,
		},
	})

	// Ignore "already exists" error
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	return &QdrantStore{
		client:       client,
		collectionName: collectionName,
		vectorSize:  vectorSize,
		embedder:    embedder,
	}, nil
}

// SaveEmbedding stores a vector embedding
func (s *QdrantStore) SaveEmbedding(ctx context.Context, embedding memory.Embedding) error {
	// Generate ID if not provided
	if embedding.ID == "" {
		embedding.ID = uuid.New().String()
	}

	// Convert metadata to Qdrant payload
	payload := make(map[string]*qdrant.Value)
	payload["text"] = &qdrant.Value{Value: &qdrant.Value_StringValue{StringValue: embedding.Text}}
	payload["source"] = &qdrant.Value{Value: &qdrant.Value_StringValue{StringValue: embedding.Source}}

	for k, v := range embedding.Metadata {
		switch val := v.(type) {
		case string:
			payload[k] = &qdrant.Value{Value: &qdrant.Value_StringValue{StringValue: val}}
		case int:
			payload[k] = &qdrant.Value{Value: &qdrant.Value_IntegerValue{IntegerValue: int64(val)}}
		case float64:
			payload[k] = &qdrant.Value{Value: &qdrant.Value_DoubleValue{DoubleValue: val}}
		}
	}

	// If no vector provided, generate it
	vector := embedding.Vector
	if len(vector) == 0 {
		var err error
		vector, err = s.embedder.Embed(ctx, embedding.Text)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
	}

	// Add point to Qdrant
	_, err := s.client.UpsertPoints(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collectionName,
		Points: []*qdrant.PointStruct{
			{
				Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: embedding.ID}},
				Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: vector}}},
				Payload: payload,
			},
		},
	})

	return err
}

// QueryEmbeddings performs a similarity search
func (s *QdrantStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]memory.EmbeddingMatch, error) {
	// Generate query embedding
	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Perform search
	result, err := s.client.Search(ctx, &qdrant.SearchPoints{
		CollectionName: s.collectionName,
		Vector:         vector,
		Limit:          uint64(limit),
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	// Convert results to EmbeddingMatch
	matches := make([]memory.EmbeddingMatch, 0, len(result.Result))
	for _, point := range result.Result {
		match := memory.EmbeddingMatch{
			ID:    point.Id.GetUuid(),
			Score: point.Score,
			Metadata: make(map[string]interface{}),
		}

		// Extract text and source from payload
		if textVal, ok := point.Payload["text"]; ok {
			match.Text = textVal.GetStringValue()
		}

		if sourceVal, ok := point.Payload["source"]; ok {
			match.Source = sourceVal.GetStringValue()
		}

		// Extract other metadata
		for k, v := range point.Payload {
			if k == "text" || k == "source" {
				continue
			}

			switch {
			case v.GetStringValue() != "":
				match.Metadata[k] = v.GetStringValue()
			case v.GetIntegerValue() != 0:
				match.Metadata[k] = v.GetIntegerValue()
			case v.GetDoubleValue() != 0:
				match.Metadata[k] = v.GetDoubleValue()
			}
		}

		matches = append(matches, match)
	}

	return matches, nil
}

// Additional methods to implement memory.Store interface...
```

### Combined Memory Store

```go
// pkg/memory/combined/store.go
package combined

import (
	"context"

	"github.com/your-username/guild/pkg/memory"
	"github.com/your-username/guild/pkg/memory/boltdb"
	"github.com/your-username/guild/pkg/memory/vector"
)

// CombinedStore implements memory.Store with separate backends
type CombinedStore struct {
	symbolStore memory.Store // BoltDB for symbolic data
	vectorStore memory.Store // Qdrant for vector data
}

// NewCombinedStore creates a new combined store
func NewCombinedStore(boltPath, qdrantAddress, collectionName string, vectorSize int, embedder vector.Embedder) (*CombinedStore, error) {
	// Create BoltDB store
	symbolStore, err := boltdb.NewStore(boltPath)
	if err != nil {
		return nil, err
	}

	// Create Qdrant store
	vectorStore, err := vector.NewQdrantStore(qdrantAddress, collectionName, vectorSize, embedder)
	if err != nil {
		return nil, err
	}

	return &CombinedStore{
		symbolStore: symbolStore,
		vectorStore: vectorStore,
	}, nil
}

// Forward symbolic operations to BoltDB
func (s *CombinedStore) SavePromptChain(ctx context.Context, chain memory.PromptChain) error {
	return s.symbolStore.SavePromptChain(ctx, chain)
}

func (s *CombinedStore) GetPromptChain(ctx context.Context, id string) (memory.PromptChain, error) {
	return s.symbolStore.GetPromptChain(ctx, id)
}

func (s *CombinedStore) GetPromptChainsByTask(ctx context.Context, taskID string) ([]memory.PromptChain, error) {
	return s.symbolStore.GetPromptChainsByTask(ctx, taskID)
}

func (s *CombinedStore) GetPromptChainsByAgent(ctx context.Context, agentID string) ([]memory.PromptChain, error) {
	return s.symbolStore.GetPromptChainsByAgent(ctx, agentID)
}

// Forward vector operations to Qdrant
func (s *CombinedStore) SaveEmbedding(ctx context.Context, embedding memory.Embedding) error {
	return s.vectorStore.SaveEmbedding(ctx, embedding)
}

func (s *CombinedStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]memory.EmbeddingMatch, error) {
	return s.vectorStore.QueryEmbeddings(ctx, query, limit)
}

// Close closes both stores
func (s *CombinedStore) Close() error {
	errSymbol := s.symbolStore.Close()
	errVector := s.vectorStore.Close()

	if errSymbol != nil {
		return errSymbol
	}
	return errVector
}
```

### RAG Implementation

```go
// pkg/memory/rag/retriever.go
package rag

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/your-username/guild/pkg/memory"
)

// Retriever provides retrieval-augmented generation capabilities
type Retriever struct {
	store memory.Store
}

// NewRetriever creates a new RAG retriever
func NewRetriever(store memory.Store) *Retriever {
	return &Retriever{
		store: store,
	}
}

// RetrieveContext gets relevant context for a prompt
func (r *Retriever) RetrieveContext(ctx context.Context, query string, taskID string, agentID string, limit int) (string, error) {
	// Combine relevant information from multiple sources
	var contextParts []string

	// 1. Get relevant embeddings
	matches, err := r.store.QueryEmbeddings(ctx, query, limit)
	if err == nil && len(matches) > 0 {
		// Add vector matches to context
		for _, match := range matches {
			contextParts = append(contextParts, fmt.Sprintf("REFERENCE [%s] (score: %.2f):\n%s",
				match.Source, match.Score, match.Text))
		}
	}

	// 2. Get previous prompt chains for this task
	if taskID != "" {
		chains, err := r.store.GetPromptChainsByTask(ctx, taskID)
		if err == nil && len(chains) > 0 {
			// Sort by creation date
			sort.Slice(chains, func(i, j int) bool {
				return chains[i].CreatedAt.Before(chains[j].CreatedAt)
			})

			// Add the most recent chains (up to 2)
			maxChains := min(2, len(chains))
			for i := len(chains) - maxChains; i < len(chains); i++ {
				chain := chains[i]

				// Only include the last 3 entries maximum
				entriesOffset := max(0, len(chain.Entries)-3)
				for j := entriesOffset; j < len(chain.Entries); j++ {
					entry := chain.Entries[j]
					contextParts = append(contextParts, fmt.Sprintf("PREVIOUS EXCHANGE:\nPrompt: %s\nResponse: %s",
						truncateText(entry.Prompt, 200), truncateText(entry.Response, 500)))
				}
			}
		}
	}

	// 3. Get agent's recent prompt chains for similar tasks
	if agentID != "" && taskID != "" {
		chains, err := r.store.GetPromptChainsByAgent(ctx, agentID)
		if err == nil && len(chains) > 0 {
			var relevantChains []memory.PromptChain

			// Filter out the current task
			for _, chain := range chains {
				if chain.TaskID != taskID {
					relevantChains = append(relevantChains, chain)
				}
			}

			// Sort by creation date
			sort.Slice(relevantChains, func(i, j int) bool {
				return relevantChains[i].CreatedAt.After(relevantChains[j].CreatedAt)
			})

			// Add the most recent relevant chain
			if len(relevantChains) > 0 {
				chain := relevantChains[0]
				lastEntry := chain.Entries[len(chain.Entries)-1]
				contextParts = append(contextParts, fmt.Sprintf("RELATED TASK [%s]:\nFinal Response: %s",
					chain.TaskID, truncateText(lastEntry.Response, 300)))
			}
		}
	}

	// Combine all parts with separators
	return strings.Join(contextParts, "\n\n"), nil
}

// Helper functions
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
```

## Provider Interface

Based on your existing `pkg/providers` structure, let's formalize the provider interface:

```go
// pkg/providers/interface.go
package providers

import (
	"context"
)

// ProviderType identifies the LLM provider
type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderDeepseek  ProviderType = "deepseek"
	ProviderOra       ProviderType = "ora"
	ProviderOllama    ProviderType = "ollama"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Type returns the provider type
	Type() ProviderType

	// Models returns the available models
	Models() []string

	// Generate produces text from a prompt
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)

	// GenerateStream produces a stream of tokens
	GenerateStream(ctx context.Context, req GenerateRequest) (<-chan GenerateResponseChunk, error)

	// IsLocal returns true if this is a local provider
	IsLocal() bool

	// Cost returns the estimated cost for a request
	Cost(req GenerateRequest) float64
}

// GenerateRequest contains parameters for text generation
type GenerateRequest struct {
	// Model is the specific model to use
	Model string

	// Prompt is the input text
	Prompt string

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int

	// Temperature controls randomness (0.0-1.0)
	Temperature float64

	// SystemPrompt is an optional system instruction
	SystemPrompt string

	// StopSequences are strings that stop generation
	StopSequences []string

	// AdditionalParams contains provider-specific parameters
	AdditionalParams map[string]interface{}
}

// GenerateResponse contains the result of text generation
type GenerateResponse struct {
	// Text is the generated text
	Text string

	// TokensUsed is the total tokens consumed
	TokensUsed int

	// FinishReason explains why generation stopped
	FinishReason string

	// Raw contains the raw provider response
	Raw interface{}
}

// GenerateResponseChunk contains a partial response
type GenerateResponseChunk struct {
	// Text is the generated text chunk
	Text string

	// IsFinal indicates the last chunk
	IsFinal bool

	// Error contains any error that occurred
	Error error
}

// ProviderConfig defines configuration for a provider
type ProviderConfig struct {
	// Type is the provider type
	Type ProviderType

	// Model is the specific model to use
	Model string

	// APIKey is the authentication key
	APIKey string

	// BaseURL is an optional custom endpoint
	BaseURL string

	// AdditionalParams contains provider-specific parameters
	AdditionalParams map[string]interface{}
}

// Factory creates providers
type Factory interface {
	// Create returns a provider for a configuration
	Create(config ProviderConfig) (Provider, error)

	// CreateWithAPIKey returns a provider with an API key
	CreateWithAPIKey(providerType ProviderType, model string, apiKey string) (Provider, error)
}
```

## Next Steps

With these core interfaces defined, you can proceed to implement the concrete types that fulfill them. Here's a suggested roadmap:

1. **Complete the Provider Implementations**

   - Implement the Provider interface for each LLM backend
   - Create a registry for provider factories

2. **Build the Objective Parser**

   - Implement the markdown parser for objectives
   - Create templates for common objective types

3. **Implement the Kanban System**

   - Complete the board and task management
   - Add persistence with BoltDB
   - Implement the event system with ZeroMQ

4. **Develop the Agent Implementations**

   - Create agent types with different specializations
   - Implement prompt construction and context loading
   - Add tool discovery and invocation

5. **Build the Orchestration Layer**

   - Implement the task distribution logic
   - Add coordination between agents
   - Create the manager agent for oversight

6. **Implement the Command-Line Interface**
   - Complete the CLI commands
   - Add interactive prompting
   - Implement the dashboard

The goal is to create a minimal viable product where an agent can:

1. Parse objectives from markdown files
2. Break them down into tasks
3. Execute tasks using an LLM
4. Use tools when appropriate
5. Store results in the Kanban board
6. Allow for human intervention when needed

## Testing Your Implementation

Create comprehensive tests for each component:

```go
// Example test for the agent interface
func TestAgentExecution(t *testing.T) {
	// Setup test environment
	ctx := context.Background()
	mockStore := memory.NewMockStore()
	mockBoard := kanban.NewMockBoard()

	// Create a mock provider
	mockProvider := providers.NewMockProvider("test", []string{"test-model"})

	// Create the agent
	agent := agent.NewBasicAgent(agent.Config{
		ID:   "test-agent",
		Name: "Test Agent",
		ProviderConfig: providers.ProviderConfig{
			Type:  "mock",
			Model: "test-model",
		},
	}, mockProvider, mockBoard, mockStore)

	// Create a test task
	task := kanban.Task{
		ID:          "task-1",
		Title:       "Test Task",
		Description: "Generate a hello world program in Go",
		Status:      kanban.StatusToDo,
		AgentID:     "test-agent",
	}

	// Execute the task
	result, err := agent.Execute(ctx, task)

	// Verify the result
	if err != nil {
		t.Fatalf("Failed to execute task: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected task to succeed, but it failed: %s", result.Error)
	}

	if !strings.Contains(result.Output, "package main") {
		t.Errorf("Expected Go program in output, got: %s", result.Output)
	}
}
```

## Conclusion

This guide provides a foundation for building the Guild framework. The core interfaces define clear boundaries between components, making it easier to implement, test, and extend the system.

Remember these key principles:

1. **Interface-First Design**: Define clear interfaces before implementing concrete types
2. **Context Propagation**: Use `context.Context` for cancellation and timeouts
3. **Error Handling**: Provide clear, actionable error messages
4. **Modularity**: Keep components loosely coupled through interfaces
5. **Testing**: Write tests for each component in isolation

As you implement Guild, focus on creating a usable MVP first, then add more advanced features like the Meta-Coordination Protocol (MCP), advanced RAG, and the Git workflow.

## Next Steps

With these core interfaces defined, we can begin implementing the concrete types that fulfill them:

1. **Agent implementations** - Create specific agent types (Planner, Implementer, Reviewer, etc.)
2. **Guild coordination** - Implement the logic for running a guild with multiple agents
3. **Tool wrappers** - Build wrappers for common tools (CLI, HTTP, file system, etc.)
4. **Persistence layer** - Implement BoltDB storage for tasks and prompt chains
5. **Vector store** - Set up Qdrant for semantic search of past contexts
6. **CLI commands** - Complete the command-line interface for managing guilds

The goal is to create a minimal working example where a guild of agents can complete a simple task, such as generating a basic "Hello World" application, with proper task tracking and coordination.

## Testing Your Implementation

Create simple test cases for each component:

```go
// pkg/agent/agent_test.go
package agent_test

import (
	"context"
	"testing"

	"github.com/your-username/guild/pkg/agent"
	"github.com/your-username/guild/pkg/kanban"
)

func TestAgentExecution(t *testing.T) {
	// Create a mock agent
	mockAgent := &MockAgent{
		id: "test-agent",
		tools: []tools.Tool{
			&MockTool{id: "echo", name: "Echo"},
		},
		model: agent.ModelConfig{
			Provider: "mock",
			Name:     "test-model",
			Cost:     1,
			Local:    true,
		},
	}

	// Create a test task
	task := kanban.Task{
		ID:          "task-1",
		Title:       "Test Task",
		Description: "This is a test task",
		Status:      kanban.StatusToDo,
		AgentID:     mockAgent.ID(),
	}

	// Execute the task
	result, err := mockAgent.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("Failed to execute task: %v", err)
	}

	// Verify the result
	if !result.Success {
		t.Errorf("Expected task to succeed, but it failed: %s", result.Error)
	}
}

// Mock implementations...
```

## Running Your First Guild

Once you have implemented the basic functionality, you can create your first guild:

```bash
# Initialize a new project
guild init hello-world

# Add agents
guild add-agent planner --model claude-3-opus
guild add-agent implementer --model ollama:llama3-8b --tools echo,file-write

# Add an objective
guild add-objective --title "Create Hello World App" --description "Generate a simple Hello World application in Go"

# Run the guild
guild run hello-world
```

This should:

1. Create a new project with a configuration file
2. Add two agents with different models and tools
3. Add a simple objective
4. Run the guild, which should:
   - Break down the objective into tasks
   - Assign tasks to agents
   - Execute the tasks
   - Update the Kanban board
   - Generate a simple Hello World application

The output should include the generated code and a summary of the tasks completed.
