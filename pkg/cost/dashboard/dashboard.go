// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/guild-framework/guild-core/pkg/cost"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// CostDashboard provides real-time cost monitoring and visualization
type CostDashboard struct {
	metrics    *MetricsCollector
	aggregator *cost.CostAggregator
	visualizer *DataVisualizer
	alerts     *AlertManager
	tracker    *cost.CostTracker
	config     *DashboardConfig
}

// DashboardView represents the complete dashboard view
type DashboardView struct {
	CurrentCosts   *CostSnapshot            `json:"current_costs"`
	Historical     *HistoricalData          `json:"historical"`
	Projections    *cost.CostProjection     `json:"projections"`
	Optimizations  []cost.Optimization      `json:"optimizations"`
	Alerts         []Alert                  `json:"alerts"`
	Visualizations map[string]Visualization `json:"visualizations"`
	LastUpdated    time.Time                `json:"last_updated"`
}

// CostSnapshot represents current cost information
type CostSnapshot struct {
	Period            cost.TimePeriod    `json:"period"`
	TotalCost         float64            `json:"total_cost"`
	CostByAgent       map[string]float64 `json:"cost_by_agent"`
	CostByProvider    map[string]float64 `json:"cost_by_provider"`
	CostByModel       map[string]float64 `json:"cost_by_model"`
	HourlyRate        float64            `json:"hourly_rate"`
	DailyProjection   float64            `json:"daily_projection"`
	MonthlyProjection float64            `json:"monthly_projection"`
	BudgetUsedPercent float64            `json:"budget_used_percent"`
	DaysUntilLimit    int                `json:"days_until_limit"`
	Currency          string             `json:"currency"`
}

// HistoricalData contains historical cost trends
type HistoricalData struct {
	Period      cost.TimePeriod `json:"period"`
	DataPoints  []TimePoint     `json:"data_points"`
	TotalCost   float64         `json:"total_cost"`
	AverageCost float64         `json:"average_cost"`
	Currency    string          `json:"currency"`
}

