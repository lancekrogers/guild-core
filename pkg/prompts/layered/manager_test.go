package layered_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/internal/prompts/layered"
)

// mockManagerWithFormatter implements both Manager and Formatter interfaces
type mockManagerWithFormatter struct {
	mock.Mock
}

// Manager interface methods
func (m *mockManagerWithFormatter) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	args := m.Called(ctx, role, domain)
	return args.String(0), args.Error(1)
}

func (m *mockManagerWithFormatter) GetTemplate(ctx context.Context, templateName string) (string, error) {
	args := m.Called(ctx, templateName)
	return args.String(0), args.Error(1)
}

func (m *mockManagerWithFormatter) FormatContext(ctx context.Context, context layered.Context) (string, error) {
	args := m.Called(ctx, context)
	return args.String(0), args.Error(1)
}

func (m *mockManagerWithFormatter) ListRoles(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockManagerWithFormatter) ListDomains(ctx context.Context, role string) ([]string, error) {
	args := m.Called(ctx, role)
	return args.Get(0).([]string), args.Error(1)
}

// Formatter interface methods
func (m *mockManagerWithFormatter) FormatAsXML(ctx layered.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *mockManagerWithFormatter) FormatAsMarkdown(ctx layered.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *mockManagerWithFormatter) OptimizeForTokens(content string, maxTokens int) (string, error) {
	args := m.Called(content, maxTokens)
	return args.String(0), args.Error(1)
}

// Separate mock for just LayeredManager interface methods
type mockLayeredManager struct {
	mock.Mock
}

func (m *mockLayeredManager) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	args := m.Called(ctx, role, domain)
	return args.String(0), args.Error(1)
}

func (m *mockLayeredManager) GetTemplate(ctx context.Context, templateName string) (string, error) {
	args := m.Called(ctx, templateName)
	return args.String(0), args.Error(1)
}

func (m *mockLayeredManager) FormatContext(ctx context.Context, context layered.Context) (string, error) {
	args := m.Called(ctx, context)
	return args.String(0), args.Error(1)
}

func (m *mockLayeredManager) ListRoles(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockLayeredManager) ListDomains(ctx context.Context, role string) ([]string, error) {
	args := m.Called(ctx, role)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockLayeredManager) BuildLayeredPrompt(ctx context.Context, artisanID, sessionID string, turnCtx layered.TurnContext) (*layered.LayeredPrompt, error) {
	args := m.Called(ctx, artisanID, sessionID, turnCtx)
	return args.Get(0).(*layered.LayeredPrompt), args.Error(1)
}

func (m *mockLayeredManager) GetPromptLayer(ctx context.Context, layer layered.PromptLayer, artisanID, sessionID string) (*layered.SystemPrompt, error) {
	args := m.Called(ctx, layer, artisanID, sessionID)
	return args.Get(0).(*layered.SystemPrompt), args.Error(1)
}

func (m *mockLayeredManager) SetPromptLayer(ctx context.Context, prompt layered.SystemPrompt) error {
	args := m.Called(ctx, prompt)
	return args.Error(0)
}

func (m *mockLayeredManager) DeletePromptLayer(ctx context.Context, layer layered.PromptLayer, artisanID, sessionID string) error {
	args := m.Called(ctx, layer, artisanID, sessionID)
	return args.Error(0)
}

func (m *mockLayeredManager) ListPromptLayers(ctx context.Context, artisanID, sessionID string) ([]layered.SystemPrompt, error) {
	args := m.Called(ctx, artisanID, sessionID)
	return args.Get(0).([]layered.SystemPrompt), args.Error(1)
}

func (m *mockLayeredManager) InvalidateCache(ctx context.Context, artisanID, sessionID string) error {
	args := m.Called(ctx, artisanID, sessionID)
	return args.Error(0)
}

