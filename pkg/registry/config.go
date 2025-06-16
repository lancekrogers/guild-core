// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"os"

	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// LoadConfig loads configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read config file").
			WithComponent("registry").
			WithOperation("LoadConfig").
			WithDetails("filename", filename)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "failed to parse config file").
			WithComponent("registry").
			WithOperation("LoadConfig").
			WithDetails("filename", filename)
	}

	return &config, nil
}

// LoadConfigFromBytes loads configuration from YAML bytes
func LoadConfigFromBytes(data []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "failed to parse config").
			WithComponent("registry").
			WithOperation("LoadConfigFromBytes")
	}

	return &config, nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(config *Config, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal config").
			WithComponent("registry").
			WithOperation("SaveConfig")
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write config file").
			WithComponent("registry").
			WithOperation("SaveConfig").
			WithDetails("filename", filename)
	}

	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Agents: AgentConfigYaml{
			DefaultType: "worker",
			Types: map[string]interface{}{
				"worker": map[string]interface{}{
					"enabled": true,
				},
				"manager": map[string]interface{}{
					"enabled": true,
				},
			},
		},
		Tools: ToolConfig{
			EnabledTools: []string{
				"file",
				"shell",
				"http",
			},
			Settings: map[string]interface{}{
				"timeout": "30s",
			},
		},
		Providers: ProviderConfig{
			DefaultProvider: "openai",
			Providers: map[string]interface{}{
				"openai": map[string]interface{}{
					"model":       "gpt-4.1",
					"api_key_env": "OPENAI_API_KEY",
				},
				"anthropic": map[string]interface{}{
					"model":       "claude-4-sonnet",
					"api_key_env": "ANTHROPIC_API_KEY",
				},
				"google": map[string]interface{}{
					"model":       "gemini-2.5-flash",
					"api_key_env": "GOOGLE_API_KEY",
				},
				"ollama": map[string]interface{}{
					"model": "llama3.1:8b",
					"url":   "http://localhost:11434",
				},
				"claudecode": map[string]interface{}{
					"model":    "sonnet",
					"bin_path": "claude-code",
				},
			},
		},
		Memory: MemoryConfig{
			DefaultMemoryStore: "sqlite",
			DefaultVectorStore: "chromem",
			Stores: map[string]interface{}{
				"sqlite": map[string]interface{}{
					"path": "./.guild/memory.db",
				},
				"chromem": map[string]interface{}{
					"persistence_path": "./.guild/vectors",
					"dimension":        1536,
				},
			},
		},
	}
}

// ValidateConfig validates the configuration
func ValidateConfig(config *Config) error {
	// Validate agents configuration
	if config.Agents.DefaultType == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "agents.default_type cannot be empty", nil).
			WithComponent("registry").
			WithOperation("ValidateConfig")
	}

	// Validate providers configuration
	if config.Providers.DefaultProvider == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "providers.default_provider cannot be empty", nil).
			WithComponent("registry").
			WithOperation("ValidateConfig")
	}

	// Check if default provider exists in providers map
	if _, exists := config.Providers.Providers[config.Providers.DefaultProvider]; !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "default provider '%s' not found in providers configuration", config.Providers.DefaultProvider).
			WithComponent("registry").
			WithOperation("ValidateConfig").
			WithDetails("provider", config.Providers.DefaultProvider)
	}

	// Validate memory configuration
	if config.Memory.DefaultMemoryStore == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "memory.default_memory_store cannot be empty", nil).
			WithComponent("registry").
			WithOperation("ValidateConfig")
	}

	if config.Memory.DefaultVectorStore == "" {
		return gerror.New(gerror.ErrCodeMissingRequired, "memory.default_vector_store cannot be empty", nil).
			WithComponent("registry").
			WithOperation("ValidateConfig")
	}

	// Check if default stores exist in stores map
	if _, exists := config.Memory.Stores[config.Memory.DefaultMemoryStore]; !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "default memory store '%s' not found in stores configuration", config.Memory.DefaultMemoryStore).
			WithComponent("registry").
			WithOperation("ValidateConfig").
			WithDetails("memoryStore", config.Memory.DefaultMemoryStore)
	}

	if _, exists := config.Memory.Stores[config.Memory.DefaultVectorStore]; !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "default vector store '%s' not found in stores configuration", config.Memory.DefaultVectorStore).
			WithComponent("registry").
			WithOperation("ValidateConfig").
			WithDetails("vectorStore", config.Memory.DefaultVectorStore)
	}

	return nil
}

// GetProviderConfig extracts provider-specific configuration
func (c *Config) GetProviderConfig(providerName string) (map[string]interface{}, error) {
	config, exists := c.Providers.Providers[providerName]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "provider '%s' not found in configuration", providerName).
			WithComponent("registry").
			WithOperation("GetProviderConfig").
			WithDetails("provider", providerName)
	}

	configMap, ok := config.(map[string]interface{})
	if !ok {
		return nil, gerror.Newf(gerror.ErrCodeInvalidFormat, "invalid provider configuration for '%s'", providerName).
			WithComponent("registry").
			WithOperation("GetProviderConfig").
			WithDetails("provider", providerName)
	}

	return configMap, nil
}

// GetMemoryStoreConfig extracts memory store-specific configuration
func (c *Config) GetMemoryStoreConfig(storeName string) (map[string]interface{}, error) {
	config, exists := c.Memory.Stores[storeName]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "memory store '%s' not found in configuration", storeName).
			WithComponent("registry").
			WithOperation("GetMemoryStoreConfig").
			WithDetails("store", storeName)
	}

	configMap, ok := config.(map[string]interface{})
	if !ok {
		return nil, gerror.Newf(gerror.ErrCodeInvalidFormat, "invalid memory store configuration for '%s'", storeName).
			WithComponent("registry").
			WithOperation("GetMemoryStoreConfig").
			WithDetails("store", storeName)
	}

	return configMap, nil
}

// GetToolConfig extracts tool-specific configuration
func (c *Config) GetToolConfig(toolName string) (map[string]interface{}, error) {
	if c.Tools.Settings == nil {
		return make(map[string]interface{}), nil
	}

	return c.Tools.Settings, nil
}

// IsToolEnabled checks if a tool is enabled
func (c *Config) IsToolEnabled(toolName string) bool {
	for _, enabledTool := range c.Tools.EnabledTools {
		if enabledTool == toolName {
			return true
		}
	}
	return false
}
