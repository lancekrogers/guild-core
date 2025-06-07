package execution

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPromptBuilder(t *testing.T) {
	builder, err := NewPromptBuilder()
	require.NoError(t, err)
	assert.NotNil(t, builder)

	// Verify all templates are loaded
	layers := []Layer{LayerBase, LayerContext, LayerTask, LayerTool, LayerExecution}
	for _, layer := range layers {
		assert.NotNil(t, builder.GetLayerTemplate(layer), "Template for %s should be loaded", layer)
	}
}

func TestBuildPrompt(t *testing.T) {
	builder, err := NewPromptBuilder()
	require.NoError(t, err)

	testData := map[string]interface{}{
		"AgentName":     "TestArtisan",
		"AgentRole":     "Code generation specialist",
		"Capabilities":  []string{"coding", "testing", "documentation"},
		"GuildID":       "guild-123",
		"ProjectName":   "TestProject",
		"WorkspaceDir":  "/tmp/workspace",
	}

	tests := []struct {
		name   string
		layers []Layer
		verify func(t *testing.T, prompt string)
	}{
		{
			name:   "single layer - base",
			layers: []Layer{LayerBase},
			verify: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "TestArtisan")
				assert.Contains(t, prompt, "Guild Artisan")
				assert.Contains(t, prompt, "coding")
			},
		},
		{
			name:   "multiple layers",
			layers: []Layer{LayerBase, LayerContext},
			verify: func(t *testing.T, prompt string) {
				assert.Contains(t, prompt, "TestArtisan")
				assert.Contains(t, prompt, "---") // Layer separator
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := builder.BuildPrompt(tt.layers, testData)
			require.NoError(t, err)
			assert.NotEmpty(t, prompt)

			// Verify metadata
			assert.Contains(t, prompt, "<!-- Generated:")
			assert.Contains(t, prompt, "<!-- Layers:")

			tt.verify(t, prompt)
		})
	}
}

func TestBuildFullExecutionPrompt(t *testing.T) {
	builder, err := NewPromptBuilder()
	require.NoError(t, err)

	data := ExecutionPromptData{
		Agent: AgentData{
			Name:         "WorkerBee",
			Role:         "Implementation specialist",
			Capabilities: []string{"coding", "testing"},
		},
		Context: ContextData{
			GuildID:            "guild-456",
			ProjectName:        "GuildFramework",
			ProjectDescription: "AI agent orchestration framework",
			WorkspaceDir:       "/workspace/guild",
			TechStack:          "Go",
			Architecture:       "Microservices",
			Dependencies:       "gRPC, NATS",
		},
		Commission: CommissionData{
			Title:           "Implement task execution",
			Description:     "Create the task execution system",
			SuccessCriteria: []string{"Tests pass", "Documentation complete"},
		},
		Task: TaskData{
			Title:          "Create executor package",
			Description:    "Implement the basic task executor",
			Requirements:   []string{"Use interfaces", "Add tests"},
			Constraints:    []string{"Must be thread-safe"},
			Priority:       "High",
			DueDate:        "2024-01-15",
			EstimatedHours: 8.0,
		},
		Tools: []ToolData{
			{
				Name:        "file_system",
				Description: "Read and write files",
				Usage:       "file_system.read(path)",
				Parameters: []ToolParameter{
					{Name: "path", Type: "string", Description: "File path"},
				},
				ReturnType: "string",
				Examples:   []string{"content = file_system.read('/tmp/test.txt')"},
			},
		},
		ToolConfig: ToolConfigData{
			MaxCalls:   100,
			Timeout:    30 * time.Second,
			RateLimits: "10 calls/minute",
		},
		Execution: ExecutionData{
			Phase:                  "Implementation",
			StepNumber:             2,
			TotalSteps:             5,
			StepName:               "Create interfaces",
			StepObjective:          "Define the TaskExecutor interface",
			ExpectedActions:        []string{"Create interface file", "Define methods"},
			SuccessIndicators:      []string{"Interface compiles", "Methods documented"},
			PotentialIssues:        []string{"Circular dependencies"},
			OverallProgress:        40,
			PhaseProgress:          60,
			TimeElapsed:            "30m",
			EstimatedTimeRemaining: "45m",
			PreviousStepResult:     "Package structure created successfully",
			NextSteps:              []string{"Implement interface", "Add tests"},
		},
	}

	prompt, err := builder.BuildFullExecutionPrompt(data)
	require.NoError(t, err)
	assert.NotEmpty(t, prompt)

	// Verify all layers are present
	assert.Contains(t, prompt, "WorkerBee")
	assert.Contains(t, prompt, "Implement task execution")
	assert.Contains(t, prompt, "Create executor package")
	assert.Contains(t, prompt, "file_system")
	assert.Contains(t, prompt, "Step 2 of 5")
}

