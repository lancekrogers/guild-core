// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package init

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/agents"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

// doInitialization performs the main initialization with proper context handling
func (m *InitTUIModelV2) doInitialization() tea.Cmd {
	return func() tea.Msg {
		// Create a sub-context with timeout for the entire operation
		ctx, cancel := context.WithTimeout(m.ctx, 5*time.Minute)
		defer cancel()

		// Check context at start
		if err := ctx.Err(); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeCancelled, "initialization cancelled").
				WithComponent("InitTUIV2").
				WithOperation("doInitialization")}
		}

		// Step 1: Check existing campaign
		if err := m.checkExistingCampaign(ctx); err != nil {
			return errMsg{err: err}
		}

		// Report progress
		select {
		case <-ctx.Done():
			return errMsg{err: gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "cancelled during campaign check")}
		default:
			// Continue
		}

		// Step 2: Initialize project structure
		if !m.projectInit.IsProjectInitialized(m.config.ProjectPath) {
			if err := m.projectInit.InitializeProject(ctx, m.config.ProjectPath); err != nil {
				return errMsg{err: gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize project").
					WithComponent("InitTUIV2").
					WithOperation("doInitialization").
					WithDetails("path", m.config.ProjectPath)}
			}
		}

		// Check context after I/O operation
		if err := ctx.Err(); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeCancelled, "cancelled after project init")}
		}

		// Step 3: Detect available providers
		if err := m.detectProviders(ctx); err != nil {
			// Provider detection failure is not fatal, continue with defaults
			return warnMsg{message: fmt.Sprintf("Provider detection warning: %v", err)}
		}

		// Check context after provider detection
		if err := ctx.Err(); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeCancelled, "cancelled after provider detection")}
		}

		// Step 4: Create Phase 0 configuration with enhanced agents
		if err := m.configManager.EstablishGuildFoundation(ctx, m.config.ProjectPath, m.campaignName, m.projectName); err != nil {
			return errMsg{err: err}
		}

		// Step 5: Create Elena and specialist agents
		if err := m.createEnhancedAgents(ctx); err != nil {
			// Enhanced agent creation failure is not fatal, continue with defaults
			return warnMsg{message: fmt.Sprintf("Enhanced agent creation warning: %v", err)}
		}

		// Check context after configuration
		if err := ctx.Err(); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeCancelled, "cancelled after config creation")}
		}

		// Complete initialization with enhanced messaging
		agentCount := m.enhancedAgentCount
		if agentCount == 0 {
			agentCount = 3 // Default expected count
		}

		providerCount := 0
		for _, result := range m.providerResults {
			if result.Available {
				providerCount++
			}
		}

		completeMessage := fmt.Sprintf("🏰 Guild established! Elena + %d artisans ready with %d AI providers",
			agentCount-1, providerCount) // -1 because Elena is separate from other artisans

		return initProgressMsg{
			phase:   "complete",
			percent: 1.0,
			message: completeMessage,
		}
	}
}

// createDemoCommission creates a demo with proper error handling
func (m *InitTUIModelV2) createDemoCommission() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
		defer cancel()

		// Create commission directory
		commissionsDir := filepath.Join(m.config.ProjectPath, ".campaign", "commissions")
		if err := os.MkdirAll(commissionsDir, 0755); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create commissions directory").
				WithComponent("InitTUIV2").
				WithOperation("createDemoCommission").
				WithDetails("dir", commissionsDir)}
		}

		// Check context after I/O
		if err := ctx.Err(); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeCancelled, "cancelled during demo creation")}
		}

		// Generate commission content
		content, err := m.demoGen.GenerateCommission(ctx, m.demoType)
		if err != nil {
			// Don't fail entire init for demo
			return warnMsg{message: fmt.Sprintf("Could not generate demo commission: %v", err)}
		}

		// Write commission file
		fileName := fmt.Sprintf("demo-%s.md", string(m.demoType))
		commissionPath := filepath.Join(commissionsDir, fileName)

		// Write with context awareness
		if err := writeFileWithContext(ctx, commissionPath, []byte(content), 0644); err != nil {
			return warnMsg{message: fmt.Sprintf("Could not save demo commission: %v", err)}
		}

		return successMsg{message: fmt.Sprintf("Created demo commission: %s", fileName)}
	}
}

