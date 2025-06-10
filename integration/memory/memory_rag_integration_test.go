package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/internal/testutil"
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/context"
	"github.com/guild-ventures/guild-core/pkg/corpus"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/memory/rag"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
	"github.com/guild-ventures/guild-core/pkg/project"
)

// TestCorpusScanningAndIndexing tests scanning project files and indexing in vector store
func TestCorpusScanningAndIndexing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Create test project files
	projectDir := projCtx.GetProjectPath()
	testFiles := map[string]string{
		"README.md": `# Test Project
This is a test project for RAG integration.

## Features
- User authentication
- Product catalog
- Shopping cart
- Payment processing`,

		"api/auth.go": `package api

// AuthHandler handles user authentication
func AuthHandler(w http.ResponseWriter, r *http.Request) {
    // Validate credentials
    username := r.FormValue("username")
    password := r.FormValue("password")
    
    if validateUser(username, password) {
        // Generate JWT token
        token := generateJWT(username)
        w.Write([]byte(token))
    }
}`,

		"api/products.go": `package api

// ProductHandler manages product operations
type ProductHandler struct {
    db *sql.DB
}

// ListProducts returns all products
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
    products := h.getProducts()
    json.NewEncoder(w).Encode(products)
}`,

		"docs/api.md": `# API Documentation

## Authentication Endpoints

### POST /auth/login
Authenticates a user and returns a JWT token.

### POST /auth/logout
Invalidates the current session.

## Product Endpoints

### GET /products
Returns a list of all products.

### GET /products/:id
Returns details for a specific product.`,
	}

	// Create test files
	for path, content := range testFiles {
		fullPath := filepath.Join(projectDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create corpus storage
	corpusStorage := corpus.NewStorage(filepath.Join(projCtx.GetGuildPath(), "corpus"))

	// Create vector store
	vectorStore := testutil.NewMockVectorStore()

	// Create corpus configuration
	config := &corpus.Config{
		RootDir: projectDir,
		IncludePatterns: []string{
			"**/*.go",
			"**/*.md",
		},
		ExcludePatterns: []string{
			"**/vendor/**",
			"**/.git/**",
		},
	}

	// Scan project files
	activity := corpus.NewActivity("test-scan", "Testing corpus scan")
	
	scannedFiles := make([]string, 0)
	err := corpus.ScanDirectory(config.RootDir, config, func(path string, info os.FileInfo) error {
		if !info.IsDir() {
			scannedFiles = append(scannedFiles, path)
			activity.AddFile(path)
		}
		return nil
	})
	require.NoError(t, err)

	// Verify files were scanned
	assert.Len(t, scannedFiles, 4, "Should scan all test files")

	// Index files in vector store
	chunker := rag.NewChunker(rag.ChunkerConfig{
		ChunkSize:    500,
		ChunkOverlap: 50,
	})

	indexedCount := 0
	for _, filePath := range scannedFiles {
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)

		// Chunk the content
		chunks := chunker.Chunk(string(content))

		// Index chunks
		for i, chunk := range chunks {
			doc := vector.Document{
				ID:      fmt.Sprintf("%s-chunk-%d", filePath, i),
				Content: chunk,
				Metadata: map[string]interface{}{
					"file":      filePath,
					"chunk_idx": i,
					"type":      detectFileType(filePath),
				},
			}

			err = vectorStore.Add(ctx, doc)
			require.NoError(t, err)
			indexedCount++
		}
	}

	// Verify indexing
	assert.Greater(t, indexedCount, len(scannedFiles), "Should create multiple chunks per file")

	// Test corpus graph building
	graph := corpus.NewGraph()
	
	// Add nodes for files
	for _, file := range scannedFiles {
		node := &corpus.Node{
			ID:   file,
			Type: corpus.NodeTypeFile,
			Data: map[string]interface{}{
				"path": file,
				"type": detectFileType(file),
			},
		}
		graph.AddNode(node)
	}

	// Add relationships based on content
	// For example, link auth.go to api.md because both mention authentication
	graph.AddLink(&corpus.Link{
		Source: filepath.Join(projectDir, "api/auth.go"),
		Target: filepath.Join(projectDir, "docs/api.md"),
		Type:   corpus.LinkTypeReference,
		Weight: 0.8,
	})

	// Verify graph structure
	nodes := graph.GetNodes()
	assert.Len(t, nodes, 4, "Should have all files as nodes")

	links := graph.GetLinks()
	assert.Greater(t, len(links), 0, "Should have relationships between files")

	// Save corpus activity
	err = corpusStorage.SaveActivity(activity)
	require.NoError(t, err)

	// Verify activity was saved
	activities, err := corpusStorage.ListActivities()
	require.NoError(t, err)
	assert.Len(t, activities, 1, "Should have saved activity")
}

