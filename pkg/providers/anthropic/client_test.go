// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guild-framework/guild-core/pkg/providers/interfaces"
)

func TestAnthropicProvider(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != "test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"type":    "authentication_error",
					"message": "Invalid API key",
				},
			})
			return
		}

		if r.Header.Get("anthropic-version") != "2023-06-01" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Parse request
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		// Mock response
		response := map[string]interface{}{
			"id":    "msg_test123",
			"type":  "message",
			"role":  "assistant",
			"model": req["model"],
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "Mock Anthropic response",
				},
			},
			"stop_reason":   "stop",
			"stop_sequence": nil,
			"usage": map[string]int{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server
	client := &Client{
		apiKey:  "test-api-key",
		baseURL: server.URL,
		client:  &http.Client{},
		capabilities: interfaces.ProviderCapabilities{
			MaxTokens:      200000,
			ContextWindow:  200000,
			SupportsVision: true,
			SupportsTools:  true,
			SupportsStream: true,
			Models: []interfaces.ModelInfo{
				{ID: Claude4Sonnet, Name: "Claude 4 Sonnet"},
			},
		},
	}

	t.Run("ChatCompletion", func(t *testing.T) {
		ctx := context.Background()
		req := interfaces.ChatRequest{
			Model: Claude4Sonnet,
			Messages: []interfaces.ChatMessage{
				{Role: "user", Content: "Hello"},
			},
		}

		resp, err := client.ChatCompletion(ctx, req)
		if err != nil {
			t.Fatalf("ChatCompletion failed: %v", err)
		}

		if len(resp.Choices) == 0 {
			t.Fatal("No choices in response")
		}

		if resp.Choices[0].Message.Content != "Mock Anthropic response" {
			t.Errorf("Unexpected response: %s", resp.Choices[0].Message.Content)
		}
	})

	t.Run("SystemMessage", func(t *testing.T) {
		ctx := context.Background()
		req := interfaces.ChatRequest{
			Model: Claude4Sonnet,
			Messages: []interfaces.ChatMessage{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello"},
			},
		}

		// Should handle system message separately
		_, err := client.ChatCompletion(ctx, req)
		if err != nil {
			t.Fatalf("ChatCompletion with system message failed: %v", err)
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with wrong API key
		badClient := &Client{
			apiKey:  "wrong-key",
			baseURL: server.URL,
			client:  &http.Client{},
		}

		ctx := context.Background()
		req := interfaces.ChatRequest{
			Model: Claude4Sonnet,
			Messages: []interfaces.ChatMessage{
				{Role: "user", Content: "Hello"},
			},
		}

		_, err := badClient.ChatCompletion(ctx, req)
		if err == nil {
			t.Error("Expected error with wrong API key")
		}

		// Check error type
		if provErr, ok := err.(*interfaces.ProviderError); ok {
			if provErr.Type != interfaces.ErrorTypeAuth {
				t.Errorf("Expected auth error, got %s", provErr.Type)
			}
			if provErr.Retryable {
				t.Error("Auth error should not be retryable")
			}
		}
	})

	t.Run("Capabilities", func(t *testing.T) {
		caps := client.GetCapabilities()

		if caps.MaxTokens != 200000 {
			t.Errorf("Wrong max tokens: %d", caps.MaxTokens)
		}

		if !caps.SupportsVision {
			t.Error("Should support vision")
		}

		if len(caps.Models) == 0 {
			t.Error("No models in capabilities")
		}
	})

	t.Run("ModelRecommendation", func(t *testing.T) {
		testCases := []struct {
			useCase  string
			expected string
		}{
			{"coding", Claude4Opus},
			{"reasoning", Claude4Opus},
			{"cost-efficient", Claude35Haiku},
			{"general", Claude4Sonnet},
		}

		for _, tc := range testCases {
			t.Run(tc.useCase, func(t *testing.T) {
				model := GetRecommendedModel(tc.useCase)
				if model != tc.expected {
					t.Errorf("Expected %s for %s, got %s", tc.expected, tc.useCase, model)
				}
			})
		}
	})

	t.Run("NoEmbeddings", func(t *testing.T) {
		ctx := context.Background()
		req := interfaces.EmbeddingRequest{
			Model: "claude-3",
			Input: []string{"test"},
		}

		_, err := client.CreateEmbedding(ctx, req)
		if err == nil {
			t.Error("Expected error for embeddings (not supported)")
		}
	})
}
