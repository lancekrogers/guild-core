package rag

import (
	"github.com/blockhead-consulting/guild/pkg/agent"
	"github.com/blockhead-consulting/guild/pkg/corpus"
	"github.com/blockhead-consulting/guild/pkg/memory"
	"github.com/blockhead-consulting/guild/pkg/memory/vector"
	"github.com/blockhead-consulting/guild/pkg/objective"
	"github.com/blockhead-consulting/guild/pkg/providers"
	"github.com/blockhead-consulting/guild/tools"
)

// FactoryOptions contains options for the RAG factory
type FactoryOptions struct {
	// VectorStoreConfig is the configuration for the vector store
	VectorStoreConfig vector.StoreConfig

	// RagOptions is the configuration for the RAG agent
	RagOptions RagAgentOptions

	// CorpusConfig is the configuration for the corpus
	CorpusConfig corpus.Config
}

// DefaultFactoryOptions returns default options for the RAG factory
func DefaultFactoryOptions() FactoryOptions {
	return FactoryOptions{
		VectorStoreConfig: vector.StoreConfig{
			Type: vector.StoreTypeChromem,
			ChromemConfig: vector.ChromemConfig{
				PersistencePath:  "./data/vectorstore",
				DefaultDimension: 1536,
			},
		},
		RagOptions:   DefaultRagAgentOptions(),
		CorpusConfig: corpus.DefaultConfig(),
	}
}

// Factory creates components with RAG capabilities
type Factory struct {
	options      FactoryOptions
	vectorStore  vector.VectorStore
	retriever    *Retriever
	memoryManager memory.ChainManager
	toolRegistry *tools.ToolRegistry
	objectiveMgr *objective.Manager
}

// NewFactory creates a new RAG factory
func NewFactory(
	memoryManager memory.ChainManager,
	toolRegistry *tools.ToolRegistry,
	objectiveMgr *objective.Manager,
	options FactoryOptions,
	vectorStore vector.VectorStore,
) (*Factory, error) {
	// Create retriever
	retriever := NewRetriever(vectorStore, options.CorpusConfig)

	// Create chunker with configured options
	chunker := NewChunker(
		WithChunkSize(options.RagOptions.ChunkSize),
		WithChunkOverlap(options.RagOptions.ChunkOverlap),
		WithSplitStrategy(options.RagOptions.ChunkStrategy),
	)
	retriever.WithChunker(chunker)

	return &Factory{
		options:      options,
		vectorStore:  vectorStore,
		retriever:    retriever,
		memoryManager: memoryManager,
		toolRegistry: toolRegistry,
		objectiveMgr: objectiveMgr,
	}, nil
}

// EnhanceAgent wraps an existing agent with RAG capabilities
func (f *Factory) EnhanceAgent(agent agent.GuildArtisan) agent.GuildArtisan {
	// Create retrieval config
	config := RetrievalConfig{
		MaxResults:    f.options.RagOptions.MaxResults,
		MinScore:      f.options.RagOptions.MinScore,
		IncludeCorpus: f.options.RagOptions.IncludeCorpus,
		ChunkSize:     f.options.RagOptions.ChunkSize,
		ChunkOverlap:  f.options.RagOptions.ChunkOverlap,
		ChunkStrategy: f.options.RagOptions.ChunkStrategy,
	}

	// Wrap the agent with RAG capabilities
	return NewAgentWrapper(agent, f.retriever, config)
}

// CreateRagAgent creates a new agent with RAG capabilities using the base agent factory
func (f *Factory) CreateRagAgent(
	baseAgentFactory agent.Factory,
	config *agent.AgentConfig,
	llmClient providers.LLMClient,
) (agent.GuildArtisan, error) {
	// Create a base agent
	baseAgent, err := baseAgentFactory.CreateAgent(config, llmClient)
	if err != nil {
		return nil, err
	}

	// Enhance with RAG capabilities
	return f.EnhanceAgent(baseAgent), nil
}

// GetRetriever returns the RAG retriever
func (f *Factory) GetRetriever() *Retriever {
	return f.retriever
}

// Close closes the vector store
func (f *Factory) Close() error {
	return f.vectorStore.Close()
}