// TestVectorSearchIntegration tests vector search functionality
func TestVectorSearchIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Create vector store with mock embedder
	vectorStore := vector.NewChromemStore(vector.ChromemConfig{
		PersistencePath: filepath.Join(projCtx.GetGuildPath(), "test-vectors"),
		Dimension:       384, // Mock dimension
	})

	// Create mock embedder
	mockEmbedder := &mockEmbedder{
		dimension: 384,
	}

	// Initialize vector store with embedder
	err := vectorStore.Initialize(ctx, mockEmbedder)
	require.NoError(t, err)

	// Add test documents
	testDocs := []vector.Document{
		{
			ID:      "doc-1",
			Content: "User authentication is handled by the AuthHandler function which validates credentials and generates JWT tokens",
			Metadata: map[string]interface{}{
				"file": "auth.go",
				"type": "code",
			},
		},
		{
			ID:      "doc-2",
			Content: "The product catalog API provides endpoints for listing products and retrieving product details",
			Metadata: map[string]interface{}{
				"file": "products.go",
				"type": "code",
			},
		},
		{
			ID:      "doc-3",
			Content: "Shopping cart functionality allows users to add, remove, and update items in their cart",
			Metadata: map[string]interface{}{
				"file": "cart.go",
				"type": "code",
			},
		},
		{
			ID:      "doc-4",
			Content: "Payment processing integrates with Stripe API for secure credit card transactions",
			Metadata: map[string]interface{}{
				"file": "payment.go",
				"type": "code",
			},
		},
	}

	// Index documents
	for _, doc := range testDocs {
		err = vectorStore.Add(ctx, doc)
		require.NoError(t, err)
	}

	// Test searches
	searches := []struct {
		query          string
		expectedInTop3 []string
	}{
		{
			query:          "How does user authentication work?",
			expectedInTop3: []string{"doc-1"}, // Should find auth doc
		},
		{
			query:          "Show me the product API",
			expectedInTop3: []string{"doc-2"}, // Should find product doc
		},
		{
			query:          "How to process payments?",
			expectedInTop3: []string{"doc-4"}, // Should find payment doc
		},
		{
			query:          "JWT token generation authentication",
			expectedInTop3: []string{"doc-1"}, // Should find auth doc with JWT mention
		},
	}

	for _, search := range searches {
		t.Run(search.query, func(t *testing.T) {
			results, err := vectorStore.Search(ctx, search.query, 3)
			require.NoError(t, err)
			assert.NotEmpty(t, results, "Should return search results")

			// Check if expected documents are in top results
			resultIDs := make([]string, len(results))
			for i, result := range results {
				resultIDs[i] = result.ID
			}

			for _, expectedID := range search.expectedInTop3 {
				assert.Contains(t, resultIDs, expectedID, 
					"Expected %s in top 3 results for query: %s", expectedID, search.query)
			}
		})
	}

	// Test similarity search
	t.Run("SimilaritySearch", func(t *testing.T) {
		// Find similar documents to authentication
		similar, err := vectorStore.Search(ctx, testDocs[0].Content, 2)
		require.NoError(t, err)
		assert.Len(t, similar, 2, "Should return 2 similar documents")
		
		// First result should be the document itself
		assert.Equal(t, "doc-1", similar[0].ID)
	})
}

