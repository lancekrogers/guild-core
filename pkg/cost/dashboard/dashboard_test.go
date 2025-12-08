// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/pkg/cost"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCostDashboard tests the dashboard functionality
func TestCostDashboard(t *testing.T) {
	ctx := context.Background()

	t.Run("NewCostDashboard", func(t *testing.T) {
		tracker, aggregator := setupTestDashboardDeps(t)

		config := &DashboardConfig{
			RefreshInterval: 5 * time.Second,
			HistoryWindow:   time.Hour,
			EnableRealTime:  true,
			MaxAlerts:       10,
			Currency:        "USD",
			Theme:           "default",
		}

		dashboard, err := NewCostDashboard(ctx, tracker, aggregator, config)
		assert.NoError(t, err)
		assert.NotNil(t, dashboard)
		assert.Equal(t, config.RefreshInterval, dashboard.config.RefreshInterval)
	})

	t.Run("GenerateView", func(t *testing.T) {
		dashboard := setupTestDashboard(t)

		view, err := dashboard.GenerateView(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, view)
		assert.NotNil(t, view.CurrentCosts)
		assert.NotNil(t, view.Historical)
		assert.NotNil(t, view.Projections)
		assert.NotNil(t, view.Visualizations)
		assert.False(t, view.LastUpdated.IsZero())
	})

	t.Run("GetCurrentSnapshot", func(t *testing.T) {
		dashboard := setupTestDashboard(t)

		snapshot, err := dashboard.getCurrentSnapshot(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, snapshot)
		assert.Equal(t, "USD", snapshot.Currency)
		assert.GreaterOrEqual(t, snapshot.BudgetUsedPercent, 0.0)
	})

	t.Run("GetHistoricalData", func(t *testing.T) {
		dashboard := setupTestDashboard(t)

		historical, err := dashboard.getHistoricalData(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, historical)
		assert.Equal(t, "USD", historical.Currency)
		assert.GreaterOrEqual(t, historical.AverageCost, 0.0)
	})

	t.Run("GenerateVisualizations", func(t *testing.T) {
		dashboard := setupTestDashboard(t)

		view := &DashboardView{
			CurrentCosts: &CostSnapshot{
				CostByAgent: map[string]float64{
					"elena":  10.50,
					"marcus": 8.25,
					"vera":   5.75,
				},
				CostByProvider: map[string]float64{
					"openai":    12.00,
					"anthropic": 12.50,
				},
			},
			Historical: &HistoricalData{
				DataPoints: []TimePoint{
					{Timestamp: time.Now().Add(-time.Hour), Value: 5.0},
					{Timestamp: time.Now().Add(-30 * time.Minute), Value: 7.5},
					{Timestamp: time.Now(), Value: 10.0},
				},
			},
			Optimizations: []cost.Optimization{
				{
					Description: "Switch to gpt-3.5-turbo for simple tasks",
					Savings:     15.50,
				},
			},
			Visualizations: make(map[string]Visualization),
		}

		err := dashboard.generateVisualizations(ctx, view)
		assert.NoError(t, err)
		assert.Len(t, view.Visualizations, 4) // trend, agent, provider, savings
		assert.Contains(t, view.Visualizations, "cost_trend")
		assert.Contains(t, view.Visualizations, "cost_by_agent")
		assert.Contains(t, view.Visualizations, "cost_by_provider")
		assert.Contains(t, view.Visualizations, "savings_opportunities")
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		// Test nil tracker
		_, err := NewCostDashboard(ctx, nil, nil, nil)
		assert.Error(t, err)

		// Test nil aggregator
		tracker := setupTestTracker(t)
		_, err = NewCostDashboard(ctx, tracker, nil, nil)
		assert.Error(t, err)
	})
}

