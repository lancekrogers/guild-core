// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/pkg/prompts/layered"
)

// SimpleMockArtisanClient for testing
type SimpleMockArtisanClient struct {
	response string
	err      error
}

func (m *SimpleMockArtisanClient) Complete(ctx context.Context, request ArtisanRequest) (*ArtisanResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &ArtisanResponse{
		Content: m.response,
		Metadata: map[string]interface{}{
			"model": "test-model",
		},
	}, nil
}

// mockLayeredManager is a minimal mock for layered.LayeredManager
type mockLayeredManager struct{}

func (m *mockLayeredManager) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	return "", nil
}

func (m *mockLayeredManager) GetTemplate(ctx context.Context, templateName string) (string, error) {
	return "", nil
}

func (m *mockLayeredManager) FormatContext(ctx context.Context, context layered.Context) (string, error) {
	return "", nil
}

func (m *mockLayeredManager) ListRoles(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockLayeredManager) ListDomains(ctx context.Context, role string) ([]string, error) {
	return nil, nil
}

func (m *mockLayeredManager) BuildLayeredPrompt(ctx context.Context, artisanID, sessionID string, turnCtx layered.TurnContext) (*layered.LayeredPrompt, error) {
	return nil, nil
}

func (m *mockLayeredManager) GetPromptLayer(ctx context.Context, layer layered.PromptLayer, artisanID, sessionID string) (*layered.SystemPrompt, error) {
	return nil, nil
}

func (m *mockLayeredManager) SetPromptLayer(ctx context.Context, prompt layered.SystemPrompt) error {
	return nil
}

func (m *mockLayeredManager) DeletePromptLayer(ctx context.Context, layer layered.PromptLayer, artisanID, sessionID string) error {
	return nil
}

func (m *mockLayeredManager) ListPromptLayers(ctx context.Context, artisanID, sessionID string) ([]layered.SystemPrompt, error) {
	return nil, nil
}

func (m *mockLayeredManager) InvalidateCache(ctx context.Context, artisanID, sessionID string) error {
	return nil
}

func TestTaskExtractor_ExtractTasks(t *testing.T) {
	// Create a mock response that simulates what an LLM would return
	mockExtractionResult := ExtractionResult{
		ExtractionMetadata: ExtractionMetadata{
			CommissionID: "test-commission-001",
			ExtractedAt:  "2025-01-06T12:00:00Z",
			TotalTasks:   3,
			ContentAnalysis: ContentAnalysis{
				Structure:    "Hierarchical markdown with clear sections",
				Completeness: "All major components covered",
				Clarity:      "Clear and actionable tasks identified",
			},
		},
		Tasks: []ExtractedTask{
			{
				ID:                   "BACKEND-001",
				Title:                "Set up Express.js API server",
				Description:          "Initialize a new Express.js server with middleware for JSON parsing, CORS, and error handling",
				Category:             "BACKEND",
				Priority:             "high",
				EstimatedHours:       floatPtr(4),
				Dependencies:         []string{},
				RequiredCapabilities: []string{"backend", "nodejs", "api"},
				Metadata: ExtractedTaskMetadata{
					SourceSection: "Backend Implementation",
					Rationale:     "Foundation for all API endpoints",
					AcceptanceCriteria: []string{
						"Server starts on configured port",
						"Middleware properly configured",
						"Basic health check endpoint works",
					},
				},
			},
			{
				ID:                   "BACKEND-002",
				Title:                "Implement JWT authentication",
				Description:          "Add JWT-based authentication with token generation and validation middleware",
				Category:             "AUTH",
				Priority:             "high",
				EstimatedHours:       floatPtr(6),
				Dependencies:         []string{"BACKEND-001"},
				RequiredCapabilities: []string{"backend", "security", "authentication"},
				Metadata: ExtractedTaskMetadata{
					SourceSection:  "Authentication Requirements",
					Rationale:      "Secure API access control",
					TechnicalNotes: "Use RS256 algorithm for better security",
				},
			},
			{
				ID:                   "FRONTEND-001",
				Title:                "Create React application scaffold",
				Description:          "Initialize React app with TypeScript, routing, and state management setup",
				Category:             "FRONTEND",
				Priority:             "medium",
				EstimatedHours:       floatPtr(3),
				Dependencies:         []string{},
				RequiredCapabilities: []string{"frontend", "react", "typescript"},
				Metadata: ExtractedTaskMetadata{
					SourceSection: "Frontend Architecture",
					Rationale:     "User interface foundation",
				},
			},
		},
		TaskRelationships: TaskRelationships{
			Phases: []TaskPhase{
				{
					Name:        "Foundation",
					Description: "Core infrastructure setup",
					TaskIDs:     []string{"BACKEND-001", "FRONTEND-001"},
				},
				{
					Name:        "Security",
					Description: "Authentication and authorization",
					TaskIDs:     []string{"BACKEND-002"},
				},
			},
			CriticalPath: []string{"BACKEND-001", "BACKEND-002"},
		},
	}

	// Convert to JSON for mock response
	jsonResponse, err := json.MarshalIndent(mockExtractionResult, "", "  ")
	require.NoError(t, err)

	// Create mock client
	mockClient := &SimpleMockArtisanClient{
		response: "```json\n" + string(jsonResponse) + "\n```",
	}

	// Create task extractor (without prompt manager for simplicity)
	extractor := NewTaskExtractor(mockClient, nil)

	// Create a refined commission to extract from
	refinedCommission := &RefinedCommission{
		CommissionID: "test-commission-001",
		Structure: &FileStructure{
			RootDir: ".",
			Files: []*FileEntry{
				{
					Path: "README.md",
					Content: `# E-commerce Platform

## Backend Implementation
- Set up Express.js server with proper middleware
- Implement JWT authentication for secure access

## Frontend Architecture
- Create React application with TypeScript
- Set up routing and state management`,
					Type: FileTypeMarkdown,
				},
			},
		},
		Metadata: map[string]interface{}{
			"domain":         "web-app",
			"original_title": "E-commerce Platform Development",
		},
	}

	// Test extraction
	result, err := extractor.ExtractTasks(context.Background(), refinedCommission)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify extraction results
	assert.Equal(t, "test-commission-001", result.ExtractionMetadata.CommissionID)
	assert.Equal(t, 3, result.ExtractionMetadata.TotalTasks)
	assert.Len(t, result.Tasks, 3)

	// Verify first task details
	firstTask := result.Tasks[0]
	assert.Equal(t, "BACKEND-001", firstTask.ID)
	assert.Equal(t, "Set up Express.js API server", firstTask.Title)
	assert.Equal(t, "high", firstTask.Priority)
	assert.Equal(t, float64(4), *firstTask.EstimatedHours)
	assert.Contains(t, firstTask.RequiredCapabilities, "backend")

	// Verify task relationships
	assert.Len(t, result.TaskRelationships.Phases, 2)
	assert.Equal(t, "Foundation", result.TaskRelationships.Phases[0].Name)
	assert.Contains(t, result.TaskRelationships.CriticalPath, "BACKEND-001")
}

