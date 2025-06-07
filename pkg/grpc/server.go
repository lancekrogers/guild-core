package grpc

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/guild-ventures/guild-core/pkg/campaign"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/orchestrator"
	"github.com/guild-ventures/guild-core/pkg/prompts"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// Server implements the Guild gRPC service
type Server struct {
	pb.UnimplementedGuildServer
	
	campaignMgr   campaign.Manager
	commissionMgr *commission.Manager
	kanbanMgr     *kanban.Manager
	agentReg      registry.AgentRegistry
	orchestrator  *orchestrator.Orchestrator
	promptManager prompts.LayeredManager // Added for prompt management
	
	frameBuilder *FrameBuilder
	watchers     map[string]*watcher
	watchersMu   sync.RWMutex
	
	grpcServer    *grpc.Server
	listener      net.Listener
	promptServer  *PromptsServer // Added for prompt service
	chatService   *ChatService   // Added for real-time chat
}

// watcher represents an active campaign watcher
type watcher struct {
	campaignID string
	stream     pb.Guild_WatchCampaignServer
	done       chan struct{}
	options    watchOptions
}

// watchOptions contains options for watching campaigns
type watchOptions struct {
	includeAgents   bool
	includeKanban   bool
	includeProgress bool
}

// NewServer creates a new gRPC server following the registry pattern
func NewServer(
	registry registry.ComponentRegistry,
	eventBus EventBus,
) *Server {
	// Get required components from registry
	campaignMgr := getCampaignManager(registry)
	commissionMgr := getCommissionManager(registry)
	kanbanMgr := getKanbanManager(registry)
	agentReg := registry.Agents()
	orchestrator := getOrchestrator(registry)
	_ = registry.Prompts() // TODO: Fix interface mismatch
	
	// TODO: Fix interface mismatch between registry and prompts
	promptServer := NewPromptsServer(nil) // Temporarily pass nil
	chatService := NewChatService(registry, eventBus)
	
	return &Server{
		campaignMgr:   campaignMgr,
		commissionMgr: commissionMgr,
		kanbanMgr:     kanbanMgr,
		agentReg:      agentReg,
		orchestrator:  orchestrator,
		promptManager: nil, // TODO: Fix interface mismatch
		frameBuilder:  NewFrameBuilder(campaignMgr, commissionMgr, kanbanMgr, agentReg),
		watchers:      make(map[string]*watcher),
		promptServer:  promptServer,
		chatService:   chatService,
	}
}

