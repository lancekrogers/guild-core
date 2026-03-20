// Package services provides service wrappers for Guild components
package services

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/agents"
	"github.com/lancekrogers/guild-core/pkg/agents/backstory"
	"github.com/lancekrogers/guild-core/pkg/config"
	"github.com/lancekrogers/guild-core/pkg/events"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/observability"
	"github.com/lancekrogers/guild-core/pkg/prompts/layered"
	"github.com/lancekrogers/guild-core/pkg/registry"
)

// AgentManagerService wraps the enhanced agent manager to integrate with the service framework
type AgentManagerService struct {
	manager  agents.EnhancedAgentManager
	registry registry.ComponentRegistry
	eventBus events.EventBus
	logger   observability.Logger
	config   AgentManagerServiceConfig

	// Service state
	started bool
	mu      sync.RWMutex

	// Metrics
	agentsCreated     uint64
	agentsEnhanced    uint64
	guildsInitialized uint64
	avgCreationTime   time.Duration
}

// AgentManagerServiceConfig configures the agent manager service
type AgentManagerServiceConfig struct {
	DefaultProjectPath string
	EnableAutoElena    bool   // Automatically create Elena if missing
	BackstoryPath      string // Path to backstory files
}

// DefaultAgentManagerServiceConfig returns default configuration
func DefaultAgentManagerServiceConfig() AgentManagerServiceConfig {
	return AgentManagerServiceConfig{
		DefaultProjectPath: ".guild",
		EnableAutoElena:    true,
		BackstoryPath:      "backstories",
	}
}

// NewAgentManagerService creates a new agent manager service wrapper
func NewAgentManagerService(
	registry registry.ComponentRegistry,
	eventBus events.EventBus,
	logger observability.Logger,
	config AgentManagerServiceConfig,
) (*AgentManagerService, error) {
	if registry == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "registry cannot be nil", nil).
			WithComponent("AgentManagerService")
	}
	if eventBus == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "event bus cannot be nil", nil).
			WithComponent("AgentManagerService")
	}
	if logger == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "logger cannot be nil", nil).
			WithComponent("AgentManagerService")
	}

	// Create backstory manager
	// TODO: Get layered registry from main registry
	backstoryManager := backstory.NewBackstoryManager(nil)

	// Create enhanced agent manager
	// TODO: This should come from registry or be created with proper dependencies
	manager := &defaultEnhancedAgentManager{
		backstoryManager: backstoryManager,
		registry:         registry,
		logger:           logger,
	}

	return &AgentManagerService{
		manager:  manager,
		registry: registry,
		eventBus: eventBus,
		logger:   logger,
		config:   config,
	}, nil
}

// Name returns the service name
func (s *AgentManagerService) Name() string {
	return "agent-manager-service"
}

// Start initializes and starts the service
func (s *AgentManagerService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "service already started", nil).
			WithComponent("AgentManagerService")
	}

	s.started = true

	// Emit service started event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"agent-manager-service-started",
		"service.started",
		"agent-manager",
		map[string]interface{}{
			"auto_elena_enabled": s.config.EnableAutoElena,
			"backstory_path":     s.config.BackstoryPath,
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service started event", "error", err)
	}

	s.logger.InfoContext(ctx, "Agent manager service started",
		"auto_elena", s.config.EnableAutoElena)

	return nil
}

// Stop gracefully shuts down the service
func (s *AgentManagerService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("AgentManagerService")
	}

	// Emit service stopped event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"agent-manager-service-stopped",
		"service.stopped",
		"agent-manager",
		map[string]interface{}{
			"agents_created":     s.agentsCreated,
			"agents_enhanced":    s.agentsEnhanced,
			"guilds_initialized": s.guildsInitialized,
			"avg_creation_time":  s.avgCreationTime.Milliseconds(),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service stopped event", "error", err)
	}

	s.started = false

	s.logger.InfoContext(ctx, "Agent manager service stopped",
		"total_agents_created", s.agentsCreated)

	return nil
}

// Health checks if the service is healthy
func (s *AgentManagerService) Health(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeResourceExhausted, "service not started", nil).
			WithComponent("AgentManagerService")
	}

	return nil
}

// Ready checks if the service is ready to handle requests
func (s *AgentManagerService) Ready(ctx context.Context) error {
	return s.Health(ctx)
}

