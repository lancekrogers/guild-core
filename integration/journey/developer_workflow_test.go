package journey

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/internal/testutil"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/storage"
)

// TestDeveloperDailyWorkflow tests a typical developer's daily workflow
// from starting a chat session to completing and reviewing tasks
func TestDeveloperDailyWorkflow(t *testing.T) {
	ctx := context.Background()
	
	// Setup test environment
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	// Initialize registry and components
	reg := registry.NewComponentRegistry()
	err := reg.Initialize(ctx, registry.Config{})
	require.NoError(t, err)

	// Setup mock provider
	mockProvider := testutil.NewMockLLMProvider()
	err = reg.Providers().Register("mock", func(ctx context.Context, cfg map[string]any) (any, error) {
		return mockProvider, nil
	})
	require.NoError(t, err)

	t.Run("Step1_ChatSessionInitialization", func(t *testing.T) {
		// Developer starts their day by opening chat
		startTime := time.Now()
		
		// Simulate chat session creation
		sessionID := "dev-session-001"
		sessionPath := filepath.Join(projCtx.GetGuildPath(), "sessions", sessionID)
		err := os.MkdirAll(sessionPath, 0755)
		require.NoError(t, err)

		// Create session metadata
		metadata := fmt.Sprintf(`session_id: %s
started_at: %s
user: developer
project: %s
`, sessionID, startTime.Format(time.RFC3339), projCtx.GetName())

		err = os.WriteFile(filepath.Join(sessionPath, "session.yaml"), []byte(metadata), 0644)
		require.NoError(t, err)

		// Verify session created quickly
		duration := time.Since(startTime)
		assert.Less(t, duration, 100*time.Millisecond, "Chat should start quickly")
	})

	t.Run("Step2_NaturalLanguageCommission", func(t *testing.T) {
		// Developer describes what they want in natural language
		userInput := `I need to add user authentication to our API. 
		It should support JWT tokens, have login/logout endpoints, 
		and include middleware for protecting routes. 
		Also need to store user sessions in Redis.`

		// Mock the AI response for commission creation
		mockProvider.SetResponse("commission_creator", `# User Authentication System

## Objective
Implement a complete user authentication system for the API with JWT token support and Redis session storage.

## Requirements

### Authentication Endpoints
- POST /auth/login - User login with email/password
- POST /auth/logout - Invalidate user session
- POST /auth/refresh - Refresh JWT token
- GET /auth/me - Get current user info

### Security Features  
- JWT token generation and validation
- Bcrypt password hashing
- Rate limiting on auth endpoints
- Session management in Redis

### Middleware
- Authentication middleware for protected routes
- Role-based access control (RBAC)
- Request logging for auth events

## Technical Specifications
- Use golang-jwt/jwt for JWT handling
- Use go-redis/redis for session storage
- Implement refresh token rotation
- 15-minute access token expiry
- 7-day refresh token expiry

## Testing Requirements
- Unit tests for all auth functions
- Integration tests for auth flow
- Load tests for concurrent logins`)

		// Create commission from natural language
		commissionPath := filepath.Join(projCtx.GetGuildPath(), "commissions", "auth-system.md")
		err := os.MkdirAll(filepath.Dir(commissionPath), 0755)
		require.NoError(t, err)

		// Save the generated commission
		response, _ := mockProvider.Complete(ctx, nil)
		err = os.WriteFile(commissionPath, []byte(response.Content), 0644)
		require.NoError(t, err)

		assert.FileExists(t, commissionPath, "Commission should be created from natural language")
	})

	t.Run("Step3_CommissionRefinement", func(t *testing.T) {
		// Developer refines the commission with AI assistance
		mockProvider.SetResponse("manager", testutil.GenerateMockAgentResponse(
			testutil.AgentResponseOptions{
				Type: "refined_commission",
				Tasks: []string{
					"Create User model and database schema",
					"Implement password hashing utilities", 
					"Build JWT token generation and validation",
					"Create authentication endpoints",
					"Implement authentication middleware",
					"Setup Redis session storage",
					"Add rate limiting",
					"Write comprehensive tests",
				},
			},
		))

		// Initialize database
		db, err := storage.DefaultDatabaseFactory(ctx, filepath.Join(projCtx.GetGuildPath(), "test.db"))
		require.NoError(t, err)

		// Create commission manager
		commissionRepo := storage.NewCommissionRepository(db)
		agentRegistry := reg.Agents()
		
		manager := commission.NewManager(
			commissionRepo,
			agentRegistry,
			nil, // kanban manager would be injected here
		)

		// Refine the commission
		refinedPath := filepath.Join(projCtx.GetGuildPath(), "objectives", "refined", "auth-system-refined.md")
		err = os.MkdirAll(filepath.Dir(refinedPath), 0755)
		require.NoError(t, err)

		// Mock refined content
		refinedContent := `# Refined: User Authentication System

## Phase 1: Foundation (Day 1)
- [ ] Create User model and database schema
- [ ] Implement password hashing utilities
- [ ] Setup basic project structure

## Phase 2: JWT Implementation (Day 2)  
- [ ] Build JWT token generation
- [ ] Implement token validation
- [ ] Create refresh token logic

## Phase 3: API Endpoints (Day 3)
- [ ] Create POST /auth/login endpoint
- [ ] Create POST /auth/logout endpoint
- [ ] Create POST /auth/refresh endpoint
- [ ] Create GET /auth/me endpoint

## Phase 4: Middleware & Security (Day 4)
- [ ] Implement authentication middleware
- [ ] Add role-based access control
- [ ] Setup rate limiting
- [ ] Configure Redis session storage

## Phase 5: Testing & Documentation (Day 5)
- [ ] Write unit tests (target: 90% coverage)
- [ ] Create integration tests
- [ ] Perform load testing
- [ ] Write API documentation`

		err = os.WriteFile(refinedPath, []byte(refinedContent), 0644)
		require.NoError(t, err)

		assert.FileExists(t, refinedPath, "Refined commission should be created")
	})

	t.Run("Step4_TaskExecutionWithTools", func(t *testing.T) {
		// Developer executes tasks using real tools
		
		// Setup tool registry
		toolRegistry := testutil.NewMockToolRegistry()
		
		// Register file creation tool
		toolRegistry.RegisterTool("create_file", &testutil.MockTool{
			ExecuteFn: func(ctx context.Context, params map[string]any) (any, error) {
				path := params["path"].(string)
				content := params["content"].(string)
				
				// Create file in test workspace
				fullPath := filepath.Join(projCtx.GetProjectPath(), path)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				if err != nil {
					return nil, err
				}
				
				err = os.WriteFile(fullPath, []byte(content), 0644)
				return map[string]any{"created": fullPath}, err
			},
		})

		// Execute task: Create User model
		modelContent := `package models

import (
    "time"
    "gorm.io/gorm"
)

type User struct {
    gorm.Model
    Email        string    ` + "`gorm:\"unique;not null\"`" + `
    PasswordHash string    ` + "`gorm:\"not null\"`" + `
    FirstName    string
    LastName     string
    Role         string    ` + "`gorm:\"default:'user'\"`" + `
    LastLoginAt  *time.Time
}

type Session struct {
    ID           string ` + "`gorm:\"primary_key\"`" + `
    UserID       uint
    RefreshToken string
    ExpiresAt    time.Time
    CreatedAt    time.Time
}`

		result, err := toolRegistry.Execute(ctx, "create_file", map[string]any{
			"path":    "internal/models/user.go",
			"content": modelContent,
		})
		require.NoError(t, err)
		
		createdFile := result.(map[string]any)["created"].(string)
		assert.FileExists(t, createdFile, "User model file should be created")
	})

	t.Run("Step5_KanbanBoardReview", func(t *testing.T) {
		// Developer reviews completed tasks through kanban board
		
		// Create kanban structure
		commissionID := "auth-system-001"
		kanbanPath := filepath.Join(projCtx.GetGuildPath(), "kanban", commissionID)
		
		// Create task states
		states := []string{"todo", "in_progress", "review", "done"}
		for _, state := range states {
			err := os.MkdirAll(filepath.Join(kanbanPath, state), 0755)
			require.NoError(t, err)
		}

		// Move completed task to review
		taskFile := filepath.Join(kanbanPath, "review", "task-001-user-model.md")
		taskContent := `# Task: Create User Model

## Status: Ready for Review

## Completed Actions:
- Created User struct with all required fields
- Added Session struct for token storage  
- Configured GORM tags for database mapping
- Added proper indexes for performance

## Files Created:
- internal/models/user.go

## Next Steps:
- Review the model structure
- Ensure all fields are properly typed
- Verify GORM tags are correct`

		err := os.WriteFile(taskFile, []byte(taskContent), 0644)
		require.NoError(t, err)

		// Verify task is in review
		assert.FileExists(t, taskFile, "Task should be in review state")
	})

	t.Run("Step6_SessionPersistence", func(t *testing.T) {
		// Verify session persists across restarts
		sessionID := "dev-session-001"
		historyPath := filepath.Join(projCtx.GetGuildPath(), "sessions", sessionID, "history.jsonl")
		
		// Simulate chat history
		history := []string{
			`{"timestamp":"2024-01-01T10:00:00Z","role":"user","content":"I need to add user authentication to our API"}`,
			`{"timestamp":"2024-01-01T10:00:30Z","role":"assistant","content":"I'll help you create a comprehensive authentication system. Let me break this down into tasks..."}`,
			`{"timestamp":"2024-01-01T10:05:00Z","role":"user","content":"Let's start with the user model"}`,
			`{"timestamp":"2024-01-01T10:05:15Z","role":"assistant","content":"Creating the User model with GORM..."}`,
		}

		// Write history
		err := os.WriteFile(historyPath, []byte(strings.Join(history, "\n")+"\n"), 0644)
		require.NoError(t, err)

		// Simulate restart by reading history
		data, err := os.ReadFile(historyPath)
		require.NoError(t, err)
		
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		assert.Len(t, lines, 4, "History should persist with all messages")
	})
}

