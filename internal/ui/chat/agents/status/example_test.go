// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package status_test

import (
	"context"
	"fmt"
	"time"

	"github.com/guild-ventures/guild-core/internal/ui/chat/agents/status"
)

// ExampleStatusTracker demonstrates basic usage of the status tracking system
func ExampleStatusTracker() {
	ctx := context.Background()
	
	// Create a new status tracker
	tracker := status.NewStatusTracker(ctx)
	
	// Register some agents
	tracker.RegisterAgent(status.AgentInfo{
		ID:     "manager-1",
		Name:   "Task Manager",
		Type:   "manager",
		Status: status.StatusIdle,
	})
	
	tracker.RegisterAgent(status.AgentInfo{
		ID:     "dev-1",
		Name:   "Code Artisan",
		Type:   "developer",
		Status: status.StatusIdle,
	})
	
	// Update agent status
	tracker.UpdateAgentStatus("manager-1", status.StatusThinking, "Planning tasks")
	tracker.UpdateAgentStatus("dev-1", status.StatusWorking, "Implementing feature")
	
	// Get all agents
	agents, _ := tracker.GetAllAgents()
	fmt.Printf("Total agents: %d\n", len(agents))
	
	// Get statistics
	stats := tracker.GetStats()
	fmt.Printf("Active agents: %d\n", stats.ActiveAgents)
	
	// Output:
	// Total agents: 2
	// Active agents: 2
}

// ExampleAgentDisplay demonstrates formatting agent status
func ExampleAgentDisplay() {
	display := status.NewAgentDisplay()
	
	agent := status.AgentInfo{
		ID:          "dev-1",
		Name:        "Code Artisan",
		Type:        "developer",
		Status:      status.StatusWorking,
		CurrentTask: "Implementing user authentication",
		TaskCount:   3,
		LastSeen:    time.Now(),
	}
	
	// Format compact display
	compact := display.FormatAgentCompact(agent)
	fmt.Println(compact)
	
	// Get status icon
	icon := display.GetStatusIcon(status.StatusWorking)
	fmt.Printf("Working icon: %s\n", icon)
	
	// Output:
	// ⚙️ Code Artisan [working] (3)
	// Working icon: ⚙️
}

// ExampleIndicatorManager demonstrates animated indicators
func ExampleIndicatorManager() {
	indicators := status.NewIndicatorManager()
	
	// Set indicators for agents
	indicators.SetIndicator("agent-1", status.IndicatorSpinner, status.StatusWorking)
	indicators.SetIndicator("agent-2", status.IndicatorProgress, status.StatusThinking)
	
	// Get indicator frames
	for i := 0; i < 3; i++ {
		indicators.Update()
		frame1 := indicators.GetIndicator("agent-1")
		frame2 := indicators.GetIndicator("agent-2")
		fmt.Printf("Frame %d: Agent1=%s Agent2=%s\n", i, frame1, frame2)
		time.Sleep(100 * time.Millisecond)
	}
	
	// Output will vary based on animation frames
}

// ExampleStatusIntegration demonstrates UI integration
func ExampleStatusIntegration() {
	// This example shows the pattern for integrating with the UI
	// In real usage, this would be connected to the actual StatusPane
	
	ctx := context.Background()
	
	// Create components
	tracker := status.NewStatusTracker(ctx)
	display := status.NewAgentDisplay()
	
	// Register an agent
	tracker.RegisterAgent(status.AgentInfo{
		ID:     "agent-1",
		Name:   "Example Agent",
		Type:   "worker",
		Status: status.StatusIdle,
	})
	
	// Simulate status changes
	tracker.UpdateAgentStatus("agent-1", status.StatusWorking, "Processing task")
	
	// Get formatted display
	agents, _ := tracker.GetAllAgents()
	summary := display.FormatAgentSummary(agents)
	fmt.Println(summary)
	
	// Output:
	// Agents (1): ⚙️1
}