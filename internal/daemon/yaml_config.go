// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// YAMLConfig represents the daemon configuration loaded from YAML
type YAMLConfig struct {
	// Server configuration
	Server ServerConfig `yaml:"server"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging"`

	// Storage configuration
	Storage StorageConfig `yaml:"storage"`

	// Health check configuration
	Health HealthConfig `yaml:"health"`
}

// ServerConfig configures the gRPC server
type ServerConfig struct {
	// Socket path for Unix domain socket
	Socket string `yaml:"socket,omitempty"`

	// TCP address for network binding (future use)
	Address string `yaml:"address,omitempty"`

	// Port for TCP binding (future use)
	Port int `yaml:"port,omitempty"`

	// Maximum concurrent streams
	MaxConcurrentStreams uint32 `yaml:"max_concurrent_streams,omitempty"`

	// Connection timeout in seconds
	ConnectionTimeout int `yaml:"connection_timeout,omitempty"`
}

// LoggingConfig configures logging
type LoggingConfig struct {
	// Log level (debug, info, warn, error)
	Level string `yaml:"level,omitempty"`

	// Log format (json, text)
	Format string `yaml:"format,omitempty"`

	// Log file path
	File string `yaml:"file,omitempty"`

	// Maximum log file size in MB
	MaxSize int `yaml:"max_size,omitempty"`

	// Maximum number of log files to keep
	MaxBackups int `yaml:"max_backups,omitempty"`

	// Maximum age of log files in days
	MaxAge int `yaml:"max_age,omitempty"`
}

// StorageConfig configures storage backends
type StorageConfig struct {
	// Database type (sqlite, postgres, mysql)
	Type string `yaml:"type,omitempty"`

	// Database connection string or path
	DSN string `yaml:"dsn,omitempty"`

	// Maximum open connections
	MaxOpenConns int `yaml:"max_open_conns,omitempty"`

	// Maximum idle connections
	MaxIdleConns int `yaml:"max_idle_conns,omitempty"`

	// Connection maximum lifetime in minutes
	ConnMaxLifetime int `yaml:"conn_max_lifetime,omitempty"`
}

// HealthConfig configures health checks
type HealthConfig struct {
	// Enable HTTP health endpoint
	HTTPEnabled bool `yaml:"http_enabled,omitempty"`

	// HTTP health endpoint port
	HTTPPort int `yaml:"http_port,omitempty"`

	// HTTP health endpoint path
	HTTPPath string `yaml:"http_path,omitempty"`

	// Enable gRPC health service
	GRPCEnabled bool `yaml:"grpc_enabled,omitempty"`
}

// DefaultYAMLConfig returns a default YAML configuration
func DefaultYAMLConfig() *YAMLConfig {
	return &YAMLConfig{
		Server: ServerConfig{
			MaxConcurrentStreams: 100,
			ConnectionTimeout:    300, // 5 minutes
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			MaxSize:    100, // 100 MB
			MaxBackups: 3,
			MaxAge:     7, // 7 days
		},
		Storage: StorageConfig{
			Type:            "sqlite",
			DSN:             ".guild/memory.db",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5, // 5 minutes
		},
		Health: HealthConfig{
			HTTPEnabled: false,
			HTTPPort:    8080,
			HTTPPath:    "/healthz",
			GRPCEnabled: false,
		},
	}
}

// LoadYAMLConfig loads configuration from a YAML file
func LoadYAMLConfig(path string) (*YAMLConfig, error) {
	// Start with default config
	config := DefaultYAMLConfig()

	// If no path specified, look for default locations
	if path == "" {
		path = findDefaultConfigFile()
		if path == "" {
			return config, nil // Use defaults if no config file found
		}
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return config, nil // Use defaults if file doesn't exist
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to read config file").
			WithComponent("daemon").
			WithOperation("LoadYAMLConfig").
			WithDetails("path", path)
	}

	// Unmarshal YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "failed to parse config file").
			WithComponent("daemon").
			WithOperation("LoadYAMLConfig").
			WithDetails("path", path)
	}

	// Apply environment variable overrides
	config.applyEnvOverrides()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid configuration").
			WithComponent("daemon").
			WithOperation("LoadYAMLConfig")
	}

	return config, nil
}

