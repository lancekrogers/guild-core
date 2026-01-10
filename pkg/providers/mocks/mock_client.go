// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"context"
	"fmt"

	"github.com/lancekrogers/guild-core/pkg/providers/interfaces"
)

// MockClient provides a mock implementation of the LLMClient interface for testing
type MockClient struct {
	CompletionResponses []string
	EmbeddingResponses  [][]float32
	CallCount           int
}

// NewMockClient creates a new mock client
func NewMockClient() *MockClient {
	return &MockClient{
		CompletionResponses: []string{"Mock response"},
		EmbeddingResponses:  [][]float32{{0.1, 0.2, 0.3}},
	}
}

// Complete generates a completion for the given prompt
func (c *MockClient) Complete(ctx context.Context, prompt string) (string, error) {
	c.CallCount++
	if len(c.CompletionResponses) > 0 {
		return c.CompletionResponses[0], nil
	}
	return fmt.Sprintf("Mock response for: %s", prompt), nil
}

// CreateCompletion is a lower-level method to create a completion
func (c *MockClient) CreateCompletion(ctx context.Context, req *interfaces.CompletionRequest) (*interfaces.CompletionResponse, error) {
	c.CallCount++
	return &interfaces.CompletionResponse{
		Text:         fmt.Sprintf("Mock response for: %s", req.Prompt),
		TokensUsed:   10,
		TokensInput:  5,
		TokensOutput: 5,
		ModelUsed:    "mock-model",
	}, nil
}

// CreateEmbedding generates an embedding for the given text
func (c *MockClient) CreateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	c.CallCount++
	return &interfaces.EmbeddingResponse{
		Model: "mock-embedding-model",
		Embeddings: []interfaces.Embedding{
			{
				Index:     0,
				Embedding: []float64{0.1, 0.2, 0.3},
			},
		},
		Usage: interfaces.UsageInfo{
			PromptTokens:     10,
			CompletionTokens: 0,
			TotalTokens:      10,
		},
	}, nil
}

// CreateEmbeddings generates embeddings for multiple texts
func (c *MockClient) CreateEmbeddings(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	c.CallCount++
	embeddings := []interfaces.Embedding{}
	for i := range req.Input {
		embeddings = append(embeddings, interfaces.Embedding{
			Index:     i,
			Embedding: []float64{0.1 * float64(i+1), 0.2 * float64(i+1), 0.3 * float64(i+1)},
		})
	}
	return &interfaces.EmbeddingResponse{
		Model:      "mock-embedding-model",
		Embeddings: embeddings,
		Usage: interfaces.UsageInfo{
			PromptTokens:     len(req.Input) * 10,
			CompletionTokens: 0,
			TotalTokens:      len(req.Input) * 10,
		},
	}, nil
}
