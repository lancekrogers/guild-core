// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/guild-ventures/guild-core/internal/daemon"
	chatui "github.com/guild-ventures/guild-core/internal/ui/chat"
	"github.com/guild-ventures/guild-core/pkg/campaign"
	"github.com/guild-ventures/guild-core/pkg/config"
	pkgDaemon "github.com/guild-ventures/guild-core/pkg/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/registry"
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
	// Check if guild is initialized first
	_, err := loadGuildConfig()
	if err != nil {
		// Check if it's a "not found" error indicating guild is not initialized
		if gerror.GetCode(err) == gerror.ErrCodeNotFound {
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
				
				// Run guild init in quick mode
				initCmd := exec.Command(os.Args[0], "init", "--quick")
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
		// Other errors pass through
		return err
	}
	// Guild is initialized, continue
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
	guildConfig, err := loadGuildConfig()
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

	// Get daemon config for connection (no auto-start)
	daemonConfig, err := daemon.GetDaemonConfig(campaignName, 0)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get daemon config").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Check if server is reachable
	isReachable := pkgDaemon.CanConnect(daemonConfig.SocketPath)

	if !isReachable {
		fmt.Println("🚨 Guild daemon is not running")
		fmt.Println()
		fmt.Println("The Guild daemon must be running to start a chat session.")
		fmt.Println()
		fmt.Println("To start the daemon, run:")
		fmt.Printf("  guild serve --campaign %s\n", campaignName)
		fmt.Println()
		fmt.Println("Then start chat again:")
		fmt.Println("  guild chat")
		fmt.Println()
		fmt.Println("💡 Pro tip: Run 'guild serve' in a separate terminal to keep it running in the background")
		return nil // Don't return error, just exit cleanly with instructions
	}

	// Connect to gRPC server
	conn, err := grpc.Dial("unix://"+daemonConfig.SocketPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to Guild server").
			WithComponent("cli").
			WithOperation("chat.run").
			WithDetails("daemon_config", daemonConfig.GetServerAddress())
	}
	defer conn.Close()

	// Create gRPC clients
	guildClient := pb.NewGuildClient(conn)
	promptClient := promptspb.NewPromptServiceClient(conn)

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

	// Create and run chat interface using v2 implementation
	app := chatui.NewApp(ctx, guildConfig, conn, guildClient, promptClient, reg)
	app.SetSelectedGuild(selectedGuild)
	return app.Run()
}

// loadGuildConfig loads the guild configuration from the project
func loadGuildConfig() (*config.GuildConfig, error) {
	// Load from current directory (LoadGuildConfig will add .guild/guild.yaml)
	return config.LoadGuildConfig(".")
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
