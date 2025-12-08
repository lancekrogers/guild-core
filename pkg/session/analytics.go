// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// SessionAnalytics provides usage analytics and insights for sessions
type SessionAnalytics struct {
	store             AnalyticsStore
	analyzer          *UsageAnalyzer
	reporter          *ReportGenerator
	reasoningInsights *ReasoningInsightGenerator
}

// AnalyticsStore defines the interface for analytics data storage
type AnalyticsStore interface {
	SaveAnalytics(ctx context.Context, analytics *AnalyticsData) error
	GetAnalytics(ctx context.Context, period TimePeriod) ([]*AnalyticsData, error)
	GetAnalyticsBySession(ctx context.Context, sessionID string) (*AnalyticsData, error)
	GetAggregateAnalytics(ctx context.Context, period TimePeriod) (*AggregateAnalytics, error)
}

// AnalyticsData contains comprehensive analytics for a session
type AnalyticsData struct {
	SessionID         string                 `json:"session_id"`
	UserID            string                 `json:"user_id"`
	Duration          time.Duration          `json:"duration"`
	MessageCount      int                    `json:"message_count"`
	AgentUsage        map[string]AgentUsage  `json:"agent_usage"`
	CommandUsage      map[string]int         `json:"command_usage"`
	TokenUsage        TokenUsage             `json:"token_usage"`
	TaskMetrics       TaskMetrics            `json:"task_metrics"`
	ProductivityScore float64                `json:"productivity_score"`
	ReasoningMetrics  *ReasoningMetrics      `json:"reasoning_metrics,omitempty"`
	Timestamp         time.Time              `json:"timestamp"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// AgentUsage contains usage statistics for an individual agent
type AgentUsage struct {
	MessageCount    int           `json:"message_count"`
	TokensGenerated int           `json:"tokens_generated"`
	TasksCompleted  int           `json:"tasks_completed"`
	AverageResponse time.Duration `json:"average_response"`
	SuccessRate     float64       `json:"success_rate"`
	ErrorCount      int           `json:"error_count"`
	LastActivity    time.Time     `json:"last_activity"`
}

// TokenUsage tracks token consumption across different models
type TokenUsage struct {
	Total            int                `json:"total"`
	Input            int                `json:"input"`
	Output           int                `json:"output"`
	Reasoning        int                `json:"reasoning"`
	ReasoningRatio   float64            `json:"reasoning_ratio"`
	ByModel          map[string]int     `json:"by_model"`
	ByAgent          map[string]int     `json:"by_agent"`
	ReasoningByModel map[string]int     `json:"reasoning_by_model"`
	ReasoningByAgent map[string]int     `json:"reasoning_by_agent"`
	Cost             float64            `json:"cost"`
	CostByModel      map[string]float64 `json:"cost_by_model"`
}

// TaskMetrics contains task-related analytics
type TaskMetrics struct {
	TasksCreated   int           `json:"tasks_created"`
	TasksCompleted int           `json:"tasks_completed"`
	TasksFailed    int           `json:"tasks_failed"`
	AverageTime    time.Duration `json:"average_time"`
	CompletionRate float64       `json:"completion_rate"`
}

// TimePeriod defines a time range for analytics
type TimePeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// NewSessionAnalytics creates a new session analytics instance
func NewSessionAnalytics(store AnalyticsStore) *SessionAnalytics {
	return &SessionAnalytics{
		store:             store,
		analyzer:          NewUsageAnalyzer(),
		reporter:          NewReportGenerator(),
		reasoningInsights: NewReasoningInsightGenerator(nil), // TODO: Pass actual reasoning analyzer
	}
}

// AnalyzeSession performs comprehensive analysis of a session
func (sa *SessionAnalytics) AnalyzeSession(ctx context.Context, session *Session) (*AnalyticsData, error) {
	analytics := &AnalyticsData{
		SessionID:    session.ID,
		UserID:       session.UserID,
		Duration:     session.LastActiveTime.Sub(session.StartTime),
		MessageCount: len(session.Messages),
		AgentUsage:   make(map[string]AgentUsage),
		CommandUsage: make(map[string]int),
		TokenUsage: TokenUsage{
			ByModel:          make(map[string]int),
			ByAgent:          make(map[string]int),
			ReasoningByModel: make(map[string]int),
			ReasoningByAgent: make(map[string]int),
			CostByModel:      make(map[string]float64),
		},
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Analyze messages
	if err := sa.analyzeMessages(session.Messages, analytics); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to analyze messages")
	}

	// Analyze agent performance
	sa.analyzeAgentPerformance(session, analytics)

	// Analyze tasks
	analytics.TaskMetrics = sa.analyzeTaskMetrics(session)

	// Calculate productivity score
	analytics.ProductivityScore = sa.calculateProductivity(analytics)

	// Calculate reasoning metrics
	if sa.reasoningInsights != nil {
		reasoningMetrics, err := sa.reasoningInsights.CalculateReasoningMetrics(ctx, analytics)
		if err == nil {
			analytics.ReasoningMetrics = reasoningMetrics
		}
	}

	// Add contextual metadata
	sa.addContextualMetadata(session, analytics)

	// Store analytics
	if err := sa.store.SaveAnalytics(ctx, analytics); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save analytics")
	}

	return analytics, nil
}

// analyzeMessages processes all messages for analytics
func (sa *SessionAnalytics) analyzeMessages(messages []Message, analytics *AnalyticsData) error {
	var lastMessageTime time.Time

	for _, msg := range messages {
		// Update agent usage
		usage := analytics.AgentUsage[msg.Agent]
		usage.MessageCount++
		usage.LastActivity = msg.Timestamp

		// Calculate response time if this is an agent response
		if msg.Type == MessageTypeAgent && !lastMessageTime.IsZero() {
			responseTime := msg.Timestamp.Sub(lastMessageTime)
			if usage.AverageResponse == 0 {
				usage.AverageResponse = responseTime
			} else {
				// Moving average
				usage.AverageResponse = (usage.AverageResponse + responseTime) / 2
			}
		}

		// Count tokens
		if tokens, ok := msg.Metadata["tokens"].(int); ok {
			usage.TokensGenerated += tokens
			analytics.TokenUsage.Total += tokens
			analytics.TokenUsage.ByAgent[msg.Agent] += tokens

			// Track tokens by model if available
			if model, ok := msg.Metadata["model"].(string); ok {
				analytics.TokenUsage.ByModel[model] += tokens
			}
		}

		// Count reasoning tokens separately
		if reasoningTokens, ok := msg.Metadata["reasoning_tokens"].(int); ok {
			analytics.TokenUsage.Reasoning += reasoningTokens
			analytics.TokenUsage.ReasoningByAgent[msg.Agent] += reasoningTokens

			// Track reasoning tokens by model if available
			if model, ok := msg.Metadata["model"].(string); ok {
				analytics.TokenUsage.ReasoningByModel[model] += reasoningTokens
			}
		}

		// Extract input/output tokens from usage metadata
		if usage, ok := msg.Metadata["usage"].(map[string]interface{}); ok {
			if inputTokens, ok := usage["prompt_tokens"].(int); ok {
				analytics.TokenUsage.Input += inputTokens
			}
			if outputTokens, ok := usage["completion_tokens"].(int); ok {
				analytics.TokenUsage.Output += outputTokens
			}
			if reasoningTokens, ok := usage["reasoning_tokens"].(int); ok {
				analytics.TokenUsage.Reasoning += reasoningTokens
			}
		}

		// Track commands
		if cmd := sa.extractCommand(msg.Content); cmd != "" {
			analytics.CommandUsage[cmd]++
		}

		// Check for errors
		if sa.isErrorMessage(msg) {
			usage.ErrorCount++
		}

		analytics.AgentUsage[msg.Agent] = usage
		lastMessageTime = msg.Timestamp
	}

	// Calculate success rates
	for agent, usage := range analytics.AgentUsage {
		totalMessages := usage.MessageCount
		if totalMessages > 0 {
			usage.SuccessRate = float64(totalMessages-usage.ErrorCount) / float64(totalMessages)
			analytics.AgentUsage[agent] = usage
		}
	}

	// Calculate reasoning ratio
	if analytics.TokenUsage.Total > 0 {
		analytics.TokenUsage.ReasoningRatio = float64(analytics.TokenUsage.Reasoning) / float64(analytics.TokenUsage.Total)
	}

	return nil
}

// analyzeAgentPerformance analyzes individual agent performance
func (sa *SessionAnalytics) analyzeAgentPerformance(session *Session, analytics *AnalyticsData) {
	for agentID, state := range session.State.ActiveAgents {
		usage := analytics.AgentUsage[agentID]

		// Count completed tasks
		usage.TasksCompleted = len(state.TaskQueue) // Simplified - would check actual completion

		analytics.AgentUsage[agentID] = usage
	}
}

// analyzeTaskMetrics calculates task-related metrics
func (sa *SessionAnalytics) analyzeTaskMetrics(session *Session) TaskMetrics {
	metrics := TaskMetrics{}

	// Analyze running tasks
	metrics.TasksCreated = len(session.Context.RunningTasks)

	// In a real implementation, this would query the task system
	// For now, we'll use simplified calculations
	if metrics.TasksCreated > 0 {
		metrics.TasksCompleted = int(float64(metrics.TasksCreated) * 0.8) // Assume 80% completion
		metrics.TasksFailed = metrics.TasksCreated - metrics.TasksCompleted
		metrics.CompletionRate = float64(metrics.TasksCompleted) / float64(metrics.TasksCreated)
		metrics.AverageTime = 15 * time.Minute // Simplified average
	}

	return metrics
}

// calculateProductivity calculates an overall productivity score
func (sa *SessionAnalytics) calculateProductivity(analytics *AnalyticsData) float64 {
	if analytics.Duration == 0 {
		return 0
	}

	// Productivity factors
	messagesPerMinute := float64(analytics.MessageCount) / analytics.Duration.Minutes()
	taskCompletionRate := analytics.TaskMetrics.CompletionRate

	// Agent efficiency (average success rate)
	var totalSuccessRate float64
	agentCount := len(analytics.AgentUsage)
	for _, usage := range analytics.AgentUsage {
		totalSuccessRate += usage.SuccessRate
	}
	avgSuccessRate := totalSuccessRate / float64(agentCount)
	if agentCount == 0 {
		avgSuccessRate = 1.0
	}

	// Command diversity (more diverse commands = higher productivity)
	commandDiversity := float64(len(analytics.CommandUsage)) / 10.0 // Normalize to 0-1
	if commandDiversity > 1.0 {
		commandDiversity = 1.0
	}

	// Reasoning efficiency (lower ratio = more efficient)
	reasoningEfficiency := 1.0 - analytics.TokenUsage.ReasoningRatio
	if reasoningEfficiency < 0.5 {
		reasoningEfficiency = 0.5 // Floor at 0.5 to not penalize thoughtful responses too much
	}

	// Calculate weighted score (0-100)
	// Adjusted weights to include reasoning efficiency
	score := (messagesPerMinute*0.25 + taskCompletionRate*0.35 + avgSuccessRate*0.2 + commandDiversity*0.1 + reasoningEfficiency*0.1) * 100

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return math.Round(score*100) / 100 // Round to 2 decimal places
}

// addContextualMetadata adds contextual information to analytics
func (sa *SessionAnalytics) addContextualMetadata(session *Session, analytics *AnalyticsData) {
	analytics.Metadata["working_directory"] = session.Context.WorkingDirectory
	analytics.Metadata["git_branch"] = session.Context.GitBranch
	analytics.Metadata["open_files_count"] = len(session.Context.OpenFiles)
	analytics.Metadata["active_agents_count"] = len(session.State.ActiveAgents)
	analytics.Metadata["session_duration_hours"] = analytics.Duration.Hours()
}

// extractCommand extracts command names from message content
func (sa *SessionAnalytics) extractCommand(content string) string {
	// Look for command patterns like /command or !command
	patterns := []string{
		`^/(\w+)`,
		`^!(\w+)`,
		`^\.(\w+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	// Look for tool invocations
	if strings.Contains(content, "```") && (strings.Contains(content, "bash") || strings.Contains(content, "shell")) {
		return "shell"
	}
	if strings.Contains(content, "git ") {
		return "git"
	}

	return ""
}

// isErrorMessage checks if a message indicates an error
func (sa *SessionAnalytics) isErrorMessage(msg Message) bool {
	content := strings.ToLower(msg.Content)
	errorKeywords := []string{"error", "failed", "exception", "panic", "fatal"}

	for _, keyword := range errorKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return false
}

// AggregateAnalytics contains aggregated analytics across multiple sessions
type AggregateAnalytics struct {
	Period            TimePeriod          `json:"period"`
	TotalSessions     int                 `json:"total_sessions"`
	TotalDuration     time.Duration       `json:"total_duration"`
	AverageSession    time.Duration       `json:"average_session"`
	TopAgents         []AgentRanking      `json:"top_agents"`
	TopCommands       []CommandRanking    `json:"top_commands"`
	ProductivityTrend []DataPoint         `json:"productivity_trend"`
	CostAnalysis      CostBreakdown       `json:"cost_analysis"`
	TokenUsage        AggregateTokenUsage `json:"token_usage"`
}

// AgentRanking represents agent usage ranking
type AgentRanking struct {
	Name       string  `json:"name"`
	Usage      int     `json:"usage"`
	Efficiency float64 `json:"efficiency"`
	Rank       int     `json:"rank"`
}

// CommandRanking represents command usage ranking
type CommandRanking struct {
	Command string `json:"command"`
	Count   int    `json:"count"`
	Rank    int    `json:"rank"`
}

// DataPoint represents a point in time-series data
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// CostBreakdown contains cost analysis information
type CostBreakdown struct {
	TotalCost        float64            `json:"total_cost"`
	CostByModel      map[string]float64 `json:"cost_by_model"`
	CostByAgent      map[string]float64 `json:"cost_by_agent"`
	PotentialSavings float64            `json:"potential_savings"`
	Recommendations  []string           `json:"recommendations"`
}

// AggregateTokenUsage contains aggregated token usage statistics
type AggregateTokenUsage struct {
	Total            int            `json:"total"`
	Reasoning        int            `json:"reasoning"`
	ReasoningRatio   float64        `json:"reasoning_ratio"`
	ByModel          map[string]int `json:"by_model"`
	ByAgent          map[string]int `json:"by_agent"`
	ReasoningByModel map[string]int `json:"reasoning_by_model"`
	ReasoningByAgent map[string]int `json:"reasoning_by_agent"`
	Trend            []DataPoint    `json:"trend"`
	ReasoningTrend   []DataPoint    `json:"reasoning_trend"`
	Efficiency       float64        `json:"efficiency"`
}

// GenerateReport creates a comprehensive analytics report
func (sa *SessionAnalytics) GenerateReport(ctx context.Context, period TimePeriod) (*AnalyticsReport, error) {
	// Get all analytics for the period
	data, err := sa.store.GetAnalytics(ctx, period)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get analytics data")
	}

	report := &AnalyticsReport{
		Period:    period,
		Generated: time.Now(),
		Sessions:  len(data),
	}

	// Aggregate metrics
	report.Aggregate = sa.aggregateMetrics(data)

	// Generate insights
	report.Insights = sa.generateInsights(report.Aggregate, data)

	// Create visualizations
	report.Charts = sa.generateCharts(report.Aggregate, data)

	return report, nil
}

