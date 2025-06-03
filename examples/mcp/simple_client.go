// Package main demonstrates a simple MCP client
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/guild-ventures/guild-core/pkg/mcp/client"
	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
	"github.com/guild-ventures/guild-core/pkg/mcp/transport"
)

func main() {
	ctx := context.Background()

	// Configure MCP client
	clientConfig := &client.Config{
		ClientID:   "example-mcp-client",
		ClientName: "Example MCP Client",
		Version:    "1.0.0",
		TransportConfig: &transport.TransportConfig{
			Type:           "nats",
			Address:        "nats://localhost:4222",
			ConnectTimeout: 10 * time.Second,
			MaxReconnects:  3,
			ReconnectWait:  2 * time.Second,
		},
		RequestTimeout: 10 * time.Second,
		EnableTracing:  true,
	}

	// Create and connect MCP client
	mcpClient, err := client.NewClient(clientConfig)
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v", err)
	}

	log.Println("Connecting to MCP server...")
	if err := mcpClient.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer mcpClient.Disconnect(ctx)

	log.Println("Connected successfully!")

	// Test server connectivity
	testServerConnectivity(ctx, mcpClient)

	// Discover available tools
	discoverTools(ctx, mcpClient)

	// Execute some tool operations
	executeToolOperations(ctx, mcpClient)

	// Query cost analytics
	queryCostAnalytics(ctx, mcpClient)

	log.Println("Client operations completed successfully!")
}

func testServerConnectivity(ctx context.Context, client *client.Client) {
	log.Println("\n=== Testing Server Connectivity ===")

	// Ping the server
	timestamp, err := client.Ping(ctx)
	if err != nil {
		log.Printf("Ping failed: %v", err)
		return
	}
	log.Printf("Ping successful - Server timestamp: %v", timestamp)

	// Get system information
	info, err := client.GetSystemInfo(ctx)
	if err != nil {
		log.Printf("Failed to get system info: %v", err)
		return
	}

	log.Println("Server Information:")
	infoJSON, _ := json.MarshalIndent(info, "  ", "  ")
	log.Printf("  %s", string(infoJSON))
}

func discoverTools(ctx context.Context, client *client.Client) {
	log.Println("\n=== Discovering Tools ===")

	// Discover all available tools
	query := &protocol.ToolQuery{
		MaxCost:    10.0,
		MaxLatency: 5 * time.Second,
		Limit:      50,
	}

	discovery, err := client.DiscoverTools(ctx, query)
	if err != nil {
		log.Printf("Tool discovery failed: %v", err)
		return
	}

	log.Printf("Found %d tools:", len(discovery.Tools))
	for _, tool := range discovery.Tools {
		log.Printf("  - %s (%s): %s", tool.ID, tool.Name, tool.Description)
		log.Printf("    Capabilities: %v", tool.Capabilities)
		log.Printf("    Cost: Compute=%.4f, Financial=%.4f", 
			tool.CostProfile.ComputeCost, tool.CostProfile.FinancialCost)
	}

	// Discover tools with specific capabilities
	mathQuery := &protocol.ToolQuery{
		RequiredCapabilities: []string{"math"},
		MaxCost:              1.0,
		Limit:                10,
	}

	mathDiscovery, err := client.DiscoverTools(ctx, mathQuery)
	if err != nil {
		log.Printf("Math tool discovery failed: %v", err)
		return
	}

	log.Printf("\nMath tools found: %d", len(mathDiscovery.Tools))
	for _, tool := range mathDiscovery.Tools {
		log.Printf("  - %s: %s", tool.ID, tool.Name)
	}
}

func executeToolOperations(ctx context.Context, client *client.Client) {
	log.Println("\n=== Executing Tool Operations ===")

	// Test calculator tool
	executeCalculatorOperations(ctx, client)

	// Test text processor tool
	executeTextProcessorOperations(ctx, client)
}

func executeCalculatorOperations(ctx context.Context, client *client.Client) {
	log.Println("\nTesting Calculator Tool:")

	// Check if calculator tool is healthy
	healthy, err := client.CheckToolHealth(ctx, "calculator")
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}
	log.Printf("Calculator tool health: %v", healthy)

	if !healthy {
		log.Println("Calculator tool is not healthy, skipping operations")
		return
	}

	// Perform addition
	addRequest := &protocol.ToolExecutionRequest{
		ToolID:      "calculator",
		ExecutionID: fmt.Sprintf("calc-add-%d", time.Now().UnixNano()),
		Parameters: map[string]interface{}{
			"operation": "add",
			"a":         15.5,
			"b":         24.3,
		},
	}

	startTime := time.Now()
	addResponse, err := client.ExecuteTool(ctx, addRequest)
	if err != nil {
		log.Printf("Addition failed: %v", err)
	} else {
		duration := time.Since(startTime)
		var result map[string]interface{}
		json.Unmarshal(addResponse.Result, &result)
		log.Printf("Addition result: 15.5 + 24.3 = %v (took %v)", result["result"], duration)

		// Report the cost
		cost := &protocol.CostReport{
			OperationID:   addRequest.ExecutionID,
			StartTime:     addResponse.StartTime,
			EndTime:       addResponse.EndTime,
			ComputeCost:   0.001,
			MemoryCost:    512,
			LatencyCost:   time.Duration(addResponse.EndTime.Unix()-addResponse.StartTime.Unix()) * time.Second,
			TokensCost:    0,
			APICallsCost:  1,
			FinancialCost: 0.0001,
		}
		client.ReportCost(ctx, cost)
	}

	// Perform division
	divRequest := &protocol.ToolExecutionRequest{
		ToolID:      "calculator",
		ExecutionID: fmt.Sprintf("calc-div-%d", time.Now().UnixNano()),
		Parameters: map[string]interface{}{
			"operation": "divide",
			"a":         100.0,
			"b":         4.0,
		},
	}

	divResponse, err := client.ExecuteTool(ctx, divRequest)
	if err != nil {
		log.Printf("Division failed: %v", err)
	} else {
		var result map[string]interface{}
		json.Unmarshal(divResponse.Result, &result)
		log.Printf("Division result: 100 / 4 = %v", result["result"])

		// Report the cost
		cost := &protocol.CostReport{
			OperationID:   divRequest.ExecutionID,
			StartTime:     divResponse.StartTime,
			EndTime:       divResponse.EndTime,
			ComputeCost:   0.001,
			MemoryCost:    512,
			LatencyCost:   time.Duration(divResponse.EndTime.Unix()-divResponse.StartTime.Unix()) * time.Second,
			TokensCost:    0,
			APICallsCost:  1,
			FinancialCost: 0.0001,
		}
		client.ReportCost(ctx, cost)
	}
}

