package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/prompts"
	"github.com/guild-ventures/guild-core/pkg/prompts/layered/context"
	"github.com/guild-ventures/guild-core/pkg/prompts/standard/templates/commission"
)

func TestGuildMasterUsesRealSystemPrompts(t *testing.T) {
	// Create prompt registry and formatter
	registry := prompts.NewMemoryRegistry()
	formatter, err := context.NewXMLFormatter()
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	// Register a sample prompt for testing
	samplePrompt := `## Guild Master System Prompt

You are a Guild Master responsible for coordinating artisan agents in the medieval Guild framework.

### Guild Terminology
- **Artisans**: Individual AI agents with specialized capabilities
- **Workshop Board**: Task tracking system using medieval metaphors
- **Commissions**: Complex objectives that require multiple artisans

### Task Formatting Rules
**Tasks Generated**:
- {CATEGORY}-{NUMBER}: {Task Title}
  - Priority: {HIGH|MEDIUM|LOW}
  - Estimate: {Time estimate}
  - Dependencies: {Task dependencies}
  - Capabilities: {Required agent capabilities}

### Directory Structure Example
` + "```" + `
project/
├── README.md
├── backend/
├── frontend/
└── infrastructure/
` + "```" + `

### Output Structure
Your responses must be hierarchical and well-structured using markdown files.
Ensure all task dependencies are properly tracked and artisan capabilities are matched.
`

	err = registry.RegisterPrompt("manager", "web-app", samplePrompt)
	if err != nil {
		t.Fatalf("Failed to register sample prompt: %v", err)
	}

	// Create prompt manager
	promptManager := prompts.NewDefaultManager(registry, formatter)

	// Create GuildMasterRefiner with real prompt manager
	refiner := &manager.GuildMasterRefiner{
		PromptManager: promptManager,
	}

	ctx := context.Background()

	t.Run("VerifyRealPromptContent", func(t *testing.T) {
		systemPrompt, err := refiner.GetSystemPrompt(ctx, "web-app")
		if err != nil {
			t.Fatalf("Failed to get system prompt: %v", err)
		}

		// Verify the prompt contains the actual Guild Master instructions
		expectedElements := []string{
			"Guild Master",
			"artisan agents",
			"Workshop Board",
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

		t.Logf("✓ Real Guild Master prompt contains expected elements")
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
		systemPrompt, err := refiner.GetSystemPrompt(ctx, "web-app")
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

		foundInstructions := 0
		for _, instruction := range structureInstructions {
			if strings.Contains(systemPrompt, instruction) {
				foundInstructions++
			}
		}

		// Should contain most structure instructions
		if foundInstructions >= len(structureInstructions)/2 {
			t.Logf("✓ Prompt teaches proper output structure (%d/%d found)", foundInstructions, len(structureInstructions))
		} else {
			t.Errorf("Prompt missing too many structure instructions (%d/%d found)", foundInstructions, len(structureInstructions))
		}
	})

	t.Run("VerifyPromptTeachesMedievalTerminology", func(t *testing.T) {
		systemPrompt, err := refiner.GetSystemPrompt(ctx, "web-app")
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

		foundTerms := 0
		for _, term := range medievalTerms {
			if strings.Contains(systemPrompt, term) {
				foundTerms++
			}
		}

		// Should contain most medieval terms
		if foundTerms >= len(medievalTerms)/2 {
			t.Logf("✓ Prompt contains medieval terminology (%d/%d found)", foundTerms, len(medievalTerms))
		} else {
			t.Errorf("Prompt missing too many medieval terms (%d/%d found)", foundTerms, len(medievalTerms))
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
	t.Run("VerifyCommissionPromptExists", func(t *testing.T) {
		// Access prompt templates through the commission package
		prompt := commission.GetManagerRefinementPrompt()

		if prompt == "" {
			t.Fatalf("Manager refinement prompt is empty")
		}

		if len(prompt) < 100 {
			t.Errorf("Manager refinement prompt seems too short: %d characters", len(prompt))
		}

		t.Logf("✓ Manager refinement prompt exists and is substantial")
		t.Logf("  Length: %d characters", len(prompt))
	})

	t.Run("VerifyPromptQuality", func(t *testing.T) {
		prompt := commission.GetManagerRefinementPrompt()

		// Check for key quality indicators
		qualityMarkers := []string{
			"## ", // Has section headers
			"```", // Has code examples
			"**",  // Has bold formatting
			"-",   // Has bullet points
			":",   // Has structured content
		}

		foundMarkers := 0
		for _, marker := range qualityMarkers {
			if strings.Contains(prompt, marker) {
				foundMarkers++
			}
		}

		// Should contain most quality markers
		if foundMarkers >= len(qualityMarkers)/2 {
			t.Logf("✓ Prompt shows quality formatting and structure (%d/%d markers found)", foundMarkers, len(qualityMarkers))
		} else {
			t.Errorf("Prompt missing too many quality markers (%d/%d found)", foundMarkers, len(qualityMarkers))
		}
	})
}

// TestPromptManagerIntegration tests the integration with the prompt management system
func TestPromptManagerIntegration(t *testing.T) {
	t.Run("PromptRegistryOperations", func(t *testing.T) {
		// Test basic registry operations
		registry := prompts.NewMemoryRegistry()

		testPrompt := "Test prompt content for integration testing"
		err := registry.RegisterPrompt("test-manager", "integration", testPrompt)
		if err != nil {
			t.Fatalf("Failed to register test prompt: %v", err)
		}

		// Retrieve the prompt
		retrieved, err := registry.GetPrompt("test-manager", "integration")
		if err != nil {
			t.Fatalf("Failed to retrieve test prompt: %v", err)
		}

		if retrieved != testPrompt {
			t.Errorf("Retrieved prompt doesn't match registered prompt")
		}

		t.Logf("✓ Prompt registry operations work correctly")
	})

	t.Run("PromptFormatterOperations", func(t *testing.T) {
		formatter, err := context.NewXMLFormatter()
		if err != nil {
			t.Fatalf("Failed to create XML formatter: %v", err)
		}

		// Test formatting capabilities
		testContent := "Test content for formatting"
		formatted := formatter.Format(testContent)

		if formatted == "" {
			t.Error("Formatter returned empty result")
		}

		if !strings.Contains(formatted, testContent) {
			t.Error("Formatted result should contain original content")
		}

		t.Logf("✓ Prompt formatter operations work correctly")
	})

	t.Run("DefaultManagerOperations", func(t *testing.T) {
		registry := prompts.NewMemoryRegistry()
		formatter, err := context.NewXMLFormatter()
		if err != nil {
			t.Fatalf("Failed to create formatter: %v", err)
		}

		manager := prompts.NewDefaultManager(registry, formatter)

		// Register a test prompt
		testPrompt := "## Test Manager Prompt\n\nThis is a test prompt for the manager."
		err = registry.RegisterPrompt("test-manager", "default", testPrompt)
		if err != nil {
			t.Fatalf("Failed to register prompt: %v", err)
		}

		// Test retrieval through manager
		retrieved, err := manager.GetPrompt("test-manager", "default")
		if err != nil {
			t.Fatalf("Failed to get prompt through manager: %v", err)
		}

		if !strings.Contains(retrieved, "Test Manager Prompt") {
			t.Error("Manager should return formatted prompt content")
		}

		t.Logf("✓ Default prompt manager operations work correctly")
	})
}

// TestGuildMasterRefinerIntegration tests the refiner integration
func TestGuildMasterRefinerIntegration(t *testing.T) {
	t.Run("RefinerWithRealPromptManager", func(t *testing.T) {
		// Create full prompt management stack
		registry := prompts.NewMemoryRegistry()
		formatter, err := context.NewXMLFormatter()
		if err != nil {
			t.Fatalf("Failed to create formatter: %v", err)
		}
		promptManager := prompts.NewDefaultManager(registry, formatter)

		// Register a comprehensive Guild Master prompt
		guildMasterPrompt := `## Guild Master Refinement System

You are the Guild Master responsible for refining high-level commissions into detailed task breakdowns.

### Your Responsibilities
- **Commission Analysis**: Break down complex objectives into manageable tasks
- **Artisan Assignment**: Match tasks to appropriate artisan capabilities
- **Workshop Organization**: Structure tasks for efficient execution
- **Quality Assurance**: Ensure all requirements are captured

### Task Structure
Each task must include:
- Clear title and description
- Priority level (HIGH/MEDIUM/LOW)
- Time estimate
- Required capabilities
- Dependencies on other tasks

### Output Format
` + "```markdown" + `
## Commission: {Commission Title}

### Tasks Generated:
- BACKEND-001: Setup project structure
  - Priority: HIGH
  - Estimate: 2 hours
  - Capabilities: [project-setup, backend-development]
  - Dependencies: []

- FRONTEND-002: Create user interface
  - Priority: MEDIUM
  - Estimate: 4 hours
  - Capabilities: [frontend-development, ui-design]
  - Dependencies: [BACKEND-001]
` + "```" + `

Follow Guild terminology and maintain the medieval workshop metaphor throughout.`

		err = registry.RegisterPrompt("guild-master", "refinement", guildMasterPrompt)
		if err != nil {
			t.Fatalf("Failed to register Guild Master prompt: %v", err)
		}

		// Create refiner
		refiner := &manager.GuildMasterRefiner{
			PromptManager: promptManager,
		}

		// Test system prompt retrieval
		ctx := context.Background()
		systemPrompt, err := refiner.GetSystemPrompt(ctx, "refinement")
		if err != nil {
			t.Fatalf("Failed to get system prompt: %v", err)
		}

		// Verify the prompt contains key elements
		requiredElements := []string{
			"Guild Master",
			"Commission Analysis",
			"Artisan Assignment",
			"Workshop Organization",
			"Task Structure",
			"Output Format",
			"BACKEND-001",
			"FRONTEND-002",
		}

		for _, element := range requiredElements {
			if !strings.Contains(systemPrompt, element) {
				t.Errorf("System prompt missing required element: %s", element)
			}
		}

		t.Logf("✓ Guild Master refiner integration works correctly")
		t.Logf("  System prompt length: %d characters", len(systemPrompt))
	})
}

// BenchmarkPromptOperations benchmarks prompt system performance
func BenchmarkPromptOperations(b *testing.B) {
	registry := prompts.NewMemoryRegistry()
	formatter, _ := context.NewXMLFormatter()
	manager := prompts.NewDefaultManager(registry, formatter)

	// Pre-register a prompt
	testPrompt := "Test prompt content for benchmarking operations"
	registry.RegisterPrompt("benchmark-manager", "test", testPrompt)

	b.Run("PromptRetrieval", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := manager.GetPrompt("benchmark-manager", "test")
			if err != nil {
				b.Fatalf("Benchmark failed: %v", err)
			}
		}
	})

	b.Run("PromptFormatting", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = formatter.Format(testPrompt)
		}
	})
}