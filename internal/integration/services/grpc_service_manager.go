// Package services provides service wrappers for Guild components
package services

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	pb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	promptspb "github.com/lancekrogers/guild/pkg/grpc/pb/prompts/v1"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/registry"
)

// GRPCServiceManager manages all gRPC service registrations
type GRPCServiceManager struct {
	registry   registry.ComponentRegistry
	eventBus   events.EventBus
	logger     observability.Logger
	config     GRPCServiceManagerConfig
	
	// Service implementations
	sessionService pb.SessionServiceServer
	// agentService   pb.AgentServiceServer // TODO: Define in proto
	// memoryService  pb.MemoryServiceServer // TODO: Define in proto
	chatService    pb.ChatServiceServer
	guildService   pb.GuildServer
	promptService  promptspb.PromptServiceServer
	
	// Health management
	healthServer *health.Server
	
	// State
	started bool
	mu      sync.RWMutex
}

// GRPCServiceManagerConfig configures the gRPC service manager
type GRPCServiceManagerConfig struct {
	EnableHealth     bool
	EnableReflection bool
	EnableTracing    bool
	EnableMetrics    bool
}

// DefaultGRPCServiceManagerConfig returns default configuration
func DefaultGRPCServiceManagerConfig() GRPCServiceManagerConfig {
	return GRPCServiceManagerConfig{
		EnableHealth:     true,
		EnableReflection: true,
		EnableTracing:    false,
		EnableMetrics:    true,
	}
}

// NewGRPCServiceManager creates a new gRPC service manager
func NewGRPCServiceManager(
	registry registry.ComponentRegistry,
	eventBus events.EventBus,
	logger observability.Logger,
	config GRPCServiceManagerConfig,
) (*GRPCServiceManager, error) {
	if registry == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "registry cannot be nil", nil).
			WithComponent("GRPCServiceManager")
	}
	if eventBus == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "event bus cannot be nil", nil).
			WithComponent("GRPCServiceManager")
	}
	if logger == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "logger cannot be nil", nil).
			WithComponent("GRPCServiceManager")
	}

	manager := &GRPCServiceManager{
		registry:     registry,
		eventBus:     eventBus,
		logger:       logger,
		config:       config,
		healthServer: health.NewServer(),
	}

	// Initialize service implementations
	if err := manager.initializeServices(); err != nil {
		return nil, err
	}

	return manager, nil
}

// initializeServices creates all gRPC service implementations
func (m *GRPCServiceManager) initializeServices() error {
	// Session Service
	// TODO: Get actual implementation from registry
	// For now, we'll need to create adapters or use existing implementations
	
	// Agent Service
	// TODO: Create agent service implementation
	
	// Memory Service
	// TODO: Create memory service implementation
	
	// Chat Service
	// TODO: Create chat service implementation
	
	// Guild Service
	// TODO: Create guild service implementation
	
	// Prompt Service
	// TODO: Create prompt service implementation
	
	m.logger.Info("gRPC services initialized",
		"health_enabled", m.config.EnableHealth,
		"reflection_enabled", m.config.EnableReflection)
	
	return nil
}

// RegisterServices registers all gRPC services with the server
func (m *GRPCServiceManager) RegisterServices(server *grpc.Server) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "services already registered", nil).
			WithComponent("GRPCServiceManager")
	}

	// Register session service
	if m.sessionService != nil {
		pb.RegisterSessionServiceServer(server, m.sessionService)
		m.logger.Debug("Registered session service")
		
		// Set health status
		if m.config.EnableHealth {
			m.healthServer.SetServingStatus("guild.v1.SessionService", grpc_health_v1.HealthCheckResponse_SERVING)
		}
	}

	// Register agent service
	// TODO: Uncomment when AgentService is defined in proto
	// if m.agentService != nil {
	// 	pb.RegisterAgentServiceServer(server, m.agentService)
	// 	m.logger.Debug("Registered agent service")
	// 	
	// 	if m.config.EnableHealth {
	// 		m.healthServer.SetServingStatus("guild.v1.AgentService", grpc_health_v1.HealthCheckResponse_SERVING)
	// 	}
	// }

	// Register memory service
	// TODO: Uncomment when MemoryService is defined in proto
	// if m.memoryService != nil {
	// 	pb.RegisterMemoryServiceServer(server, m.memoryService)
	// 	m.logger.Debug("Registered memory service")
	// 	
	// 	if m.config.EnableHealth {
	// 		m.healthServer.SetServingStatus("guild.v1.MemoryService", grpc_health_v1.HealthCheckResponse_SERVING)
	// 	}
	// }

	// Register chat service
	if m.chatService != nil {
		pb.RegisterChatServiceServer(server, m.chatService)
		m.logger.Debug("Registered chat service")
		
		if m.config.EnableHealth {
			m.healthServer.SetServingStatus("guild.v1.ChatService", grpc_health_v1.HealthCheckResponse_SERVING)
		}
	}

	// Register guild service
	if m.guildService != nil {
		pb.RegisterGuildServer(server, m.guildService)
		m.logger.Debug("Registered guild service")
		
		if m.config.EnableHealth {
			m.healthServer.SetServingStatus("guild.v1.Guild", grpc_health_v1.HealthCheckResponse_SERVING)
		}
	}

	// Register prompt service
	if m.promptService != nil {
		promptspb.RegisterPromptServiceServer(server, m.promptService)
		m.logger.Debug("Registered prompt service")
		
		if m.config.EnableHealth {
			m.healthServer.SetServingStatus("prompts.v1.PromptService", grpc_health_v1.HealthCheckResponse_SERVING)
		}
	}

	// Register health service itself if enabled
	if m.config.EnableHealth {
		grpc_health_v1.RegisterHealthServer(server, m.healthServer)
		m.logger.Debug("Registered health service")
	}

	m.started = true

	// Emit services registered event
	if err := m.eventBus.Publish(context.Background(), events.NewBaseEvent(
		"grpc-services-registered",
		"grpc.services.registered",
		"grpc-manager",
		map[string]interface{}{
			"session_service": m.sessionService != nil,
			// "agent_service":   m.agentService != nil,
			// "memory_service":  m.memoryService != nil,
			"chat_service":    m.chatService != nil,
			"guild_service":   m.guildService != nil,
			"prompt_service":  m.promptService != nil,
			"health_enabled":  m.config.EnableHealth,
		},
	)); err != nil {
		m.logger.Warn("Failed to publish services registered event", "error", err)
	}

	m.logger.Info("All gRPC services registered successfully")
	return nil
}