func executeTextProcessorOperations(ctx context.Context, client *client.Client) {
	log.Println("\nTesting Text Processor Tool:")

	// Check if text processor tool is healthy
	healthy, err := client.CheckToolHealth(ctx, "text-processor")
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}
	log.Printf("Text processor tool health: %v", healthy)

	if !healthy {
		log.Println("Text processor tool is not healthy, skipping operations")
		return
	}

	// Reverse text
	reverseRequest := &protocol.ToolExecutionRequest{
		ToolID:      "text-processor",
		ExecutionID: fmt.Sprintf("text-reverse-%d", time.Now().UnixNano()),
		Parameters: map[string]interface{}{
			"operation": "reverse",
			"text":      "Hello, MCP World!",
		},
	}

	reverseResponse, err := client.ExecuteTool(ctx, reverseRequest)
	if err != nil {
		log.Printf("Text reversal failed: %v", err)
	} else {
		var result map[string]interface{}
		json.Unmarshal(reverseResponse.Result, &result)
		log.Printf("Text reversal result: 'Hello, MCP World!' -> '%v'", result["result"])

		// Report the cost
		cost := &protocol.CostReport{
			OperationID:   reverseRequest.ExecutionID,
			StartTime:     reverseResponse.StartTime,
			EndTime:       reverseResponse.EndTime,
			ComputeCost:   0.002,
			MemoryCost:    1024,
			LatencyCost:   time.Duration(reverseResponse.EndTime.Unix()-reverseResponse.StartTime.Unix()) * time.Second,
			TokensCost:    10,
			APICallsCost:  1,
			FinancialCost: 0.0002,
		}
		client.ReportCost(ctx, cost)
	}

	// Get text length
	lengthRequest := &protocol.ToolExecutionRequest{
		ToolID:      "text-processor",
		ExecutionID: fmt.Sprintf("text-length-%d", time.Now().UnixNano()),
		Parameters: map[string]interface{}{
			"operation": "length",
			"text":      "Meta-Coordination Protocol",
		},
	}

	lengthResponse, err := client.ExecuteTool(ctx, lengthRequest)
	if err != nil {
		log.Printf("Text length failed: %v", err)
	} else {
		var result map[string]interface{}
		json.Unmarshal(lengthResponse.Result, &result)
		log.Printf("Text length result: 'Meta-Coordination Protocol' has %v characters", result["result"])

		// Report the cost
		cost := &protocol.CostReport{
			OperationID:   lengthRequest.ExecutionID,
			StartTime:     lengthResponse.StartTime,
			EndTime:       lengthResponse.EndTime,
			ComputeCost:   0.002,
			MemoryCost:    1024,
			LatencyCost:   time.Duration(lengthResponse.EndTime.Unix()-lengthResponse.StartTime.Unix()) * time.Second,
			TokensCost:    10,
			APICallsCost:  1,
			FinancialCost: 0.0002,
		}
		client.ReportCost(ctx, cost)
	}
}

func queryCostAnalytics(ctx context.Context, client *client.Client) {
	log.Println("\n=== Querying Cost Analytics ===")

	// Query costs from the last hour
	query := &protocol.CostQuery{
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
		GroupBy:   "operation",
		Limit:     20,
	}

	analysis, err := client.QueryCosts(ctx, query)
	if err != nil {
		log.Printf("Cost query failed: %v", err)
		return
	}

	log.Println("Cost Analysis:")
	log.Printf("  Total Operations: %d", len(analysis.Breakdown))
	log.Printf("  Total Compute Cost: %.6f", analysis.TotalCost.ComputeCost)
	log.Printf("  Total Memory Cost: %d bytes", analysis.TotalCost.MemoryCost)
	log.Printf("  Total Latency: %v", analysis.TotalCost.LatencyCost)
	log.Printf("  Total Tokens: %d", analysis.TotalCost.TokensCost)
	log.Printf("  Total API Calls: %d", analysis.TotalCost.APICallsCost)
	log.Printf("  Total Financial Cost: $%.6f", analysis.TotalCost.FinancialCost)

	if len(analysis.Breakdown) > 0 {
		log.Println("\n  Breakdown by Operation:")
		for i, cost := range analysis.Breakdown {
			if i >= 5 { // Limit to first 5 for readability
				break
			}
			log.Printf("    %s: Compute=%.6f, Financial=$%.6f",
				cost.OperationID, cost.ComputeCost, cost.FinancialCost)
		}
	}

	if len(analysis.Recommendations) > 0 {
		log.Println("\n  Recommendations:")
		for _, rec := range analysis.Recommendations {
			log.Printf("    - %s", rec)
		}
	}
}