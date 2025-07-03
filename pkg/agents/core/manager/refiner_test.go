// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/agents/core/manager"
	"github.com/lancekrogers/guild/pkg/prompts/layered"
)

// Mock implementations for testing
type mockArtisanClient struct {
	mock.Mock
}

func (m *mockArtisanClient) Complete(ctx context.Context, request manager.ArtisanRequest) (*manager.ArtisanResponse, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*manager.ArtisanResponse), args.Error(1)
}

type mockPromptManager struct {
	mock.Mock
}

func (m *mockPromptManager) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	args := m.Called(ctx, role, domain)
	return args.String(0), args.Error(1)
}

func (m *mockPromptManager) GetTemplate(ctx context.Context, templateName string) (string, error) {
	args := m.Called(ctx, templateName)
	return args.String(0), args.Error(1)
}

func (m *mockPromptManager) FormatContext(ctx context.Context, context layered.Context) (string, error) {
	args := m.Called(ctx, context)
	return args.String(0), args.Error(1)
}

func (m *mockPromptManager) ListRoles(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockPromptManager) ListDomains(ctx context.Context, role string) ([]string, error) {
	args := m.Called(ctx, role)
	return args.Get(0).([]string), args.Error(1)
}

type mockResponseParser struct {
	mock.Mock
}

func (m *mockResponseParser) ParseResponse(response *manager.ArtisanResponse) (*manager.FileStructure, error) {
	args := m.Called(response)
	return args.Get(0).(*manager.FileStructure), args.Error(1)
}

func (m *mockResponseParser) ParseResponseWithContext(ctx context.Context, response *manager.ArtisanResponse) (*manager.FileStructure, error) {
	args := m.Called(ctx, response)
	return args.Get(0).(*manager.FileStructure), args.Error(1)
}

type mockStructureValidator struct {
	mock.Mock
}

func (m *mockStructureValidator) ValidateStructure(structure *manager.FileStructure) error {
	args := m.Called(structure)
	return args.Error(0)
}

