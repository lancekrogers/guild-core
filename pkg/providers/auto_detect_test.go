// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

func TestNewAutoDetector(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "with custom timeout",
			timeout:         5 * time.Second,
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "with zero timeout uses default",
			timeout:         0,
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewAutoDetector(tt.timeout)

			if detector.timeout != tt.expectedTimeout {
				t.Errorf("NewAutoDetector() timeout = %v, want %v", detector.timeout, tt.expectedTimeout)
			}

			if detector.httpClient == nil {
				t.Error("NewAutoDetector() httpClient is nil")
			}

			if detector.capabilities == nil {
				t.Error("NewAutoDetector() capabilities is nil")
			}
		})
	}
}

func TestAutoDetector_DetectAll(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		expectError bool
		errorCode   gerror.ErrorCode
	}{
		{
			name:        "valid context",
			ctx:         context.Background(),
			expectError: false,
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			expectError: true,
			errorCode:   gerror.ErrCodeCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewAutoDetector(1 * time.Second)

			results, err := detector.DetectAll(tt.ctx)

			if tt.expectError {
				if err == nil {
					t.Error("DetectAll() expected error but got none")
				}
				if gErr, ok := err.(*gerror.GuildError); ok {
					if gErr.Code != tt.errorCode {
						t.Errorf("DetectAll() error code = %v, want %v", gErr.Code, tt.errorCode)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("DetectAll() unexpected error = %v", err)
				return
			}

			// Should return results for Claude Code and Ollama
			if len(results) != 2 {
				t.Errorf("DetectAll() returned %d results, want 2", len(results))
			}

			// Verify providers are included
			providers := make(map[ProviderType]bool)
			for _, result := range results {
				providers[result.Provider] = true
			}

			if !providers[ProviderClaudeCode] {
				t.Error("DetectAll() missing Claude Code provider")
			}

			if !providers[ProviderOllama] {
				t.Error("DetectAll() missing Ollama provider")
			}
		})
	}
}

func TestAutoDetector_DetectClaudeCode(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		expectError bool
		errorCode   gerror.ErrorCode
	}{
		{
			name:        "valid context",
			ctx:         context.Background(),
			expectError: false,
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			expectError: true,
			errorCode:   gerror.ErrCodeCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewAutoDetector(1 * time.Second)

			result, err := detector.DetectClaudeCode(tt.ctx)

			if tt.expectError {
				if err == nil {
					t.Error("DetectClaudeCode() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("DetectClaudeCode() unexpected error = %v", err)
				return
			}

			if result.Provider != ProviderClaudeCode {
				t.Errorf("DetectClaudeCode() provider = %v, want %v", result.Provider, ProviderClaudeCode)
			}

			if result.DetectedAt.IsZero() {
				t.Error("DetectClaudeCode() DetectedAt is zero")
			}
		})
	}
}

func TestAutoDetector_DetectOllama(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		expectError bool
		errorCode   gerror.ErrorCode
	}{
		{
			name:        "valid context",
			ctx:         context.Background(),
			expectError: false,
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			expectError: true,
			errorCode:   gerror.ErrCodeCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewAutoDetector(1 * time.Second)

			result, err := detector.DetectOllama(tt.ctx)

			if tt.expectError {
				if err == nil {
					t.Error("DetectOllama() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("DetectOllama() unexpected error = %v", err)
				return
			}

			if result.Provider != ProviderOllama {
				t.Errorf("DetectOllama() provider = %v, want %v", result.Provider, ProviderOllama)
			}

			if result.DetectedAt.IsZero() {
				t.Error("DetectOllama() DetectedAt is zero")
			}
		})
	}
}

