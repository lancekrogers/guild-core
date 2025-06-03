// Package main demonstrates proper MCP integration with Guild Framework
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
	"github.com/guild-ventures/guild-core/pkg/mcp/tools"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Starting Guild Framework with MCP integration...")

	// Create MCP configuration
	mcpConfig := createMCPConfig()
	log.Printf("MCP Configuration: Enabled=%v, Transport=%s", 
		mcpConfig.Enabled, mcpConfig.Transport.Type)

	// Convert to integration config
	integrationConfig, err := mcpConfig.ToIntegrationConfig()
	if err != nil {
		log.Fatalf("Failed to convert MCP config: %v", err)
	}

	// Create extended Guild registry with MCP support
	guildRegistry, err := registry.NewExtendedComponentRegistry(ctx, integrationConfig)
	if err != nil {
		log.Fatalf("Failed to create Guild registry: %v", err)
	}

	log.Printf("Guild Framework initialized successfully")
	log.Printf("MCP Enabled: %v", guildRegistry.IsMCPEnabled())

	if guildRegistry.IsMCPEnabled() {
		// Register example tools
		if err := registerExampleTools(ctx, guildRegistry); err != nil {
			log.Fatalf("Failed to register tools: %v", err)
		}

		// Demonstrate tool synchronization
		if err := demonstrateToolSync(ctx, guildRegistry); err != nil {
			log.Printf("Warning: Tool sync demonstration failed: %v", err)
		}

		// Show registry integration
		demonstrateRegistryIntegration(guildRegistry)
	}

	log.Println("Guild Framework is running...")
	log.Println("Press Ctrl+C to shutdown gracefully")

	// Wait for shutdown signal
	<-sigCh
	log.Println("Shutdown signal received, shutting down gracefully...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := guildRegistry.Close(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Guild Framework shutdown complete")
}

func createMCPConfig() *config.MCPConfig {
	// Check environment for configuration type
	configType := os.Getenv("GUILD_MCP_CONFIG")
	
	switch configType {
	case "production":
		log.Println("Using production MCP configuration")
		return config.ProductionMCPConfig()
	case "nats":
		log.Println("Using NATS MCP configuration")
		return config.ExampleNATSConfig()
	case "development":
		fallthrough
	default:
		log.Println("Using development MCP configuration")
		return config.ExampleDevelopmentConfig()
	}
}

func registerExampleTools(ctx context.Context, guildRegistry *registry.ExtendedComponentRegistry) error {
	if !guildRegistry.IsMCPEnabled() {
		return fmt.Errorf("MCP not enabled")
	}

	mcpServer := guildRegistry.GetMCPExtension().GetMCPServer()
	toolRegistry := mcpServer.GetToolRegistry()

	// Register a file system tool
	fileSystemTool := tools.NewBaseTool(
		"filesystem-tool",
		"File System Tool",
		"Provides file system operations with proper context handling",
		[]string{"filesystem", "io", "files"},
		protocol.CostProfile{
			ComputeCost:   0.01,
			MemoryCost:    2048,
			LatencyCost:   100 * time.Millisecond,
			TokensCost:    0,
			APICallsCost:  0,
			FinancialCost: 0.001,
		},
		[]protocol.ToolParameter{
			{
				Name:        "operation",
				Type:        "string",
				Description: "File operation: read, write, list, exists",
				Required:    true,
			},
			{
				Name:        "path",
				Type:        "string",
				Description: "File or directory path",
				Required:    true,
			},
			{
				Name:        "content",
				Type:        "string",
				Description: "Content for write operations",
				Required:    false,
			},
		},
		[]protocol.ToolParameter{
			{
				Name:        "result",
				Type:        "object",
				Description: "Operation result",
			},
		},
		func(execCtx context.Context, params map[string]interface{}) (interface{}, error) {
			// Demonstrate proper context usage
			if requestID := execCtx.Value("request_id"); requestID != nil {
				log.Printf("FileSystem tool executing for request: %v", requestID)
			}

			operation, ok := params["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation parameter required")
			}

			path, ok := params["path"].(string)
			if !ok {
				return nil, fmt.Errorf("path parameter required")
			}

			// Simulate file operations (in production, implement real file ops)
			switch operation {
			case "exists":
				return map[string]interface{}{
					"exists": true, // Simplified
					"path":   path,
				}, nil
			case "list":
				return map[string]interface{}{
					"files": []string{"file1.txt", "file2.txt"}, // Simplified
					"path":  path,
				}, nil
			case "read":
				return map[string]interface{}{
					"content": "file content here", // Simplified
					"path":    path,
				}, nil
			case "write":
				content := params["content"]
				return map[string]interface{}{
					"success": true,
					"path":    path,
					"bytes":   len(fmt.Sprintf("%v", content)),
				}, nil
			default:
				return nil, fmt.Errorf("unsupported operation: %s", operation)
			}
		},
	)

	if err := toolRegistry.RegisterTool(fileSystemTool); err != nil {
		return fmt.Errorf("failed to register filesystem tool: %w", err)
	}

	// Register a Guild-specific tool that demonstrates integration
	guildTool := tools.NewBaseTool(
		"guild-coordinator",
		"Guild Coordinator",
		"Coordinates Guild agents and demonstrates registry integration",
		[]string{"guild", "coordination", "agents"},
		protocol.CostProfile{
			ComputeCost:   0.005,
			MemoryCost:    1024,
			LatencyCost:   50 * time.Millisecond,
			TokensCost:    0,
			APICallsCost:  0,
			FinancialCost: 0.0005,
		},
		[]protocol.ToolParameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "Coordination action: status, list_agents, create_task",
				Required:    true,
			},
			{
				Name:        "target",
				Type:        "string",
				Description: "Target agent or resource",
				Required:    false,
			},
		},
		[]protocol.ToolParameter{
			{
				Name:        "result",
				Type:        "object",
				Description: "Coordination result",
			},
		},
		func(execCtx context.Context, params map[string]interface{}) (interface{}, error) {
			// Extract Guild-specific context information
			if guildContext := execCtx.Value("guild_context"); guildContext != nil {
				log.Printf("Guild coordinator operating with Guild context: %v", guildContext)
			}

			action, ok := params["action"].(string)
			if !ok {
				return nil, fmt.Errorf("action parameter required")
			}

			// Demonstrate coordination operations
			switch action {
			case "status":
				return map[string]interface{}{
					"status": "active",
					"agents": 3,
					"tasks":  5,
					"mcp_enabled": true,
				}, nil
			case "list_agents":
				return map[string]interface{}{
					"agents": []map[string]interface{}{
						{"id": "agent-1", "type": "worker", "status": "idle"},
						{"id": "agent-2", "type": "researcher", "status": "busy"},
						{"id": "agent-3", "type": "coordinator", "status": "active"},
					},
				}, nil
			case "create_task":
				target := params["target"]
				return map[string]interface{}{
					"task_id": "task-123",
					"assigned_to": target,
					"status": "created",
					"created_at": time.Now().Format(time.RFC3339),
				}, nil
			default:
				return nil, fmt.Errorf("unsupported action: %s", action)
			}
		},
	)

	if err := toolRegistry.RegisterTool(guildTool); err != nil {
		return fmt.Errorf("failed to register guild tool: %w", err)
	}

	log.Printf("Registered %d tools in MCP registry", len(toolRegistry.ListTools()))
	return nil
}

