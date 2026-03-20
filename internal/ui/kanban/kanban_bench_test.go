// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	viewutil "github.com/lancekrogers/guild-core/internal/ui/view"
	"github.com/lancekrogers/guild-core/pkg/kanban"
)

// setupKanbanWithCards creates a kanban model with the specified number of cards
func setupKanbanWithCards(cardCount int) *Model {
	ctx := context.Background()

	// Create a minimal model for testing without a real manager
	model := &Model{
		ctx:        ctx,
		boardID:    "test-board",
		taskCache:  make(map[string][]*kanban.Task),
		lastRender: time.Now(),
		viewportState: ViewportState{
			Width:         120,
			Height:        30,
			VisibleRows:   10,
			FocusedColumn: 0,
		},
		// Initialize performance components
		profiler:      NewKanbanProfiler(60),
		viewport:      NewViewport(120, 30),
		renderer:      NewOptimizedCardRenderer(),
		cardCache:     NewCompactCardCache(1000, 10*time.Minute),
		virtualWindow: NewVirtualWindow(100),
	}
	// Initialize columns
	model.columns = [5]Column{
		{Status: kanban.StatusTodo, Title: "TODO"},
		{Status: kanban.StatusInProgress, Title: "IN PROGRESS"},
		{Status: kanban.StatusBlocked, Title: "BLOCKED"},
		{Status: kanban.StatusReadyForReview, Title: "READY FOR REVIEW"},
		{Status: kanban.StatusDone, Title: "DONE"},
	}

	model.calculateVisibleRows()

	// Generate and distribute cards across columns
	cards := generateMockCards(cardCount)
	tasks := make(map[string][]*kanban.Task)
	cardsPerColumn := cardCount / 5
	statuses := []kanban.TaskStatus{
		kanban.StatusTodo,
		kanban.StatusInProgress,
		kanban.StatusBlocked,
		kanban.StatusReadyForReview,
		kanban.StatusDone,
	}

	for i, status := range statuses {
		start := i * cardsPerColumn
		end := start + cardsPerColumn
		if i == len(statuses)-1 {
			end = cardCount // Put remaining cards in the last column
		}

		if start < len(cards) {
			if end > len(cards) {
				end = len(cards)
			}
			tasks[string(status)] = cards[start:end]
		}
	}

	model.taskCache = tasks
	model.updateColumns()

	return model
}

// generateMockCards creates mock cards for testing
func generateMockCards(count int) []*kanban.Task {
	cards := make([]*kanban.Task, count)
	statuses := []kanban.TaskStatus{
		kanban.StatusTodo,
		kanban.StatusInProgress,
		kanban.StatusBlocked,
		kanban.StatusReadyForReview,
		kanban.StatusDone,
	}
	priorities := []kanban.TaskPriority{
		kanban.PriorityHigh,
		kanban.PriorityMedium,
		kanban.PriorityLow,
	}

	for i := 0; i < count; i++ {
		cards[i] = &kanban.Task{
			ID:          fmt.Sprintf("TASK-%04d", i+1),
			Title:       fmt.Sprintf("Test Task %d - Complex Description with Multiple Words", i+1),
			Description: fmt.Sprintf("This is a detailed description for task %d with enough content to test rendering performance under various scenarios", i+1),
			Status:      statuses[i%len(statuses)],
			Priority:    priorities[i%len(priorities)],
			AssignedTo:  fmt.Sprintf("user%d", (i%10)+1),
			Progress:    (i * 7) % 101, // Pseudo-random progress
			CreatedAt:   time.Now().Add(-time.Duration(i) * time.Hour),
			UpdatedAt:   time.Now().Add(-time.Duration(i) * time.Minute),
		}
	}

	return cards
}

// Removed unused mock code - benchmarks use setupKanbanWithCards which doesn't need a manager

// BenchmarkKanbanRender benchmarks the main rendering performance
func BenchmarkKanbanRender(b *testing.B) {
	scenarios := []struct {
		name  string
		cards int
	}{
		{"Small", 20},
		{"Medium", 100},
		{"Large", 200},
		{"XLarge", 500},
		{"XXLarge", 1000},
		{"Extreme", 2000},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			model := setupKanbanWithCards(s.cards)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = model.View()
			}

			// Calculate and report metrics
			renderTime := time.Duration(b.Elapsed().Nanoseconds() / int64(b.N))
			b.ReportMetric(float64(renderTime.Nanoseconds())/1e6, "ms/render")

			// Check if we're hitting our performance targets
			targetMs := 16.67 // 60 FPS
			actualMs := float64(renderTime.Nanoseconds()) / 1e6

			if actualMs > targetMs {
				b.Logf("WARNING: Render time %.2fms exceeds 60 FPS target (%.2fms) for %d cards",
					actualMs, targetMs, s.cards)
			}
		})
	}
}

