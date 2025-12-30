// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commands

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/guild-framework/guild-core/pkg/commission"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// CommissionCommand handles /commission commands
type CommissionCommand struct {
	manager commission.CommissionManager
}

// NewCommissionCommand creates a new commission command handler
func NewCommissionCommand(manager commission.CommissionManager) *CommissionCommand {
	return &CommissionCommand{
		manager: manager,
	}
}

// Execute handles the commission command
func (c *CommissionCommand) Execute(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return c.showHelp()
	}

	switch args[0] {
	case "new":
		return c.startNewCommission()
	case "list":
		return c.listCommissions(ctx)
	case "status":
		return c.showCommissionStatus(ctx)
	case "refine":
		return c.triggerRefinement(ctx, args[1:])
	case "help":
		return c.showHelp()
	default:
		// Check if it's a commission ID for resuming
		return c.resumeCommission(ctx, args[0])
	}
}

// CommissionCommandResult represents the result of a commission command
type CommissionCommandResult struct {
	Type    string
	Message string
	Data    interface{}
	Error   error
}

// startNewCommission starts a new commission creation workflow
func (c *CommissionCommand) startNewCommission() tea.Cmd {
	return func() tea.Msg {
		return CommissionCommandResult{
			Type:    "start_new",
			Message: "Starting new commission creation with Elena...",
			Data: map[string]interface{}{
				"mode":  "planning",
				"agent": "elena-guild-master",
			},
		}
	}
}

// listCommissions lists all existing commissions
func (c *CommissionCommand) listCommissions(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		commissions, err := c.manager.ListCommissions(ctx)
		if err != nil {
			return CommissionCommandResult{
				Type: "error",
				Error: gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list commissions").
					WithComponent("chat.commands").
					WithOperation("listCommissions"),
			}
		}

		if len(commissions) == 0 {
			return CommissionCommandResult{
				Type:    "list",
				Message: "No commissions found. Use '/commission new' to create your first commission.",
				Data:    commissions,
			}
		}

		// Format commission list
		var sb strings.Builder
		sb.WriteString("📋 **Available Commissions:**\n\n")
		for i, comm := range commissions {
			status := "📝"
			switch comm.Status {
			case commission.CommissionStatusActive:
				status = "🚀"
			case commission.CommissionStatusCompleted:
				status = "✅"
			case commission.CommissionStatusCancelled:
				status = "❌"
			}

			sb.WriteString(fmt.Sprintf("%d. %s %s - %s\n", i+1, status, comm.Title, comm.ID))
			if comm.Description != "" {
				sb.WriteString(fmt.Sprintf("   %s\n", comm.Description))
			}
		}

		sb.WriteString("\nUse '/commission <id>' to resume a commission.")

		return CommissionCommandResult{
			Type:    "list",
			Message: sb.String(),
			Data:    commissions,
		}
	}
}

// showCommissionStatus shows the current commission status
func (c *CommissionCommand) showCommissionStatus(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		// This would check if there's an active commission in the current session
		// For now, return a placeholder
		return CommissionCommandResult{
			Type:    "status",
			Message: "No active commission in current session. Use '/commission new' to start or '/commission <id>' to resume.",
		}
	}
}

// triggerRefinement triggers commission refinement
func (c *CommissionCommand) triggerRefinement(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		if len(args) == 0 {
			return CommissionCommandResult{
				Type:    "error",
				Message: "Please specify a commission ID to refine.",
			}
		}

		commissionID := args[0]

		// Load the commission
		comm, err := c.manager.GetCommission(ctx, commissionID)
		if err != nil {
			return CommissionCommandResult{
				Type: "error",
				Error: gerror.Wrap(err, gerror.ErrCodeNotFound, "commission not found").
					WithComponent("chat.commands").
					WithOperation("triggerRefinement").
					WithDetails("id", commissionID),
			}
		}

		return CommissionCommandResult{
			Type:    "refine",
			Message: fmt.Sprintf("Starting refinement for commission: %s", comm.Title),
			Data:    comm,
		}
	}
}

// resumeCommission resumes an existing commission
func (c *CommissionCommand) resumeCommission(ctx context.Context, commissionID string) tea.Cmd {
	return func() tea.Msg {
		comm, err := c.manager.GetCommission(ctx, commissionID)
		if err != nil {
			// Try to list commissions to help the user
			commissions, _ := c.manager.ListCommissions(ctx)
			var suggestions []string
			for _, c := range commissions {
				if strings.Contains(c.ID, commissionID) || strings.Contains(strings.ToLower(c.Title), strings.ToLower(commissionID)) {
					suggestions = append(suggestions, c.ID)
				}
			}

			errMsg := fmt.Sprintf("Commission '%s' not found.", commissionID)
			if len(suggestions) > 0 {
				errMsg += fmt.Sprintf(" Did you mean: %s?", strings.Join(suggestions, ", "))
			}

			return CommissionCommandResult{
				Type:    "error",
				Message: errMsg,
			}
		}

		return CommissionCommandResult{
			Type:    "resume",
			Message: fmt.Sprintf("Resuming commission: %s", comm.Title),
			Data:    comm,
		}
	}
}

// showHelp shows help for commission commands
func (c *CommissionCommand) showHelp() tea.Cmd {
	return func() tea.Msg {
		helpText := `📋 **Commission Commands:**

• **/commission new** - Start creating a new commission with Elena
• **/commission list** - Show all existing commissions
• **/commission status** - Show current commission progress
• **/commission refine <id>** - Trigger refinement for a commission
• **/commission <id>** - Resume working on an existing commission

**During Commission Planning:**
• **/switch-planning-agent <agent-id>** - Switch to a different planning agent
• **/exit-plan-session** - Exit the planning session
• **/end** - Complete the current commission planning

Commissions are saved in the 'commissions/' directory at your project root.`

		return CommissionCommandResult{
			Type:    "help",
			Message: helpText,
		}
	}
}

// GetCompletions returns tab completions for the commission command
func (c *CommissionCommand) GetCompletions(ctx context.Context, args []string) []string {
	if len(args) == 0 {
		return []string{"new", "list", "status", "refine", "help"}
	}

	if len(args) == 1 {
		// Complete subcommands
		commands := []string{"new", "list", "status", "refine", "help"}
		var completions []string
		for _, cmd := range commands {
			if strings.HasPrefix(cmd, args[0]) {
				completions = append(completions, cmd)
			}
		}

		// Also include commission IDs for direct access
		if commissions, err := c.manager.ListCommissions(ctx); err == nil {
			for _, comm := range commissions {
				if strings.HasPrefix(comm.ID, args[0]) || strings.HasPrefix(strings.ToLower(comm.Title), strings.ToLower(args[0])) {
					completions = append(completions, comm.ID)
				}
			}
		}

		return completions
	}

	if len(args) == 2 && args[0] == "refine" {
		// Complete commission IDs for refine command
		var completions []string
		if commissions, err := c.manager.ListCommissions(ctx); err == nil {
			for _, comm := range commissions {
				if strings.HasPrefix(comm.ID, args[1]) {
					completions = append(completions, comm.ID)
				}
			}
		}
		return completions
	}

	return nil
}
