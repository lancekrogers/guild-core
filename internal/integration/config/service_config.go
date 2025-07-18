// Package config provides configuration management for Guild services
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/lancekrogers/guild/internal/integration/services"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// ServiceConfig represents the complete service configuration
type ServiceConfig struct {
	// Global settings
	Global GlobalConfig `yaml:"global"`
	
	// Service-specific configurations
	Services ServicesConfig `yaml:"services"`
	
	// Integration settings
	Integration IntegrationConfig `yaml:"integration"`
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	// Runtime settings
	LogLevel      string        `yaml:"log_level"`
	LogFile       string        `yaml:"log_file"`
	DataDir       string        `yaml:"data_dir"`
	
	// Performance settings
	MaxWorkers    int           `yaml:"max_workers"`
	MaxMemoryMB   int           `yaml:"max_memory_mb"`
	
	// Timeouts
	StartTimeout  time.Duration `yaml:"start_timeout"`
	StopTimeout   time.Duration `yaml:"stop_timeout"`
}

// ServicesConfig contains all service configurations
type ServicesConfig struct {
	// Core services
	Kanban       KanbanConfig       `yaml:"kanban"`
	Memory       MemoryConfig       `yaml:"memory"`
	Session      SessionConfig      `yaml:"session"`
	
	// Agent services
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
	AgentManager AgentManagerConfig `yaml:"agent_manager"`
	
	// UI services
	ChatUI       ChatUIConfig       `yaml:"chat_ui"`
	
	// Infrastructure services
	Daemon       DaemonConfig       `yaml:"daemon"`
	Corpus       CorpusConfig       `yaml:"corpus"`
}

// IntegrationConfig contains integration settings
type IntegrationConfig struct {
	// Event system
	Events EventsConfig `yaml:"events"`
	
	// Bridges
	Bridges BridgesConfig `yaml:"bridges"`
	
	// Service discovery
	Discovery DiscoveryConfig `yaml:"discovery"`
}

// KanbanConfig configures the Kanban service
type KanbanConfig struct {
	Enabled      bool          `yaml:"enabled"`
	Path         string        `yaml:"path"`
	AutoSave     bool          `yaml:"auto_save"`
	SaveInterval time.Duration `yaml:"save_interval"`
}

// MemoryConfig configures the Memory service
type MemoryConfig struct {
	Enabled         bool   `yaml:"enabled"`
	DatabasePath    string `yaml:"database_path"`
	CacheSize       int    `yaml:"cache_size"`
	EnableVectors   bool   `yaml:"enable_vectors"`
	VectorDimension int    `yaml:"vector_dimension"`
}

// SessionConfig configures the Session service
type SessionConfig struct {
	Enabled           bool          `yaml:"enabled"`
	MaxActiveSessions int           `yaml:"max_active_sessions"`
	SessionTimeout    time.Duration `yaml:"session_timeout"`
	PersistSessions   bool          `yaml:"persist_sessions"`
	CleanupInterval   time.Duration `yaml:"cleanup_interval"`
}

// OrchestratorConfig configures the Orchestrator service
type OrchestratorConfig struct {
	Enabled              bool `yaml:"enabled"`
	MaxConcurrentTasks   int  `yaml:"max_concurrent_tasks"`
	TaskQueueSize        int  `yaml:"task_queue_size"`
	WorkerPoolSize       int  `yaml:"worker_pool_size"`
}

// AgentManagerConfig configures the Agent Manager service
type AgentManagerConfig struct {
	Enabled            bool   `yaml:"enabled"`
	DefaultProjectPath string `yaml:"default_project_path"`
	EnableAutoElena    bool   `yaml:"enable_auto_elena"`
	BackstoryPath      string `yaml:"backstory_path"`
}

// ChatUIConfig configures the Chat UI service
type ChatUIConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Theme         string `yaml:"theme"`
	EnableLogging bool   `yaml:"enable_logging"`
	LogPath       string `yaml:"log_path"`
}

// DaemonConfig configures the Daemon service
type DaemonConfig struct {
	Enabled             bool          `yaml:"enabled"`
	GRPCPort            int           `yaml:"grpc_port"`
	HTTPPort            int           `yaml:"http_port"`
	MetricsPort         int           `yaml:"metrics_port"`
	TLSEnabled          bool          `yaml:"tls_enabled"`
	CertPath            string        `yaml:"cert_path"`
	KeyPath             string        `yaml:"key_path"`
	MaxConnections      int           `yaml:"max_connections"`
	ConnectionTimeout   time.Duration `yaml:"connection_timeout"`
	GracefulStopTimeout time.Duration `yaml:"graceful_stop_timeout"`
}

