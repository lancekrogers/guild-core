// Package config provides MCP configuration types
package config

import "github.com/guild-ventures/guild-core/pkg/mcp/transport"

// MCPConfig represents MCP configuration for Guild
type MCPConfig struct {
	Enabled         bool                        `yaml:"enabled"`
	ServerID        string                      `yaml:"server_id"`
	ServerName      string                      `yaml:"server_name"`
	Transport       *transport.TransportConfig  `yaml:"transport"`
	EnableAuth      bool                        `yaml:"enable_auth"`
	EnableTLS       bool                        `yaml:"enable_tls"`
	EnableMetrics   bool                        `yaml:"enable_metrics"`
	EnableTracing   bool                        `yaml:"enable_tracing"`
	EnableCost      bool                        `yaml:"enable_cost_tracking"`
	HealthCheckRate string                      `yaml:"health_check_rate"`

	// Internal fields
	serverInstance interface{} `yaml:"-"` // Avoid circular dependency by using interface{}
}