// TestContextRetrievalForAgents tests retrieving relevant context for agent queries
func TestContextRetrievalForAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Create RAG components
	vectorStore := testutil.NewMockVectorStore()
	
	// Add knowledge base
	knowledgeBase := []vector.Document{
		{
			ID:      "kb-1",
			Content: "The Guild Framework uses a manager agent to break down commissions into tasks",
			Metadata: map[string]interface{}{"source": "architecture.md"},
		},
		{
			ID:      "kb-2",
			Content: "Worker agents execute assigned tasks and can use tools like file operations and HTTP requests",
			Metadata: map[string]interface{}{"source": "agents.md"},
		},
		{
			ID:      "kb-3",
			Content: "The orchestrator coordinates multiple agents using an event-driven architecture",
			Metadata: map[string]interface{}{"source": "orchestration.md"},
		},
		{
			ID:      "kb-4",
			Content: "Campaigns represent long-running initiatives with multiple objectives and tasks",
			Metadata: map[string]interface{}{"source": "campaigns.md"},
		},
	}

	for _, doc := range knowledgeBase {
		err := vectorStore.Add(ctx, doc)
		require.NoError(t, err)
	}

	// Create retriever
	retriever := rag.NewRetriever(vectorStore, rag.RetrieverConfig{
		TopK:             3,
		ScoreThreshold:   0.7,
		IncludeMetadata:  true,
	})

	// Create mock provider
	mockProvider := testutil.NewMockLLMProvider()

	// Test agent queries with context retrieval
	testQueries := []struct {
		agentType      string
		query          string
		expectedContext []string
	}{
		{
			agentType:      "manager",
			query:          "How should I break down this commission?",
			expectedContext: []string{"kb-1", "kb-4"}, // Manager and campaign info
		},
		{
			agentType:      "worker",
			query:          "What tools can I use for this task?",
			expectedContext: []string{"kb-2"}, // Worker agent info
		},
		{
			agentType:      "coordinator",
			query:          "How do agents work together?",
			expectedContext: []string{"kb-3", "kb-1"}, // Orchestration info
		},
	}

	for _, test := range testQueries {
		t.Run(test.agentType, func(t *testing.T) {
			// Retrieve context
			contexts, err := retriever.Retrieve(ctx, test.query)
			require.NoError(t, err)

			// Verify relevant context was retrieved
			retrievedIDs := make([]string, len(contexts))
			for i, ctx := range contexts {
				retrievedIDs[i] = ctx.ID
			}

			for _, expectedID := range test.expectedContext {
				assert.Contains(t, retrievedIDs, expectedID,
					"Expected context %s for query: %s", expectedID, test.query)
			}

			// Create agent with RAG context
			agentCtx := &context.AgentContext{
				ProjectContext: projCtx,
				CostManager:    context.NewCostManager(nil),
				ToolRegistry:   testutil.NewMockToolRegistry(),
				ProviderName:   "mock",
				Provider:       mockProvider,
			}

			// Configure mock response that uses context
			contextStr := formatContexts(contexts)
			mockProvider.SetResponse(test.agentType, &testutil.MockAgentResponse{
				Content: fmt.Sprintf("Based on the context:\n%s\n\nMy response to '%s'", 
					contextStr, test.query),
			})

			// Create agent
			agent := agent.NewContextAgent(
				test.agentType,
				fmt.Sprintf("Test %s", test.agentType),
				test.agentType,
				agentCtx,
			)

			// Execute with context
			response, err := agent.Execute(ctx, test.query)
			require.NoError(t, err)
			assert.Contains(t, response, "Based on the context")
		})
	}
}

// TestKnowledgePersistenceAcrossSessions tests that knowledge persists between sessions
func TestKnowledgePersistenceAcrossSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)
	persistPath := filepath.Join(projCtx.GetGuildPath(), "persistent-vectors")

	// Session 1: Add knowledge
	t.Run("Session1_AddKnowledge", func(t *testing.T) {
		// Create vector store
		vectorStore := vector.NewChromemStore(vector.ChromemConfig{
			PersistencePath: persistPath,
			Dimension:       384,
		})

		mockEmbedder := &mockEmbedder{dimension: 384}
		err := vectorStore.Initialize(ctx, mockEmbedder)
		require.NoError(t, err)

		// Add documents
		docs := []vector.Document{
			{
				ID:      "session1-1",
				Content: "In session 1, we learned about user authentication patterns",
			},
			{
				ID:      "session1-2",
				Content: "Session 1 also covered database schema design principles",
			},
		}

		for _, doc := range docs {
			err = vectorStore.Add(ctx, doc)
			require.NoError(t, err)
		}

		// Force persistence
		if persister, ok := vectorStore.(interface{ Persist() error }); ok {
			err = persister.Persist()
			require.NoError(t, err)
		}
	})

	// Session 2: Verify persistence and add more
	t.Run("Session2_VerifyAndExtend", func(t *testing.T) {
		// Create new vector store instance
		vectorStore := vector.NewChromemStore(vector.ChromemConfig{
			PersistencePath: persistPath,
			Dimension:       384,
		})

		mockEmbedder := &mockEmbedder{dimension: 384}
		err := vectorStore.Initialize(ctx, mockEmbedder)
		require.NoError(t, err)

		// Search for session 1 knowledge
		results, err := vectorStore.Search(ctx, "authentication patterns from session 1", 2)
		require.NoError(t, err)
		assert.NotEmpty(t, results, "Should find session 1 documents")

		foundSession1 := false
		for _, result := range results {
			if strings.Contains(result.ID, "session1") {
				foundSession1 = true
				break
			}
		}
		assert.True(t, foundSession1, "Should find documents from session 1")

		// Add new knowledge
		newDoc := vector.Document{
			ID:      "session2-1",
			Content: "In session 2, we built upon session 1 knowledge about authentication",
		}
		err = vectorStore.Add(ctx, newDoc)
		require.NoError(t, err)
	})

	// Session 3: Verify all knowledge
	t.Run("Session3_VerifyAll", func(t *testing.T) {
		// Create another new instance
		vectorStore := vector.NewChromemStore(vector.ChromemConfig{
			PersistencePath: persistPath,
			Dimension:       384,
		})

		mockEmbedder := &mockEmbedder{dimension: 384}
		err := vectorStore.Initialize(ctx, mockEmbedder)
		require.NoError(t, err)

		// Search across all sessions
		results, err := vectorStore.Search(ctx, "authentication knowledge from all sessions", 5)
		require.NoError(t, err)

		// Verify we have documents from both sessions
		session1Count := 0
		session2Count := 0
		for _, result := range results {
			if strings.Contains(result.ID, "session1") {
				session1Count++
			} else if strings.Contains(result.ID, "session2") {
				session2Count++
			}
		}

		assert.Greater(t, session1Count, 0, "Should have session 1 documents")
		assert.Greater(t, session2Count, 0, "Should have session 2 documents")
	})
}

