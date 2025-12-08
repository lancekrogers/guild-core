// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package vector

import (
	"context"

	openai "github.com/sashabaranov/go-openai"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// OpenAIEmbedder implements the Embedder interface using OpenAI
type OpenAIEmbedder struct {
	client *openai.Client
	model  openai.EmbeddingModel
}

// NewOpenAIEmbedder creates a new OpenAI embedder
func NewOpenAIEmbedder(apiKey string, modelStr string) (*OpenAIEmbedder, error) {
	if apiKey == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "API key is required", nil).
			WithComponent("memory").
			WithOperation("NewOpenAIEmbedder")
	}

	// Convert to EmbeddingModel
	model := openai.AdaEmbeddingV2
	if modelStr != "" {
		model = openai.EmbeddingModel(modelStr)
	}

	client := openai.NewClient(apiKey)

	return &OpenAIEmbedder{
		client: client,
		model:  model,
	}, nil
}

// Embed generates an embedding from text
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	response, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: e.model,
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create embeddings").
			WithComponent("memory").
			WithOperation("Embed")
	}

	if len(response.Data) == 0 {
		return nil, gerror.New(gerror.ErrCodeInternal, "no embeddings returned", nil).
			WithComponent("memory").
			WithOperation("Embed")
	}

	return response.Data[0].Embedding, nil
}

// GetEmbedding is an alias for Embed
func (e *OpenAIEmbedder) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	return e.Embed(ctx, text)
}

// GetEmbeddings gets embeddings for multiple texts
func (e *OpenAIEmbedder) GetEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	response, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: e.model,
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create embeddings").
			WithComponent("memory").
			WithOperation("GetEmbeddings")
	}

	if len(response.Data) != len(texts) {
		return nil, gerror.Newf(gerror.ErrCodeInternal, "expected %d embeddings, got %d", len(texts), len(response.Data)).
			WithComponent("memory").
			WithOperation("GetEmbeddings")
	}

	embeddings := make([][]float32, len(response.Data))
	for i, data := range response.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}
