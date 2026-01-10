package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lancekrogers/guild-core/pkg/events"
	"github.com/lancekrogers/guild-core/pkg/observability"
	"github.com/spf13/cobra"
)

// testEventsCmd provides a command to monitor event flow during testing
var testEventsCmd = &cobra.Command{
	Use:   "test-events",
	Short: "Monitor event flow for orchestration testing",
	Long: `Monitor all events flowing through the system to verify multi-agent orchestration.
	
This command subscribes to all orchestration-related events and displays them in real-time,
helping to debug and verify the event flow.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		logger := observability.GetLogger(ctx)

		// Create event bus
		eventBus := events.NewMemoryEventBusWithDefaults()

		// Events to monitor
		eventTypes := []string{
			// Campaign events
			"campaign.created",
			"campaign.started",
			"campaign.planning_started",
			"campaign.ready",
			"campaign.completed",

			// Commission events
			"commission.created",
			"commission.process_requested",
			"commission.refined",
			"commission.tasks_extracted",

			// Task events
			"task.created",
			"task.assigned",
			"task.started",
			"task.completed",

			// Agent events
			"agent.discovered",
			"agent.registered",
			"agent.task_received",
			"agent.task_completed",

			// Orchestration events
			"orchestration.started",
			"orchestration.agent_assigned",
			"orchestration.completed",
		}

		// Subscribe to all events
		for _, eventType := range eventTypes {
			et := eventType // capture loop variable
			_, err := eventBus.Subscribe(ctx, et, func(ctx context.Context, e events.CoreEvent) error {
				timestamp := time.Now().Format("15:04:05.000")
				data := e.GetData()

				// Format event display
				fmt.Printf("\n[%s] %s %s\n", timestamp, colorEvent(e.GetType()), e.GetID())
				fmt.Printf("  Source: %s\n", e.GetSource())

				// Display key data fields
				if campaignID, ok := data["campaign_id"]; ok {
					fmt.Printf("  Campaign: %s\n", campaignID)
				}
				if commissionID, ok := data["commission_id"]; ok {
					fmt.Printf("  Commission: %s\n", commissionID)
				}
				if taskID, ok := data["task_id"]; ok {
					fmt.Printf("  Task: %s\n", taskID)
				}
				if agentID, ok := data["agent_id"]; ok {
					fmt.Printf("  Agent: %s\n", agentID)
				}

				// Display any error
				if err, ok := data["error"]; ok {
					fmt.Printf("  ❌ Error: %v\n", err)
				}

				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to subscribe to %s: %w", et, err)
			}
		}

		logger.InfoContext(ctx, "Event monitor started. Press Ctrl+C to stop.")
		fmt.Println("\n🔍 Monitoring orchestration events...")
		fmt.Println("Run 'guild campaign start' in another terminal to test the flow.")
		fmt.Println("\nExpected event sequence:")
		fmt.Println("1. campaign.started")
		fmt.Println("2. commission.process_requested")
		fmt.Println("3. commission.refined")
		fmt.Println("4. task.created (multiple)")
		fmt.Println("5. agent.discovered (multiple)")
		fmt.Println("6. agent.registered")
		fmt.Println("7. task.assigned")
		fmt.Println("")

		// Keep running until interrupted
		<-ctx.Done()
		return nil
	},
}

// colorEvent adds color to event types for better visibility
func colorEvent(eventType string) string {
	switch {
	case contains(eventType, "campaign"):
		return fmt.Sprintf("\033[36m%s\033[0m", eventType) // Cyan
	case contains(eventType, "commission"):
		return fmt.Sprintf("\033[33m%s\033[0m", eventType) // Yellow
	case contains(eventType, "task"):
		return fmt.Sprintf("\033[32m%s\033[0m", eventType) // Green
	case contains(eventType, "agent"):
		return fmt.Sprintf("\033[35m%s\033[0m", eventType) // Magenta
	case contains(eventType, "orchestration"):
		return fmt.Sprintf("\033[34m%s\033[0m", eventType) // Blue
	default:
		return eventType
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func init() {
	rootCmd.AddCommand(testEventsCmd)
}
