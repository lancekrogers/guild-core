// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/commission"
	"github.com/guild-framework/guild-core/pkg/lsp"
	"github.com/guild-framework/guild-core/pkg/memory"
	"github.com/guild-framework/guild-core/pkg/providers"
	"github.com/guild-framework/guild-core/pkg/suggestions"
	"github.com/guild-framework/guild-core/pkg/templates"
	"github.com/guild-framework/guild-core/pkg/tools"
	fsTool "github.com/guild-framework/guild-core/tools/fs"
)

func main() {
	ctx := context.Background()

	fmt.Println("🤖 Guild Agent-Suggestion Integration Demo")
	fmt.Println("==========================================")

	// Step 1: Set up dependencies
	fmt.Println("\n📋 Step 1: Setting up dependencies...")

	// Create mock dependencies (in real usage, these would be properly configured)
	llmClient := &MockLLMClient{}
	memoryManager := &MockMemoryManager{}
	toolRegistry := tools.NewToolRegistry() // This returns *ToolRegistry that implements Registry interface
	commissionManager := &MockCommissionManager{}
	costManager := &MockCostManager{}

	// Register some tools using the Registry interface
	globTool := fsTool.NewGlobTool("/workspace")
	if err := toolRegistry.RegisterTool(globTool.Name(), globTool); err != nil {
		log.Printf("Warning: Failed to register glob tool: %v", err)
	}
	// Note: gitTool.RegisterGitTools expects concrete ToolRegistry, use direct registration
	// For demo, we'll register tools individually using the interface

	fmt.Printf("✅ Registered %d tools in registry\n", len(toolRegistry.ListTools()))

	// Step 2: Create suggestion-aware agent factory
	fmt.Println("\n🏭 Step 2: Creating suggestion-aware agent factory...")

	factory := core.NewSuggestionAwareAgentFactory(
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
		costManager,
	)

	// Optional: Configure with LSP and template managers
	templateManager := &MockTemplateManager{}
	lspManager, _ := lsp.NewManager("") // This would fail but we'll handle it gracefully

	if err := factory.ConfigureSuggestionProviders(templateManager, lspManager); err != nil {
		log.Printf("Warning: Could not configure all providers: %v", err)
	}

	fmt.Println("✅ Factory created with suggestion support")

	// Step 3: Create a suggestion-aware agent
	fmt.Println("\n🎭 Step 3: Creating suggestion-aware core...")

	enhancedAgent := factory.CreateWorkerAgentWithCapabilities(
		"demo-agent-001",
		"Demo Agent with Suggestions",
		[]string{"file_operations", "git_operations", "code_analysis"},
	)

	fmt.Printf("✅ Created agent: %s (%s)\n", enhancedAgent.GetName(), enhancedAgent.GetID())

	// Step 4: Create chat integration handler
	fmt.Println("\n💬 Step 4: Setting up chat integration...")

	chatHandler := core.NewChatSuggestionHandler(enhancedAgent)
	config := core.DefaultChatSuggestionConfig()

	fmt.Println("✅ Chat integration handler ready")

	// Step 5: Demonstrate suggestion functionality
	fmt.Println("\n🔮 Step 5: Demonstrating suggestions...")

	// Scenario 1: File search request
	fmt.Println("\n--- Scenario 1: File Search ---")
	request1 := core.SuggestionRequest{
		Message:        "I need to find all Go files in the project",
		MaxSuggestions: 5,
		MinConfidence:  0.4,
	}

	chatHandler.ApplyConfig(config, &request1)
	response1, err := chatHandler.GetSuggestions(ctx, request1)
	if err != nil {
		log.Printf("Error getting suggestions: %v", err)
	} else {
		fmt.Printf("Got %d suggestions:\n", len(response1.Suggestions))
		for i, s := range response1.Suggestions {
			fmt.Printf("  %d. %s: %s (%.2f confidence)\n", i+1, s.Display, s.Description, s.Confidence)
		}
	}

	// Scenario 2: Code analysis request
	fmt.Println("\n--- Scenario 2: Code Analysis ---")
	request2 := core.SuggestionRequest{
		Message: "analyze this code for issues",
		FileContext: &suggestions.FileContext{
			FilePath: "/workspace/main.go",
			Line:     10,
			Column:   5,
		},
	}

	response2, err := chatHandler.GetSuggestions(ctx, request2)
	if err != nil {
		log.Printf("Error getting suggestions: %v", err)
	} else {
		fmt.Printf("Got %d suggestions with file context:\n", len(response2.Suggestions))
		for i, s := range response2.Suggestions {
			fmt.Printf("  %d. %s (%s) - %s\n", i+1, s.Content, s.Type, s.Description)
		}
	}

	// Step 6: Demonstrate execution with suggestions
	fmt.Println("\n🚀 Step 6: Execution with suggestions...")

	result, err := chatHandler.ExecuteWithSuggestions(ctx, "find all TODO comments in the codebase", true)
	if err != nil {
		log.Printf("Execution error: %v", err)
	} else {
		fmt.Printf("Execution result: %s\n", result.Response)
		fmt.Printf("Included %d suggestions\n", len(result.Suggestions))

		for i, s := range result.Suggestions {
			if i < 3 { // Show first 3 suggestions
				fmt.Printf("  Suggestion %d: %s\n", i+1, s.Display)
			}
		}
	}

	// Step 7: Filter suggestions by type
	fmt.Println("\n🔍 Step 7: Filtered suggestions...")

	request3 := core.SuggestionRequest{
		Message: "help me with git operations",
		Filter: &suggestions.SuggestionFilter{
			Types: []suggestions.SuggestionType{suggestions.SuggestionTypeTool},
			Tags:  []string{"git"},
		},
	}

	response3, err := chatHandler.GetSuggestions(ctx, request3)
	if err != nil {
		log.Printf("Error getting filtered suggestions: %v", err)
	} else {
		fmt.Printf("Got %d git tool suggestions:\n", len(response3.Suggestions))
		for _, s := range response3.Suggestions {
			fmt.Printf("  🔧 %s - %s\n", s.Content, s.Description)
		}
	}

	fmt.Println("\n🎉 Demo completed successfully!")
	fmt.Println("\n📚 Integration Summary:")
	fmt.Println("  • Agents can now provide context-aware suggestions")
	fmt.Println("  • Chat interfaces can request suggestions through handlers")
	fmt.Println("  • Execution can include suggestions for follow-up actions")
	fmt.Println("  • Tool registry automatically provides tool suggestions")
	fmt.Println("  • LSP integration provides code intelligence suggestions")
	fmt.Println("  • Template system provides reusable prompt suggestions")
}

