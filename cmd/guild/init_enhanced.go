// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/paths"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/storage"
)

// campaignDirectoryStructure defines the complete campaign directory structure
var campaignDirectoryStructure = []string{
	"agents",
	"guilds", 
	"memory",
	"prompts",
	"tools",
	"workspaces",
}

// createEnhancedCampaignStructure creates the complete .campaign directory tree
func createEnhancedCampaignStructure(ctx context.Context, projectPath string) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("cli").
			WithOperation("createEnhancedCampaignStructure")
	}

	campaignDir := filepath.Join(projectPath, paths.DefaultCampaignDir)

	// Create campaign directory
	if err := os.MkdirAll(campaignDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create campaign directory").
			WithComponent("cli").
			WithOperation("createEnhancedCampaignStructure").
			WithDetails("path", campaignDir)
	}

	// Create subdirectories
	for _, dir := range campaignDirectoryStructure {
		dirPath := filepath.Join(campaignDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create directory").
				WithComponent("cli").
				WithOperation("createEnhancedCampaignStructure").
				WithDetails("dir", dir)
		}
	}

	// Create user-facing directories (outside .campaign)
	userDirs := []string{
		"commissions",
		"commissions/refined", 
		"corpus",
		"corpus/index",
		"kanban",
	}

	for _, dir := range userDirs {
		dirPath := filepath.Join(projectPath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create user directory").
				WithComponent("cli").
				WithOperation("createEnhancedCampaignStructure").
				WithDetails("dir", dir)
		}
	}

	return nil
}

// generateCampaignHash generates a unique campaign hash
func generateCampaignHash(projectPath string) string {
	data := fmt.Sprintf("%s:%d:%s", projectPath, time.Now().UnixNano(), os.Getenv("USER"))
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8]) // 16 character hash
}

// createCampaignConfig creates the campaign.yaml configuration
func createCampaignConfig(ctx context.Context, projectPath, campaignName, projectName, projectType string) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("cli").
			WithOperation("createCampaignConfig")
	}

	campaignHash := generateCampaignHash(projectPath)
	
	// Create campaign configuration
	campaignConfig := map[string]interface{}{
		"campaign": map[string]interface{}{
			"hash":         campaignHash,
			"name":         campaignName,
			"project_name": projectName,
			"project_type": projectType,
			"created_at":   time.Now().Format(time.RFC3339),
			"version":      "1.0.0",
		},
		"daemon": map[string]interface{}{
			"socket_path": fmt.Sprintf("/tmp/guild-%s.sock", campaignHash),
			"log_level":   "info",
		},
		"storage": map[string]interface{}{
			"database": "memory.db",
			"backend":  "sqlite",
		},
		"settings": map[string]interface{}{
			"auto_start_daemon": true,
			"session_timeout":   "24h",
			"max_agents":        10,
		},
	}

	// Write campaign.yaml
	campaignPath := filepath.Join(projectPath, paths.DefaultCampaignDir, "campaign.yaml")
	data, err := yaml.Marshal(campaignConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal campaign config").
			WithComponent("cli").
			WithOperation("createCampaignConfig")
	}

	if err := os.WriteFile(campaignPath, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write campaign config").
			WithComponent("cli").
			WithOperation("createCampaignConfig").
			WithDetails("path", campaignPath)
	}

	// Create .hash file for ultra-fast detection
	hashPath := filepath.Join(projectPath, paths.DefaultCampaignDir, paths.CampaignHashFile)
	if err := os.WriteFile(hashPath, []byte(campaignHash), 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write hash file").
			WithComponent("cli").
			WithOperation("createCampaignConfig").
			WithDetails("path", hashPath)
	}

	return nil
}

// createSocketRegistry creates the socket-registry.yaml file
func createSocketRegistry(ctx context.Context, projectPath, campaignName, campaignHash string) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("cli").
			WithOperation("createSocketRegistry")
	}

	socketRegistry := map[string]interface{}{
		"campaign_hash": campaignHash,
		"campaign_name": campaignName,
		"socket_path":   fmt.Sprintf("/tmp/guild-%s.sock", campaignHash),
		"created_at":    time.Now().Format(time.RFC3339),
		"daemon": map[string]interface{}{
			"status":      "stopped",
			"last_start":  nil,
			"pid":         0,
		},
	}

	registryPath := filepath.Join(projectPath, paths.DefaultCampaignDir, paths.SocketRegistryFile)
	data, err := yaml.Marshal(socketRegistry)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal socket registry").
			WithComponent("cli").
			WithOperation("createSocketRegistry")
	}

	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write socket registry").
			WithComponent("cli").
			WithOperation("createSocketRegistry").
			WithDetails("path", registryPath)
	}

	return nil
}

