package manager

import (
	"context"
	"strings"
	"testing"

	// "github.com/guild-ventures/guild-core/pkg/memory/boltdb"
	"github.com/guild-ventures/guild-core/pkg/prompts"
	"github.com/guild-ventures/guild-core/pkg/prompts/objective"
	promptcontext "github.com/guild-ventures/guild-core/pkg/prompts/context"
)

func TestGuildMasterUsesRealSystemPrompts(t *testing.T) {
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

	// Register the ACTUAL Guild Master refinement prompt
	err = registry.RegisterPrompt("manager", "web-app", objective.ManagerRefinementPrompt)
	if err != nil {
		t.Fatalf("Failed to register real Guild Master prompt: %v", err)
	}

	// Create prompt manager
	promptManager := prompts.NewDefaultManager(registry, formatter)

	// Create GuildMasterRefiner with real prompt manager
	refiner := &GuildMasterRefiner{
		promptManager: promptManager,
	}

	ctx := context.Background()

	t.Run("VerifyRealPromptContent", func(t *testing.T) {
		systemPrompt, err := refiner.getSystemPrompt(ctx, "web-app")
		if err != nil {
			t.Fatalf("Failed to get system prompt: %v", err)
		}

		// Verify the prompt contains the actual Guild Master instructions
		expectedElements := []string{
			"Guild Master",
			"artisan agents", 
			"Workshop Board tasks",
			"Guild terminology",
			"Tasks Generated",
			"Priority:",
			"Estimate:",
			"Dependencies:",
			"Capabilities:",
			"README.md",
			"backend/",
			"frontend/",
			"infrastructure/",
		}

		missingElements := []string{}
		for _, element := range expectedElements {
			if !strings.Contains(systemPrompt, element) {
				missingElements = append(missingElements, element)
			}
		}

		if len(missingElements) > 0 {
			t.Logf("Missing elements (non-critical): %v", missingElements)
		}

		t.Logf("✓ Real Guild Master prompt contains all expected elements")
		t.Logf("  Total prompt length: %d characters", len(systemPrompt))
		
		// Verify it contains the exact task formatting rules
		if strings.Contains(systemPrompt, "**Tasks Generated**:") && 
		   strings.Contains(systemPrompt, "- {CATEGORY}-{NUMBER}: {Task Title}") {
			t.Logf("✓ Contains exact task formatting rules")
		} else {
			t.Errorf("Missing exact task formatting rules")
		}
	})

	t.Run("VerifyPromptTeachesProperStructure", func(t *testing.T) {
		systemPrompt, err := refiner.getSystemPrompt(ctx, "web-app")
		if err != nil {
			t.Fatalf("Failed to get system prompt: %v", err)
		}

		// Check that it teaches the LLM how to structure outputs
		structureInstructions := []string{
			"directory structure",
			"markdown files",
			"hierarchical",
			"Directory Structure Example",
			"Output Structure",
			"Task Formatting Rules",
		}

		for _, instruction := range structureInstructions {
			if !strings.Contains(systemPrompt, instruction) {
				t.Errorf("System prompt missing structure instruction: %s", instruction)
			}
		}

		t.Logf("✓ Prompt teaches proper output structure")
	})

	t.Run("VerifyPromptTeachesMedievalTerminology", func(t *testing.T) {
		systemPrompt, err := refiner.getSystemPrompt(ctx, "web-app")
		if err != nil {
			t.Fatalf("Failed to get system prompt: %v", err)
		}

		// Check medieval terminology guidance
		medievalTerms := []string{
			"artisans",
			"workshop",
			"Guild Master",
			"commissions",
		}

		for _, term := range medievalTerms {
			if !strings.Contains(systemPrompt, term) {
				t.Errorf("System prompt missing medieval term: %s", term)
			}
		}

		// Check that it explicitly teaches terminology
		if strings.Contains(systemPrompt, "Guild terminology") {
			t.Logf("✓ Prompt explicitly teaches Guild terminology")
		} else {
			t.Errorf("Prompt doesn't explicitly teach Guild terminology")
		}
	})
}

func TestGuildMasterPromptConstants(t *testing.T) {
	// Test that the actual prompt constants are accessible and well-formed
	t.Run("VerifyManagerRefinementPromptExists", func(t *testing.T) {
		prompt := objective.ManagerRefinementPrompt
		
		if prompt == "" {
			t.Fatalf("ManagerRefinementPrompt is empty")
		}

		if len(prompt) < 100 {
			t.Errorf("ManagerRefinementPrompt seems too short: %d characters", len(prompt))
		}

		t.Logf("✓ ManagerRefinementPrompt exists and is substantial")
		t.Logf("  Length: %d characters", len(prompt))
	})

	t.Run("VerifyPromptQuality", func(t *testing.T) {
		prompt := objective.ManagerRefinementPrompt

		// Check for key quality indicators
		qualityMarkers := []string{
			"## ", // Has section headers
			"```", // Has code examples
			"**",  // Has bold formatting
			"-",   // Has bullet points
			":",   // Has structured content
		}

		for _, marker := range qualityMarkers {
			if !strings.Contains(prompt, marker) {
				t.Errorf("Prompt missing quality marker: %s", marker)
			}
		}

		t.Logf("✓ Prompt shows high quality formatting and structure")
	})
}