// CorpusConfig configures the Corpus service
type CorpusConfig struct {
	Enabled           bool          `yaml:"enabled"`
	BasePath          string        `yaml:"base_path"`
	FilePatterns      []string      `yaml:"file_patterns"`
	IgnorePatterns    []string      `yaml:"ignore_patterns"`
	MaxWorkers        int           `yaml:"max_workers"`
	ScanOnStart       bool          `yaml:"scan_on_start"`
	RescanInterval    time.Duration `yaml:"rescan_interval"`
	MaxFileSize       int64         `yaml:"max_file_size"`
}

// EventsConfig configures the event system
type EventsConfig struct {
	BufferSize       int           `yaml:"buffer_size"`
	WorkerCount      int           `yaml:"worker_count"`
	FlushInterval    time.Duration `yaml:"flush_interval"`
	MaxRetries       int           `yaml:"max_retries"`
}

// BridgesConfig configures integration bridges
type BridgesConfig struct {
	EventLogging      bool `yaml:"event_logging"`
	PersistenceEvents bool `yaml:"persistence_events"`
	UIEvents          bool `yaml:"ui_events"`
}

// DiscoveryConfig configures service discovery
type DiscoveryConfig struct {
	Mode            string        `yaml:"mode"` // static, dynamic
	RefreshInterval time.Duration `yaml:"refresh_interval"`
}

// DefaultServiceConfig returns the default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		Global: GlobalConfig{
			LogLevel:     "info",
			LogFile:      ".guild/logs/guild.log",
			DataDir:      ".guild",
			MaxWorkers:   0, // Auto-detect
			MaxMemoryMB:  1024,
			StartTimeout: 30 * time.Second,
			StopTimeout:  30 * time.Second,
		},
		Services: ServicesConfig{
			Kanban: KanbanConfig{
				Enabled:      true,
				Path:         ".guild/kanban",
				AutoSave:     true,
				SaveInterval: 30 * time.Second,
			},
			Memory: MemoryConfig{
				Enabled:         true,
				DatabasePath:    ".guild/memory.db",
				CacheSize:       1000,
				EnableVectors:   false,
				VectorDimension: 384,
			},
			Session: SessionConfig{
				Enabled:           true,
				MaxActiveSessions: 100,
				SessionTimeout:    24 * time.Hour,
				PersistSessions:   true,
				CleanupInterval:   1 * time.Hour,
			},
			Orchestrator: OrchestratorConfig{
				Enabled:            true,
				MaxConcurrentTasks: 10,
				TaskQueueSize:      100,
				WorkerPoolSize:     5,
			},
			AgentManager: AgentManagerConfig{
				Enabled:            true,
				DefaultProjectPath: ".guild",
				EnableAutoElena:    true,
				BackstoryPath:      "backstories",
			},
			ChatUI: ChatUIConfig{
				Enabled:       true,
				Theme:         "dark",
				EnableLogging: true,
				LogPath:       ".guild/logs/chat-ui.log",
			},
			Daemon: DaemonConfig{
				Enabled:             true,
				GRPCPort:            9090,
				HTTPPort:            8080,
				MetricsPort:         9091,
				TLSEnabled:          false,
				MaxConnections:      1000,
				ConnectionTimeout:   30 * time.Second,
				GracefulStopTimeout: 30 * time.Second,
			},
			Corpus: CorpusConfig{
				Enabled:    true,
				BasePath:   ".",
				FilePatterns: []string{
					"*.md", "*.yaml", "*.yml", "*.go", "*.js", "*.ts", "*.py",
				},
				IgnorePatterns: []string{
					".git/**", "node_modules/**", "vendor/**", "*.test", "*.tmp",
				},
				MaxWorkers:     4,
				ScanOnStart:    false,
				RescanInterval: 30 * time.Minute,
				MaxFileSize:    10 * 1024 * 1024, // 10MB
			},
		},
		Integration: IntegrationConfig{
			Events: EventsConfig{
				BufferSize:    10000,
				WorkerCount:   4,
				FlushInterval: 100 * time.Millisecond,
				MaxRetries:    3,
			},
			Bridges: BridgesConfig{
				EventLogging:      true,
				PersistenceEvents: true,
				UIEvents:          true,
			},
			Discovery: DiscoveryConfig{
				Mode:            "static",
				RefreshInterval: 5 * time.Minute,
			},
		},
	}
}

