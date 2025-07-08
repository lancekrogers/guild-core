// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/storage"
)

// TestReasoningSystemSQLiteIntegration tests the complete reasoning system with SQLite
func TestReasoningSystemSQLiteIntegration(t *testing.T) {
	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "test")

	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_reasoning.db")

	// Create reasoning system with SQLite
	config := DefaultReasoningSystemConfig()
	config.DatabasePath = dbPath
	config.StorageConfig.RetentionDays = 7 // Short retention for testing

	system, err := NewReasoningSystem(ctx, config)
	require.NoError(t, err)
	defer system.Close(ctx)

	// Create test agents to satisfy foreign key constraints
	createTestAgents(t, ctx, system.database)

	// Test 1: Store and retrieve reasoning chains
	t.Run("StoreAndRetrieve", func(t *testing.T) {
		chain := &ReasoningChain{
			ID:         fmt.Sprintf("test_chain_%d", time.Now().UnixNano()),
			AgentID:    "test_agent",
			SessionID:  "test_session",
			Content:    "The capital of France is Paris.",
			Reasoning:  "This is a straightforward factual question about geography.",
			Confidence: 0.95,
			TaskType:   "factual_query",
			Success:    true,
			TokensUsed: 50,
			Duration:   100 * time.Millisecond,
			CreatedAt:  time.Now(),
			Metadata: map[string]interface{}{
				"test":     true,
				"category": "geography",
			},
		}

		// Store chain
		err := system.Storage.Store(ctx, chain)
		require.NoError(t, err)

		// Retrieve chain
		retrieved, err := system.Storage.Get(ctx, chain.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		// Verify data integrity
		assert.Equal(t, chain.ID, retrieved.ID)
		assert.Equal(t, chain.AgentID, retrieved.AgentID)
		assert.Equal(t, chain.Content, retrieved.Content)
		assert.Equal(t, chain.Reasoning, retrieved.Reasoning)
		assert.Equal(t, chain.Confidence, retrieved.Confidence)
		assert.Equal(t, chain.TaskType, retrieved.TaskType)
		assert.Equal(t, chain.Success, retrieved.Success)
		assert.Equal(t, true, retrieved.Metadata["test"])
		assert.Equal(t, "geography", retrieved.Metadata["category"])
	})

	// Test 2: Query reasoning chains
	t.Run("QueryChains", func(t *testing.T) {
		// Store multiple chains
		for i := 0; i < 10; i++ {
			chain := &ReasoningChain{
				ID:         fmt.Sprintf("query_test_%d_%d", i, time.Now().UnixNano()),
				AgentID:    "test_agent",
				Content:    fmt.Sprintf("Test content %d", i),
				Reasoning:  fmt.Sprintf("Test reasoning %d", i),
				Confidence: float64(i) / 10.0,
				TaskType:   fmt.Sprintf("type_%d", i%3),
				Success:    i%2 == 0,
				CreatedAt:  time.Now().Add(-time.Duration(i) * time.Hour),
			}
			err := system.Storage.Store(ctx, chain)
			require.NoError(t, err)
		}

		// Query by agent
		query := &ReasoningQuery{
			AgentID:   "test_agent",
			Limit:     5,
			OrderBy:   "confidence",
			Ascending: false,
		}

		results, err := system.Storage.Query(ctx, query)
		require.NoError(t, err)
		assert.Len(t, results, 5)

		// Verify ordering
		for i := 0; i < len(results)-1; i++ {
			assert.GreaterOrEqual(t, results[i].Confidence, results[i+1].Confidence)
		}

		// Query by confidence range
		query2 := &ReasoningQuery{
			AgentID:       "test_agent",
			MinConfidence: 0.5,
			MaxConfidence: 0.8,
		}

		results2, err := system.Storage.Query(ctx, query2)
		require.NoError(t, err)
		for _, chain := range results2 {
			assert.GreaterOrEqual(t, chain.Confidence, 0.5)
			assert.LessOrEqual(t, chain.Confidence, 0.8)
		}
	})

	// Test 3: Statistics aggregation
	t.Run("Statistics", func(t *testing.T) {
		// Get stats for test agent
		stats, err := system.Storage.GetStats(ctx, "test_agent", time.Time{}, time.Now())
		require.NoError(t, err)
		require.NotNil(t, stats)

		assert.Greater(t, stats.TotalChains, int64(0))
		assert.Greater(t, stats.AvgConfidence, 0.0)
		assert.GreaterOrEqual(t, stats.SuccessRate, 0.0)
		assert.LessOrEqual(t, stats.SuccessRate, 1.0)

		// Check distributions
		assert.NotEmpty(t, stats.ConfidenceDistrib)
		assert.NotEmpty(t, stats.TaskTypeDistrib)
	})

	// Test 4: Pattern identification
	t.Run("PatternIdentification", func(t *testing.T) {
		// Store chains with patterns
		for i := 0; i < 5; i++ {
			chain := &ReasoningChain{
				ID:         fmt.Sprintf("pattern_test_%d", time.Now().UnixNano()+int64(i)),
				AgentID:    "pattern_agent",
				Content:    "API implementation guide",
				Reasoning:  "First, I need to understand the requirements. Then design the endpoints. Finally, implement with proper error handling.",
				Confidence: 0.8,
				TaskType:   "code_generation",
				Success:    true,
				CreatedAt:  time.Now(),
			}
			err := system.Storage.Store(ctx, chain)
			require.NoError(t, err)
		}

		// Query recent chains
		query := &ReasoningQuery{
			AgentID: "pattern_agent",
			Limit:   100,
		}

		chains, err := system.Storage.Query(ctx, query)
		require.NoError(t, err)

		// Identify patterns
		patterns, err := system.Analyzer.IdentifyPatterns(ctx, chains)
		require.NoError(t, err)
		assert.NotEmpty(t, patterns)

		// Store patterns
		for _, pattern := range patterns {
			err := system.Storage.UpdatePattern(ctx, pattern)
			require.NoError(t, err)
		}

		// Retrieve patterns
		storedPatterns, err := system.Storage.GetPatterns(ctx, "code_generation", 10)
		require.NoError(t, err)
		assert.NotEmpty(t, storedPatterns)
	})

	// Test 5: Insights generation
	t.Run("InsightsGeneration", func(t *testing.T) {
		insights, err := system.GetInsights(ctx, "test_agent")
		require.NoError(t, err)
		assert.NotEmpty(t, insights)

		// Verify insights are meaningful
		for _, insight := range insights {
			assert.NotEmpty(t, insight)
			assert.Greater(t, len(insight), 10) // Should be substantial text
		}
	})

	// Test 6: Retention cleanup
	t.Run("RetentionCleanup", func(t *testing.T) {
		// Store old chains
		oldChain := &ReasoningChain{
			ID:         "old_chain_test",
			AgentID:    "test_agent",
			Content:    "Old content",
			Reasoning:  "Old reasoning",
			Confidence: 0.5,
			CreatedAt:  time.Now().Add(-30 * 24 * time.Hour), // 30 days old
		}

		// Use raw SQL to insert with old timestamp
		db := system.database.DB()
		_, err := db.ExecContext(ctx, `
			INSERT INTO reasoning_chains (
				id, agent_id, content, reasoning, confidence, 
				success, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`, oldChain.ID, oldChain.AgentID, oldChain.Content,
			oldChain.Reasoning, oldChain.Confidence, true, oldChain.CreatedAt)
		require.NoError(t, err)

		// Verify it exists
		_, err = system.Storage.Get(ctx, oldChain.ID)
		require.NoError(t, err)

		// Run cleanup
		cutoff := time.Now().Add(-7 * 24 * time.Hour)
		deleted, err := system.Storage.Delete(ctx, cutoff)
		require.NoError(t, err)
		assert.Greater(t, deleted, int64(0))

		// Verify it's gone
		_, err = system.Storage.Get(ctx, oldChain.ID)
		assert.Error(t, err)
	})

	// Test 7: Concurrent operations
	t.Run("ConcurrentOperations", func(t *testing.T) {
		const numGoroutines = 10
		const numOperations = 5

		errChan := make(chan error, numGoroutines*numOperations)

		for i := 0; i < numGoroutines; i++ {
			go func(workerID int) {
				for j := 0; j < numOperations; j++ {
					chain := &ReasoningChain{
						ID:         fmt.Sprintf("concurrent_%d_%d_%d", workerID, j, time.Now().UnixNano()),
						AgentID:    fmt.Sprintf("worker_%d", workerID),
						Content:    fmt.Sprintf("Content from worker %d operation %d", workerID, j),
						Reasoning:  fmt.Sprintf("Reasoning from worker %d", workerID),
						Confidence: 0.7,
						Success:    true,
						CreatedAt:  time.Now(),
					}

					if err := system.Storage.Store(ctx, chain); err != nil {
						errChan <- err
						return
					}

					// Try to retrieve it
					if _, err := system.Storage.Get(ctx, chain.ID); err != nil {
						errChan <- err
						return
					}
				}
			}(i)
		}

		// Wait and check for errors
		time.Sleep(2 * time.Second)
		close(errChan)

		for err := range errChan {
			assert.NoError(t, err)
		}

		// Verify all chains were stored
		for i := 0; i < numGoroutines; i++ {
			query := &ReasoningQuery{
				AgentID: fmt.Sprintf("worker_%d", i),
			}
			results, err := system.Storage.Query(ctx, query)
			require.NoError(t, err)
			assert.Len(t, results, numOperations)
		}
	})

	// Verify database file exists and has content
	info, err := os.Stat(dbPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

// TestReasoningSystemProduction tests production-like scenarios
func TestReasoningSystemProduction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production test in short mode")
	}

	ctx := context.Background()
	ctx = observability.WithComponent(ctx, "test_production")

	// Create system with production config
	config := DefaultReasoningSystemConfig()
	config.DatabasePath = filepath.Join(t.TempDir(), "production_test.db")

	system, err := NewReasoningSystem(ctx, config)
	require.NoError(t, err)
	defer system.Close(ctx)

	// Create test agents to satisfy foreign key constraints
	createTestAgents(t, ctx, system.database)

	// Start maintenance tasks
	maintenanceCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	system.StartMaintenance(maintenanceCtx)

	// Simulate production load
	agents := []string{"api_agent", "data_agent", "ui_agent", "test_agent"}
	taskTypes := []string{"api_design", "data_processing", "ui_update", "testing"}

	// Store many chains over time
	for hour := 0; hour < 24; hour++ {
		for _, agentID := range agents {
			for i := 0; i < 10; i++ {
				chain := &ReasoningChain{
					ID:         fmt.Sprintf("prod_%s_%d_%d", agentID, hour, i),
					AgentID:    agentID,
					Content:    fmt.Sprintf("Task executed by %s at hour %d", agentID, hour),
					Reasoning:  "Analyzing task requirements and executing with appropriate strategy.",
					Confidence: 0.5 + float64(i%5)/10,
					TaskType:   taskTypes[hour%len(taskTypes)],
					Success:    i%3 != 0, // ~66% success rate
					TokensUsed: 100 + i*50,
					Duration:   time.Duration(100+i*20) * time.Millisecond,
					CreatedAt:  time.Now().Add(-time.Duration(24-hour) * time.Hour),
				}

				err := system.Storage.Store(ctx, chain)
				require.NoError(t, err)
			}
		}
	}

	// Test complex queries
	t.Run("ComplexQueries", func(t *testing.T) {
		// High confidence successful operations
		query := &ReasoningQuery{
			MinConfidence: 0.8,
			StartTime:     time.Now().Add(-12 * time.Hour),
			OrderBy:       "created_at",
			Limit:         50,
		}

		results, err := system.Storage.Query(ctx, query)
		require.NoError(t, err)
		assert.NotEmpty(t, results)

		for _, chain := range results {
			assert.GreaterOrEqual(t, chain.Confidence, 0.8)
			assert.GreaterOrEqual(t, chain.CreatedAt, query.StartTime)
		}
	})

	// Test analytics at scale
	t.Run("ScaleAnalytics", func(t *testing.T) {
		for _, agentID := range agents {
			stats, err := system.Storage.GetStats(ctx, agentID, time.Time{}, time.Now())
			require.NoError(t, err)

			assert.Greater(t, stats.TotalChains, int64(200)) // 24 hours * 10 chains
			assert.Greater(t, stats.AvgConfidence, 0.5)
			assert.Less(t, stats.AvgConfidence, 0.9)

			// Generate insights
			insights, err := system.GetInsights(ctx, agentID)
			require.NoError(t, err)
			assert.NotEmpty(t, insights)
		}
	})

	// Test pattern learning
	t.Run("PatternLearning", func(t *testing.T) {
		// Let aggregation run
		time.Sleep(2 * time.Second)

		// Check patterns were identified
		for _, taskType := range taskTypes {
			patterns, err := system.Storage.GetPatterns(ctx, taskType, 5)
			require.NoError(t, err)

			if len(patterns) > 0 {
				// Verify patterns have meaningful data
				for _, pattern := range patterns {
					assert.NotEmpty(t, pattern.Pattern)
					assert.Greater(t, pattern.Occurrences, 0)
				}
			}
		}
	})

	// Test performance under load
	t.Run("PerformanceUnderLoad", func(t *testing.T) {
		start := time.Now()

		// Perform many reads
		for i := 0; i < 100; i++ {
			query := &ReasoningQuery{
				AgentID: agents[i%len(agents)],
				Limit:   10,
			}
			_, err := system.Storage.Query(ctx, query)
			require.NoError(t, err)
		}

		elapsed := time.Since(start)
		avgQueryTime := elapsed / 100

		// Should be fast even with large dataset
		assert.Less(t, avgQueryTime, 50*time.Millisecond)
	})
}

