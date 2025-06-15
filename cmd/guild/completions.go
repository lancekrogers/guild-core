package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// completeCampaignIDs provides completion for campaign IDs
func completeCampaignIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := context.Background()

	// Try to get project context
	projCtx, err := project.GetContext()
	if err != nil {
		// No project context, can't complete
		return nil, cobra.ShellCompDirectiveError
	}

	// Try to initialize registry for database access
	reg := registry.NewComponentRegistry()
	if err := reg.Initialize(ctx, registry.Config{}); err != nil {
		// Fall back to file system
		return completeCampaignIDsFromFS(projCtx.GetGuildPath(), toComplete)
	}

	// Get campaign repository
	storageReg := reg.Storage()
	if storageReg == nil {
		return completeCampaignIDsFromFS(projCtx.GetGuildPath(), toComplete)
	}

	campaignRepo := storageReg.GetCampaignRepository()
	if campaignRepo == nil {
		return completeCampaignIDsFromFS(projCtx.GetGuildPath(), toComplete)
	}

	// Get campaigns from repository
	campaigns, err := campaignRepo.ListCampaigns(ctx)
	if err != nil {
		return completeCampaignIDsFromFS(projCtx.GetGuildPath(), toComplete)
	}

	var suggestions []string
	for _, campaign := range campaigns {
		if strings.HasPrefix(campaign.ID, toComplete) {
			// Format: "id\tdescription" for rich completion
			suggestions = append(suggestions, fmt.Sprintf("%s\t%s", campaign.ID, campaign.Name))
		}
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completeCampaignIDsFromFS fallback to file system for campaign IDs
func completeCampaignIDsFromFS(guildPath string, toComplete string) ([]string, cobra.ShellCompDirective) {
	campaignsPath := filepath.Join(guildPath, "campaigns")
	pattern := filepath.Join(campaignsPath, "*.json")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var suggestions []string
	for _, match := range matches {
		base := filepath.Base(match)
		id := strings.TrimSuffix(base, ".json")
		if strings.HasPrefix(id, toComplete) {
			suggestions = append(suggestions, id)
		}
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completeAgentIDs provides completion for agent IDs
func completeAgentIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := context.Background()

	// Initialize registry
	reg := registry.NewComponentRegistry()
	if err := reg.Initialize(ctx, registry.Config{}); err != nil {
		// Provide default agent suggestions
		return completeDefaultAgents(toComplete)
	}

	// Get agent registry
	agentReg := reg.Agents()
	if agentReg == nil {
		return completeDefaultAgents(toComplete)
	}

	// Get available agent types
	agentTypes := agentReg.ListAgentTypes()

	var suggestions []string
	for _, agentType := range agentTypes {
		if strings.HasPrefix(agentType, toComplete) {
			// Try to get agent to see capabilities
			agent, err := agentReg.GetAgent(agentType)
			if err == nil && agent != nil {
				caps := agent.GetCapabilities()
				if len(caps) > 0 {
					// Show first few capabilities
					capStr := strings.Join(caps[:min(3, len(caps))], ", ")
					if len(caps) > 3 {
						capStr += "..."
					}
					suggestions = append(suggestions, fmt.Sprintf("%s\t%s", agentType, capStr))
				} else {
					suggestions = append(suggestions, agentType)
				}
			} else {
				suggestions = append(suggestions, agentType)
			}
		}
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

// completeCommissionFiles provides completion for commission markdown files
func completeCommissionFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	projCtx, err := project.GetContext()
	if err != nil {
		// Allow default file completion
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

// completeCampaignNames provides completion for campaign names (not IDs)
func completeCampaignNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := context.Background()

	// Try to get project context
	_, err := project.GetContext()
	if err != nil {
		return completeDefaultCampaignNames(toComplete)
	}

	// Initialize registry
	reg := registry.NewComponentRegistry()
	if err := reg.Initialize(ctx, registry.Config{}); err != nil {
		return completeDefaultCampaignNames(toComplete)
	}

	// Get campaign repository
	storageReg := reg.Storage()
	if storageReg == nil {
		return completeDefaultCampaignNames(toComplete)
	}

	campaignRepo := storageReg.GetCampaignRepository()
	if campaignRepo == nil {
		return completeDefaultCampaignNames(toComplete)
	}

	// Get campaigns
	campaigns, err := campaignRepo.ListCampaigns(ctx)
	if err != nil {
		return completeDefaultCampaignNames(toComplete)
	}

	var suggestions []string
	for _, campaign := range campaigns {
		if strings.Contains(strings.ToLower(campaign.Name), strings.ToLower(toComplete)) {
			// Campaign struct doesn't have Description, use Status instead
			suggestions = append(suggestions, fmt.Sprintf("%s\tStatus: %s", campaign.Name, campaign.Status))
		}
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