// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package observability provides production-ready logging, metrics, and tracing for Guild.
package observability

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/paths"
)

// LogLevel represents logging levels
type LogLevel = slog.Level

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// Logger is the Guild framework logger interface
type Logger interface {
	// Basic logging methods
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)

	// Context-aware logging
	DebugContext(ctx context.Context, msg string, fields ...any)
	InfoContext(ctx context.Context, msg string, fields ...any)
	WarnContext(ctx context.Context, msg string, fields ...any)
	ErrorContext(ctx context.Context, msg string, fields ...any)

	// With methods for adding persistent fields
	With(fields ...any) Logger
	WithContext(ctx context.Context) Logger
	WithError(err error) Logger
	WithComponent(component string) Logger
	WithOperation(operation string) Logger

	// Performance logging
	Duration(operation string, duration time.Duration, fields ...any)
	DurationContext(ctx context.Context, operation string, duration time.Duration, fields ...any)
}

// GuildLogger implements the Logger interface with production features
type GuildLogger struct {
	slogger   *slog.Logger
	component string
	operation string
}

// Config holds logger configuration
type Config struct {
	Level       LogLevel
	Format      string // "json" or "text"
	Output      io.Writer
	AddSource   bool
	Environment string
	Service     string
	Version     string
	EnableFile  bool // Enable file logging to .guild/logs/
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	// For CLI tools, logs should go to files only, not console
	// Use /dev/null as default output to avoid console pollution
	var defaultOutput io.Writer = os.Stdout
	if devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0); err == nil {
		defaultOutput = devNull
	}

	return &Config{
		Level:       LevelInfo,
		Format:      "json",
		Output:      defaultOutput,
		AddSource:   true,
		Environment: getEnv("GUILD_ENV", "development"),
		Service:     getEnv("GUILD_SERVICE", "guild"),
		Version:     getEnv("GUILD_VERSION", "unknown"),
		EnableFile:  true, // Enable file logging by default for CLI
	}
}

// NewLogger creates a new Guild logger
func NewLogger(config *Config) Logger {
	if config == nil {
		config = DefaultConfig()
	}

	opts := &slog.HandlerOptions{
		Level:     config.Level,
		AddSource: config.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Add custom formatting for certain attributes
			switch a.Key {
			case slog.TimeKey:
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339Nano))
			case slog.LevelKey:
				return slog.String("level", a.Value.Any().(slog.Level).String())
			}
			return a
		},
	}

	// Set up multi-writer for both console and file logging
	output := setupLogOutput(config.Output)

	var handler slog.Handler
	if config.Format == "text" {
		handler = slog.NewTextHandler(output, opts)
	} else {
		handler = slog.NewJSONHandler(output, opts)
	}

	// Add default fields
	slogger := slog.New(handler).With(
		"service", config.Service,
		"version", config.Version,
		"env", config.Environment,
	)

	return &GuildLogger{
		slogger: slogger,
	}
}

// setupLogOutput creates a writer that logs only to .guild/logs/ for CLI tools
func setupLogOutput(consoleOutput io.Writer) io.Writer {
	// For CLI tools, system logs should only go to files
	// User-facing messages are handled via fmt.Print* functions

	// Try to create log file
	if fileWriter := createLogFile(); fileWriter != nil {
		return fileWriter
	}

	// If file logging fails, use /dev/null to avoid console pollution
	if devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0); err == nil {
		return devNull
	}

	// Last resort fallback (should rarely happen)
	return consoleOutput
}

