// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lancekrogers/guild-core/pkg/config"
	"github.com/lancekrogers/guild-core/pkg/providers"
	providerif "github.com/lancekrogers/guild-core/pkg/providers/interfaces"

	pb "github.com/lancekrogers/guild-core/pkg/grpc/pb/guild/v1"
)

type localAgentState struct {
	cfg          config.AgentConfig
	history      []providerif.ChatMessage
	status       pb.AgentStatus_State
	currentTask  string
	lastActivity int64
}

// localGuildClient is a direct-mode (no daemon) implementation of pb.GuildClient.
// It supports the subset of methods used by the chat UI for agent messaging.
type localGuildClient struct {
	mu      sync.Mutex
	guild   *config.GuildConfig
	factory *providers.FactoryV2

	providers map[providers.ProviderType]providerif.AIProvider
	agents    map[string]*localAgentState
}

func newLocalGuildClient(guild *config.GuildConfig) (*localGuildClient, error) {
	if guild == nil {
		return nil, status.Error(codes.InvalidArgument, "guild config is nil")
	}

	c := &localGuildClient{
		guild:     guild,
		factory:   providers.NewFactoryV2(),
		providers: make(map[providers.ProviderType]providerif.AIProvider),
		agents:    make(map[string]*localAgentState),
	}

	for _, agent := range guild.Agents {
		agentCfg := agent
		c.agents[agentCfg.ID] = &localAgentState{
			cfg:          agentCfg,
			history:      make([]providerif.ChatMessage, 0, 32),
			status:       pb.AgentStatus_IDLE,
			currentTask:  "",
			lastActivity: time.Now().Unix(),
		}
	}

	return c, nil
}

func (c *localGuildClient) WatchCampaign(ctx context.Context, in *pb.WatchRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.BoardUpdate], error) {
	return nil, status.Error(codes.Unimplemented, "campaign streaming requires daemon mode")
}

func (c *localGuildClient) GetCampaign(ctx context.Context, in *pb.GetCampaignRequest, opts ...grpc.CallOption) (*pb.Campaign, error) {
	return nil, status.Error(codes.Unimplemented, "campaign operations require daemon mode")
}

func (c *localGuildClient) ListCampaigns(ctx context.Context, in *pb.ListCampaignsRequest, opts ...grpc.CallOption) (*pb.ListCampaignsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "campaign operations require daemon mode")
}

func (c *localGuildClient) CreateCampaign(ctx context.Context, in *pb.CreateCampaignRequest, opts ...grpc.CallOption) (*pb.Campaign, error) {
	return nil, status.Error(codes.Unimplemented, "campaign operations require daemon mode")
}

func (c *localGuildClient) UpdateCampaign(ctx context.Context, in *pb.UpdateCampaignRequest, opts ...grpc.CallOption) (*pb.Campaign, error) {
	return nil, status.Error(codes.Unimplemented, "campaign operations require daemon mode")
}

func (c *localGuildClient) DeleteCampaign(ctx context.Context, in *pb.DeleteCampaignRequest, opts ...grpc.CallOption) (*pb.DeleteCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "campaign operations require daemon mode")
}

func (c *localGuildClient) StartPlanningCampaign(ctx context.Context, in *pb.CampaignActionRequest, opts ...grpc.CallOption) (*pb.Campaign, error) {
	return nil, status.Error(codes.Unimplemented, "campaign operations require daemon mode")
}

func (c *localGuildClient) MarkCampaignReady(ctx context.Context, in *pb.CampaignActionRequest, opts ...grpc.CallOption) (*pb.Campaign, error) {
	return nil, status.Error(codes.Unimplemented, "campaign operations require daemon mode")
}

func (c *localGuildClient) StartCampaign(ctx context.Context, in *pb.CampaignActionRequest, opts ...grpc.CallOption) (*pb.Campaign, error) {
	return nil, status.Error(codes.Unimplemented, "campaign operations require daemon mode")
}

func (c *localGuildClient) PauseCampaign(ctx context.Context, in *pb.CampaignActionRequest, opts ...grpc.CallOption) (*pb.Campaign, error) {
	return nil, status.Error(codes.Unimplemented, "campaign operations require daemon mode")
}