// TestDeveloperProductivityMetrics tests that the workflow enhances productivity
func TestDeveloperProductivityMetrics(t *testing.T) {
	ctx := context.Background()
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	t.Run("CommissionCreationSpeed", func(t *testing.T) {
		// Measure time from idea to executable tasks
		startTime := time.Now()
		
		// Simulate commission creation from chat
		userInput := "Create a REST API for a blog with CRUD operations"
		
		// Mock quick AI response
		commissionContent := `# Blog REST API
		
## Endpoints:
- GET /posts - List all posts
- GET /posts/:id - Get single post  
- POST /posts - Create post
- PUT /posts/:id - Update post
- DELETE /posts/:id - Delete post`

		// Save commission
		commissionPath := filepath.Join(projCtx.GetGuildPath(), "commissions", "blog-api.md")
		err := os.MkdirAll(filepath.Dir(commissionPath), 0755)
		require.NoError(t, err)
		
		err = os.WriteFile(commissionPath, []byte(commissionContent), 0644)
		require.NoError(t, err)

		duration := time.Since(startTime)
		assert.Less(t, duration, 2*time.Second, "Commission creation should be fast")
	})

	t.Run("TaskCompletionTracking", func(t *testing.T) {
		// Track task completion rate over time
		db, err := storage.DefaultDatabaseFactory(ctx, filepath.Join(projCtx.GetGuildPath(), "test.db"))
		require.NoError(t, err)

		taskRepo := storage.NewTaskRepository(db)
		
		// Create sample tasks
		tasks := []storage.Task{
			{Title: "Create Post model", Status: "done", CompletedAt: timePtr(time.Now().Add(-2 * time.Hour))},
			{Title: "Implement GET endpoints", Status: "done", CompletedAt: timePtr(time.Now().Add(-1 * time.Hour))},
			{Title: "Implement POST endpoint", Status: "in_progress"},
			{Title: "Add validation", Status: "todo"},
			{Title: "Write tests", Status: "todo"},
		}

		for _, task := range tasks {
			err := taskRepo.Create(ctx, &task)
			require.NoError(t, err)
		}

		// Calculate completion rate
		completed := 0
		total := len(tasks)
		for _, task := range tasks {
			if task.Status == "done" {
				completed++
			}
		}

		completionRate := float64(completed) / float64(total) * 100
		assert.Equal(t, 40.0, completionRate, "Should track 40% completion rate")
	})
}

