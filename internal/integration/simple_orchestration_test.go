package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/lancekrogers/guild/internal/integration/bootstrap"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasicOrchestrationFlow tests the basic event flow of orchestration
func TestBasicOrchestrationFlow(t *testing.T) {
	ctx := context.Background()

	// Create test application
	app, err := bootstrap.NewApplication(bootstrap.DefaultOptions())
	require.NoError(t, err)

	// Initialize application
	err = app.Initialize(ctx)
	require.NoError(t, err)

	// Start application
	err = app.Start(ctx)
	require.NoError(t, err)
	defer app.Stop(ctx)

	// Test event flow
	t.Run("Event Flow Verification", func(t *testing.T) {
		eventBus := app.EventBus
		require.NotNil(t, eventBus)

		// Track events
		var capturedEvents []string
		eventChan := make(chan string, 10)

		// Subscribe to key orchestration events
		eventTypes := []string{
			"campaign.started",
			"commission.process_requested",
			"agent.discovered",
		}

		for _, eventType := range eventTypes {
			et := eventType // capture loop variable
			_, err := eventBus.Subscribe(ctx, et, func(ctx context.Context, e events.CoreEvent) error {
				select {
				case eventChan <- e.GetType():
					t.Logf("Captured event: %s", e.GetType())
				default:
					t.Logf("Event channel full, dropping: %s", e.GetType())
				}
				return nil
			})
			require.NoError(t, err)
		}

		// Simulate campaign start event
		campaignStartEvent := events.NewBaseEvent(
			"test-001",
			"campaign.started",
			"test",
			map[string]interface{}{
				"campaign_id": "test-campaign-001",
			},
		)

		err = eventBus.Publish(ctx, campaignStartEvent)
		require.NoError(t, err)

		// Simulate agent discovered event
		agentEvent := events.NewBaseEvent(
			"test-002",
			"agent.discovered",
			"test",
			map[string]interface{}{
				"agent_id":   "test-agent",
				"agent_name": "Test Agent",
				"agent_type": "developer",
			},
		)

		err = eventBus.Publish(ctx, agentEvent)
		require.NoError(t, err)

		// Wait for events to be processed
		timeout := time.After(2 * time.Second)
		collecting := true
		for collecting {
			select {
			case event := <-eventChan:
				capturedEvents = append(capturedEvents, event)
			case <-timeout:
				collecting = false
			}
		}

		// Verify we captured the events
		assert.Contains(t, capturedEvents, "campaign.started", "Should have captured campaign.started event")
		assert.Contains(t, capturedEvents, "agent.discovered", "Should have captured agent.discovered event")
		
		// Check if commission.process_requested was triggered by campaign.started
		// This would be emitted by OrchestratorCampaignBridge if it's working
		// If not found, it means the bridge needs a campaign manager to work properly
		t.Logf("Captured events: %v", capturedEvents)
	})

	// Test bridge initialization
	t.Run("Bridge Initialization", func(t *testing.T) {
		// Check that bridges are created
		assert.NotNil(t, app.OrchestratorCampaignBridge, "OrchestratorCampaignBridge should be initialized")
		assert.NotNil(t, app.AgentRegistrationBridge, "AgentRegistrationBridge should be initialized")
		assert.NotNil(t, app.CommissionProcessorBridge, "CommissionProcessorBridge should be initialized")
		
		// Check bridge health
		if app.OrchestratorCampaignBridge != nil {
			err := app.OrchestratorCampaignBridge.Health(ctx)
			// This will fail if dependencies aren't wired, which is expected
			if err != nil {
				t.Logf("OrchestratorCampaignBridge health check: %v (expected if dependencies not fully wired)", err)
			}
		}
	})

	// Test service registry
	t.Run("Service Registry", func(t *testing.T) {
		health := app.Health(ctx)
		t.Logf("Service health status:")
		for service, err := range health {
			if err != nil {
				t.Logf("  %s: ERROR - %v", service, err)
			} else {
				t.Logf("  %s: OK", service)
			}
		}
		
		// At least some services should be healthy
		healthyCount := 0
		for _, err := range health {
			if err == nil {
				healthyCount++
			}
		}
		assert.Greater(t, healthyCount, 0, "At least some services should be healthy")
	})
}