// TestTerminalDashboard tests the terminal UI dashboard
func TestTerminalDashboard(t *testing.T) {
	ctx := context.Background()

	t.Run("NewTerminalDashboard", func(t *testing.T) {
		dashboard := setupTestDashboard(t)

		termDashboard := NewTerminalDashboard(ctx, dashboard)
		assert.NotNil(t, termDashboard)
		assert.Equal(t, 120, termDashboard.width)
		assert.Equal(t, 40, termDashboard.height)
	})

	t.Run("Render", func(t *testing.T) {
		dashboard := setupTestDashboard(t)
		termDashboard := NewTerminalDashboard(ctx, dashboard)

		output, err := termDashboard.Render(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, output)
		assert.Contains(t, output, "GUILD COST MONITORING DASHBOARD")
	})

	t.Run("RenderComponents", func(t *testing.T) {
		dashboard := setupTestDashboard(t)
		termDashboard := NewTerminalDashboard(ctx, dashboard)

		// Test header rendering
		header := termDashboard.renderHeader()
		assert.NotEmpty(t, header)
		assert.Contains(t, header, "GUILD COST MONITORING DASHBOARD")

		// Test footer rendering
		footer := termDashboard.renderFooter()
		assert.NotEmpty(t, footer)
		assert.Contains(t, footer, "[R]efresh")

		// Test cost summary rendering
		snapshot := &CostSnapshot{
			Period: cost.TimePeriod{
				Start: time.Now().Add(-time.Hour),
				End:   time.Now(),
			},
			TotalCost:         25.50,
			HourlyRate:        25.50,
			DailyProjection:   612.00,
			MonthlyProjection: 18360.00,
			BudgetUsedPercent: 61.2,
			DaysUntilLimit:    15,
		}

		summary := termDashboard.renderCostSummary(snapshot)
		assert.NotEmpty(t, summary)
		assert.Contains(t, summary, "$25.50")
		assert.Contains(t, summary, "61.2%")

		// Test agent costs rendering
		costByAgent := map[string]float64{
			"elena":  12.00,
			"marcus": 8.50,
			"vera":   5.00,
		}

		agentCosts := termDashboard.renderAgentCosts(costByAgent)
		assert.NotEmpty(t, agentCosts)
		assert.Contains(t, agentCosts, "elena")
		assert.Contains(t, agentCosts, "$12.00")
	})

	t.Run("SetSize", func(t *testing.T) {
		dashboard := setupTestDashboard(t)
		termDashboard := NewTerminalDashboard(ctx, dashboard)

		termDashboard.SetSize(100, 30)
		assert.Equal(t, 100, termDashboard.width)
		assert.Equal(t, 30, termDashboard.height)
	})
}

// TestDataVisualizer tests the visualization system
func TestDataVisualizer(t *testing.T) {
	ctx := context.Background()

	t.Run("NewDataVisualizer", func(t *testing.T) {
		visualizer, err := NewDataVisualizer(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, visualizer)
		assert.Equal(t, 80, visualizer.config.DefaultWidth)
		assert.Equal(t, 20, visualizer.config.DefaultHeight)
	})

	t.Run("GenerateTrend", func(t *testing.T) {
		visualizer := setupTestVisualizer(t)

		historical := &HistoricalData{
			DataPoints: []TimePoint{
				{Timestamp: time.Now().Add(-2 * time.Hour), Value: 5.0},
				{Timestamp: time.Now().Add(-time.Hour), Value: 8.0},
				{Timestamp: time.Now(), Value: 12.0},
			},
		}

		viz := visualizer.GenerateTrend(historical, "1h", "24h")
		assert.Equal(t, VisualizationTrend, viz.Type)
		assert.Equal(t, "Cost Trend", viz.Title)
		assert.NotNil(t, viz.Data)
		assert.Contains(t, viz.Config, "interval")
		assert.Contains(t, viz.Config, "window")
	})

	t.Run("GeneratePieChart", func(t *testing.T) {
		visualizer := setupTestVisualizer(t)

		data := map[string]float64{
			"elena":  15.50,
			"marcus": 12.25,
			"vera":   8.75,
		}

		viz := visualizer.GeneratePieChart(data, "Cost by Agent")
		assert.Equal(t, VisualizationPieChart, viz.Type)
		assert.Equal(t, "Cost by Agent", viz.Title)
		assert.NotNil(t, viz.Data)
		assert.Contains(t, viz.Config, "total")
		assert.Contains(t, viz.Config, "show_labels")
	})

	t.Run("GenerateBarChart", func(t *testing.T) {
		visualizer := setupTestVisualizer(t)

		data := map[string]float64{
			"openai":    18.50,
			"anthropic": 15.25,
			"ollama":    2.75,
		}

		viz := visualizer.GenerateBarChart(data, "Cost by Provider")
		assert.Equal(t, VisualizationBarChart, viz.Type)
		assert.Equal(t, "Cost by Provider", viz.Title)
		assert.NotNil(t, viz.Data)
		assert.Contains(t, viz.Config, "max_value")
		assert.Contains(t, viz.Config, "horizontal")
	})

	t.Run("GenerateSavingsChart", func(t *testing.T) {
		visualizer := setupTestVisualizer(t)

		optimizations := []cost.Optimization{
			{
				Description: "Switch to gpt-3.5-turbo for simple tasks",
				Savings:     15.50,
				Type:        cost.OptimizationModelSwitch,
				Priority:    100,
				Confidence:  0.85,
			},
			{
				Description: "Enable response caching",
				Savings:     8.25,
				Type:        cost.OptimizationCaching,
				Priority:    80,
				Confidence:  0.90,
			},
		}

		viz := visualizer.GenerateSavingsChart(optimizations)
		assert.Equal(t, VisualizationSavings, viz.Type)
		assert.Equal(t, "Cost Optimization Opportunities", viz.Title)
		assert.NotNil(t, viz.Data)
		assert.Contains(t, viz.Config, "total_savings")
		assert.Contains(t, viz.Config, "max_items")
	})
}

