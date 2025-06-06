// Package config provides MCP configuration integration
package config

import (
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// MCPConfig represents MCP configuration section
type MCPConfig struct {
	// Core settings
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	ServerID   string `yaml:"server_id" json:"server_id"`
	ServerName string `yaml:"server_name" json:"server_name"`

	// Transport configuration
	Transport TransportConfig `yaml:"transport" json:"transport"`

	// Security settings
	Security SecurityConfig `yaml:"security" json:"security"`

	// Performance settings
	Performance PerformanceConfig `yaml:"performance" json:"performance"`

	// Feature flags
	Features FeatureConfig `yaml:"features" json:"features"`
}

// TransportConfig represents transport configuration
type TransportConfig struct {
	Type           string            `yaml:"type" json:"type"`                       // "nats", "memory", "grpc"
	Address        string            `yaml:"address" json:"address"`                 // Transport address
	ConnectTimeout string            `yaml:"connect_timeout" json:"connect_timeout"` // e.g., "10s"
	MaxReconnects  int               `yaml:"max_reconnects" json:"max_reconnects"`
	ReconnectWait  string            `yaml:"reconnect_wait" json:"reconnect_wait"` // e.g., "2s"
	Config         map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	EnableTLS   bool   `yaml:"enable_tls" json:"enable_tls"`
	TLSCertFile string `yaml:"tls_cert_file,omitempty" json:"tls_cert_file,omitempty"`
	TLSKeyFile  string `yaml:"tls_key_file,omitempty" json:"tls_key_file,omitempty"`
	EnableAuth  bool   `yaml:"enable_auth" json:"enable_auth"`
	JWTSecret   string `yaml:"jwt_secret,omitempty" json:"jwt_secret,omitempty"`
}

// PerformanceConfig represents performance configuration
type PerformanceConfig struct {
	MaxConcurrentRequests int    `yaml:"max_concurrent_requests" json:"max_concurrent_requests"`
	RequestTimeout        string `yaml:"request_timeout" json:"request_timeout"` // e.g., "30s"
	ShutdownTimeout       string `yaml:"shutdown_timeout" json:"shutdown_timeout"`
}

// FeatureConfig represents feature flags
type FeatureConfig struct {
	EnableMetrics      bool `yaml:"enable_metrics" json:"enable_metrics"`
	EnableTracing      bool `yaml:"enable_tracing" json:"enable_tracing"`
	EnableCostTracking bool `yaml:"enable_cost_tracking" json:"enable_cost_tracking"`
}

// DefaultMCPConfig returns a default MCP configuration
func DefaultMCPConfig() *MCPConfig {
	return &MCPConfig{
		Enabled:    false, // Disabled by default
		ServerID:   "guild-mcp-server",
		ServerName: "Guild MCP Server",
		Transport: TransportConfig{
			Type:           "memory",
			Address:        "memory://default",
			ConnectTimeout: "10s",
			MaxReconnects:  3,
			ReconnectWait:  "2s",
		},
		Security: SecurityConfig{
			EnableTLS:  false,
			EnableAuth: false,
		},
		Performance: PerformanceConfig{
			MaxConcurrentRequests: 100,
			RequestTimeout:        "30s",
			ShutdownTimeout:       "10s",
		},
		Features: FeatureConfig{
			EnableMetrics:      true,
			EnableTracing:      false,
			EnableCostTracking: true,
		},
	}
}

// ProductionMCPConfig returns a production-ready MCP configuration
func ProductionMCPConfig() *MCPConfig {
	config := DefaultMCPConfig()
	config.Enabled = true
	config.Transport.Type = "nats"
	config.Transport.Address = "nats://localhost:4222"
	config.Security.EnableTLS = true
	config.Security.EnableAuth = true
	config.Performance.MaxConcurrentRequests = 1000
	config.Features.EnableTracing = true
	return config
}

// Note: ToIntegrationConfig method removed to break circular dependency.
// MCP integration should read this config directly and perform its own conversion.

// Validate validates the MCP configuration
func (c *MCPConfig) Validate() error {
	if c == nil {
		return nil
	}

	if !c.Enabled {
		return nil // No validation needed if disabled
	}

	if c.ServerID == "" {
		return gerror.New(gerror.ErrCodeValidation, "server_id is required when MCP is enabled", nil).
			WithComponent("config").
			WithOperation("validate_mcp")
	}

	if c.Transport.Type == "" {
		return gerror.New(gerror.ErrCodeValidation, "transport type is required", nil).
			WithComponent("config").
			WithOperation("validate_mcp")
	}

	// Validate transport type
	validTypes := map[string]bool{
		"nats":   true,
		"memory": true,
		"grpc":   true,
	}
	if !validTypes[c.Transport.Type] {
		return gerror.Newf(gerror.ErrCodeValidation, "invalid transport type: %s", c.Transport.Type).
			WithComponent("config").
			WithOperation("validate_mcp")
	}

	// Validate timeouts
	if c.Transport.ConnectTimeout != "" {
		if _, err := time.ParseDuration(c.Transport.ConnectTimeout); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeValidation, "invalid connect_timeout").
				WithComponent("config").
				WithOperation("validate_mcp")
		}
	}

	if c.Transport.ReconnectWait != "" {
		if _, err := time.ParseDuration(c.Transport.ReconnectWait); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeValidation, "invalid reconnect_wait").
				WithComponent("config").
				WithOperation("validate_mcp")
		}
	}

	if c.Performance.RequestTimeout != "" {
		if _, err := time.ParseDuration(c.Performance.RequestTimeout); err != nil {
			return gerror.New(gerror.InvalidArgument, "config", "validate_mcp", "invalid request_timeout: %v", err)
		}
	}

	// Security validation
	if c.Security.EnableTLS {
		if c.Security.TLSCertFile == "" || c.Security.TLSKeyFile == "" {
			return gerror.New(gerror.InvalidArgument, "config", "validate_mcp", "TLS cert and key files are required when TLS is enabled")
		}
	}

	if c.Security.EnableAuth && c.Security.JWTSecret == "" {
		return gerror.New(gerror.InvalidArgument, "config", "validate_mcp", "JWT secret is required when authentication is enabled")
	}

	return nil
}

