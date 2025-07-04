// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package optimization

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContinuousOptimization_HappyPath validates optimization feedback loops
func TestContinuousOptimization_HappyPath(t *testing.T) {
	framework := NewOptimizationTestFramework(t)
	defer framework.Cleanup()

	optimizationScenarios := []struct {
		name                 string
		optimizationTargets  []OptimizationTarget
		expectedImprovements map[string]float64 // metric -> improvement percentage
		testDuration         time.Duration
	}{
		{
			name: "Memory usage optimization",
			optimizationTargets: []OptimizationTarget{
				{
					Metric:       "memory_usage",
					CurrentValue: 500 * 1024 * 1024, // 500MB
					TargetValue:  400 * 1024 * 1024, // 400MB
					Strategy:     OptimizationStrategyMemoryPooling,
				},
				{
					Metric:       "gc_frequency",
					CurrentValue: 10, // GCs per minute
					TargetValue:  6,  // 6 GCs per minute
					Strategy:     OptimizationStrategyObjectReuse,
				},
			},
			expectedImprovements: map[string]float64{
				"memory_usage": 0.20, // 20% reduction
				"gc_frequency": 0.40, // 40% reduction
			},
			testDuration: 45 * time.Minute,
		},
		{
			name: "Response time optimization",
			optimizationTargets: []OptimizationTarget{
				{
					Metric:       "agent_selection_time",
					CurrentValue: 1.8, // 1.8 seconds
					TargetValue:  1.2, // 1.2 seconds
					Strategy:     OptimizationStrategyCaching,
				},
				{
					Metric:       "search_response_time",
					CurrentValue: 400, // 400ms
					TargetValue:  250, // 250ms
					Strategy:     OptimizationStrategyIndexOptimization,
				},
			},
			expectedImprovements: map[string]float64{
				"agent_selection_time": 0.33, // 33% improvement
				"search_response_time": 0.38, // 38% improvement
			},
			testDuration: 60 * time.Minute,
		},
	}

	for _, scenario := range optimizationScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// PHASE 1: Establish baseline performance
			baselineCollector, err := framework.StartBaselineCollection(BaselineConfig{
				MetricsToCollect: scenario.getMetricNames(),
				CollectionPeriod: 15 * time.Minute,
				SamplingInterval: 10 * time.Second,
			})
			require.NoError(t, err, "Failed to start baseline collection")

			// Generate realistic load during baseline
			loadGenerator := framework.CreateLoadGenerator(LoadConfig{
				UserLoad:     LoadLevelMedium,
				OperationMix: framework.GetTypicalOperationMix(),
				Duration:     15 * time.Minute,
			})

			baselineMetrics := loadGenerator.ExecuteLoad(baselineCollector)

			t.Logf("📊 Baseline Performance Established:")
			for metric, value := range baselineMetrics {
				t.Logf("   - %s: %v", metric, value)
			}

			// PHASE 2: Apply optimization strategies
			optimizationResults := make(map[string]*OptimizationResult)

			for _, target := range scenario.optimizationTargets {
				t.Logf("🔧 Applying optimization: %s -> %s", target.Metric, target.Strategy)

				optimizer, err := framework.CreateOptimizer(target.Strategy, OptimizerConfig{
					Target:            target,
					BaselineValue:     baselineMetrics[target.Metric],
					GradualRollout:    true,
					SafetyChecks:      true,
					RollbackOnFailure: true,
				})
				require.NoError(t, err, "Failed to create optimizer for %s", target.Metric)

				optimizationStart := time.Now()
				result, err := optimizer.Apply(context.Background())
				optimizationDuration := time.Since(optimizationStart)

				require.NoError(t, err, "Optimization failed for %s", target.Metric)

				result.AppliedAt = optimizationStart
				result.Duration = optimizationDuration
				optimizationResults[target.Metric] = result

				t.Logf("✅ Optimization applied for %s in %v", target.Metric, optimizationDuration)
			}

			// PHASE 3: Validate optimization impact
			postOptimizationCollector, err := framework.StartPostOptimizationCollection(
				PostOptimizationConfig{
					BaselineMetrics:     baselineMetrics,
					OptimizationResults: optimizationResults,
					ValidationPeriod:    20 * time.Minute,
				})
			require.NoError(t, err, "Failed to start post-optimization collection")

			// Generate identical load for comparison
			validationLoadGenerator := framework.CreateLoadGenerator(LoadConfig{
				UserLoad:            LoadLevelMedium,
				OperationMix:        framework.GetTypicalOperationMix(),
				Duration:            20 * time.Minute,
				IdenticalToBaseline: true, // Use same pattern as baseline
			})

			postOptimizationMetrics := validationLoadGenerator.ExecuteLoad(postOptimizationCollector)

			// PHASE 4: Analyze optimization effectiveness
			for metricName, expectedImprovement := range scenario.expectedImprovements {
				baselineValue := baselineMetrics[metricName]
				optimizedValue := postOptimizationMetrics[metricName]

				actualImprovement := framework.CalculateImprovement(
					baselineValue, optimizedValue, metricName)

				assert.GreaterOrEqual(t, actualImprovement, expectedImprovement*0.8,
					"Optimization for %s below 80%% of target: %.2f%% < %.2f%%",
					metricName, actualImprovement*100, expectedImprovement*0.8*100)

				// Check for regressions in other metrics
				for otherMetric, otherValue := range postOptimizationMetrics {
					if otherMetric != metricName {
						regression := framework.CalculateRegression(
							baselineMetrics[otherMetric], otherValue, otherMetric)

						assert.LessOrEqual(t, regression, 0.05,
							"Optimization of %s caused regression in %s: %.2f%% > 5%%",
							metricName, otherMetric, regression*100)
					}
				}

				t.Logf("📈 Optimization Results for %s:", metricName)
				t.Logf("   - Baseline: %v", baselineValue)
				t.Logf("   - Optimized: %v", optimizedValue)
				t.Logf("   - Improvement: %.2f%% (target: %.2f%%)",
					actualImprovement*100, expectedImprovement*100)
			}

			// PHASE 5: Validate optimization stability
			stabilityCtx, stabilityCancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer stabilityCancel()

			stabilityMonitor := framework.CreateStabilityMonitor(StabilityConfig{
				OptimizedMetrics:   postOptimizationMetrics,
				VarianceThreshold:  0.05, // 5% variance tolerance
				MonitoringInterval: 30 * time.Second,
			})

			stabilityResults := stabilityMonitor.Monitor(stabilityCtx)

			for metricName, stabilityMetric := range stabilityResults {
				assert.LessOrEqual(t, stabilityMetric.Variance, 0.05,
					"Optimization stability issue for %s: %.2f%% variance > 5%%",
					metricName, stabilityMetric.Variance*100)

				assert.GreaterOrEqual(t, stabilityMetric.Consistency, 0.95,
					"Optimization consistency issue for %s: %.2f%% < 95%%",
					metricName, stabilityMetric.Consistency*100)
			}

			// PHASE 6: Update performance baselines
			err = framework.UpdatePerformanceBaselines(postOptimizationMetrics)
			require.NoError(t, err, "Failed to update performance baselines")

			t.Logf("✅ Continuous optimization validation completed successfully")
		})
	}
}

// Helper method for getting metric names from optimization targets
func (scenario struct {
	name                 string
	optimizationTargets  []OptimizationTarget
	expectedImprovements map[string]float64
	testDuration         time.Duration
}) getMetricNames() []string {
	var metrics []string
	for _, target := range scenario.optimizationTargets {
		metrics = append(metrics, target.Metric)
	}
	return metrics
}