// TestASCIIChart tests ASCII chart generation
func TestASCIIChart(t *testing.T) {
	t.Run("Plot", func(t *testing.T) {
		chart := &ASCIIChart{
			Width:  40,
			Height: 10,
			Title:  "Test Chart",
		}

		values := []float64{1.0, 3.0, 2.0, 5.0, 4.0, 6.0, 7.0, 5.0, 3.0, 1.0}

		output := chart.Plot(values)
		assert.NotEmpty(t, output)
		assert.Contains(t, output, "─") // Should contain line characters
	})

	t.Run("PlotBarChart", func(t *testing.T) {
		chart := &ASCIIChart{
			Width:  60,
			Height: 5,
		}

		data := map[string]float64{
			"elena":  15.50,
			"marcus": 12.25,
			"vera":   8.75,
		}

		output := chart.PlotBarChart(data)
		assert.NotEmpty(t, output)
		assert.Contains(t, output, "elena")
		assert.Contains(t, output, "█") // Should contain bar characters
	})

	t.Run("EmptyData", func(t *testing.T) {
		chart := &ASCIIChart{Width: 40, Height: 10}

		// Test empty values
		output := chart.Plot([]float64{})
		assert.Contains(t, output, "No data available")

		// Test empty map
		output = chart.PlotBarChart(map[string]float64{})
		assert.Contains(t, output, "No data available")
	})
}

// TestAlertManager tests the alert management system
func TestAlertManager(t *testing.T) {
	ctx := context.Background()

	t.Run("NewAlertManager", func(t *testing.T) {
		manager, err := NewAlertManager(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, manager)
		assert.NotEmpty(t, manager.rules)
	})

	t.Run("EvaluateAlerts", func(t *testing.T) {
		manager := setupTestAlertManager(t)

		metrics := Metrics{
			TotalCost:      850.0,
			BudgetLimit:    1000.0,
			HourlyRate:     35.42,
			CostByAgent:    map[string]float64{"elena": 400.0, "marcus": 450.0},
			CostByProvider: map[string]float64{"openai": 500.0, "anthropic": 350.0},
			Period:         time.Hour,
			Timestamp:      time.Now(),
		}

		alerts, err := manager.EvaluateAlerts(ctx, metrics)
		assert.NoError(t, err)
		assert.NotNil(t, alerts)
		// May have budget alerts if threshold is exceeded
	})

	t.Run("GetActive", func(t *testing.T) {
		manager := setupTestAlertManager(t)

		active, err := manager.GetActive(ctx)
		assert.NoError(t, err)
		// Should be empty initially but not nil
		if active != nil {
			assert.Empty(t, active)
		}
	})

	t.Run("ResolveAlert", func(t *testing.T) {
		manager := setupTestAlertManager(t)

		// Create a test alert first
		alert := Alert{
			ID:        "test-alert",
			Type:      AlertTypeBudget,
			Severity:  "high",
			Message:   "Test alert",
			Timestamp: time.Now(),
		}

		err := manager.history.Store(ctx, alert)
		require.NoError(t, err)

		// Resolve the alert
		err = manager.ResolveAlert(ctx, "test-alert")
		assert.NoError(t, err)
	})
}

