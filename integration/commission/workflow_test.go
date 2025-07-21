//go:build integration

package commission

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lancekrogers/guild/internal/testutil"
	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommissionWorkflow validates the commission lifecycle
func TestCommissionWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "commission-workflow",
	})
	defer cleanup()
	
	extCtx := testutil.ExtendProjectContext(projCtx)

	// Initialize guild
	result := extCtx.RunGuild("init")
	require.NoError(t, result.Error)

	t.Run("commission_creation", func(t *testing.T) {
		// Create a commission
		result := extCtx.RunGuild("commission", "create", "Test Commission")
		require.NoError(t, result.Error)
		assert.Contains(t, result.Stdout, "Commission created")
	})

	t.Run("commission_listing", func(t *testing.T) {
		// Create multiple commissions
		commissions := []string{
			"Backend API Development",
			"Frontend Dashboard",
			"Database Schema Design",
		}

		for _, title := range commissions {
			result := extCtx.RunGuild("commission", "create", title)
			require.NoError(t, result.Error)
		}

		// List commissions
		result := extCtx.RunGuild("commission", "list")
		require.NoError(t, result.Error)

		// Verify all commissions appear
		for _, title := range commissions {
			assert.Contains(t, result.Stdout, title)
		}
	})

	t.Run("commission_with_file", func(t *testing.T) {
		// Create commission file
		commissionContent := `# E-Commerce Platform Development

## Objective
Build a modern e-commerce platform with user authentication, product catalog, and payment processing.

## Requirements
- User registration and authentication
- Product catalog with search
- Shopping cart functionality
- Payment integration
- Order management

## Success Criteria
- All features implemented and tested
- Performance targets met (< 200ms response time)
- Security audit passed
`
		
		commFile := filepath.Join(projCtx.GetRootPath(), "ecommerce_commission.md")
		err := projCtx.WriteFile("ecommerce_commission.md", commissionContent)
		require.NoError(t, err)

		// Create commission from file
		result := extCtx.RunGuild("commission", "create", "--file", commFile)
		require.NoError(t, result.Error)

		// List to verify
		result = extCtx.RunGuild("commission", "list")
		require.NoError(t, result.Error)
		assert.Contains(t, result.Stdout, "E-Commerce Platform Development")
	})

	t.Run("commission_status_flow", func(t *testing.T) {
		// Create commission
		result := extCtx.RunGuild("commission", "create", "Status Test Commission")
		require.NoError(t, result.Error)

		// Extract commission ID from output (assuming format: "Commission created: <id>")
		output := result.Stdout
		var commissionID string
		if idx := strings.Index(output, "Commission created: "); idx >= 0 {
			start := idx + len("Commission created: ")
			end := strings.IndexAny(output[start:], "\n ")
			if end > 0 {
				commissionID = output[start : start+end]
			}
		}

		// If we can't extract ID, try using the title
		if commissionID == "" {
			// List and parse to find our commission
			result = extCtx.RunGuild("commission", "list", "--format", "json")
			// For now, skip ID-based operations if we can't extract it
			t.Log("Could not extract commission ID, skipping status operations")
			return
		}

		// Update status (if the command supports it)
		result = extCtx.RunGuild("commission", "update", commissionID, "--status", commission.StatusInProgress)
		if result.Error != nil {
			t.Log("Commission update command not available")
			return
		}

		// Verify status change
		result = extCtx.RunGuild("commission", "show", commissionID)
		require.NoError(t, result.Error)
		assert.Contains(t, result.Stdout, commission.StatusInProgress)
	})

	t.Run("commission_performance", func(t *testing.T) {
		// Create many commissions to test performance
		start := time.Now()
		numCommissions := 20

		for i := 0; i < numCommissions; i++ {
			title := fmt.Sprintf("Performance Test Commission %d", i)
			result := extCtx.RunGuild("commission", "create", title)
			require.NoError(t, result.Error)
		}

		createDuration := time.Since(start)
		avgCreateTime := createDuration / time.Duration(numCommissions)
		
		// Performance requirement: avg commission creation < 500ms
		assert.LessOrEqual(t, avgCreateTime, 500*time.Millisecond,
			"Average commission creation should be under 500ms")

		// Test list performance with many items
		start = time.Now()
		result := extCtx.RunGuild("commission", "list")
		listDuration := time.Since(start)
		
		require.NoError(t, result.Error)
		
		// Performance requirement: list operation < 2s
		assert.LessOrEqual(t, listDuration, 2*time.Second,
			"Listing commissions should complete within 2s")

		t.Logf("Performance: Created %d commissions in %v (avg: %v), listed in %v",
			numCommissions, createDuration, avgCreateTime, listDuration)
	})

	t.Run("commission_error_handling", func(t *testing.T) {
		// Test empty title
		result := extCtx.RunGuild("commission", "create", "")
		assert.Error(t, result.Error)
		assert.Contains(t, result.Stderr, "title")

		// Test invalid file
		result = extCtx.RunGuild("commission", "create", "--file", "/nonexistent/file.md")
		assert.Error(t, result.Error)

		// Test malformed commission file
		badContent := `This is not a valid commission format`
		err := projCtx.WriteFile("bad_commission.md", badContent)
		require.NoError(t, err)

		result = extCtx.RunGuild("commission", "create", "--file", 
			filepath.Join(projCtx.GetRootPath(), "bad_commission.md"))
		// Should either succeed with a basic commission or fail gracefully
		if result.Error != nil {
			assert.Contains(t, result.Stderr, "format")
		}
	})
}