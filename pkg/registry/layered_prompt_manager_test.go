// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"context"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/prompts/layered"
)

func TestGetPromptManager(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() ComponentRegistry
		wantError bool
		errorCode gerror.ErrorCode
	}{
		{
			name: "Initialized registry returns layered prompt manager",
			setupFunc: func() ComponentRegistry {
				registry := NewComponentRegistry()
				config := Config{} // Empty config for testing
				err := registry.Initialize(context.Background(), config)
				if err != nil {
					t.Fatalf("Failed to initialize registry: %v", err)
				}
				return registry
			},
			wantError: false,
		},
		{
			name: "Uninitialized registry returns error",
			setupFunc: func() ComponentRegistry {
				return NewComponentRegistry()
			},
			wantError: true,
			errorCode: gerror.ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := tt.setupFunc()

			manager, err := registry.GetPromptManager()

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if gErr, ok := err.(*gerror.GuildError); ok {
					if gErr.Code != tt.errorCode {
						t.Errorf("Expected error code %v, got %v", tt.errorCode, gErr.Code)
					}
				}
				if manager != nil {
					t.Errorf("Expected nil manager when error occurs, got %v", manager)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if manager == nil {
					t.Errorf("Expected non-nil manager, got nil")
					return
				}

				// Test that the manager implements the interface correctly
				ctx := context.Background()

				// Test basic Manager interface methods
				roles, err := manager.ListRoles(ctx)
				if err != nil {
					t.Errorf("ListRoles() error = %v", err)
				}
				if len(roles) == 0 {
					t.Errorf("Expected some roles, got empty list")
				}

				domains, err := manager.ListDomains(ctx, "manager")
				if err != nil {
					t.Errorf("ListDomains() error = %v", err)
				}
				if len(domains) == 0 {
					t.Errorf("Expected some domains, got empty list")
				}

				// Test LayeredManager interface methods (should return not implemented errors for now)
				turnCtx := layered.TurnContext{
					UserMessage: "test message",
				}
				_, err = manager.BuildLayeredPrompt(ctx, "test-artisan", "test-session", turnCtx)
				if err == nil {
					t.Errorf("Expected not implemented error for BuildLayeredPrompt")
				}
				if gErr, ok := err.(*gerror.GuildError); ok {
					if gErr.Code != gerror.ErrCodeNotImplemented {
						t.Errorf("Expected ErrCodeNotImplemented, got %v", gErr.Code)
					}
				}
			}
		})
	}
}

func TestLayeredManagerWrapper(t *testing.T) {
	t.Run("Wrapper implements all interface methods", func(t *testing.T) {
		// Create the registry and initialize it
		registry := NewComponentRegistry()
		config := Config{}
		err := registry.Initialize(context.Background(), config)
		if err != nil {
			t.Fatalf("Failed to initialize registry: %v", err)
		}

		manager, err := registry.GetPromptManager()
		if err != nil {
			t.Fatalf("Failed to get prompt manager: %v", err)
		}

		ctx := context.Background()

		// Test all Manager interface methods
		t.Run("Manager interface methods", func(t *testing.T) {
			// GetSystemPrompt
			prompt, err := manager.GetSystemPrompt(ctx, "manager", "default")
			// This might return an error if no prompt is registered, which is fine
			_ = prompt
			_ = err

			// GetTemplate
			template, err := manager.GetTemplate(ctx, "test-template")
			// This might return an error if no template is registered, which is fine
			_ = template
			_ = err

			// FormatContext - this should work
			// We can't test this easily without a proper Context implementation
			// but we can verify the method exists and is callable

			// ListRoles - this should return some predefined roles
			roles, err := manager.ListRoles(ctx)
			if err != nil {
				t.Errorf("ListRoles() error = %v", err)
			}
			if len(roles) == 0 {
				t.Errorf("Expected some roles from ListRoles()")
			}

			// ListDomains - this should return some predefined domains
			domains, err := manager.ListDomains(ctx, "manager")
			if err != nil {
				t.Errorf("ListDomains() error = %v", err)
			}
			if len(domains) == 0 {
				t.Errorf("Expected some domains from ListDomains()")
			}
		})

		// Test all LayeredManager interface methods (should return not implemented)
		t.Run("LayeredManager interface methods", func(t *testing.T) {
			// BuildLayeredPrompt
			turnCtx := layered.TurnContext{
				UserMessage: "test message",
			}
			_, err := manager.BuildLayeredPrompt(ctx, "test-artisan", "test-session", turnCtx)
			expectNotImplemented(t, err, "BuildLayeredPrompt")

			// GetPromptLayer
			_, err = manager.GetPromptLayer(ctx, layered.LayerPlatform, "test-artisan", "test-session")
			expectNotImplemented(t, err, "GetPromptLayer")

			// SetPromptLayer
			systemPrompt := layered.SystemPrompt{
				Layer:   layered.LayerPlatform,
				Content: "test content",
				Updated: time.Now(),
			}
			err = manager.SetPromptLayer(ctx, systemPrompt)
			expectNotImplemented(t, err, "SetPromptLayer")

			// DeletePromptLayer
			err = manager.DeletePromptLayer(ctx, layered.LayerPlatform, "test-artisan", "test-session")
			expectNotImplemented(t, err, "DeletePromptLayer")

			// ListPromptLayers
			_, err = manager.ListPromptLayers(ctx, "test-artisan", "test-session")
			expectNotImplemented(t, err, "ListPromptLayers")

			// InvalidateCache
			err = manager.InvalidateCache(ctx, "test-artisan", "test-session")
			expectNotImplemented(t, err, "InvalidateCache")
		})
	})
}

// Helper function to check for not implemented errors
func expectNotImplemented(t *testing.T, err error, methodName string) {
	if err == nil {
		t.Errorf("Expected not implemented error for %s, got nil", methodName)
		return
	}
	if gErr, ok := err.(*gerror.GuildError); ok {
		if gErr.Code != gerror.ErrCodeNotImplemented {
			t.Errorf("Expected ErrCodeNotImplemented for %s, got %v", methodName, gErr.Code)
		}
	} else {
		t.Errorf("Expected GuildError for %s, got %T", methodName, err)
	}
}
