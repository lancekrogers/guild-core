package logging

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestPrettyHandlerMultilineFormat(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
		Level:     slog.LevelDebug,
		UseColor:  false,
		Multiline: true,
	})

	// Create a record with multiple attributes to trigger multiline formatting
	r := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "Complex log message",
	}

	// Add many attributes to trigger multiline formatting
	for i := 0; i < 10; i++ {
		r.AddAttrs(slog.String("key", "value"))
	}

	err := handler.Handle(context.Background(), r)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected output from multiline handler")
	}
}

func TestPrettyHandlerWithGroups(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
		Level:    slog.LevelDebug,
		UseColor: false,
	})

	// Create nested groups to test group handling
	handler2 := handler.WithGroup("service")
	handler3 := handler2.WithGroup("database")
	handler4 := handler3.WithAttrs([]slog.Attr{
		slog.String("query", "SELECT * FROM users"),
		slog.Int("rows", 42),
	})

	r := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "Database query executed",
	}

	err := handler4.Handle(context.Background(), r)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected output from grouped handler")
	}
}

func TestTextFormatterWithPC(t *testing.T) {
	formatter := NewTextFormatter()

	// Test with PC set to include source information
	r := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "Test message with source",
		PC:      1, // Non-zero PC to trigger source formatting
	}

	output, err := formatter.Format(&r)
	if err != nil {
		t.Errorf("Format error: %v", err)
	}

	if len(output) == 0 {
		t.Error("Expected output from formatter with PC")
	}
}

func TestHandlerErrorPaths(t *testing.T) {
	// Test error handling in various handlers
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
		Level:    slog.LevelDebug,
		UseColor: true,
	})

	// Test with complex values that might cause formatting issues
	r := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "Test complex values",
	}

	// Add complex values
	r.AddAttrs(
		slog.Any("nil_value", nil),
		slog.Any("complex_map", map[string]interface{}{
			"nested": map[string]interface{}{
				"deep": "value",
			},
		}),
		slog.Any("slice", []interface{}{1, "string", true}),
	)

	err := handler.Handle(context.Background(), r)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}
}