// createDefaultGuildConfig creates the default guild configuration
func createDefaultGuildConfig(ctx context.Context, projectPath, projectName string) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("cli").
			WithOperation("createDefaultGuildConfig")
	}

	guildConfig := map[string]interface{}{
		"guild": map[string]interface{}{
			"name":        projectName,
			"description": fmt.Sprintf("%s Guild - Orchestrating AI agents for development", projectName),
			"version":     "1.0.0",
			"created_at":  time.Now().Format(time.RFC3339),
		},
		"manager": map[string]interface{}{
			"default": "elena-guild-master",
		},
		"agents": []string{
			"elena-guild-master",
			"marcus-developer", 
			"vera-tester",
		},
		"workflows": map[string]interface{}{
			"default": "collaborative",
			"available": []string{
				"collaborative",
				"sequential",
				"parallel",
			},
		},
		"cost_optimization": map[string]interface{}{
			"enabled":     true,
			"max_cost":    100.0,
			"alert_at":    80.0,
			"currency":    "USD",
		},
	}

	guildPath := filepath.Join(projectPath, paths.DefaultCampaignDir, "guilds", "default-guild.yaml")
	
	// Ensure guilds directory exists
	guildsDir := filepath.Join(projectPath, paths.DefaultCampaignDir, "guilds")
	if err := os.MkdirAll(guildsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create guilds directory").
			WithComponent("cli").
			WithOperation("createDefaultGuildConfig")
	}

	data, err := yaml.Marshal(guildConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal guild config").
			WithComponent("cli").
			WithOperation("createDefaultGuildConfig")
	}

	if err := os.WriteFile(guildPath, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write guild config").
			WithComponent("cli").
			WithOperation("createDefaultGuildConfig").
			WithDetails("path", guildPath)
	}

	return nil
}

// createEnhancedAgentConfigs creates the three default agent configurations
func createEnhancedAgentConfigs(ctx context.Context, projectPath string, projectType *project.ProjectType) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("cli").
			WithOperation("createEnhancedAgentConfigs")
	}

	agentsDir := filepath.Join(projectPath, paths.DefaultCampaignDir, "agents")

	// Create Elena - Guild Master
	elenaConfig := &config.AgentConfig{
		ID:          "elena-guild-master",
		Name:        "Elena",
		Type:        "manager",
		Description: "Guild Master who orchestrates the team and manages projects",
		Provider:    "anthropic",
		Model:       "claude-3-opus-20240229",
		Capabilities: []string{
			"task_decomposition",
			"agent_coordination",
			"strategic_planning",
			"progress_monitoring",
			"commission_refinement",
		},
		Tools: []string{
			"task_planner",
			"agent_coordinator",
			"commission_refiner",
		},
		MaxTokens:   4000,
		Temperature: 0.1,
		Backstory: &config.Backstory{
			Experience: "20 years leading guilds and managing complex projects",
			Expertise:  "Project management, team coordination, strategic planning",
			Philosophy: "Success comes from clear communication and empowering team members",
			GuildRank:  "Master",
		},
		SystemPrompt: `You are Elena, the Guild Master. You coordinate the team of AI agents and ensure projects succeed through careful planning and delegation. You break down complex tasks, assign work to the right agents, and monitor progress. Maintain a professional yet supportive leadership style.`,
	}

	// Create Marcus - Developer
	marcusConfig := &config.AgentConfig{
		ID:          "marcus-developer",
		Name:        "Marcus",
		Type:        "worker",
		Description: "Senior developer who implements features and writes clean code",
		Provider:    "anthropic",
		Model:       "claude-3-sonnet-20240229",
		Capabilities: []string{
			"coding",
			"debugging",
			"refactoring",
			"architecture_design",
			"code_review",
		},
		Tools: []string{
			"code",
			"edit",
			"file",
			"git",
			"shell",
		},
		MaxTokens:   3500,
		Temperature: 0.3,
		Backstory: &config.Backstory{
			Experience: "15 years of software development across multiple languages",
			Expertise:  detectLanguageExpertise(projectType),
			Philosophy: "Code should be clean, maintainable, and well-tested",
			GuildRank:  "Artisan",
		},
		SystemPrompt: fmt.Sprintf(`You are Marcus, a senior developer specializing in %s. You write clean, efficient code following best practices. You focus on implementation quality, proper error handling, and maintainability.`, projectType.Language),
	}

	// Create Vera - Tester
	veraConfig := &config.AgentConfig{
		ID:          "vera-tester",
		Name:        "Vera",
		Type:        "specialist",
		Description: "Quality assurance specialist who ensures code reliability",
		Provider:    "openai",
		Model:       "gpt-4-turbo-preview",
		Capabilities: []string{
			"testing",
			"test_planning",
			"bug_detection",
			"coverage_analysis",
			"performance_testing",
		},
		Tools: []string{
			"code",
			"shell",
			"test_runner",
		},
		MaxTokens:   3000,
		Temperature: 0.2,
		Backstory: &config.Backstory{
			Experience: "12 years in quality assurance and test automation",
			Expertise:  "Test design, automation frameworks, edge case detection",
			Philosophy: "Quality is everyone's responsibility, but someone needs to verify it",
			GuildRank:  "Guardian",
		},
		SystemPrompt: `You are Vera, the quality guardian. You ensure code quality through comprehensive testing, identify edge cases, and help maintain high standards. You write thorough tests and provide constructive feedback.`,
	}

	// Write agent configurations
	agents := []*config.AgentConfig{elenaConfig, marcusConfig, veraConfig}
	
	for _, agent := range agents {
		filename := fmt.Sprintf("%s.yaml", agent.ID)
		filepath := filepath.Join(agentsDir, filename)
		
		data, err := yaml.Marshal(agent)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal agent config").
				WithComponent("cli").
				WithOperation("createEnhancedAgentConfigs").
				WithDetails("agent", agent.ID)
		}

		if err := os.WriteFile(filepath, data, 0644); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write agent config").
				WithComponent("cli").
				WithOperation("createEnhancedAgentConfigs").
				WithDetails("path", filepath)
		}
	}

	return nil
}

