// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lancekrogers/guild/pkg/project"
)

// completeCampaignIDs provides completion for campaign IDs (lazy filesystem-only approach)
func completeCampaignIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Try lightweight project context detection first
	projCtx, err := project.GetContext()
	if err != nil {
		// No project context, return default suggestions
		return completeDefaultCampaignIDs(toComplete)
	}

	// Use filesystem-only completion (no database/registry initialization)
	return completeCampaignIDsFromFS(projCtx.GetGuildPath(), toComplete)
}

// completeCampaignIDsFromFS provides filesystem-based campaign completion (no database)
func completeCampaignIDsFromFS(guildPath string, toComplete string) ([]string, cobra.ShellCompDirective) {
	campaignsPath := filepath.Join(guildPath, "campaigns")
	pattern := filepath.Join(campaignsPath, "*.json")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		// Fallback to defaults on filesystem error
		return completeDefaultCampaignIDs(toComplete)
	}

	var suggestions []string
	for _, match := range matches {
		base := filepath.Base(match)
		id := strings.TrimSuffix(base, ".json")
		if strings.HasPrefix(id, toComplete) {
			suggestions = append(suggestions, id)
		}
	}

	// If no filesystem matches, provide defaults
	if len(suggestions) == 0 {
		return completeDefaultCampaignIDs(toComplete)
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completeDefaultCampaignIDs provides default campaign ID suggestions
func completeDefaultCampaignIDs(toComplete string) ([]string, cobra.ShellCompDirective) {
	defaults := []string{"guild-demo", "default", "e-commerce", "api-development", "performance", "testing"}

	var suggestions []string
	for _, id := range defaults {
		if strings.HasPrefix(id, toComplete) {
			suggestions = append(suggestions, id)
		}
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completeAgentIDs provides completion for agent IDs (filesystem-based, no registry)
func completeAgentIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Try to get agent IDs from filesystem without initializing registry
	projCtx, err := project.GetContext()
	if err != nil {
		// No project context, provide default suggestions
		return completeDefaultAgents(toComplete)
	}

	// Look for agent YAML files in .campaign/agents/
	agentsPath := filepath.Join(projCtx.GetGuildPath(), "agents")
	pattern := filepath.Join(agentsPath, "*.yaml")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return completeDefaultAgents(toComplete)
	}

	var suggestions []string
	for _, match := range matches {
		base := filepath.Base(match)
		id := strings.TrimSuffix(base, ".yaml")
		if strings.HasPrefix(id, toComplete) {
			suggestions = append(suggestions, id)
		}
	}

	// Also try .yml extension
	pattern = filepath.Join(agentsPath, "*.yml")
	if ymlMatches, err := filepath.Glob(pattern); err == nil {
		for _, match := range ymlMatches {
			base := filepath.Base(match)
			id := strings.TrimSuffix(base, ".yml")
			if strings.HasPrefix(id, toComplete) {
				suggestions = append(suggestions, id)
			}
		}
	}

	// If no filesystem matches, provide defaults
	if len(suggestions) == 0 {
		return completeDefaultAgents(toComplete)
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completeDefaultAgents provides default agent suggestions
func completeDefaultAgents(toComplete string) ([]string, cobra.ShellCompDirective) {
	// Default agents based on common Guild setup
	defaultAgents := map[string]string{
		"manager":    "Project management and task decomposition",
		"backend":    "Backend development and API design",
		"frontend":   "Frontend development and UI/UX",
		"devops":     "Infrastructure and deployment",
		"tester":     "Testing and quality assurance",
		"documenter": "Documentation and technical writing",
	}

	var suggestions []string
	for id, desc := range defaultAgents {
		if strings.HasPrefix(id, toComplete) {
			suggestions = append(suggestions, fmt.Sprintf("%s\t%s", id, desc))
		}
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completeCommissionFiles provides completion for commission markdown files (filesystem-based)
func completeCommissionFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	projCtx, err := project.GetContext()
	if err != nil {
		// Allow default file completion if no project context
		return nil, cobra.ShellCompDirectiveDefault
	}

	var suggestions []string

	// Look in objectives directory
	objectivesPath := filepath.Join(projCtx.GetGuildPath(), "objectives")
	pattern := filepath.Join(objectivesPath, "*.md")

	matches, err := filepath.Glob(pattern)
	if err == nil {
		for _, match := range matches {
			relPath, _ := filepath.Rel(projCtx.GetRootPath(), match)
			if strings.Contains(relPath, toComplete) {
				// Get commission title from file for description
				title := strings.TrimSuffix(filepath.Base(match), ".md")
				title = strings.ReplaceAll(title, "-", " ")
				suggestions = append(suggestions, fmt.Sprintf("%s\t%s", relPath, title))
			}
		}
	}

	// Also look in refined subdirectory
	refinedPath := filepath.Join(objectivesPath, "refined")
	pattern = filepath.Join(refinedPath, "*.md")

	matches, err = filepath.Glob(pattern)
	if err == nil {
		for _, match := range matches {
			relPath, _ := filepath.Rel(projCtx.GetRootPath(), match)
			if strings.Contains(relPath, toComplete) {
				title := strings.TrimSuffix(filepath.Base(match), ".md")
				title = strings.ReplaceAll(title, "-", " ")
				suggestions = append(suggestions, fmt.Sprintf("%s\tRefined: %s", relPath, title))
			}
		}
	}

	// Return suggestions with file completion as fallback
	if len(suggestions) > 0 {
		return suggestions, cobra.ShellCompDirectiveDefault
	}

	// No suggestions, allow default file completion
	return nil, cobra.ShellCompDirectiveDefault
}

// completeSessionIDs provides completion for session IDs
func completeSessionIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// TODO: Implement session history tracking
	// For now, provide example format
	if toComplete == "" {
		return []string{
			"new\tCreate new session",
			"last\tResume last session",
		}, cobra.ShellCompDirectiveNoFileComp
	}

	// Could implement session history from database or logs
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// completeCampaignNames provides completion for campaign names (filesystem-based, no database)
func completeCampaignNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Try to get campaign names from filesystem without database initialization
	projCtx, err := project.GetContext()
	if err != nil {
		return completeDefaultCampaignNames(toComplete)
	}

	// Look for campaign files and try to extract names (simplified approach)
	campaignsPath := filepath.Join(projCtx.GetGuildPath(), "campaigns")
	pattern := filepath.Join(campaignsPath, "*.json")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return completeDefaultCampaignNames(toComplete)
	}

	var suggestions []string
	for _, match := range matches {
		base := filepath.Base(match)
		name := strings.TrimSuffix(base, ".json")
		// Convert filename to readable name (basic transformation)
		name = strings.ReplaceAll(name, "-", " ")
		name = strings.ReplaceAll(name, "_", " ")

		if strings.Contains(strings.ToLower(name), strings.ToLower(toComplete)) {
			suggestions = append(suggestions, name)
		}
	}

	// If no filesystem matches, provide defaults
	if len(suggestions) == 0 {
		return completeDefaultCampaignNames(toComplete)
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completeDefaultCampaignNames provides default campaign name suggestions
func completeDefaultCampaignNames(toComplete string) ([]string, cobra.ShellCompDirective) {
	defaultNames := []string{
		"e-commerce\tE-commerce platform development",
		"api-development\tREST API development",
		"frontend-redesign\tUI/UX redesign project",
		"performance\tPerformance optimization",
		"testing\tTest suite implementation",
		"documentation\tDocumentation updates",
	}

	var suggestions []string
	for _, suggestion := range defaultNames {
		parts := strings.Split(suggestion, "\t")
		if len(parts) > 0 && strings.Contains(strings.ToLower(parts[0]), strings.ToLower(toComplete)) {
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}
