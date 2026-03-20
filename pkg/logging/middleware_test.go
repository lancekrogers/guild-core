package logging

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHTTPMiddleware(t *testing.T) {
	// Create a test logger with shared log store
	var logs []mockLogEntry
	logger := &mockLogger{logs: &logs}

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)

		if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	})

	// Wrap with middleware
	wrapped := HTTPMiddleware(logger)(handler)

	t.Run("SuccessfulRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Check if logger was called
		if len(logs) < 2 {
			t.Errorf("Expected at least 2 log entries, got %d", len(logs))
		}

		// The first log should be "request started"
		// The last log should be "request completed"
		lastLog := logs[len(logs)-1]
		if lastLog.level != "INFO" {
			t.Errorf("Expected INFO level, got %s", lastLog.level)
		}

		// Check for expected fields in the completion log
		foundStatus := false
		foundDuration := false
		foundBytes := false

		for _, field := range lastLog.fields {
			switch field.Key {
			case "status":
				foundStatus = field.Value.Int64() == 200
			case "duration":
				foundDuration = field.Value.Duration() > 0
			case "bytes":
				foundBytes = field.Value.Int64() == 2 // "OK"
			}
		}

		// Check if request_id was propagated through context
		foundRequestID := false
		for _, log := range logs {
			for _, field := range log.fields {
				if field.Key == "request_id" && field.Value.String() != "" {
					foundRequestID = true
					break
				}
			}
			if foundRequestID {
				break
			}
		}

		if !foundStatus {
			t.Error("Missing or incorrect status field")
		}
		if !foundDuration {
			t.Error("Missing or incorrect duration field")
		}
		if !foundBytes {
			t.Error("Missing or incorrect bytes field")
		}
		if !foundRequestID {
			t.Error("Missing request_id field")
		}
	})

	t.Run("ErrorRequest", func(t *testing.T) {
		logs = nil // Reset logs

		req := httptest.NewRequest("POST", "/error", strings.NewReader("test body"))
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rec.Code)
		}

		// Check if logger was called with warn level
		if len(logs) == 0 {
			t.Error("Expected log entry")
		}

		lastLog := logs[len(logs)-1]
		if lastLog.level != "ERROR" {
			t.Errorf("Expected ERROR level for 500 status, got %s", lastLog.level)
		}
	})

	t.Run("RequestIDPropagation", func(t *testing.T) {
		logs = nil // Reset logs

		// Create request with existing request ID
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "existing-id-123")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		// Check response header
		if rec.Header().Get("X-Request-ID") != "existing-id-123" {
			t.Error("Request ID not propagated to response")
		}

		// Check log
		lastLog := logs[len(logs)-1]
		foundRequestID := false
		for _, field := range lastLog.fields {
			if field.Key == "request_id" && field.Value.String() == "existing-id-123" {
				foundRequestID = true
				break
			}
		}

		if !foundRequestID {
			t.Error("Request ID not found in log")
		}
	})
}

func TestResponseWriter(t *testing.T) {
	t.Run("CaptureStatus", func(t *testing.T) {
		base := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: base}

		// Write header
		rw.WriteHeader(http.StatusCreated)

		if rw.statusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", rw.statusCode)
		}

		// Write without header (should default to 200)
		base2 := httptest.NewRecorder()
		rw2 := &responseWriter{ResponseWriter: base2}
		rw2.Write([]byte("test"))

		if rw2.statusCode != http.StatusOK {
			t.Errorf("Expected default status 200, got %d", rw2.statusCode)
		}
	})

	t.Run("CaptureBytes", func(t *testing.T) {
		base := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: base}

		// Write some data
		n, err := rw.Write([]byte("hello"))
		if err != nil {
			t.Errorf("Write error: %v", err)
		}
		if n != 5 {
			t.Errorf("Expected 5 bytes written, got %d", n)
		}

		n, err = rw.Write([]byte(" world"))
		if err != nil {
			t.Errorf("Write error: %v", err)
		}

		if rw.bytesWritten != 11 {
			t.Errorf("Expected 11 total bytes, got %d", rw.bytesWritten)
		}
	})
}

