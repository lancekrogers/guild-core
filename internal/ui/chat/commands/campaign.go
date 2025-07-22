// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// CampaignCommand handles /campaign commands
type CampaignCommand struct {
	// For now, we'll create a simplified version that demonstrates the UI
	// In production, these would be injected from the registry
	agentRouter AgentRouter
}

// NewCampaignCommand creates a new campaign command handler
func NewCampaignCommand(agentRouter AgentRouter) *CampaignCommand {
	return &CampaignCommand{
		agentRouter: agentRouter,
	}
}

// Handle implements the CommandHandler interface
func (c *CampaignCommand) Handle(ctx context.Context, args []string) tea.Cmd {
	return c.Execute(ctx, args)
}

// Description implements the CommandHandler interface
func (c *CampaignCommand) Description() string {
	return "Manage multi-agent campaigns"
}

// Usage implements the CommandHandler interface
func (c *CampaignCommand) Usage() string {
	return "/campaign [start|list|status|stop|help]"
}

// Execute handles the campaign command
func (c *CampaignCommand) Execute(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return c.showHelp()
	}

	switch args[0] {
	case "start":
		if len(args) < 2 {
			return c.errorResult("Please provide a commission title or ID")
		}
		return c.startCampaign(ctx, strings.Join(args[1:], " "))
	case "list":
		return c.listCampaigns(ctx)
	case "status":
		if len(args) < 2 {
			return c.showActiveCampaignStatus(ctx)
		}
		return c.showCampaignStatus(ctx, args[1])
	case "stop":
		if len(args) < 2 {
			return c.errorResult("Please provide a campaign ID")
		}
		return c.stopCampaign(ctx, args[1])
	case "help":
		return c.showHelp()
	default:
		return c.errorResult(fmt.Sprintf("Unknown subcommand: %s", args[0]))
	}
}

// CampaignCommandResult represents the result of a campaign command
type CampaignCommandResult struct {
	Type    string
	Message string
	Data    interface{}
	Error   error
}

// startCampaign starts a new campaign for a commission
func (c *CampaignCommand) startCampaign(ctx context.Context, commissionRef string) tea.Cmd {
	return func() tea.Msg {
		// For now, demonstrate the UI flow
		// In production, this would integrate with the actual campaign manager
		
		// Simulate starting a campaign
		campaignID := fmt.Sprintf("campaign-%d", time.Now().Unix())
		
		// Send message to agents about the campaign
		if c.agentRouter != nil {
			message := fmt.Sprintf("Starting multi-agent campaign for: %s", commissionRef)
			// Execute the broadcast command
			if cmd := c.agentRouter.BroadcastToAll(message); cmd != nil {
				// In a real implementation, we would handle the broadcast result
				// For now, we'll just note that the broadcast was initiated
			}
		}

		return CampaignCommandResult{
			Type: "started",
			Message: fmt.Sprintf(
				"🚀 **Campaign Started!**\n\n"+
				"**Commission:** %s\n"+
				"**Campaign ID:** %s\n\n"+
				"Multiple agents are now working on your commission.\n"+
				"Use `/campaign status` to monitor progress.\n\n"+
				"*Note: This is a preview of the campaign feature.*",
				commissionRef,
				campaignID,
			),
			Data: map[string]string{
				"campaign_id": campaignID,
				"commission":  commissionRef,
			},
		}
	}
}

// listCampaigns lists all campaigns
func (c *CampaignCommand) listCampaigns(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		// For now, return a preview message
		// In production, this would query the campaign manager through the registry
		return CampaignCommandResult{
			Type: "list",
			Message: "📊 **Campaign System Preview**\n\n" +
				"The campaign system coordinates multiple agents to work on your commissions.\n\n" +
				"**Features coming soon:**\n" +
				"- Multi-agent task decomposition\n" +
				"- Parallel task execution\n" +
				"- Real-time progress tracking\n" +
				"- Agent collaboration\n\n" +
				"Use `/campaign start <commission>` to see a preview of how campaigns will work.",
		}
	}
}

// showActiveCampaignStatus shows the status of the active campaign
func (c *CampaignCommand) showActiveCampaignStatus(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		// For now, return a preview message
		return CampaignCommandResult{
			Type:    "status",
			Message: "No active campaign found. Use `/campaign start <commission>` to start one.",
		}
	}
}

// showCampaignStatus shows the status of a specific campaign
func (c *CampaignCommand) showCampaignStatus(ctx context.Context, campaignID string) tea.Cmd {
	return func() tea.Msg {
		// For now, return a preview message
		return CampaignCommandResult{
			Type: "status",
			Message: fmt.Sprintf(`📊 **Campaign Status Preview**

**Campaign ID:** %s

This will show:
- Real-time task progress
- Agent assignments
- Completion percentage
- Task dependencies

Campaign status tracking is coming soon!`, campaignID),
		}
	}
}

// stopCampaign stops a campaign
func (c *CampaignCommand) stopCampaign(ctx context.Context, campaignID string) tea.Cmd {
	return func() tea.Msg {
		// For now, return a preview message
		return CampaignCommandResult{
			Type:    "stopped",
			Message: fmt.Sprintf("Campaign %s stop preview - campaigns can be stopped once the system is fully integrated.", campaignID),
		}
	}
}

// showHelp shows help for campaign commands
func (c *CampaignCommand) showHelp() tea.Cmd {
	return func() tea.Msg {
		helpText := `🚀 **Campaign Commands:**

• **/campaign start <commission>** - Start a multi-agent campaign for a commission
• **/campaign list** - Show all campaigns
• **/campaign status [id]** - Show campaign progress (defaults to active campaign)
• **/campaign stop <id>** - Stop a running campaign
• **/campaign help** - Show this help message

**What are Campaigns?**
Campaigns coordinate multiple AI agents to work on your commission in parallel.
Agents automatically divide work, collaborate, and deliver results faster.

**Example:**
/campaign start "Build a TODO API"
This will start multiple agents working on different aspects of your API.`

		return CampaignCommandResult{
			Type:    "help",
			Message: helpText,
		}
	}
}

// errorResult creates an error result
func (c *CampaignCommand) errorResult(message string) tea.Cmd {
	return func() tea.Msg {
		return CampaignCommandResult{
			Type:    "error",
			Message: message,
		}
	}
}

// GetCompletions returns tab completions for the campaign command
func (c *CampaignCommand) GetCompletions(ctx context.Context, args []string) []string {
	if len(args) == 0 {
		return []string{"start", "list", "status", "stop", "help"}
	}

	if len(args) == 1 {
		// Complete subcommands
		commands := []string{"start", "list", "status", "stop", "help"}
		var completions []string
		for _, cmd := range commands {
			if strings.HasPrefix(cmd, args[0]) {
				completions = append(completions, cmd)
			}
		}
		return completions
	}

	// For now, no completion for commission names or campaign IDs
	// In production, these would be fetched from the registry
	return nil
}