// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// completeCampaignIDs provides completion for campaign IDs
func completeCampaignIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// IMPORTANT: Don't do heavy initialization in completion functions
	// Just try filesystem-based completion
	
	// TEMPORARY: Skip project context detection to avoid filesystem scanning
	// Just return common defaults
	return []string{"guild-demo", "default", "e-commerce", "api-development"}, cobra.ShellCompDirectiveNoFileComp
}

// Note: Removed filesystem-based completion to avoid heavy operations in shell completion

// completeAgentIDs provides completion for agent IDs
func completeAgentIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// IMPORTANT: Don't initialize registry in completion functions
	// Just provide default suggestions
	return completeDefaultAgents(toComplete)
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

// completeCommissionFiles provides completion for commission markdown files
func completeCommissionFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// IMPORTANT: Don't call project.GetContext() in completion functions
	// Just allow default file completion
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

// completeCampaignNames provides completion for campaign names (not IDs)
func completeCampaignNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// IMPORTANT: Don't initialize registry in completion functions
	// Just provide default suggestions
	return completeDefaultCampaignNames(toComplete)
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