// CreateElenaGuildMaster creates Elena with event emission
func (s *AgentManagerService) CreateElenaGuildMaster(ctx context.Context) (*config.AgentConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil, gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("AgentManagerService")
	}

	start := time.Now()

	// Create Elena
	elena, err := s.manager.CreateElenaGuildMaster(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create Elena").
			WithComponent("AgentManagerService")
	}

	duration := time.Since(start)
	s.updateMetrics(duration)
	s.agentsCreated++

	// Emit agent created event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"agent-elena-created",
		"agent.created",
		"agent-manager",
		map[string]interface{}{
			"agent_id":      elena.ID,
			"agent_type":    elena.Type,
			"provider":      elena.Provider,
			"model":         elena.Model,
			"creation_time": duration.Milliseconds(),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish agent created event", "error", err)
	}

	s.logger.InfoContext(ctx, "Elena Guild Master created",
		"agent_id", elena.ID,
		"duration", duration)

	return elena, nil
}

// InitializeDefaultAgents initializes default agents with event emission
func (s *AgentManagerService) InitializeDefaultAgents(ctx context.Context, projectPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("AgentManagerService")
	}

	if projectPath == "" {
		projectPath = s.config.DefaultProjectPath
	}

	start := time.Now()

	// Initialize agents
	if err := s.manager.InitializeDefaultAgents(ctx, projectPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize agents").
			WithComponent("AgentManagerService")
	}

	duration := time.Since(start)
	s.guildsInitialized++

	// Emit initialization complete event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"agents-initialized",
		"agents.initialized",
		"agent-manager",
		map[string]interface{}{
			"project_path": projectPath,
			"duration":     duration.Milliseconds(),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish agents initialized event", "error", err)
	}

	s.logger.InfoContext(ctx, "Default agents initialized",
		"project_path", projectPath,
		"duration", duration)

	return nil
}

// EnhanceExistingAgent enhances an agent with event emission
func (s *AgentManagerService) EnhanceExistingAgent(
	ctx context.Context,
	agentID, specialistTemplate string,
	guildConfig *config.GuildConfig,
	projectPath string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("AgentManagerService")
	}

	start := time.Now()

	// Enhance the agent
	if err := s.manager.EnhanceExistingAgent(ctx, agentID, specialistTemplate, guildConfig, projectPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to enhance agent").
			WithComponent("AgentManagerService").
			WithDetails("agent_id", agentID).
			WithDetails("template", specialistTemplate)
	}

	duration := time.Since(start)
	s.agentsEnhanced++

	// Emit agent enhanced event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"agent-"+agentID+"-enhanced",
		"agent.enhanced",
		"agent-manager",
		map[string]interface{}{
			"agent_id":            agentID,
			"specialist_template": specialistTemplate,
			"duration":            duration.Milliseconds(),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish agent enhanced event", "error", err)
	}

	s.logger.InfoContext(ctx, "Agent enhanced",
		"agent_id", agentID,
		"template", specialistTemplate,
		"duration", duration)

	return nil
}

// GetManager returns the wrapped agent manager (for compatibility)
func (s *AgentManagerService) GetManager() agents.EnhancedAgentManager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.manager
}

// updateMetrics updates service metrics
func (s *AgentManagerService) updateMetrics(duration time.Duration) {
	// Simple moving average for creation time
	if s.avgCreationTime == 0 {
		s.avgCreationTime = duration
	} else {
		s.avgCreationTime = (s.avgCreationTime*9 + duration) / 10
	}
}

// defaultEnhancedAgentManager is a basic implementation of EnhancedAgentManager
// In production, this would be replaced with the actual implementation from the agents package
type defaultEnhancedAgentManager struct {
	backstoryManager *backstory.BackstoryManager
	registry         registry.ComponentRegistry
	logger           observability.Logger
}

// Implement all required methods with basic functionality
// These are placeholders - in production, use the actual implementation

func (m *defaultEnhancedAgentManager) CreateElenaGuildMaster(ctx context.Context) (*config.AgentConfig, error) {
	return &config.AgentConfig{
		ID:       "elena-guild-master",
		Name:     "Elena",
		Type:     "manager",
		Provider: "anthropic",
		Model:    "claude-3-opus-20240229",
		Capabilities: []string{
			"task_decomposition",
			"agent_coordination",
			"quality_assurance",
		},
		CostMagnitude: 4,
	}, nil
}

func (m *defaultEnhancedAgentManager) CreateDefaultDeveloper(ctx context.Context) (*config.AgentConfig, error) {
	return &config.AgentConfig{
		ID:       "default-developer",
		Name:     "Developer",
		Type:     "worker",
		Provider: "openai",
		Model:    "gpt-4",
		Capabilities: []string{
			"code_generation",
			"debugging",
			"refactoring",
		},
		CostMagnitude: 3,
	}, nil
}