func TestGuildLayeredManager(t *testing.T) {
	t.Run("NewGuildLayeredManager_Creation", func(t *testing.T) {
		// Setup mocks
		baseManager := &mockManagerWithFormatter{}
		store := &mockStore{}
		
		// Create layered manager
		manager := layered.NewGuildLayeredManager(baseManager, store, nil, nil, 4000)
		
		// Verify creation
		assert.NotNil(t, manager)
	})
	
	t.Run("BuildLayeredPrompt_Integration", func(t *testing.T) {
		// Setup mocks
		baseManager := &mockManagerWithFormatter{}
		store := &mockStore{}
		
		// Create manager
		manager := layered.NewGuildLayeredManager(baseManager, store, nil, nil, 4000)
		
		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		turnCtx := layered.TurnContext{
			UserMessage:  "Build a REST API",
			TaskID:       "API-001",
			CommissionID: "COMM-001",
		}
		
		// Setup expectations for base manager
		baseManager.On("GetSystemPrompt", mock.Anything, "backend", "default").Return(
			"You are a backend artisan...", nil)
		// Domain prompt is also called since artisan ID has domain "dev"
		baseManager.On("GetSystemPrompt", mock.Anything, "backend", "dev").Return(
			"Backend dev domain prompt", nil).Maybe()
		
		// Setup expectations for store layers
		// Platform layer is always retrieved
		store.On("GetPromptLayer", mock.Anything, "platform", "default").Return(
			[]byte{}, assert.AnError).Maybe() // Not found - will use default
		
		// Guild layer is optional
		store.On("GetPromptLayer", mock.Anything, "guild", "").Return(
			[]byte{}, assert.AnError).Maybe() // Not found - optional
		
		// Session layer is optional
		store.On("GetPromptLayer", mock.Anything, "session", sessionID).Return(
			[]byte{}, assert.AnError).Maybe() // Not found - optional
		
		// Setup expectation for cache store
		store.On("CacheCompiledPrompt", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil)
		
		// Execute
		ctx := context.Background()
		result, err := manager.BuildLayeredPrompt(ctx, artisanID, sessionID, turnCtx)
		
		// Verify
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, artisanID, result.ArtisanID)
		assert.Equal(t, sessionID, result.SessionID)
		assert.Contains(t, result.Compiled, "Build a REST API")
		
		baseManager.AssertExpectations(t)
	})
	
	t.Run("SetPromptLayer_SessionPreferences", func(t *testing.T) {
		// Setup mocks
		baseManager := &mockManagerWithFormatter{}
		store := &mockStore{}
		
		manager := layered.NewGuildLayeredManager(baseManager, store, nil, nil, 4000)
		
		// Test session prompt
		sessionPrompt := layered.SystemPrompt{
			Layer:     layered.LayerSession,
			SessionID: "session_123",
			Content:   "User prefers detailed explanations with examples",
			Version:   1,
			Updated:   time.Now(),
		}
		
		// Setup expectations
		store.On("SavePromptLayer", mock.Anything, "session", "session_123", mock.AnythingOfType("[]uint8")).Return(nil)
		
		// Execute
		ctx := context.Background()
		err := manager.SetPromptLayer(ctx, sessionPrompt)
		
		// Verify
		require.NoError(t, err)
		store.AssertExpectations(t)
	})
	
	t.Run("ListPromptLayers_AllLayers", func(t *testing.T) {
		// Setup mocks
		baseManager := &mockManagerWithFormatter{}
		store := &mockStore{}
		
		manager := layered.NewGuildLayeredManager(baseManager, store, nil, nil, 4000)
		
		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		
		// Setup expectations for different layers
		// Platform layer (default)
		store.On("GetPromptLayer", mock.Anything, "platform", "default").Return(
			[]byte{}, assert.AnError) // Not found - will use default
		
		// Guild layer (default)
		store.On("GetPromptLayer", mock.Anything, "guild", "default").Return(
			[]byte{}, assert.AnError) // Not found - optional
		
		// Role layer (backend)
		store.On("GetPromptLayer", mock.Anything, "role", "backend").Return(
			[]byte{}, assert.AnError) // Not found - optional
		
		// Domain layer (backend:dev)
		store.On("GetPromptLayer", mock.Anything, "domain", "backend:dev").Return(
			[]byte{}, assert.AnError) // Not found - optional
		
		// Session layer
		store.On("GetPromptLayer", mock.Anything, "session", "session_123").Return(
			[]byte(`{"layer":"session","content":"Session preferences","version":1}`), nil)
		
		// Turn layer (backend-dev-001:session_123)
		store.On("GetPromptLayer", mock.Anything, "turn", "backend-dev-001:session_123").Return(
			[]byte{}, assert.AnError) // Not found - optional
		
		// Execute
		ctx := context.Background()
		layers, err := manager.ListPromptLayers(ctx, artisanID, sessionID)
		
		// Verify
		require.NoError(t, err)
		assert.NotNil(t, layers)
		assert.True(t, len(layers) >= 1, "Should have at least one layer")
		
		// Check that we have the expected layers
		foundSession := false
		for _, layer := range layers {
			if layer.Layer == layered.LayerSession {
				foundSession = true
				assert.Equal(t, "Session preferences", layer.Content)
			}
		}
		assert.True(t, foundSession, "Should have session layer")
		
		baseManager.AssertExpectations(t)
		store.AssertExpectations(t)
	})
	
	t.Run("InvalidateCache_ClearsCaches", func(t *testing.T) {
		// Setup mocks
		baseManager := &mockManagerWithFormatter{}
		store := &mockStore{}
		
		manager := layered.NewGuildLayeredManager(baseManager, store, nil, nil, 4000)
		
		// Test data
		artisanID := "backend-dev-001"
		sessionID := "session_123"
		
		// Setup expectations
		store.On("InvalidatePromptCache", mock.Anything, "artisan:backend-dev-001:session:session_123").Return(nil)
		
		// Execute
		ctx := context.Background()
		err := manager.InvalidateCache(ctx, artisanID, sessionID)
		
		// Verify
		require.NoError(t, err)
		store.AssertExpectations(t)
	})
	
	t.Run("LegacyMethods_Delegation", func(t *testing.T) {
		// Setup mocks
		baseManager := &mockManagerWithFormatter{}
		store := &mockStore{}
		
		manager := layered.NewGuildLayeredManager(baseManager, store, nil, nil, 4000)
		
		// Test legacy method delegation
		baseManager.On("GetSystemPrompt", mock.Anything, "backend", "web-app").Return(
			"Legacy prompt", nil)
		baseManager.On("ListRoles", mock.Anything).Return(
			[]string{"backend", "frontend", "devops"}, nil)
		
		ctx := context.Background()
		
		// Test GetSystemPrompt delegation
		prompt, err := manager.GetSystemPrompt(ctx, "backend", "web-app")
		require.NoError(t, err)
		assert.Equal(t, "Legacy prompt", prompt)
		
		// Test ListRoles delegation
		roles, err := manager.ListRoles(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"backend", "frontend", "devops"}, roles)
		
		baseManager.AssertExpectations(t)
	})
}

