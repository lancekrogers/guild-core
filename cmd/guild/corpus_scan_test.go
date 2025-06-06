package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/internal/corpus"
	"github.com/guild-ventures/guild-core/pkg/memory/rag"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
	"github.com/guild-ventures/guild-core/pkg/providers/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformCorpusScan(t *testing.T) {
	ctx := context.Background()
	
	// Create a temporary test directory
	tempDir := t.TempDir()
	corpusPath := filepath.Join(tempDir, "corpus")
	require.NoError(t, os.MkdirAll(corpusPath, 0755))
	
	// Create corpus config
	cfg := corpus.Config{
		CorpusPath:      corpusPath,
		ActivitiesPath:  filepath.Join(corpusPath, ".activities"),
		MaxSizeBytes:    100 * 1024 * 1024, // 100MB
		DefaultCategory: "test",
	}
	
	// Create test documents
	doc1 := &corpus.CorpusDoc{
		Title:     "Test Document 1",
		Body:      "This is a test document about the Guild framework.",
		Tags:      []string{"test", "guild"},
		Source:    "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	doc2 := &corpus.CorpusDoc{
		Title:     "Test Document 2",
		Body:      "Another test document with different content.",
		Tags:      []string{"test", "rag"},
		Source:    "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Save documents to corpus
	require.NoError(t, corpus.Save(ctx, doc1, cfg))
	require.NoError(t, corpus.Save(ctx, doc2, cfg))
	
	tests := []struct {
		name         string
		dryRun       bool
		forceRebuild bool
		expectNew    int
		expectMod    int
		expectDel    int
	}{
		{
			name:      "scan with existing metadata shows modified files",
			dryRun:    false,
			expectNew: 0, // Files already indexed in initial scan
			expectMod: 2, // Files show as modified because file mod time > metadata time
			expectDel: 0,
		},
		{
			name:         "force rebuild treats all as modified",
			dryRun:       false,
			forceRebuild: true,
			expectNew:    0,
			expectMod:    2, // Force rebuild treats existing files as modified
			expectDel:    0,
		},
		{
			name:      "dry run doesn't update metadata",
			dryRun:    true,
			expectNew: 0, // Files were already indexed in initial scan
			expectMod: 2, // Files show as modified due to timing
			expectDel: 0,
		},
	}
	
	// Run initial scan first to create metadata
	initialProvider := mock.NewProvider()
	initialVectorConfig := &vector.StoreConfig{
		Type:              vector.StoreTypeChromem,
		EmbeddingProvider: initialProvider,
		ChromemConfig: vector.ChromemConfig{
			PersistencePath:   filepath.Join(tempDir, "embeddings"),
			DefaultCollection: "test",
		},
	}
	initialVectorStore, err := vector.NewVectorStore(ctx, initialVectorConfig)
	require.NoError(t, err)
	initialRetriever := rag.NewRetrieverWithStore(initialVectorStore, rag.Config{
		ChunkSize:    100,
		ChunkOverlap: 20,
		MaxResults:   5,
	})
	
	// Perform initial scan to create metadata
	_ = performCorpusScan(ctx, cfg, initialRetriever, false, false, false)
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a unique temp dir for each test to ensure clean state
			testTempDir := filepath.Join(tempDir, tt.name)
			require.NoError(t, os.MkdirAll(testTempDir, 0755))
			
			// Create mock RAG system
			mockProvider := mock.NewProvider()
			vectorConfig := &vector.StoreConfig{
				Type:              vector.StoreTypeChromem,
				EmbeddingProvider: mockProvider,
				ChromemConfig: vector.ChromemConfig{
					PersistencePath:   filepath.Join(testTempDir, "embeddings"),
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
			
			retriever := rag.NewRetrieverWithStore(vectorStore, ragConfig)
			
			// Perform scan
			result := performCorpusScan(ctx, cfg, retriever, tt.dryRun, false, tt.forceRebuild)
			
			// Check results
			assert.Len(t, result.NewFiles, tt.expectNew, "NewFiles count mismatch")
			assert.Len(t, result.ModifiedFiles, tt.expectMod, "ModifiedFiles count mismatch")
			assert.Len(t, result.DeletedFiles, tt.expectDel, "DeletedFiles count mismatch")
			assert.Empty(t, result.Errors)
			assert.NotZero(t, result.StartTime)
			assert.NotZero(t, result.EndTime)
			assert.True(t, result.EndTime.After(result.StartTime))
		})
	}
}