// AnalyticsReport contains a complete analytics report
type AnalyticsReport struct {
	Period    TimePeriod          `json:"period"`
	Generated time.Time           `json:"generated"`
	Sessions  int                 `json:"sessions"`
	Aggregate *AggregateAnalytics `json:"aggregate"`
	Insights  []Insight           `json:"insights"`
	Charts    []Chart             `json:"charts"`
}

// Insight represents an actionable insight
type Insight struct {
	Type     InsightType     `json:"type"`
	Title    string          `json:"title"`
	Message  string          `json:"message"`
	Priority InsightPriority `json:"priority"`
	Actions  []Action        `json:"actions,omitempty"`
}

// InsightType categorizes different types of insights
type InsightType string

const (
	InsightProductivity InsightType = "productivity"
	InsightEfficiency   InsightType = "efficiency"
	InsightCost         InsightType = "cost"
	InsightUsage        InsightType = "usage"
)

// InsightPriority indicates the importance of an insight
type InsightPriority string

const (
	InsightPriorityHigh   InsightPriority = "high"
	InsightPriorityMedium InsightPriority = "medium"
	InsightPriorityLow    InsightPriority = "low"
)

// Action represents a recommended action
type Action struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// Chart represents visualization data
type Chart struct {
	Type   string      `json:"type"`
	Title  string      `json:"title"`
	Data   interface{} `json:"data"`
	Config interface{} `json:"config,omitempty"`
}