func TestAutoDetector_testOllamaEndpoint(t *testing.T) {
	tests := []struct {
		name            string
		serverResponse  string
		serverStatus    int
		expectAvailable bool
		expectVersion   string
		expectError     bool
	}{
		{
			name:            "successful detection with version",
			serverResponse:  `{"version": "0.1.25"}`,
			serverStatus:    http.StatusOK,
			expectAvailable: true,
			expectVersion:   "0.1.25",
			expectError:     false,
		},
		{
			name:            "successful detection without version",
			serverResponse:  `{}`,
			serverStatus:    http.StatusOK,
			expectAvailable: true,
			expectVersion:   "unknown",
			expectError:     false,
		},
		{
			name:            "server error",
			serverResponse:  `{"error": "not found"}`,
			serverStatus:    http.StatusNotFound,
			expectAvailable: false,
			expectVersion:   "",
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/version" {
					t.Errorf("Expected path /api/version, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			detector := NewAutoDetector(1 * time.Second)
			ctx := context.Background()

			available, version, err := detector.testOllamaEndpoint(ctx, server.URL)

			if tt.expectError {
				if err == nil {
					t.Error("testOllamaEndpoint() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("testOllamaEndpoint() unexpected error = %v", err)
				return
			}

			if available != tt.expectAvailable {
				t.Errorf("testOllamaEndpoint() available = %v, want %v", available, tt.expectAvailable)
			}

			if version != tt.expectVersion {
				t.Errorf("testOllamaEndpoint() version = %v, want %v", version, tt.expectVersion)
			}
		})
	}
}

func TestAutoDetector_isValidClaudeCodeOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name:     "valid claude output",
			output:   "Claude 1.2.3",
			expected: true,
		},
		{
			name:     "valid anthropic output",
			output:   "Anthropic Claude CLI v1.0.0",
			expected: true,
		},
		{
			name:     "valid version output",
			output:   "version 1.0.0",
			expected: true,
		},
		{
			name:     "invalid output",
			output:   "some random text",
			expected: false,
		},
		{
			name:     "empty output",
			output:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewAutoDetector(1 * time.Second)

			result := detector.isValidClaudeCodeOutput(tt.output)

			if result != tt.expected {
				t.Errorf("isValidClaudeCodeOutput() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAutoDetector_calculateClaudeCodeConfidence(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		version         string
		expectedMinimum float64
		expectedMaximum float64
	}{
		{
			name:            "standard path with claude version",
			path:            "claude",
			version:         "Claude 1.2.3",
			expectedMinimum: 0.9,
			expectedMaximum: 1.0,
		},
		{
			name:            "standard path without claude version",
			path:            "claude",
			version:         "version 1.0.0",
			expectedMinimum: 0.7,
			expectedMaximum: 0.9,
		},
		{
			name:            "custom path with claude version",
			path:            "/usr/local/bin/claude",
			version:         "Claude 1.2.3",
			expectedMinimum: 0.6,
			expectedMaximum: 0.8,
		},
		{
			name:            "custom path without claude version",
			path:            "/usr/local/bin/claude",
			version:         "version 1.0.0",
			expectedMinimum: 0.4,
			expectedMaximum: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewAutoDetector(1 * time.Second)

			confidence := detector.calculateClaudeCodeConfidence(tt.path, tt.version)

			if confidence < tt.expectedMinimum || confidence > tt.expectedMaximum {
				t.Errorf("calculateClaudeCodeConfidence() = %v, want between %v and %v",
					confidence, tt.expectedMinimum, tt.expectedMaximum)
			}
		})
	}
}

func TestAutoDetector_calculateOllamaConfidence(t *testing.T) {
	tests := []struct {
		name            string
		endpoint        string
		version         string
		expectedMinimum float64
		expectedMaximum float64
	}{
		{
			name:            "standard port with version",
			endpoint:        "http://localhost:11434",
			version:         "0.1.25",
			expectedMinimum: 0.9,
			expectedMaximum: 1.0,
		},
		{
			name:            "standard port without version",
			endpoint:        "http://localhost:11434",
			version:         "unknown",
			expectedMinimum: 0.8,
			expectedMaximum: 1.0,
		},
		{
			name:            "custom port with version",
			endpoint:        "http://localhost:8080",
			version:         "0.1.25",
			expectedMinimum: 0.6,
			expectedMaximum: 0.8,
		},
		{
			name:            "custom port without version",
			endpoint:        "http://localhost:8080",
			version:         "unknown",
			expectedMinimum: 0.5,
			expectedMaximum: 0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewAutoDetector(1 * time.Second)

			confidence := detector.calculateOllamaConfidence(tt.endpoint, tt.version)

			if confidence < tt.expectedMinimum || confidence > tt.expectedMaximum {
				t.Errorf("calculateOllamaConfidence() = %v, want between %v and %v",
					confidence, tt.expectedMinimum, tt.expectedMaximum)
			}
		})
	}
}