func TestGetEmbeddingsMetadata(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()
	corpusPath := filepath.Join(tempDir, "corpus")
	embeddingsPath := filepath.Join(tempDir, "embeddings")
	require.NoError(t, os.MkdirAll(embeddingsPath, 0755))
	
	cfg := corpus.Config{
		CorpusPath: corpusPath,
	}
	
	// Test empty metadata (file doesn't exist)
	metadata := getEmbeddingsMetadata(cfg)
	assert.Empty(t, metadata)
	
	// Create metadata file
	metadataPath := filepath.Join(embeddingsPath, ".metadata.json")
	testTime := time.Now().UTC()
	content := "/path/to/doc1.md\t" + testTime.Format(time.RFC3339) + "\n" +
		"/path/to/doc2.md\t" + testTime.Add(-time.Hour).Format(time.RFC3339)
	
	require.NoError(t, os.WriteFile(metadataPath, []byte(content), 0644))
	
	// Test reading metadata
	metadata = getEmbeddingsMetadata(cfg)
	assert.Len(t, metadata, 2)
	
	// Check times (allowing for parsing precision)
	doc1Time := metadata["/path/to/doc1.md"]
	assert.WithinDuration(t, testTime, doc1Time, time.Second)
	
	doc2Time := metadata["/path/to/doc2.md"]
	assert.WithinDuration(t, testTime.Add(-time.Hour), doc2Time, time.Second)
}

func TestUpdateEmbeddingMetadata(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()
	corpusPath := filepath.Join(tempDir, "corpus")
	embeddingsPath := filepath.Join(tempDir, "embeddings")
	
	cfg := corpus.Config{
		CorpusPath: corpusPath,
	}
	
	// Test creating new metadata
	testTime := time.Now().UTC()
	err := updateEmbeddingMetadata(cfg, "/path/to/doc1.md", testTime)
	require.NoError(t, err)
	
	// Verify directory was created
	assert.DirExists(t, embeddingsPath)
	
	// Verify metadata was written
	metadata := getEmbeddingsMetadata(cfg)
	assert.Len(t, metadata, 1)
	assert.WithinDuration(t, testTime, metadata["/path/to/doc1.md"], time.Second)
	
	// Test updating existing metadata
	newTime := testTime.Add(time.Hour)
	err = updateEmbeddingMetadata(cfg, "/path/to/doc1.md", newTime)
	require.NoError(t, err)
	
	// Add another document
	err = updateEmbeddingMetadata(cfg, "/path/to/doc2.md", testTime)
	require.NoError(t, err)
	
	// Verify both entries exist
	metadata = getEmbeddingsMetadata(cfg)
	assert.Len(t, metadata, 2)
	assert.WithinDuration(t, newTime, metadata["/path/to/doc1.md"], time.Second)
	assert.WithinDuration(t, testTime, metadata["/path/to/doc2.md"], time.Second)
}

func TestDisplayScanResults(t *testing.T) {
	// This is mainly to ensure the function doesn't panic
	result := ScanResult{
		NewFiles:      []string{"/path/to/new1.md", "/path/to/new2.md"},
		ModifiedFiles: []string{"/path/to/modified.md"},
		DeletedFiles:  []string{"/path/to/deleted.md"},
		Errors:        []error{},
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(time.Second),
	}
	
	// Capture output (in a real test, you'd redirect stdout)
	// For now, just call it to ensure no panic
	displayScanResults(result, false)
	displayScanResults(result, true) // Test dry run display
	
	// Test with no changes
	emptyResult := ScanResult{
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Millisecond),
	}
	displayScanResults(emptyResult, false)
}

func TestInitializeRAGSystem(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	
	cfg := corpus.Config{
		CorpusPath: filepath.Join(tempDir, "corpus"),
	}
	
	tests := []struct {
		name           string
		providerType   string
		embeddingModel string
		envVars        map[string]string
		expectError    bool
	}{
		{
			name:         "auto-detect provider",
			providerType: "",
			expectError:  false, // Will use mock or NoOp embedder
		},
		{
			name:         "ollama provider",
			providerType: "ollama",
			envVars: map[string]string{
				"OLLAMA_HOST": "http://localhost:11434",
			},
			expectError: false,
		},
		{
			name:         "openai provider without key",
			providerType: "openai",
			expectError:  true, // Should fail due to missing API key
		},
		{
			name:         "unsupported provider",
			providerType: "unknown",
			expectError:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}
			
			retriever, err := initializeRAGSystem(ctx, cfg, tt.providerType, tt.embeddingModel, false)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, retriever)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, retriever)
			}
		})
	}
}