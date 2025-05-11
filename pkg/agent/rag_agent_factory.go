package agent

import (
	"github.com/blockhead-consulting/guild/pkg/corpus"
	"github.com/blockhead-consulting/guild/pkg/memory"
	"github.com/blockhead-consulting/guild/pkg/memory/rag"
	"github.com/blockhead-consulting/guild/pkg/memory/vector"
	"github.com/blockhead-consulting/guild/pkg/objective"
	"github.com/blockhead-consulting/guild/pkg/providers"
	"github.com/blockhead-consulting/guild/tools"
)

// RagAgentFactoryOptions contains options for the RAG agent factory
type RagAgentFactoryOptions struct {
	// VectorStoreConfig is the configuration for the vector store
	VectorStoreConfig vector.StoreConfig

	// RagOptions is the configuration for the RAG agent
	RagOptions rag.RagAgentOptions

	// CorpusConfig is the configuration for the corpus
	CorpusConfig corpus.Config
}

// DefaultRagAgentFactoryOptions returns default options for the RAG agent factory
func DefaultRagAgentFactoryOptions() RagAgentFactoryOptions {
	return RagAgentFactoryOptions{
		VectorStoreConfig: vector.StoreConfig{
			Type:      vector.StoreTypeChromem,
			Path:      "./data/vectorstore",
			EmbedderConfig: vector.EmbedderConfig{
				Type:  vector.EmbedderTypeOpenAI,
				Model: "text-embedding-ada-002",
			},
		},
		RagOptions:    rag.DefaultRagAgentOptions(),
		CorpusConfig:  corpus.DefaultConfig(),
	}
}

// RagAgentFactory creates agents with RAG capabilities
type RagAgentFactory struct {
	options      RagAgentFactoryOptions
	vectorStore  vector.VectorStore
	retriever    *rag.Retriever
	memoryManager memory.ChainManager
	toolRegistry *tools.ToolRegistry
	objectiveMgr *objective.Manager
}

// NewRagAgentFactory creates a new RAG agent factory
func NewRagAgentFactory(
	memoryManager memory.ChainManager,
	toolRegistry *tools.ToolRegistry,
	objectiveMgr *objective.Manager,
	options RagAgentFactoryOptions,
) (*RagAgentFactory, error) {
	// Create vector store
	vectorStoreFactory := vector.NewFactory()
	vectorStore, err := vectorStoreFactory.CreateVectorStore(options.VectorStoreConfig)
	if err != nil {
		return nil, err
	}

	// Create retriever
	retriever := rag.NewRetriever(vectorStore, options.CorpusConfig)

	// Create chunker with configured options
	chunker := rag.NewChunker(
		rag.WithChunkSize(options.RagOptions.ChunkSize),
		rag.WithChunkOverlap(options.RagOptions.ChunkOverlap),
		rag.WithSplitStrategy(options.RagOptions.ChunkStrategy),
	)
	retriever.WithChunker(chunker)

	return &RagAgentFactory{
		options:      options,
		vectorStore:  vectorStore,
		retriever:    retriever,
		memoryManager: memoryManager,
		toolRegistry: toolRegistry,
		objectiveMgr: objectiveMgr,
	}, nil
}

// CreateAgent creates a new RAG-enabled agent
func (f *RagAgentFactory) CreateAgent(config *AgentConfig, llmClient providers.LLMClient) (GuildArtisan, error) {
	// Create a RAG agent
	ragAgent := rag.NewRagAgent(
		config,
		llmClient,
		f.memoryManager,
		f.toolRegistry,
		f.objectiveMgr,
		f.retriever,
		f.options.RagOptions,
	)

	return ragAgent, nil
}

// GetRetriever returns the RAG retriever
func (f *RagAgentFactory) GetRetriever() *rag.Retriever {
	return f.retriever
}

// Close closes the vector store
func (f *RagAgentFactory) Close() error {
	return f.vectorStore.Close()
}