// doValidation performs validation with proper context handling
func (m *InitTUIModelV2) doValidation() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 2*time.Minute)
		defer cancel()

		// Phase 0 integration
		if err := m.configManager.FinalizeGuildCharter(ctx, m.config.ProjectPath, m.campaignName, m.projectName); err != nil {
			return errMsg{err: err}
		}

		// Check context between operations
		if err := ctx.Err(); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeCancelled, "cancelled during config integration")}
		}

		// Campaign reference already created in CreatePhase0Configuration

		// Socket registry
		if err := m.daemonManager.SaveSocketRegistry(m.config.ProjectPath, m.campaignName); err != nil {
			// Non-fatal but log it
			return warnMsg{message: fmt.Sprintf("Could not save socket registry: %v", err)}
		}

		// Run validation
		if err := m.validator.Validate(ctx); err != nil {
			// Validation errors are not fatal
			results := m.validator.GetResults()
			return validationResultsMsg{
				results: results,
				failed:  true,
			}
		}

		return validationResultsMsg{
			results: m.validator.GetResults(),
			failed:  false,
		}
	}
}

// Helper functions

func (m *InitTUIModelV2) checkExistingCampaign(ctx context.Context) error {
	// This would check for existing campaign
	// For now, just a placeholder
	return nil
}

// detectProviders detects available AI providers for optimal configuration
func (m *InitTUIModelV2) detectProviders(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "provider detection cancelled").
			WithComponent("InitTUIV2").
			WithOperation("detectProviders")
	}

	// Create auto-detector with reasonable timeout
	detector := providers.NewAutoDetector(10 * time.Second)

	// Detect all available providers
	results, err := detector.DetectAll(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeProvider, "failed to detect providers").
			WithComponent("InitTUIV2").
			WithOperation("detectProviders")
	}

	// Store detection results for later use
	m.providerResults = results

	// Enhanced provider analysis for intelligent agent mapping
	availableCount := 0
	var bestProvider providers.DetectionResult
	highestConfidence := 0.0

	for _, result := range results {
		if result.Available {
			availableCount++
			if result.Confidence > highestConfidence {
				highestConfidence = result.Confidence
				bestProvider = result
			}
		}
	}

	// Store best provider for agent optimization
	if availableCount > 0 {
		m.bestProvider = &bestProvider
	}

	// Provider detection failure is not fatal - we'll use defaults
	if availableCount == 0 {
		// Log a warning but continue - the system can work with manual configuration
		return nil // Don't fail initialization, just proceed with defaults
	}

	return nil
}

// createEnhancedAgents creates Elena and specialist agents with rich backstories
func (m *InitTUIModelV2) createEnhancedAgents(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "enhanced agent creation cancelled").
			WithComponent("InitTUIV2").
			WithOperation("createEnhancedAgents")
	}

	// Create the enhanced agent creator
	creator := agents.NewDefaultAgentCreator()

	// Create the default agent set (Elena + specialists)
	agentConfigs, err := creator.CreateDefaultAgentSet(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create enhanced agent set").
			WithComponent("InitTUIV2").
			WithOperation("createEnhancedAgents")
	}

	// Optimize agent providers based on detection results
	m.optimizeAgentProviders(agentConfigs)

	// Ensure agents directory exists
	agentsDir := filepath.Join(m.config.ProjectPath, ".campaign", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create agents directory").
			WithComponent("InitTUIV2").
			WithOperation("createEnhancedAgents").
			WithDetails("dir", agentsDir)
	}

	// Save each agent configuration
	for _, agentConfig := range agentConfigs {
		if err := m.saveAgentConfig(ctx, agentsDir, agentConfig); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save agent configuration").
				WithComponent("InitTUIV2").
				WithOperation("createEnhancedAgents").
				WithDetails("agent", agentConfig.Name)
		}
	}

	// Store agent count for success messaging
	m.enhancedAgentCount = len(agentConfigs)

	return nil
}

// optimizeAgentProviders intelligently maps agents to detected providers
func (m *InitTUIModelV2) optimizeAgentProviders(agentConfigs []*config.AgentConfig) {
	// If no providers detected, use defaults
	if len(m.providerResults) == 0 {
		return
	}

	// Create provider availability map
	availableProviders := make(map[string]providers.DetectionResult)
	for _, result := range m.providerResults {
		if result.Available {
			availableProviders[string(result.Provider)] = result
		}
	}

	// If no providers available, use defaults
	if len(availableProviders) == 0 {
		return
	}

	// Intelligent provider mapping for each agent
	for _, agent := range agentConfigs {
		originalProvider := agent.Provider

		// Try to find the best provider for this agent
		switch agent.Type {
		case "manager":
			// Elena (manager) benefits most from Claude Code's planning capabilities
			if claudeCode, exists := availableProviders["claude_code"]; exists && claudeCode.Confidence > 0.7 {
				agent.Provider = "claude_code"
			} else if _, exists := availableProviders["anthropic"]; exists {
				agent.Provider = "anthropic"
			}
		case "worker":
			// Marcus (developer) benefits from Claude Code's coding features
			if agent.ID == "marcus-developer" {
				if claudeCode, exists := availableProviders["claude_code"]; exists && claudeCode.Confidence > 0.7 {
					agent.Provider = "claude_code"
				} else if _, exists := availableProviders["anthropic"]; exists {
					agent.Provider = "anthropic"
				}
			}
		case "specialist":
			// Vera (tester) can use any high-quality provider
			if _, exists := availableProviders["anthropic"]; exists {
				agent.Provider = "anthropic"
			} else if _, exists := availableProviders["claude_code"]; exists {
				agent.Provider = "claude_code"
			}
		}

		// If best provider is unavailable, fall back to any available provider
		if _, exists := availableProviders[agent.Provider]; !exists {
			// Use the provider with highest confidence as fallback
			if m.bestProvider != nil {
				agent.Provider = string(m.bestProvider.Provider)
			} else {
				// Restore original if no good options
				agent.Provider = originalProvider
			}
		}
	}
}

