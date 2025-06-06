package rag

import (
	"context"
	
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
	"github.com/guild-ventures/guild-core/internal/project"
)

// NewProjectAwareFactory creates a RAG factory using project context
func NewProjectAwareFactory(ctx context.Context) (*Factory, error) {
	// Get project context
	projCtx, err := project.GetContext()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound).
			WithComponent("memory").
			WithOperation("NewProjectAwareFactory").
			WithDetails("not in a guild project")
	}
	
	// Create project-specific configuration
	config := Config{
		ChunkSize:    1000,
		ChunkOverlap: 200,
		MaxResults:   10,
		UseCorpus:    true,
		CorpusPath:   projCtx.GetCorpusPath(),
	}
	
	// Create vector store configuration using project paths
	vectorConfig := &vector.StoreConfig{
		Type: vector.StoreTypeChromem,
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:   projCtx.GetEmbeddingsPath(),
			DefaultCollection: "project",
		},
	}
	
	// Create vector store
	vectorStore, err := vector.NewVectorStore(ctx, vectorConfig)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage).
			WithComponent("memory").
			WithOperation("NewProjectAwareFactory").
			WithDetails("failed to create vector store")
	}
	
	// Create retriever with vector store
	retriever := newRetrieverWithStore(vectorStore, config)
	
	// Create factory
	factory := &Factory{
		retriever: retriever,
		embedder:  nil, // Embedder will be set by vector store
	}
	
	return factory, nil
}

// NewProjectAwareRetriever creates a retriever using project context
func NewProjectAwareRetriever(ctx context.Context) (*Retriever, error) {
	factory, err := NewProjectAwareFactory(ctx)
	if err != nil {
		return nil, err
	}
	
	return factory.GetRetriever(), nil
}

// GetProjectRAGConfig returns RAG configuration for the current project
func GetProjectRAGConfig(ctx context.Context) (Config, error) {
	// Try to get project context from context.Context
	if projCtx, ok := project.FromContext(ctx); ok {
		return Config{
			ChunkSize:    1000,
			ChunkOverlap: 200,
			MaxResults:   10,
			UseCorpus:    true,
			CorpusPath:   projCtx.GetCorpusPath(),
		}, nil
	}
	
	// Try to get project context from current directory
	projCtx, err := project.GetContext()
	if err != nil {
		return Config{}, gerror.Wrap(err, gerror.ErrCodeNotFound).
			WithComponent("memory").
			WithOperation("GetProjectRAGConfig").
			WithDetails("not in a guild project")
	}
	
	return Config{
		ChunkSize:    1000,
		ChunkOverlap: 200,
		MaxResults:   10,
		UseCorpus:    true,
		CorpusPath:   projCtx.GetCorpusPath(),
	}, nil
}