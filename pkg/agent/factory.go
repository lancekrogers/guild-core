package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/blockhead-consulting/Guild/pkg/kanban"
	"github.com/blockhead-consulting/Guild/pkg/memory"
	"github.com/blockhead-consulting/Guild/pkg/objective"
	"github.com/blockhead-consulting/Guild/pkg/providers"
	"github.com/blockhead-consulting/Guild/tools"
	"github.com/google/uuid"
)

// Factory is responsible for creating and managing agents
type Factory struct {
	providerFactory *providers.Factory
	memoryManager   memory.ChainManager
	toolRegistry    *tools.ToolRegistry
	objectiveManager *objective.Manager
	kanbanManager   *kanban.Manager
	agents          map[string]Agent
	configDir       string
	mu              sync.RWMutex
}

// NewFactory creates a new agent factory
func NewFactory(
	providerFactory *providers.Factory,
	memoryManager memory.ChainManager,
	toolRegistry *tools.ToolRegistry,
	objectiveManager *objective.Manager,
	kanbanManager *kanban.Manager,
	configDir string,
) (*Factory, error) {
	// Create the config directory if it doesn't exist
	if configDir != "" {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	return &Factory{
		providerFactory: providerFactory,
		memoryManager:   memoryManager,
		toolRegistry:    toolRegistry,
		objectiveManager: objectiveManager,
		kanbanManager:   kanbanManager,
		agents:          make(map[string]Agent),
		configDir:       configDir,
	}, nil
}

// CreateWorkerAgent creates a new worker agent
func (f *Factory) CreateWorkerAgent(ctx context.Context, config *AgentConfig) (Agent, error) {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	// Get the LLM client
	llmClient, err := f.providerFactory.GetClient(config.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM client: %w", err)
	}

	// Create the agent
	agent := NewWorkerAgent(
		config,
		llmClient,
		f.memoryManager,
		f.toolRegistry,
		f.objectiveManager,
	)

	// Store the agent
	f.mu.Lock()
	f.agents[config.ID] = agent
	f.mu.Unlock()

	// Save the agent configuration
	if f.configDir != "" {
		if err := f.saveAgentConfig(config); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to save agent config: %v\n", err)
		}
	}

	return agent, nil
}

// CreateManagerAgent creates a new manager agent
func (f *Factory) CreateManagerAgent(ctx context.Context, config *AgentConfig) (*ManagerAgent, error) {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	// Ensure the type is set to "manager"
	config.Type = "manager"

	// Get the LLM client
	llmClient, err := f.providerFactory.GetClient(config.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM client: %w", err)
	}

	// Create the agent
	agent := NewManagerAgent(
		config,
		llmClient,
		f.memoryManager,
		f.toolRegistry,
		f.objectiveManager,
		f.kanbanManager,
	)

	// Store the agent
	f.mu.Lock()
	f.agents[config.ID] = agent
	f.mu.Unlock()

	// Save the agent configuration
	if f.configDir != "" {
		if err := f.saveAgentConfig(config); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to save agent config: %v\n", err)
		}
	}

	return agent, nil
}

// GetAgent returns an agent by ID
func (f *Factory) GetAgent(id string) (Agent, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	agent, exists := f.agents[id]
	return agent, exists
}

// ListAgents returns a list of all agents
func (f *Factory) ListAgents() []Agent {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var agents []Agent
	for _, agent := range f.agents {
		agents = append(agents, agent)
	}

	return agents
}

// RegisterAgent registers an existing agent with the factory
func (f *Factory) RegisterAgent(agent Agent) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.agents[agent.ID()] = agent
}

// LoadAgents loads agent configurations from the config directory
func (f *Factory) LoadAgents(ctx context.Context) error {
	if f.configDir == "" {
		return nil // No config directory, nothing to load
	}

	// Read the config directory
	files, err := os.ReadDir(f.configDir)
	if err != nil {
		return fmt.Errorf("failed to read config directory: %w", err)
	}

	// Load each config file
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		configPath := filepath.Join(f.configDir, file.Name())
		config, err := f.loadAgentConfig(configPath)
		if err != nil {
			// Log but continue
			fmt.Printf("Warning: failed to load agent config from %s: %v\n", configPath, err)
			continue
		}

		// Create the appropriate agent type
		var agent Agent
		switch config.Type {
		case "worker":
			agent, err = f.CreateWorkerAgent(ctx, config)
		case "manager":
			agent, err = f.CreateManagerAgent(ctx, config)
		default:
			err = fmt.Errorf("unknown agent type: %s", config.Type)
		}

		if err != nil {
			// Log but continue
			fmt.Printf("Warning: failed to create agent from config %s: %v\n", configPath, err)
			continue
		}

		// Agent is already registered by the create methods
		fmt.Printf("Loaded agent %s (%s) from %s\n", agent.Name(), agent.ID(), configPath)
	}

	return nil
}

// saveAgentConfig saves an agent configuration to disk
func (f *Factory) saveAgentConfig(config *AgentConfig) error {
	if f.configDir == "" {
		return fmt.Errorf("config directory not set")
	}

	// Create the config file path
	configPath := filepath.Join(f.configDir, config.ID+".json")

	// Marshal the config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent config: %w", err)
	}

	// Write the config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write agent config: %w", err)
	}

	return nil
}

// loadAgentConfig loads an agent configuration from disk
func (f *Factory) loadAgentConfig(configPath string) (*AgentConfig, error) {
	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent config: %w", err)
	}

	// Unmarshal the config
	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent config: %w", err)
	}

	return &config, nil
}

// CreateAgentFromConfig creates an agent from a configuration file
func (f *Factory) CreateAgentFromConfig(ctx context.Context, configPath string) (Agent, error) {
	// Load the config
	config, err := f.loadAgentConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load agent config: %w", err)
	}

	// Create the appropriate agent type
	switch config.Type {
	case "worker":
		return f.CreateWorkerAgent(ctx, config)
	case "manager":
		return f.CreateManagerAgent(ctx, config)
	default:
		return nil, fmt.Errorf("unknown agent type: %s", config.Type)
	}
}