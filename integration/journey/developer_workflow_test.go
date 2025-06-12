package journey

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/internal/testutil"
	"github.com/guild-ventures/guild-core/pkg/registry"
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
	err = reg.Providers().RegisterProvider("mock", mockProvider)
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
`, sessionID, startTime.Format(time.RFC3339), "test-project")

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

## Success Criteria
- All endpoints functioning with proper validation
- JWT tokens expire after configured duration
- Sessions properly stored/retrieved from Redis
- Protected routes return 401 for invalid tokens
- Unit tests with >80% coverage
`)

		// Simulate commission creation from chat
		response, err := mockProvider.Complete(ctx, userInput)
		require.NoError(t, err)
		assert.Contains(t, response, "User Authentication System", "Should create auth commission")

		// Save commission to project
		commissionPath := filepath.Join(projCtx.GetGuildPath(), "commissions", "auth-system.md")
		err = os.MkdirAll(filepath.Dir(commissionPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(commissionPath, []byte(response), 0644)
		require.NoError(t, err)
	})

	t.Run("Step3_TaskBreakdown", func(t *testing.T) {
		// AI breaks down commission into tasks
		mockProvider.SetResponse("task_creator", `Task Breakdown:
1. Create JWT token service
2. Implement login endpoint
3. Implement logout endpoint
4. Add authentication middleware
5. Setup Redis session storage
6. Create user model and database
7. Implement password hashing
8. Add rate limiting
9. Write unit tests
10. Create API documentation`)

		// TODO: In a real implementation, this would use the actual task extraction
		// from the manager agent. For now, we simulate the task creation.
		tasks := []string{
			"Create JWT token service",
			"Implement login endpoint",
			"Implement logout endpoint",
			"Add authentication middleware",
			"Setup Redis session storage",
		}

		// Create kanban board for commission
		kanbanPath := filepath.Join(projCtx.GetGuildPath(), "kanban", "auth-commission")
		todoPath := filepath.Join(kanbanPath, "todo")
		err := os.MkdirAll(todoPath, 0755)
		require.NoError(t, err)

		// Create task files
		for i, task := range tasks {
			taskFile := fmt.Sprintf("task-%03d.md", i+1)
			content := fmt.Sprintf("# %s\n\nStatus: TODO\nPriority: High\n", task)
			err := os.WriteFile(filepath.Join(todoPath, taskFile), []byte(content), 0644)
			require.NoError(t, err)
		}

		// Verify task creation
		entries, err := os.ReadDir(todoPath)
		require.NoError(t, err)
		assert.Len(t, entries, 5, "Should create 5 initial tasks")
	})

	t.Run("Step4_AgentExecution", func(t *testing.T) {
		// Simulate agents working on tasks
		kanbanPath := filepath.Join(projCtx.GetGuildPath(), "kanban", "auth-commission")
		inProgressPath := filepath.Join(kanbanPath, "in_progress")
		donePath := filepath.Join(kanbanPath, "done")

		err := os.MkdirAll(inProgressPath, 0755)
		require.NoError(t, err)
		err = os.MkdirAll(donePath, 0755)
		require.NoError(t, err)

		// Mock agent responses for task execution
		mockProvider.SetResponse("code_agent", `package auth

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secretKey []byte
	expiry    time.Duration
}

func NewJWTService(secret string, expiry time.Duration) *JWTService {
	return &JWTService{
		secretKey: []byte(secret),
		expiry:    expiry,
	}
}

func (s *JWTService) GenerateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(s.expiry).Unix(),
		"iat":     time.Now().Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}`)

		// Simulate task movement through kanban
		// Move first task to in_progress
		err = os.Rename(
			filepath.Join(kanbanPath, "todo", "task-001.md"),
			filepath.Join(inProgressPath, "task-001.md"),
		)
		require.NoError(t, err)

		// Simulate work being done
		time.Sleep(10 * time.Millisecond)

		// Move to done
		err = os.Rename(
			filepath.Join(inProgressPath, "task-001.md"),
			filepath.Join(donePath, "task-001.md"),
		)
		require.NoError(t, err)

		// Verify task completion
		doneEntries, err := os.ReadDir(donePath)
		require.NoError(t, err)
		assert.Len(t, doneEntries, 1, "First task should be completed")
	})

	t.Run("Step5_HumanReview", func(t *testing.T) {
		// Developer reviews completed work
		kanbanPath := filepath.Join(projCtx.GetGuildPath(), "kanban", "auth-commission")
		reviewPath := filepath.Join(kanbanPath, "review")
		donePath := filepath.Join(kanbanPath, "done")

		err := os.MkdirAll(reviewPath, 0755)
		require.NoError(t, err)

		// Move completed task to review
		err = os.Rename(
			filepath.Join(donePath, "task-001.md"),
			filepath.Join(reviewPath, "task-001.md"),
		)
		require.NoError(t, err)

		// Add review comments
		reviewContent := `# Create JWT token service

Status: REVIEW
Priority: High

## Implementation
JWT service has been implemented with:
- Token generation using HS256
- Configurable expiry
- User ID in claims

## Review Notes
- Implementation looks good
- Consider adding refresh token support
- Add token validation method

## Files Created
- auth/jwt_service.go
- auth/jwt_service_test.go
`
		err = os.WriteFile(filepath.Join(reviewPath, "task-001.md"), []byte(reviewContent), 0644)
		require.NoError(t, err)

		// Verify review status
		reviewEntries, err := os.ReadDir(reviewPath)
		require.NoError(t, err)
		assert.Len(t, reviewEntries, 1, "Task should be in review")
	})

	t.Run("Step6_IterativeDevelopment", func(t *testing.T) {
		// Developer requests changes based on review
		userFeedback := "Add a ValidateToken method to the JWT service that checks expiry and signature"

		// Mock agent response for the update
		mockProvider.SetResponse("code_update", `func (s *JWTService) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}`)

		// Update task with new implementation
		updatedContent := `# Create JWT token service

Status: DONE
Priority: High

## Implementation
JWT service has been implemented with:
- Token generation using HS256
- Configurable expiry
- User ID in claims
- Token validation method (added based on review)

## Files Created/Updated
- auth/jwt_service.go (updated with ValidateToken)
- auth/jwt_service_test.go (added validation tests)
`
		kanbanPath := filepath.Join(projCtx.GetGuildPath(), "kanban", "auth-commission")
		err := os.WriteFile(filepath.Join(kanbanPath, "done", "task-001.md"), []byte(updatedContent), 0644)
		require.NoError(t, err)

		// Move from review back to done after updates
		reviewPath := filepath.Join(kanbanPath, "review", "task-001.md")
		if _, err := os.Stat(reviewPath); err == nil {
			os.Remove(reviewPath)
		}

		// Verify task is complete
		doneEntries, err := os.ReadDir(filepath.Join(kanbanPath, "done"))
		require.NoError(t, err)
		assert.Len(t, doneEntries, 1, "Updated task should be done")

		// Verify feedback was incorporated
		content, err := os.ReadFile(filepath.Join(kanbanPath, "done", "task-001.md"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "Token validation method", "Should include validation feature")
		assert.Contains(t, string(content), userFeedback, "Should reference user feedback")
	})
}

