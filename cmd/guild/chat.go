// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

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
- If no campaign workspace is detected, runs in Global Mode (no .campaign/ required)
- In a campaign workspace, the UI uses the daemon if available; otherwise it falls back to direct (in-process) mode
- Outside campaigns, Global Mode runs in direct (in-process) mode (no daemon required)

Examples:
  guild chat                         # Start chat (Global Mode outside campaigns)
  guild serve --foreground            # Optional: start daemon for campaign features
  guild chat                          # Connects if daemon is available, otherwise uses direct mode
  guild chat --campaign e-commerce   # Chat with specific campaign
  guild chat --session my-session    # Use specific session ID`,
	RunE: runChat,
}

func runChat(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Detect campaign root for current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get current directory").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	projectRoot, err := project.FindProjectRoot(cwd)
	inCampaign := err == nil
	if err != nil && !errors.Is(err, project.ErrNotInProject) {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect campaign root").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Use campaign detection logic
	campaignName := ""
	if inCampaign {
		campaignName, err = campaign.DetectCampaign(cwd, chatCampaignID)
		if err != nil {
			// Campaign features are optional; don't block chat startup on campaign detection issues.
			fmt.Fprintf(os.Stderr, "Warning: failed to detect campaign (%v). Starting chat without campaign features.\n", err)
			campaignName = ""
		}
	}

	// Load guild configuration
	var guildConfig *config.GuildConfig
	if inCampaign {
		guildConfig, err = loadGuildConfig(ctx, projectRoot)
		if err != nil {
			// Fall back to default template so chat can still open
			guildConfig = config.DefaultGuildTemplate()
		}
	} else {
		guildConfig = config.DefaultGuildTemplate()
	}

	// Run guild selector to let user choose which guild to work with
	selectedGuild := guildConfig.Name
	if inCampaign {
		if picked, err := chatui.RunGuildSelector(ctx); err == nil && picked != "" {
			selectedGuild = picked
		}
	}

	// Generate session ID if not provided
	if chatSessionID == "" {
		chatSessionID = generateUUID()
	}

	// Get user ID (for Sprint 2 preferences)
	userID := os.Getenv("USER")
	if userID == "" {
		userID = "default"
	}

	// Registry is optional for chat; direct mode can operate without it.
	var reg registry.ComponentRegistry

	// Create and run chat interface with daemon connection management
	app := chatui.NewApp(ctx, guildConfig, reg)
	app.SetSelectedGuild(selectedGuild)
	app.SetCampaignID(campaignName)
	app.SetSessionID(chatSessionID)
	app.SetUserID(userID) // Sprint 2: Set user ID for preferences
	return app.Run()
}

// loadGuildConfig loads the guild configuration from the project
func loadGuildConfig(ctx context.Context, projectRoot string) (*config.GuildConfig, error) {
	if projectRoot == "" {
		projectRoot = "."
	}
	return config.LoadGuildConfig(ctx, projectRoot)
}

// generateUUID generates a new UUID for session ID
func generateUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fall back to a timestamp-based identifier
		return fmt.Sprintf("session-%d", os.Getpid())
	}

	// UUID v4 variant
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