// BenchmarkViewportCulling benchmarks viewport culling performance
func BenchmarkViewportCulling(b *testing.B) {
	scenarios := []struct {
		name    string
		cards   int
		culling bool
	}{
		{"NoCulling_100", 100, false},
		{"WithCulling_100", 100, true},
		{"NoCulling_500", 500, false},
		{"WithCulling_500", 500, true},
		{"NoCulling_1000", 1000, false},
		{"WithCulling_1000", 1000, true},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			model := setupKanbanWithCards(s.cards)

			// Create viewport
			viewport := NewViewport(120, 30)
			viewport.CullingEnabled = s.culling

			// Convert model columns to viewport format
			columns := make([]*Column, len(model.columns))
			for i := range model.columns {
				columns[i] = &model.columns[i]
			}

			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := viewport.VisibleCards(ctx, columns)
				if err != nil {
					b.Fatal(err)
				}
			}

			// Report culling effectiveness
			visibleCards, _ := viewport.VisibleCards(ctx, columns)
			cullingRatio := float64(len(visibleCards)) / float64(s.cards) * 100
			b.ReportMetric(cullingRatio, "visible_ratio_%")
		})
	}
}

// BenchmarkRenderCache benchmarks render caching performance
func BenchmarkRenderCache(b *testing.B) {
	scenarios := []struct {
		name      string
		cacheSize int
		cardCount int
	}{
		{"NoCache", 0, 100},
		{"SmallCache", 50, 100},
		{"LargeCache", 200, 100},
		{"HugeCache", 1000, 500},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			var renderer *OptimizedCardRenderer
			if s.cacheSize > 0 {
				renderer = NewOptimizedCardRenderer()
				renderer.cache.maxSize = s.cacheSize
			}

			cards := generateMockCards(s.cardCount)
			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				card := cards[i%len(cards)]

				if renderer != nil {
					_, err := renderer.RenderCard(ctx, card, 20, false)
					if err != nil {
						b.Fatal(err)
					}
				} else {
					// Direct rendering without cache - simulate simple rendering
					_ = fmt.Sprintf("%-20s", card.Title)
				}
			}

			// Report cache hit rate if applicable
			if renderer != nil {
				metrics := renderer.GetMetrics()
				if metrics.CacheHits+metrics.CacheMisses > 0 {
					hitRate := float64(metrics.CacheHits) / float64(metrics.CacheHits+metrics.CacheMisses) * 100
					b.ReportMetric(hitRate, "cache_hit_rate_%")
				}
			}
		})
	}
}

// BenchmarkMemoryDataStructures benchmarks memory-efficient data structures
func BenchmarkMemoryDataStructures(b *testing.B) {
	b.Run("VirtualWindow", func(b *testing.B) {
		cards := generateMockCards(1000)
		window := NewVirtualWindow(50)
		ctx := context.Background()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			center := i % len(cards)
			err := window.LoadWindow(ctx, cards, center)
			if err != nil {
				b.Fatal(err)
			}

			_, err = window.GetCards(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}

		memUsage := window.GetMemoryUsage()
		b.ReportMetric(float64(memUsage)/1024, "memory_kb")
	})

	b.Run("CompactCardCache", func(b *testing.B) {
		cache := NewCompactCardCache(500, 5*time.Minute)
		cards := generateMockCards(1000)
		ctx := context.Background()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			card := cards[i%len(cards)]

			// Add to cache
			err := cache.AddCard(ctx, card)
			if err != nil {
				b.Fatal(err)
			}

			// Retrieve from cache
			_, err = cache.GetCard(ctx, card.ID)
			if err != nil {
				b.Fatal(err)
			}
		}

		stats := cache.GetStats()
		b.ReportMetric(float64(stats["total_cards"].(int)), "cached_cards")
	})

	b.Run("ObjectPools", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Test card pool
			card := GetCard()
			card.ID = strconv.Itoa(i)
			card.Title = "Test Card"
			PutCard(card)

			// Test render info pool
			info := GetRenderInfo()
			info.X = i
			info.Y = i * 2
			PutRenderInfo(info)

			// Test slice pool
			slice := GetTaskSlice()
			slice = append(slice, card)
			PutTaskSlice(slice)
		}
	})
}