// Start starts the gRPC server
func (s *Server) Start(ctx context.Context, address string) error {
	var err error
	s.listener, err = net.Listen("tcp", address)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to listen").
			WithComponent("grpc").
			WithOperation("Start").
			WithDetails("address", address).
			FromContext(ctx)
	}

	s.grpcServer = grpc.NewServer()
	pb.RegisterGuildServer(s.grpcServer, s)
	// TODO: Register chat service when protobuf is fixed
	// chatpb.RegisterChatServiceServer(s.grpcServer, s.chatService)
	promptspb.RegisterPromptServiceServer(s.grpcServer, s.promptServer)

	// Start server in goroutine
	go func() {
		if err := s.grpcServer.Serve(s.listener); err != nil {
			fmt.Printf("gRPC server error: %v\n", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	s.grpcServer.GracefulStop()
	
	return nil
}

// Stop stops the gRPC server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

// WatchCampaign streams live dashboard updates for a campaign
func (s *Server) WatchCampaign(req *pb.WatchRequest, stream pb.Guild_WatchCampaignServer) error {
	// Get campaign ID (use active campaign if not specified)
	campaignID := req.CampaignId
	if campaignID == "" {
		// Get active campaign
		campaigns, err := s.campaignMgr.List(stream.Context())
		if err != nil {
			return status.Errorf(codes.Internal, "failed to list campaigns: %v", err)
		}
		
		// Find active campaign
		for _, c := range campaigns {
			if c.Status == campaign.CampaignStatusActive {
				campaignID = c.ID
				break
			}
		}
		
		if campaignID == "" {
			return status.Error(codes.NotFound, "no active campaign found")
		}
	}

	// Verify campaign exists
	_, err := s.campaignMgr.Get(stream.Context(), campaignID)
	if err != nil {
		return status.Errorf(codes.NotFound, "campaign not found: %v", err)
	}

	// Create watcher
	w := &watcher{
		campaignID: campaignID,
		stream:     stream,
		done:       make(chan struct{}),
		options: watchOptions{
			includeAgents:   req.IncludeAgents,
			includeKanban:   req.IncludeKanban,
			includeProgress: req.IncludeProgress,
		},
	}

	// Register watcher
	watcherID := fmt.Sprintf("%s-%d", campaignID, time.Now().UnixNano())
	s.watchersMu.Lock()
	s.watchers[watcherID] = w
	s.watchersMu.Unlock()

	// Clean up on exit
	defer func() {
		s.watchersMu.Lock()
		delete(s.watchers, watcherID)
		s.watchersMu.Unlock()
		close(w.done)
	}()

	// Subscribe to campaign events
	eventChan := make(chan struct{}, 1)
	handler := func(ctx context.Context, event campaign.CampaignEvent) error {
		if event.CampaignID == campaignID {
			select {
			case eventChan <- struct{}{}:
			default:
			}
		}
		return nil
	}
	
	// Subscribe to all campaign events
	s.campaignMgr.Subscribe("*", handler)
	defer s.campaignMgr.Unsubscribe("*", handler)

	// Start streaming updates
	ticker := time.NewTicker(16 * time.Millisecond) // 60 FPS cap
	defer ticker.Stop()

	// Send initial frame
	if err := s.sendFrame(w); err != nil {
		return err
	}

	// Main streaming loop
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-w.done:
			return nil
		case <-eventChan:
			// Event occurred, send update
			if err := s.sendFrame(w); err != nil {
				return err
			}
		case <-ticker.C:
			// Regular update tick (if needed for animations, etc.)
			// For now, we only send on events to minimize bandwidth
		}
	}
}

// sendFrame sends a frame update to the watcher
func (s *Server) sendFrame(w *watcher) error {
	// Get campaign
	campaign, err := s.campaignMgr.Get(w.stream.Context(), w.campaignID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	// Build frame
	frame, metadata := s.frameBuilder.BuildFrame(w.stream.Context(), campaign, w.options)

	// Create board update
	update := &pb.BoardUpdate{
		Frame:     string(frame),
		Timestamp: time.Now().Unix(),
		Metadata: &pb.BoardMetadata{
			Width:          int32(metadata.Width),
			Height:         int32(metadata.Height),
			ActiveAgents:   int32(metadata.ActiveAgents),
			TotalTasks:     int32(metadata.TotalTasks),
			CompletedTasks: int32(metadata.CompletedTasks),
			Fps:            metadata.FPS,
		},
	}

	// Send update
	return w.stream.Send(update)
}

// GetCampaign retrieves a campaign by ID
func (s *Server) GetCampaign(ctx context.Context, req *pb.GetCampaignRequest) (*pb.Campaign, error) {
	campaign, err := s.campaignMgr.Get(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "campaign not found: %v", err)
	}

	return campaignToProto(campaign), nil
}

// ListCampaigns returns a list of campaigns
func (s *Server) ListCampaigns(ctx context.Context, req *pb.ListCampaignsRequest) (*pb.ListCampaignsResponse, error) {
	campaigns, err := s.campaignMgr.List(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list campaigns: %v", err)
	}

	// Apply status filter if provided
	if req.StatusFilter != "" {
		filtered := make([]*campaign.Campaign, 0)
		for _, c := range campaigns {
			if string(c.Status) == req.StatusFilter {
				filtered = append(filtered, c)
			}
		}
		campaigns = filtered
	}

	// Convert to proto
	protoCampaigns := make([]*pb.Campaign, len(campaigns))
	for i, c := range campaigns {
		protoCampaigns[i] = campaignToProto(c)
	}

	return &pb.ListCampaignsResponse{
		Campaigns:  protoCampaigns,
		TotalCount: int32(len(protoCampaigns)),
	}, nil
}

