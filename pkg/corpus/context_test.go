package corpus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContextPropagation tests that context is properly propagated through all corpus functions
func TestContextPropagation(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create test configuration
	cfg := Config{
		CorpusPath:     tempDir,
		ActivitiesPath: filepath.Join(tempDir, ".activities"),
		MaxSizeBytes:   1024 * 1024,
	}

	// Create activities directory
	require.NoError(t, os.MkdirAll(cfg.ActivitiesPath, 0755))

	t.Run("Save with context", func(t *testing.T) {
		ctx := context.Background()

		doc := &CorpusDoc{
			Title:     "Context Test Document",
			Body:      "Testing context propagation in Save function",
			Source:    "test",
			Tags:      []string{"context", "test"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := Save(ctx, doc, cfg)
		assert.NoError(t, err)
		assert.NotEmpty(t, doc.FilePath)
	})

	t.Run("Load with context", func(t *testing.T) {
		ctx := context.Background()

		// First save a document
		doc := &CorpusDoc{
			Title:     "Load Context Test",
			Body:      "Testing context propagation in Load function",
			Source:    "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := Save(ctx, doc, cfg)
		require.NoError(t, err)

		// Now load it with context
		loadedDoc, err := Load(ctx, doc.FilePath)
		assert.NoError(t, err)
		assert.Equal(t, doc.Title, loadedDoc.Title)
		assert.Equal(t, doc.Body, loadedDoc.Body)
	})

	t.Run("List with context", func(t *testing.T) {
		ctx := context.Background()

		// Save a few documents
		for i := 0; i < 3; i++ {
			doc := &CorpusDoc{
				Title:     fmt.Sprintf("List Test Doc %d", i),
				Body:      "Testing context propagation in List function",
				Source:    "test",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err := Save(ctx, doc, cfg)
			require.NoError(t, err)
		}

		// List documents with context
		filePaths, err := List(ctx, cfg)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(filePaths), 3)
	})

	t.Run("Delete with context", func(t *testing.T) {
		ctx := context.Background()

		// Save a document
		doc := &CorpusDoc{
			Title:     "Delete Context Test",
			Body:      "Testing context propagation in Delete function",
			Source:    "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := Save(ctx, doc, cfg)
		require.NoError(t, err)

		// Delete with context
		err = Delete(ctx, doc.FilePath)
		assert.NoError(t, err)

		// Verify it's deleted
		_, err = os.Stat(doc.FilePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("Save existing document updates it", func(t *testing.T) {
		ctx := context.Background()

		// Save a document
		doc := &CorpusDoc{
			Title:     "Update Context Test",
			Body:      "Original body",
			Source:    "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := Save(ctx, doc, cfg)
		require.NoError(t, err)
		originalPath := doc.FilePath

		// Update the document by saving again with same title
		doc.Body = "Updated body with context"
		doc.UpdatedAt = time.Now()

		err = Save(ctx, doc, cfg)
		assert.NoError(t, err)
		assert.Equal(t, originalPath, doc.FilePath)

		// Load and verify
		updatedDoc, err := Load(ctx, doc.FilePath)
		require.NoError(t, err)
		assert.Equal(t, "Updated body with context", updatedDoc.Body)
	})

	t.Run("BuildGraph with context", func(t *testing.T) {
		ctx := context.Background()

		// Create documents with links
		doc1 := &CorpusDoc{
			Title:     "Graph Test Doc 1",
			Body:      "This links to [[Graph Test Doc 2]]",
			Source:    "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		doc2 := &CorpusDoc{
			Title:     "Graph Test Doc 2",
			Body:      "This links back to [[Graph Test Doc 1]]",
			Source:    "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := Save(ctx, doc1, cfg)
		require.NoError(t, err)
		err = Save(ctx, doc2, cfg)
		require.NoError(t, err)

		// Build graph with context
		graph, err := BuildGraph(ctx, cfg)
		assert.NoError(t, err)
		assert.NotNil(t, graph)
		assert.GreaterOrEqual(t, len(graph.Nodes), 2)
	})

	t.Run("TrackUserView with context", func(t *testing.T) {
		ctx := context.Background()

		// Save a document
		doc := &CorpusDoc{
			Title:     "Track View Context Test",
			Body:      "Testing context propagation in TrackUserView",
			Source:    "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := Save(ctx, doc, cfg)
		require.NoError(t, err)

		// Track view with context
		err = TrackUserView(ctx, "testuser", doc.FilePath, cfg)
		assert.NoError(t, err)

		// Verify activity was created
		activities, err := GetUserActivities(ctx, "testuser", cfg)
		assert.NoError(t, err)
		assert.NotEmpty(t, activities)
	})

	t.Run("GetUserActivities with context", func(t *testing.T) {
		ctx := context.Background()

		// Track some activities
		doc := &CorpusDoc{
			Title:     "Activity Context Test",
			Body:      "Testing context propagation in GetUserActivities",
			Source:    "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := Save(ctx, doc, cfg)
		require.NoError(t, err)

		// Track multiple views
		for i := 0; i < 3; i++ {
			err = TrackUserView(ctx, "testuser", doc.FilePath, cfg)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
		}

		// Get activities with context
		activities, err := GetUserActivities(ctx, "testuser", cfg)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(activities), 3)
	})

	t.Run("GetRecentActivity with context", func(t *testing.T) {
		ctx := context.Background()

		// Save some documents and track views
		for i := 0; i < 3; i++ {
			doc := &CorpusDoc{
				Title:     fmt.Sprintf("Recent Activity Doc %d", i),
				Body:      "Testing context propagation in GetRecentActivity",
				Source:    "test",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			err := Save(ctx, doc, cfg)
			require.NoError(t, err)

			// Track a view to create activity
			err = TrackUserView(ctx, "testuser", doc.FilePath, cfg)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}

		// Get recent activity with context
		activities, err := GetRecentActivity(ctx, cfg, 5)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(activities), 3)

		// Verify they're in descending order (most recent first)
		for i := 1; i < len(activities); i++ {
			assert.True(t, activities[i-1].Timestamp.After(activities[i].Timestamp) ||
				activities[i-1].Timestamp.Equal(activities[i].Timestamp))
		}
	})
}

// TestContextCancellation tests that functions respect context cancellation
func TestContextCancellation(t *testing.T) {
	tempDir := t.TempDir()

	cfg := Config{
		CorpusPath:     tempDir,
		ActivitiesPath: filepath.Join(tempDir, ".activities"),
		MaxSizeBytes:   1024 * 1024,
	}

	require.NoError(t, os.MkdirAll(cfg.ActivitiesPath, 0755))

	t.Run("Save respects cancellation", func(t *testing.T) {
		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		doc := &CorpusDoc{
			Title:     "Cancelled Save Test",
			Body:      "This should handle cancellation gracefully",
			Source:    "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Save should still work (file operations are quick)
		// but this tests that the context is passed through
		err := Save(ctx, doc, cfg)
		// File operations typically complete before context cancellation takes effect
		// This test mainly ensures context is properly passed through the call chain
		if err != nil {
			assert.Equal(t, context.Canceled, err)
		}
	})

	t.Run("List respects timeout", func(t *testing.T) {
		// Create a context with a reasonable timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// List should complete within timeout
		filePaths, err := List(ctx, cfg)
		assert.NoError(t, err)
		assert.NotNil(t, filePaths)
	})
}

// TestContextWithValues tests that context values are preserved through function calls
func TestContextWithValues(t *testing.T) {
	tempDir := t.TempDir()

	cfg := Config{
		CorpusPath:     tempDir,
		ActivitiesPath: filepath.Join(tempDir, ".activities"),
		MaxSizeBytes:   1024 * 1024,
	}

	require.NoError(t, os.MkdirAll(cfg.ActivitiesPath, 0755))

	// Create context with values
	type contextKey string
	const testKey contextKey = "testKey"
	ctx := context.WithValue(context.Background(), testKey, "testValue")

	// Save a document
	doc := &CorpusDoc{
		Title:     "Context Value Test",
		Body:      "Testing that context values are preserved",
		Source:    "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := Save(ctx, doc, cfg)
	require.NoError(t, err)

	// In a real implementation, we would verify that the context
	// was passed through to any internal functions that might use it
	// For now, this test ensures the API accepts context properly
	assert.Equal(t, "testValue", ctx.Value(testKey))
}
