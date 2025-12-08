package logging

import (
	"io"
	"log/slog"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Config defines the logger configuration
type Config struct {
	// Level is the minimum log level
	Level slog.Level

	// Format specifies the output format: "json", "pretty", "text"
	Format string

	// Output specifies where logs go: "stdout", "file", "multi"
	Output string

	// Writer allows custom output writer (optional)
	Writer io.Writer

	// File output settings
	FilePath   string
	MaxSize    int64 // Max size per file in bytes
	MaxAge     int   // Days to keep
	MaxBackups int   // Number of backups
	Compress   bool  // Gzip old logs

	// Sampling configuration
	Sampling SamplingConfig

	// Hooks for log processing
	Hooks []Hook

	// Security settings
	EnableSensitive bool // Enable PII scrubbing

	// Development settings
	AddSource   bool // Add source file information
	Development bool // Enable development mode features
}

// SamplingConfig controls log sampling behavior
type SamplingConfig struct {
	Enabled bool
	Type    string // "level", "rate", "adaptive"

	// Level-based sampling
	DebugRate float64
	InfoRate  float64

	// Rate-based sampling
	Rate float64

	// Adaptive sampling
	TargetRate int
	Window     time.Duration
}

// DefaultConfig returns a production-ready configuration
func DefaultConfig() Config {
	return Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Sampling: SamplingConfig{
			Enabled:   true,
			Type:      "level",
			DebugRate: 0.1, // Sample 10% of debug logs
			InfoRate:  1.0, // Keep all info logs
		},
		EnableSensitive: true,
		Development:     false,
	}
}

// DevelopmentConfig returns a developer-friendly configuration
func DevelopmentConfig() Config {
	return Config{
		Level:       slog.LevelDebug,
		Format:      "pretty",
		Output:      "stdout",
		AddSource:   true,
		Development: true,
		Sampling: SamplingConfig{
			Enabled: false, // No sampling in development
		},
		EnableSensitive: false, // Show all data in development
	}
}

// ProductionConfig returns a production configuration with file output
func ProductionConfig(logPath string) Config {
	return Config{
		Level:      slog.LevelInfo,
		Format:     "json",
		Output:     "multi",
		FilePath:   logPath,
		MaxSize:    100 * 1024 * 1024, // 100MB
		MaxAge:     30,                // 30 days
		MaxBackups: 10,
		Compress:   true,
		Sampling: SamplingConfig{
			Enabled:    true,
			Type:       "adaptive",
			TargetRate: 1000, // Target 1000 logs/sec
			Window:     time.Minute,
		},
		EnableSensitive: true,
		Development:     false,
	}
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	// Validate format
	switch c.Format {
	case "json", "pretty", "text":
		// Valid formats
	default:
		return gerror.New(gerror.ErrCodeInvalidInput, "invalid log format", nil).
			WithDetails("format", c.Format).
			WithDetails("valid", []string{"json", "pretty", "text"})
	}

	// Validate output
	switch c.Output {
	case "stdout", "file", "multi":
		// Valid outputs
	default:
		return gerror.New(gerror.ErrCodeInvalidInput, "invalid log output", nil).
			WithDetails("output", c.Output).
			WithDetails("valid", []string{"stdout", "file", "multi"})
	}

	// Validate file settings if using file output
	if c.Output == "file" || c.Output == "multi" {
		if c.FilePath == "" && c.Output == "file" {
			return gerror.New(gerror.ErrCodeMissingRequired, "file path required for file output", nil)
		}
		if c.MaxSize <= 0 {
			c.MaxSize = 100 * 1024 * 1024 // Default 100MB
		}
		if c.MaxAge <= 0 {
			c.MaxAge = 30 // Default 30 days
		}
		if c.MaxBackups < 0 {
			c.MaxBackups = 10 // Default 10 backups
		}
	}

	// Validate sampling
	if c.Sampling.Enabled {
		switch c.Sampling.Type {
		case "level":
			if c.Sampling.DebugRate < 0 || c.Sampling.DebugRate > 1 {
				return gerror.New(gerror.ErrCodeInvalidInput, "invalid debug sampling rate", nil).
					WithDetails("rate", c.Sampling.DebugRate)
			}
			if c.Sampling.InfoRate < 0 || c.Sampling.InfoRate > 1 {
				return gerror.New(gerror.ErrCodeInvalidInput, "invalid info sampling rate", nil).
					WithDetails("rate", c.Sampling.InfoRate)
			}
		case "rate":
			if c.Sampling.Rate < 0 || c.Sampling.Rate > 1 {
				return gerror.New(gerror.ErrCodeInvalidInput, "invalid sampling rate", nil).
					WithDetails("rate", c.Sampling.Rate)
			}
		case "adaptive":
			if c.Sampling.TargetRate <= 0 {
				return gerror.New(gerror.ErrCodeInvalidInput, "invalid target rate", nil).
					WithDetails("rate", c.Sampling.TargetRate)
			}
			if c.Sampling.Window <= 0 {
				c.Sampling.Window = time.Minute // Default 1 minute window
			}
		default:
			return gerror.New(gerror.ErrCodeInvalidInput, "invalid sampling type", nil).
				WithDetails("type", c.Sampling.Type).
				WithDetails("valid", []string{"level", "rate", "adaptive"})
		}
	}

	return nil
}

// ConfigOption is a functional option for modifying Config
type ConfigOption func(*Config)

// WithLevel sets the log level
func WithLevel(level slog.Level) ConfigOption {
	return func(c *Config) {
		c.Level = level
	}
}

// WithFormat sets the output format
func WithFormat(format string) ConfigOption {
	return func(c *Config) {
		c.Format = format
	}
}

// WithOutput sets the output destination
func WithOutput(output string) ConfigOption {
	return func(c *Config) {
		c.Output = output
	}
}

// WithFile configures file output
func WithFile(path string, maxSize int64, maxAge, maxBackups int) ConfigOption {
	return func(c *Config) {
		c.FilePath = path
		c.MaxSize = maxSize
		c.MaxAge = maxAge
		c.MaxBackups = maxBackups
	}
}

// WithSampling configures log sampling
func WithSampling(enabled bool, samplingType string) ConfigOption {
	return func(c *Config) {
		c.Sampling.Enabled = enabled
		c.Sampling.Type = samplingType
	}
}

// WithDevelopment enables development mode
func WithDevelopment(enabled bool) ConfigOption {
	return func(c *Config) {
		c.Development = enabled
		if enabled {
			c.AddSource = true
			c.EnableSensitive = false
		}
	}
}

// NewConfig creates a config with options
func NewConfig(opts ...ConfigOption) Config {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