func (c *localGuildClient) ResumeCampaign(ctx context.Context, in *pb.CampaignActionRequest, opts ...grpc.CallOption) (*pb.Campaign, error) {
	return nil, status.Error(codes.Unimplemented, "campaign operations require daemon mode")
}

func (c *localGuildClient) CompleteCampaign(ctx context.Context, in *pb.CampaignActionRequest, opts ...grpc.CallOption) (*pb.Campaign, error) {
	return nil, status.Error(codes.Unimplemented, "campaign operations require daemon mode")
}

func (c *localGuildClient) CancelCampaign(ctx context.Context, in *pb.CampaignActionRequest, opts ...grpc.CallOption) (*pb.Campaign, error) {
	return nil, status.Error(codes.Unimplemented, "campaign operations require daemon mode")
}

func (c *localGuildClient) AddCommissionToCampaign(ctx context.Context, in *pb.AddCommissionRequest, opts ...grpc.CallOption) (*pb.Campaign, error) {
	return nil, status.Error(codes.Unimplemented, "commission operations require daemon mode")
}

func (c *localGuildClient) RemoveCommissionFromCampaign(ctx context.Context, in *pb.RemoveCommissionRequest, opts ...grpc.CallOption) (*pb.Campaign, error) {
	return nil, status.Error(codes.Unimplemented, "commission operations require daemon mode")
}

func (c *localGuildClient) StreamAgentConversation(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[pb.AgentStreamRequest, pb.AgentStreamResponse], error) {
	return nil, status.Error(codes.Unimplemented, "streaming requires daemon mode")
}

func (c *localGuildClient) ListAvailableAgents(ctx context.Context, in *pb.ListAgentsRequest, opts ...grpc.CallOption) (*pb.ListAgentsResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	agentIDs := make([]string, 0, len(c.agents))
	for id := range c.agents {
		agentIDs = append(agentIDs, id)
	}
	sort.Strings(agentIDs)

	agents := make([]*pb.AgentInfo, 0, len(agentIDs))
	for _, id := range agentIDs {
		state := c.agents[id]
		var statusMsg *pb.AgentStatus
		if in.GetIncludeStatus() {
			statusMsg = &pb.AgentStatus{
				State:        state.status,
				CurrentTask:  state.currentTask,
				LastActivity: state.lastActivity,
				Metadata:     map[string]string{"mode": "direct"},
			}
		}
		agents = append(agents, &pb.AgentInfo{
			Id:           state.cfg.ID,
			Name:         state.cfg.Name,
			Type:         state.cfg.Type,
			Capabilities: append([]string(nil), state.cfg.Capabilities...),
			Status:       statusMsg,
			Metadata: map[string]string{
				"mode":     "direct",
				"provider": state.cfg.Provider,
				"model":    state.cfg.Model,
			},
		})
	}

	return &pb.ListAgentsResponse{
		Agents:     agents,
		TotalCount: int32(len(agents)),
	}, nil
}

func (c *localGuildClient) GetAgentStatus(ctx context.Context, in *pb.GetAgentStatusRequest, opts ...grpc.CallOption) (*pb.AgentStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	state, ok := c.agents[in.GetAgentId()]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "agent %q not found", in.GetAgentId())
	}

	return &pb.AgentStatus{
		State:        state.status,
		CurrentTask:  state.currentTask,
		LastActivity: state.lastActivity,
		Metadata:     map[string]string{"mode": "direct"},
	}, nil
}