func TestGRPCUnaryServerInterceptor(t *testing.T) {
	var logs []mockLogEntry
	logger := &mockLogger{logs: &logs}
	interceptor := GRPCUnaryServerInterceptor(logger)

	// Mock handler
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Simulate processing
		time.Sleep(10 * time.Millisecond)

		if req.(string) == "error" {
			return nil, status.Error(codes.InvalidArgument, "invalid request")
		}
		return "response", nil
	}

	t.Run("SuccessfulCall", func(t *testing.T) {
		logs = nil

		ctx := context.Background()
		info := &grpc.UnaryServerInfo{
			FullMethod: "/service.Test/Method",
		}

		resp, err := interceptor(ctx, "request", info, handler)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if resp != "response" {
			t.Errorf("Expected 'response', got %v", resp)
		}

		// Check log
		if len(logs) == 0 {
			t.Error("Expected log entry")
		}

		lastLog := logs[len(logs)-1]
		if lastLog.level != "INFO" {
			t.Errorf("Expected INFO level, got %s", lastLog.level)
		}

		// Check fields
		foundMethod := false
		foundDuration := false
		foundCode := false

		for _, field := range lastLog.fields {
			switch field.Key {
			case "grpc.method":
				foundMethod = field.Value.String() == "/service.Test/Method"
			case "duration":
				foundDuration = field.Value.Duration() > 0
			case "grpc.code":
				foundCode = field.Value.String() == "OK"
			}
		}

		if !foundMethod {
			t.Error("Missing or incorrect grpc.method field")
		}
		if !foundDuration {
			t.Error("Missing or incorrect duration field")
		}
		if !foundCode {
			t.Error("Missing or incorrect code field")
		}
	})

	t.Run("ErrorCall", func(t *testing.T) {
		logs = nil

		ctx := context.Background()
		info := &grpc.UnaryServerInfo{
			FullMethod: "/service.Test/Error",
		}

		resp, err := interceptor(ctx, "error", info, handler)

		if err == nil {
			t.Error("Expected error")
		}
		if resp != nil {
			t.Errorf("Expected nil response, got %v", resp)
		}

		// Check log
		if len(logs) == 0 {
			t.Error("Expected log entry")
		}

		lastLog := logs[len(logs)-1]
		if lastLog.level != "WARN" {
			t.Errorf("Expected WARN level for InvalidArgument, got %s", lastLog.level)
		}

		// Check for error field
		foundError := false
		foundCode := false

		for _, field := range lastLog.fields {
			if field.Key == "error" {
				foundError = true
			}
			if field.Key == "grpc.code" && field.Value.String() == "InvalidArgument" {
				foundCode = true
			}
		}

		if !foundError || !foundCode {
			t.Error("Missing error or grpc.code field in log")
		}
	})
}

func TestGRPCStreamServerInterceptor(t *testing.T) {
	var logs []mockLogEntry
	logger := &mockLogger{logs: &logs}
	interceptor := GRPCStreamServerInterceptor(logger)

	// Mock handler
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		// Simulate stream processing
		time.Sleep(10 * time.Millisecond)

		// Get wrapped stream
		wrapped := stream.(*loggingServerStream)

		// Simulate receiving and sending messages
		for i := 0; i < 3; i++ {
			msg := &mockMessage{}
			if err := wrapped.RecvMsg(msg); err != nil {
				return err
			}

			if err := wrapped.SendMsg(msg); err != nil {
				return err
			}
		}

		return nil
	}

	t.Run("SuccessfulStream", func(t *testing.T) {
		logs = nil

		info := &grpc.StreamServerInfo{
			FullMethod: "/service.Test/Stream",
		}

		stream := &mockServerStream{
			ctx: context.Background(),
		}

		err := interceptor(nil, stream, info, handler)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Check log
		if len(logs) == 0 {
			t.Error("Expected log entry")
		}

		lastLog := logs[len(logs)-1]
		if lastLog.level != "INFO" {
			t.Errorf("Expected INFO level, got %s", lastLog.level)
		}

		// Check fields
		foundMsgSent := false
		foundMsgRecv := false

		for _, field := range lastLog.fields {
			if field.Key == "messages_sent" && field.Value.Int64() == 3 {
				foundMsgSent = true
			}
			if field.Key == "messages_received" && field.Value.Int64() == 3 {
				foundMsgRecv = true
			}
		}

		if !foundMsgSent || !foundMsgRecv {
			t.Error("Missing message count fields")
		}
	})
}

func TestGenerateRequestID(t *testing.T) {
	// Generate multiple IDs
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := generateRequestID()

		// Check format (simple UUID-like format)
		if len(id) < 8 {
			t.Error("Request ID too short")
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("Duplicate request ID: %s", id)
		}
		ids[id] = true
	}
}

// Mock types for testing

type mockLogger struct {
	logs       *[]mockLogEntry // Shared log store
	baseFields []Field
}

type mockLogEntry struct {
	level   string
	message string
	fields  []Field
}

func (m *mockLogger) Debug(msg string, fields ...Field) {
	allFields := append(m.baseFields, fields...)
	*m.logs = append(*m.logs, mockLogEntry{level: "DEBUG", message: msg, fields: allFields})
}

func (m *mockLogger) Info(msg string, fields ...Field) {
	allFields := append(m.baseFields, fields...)
	*m.logs = append(*m.logs, mockLogEntry{level: "INFO", message: msg, fields: allFields})
}

func (m *mockLogger) Warn(msg string, fields ...Field) {
	allFields := append(m.baseFields, fields...)
	*m.logs = append(*m.logs, mockLogEntry{level: "WARN", message: msg, fields: allFields})
}

func (m *mockLogger) Error(msg string, fields ...Field) {
	allFields := append(m.baseFields, fields...)
	*m.logs = append(*m.logs, mockLogEntry{level: "ERROR", message: msg, fields: allFields})
}

func (m *mockLogger) With(fields ...Field) Logger {
	newLogger := &mockLogger{
		logs:       m.logs,
		baseFields: append(m.baseFields, fields...),
	}
	return newLogger
}

func (m *mockLogger) WithContext(ctx context.Context) Logger {
	// Extract request ID from context and add it as a field
	if requestID, ok := RequestID(ctx); ok {
		return m.With(RequestIDField(requestID))
	}
	return m
}

type mockServerStream struct {
	grpc.ServerStream
	ctx          context.Context
	sentMessages int
	recvMessages int
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) SendMsg(msg interface{}) error {
	m.sentMessages++
	return nil
}

func (m *mockServerStream) RecvMsg(msg interface{}) error {
	m.recvMessages++
	if m.recvMessages > 3 {
		return errors.New("EOF")
	}
	return nil
}

type mockMessage struct{}