// Mock implementations for demo purposes

type MockLLMClient struct{}

func (m *MockLLMClient) GenerateCompletion(ctx context.Context, request providers.CompletionRequest) (*providers.CompletionResponse, error) {
	return &providers.CompletionResponse{
		Content: "Mock response: Task completed successfully",
		Usage: providers.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}, nil
}

func (m *MockLLMClient) GetModel() string  { return "mock-model" }
func (m *MockLLMClient) GetMaxTokens() int { return 4096 }

type MockMemoryManager struct{}

func (m *MockMemoryManager) AddMessage(ctx context.Context, conversationID string, message memory.Message) error {
	return nil
}

func (m *MockMemoryManager) GetMessages(ctx context.Context, conversationID string, limit int) ([]memory.Message, error) {
	return []memory.Message{}, nil
}

func (m *MockMemoryManager) CreateConversation(ctx context.Context, conversationID string) error {
	return nil
}

func (m *MockMemoryManager) DeleteConversation(ctx context.Context, conversationID string) error {
	return nil
}

func (m *MockMemoryManager) GetConversations(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

type MockCommissionManager struct{}

func (m *MockCommissionManager) CreateCommission(ctx context.Context, commission commission.Commission) error {
	return nil
}

func (m *MockCommissionManager) GetCommission(ctx context.Context, id string) (commission.Commission, error) {
	return commission.Commission{}, nil
}

type MockCostManager struct{}

func (m *MockCostManager) TrackCost(costType core.CostType, amount float64) error    { return nil }
func (m *MockCostManager) GetCostReport() map[string]interface{}                     { return map[string]interface{}{} }
func (m *MockCostManager) SetBudget(costType core.CostType, amount float64)          {}
func (m *MockCostManager) GetBudgetRemaining(costType core.CostType) float64         { return 100.0 }
func (m *MockCostManager) GetTotalCost() float64                                     { return 0.0 }
func (m *MockCostManager) Reset()                                                    {}
func (m *MockCostManager) ExceedsBudget(costType core.CostType, amount float64) bool { return false }
func (m *MockCostManager) EstimateLLMCost(model string, estimatedTokens int) float64 { return 0.01 }
func (m *MockCostManager) CanAfford(costType core.CostType, amount float64) bool     { return true }
func (m *MockCostManager) RecordLLMCost(model string, promptTokens, completionTokens int, metadata map[string]string) error {
	return nil
}

type MockTemplateManager struct{}

func (m *MockTemplateManager) Create(ctx context.Context, template *templates.Template) error {
	return nil
}

func (m *MockTemplateManager) Get(ctx context.Context, id string) (*templates.Template, error) {
	return nil, nil
}

func (m *MockTemplateManager) GetByName(ctx context.Context, name string) (*templates.Template, error) {
	return nil, nil
}

func (m *MockTemplateManager) List(ctx context.Context, filter *templates.TemplateFilter) ([]*templates.Template, error) {
	return []*templates.Template{}, nil
}

func (m *MockTemplateManager) Update(ctx context.Context, template *templates.Template) error {
	return nil
}
func (m *MockTemplateManager) Delete(ctx context.Context, id string) error { return nil }
func (m *MockTemplateManager) Search(ctx context.Context, query string) ([]*templates.Template, error) {
	return []*templates.Template{}, nil
}

func (m *MockTemplateManager) GetByCategory(ctx context.Context, category string) ([]*templates.Template, error) {
	return []*templates.Template{}, nil
}

func (m *MockTemplateManager) GetMostUsed(ctx context.Context, limit int) ([]*templates.Template, error) {
	return []*templates.Template{}, nil
}

func (m *MockTemplateManager) GetVariables(ctx context.Context, templateID string) ([]*templates.TemplateVariable, error) {
	return []*templates.TemplateVariable{}, nil
}

func (m *MockTemplateManager) SetVariables(ctx context.Context, templateID string, variables []*templates.TemplateVariable) error {
	return nil
}

func (m *MockTemplateManager) Render(ctx context.Context, templateID string, variables map[string]interface{}) (string, error) {
	return "", nil
}

func (m *MockTemplateManager) RenderContent(ctx context.Context, content string, variables map[string]interface{}) (string, error) {
	return content, nil
}

func (m *MockTemplateManager) RecordUsage(ctx context.Context, templateID string, campaignID *string, variables map[string]interface{}, context string) error {
	return nil
}

func (m *MockTemplateManager) GetUsageStats(ctx context.Context, templateID string) (*templates.UsageStats, error) {
	return nil, nil
}

func (m *MockTemplateManager) ListCategories(ctx context.Context) ([]*templates.TemplateCategory, error) {
	return []*templates.TemplateCategory{}, nil
}

func (m *MockTemplateManager) CreateCategory(ctx context.Context, category *templates.TemplateCategory) error {
	return nil
}

func (m *MockTemplateManager) Export(ctx context.Context, templateIDs []string) ([]byte, error) {
	return []byte{}, nil
}

func (m *MockTemplateManager) Import(ctx context.Context, data []byte, overwrite bool) (*templates.ImportResult, error) {
	return nil, nil
}
func (m *MockTemplateManager) InstallBuiltInTemplates(ctx context.Context) error { return nil }
func (m *MockTemplateManager) GetBuiltInTemplates() []*templates.Template {
	return []*templates.Template{}
}

func (m *MockTemplateManager) GetContextualSuggestions(context map[string]interface{}) ([]*templates.Template, error) {
	return []*templates.Template{}, nil
}

func (m *MockTemplateManager) RenderTemplate(templateID string, variables map[string]interface{}) (string, error) {
	return "", nil
}

func (m *MockTemplateManager) SearchTemplates(query string, limit int) ([]*templates.TemplateSearchResult, error) {
	return []*templates.TemplateSearchResult{}, nil
}
