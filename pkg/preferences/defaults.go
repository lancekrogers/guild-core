// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package preferences

import "github.com/guild-framework/guild-core/pkg/gerror"

// DefaultPreferences defines system-wide default values for preferences
var DefaultPreferences = map[string]interface{}{
	// UI Preferences
	"ui.theme":            "dark",
	"ui.language":         "en",
	"ui.fontSize":         14,
	"ui.showLineNumbers":  true,
	"ui.wordWrap":         false,
	"ui.autoSave":         true,
	"ui.autoSaveInterval": 300, // seconds

	// Agent Preferences
	"agent.maxConcurrent": 5,
	"agent.timeout":       3600, // seconds
	"agent.retryAttempts": 3,
	"agent.retryDelay":    5, // seconds
	"agent.verbose":       false,
	"agent.autoAssign":    true,

	// Guild Preferences
	"guild.maxAgents":           10,
	"guild.coordinationMode":    "collaborative",
	"guild.loadBalancing":       "round-robin",
	"guild.healthCheckInterval": 60, // seconds

	// Memory Preferences
	"memory.maxWorkingSize":     1000, // items
	"memory.promotionThreshold": 0.8,  // importance score
	"memory.retentionDays":      30,   // days
	"memory.compressionEnabled": true,
	"memory.vectorDimensions":   1536, // OpenAI ada-002 dimensions

	// Session Preferences
	"session.autoRestore":        true,
	"session.checkpointInterval": 300, // seconds
	"session.maxHistory":         100, // messages
	"session.compressionEnabled": true,
	"session.encryptionEnabled":  false,

	// Provider Preferences
	"provider.default":     "openai",
	"provider.maxRetries":  3,
	"provider.timeout":     30, // seconds
	"provider.temperature": 0.7,
	"provider.maxTokens":   4096,

	// Development Preferences
	"dev.debug":          false,
	"dev.logLevel":       "info",
	"dev.profiling":      false,
	"dev.metricsEnabled": true,
	"dev.tracingEnabled": false,
}

// PreferenceTypes defines the expected type for each preference key
var PreferenceTypes = map[string]string{
	// UI Types
	"ui.theme":            "string",
	"ui.language":         "string",
	"ui.fontSize":         "int",
	"ui.showLineNumbers":  "bool",
	"ui.wordWrap":         "bool",
	"ui.autoSave":         "bool",
	"ui.autoSaveInterval": "int",

	// Agent Types
	"agent.maxConcurrent": "int",
	"agent.timeout":       "int",
	"agent.retryAttempts": "int",
	"agent.retryDelay":    "int",
	"agent.verbose":       "bool",
	"agent.autoAssign":    "bool",

	// Guild Types
	"guild.maxAgents":           "int",
	"guild.coordinationMode":    "string",
	"guild.loadBalancing":       "string",
	"guild.healthCheckInterval": "int",

	// Memory Types
	"memory.maxWorkingSize":     "int",
	"memory.promotionThreshold": "float",
	"memory.retentionDays":      "int",
	"memory.compressionEnabled": "bool",
	"memory.vectorDimensions":   "int",

	// Session Types
	"session.autoRestore":        "bool",
	"session.checkpointInterval": "int",
	"session.maxHistory":         "int",
	"session.compressionEnabled": "bool",
	"session.encryptionEnabled":  "bool",

	// Provider Types
	"provider.default":     "string",
	"provider.maxRetries":  "int",
	"provider.timeout":     "int",
	"provider.temperature": "float",
	"provider.maxTokens":   "int",

	// Development Types
	"dev.debug":          "bool",
	"dev.logLevel":       "string",
	"dev.profiling":      "bool",
	"dev.metricsEnabled": "bool",
	"dev.tracingEnabled": "bool",
}

