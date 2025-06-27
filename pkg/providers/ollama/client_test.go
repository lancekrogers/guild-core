// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lancekrogers/guild/pkg/providers/interfaces"
)

func TestOllamaProvider(t *testing.T) {
	// Create mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/chat":
			handleChatRequest(t, w, r)
		case "/api/embeddings":
			handleEmbeddingsRequest(t, w, r)
		case "/api/tags":
			handleTagsRequest(t, w, r)
		case "/api/pull":
			handlePullRequest(t, w, r)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL: server.URL,
		client:  &http.Client{},
		capabilities: interfaces.ProviderCapabilities{
			MaxTokens:      128000,
			ContextWindow:  128000,
			SupportsVision: true,
			SupportsTools:  false,
			SupportsStream: true,
			Models: []interfaces.ModelInfo{
				{ID: Llama31_8B, Name: "Llama 3.1 8B"},
			},
		},
	}

	t.Run("ChatCompletion", func(t *testing.T) {
		ctx := context.Background()
		req := interfaces.ChatRequest{
			Model: Llama31_8B,
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

		if resp.Model != Llama31_8B {
			t.Errorf("Wrong model in response: %s", resp.Model)
		}
	})

	t.Run("ListModels", func(t *testing.T) {
		ctx := context.Background()
		models, err := client.ListModels(ctx)
		if err != nil {
			t.Fatalf("ListModels failed: %v", err)
		}

		if len(models) == 0 {
			t.Error("No models returned")
		}

		// Check for expected models
		hasLlama := false
		for _, model := range models {
			if model == Llama31_8B {
				hasLlama = true
				break
			}
		}

		if !hasLlama {
			t.Error("Expected Llama model not found")
		}
	})

	t.Run("Embeddings", func(t *testing.T) {
		ctx := context.Background()
		req := interfaces.EmbeddingRequest{
			Model: "nomic-embed-text",
			Input: []string{"test embedding"},
		}

		resp, err := client.CreateEmbedding(ctx, req)
		if err != nil {
			t.Fatalf("CreateEmbedding failed: %v", err)
		}

		if len(resp.Embeddings) != 1 {
			t.Errorf("Expected 1 embedding, got %d", len(resp.Embeddings))
		}
	})

	t.Run("PullModel", func(t *testing.T) {
		ctx := context.Background()
		err := client.PullModel(ctx, "llama3.1:8b")
		if err != nil {
			t.Errorf("PullModel failed: %v", err)
		}
	})

	t.Run("FreeLocalModel", func(t *testing.T) {
		caps := client.GetCapabilities()

		// Verify all models are free (local)
		for _, model := range caps.Models {
			if model.InputCost != 0 || model.OutputCost != 0 {
				t.Errorf("Ollama model %s should be free, got costs: $%.2f/$%.2f",
					model.ID, model.InputCost, model.OutputCost)
			}
		}
	})

	t.Run("NoToolSupport", func(t *testing.T) {
		caps := client.GetCapabilities()
		if caps.SupportsTools {
			t.Error("Ollama should not support tools/functions yet")
		}
	})
}

func TestOllamaModelRecommendation(t *testing.T) {
	testCases := []struct {
		useCase  string
		expected string
	}{
		{"coding", Llama31_8B},
		{"reasoning", DeepSeekR1},
		{"vision", Llama32Vision},
		{"fast", Gemma2_2B},
		{"general", Llama31_8B},
		{"default", Mistral},
	}

	for _, tc := range testCases {
		t.Run(tc.useCase, func(t *testing.T) {
			model := GetRecommendedModel(tc.useCase)
			if model != tc.expected {
				t.Errorf("Expected %s for %s, got %s", tc.expected, tc.useCase, model)
			}
		})
	}
}

// Helper functions for mock server
func handleChatRequest(t *testing.T, w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"model":      req["model"],
		"created_at": "2024-01-01T00:00:00Z",
		"message": map[string]string{
			"role":    "assistant",
			"content": "Hello from Ollama!",
		},
		"done":                 true,
		"total_duration":       1000000000,
		"load_duration":        500000000,
		"prompt_eval_count":    10,
		"prompt_eval_duration": 100000000,
		"eval_count":           5,
		"eval_duration":        50000000,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleEmbeddingsRequest(t *testing.T, w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"embedding": make([]float64, 384), // Mock embedding
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleTagsRequest(t *testing.T, w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"models": []map[string]interface{}{
			{"name": Llama31_8B},
			{"name": Mistral},
			{"name": Gemma2_9B},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handlePullRequest(t *testing.T, w http.ResponseWriter, r *http.Request) {
	// Stream progress updates
	w.Header().Set("Content-Type", "application/x-ndjson")

	updates := []map[string]interface{}{
		{"status": "pulling manifest"},
		{"status": "downloading", "completed": 1000, "total": 5000},
		{"status": "verifying"},
		{"status": "success"},
	}

	for _, update := range updates {
		json.NewEncoder(w).Encode(update)
		w.(http.Flusher).Flush()
	}
}
