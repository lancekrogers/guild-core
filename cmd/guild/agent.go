package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/internal/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

var (
	agentNoDaemon bool // Don't auto-start the Guild server
)

// agentListCmd represents the agent list command
var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available agents",
	Long:  `Display all registered agents in the Guild, showing their capabilities and status.`,
	RunE:  runAgentList,
}

// agentStopCmd represents the agent stop command
var agentStopCmd = &cobra.Command{
	Use:   "stop [agent-id]",
	Short: "Stop an agent",
	Long:  `Stop a specific agent or all agents if no ID is provided.`,
	ValidArgsFunction: completeAgentIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			fmt.Printf("Stopping agent %s...\n", args[0])
		} else {
			fmt.Println("Stopping all agents...")
		}
		// TODO: Implement actual stop functionality when agent management is ready
		fmt.Println("Agent management functionality coming soon.")
		return nil
	},
}

// agentStatusCmd represents the agent status command
var agentStatusCmd = &cobra.Command{
	Use:   "status [agent-id]",
	Short: "Show agent status",
	Long:  `Display the current status of a specific agent or all agents.`,
	ValidArgsFunction: completeAgentIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			fmt.Printf("Status for agent %s:\n", args[0])
		} else {
			fmt.Println("Status for all agents:")
		}
		// TODO: Implement actual status functionality when agent management is ready
		fmt.Println("Agent management functionality coming soon.")
		return nil
	},
}

func init() {
	// Add flags for list command
	agentListCmd.Flags().BoolP("verbose", "v", false, "Show detailed agent information")
	agentListCmd.Flags().StringP("type", "t", "", "Filter agents by type")
	agentListCmd.Flags().IntP("max-cost", "c", 0, "Show only agents with cost <= value")
	
	// Add persistent flags
	agentCmd.PersistentFlags().BoolVar(&agentNoDaemon, "no-daemon", false, "Don't auto-start the Guild server")
	
	// Register agent subcommands
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentStopCmd)
	agentCmd.AddCommand(agentStatusCmd)
}

// runAgentList handles the agent list command
func runAgentList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Auto-start daemon unless --no-daemon flag is set
	if !agentNoDaemon {
		if !daemon.IsReachable(ctx) {
			fmt.Println("🚀 Starting Guild server...")
			if err := daemon.EnsureRunning(ctx); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start Guild server").
					WithComponent("cli").
					WithOperation("agent.list.daemon_start")
			}
			// Give the server a moment to fully initialize
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Check if server is reachable
	if !daemon.IsReachable(ctx) {
		return gerror.New(gerror.ErrCodeConnection, "Guild server is not reachable", nil).
			WithComponent("cli").
			WithOperation("agent.list").
			WithDetails("help", "Try running 'guild serve' manually or check 'guild status'")
	}

	// Get flags
	verbose, _ := cmd.Flags().GetBool("verbose")
	filterType, _ := cmd.Flags().GetString("type")
	maxCost, _ := cmd.Flags().GetInt("max-cost")

	// Initialize project context
	_, err := project.GetContext()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get project context").
			WithComponent("agent").
			WithOperation("list")
	}

	// Initialize component registry
	componentRegistry := registry.NewComponentRegistry()
	if err := componentRegistry.Initialize(ctx, registry.Config{}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("agent").
			WithOperation("list")
	}

	// Get available agent types from registry
	agentTypes := componentRegistry.Agents().ListAgentTypes()
	
	// Get configured agents from storage if available
	var configuredAgents []registry.AgentInfo
	if storageRegistry := componentRegistry.Storage(); storageRegistry != nil {
		if agentRepo := storageRegistry.GetAgentRepository(); agentRepo != nil {
			storageAgents, err := agentRepo.ListAgents(ctx)
			if err == nil && len(storageAgents) > 0 {
				// Convert storage agents to registry format
				for _, sa := range storageAgents {
					// Extract capabilities from map
					var capabilities []string
					if sa.Capabilities != nil {
						for cap := range sa.Capabilities {
							capabilities = append(capabilities, cap)
						}
					}
					
					configuredAgents = append(configuredAgents, registry.AgentInfo{
						ID:            sa.ID,
						Name:          sa.Name,
						Type:          sa.Type,
						CostMagnitude: int(sa.CostMagnitude),
						Capabilities:  capabilities,
					})
				}
			}
		}
	}

	// If no configured agents, use registered agent types with defaults
	if len(configuredAgents) == 0 {
		configuredAgents = componentRegistry.GetAgentsByCost(100) // Get all agents
	}

	// Apply filters
	var filteredAgents []registry.AgentInfo
	for _, agent := range configuredAgents {
		// Apply type filter
		if filterType != "" && agent.Type != filterType {
			continue
		}
		// Apply cost filter
		if maxCost > 0 && agent.CostMagnitude > maxCost {
			continue
		}
		filteredAgents = append(filteredAgents, agent)
	}

	// Display results
	if len(filteredAgents) == 0 {
		fmt.Println("No agents found matching the criteria.")
		return nil
	}

	if verbose {
		displayVerboseAgentList(filteredAgents, agentTypes)
	} else {
		displayCompactAgentList(filteredAgents)
	}

	return nil
}

// displayCompactAgentList shows a simple table of agents
func displayCompactAgentList(agents []registry.AgentInfo) {
	fmt.Println("🏰 Guild Agents")
	fmt.Println("═══════════════")
	fmt.Println()

	// Create a tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tCOST\tCAPABILITIES")
	fmt.Fprintln(w, "──\t────\t────\t────\t────────────")

	for _, agent := range agents {
		capabilities := strings.Join(agent.Capabilities, ", ")
		if len(capabilities) > 40 {
			capabilities = capabilities[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			agent.ID,
			agent.Name,
			agent.Type,
			agent.CostMagnitude,
			capabilities,
		)
	}

	w.Flush()
	fmt.Println()
	fmt.Printf("Total: %d agents\n", len(agents))
}

// displayVerboseAgentList shows detailed information about each agent
func displayVerboseAgentList(agents []registry.AgentInfo, agentTypes []string) {
	fmt.Println("🏰 Guild Agents (Detailed View)")
	fmt.Println("═══════════════════════════════")
	fmt.Println()

	for i, agent := range agents {
		if i > 0 {
			fmt.Println(strings.Repeat("─", 40))
		}

		costIcon := getAgentCostIcon(agent.CostMagnitude)
		fmt.Printf("%s %s (ID: %s)\n", costIcon, agent.Name, agent.ID)
		fmt.Printf("   Type: %s\n", agent.Type)
		fmt.Printf("   Cost: %d\n", agent.CostMagnitude)
		fmt.Printf("   Capabilities:\n")
		for _, cap := range agent.Capabilities {
			fmt.Printf("     • %s\n", cap)
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d agents\n", len(agents))
	fmt.Printf("Available Types: %s\n", strings.Join(agentTypes, ", "))
}

// getAgentCostIcon returns an emoji representing the cost magnitude
func getAgentCostIcon(cost int) string {
	switch {
	case cost <= 1:
		return "💰"  // Very cheap
	case cost <= 3:
		return "💰💰" // Moderate
	case cost <= 5:
		return "💰💰💰" // Expensive
	default:
		return "💰💰💰💰" // Very expensive
	}
}