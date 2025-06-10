// +build integration

package commission_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/guild-ventures/guild-core/internal/testutil"
	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/orchestrator"
	"github.com/guild-ventures/guild-core/pkg/providers/mock"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDebugCommissionIntegration debugs what's happening in commission integration
func TestDebugCommissionIntegration(t *testing.T) {
	// Setup test project
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "guild-debug-test",
	})
	defer cleanup()

	fmt.Printf("🐛 Debugging commission integration service\n")

	ctx := context.Background()

	// Setup registry with test helpers
	reg := setupTestRegistry(t, ctx, projCtx)

	// Setup mock provider
	mockProvider := mock.NewProvider()

	// Setup response that should work
	originalResponse := `## File: commission_refined.md

# Simple Commission

## Task List

- BACKEND-001: First Task (priority: high, estimate: 2h)

## File: README.md

# Simple Commission

This is a simple commission.`

	mockProvider.SetResponse("Guild Master", originalResponse)

	// Register provider with registry
	err := reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	// Initialize registry with configuration
	registryConfig := registry.Config{
		Providers: registry.ProviderConfig{
			DefaultProvider: "mock",
		},
	}
	err = reg.Initialize(ctx, registryConfig)
	require.NoError(t, err)

	// Create integration service using factory
	service, err := orchestrator.DefaultCommissionIntegrationServiceFactory(reg)
	require.NoError(t, err)

	// Create simple commission
	commission := manager.Commission{
		ID:          "debug-commission",
		Title:       "Debug Commission",
		Description: "A simple commission for debugging",
		Domain:      "default",
	}

	// Simplified guild config
	guildConfig := &config.GuildConfig{
		Name: "Debug Guild",
		Agents: []config.AgentConfig{
			{
				ID:       "debug-agent",
				Name:     "Debug Agent",
				Type:     "worker",
				Provider: "mock",
				Model:    "mock-model",
			},
		},
	}

	fmt.Printf("📤 Processing commission through integration service...\n")

	// Process commission - this should trigger the mock provider
	result, err := service.ProcessCommissionToTasks(ctx, commission, guildConfig)

	fmt.Printf("\n📋 Results:\n")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		require.NoError(t, err) // Fail the test
	} else {
		fmt.Printf("✅ Success! Tasks created: %d\n", len(result.Tasks))
		for i, task := range result.Tasks {
			fmt.Printf("  %d. %s (ID: %s)\n", i+1, task.Title, task.ID)
		}
		
		// Verify results
		require.NotNil(t, result)
		require.NotNil(t, result.RefinedCommission)
		require.Greater(t, len(result.Tasks), 0, "Should create at least one task")
		
		// Verify task details
		firstTask := result.Tasks[0]
		assert.Contains(t, firstTask.Title, "First Task")
		assert.Equal(t, kanban.PriorityHigh, firstTask.Priority)
	}

	fmt.Printf("\n🏁 Debug test completed successfully\n")
}