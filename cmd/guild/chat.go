package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/guild-ventures/guild-core/internal/chat"
	"github.com/guild-ventures/guild-core/internal/daemon"
	"github.com/guild-ventures/guild-core/pkg/config"
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
- Manage prompts and view agent status`,
	RunE: runChat,
}

func runChat(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

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
	if !chatNoDaemon {
		if !daemon.IsReachable() {
			fmt.Println("🚀 Starting Guild server...")
			if err := daemon.EnsureRunning(ctx); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start Guild server").
					WithComponent("cli").
					WithOperation("chat.daemon_start")
			}
			// Give the server a moment to fully initialize
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Check if server is reachable
	if !daemon.IsReachable() {
		return gerror.New(gerror.ErrCodeConnection, "Guild server is not reachable", nil).
			WithComponent("cli").
			WithOperation("chat.run").
			WithDetails("help", "Try running 'guild serve' manually or check 'guild status'")
	}

	// Connect to gRPC server
	serverAddr := "localhost:9090" // Default gRPC server port
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to Guild server").
			WithComponent("cli").
			WithOperation("chat.run").
			WithDetails("server_address", serverAddr)
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

	// Create and run chat interface
	return chat.Run(ctx, guildConfig, conn, guildClient, promptClient, reg,
		chat.WithCampaign(chatCampaignID),
		chat.WithSession(chatSessionID))
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