func (c *localGuildClient) SendMessageToAgent(ctx context.Context, in *pb.AgentMessageRequest, opts ...grpc.CallOption) (*pb.AgentMessageResponse, error) {
	agentID := in.GetAgentId()
	message := in.GetMessage()
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	if message == "" {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	c.mu.Lock()
	state, ok := c.agents[agentID]
	if !ok {
		c.mu.Unlock()
		return nil, status.Errorf(codes.NotFound, "agent %q not found", agentID)
	}

	// Update local state (append user message immediately).
	state.status = pb.AgentStatus_THINKING
	state.lastActivity = time.Now().Unix()
	state.history = append(state.history, providerif.ChatMessage{Role: "user", Content: message})
	historySnapshot := append([]providerif.ChatMessage(nil), state.history...)
	agentCfg := state.cfg
	c.mu.Unlock()

	providerType := providers.ConvertToProviderType(agentCfg.Provider)
	provider, err := c.getProvider(providerType)
	if err != nil {
		c.setAgentError(agentID, err)
		return nil, err
	}

	model := agentCfg.Model
	if model == "" {
		model = providers.GetDefaultModel(agentCfg.Provider)
	}
	if model == "" {
		c.setAgentError(agentID, status.Error(codes.InvalidArgument, "agent model is required"))
		return nil, status.Error(codes.InvalidArgument, "agent model is required")
	}

	// Limit history to reduce context bloat (last N messages).
	const maxHistory = 24
	if len(historySnapshot) > maxHistory {
		historySnapshot = historySnapshot[len(historySnapshot)-maxHistory:]
	}

	messages := make([]providerif.ChatMessage, 0, len(historySnapshot)+1)
	if systemPrompt := buildAgentSystemPrompt(agentCfg); systemPrompt != "" {
		messages = append(messages, providerif.ChatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, historySnapshot...)

	req := providerif.ChatRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   agentCfg.MaxTokens,
		Temperature: agentCfg.Temperature,
		Stream:      false,
	}

	resp, err := provider.ChatCompletion(ctx, req)
	if err != nil {
		c.setAgentError(agentID, err)
		return nil, err
	}

	reply := ""
	if resp != nil && len(resp.Choices) > 0 {
		reply = resp.Choices[0].Message.Content
	}
	if reply == "" {
		err := status.Error(codes.Internal, "provider returned empty response")
		c.setAgentError(agentID, err)
		return nil, err
	}

	c.mu.Lock()
	// Append assistant message and mark agent idle again.
	if state, ok := c.agents[agentID]; ok {
		state.history = append(state.history, providerif.ChatMessage{Role: "assistant", Content: reply})
		state.status = pb.AgentStatus_IDLE
		state.currentTask = ""
		state.lastActivity = time.Now().Unix()
	}
	c.mu.Unlock()

	return &pb.AgentMessageResponse{
		AgentId:   agentID,
		Response:  reply,
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"mode":     "direct",
			"provider": agentCfg.Provider,
			"model":    model,
		},
		Status: &pb.AgentStatus{
			State:        pb.AgentStatus_IDLE,
			LastActivity: time.Now().Unix(),
			Metadata:     map[string]string{"mode": "direct"},
		},
	}, nil
}

func (c *localGuildClient) getProvider(providerType providers.ProviderType) (providerif.AIProvider, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if provider, ok := c.providers[providerType]; ok {
		return provider, nil
	}

	// Claude Code is gRPC/daemon-oriented in this repo; direct-mode uses AIProvider implementations.
	if providerType == providers.ProviderClaudeCode {
		return nil, status.Error(codes.FailedPrecondition, "claude_code provider requires daemon mode; use anthropic/openai/ollama/etc for direct mode")
	}

	provider, err := c.factory.CreateAIProvider(providerType, "")
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to create provider %s: %v", providerType, err)
	}

	c.providers[providerType] = provider
	return provider, nil
}

func (c *localGuildClient) setAgentError(agentID string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if state, ok := c.agents[agentID]; ok {
		state.status = pb.AgentStatus_ERROR
		state.currentTask = ""
		state.lastActivity = time.Now().Unix()
		if state.history != nil && err != nil {
			state.history = append(state.history, providerif.ChatMessage{
				Role:    "assistant",
				Content: fmt.Sprintf("Error: %v", err),
			})
		}
	}
}

func buildAgentSystemPrompt(agentCfg config.AgentConfig) string {
	if agentCfg.SystemPrompt != "" {
		return agentCfg.SystemPrompt
	}
	if agentCfg.Name != "" && agentCfg.Description != "" {
		return fmt.Sprintf("You are %s. %s", agentCfg.Name, agentCfg.Description)
	}
	if agentCfg.Description != "" {
		return agentCfg.Description
	}
	if agentCfg.Name != "" {
		return fmt.Sprintf("You are %s.", agentCfg.Name)
	}
	return ""
}