// TimePoint represents a single data point in time series
type TimePoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Visualization represents chart or graph data
type Visualization struct {
	Type        VisualizationType      `json:"type"`
	Title       string                 `json:"title"`
	Data        interface{}            `json:"data"`
	Config      map[string]interface{} `json:"config,omitempty"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// VisualizationType defines types of visualizations
type VisualizationType string

const (
	VisualizationTrend    VisualizationType = "trend"
	VisualizationPieChart VisualizationType = "pie_chart"
	VisualizationBarChart VisualizationType = "bar_chart"
	VisualizationSavings  VisualizationType = "savings"
	VisualizationASCII    VisualizationType = "ascii"
)

// DashboardConfig contains dashboard configuration
type DashboardConfig struct {
	RefreshInterval time.Duration `json:"refresh_interval"`
	HistoryWindow   time.Duration `json:"history_window"`
	EnableRealTime  bool          `json:"enable_real_time"`
	MaxAlerts       int           `json:"max_alerts"`
	Currency        string        `json:"currency"`
	Theme           string        `json:"theme"`
}

// NewCostDashboard creates a new cost dashboard
func NewCostDashboard(ctx context.Context, tracker *cost.CostTracker, aggregator *cost.CostAggregator, config *DashboardConfig) (*CostDashboard, error) {
	ctx = observability.WithComponent(ctx, "cost.dashboard")
	ctx = observability.WithOperation(ctx, "NewCostDashboard")

	if tracker == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "tracker cannot be nil", nil).
			WithComponent("cost.dashboard").
			WithOperation("NewCostDashboard")
	}

	if aggregator == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "aggregator cannot be nil", nil).
			WithComponent("cost.dashboard").
			WithOperation("NewCostDashboard")
	}

	if config == nil {
		config = &DashboardConfig{
			RefreshInterval: 10 * time.Second,
			HistoryWindow:   24 * time.Hour,
			EnableRealTime:  true,
			MaxAlerts:       10,
			Currency:        "USD",
			Theme:           "default",
		}
	}

	metrics, err := NewMetricsCollector(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create metrics collector").
			WithComponent("cost.dashboard").
			WithOperation("NewCostDashboard")
	}

	visualizer, err := NewDataVisualizer(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create data visualizer").
			WithComponent("cost.dashboard").
			WithOperation("NewCostDashboard")
	}

	alertManager, err := NewAlertManager(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create alert manager").
			WithComponent("cost.dashboard").
			WithOperation("NewCostDashboard")
	}

	return &CostDashboard{
		metrics:    metrics,
		aggregator: aggregator,
		visualizer: visualizer,
		alerts:     alertManager,
		tracker:    tracker,
		config:     config,
	}, nil
}

// GenerateView generates a complete dashboard view
func (cd *CostDashboard) GenerateView(ctx context.Context) (*DashboardView, error) {
	ctx = observability.WithComponent(ctx, "cost.dashboard")
	ctx = observability.WithOperation(ctx, "GenerateView")

	view := &DashboardView{
		Visualizations: make(map[string]Visualization),
		LastUpdated:    time.Now(),
	}

	// Get current costs
	currentCosts, err := cd.getCurrentSnapshot(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get current snapshot").
			WithComponent("cost.dashboard").
			WithOperation("GenerateView")
	}
	view.CurrentCosts = currentCosts

	// Get historical data
	historical, err := cd.getHistoricalData(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get historical data").
			WithComponent("cost.dashboard").
			WithOperation("GenerateView")
	}
	view.Historical = historical

	// Get projections
	projections := cd.aggregator.GetProjections(ctx)
	view.Projections = projections

	// Get active optimizations (simplified for now)
	view.Optimizations = cd.getActiveOptimizations(ctx)

	// Get active alerts
	alerts, err := cd.alerts.GetActive(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get active alerts").
			WithComponent("cost.dashboard").
			WithOperation("GenerateView")
	}
	view.Alerts = alerts

	// Generate visualizations
	if err := cd.generateVisualizations(ctx, view); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to generate visualizations").
			WithComponent("cost.dashboard").
			WithOperation("GenerateView")
	}

	return view, nil
}

// getCurrentSnapshot gets current cost snapshot
func (cd *CostDashboard) getCurrentSnapshot(ctx context.Context) (*CostSnapshot, error) {
	summary, err := cd.aggregator.GetCurrentCosts(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get current costs").
			WithComponent("cost.dashboard").
			WithOperation("getCurrentSnapshot")
	}

	// Get budget information (would come from configuration in production)
	monthlyBudget := 1000.0 // Default budget
	budgetUsedPercent := (summary.DailyProjection * 30 / monthlyBudget) * 100

	return &CostSnapshot{
		Period:            summary.Period,
		TotalCost:         summary.TotalCost,
		CostByAgent:       summary.CostByAgent,
		CostByProvider:    summary.CostByProvider,
		CostByModel:       summary.CostByModel,
		HourlyRate:        summary.HourlyRate,
		DailyProjection:   summary.DailyProjection,
		MonthlyProjection: summary.DailyProjection * 30,
		BudgetUsedPercent: budgetUsedPercent,
		DaysUntilLimit:    int((monthlyBudget - summary.DailyProjection*30) / summary.DailyProjection),
		Currency:          summary.Currency,
	}, nil
}

// getHistoricalData gets historical cost data
func (cd *CostDashboard) getHistoricalData(ctx context.Context) (*HistoricalData, error) {
	period := cost.TimePeriod{
		Start: time.Now().Add(-cd.config.HistoryWindow),
		End:   time.Now(),
	}

	historicalCosts, err := cd.tracker.GetHistoricalCosts(ctx, period)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get historical costs").
			WithComponent("cost.dashboard").
			WithOperation("getHistoricalData")
	}

	// Convert to TimePoint format
	var dataPoints []TimePoint
	totalCost := 0.0
	for _, dp := range historicalCosts.DataPoints {
		dataPoints = append(dataPoints, TimePoint{
			Timestamp: dp.Timestamp,
			Value:     dp.Cost,
			Metadata:  dp.Metadata,
		})
		totalCost += dp.Cost
	}

	averageCost := 0.0
	if len(dataPoints) > 0 {
		averageCost = totalCost / float64(len(dataPoints))
	}

	return &HistoricalData{
		Period:      period,
		DataPoints:  dataPoints,
		TotalCost:   totalCost,
		AverageCost: averageCost,
		Currency:    historicalCosts.Currency,
	}, nil
}

// getActiveOptimizations gets active optimization opportunities
func (cd *CostDashboard) getActiveOptimizations(ctx context.Context) []cost.Optimization {
	// In production, this would query for active optimizations
	// For now, return empty slice
	return []cost.Optimization{}
}

// generateVisualizations generates all dashboard visualizations
func (cd *CostDashboard) generateVisualizations(ctx context.Context, view *DashboardView) error {
	// Generate cost trend chart
	if view.Historical != nil {
		trendViz := cd.visualizer.GenerateTrend(view.Historical, "1h", "24h")
		view.Visualizations["cost_trend"] = trendViz
	}

	// Generate cost by agent pie chart
	if view.CurrentCosts != nil && len(view.CurrentCosts.CostByAgent) > 0 {
		agentViz := cd.visualizer.GeneratePieChart(view.CurrentCosts.CostByAgent, "Cost by Agent")
		view.Visualizations["cost_by_agent"] = agentViz
	}

	// Generate cost by provider bar chart
	if view.CurrentCosts != nil && len(view.CurrentCosts.CostByProvider) > 0 {
		providerViz := cd.visualizer.GenerateBarChart(view.CurrentCosts.CostByProvider, "Cost by Provider")
		view.Visualizations["cost_by_provider"] = providerViz
	}

	// Generate savings opportunities chart
	if len(view.Optimizations) > 0 {
		savingsViz := cd.visualizer.GenerateSavingsChart(view.Optimizations)
		view.Visualizations["savings_opportunities"] = savingsViz
	}

	return nil
}

// TerminalDashboard provides terminal UI for the cost dashboard
type TerminalDashboard struct {
	dashboard *CostDashboard
	width     int
	height    int
}

// NewTerminalDashboard creates a new terminal dashboard
func NewTerminalDashboard(ctx context.Context, dashboard *CostDashboard) *TerminalDashboard {
	return &TerminalDashboard{
		dashboard: dashboard,
		width:     120,
		height:    40,
	}
}

// Render renders the dashboard for terminal display
func (td *TerminalDashboard) Render(ctx context.Context) (string, error) {
	ctx = observability.WithComponent(ctx, "cost.terminal_dashboard")
	ctx = observability.WithOperation(ctx, "Render")

	view, err := td.dashboard.GenerateView(ctx)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to generate dashboard view").
			WithComponent("cost.terminal_dashboard").
			WithOperation("Render")
	}

	var output strings.Builder

	// Header
	output.WriteString(td.renderHeader())
	output.WriteString("\n")

	// Current costs summary
	if view.CurrentCosts != nil {
		output.WriteString(td.renderCostSummary(view.CurrentCosts))
		output.WriteString("\n")
	}

	// Cost trend chart
	if trendViz, exists := view.Visualizations["cost_trend"]; exists {
		output.WriteString(td.renderTrendChart(trendViz))
		output.WriteString("\n")
	}

	// Agent costs table
	if view.CurrentCosts != nil && len(view.CurrentCosts.CostByAgent) > 0 {
		output.WriteString(td.renderAgentCosts(view.CurrentCosts.CostByAgent))
		output.WriteString("\n")
	}

	// Alerts
	if len(view.Alerts) > 0 {
		output.WriteString(td.renderAlerts(view.Alerts))
		output.WriteString("\n")
	}

	// Optimization opportunities
	if len(view.Optimizations) > 0 {
		output.WriteString(td.renderOptimizations(view.Optimizations))
		output.WriteString("\n")
	}

	// Footer with help
	output.WriteString(td.renderFooter())

	return output.String(), nil
}

// renderHeader renders the dashboard header
func (td *TerminalDashboard) renderHeader() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("69")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		Width(td.width)

	title := "GUILD COST MONITORING DASHBOARD"
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	headerText := fmt.Sprintf("%s%s%s",
		title,
		strings.Repeat(" ", td.width-len(title)-len(timestamp)-4),
		timestamp)

	return style.Render(headerText)
}

// renderCostSummary renders the current cost summary
func (td *TerminalDashboard) renderCostSummary(snapshot *CostSnapshot) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(60)

	content := fmt.Sprintf(`Current Period: %s - %s
Total Cost: $%.2f
Hourly Rate: $%.2f/hr
Daily Projection: $%.2f
Monthly Projection: $%.2f

Budget Used: %.1f%%
Days Until Limit: %d`,
		snapshot.Period.Start.Format("15:04"),
		snapshot.Period.End.Format("15:04"),
		snapshot.TotalCost,
		snapshot.HourlyRate,
		snapshot.DailyProjection,
		snapshot.MonthlyProjection,
		snapshot.BudgetUsedPercent,
		snapshot.DaysUntilLimit,
	)

	return style.Render(content)
}

// renderTrendChart renders the cost trend chart
func (td *TerminalDashboard) renderTrendChart(viz Visualization) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(80)

	title := lipgloss.NewStyle().Bold(true).Render("Cost Trend (24h)")

	// Simple ASCII chart representation
	chart := td.generateASCIIChart(viz)

	content := fmt.Sprintf("%s\n\n%s", title, chart)
	return style.Render(content)
}

// generateASCIIChart generates a simple ASCII chart
func (td *TerminalDashboard) generateASCIIChart(viz Visualization) string {
	// Simplified ASCII chart - in production, this would use the data
	return `$20 ┤                                    ╭─╮           
$15 ┤                    ╭───╮          ╭╯ ╰╮          
$10 ┤         ╭─────────╯   ╰──────────╯   ╰─         
 $5 ┤    ╭────╯                                        
 $0 └────┴────┴────┴────┴────┴────┴────┴────┴────     
    00:00   06:00   12:00   18:00   00:00`
}

// renderAgentCosts renders the cost breakdown by agent
func (td *TerminalDashboard) renderAgentCosts(costByAgent map[string]float64) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(60)

	title := lipgloss.NewStyle().Bold(true).Render("Cost by Agent")

	var lines []string
	total := 0.0
	for _, cost := range costByAgent {
		total += cost
	}

	for agent, cost := range costByAgent {
		percentage := (cost / total) * 100
		barLength := int(percentage / 5) // Scale to reasonable bar length
		bar := strings.Repeat("█", barLength)

		line := fmt.Sprintf("%-8s %s $%.2f (%.1f%%)",
			agent, bar, cost, percentage)
		lines = append(lines, line)
	}

	content := fmt.Sprintf("%s\n\n%s", title, strings.Join(lines, "\n"))
	return style.Render(content)
}

// renderAlerts renders active alerts
func (td *TerminalDashboard) renderAlerts(alerts []Alert) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")). // Red border for alerts
		Padding(1).
		Width(80)

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render("Active Alerts")

	var lines []string
	for _, alert := range alerts {
		severity := strings.ToUpper(alert.Severity)
		icon := td.getSeverityIcon(alert.Severity)

		line := fmt.Sprintf("%s %s: %s", icon, severity, alert.Message)
		lines = append(lines, line)
	}

	content := fmt.Sprintf("%s\n\n%s", title, strings.Join(lines, "\n"))
	return style.Render(content)
}

// getSeverityIcon returns an icon for alert severity
func (td *TerminalDashboard) getSeverityIcon(severity string) string {
	switch strings.ToLower(severity) {
	case "high":
		return "🚨"
	case "medium":
		return "⚠️"
	case "low":
		return "ℹ️"
	default:
		return "📢"
	}
}

// renderOptimizations renders optimization opportunities
func (td *TerminalDashboard) renderOptimizations(optimizations []cost.Optimization) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("34")). // Green border for savings
		Padding(1).
		Width(80)

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("34")).Render("Optimization Opportunities")

	var lines []string
	for i, opt := range optimizations {
		if i >= 3 { // Show top 3 optimizations
			break
		}

		line := fmt.Sprintf("%d. %s", i+1, opt.Description)
		savingsLine := fmt.Sprintf("   Potential savings: $%.2f/day (%.1f%%)",
			opt.Savings, (opt.Savings/100)*100) // Simplified percentage

		lines = append(lines, line)
		lines = append(lines, savingsLine)
		if i < len(optimizations)-1 && i < 2 {
			lines = append(lines, "")
		}
	}

	content := fmt.Sprintf("%s\n\n%s", title, strings.Join(lines, "\n"))
	return style.Render(content)
}

// renderFooter renders the dashboard footer with help
func (td *TerminalDashboard) renderFooter() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		Width(td.width)

	helpText := "[R]efresh [O]ptimize [H]istory [S]ettings [Q]uit"
	return style.Render(helpText)
}

// SetSize sets the terminal dashboard size
func (td *TerminalDashboard) SetSize(width, height int) {
	td.width = width
	td.height = height
}
