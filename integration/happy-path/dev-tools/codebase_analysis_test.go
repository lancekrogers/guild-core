// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package dev_tools

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodebaseAnalysisPerformance_HappyPath validates development tools performance
func TestCodebaseAnalysisPerformance_HappyPath(t *testing.T) {
	framework := NewDevToolsTestFramework(t)
	defer framework.Cleanup()

	analysisScenarios := []struct {
		name                 string
		codebaseProfile      CodebaseProfile
		expectedAnalysisTime time.Duration
		expectedIndexTime    time.Duration
		expectedQueryTime    time.Duration
		expectedMemoryUsage  int64 // bytes
		concurrentUsers      int
	}{
		{
			name: "Small Go project analysis",
			codebaseProfile: CodebaseProfile{
				Languages:   []string{"go", "yaml", "markdown"},
				FileCount:   150,
				LinesOfCode: 15000,
				TotalSizeMB: 5,
				Complexity:  CodeComplexityLow,
			},
			expectedAnalysisTime: 10 * time.Second,
			expectedIndexTime:    5 * time.Second,
			expectedQueryTime:    50 * time.Millisecond,
			expectedMemoryUsage:  50 * 1024 * 1024, // 50MB
			concurrentUsers:      3,
		},
		{
			name: "Large multi-language project analysis",
			codebaseProfile: CodebaseProfile{
				Languages:   []string{"go", "typescript", "python", "rust", "yaml", "json", "markdown"},
				FileCount:   2500,
				LinesOfCode: 250000,
				TotalSizeMB: 100,
				Complexity:  CodeComplexityHigh,
			},
			expectedAnalysisTime: 120 * time.Second,
			expectedIndexTime:    45 * time.Second,
			expectedQueryTime:    200 * time.Millisecond,
			expectedMemoryUsage:  500 * 1024 * 1024, // 500MB
			concurrentUsers:      8,
		},
		{
			name: "Enterprise monorepo analysis",
			codebaseProfile: CodebaseProfile{
				Languages:   []string{"go", "typescript", "python", "java", "cpp", "rust", "dockerfile", "yaml", "json"},
				FileCount:   10000,
				LinesOfCode: 1000000,
				TotalSizeMB: 500,
				Complexity:  CodeComplexityEnterprise,
			},
			expectedAnalysisTime: 600 * time.Second, // 10 minutes
			expectedIndexTime:    180 * time.Second, // 3 minutes
			expectedQueryTime:    500 * time.Millisecond,
			expectedMemoryUsage:  2 * 1024 * 1024 * 1024, // 2GB
			concurrentUsers:      15,
		},
	}

	for _, scenario := range analysisScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// PHASE 1: Generate realistic codebase structure
			codebaseStart := time.Now()
			codebase, err := framework.GenerateRealisticCodebase(scenario.codebaseProfile, CodebaseGenConfig{
				IncludeTests:         true,
				IncludeDependencies:  true,
				IncludeDocumentation: true,
				SimulateRealPatterns: true,
				VariableComplexity:   true,
			})
			codebaseGenTime := time.Since(codebaseStart)

			require.NoError(t, err, "Failed to generate codebase")
			t.Logf("📁 Generated codebase in %v: %d files, %d LOC",
				codebaseGenTime, len(codebase.Files), codebase.TotalLOC)

			// PHASE 2: Codebase Analysis Performance
			analysisStart := time.Now()

			analyzer, err := framework.CreateCodebaseAnalyzer(AnalyzerConfig{
				Languages:              scenario.codebaseProfile.Languages,
				EnableSemanticAnalysis: true,
				EnableDependencyGraph:  true,
				EnableMetrics:          true,
				ParallelWorkers:        4,
				MemoryLimit:            scenario.expectedMemoryUsage * 2, // Allow 2x expected
			})
			require.NoError(t, err, "Failed to create analyzer")

			analysisResult, err := analyzer.AnalyzeCodebase(context.Background(), codebase.RootPath)
			analysisTime := time.Since(analysisStart)

			require.NoError(t, err, "Codebase analysis failed")
			assert.LessOrEqual(t, analysisTime, scenario.expectedAnalysisTime,
				"Analysis time exceeded target: %v > %v", analysisTime, scenario.expectedAnalysisTime)

			// Validate analysis completeness
			assert.Equal(t, len(codebase.Files), len(analysisResult.AnalyzedFiles),
				"Not all files were analyzed: %d != %d", len(analysisResult.AnalyzedFiles), len(codebase.Files))

			assert.GreaterOrEqual(t, analysisResult.CoveragePercentage, 0.95,
				"Analysis coverage too low: %.2f%% < 95%%", analysisResult.CoveragePercentage*100)

			// PHASE 3: Indexing Performance
			indexingStart := time.Now()

			indexer, err := framework.CreateSemanticIndexer(IndexerConfig{
				IndexTypes:             []IndexType{IndexTypeSymbols, IndexTypeReferences, IndexTypeDependencies},
				EnableIncrementalIndex: true,
				CompressionLevel:       6,
				CacheSize:              100 * 1024 * 1024, // 100MB cache
			})
			require.NoError(t, err, "Failed to create indexer")

			indexResult, err := indexer.BuildIndex(context.Background(), analysisResult)
			indexingTime := time.Since(indexingStart)

			require.NoError(t, err, "Indexing failed")
			assert.LessOrEqual(t, indexingTime, scenario.expectedIndexTime,
				"Indexing time exceeded target: %v > %v", indexingTime, scenario.expectedIndexTime)

			// Validate index quality
			assert.GreaterOrEqual(t, indexResult.IndexCompleteness, 0.98,
				"Index completeness too low: %.2f%% < 98%%", indexResult.IndexCompleteness*100)

			indexSizeMB := float64(indexResult.IndexSizeBytes) / (1024 * 1024)
			expectedMaxIndexSize := float64(scenario.codebaseProfile.TotalSizeMB) * 0.3 // Index should be ≤30% of source
			assert.LessOrEqual(t, indexSizeMB, expectedMaxIndexSize,
				"Index size too large: %.1f MB > %.1f MB", indexSizeMB, expectedMaxIndexSize)

			// PHASE 4: Query Performance Under Load
			queryTestCtx, queryCancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer queryCancel()

			// Generate realistic query patterns
			queryPatterns := framework.GenerateRealisticQueries(scenario.codebaseProfile, QueryPatternConfig{
				SymbolLookups:     30,
				ReferenceFinds:    25,
				DependencyQueries: 15,
				SemanticSearches:  20,
				CrossLanguageRefs: 10,
			})

			// Execute concurrent queries
			var queryWg sync.WaitGroup
			queryMetrics := make([]*QueryMetrics, scenario.concurrentUsers)

			for userIdx := 0; userIdx < scenario.concurrentUsers; userIdx++ {
				queryWg.Add(1)
				go func(idx int) {
					defer queryWg.Done()

					userMetrics := NewQueryMetrics(idx)
					queryMetrics[idx] = userMetrics

					for _, queryPattern := range queryPatterns {
						queryStart := time.Now()

						queryResult, err := indexer.ExecuteQuery(queryTestCtx, Query{
							Type:       queryPattern.Type,
							Pattern:    queryPattern.Pattern,
							Language:   queryPattern.Language,
							MaxResults: 50,
						})
						queryDuration := time.Since(queryStart)

						userMetrics.RecordQuery(queryPattern.Type, queryDuration, err == nil)

						if err == nil {
							assert.LessOrEqual(t, queryDuration, scenario.expectedQueryTime,
								"Query time exceeded target for user %d: %v > %v",
								idx, queryDuration, scenario.expectedQueryTime)

							assert.NotEmpty(t, queryResult.Results,
								"Empty query result for pattern: %s", queryPattern.Pattern)

							// Validate result relevance
							relevanceScore := framework.CalculateQueryRelevance(queryPattern, queryResult)
							assert.GreaterOrEqual(t, relevanceScore, 0.8,
								"Query relevance too low: %.2f < 0.8", relevanceScore)
						}

						// Realistic inter-query delay
						time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)
					}
				}(userIdx)
			}

			queryWg.Wait()

			// PHASE 5: Memory Usage Validation
			memoryUsage := framework.MeasureMemoryUsage()
			assert.LessOrEqual(t, memoryUsage.CurrentBytes, scenario.expectedMemoryUsage,
				"Memory usage exceeded target: %d > %d bytes", memoryUsage.CurrentBytes, scenario.expectedMemoryUsage)

			// Check for memory leaks
			framework.ForceGarbageCollection()
			time.Sleep(1 * time.Second)
			postGCMemory := framework.MeasureMemoryUsage()

			memoryReleaseRatio := 1.0 - float64(postGCMemory.CurrentBytes)/float64(memoryUsage.CurrentBytes)
			assert.GreaterOrEqual(t, memoryReleaseRatio, 0.2,
				"Insufficient memory release after GC: %.2f%% < 20%%", memoryReleaseRatio*100)

			// PHASE 6: Performance Metrics Analysis
			totalQueries := 0
			totalQueryTime := time.Duration(0)
			successfulQueries := 0

			for _, metrics := range queryMetrics {
				summary := metrics.GetSummary()
				totalQueries += summary.TotalQueries
				totalQueryTime += summary.TotalTime
				successfulQueries += summary.SuccessfulQueries

				assert.GreaterOrEqual(t, summary.SuccessRate, 0.98,
					"Query success rate too low for user: %.2f%% < 98%%", summary.SuccessRate*100)
			}

			averageQueryTime := totalQueryTime / time.Duration(totalQueries)
			overallSuccessRate := float64(successfulQueries) / float64(totalQueries)

			assert.LessOrEqual(t, averageQueryTime, scenario.expectedQueryTime,
				"Average query time exceeded target: %v > %v", averageQueryTime, scenario.expectedQueryTime)
			assert.GreaterOrEqual(t, overallSuccessRate, 0.98,
				"Overall query success rate too low: %.2f%% < 98%%", overallSuccessRate*100)

			t.Logf("✅ Development tools performance test completed successfully")
			t.Logf("📊 Performance Summary:")
			t.Logf("   - Analysis Time: %v", analysisTime)
			t.Logf("   - Indexing Time: %v", indexingTime)
			t.Logf("   - Average Query Time: %v", averageQueryTime)
			t.Logf("   - Memory Usage: %.1f MB", float64(memoryUsage.CurrentBytes)/(1024*1024))
			t.Logf("   - Index Size: %.1f MB", indexSizeMB)
			t.Logf("   - Query Success Rate: %.2f%%", overallSuccessRate*100)
		})
	}
}