// TestMultiAgentMemorySharing tests memory sharing between multiple agents
func TestMultiAgentMemorySharing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Create shared memory components
	sharedVectorStore := testutil.NewMockVectorStore()
	sharedMemoryManager := memory.NewManager(memory.ManagerConfig{
		VectorStore: sharedVectorStore,
	})

	// Create mock provider
	mockProvider := testutil.NewMockLLMProvider()

	// Agent 1: Architect adds design knowledge
	t.Run("ArchitectAddsKnowledge", func(t *testing.T) {
		// Create architect agent
		architectCtx := &context.AgentContext{
			ProjectContext:  projCtx,
			CostManager:     context.NewCostManager(nil),
			ToolRegistry:    testutil.NewMockToolRegistry(),
			MemoryManager:   sharedMemoryManager,
			ProviderName:    "mock",
			Provider:        mockProvider,
		}

		architect := agent.NewContextAgent(
			"architect",
			"System Architect",
			"specialist",
			architectCtx,
		)

		// Architect creates design
		design := "The system will use microservices architecture with API Gateway pattern"
		
		// Store in shared memory
		err := sharedMemoryManager.Store(ctx, memory.Entry{
			ID:      "design-1",
			AgentID: "architect",
			Type:    "design",
			Content: design,
			Metadata: map[string]interface{}{
				"timestamp": time.Now(),
				"category":  "architecture",
			},
		})
		require.NoError(t, err)

		// Architect executes task
		_, err = architect.Execute(ctx, "Document the system architecture")
		require.NoError(t, err)
	})

	// Agent 2: Developer uses architect's knowledge
	t.Run("DeveloperUsesKnowledge", func(t *testing.T) {
		// Create developer agent with same memory
		developerCtx := &context.AgentContext{
			ProjectContext:  projCtx,
			CostManager:     context.NewCostManager(nil),
			ToolRegistry:    testutil.NewMockToolRegistry(),
			MemoryManager:   sharedMemoryManager, // Shared memory
			ProviderName:    "mock",
			Provider:        mockProvider,
		}

		developer := agent.NewContextAgent(
			"developer",
			"Senior Developer",
			"worker",
			developerCtx,
		)

		// Developer queries shared memory
		memories, err := sharedMemoryManager.Recall(ctx, "microservices architecture", 1)
		require.NoError(t, err)
		assert.NotEmpty(t, memories, "Developer should find architect's design")

		// Verify it's the architect's design
		assert.Equal(t, "architect", memories[0].AgentID)
		assert.Contains(t, memories[0].Content, "microservices")

		// Developer adds implementation details
		err = sharedMemoryManager.Store(ctx, memory.Entry{
			ID:      "impl-1",
			AgentID: "developer",
			Type:    "implementation",
			Content: "Implemented API Gateway using Kong, following architect's microservices design",
			Metadata: map[string]interface{}{
				"references": "design-1",
				"timestamp":  time.Now(),
			},
		})
		require.NoError(t, err)
	})

	// Agent 3: Tester uses both agents' knowledge
	t.Run("TesterUsesAllKnowledge", func(t *testing.T) {
		// Create tester agent
		testerCtx := &context.AgentContext{
			ProjectContext:  projCtx,
			CostManager:     context.NewCostManager(nil),
			ToolRegistry:    testutil.NewMockToolRegistry(),
			MemoryManager:   sharedMemoryManager, // Shared memory
			ProviderName:    "mock",
			Provider:        mockProvider,
		}

		tester := agent.NewContextAgent(
			"tester",
			"QA Engineer",
			"worker",
			testerCtx,
		)

		// Tester queries for all relevant knowledge
		memories, err := sharedMemoryManager.Recall(ctx, "API Gateway Kong testing", 3)
		require.NoError(t, err)
		assert.Len(t, memories, 2, "Should find both architect and developer entries")

		// Verify cross-agent knowledge sharing
		agentIDs := make(map[string]bool)
		for _, mem := range memories {
			agentIDs[mem.AgentID] = true
		}
		assert.True(t, agentIDs["architect"], "Should have architect's knowledge")
		assert.True(t, agentIDs["developer"], "Should have developer's knowledge")
	})

	// Verify memory statistics
	t.Run("MemoryStatistics", func(t *testing.T) {
		stats := sharedMemoryManager.GetStatistics()
		assert.Equal(t, 2, stats.TotalEntries, "Should have 2 memory entries")
		assert.Equal(t, 2, stats.UniqueAgents, "Should have 2 unique agents")
		assert.Contains(t, stats.EntriesByType, "design")
		assert.Contains(t, stats.EntriesByType, "implementation")
	})
}