func demonstrateToolSync(ctx context.Context, guildRegistry *registry.ExtendedComponentRegistry) error {
	log.Println("Demonstrating tool synchronization between MCP and Guild registries...")

	// Sync tools between registries
	if err := guildRegistry.SyncMCPTools(ctx); err != nil {
		return fmt.Errorf("failed to sync MCP tools: %w", err)
	}

	log.Println("Tool synchronization completed successfully")

	// Show tool counts
	mcpServer := guildRegistry.GetMCPExtension().GetMCPServer()
	mcpTools := mcpServer.GetToolRegistry().ListTools()
	log.Printf("Tools in MCP registry: %d", len(mcpTools))

	for _, tool := range mcpTools {
		log.Printf("  - %s (%s): %s", tool.ID(), tool.Name(), tool.Description())
	}

	return nil
}

func demonstrateRegistryIntegration(guildRegistry *registry.ExtendedComponentRegistry) {
	log.Println("Demonstrating Guild registry integration patterns...")

	// Show MCP extension access
	mcpExtension := guildRegistry.GetMCPExtension()
	if mcpExtension != nil {
		log.Println("✓ MCP extension accessible through Guild registry")
		
		server := mcpExtension.GetMCPServer()
		if server != nil {
			config := server.GetConfig()
			log.Printf("✓ MCP server accessible with ID: %s", config.ServerID)
		}

		toolBridge := mcpExtension.GetToolBridge()
		if toolBridge != nil {
			log.Println("✓ Tool bridge accessible for registry synchronization")
		}
	}

	// Show interface compliance
	var mcpRegistryInterface registry.MCPRegistry = guildRegistry
	if mcpRegistryInterface.IsMCPEnabled() {
		log.Println("✓ Guild registry properly implements MCPRegistry interface")
	}

	// Show configuration integration
	log.Println("✓ MCP configuration properly integrated with Guild config system")

	log.Println("Registry integration demonstration complete")
}

// Example configuration for different environments
func exampleProductionSetup() *config.MCPConfig {
	return &config.MCPConfig{
		Enabled:    true,
		ServerID:   "guild-prod-mcp",
		ServerName: "Guild Production MCP Server",
		Transport: config.TransportConfig{
			Type:           "nats",
			Address:        "nats://nats-cluster:4222",
			ConnectTimeout: "10s",
			MaxReconnects:  5,
			ReconnectWait:  "2s",
			Config: map[string]string{
				"cluster_id": "guild-prod-cluster",
			},
		},
		Security: config.SecurityConfig{
			EnableTLS:   true,
			TLSCertFile: "/etc/guild/tls/server.crt",
			TLSKeyFile:  "/etc/guild/tls/server.key",
			EnableAuth:  true,
			JWTSecret:   "${GUILD_JWT_SECRET}",
		},
		Performance: config.PerformanceConfig{
			MaxConcurrentRequests: 1000,
			RequestTimeout:        "30s",
			ShutdownTimeout:       "15s",
		},
		Features: config.FeatureConfig{
			EnableMetrics:      true,
			EnableTracing:      true,
			EnableCostTracking: true,
		},
	}
}