// ExampleConfigs provides example configurations

// ExampleDevelopmentConfig returns a development MCP configuration
func ExampleDevelopmentConfig() *MCPConfig {
	return &MCPConfig{
		Enabled:    true,
		ServerID:   "dev-mcp-server",
		ServerName: "Development MCP Server",
		Transport: TransportConfig{
			Type:           "memory",
			Address:        "memory://dev",
			ConnectTimeout: "5s",
			MaxReconnects:  1,
			ReconnectWait:  "1s",
		},
		Security: SecurityConfig{
			EnableTLS:  false,
			EnableAuth: false,
		},
		Performance: PerformanceConfig{
			MaxConcurrentRequests: 10,
			RequestTimeout:        "10s",
			ShutdownTimeout:       "5s",
		},
		Features: FeatureConfig{
			EnableMetrics:      true,
			EnableTracing:      true,
			EnableCostTracking: true,
		},
	}
}

// ExampleNATSConfig returns a NATS-based MCP configuration
func ExampleNATSConfig() *MCPConfig {
	return &MCPConfig{
		Enabled:    true,
		ServerID:   "nats-mcp-server",
		ServerName: "NATS MCP Server",
		Transport: TransportConfig{
			Type:           "nats",
			Address:        "nats://localhost:4222",
			ConnectTimeout: "10s",
			MaxReconnects:  5,
			ReconnectWait:  "2s",
			Config: map[string]string{
				"cluster_id": "guild-cluster",
				"client_id":  "mcp-server",
			},
		},
		Security: SecurityConfig{
			EnableTLS:  false,
			EnableAuth: false,
		},
		Performance: PerformanceConfig{
			MaxConcurrentRequests: 500,
			RequestTimeout:        "30s",
			ShutdownTimeout:       "10s",
		},
		Features: FeatureConfig{
			EnableMetrics:      true,
			EnableTracing:      true,
			EnableCostTracking: true,
		},
	}
}