// BenchmarkScrolling benchmarks scrolling performance with different card counts
func BenchmarkScrolling(b *testing.B) {
	scenarios := []struct {
		name  string
		cards int
	}{
		{"Small", 50},
		{"Medium", 200},
		{"Large", 500},
		{"XLarge", 1000},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			model := setupKanbanWithCards(s.cards)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Simulate scrolling down
				if i%10 == 0 {
					model.scrollColumn(1)
				}

				// Render after scroll
				_ = model.View()
			}
		})
	}
}

// BenchmarkEventHandling benchmarks real-time event processing
func BenchmarkEventHandling(b *testing.B) {
	model := setupKanbanWithCards(100)

	events := []taskEventMsg{
		{eventType: "task.created", taskID: "NEW-TASK", boardID: "test-board"},
		{eventType: "task.moved", taskID: "TASK-001", boardID: "test-board"},
		{eventType: "task.updated", taskID: "TASK-002", boardID: "test-board"},
		{eventType: "task.completed", taskID: "TASK-003", boardID: "test-board"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		event := events[i%len(events)]
		model.handleTaskEvent(event)

		// Simulate periodic render after events
		if i%10 == 0 {
			_ = model.View()
		}
	}
}

// BenchmarkFullWorkflow benchmarks a complete user workflow
func BenchmarkFullWorkflow(b *testing.B) {
	scenarios := []struct {
		name  string
		cards int
	}{
		{"Workflow_100", 100},
		{"Workflow_500", 500},
		{"Workflow_1000", 1000},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			model := setupKanbanWithCards(s.cards)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Simulate user workflow
				cycle := i % 20

				switch cycle {
				case 0, 5, 10, 15:
					// Render view
					_ = model.View()
				case 1, 6, 11, 16:
					// Scroll down
					model.scrollColumn(1)
				case 2, 7, 12, 17:
					// Change columns
					if model.viewportState.FocusedColumn < 4 {
						model.viewportState.FocusedColumn++
					} else {
						model.viewportState.FocusedColumn = 0
					}
				case 3, 8, 13, 18:
					// Search filter
					model.viewportState.SearchFilter = fmt.Sprintf("Task %d", i%10)
					model.updateColumns()
				case 4, 9, 14, 19:
					// Clear search
					model.viewportState.SearchFilter = ""
					model.updateColumns()
				}
			}

			// Report final render time
			start := time.Now()
			_ = model.View()
			renderTime := time.Since(start)

			targetMs := 16.67 // 60 FPS
			actualMs := float64(renderTime.Nanoseconds()) / 1e6

			b.ReportMetric(actualMs, "final_render_ms")

			if actualMs > targetMs {
				b.Logf("WARNING: Final render time %.2fms exceeds 60 FPS target for %d cards",
					actualMs, s.cards)
			}
		})
	}
}

// BenchmarkMemoryUsage measures memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	scenarios := []struct {
		name  string
		cards int
	}{
		{"Memory_100", 100},
		{"Memory_500", 500},
		{"Memory_1000", 1000},
		{"Memory_2000", 2000},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			var models []*Model

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				model := setupKanbanWithCards(s.cards)
				_ = model.View() // Force rendering to allocate structures
				models = append(models, model)

				// Prevent optimization from removing the models
				if len(models) > 10 {
					models = models[1:]
				}
			}

			// Memory usage will be reported through b.ReportAllocs()
			// which shows allocations per operation
			if len(models) > 0 {
				// Estimate based on model count
				b.Logf("Created %d models with %d cards each", len(models), s.cards)
			}
		})
	}
}

// BenchmarkProfiling benchmarks the profiling system itself
func BenchmarkProfiling(b *testing.B) {
	profiler := NewKanbanProfiler(60)
	profiler.Enable()
	defer profiler.Disable()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		done := profiler.StartFrame(ctx)

		// Simulate some work
		time.Sleep(time.Microsecond)

		done()
	}

	// Verify profiler is collecting data
	report, err := profiler.GenerateReport(ctx)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportMetric(float64(report.TotalFrames), "profiled_frames")
	b.ReportMetric(report.ActualFPS, "actual_fps")
}