// createLogFile creates a log file in ~/.guild/logs/ directory
func createLogFile() io.Writer {
	// Get global Guild config directory
	guildDir, err := paths.GetGuildConfigDir()
	if err != nil {
		// If we can't get the Guild directory, skip file logging silently
		return nil
	}

	// Create logs subdirectory
	logDir := filepath.Join(guildDir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		// If we can't create the directory, skip file logging silently
		return nil
	}

	// Create log file with date
	date := time.Now().Format("2006-01-02")
	logFileName := fmt.Sprintf("guild-%s.log", date)
	logPath := filepath.Join(logDir, logFileName)

	// Open or create log file with append mode
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		// If we can't create the file, skip file logging silently
		return nil
	}

	// Create a symlink to latest log for easy access
	latestPath := filepath.Join(logDir, "latest.log")
	if err := os.Remove(latestPath); err != nil && !os.IsNotExist(err) {
		// Ignore errors - symlink creation is non-critical
		_ = gerror.Wrap(err, gerror.ErrCodeStorage, "failed to remove old symlink").
			WithComponent("logger").
			WithOperation("createLogFile")
	}
	if err := os.Symlink(logFileName, latestPath); err != nil {
		// Ignore errors - symlink creation is non-critical
		_ = gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create symlink").
			WithComponent("logger").
			WithOperation("createLogFile")
	}

	return logFile
}

// GetLogger gets a logger from context or returns default
func GetLogger(ctx context.Context) Logger {
	if logger, ok := ctx.Value("guild.logger").(Logger); ok {
		return logger
	}
	return NewLogger(nil)
}

// WithLogger adds a logger to context
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, "guild.logger", logger)
}

// Basic logging methods

func (l *GuildLogger) Debug(msg string, fields ...any) {
	l.log(context.Background(), LevelDebug, msg, fields...)
}

func (l *GuildLogger) Info(msg string, fields ...any) {
	l.log(context.Background(), LevelInfo, msg, fields...)
}

func (l *GuildLogger) Warn(msg string, fields ...any) {
	l.log(context.Background(), LevelWarn, msg, fields...)
}

func (l *GuildLogger) Error(msg string, fields ...any) {
	l.log(context.Background(), LevelError, msg, fields...)
}

// Context-aware logging methods

func (l *GuildLogger) DebugContext(ctx context.Context, msg string, fields ...any) {
	l.log(ctx, LevelDebug, msg, fields...)
}

func (l *GuildLogger) InfoContext(ctx context.Context, msg string, fields ...any) {
	l.log(ctx, LevelInfo, msg, fields...)
}

func (l *GuildLogger) WarnContext(ctx context.Context, msg string, fields ...any) {
	l.log(ctx, LevelWarn, msg, fields...)
}

func (l *GuildLogger) ErrorContext(ctx context.Context, msg string, fields ...any) {
	l.log(ctx, LevelError, msg, fields...)
}

// With methods

func (l *GuildLogger) With(fields ...any) Logger {
	return &GuildLogger{
		slogger:   l.slogger.With(fields...),
		component: l.component,
		operation: l.operation,
	}
}

func (l *GuildLogger) WithContext(ctx context.Context) Logger {
	fields := extractContextFields(ctx)
	return l.With(fields...)
}

func (l *GuildLogger) WithError(err error) Logger {
	fields := []any{"error", err.Error()}

	// Extract Guild error details
	var gerr *gerror.GuildError
	if gerror.As(err, &gerr) {
		fields = append(fields,
			"error_code", gerr.Code,
			"error_component", gerr.Component,
			"error_operation", gerr.Operation,
			"error_retryable", gerr.Retryable,
		)

		if gerr.Details != nil {
			fields = append(fields, "error_details", gerr.Details)
		}

		if len(gerr.Stack) > 0 && l.slogger.Enabled(context.Background(), LevelDebug) {
			// Only include stack trace in debug mode
			fields = append(fields, "error_stack", gerr.Stack)
		}
	}

	return l.With(fields...)
}

func (l *GuildLogger) WithComponent(component string) Logger {
	return &GuildLogger{
		slogger:   l.slogger.With("component", component),
		component: component,
		operation: l.operation,
	}
}