func TestGuildMasterRefiner(t *testing.T) {
	t.Run("RefineCommission_Success", func(t *testing.T) {
		// Setup mocks
		artisanClient := &mockArtisanClient{}
		promptManager := &mockPromptManager{}
		parser := &mockResponseParser{}
		validator := &mockStructureValidator{}

		refiner := manager.NewGuildMasterRefiner(artisanClient, promptManager, parser, validator)

		// Setup test data
		commission := manager.Commission{
			ID:          "COMM-001",
			Title:       "Build Test System",
			Description: "Create a comprehensive testing framework for the Guild",
			Domain:      "web-app",
			Context: map[string]interface{}{
				"technology": "Go",
				"deadline":   "2 weeks",
			},
		}

		expectedPrompt := "You are a Guild Master, responsible for taking high-level commissions..."
		expectedResponse := &manager.ArtisanResponse{
			Content: `## File: README.md
# Test System

**Tasks Generated**:
- TEST-001: Create test framework
  - Priority: high
  - Estimate: 4h
  - Dependencies: none
  - Capabilities: backend, testing
  - Description: Build the core testing framework`,
			Metadata: map[string]interface{}{
				"model": "claude-3",
			},
		}

		expectedStructure := &manager.FileStructure{
			RootDir: ".",
			Files: []*manager.FileEntry{
				{
					Path:       "README.md",
					Content:    "# Test System\n\n**Tasks Generated**:\n- TEST-001: Create test framework",
					Type:       manager.FileTypeMarkdown,
					TasksCount: 1,
					Metadata: map[string]interface{}{
						"source": "guild_master_response",
					},
				},
			},
		}

		// Setup expectations
		promptManager.On("GetSystemPrompt", mock.Anything, "manager", "web-app").Return(expectedPrompt, nil)
		artisanClient.On("Complete", mock.Anything, mock.MatchedBy(func(req manager.ArtisanRequest) bool {
			return req.SystemPrompt == expectedPrompt &&
				req.Temperature == 0.7 &&
				req.MaxTokens == 4000
		})).Return(expectedResponse, nil)
		parser.On("ParseResponse", expectedResponse).Return(expectedStructure, nil)
		validator.On("ValidateStructure", expectedStructure).Return(nil)

		// Execute
		ctx := context.Background()
		result, err := refiner.RefineCommission(ctx, commission)

		// Verify
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "COMM-001", result.CommissionID)
		assert.Equal(t, expectedStructure, result.Structure)
		assert.Equal(t, "web-app", result.Metadata["domain"])
		assert.Equal(t, "Build Test System", result.Metadata["original_title"])
		assert.Equal(t, "auto-refiner", result.Metadata["guild_master"])

		// Verify all mocks were called
		promptManager.AssertExpectations(t)
		artisanClient.AssertExpectations(t)
		parser.AssertExpectations(t)
		validator.AssertExpectations(t)
	})

	t.Run("RefineCommission_DefaultDomain", func(t *testing.T) {
		// Setup mocks
		artisanClient := &mockArtisanClient{}
		promptManager := &mockPromptManager{}
		parser := &mockResponseParser{}
		validator := &mockStructureValidator{}

		refiner := manager.NewGuildMasterRefiner(artisanClient, promptManager, parser, validator)

		// Commission without domain
		commission := manager.Commission{
			ID:          "COMM-002",
			Title:       "Simple Task",
			Description: "A simple commission",
		}

		expectedPrompt := "Default Guild Master prompt..."
		expectedResponse := &manager.ArtisanResponse{Content: "# Simple Task"}
		expectedStructure := &manager.FileStructure{RootDir: ".", Files: []*manager.FileEntry{}}

		// Should use "default" domain when empty
		promptManager.On("GetSystemPrompt", mock.Anything, "manager", "default").Return(expectedPrompt, nil)
		artisanClient.On("Complete", mock.Anything, mock.Anything).Return(expectedResponse, nil)
		parser.On("ParseResponse", expectedResponse).Return(expectedStructure, nil)
		validator.On("ValidateStructure", expectedStructure).Return(nil)

		// Execute
		ctx := context.Background()
		_, err := refiner.RefineCommission(ctx, commission)

		// Verify
		require.NoError(t, err)
		promptManager.AssertExpectations(t)
	})

	t.Run("RefineCommission_PromptError", func(t *testing.T) {
		// Setup mocks
		artisanClient := &mockArtisanClient{}
		promptManager := &mockPromptManager{}
		parser := &mockResponseParser{}
		validator := &mockStructureValidator{}

		refiner := manager.NewGuildMasterRefiner(artisanClient, promptManager, parser, validator)

		commission := manager.Commission{
			ID:     "COMM-003",
			Domain: "invalid-domain",
		}

		// Prompt manager returns error
		promptManager.On("GetSystemPrompt", mock.Anything, "manager", "invalid-domain").Return("", assert.AnError)

		// Execute
		ctx := context.Background()
		result, err := refiner.RefineCommission(ctx, commission)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get system prompt")
		promptManager.AssertExpectations(t)
	})

	t.Run("RefineCommission_ArtisanError", func(t *testing.T) {
		// Setup mocks
		artisanClient := &mockArtisanClient{}
		promptManager := &mockPromptManager{}
		parser := &mockResponseParser{}
		validator := &mockStructureValidator{}

		refiner := manager.NewGuildMasterRefiner(artisanClient, promptManager, parser, validator)

		commission := manager.Commission{ID: "COMM-004"}
		expectedPrompt := "Test prompt"

		promptManager.On("GetSystemPrompt", mock.Anything, "manager", "default").Return(expectedPrompt, nil)
		artisanClient.On("Complete", mock.Anything, mock.Anything).Return((*manager.ArtisanResponse)(nil), assert.AnError)

		// Execute
		ctx := context.Background()
		result, err := refiner.RefineCommission(ctx, commission)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get Artisan response")
	})

	t.Run("RefineCommission_ParseError", func(t *testing.T) {
		// Setup mocks
		artisanClient := &mockArtisanClient{}
		promptManager := &mockPromptManager{}
		parser := &mockResponseParser{}
		validator := &mockStructureValidator{}

		refiner := manager.NewGuildMasterRefiner(artisanClient, promptManager, parser, validator)

		commission := manager.Commission{ID: "COMM-005"}
		expectedPrompt := "Test prompt"
		expectedResponse := &manager.ArtisanResponse{Content: "Invalid content"}

		promptManager.On("GetSystemPrompt", mock.Anything, "manager", "default").Return(expectedPrompt, nil)
		artisanClient.On("Complete", mock.Anything, mock.Anything).Return(expectedResponse, nil)
		parser.On("ParseResponse", expectedResponse).Return((*manager.FileStructure)(nil), assert.AnError)

		// Execute
		ctx := context.Background()
		result, err := refiner.RefineCommission(ctx, commission)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to parse Artisan response")
	})

	t.Run("RefineCommission_ValidationError", func(t *testing.T) {
		// Setup mocks
		artisanClient := &mockArtisanClient{}
		promptManager := &mockPromptManager{}
		parser := &mockResponseParser{}
		validator := &mockStructureValidator{}

		refiner := manager.NewGuildMasterRefiner(artisanClient, promptManager, parser, validator)

		commission := manager.Commission{ID: "COMM-006"}
		expectedPrompt := "Test prompt"
		expectedResponse := &manager.ArtisanResponse{Content: "Content"}
		expectedStructure := &manager.FileStructure{RootDir: ".", Files: []*manager.FileEntry{}}

		promptManager.On("GetSystemPrompt", mock.Anything, "manager", "default").Return(expectedPrompt, nil)
		artisanClient.On("Complete", mock.Anything, mock.Anything).Return(expectedResponse, nil)
		parser.On("ParseResponse", expectedResponse).Return(expectedStructure, nil)
		validator.On("ValidateStructure", expectedStructure).Return(assert.AnError)

		// Execute
		ctx := context.Background()
		result, err := refiner.RefineCommission(ctx, commission)

		// Verify
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "structure does not meet Guild standards")
	})

	t.Run("BuildUserPrompt_WithContext", func(t *testing.T) {
		artisanClient := &mockArtisanClient{}
		promptManager := &mockPromptManager{}
		parser := &mockResponseParser{}
		validator := &mockStructureValidator{}

		refiner := manager.NewGuildMasterRefiner(artisanClient, promptManager, parser, validator)

		commission := manager.Commission{
			ID:          "COMM-007",
			Title:       "Test Commission",
			Description: "A test commission for the Guild",
			Context: map[string]interface{}{
				"technology":  "Go",
				"deadline":    "2 weeks",
				"complex_obj": map[string]string{"key": "value"},
			},
		}

		// We can't directly test buildUserPrompt as it's private, but we can verify
		// the artisan client is called with the right user prompt by checking the call
		expectedPrompt := "Test prompt"
		expectedResponse := &manager.ArtisanResponse{Content: "Content"}
		expectedStructure := &manager.FileStructure{RootDir: ".", Files: []*manager.FileEntry{}}

		promptManager.On("GetSystemPrompt", mock.Anything, "manager", "default").Return(expectedPrompt, nil)
		artisanClient.On("Complete", mock.Anything, mock.MatchedBy(func(req manager.ArtisanRequest) bool {
			// Verify the user prompt contains commission details
			userPrompt := req.UserPrompt
			return strings.Contains(userPrompt, "Guild Master") &&
				strings.Contains(userPrompt, "COMM-007") &&
				strings.Contains(userPrompt, "Test Commission") &&
				strings.Contains(userPrompt, "A test commission for the Guild") &&
				strings.Contains(userPrompt, "technology: Go") &&
				strings.Contains(userPrompt, "deadline: 2 weeks") &&
				strings.Contains(userPrompt, "artisans") &&
				strings.Contains(userPrompt, "Workshop Board")
		})).Return(expectedResponse, nil)
		parser.On("ParseResponse", expectedResponse).Return(expectedStructure, nil)
		validator.On("ValidateStructure", expectedStructure).Return(nil)

		// Execute
		ctx := context.Background()
		_, err := refiner.RefineCommission(ctx, commission)

		// Verify
		require.NoError(t, err)
		artisanClient.AssertExpectations(t)
	})
}
