// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package monitoring

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// PerformanceDashboard renders real-time performance dashboards
type PerformanceDashboard struct {
	renderer *DashboardRenderer
	config   *DashboardConfig
	mu       sync.RWMutex
}

// DashboardConfig configures dashboard behavior
type DashboardConfig struct {
	Width              int  `json:"width"`
	ShowColors         bool `json:"show_colors"`
	ShowSparklines     bool `json:"show_sparklines"`
	ShowRecommendations bool `json:"show_recommendations"`
	CompactMode        bool `json:"compact_mode"`
}

// DefaultDashboardConfig returns default dashboard configuration
func DefaultDashboardConfig() *DashboardConfig {
	return &DashboardConfig{
		Width:              80,
		ShowColors:         true,
		ShowSparklines:     true,
		ShowRecommendations: true,
		CompactMode:        false,
	}
}

// NewPerformanceDashboard creates a new performance dashboard
func NewPerformanceDashboard() *PerformanceDashboard {
	return &PerformanceDashboard{
		renderer: NewDashboardRenderer(),
		config:   DefaultDashboardConfig(),
	}
}

// Render renders the complete performance dashboard
func (pd *PerformanceDashboard) Render(metrics *MetricsCollector) string {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	var dashboard strings.Builder

	// Header
	dashboard.WriteString(pd.renderHeader())
	dashboard.WriteString("\n")

	// Response time metrics
	dashboard.WriteString(pd.renderResponseTime(metrics))
	dashboard.WriteString("\n")

	// Throughput metrics
	dashboard.WriteString(pd.renderThroughput(metrics))
	dashboard.WriteString("\n")

	// Resource usage
	dashboard.WriteString(pd.renderResourceUsage(metrics))
	dashboard.WriteString("\n")

	// Cache performance
	dashboard.WriteString(pd.renderCachePerformance(metrics))
	dashboard.WriteString("\n")

	// Error metrics
	dashboard.WriteString(pd.renderErrorMetrics(metrics))
	dashboard.WriteString("\n")

	// System health
	dashboard.WriteString(pd.renderSystemHealth(metrics))

	if pd.config.ShowRecommendations {
		dashboard.WriteString("\n")
		dashboard.WriteString(pd.renderRecommendations(metrics))
	}

	return dashboard.String()
}

// renderHeader renders the dashboard header
func (pd *PerformanceDashboard) renderHeader() string {
	separator := strings.Repeat("═", pd.config.Width)
	title := "GUILD FRAMEWORK PERFORMANCE DASHBOARD"
	
	// Center the title
	padding := (pd.config.Width - len(title)) / 2
	centeredTitle := strings.Repeat(" ", padding) + title
	
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	var header strings.Builder
	header.WriteString(separator + "\n")
	header.WriteString(centeredTitle + "\n")
	header.WriteString(fmt.Sprintf("%s%s\n", 
		strings.Repeat(" ", (pd.config.Width-len(timestamp))/2), 
		timestamp))
	header.WriteString(separator)

	return header.String()
}

// renderResponseTime renders response time metrics
func (pd *PerformanceDashboard) renderResponseTime(metrics *MetricsCollector) string {
	currentMetrics := metrics.GetCurrentMetrics()
	
	p50 := currentMetrics.ResponseTime.Percentile(0.50)
	p95 := currentMetrics.ResponseTime.Percentile(0.95)
	p99 := currentMetrics.ResponseTime.Percentile(0.99)

	var section strings.Builder
	section.WriteString("Response Time:\n")
	section.WriteString(fmt.Sprintf("  P50: %s\n", pd.formatDuration(p50)))
	section.WriteString(fmt.Sprintf("  P95: %s %s\n", 
		pd.formatDuration(p95), 
		pd.getSLOIndicator(p95, 100))) // 100ms threshold
	section.WriteString(fmt.Sprintf("  P99: %s %s\n", 
		pd.formatDuration(p99), 
		pd.getSLOIndicator(p99, 500))) // 500ms threshold

	if pd.config.ShowSparklines {
		section.WriteString(fmt.Sprintf("  Trend: %s\n", 
			pd.generateSparkline([]float64{p50, p95, p99})))
	}

	return section.String()
}

// renderThroughput renders throughput metrics
func (pd *PerformanceDashboard) renderThroughput(metrics *MetricsCollector) string {
	currentMetrics := metrics.GetCurrentMetrics()
	throughput := currentMetrics.Throughput.Value()

	// Calculate requests per second (simplified)
	rps := throughput / 60.0 // Rough estimate

	var section strings.Builder
	section.WriteString("Throughput:\n")
	section.WriteString(fmt.Sprintf("  Current: %.1f req/s\n", rps))
	section.WriteString(fmt.Sprintf("  Total Requests: %.0f\n", throughput))

	// Show trend indicator
	trend := pd.calculateTrend(throughput, 1000) // Compare to baseline of 1000
	section.WriteString(fmt.Sprintf("  Trend: %s\n", trend))

	return section.String()
}