func TestTaskExtractor_ConvertToTaskInfo(t *testing.T) {
	// Test ExtractedTask to TaskInfo conversion
	extractedTask := ExtractedTask{
		ID:                   "API-003",
		Title:                "Create user endpoints",
		Description:          "Implement CRUD endpoints for user management",
		Category:             "API",
		Priority:             "medium",
		EstimatedHours:       floatPtr(8),
		Dependencies:         []string{"BACKEND-001", "AUTH-001"},
		RequiredCapabilities: []string{"backend", "api", "database"},
		Metadata: ExtractedTaskMetadata{
			SourceSection: "API Specification",
			Rationale:     "Core user management functionality",
		},
	}

	// Convert to TaskInfo
	taskInfo := extractedTask.ConvertToTaskInfo()

	// Verify conversion
	assert.Equal(t, "API-003", taskInfo.ID)
	assert.Equal(t, "API", taskInfo.Category)
	assert.Equal(t, "003", taskInfo.Number)
	assert.Equal(t, "Create user endpoints", taskInfo.Title)
	assert.Equal(t, "medium", taskInfo.Priority)
	assert.Equal(t, "1d", taskInfo.Estimate) // 8 hours = 1 day
	assert.Equal(t, []string{"BACKEND-001", "AUTH-001"}, taskInfo.Dependencies)
	assert.Equal(t, "API Specification", taskInfo.Section)
}

func TestIntelligentParser_Modes(t *testing.T) {
	tests := []struct {
		name         string
		mode         ParserMode
		hasClient    bool
		hasManager   bool
		expectedMode ParserMode
	}{
		{
			name:         "pattern_mode",
			mode:         ParserModePattern,
			hasClient:    false,
			hasManager:   false,
			expectedMode: ParserModePattern,
		},
		{
			name:         "extractor_mode_with_dependencies",
			mode:         ParserModeExtractor,
			hasClient:    true,
			hasManager:   true,
			expectedMode: ParserModeExtractor,
		},
		{
			name:         "extractor_mode_without_dependencies_falls_back",
			mode:         ParserModeExtractor,
			hasClient:    false,
			hasManager:   false,
			expectedMode: ParserModePattern,
		},
		{
			name:         "auto_mode",
			mode:         ParserModeAuto,
			hasClient:    true,
			hasManager:   true,
			expectedMode: ParserModeExtractor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := IntelligentParserConfig{
				Mode: tt.mode,
			}

			if tt.hasClient {
				config.ArtisanClient = &SimpleMockArtisanClient{}
			}
			if tt.hasManager {
				// Create a minimal mock that satisfies the interface check
				config.PromptManager = &mockLayeredManager{}
			}

			parser := NewIntelligentParser(config)
			assert.Equal(t, tt.expectedMode, parser.GetExtractionMode())
		})
	}
}

// Helper function
func floatPtr(f float64) *float64 {
	return &f
}
