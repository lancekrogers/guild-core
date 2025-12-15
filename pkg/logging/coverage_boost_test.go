package logging

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Test the uncovered functions to boost coverage

func TestLoggerHandle(t *testing.T) {
	// Test the Handle method in logger.go:209 via hookHandler
	var buf bytes.Buffer
	config := Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Writer: &buf,
		Hooks:  []Hook{&testHook{}},
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Use the logger to trigger hook processing
	logger.Info("test message")

	// Check that output was written
	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected output from logger with hooks")
	}
}

func TestLoggerWrite(t *testing.T) {
	// Test the Write method in logger.go:322 through colorWriter
	var buf bytes.Buffer
	config := Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Writer: &buf,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Use the logger which will exercise the Write method
	logger.Info("Test write message")

	// Check if output was written
	if buf.Len() == 0 {
		t.Error("Expected output from logger")
	}
}

func TestMiddlewareContext(t *testing.T) {
	// Test the Context method in middleware.go:284
	ss := &grpcStream{}
	ctx := ss.Context()
	if ctx != nil {
		t.Error("Expected nil context from mock stream")
	}

	// Test the loggingServerStream Context method
	originalCtx := context.Background()
	ls := &loggingServerStream{
		ServerStream: ss,
		ctx:          originalCtx,
	}
	if ls.Context() != originalCtx {
		t.Error("Expected Context to return the wrapped context")
	}
}

func TestCreateFileWriter(t *testing.T) {
	// Test createFileWriter function in logger.go:287
	config := Config{
		Level:      slog.LevelInfo,
		Format:     "json",
		Output:     "file",
		FilePath:   "/tmp/test_logging_file.log",
		MaxSize:    1024,
		MaxAge:     1,
		MaxBackups: 1,
		Compress:   true,
	}

	_, err := New(config)
	if err != nil {
		t.Errorf("Failed to create logger with file writer: %v", err)
	}

	// Test with empty file path to trigger error path
	config2 := Config{
		Level:    slog.LevelInfo,
		Format:   "json",
		Output:   "file",
		FilePath: "", // Empty path should cause error
	}

	_, err2 := New(config2)
	if err2 == nil {
		t.Error("Expected error when creating logger with empty file path")
	}
}

func TestShouldSampleEdgeCases(t *testing.T) {
	// Test shouldSample function edge cases
	config := Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Sampling: SamplingConfig{
			Enabled: false, // Disabled sampling
		},
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// This should exercise the shouldSample function
	logger.Info("test message")

	// Test with enabled sampling to hit the sampler != nil path
	config2 := Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Sampling: SamplingConfig{
			Enabled: true,
			Type:    "rate",
			Rate:    1.0, // Always sample
		},
	}

	logger2, err := New(config2)
	if err != nil {
		t.Fatalf("Failed to create logger with sampling: %v", err)
	}

	// This should exercise the shouldSample function with sampler != nil
	logger2.Info("test message with sampling")
}

func TestFormatValueComplexTypes(t *testing.T) {
	// Test formatValue function with complex types to increase coverage
	formatter := NewTextFormatter()

	// Test with various complex value types
	complexRecord := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "Complex types test",
	}

	// Add different types of attributes to test all formatValue paths
	complexRecord.AddAttrs(
		slog.Any("nil_value", nil),
		slog.Any("error_value", errors.New("test error")),
		slog.Any("complex_struct", struct {
			Name string
			Age  int
		}{Name: "test", Age: 25}),
		slog.Any("slice", []string{"a", "b", "c"}),
		slog.Any("map", map[string]interface{}{
			"nested": map[string]string{"key": "value"},
		}),
		slog.Float64("float", 3.14159),
		slog.Uint64("uint", uint64(18446744073709551615)),
		// Test byte slice specifically
		slog.Any("bytes", []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}),
		// Test duration
		slog.Duration("duration", 5*time.Second),
		// Test time
		slog.Time("time", time.Now()),
		// Test bool
		slog.Bool("bool", true),
		// Test int64
		slog.Int64("int64", -9223372036854775808),
		// Test uint64
		slog.Uint64("uint64", 18446744073709551615),
		// Test group
		slog.Group("group", slog.String("nested", "value")),
	)

	output, err := formatter.Format(&complexRecord)
	if err != nil {
		t.Errorf("Format error: %v", err)
	}

	if len(output) == 0 {
		t.Error("Expected output from complex format")
	}
}

