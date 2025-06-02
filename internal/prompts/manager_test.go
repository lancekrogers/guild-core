package prompts

import (
	"strings"
	"testing"
)

func TestNewEnhancedPromptManager(t *testing.T) {
	manager, err := NewEnhancedPromptManager()
	if err != nil {
		t.Fatalf("Failed to create enhanced prompt manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	// Check that some prompts were loaded
	prompts := manager.ListPrompts()
	if len(prompts) == 0 {
		t.Error("Expected at least one prompt to be loaded")
	}

	// Check for specific prompts we know should exist
	expectedPrompts := []string{
		"objective.creation",
		"objective.ai_docs_gen",
		"objective.refinement",
		"objective.specs_gen",
		"objective.suggestion",
	}

	for _, expected := range expectedPrompts {
		if _, exists := prompts[expected]; !exists {
			t.Errorf("Expected prompt %s to be loaded", expected)
		}
	}
}

func TestEnhancedPromptManager_RenderPrompt(t *testing.T) {
	manager, err := NewEnhancedPromptManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name       string
		promptName string
		data       interface{}
		wantErr    bool
		checkFunc  func(t *testing.T, result string)
	}{
		{
			name:       "render with valid data",
			promptName: "objective.creation",
			data: map[string]interface{}{
				"Description": "Test description for objective",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "Test description for objective") {
					t.Error("Expected rendered content to contain the description")
				}
				if !strings.Contains(result, "System Prompt for Creating Guild Objectives") {
					t.Error("Expected rendered content to contain the prompt title")
				}
			},
		},
		{
			name:       "render with missing required variable",
			promptName: "objective.creation",
			data:       map[string]interface{}{},
			wantErr:    true,
		},
		{
			name:       "render with optional variables",
			promptName: "objective.creation",
			data: map[string]interface{}{
				"Description": "Test description",
				"UserContext": "Additional context here",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result string) {
				// Should process conditional blocks
				if !strings.Contains(result, "Additional Context Provided") {
					t.Error("Expected conditional block to be included")
				}
				if !strings.Contains(result, "Additional context here") {
					t.Error("Expected user context to be included")
				}
			},
		},
		{
			name:       "render non-existent prompt",
			promptName: "non.existent",
			data:       map[string]interface{}{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.RenderPrompt(tt.promptName, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestEnhancedPromptManager_GetMetadata(t *testing.T) {
	manager, err := NewEnhancedPromptManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name       string
		promptName string
		wantErr    bool
		checkFunc  func(t *testing.T, meta *PromptMetadata)
	}{
		{
			name:       "get metadata for existing prompt",
			promptName: "objective.creation",
			wantErr:    false,
			checkFunc: func(t *testing.T, meta *PromptMetadata) {
				if meta.ID != "objective-creation" {
					t.Errorf("Expected ID 'objective-creation', got %s", meta.ID)
				}
				if meta.Category != "objective" {
					t.Errorf("Expected category 'objective', got %s", meta.Category)
				}
				if meta.Complexity < 1 || meta.Complexity > 10 {
					t.Errorf("Expected complexity between 1-10, got %d", meta.Complexity)
				}
			},
		},
		{
			name:       "get metadata for non-existent prompt",
			promptName: "non.existent",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := manager.GetMetadata(tt.promptName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, meta)
			}
		})
	}
}

func TestEnhancedPromptManager_ValidatePrompt(t *testing.T) {
	manager, err := NewEnhancedPromptManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name       string
		promptName string
		data       map[string]interface{}
		wantErr    bool
	}{
		{
			name:       "valid data",
			promptName: "objective.creation",
			data: map[string]interface{}{
				"Description": "Test description",
			},
			wantErr: false,
		},
		{
			name:       "missing required variable",
			promptName: "objective.creation",
			data:       map[string]interface{}{},
			wantErr:    true,
		},
		{
			name:       "prompt without metadata (no validation)",
			promptName: "objective.test", // Assuming this doesn't exist
			data:       map[string]interface{}{},
			wantErr:    false, // No metadata means no validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidatePrompt(tt.promptName, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrompt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnhancedPromptManager_GetPromptsByCategory(t *testing.T) {
	manager, err := NewEnhancedPromptManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	objectivePrompts := manager.GetPromptsByCategory("objective")
	if len(objectivePrompts) == 0 {
		t.Error("Expected at least one objective prompt")
	}

	for name, meta := range objectivePrompts {
		if meta.Category != "objective" {
			t.Errorf("Prompt %s has category %s, expected 'objective'", name, meta.Category)
		}
	}

	docPrompts := manager.GetPromptsByCategory("documentation")
	// This might be empty if we haven't categorized ai_docs_gen as documentation
	_ = docPrompts
}

func TestEnhancedPromptManager_GetPromptsByTag(t *testing.T) {
	manager, err := NewEnhancedPromptManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Look for prompts with "planning" tag
	planningPrompts := manager.GetPromptsByTag("planning")
	if len(planningPrompts) == 0 {
		t.Error("Expected at least one prompt with 'planning' tag")
	}

	// Verify all returned prompts have the tag
	for name, meta := range planningPrompts {
		found := false
		for _, tag := range meta.Tags {
			if strings.EqualFold(tag, "planning") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Prompt %s doesn't have 'planning' tag", name)
		}
	}
}

func TestEnhancedPromptManager_IsModelCompatible(t *testing.T) {
	manager, err := NewEnhancedPromptManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name       string
		promptName string
		modelName  string
		want       bool
	}{
		{
			name:       "compatible model",
			promptName: "objective.creation",
			modelName:  "gpt-4",
			want:       true,
		},
		{
			name:       "another compatible model",
			promptName: "objective.creation",
			modelName:  "claude-3",
			want:       true,
		},
		{
			name:       "prompt without metadata (compatible with all)",
			promptName: "non.existent",
			modelName:  "any-model",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := manager.IsModelCompatible(tt.promptName, tt.modelName); got != tt.want {
				t.Errorf("IsModelCompatible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessConditionalBlocks(t *testing.T) {
	tests := []struct {
		name    string
		content string
		data    interface{}
		want    string
	}{
		{
			name: "include block when condition is met",
			content: `Before
<if_block condition="has_context">
This should be included
</if_block>
After`,
			data: map[string]interface{}{
				"Context": "Some context",
			},
			want: `Before
This should be included
After`,
		},
		{
			name: "exclude block when condition not met",
			content: `Before
<if_block condition="has_context">
This should be excluded
</if_block>
After`,
			data: map[string]interface{}{},
			want: `Before
After`,
		},
		{
			name: "multiple conditional blocks",
			content: `Start
<if_block condition="has_user_context">
User context present
</if_block>
Middle
<if_block condition="has_document_context">
Document context present
</if_block>
End`,
			data: map[string]interface{}{
				"UserContext": "User data",
			},
			want: `Start
User context present
Middle
End`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processConditionalBlocks(tt.content, tt.data)
			if got != tt.want {
				t.Errorf("processConditionalBlocks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user_context", "UserContext"},
		{"document_context", "DocumentContext"},
		{"simple", "Simple"},
		{"multiple_word_example", "MultipleWordExample"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := toCamelCase(tt.input); got != tt.want {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}