// saveAgentConfig saves an agent configuration to disk
func (m *InitTUIModelV2) saveAgentConfig(ctx context.Context, agentsDir string, agentConfig *config.AgentConfig) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "agent config save cancelled").
			WithComponent("InitTUIV2").
			WithOperation("saveAgentConfig")
	}

	// Convert to YAML-friendly format
	configData := map[string]interface{}{
		"id":           agentConfig.ID,
		"name":         agentConfig.Name,
		"type":         agentConfig.Type,
		"description":  agentConfig.Description,
		"provider":     agentConfig.Provider,
		"model":        agentConfig.Model,
		"capabilities": agentConfig.Capabilities,
		"tools":        agentConfig.Tools,
	}
	
	// Add context window if set
	if agentConfig.ContextWindow > 0 {
		configData["context_window"] = agentConfig.ContextWindow
	}
	
	// Add cost magnitude if set
	if agentConfig.CostMagnitude > 0 {
		configData["cost_magnitude"] = agentConfig.CostMagnitude
	}
	
	// Add context reset if set
	if agentConfig.ContextReset != "" {
		configData["context_reset"] = agentConfig.ContextReset
	}

	// Add backstory information if available
	if agentConfig.Backstory != nil {
		configData["backstory"] = map[string]interface{}{
			"experience":     agentConfig.Backstory.Experience,
			"previous_roles": agentConfig.Backstory.PreviousRoles,
			"expertise":      agentConfig.Backstory.Expertise,
			"achievements":   agentConfig.Backstory.Achievements,
			"philosophy":     agentConfig.Backstory.Philosophy,
			"guild_rank":     agentConfig.Backstory.GuildRank,
			"specialties":    agentConfig.Backstory.Specialties,
		}
	}

	// Add personality information if available
	if agentConfig.Personality != nil {
		configData["personality"] = map[string]interface{}{
			"formality":      agentConfig.Personality.Formality,
			"detail_level":   agentConfig.Personality.DetailLevel,
			"humor_level":    agentConfig.Personality.HumorLevel,
			"approach_style": agentConfig.Personality.ApproachStyle,
			"assertiveness":  agentConfig.Personality.Assertiveness,
			"empathy":        agentConfig.Personality.Empathy,
			"patience":       agentConfig.Personality.Patience,
			"honor":          agentConfig.Personality.Honor,
			"wisdom":         agentConfig.Personality.Wisdom,
			"craftsmanship":  agentConfig.Personality.Craftsmanship,
		}
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(configData)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal agent config").
			WithComponent("InitTUIV2").
			WithOperation("saveAgentConfig").
			WithDetails("agent", agentConfig.Name)
	}

	// Save to file
	filename := fmt.Sprintf("%s.yaml", agentConfig.ID)
	filepath := filepath.Join(agentsDir, filename)

	if err := os.WriteFile(filepath, yamlData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write agent config file").
			WithComponent("InitTUIV2").
			WithOperation("saveAgentConfig").
			WithDetails("path", filepath)
	}

	return nil
}

// writeFileWithContext writes a file with context cancellation support
func writeFileWithContext(ctx context.Context, path string, data []byte, perm os.FileMode) error {
	// Create a channel to signal completion
	done := make(chan error, 1)

	go func() {
		done <- os.WriteFile(path, data, perm)
	}()

	select {
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "file write cancelled").
			WithDetails("path", path)
	case err := <-done:
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write file").
				WithDetails("path", path)
		}
		return nil
	}
}

// Message types with better structure

type initProgressMsg struct {
	phase   string
	percent float64
	message string
}

type successMsg struct {
	message string
}

type warnMsg struct {
	message string
}

type errMsg struct {
	err error
}

type validationResultsMsg struct {
	results []ValidationResult
	failed  bool
}