func (m *defaultEnhancedAgentManager) CreateDefaultTester(ctx context.Context) (*config.AgentConfig, error) {
	return &config.AgentConfig{
		ID:       "default-tester",
		Name:     "Tester",
		Type:     "specialist",
		Provider: "openai",
		Model:    "gpt-3.5-turbo",
		Capabilities: []string{
			"test_generation",
			"test_execution",
			"bug_reporting",
		},
		CostMagnitude: 2,
	}, nil
}

func (m *defaultEnhancedAgentManager) CreateDefaultAgentSet(ctx context.Context) ([]*config.AgentConfig, error) {
	elena, _ := m.CreateElenaGuildMaster(ctx)
	dev, _ := m.CreateDefaultDeveloper(ctx)
	tester, _ := m.CreateDefaultTester(ctx)
	return []*config.AgentConfig{elena, dev, tester}, nil
}

func (m *defaultEnhancedAgentManager) GetOptimalProvider(agentType, agentID string) string {
	// Simple logic - in production, this would be more sophisticated
	if agentType == "manager" {
		return "anthropic"
	}
	return "openai"
}

func (m *defaultEnhancedAgentManager) GetSpecialistTemplate(specialistID string) (*config.AgentConfig, error) {
	// Placeholder implementation
	return nil, gerror.New(gerror.ErrCodeNotFound, "specialist template not found", nil).
		WithComponent("defaultEnhancedAgentManager").
		WithDetails("specialist_id", specialistID)
}

func (m *defaultEnhancedAgentManager) ListAvailableSpecialists() []string {
	return []string{"developer", "tester", "architect", "security-specialist"}
}

func (m *defaultEnhancedAgentManager) EnhanceAgentWithBackstory(ctx context.Context, agent *config.AgentConfig, backstoryID string) error {
	// Placeholder - would use backstory manager
	return nil
}

func (m *defaultEnhancedAgentManager) InitializeDefaultAgents(ctx context.Context, projectPath string) error {
	// Placeholder - would create and save agents
	return nil
}

func (m *defaultEnhancedAgentManager) LoadAndEnhanceAgents(ctx context.Context, guildConfig *config.GuildConfig) error {
	// Placeholder - would load and enhance agents
	return nil
}

func (m *defaultEnhancedAgentManager) CreateElenaIfMissing(ctx context.Context, guildConfig *config.GuildConfig, projectPath string) error {
	// Check if Elena exists
	for _, agent := range guildConfig.Agents {
		if agent.ID == "elena-guild-master" {
			return nil
		}
	}

	// Create Elena
	elena, err := m.CreateElenaGuildMaster(ctx)
	if err != nil {
		return err
	}

	guildConfig.Agents = append(guildConfig.Agents, *elena)
	return nil
}

func (m *defaultEnhancedAgentManager) EnhanceExistingAgent(ctx context.Context, agentID, specialistTemplate string, guildConfig *config.GuildConfig, projectPath string) error {
	// Placeholder - would enhance agent with template
	return nil
}

func (m *defaultEnhancedAgentManager) GetBackstoryManager() *backstory.BackstoryManager {
	return m.backstoryManager
}

func (m *defaultEnhancedAgentManager) GetAvailableSpecialists() []string {
	return m.ListAvailableSpecialists()
}

func (m *defaultEnhancedAgentManager) GeneratePersonalityPrompt(ctx context.Context, agentID, basePrompt string, turnContext *layered.TurnContext) (string, error) {
	// Placeholder - would generate enhanced prompt
	return basePrompt + "\n\n[Enhanced with personality traits]", nil
}

func (m *defaultEnhancedAgentManager) CreateGuildConfigWithElena(ctx context.Context, guildName string) (*config.GuildConfig, error) {
	elena, err := m.CreateElenaGuildMaster(ctx)
	if err != nil {
		return nil, err
	}

	return &config.GuildConfig{
		Name:        guildName,
		Description: "Guild managed by Elena",
		Manager: config.ManagerConfig{
			Default: elena.ID,
		},
		Agents: []config.AgentConfig{*elena},
	}, nil
}

func (m *defaultEnhancedAgentManager) UpgradeExistingGuild(ctx context.Context, guildConfig *config.GuildConfig, projectPath string) error {
	// Ensure Elena exists
	return m.CreateElenaIfMissing(ctx, guildConfig, projectPath)
}
