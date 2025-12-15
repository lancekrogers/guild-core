// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// LoadAgentConfig loads an enhanced agent configuration from a YAML file
func LoadAgentConfig(ctx context.Context, path string) (*EnhancedAgentConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("config.agent_loader").
			WithOperation("LoadAgentConfig")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "config.agent_loader")
	ctx = observability.WithOperation(ctx, "LoadAgentConfig")

	logger.InfoContext(ctx, "Loading agent configuration", "path", path)

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to read agent config file").
			WithComponent("config.agent_loader").
			WithOperation("LoadAgentConfig").
			WithDetails("path", path)
	}

	// Check context before parsing
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before parsing").
			WithComponent("config.agent_loader").
			WithOperation("LoadAgentConfig")
	}

	logger.DebugContext(ctx, "Parsing YAML configuration", "size_bytes", len(data))

	// Parse YAML
	var config EnhancedAgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to parse YAML").
			WithComponent("config.agent_loader").
			WithOperation("LoadAgentConfig").
			WithDetails("path", path)
	}

	// Check context before validation
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before validation").
			WithComponent("config.agent_loader").
			WithOperation("LoadAgentConfig")
	}

	logger.DebugContext(ctx, "Validating agent configuration", "agent_id", config.ID)

	// Validate the configuration
	if err := config.Validate(ctx); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "agent configuration validation failed").
			WithComponent("config.agent_loader").
			WithOperation("LoadAgentConfig").
			WithDetails("path", path).
			WithDetails("agent_id", config.ID)
	}

	logger.InfoContext(ctx, "Agent configuration loaded successfully",
		"agent_id", config.ID,
		"agent_name", config.Name,
		"agent_type", config.Type,
		"provider", config.GetEffectiveProvider(),
		"cost_magnitude", config.GetEffectiveCostMagnitude())

	return &config, nil
}

// SaveAgentConfig saves an enhanced agent configuration to a YAML file
func SaveAgentConfig(ctx context.Context, path string, config *EnhancedAgentConfig) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("config.agent_loader").
			WithOperation("SaveAgentConfig")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "config.agent_loader")
	ctx = observability.WithOperation(ctx, "SaveAgentConfig")

	logger.InfoContext(ctx, "Saving agent configuration", "path", path, "agent_id", config.ID)

	// Validate before saving
	if err := config.Validate(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "cannot save invalid agent configuration").
			WithComponent("config.agent_loader").
			WithOperation("SaveAgentConfig").
			WithDetails("path", path).
			WithDetails("agent_id", config.ID)
	}

	// Check context before marshaling
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before marshal").
			WithComponent("config.agent_loader").
			WithOperation("SaveAgentConfig")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create directory").
			WithComponent("config.agent_loader").
			WithOperation("SaveAgentConfig").
			WithDetails("path", path)
	}

	logger.DebugContext(ctx, "Marshaling configuration to YAML")

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeParsing, "failed to marshal configuration to YAML").
			WithComponent("config.agent_loader").
			WithOperation("SaveAgentConfig").
			WithDetails("path", path).
			WithDetails("agent_id", config.ID)
	}

	// Check context before file write
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before file write").
			WithComponent("config.agent_loader").
			WithOperation("SaveAgentConfig")
	}

	logger.DebugContext(ctx, "Writing configuration file", "size_bytes", len(data))

	// Write file
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write agent config file").
			WithComponent("config.agent_loader").
			WithOperation("SaveAgentConfig").
			WithDetails("path", path).
			WithDetails("agent_id", config.ID)
	}

	logger.InfoContext(ctx, "Agent configuration saved successfully", "path", path, "agent_id", config.ID)

	return nil
}

// LoadAgentConfigsFromDirectory loads all agent configurations from a directory
func LoadAgentConfigsFromDirectory(ctx context.Context, dirPath string) ([]*EnhancedAgentConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("config.agent_loader").
			WithOperation("LoadAgentConfigsFromDirectory")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "config.agent_loader")
	ctx = observability.WithOperation(ctx, "LoadAgentConfigsFromDirectory")

	logger.InfoContext(ctx, "Loading agent configurations from directory", "dir_path", dirPath)

	// Read directory
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "agent configuration directory not found").
				WithComponent("config.agent_loader").
				WithOperation("LoadAgentConfigsFromDirectory").
				WithDetails("dir_path", dirPath)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read agent configuration directory").
			WithComponent("config.agent_loader").
			WithOperation("LoadAgentConfigsFromDirectory").
			WithDetails("dir_path", dirPath)
	}

	var configs []*EnhancedAgentConfig
	var loadErrors []error

	// Load each YAML file
	for _, entry := range entries {
		// Check context in loop
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during directory loading").
				WithComponent("config.agent_loader").
				WithOperation("LoadAgentConfigsFromDirectory")
		}

		// Skip non-YAML files
		name := entry.Name()
		if !entry.Type().IsRegular() || (filepath.Ext(name) != ".yaml" && filepath.Ext(name) != ".yml") {
			logger.DebugContext(ctx, "Skipping non-YAML file", "filename", name)
			continue
		}

		filePath := filepath.Join(dirPath, name)
		logger.DebugContext(ctx, "Loading agent configuration file", "filepath", filePath)

		config, err := LoadAgentConfig(ctx, filePath)
		if err != nil {
			logger.WarnContext(ctx, "Failed to load agent configuration", "filepath", filePath, "error", err)
			loadErrors = append(loadErrors, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to load agent config").
				WithDetails("file", filePath))
			continue
		}

		configs = append(configs, config)
	}

	// If we have some configs but also errors, log warnings but continue
	if len(loadErrors) > 0 && len(configs) > 0 {
		logger.WarnContext(ctx, "Some agent configurations failed to load",
			"successful_count", len(configs),
			"failed_count", len(loadErrors))
	}

	// If all failed to load, return error
	if len(configs) == 0 && len(loadErrors) > 0 {
		return nil, gerror.New(gerror.ErrCodeValidation, "no valid agent configurations found", nil).
			WithComponent("config.agent_loader").
			WithOperation("LoadAgentConfigsFromDirectory").
			WithDetails("dir_path", dirPath).
			WithDetails("error_count", len(loadErrors))
	}

	logger.InfoContext(ctx, "Agent configurations loaded from directory",
		"dir_path", dirPath,
		"config_count", len(configs))

	return configs, nil
}

// ValidateAgentConfigFile validates an agent configuration file without fully loading it
func ValidateAgentConfigFile(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("config.agent_loader").
			WithOperation("ValidateAgentConfigFile")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "config.agent_loader")
	ctx = observability.WithOperation(ctx, "ValidateAgentConfigFile")

	logger.DebugContext(ctx, "Validating agent configuration file", "path", path)

	// Try to load the config (this will validate it)
	_, err := LoadAgentConfig(ctx, path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "agent configuration file validation failed").
			WithComponent("config.agent_loader").
			WithOperation("ValidateAgentConfigFile").
			WithDetails("path", path)
	}

	logger.DebugContext(ctx, "Agent configuration file validation passed", "path", path)
	return nil
}
