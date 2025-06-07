//go:build example
// +build example

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/guild-ventures/guild-core/pkg/memory/boltdb"
	"github.com/guild-ventures/guild-core/pkg/prompts"
)

// Example demonstrating the Guild layered prompt system
func main() {
	fmt.Println("🏰 Guild Layered Prompt System Example")
	fmt.Println("====================================")

	// 1. Initialize storage (BoltDB)
	store, err := boltdb.NewStore("/tmp/guild_prompts_example.db")
	if err != nil {
		log.Fatal("Failed to create store:", err)
	}
	defer store.Close()

	// 2. Create a mock registry and manager
	mockRegistry := &MockRegistry{}
	mockBaseManager := &MockManager{}

	// 3. Create the layered manager
	layeredManager := prompts.NewGuildLayeredManager(
		mockBaseManager,
		store,
		mockRegistry,
		nil, // No RAG retriever for this example
		4000, // 4k token budget
	)

	ctx := context.Background()

	// 4. Set up some example prompt layers
	fmt.Println("\n📝 Setting up prompt layers...")

	// Platform layer (global)
	platformPrompt := prompts.SystemPrompt{
		Layer:   prompts.LayerPlatform,
		Content: "You are part of the Guild Framework. Always maintain professionalism and use Guild terminology.",
		Version: 1,
		Updated: time.Now(),
		Metadata: map[string]interface{}{
			"source": "example",
		},
	}

	if err := layeredManager.SetPromptLayer(ctx, platformPrompt); err != nil {
		log.Printf("Failed to set platform prompt: %v", err)
	} else {
		fmt.Println("✅ Platform layer set")
	}

	// Session layer (user preferences)
	sessionPrompt := prompts.SystemPrompt{
		Layer:     prompts.LayerSession,
		SessionID: "demo_session_123",
		Content:   "The user prefers detailed explanations with practical examples. Keep responses concise but informative.",
		Version:   1,
		Updated:   time.Now(),
		Metadata: map[string]interface{}{
			"user_preference": "detailed",
			"source":          "example",
		},
	}

	if err := layeredManager.SetPromptLayer(ctx, sessionPrompt); err != nil {
		log.Printf("Failed to set session prompt: %v", err)
	} else {
		fmt.Println("✅ Session layer set")
	}

	// 5. Build a layered prompt
	fmt.Println("\n🔨 Building layered prompt...")

	turnContext := prompts.TurnContext{
		UserMessage:  "Explain how the Guild layered prompt system works",
		TaskID:       "DEMO-001",
		CommissionID: "EXAMPLE-COMMISSION",
		Urgency:      "medium",
		Instructions: []string{
			"Include examples",
			"Use Guild terminology",
		},
	}

	layeredPrompt, err := layeredManager.BuildLayeredPrompt(
		ctx,
		"demo-artisan-001",
		"demo_session_123",
		turnContext,
	)
	if err != nil {
		log.Fatal("Failed to build layered prompt:", err)
	}

	// 6. Display the results
	fmt.Println("\n🎯 Layered Prompt Results:")
	fmt.Printf("Artisan: %s\n", layeredPrompt.ArtisanID)
	fmt.Printf("Session: %s\n", layeredPrompt.SessionID)
	fmt.Printf("Layers: %d\n", len(layeredPrompt.Layers))
	fmt.Printf("Token Count: %d\n", layeredPrompt.TokenCount)
	fmt.Printf("Truncated: %v\n", layeredPrompt.Truncated)
	fmt.Printf("Cache Key: %s\n", layeredPrompt.CacheKey)

	fmt.Println("\n📋 Layer Breakdown:")
	for i, layer := range layeredPrompt.Layers {
		fmt.Printf("%d. %s (Priority: %d)\n", i+1, layer.Layer, layer.Priority)
		contentPreview := layer.Content
		if len(contentPreview) > 100 {
			contentPreview = contentPreview[:100] + "..."
		}
		fmt.Printf("   Content: %s\n", contentPreview)
	}

	fmt.Println("\n📄 Compiled Prompt:")
	fmt.Println("==================")
	fmt.Println(layeredPrompt.Compiled)
	fmt.Println("==================")

	// 7. List all prompt layers
	fmt.Println("\n📂 All Prompt Layers:")
	layers, err := layeredManager.ListPromptLayers(ctx, "demo-artisan-001", "demo_session_123")
	if err != nil {
		log.Printf("Failed to list layers: %v", err)
	} else {
		for i, layer := range layers {
			fmt.Printf("%d. %s (%s)\n", i+1, layer.Layer, layer.Updated.Format(time.RFC3339))
		}
	}

	// 8. Test cache invalidation
	fmt.Println("\n🗑️  Testing cache invalidation...")
	if err := layeredManager.InvalidateCache(ctx, "demo-artisan-001", "demo_session_123"); err != nil {
		log.Printf("Failed to invalidate cache: %v", err)
	} else {
		fmt.Println("✅ Cache invalidated successfully")
	}

	fmt.Println("\n🎉 Guild layered prompt system example completed!")
}

// Mock implementations for the example

type MockRegistry struct{}

func (m *MockRegistry) RegisterPrompt(role, domain, prompt string) error {
	return nil
}

func (m *MockRegistry) RegisterTemplate(name, template string) error {
	return nil
}

func (m *MockRegistry) GetPrompt(role, domain string) (string, error) {
	return "Mock role prompt for " + role, nil
}

func (m *MockRegistry) GetTemplate(name string) (string, error) {
	return "Mock template: " + name, nil
}

func (m *MockRegistry) RegisterLayeredPrompt(layer prompts.PromptLayer, identifier string, prompt prompts.SystemPrompt) error {
	return nil
}

func (m *MockRegistry) GetLayeredPrompt(layer prompts.PromptLayer, identifier string) (*prompts.SystemPrompt, error) {
	return nil, prompts.ErrLayerNotFound
}

func (m *MockRegistry) ListLayeredPrompts(layer prompts.PromptLayer) ([]prompts.SystemPrompt, error) {
	return nil, nil
}

func (m *MockRegistry) DeleteLayeredPrompt(layer prompts.PromptLayer, identifier string) error {
	return nil
}

func (m *MockRegistry) GetDefaultPrompts(layer prompts.PromptLayer) ([]prompts.SystemPrompt, error) {
	return nil, nil
}

type MockManager struct{}

func (m *MockManager) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	return fmt.Sprintf("You are a %s artisan specialized in %s development for the Guild.", role, domain), nil
}

func (m *MockManager) GetTemplate(ctx context.Context, templateName string) (string, error) {
	return "Mock template content", nil
}

func (m *MockManager) FormatContext(ctx context.Context, context prompts.Context) (string, error) {
	return "Mock formatted context", nil
}

func (m *MockManager) ListRoles(ctx context.Context) ([]string, error) {
	return []string{"backend", "frontend", "devops"}, nil
}

func (m *MockManager) ListDomains(ctx context.Context, role string) ([]string, error) {
	return []string{"web-app", "cli-tool", "microservice"}, nil
}

type MockFormatter struct{}

func (m *MockFormatter) FormatAsXML(ctx prompts.Context) (string, error) {
	return "<context>Mock XML context</context>", nil
}

func (m *MockFormatter) FormatAsMarkdown(ctx prompts.Context) (string, error) {
	return "# Mock Markdown Context", nil
}

func (m *MockFormatter) OptimizeForTokens(content string, maxTokens int) (string, error) {
	if len(content) > maxTokens*4 {
		return content[:maxTokens*4] + "...", nil
	}
	return content, nil
}
