package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/corpus"
	"github.com/guild-ventures/guild-core/pkg/memory/rag"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
	"github.com/guild-ventures/guild-core/pkg/providers/mock"
)

func TestNewCorpusAgent(t *testing.T) {
	// Create mock dependencies
	mockProvider, err := mock.NewProvider()
	require.NoError(t, err)
	vectorConfig := &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		EmbeddingProvider: mockProvider,
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:   t.TempDir(),
			DefaultCollection: "test",
		},
	}

	ctx := context.Background()
	vectorStore, err := vector.NewVectorStore(ctx, vectorConfig)
	require.NoError(t, err)

	ragConfig := rag.Config{
		ChunkSize:    100,
		ChunkOverlap: 20,
		MaxResults:   5,
	}

	ragSystem := rag.NewRetrieverWithStore(vectorStore, ragConfig)

	corpusConfig := corpus.Config{
		CorpusPath:      t.TempDir(),
		MaxSizeBytes:    100 * 1024 * 1024,
		DefaultCategory: "test",
	}

	// Create agent
	agent := NewCorpusAgent(ragSystem, mockProvider, corpusConfig)

	// Verify initialization
	assert.NotNil(t, agent)
	assert.Equal(t, "corpus-agent-001", agent.ID)
	assert.Equal(t, "Corpus Knowledge Navigator", agent.Name)
	assert.NotNil(t, agent.ragSystem)
	assert.NotNil(t, agent.llmProvider)
	assert.Equal(t, corpusConfig, agent.corpusConfig)
	assert.Empty(t, agent.conversationHistory)
}

func TestCorpusAgent_Execute(t *testing.T) {
	ctx := context.Background()

	// Create mock provider with predefined response
	mockProvider, err := mock.NewProvider()
	require.NoError(t, err)
	mockProvider.SetResponse("What are agents called in Guild?",
		"Based on the documentation, agents in the Guild framework are called 'Artisans'. They work together in teams called 'Guilds'.")

	// Create RAG system with some test data
	vectorConfig := &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		EmbeddingProvider: mockProvider,
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:   t.TempDir(),
			DefaultCollection: "test",
		},
	}

	vectorStore, err := vector.NewVectorStore(ctx, vectorConfig)
	require.NoError(t, err)

	ragConfig := rag.Config{
		ChunkSize:    100,
		ChunkOverlap: 20,
		MaxResults:   5,
	}

	ragSystem := rag.NewRetrieverWithStore(vectorStore, ragConfig)

	// Add test document to RAG
	err = ragSystem.AddDocument(ctx, "guild-intro",
		"The Guild Framework orchestrates AI agents to work together on complex tasks. "+
			"Agents are called Artisans and work in teams called Guilds.", "test")
	require.NoError(t, err)

	corpusConfig := corpus.Config{
		CorpusPath:      t.TempDir(),
		MaxSizeBytes:    100 * 1024 * 1024,
		DefaultCategory: "test",
	}

	// Create agent and test execute
	agent := NewCorpusAgent(ragSystem, mockProvider, corpusConfig)

	response, err := agent.Execute(ctx, "What are agents called in Guild?")
	require.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Contains(t, response, "Artisans")

	// Check conversation history
	assert.Len(t, agent.conversationHistory, 2) // User query + assistant response
	assert.Equal(t, "user", agent.conversationHistory[0].Role)
	assert.Equal(t, "assistant", agent.conversationHistory[1].Role)
}

