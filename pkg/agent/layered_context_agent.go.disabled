package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	guildcontext "github.com/guild-ventures/guild-core/pkg/context"
	"github.com/guild-ventures/guild-core/pkg/prompts"
)

// LayeredContextAgent extends ContextAwareAgent with Guild layered prompt system
type LayeredContextAgent struct {
	*ContextAwareAgent // Embed the base agent
	
	// Layered prompt system components
	promptManager prompts.LayeredManager
	sessionID     string
	promptVersion int
	
	// Guild-specific metadata
	role     string
	domain   string
	guildID  string
}

// NewLayeredContextAgent creates a new Guild artisan with layered prompt support
func NewLayeredContextAgent(
	id, name, agentType string,
	capabilities []string,
	role, domain string,
	promptManager prompts.LayeredManager,
) *LayeredContextAgent {
	baseAgent := NewContextAwareAgent(id, name, agentType, capabilities)
	
	return &LayeredContextAgent{
		ContextAwareAgent: baseAgent,
		promptManager:     promptManager,
		role:             role,
		domain:           domain,
		sessionID:        generateSessionID(id),
	}
}

// Execute runs a task using the Guild layered prompt system
func (lca *LayeredContextAgent) Execute(ctx context.Context, request string) (string, error) {
	startTime := time.Now()
	
	// Update agent status
	lca.status.State = "busy"
	lca.status.CurrentTask = request
	lca.status.LastActive = time.Now()
	lca.taskCount++
	
	// Create enhanced Guild context for this execution
	execCtx := guildcontext.CreateComponentContext(ctx, "artisan", lca.ID, "execute")
	execCtx = guildcontext.WithPromptVersion(execCtx, lca.promptVersion)
	
	// Log the execution start with Guild terminology
	if logger := guildcontext.GetLogger(execCtx); logger != nil {
		logger.Info("Guild artisan executing commission task", append(
			guildcontext.LogFields(execCtx),
			"artisan_role", lca.role,
			"domain", lca.domain,
			"session_id", lca.sessionID,
		)...)
	}
	
	// Execute using layered prompts
	result, err := lca.executeWithLayeredPrompt(execCtx, request)
	
	// Update status and metrics
	duration := time.Since(startTime)
	lca.status.LastActive = time.Now()
	
	if err != nil {
		lca.status.State = "error"
		lca.errorCount++
		
		// Log error with Guild context
		if logger := guildcontext.GetLogger(execCtx); logger != nil {
			logger.Error("Guild artisan task failed", append(
				guildcontext.LogFields(execCtx),
				"error", err.Error(),
				"duration_ms", duration.Milliseconds(),
				"artisan_role", lca.role,
			)...)
		}
	} else {
		lca.status.State = "idle"
		lca.status.SuccessCount++
		
		// Log success with Guild context
		if logger := guildcontext.GetLogger(execCtx); logger != nil {
			logger.Info("Guild artisan task completed successfully", append(
				guildcontext.LogFields(execCtx),
				"duration_ms", duration.Milliseconds(),
				"artisan_role", lca.role,
				"result_length", len(result),
			)...)
		}
	}
	
	// Update average latency
	if lca.taskCount > 0 {
		totalLatency := lca.status.AverageLatency * time.Duration(lca.taskCount-1)
		lca.status.AverageLatency = (totalLatency + duration) / time.Duration(lca.taskCount)
	}
	
	return result, err
}

// executeWithLayeredPrompt performs execution using Guild's layered prompt system
func (lca *LayeredContextAgent) executeWithLayeredPrompt(ctx context.Context, request string) (string, error) {
	// Determine the best provider for this request
	providerName, err := lca.selectProvider(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to select provider: %w", err)
	}
	
	// Enhance context with provider information
	ctx = guildcontext.WithProvider(ctx, providerName)
	
	// Create layered prompt using Guild's prompt system
	layeredPrompt, err := lca.buildLayeredPrompt(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to build layered prompt: %w", err)
	}
	
	// Log prompt metrics for monitoring
	if logger := guildcontext.GetLogger(ctx); logger != nil {
		logger.Debug("Guild layered prompt assembled", 
			"layer_count", len(layeredPrompt.Layers),
			"token_count", layeredPrompt.TokenCount,
			"truncated", layeredPrompt.Truncated,
			"cached", layeredPrompt.CacheKey != "",
		)
	}
	
	// Execute with the selected provider using the compiled prompt
	result, err := guildcontext.CompleteWithProvider(ctx, providerName, layeredPrompt.Compiled)
	if err != nil {
		return "", fmt.Errorf("provider execution failed: %w", err)
	}
	
	// Post-process the result if needed
	processedResult := lca.postProcessResult(ctx, request, result)
	
	// Update prompt metrics
	lca.updatePromptMetrics(layeredPrompt, time.Since(time.Now()), err == nil)
	
	return processedResult, nil
}