// BenchmarkDemo200Plus focuses specifically on 200+ card performance scenarios
// that match the demo requirements from the marketing materials
func BenchmarkDemo200Plus(b *testing.B) {
	scenarios := []struct {
		name        string
		cards       int
		description string
	}{
		{"Demo_Target_200", 200, "Target demo performance with 200 cards"},
		{"Demo_Stress_300", 300, "Stress test above demo target"},
		{"Demo_Extreme_500", 500, "Extreme load test for edge cases"},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			model := setupKanbanWithCards(s.cards)

			// Enable profiling to match demo scenario
			model.EnableProfiling()
			defer model.DisableProfiling()

			b.ResetTimer()
			b.ReportAllocs()

			var totalRenderTime time.Duration
			var renderCount int

			for i := 0; i < b.N; i++ {
				start := time.Now()
				view := viewutil.String(model.View())
				renderTime := time.Since(start)
				totalRenderTime += renderTime
				renderCount++

				// Verify non-empty output (prevent optimization)
				if len(view) < 100 {
					b.Fatalf("Rendered view too small: %d chars", len(view))
				}
			}

			// Calculate metrics that match demo claims
			avgRenderTime := totalRenderTime / time.Duration(renderCount)
			avgRenderMs := float64(avgRenderTime.Nanoseconds()) / 1e6
			calculatedFPS := 1000.0 / avgRenderMs

			// Report key demo metrics
			b.ReportMetric(avgRenderMs, "avg_render_ms")
			b.ReportMetric(calculatedFPS, "calculated_fps")

			// Verify demo performance claims
			const targetFPS = 30.0
			const maxLatencyMs = 16.67 // ~60 FPS for smoothness

			if calculatedFPS < targetFPS {
				b.Logf("WARNING: FPS %.1f below demo target %.1f for %d cards",
					calculatedFPS, targetFPS, s.cards)
			}

			if avgRenderMs > maxLatencyMs {
				b.Logf("WARNING: Render latency %.2fms exceeds smoothness target %.2fms for %d cards",
					avgRenderMs, maxLatencyMs, s.cards)
			}

			// Check memory usage
			memUsage, err := model.GetMemoryUsage()
			if err == nil {
				memUsageMB := float64(memUsage) / (1024 * 1024)
				b.ReportMetric(memUsageMB, "memory_mb")

				// Memory efficiency check - should be reasonable for 200+ cards
				memPerCard := memUsageMB / float64(s.cards) * 1024 // KB per card
				b.ReportMetric(memPerCard, "memory_per_card_kb")

				if memPerCard > 5.0 { // More than 5KB per card seems excessive
					b.Logf("WARNING: Memory usage %.2f KB/card may be too high for %d cards",
						memPerCard, s.cards)
				}
			}
		})
	}
}

// BenchmarkDemoSearch tests search performance with 200+ cards
// matching the search demo scenario
func BenchmarkDemoSearch(b *testing.B) {
	scenarios := []struct {
		name        string
		cards       int
		searchTerms []string
	}{
		{
			"Search_200_Common", 200,
			[]string{"task", "test", "performance", "demo", "api"},
		},
		{
			"Search_200_Specific", 200,
			[]string{"TASK-0001", "user5", "authentication", "database"},
		},
		{
			"Search_500_Complex", 500,
			[]string{"complex description", "multiple words", "detailed task"},
		},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			model := setupKanbanWithCards(s.cards)

			b.ResetTimer()
			b.ReportAllocs()

			var totalSearchTime time.Duration
			var searchCount int

			for i := 0; i < b.N; i++ {
				searchTerm := s.searchTerms[i%len(s.searchTerms)]

				start := time.Now()
				model.viewportState.SearchFilter = searchTerm
				model.updateColumns()
				_ = model.View() // Render with search results
				searchTime := time.Since(start)

				totalSearchTime += searchTime
				searchCount++

				// Clear search for next iteration
				model.viewportState.SearchFilter = ""
				model.updateColumns()
			}

			avgSearchTime := totalSearchTime / time.Duration(searchCount)
			avgSearchMs := float64(avgSearchTime.Nanoseconds()) / 1e6

			b.ReportMetric(avgSearchMs, "avg_search_ms")

			// Demo claims search results in <100ms
			const maxSearchMs = 100.0
			if avgSearchMs > maxSearchMs {
				b.Logf("WARNING: Search time %.2fms exceeds demo target %.2fms for %d cards",
					avgSearchMs, maxSearchMs, s.cards)
			}
		})
	}
}