// TestRAGEnhancedAgentResponses tests agents using RAG to enhance their responses
func TestRAGEnhancedAgentResponses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Create RAG system
	vectorStore := testutil.NewMockVectorStore()
	retriever := rag.NewRetriever(vectorStore, rag.RetrieverConfig{
		TopK:            3,
		ScoreThreshold:  0.7,
		IncludeMetadata: true,
	})

	// Create RAG agent
	ragAgent := rag.NewRAGAgent(rag.RAGAgentConfig{
		VectorStore: vectorStore,
		Retriever:   retriever,
		AgentID:     "rag-enhanced-agent",
	})

	// Populate knowledge base
	knowledgeBase := []struct {
		id      string
		content string
		tags    []string
	}{
		{
			id:      "auth-best-practices",
			content: "Authentication best practices: Use bcrypt for password hashing, implement rate limiting, use JWT with short expiration times",
			tags:    []string{"security", "authentication"},
		},
		{
			id:      "api-design",
			content: "RESTful API design: Use proper HTTP methods, implement versioning, return consistent error responses",
			tags:    []string{"api", "design"},
		},
		{
			id:      "testing-strategy",
			content: "Testing strategy: Write unit tests first, use table-driven tests in Go, aim for 80% code coverage",
			tags:    []string{"testing", "quality"},
		},
	}

	// Index knowledge
	for _, kb := range knowledgeBase {
		doc := vector.Document{
			ID:      kb.id,
			Content: kb.content,
			Metadata: map[string]interface{}{
				"tags": kb.tags,
			},
		}
		err := vectorStore.Add(ctx, doc)
		require.NoError(t, err)
	}

	// Test RAG-enhanced responses
	queries := []struct {
		question       string
		expectedTopics []string
	}{
		{
			question:       "How should I implement user authentication?",
			expectedTopics: []string{"bcrypt", "JWT", "rate limiting"},
		},
		{
			question:       "What's the best way to design our API?",
			expectedTopics: []string{"HTTP methods", "versioning", "error responses"},
		},
		{
			question:       "How should we approach testing?",
			expectedTopics: []string{"unit tests", "table-driven", "coverage"},
		},
	}

	for _, query := range queries {
		t.Run(query.question, func(t *testing.T) {
			// Get RAG-enhanced response
			response, contexts, err := ragAgent.RespondWithContext(ctx, query.question)
			require.NoError(t, err)
			assert.NotEmpty(t, response)
			assert.NotEmpty(t, contexts)

			// Verify response includes expected topics
			for _, topic := range query.expectedTopics {
				assert.Contains(t, response, topic,
					"Response should include topic: %s", topic)
			}

			// Verify contexts were used
			assert.Greater(t, len(contexts), 0, "Should retrieve relevant contexts")
		})
	}

	// Test incremental learning
	t.Run("IncrementalLearning", func(t *testing.T) {
		// Add new knowledge
		newKnowledge := vector.Document{
			ID:      "new-security-update",
			Content: "New security update: OAuth 2.0 with PKCE is now recommended for mobile apps",
			Metadata: map[string]interface{}{
				"tags":      []string{"security", "oauth", "mobile"},
				"timestamp": time.Now(),
			},
		}
		err := vectorStore.Add(ctx, newKnowledge)
		require.NoError(t, err)

		// Query about mobile security
		response, contexts, err := ragAgent.RespondWithContext(ctx, 
			"What security measures should I use for mobile app authentication?")
		require.NoError(t, err)

		// Should include the new knowledge
		assert.Contains(t, response, "OAuth 2.0")
		assert.Contains(t, response, "PKCE")
		
		// Verify new knowledge was retrieved
		foundNewKnowledge := false
		for _, ctx := range contexts {
			if ctx.ID == "new-security-update" {
				foundNewKnowledge = true
				break
			}
		}
		assert.True(t, foundNewKnowledge, "Should retrieve newly added knowledge")
	})
}

