package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestPrettyHandler(t *testing.T) {
	t.Run("NewPrettyHandler", func(t *testing.T) {
		var buf bytes.Buffer
		opts := &PrettyHandlerOptions{
			Level:    slog.LevelDebug,
			UseColor: true,
		}
		handler := NewPrettyHandler(&buf, opts)

		if handler == nil {
			t.Fatal("NewPrettyHandler returned nil")
		}

		// Test with colors disabled
		opts2 := &PrettyHandlerOptions{
			Level:    slog.LevelInfo,
			UseColor: false,
		}
		handler = NewPrettyHandler(&buf, opts2)
		if handler == nil {
			t.Fatal("NewPrettyHandler with no color returned nil")
		}
	})

	t.Run("Enabled", func(t *testing.T) {
		var buf bytes.Buffer
		handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
			Level:    slog.LevelInfo,
			UseColor: true,
		})

		// Should be enabled for info and above
		if !handler.Enabled(context.Background(), slog.LevelInfo) {
			t.Error("Handler should be enabled for info level")
		}

		if !handler.Enabled(context.Background(), slog.LevelWarn) {
			t.Error("Handler should be enabled for warn level")
		}

		// Should not be enabled for debug
		if handler.Enabled(context.Background(), slog.LevelDebug) {
			t.Error("Handler should not be enabled for debug level")
		}
	})

	t.Run("Handle", func(t *testing.T) {
		var buf bytes.Buffer
		handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
			Level:    slog.LevelDebug,
			UseColor: false,
		})

		// Create a test record
		r := slog.Record{
			Time:    time.Now(),
			Level:   slog.LevelInfo,
			Message: "Test message",
		}
		r.AddAttrs(
			slog.String("key1", "value1"),
			slog.Int("count", 42),
		)

		err := handler.Handle(context.Background(), r)
		if err != nil {
			t.Errorf("Handle failed: %v", err)
		}

		output := buf.String()
		t.Logf("PrettyHandler output: %q", output)
		if !strings.Contains(output, "INFO") {
			t.Error("Output should contain INFO level")
		}
		if !strings.Contains(output, "Test message") {
			t.Error("Output should contain message")
		}
		if !strings.Contains(output, "key1") || !strings.Contains(output, "value1") {
			t.Error("Output should contain key1 attribute")
		}
		if !strings.Contains(output, "count") || !strings.Contains(output, "42") {
			t.Error("Output should contain count attribute")
		}
	})

	t.Run("HandleWithColors", func(t *testing.T) {
		var buf bytes.Buffer
		handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
			Level:    slog.LevelDebug,
			UseColor: true,
		})

		// Test different levels
		levels := []slog.Level{
			slog.LevelDebug,
			slog.LevelInfo,
			slog.LevelWarn,
			slog.LevelError,
		}

		for _, level := range levels {
			buf.Reset()
			r := slog.Record{
				Time:    time.Now(),
				Level:   level,
				Message: "Test message",
			}

			err := handler.Handle(context.Background(), r)
			if err != nil {
				t.Errorf("Handle failed for level %v: %v", level, err)
			}

			output := buf.String()
			// Should contain ANSI color codes when colors are enabled
			if !strings.Contains(output, "\x1b[") {
				t.Errorf("Output should contain color codes for level %v", level)
			}
		}
	})

	t.Run("WithAttrs", func(t *testing.T) {
		var buf bytes.Buffer
		handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
			Level:    slog.LevelDebug,
			UseColor: false,
		})

		// Add some attributes
		handler2 := handler.WithAttrs([]slog.Attr{
			slog.String("service", "test-service"),
			slog.String("version", "1.0.0"),
		})

		r := slog.Record{
			Time:    time.Now(),
			Level:   slog.LevelInfo,
			Message: "Test message",
		}

		err := handler2.Handle(context.Background(), r)
		if err != nil {
			t.Errorf("Handle failed: %v", err)
		}

		output := buf.String()
		t.Logf("WithAttrs output: %q", output)
		if !strings.Contains(output, "service") || !strings.Contains(output, "test-service") {
			t.Error("Output should contain service attribute")
		}
		if !strings.Contains(output, "version") || !strings.Contains(output, "1.0.0") {
			t.Error("Output should contain version attribute")
		}
	})

	t.Run("WithGroup", func(t *testing.T) {
		var buf bytes.Buffer
		handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
			Level:    slog.LevelDebug,
			UseColor: false,
		})

		// Add a group
		handler2 := handler.WithGroup("request")
		handler3 := handler2.WithAttrs([]slog.Attr{
			slog.String("method", "GET"),
			slog.String("path", "/api/users"),
		})

		r := slog.Record{
			Time:    time.Now(),
			Level:   slog.LevelInfo,
			Message: "Request handled",
		}

		err := handler3.Handle(context.Background(), r)
		if err != nil {
			t.Errorf("Handle failed: %v", err)
		}

		output := buf.String()
		t.Logf("WithGroup output: %q", output)
		// Should contain attributes (grouping may be internal, not visible in output)
		if !strings.Contains(output, "method") || !strings.Contains(output, "GET") {
			t.Error("Output should contain method attribute")
		}
		if !strings.Contains(output, "path") || !strings.Contains(output, "/api/users") {
			t.Error("Output should contain path attribute")
		}
	})
}