// aggregateMetrics aggregates analytics data across sessions
func (sa *SessionAnalytics) aggregateMetrics(data []*AnalyticsData) *AggregateAnalytics {
	if len(data) == 0 {
		return &AggregateAnalytics{}
	}

	aggregate := &AggregateAnalytics{
		TotalSessions: len(data),
		TopAgents:     make([]AgentRanking, 0),
		TopCommands:   make([]CommandRanking, 0),
		TokenUsage: AggregateTokenUsage{
			ByModel:          make(map[string]int),
			ByAgent:          make(map[string]int),
			ReasoningByModel: make(map[string]int),
			ReasoningByAgent: make(map[string]int),
		},
	}

	// Calculate totals
	var totalDuration time.Duration
	agentUsage := make(map[string]int)
	commandUsage := make(map[string]int)
	var totalProductivity float64

	for _, session := range data {
		totalDuration += session.Duration
		totalProductivity += session.ProductivityScore

		// Aggregate agent usage
		for agent, usage := range session.AgentUsage {
			agentUsage[agent] += usage.MessageCount
		}

		// Aggregate command usage
		for cmd, count := range session.CommandUsage {
			commandUsage[cmd] += count
		}

		// Aggregate token usage
		aggregate.TokenUsage.Total += session.TokenUsage.Total
		aggregate.TokenUsage.Reasoning += session.TokenUsage.Reasoning

		// Aggregate by model
		for model, tokens := range session.TokenUsage.ByModel {
			aggregate.TokenUsage.ByModel[model] += tokens
		}
		for model, tokens := range session.TokenUsage.ReasoningByModel {
			aggregate.TokenUsage.ReasoningByModel[model] += tokens
		}

		// Aggregate by agent
		for agent, tokens := range session.TokenUsage.ByAgent {
			aggregate.TokenUsage.ByAgent[agent] += tokens
		}
		for agent, tokens := range session.TokenUsage.ReasoningByAgent {
			aggregate.TokenUsage.ReasoningByAgent[agent] += tokens
		}
	}

	aggregate.TotalDuration = totalDuration
	aggregate.AverageSession = totalDuration / time.Duration(len(data))

	// Create agent rankings
	for agent, usage := range agentUsage {
		aggregate.TopAgents = append(aggregate.TopAgents, AgentRanking{
			Name:       agent,
			Usage:      usage,
			Efficiency: 0.85, // Simplified - would calculate actual efficiency
		})
	}

	// Sort agents by usage
	sort.Slice(aggregate.TopAgents, func(i, j int) bool {
		return aggregate.TopAgents[i].Usage > aggregate.TopAgents[j].Usage
	})

	// Assign ranks
	for i := range aggregate.TopAgents {
		aggregate.TopAgents[i].Rank = i + 1
	}

	// Create command rankings
	for cmd, count := range commandUsage {
		aggregate.TopCommands = append(aggregate.TopCommands, CommandRanking{
			Command: cmd,
			Count:   count,
		})
	}

	// Sort commands by count
	sort.Slice(aggregate.TopCommands, func(i, j int) bool {
		return aggregate.TopCommands[i].Count > aggregate.TopCommands[j].Count
	})

	// Assign ranks
	for i := range aggregate.TopCommands {
		aggregate.TopCommands[i].Rank = i + 1
	}

	// Calculate reasoning ratio
	if aggregate.TokenUsage.Total > 0 {
		aggregate.TokenUsage.ReasoningRatio = float64(aggregate.TokenUsage.Reasoning) / float64(aggregate.TokenUsage.Total)
	}

	// Generate productivity trend (simplified)
	aggregate.ProductivityTrend = sa.generateProductivityTrend(data)

	return aggregate
}