// BenchmarkDemoEventThroughput tests event processing throughput
// matching the ">5k events/second" demo claim
func BenchmarkDemoEventThroughput(b *testing.B) {
	model := setupKanbanWithCards(200)

	// Create a variety of events
	eventTypes := []string{
		"task.created", "task.moved", "task.updated",
		"task.completed", "task.assigned", "task.blocked", "task.unblocked",
	}

	events := make([]taskEventMsg, 1000)
	for i := range events {
		events[i] = taskEventMsg{
			eventType: eventTypes[i%len(eventTypes)],
			taskID:    fmt.Sprintf("TASK-%04d", (i%200)+1),
			boardID:   "test-board",
			data:      map[string]interface{}{"timestamp": time.Now()},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	var totalEventTime time.Duration
	var eventCount int

	for i := 0; i < b.N; i++ {
		event := events[i%len(events)]

		start := time.Now()
		model.handleTaskEvent(event)
		eventTime := time.Since(start)

		totalEventTime += eventTime
		eventCount++
	}

	avgEventTime := totalEventTime / time.Duration(eventCount)
	eventsPerSecond := float64(time.Second) / float64(avgEventTime)

	b.ReportMetric(eventsPerSecond, "events_per_second")
	b.ReportMetric(float64(avgEventTime.Nanoseconds())/1000, "avg_event_us")

	// Demo claims >5k events/second
	const minEventsPerSecond = 5000.0
	if eventsPerSecond < minEventsPerSecond {
		b.Logf("WARNING: Event throughput %.0f/s below demo target %.0f/s",
			eventsPerSecond, minEventsPerSecond)
	}
}

// BenchmarkDemoRealtimeUpdates tests the <200ms real-time update claim
func BenchmarkDemoRealtimeUpdates(b *testing.B) {
	scenarios := []struct {
		name  string
		cards int
	}{
		{"Realtime_200", 200},
		{"Realtime_300", 300},
		{"Realtime_500", 500},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			model := setupKanbanWithCards(s.cards)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Simulate complete real-time update cycle:
				// 1. Event received -> 2. Model updated -> 3. UI rendered

				start := time.Now()

				// Step 1: Handle incoming event
				event := taskEventMsg{
					eventType: "task.moved",
					taskID:    fmt.Sprintf("TASK-%04d", (i%s.cards)+1),
					boardID:   "test-board",
					data: map[string]interface{}{
						"from_status": "todo",
						"to_status":   "in_progress",
					},
				}
				model.handleTaskEvent(event)

				// Step 2: Update model state
				model.updateColumns()

				// Step 3: Render updated view
				_ = model.View()

				updateLatency := time.Since(start)
				updateLatencyMs := float64(updateLatency.Nanoseconds()) / 1e6

				// Track individual update times
				if i < 10 { // Log first few for verification
					b.Logf("Update %d latency: %.2fms", i+1, updateLatencyMs)
				}
			}

			// The benchmark framework will report average time per operation
			// which represents our end-to-end update latency
		})
	}
}

// BenchmarkDemoLowQualityMode tests performance in low quality mode
// for maintaining performance under extreme load
func BenchmarkDemoLowQualityMode(b *testing.B) {
	scenarios := []struct {
		name       string
		cards      int
		lowQuality bool
	}{
		{"HighQuality_1000", 1000, false},
		{"LowQuality_1000", 1000, true},
		{"HighQuality_2000", 2000, false},
		{"LowQuality_2000", 2000, true},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			model := setupKanbanWithCards(s.cards)

			err := model.SetLowQualityMode(s.lowQuality)
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = model.View()
			}

			// Get performance report
			if model.profiler != nil && model.profiler.IsEnabled() {
				report, err := model.GetPerformanceReport()
				if err == nil {
					b.ReportMetric(report.ActualFPS, "actual_fps")
					b.ReportMetric(float64(report.AvgRenderTime.Nanoseconds())/1e6, "avg_frame_ms")
				}
			}
		})
	}
}