// TestAlertRules tests individual alert rules
func TestAlertRules(t *testing.T) {
	ctx := context.Background()

	t.Run("BudgetAlertRule", func(t *testing.T) {
		rule := NewBudgetAlertRule(80.0, "high")
		assert.Equal(t, AlertTypeBudget, rule.GetType())

		// Test threshold exceeded
		metrics := Metrics{
			TotalCost:   850.0,
			BudgetLimit: 1000.0,
		}

		alert, err := rule.Evaluate(ctx, metrics)
		assert.NoError(t, err)
		assert.NotNil(t, alert) // Should trigger alert at 85% usage
		assert.Equal(t, "high", alert.Severity)

		// Test threshold not exceeded
		metrics.TotalCost = 700.0
		alert, err = rule.Evaluate(ctx, metrics)
		assert.NoError(t, err)
		assert.Nil(t, alert) // Should not trigger at 70% usage
	})

	t.Run("AnomalyAlertRule", func(t *testing.T) {
		rule := NewAnomalyAlertRule(2.0)
		assert.Equal(t, AlertTypeAnomaly, rule.GetType())

		// Test with normal usage (should not trigger)
		metrics := Metrics{
			HourlyRate: 15.0, // Within normal range
		}

		_, err := rule.Evaluate(ctx, metrics)
		assert.NoError(t, err)
		// May or may not trigger depending on baseline
	})

	t.Run("SpikeAlertRule", func(t *testing.T) {
		rule := NewSpikeAlertRule(5.0, time.Hour)
		assert.Equal(t, AlertTypeSpike, rule.GetType())

		// Test with high cost spike
		metrics := Metrics{
			TotalCost: 100.0, // Very high for 1 hour
			Period:    time.Hour,
		}

		alert, err := rule.Evaluate(ctx, metrics)
		assert.NoError(t, err)
		assert.NotNil(t, alert) // Should trigger spike alert
	})

	t.Run("UsageAlertRule", func(t *testing.T) {
		rule := NewUsageAlertRule(50.0, "daily")
		assert.Equal(t, AlertTypeUsage, rule.GetType())

		// Test with high daily rate
		metrics := Metrics{
			HourlyRate: 3.0, // $72/day
		}

		alert, err := rule.Evaluate(ctx, metrics)
		assert.NoError(t, err)
		assert.NotNil(t, alert) // Should trigger usage alert
	})
}

// Helper functions for dashboard tests

// setupTestDashboard creates a test dashboard
func setupTestDashboard(t *testing.T) *CostDashboard {
	ctx := context.Background()
	tracker, aggregator := setupTestDashboardDeps(t)

	dashboard, err := NewCostDashboard(ctx, tracker, aggregator, nil)
	require.NoError(t, err)
	return dashboard
}

// setupTestDashboardDeps creates test dependencies
func setupTestDashboardDeps(t *testing.T) (*cost.CostTracker, *cost.CostAggregator) {
	ctx := context.Background()

	tracker := setupTestTracker(t)
	aggregator, err := cost.NewCostAggregator(ctx, tracker, 5*time.Minute)
	require.NoError(t, err)

	return tracker, aggregator
}

// setupTestTracker creates a mock cost tracker
func setupTestTracker(t *testing.T) *cost.CostTracker {
	ctx := context.Background()
	config := &cost.TrackerConfig{
		UpdateInterval:       time.Minute,
		RetentionPeriod:      30 * 24 * time.Hour,
		AggregationWindow:    time.Hour,
		EnableRealTimeAlerts: true,
		BudgetLimits:         map[string]float64{"daily": 100.0},
	}
	tracker, err := cost.NewCostTracker(ctx, config)
	require.NoError(t, err)
	return tracker
}

// setupTestVisualizer creates a test visualizer
func setupTestVisualizer(t *testing.T) *DataVisualizer {
	ctx := context.Background()
	visualizer, err := NewDataVisualizer(ctx)
	require.NoError(t, err)
	return visualizer
}

// setupTestAlertManager creates a test alert manager
func setupTestAlertManager(t *testing.T) *AlertManager {
	ctx := context.Background()
	manager, err := NewAlertManager(ctx)
	require.NoError(t, err)
	return manager
}

// BenchmarkDashboardRendering benchmarks dashboard rendering performance
func BenchmarkDashboardRendering(b *testing.B) {
	ctx := context.Background()
	dashboard := setupBenchmarkDashboard(b)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := dashboard.GenerateView(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// setupBenchmarkDashboard creates a dashboard for benchmarking
func setupBenchmarkDashboard(b *testing.B) *CostDashboard {
	// Create a simplified dashboard for benchmarking
	// This would need actual implementations in practice
	return nil
}