// LoadServiceConfig loads configuration from a file
func LoadServiceConfig(path string) (*ServiceConfig, error) {
	// Start with defaults
	config := DefaultServiceConfig()
	
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// No config file, use defaults
		return config, nil
	}
	
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to read config file").
			WithComponent("service-config").
			WithDetails("path", path)
	}
	
	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse config file").
			WithComponent("service-config").
			WithDetails("path", path)
	}
	
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}
	
	// Apply environment variable overrides
	config.ApplyEnvironmentOverrides()
	
	return config, nil
}

// SaveServiceConfig saves configuration to a file
func SaveServiceConfig(config *ServiceConfig, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create config directory").
			WithComponent("service-config")
	}
	
	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal config").
			WithComponent("service-config")
	}
	
	// Write atomically
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write config file").
			WithComponent("service-config")
	}
	
	if err := os.Rename(tmpPath, path); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to rename config file").
			WithComponent("service-config")
	}
	
	return nil
}

// Validate validates the configuration
func (c *ServiceConfig) Validate() error {
	// Validate global settings
	if c.Global.MaxMemoryMB < 0 {
		return gerror.New(gerror.ErrCodeValidation, "max_memory_mb must be positive", nil).
			WithComponent("service-config")
	}
	
	// Validate service settings
	if c.Services.Daemon.GRPCPort < 1 || c.Services.Daemon.GRPCPort > 65535 {
		return gerror.New(gerror.ErrCodeValidation, "invalid gRPC port", nil).
			WithComponent("service-config").
			WithDetails("port", c.Services.Daemon.GRPCPort)
	}
	
	// Validate paths
	if c.Global.DataDir == "" {
		return gerror.New(gerror.ErrCodeValidation, "data_dir cannot be empty", nil).
			WithComponent("service-config")
	}
	
	return nil
}

// ApplyEnvironmentOverrides applies environment variable overrides
func (c *ServiceConfig) ApplyEnvironmentOverrides() {
	// GUILD_LOG_LEVEL
	if level := os.Getenv("GUILD_LOG_LEVEL"); level != "" {
		c.Global.LogLevel = level
	}
	
	// GUILD_DATA_DIR
	if dir := os.Getenv("GUILD_DATA_DIR"); dir != "" {
		c.Global.DataDir = dir
	}
	
	// GUILD_GRPC_PORT
	if port := os.Getenv("GUILD_GRPC_PORT"); port != "" {
		if p, err := fmt.Sscanf(port, "%d", &c.Services.Daemon.GRPCPort); err == nil && p == 1 {
			// Successfully parsed
		}
	}
	
	// GUILD_MAX_WORKERS
	if workers := os.Getenv("GUILD_MAX_WORKERS"); workers != "" {
		if w, err := fmt.Sscanf(workers, "%d", &c.Global.MaxWorkers); err == nil && w == 1 {
			// Successfully parsed
		}
	}
}

// ToServiceConfigs converts to individual service configurations
func (c *ServiceConfig) ToServiceConfigs() ServiceConfigurations {
	return ServiceConfigurations{
		Kanban:       c.toKanbanServiceConfig(),
		Memory:       c.toMemoryServiceConfig(),
		Session:      c.toSessionServiceConfig(),
		Orchestrator: c.toOrchestratorServiceConfig(),
		AgentManager: c.toAgentManagerServiceConfig(),
		ChatUI:       c.toChatUIServiceConfig(),
		Daemon:       c.toDaemonServiceConfig(),
		Corpus:       c.toCorpusServiceConfig(),
	}
}

// ServiceConfigurations holds all service configurations
type ServiceConfigurations struct {
	Kanban       services.KanbanServiceConfig
	Memory       services.MemoryServiceConfig
	Session      services.SessionServiceConfig
	Orchestrator services.OrchestratorServiceConfig
	AgentManager services.AgentManagerServiceConfig
	ChatUI       services.ChatUIServiceConfig
	Daemon       services.DaemonServiceConfig
	Corpus       services.CorpusServiceConfig
}

// Conversion methods