func TestPromptValidation(t *testing.T) {
	t.Run("ValidatePrompt_RequiredFields", func(t *testing.T) {
		// Setup
		baseManager := &mockManagerWithFormatter{}
		store := &mockStore{}
		
		manager := layered.NewGuildLayeredManager(baseManager, store, nil, nil, 4000)
		
		tests := []struct {
			name    string
			prompt  layered.SystemPrompt
			wantErr bool
			errMsg  string
		}{
			{
				name: "valid_platform_prompt",
				prompt: layered.SystemPrompt{
					Layer:   layered.LayerPlatform,
					Content: "Platform guidelines",
					Version: 1,
				},
				wantErr: false,
			},
			{
				name: "missing_layer",
				prompt: layered.SystemPrompt{
					Content: "Some content",
					Version: 1,
				},
				wantErr: true,
				errMsg:  "prompt layer is required",
			},
			{
				name: "missing_content",
				prompt: layered.SystemPrompt{
					Layer:   layered.LayerRole,
					Version: 1,
				},
				wantErr: true,
				errMsg:  "prompt content is required",
			},
			{
				name: "session_without_session_id",
				prompt: layered.SystemPrompt{
					Layer:   layered.LayerSession,
					Content: "Session content",
					Version: 1,
				},
				wantErr: true,
				errMsg:  "session ID is required",
			},
			{
				name: "valid_session_prompt",
				prompt: layered.SystemPrompt{
					Layer:     layered.LayerSession,
					SessionID: "session_123",
					Content:   "Session preferences",
					Version:   1,
				},
				wantErr: false,
			},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Mock store expectations for valid cases
				if !tt.wantErr {
					store.On("SavePromptLayer", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
					store.On("InvalidatePromptCache", mock.Anything, mock.Anything).Return(nil).Maybe()
				}
				
				ctx := context.Background()
				err := manager.SetPromptLayer(ctx, tt.prompt)
				
				if tt.wantErr {
					require.Error(t, err)
					assert.Contains(t, err.Error(), tt.errMsg)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}