// CreateCampaign creates a new campaign
func (s *Server) CreateCampaign(ctx context.Context, req *pb.CreateCampaignRequest) (*pb.Campaign, error) {
	// Create campaign
	c := campaign.NewCampaign(req.Name, req.Description)
	c.Tags = req.Tags
	for k, v := range req.Metadata {
		c.Metadata[k] = v
	}

	// Save campaign
	if err := s.campaignMgr.Create(ctx, c); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create campaign: %v", err)
	}

	return campaignToProto(c), nil
}

// UpdateCampaign updates an existing campaign
func (s *Server) UpdateCampaign(ctx context.Context, req *pb.UpdateCampaignRequest) (*pb.Campaign, error) {
	// Get existing campaign
	c, err := s.campaignMgr.Get(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "campaign not found: %v", err)
	}

	// Update fields
	if req.Name != "" {
		c.Name = req.Name
	}
	if req.Description != "" {
		c.Description = req.Description
	}
	if len(req.Tags) > 0 {
		c.Tags = req.Tags
	}
	for k, v := range req.Metadata {
		c.Metadata[k] = v
	}

	// Save campaign
	if err := s.campaignMgr.Update(ctx, c); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update campaign: %v", err)
	}

	return campaignToProto(c), nil
}

// DeleteCampaign deletes a campaign
func (s *Server) DeleteCampaign(ctx context.Context, req *pb.DeleteCampaignRequest) (*pb.DeleteCampaignResponse, error) {
	if err := s.campaignMgr.Delete(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete campaign: %v", err)
	}

	return &pb.DeleteCampaignResponse{
		Success: true,
		Message: "Campaign deleted successfully",
	}, nil
}