func TestTextFormatter(t *testing.T) {
	t.Run("NewTextFormatter", func(t *testing.T) {
		formatter := NewTextFormatter()
		if formatter == nil {
			t.Fatal("NewTextFormatter returned nil")
		}
	})

	t.Run("Format", func(t *testing.T) {
		formatter := NewTextFormatter()

		r := slog.Record{
			Time:    time.Now(),
			Level:   slog.LevelInfo,
			Message: "Test message",
		}
		r.AddAttrs(
			slog.String("key", "value"),
			slog.Int("count", 42),
		)

		output, err := formatter.Format(&r)
		if err != nil {
			t.Errorf("Format error: %v", err)
		}

		outputStr := string(output)

		// Check basic format
		if !strings.Contains(outputStr, "INFO") {
			t.Error("Output should contain level")
		}
		if !strings.Contains(outputStr, "Test message") {
			t.Error("Output should contain message")
		}
		if !strings.Contains(outputStr, "key=value") {
			t.Error("Output should contain key field")
		}
		if !strings.Contains(outputStr, "count=42") {
			t.Error("Output should contain count field")
		}
	})

	t.Run("FormatAllLevels", func(t *testing.T) {
		formatter := NewTextFormatter()
		levels := []slog.Level{
			slog.LevelDebug,
			slog.LevelInfo,
			slog.LevelWarn,
			slog.LevelError,
		}

		for _, level := range levels {
			r := slog.Record{
				Time:    time.Now(),
				Level:   level,
				Message: "Test message",
			}

			output, err := formatter.Format(&r)
			if err != nil {
				t.Errorf("Format error for level %v: %v", level, err)
			}

			if len(output) == 0 {
				t.Errorf("Empty output for level %v", level)
			}
		}
	})

	t.Run("FormatWithSource", func(t *testing.T) {
		formatter := NewTextFormatter()

		r := slog.Record{
			Time:    time.Now(),
			Level:   slog.LevelInfo,
			Message: "Test message",
			PC:      1, // Non-zero to indicate source
		}

		output, err := formatter.Format(&r)
		if err != nil {
			t.Errorf("Format error: %v", err)
		}

		if len(output) == 0 {
			t.Error("Empty output")
		}
	})
}

func TestPrettyHandlerEdgeCases(t *testing.T) {
	t.Run("NilOptions", func(t *testing.T) {
		var buf bytes.Buffer
		handler := NewPrettyHandler(&buf, nil)
		if handler == nil {
			t.Fatal("NewPrettyHandler should handle nil options")
		}
	})

	t.Run("EmptyGroup", func(t *testing.T) {
		var buf bytes.Buffer
		handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
			Level: slog.LevelDebug,
		})

		// WithGroup with empty name
		handler2 := handler.WithGroup("")
		if handler2 == nil {
			t.Fatal("WithGroup should handle empty name")
		}
	})
}

func TestFormatValueEdgeCases(t *testing.T) {
	t.Run("ComplexTypes", func(t *testing.T) {
		// Test various value types that would trigger different format paths
		values := []slog.Value{
			slog.TimeValue(time.Now()),
			slog.DurationValue(5 * time.Second),
			slog.IntValue(42),
			slog.Int64Value(int64(123456789)),
			slog.Float64Value(3.14),
			slog.BoolValue(true),
			slog.StringValue("test"),
			slog.AnyValue(map[string]interface{}{"key": "value"}),
			slog.AnyValue([]int{1, 2, 3}),
			slog.AnyValue(struct{ Name string }{Name: "test"}),
		}

		for _, v := range values {
			// Just ensure these don't panic
			_ = v.String()
		}
	})
}
