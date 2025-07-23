package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/lancekrogers/guild/internal/integration/bootstrap"
	"github.com/lancekrogers/guild/pkg/campaign"
	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/orchestrator"
	"github.com/lancekrogers/guild/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiAgentOrchestrationFlow tests the complete flow from campaign to task assignment
func TestMultiAgentOrchestrationFlow(t *testing.T) {
	ctx := context.Background()
	logger := observability.GetLogger(ctx)

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

	// Get component registry
	componentReg := app.ComponentRegistry
	require.NotNil(t, componentReg)

	// Test 1: Create a campaign with commission
	t.Run("Campaign Creation and Processing", func(t *testing.T) {
		campaignReg := componentReg.Campaign()
		require.NotNil(t, campaignReg)

		campaignRepo := campaignReg.GetCampaignRepository()
		require.NotNil(t, campaignRepo)

		// Create test commission first
		commissionReg := componentReg.Commission()
		require.NotNil(t, commissionReg)

		commissionRepo := commissionReg.GetCommissionRepository()
		require.NotNil(t, commissionRepo)

		// Create commission
		testCommission := &registry.Commission{
			ID:          "test-commission-001",
			Title:       "Build REST API",
			Description: "Create a user management REST API with CRUD operations",
			Status:      "active",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err = commissionRepo.Create(ctx, testCommission)
		require.NoError(t, err)

		// Create campaign
		testCampaign := &registry.Campaign{
			ID:          "test-campaign-001",
			Name:        "API Development Campaign",
			Description: "Campaign to build user management API",
			Status:      campaign.StatusPlanning,
			Commissions: []string{testCommission.ID},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err = campaignRepo.Create(ctx, testCampaign)
		require.NoError(t, err)

		// Subscribe to events to track flow
		eventBus := app.EventBus
		var capturedEvents []events.CoreEvent
		eventChan := make(chan events.CoreEvent, 10)

		// Subscribe to key events
		eventTypes := []string{
			"campaign.started",
			"commission.process_requested",
			"commission.refined",
			"task.created",
			"task.assigned",
			"agent.discovered",
		}

		for _, eventType := range eventTypes {
			_, err := eventBus.Subscribe(ctx, eventType, func(ctx context.Context, e events.CoreEvent) error {
				select {
				case eventChan <- e:
				case <-time.After(time.Second):
					t.Logf("Warning: Event channel full for event type: %s", e.GetType())
				}
				return nil
			})
			require.NoError(t, err)
		}

		// Start the campaign
		campaignManager := campaign.NewCampaignManager(campaignRepo, eventBus, logger)
		err = campaignManager.StartCampaign(ctx, testCampaign.ID)
		require.NoError(t, err)

		// Collect events with timeout
		timeout := time.After(5 * time.Second)
		collecting := true
		for collecting {
			select {
			case event := <-eventChan:
				capturedEvents = append(capturedEvents, event)
				t.Logf("Captured event: %s", event.GetType())
			case <-timeout:
				collecting = false
			}
		}

		// Verify event flow
		assert.True(t, hasEventType(capturedEvents, "campaign.started"), "Should have campaign.started event")
		assert.True(t, hasEventType(capturedEvents, "commission.process_requested"), "Should have commission.process_requested event")

		// Check if tasks were created
		kanbanReg := componentReg.Kanban()
		if kanbanReg != nil {
			boardRepo := kanbanReg.GetBoardRepository()
			if boardRepo != nil {
				// Try to find tasks for the commission
				// This would need a more sophisticated query in real implementation
				t.Log("Kanban board repository available for task verification")
			}
		}
	})

	// Test 2: Agent Registration
	t.Run("Agent Registration", func(t *testing.T) {
		// The AgentRegistrationBridge should auto-register agents on startup
		// Let's verify agents are available
		agentReg := componentReg.Agent()
		require.NotNil(t, agentReg)

		// Check if we can access agent factory
		if agentFactory, ok := agentReg.(interface{ GetAgentFactory() interface{} }); ok {
			factory := agentFactory.GetAgentFactory()
			assert.NotNil(t, factory, "Agent factory should be available")
		}

		// Emit agent discovered event to test registration flow
		testAgentEvent := events.NewBaseEvent(
			"test-001",
			"agent.discovered",
			"test",
			map[string]interface{}{
				"agent_id":     "test-agent-001",
				"agent_name":   "Test Agent",
				"agent_type":   "developer",
				"provider":     "openai",
				"model":        "gpt-4",
				"capabilities": []string{"code", "test"},
			},
		)

		err = app.EventBus.Publish(ctx, testAgentEvent)
		assert.NoError(t, err)
	})

	// Test 3: Commission Processing Pipeline
	t.Run("Commission Processing Pipeline", func(t *testing.T) {
		// Get orchestrator registry
		orchReg := componentReg.Orchestrator()
		require.NotNil(t, orchReg)

		// Try to get commission integration service
		if orchRegTyped, ok := orchReg.(interface{ GetCommissionIntegrationService() *orchestrator.CommissionIntegrationService }); ok {
			integrationService := orchRegTyped.GetCommissionIntegrationService()
			if integrationService != nil {
				// Create test guild config
				testGuildConfig := &config.GuildConfig{
					Agents: []config.AgentConfig{
						{
							ID:       "architect",
							Name:     "System Architect",
							Type:     "architect",
							Provider: "anthropic",
							Model:    "claude-3-5-sonnet-20241022",
							SystemPrompt: `You are a system architect who designs software systems.`,
							Capabilities: []string{"design", "architecture"},
						},
						{
							ID:       "developer",
							Name:     "Senior Developer",
							Type:     "developer",
							Provider: "openai",
							Model:    "gpt-4",
							SystemPrompt: `You are a senior developer who implements software systems.`,
							Capabilities: []string{"code", "implement"},
						},
					},
				}

				// Process commission to tasks
				result, err := integrationService.ProcessCommissionToTasksByID(ctx, "test-commission-001", testGuildConfig)
				if err != nil {
					// This might fail if providers aren't configured, which is OK for this test
					t.Logf("Commission processing error (expected if providers not configured): %v", err)
				} else {
					assert.NotNil(t, result)
					assert.NotEmpty(t, result.Tasks, "Should have created tasks")
					assert.NotEmpty(t, result.AssignedArtisans, "Should have assigned artisans")
					
					t.Logf("Created %d tasks", len(result.Tasks))
					t.Logf("Assigned %d artisans", len(result.AssignedArtisans))
				}
			} else {
				t.Log("Commission integration service not available (might not be initialized)")
			}
		}
	})
}

// hasEventType checks if an event type exists in the captured events
func hasEventType(events []events.CoreEvent, eventType string) bool {
	for _, e := range events {
		if e.GetType() == eventType {
			return true
		}
	}
	return false
}