func TestBuildPlanningPrompt(t *testing.T) {
	builder, err := NewPromptBuilder()
	require.NoError(t, err)

	data := ExecutionPromptData{
		Agent: AgentData{
			Name: "PlannerBot",
			Role: "Planning specialist",
		},
		Commission: CommissionData{
			Title: "Test Commission",
		},
		Task: TaskData{
			Title: "Test Task",
		},
	}

	prompt, err := builder.BuildPlanningPrompt(data)
	require.NoError(t, err)
	assert.NotEmpty(t, prompt)

	// Should not contain execution layer
	assert.NotContains(t, prompt, "Current Execution Phase")
	assert.Contains(t, prompt, "PlannerBot")
}

func TestPromptCache(t *testing.T) {
	cache := NewPromptCache(2, 1*time.Minute)

	// Test set and get
	cache.Set("key1", "prompt1")
	prompt, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "prompt1", prompt)

	// Test missing key
	_, found = cache.Get("missing")
	assert.False(t, found)

	// Test eviction
	cache.Set("key2", "prompt2")
	cache.Set("key3", "prompt3") // Should evict key1

	stats := cache.Stats()
	assert.Equal(t, 2, stats.TotalEntries)

	// Test expiration
	cache = NewPromptCache(10, 1*time.Millisecond)
	cache.Set("expire", "prompt")
	time.Sleep(2 * time.Millisecond)
	_, found = cache.Get("expire")
	assert.False(t, found)
}

func TestCachedPromptBuilder(t *testing.T) {
	builder, err := NewCachedPromptBuilder()
	require.NoError(t, err)

	data := map[string]interface{}{
		"AgentName": "CacheTest",
	}

	// First call should build
	prompt1, err := builder.BuildPromptCached([]Layer{LayerBase}, data)
	require.NoError(t, err)
	assert.NotEmpty(t, prompt1)

	// Second call should use cache
	prompt2, err := builder.BuildPromptCached([]Layer{LayerBase}, data)
	require.NoError(t, err)
	assert.Equal(t, prompt1, prompt2)

	// Check cache stats
	stats := builder.GetCacheStats()
	assert.Equal(t, 1, stats.TotalEntries)
	assert.Equal(t, 1, stats.ValidEntries)
}

func TestGenerateKey(t *testing.T) {
	layers := []Layer{LayerBase, LayerContext}
	data1 := map[string]interface{}{"key": "value1"}
	data2 := map[string]interface{}{"key": "value2"}

	key1, err := GenerateKey(layers, data1)
	require.NoError(t, err)
	assert.NotEmpty(t, key1)

	key2, err := GenerateKey(layers, data2)
	require.NoError(t, err)
	assert.NotEmpty(t, key2)

	// Different data should produce different keys
	assert.NotEqual(t, key1, key2)

	// Same data should produce same key
	key3, err := GenerateKey(layers, data1)
	require.NoError(t, err)
	assert.Equal(t, key1, key3)
}

func TestLayerSeparation(t *testing.T) {
	builder, err := NewPromptBuilder()
	require.NoError(t, err)

	data := map[string]interface{}{
		"AgentName": "Test",
	}

	prompt, err := builder.BuildPrompt([]Layer{LayerBase, LayerContext}, data)
	require.NoError(t, err)

	// Count separators
	separatorCount := strings.Count(prompt, "---")
	assert.Equal(t, 1, separatorCount, "Should have exactly one separator between two layers")
}