// BenchmarkSQLiteReasoningStorage benchmarks SQLite storage operations
func BenchmarkSQLiteReasoningStorage(b *testing.B) {
	ctx := context.Background()

	// Create system
	config := DefaultReasoningSystemConfig()
	config.DatabasePath = filepath.Join(b.TempDir(), "benchmark.db")

	system, err := NewReasoningSystem(ctx, config)
	require.NoError(b, err)
	defer system.Close(ctx)

	// Create test agents to satisfy foreign key constraints
	db := system.database.DB()
	query := `INSERT OR IGNORE INTO agents (id, name, type) VALUES (?, ?, ?)`
	_, err = db.ExecContext(ctx, query, "bench_agent", "Benchmark Agent", "worker")
	require.NoError(b, err)

	b.Run("Store", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			chain := &ReasoningChain{
				ID:         fmt.Sprintf("bench_%d_%d", time.Now().UnixNano(), i),
				AgentID:    "bench_agent",
				Content:    "Benchmark content",
				Reasoning:  "Benchmark reasoning",
				Confidence: 0.8,
				Success:    true,
				CreatedAt:  time.Now(),
			}
			err := system.Storage.Store(ctx, chain)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Pre-populate for query benchmarks
	for i := 0; i < 1000; i++ {
		chain := &ReasoningChain{
			ID:         fmt.Sprintf("prepop_%d", i),
			AgentID:    "bench_agent",
			Content:    fmt.Sprintf("Content %d", i),
			Reasoning:  "Reasoning",
			Confidence: float64(i%100) / 100,
			TaskType:   fmt.Sprintf("type_%d", i%5),
			Success:    true,
			CreatedAt:  time.Now(),
		}
		system.Storage.Store(ctx, chain)
	}

	b.Run("Query", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := &ReasoningQuery{
				AgentID: "bench_agent",
				Limit:   10,
			}
			_, err := system.Storage.Query(ctx, query)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Stats", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := system.Storage.GetStats(ctx, "bench_agent", time.Time{}, time.Now())
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// createTestAgents creates test agents in the database to satisfy foreign key constraints
func createTestAgents(t *testing.T, ctx context.Context, db *storage.Database) {
	// Create test session first
	sessionQuery := `INSERT OR IGNORE INTO chat_sessions (id, name) VALUES (?, ?)`
	_, err := db.DB().ExecContext(ctx, sessionQuery, "test_session", "Test Session")
	require.NoError(t, err, "Failed to create test session")

	agents := []struct {
		id   string
		name string
		typ  string
	}{
		{"test_agent", "Test Agent", "worker"},
		{"pattern_agent", "Pattern Agent", "worker"},
		{"api_agent", "API Agent", "worker"},
		{"data_agent", "Data Agent", "worker"},
		{"ui_agent", "UI Agent", "worker"},
		{"bench_agent", "Benchmark Agent", "worker"},
	}

	// Add worker agents for concurrent test
	for i := 0; i < 10; i++ {
		agents = append(agents, struct {
			id   string
			name string
			typ  string
		}{
			fmt.Sprintf("worker_%d", i),
			fmt.Sprintf("Worker %d", i),
			"worker",
		})
	}

	query := `INSERT OR IGNORE INTO agents (id, name, type) VALUES (?, ?, ?)`

	for _, agent := range agents {
		_, err := db.DB().ExecContext(ctx, query, agent.id, agent.name, agent.typ)
		require.NoError(t, err, "Failed to create test agent %s", agent.id)
	}
}