// TestConcurrentMemoryOperations tests concurrent read/write operations
func TestConcurrentMemoryOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)

	// Create vector store
	vectorStore := testutil.NewMockVectorStore()

	// Number of concurrent operations
	numWriters := 10
	numReaders := 20
	docsPerWriter := 5

	// Synchronization
	var wg sync.WaitGroup
	errors := make(chan error, numWriters+numReaders)

	// Writer goroutines
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()

			for d := 0; d < docsPerWriter; d++ {
				doc := vector.Document{
					ID:      fmt.Sprintf("writer-%d-doc-%d", writerID, d),
					Content: fmt.Sprintf("Document from writer %d, number %d", writerID, d),
					Metadata: map[string]interface{}{
						"writer": writerID,
						"seq":    d,
					},
				}

				if err := vectorStore.Add(ctx, doc); err != nil {
					errors <- fmt.Errorf("writer %d: %w", writerID, err)
				}

				// Small delay to simulate real work
				time.Sleep(time.Millisecond)
			}
		}(w)
	}

	// Reader goroutines
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()

			// Perform multiple searches
			for s := 0; s < 3; s++ {
				query := fmt.Sprintf("Document from writer %d", readerID%numWriters)
				results, err := vectorStore.Search(ctx, query, 5)
				
				if err != nil {
					errors <- fmt.Errorf("reader %d: %w", readerID, err)
				} else if len(results) == 0 && s > 0 {
					// After first iteration, should find some results
					errors <- fmt.Errorf("reader %d: no results found", readerID)
				}

				// Small delay between searches
				time.Sleep(2 * time.Millisecond)
			}
		}(r)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errors)

	// Check for errors
	var errorCount int
	for err := range errors {
		errorCount++
		t.Logf("Concurrent operation error: %v", err)
	}

	assert.Equal(t, 0, errorCount, "Should have no errors during concurrent operations")

	// Verify all documents were written
	totalDocs := numWriters * docsPerWriter
	
	// Search for all documents
	allResults, err := vectorStore.Search(ctx, "Document from writer", totalDocs)
	require.NoError(t, err)
	
	// We might not get all documents in search due to relevance scoring,
	// but we should get a significant portion
	assert.Greater(t, len(allResults), totalDocs/2, 
		"Should retrieve at least half of the documents")
}

// Helper functions

func detectFileType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".go":
		return "code"
	case ".md":
		return "documentation"
	case ".yaml", ".yml":
		return "config"
	default:
		return "unknown"
	}
}

func formatContexts(contexts []vector.SearchResult) string {
	var builder strings.Builder
	for i, ctx := range contexts {
		builder.WriteString(fmt.Sprintf("\n[Context %d]: %s", i+1, ctx.Content))
	}
	return builder.String()
}

// Mock implementations

type mockEmbedder struct {
	dimension int
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// Create deterministic embeddings based on text
	embedding := make([]float32, m.dimension)
	
	// Simple hash-based embedding for testing
	hash := 0
	for _, char := range text {
		hash = (hash*31 + int(char)) % 1000000
	}
	
	// Fill embedding with values derived from hash
	for i := 0; i < m.dimension; i++ {
		embedding[i] = float32((hash+i)%100) / 100.0
	}
	
	return embedding, nil
}

func (m *mockEmbedder) GetDimension() int {
	return m.dimension
}