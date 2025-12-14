// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	chatui "github.com/guild-framework/guild-core/internal/ui/chat"
	chatui "github.com/guild-framework/guild-core/internal/ui/chat"
	"github.com/guild-framework/guild-core/pkg/campaign"
	"github.com/guild-framework/guild-core/pkg/config"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/project"
	"github.com/guild-framework/guild-core/pkg/registry"
)

var (
	chatCampaignID string
	chatSessionID  string
)

func init() {
	chatCmd.Flags().StringVar(&chatCampaignID, "campaign", "", "Campaign ID to use for the chat session")
	chatCmd.Flags().StringVar(&chatSessionID, "session", "", "Session ID to use (defaults to new UUID)")

	// Register completion functions
	// Temporarily disabled to avoid early config loading
	// chatCmd.RegisterFlagCompletionFunc("campaign", completeCampaignNames)
	// chatCmd.RegisterFlagCompletionFunc("session", completeSessionIDs)
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session with Guild agents",
	Long: `Start an interactive chat session with Guild agents.

This opens a terminal-based chat interface where you can:
- Send messages to all agents or specific agents using @mentions
- View agent responses with rich markdown formatting
- Execute tools with agent assistance
- Manage prompts and view agent status

Campaign Support:
- Use --campaign to specify which campaign to connect to
- Without --campaign, detects campaign from current directory
- Requires Guild daemon to be running (use 'guild serve')

Examples:
  guild serve                        # Start daemon in background (separate terminal)
  guild chat                         # Connect to running daemon
  guild chat --campaign e-commerce   # Chat with specific campaign
  guild chat --session my-session    # Use specific session ID`,
	RunE: runChat,
}

func checkGuildInitialized(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check if campaign is initialized by looking for .campaign directory
	campaignDir := filepath.Join(".", ".campaign")
	if _, err := os.Stat(campaignDir); os.IsNotExist(err) {
		// Campaign not initialized
		// Offer to initialize guild
		fmt.Println("🏰 Guild not initialized in this directory.")
		fmt.Println()
		fmt.Println("The Guild Framework needs to be initialized before you can chat with Elena and the specialists.")
		fmt.Println()
		fmt.Println("Would you like to initialize Guild now? This will:")
		fmt.Println("  ✨ Set up your Guild with Elena as Guild Master")
		fmt.Println("  🤖 Detect available AI providers automatically")
		fmt.Println("  👥 Create Marcus (backend) and Vera (frontend) specialists")
		fmt.Println("  🚀 Get you chatting in under 30 seconds")
		fmt.Println()
		fmt.Print("Initialize Guild? [Y/n]: ")

		// Read user input
		var response string
		fmt.Scanln(&response)

		// Default to yes if empty or starts with y/Y
		if response == "" || (len(response) > 0 && strings.ToLower(response)[0] == 'y') {
			fmt.Println()
			fmt.Println("🎯 Starting Guild initialization...")
			fmt.Println()

			// Run guild init with sensible defaults
			= exec.Command(os.Args[0], "init", "--force")
			initCmd.Stdout = os.Stdout
			initCmd.Stderr = os.Stderr
			initCmd.Stdin = os.Stdin

			if err := initCmd.Run(); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run guild init").
					WithComponent("cli").
					WithOperation("checkGuildInitialized")
			}

			fmt.Println()
			fmt.Println("✅ Guild initialized! Continuing to chat...")
			fmt.Println()

			// Small pause for user to see the message
			time.Sleep(1 * time.Second)

			// Return nil to continue to runChat
			return nil
		} else {
			fmt.Println()
			fmt.Println("To initialize Guild manually, run:")
			fmt.Println("  guild init")
			fmt.Println()
			// Return the error which will prevent runChat from executing
			return gerror.New(gerror.ErrCodeCancelled, "guild initialization required", nil).
				WithComponent("cli").
				WithOperation("checkGuildInitialized")
		}
	}
	// Campaign is initialized, continue
	return nil
}

func runChat(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// First check if guild is initialized
	if err := checkGuildInitialized(cmd, args); err != nil {
		// If checkGuildInitialized returns an error, it means user cancelled
		// The function already printed the message, so just return nil
		return nil
	}

	// Detect campaign for current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get current directory").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Load configuration (now we know it exists)
	guildConfig, err := loadGuildConfig(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to load guild configuration").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Use campaign detection logic
	campaignName, err := campaign.DetectCampaign(cwd, chatCampaignID)
	if err != nil {
		// If no campaign detected after initialization, use a default
		if gerror.GetCode(err) == gerror.ErrCodeNotFound {
			campaignName = "guild-demo" // Use default campaign name
		} else {
			return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to detect campaign").
				WithComponent("cli").
				WithOperation("chat.run").
				WithDetails("help", "Make sure you're in a campaign directory or specify --campaign")
		}
	}

	// Initialize project
	_, err = project.GetContext()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load project").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Run guild selector to let user choose which guild to work with
	selectedGuild, err := chatui.RunGuildSelector(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to select guild").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Store selected guild for future use
	fmt.Printf("Selected guild: %s\n", selectedGuild)

	// Generate session ID if not provided
	if chatSessionID == "" {
		chatSessionID = generateUUID()
	}

	// Get user ID (for Sprint 2 preferences)
	userID := os.Getenv("USER")
	if userID == "" {
		userID = "default"
	}

	// Ensure daemon is running (auto-start with retries for a smooth UX)
	if !daemon.IsReachable(ctx) {
		fmt.Println("🚀 Starting Guild server...")
		if err := daemon.EnsureRunning(ctx); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start Guild server").
				WithComponent("cli").
				WithOperation("chat.daemon_start")
		}
		for i := 0; i < 10; i++ { // wait up to ~5s
			if daemon.IsReachable(ctx) {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	if !daemon.IsReachable(ctx) {
		return gerror.New(gerror.ErrCodeConnection, "Guild server is not reachable", nil).
			WithComponent("cli").
			WithOperation("chat.preflight").
			WithDetails("help", "Start the daemon first: 'guild serve --foreground' then run 'guild chat'")
	}

	// Initialize registry
	reg := registry.NewComponentRegistry()
	registryConfig := registry.Config{
		// Basic registry configuration - will be enhanced later
	}

	if err := reg.Initialize(context.Background(), registryConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Create and run chat interface with daemon connection management
	app := chatui.NewApp(ctx, guildConfig, reg)
	app.SetSelectedGuild(selectedGuild)
	app.SetCampaignID(campaignName)
	app.SetSessionID(chatSessionID)
	app.SetUserID(userID) // Sprint 2: Set user ID for preferences
	return app.Run()
}

// loadGuildConfig loads the guild configuration from the project
func loadGuildConfig(ctx context.Context) (*config.GuildConfig, error) {
	// Load from current directory (LoadGuildConfig will add .guild/guild.yaml)
	return config.LoadGuildConfig(ctx, ".")
}

// generateUUID generates a new UUID for session ID
func generateUUID() string {
	// Simple UUID v4 generation (you might want to use a proper UUID library)
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		make([]byte, 4),
		make([]byte, 2),
		make([]byte, 2),
		make([]byte, 2),
		make([]byte, 6))
}