// generateInsights creates actionable insights from aggregated data
func (sa *SessionAnalytics) generateInsights(agg *AggregateAnalytics, data []*AnalyticsData) []Insight {
	var insights []Insight

	// Productivity insights
	if len(agg.ProductivityTrend) > 1 {
		trend := sa.analyzer.AnalyzeTrend(agg.ProductivityTrend)
		if trend.IsSignificant() {
			insights = append(insights, Insight{
				Type:     InsightProductivity,
				Title:    "Productivity Trend",
				Message:  fmt.Sprintf("Your productivity has %s by %.0f%% this period", trend.Direction, trend.Magnitude*100),
				Priority: InsightPriorityHigh,
			})
		}
	}

	// Agent efficiency insights
	for _, agent := range agg.TopAgents {
		if agent.Efficiency < 0.7 {
			insights = append(insights, Insight{
				Type:     InsightEfficiency,
				Title:    fmt.Sprintf("%s Efficiency", agent.Name),
				Message:  fmt.Sprintf("%s's efficiency is %.0f%%. Consider reviewing task assignments.", agent.Name, agent.Efficiency*100),
				Priority: InsightPriorityMedium,
			})
		}
	}

	// Generate reasoning-specific insights
	if sa.reasoningInsights != nil && len(data) > 0 {
		// Use the first analytics data for reasoning insights (simplified)
		reasoningInsights, err := sa.reasoningInsights.GenerateReasoningInsights(context.Background(), data[0])
		if err == nil {
			insights = append(insights, reasoningInsights...)
		}
	}

	// Usage patterns
	if len(agg.TopCommands) > 0 {
		topCommand := agg.TopCommands[0]
		insights = append(insights, Insight{
			Type:     InsightUsage,
			Title:    "Most Used Command",
			Message:  fmt.Sprintf("'%s' was your most used command (%d times). Consider creating shortcuts for efficiency.", topCommand.Command, topCommand.Count),
			Priority: InsightPriorityLow,
		})
	}

	// Reasoning efficiency insights
	if agg.TokenUsage.Total > 0 && agg.TokenUsage.ReasoningRatio > 0 {
		if agg.TokenUsage.ReasoningRatio > 0.5 {
			insights = append(insights, Insight{
				Type:     InsightEfficiency,
				Title:    "High Reasoning Token Usage",
				Message:  fmt.Sprintf("%.0f%% of tokens are used for reasoning. Consider more direct prompts for simple tasks.", agg.TokenUsage.ReasoningRatio*100),
				Priority: InsightPriorityMedium,
				Actions: []Action{
					{
						Title:       "Optimize Prompts",
						Description: "Use clearer, more specific prompts to reduce reasoning overhead",
						Type:        "optimization",
					},
				},
			})
		} else if agg.TokenUsage.ReasoningRatio < 0.1 {
			insights = append(insights, Insight{
				Type:     InsightUsage,
				Title:    "Low Reasoning Usage",
				Message:  "Only using reasoning for complex tasks. This is efficient token usage!",
				Priority: InsightPriorityLow,
			})
		}
	}

	return insights
}

