// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/guild-ventures/guild-core/cmd/guild/chat"
	"github.com/guild-ventures/guild-core/internal/daemon"
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
	chatNoDaemon   bool
)

func init() {
	chatCmd.Flags().StringVar(&chatCampaignID, "campaign", "", "Campaign ID to use for the chat session")
	chatCmd.Flags().StringVar(&chatSessionID, "session", "", "Session ID to use (defaults to new UUID)")
	chatCmd.Flags().BoolVar(&chatNoDaemon, "no-daemon", false, "Don't auto-start the Guild server")

	// Register completion functions
	chatCmd.RegisterFlagCompletionFunc("campaign", completeCampaignNames)
	chatCmd.RegisterFlagCompletionFunc("session", completeSessionIDs)
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
- Auto-starts the appropriate daemon if not running
- Use --no-daemon to prevent auto-starting the server

Examples:
  guild chat                         # Auto-detect campaign, start chat
  guild chat --campaign e-commerce   # Chat with specific campaign
  guild chat --no-daemon             # Connect without auto-starting daemon
  guild chat --session my-session    # Use specific session ID`,
	RunE: runChat,
}

func runChat(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Detect campaign for current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get current directory").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Use campaign detection logic
	campaignName, err := campaign.DetectCampaign(cwd, chatCampaignID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to detect campaign").
			WithComponent("cli").
			WithOperation("chat.run").
			WithDetails("help", "Make sure you're in a campaign directory or specify --campaign")
	}

	// Load configuration
	guildConfig, err := loadGuildConfig()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to load guild configuration").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Initialize project
	_, err = project.GetContext()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load project").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Generate session ID if not provided
	if chatSessionID == "" {
		chatSessionID = generateUUID()
	}

	// Auto-start daemon unless --no-daemon flag is set
	var daemonConfig *daemon.DaemonConfig
	if !chatNoDaemon {
		// Use the lifecycle manager for auto-start with session management
		lifecycleManager := daemon.DefaultLifecycleManager
		config, err := lifecycleManager.AutoStartDaemon(ctx, campaignName)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start campaign daemon").
				WithComponent("cli").
				WithOperation("chat.daemon_start")
		}
		daemonConfig = config
		// Give the server a moment to fully initialize
		time.Sleep(500 * time.Millisecond)
		
		// Start monitoring for idle timeout and crashes
		lifecycleManager.MonitorSessions(ctx)
	} else {
		// No auto-start, but we still need daemon config for connection
		config, err := daemon.GetDaemonConfig(campaignName, 0)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get daemon config").
				WithComponent("cli").
				WithOperation("chat.run")
		}
		daemonConfig = config
	}

	// Check if server is reachable
	isReachable := pkgDaemon.CanConnect(daemonConfig.SocketPath)

	if !isReachable {
		return gerror.New(gerror.ErrCodeConnection, "Guild server is not reachable", nil).
			WithComponent("cli").
			WithOperation("chat.run").
			WithDetails("help", "Try running 'guild serve --campaign "+campaignName+"' manually or check 'guild status'")
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

	// Create the modular chat application
	app, err := chat.NewApp(ctx, guildConfig, conn, guildClient, promptClient, reg, campaignName, chatSessionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create chat application").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Start the Bubble Tea program
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "chat interface error").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	return nil
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