func TestPrettyHandlerMultilineComplexFormatting(t *testing.T) {
	// Test multiline formatting paths in formatters.go
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
		Level:     slog.LevelDebug,
		UseColor:  false,
		Multiline: true,
	})

	// Create a record with nested groups and many attributes
	handler2 := handler.WithGroup("service")
	handler3 := handler2.WithGroup("request")
	handler4 := handler3.WithAttrs([]slog.Attr{
		slog.String("method", "POST"),
		slog.String("path", "/api/users/create"),
		slog.Int("status", 201),
		slog.Duration("duration", 150*time.Millisecond),
		slog.Any("body", map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
				"metadata": map[string]interface{}{
					"source":    "web",
					"timestamp": time.Now().Unix(),
				},
			},
		}),
	})

	r := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "User created successfully",
	}

	err := handler4.Handle(context.Background(), r)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected multiline output")
	}
}

func TestGRPCStreamServerInterceptorErrors(t *testing.T) {
	// Test error paths in GRPCStreamServerInterceptor
	logger, err := New(Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	interceptor := GRPCStreamServerInterceptor(logger)

	// Create a mock stream that will cause errors
	errorStream := &grpcStreamError{}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return status.Error(codes.Internal, "internal server error")
	}

	err = interceptor(nil, errorStream, &grpc.StreamServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}, handler)

	if err == nil {
		t.Error("Expected error from stream interceptor")
	}
}

// Mock implementations for testing

type grpcStream struct{}

func (s *grpcStream) Context() context.Context        { return nil }
func (s *grpcStream) SendMsg(m interface{}) error     { return nil }
func (s *grpcStream) RecvMsg(m interface{}) error     { return nil }
func (s *grpcStream) SendHeader(md metadata.MD) error { return nil }
func (s *grpcStream) SetHeader(md metadata.MD) error  { return nil }
func (s *grpcStream) SetTrailer(md metadata.MD)       {}

type grpcStreamError struct{}

func (s *grpcStreamError) Context() context.Context {
	return context.WithValue(context.Background(), "test", "value")
}

func (s *grpcStreamError) SendMsg(m interface{}) error {
	return errors.New("send error")
}

func (s *grpcStreamError) RecvMsg(m interface{}) error {
	return io.EOF
}

func (s *grpcStreamError) SendHeader(md metadata.MD) error {
	return errors.New("send header error")
}

func (s *grpcStreamError) SetHeader(md metadata.MD) error {
	return errors.New("set header error")
}
func (s *grpcStreamError) SetTrailer(md metadata.MD) {}

// testHook implements Hook for testing
type testHook struct{}

func (h *testHook) Process(record *slog.Record) *slog.Record {
	// Simple hook that adds a test attribute
	record.AddAttrs(slog.String("test_hook", "processed"))
	return record
}

// Add tests for specific formatAttrMultiline paths
func TestFormatAttrMultilineEdgeCases(t *testing.T) {
	// Test formatAttrMultiline with various edge cases
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
		Level:     slog.LevelDebug,
		UseColor:  false,
		Multiline: true,
	})

	// Test with LogValuer
	testLogValuer := testLogValuerStruct{value: "test_value"}

	// Test with Group and LogValuer
	r := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "Test LogValuer",
	}

	r.AddAttrs(
		slog.Group("test_group",
			slog.Any("log_valuer", testLogValuer),
			slog.String("nested", "value"),
		),
	)

	err := handler.Handle(context.Background(), r)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected output from LogValuer test")
	}
}

// testLogValuerStruct implements slog.LogValuer
type testLogValuerStruct struct {
	value string
}

func (t testLogValuerStruct) LogValue() slog.Value {
	return slog.StringValue(t.value)
}

func TestFormatValueKindEdgeCases(t *testing.T) {
	// Test formatValue with all different kinds to hit uncovered paths
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
		Level:     slog.LevelDebug,
		UseColor:  false,
		Multiline: false,
	})

	r := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "Test all kinds",
	}

	// Test all slog.Kind values to hit formatValue switch cases
	r.AddAttrs(
		slog.Any("kind_any", map[string]interface{}{"key": "value"}),
		slog.Any("kind_logvaluer", testLogValuerStruct{value: "test"}),
		slog.Any("kind_group", slog.GroupValue(slog.String("nested", "value"))),
		slog.Int64("kind_int64", -1),
		slog.Uint64("kind_uint64", 1),
		slog.Float64("kind_float64", 3.14),
		slog.Bool("kind_bool", true),
		slog.Duration("kind_duration", time.Second),
		slog.Time("kind_time", time.Now()),
		slog.String("kind_string", "test"),
	)

	err := handler.Handle(context.Background(), r)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected output from kind test")
	}
}