func (c *ServiceConfig) toKanbanServiceConfig() services.KanbanServiceConfig {
	return services.KanbanServiceConfig{
		BoardPath:    filepath.Join(c.Global.DataDir, c.Services.Kanban.Path),
		BoardName:    "Guild Tasks",
		Description:  "Task management board for Guild operations",
		AutoSave:     c.Services.Kanban.AutoSave,
		SaveInterval: c.Services.Kanban.SaveInterval,
	}
}

func (c *ServiceConfig) toMemoryServiceConfig() services.MemoryServiceConfig {
	return services.MemoryServiceConfig{
		CacheSize:       c.Services.Memory.CacheSize,
		EnableVectors:   c.Services.Memory.EnableVectors,
		VectorDimension: c.Services.Memory.VectorDimension,
		SyncInterval:    5 * time.Second,
	}
}

func (c *ServiceConfig) toSessionServiceConfig() services.SessionServiceConfig {
	return services.SessionServiceConfig{
		MaxActiveSessions: c.Services.Session.MaxActiveSessions,
		SessionTimeout:    c.Services.Session.SessionTimeout,
		PersistSessions:   c.Services.Session.PersistSessions,
		RestoreOnStartup:  true,
		CleanupInterval:   c.Services.Session.CleanupInterval,
	}
}

func (c *ServiceConfig) toOrchestratorServiceConfig() services.OrchestratorServiceConfig {
	return services.OrchestratorServiceConfig{
		MaxConcurrentAgents: c.Services.Orchestrator.WorkerPoolSize,
		TaskQueueSize:       c.Services.Orchestrator.TaskQueueSize,
		WorkerPoolSize:      c.Services.Orchestrator.WorkerPoolSize,
		TaskTimeout:         30 * time.Minute,
		RetryAttempts:       3,
		RetryDelay:          5 * time.Second,
	}
}

func (c *ServiceConfig) toAgentManagerServiceConfig() services.AgentManagerServiceConfig {
	return services.AgentManagerServiceConfig{
		DefaultProjectPath: c.Services.AgentManager.DefaultProjectPath,
		EnableAutoElena:    c.Services.AgentManager.EnableAutoElena,
		BackstoryPath:      c.Services.AgentManager.BackstoryPath,
	}
}

func (c *ServiceConfig) toChatUIServiceConfig() services.ChatUIServiceConfig {
	return services.ChatUIServiceConfig{
		// Note: GuildConfig needs to be set separately
		EnableLogging: c.Services.ChatUI.EnableLogging,
		LogPath:       filepath.Join(c.Global.DataDir, c.Services.ChatUI.LogPath),
	}
}

func (c *ServiceConfig) toDaemonServiceConfig() services.DaemonServiceConfig {
	return services.DaemonServiceConfig{
		GRPCPort:            c.Services.Daemon.GRPCPort,
		HTTPPort:            c.Services.Daemon.HTTPPort,
		MetricsPort:         c.Services.Daemon.MetricsPort,
		TLSEnabled:          c.Services.Daemon.TLSEnabled,
		CertPath:            c.Services.Daemon.CertPath,
		KeyPath:             c.Services.Daemon.KeyPath,
		MaxConnections:      c.Services.Daemon.MaxConnections,
		ConnectionTimeout:   c.Services.Daemon.ConnectionTimeout,
		KeepAliveInterval:   30 * time.Second,
		GracefulStopTimeout: c.Services.Daemon.GracefulStopTimeout,
		EnableReflection:    true,
		EnableMetrics:       true,
		EnableProfiling:     false,
	}
}

func (c *ServiceConfig) toCorpusServiceConfig() services.CorpusServiceConfig {
	return services.CorpusServiceConfig{
		BasePath:           c.Services.Corpus.BasePath,
		FilePatterns:       c.Services.Corpus.FilePatterns,
		IgnorePatterns:     c.Services.Corpus.IgnorePatterns,
		MaxWorkers:         c.Services.Corpus.MaxWorkers,
		ScanOnStart:        c.Services.Corpus.ScanOnStart,
		RescanInterval:     c.Services.Corpus.RescanInterval,
		IndexPath:          filepath.Join(c.Global.DataDir, "corpus.db"),
		EnableFullText:     true,
		EnableVectorSearch: false,
		MaxFileSize:        c.Services.Corpus.MaxFileSize,
		MaxScanDuration:    5 * time.Minute,
		MemoryLimit:        512 * 1024 * 1024, // 512MB
	}
}