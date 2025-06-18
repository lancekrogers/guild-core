// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/grpc"

	"github.com/guild-ventures/guild-core/pkg/config"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// Option configures a ChatModel
type Option func(*ChatModel)

// WithCampaign sets the campaign ID
func WithCampaign(id string) Option {
	return func(m *ChatModel) {
		m.campaignID = id
	}
}

// WithSession sets the session ID
func WithSession(id string) Option {
	return func(m *ChatModel) {
		m.sessionID = id
	}
}

// WithGuild sets the selected guild
func WithGuild(guildName string) Option {
	return func(m *ChatModel) {
		m.selectedGuild = guildName
	}
}

// New creates a new chat model with the given configuration
func New(ctx context.Context, cfg *config.GuildConfig, conn *grpc.ClientConn,
	guildClient pb.GuildClient, promptsClient promptspb.PromptServiceClient,
	registry registry.ComponentRegistry, opts ...Option) *ChatModel {

	// Create base model
	model := newChatModel(cfg, "", "", conn, guildClient, promptsClient, registry)

	// Apply options
	for _, opt := range opts {
		opt(&model)
	}

	return &model
}

// Run starts the chat TUI application
func Run(ctx context.Context, cfg *config.GuildConfig, conn *grpc.ClientConn,
	guildClient pb.GuildClient, promptsClient promptspb.PromptServiceClient,
	registry registry.ComponentRegistry, opts ...Option) error {

	// Create the chat model
	model := newChatModel(cfg, "", "", conn, guildClient, promptsClient, registry)

	// Apply options
	for _, opt := range opts {
		opt(&model)
	}

	// Start the TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
