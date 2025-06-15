package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCompletionFunctions(t *testing.T) {
	// Test the full completion functions
	fullTests := []struct {
		name         string
		function     func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)
		toComplete   string
		expectError  bool
		checkResults func(t *testing.T, suggestions []string, directive cobra.ShellCompDirective)
	}{
		{
			name:        "complete session IDs with empty input",
			function:    completeSessionIDs,
			toComplete:  "",
			expectError: false,
			checkResults: func(t *testing.T, suggestions []string, directive cobra.ShellCompDirective) {
				assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
				assert.Equal(t, 2, len(suggestions))
				assert.Contains(t, suggestions, "new\tCreate new session")
				assert.Contains(t, suggestions, "last\tResume last session")
			},
		},
		{
			name:        "complete campaign IDs without project context",
			function:    completeCampaignIDs,
			toComplete:  "camp",
			expectError: false,
			checkResults: func(t *testing.T, suggestions []string, directive cobra.ShellCompDirective) {
				// Without project context, should get error directive
				assert.Equal(t, cobra.ShellCompDirectiveError, directive)
			},
		},
		{
			name:        "complete agent IDs without registry",
			function:    completeAgentIDs,
			toComplete:  "back",
			expectError: false,
			checkResults: func(t *testing.T, suggestions []string, directive cobra.ShellCompDirective) {
				// Should fall back to default agents
				assert.NotEqual(t, cobra.ShellCompDirectiveError, directive)
				found := false
				for _, s := range suggestions {
					if s == "backend\tBackend development and API design" {
						found = true
						break
					}
				}
				assert.True(t, found, "Should find backend agent in defaults")
			},
		},
	}

	for _, tt := range fullTests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions, directive := tt.function(nil, nil, tt.toComplete)
			
			if tt.expectError {
				assert.Equal(t, cobra.ShellCompDirectiveError, directive)
			} else if tt.checkResults != nil {
				tt.checkResults(t, suggestions, directive)
			}
		})
	}

	// Test helper functions with different signatures
	t.Run("complete default agents helper", func(t *testing.T) {
		suggestions, directive := completeDefaultAgents("man")
		assert.NotEqual(t, cobra.ShellCompDirectiveError, directive)
		assert.Greater(t, len(suggestions), 0)
		// Should find "manager" agent
		found := false
		for _, s := range suggestions {
			if s == "manager\tProject management and task decomposition" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find manager agent")
	})

	t.Run("complete default campaign names helper", func(t *testing.T) {
		suggestions, directive := completeDefaultCampaignNames("e-com")
		assert.NotEqual(t, cobra.ShellCompDirectiveError, directive)
		assert.Greater(t, len(suggestions), 0)
		// Should find "e-commerce" campaign
		found := false
		for _, s := range suggestions {
			if s == "e-commerce\tE-commerce platform development" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find e-commerce campaign")
	})
}

func TestCompletionCommandExists(t *testing.T) {
	// Verify the completion command is registered
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "completion" {
			found = true
			break
		}
	}
	assert.True(t, found, "completion command should be registered")
}

func TestCompletionCommandValidArgs(t *testing.T) {
	// Find the completion command
	var completionCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "completion" {
			completionCmd = cmd
			break
		}
	}
	
	assert.NotNil(t, completionCmd)
	assert.Equal(t, []string{"bash", "zsh", "fish", "powershell"}, completionCmd.ValidArgs)
}