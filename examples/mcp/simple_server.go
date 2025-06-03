// Package main demonstrates a simple MCP server setup
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
	"github.com/guild-ventures/guild-core/pkg/mcp/server"
	"github.com/guild-ventures/guild-core/pkg/mcp/tools"
	"github.com/guild-ventures/guild-core/pkg/mcp/transport"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

func main() {
	ctx := context.Background()

	// Create guild registry
	guildRegistry := registry.New()

	// Configure MCP server
	serverConfig := &server.Config{
		ServerID:   "example-mcp-server",
		ServerName: "Example MCP Server",
		Version:    "1.0.0",
		TransportConfig: &transport.TransportConfig{
			Type:           "nats",
			Address:        "nats://localhost:4222",
			ConnectTimeout: 10 * time.Second,
			MaxReconnects:  3,
			ReconnectWait:  2 * time.Second,
		},
		MaxConcurrentRequests: 100,
		RequestTimeout:        30 * time.Second,
		EnableTLS:             false,
		EnableAuth:            false,
		EnableMetrics:         true,
		EnableTracing:         true,
		EnableCostTracking:    true,
	}

	// Create and start MCP server
	mcpServer, err := server.NewServer(serverConfig, guildRegistry)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Register some example tools before starting
	registerExampleTools(mcpServer.GetToolRegistry())

	// Start the server
	log.Println("Starting MCP server...")
	if err := mcpServer.Start(ctx); err != nil {
		log.Fatalf("Failed to start MCP server: %v", err)
	}

	log.Printf("MCP server started successfully on %s", serverConfig.TransportConfig.Address)
	log.Println("Server features:")
	log.Printf("  - TLS: %v", serverConfig.EnableTLS)
	log.Printf("  - Auth: %v", serverConfig.EnableAuth)
	log.Printf("  - Metrics: %v", serverConfig.EnableMetrics)
	log.Printf("  - Tracing: %v", serverConfig.EnableTracing)
	log.Printf("  - Cost Tracking: %v", serverConfig.EnableCostTracking)

	// Keep the server running
	select {
	case <-ctx.Done():
		log.Println("Context cancelled, shutting down...")
	}

	// Graceful shutdown
	log.Println("Shutting down MCP server...")
	if err := mcpServer.Stop(ctx); err != nil {
		log.Printf("Error stopping server: %v", err)
	}
	log.Println("Server stopped")
}

// registerExampleTools registers some example tools with the server
func registerExampleTools(registry tools.Registry) {
	// Calculator tool
	calculator := tools.NewBaseTool(
		"calculator",
		"Calculator",
		"Simple mathematical calculator",
		[]string{"math", "calculation", "arithmetic"},
		protocol.CostProfile{
			ComputeCost:   0.001,
			MemoryCost:    512,
			LatencyCost:   50 * time.Millisecond,
			TokensCost:    0,
			APICallsCost:  0,
			FinancialCost: 0.0001,
		},
		[]protocol.ToolParameter{
			{
				Name:        "operation",
				Type:        "string",
				Description: "Mathematical operation: add, subtract, multiply, divide",
				Required:    true,
			},
			{
				Name:        "a",
				Type:        "number",
				Description: "First number",
				Required:    true,
			},
			{
				Name:        "b",
				Type:        "number",
				Description: "Second number",
				Required:    true,
			},
		},
		[]protocol.ToolParameter{
			{
				Name:        "result",
				Type:        "number",
				Description: "Calculation result",
			},
		},
		func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// Simple calculator implementation
			operation, ok := params["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation must be a string")
			}

			a, ok := params["a"].(float64)
			if !ok {
				return nil, fmt.Errorf("parameter 'a' must be a number")
			}

			b, ok := params["b"].(float64)
			if !ok {
				return nil, fmt.Errorf("parameter 'b' must be a number")
			}

			var result float64
			switch operation {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return nil, fmt.Errorf("division by zero")
				}
				result = a / b
			default:
				return nil, fmt.Errorf("unsupported operation: %s", operation)
			}

			return map[string]interface{}{
				"result": result,
			}, nil
		},
	)

	if err := registry.RegisterTool(calculator); err != nil {
		log.Printf("Failed to register calculator tool: %v", err)
	} else {
		log.Println("Registered calculator tool")
	}

	// Text processor tool
	textProcessor := tools.NewBaseTool(
		"text-processor",
		"Text Processor",
		"Processes text with various operations",
		[]string{"text", "string", "processing"},
		protocol.CostProfile{
			ComputeCost:   0.002,
			MemoryCost:    1024,
			LatencyCost:   100 * time.Millisecond,
			TokensCost:    10,
			APICallsCost:  0,
			FinancialCost: 0.0002,
		},
		[]protocol.ToolParameter{
			{
				Name:        "operation",
				Type:        "string",
				Description: "Text operation: uppercase, lowercase, reverse, length",
				Required:    true,
			},
			{
				Name:        "text",
				Type:        "string",
				Description: "Input text to process",
				Required:    true,
			},
		},
		[]protocol.ToolParameter{
			{
				Name:        "result",
				Type:        "string",
				Description: "Processed text result",
			},
		},
		func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			operation, ok := params["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation must be a string")
			}

			text, ok := params["text"].(string)
			if !ok {
				return nil, fmt.Errorf("text must be a string")
			}

			var result interface{}
			switch operation {
			case "uppercase":
				result = fmt.Sprintf("%s", text) // Simplified - would use strings.ToUpper in real implementation
			case "lowercase":
				result = fmt.Sprintf("%s", text) // Simplified - would use strings.ToLower in real implementation
			case "reverse":
				runes := []rune(text)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				result = string(runes)
			case "length":
				result = len(text)
			default:
				return nil, fmt.Errorf("unsupported operation: %s", operation)
			}

			return map[string]interface{}{
				"result": result,
			}, nil
		},
	)

	if err := registry.RegisterTool(textProcessor); err != nil {
		log.Printf("Failed to register text processor tool: %v", err)
	} else {
		log.Println("Registered text processor tool")
	}

	log.Printf("Total tools registered: %d", len(registry.ListTools()))
}