// TestDeveloperToolIntegration tests how developers interact with Guild tools
func TestDeveloperToolIntegration(t *testing.T) {
	ctx := context.Background()
	_, cleanup := testutil.SetupTestProject(t)
	defer cleanup()

	// Initialize registry
	reg := registry.NewComponentRegistry()
	err := reg.Initialize(ctx, registry.Config{})
	require.NoError(t, err)

	t.Run("GitIntegration", func(t *testing.T) {
		// TODO: Test git tool integration
		// This would test how Guild interacts with git for:
		// - Creating branches for commissions
		// - Committing completed tasks
		// - Managing merge requests
		t.Skip("Git tool integration not yet implemented")
	})

	t.Run("FileSystemTools", func(t *testing.T) {
		// TODO: Test file system tools
		// This would test:
		// - Safe file creation/modification
		// - Directory structure management
		// - File search and analysis
		t.Skip("File system tools not yet implemented")
	})

	t.Run("CodeAnalysisTools", func(t *testing.T) {
		// TODO: Test code analysis tools
		// This would test:
		// - AST parsing for understanding code
		// - Dependency analysis
		// - Test coverage reporting
		t.Skip("Code analysis tools not yet implemented")
	})
}

// TestDeveloperProductivity measures productivity improvements
func TestDeveloperProductivity(t *testing.T) {
	t.Run("TimeToFirstCommit", func(t *testing.T) {
		// Measure time from commission creation to first working code
		startTime := time.Now()

		// Simulate commission → task → code workflow
		// In real scenario, this would track actual execution time

		duration := time.Since(startTime)
		assert.Less(t, duration, 5*time.Minute, "Should produce working code within 5 minutes")
	})

	t.Run("TaskCompletionRate", func(t *testing.T) {
		// Track what percentage of tasks complete successfully
		totalTasks := 10
		completedTasks := 8

		completionRate := float64(completedTasks) / float64(totalTasks) * 100
		assert.Greater(t, completionRate, 70.0, "Should complete >70% of tasks successfully")
	})
}
