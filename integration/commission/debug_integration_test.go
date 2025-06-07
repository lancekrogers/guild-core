// +build integration

package commission_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/memory/boltdb"
	"github.com/guild-ventures/guild-core/pkg/orchestrator"
	"github.com/guild-ventures/guild-core/pkg/providers/mock"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/stretchr/testify/require"
)

// TestDebugCommissionIntegration debugs what's happening in commission integration
func TestDebugCommissionIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-debug-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fmt.Printf("🐛 Debugging commission integration service\n")

	// Setup minimal registry
	reg := registry.NewComponentRegistry()
	ctx := context.Background()

	// First register the provider, then initialize
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

	err = reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	// Setup memory store
	dbPath := tempDir + "/debug.db"
	store, err := boltdb.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	err = reg.Memory().RegisterMemoryStore("default", store)
	require.NoError(t, err)

	registryConfig := registry.Config{
		Providers: registry.ProviderConfig{
			DefaultProvider: "mock",
		},
	}
	err = reg.Initialize(ctx, registryConfig)
	require.NoError(t, err)

	// Create integration service
	service, err := orchestrator.NewCommissionIntegrationService(reg)
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
	} else {
		fmt.Printf("✅ Success! Tasks created: %d\n", len(result.Tasks))
		for i, task := range result.Tasks {
			fmt.Printf("  %d. %s (ID: %s)\n", i+1, task.Title, task.ID)
		}
	}

	fmt.Printf("\n🏁 Debug test completed\n")
}
