// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package ora

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guild-framework/guild-core/pkg/providers/interfaces"
)

func TestOraProvider(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"message": "Invalid API key",
					"type":    "authentication_error",
				},
			})
			return
		}

		// Parse request
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		// Mock response (OpenAI-compatible format)
		response := map[string]interface{}{
			"id":      "ora-test123",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   req["model"],
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Mock Ora response",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		apiKey:  "test-api-key",
		baseURL: server.URL,
		client:  &http.Client{},
		capabilities: interfaces.ProviderCapabilities{
			MaxTokens:      64000,
			ContextWindow:  64000,
			SupportsVision: false,
			SupportsTools:  true,
			SupportsStream: true,
			Models: []interfaces.ModelInfo{
				{ID: DeepSeekV3, Name: "DeepSeek V3"},
				{ID: DeepSeekR1, Name: "DeepSeek R1"},
			},
		},
	}

	t.Run("ChatCompletion", func(t *testing.T) {
		ctx := context.Background()
		req := interfaces.ChatRequest{
			Model: DeepSeekV3,
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

		if resp.Choices[0].Message.Content != "Mock Ora response" {
			t.Errorf("Unexpected response: %s", resp.Choices[0].Message.Content)
		}
	})

	t.Run("AuthError", func(t *testing.T) {
		badClient := &Client{
			apiKey:  "wrong-key",
			baseURL: server.URL,
			client:  &http.Client{},
		}

		ctx := context.Background()
		req := interfaces.ChatRequest{
			Model: DeepSeekV3,
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
			if provErr.StatusCode != 401 {
				t.Errorf("Expected 401 status, got %d", provErr.StatusCode)
			}
			if provErr.Retryable {
				t.Error("Auth error should not be retryable")
			}
		}
	})

	t.Run("Capabilities", func(t *testing.T) {
		caps := client.GetCapabilities()

		if caps.MaxTokens != 64000 {
			t.Errorf("Wrong max tokens: %d", caps.MaxTokens)
		}

		if caps.SupportsVision {
			t.Error("Ora should not support vision")
		}

		if !caps.SupportsTools {
			t.Error("Ora should support tools")
		}

		// Check models
		if len(caps.Models) != 2 {
			t.Errorf("Expected 2 models, got %d", len(caps.Models))
		}
	})

	t.Run("ModelRecommendation", func(t *testing.T) {
		testCases := []struct {
			useCase  string
			expected string
		}{
			{"reasoning", DeepSeekR1},
			{"general", DeepSeekV3},
			{"cost-efficient", DeepSeekV3},
			{"default", DeepSeekV3},
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
			Model: DeepSeekV3,
			Input: []string{"test"},
		}

		_, err := client.CreateEmbedding(ctx, req)
		if err == nil {
			t.Error("Expected error for embeddings (not supported)")
		}
	})

	t.Run("StreamingNotImplemented", func(t *testing.T) {
		ctx := context.Background()
		req := interfaces.ChatRequest{
			Model: DeepSeekV3,
			Messages: []interfaces.ChatMessage{
				{Role: "user", Content: "Hello"},
			},
		}

		_, err := client.StreamChatCompletion(ctx, req)
		if err == nil {
			t.Error("Expected error for streaming (not implemented)")
		}
	})
}

func TestOraDeepSeekModels(t *testing.T) {
	// Ora specializes in DeepSeek models
	client := NewClient("test-key")
	caps := client.GetCapabilities()

	deepSeekModels := 0
	for _, model := range caps.Models {
		if model.ID == DeepSeekV3 || model.ID == DeepSeekR1 {
			deepSeekModels++
		}
	}

	if deepSeekModels < 2 {
		t.Errorf("Expected at least 2 DeepSeek models, got %d", deepSeekModels)
	}
}
