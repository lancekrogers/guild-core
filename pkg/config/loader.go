package config

import (
	"path/filepath"
)

// ConfigLoader loads configuration from files and environment
type ConfigLoader struct {
	configDir string
}

// NewConfigLoader creates a new config loader
func NewConfigLoader(configDir string) *ConfigLoader {
	return &ConfigLoader{
		configDir: configDir,
	}
}

// LoadConfig loads configuration from a file
func (l *ConfigLoader) LoadConfig(filename string) (map[string]interface{}, error) {
	// For now, return empty config since actual config loading is handled by LoadGuildConfig
	// This exists for compatibility with corpus package
	// TODO: Consolidate config loading approaches
	return map[string]interface{}{}, nil
}

// GetConfigPath returns the absolute path to a config file
func (l *ConfigLoader) GetConfigPath(filename string) string {
	return filepath.Join(l.configDir, filename)
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv(prefix string) map[string]string {
	// For now, return empty map since env loading is handled by GetProviderAPIKey
	// This exists for compatibility with corpus package
	// TODO: Implement proper env var scanning with prefix
	return make(map[string]string)
}