// UnregisterServices unregisters all services (for cleanup)
func (m *GRPCServiceManager) UnregisterServices() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return gerror.New(gerror.ErrCodeValidation, "services not registered", nil).
			WithComponent("GRPCServiceManager")
	}

	// Update health status to NOT_SERVING
	if m.config.EnableHealth {
		if m.sessionService != nil {
			m.healthServer.SetServingStatus("guild.v1.SessionService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		}
		// if m.agentService != nil {
		// 	m.healthServer.SetServingStatus("guild.v1.AgentService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		// }
		// if m.memoryService != nil {
		// 	m.healthServer.SetServingStatus("guild.v1.MemoryService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		// }
		if m.chatService != nil {
			m.healthServer.SetServingStatus("guild.v1.ChatService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		}
		if m.guildService != nil {
			m.healthServer.SetServingStatus("guild.v1.Guild", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		}
		if m.promptService != nil {
			m.healthServer.SetServingStatus("prompts.v1.PromptService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		}
	}

	m.started = false

	// Emit services unregistered event
	if err := m.eventBus.Publish(context.Background(), events.NewBaseEvent(
		"grpc-services-unregistered",
		"grpc.services.unregistered",
		"grpc-manager",
		nil,
	)); err != nil {
		m.logger.Warn("Failed to publish services unregistered event", "error", err)
	}

	m.logger.Info("All gRPC services unregistered")
	return nil
}

// Health checks if all services are healthy
func (m *GRPCServiceManager) Health(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return gerror.New(gerror.ErrCodeResourceExhausted, "services not started", nil).
			WithComponent("GRPCServiceManager")
	}

	// TODO: Check individual service health
	// For now, assume healthy if started
	return nil
}

// GetHealthServer returns the health server for external use
func (m *GRPCServiceManager) GetHealthServer() *health.Server {
	return m.healthServer
}

// SetSessionService sets the session service implementation
func (m *GRPCServiceManager) SetSessionService(service pb.SessionServiceServer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionService = service
}

// SetAgentService sets the agent service implementation
// TODO: Uncomment when AgentService is defined in proto
// func (m *GRPCServiceManager) SetAgentService(service pb.AgentServiceServer) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()
// 	m.agentService = service
// }

// SetMemoryService sets the memory service implementation  
// TODO: Uncomment when MemoryService is defined in proto
// func (m *GRPCServiceManager) SetMemoryService(service pb.MemoryServiceServer) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()
// 	m.memoryService = service
// }

// SetChatService sets the chat service implementation
func (m *GRPCServiceManager) SetChatService(service pb.ChatServiceServer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chatService = service
}

// SetGuildService sets the guild service implementation
func (m *GRPCServiceManager) SetGuildService(service pb.GuildServer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.guildService = service
}

// SetPromptService sets the prompt service implementation
func (m *GRPCServiceManager) SetPromptService(service promptspb.PromptServiceServer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.promptService = service
}

// GetMetrics returns service manager metrics
func (m *GRPCServiceManager) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"services_registered": m.started,
		"session_service":     m.sessionService != nil,
		// "agent_service":       m.agentService != nil,
		// "memory_service":      m.memoryService != nil,
		"chat_service":        m.chatService != nil,
		"guild_service":       m.guildService != nil,
		"prompt_service":      m.promptService != nil,
		"health_enabled":      m.config.EnableHealth,
	}
}