// buildLayeredPrompt creates a Guild layered prompt for the current request
func (lca *LayeredContextAgent) buildLayeredPrompt(ctx context.Context, request string) (*prompts.LayeredPrompt, error) {
	// Extract task context if available
	var taskContext prompts.Context
	if taskID := guildcontext.GetCurrentTask(ctx); taskID != "" {
		// TODO: Retrieve task context from Workshop Board
		taskContext = &basicTaskContext{
			taskID:       taskID,
			commissionID: guildcontext.GetCommissionID(ctx),
		}
	}
	
	// Create turn context for this specific request
	turnCtx := prompts.TurnContext{
		UserMessage:  request,
		TaskID:       guildcontext.GetCurrentTask(ctx),
		CommissionID: guildcontext.GetCommissionID(ctx),
		Urgency:      lca.determineUrgency(request),
		Instructions: lca.extractInstructions(request),
		Context:      taskContext,
		Metadata: map[string]interface{}{
			"agent_type":     lca.AgentType,
			"capabilities":   lca.Capabilities,
			"request_length": len(request),
			"timestamp":      time.Now(),
		},
	}
	
	// Build the layered prompt using the Guild prompt manager
	return lca.promptManager.BuildLayeredPrompt(ctx, lca.ID, lca.sessionID, turnCtx)
}

// SetSessionPreferences allows setting session-level prompt preferences
func (lca *LayeredContextAgent) SetSessionPreferences(ctx context.Context, preferences string) error {
	sessionPrompt := prompts.SystemPrompt{
		Layer:     prompts.LayerSession,
		SessionID: lca.sessionID,
		Content:   preferences,
		Version:   1,
		Updated:   time.Now(),
		Metadata: map[string]interface{}{
			"source":      "user_preferences",
			"artisan_id":  lca.ID,
			"set_at":      time.Now(),
		},
	}
	
	if err := lca.promptManager.SetPromptLayer(ctx, sessionPrompt); err != nil {
		return fmt.Errorf("failed to set session preferences: %w", err)
	}
	
	// Invalidate cache to pick up new preferences
	return lca.promptManager.InvalidateCache(ctx, lca.ID, lca.sessionID)
}

// UpdateGuildPrompt allows updating guild-wide prompts (requires appropriate permissions)
func (lca *LayeredContextAgent) UpdateGuildPrompt(ctx context.Context, guildPrompt string) error {
	if lca.guildID == "" {
		return fmt.Errorf("no guild ID associated with this artisan")
	}
	
	prompt := prompts.SystemPrompt{
		Layer:   prompts.LayerGuild,
		Content: guildPrompt,
		Version: 1,
		Updated: time.Now(),
		Metadata: map[string]interface{}{
			"source":      "artisan_update",
			"artisan_id":  lca.ID,
			"guild_id":    lca.guildID,
			"updated_by":  lca.Name,
		},
	}
	
	return lca.promptManager.SetPromptLayer(ctx, prompt)
}

// GetActivePromptLayers returns the currently active prompt layers for inspection
func (lca *LayeredContextAgent) GetActivePromptLayers(ctx context.Context) ([]prompts.SystemPrompt, error) {
	return lca.promptManager.ListPromptLayers(ctx, lca.ID, lca.sessionID)
}

// GetPromptMetrics returns performance metrics for this artisan's prompt usage
func (lca *LayeredContextAgent) GetPromptMetrics(ctx context.Context) (*prompts.PromptMetrics, error) {
	// Delegate to the prompt manager
	return lca.promptManager.GetMetrics(ctx)
}