func (l *GuildLogger) WithOperation(operation string) Logger {
	return &GuildLogger{
		slogger:   l.slogger.With("operation", operation),
		component: l.component,
		operation: operation,
	}
}

// Performance logging

func (l *GuildLogger) Duration(operation string, duration time.Duration, fields ...any) {
	l.DurationContext(context.Background(), operation, duration, fields...)
}

func (l *GuildLogger) DurationContext(ctx context.Context, operation string, duration time.Duration, fields ...any) {
	allFields := append(fields,
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
		"duration_human", duration.String(),
	)

	level := LevelInfo
	if duration > 5*time.Second {
		level = LevelWarn
		allFields = append(allFields, "slow_operation", true)
	}

	l.log(ctx, level, fmt.Sprintf("Operation completed: %s", operation), allFields...)
}

// Internal logging method
func (l *GuildLogger) log(ctx context.Context, level LogLevel, msg string, fields ...any) {
	// Add component and operation if set
	if l.component != "" {
		fields = append([]any{"component", l.component}, fields...)
	}
	if l.operation != "" {
		fields = append([]any{"operation", l.operation}, fields...)
	}

	// Extract context fields
	contextFields := extractContextFields(ctx)
	fields = append(contextFields, fields...)

	// Add caller information for errors
	if level >= LevelError {
		if pc, file, line, ok := runtime.Caller(3); ok {
			fn := runtime.FuncForPC(pc)
			fields = append(fields,
				"caller", fmt.Sprintf("%s:%d", file, line),
				"function", fn.Name(),
			)
		}
	}

	l.slogger.LogAttrs(ctx, level, msg, fieldsToAttrs(fields)...)
}

// extractContextFields extracts standard fields from context
func extractContextFields(ctx context.Context) []any {
	var fields []any

	// Standard context values
	if requestID, ok := ctx.Value("request_id").(string); ok {
		fields = append(fields, "request_id", requestID)
	}
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		fields = append(fields, "trace_id", traceID)
	}
	if spanID, ok := ctx.Value("span_id").(string); ok {
		fields = append(fields, "span_id", spanID)
	}
	if userID, ok := ctx.Value("user_id").(string); ok {
		fields = append(fields, "user_id", userID)
	}
	if sessionID, ok := ctx.Value("session_id").(string); ok {
		fields = append(fields, "session_id", sessionID)
	}

	// Guild-specific context
	if agentID, ok := ctx.Value("agent_id").(string); ok {
		fields = append(fields, "agent_id", agentID)
	}
	if taskID, ok := ctx.Value("task_id").(string); ok {
		fields = append(fields, "task_id", taskID)
	}
	if commissionID, ok := ctx.Value("commission_id").(string); ok {
		fields = append(fields, "commission_id", commissionID)
	}
	if campaignID, ok := ctx.Value("campaign_id").(string); ok {
		fields = append(fields, "campaign_id", campaignID)
	}

	return fields
}

// fieldsToAttrs converts field pairs to slog attributes
func fieldsToAttrs(fields []any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(fields)/2)

	for i := 0; i < len(fields)-1; i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}

		attrs = append(attrs, slog.Any(key, fields[i+1]))
	}

	return attrs
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper functions for common logging patterns

// LogError logs an error with appropriate context
func LogError(ctx context.Context, err error, msg string, fields ...any) {
	logger := GetLogger(ctx)
	logger.WithError(err).ErrorContext(ctx, msg, fields...)
}

// LogOperation logs the start and end of an operation with duration
func LogOperation(ctx context.Context, operation string, fn func() error) error {
	logger := GetLogger(ctx)
	logger.InfoContext(ctx, fmt.Sprintf("Starting operation: %s", operation))

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	if err != nil {
		logger.WithError(err).ErrorContext(ctx,
			fmt.Sprintf("Operation failed: %s", operation),
			"duration_ms", duration.Milliseconds(),
		)
	} else {
		logger.DurationContext(ctx, operation, duration)
	}

	return err
}
