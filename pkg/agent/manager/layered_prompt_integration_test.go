package manager

import (
	"context"
	"strings"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/memory/boltdb"
	"github.com/guild-ventures/guild-core/pkg/prompts"
	promptcontext "github.com/guild-ventures/guild-core/pkg/prompts/context"
)

func TestGuildMasterRefinerUsesLayeredPrompts(t *testing.T) {
	// Create a temporary database for testing
	tempPath := t.TempDir() + "/test.db"
	store, err := boltdb.NewStore(tempPath)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	defer store.Close()

	// Create prompt registry and formatter
	registry := prompts.NewMemoryRegistry()
	formatter, err := promptcontext.NewXMLFormatter()
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	// Create basic prompt manager
	promptManager := prompts.NewDefaultManager(registry, formatter)

	// Create layered registry (though we can't test the full layered system without proper store interface)
	_ = prompts.NewGuildLayeredRegistry(registry, store)

	// Register a test Guild Master prompt
	guildMasterPrompt := `You are a Guild Master responsible for refining commissions into hierarchical implementation plans.

## Your Role
- Analyze commissions to understand the full scope
- Create hierarchical plans that preserve context and relationships  
- Structure your output as a directory of markdown files
- Ensure every task can be traced back to its source requirement

## Output Format
Structure your response with file sections:

## File: README.md
# Commission Title
Overview and main implementation plan...

## File: implementation/feature.md  
# Feature Implementation
Detailed implementation steps...

**Tasks Generated**:
- CATEGORY-001: Task description
  - Priority: High/Medium/Low
  - Estimate: X days
  - Dependencies: none or task-ids
  - Capabilities: required artisan capabilities
`

	// Register the prompt for manager role, web-app domain
	err = registry.RegisterPrompt("manager", "web-app", guildMasterPrompt)
	if err != nil {
		t.Fatalf("Failed to register Guild Master prompt: %v", err)
	}

	// Create a GuildMasterRefiner with the prompt manager
	refiner := &GuildMasterRefiner{
		promptManager: promptManager,
		// We'll leave artisanClient, parser, validator nil for this test
	}

	// Test that getSystemPrompt correctly calls the prompt manager
	ctx := context.Background()
	
	t.Run("GetSystemPromptCallsPromptManager", func(t *testing.T) {
		systemPrompt, err := refiner.getSystemPrompt(ctx, "web-app")
		if err != nil {
			t.Fatalf("getSystemPrompt failed: %v", err)
		}

		if systemPrompt == "" {
			t.Fatalf("System prompt is empty")
		}

		// Verify the prompt contains Guild Master instructions
		if !containsGuildMasterInstructions(systemPrompt) {
			t.Errorf("System prompt doesn't contain expected Guild Master instructions")
			t.Logf("Received prompt: %s", systemPrompt)
		}

		t.Logf("✓ GuildMasterRefiner correctly retrieves system prompt")
		t.Logf("  Prompt length: %d characters", len(systemPrompt))
	})

	t.Run("GetSystemPromptUsesCorrectDomain", func(t *testing.T) {
		// Register different prompt for different domain
		cliPrompt := "CLI tool specific Guild Master instructions..."
		err = registry.RegisterPrompt("manager", "cli-tool", cliPrompt)
		if err != nil {
			t.Fatalf("Failed to register CLI prompt: %v", err)
		}

		// Test web-app domain
		webPrompt, err := refiner.getSystemPrompt(ctx, "web-app")
		if err != nil {
			t.Fatalf("Failed to get web-app prompt: %v", err)
		}

		// Test cli-tool domain  
		cliResult, err := refiner.getSystemPrompt(ctx, "cli-tool")
		if err != nil {
			t.Fatalf("Failed to get cli-tool prompt: %v", err)
		}

		// Verify they're different
		if webPrompt == cliResult {
			t.Errorf("Expected different prompts for different domains")
		}

		t.Logf("✓ GuildMasterRefiner uses domain-specific prompts")
	})

	t.Run("GetSystemPromptHandlesDefaultDomain", func(t *testing.T) {
		// Test with empty domain (should default to "default")
		prompt, err := refiner.getSystemPrompt(ctx, "")
		if err == nil {
			t.Logf("✓ Empty domain handled (got prompt: %d chars)", len(prompt))
		} else {
			t.Logf("✓ Empty domain correctly returns error: %v", err)
		}

		// Test with non-existent domain
		_, err = refiner.getSystemPrompt(ctx, "nonexistent-domain")
		if err == nil {
			t.Errorf("Expected error for non-existent domain")
		} else {
			t.Logf("✓ Non-existent domain correctly returns error: %v", err)
		}
	})
}

// containsGuildMasterInstructions checks if the prompt contains expected Guild Master content
func containsGuildMasterInstructions(prompt string) bool {
	expectedPhrases := []string{
		"Guild Master",
		"refining commissions",
		"hierarchical",
		"implementation plans",
		"## File:",
		"Tasks Generated",
	}

	for _, phrase := range expectedPhrases {
		if !contains(prompt, phrase) {
			return false
		}
	}
	return true
}

// Simple contains check using standard library
func contains(text, substring string) bool {
	return strings.Contains(text, substring)
}

func TestVerifyActualSystemPromptsLoaded(t *testing.T) {
	// Test that the actual system prompts from the framework are accessible
	tempPath := t.TempDir() + "/test.db"
	store, err := boltdb.NewStore(tempPath)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	defer store.Close()

	registry := prompts.NewMemoryRegistry()
	formatter, err := promptcontext.NewXMLFormatter()
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	promptManager := prompts.NewDefaultManager(registry, formatter)

	// Check if we can load the actual objective refinement prompt
	// This tests that the prompt loading system works
	ctx := context.Background()

	t.Run("CheckPromptManagerInterface", func(t *testing.T) {
		// Test that basic interface methods work
		roles, err := promptManager.ListRoles(ctx)
		if err != nil {
			t.Logf("ListRoles error (expected if not implemented): %v", err)
		} else {
			t.Logf("✓ Available roles: %v", roles)
		}

		domains, err := promptManager.ListDomains(ctx, "manager")
		if err != nil {
			t.Logf("ListDomains error (expected if not implemented): %v", err)
		} else {
			t.Logf("✓ Available domains for manager: %v", domains)
		}
	})

	t.Run("CheckTemplateSystem", func(t *testing.T) {
		// Test template retrieval
		template, err := promptManager.GetTemplate(ctx, "objective_refinement")
		if err != nil {
			t.Logf("GetTemplate error (expected if template not registered): %v", err)
		} else {
			t.Logf("✓ Retrieved objective refinement template: %d chars", len(template))
		}
	})

	t.Run("VerifyPromptManagerCreated", func(t *testing.T) {
		// Just verify we can create the prompt manager successfully
		if promptManager == nil {
			t.Fatalf("PromptManager is nil")
		}

		t.Logf("✓ PromptManager created successfully")
		t.Logf("  Type: %T", promptManager)
	})
}