// detectLanguageExpertise returns appropriate language expertise based on project type
func detectLanguageExpertise(projectType *project.ProjectType) string {
	switch projectType.Language {
	case "go":
		return "Go, concurrency patterns, error handling, testing"
	case "python":
		return "Python, Django/Flask, async programming, data science libraries"
	case "javascript", "typescript":
		return "JavaScript/TypeScript, React/Vue/Angular, Node.js, modern web development"
	case "rust":
		return "Rust, memory safety, performance optimization, systems programming"
	case "java":
		return "Java, Spring framework, enterprise patterns, JVM optimization"
	default:
		return "Multiple programming languages and frameworks"
	}
}

// initializeCampaignDatabase creates and initializes the SQLite database
func initializeCampaignDatabase(ctx context.Context, projectPath, campaignName string) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("cli").
			WithOperation("initializeCampaignDatabase")
	}

	dbPath := filepath.Join(projectPath, paths.DefaultCampaignDir, paths.DefaultMemoryDB)
	
	// Create database connection
	db, err := storage.DefaultDatabaseFactory(ctx, dbPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create database").
			WithComponent("cli").
			WithOperation("initializeCampaignDatabase")
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to run migrations").
			WithComponent("cli").
			WithOperation("initializeCampaignDatabase")
	}

	// TODO: Insert campaign record when campaign repository is available
	// For now, just ensure database is created and migrated

	// Create initial session
	// TODO: Implement session creation when session package is available

	return nil
}

// adaptAgentConfigsToProjectType adjusts agent configurations based on detected project type
func adaptAgentConfigsToProjectType(ctx context.Context, projectPath string, projectType *project.ProjectType) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("cli").
			WithOperation("adaptAgentConfigsToProjectType")
	}

	agentsDir := filepath.Join(projectPath, paths.DefaultCampaignDir, "agents")
	
	// Update Marcus's configuration based on project type
	marcusPath := filepath.Join(agentsDir, "marcus-developer.yaml")
	
	// Read existing config
	data, err := os.ReadFile(marcusPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read Marcus config").
			WithComponent("cli").
			WithOperation("adaptAgentConfigsToProjectType")
	}

	var marcusConfig config.AgentConfig
	if err := yaml.Unmarshal(data, &marcusConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal Marcus config").
			WithComponent("cli").
			WithOperation("adaptAgentConfigsToProjectType")
	}

	// Add language-specific tools
	switch projectType.Language {
	case "go":
		marcusConfig.Tools = append(marcusConfig.Tools, "go_test", "go_build")
		marcusConfig.Capabilities = append(marcusConfig.Capabilities, "goroutines", "channels")
	case "python":
		marcusConfig.Tools = append(marcusConfig.Tools, "pytest", "pip")
		marcusConfig.Capabilities = append(marcusConfig.Capabilities, "data_analysis", "machine_learning")
	case "javascript", "typescript":
		marcusConfig.Tools = append(marcusConfig.Tools, "npm", "webpack")
		marcusConfig.Capabilities = append(marcusConfig.Capabilities, "frontend", "backend")
	case "rust":
		marcusConfig.Tools = append(marcusConfig.Tools, "cargo", "rustfmt")
		marcusConfig.Capabilities = append(marcusConfig.Capabilities, "memory_safety", "zero_cost_abstractions")
	}

	// Store project info in config (metadata field not available in AgentConfig)
	// This information can be used by the agent through its system prompt

	// Write updated config
	updatedData, err := yaml.Marshal(marcusConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal updated Marcus config").
			WithComponent("cli").
			WithOperation("adaptAgentConfigsToProjectType")
	}

	if err := os.WriteFile(marcusPath, updatedData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write updated Marcus config").
			WithComponent("cli").
			WithOperation("adaptAgentConfigsToProjectType")
	}

	return nil
}