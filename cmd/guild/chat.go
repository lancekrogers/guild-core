package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/guild-ventures/guild-core/internal/chat"
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
)

func init() {
	chatCmd.Flags().StringVar(&chatCampaignID, "campaign", "", "Campaign ID to use for the chat session")
	chatCmd.Flags().StringVar(&chatSessionID, "session", "", "Session ID to use (defaults to new UUID)")
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
		return gerror.Wrap(err, gerror.ErrCodeConfig, "failed to load guild configuration").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Initialize project
	proj, err := project.LoadProject(".")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load project").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Generate session ID if not provided
	if chatSessionID == "" {
		chatSessionID = generateUUID()
	}

	// Connect to gRPC server
	serverAddr := fmt.Sprintf("localhost:%d", guildConfig.Server.Port)
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
	registryConfig := &registry.Config{
		GuildConfig: guildConfig,
		GRPCConn:    conn,
		ProjectPath: proj.Path,
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
	// Implementation from original file
	return config.LoadGuildConfig(".guild/guild.yaml")
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