// findDefaultConfigFile looks for config files in standard locations
func findDefaultConfigFile() string {
	// Check locations in order of preference
	locations := []string{
		"guild-daemon.yaml",
		"guild-daemon.yml",
		".guild/daemon.yaml",
		".guild/daemon.yml",
	}

	// Check current directory
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	// Check home directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		for _, loc := range locations {
			path := filepath.Join(homeDir, loc)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}

// applyEnvOverrides applies environment variable overrides
func (c *YAMLConfig) applyEnvOverrides() {
	// Server overrides
	if socket := os.Getenv("GUILD_DAEMON_SOCKET"); socket != "" {
		c.Server.Socket = socket
	}
	if port := os.Getenv("GUILD_DAEMON_PORT"); port != "" {
		var p int
		if n, err := fmt.Sscanf(port, "%d", &p); err == nil && n == 1 {
			c.Server.Port = p
		}
	}

	// Logging overrides
	if level := os.Getenv("GUILD_LOG_LEVEL"); level != "" {
		c.Logging.Level = level
	}
	if format := os.Getenv("GUILD_LOG_FORMAT"); format != "" {
		c.Logging.Format = format
	}
	if file := os.Getenv("GUILD_LOG_FILE"); file != "" {
		c.Logging.File = file
	}

	// Storage overrides
	if dbType := os.Getenv("GUILD_DB_TYPE"); dbType != "" {
		c.Storage.Type = dbType
	}
	if dsn := os.Getenv("GUILD_DB_DSN"); dsn != "" {
		c.Storage.DSN = dsn
	}

	// Health overrides
	if httpEnabled := os.Getenv("GUILD_HEALTH_HTTP_ENABLED"); httpEnabled == "true" {
		c.Health.HTTPEnabled = true
	}
	if httpPort := os.Getenv("GUILD_HEALTH_HTTP_PORT"); httpPort != "" {
		var p int
		if n, err := fmt.Sscanf(httpPort, "%d", &p); err == nil && n == 1 {
			c.Health.HTTPPort = p
		}
	}
}

// Validate validates the configuration
func (c *YAMLConfig) Validate() error {
	// Validate log level
	switch c.Logging.Level {
	case "debug", "info", "warn", "error":
		// Valid
	default:
		return gerror.Newf(gerror.ErrCodeValidation, "invalid log level: %s", c.Logging.Level).
			WithComponent("daemon").
			WithOperation("YAMLConfig.Validate")
	}

	// Validate log format
	switch c.Logging.Format {
	case "json", "text":
		// Valid
	default:
		return gerror.Newf(gerror.ErrCodeValidation, "invalid log format: %s", c.Logging.Format).
			WithComponent("daemon").
			WithOperation("YAMLConfig.Validate")
	}

	// Validate storage type
	switch c.Storage.Type {
	case "sqlite":
		// Valid
	case "postgres", "mysql":
		return gerror.New(gerror.ErrCodeNotImplemented, "only sqlite is currently supported", nil).
			WithComponent("daemon").
			WithOperation("YAMLConfig.Validate")
	default:
		return gerror.Newf(gerror.ErrCodeValidation, "invalid storage type: %s", c.Storage.Type).
			WithComponent("daemon").
			WithOperation("YAMLConfig.Validate")
	}

	// Validate health configuration
	if c.Health.HTTPEnabled && (c.Health.HTTPPort <= 0 || c.Health.HTTPPort > 65535) {
		return gerror.Newf(gerror.ErrCodeValidation, "invalid HTTP health port: %d", c.Health.HTTPPort).
			WithComponent("daemon").
			WithOperation("YAMLConfig.Validate")
	}

	return nil
}

// ApplyToConfig applies YAML configuration to DaemonConfig
func (c *YAMLConfig) ApplyToConfig(dc *DaemonConfig) {
	// Apply socket path if specified
	if c.Server.Socket != "" {
		dc.SocketPath = c.Server.Socket
	}

	// Apply log file path if specified
	if c.Logging.File != "" {
		dc.LogFile = c.Logging.File
	}

	// Apply resource limits
	if c.Server.ConnectionTimeout > 0 {
		dc.IdleTimeout = time.Duration(c.Server.ConnectionTimeout) * time.Second
	}
}