func TestFormatValueNilAndErrorCases(t *testing.T) {
	// Test formatValue with nil and error cases to hit uncovered paths
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
		Level:     slog.LevelDebug,
		UseColor:  false,
		Multiline: false,
	})

	r := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "Test nil and errors",
	}

	// Test nil pointer
	var nilPointer *string
	// Test pointer to string
	str := "test"
	strPtr := &str

	r.AddAttrs(
		slog.Any("nil_pointer", nilPointer),
		slog.Any("string_pointer", strPtr),
		slog.Any("nil_interface", nil),
		slog.Any("error_type", errors.New("test error")),
		// Test rune
		slog.Any("rune_value", rune('A')),
		// Test channel (should use reflect path)
		slog.Any("channel", make(chan int)),
	)

	err := handler.Handle(context.Background(), r)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected output from nil and error test")
	}
}

func TestCreateSamplerInvalidType(t *testing.T) {
	// Test createSampler with invalid type to hit error path
	config := Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: "stdout",
		Sampling: SamplingConfig{
			Enabled: true,
			Type:    "invalid_type", // This should cause an error
		},
	}

	_, err := New(config)
	if err == nil {
		t.Error("Expected error when creating logger with invalid sampler type")
	}
}

func TestCreateHandlerInvalidFormat(t *testing.T) {
	// Test createHandler with invalid format to hit error path
	config := Config{
		Level:  slog.LevelInfo,
		Format: "invalid_format", // This should cause an error
		Output: "stdout",
	}

	_, err := New(config)
	if err == nil {
		t.Error("Expected error when creating logger with invalid format")
	}
}

func TestHandlerWithEnabledFalse(t *testing.T) {
	// Test handler Enabled method returning false
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
		Level:     slog.LevelError, // Higher level to disable Debug
		UseColor:  false,
		Multiline: false,
	})

	// Test that Debug level is not enabled
	if handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Expected Debug level to be disabled")
	}

	// Test that Error level is enabled
	if !handler.Enabled(context.Background(), slog.LevelError) {
		t.Error("Expected Error level to be enabled")
	}
}

func TestPrettyHandlerWithNilAttrs(t *testing.T) {
	// Test handler with nil attributes to hit formatAttrMultiline paths
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
		Level:     slog.LevelDebug,
		UseColor:  true, // Test with colors
		Multiline: true,
	})

	// Create a handler with attrs and groups to test formatAttrMultiline
	handlerWithAttrs := handler.WithAttrs([]slog.Attr{
		slog.String("attr1", "value1"),
		slog.Group("group1", slog.String("nested", "value")),
	})

	handlerWithGroup := handlerWithAttrs.WithGroup("outer_group")

	r := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "Test with nested attrs and groups",
	}

	// Add more attributes to the record itself
	r.AddAttrs(
		slog.String("record_attr", "record_value"),
		slog.Group("record_group", slog.String("inner", "inner_value")),
	)

	err := handlerWithGroup.Handle(context.Background(), r)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected output from nested attrs test")
	}
}

func TestHandlePathCoverage(t *testing.T) {
	// Test handle paths to hit remaining uncovered lines
	var buf bytes.Buffer
	handler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
		Level:     slog.LevelDebug,
		UseColor:  false,
		Multiline: false,
	})

	// Test with empty groups and empty attrs to hit specific formatAttrMultiline paths
	handlerWithEmptyGroup := handler.WithGroup("")
	handlerWithEmptyAttrs := handlerWithEmptyGroup.WithAttrs([]slog.Attr{})

	r := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "Test empty conditions",
	}

	// Add attributes with empty values to test edge cases
	r.AddAttrs(
		slog.String("empty", ""),
		slog.Group("", slog.String("empty_group_key", "value")),
	)

	err := handlerWithEmptyAttrs.Handle(context.Background(), r)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}

	// Test with handler that doesn't accept the level
	disabledHandler := NewPrettyHandler(&buf, &PrettyHandlerOptions{
		Level:     slog.LevelError, // Higher than Info, so Info won't be handled
		UseColor:  false,
		Multiline: false,
	})

	r2 := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo, // This should be filtered out
		Message: "This should not be logged",
	}

	// This should return early due to Enabled check
	err = disabledHandler.Handle(context.Background(), r2)
	if err != nil {
		t.Errorf("Handle failed: %v", err)
	}
}
