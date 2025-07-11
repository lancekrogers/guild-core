package logging

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// captureWriter captures log output for testing
type captureWriter struct {
	buf bytes.Buffer
}

func (w *captureWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *captureWriter) String() string {
	return w.buf.String()
}

func TestLoggerCreation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name:    "development config",
			config:  DevelopmentConfig(),
			wantErr: false,
		},
		{
			name: "invalid format",
			config: Config{
				Format: "invalid",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "invalid output",
			config: Config{
				Format: "json",
				Output: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && logger == nil {
				t.Error("New() returned nil logger")
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	w := &captureWriter{}
	config := Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Writer: w,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Debug should not be logged
	logger.Debug("debug message")
	if strings.Contains(w.String(), "debug message") {
		t.Error("Debug message was logged when level is Info")
	}

	// Info should be logged
	logger.Info("info message")
	if !strings.Contains(w.String(), "info message") {
		t.Error("Info message was not logged")
	}

	// Warn should be logged
	w.buf.Reset()
	logger.Warn("warn message")
	if !strings.Contains(w.String(), "warn message") {
		t.Error("Warn message was not logged")
	}

	// Error should be logged
	w.buf.Reset()
	logger.Error("error message")
	if !strings.Contains(w.String(), "error message") {
		t.Error("Error message was not logged")
	}
}

func TestLoggerWithFields(t *testing.T) {
	w := &captureWriter{}
	config := Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Writer: w,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test With method
	logger2 := logger.With(
		String("service", "test-service"),
		Int("version", 1),
	)

	logger2.Info("test message")
	output := w.String()

	if !strings.Contains(output, `"service":"test-service"`) {
		t.Error("Service field not found in output")
	}
	if !strings.Contains(output, `"version":1`) {
		t.Error("Version field not found in output")
	}
}

func TestLoggerWithContext(t *testing.T) {
	w := &captureWriter{}
	config := Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Writer: w,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create context with various IDs
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithUserID(ctx, "user-456")
	ctx = WithSessionID(ctx, "session-789")

	// Log with context
	logger.WithContext(ctx).Info("context test")
	output := w.String()

	if !strings.Contains(output, `"request_id":"req-123"`) {
		t.Error("Request ID not found in output")
	}
	if !strings.Contains(output, `"user_id":"user-456"`) {
		t.Error("User ID not found in output")
	}
	if !strings.Contains(output, `"session_id":"session-789"`) {
		t.Error("Session ID not found in output")
	}
}

func TestErrorFieldHandling(t *testing.T) {
	w := &captureWriter{}
	config := Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Writer: w,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test with regular error
	regularErr := errors.New("regular error")
	logger.Error("test error", ErrorField(regularErr))
	output := w.String()
	if !strings.Contains(output, "regular error") {
		t.Error("Regular error message not found")
	}

	// Test with gerror
	w.buf.Reset()
	gErr := gerror.New(gerror.ErrCodeInternal, "gerror test", nil).
		WithDetails("code", "TEST001").
		WithDetails("retry", true)
	logger.Error("test gerror", ErrorField(gErr))
	output = w.String()
	if !strings.Contains(output, "gerror test") {
		t.Error("Gerror message not found")
	}
	if !strings.Contains(output, "TEST001") {
		t.Error("Gerror field 'code' not found")
	}

	// Test with nil error
	w.buf.Reset()
	logger.Info("test nil error", ErrorField(nil))
	output = w.String()
	if strings.Contains(output, "panic") {
		t.Error("Nil error caused panic")
	}
}

func TestCommonFields(t *testing.T) {
	w := &captureWriter{}
	config := Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Writer: w,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test various field types
	logger.Info("field test",
		String("string", "value"),
		Int("int", 42),
		Int64("int64", int64(123456789)),
		Bool("bool", true),
		Duration("duration", 5*time.Second),
		Time("time", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
		Any("any", map[string]int{"key": 1}),
	)

	output := w.String()
	expectedFields := []string{
		`"string":"value"`,
		`"int":42`,
		`"int64":123456789`,
		`"bool":true`,
		`"duration":5000000000`, // 5 seconds in nanoseconds
		`"time":"2023-01-01T00:00:00Z"`,
		`"any":{"key":1}`,
	}

	for _, expected := range expectedFields {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected field %s not found in output", expected)
		}
	}
}

func TestFieldSet(t *testing.T) {
	fs := NewFieldSet()

	// Test chaining
	fs.Add(String("key1", "value1")).
		AddIf(true, String("key2", "value2")).
		AddIf(false, String("key3", "value3")).
		AddString("key4", "value4").
		AddString("key5", ""). // Should not be added
		AddInt("key6", 42).
		AddInt("key7", 0). // Should not be added
		AddDuration("key8", 5*time.Second).
		AddDuration("key9", 0). // Should not be added
		AddError(errors.New("test error")).
		AddError(nil) // Should not be added

	fields := fs.Fields()

	// Check expected number of fields
	expectedCount := 6 // key1, key2, key4, key6, key8, error
	if len(fields) != expectedCount {
		t.Errorf("Expected %d fields, got %d", expectedCount, len(fields))
	}

	// Check specific fields
	fieldMap := make(map[string]bool)
	for _, f := range fields {
		fieldMap[f.Key] = true
	}

	if !fieldMap["key1"] {
		t.Error("key1 not found")
	}
	if !fieldMap["key2"] {
		t.Error("key2 not found")
	}
	if fieldMap["key3"] {
		t.Error("key3 should not be present")
	}
	if !fieldMap["key4"] {
		t.Error("key4 not found")
	}
	if fieldMap["key5"] {
		t.Error("key5 should not be present (empty string)")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Format: "json",
				Output: "stdout",
			},
			wantErr: false,
		},
		{
			name: "invalid format",
			config: Config{
				Format: "xml",
				Output: "stdout",
			},
			wantErr: true,
		},
		{
			name: "invalid output",
			config: Config{
				Format: "json",
				Output: "database",
			},
			wantErr: true,
		},
		{
			name: "file output without path",
			config: Config{
				Format:   "json",
				Output:   "file",
				FilePath: "",
			},
			wantErr: true,
		},
		{
			name: "invalid sampling rate",
			config: Config{
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled:   true,
					Type:      "level",
					DebugRate: 1.5, // Invalid: > 1
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigOptions(t *testing.T) {
	config := NewConfig(
		WithLevel(slog.LevelDebug),
		WithFormat("pretty"),
		WithOutput("file"),
		WithFile("/tmp/test.log", 1024*1024, 7, 5),
		WithSampling(true, "rate"),
		WithDevelopment(true),
	)

	if config.Level != slog.LevelDebug {
		t.Error("Level not set correctly")
	}
	if config.Format != "pretty" {
		t.Error("Format not set correctly")
	}
	if config.Output != "file" {
		t.Error("Output not set correctly")
	}
	if config.FilePath != "/tmp/test.log" {
		t.Error("FilePath not set correctly")
	}
	if !config.Development {
		t.Error("Development not set correctly")
	}
	if !config.AddSource {
		t.Error("AddSource should be true in development mode")
	}
}

func TestLoggerHandlerCreation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "text handler",
			config: Config{
				Level:  slog.LevelInfo,
				Format: "text",
				Output: "stdout",
			},
		},
		{
			name: "json handler",
			config: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
			},
		},
		{
			name: "pretty handler",
			config: Config{
				Level:  slog.LevelInfo,
				Format: "pretty",
				Output: "stdout",
			},
		},
		{
			name: "file output",
			config: Config{
				Level:    slog.LevelInfo,
				Format:   "json",
				Output:   "file",
				FilePath: "/tmp/test.log",
			},
		},
		{
			name: "multi output",
			config: Config{
				Level:    slog.LevelInfo,
				Format:   "json",
				Output:   "multi",
				FilePath: "/tmp/test.log",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if err != nil {
				t.Errorf("Failed to create logger: %v", err)
			}
		})
	}
}

func TestLoggerSamplerCreation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "level sampler",
			config: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled:   true,
					Type:      "level",
					DebugRate: 0.1,
					InfoRate:  0.5,
				},
			},
		},
		{
			name: "rate sampler",
			config: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled: true,
					Type:    "rate",
					Rate:    0.5,
				},
			},
		},
		{
			name: "adaptive sampler",
			config: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled:    true,
					Type:       "adaptive",
					TargetRate: 1000,
					Window:     time.Minute,
				},
			},
		},
		{
			name: "valid level sampler rates",
			config: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled:   true,
					Type:      "level",
					DebugRate: 0.1,
					InfoRate:  0.5,
				},
			},
		},
		{
			name: "no sampling",
			config: Config{
				Level:  slog.LevelInfo,
				Format: "json",
				Output: "stdout",
				Sampling: SamplingConfig{
					Enabled: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if err != nil {
				t.Errorf("Failed to create logger: %v", err)
			}
		})
	}
}

func TestLoggerWithContextNil(t *testing.T) {
	w := &captureWriter{}
	config := Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Writer: w,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test with nil context
	logger.WithContext(nil).Info("test message")
	output := w.String()
	if !strings.Contains(output, "test message") {
		t.Error("Message not found in output")
	}
}

func TestLoggerEdgeCases(t *testing.T) {
	// Test logger with custom writer
	w := &captureWriter{}
	config := Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Writer: w,
	}

	_, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test logger with source information
	config.AddSource = true
	logger2, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger with source: %v", err)
	}

	logger2.Info("test with source")
	output := w.String()
	if !strings.Contains(output, "test with source") {
		t.Error("Message not found in output")
	}
}
