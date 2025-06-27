// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package lsp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/lsp"
	"github.com/lancekrogers/guild/pkg/lsp/mocks"
)

// TestManagerWithMocks tests the LSP manager functionality using mocks
// This doesn't require a real LSP server like gopls
func TestManagerWithMocks(t *testing.T) {
	ctx := context.Background()

	t.Run("MockClient_BasicOperations", func(t *testing.T) {
		// Create a mock client
		mockClient := mocks.NewMockLSPClient()

		// Test Start
		err := mockClient.Start(ctx)
		require.NoError(t, err)
		assert.True(t, mockClient.IsReady())

		// Test Initialize
		params := lsp.InitializeParams{
			RootURI: "file:///test/workspace",
			Capabilities: lsp.ClientCapabilities{
				TextDocument: &lsp.TextDocumentClientCapabilities{
					Completion: &lsp.CompletionClientCapabilities{
						CompletionItem: &lsp.CompletionItemCapabilities{
							SnippetSupport: true,
						},
					},
				},
			},
		}

		result, err := mockClient.Initialize(ctx, params)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test Completion
		completionParams := lsp.CompletionParams{
			TextDocumentPositionParams: lsp.TextDocumentPositionParams{
				TextDocument: lsp.TextDocumentIdentifier{
					URI: "file:///test/main.go",
				},
				Position: lsp.Position{
					Line:      6,
					Character: 5,
				},
			},
		}

		completions, err := mockClient.Completion(ctx, completionParams)
		require.NoError(t, err)
		assert.NotNil(t, completions)
		assert.Equal(t, 1, mockClient.GetCompletionCalls())

		// Test Hover
		hoverParams := lsp.HoverParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: "file:///test/main.go",
			},
			Position: lsp.Position{
				Line:      6,
				Character: 5,
			},
		}

		hover, err := mockClient.Hover(ctx, hoverParams)
		require.NoError(t, err)
		assert.NotNil(t, hover)
		assert.Equal(t, 1, mockClient.GetHoverCalls())

		// Test Definition
		defParams := lsp.DefinitionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: "file:///test/main.go",
			},
			Position: lsp.Position{
				Line:      2,
				Character: 8,
			},
		}

		locations, err := mockClient.Definition(ctx, defParams)
		require.NoError(t, err)
		assert.NotNil(t, locations)

		// Test References
		refParams := lsp.ReferenceParams{
			TextDocumentPositionParams: lsp.TextDocumentPositionParams{
				TextDocument: lsp.TextDocumentIdentifier{
					URI: "file:///test/main.go",
				},
				Position: lsp.Position{
					Line:      5,
					Character: 1,
				},
			},
			Context: lsp.ReferenceContext{
				IncludeDeclaration: true,
			},
		}

		refs, err := mockClient.References(ctx, refParams)
		require.NoError(t, err)
		assert.NotNil(t, refs)

		// Test Stop
		err = mockClient.Stop(ctx)
		require.NoError(t, err)
		assert.False(t, mockClient.IsReady())
	})

	t.Run("MockClient_ErrorHandling", func(t *testing.T) {
		mockClient := mocks.NewMockLSPClient()

		// Test start error
		mockClient.SetStartError(assert.AnError)
		err := mockClient.Start(ctx)
		assert.Error(t, err)

		// Reset for other tests
		mockClient.SetStartError(nil)
		err = mockClient.Start(ctx)
		require.NoError(t, err)

		// Test completion error by setting client as not ready
		mockClient.SetReady(false)
		completions, err := mockClient.Completion(ctx, lsp.CompletionParams{})
		// Should work even if not ready for mock
		assert.NoError(t, err)
		assert.NotNil(t, completions)
	})

	t.Run("MockProcessLauncher", func(t *testing.T) {
		launcher := mocks.NewMockProcessLauncher()

		// Test successful launch
		client, err := launcher.LaunchServer(ctx, "gopls", []string{"-mode=stdio"}, "/test/workspace")
		require.NoError(t, err)
		assert.NotNil(t, client)

		// Verify the returned client is ready
		assert.True(t, client.IsReady())

		// Test launch error
		launcher.SetLaunchError(assert.AnError)
		client, err = launcher.LaunchServer(ctx, "invalid-lsp", []string{}, "/test")
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

// TestLSPUtilities tests utility functions without requiring a real LSP server
func TestLSPUtilities(t *testing.T) {
	t.Run("DetectLanguage", func(t *testing.T) {
		tests := []struct {
			file     string
			expected string
		}{
			{"main.go", "go"},
			{"app.ts", "typescript"},
			{"script.py", "python"},
			{"lib.rs", "rust"},
			{"Main.java", "java"},
			{"Program.cs", "csharp"},
			{"index.js", "javascript"},
			{"style.css", "css"},
			{"index.html", "html"},
			{"data.json", "json"},
			{"config.yaml", "yaml"},
			{"config.yml", "yaml"},
			{"unknown.txt", ""},
			{"", ""},
		}

		for _, tt := range tests {
			t.Run(tt.file, func(t *testing.T) {
				result := lsp.DetectLanguage(tt.file)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("DefaultConfigs", func(t *testing.T) {
		configs := lsp.DefaultConfigs()
		assert.NotEmpty(t, configs)

		// Check that common languages are configured
		languages := []string{"go", "typescript", "python", "rust"}
		for _, lang := range languages {
			config, exists := configs[lang]
			assert.True(t, exists, "Config for %s should exist", lang)
			assert.Equal(t, lang, config.Language)
			assert.NotEmpty(t, config.Command)
			assert.NotEmpty(t, config.FilePatterns)
			assert.NotEmpty(t, config.RootMarkers)
		}
	})
}