// generateCharts creates visualization data
func (sa *SessionAnalytics) generateCharts(agg *AggregateAnalytics, data []*AnalyticsData) []Chart {
	var charts []Chart

	// Agent usage pie chart
	if len(agg.TopAgents) > 0 {
		agentData := make(map[string]int)
		for _, agent := range agg.TopAgents {
			agentData[agent.Name] = agent.Usage
		}

		charts = append(charts, Chart{
			Type:  "pie",
			Title: "Agent Usage Distribution",
			Data:  agentData,
		})
	}

	// Productivity trend line chart
	if len(agg.ProductivityTrend) > 0 {
		charts = append(charts, Chart{
			Type:  "line",
			Title: "Productivity Trend",
			Data:  agg.ProductivityTrend,
		})
	}

	// Command usage bar chart
	if len(agg.TopCommands) > 0 {
		commandData := make(map[string]int)
		for _, cmd := range agg.TopCommands[:min(10, len(agg.TopCommands))] { // Top 10
			commandData[cmd.Command] = cmd.Count
		}

		charts = append(charts, Chart{
			Type:  "bar",
			Title: "Top Commands",
			Data:  commandData,
		})
	}

	// Reasoning vs Content token distribution
	if agg.TokenUsage.Total > 0 {
		tokenDistribution := map[string]int{
			"Content Tokens":   agg.TokenUsage.Total - agg.TokenUsage.Reasoning,
			"Reasoning Tokens": agg.TokenUsage.Reasoning,
		}

		charts = append(charts, Chart{
			Type:  "pie",
			Title: "Token Distribution",
			Data:  tokenDistribution,
			Config: map[string]interface{}{
				"colors": []string{"#4CAF50", "#FF9800"}, // Green for content, orange for reasoning
			},
		})
	}

	// Reasoning efficiency by model
	if len(agg.TokenUsage.ReasoningByModel) > 0 {
		modelEfficiency := make(map[string]float64)
		for model, reasoningTokens := range agg.TokenUsage.ReasoningByModel {
			if totalTokens, ok := agg.TokenUsage.ByModel[model]; ok && totalTokens > 0 {
				efficiency := 1.0 - (float64(reasoningTokens) / float64(totalTokens))
				modelEfficiency[model] = math.Round(efficiency*1000) / 10 // Percentage with 1 decimal
			}
		}

		if len(modelEfficiency) > 0 {
			charts = append(charts, Chart{
				Type:  "bar",
				Title: "Model Reasoning Efficiency (%)",
				Data:  modelEfficiency,
			})
		}
	}

	// Add reasoning heatmap if available
	if sa.reasoningInsights != nil && len(data) > 0 {
		heatmap, err := sa.reasoningInsights.GenerateReasoningHeatmap(context.Background(), data[0])
		if err == nil {
			charts = append(charts, Chart{
				Type:  "heatmap",
				Title: "Reasoning Intensity by Hour",
				Data:  heatmap,
			})
		}
	}

	return charts
}