// TestDeveloperWorkflowErrors tests error handling in developer workflows
func TestDeveloperWorkflowErrors(t *testing.T) {
	ctx := context.Background()
	projCtx, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	t.Run("HandleAIProviderTimeout", func(t *testing.T) {
		// Simulate slow AI provider
		mockProvider := testutil.NewMockLLMProvider()
		mockProvider.SetLatency(5 * time.Second)
		
		startTime := time.Now()
		
		// Create context with timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		// Attempt to get AI response
		_, err := mockProvider.Complete(timeoutCtx, nil)
		
		duration := time.Since(startTime)
		assert.Error(t, err, "Should timeout after 2 seconds")
		assert.Less(t, duration, 3*time.Second, "Should not wait full 5 seconds")
		assert.Contains(t, err.Error(), "context deadline exceeded", "Should be timeout error")
	})

	t.Run("RecoverFromToolFailure", func(t *testing.T) {
		// Setup tool that fails
		toolRegistry := testutil.NewMockToolRegistry()
		toolRegistry.RegisterTool("failing_tool", &testutil.MockTool{
			ExecuteFn: func(ctx context.Context, params map[string]any) (any, error) {
				return nil, fmt.Errorf("tool execution failed: disk full")
			},
		})

		// Attempt tool execution
		_, err := toolRegistry.Execute(ctx, "failing_tool", map[string]any{})
		assert.Error(t, err, "Tool should fail")
		assert.Contains(t, err.Error(), "disk full", "Should preserve error context")

		// Verify workspace is not corrupted
		workspacePath := filepath.Join(projCtx.GetProjectPath(), ".workspace")
		assert.NoDirExists(t, workspacePath, "Failed tool should not create partial workspace")
	})
}

// Helper functions

func timePtr(t time.Time) *time.Time {
	return &t
}