func TestCorpusAgent_GenerateDocument(t *testing.T) {
	ctx := context.Background()

	// Create mock provider
	mockProvider, err := mock.NewProvider()
	require.NoError(t, err)
	mockProvider.SetResponse("Explain the Guild architecture",
		"The Guild Framework uses a modular architecture with agents (Artisans), orchestrators, and a task management system.")

	// Create test environment
	tempDir := t.TempDir()

	vectorConfig := &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		EmbeddingProvider: mockProvider,
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:   tempDir + "/embeddings",
			DefaultCollection: "test",
		},
	}

	vectorStore, err := vector.NewVectorStore(ctx, vectorConfig)
	require.NoError(t, err)

	ragSystem := rag.NewRetrieverWithStore(vectorStore, rag.Config{
		ChunkSize:    100,
		ChunkOverlap: 20,
		MaxResults:   5,
	})

	corpusConfig := corpus.Config{
		CorpusPath:      tempDir + "/corpus",
		MaxSizeBytes:    100 * 1024 * 1024,
		DefaultCategory: "architecture",
	}

	// Create agent
	agent := NewCorpusAgent(ragSystem, mockProvider, corpusConfig)

	// Generate document
	title := "Guild Architecture Overview"
	doc, err := agent.GenerateDocument(ctx, "Explain the Guild architecture", title)

	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, title, doc.Title)
	assert.Contains(t, doc.Body, "modular architecture")
	assert.Contains(t, doc.Tags, "generated")
	assert.Contains(t, doc.Tags, "corpus-agent")
	assert.Contains(t, doc.Tags, "architecture") // Should extract from content
	assert.Equal(t, "corpus-agent", doc.Source)
	assert.Equal(t, "corpus", doc.GuildID)
	assert.Equal(t, agent.ID, doc.AgentID)
}

func TestCorpusAgent_SaveGeneratedDocument(t *testing.T) {
	ctx := context.Background()

	// Create minimal setup
	tempDir := t.TempDir()
	corpusPath := tempDir + "/corpus"

	mockProvider, err := mock.NewProvider()
	require.NoError(t, err)
	vectorStore, _ := vector.NewVectorStore(ctx, &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		EmbeddingProvider: mockProvider,
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:   tempDir + "/embeddings",
			DefaultCollection: "test",
		},
	})

	ragSystem := rag.NewRetrieverWithStore(vectorStore, rag.Config{})

	corpusConfig := corpus.Config{
		CorpusPath:      corpusPath,
		MaxSizeBytes:    100 * 1024 * 1024,
		DefaultCategory: "test",
	}

	agent := NewCorpusAgent(ragSystem, mockProvider, corpusConfig)

	// Test saving valid document
	doc := &corpus.CorpusDoc{
		Title:     "Test Document",
		Body:      "Test content",
		Tags:      []string{"test"},
		Source:    "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = agent.SaveGeneratedDocument(ctx, doc)
	require.NoError(t, err)

	// Verify file was saved
	savedDocs, err := corpus.List(ctx, corpusConfig)
	assert.NoError(t, err)
	assert.Len(t, savedDocs, 1)

	// Test validation errors
	emptyTitleDoc := &corpus.CorpusDoc{Body: "content"}
	err = agent.SaveGeneratedDocument(ctx, emptyTitleDoc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title is required")

	emptyBodyDoc := &corpus.CorpusDoc{Title: "title"}
	err = agent.SaveGeneratedDocument(ctx, emptyBodyDoc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "body is required")
}

func TestCorpusAgent_ConversationHistory(t *testing.T) {
	ctx := context.Background()

	// Create mock provider with default response
	mockProvider, err := mock.NewProvider()
	require.NoError(t, err)
	mockProvider.SetDefaultResponse("This is a response from the AI provider.")

	// Create minimal setup
	vectorStore, _ := vector.NewVectorStore(ctx, &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		EmbeddingProvider: mockProvider,
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:   t.TempDir(),
			DefaultCollection: "test",
		},
	})

	ragSystem := rag.NewRetrieverWithStore(vectorStore, rag.Config{})
	corpusConfig := corpus.Config{CorpusPath: t.TempDir()}

	agent := NewCorpusAgent(ragSystem, mockProvider, corpusConfig)

	// First message
	response1, err := agent.Execute(ctx, "What is Guild?")
	require.NoError(t, err)
	assert.NotEmpty(t, response1)

	// Second message should have context
	response2, err := agent.Execute(ctx, "Tell me more about agents")
	require.NoError(t, err)
	assert.NotEmpty(t, response2)

	// Verify conversation history
	assert.Len(t, agent.conversationHistory, 4) // 2 user + 2 assistant messages

	// Test history limit (20 messages)
	for i := 0; i < 20; i++ {
		_, _ = agent.Execute(ctx, "test message")
	}
	assert.Len(t, agent.conversationHistory, 20) // Should be capped at 20
}