// StartPlanningCampaign transitions a campaign from dream to planning
func (s *Server) StartPlanningCampaign(ctx context.Context, req *pb.CampaignActionRequest) (*pb.Campaign, error) {
	if err := s.campaignMgr.StartPlanning(ctx, req.CampaignId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to start planning: %v", err)
	}

	c, err := s.campaignMgr.Get(ctx, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	return campaignToProto(c), nil
}

// MarkCampaignReady transitions a campaign from planning to ready
func (s *Server) MarkCampaignReady(ctx context.Context, req *pb.CampaignActionRequest) (*pb.Campaign, error) {
	if err := s.campaignMgr.MarkReady(ctx, req.CampaignId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mark campaign ready: %v", err)
	}

	c, err := s.campaignMgr.Get(ctx, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	return campaignToProto(c), nil
}

// StartCampaign starts a campaign
func (s *Server) StartCampaign(ctx context.Context, req *pb.CampaignActionRequest) (*pb.Campaign, error) {
	if err := s.campaignMgr.Start(ctx, req.CampaignId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to start campaign: %v", err)
	}

	c, err := s.campaignMgr.Get(ctx, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	return campaignToProto(c), nil
}

// PauseCampaign pauses a campaign
func (s *Server) PauseCampaign(ctx context.Context, req *pb.CampaignActionRequest) (*pb.Campaign, error) {
	if err := s.campaignMgr.Pause(ctx, req.CampaignId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to pause campaign: %v", err)
	}

	c, err := s.campaignMgr.Get(ctx, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	return campaignToProto(c), nil
}

// ResumeCampaign resumes a campaign
func (s *Server) ResumeCampaign(ctx context.Context, req *pb.CampaignActionRequest) (*pb.Campaign, error) {
	if err := s.campaignMgr.Resume(ctx, req.CampaignId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resume campaign: %v", err)
	}

	c, err := s.campaignMgr.Get(ctx, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	return campaignToProto(c), nil
}

// CompleteCampaign completes a campaign
func (s *Server) CompleteCampaign(ctx context.Context, req *pb.CampaignActionRequest) (*pb.Campaign, error) {
	if err := s.campaignMgr.Complete(ctx, req.CampaignId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to complete campaign: %v", err)
	}

	c, err := s.campaignMgr.Get(ctx, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	return campaignToProto(c), nil
}

// CancelCampaign cancels a campaign
func (s *Server) CancelCampaign(ctx context.Context, req *pb.CampaignActionRequest) (*pb.Campaign, error) {
	if err := s.campaignMgr.Cancel(ctx, req.CampaignId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to cancel campaign: %v", err)
	}

	c, err := s.campaignMgr.Get(ctx, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	return campaignToProto(c), nil
}

// AddCommissionToCampaign adds a commission to a campaign
func (s *Server) AddCommissionToCampaign(ctx context.Context, req *pb.AddCommissionRequest) (*pb.Campaign, error) {
	// Use the existing objective method since commissions and objectives refer to the same concept
	if err := s.campaignMgr.AddObjective(ctx, req.CampaignId, req.CommissionId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add commission: %v", err)
	}

	c, err := s.campaignMgr.Get(ctx, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	return campaignToProto(c), nil
}

// RemoveCommissionFromCampaign removes a commission from a campaign
func (s *Server) RemoveCommissionFromCampaign(ctx context.Context, req *pb.RemoveCommissionRequest) (*pb.Campaign, error) {
	// Use the existing objective method since commissions and objectives refer to the same concept
	if err := s.campaignMgr.RemoveObjective(ctx, req.CampaignId, req.CommissionId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove commission: %v", err)
	}

	c, err := s.campaignMgr.Get(ctx, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	return campaignToProto(c), nil
}

// SendMessageToAgent sends a message to a specific agent
func (s *Server) SendMessageToAgent(ctx context.Context, req *pb.AgentMessageRequest) (*pb.AgentMessageResponse, error) {
	// Validate request
	if req.AgentId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	if req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	// Get agent from registry
	agentRegistry := s.agentReg
	if agentRegistry == nil {
		return nil, status.Error(codes.Internal, "agent registry not initialized")
	}

	// Find the agent configuration
	registeredAgents := agentRegistry.GetRegisteredAgents()
	var agentConfig *registry.GuildAgentConfig
	for _, agent := range registeredAgents {
		if agent.ID == req.AgentId {
			agentConfig = &agent
			break
		}
	}

	if agentConfig == nil {
		return nil, status.Errorf(codes.NotFound, "agent '%s' not found", req.AgentId)
	}

	// For now, create a simple mock response
	// TODO: Integrate with actual agent factory when orchestrator is ready
	mockResponse := fmt.Sprintf("I'm %s, and I received your message: %s. (Note: This is a mock response - actual agent integration pending)", agentConfig.Name, req.Message)

	// Add context from request
	if req.SessionId != "" {
		ctx = context.WithValue(ctx, "session_id", req.SessionId)
	}
	if req.CampaignId != "" {
		ctx = context.WithValue(ctx, "campaign_id", req.CampaignId)
	}

	// Use the mock response for now
	response := mockResponse

	return &pb.AgentMessageResponse{
		AgentId:   req.AgentId,
		Response:  response,
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"agent_type": agentConfig.Type,
			"model":      agentConfig.Model,
		},
		Status: &pb.AgentStatus{
			State:        pb.AgentStatus_IDLE,
			LastActivity: time.Now().Unix(),
		},
	}, nil
}

// StreamAgentConversation handles bidirectional streaming for agent conversations
func (s *Server) StreamAgentConversation(stream pb.Guild_StreamAgentConversationServer) error {
	// Create a context for this stream
	_ = stream.Context()
	
	// Track active agents and their states
	activeAgents := make(map[string]*pb.AgentStatus)
	
	for {
		// Receive message from client
		req, err := stream.Recv()
		if err != nil {
			// Client disconnected
			return nil
		}

		switch request := req.Request.(type) {
		case *pb.AgentStreamRequest_Message:
			// Handle agent message
			msg := request.Message
			
			// Update agent status to thinking
			if _, exists := activeAgents[msg.AgentId]; !exists {
				activeAgents[msg.AgentId] = &pb.AgentStatus{
					State:        pb.AgentStatus_THINKING,
					LastActivity: time.Now().Unix(),
				}
			}
			
			// Send thinking event
			if err := stream.Send(&pb.AgentStreamResponse{
				Response: &pb.AgentStreamResponse_Event{
					Event: &pb.StreamEvent{
						Type:        pb.StreamEvent_AGENT_THINKING,
						Description: fmt.Sprintf("%s is thinking...", msg.AgentId),
						Data: map[string]string{
							"agent_id": msg.AgentId,
						},
					},
				},
			}); err != nil {
				return err
			}
			
			// Mock agent creation for now
			// TODO: Integrate with actual agent factory
			var mockAgent interface{} = nil
			err := gerror.New(gerror.ErrCodeInternal, "agent factory not yet integrated", nil).
				WithComponent("grpc").
				WithOperation("StreamAgentConversation").
				FromContext(stream.Context())
			if err != nil {
				// Send error event
				if err := stream.Send(&pb.AgentStreamResponse{
					Response: &pb.AgentStreamResponse_Event{
						Event: &pb.StreamEvent{
							Type:        pb.StreamEvent_ERROR,
							Description: fmt.Sprintf("Failed to get agent: %v", err),
						},
					},
				}); err != nil {
					return err
				}
				continue
			}
			
			// Update status to working
			activeAgents[msg.AgentId].State = pb.AgentStatus_WORKING
			if err := stream.Send(&pb.AgentStreamResponse{
				Response: &pb.AgentStreamResponse_Status{
					Status: activeAgents[msg.AgentId],
				},
			}); err != nil {
				return err
			}
			
			// Mock response for now
			response := fmt.Sprintf("Mock response from agent %s: I received '%s'", msg.AgentId, msg.Message)
			_ = mockAgent // Suppress unused variable warning
			
			// Simulate error for demonstration (remove this in real implementation)
			if false {
				// Send error
				if err := stream.Send(&pb.AgentStreamResponse{
					Response: &pb.AgentStreamResponse_Event{
						Event: &pb.StreamEvent{
							Type:        pb.StreamEvent_ERROR,
							Description: fmt.Sprintf("Agent execution failed: %v", err),
						},
					},
				}); err != nil {
					return err
				}
				continue
			}
			
			// Stream response in fragments (simulate streaming)
			// In real implementation, agent.Execute would return a channel
			fragments := splitIntoFragments(response, 100) // 100 chars per fragment
			for i, fragment := range fragments {
				if err := stream.Send(&pb.AgentStreamResponse{
					Response: &pb.AgentStreamResponse_Fragment{
						Fragment: &pb.AgentMessageFragment{
							AgentId:    msg.AgentId,
							Content:    fragment,
							IsComplete: i == len(fragments)-1,
							Timestamp:  time.Now().Unix(),
						},
					},
				}); err != nil {
					return err
				}
				
				// Small delay to simulate streaming
				time.Sleep(50 * time.Millisecond)
			}
			
			// Update status to idle
			activeAgents[msg.AgentId].State = pb.AgentStatus_IDLE
			if err := stream.Send(&pb.AgentStreamResponse{
				Response: &pb.AgentStreamResponse_Status{
					Status: activeAgents[msg.AgentId],
				},
			}); err != nil {
				return err
			}
			
		case *pb.AgentStreamRequest_Control:
			// Handle stream control commands
			control := request.Control
			switch control.Command {
			case pb.StreamControl_STOP:
				return nil
			case pb.StreamControl_PAUSE:
				// Implementation for pause
			case pb.StreamControl_RESUME:
				// Implementation for resume
			}
		}
	}
}

// ListAvailableAgents returns all available agents
func (s *Server) ListAvailableAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	registeredAgents := s.agentReg.GetRegisteredAgents()
	
	agents := make([]*pb.AgentInfo, 0, len(registeredAgents))
	for _, agent := range registeredAgents {
		agentInfo := &pb.AgentInfo{
			Id:           agent.ID,
			Name:         agent.Name,
			Type:         agent.Type,
			Capabilities: agent.Capabilities,
			Metadata: map[string]string{
				"provider": agent.Provider,
				"model":    agent.Model,
			},
		}
		
		// Add status if requested
		if req.IncludeStatus {
			agentInfo.Status = &pb.AgentStatus{
				State:        pb.AgentStatus_IDLE,
				LastActivity: time.Now().Unix(),
			}
		}
		
		agents = append(agents, agentInfo)
	}
	
	return &pb.ListAgentsResponse{
		Agents:     agents,
		TotalCount: int32(len(agents)),
	}, nil
}

// GetAgentStatus returns the current status of an agent
func (s *Server) GetAgentStatus(ctx context.Context, req *pb.GetAgentStatusRequest) (*pb.AgentStatus, error) {
	if req.AgentId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	
	// In real implementation, this would check actual agent state
	// For now, return a mock status
	return &pb.AgentStatus{
		State:        pb.AgentStatus_IDLE,
		LastActivity: time.Now().Unix(),
		Metadata: map[string]string{
			"health": "healthy",
		},
	}, nil
}

// Helper function to split response into fragments
func splitIntoFragments(text string, chunkSize int) []string {
	if chunkSize <= 0 {
		return []string{text}
	}
	
	var fragments []string
	for i := 0; i < len(text); i += chunkSize {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		fragments = append(fragments, text[i:end])
	}
	
	return fragments
}

// campaignToProto converts a campaign to proto format
func campaignToProto(c *campaign.Campaign) *pb.Campaign {
	metadata := make(map[string]string)
	for k, v := range c.Metadata {
		if str, ok := v.(string); ok {
			metadata[k] = str
		}
	}

	proto := &pb.Campaign{
		Id:                     c.ID,
		Name:                   c.Name,
		Description:            c.Description,
		Status:                 string(c.Status),
		CommissionIds:          c.Objectives, // Map Objectives to CommissionIds for backwards compatibility
		Tags:                   c.Tags,
		Progress:               c.Progress,
		TotalCommissions:       int32(c.TotalObjectives), // Map TotalObjectives to TotalCommissions
		CompletedCommissions:   int32(c.CompletedObjectives), // Map CompletedObjectives to CompletedCommissions
		CreatedAt:              c.CreatedAt.Unix(),
		UpdatedAt:              c.UpdatedAt.Unix(),
		Metadata:               metadata,
	}

	if c.StartedAt != nil {
		proto.StartedAt = c.StartedAt.Unix()
	}
	if c.CompletedAt != nil {
		proto.CompletedAt = c.CompletedAt.Unix()
	}

	return proto
}

// EventBus interface for broadcasting events within the Guild system
// (Moved to chat_service.go to avoid duplication)

// Helper functions to extract components from registry
// These handle nil cases gracefully and provide meaningful errors for debugging

func getCampaignManager(registry registry.ComponentRegistry) campaign.Manager {
	// In the current implementation, we would get this from a campaign registry
	// For now, return nil - this will be filled in when campaign manager is registry-integrated
	return nil
}

func getCommissionManager(registry registry.ComponentRegistry) *commission.Manager {
	// Similar to campaign manager, this would come from registry
	return nil
}

func getKanbanManager(registry registry.ComponentRegistry) *kanban.Manager {
	// This should get the kanban manager from registry when available
	return nil
}

func getOrchestrator(registry registry.ComponentRegistry) *orchestrator.Orchestrator {
	// Orchestrator would come from registry.Orchestrator() when implemented
	return nil
}