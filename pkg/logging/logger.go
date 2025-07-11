package logging

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// Logger provides structured logging with context propagation
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger
}

// Field is an alias for slog.Attr for consistent API
type Field = slog.Attr

// Common field constructors
func String(key, value string) Field                 { return slog.String(key, value) }
func Int(key string, value int) Field                { return slog.Int(key, value) }
func Int64(key string, value int64) Field            { return slog.Int64(key, value) }
func Bool(key string, value bool) Field              { return slog.Bool(key, value) }
func Duration(key string, value time.Duration) Field { return slog.Duration(key, value) }
func Time(key string, value time.Time) Field         { return slog.Time(key, value) }
func Any(key string, value any) Field                { return slog.Any(key, value) }

// Error field constructor for consistent error logging
func ErrorField(err error) Field {
	if err == nil {
		return slog.Any("error", nil)
	}
	// Extract gerror fields if available
	if gErr, ok := err.(*gerror.GuildError); ok {
		details := gErr.Details
		// Convert map to attrs
		attrs := make([]any, 0, len(details)*2+2)
		attrs = append(attrs, "error", gErr.Error())
		for k, v := range details {
			attrs = append(attrs, k, v)
		}
		return slog.Group("error_details", attrs...)
	}
	return slog.Any("error", err)
}

// Hook processes log records before output
type Hook interface {
	Process(record *slog.Record) *slog.Record
}

// Sampler determines if a log record should be output
type Sampler interface {
	Sample(level slog.Level, msg string) bool
}

// guildLogger implements the Logger interface
type guildLogger struct {
	base    *slog.Logger
	fields  []Field
	hooks   []Hook
	sampler Sampler
	mu      sync.RWMutex
}

// New creates a new logger instance
func New(config Config) (Logger, error) {
	if err := config.Validate(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConfiguration, "invalid logger config")
	}

	// Create handler based on config
	handler, err := createHandler(config)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create handler")
	}

	// Wrap handler with hooks if configured
	if len(config.Hooks) > 0 || config.EnableSensitive {
		handler = &hookHandler{
			Handler: handler,
			hooks:   config.Hooks,
		}
	}

	// Create base slog logger
	base := slog.New(handler)

	// Configure sampler
	var sampler Sampler
	if config.Sampling.Enabled {
		sampler, err = createSampler(config.Sampling)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeConfiguration, "failed to create sampler")
		}
	}

	return &guildLogger{
		base:    base,
		hooks:   config.Hooks,
		sampler: sampler,
	}, nil
}

// Debug logs at debug level
func (l *guildLogger) Debug(msg string, fields ...Field) {
	if l.shouldSample(slog.LevelDebug, msg) {
		l.log(slog.LevelDebug, msg, fields...)
	}
}

// Info logs at info level
func (l *guildLogger) Info(msg string, fields ...Field) {
	if l.shouldSample(slog.LevelInfo, msg) {
		l.log(slog.LevelInfo, msg, fields...)
	}
}

// Warn logs at warn level
func (l *guildLogger) Warn(msg string, fields ...Field) {
	if l.shouldSample(slog.LevelWarn, msg) {
		l.log(slog.LevelWarn, msg, fields...)
	}
}

// Error logs at error level
func (l *guildLogger) Error(msg string, fields ...Field) {
	if l.shouldSample(slog.LevelError, msg) {
		l.log(slog.LevelError, msg, fields...)
	}
}

// With returns a new logger with additional fields
func (l *guildLogger) With(fields ...Field) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := make([]Field, 0, len(l.fields)+len(fields))
	newFields = append(newFields, l.fields...)
	newFields = append(newFields, fields...)

	return &guildLogger{
		base:    l.base.With(fieldsToArgs(fields)...),
		fields:  newFields,
		hooks:   l.hooks,
		sampler: l.sampler,
	}
}

// WithContext returns a new logger with context fields
func (l *guildLogger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	fields := extractContextFields(ctx)
	if len(fields) == 0 {
		return l
	}

	return l.With(fields...)
}

// Internal methods

func (l *guildLogger) log(level slog.Level, msg string, fields ...Field) {
	l.mu.RLock()
	allFields := make([]Field, 0, len(l.fields)+len(fields))
	allFields = append(allFields, l.fields...)
	allFields = append(allFields, fields...)
	l.mu.RUnlock()

	// Convert fields to args for slog
	args := fieldsToArgs(allFields)

	// Log with appropriate level
	switch level {
	case slog.LevelDebug:
		l.base.Debug(msg, args...)
	case slog.LevelInfo:
		l.base.Info(msg, args...)
	case slog.LevelWarn:
		l.base.Warn(msg, args...)
	case slog.LevelError:
		l.base.Error(msg, args...)
	}
}

func (l *guildLogger) shouldSample(level slog.Level, msg string) bool {
	if l.sampler == nil {
		return true
	}
	return l.sampler.Sample(level, msg)
}

// hookHandler wraps a handler to apply hooks
type hookHandler struct {
	slog.Handler
	hooks []Hook
}

func (h *hookHandler) Handle(ctx context.Context, record slog.Record) error {
	// Apply hooks in order
	r := &record
	for _, hook := range h.hooks {
		r = hook.Process(r)
		if r == nil {
			// Hook filtered out the record
			return nil
		}
	}
	return h.Handler.Handle(ctx, *r)
}

// Helper functions

func fieldsToArgs(fields []Field) []any {
	args := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		args = append(args, f.Key, f.Value.Any())
	}
	return args
}

func createHandler(config Config) (slog.Handler, error) {
	// Create writer based on output config
	var w io.Writer
	switch config.Output {
	case "stdout":
		w = config.Writer
		if w == nil {
			w = io.Writer(colorWriter{})
		}
	case "file":
		writer, err := createFileWriter(config)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create file writer")
		}
		w = writer
	case "multi":
		writers := []io.Writer{colorWriter{}}
		if config.FilePath != "" {
			fileWriter, err := createFileWriter(config)
			if err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create file writer")
			}
			writers = append(writers, fileWriter)
		}
		w = io.MultiWriter(writers...)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported output type", nil).
			WithDetails("output", config.Output)
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level:     config.Level,
		AddSource: config.AddSource,
	}

	// Create handler based on format
	switch config.Format {
	case "json":
		return slog.NewJSONHandler(w, opts), nil
	case "text":
		return slog.NewTextHandler(w, opts), nil
	case "pretty":
		return NewPrettyHandler(w, &PrettyHandlerOptions{
			Level:     config.Level,
			AddSource: config.AddSource,
			UseColor:  config.Development,
			Multiline: config.Development,
		}), nil
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported format", nil).
			WithDetails("format", config.Format)
	}
}

func createFileWriter(config Config) (io.Writer, error) {
	if config.FilePath == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "file path required for file output", nil)
	}

	// For now, return a simple file writer
	// TODO: Implement rotating writer
	return colorWriter{}, nil
}

func createSampler(config SamplingConfig) (Sampler, error) {
	switch config.Type {
	case "level":
		return &LevelSampler{
			debugRate: config.DebugRate,
			infoRate:  config.InfoRate,
		}, nil
	case "rate":
		return &RateSampler{
			rate: config.Rate,
		}, nil
	case "adaptive":
		return &AdaptiveSampler{
			targetRate: config.TargetRate,
			window:     config.Window,
		}, nil
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported sampler type", nil).
			WithDetails("type", config.Type)
	}
}

// colorWriter is a placeholder for colored output
type colorWriter struct{}

func (colorWriter) Write(p []byte) (n int, err error) {
	// TODO: Implement colored output
	return len(p), nil
}