func TestCorpusAgent_ClearHistory(t *testing.T) {
	// Create minimal agent
	agent := &CorpusAgent{
		conversationHistory: []Message{
			{Role: "user", Content: "test1", Timestamp: time.Now()},
			{Role: "assistant", Content: "response1", Timestamp: time.Now()},
		},
	}

	// Verify history exists
	assert.Len(t, agent.conversationHistory, 2)

	// Clear history
	agent.ClearHistory()

	// Verify history is cleared
	assert.Empty(t, agent.conversationHistory)
}

func TestCorpusAgent_ExtractTags(t *testing.T) {
	ctx := context.Background()

	mockProvider, err := mock.NewProvider()
	require.NoError(t, err)

	// Create minimal setup
	vectorStore, _ := vector.NewVectorStore(ctx, &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		EmbeddingProvider: mockProvider,
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:   t.TempDir(),
			DefaultCollection: "test",
		},
	})

	ragSystem := rag.NewRetrieverWithStore(vectorStore, rag.Config{})
	agent := NewCorpusAgent(ragSystem, mockProvider, corpus.Config{})

	// Test tag extraction
	tests := []struct {
		name         string
		query        string
		response     string
		expectedTags []string
	}{
		{
			name:         "API and interface keywords",
			query:        "How to implement the API interface?",
			response:     "The API interface implementation can be done by creating a class that follows the interface pattern.",
			expectedTags: []string{"generated", "corpus-agent", "api", "interface", "implementation", "class", "pattern"},
		},
		{
			name:         "Architecture keywords",
			query:        "Explain the system architecture",
			response:     "The system uses a modular design with components organized into modules.",
			expectedTags: []string{"generated", "corpus-agent", "system", "architecture", "design", "component", "module"},
		},
		{
			name:         "No matching keywords",
			query:        "Hello world",
			response:     "Hello! How can I help you today?",
			expectedTags: []string{"generated", "corpus-agent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := agent.extractTags(tt.query, tt.response)

			// Verify expected tags are present
			for _, expectedTag := range tt.expectedTags {
				assert.Contains(t, tags, expectedTag)
			}

			// Verify tag limit
			assert.LessOrEqual(t, len(tags), 10)
		})
	}
}

func TestCorpusAgent_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		query       string
		setupMock   func(*mock.Provider)
		expectError string
	}{
		{
			name:  "LLM error",
			query: "test query",
			setupMock: func(p *mock.Provider) {
				p.SetError("test query", errors.New("LLM service unavailable"))
			},
			expectError: "failed to generate response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock provider
			mockProvider, err := mock.NewProvider()
			require.NoError(t, err)
			if tt.setupMock != nil {
				tt.setupMock(mockProvider)
			}

			// Create minimal setup
			vectorStore, _ := vector.NewVectorStore(ctx, &vector.StoreConfig{
				Type:              vector.StoreTypeChromem,
				EmbeddingProvider: mockProvider,
				ChromemConfig: vector.ChromemConfig{
					PersistencePath:   t.TempDir(),
					DefaultCollection: "test",
				},
			})

			ragSystem := rag.NewRetrieverWithStore(vectorStore, rag.Config{})
			agent := NewCorpusAgent(ragSystem, mockProvider, corpus.Config{})

			// Test execute
			_, err = agent.Execute(ctx, tt.query)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

func TestCorpusAgent_GettersAndIdentity(t *testing.T) {
	agent := &CorpusAgent{
		ID:   "test-id",
		Name: "Test Agent",
	}

	assert.Equal(t, "test-id", agent.GetID())
	assert.Equal(t, "Test Agent", agent.GetName())
}