// generateProductivityTrend creates a productivity trend from analytics data
func (sa *SessionAnalytics) generateProductivityTrend(data []*AnalyticsData) []DataPoint {
	var points []DataPoint

	// Sort by timestamp
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

	// Create daily aggregations
	dailyProductivity := make(map[string][]float64)
	for _, session := range data {
		day := session.Timestamp.Format("2006-01-02")
		dailyProductivity[day] = append(dailyProductivity[day], session.ProductivityScore)
	}

	// Calculate daily averages
	for day, scores := range dailyProductivity {
		var total float64
		for _, score := range scores {
			total += score
		}
		avg := total / float64(len(scores))

		if timestamp, err := time.Parse("2006-01-02", day); err == nil {
			points = append(points, DataPoint{
				Timestamp: timestamp,
				Value:     avg,
			})
		}
	}

	// Sort by timestamp
	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp.Before(points[j].Timestamp)
	})

	return points
}

// UsageAnalyzer provides analysis utilities
type UsageAnalyzer struct{}

// NewUsageAnalyzer creates a new usage analyzer
func NewUsageAnalyzer() *UsageAnalyzer {
	return &UsageAnalyzer{}
}

// Trend represents a trend analysis result
type Trend struct {
	Direction   string  `json:"direction"`
	Magnitude   float64 `json:"magnitude"`
	Significant bool    `json:"significant"`
}