// SetRole updates the artisan's role and invalidates relevant caches
func (lca *LayeredContextAgent) SetRole(ctx context.Context, newRole string) error {
	oldRole := lca.role
	lca.role = newRole
	
	// Invalidate cache since role change affects prompt assembly
	if err := lca.promptManager.InvalidateCache(ctx, lca.ID, lca.sessionID); err != nil {
		lca.role = oldRole // Rollback on error
		return fmt.Errorf("failed to invalidate cache after role change: %w", err)
	}
	
	return nil
}

// SetDomain updates the artisan's domain specialization
func (lca *LayeredContextAgent) SetDomain(ctx context.Context, newDomain string) error {
	oldDomain := lca.domain
	lca.domain = newDomain
	
	// Invalidate cache since domain change affects prompt assembly
	if err := lca.promptManager.InvalidateCache(ctx, lca.ID, lca.sessionID); err != nil {
		lca.domain = oldDomain // Rollback on error
		return fmt.Errorf("failed to invalidate cache after domain change: %w", err)
	}
	
	return nil
}

// Helper methods

// determineUrgency analyzes the request to determine urgency level
func (lca *LayeredContextAgent) determineUrgency(request string) string {
	urgencyKeywords := map[string]string{
		"urgent":     "high",
		"asap":       "high",
		"emergency":  "high",
		"critical":   "high",
		"immediate":  "high",
		"quickly":    "medium",
		"soon":       "medium",
		"when possible": "low",
		"eventually": "low",
	}
	
	requestLower := strings.ToLower(request)
	for keyword, urgency := range urgencyKeywords {
		if strings.Contains(requestLower, keyword) {
			return urgency
		}
	}
	
	// Default urgency based on request length and complexity
	if len(request) > 500 {
		return "medium" // Longer requests often need more careful consideration
	}
	
	return "normal"
}

// extractInstructions pulls out specific instructions from the request
func (lca *LayeredContextAgent) extractInstructions(request string) []string {
	var instructions []string
	
	// Look for explicit instruction patterns
	patterns := []string{
		"make sure to",
		"ensure that",
		"remember to",
		"don't forget to",
		"be careful to",
		"please",
	}
	
	requestLower := strings.ToLower(request)
	for _, pattern := range patterns {
		if strings.Contains(requestLower, pattern) {
			// Extract the instruction (simplified - could be more sophisticated)
			instructions = append(instructions, fmt.Sprintf("Request contains instruction: %s", pattern))
		}
	}
	
	return instructions
}

// updatePromptMetrics tracks prompt performance for optimization
func (lca *LayeredContextAgent) updatePromptMetrics(
	prompt *prompts.LayeredPrompt,
	duration time.Duration,
	success bool,
) {
	// TODO: Implement comprehensive metrics tracking
	// This would integrate with the metrics system
}

// generateSessionID creates a session ID for this artisan
func generateSessionID(artisanID string) string {
	// Simple session ID generation - in production this might be more sophisticated
	return fmt.Sprintf("session_%s_%d", artisanID, time.Now().Unix())
}

// basicTaskContext implements the prompts.Context interface for basic task information
type basicTaskContext struct {
	taskID       string
	commissionID string
}

func (btc *basicTaskContext) GetCommissionID() string {
	return btc.commissionID
}

func (btc *basicTaskContext) GetCommissionTitle() string {
	return ""
}

func (btc *basicTaskContext) GetCurrentTask() prompts.TaskContext {
	return prompts.TaskContext{
		ID: btc.taskID,
	}
}

func (btc *basicTaskContext) GetRelevantSections() []prompts.Section {
	return nil
}

func (btc *basicTaskContext) GetRelatedTasks() []prompts.TaskContext {
	return nil
}

// GetRole returns the artisan's role
func (lca *LayeredContextAgent) GetRole() string {
	return lca.role
}

// GetDomain returns the artisan's domain specialization  
func (lca *LayeredContextAgent) GetDomain() string {
	return lca.domain
}

// GetSessionID returns the current session ID
func (lca *LayeredContextAgent) GetSessionID() string {
	return lca.sessionID
}

// GetGuildID returns the artisan's guild ID
func (lca *LayeredContextAgent) GetGuildID() string {
	return lca.guildID
}

// SetGuildID assigns the artisan to a guild
func (lca *LayeredContextAgent) SetGuildID(guildID string) {
	lca.guildID = guildID
}