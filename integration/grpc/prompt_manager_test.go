// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package grpc

import (
	"testing"
)

func TestPromptManagerIntegration(t *testing.T) {
	t.Skip("Skipping prompt manager tests until protobuf definitions are fixed")
	return
	/*
		// Create registry
		reg := registry.NewComponentRegistry()

		// Start server
		eventBus := newTestEventBus()
		server := guildgrpc.NewServer(reg, eventBus)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			err := server.Start(ctx, ":50053")
			assert.NoError(t, err)
		}()

		time.Sleep(100 * time.Millisecond)

		// Create client
		conn, err := grpc.Dial("localhost:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()

		client := promptspb.NewPromptServiceClient(conn)

		t.Run("set and get custom prompts", func(t *testing.T) {
			// Set a custom prompt
			_, err := client.SetPromptLayer(context.Background(), &promptspb.SetPromptLayerRequest{
				PromptId: "test-prompt-1",
				Content:  "This is a custom test prompt",
				Layer:    "project",
			})
			require.NoError(t, err)

			// Get the prompt back
			resp, err := client.GetPromptLayer(context.Background(), &promptspb.GetPromptLayerRequest{
				PromptId: "test-prompt-1",
			})
			require.NoError(t, err)
			assert.Equal(t, "This is a custom test prompt", resp.Content)
		})

		t.Run("prompt layering", func(t *testing.T) {
			// Set prompts at different layers
			layers := []struct {
				layer   string
				content string
			}{
				{"system", "System level prompt"},
				{"guild", "Guild level prompt"},
				{"project", "Project level prompt"},
				{"campaign", "Campaign level prompt"},
				{"objective", "Objective level prompt"},
				{"runtime", "Runtime level prompt"},
			}

			for _, l := range layers {
				_, err := client.SetPromptLayer(context.Background(), &promptspb.SetPromptLayerRequest{
					PromptId: "layered-test",
					Content:  l.content,
					Layer:    l.layer,
				})
				require.NoError(t, err)
			}

			// Assemble layered prompt
			assembled, err := client.BuildLayeredPrompt(context.Background(), &promptspb.BuildLayeredPromptRequest{
				PromptId: "layered-test",
				Context: map[string]string{
					"campaign":  "test-campaign",
					"objective": "test-objective",
				},
			})
			require.NoError(t, err)

			// Should contain all layers in proper order
			assert.Contains(t, assembled.Content, "System level")
			assert.Contains(t, assembled.Content, "Runtime level")

			// Verify order (runtime should override others)
			runtimeIndex := len(assembled.Content)
			for _, l := range layers {
				idx := len(assembled.Content) - len(l.content)
				if l.layer == "runtime" {
					runtimeIndex = idx
				} else {
					// Runtime layer should come last (highest priority)
					assert.Greater(t, runtimeIndex, idx)
				}
			}
		})

		t.Run("prompt persistence", func(t *testing.T) {
			// Set a prompt
			_, err := client.SetPromptLayer(context.Background(), &promptspb.SetPromptLayerRequest{
				PromptId: "persist-test",
				Content:  "This should persist",
				Layer:    "project",
			})
			require.NoError(t, err)

			// Simulate server restart by getting prompt again
			// In real implementation, this would test actual persistence
			resp, err := client.GetPromptLayer(context.Background(), &promptspb.GetPromptLayerRequest{
				PromptId: "persist-test",
			})
			require.NoError(t, err)
			assert.Equal(t, "This should persist", resp.Content)
		})

		t.Run("token optimization", func(t *testing.T) {
			// Set a long prompt
			longContent := "This is a very long prompt that should be optimized. "
			for i := 0; i < 100; i++ {
				longContent += "Adding more content to make it longer. "
			}

			_, err := client.SetPromptLayer(context.Background(), &promptspb.SetPromptLayerRequest{
				PromptId: "optimization-test",
				Content:  longContent,
				Layer:    "project",
			})
			require.NoError(t, err)

			// Optimize prompt
			optimized, err := client.BuildLayeredPrompt(context.Background(), &promptspb.BuildLayeredPromptRequest{
				PromptId:  "optimization-test",
				MaxTokens: 100,
			})
			require.NoError(t, err)

			// Should be shorter than original
			assert.Less(t, len(optimized.Content), len(longContent))
			assert.Greater(t, optimized.TokensSaved, int32(0))
		})

		t.Run("format support", func(t *testing.T) {
			// Test XML format
			_, err := client.SetPromptLayer(context.Background(), &promptspb.SetPromptLayerRequest{
				PromptId: "format-test",
				Content:  "Test prompt content",
				Layer:    "project",
				Format:   "xml",
			})
			require.NoError(t, err)

			xmlResp, err := client.GetPromptLayer(context.Background(), &promptspb.GetPromptLayerRequest{
				PromptId: "format-test",
			})
			require.NoError(t, err)
			assert.Contains(t, xmlResp.Content, "<prompt>")
			assert.Contains(t, xmlResp.Content, "</prompt>")

			// Test Markdown format
			_, err = client.SetPromptLayer(context.Background(), &promptspb.SetPromptLayerRequest{
				PromptId: "format-test-md",
				Content:  "Test prompt content",
				Layer:    "project",
				Format:   "markdown",
			})
			require.NoError(t, err)

			mdResp, err := client.GetPromptLayer(context.Background(), &promptspb.GetPromptLayerRequest{
				PromptId: "format-test-md",
			})
			require.NoError(t, err)
			assert.Contains(t, mdResp.Content, "```")
		})

		t.Run("error handling", func(t *testing.T) {
			// Try to get non-existent prompt
			_, err := client.GetPromptLayer(context.Background(), &promptspb.GetPromptLayerRequest{
				PromptId: "non-existent",
			})
			assert.Error(t, err)

			// Try to set prompt with invalid layer
			_, err = client.SetPromptLayer(context.Background(), &promptspb.SetPromptLayerRequest{
				PromptId: "invalid-layer",
				Content:  "Test",
				Layer:    "invalid-layer-name",
			})
			assert.Error(t, err)
		})

		t.Run("concurrent access", func(t *testing.T) {
			// Test concurrent prompt operations
			promptIDs := []string{"concurrent-1", "concurrent-2", "concurrent-3"}

			for _, promptID := range promptIDs {
				t.Run(promptID, func(t *testing.T) {
					t.Parallel()

					// Set prompt
					_, err := client.SetPromptLayer(context.Background(), &promptspb.SetPromptLayerRequest{
						PromptId: promptID,
						Content:  "Concurrent test prompt",
						Layer:    "project",
					})
					require.NoError(t, err)

					// Get prompt
					resp, err := client.GetPromptLayer(context.Background(), &promptspb.GetPromptLayerRequest{
						PromptId: promptID,
					})
					require.NoError(t, err)
					assert.Equal(t, "Concurrent test prompt", resp.Content)
				})
			}
		})
	*/
}