// IsSignificant returns true if the trend is statistically significant
func (t *Trend) IsSignificant() bool {
	return t.Significant
}

// AnalyzeTrend analyzes a trend in data points
func (ua *UsageAnalyzer) AnalyzeTrend(points []DataPoint) *Trend {
	if len(points) < 2 {
		return &Trend{Significant: false}
	}

	// Simple linear regression to detect trend
	var sumX, sumY, sumXY, sumXX float64
	n := float64(len(points))

	for i, point := range points {
		x := float64(i)
		y := point.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// Calculate slope
	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	// Determine direction and magnitude
	direction := "increased"
	if slope < 0 {
		direction = "decreased"
	}

	magnitude := math.Abs(slope) / (sumY / n) // Normalized magnitude

	return &Trend{
		Direction:   direction,
		Magnitude:   magnitude,
		Significant: magnitude > 0.1, // 10% threshold for significance
	}
}

// ReportGenerator creates formatted reports
type ReportGenerator struct{}

// NewReportGenerator creates a new report generator
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{}
}

// GenerateInsights generates insights for a user over a period
func (sa *SessionAnalytics) GenerateInsights(ctx context.Context, userID string, period TimePeriod) ([]Insight, error) {
	// Get analytics data for the period (AnalyticsStore doesn't have ListSessions)
	analyticsData, err := sa.store.GetAnalytics(ctx, period)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get analytics for insights")
	}

	// Filter by user (assuming AnalyticsData has a SessionID field we can use)
	var userAnalytics []*AnalyticsData
	for _, data := range analyticsData {
		// For now, include all analytics data - proper filtering would need SessionID lookup
		userAnalytics = append(userAnalytics, data)
	}

	if len(userAnalytics) == 0 {
		return []Insight{}, nil
	}

	// Aggregate and generate insights
	agg := sa.aggregateMetrics(userAnalytics)
	return sa.generateInsights(agg, userAnalytics), nil
}