// PreferenceValidators defines custom validation functions for specific preferences
var PreferenceValidators = map[string]func(value interface{}) error{
	"ui.theme": func(value interface{}) error {
		theme, ok := value.(string)
		if !ok {
			return gerror.New(gerror.ErrCodeInvalidInput, "theme must be a string", nil)
		}
		validThemes := []string{"light", "dark", "auto"}
		for _, valid := range validThemes {
			if theme == valid {
				return nil
			}
		}
		return gerror.New(gerror.ErrCodeInvalidInput, "invalid theme", nil).
			WithDetails("valid_themes", validThemes)
	},

	"ui.language": func(value interface{}) error {
		lang, ok := value.(string)
		if !ok {
			return gerror.New(gerror.ErrCodeInvalidInput, "language must be a string", nil)
		}
		// Basic ISO 639-1 validation
		if len(lang) != 2 {
			return gerror.New(gerror.ErrCodeInvalidInput, "language must be a 2-letter ISO code", nil)
		}
		return nil
	},

	"ui.fontSize": func(value interface{}) error {
		var size int
		switch v := value.(type) {
		case int:
			size = v
		case float64:
			size = int(v)
		default:
			return gerror.New(gerror.ErrCodeInvalidInput, "fontSize must be a number", nil)
		}
		if size < 8 || size > 32 {
			return gerror.New(gerror.ErrCodeInvalidInput, "fontSize must be between 8 and 32", nil)
		}
		return nil
	},

	"guild.coordinationMode": func(value interface{}) error {
		mode, ok := value.(string)
		if !ok {
			return gerror.New(gerror.ErrCodeInvalidInput, "coordinationMode must be a string", nil)
		}
		validModes := []string{"collaborative", "hierarchical", "autonomous"}
		for _, valid := range validModes {
			if mode == valid {
				return nil
			}
		}
		return gerror.New(gerror.ErrCodeInvalidInput, "invalid coordination mode", nil).
			WithDetails("valid_modes", validModes)
	},

	"guild.loadBalancing": func(value interface{}) error {
		strategy, ok := value.(string)
		if !ok {
			return gerror.New(gerror.ErrCodeInvalidInput, "loadBalancing must be a string", nil)
		}
		validStrategies := []string{"round-robin", "least-loaded", "random", "weighted"}
		for _, valid := range validStrategies {
			if strategy == valid {
				return nil
			}
		}
		return gerror.New(gerror.ErrCodeInvalidInput, "invalid load balancing strategy", nil).
			WithDetails("valid_strategies", validStrategies)
	},

	"memory.promotionThreshold": func(value interface{}) error {
		var threshold float64
		switch v := value.(type) {
		case float64:
			threshold = v
		case float32:
			threshold = float64(v)
		case int:
			threshold = float64(v)
		default:
			return gerror.New(gerror.ErrCodeInvalidInput, "promotionThreshold must be a number", nil)
		}
		if threshold < 0 || threshold > 1 {
			return gerror.New(gerror.ErrCodeInvalidInput, "promotionThreshold must be between 0 and 1", nil)
		}
		return nil
	},

	"provider.temperature": func(value interface{}) error {
		var temp float64
		switch v := value.(type) {
		case float64:
			temp = v
		case float32:
			temp = float64(v)
		case int:
			temp = float64(v)
		default:
			return gerror.New(gerror.ErrCodeInvalidInput, "temperature must be a number", nil)
		}
		if temp < 0 || temp > 2 {
			return gerror.New(gerror.ErrCodeInvalidInput, "temperature must be between 0 and 2", nil)
		}
		return nil
	},

	"dev.logLevel": func(value interface{}) error {
		level, ok := value.(string)
		if !ok {
			return gerror.New(gerror.ErrCodeInvalidInput, "logLevel must be a string", nil)
		}
		validLevels := []string{"debug", "info", "warn", "error"}
		for _, valid := range validLevels {
			if level == valid {
				return nil
			}
		}
		return gerror.New(gerror.ErrCodeInvalidInput, "invalid log level", nil).
			WithDetails("valid_levels", validLevels)
	},
}
