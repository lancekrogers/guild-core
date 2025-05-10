package zeromq

import (
	"errors"
	"fmt"
)

// Config holds ZeroMQ configuration
type Config struct {
	// PubEndpoint is the ZeroMQ publisher endpoint (e.g., "tcp://127.0.0.1:5556")
	PubEndpoint string

	// SubEndpoint is the ZeroMQ subscriber endpoint (e.g., "tcp://127.0.0.1:5557")
	SubEndpoint string

	// Identity is an optional identity for the socket
	Identity string

	// HighWaterMark limits queue size (0 = no limit)
	HighWaterMark int

	// Timeout in milliseconds (0 = no timeout)
	Timeout int
}

// Validate checks the configuration
func (c *Config) Validate() error {
	if c.PubEndpoint == "" && c.SubEndpoint == "" {
		return errors.New("at least one endpoint must be specified")
	}

	return nil
}

// FromMap creates a Config from a map
func FromMap(m map[string]interface{}) (*Config, error) {
	config := &Config{}

	if v, ok := m["pub_endpoint"].(string); ok {
		config.PubEndpoint = v
	}

	if v, ok := m["sub_endpoint"].(string); ok {
		config.SubEndpoint = v
	}

	if v, ok := m["identity"].(string); ok {
		config.Identity = v
	}

	if v, ok := m["high_water_mark"].(int); ok {
		config.HighWaterMark = v
	}

	if v, ok := m["timeout"].(int); ok {
		config.Timeout = v
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ZeroMQ configuration: %w", err)
	}

	return config, nil
}