// renderResourceUsage renders resource usage metrics
func (pd *PerformanceDashboard) renderResourceUsage(metrics *MetricsCollector) string {
	currentMetrics := metrics.GetCurrentMetrics()
	
	cpuUsage := currentMetrics.CPUUsage.Value() * 100
	memoryUsage := currentMetrics.MemoryUsage.Value()
	goroutines := currentMetrics.GoroutineCount.Value()

	var section strings.Builder
	section.WriteString("Resource Usage:\n")
	section.WriteString(fmt.Sprintf("  CPU:    %.1f%% %s\n", 
		cpuUsage, 
		pd.renderProgressBar(cpuUsage/100, 20)))
	section.WriteString(fmt.Sprintf("  Memory: %.1f MB %s\n", 
		memoryUsage,
		pd.getMemoryIndicator(memoryUsage)))
	section.WriteString(fmt.Sprintf("  Goroutines: %.0f\n", goroutines))

	return section.String()
}

// renderCachePerformance renders cache performance metrics
func (pd *PerformanceDashboard) renderCachePerformance(metrics *MetricsCollector) string {
	currentMetrics := metrics.GetCurrentMetrics()
	hitRate := currentMetrics.CacheHitRate.Value() * 100

	var section strings.Builder
	section.WriteString("Cache Performance:\n")
	section.WriteString(fmt.Sprintf("  Hit Rate: %.1f%% %s\n", 
		hitRate, 
		pd.getCacheIndicator(hitRate)))
	
	if pd.config.ShowSparklines {
		section.WriteString(fmt.Sprintf("  Trend: %s\n", 
			pd.generateSparkline([]float64{hitRate, hitRate+2, hitRate-1})))
	}

	return section.String()
}

// renderErrorMetrics renders error rate metrics
func (pd *PerformanceDashboard) renderErrorMetrics(metrics *MetricsCollector) string {
	currentMetrics := metrics.GetCurrentMetrics()
	errorRate := currentMetrics.ErrorRate.Value() * 100

	var section strings.Builder
	section.WriteString("Error Metrics:\n")
	section.WriteString(fmt.Sprintf("  Error Rate: %.2f%% %s\n", 
		errorRate, 
		pd.getErrorIndicator(errorRate)))

	return section.String()
}

// renderSystemHealth renders overall system health
func (pd *PerformanceDashboard) renderSystemHealth(metrics *MetricsCollector) string {
	health := pd.calculateOverallHealth(metrics)
	
	var section strings.Builder
	section.WriteString("System Health:\n")
	section.WriteString(fmt.Sprintf("  Overall: %.1f%% %s\n", 
		health, 
		pd.getHealthIndicator(health)))

	return section.String()
}

// renderRecommendations renders performance recommendations
func (pd *PerformanceDashboard) renderRecommendations(metrics *MetricsCollector) string {
	recommendations := pd.generateRecommendations(metrics)
	
	if len(recommendations) == 0 {
		return "Recommendations: ✅ All metrics within normal ranges"
	}

	var section strings.Builder
	section.WriteString("Recommendations:\n")
	for i, rec := range recommendations {
		section.WriteString(fmt.Sprintf("  %d. %s\n", i+1, rec))
	}

	return section.String()
}

// Helper methods for formatting and indicators

// formatDuration formats a duration value in milliseconds
func (pd *PerformanceDashboard) formatDuration(ms float64) string {
	if ms < 1 {
		return fmt.Sprintf("%.2fms", ms)
	}
	if ms < 1000 {
		return fmt.Sprintf("%.0fms", ms)
	}
	return fmt.Sprintf("%.2fs", ms/1000)
}

// getSLOIndicator returns an SLO status indicator
func (pd *PerformanceDashboard) getSLOIndicator(value, threshold float64) string {
	if !pd.config.ShowColors {
		if value <= threshold {
			return "[OK]"
		}
		return "[WARN]"
	}

	if value <= threshold {
		return "✅"
	} else if value <= threshold*1.5 {
		return "⚠️"
	}
	return "❌"
}

// getMemoryIndicator returns a memory usage indicator
func (pd *PerformanceDashboard) getMemoryIndicator(usage float64) string {
	if !pd.config.ShowColors {
		if usage < 400 {
			return "[OK]"
		} else if usage < 500 {
			return "[WARN]"
		}
		return "[CRIT]"
	}

	if usage < 400 {
		return "✅"
	} else if usage < 500 {
		return "⚠️"
	}
	return "❌"
}

// getCacheIndicator returns a cache hit rate indicator
func (pd *PerformanceDashboard) getCacheIndicator(hitRate float64) string {
	if !pd.config.ShowColors {
		if hitRate >= 90 {
			return "[GOOD]"
		} else if hitRate >= 80 {
			return "[OK]"
		}
		return "[POOR]"
	}

	if hitRate >= 90 {
		return "✅"
	} else if hitRate >= 80 {
		return "⚠️"
	}
	return "❌"
}

