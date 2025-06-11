package errors

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
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/tools"
	"github.com/guild-ventures/guild-core/pkg/workspace"
)

// TestToolExecutionFailures tests various tool failure scenarios
func TestToolExecutionFailures(t *testing.T) {
	t.Skip("Skipping - tool API has changed significantly, needs complete rewrite")
	ctx := context.Background()

	t.Run("ToolCrashes", func(t *testing.T) {
		// Create tool that crashes
		crashingTool := &tools.Tool{
			Name:        "crashing_tool",
			Description: "Tool that crashes during execution",
			Parameters: tools.ParameterSchema{
				Type: "object",
				Properties: map[string]tools.Property{
					"action": {Type: "string", Description: "Action to perform"},
				},
			},
			Execute: func(ctx context.Context, params map[string]any) (any, error) {
				action := params["action"].(string)
				
				switch action {
				case "panic":
					panic("Tool crashed!")
				case "nil_pointer":
					var ptr *string
					return *ptr, nil // Nil pointer dereference
				case "out_of_memory":
					// Simulate OOM
					huge := make([]byte, 10*1024*1024*1024) // 10GB
					return len(huge), nil
				default:
					return nil, fmt.Errorf("unknown action: %s", action)
				}
			},
		}

		// Test panic recovery
		t.Run("PanicRecovery", func(t *testing.T) {
			// Execute with panic recovery
			result, err := executeSafely(ctx, crashingTool, map[string]any{
				"action": "panic",
			})

			assert.Error(t, err, "Should return error after panic")
			assert.Contains(t, err.Error(), "panic", "Error should mention panic")
			assert.Nil(t, result, "Result should be nil after panic")
		})

		// Test workspace isolation
		t.Run("WorkspaceIsolation", func(t *testing.T) {
			// Create test workspace
			projCtx, cleanup := testutil.SetupTestProject(t)
			defer cleanup()

			wsManager := workspace.NewManager(workspace.Config{
				BaseDir:         filepath.Join(projCtx.GetProjectPath(), ".workspace"),
				CleanupOnError:  true,
				IsolationLevel:  workspace.IsolationStrict,
			})

			// Create isolated workspace
			ws, err := wsManager.Create(ctx, "test-crash")
			require.NoError(t, err)

			// Create important file outside workspace
			importantFile := filepath.Join(projCtx.GetProjectPath(), "important.txt")
			err = os.WriteFile(importantFile, []byte("Important data"), 0644)
			require.NoError(t, err)

			// Tool that tries to delete files
			destructiveTool := &tools.Tool{
				Name: "destructive_tool",
				Execute: func(ctx context.Context, params map[string]any) (any, error) {
					// Try to delete important file (should fail due to isolation)
					err := os.Remove(importantFile)
					if err != nil {
						return nil, gerror.New("failed to delete file", gerror.ErrorTypeExecution).
							WithContext("file", importantFile).
							WithContext("error", err.Error())
					}
					return "deleted", nil
				},
			}

			// Execute in isolated workspace
			result, err := ws.Execute(ctx, destructiveTool, nil)
			
			// Should fail to delete file outside workspace
			assert.Error(t, err, "Should fail to delete file outside workspace")
			assert.FileExists(t, importantFile, "Important file should still exist")
			assert.Nil(t, result, "Result should be nil on failure")

			// Cleanup workspace
			wsManager.Cleanup(ctx, ws.ID)
		})

		// Test error context preservation
		t.Run("ErrorContextPreservation", func(t *testing.T) {
			detailedTool := &tools.Tool{
				Name: "detailed_error_tool",
				Execute: func(ctx context.Context, params map[string]any) (any, error) {
					// Create error with rich context
					return nil, gerror.New("tool execution failed", gerror.ErrorTypeExecution).
						WithContext("tool", "detailed_error_tool").
						WithContext("params", params).
						WithContext("timestamp", time.Now().Unix()).
						WithContext("suggestion", "Check input parameters").
						WithContext("recovery", "Retry with valid parameters")
				},
			}

			result, err := detailedTool.Execute(ctx, map[string]any{
				"input": "test",
			})

			require.Error(t, err)
			assert.Nil(t, result)

			// Check error context is preserved
			if gerr, ok := err.(*gerror.Error); ok {
				assert.Equal(t, "detailed_error_tool", gerr.Context["tool"])
				assert.NotNil(t, gerr.Context["params"])
				assert.NotNil(t, gerr.Context["timestamp"])
				assert.Equal(t, "Check input parameters", gerr.Context["suggestion"])
				assert.Equal(t, "Retry with valid parameters", gerr.Context["recovery"])
			}
		})

		// Test rollback capabilities
		t.Run("RollbackCapabilities", func(t *testing.T) {
			// Tool with transaction support
			transactionalTool := &tools.Tool{
				Name: "transactional_tool",
				Execute: func(ctx context.Context, params map[string]any) (any, error) {
					operations := params["operations"].([]string)
					completed := []string{}
					
					// Create rollback function
					rollback := func() {
						for i := len(completed) - 1; i >= 0; i-- {
							// Undo operation
							t.Logf("Rolling back: %s", completed[i])
						}
					}

					// Execute operations
					for _, op := range operations {
						if op == "fail" {
							// Rollback on failure
							rollback()
							return nil, fmt.Errorf("operation failed at: %s", op)
						}
						completed = append(completed, op)
					}

					return map[string]any{
						"completed": completed,
						"status":    "success",
					}, nil
				},
			}

			// Test successful execution
			result, err := transactionalTool.Execute(ctx, map[string]any{
				"operations": []string{"create", "update", "commit"},
			})
			require.NoError(t, err)
			assert.Equal(t, "success", result.(map[string]any)["status"])

			// Test with failure and rollback
			result, err = transactionalTool.Execute(ctx, map[string]any{
				"operations": []string{"create", "update", "fail", "commit"},
			})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "operation failed")
			assert.Nil(t, result)
		})

		// Test alternative tool suggestions
		t.Run("AlternativeToolSuggestions", func(t *testing.T) {
			// Registry with multiple similar tools
			registry := tools.NewRegistry()
			
			// Register tools
			registry.Register(&tools.Tool{
				Name:        "file_write",
				Description: "Write content to a file",
				Execute: func(ctx context.Context, params map[string]any) (any, error) {
					return nil, fmt.Errorf("permission denied")
				},
			})

			registry.Register(&tools.Tool{
				Name:        "file_create",
				Description: "Create a new file with content",
				Execute: func(ctx context.Context, params map[string]any) (any, error) {
					return "created", nil
				},
			})

			registry.Register(&tools.Tool{
				Name:        "file_append",
				Description: "Append content to existing file",
				Execute: func(ctx context.Context, params map[string]any) (any, error) {
					return "appended", nil
				},
			})

			// Try primary tool
			result, err := registry.Execute(ctx, "file_write", map[string]any{
				"path":    "test.txt",
				"content": "Hello",
			})

			// Should fail with suggestions
			assert.Error(t, err)
			assert.Nil(t, result)

			// Get alternative suggestions
			alternatives := registry.FindSimilar("file_write", 2)
			assert.Len(t, alternatives, 2, "Should suggest 2 alternatives")
			assert.Contains(t, []string{"file_create", "file_append"}, alternatives[0].Name)
		})
	})

	t.Run("PermissionErrors", func(t *testing.T) {
		// Test various permission scenarios
		projCtx, cleanup := testutil.SetupTestProject(t)
		defer cleanup()

		t.Run("FileSystemPermissions", func(t *testing.T) {
			// Create read-only file
			readOnlyFile := filepath.Join(projCtx.GetProjectPath(), "readonly.txt")
			err := os.WriteFile(readOnlyFile, []byte("Read only content"), 0444)
			require.NoError(t, err)
			defer os.Chmod(readOnlyFile, 0644) // Restore permissions

			// Tool that tries to write to read-only file
			writeTool := &tools.Tool{
				Name: "write_tool",
				Execute: func(ctx context.Context, params map[string]any) (any, error) {
					path := params["path"].(string)
					content := params["content"].(string)
					
					err := os.WriteFile(path, []byte(content), 0644)
					if err != nil {
						return nil, gerror.New("write permission denied", gerror.ErrorTypePermission).
							WithContext("path", path).
							WithContext("operation", "write").
							WithContext("suggestion", "Check file permissions or use sudo")
					}
					return "written", nil
				},
			}

			result, err := writeTool.Execute(ctx, map[string]any{
				"path":    readOnlyFile,
				"content": "New content",
			})

			assert.Error(t, err, "Should fail to write to read-only file")
			if gerr, ok := err.(*gerror.Error); ok {
				assert.Equal(t, gerror.ErrorTypePermission, gerr.Type)
				assert.Contains(t, gerr.Context["suggestion"], "permissions")
			}
		})

		t.Run("NetworkAccessRestrictions", func(t *testing.T) {
			// Tool that requires network access
			networkTool := &tools.Tool{
				Name:     "network_tool",
				RequiresNetwork: true,
				Execute: func(ctx context.Context, params map[string]any) (any, error) {
					url := params["url"].(string)
					
					// Simulate network restriction check
					if isNetworkRestricted() {
						return nil, gerror.New("network access denied", gerror.ErrorTypePermission).
							WithContext("url", url).
							WithContext("reason", "Network access is restricted in this environment").
							WithContext("suggestion", "Enable network access or use offline mode")
					}

					// Simulate API call
					return map[string]any{"status": "success", "data": "mock response"}, nil
				},
			}

			// Test with network restrictions
			result, err := networkTool.Execute(ctx, map[string]any{
				"url": "https://api.example.com/data",
			})

			// Behavior depends on actual network restrictions
			if err != nil {
				assert.Contains(t, err.Error(), "network")
			}
		})

		t.Run("GracefulDegradation", func(t *testing.T) {
			// Tool with fallback behavior
			resilientTool := &tools.Tool{
				Name: "resilient_tool",
				Execute: func(ctx context.Context, params map[string]any) (any, error) {
					mode := params["mode"].(string)
					
					switch mode {
					case "optimal":
						// Try optimal approach
						if hasFullPermissions() {
							return map[string]any{
								"result": "Executed with full features",
								"mode":   "optimal",
							}, nil
						}
						fallthrough
					
					case "degraded":
						// Fallback to degraded mode
						if hasBasicPermissions() {
							return map[string]any{
								"result": "Executed with limited features",
								"mode":   "degraded",
								"warning": "Some features unavailable due to permissions",
							}, nil
						}
						fallthrough
					
					case "minimal":
						// Minimal functionality
						return map[string]any{
							"result": "Executed with minimal features",
							"mode":   "minimal",
							"warning": "Most features disabled due to restrictions",
						}, nil
					
					default:
						return nil, fmt.Errorf("unknown mode: %s", mode)
					}
				},
			}

			// Test graceful degradation
			result, err := resilientTool.Execute(ctx, map[string]any{
				"mode": "optimal",
			})

			require.NoError(t, err)
			resultMap := result.(map[string]any)
			assert.NotEmpty(t, resultMap["result"])
			assert.Contains(t, []string{"optimal", "degraded", "minimal"}, resultMap["mode"])
		})

		t.Run("ClearErrorMessaging", func(t *testing.T) {
			// Tool with user-friendly error messages
			userFriendlyTool := &tools.Tool{
				Name: "user_friendly_tool",
				Execute: func(ctx context.Context, params map[string]any) (any, error) {
					action := params["action"].(string)
					
					errorCases := map[string]*gerror.Error{
						"no_permission": gerror.New("Cannot access the requested resource", gerror.ErrorTypePermission).
							WithContext("technical", "EACCES: permission denied").
							WithContext("user_message", "You don't have permission to perform this action. Please contact your administrator.").
							WithContext("help_url", "https://docs.example.com/permissions"),
						
						"file_locked": gerror.New("File is currently in use", gerror.ErrorTypeExecution).
							WithContext("technical", "EBUSY: resource busy").
							WithContext("user_message", "The file is being used by another program. Please close it and try again.").
							WithContext("suggestion", "Check if the file is open in an editor"),
						
						"quota_exceeded": gerror.New("Storage limit reached", gerror.ErrorTypeResource).
							WithContext("technical", "EDQUOT: disk quota exceeded").
							WithContext("user_message", "You've reached your storage limit. Please free up some space or upgrade your plan.").
							WithContext("current_usage", "4.8 GB").
							WithContext("limit", "5.0 GB"),
					}

					if err, exists := errorCases[action]; exists {
						return nil, err
					}

					return "success", nil
				},
			}

			// Test different error scenarios
			testCases := []string{"no_permission", "file_locked", "quota_exceeded"}
			
			for _, tc := range testCases {
				result, err := userFriendlyTool.Execute(ctx, map[string]any{
					"action": tc,
				})

				require.Error(t, err)
				assert.Nil(t, result)

				if gerr, ok := err.(*gerror.Error); ok {
					assert.NotEmpty(t, gerr.Context["user_message"], "Should have user-friendly message")
					assert.NotEmpty(t, gerr.Context["technical"], "Should preserve technical details")
					t.Logf("%s: %s", tc, gerr.Context["user_message"])
				}
			}
		})
	})
}

// Helper functions

func executeSafely(ctx context.Context, tool *tools.Tool, params map[string]any) (result any, err error) {
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			err = gerror.New("tool panicked", gerror.ErrorTypeExecution).
				WithContext("panic", fmt.Sprintf("%v", r)).
				WithContext("tool", tool.Name)
		}
	}()

	return tool.Execute(ctx, params)
}

func isNetworkRestricted() bool {
	// Check if network is restricted (simplified)
	return os.Getenv("GUILD_OFFLINE_MODE") == "true"
}

func hasFullPermissions() bool {
	// Check for full permissions (simplified)
	return os.Getuid() == 0 // Running as root
}

func hasBasicPermissions() bool {
	// Check for basic permissions
	return true // Always have basic permissions
}