func TestAutoDetector_GetBestProvider(t *testing.T) {
	tests := []struct {
		name               string
		ctx                context.Context
		preferredProviders []ProviderType
		expectError        bool
		errorCode          gerror.ErrorCode
	}{
		{
			name:               "valid context with preferences",
			ctx:                context.Background(),
			preferredProviders: []ProviderType{ProviderClaudeCode, ProviderOllama},
			expectError:        false,
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			preferredProviders: []ProviderType{ProviderClaudeCode},
			expectError:        true,
			errorCode:          gerror.ErrCodeCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewAutoDetector(1 * time.Second)

			result, err := detector.GetBestProvider(tt.ctx, tt.preferredProviders)

			if tt.expectError {
				if err == nil {
					t.Error("GetBestProvider() expected error but got none")
				}
				if gErr, ok := err.(*gerror.GuildError); ok {
					if gErr.Code != tt.errorCode {
						t.Errorf("GetBestProvider() error code = %v, want %v", gErr.Code, tt.errorCode)
					}
				}
				return
			}

			// Since we can't control what's actually installed, we can only test error cases
			// The success case depends on the actual system configuration
			if err != nil {
				// If error, it should be about no providers available
				if gErr, ok := err.(*gerror.GuildError); ok {
					if gErr.Code != gerror.ErrCodeProvider {
						t.Errorf("GetBestProvider() error code = %v, want %v", gErr.Code, gerror.ErrCodeProvider)
					}
				}
			} else {
				// If successful, should return a valid result
				if result == nil {
					t.Error("GetBestProvider() returned nil result without error")
				} else if !result.Available {
					t.Error("GetBestProvider() returned unavailable provider")
				}
			}
		})
	}
}

func TestAutoDetector_ValidateProvider(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		providerType ProviderType
		expectError  bool
		errorCode    gerror.ErrorCode
	}{
		{
			name:         "valid context with Claude Code",
			ctx:          context.Background(),
			providerType: ProviderClaudeCode,
			expectError:  false, // Might succeed or fail based on system
		},
		{
			name:         "valid context with Ollama",
			ctx:          context.Background(),
			providerType: ProviderOllama,
			expectError:  false, // Might succeed or fail based on system
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			providerType: ProviderClaudeCode,
			expectError:  true,
			errorCode:    gerror.ErrCodeCancelled,
		},
		{
			name:         "unsupported provider",
			ctx:          context.Background(),
			providerType: ProviderOpenAI,
			expectError:  true,
			errorCode:    gerror.ErrCodeProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewAutoDetector(1 * time.Second)

			err := detector.ValidateProvider(tt.ctx, tt.providerType)

			if tt.expectError {
				if err == nil {
					t.Error("ValidateProvider() expected error but got none")
				}
				if gErr, ok := err.(*gerror.GuildError); ok {
					if gErr.Code != tt.errorCode {
						t.Errorf("ValidateProvider() error code = %v, want %v", gErr.Code, tt.errorCode)
					}
				}
				return
			}

			// For supported providers, error depends on actual system state
			// We can't guarantee success without actual installation
		})
	}
}

func TestAutoDetector_CreateClientFromDetection(t *testing.T) {
	tests := []struct {
		name        string
		result      DetectionResult
		expectError bool
		errorCode   gerror.ErrorCode
	}{
		{
			name: "unavailable provider",
			result: DetectionResult{
				Provider:  ProviderClaudeCode,
				Available: false,
			},
			expectError: true,
			errorCode:   gerror.ErrCodeProvider,
		},
		{
			name: "Claude Code provider",
			result: DetectionResult{
				Provider:  ProviderClaudeCode,
				Available: true,
			},
			expectError: true,
			errorCode:   gerror.ErrCodeNotImplemented,
		},
		{
			name: "Ollama provider",
			result: DetectionResult{
				Provider:  ProviderOllama,
				Available: true,
				Endpoint:  "http://localhost:11434",
			},
			expectError: true,
			errorCode:   gerror.ErrCodeNotImplemented,
		},
		{
			name: "unsupported provider",
			result: DetectionResult{
				Provider:  ProviderOpenAI,
				Available: true,
			},
			expectError: true,
			errorCode:   gerror.ErrCodeProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewAutoDetector(1 * time.Second)

			client, err := detector.CreateClientFromDetection(tt.result)

			if tt.expectError {
				if err == nil {
					t.Error("CreateClientFromDetection() expected error but got none")
				}
				if gErr, ok := err.(*gerror.GuildError); ok {
					if gErr.Code != tt.errorCode {
						t.Errorf("CreateClientFromDetection() error code = %v, want %v", gErr.Code, tt.errorCode)
					}
				}
				if client != nil {
					t.Error("CreateClientFromDetection() returned non-nil client with error")
				}
				return
			}

			if err != nil {
				t.Errorf("CreateClientFromDetection() unexpected error = %v", err)
			}

			if client == nil {
				t.Error("CreateClientFromDetection() returned nil client without error")
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkAutoDetector_DetectAll(b *testing.B) {
	detector := NewAutoDetector(1 * time.Second)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectAll(ctx)
	}
}

func BenchmarkAutoDetector_DetectClaudeCode(b *testing.B) {
	detector := NewAutoDetector(1 * time.Second)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectClaudeCode(ctx)
	}
}

func BenchmarkAutoDetector_DetectOllama(b *testing.B) {
	detector := NewAutoDetector(1 * time.Second)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectOllama(ctx)
	}
}