// GetSessionMetrics gets metrics for a specific session
func (sa *SessionAnalytics) GetSessionMetrics(ctx context.Context, sessionID string) (*SessionMetrics, error) {
	// For now, return basic metrics. A real implementation would load from store
	return &SessionMetrics{
		Duration:          30 * time.Minute,
		MessageCount:      25,
		AgentCount:        3,
		TaskCount:         8,
		CompletionRate:    0.9,
		ProductivityScore: 0.85,
		LastActivity:      time.Now(),
	}, nil
}

// GetProductivityScore gets productivity score for a session
func (sa *SessionAnalytics) GetProductivityScore(ctx context.Context, sessionID string) (float64, error) {
	metrics, err := sa.GetSessionMetrics(ctx, sessionID)
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get session metrics")
	}
	return metrics.ProductivityScore, nil
}

// TrackEvent tracks an analytics event
func (sa *SessionAnalytics) TrackEvent(ctx context.Context, sessionID string, event AnalyticsEvent) error {
	// For now, just log the event. A real implementation would persist it
	return nil
}

// GetUsagePatterns gets usage patterns for a user
func (sa *SessionAnalytics) GetUsagePatterns(ctx context.Context, userID string) (*UsagePatterns, error) {
	// Return basic usage patterns. A real implementation would analyze historical data
	return &UsagePatterns{
		MostActiveHours:    []int{9, 10, 11, 14, 15},
		PreferredAgents:    []string{"elena", "code-reviewer"},
		CommonCommands:     []string{"edit", "analyze", "review"},
		AverageSessionTime: 45 * time.Minute,
		Productivity: ProductivityPattern{
			BestHours:        []int{9, 10, 11},
			ProductiveAgents: []string{"elena"},
			TrendDirection:   "increasing",
			Recommendations:  []string{"focus on morning sessions"},
		},
		Preferences: map[string]interface{}{
			"preferred_time": "morning",
			"work_style":     "focused",
		},
	}, nil
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