// getErrorIndicator returns an error rate indicator
func (pd *PerformanceDashboard) getErrorIndicator(errorRate float64) string {
	if !pd.config.ShowColors {
		if errorRate < 1 {
			return "[OK]"
		} else if errorRate < 5 {
			return "[WARN]"
		}
		return "[CRIT]"
	}

	if errorRate < 1 {
		return "✅"
	} else if errorRate < 5 {
		return "⚠️"
	}
	return "❌"
}

// getHealthIndicator returns a health status indicator
func (pd *PerformanceDashboard) getHealthIndicator(health float64) string {
	if !pd.config.ShowColors {
		if health >= 95 {
			return "[EXCELLENT]"
		} else if health >= 90 {
			return "[GOOD]"
		} else if health >= 80 {
			return "[OK]"
		}
		return "[POOR]"
	}

	if health >= 95 {
		return "✅"
	} else if health >= 90 {
		return "🟢"
	} else if health >= 80 {
		return "⚠️"
	}
	return "❌"
}

// renderProgressBar renders a text progress bar
func (pd *PerformanceDashboard) renderProgressBar(percentage float64, width int) string {
	if percentage > 1.0 {
		percentage = 1.0
	}
	if percentage < 0 {
		percentage = 0
	}

	filled := int(percentage * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("[%s]", bar)
}

// generateSparkline generates a simple text sparkline
func (pd *PerformanceDashboard) generateSparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}

	// Simple sparkline characters
	chars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	
	// Find min and max
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Normalize and convert to characters
	var sparkline strings.Builder
	for _, v := range values {
		normalized := 0.0
		if max != min {
			normalized = (v - min) / (max - min)
		}
		index := int(normalized * float64(len(chars)-1))
		if index >= len(chars) {
			index = len(chars) - 1
		}
		sparkline.WriteString(chars[index])
	}

	return sparkline.String()
}

// calculateTrend calculates a trend indicator
func (pd *PerformanceDashboard) calculateTrend(current, baseline float64) string {
	if current > baseline*1.1 {
		return "↗️ (↑ 10%)"
	} else if current < baseline*0.9 {
		return "↘️ (↓ 10%)"
	}
	return "→ (stable)"
}

// calculateOverallHealth calculates overall system health percentage
func (pd *PerformanceDashboard) calculateOverallHealth(metrics *MetricsCollector) float64 {
	currentMetrics := metrics.GetCurrentMetrics()
	
	score := 100.0

	// Response time health (P95 < 100ms = 100%, > 500ms = 0%)
	p95 := currentMetrics.ResponseTime.Percentile(0.95)
	if p95 > 100 {
		responseScore := math.Max(0, 100-((p95-100)/400)*100)
		score *= responseScore / 100
	}

	// Error rate health (< 1% = 100%, > 10% = 0%)
	errorRate := currentMetrics.ErrorRate.Value() * 100
	if errorRate > 1 {
		errorScore := math.Max(0, 100-((errorRate-1)/9)*100)
		score *= errorScore / 100
	}

	// Cache hit rate health (> 90% = 100%, < 50% = 0%)
	hitRate := currentMetrics.CacheHitRate.Value() * 100
	if hitRate < 90 {
		cacheScore := math.Max(0, ((hitRate-50)/40)*100)
		score *= cacheScore / 100
	}

	// Memory usage health (< 400MB = 100%, > 800MB = 0%)
	memUsage := currentMetrics.MemoryUsage.Value()
	if memUsage > 400 {
		memScore := math.Max(0, 100-((memUsage-400)/400)*100)
		score *= memScore / 100
	}

	return score
}

// generateRecommendations generates performance recommendations
func (pd *PerformanceDashboard) generateRecommendations(metrics *MetricsCollector) []string {
	var recommendations []string
	currentMetrics := metrics.GetCurrentMetrics()

	// Check response time
	p95 := currentMetrics.ResponseTime.Percentile(0.95)
	if p95 > 100 {
		recommendations = append(recommendations, 
			"Consider optimizing slow endpoints or adding caching")
	}

	// Check error rate
	errorRate := currentMetrics.ErrorRate.Value() * 100
	if errorRate > 1 {
		recommendations = append(recommendations, 
			"Investigate and fix sources of errors")
	}

	// Check cache hit rate
	hitRate := currentMetrics.CacheHitRate.Value() * 100
	if hitRate < 90 {
		recommendations = append(recommendations, 
			"Review cache strategy and warming policies")
	}

	// Check memory usage
	memUsage := currentMetrics.MemoryUsage.Value()
	if memUsage > 500 {
		recommendations = append(recommendations, 
			"Investigate memory leaks or optimize memory usage")
	}

	// Check goroutine count
	goroutines := currentMetrics.GoroutineCount.Value()
	if goroutines > 1000 {
		recommendations = append(recommendations, 
			"Review goroutine lifecycle and potential leaks")
	}

	return recommendations
}

// DashboardRenderer handles dashboard rendering
type DashboardRenderer struct {
	mu sync.RWMutex
}

// NewDashboardRenderer creates a new dashboard renderer
func NewDashboardRenderer() *DashboardRenderer {
	return &DashboardRenderer{}
}

// SetConfig configures the dashboard
func (pd *PerformanceDashboard) SetConfig(config *DashboardConfig) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.config = config
}