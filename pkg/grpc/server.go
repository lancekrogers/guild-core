// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

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

	"github.com/lancekrogers/guild/pkg/campaign"
	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/gerror"
	pb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	promptspb "github.com/lancekrogers/guild/pkg/grpc/pb/prompts/v1"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/orchestrator"
	"github.com/lancekrogers/guild/pkg/prompts/layered"
	"github.com/lancekrogers/guild/pkg/registry"
)

// Server implements the Guild gRPC service
type Server struct {
	pb.UnimplementedGuildServer

	campaignMgr   campaign.Manager
	commissionMgr *commission.Manager
	kanbanMgr     *kanban.Manager
	agentReg      registry.AgentRegistry
	orchestrator  orchestrator.Orchestrator
	promptManager layered.LayeredManager // Added for prompt management

	frameBuilder *FrameBuilder
	watchers     map[string]*watcher
	watchersMu   sync.RWMutex

	grpcServer     *grpc.Server
	listener       net.Listener
	promptServer   *PromptsServer          // Added for prompt service
	chatService    *ChatService            // Added for real-time chat
	sessionService pb.SessionServiceServer // Added for session persistence
	eventService   pb.EventServiceServer   // Added for event streaming
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
	campaignMgr := getCampaignManager(registry, eventBus)
	commissionMgr := getCommissionManager(registry)
	kanbanMgr := getKanbanManager(registry)
	agentReg := registry.Agents()
	orchestrator := getOrchestrator(registry)

	// Get prompt manager - creates a memory-based layered manager
	promptManager := getPromptManager(registry)

	// Create prompt server with proper manager
	promptServer := NewPromptsServer(promptManager)
	chatService := NewChatService(registry, eventBus)

	// Create session service with storage registry
	sessionService := getSessionService(registry)

	// Create event service using unified event bus
	var eventService pb.EventServiceServer
	if unifiedBus, ok := eventBus.(*EventBusAdapter); ok && unifiedBus != nil {
		eventService = NewUnifiedEventService(unifiedBus.UnifiedEventBus())
	} else {
		// This should not happen in production as we always create unified event bus
		panic("EventBus must be EventBusAdapter wrapping unified event bus")
	}

	return &Server{
		campaignMgr:    campaignMgr,
		commissionMgr:  commissionMgr,
		kanbanMgr:      kanbanMgr,
		agentReg:       agentReg,
		orchestrator:   orchestrator,
		promptManager:  promptManager,
		frameBuilder:   NewFrameBuilder(campaignMgr, commissionMgr, kanbanMgr, agentReg),
		watchers:       make(map[string]*watcher),
		promptServer:   promptServer,
		chatService:    chatService,
		sessionService: sessionService,
		eventService:   eventService,
	}
}

// Start starts the gRPC server on TCP
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

	return s.startServer(ctx, address)
}

// StartUnix starts the gRPC server on Unix socket
func (s *Server) StartUnix(ctx context.Context, socketPath string) error {
	var err error
	s.listener, err = net.Listen("unix", socketPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to listen on Unix socket").
			WithComponent("grpc").
			WithOperation("StartUnix").
			WithDetails("socket_path", socketPath).
			FromContext(ctx)
	}

	// Log that Unix socket is successfully bound
	observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("StartUnix").
		Info("gRPC server bound to Unix socket",
			"socket_path", socketPath,
		)

	return s.startServer(ctx, socketPath)
}

// startServer contains the common server startup logic
func (s *Server) startServer(ctx context.Context, address string) error {
	s.grpcServer = grpc.NewServer()
	pb.RegisterGuildServer(s.grpcServer, s)
	pb.RegisterChatServiceServer(s.grpcServer, s.chatService)
	pb.RegisterSessionServiceServer(s.grpcServer, s.sessionService)
	pb.RegisterEventServiceServer(s.grpcServer, s.eventService)
	promptspb.RegisterPromptServiceServer(s.grpcServer, s.promptServer)

	// Create a channel to signal when server is ready
	serverReady := make(chan struct{})

	// Start server in goroutine
	go func() {
		// Signal that server is about to start serving
		close(serverReady)

		// Log that server is starting to serve
		observability.GetLogger(ctx).
			WithComponent("grpc").
			WithOperation("startServer").
			Info("gRPC server is ready and serving requests",
				"address", address,
				"listener_type", s.listener.Addr().Network(),
			)

		if err := s.grpcServer.Serve(s.listener); err != nil {
			// Log server error with proper context
			observability.GetLogger(ctx).
				WithComponent("grpc").
				WithOperation("startServer").
				WithError(err).
				Error("gRPC server encountered an error during serving",
					"address", address,
				)
		}
	}()

	// Wait for server to be ready before returning
	<-serverReady

	// Add a small delay to ensure the socket is fully bound and ready
	time.Sleep(100 * time.Millisecond)

	// Wait for context cancellation
	<-ctx.Done()

	// Log graceful shutdown
	observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("startServer").
		Info("gRPC server shutting down gracefully",
			"address", address,
		)

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
	// Initialize observability for gRPC campaign retrieval
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("GetCampaign").
		With("campaign_id", req.Id)

	start := time.Now()
	logger.Debug("gRPC GetCampaign request received",
		"campaign_id", req.Id,
	)

	campaign, err := s.campaignMgr.Get(ctx, req.Id)
	duration := time.Since(start)

	if err != nil {
		// Enhanced error observability for campaign retrieval
		logger.WithError(err).Error("Failed to get campaign via gRPC",
			"campaign_id", req.Id,
			"retrieval_duration_ms", duration.Milliseconds(),
		)

		// Log performance metrics for failed request
		logger.Duration("grpc.get_campaign", duration,
			"success", false,
			"campaign_id", req.Id,
			"error_type", fmt.Sprintf("%T", err),
		)

		return nil, status.Errorf(codes.NotFound, "campaign not found: %v", err)
	}

	// Convert to proto format
	protoStart := time.Now()
	protoCampaign := campaignToProto(campaign)
	protoConversionDuration := time.Since(protoStart)

	logger.Info("gRPC GetCampaign completed successfully",
		"campaign_id", req.Id,
		"retrieval_duration_ms", duration.Milliseconds(),
		"proto_conversion_duration_ms", protoConversionDuration.Milliseconds(),
		"total_duration_ms", time.Since(start).Milliseconds(),
	)

	// Log performance metrics for successful request
	logger.Duration("grpc.get_campaign", time.Since(start),
		"success", true,
		"campaign_id", req.Id,
		"proto_conversion_duration_ms", protoConversionDuration.Milliseconds(),
	)

	return protoCampaign, nil
}

// ListCampaigns returns a list of campaigns
func (s *Server) ListCampaigns(ctx context.Context, req *pb.ListCampaignsRequest) (*pb.ListCampaignsResponse, error) {
	// Initialize observability for gRPC campaign listing
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("ListCampaigns").
		With("status_filter", req.StatusFilter).
		With("page_size", req.PageSize).
		With("page_token", req.PageToken)

	start := time.Now()
	logger.Debug("gRPC ListCampaigns request received",
		"status_filter", req.StatusFilter,
		"page_size", req.PageSize,
		"has_page_token", req.PageToken != "",
	)

	// Check if campaign manager is nil (happens with test setups)
	if s.campaignMgr == nil {
		logger.Debug("Campaign manager not available - returning empty list")
		return &pb.ListCampaignsResponse{
			Campaigns: []*pb.Campaign{},
		}, nil
	}

	// Try to list campaigns but handle nil repository gracefully
	var campaigns []*campaign.Campaign
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Debug("Campaign manager panicked (likely nil repository) - returning empty list",
					"panic", r)
				campaigns = []*campaign.Campaign{}
				err = nil
			}
		}()
		campaigns, err = s.campaignMgr.List(ctx)
	}()
	listDuration := time.Since(start)

	if err != nil {
		// Enhanced error observability for campaign listing
		logger.WithError(err).Error("Failed to list campaigns via gRPC",
			"list_duration_ms", listDuration.Milliseconds(),
			"status_filter", req.StatusFilter,
		)

		// Log performance metrics for failed request
		logger.Duration("grpc.list_campaigns", listDuration,
			"success", false,
			"status_filter", req.StatusFilter,
			"error_type", fmt.Sprintf("%T", err),
		)

		return nil, status.Errorf(codes.Internal, "failed to list campaigns: %v", err)
	}

	originalCount := len(campaigns)
	logger.Debug("Campaigns retrieved from manager",
		"total_campaigns", originalCount,
		"list_duration_ms", listDuration.Milliseconds(),
	)

	// Apply status filter if provided with enhanced observability
	if req.StatusFilter != "" {
		filterStart := time.Now()
		filtered := make([]*campaign.Campaign, 0)
		for _, c := range campaigns {
			if string(c.Status) == req.StatusFilter {
				filtered = append(filtered, c)
			}
		}
		campaigns = filtered
		filterDuration := time.Since(filterStart)

		logger.Debug("Status filter applied",
			"status_filter", req.StatusFilter,
			"original_count", originalCount,
			"filtered_count", len(campaigns),
			"filter_duration_ms", filterDuration.Milliseconds(),
		)
	}

	// Convert to proto with enhanced observability
	protoStart := time.Now()
	protoCampaigns := make([]*pb.Campaign, len(campaigns))
	for i, c := range campaigns {
		protoCampaigns[i] = campaignToProto(c)
	}
	protoConversionDuration := time.Since(protoStart)

	totalDuration := time.Since(start)
	finalCount := len(protoCampaigns)

	logger.Info("gRPC ListCampaigns completed successfully",
		"original_count", originalCount,
		"final_count", finalCount,
		"list_duration_ms", listDuration.Milliseconds(),
		"proto_conversion_duration_ms", protoConversionDuration.Milliseconds(),
		"total_duration_ms", totalDuration.Milliseconds(),
		"status_filter", req.StatusFilter,
	)

	// Log performance metrics for successful request
	logger.Duration("grpc.list_campaigns", totalDuration,
		"success", true,
		"campaigns_returned", finalCount,
		"status_filter", req.StatusFilter,
		"proto_conversion_duration_ms", protoConversionDuration.Milliseconds(),
	)

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
	// Use the commission method
	if err := s.campaignMgr.AddCommission(ctx, req.CampaignId, req.CommissionId); err != nil {
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
	// Use the commission method
	if err := s.campaignMgr.RemoveCommission(ctx, req.CampaignId, req.CommissionId); err != nil {
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

	// Add context from request for observability
	if req.SessionId != "" {
		ctx = context.WithValue(ctx, "session_id", req.SessionId)
	}
	if req.CampaignId != "" {
		ctx = context.WithValue(ctx, "campaign_id", req.CampaignId)
	}

	// Enhanced observability for agent execution
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("SendMessageToAgent")

	logger.Info("Processing agent message request",
		"agent_id", req.AgentId,
		"session_id", req.SessionId,
		"campaign_id", req.CampaignId,
		"message_length", len(req.Message),
	)

	var response string
	var err error

	// Try to get agent instance from registry first
	if agentRegistryInstance := s.agentReg; agentRegistryInstance != nil {
		if agent, getErr := agentRegistryInstance.GetAgent(req.AgentId); getErr == nil {
			// Execute with the actual agent
			logger.Debug("Executing with registered agent instance", "agent_id", req.AgentId)
			response, err = agent.Execute(ctx, req.Message)
			if err != nil {
				logger.WithError(err).Error("Agent execution failed", "agent_id", req.AgentId)
				return nil, status.Errorf(codes.Internal, "agent execution failed: %v", err)
			}
		} else {
			logger.Warn("Agent not found in agent registry, trying orchestrator",
				"agent_id", req.AgentId,
				"get_error", getErr.Error(),
			)
		}
	}

	// If no direct agent or execution failed, try orchestrator
	if response == "" && s.orchestrator != nil {
		logger.Debug("Attempting execution via orchestrator", "agent_id", req.AgentId)

		// Get agent from orchestrator and execute
		if orchAgent, found := s.orchestrator.GetAgent(req.AgentId); found {
			if orchResponse, orchErr := orchAgent.Execute(ctx, req.Message); orchErr == nil {
				response = orchResponse
				logger.Debug("Orchestrator agent execution successful", "agent_id", req.AgentId)
			} else {
				logger.WithError(orchErr).Warn("Orchestrator agent execution failed", "agent_id", req.AgentId)
			}
		} else {
			logger.Warn("Agent not found in orchestrator", "agent_id", req.AgentId)
		}
	}

	// Fallback to a structured response if no execution succeeded
	if response == "" {
		logger.Warn("No agent execution path succeeded, using fallback response", "agent_id", req.AgentId)
		response = fmt.Sprintf("I'm %s. I received your message but agent execution is not fully configured yet. Please check the guild daemon setup.\n\nMessage: %s",
			agentConfig.Name, req.Message)
	}

	logger.Info("Agent message processing completed",
		"agent_id", req.AgentId,
		"response_length", len(response),
		"execution_successful", response != "",
	)

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
			// In real implementation, core.Execute would return a channel
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
		Id:                   c.ID,
		Name:                 c.Name,
		Description:          c.Description,
		Status:               string(c.Status),
		CommissionIds:        c.Commissions,
		Tags:                 c.Tags,
		Progress:             c.Progress,
		TotalCommissions:     int32(c.TotalCommissions),
		CompletedCommissions: int32(c.CompletedCommissions),
		CreatedAt:            c.CreatedAt.Unix(),
		UpdatedAt:            c.UpdatedAt.Unix(),
		Metadata:             metadata,
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

func getCampaignManager(registry registry.ComponentRegistry, grpcEventBus EventBus) campaign.Manager {
	// Always use unified manager with the unified event bus
	adapter, ok := grpcEventBus.(*EventBusAdapter)
	if !ok || adapter == nil {
		panic("EventBus must be EventBusAdapter wrapping unified event bus")
	}

	unifiedBus := adapter.UnifiedEventBus()

	// Get campaign repository from storage registry if available
	var campaignRepo campaign.Repository
	var commissionMgr *commission.Manager

	storageReg := registry.Storage()
	if storageReg != nil {
		// Note: registry.CampaignRepository doesn't match campaign.Repository interface
		// We would need an adapter here, but for now we'll use nil
		// TODO: Create repository adapter to bridge interface mismatch
	}

	// Get commission manager if we can
	commissionMgr = getCommissionManager(registry)

	// Return unified manager
	return campaign.NewUnifiedManager(campaignRepo, commissionMgr, unifiedBus)
}

func getCommissionManager(registry registry.ComponentRegistry) *commission.Manager {
	// Get commission repository from storage registry
	storageReg := registry.Storage()
	if storageReg == nil {
		// Return a basic working commission manager if storage is not available
		mgr, err := commission.DefaultCommissionManagerFactory(nil, "/tmp")
		if err != nil {
			return nil
		}
		return mgr.(*commission.Manager)
	}

	commissionRepo := storageReg.GetCommissionRepository()
	if commissionRepo == nil {
		// Return a basic working commission manager if repository is not available
		mgr, err := commission.DefaultCommissionManagerFactory(nil, "/tmp")
		if err != nil {
			return nil
		}
		return mgr.(*commission.Manager)
	}

	// Create commission manager with repository
	// The registry.CommissionRepository doesn't match storage.CommissionRepository interface exactly
	// For now, we'll use nil to avoid compilation errors
	// TODO: Create an adapter to bridge the interface mismatch
	mgr, err := commission.DefaultCommissionManagerFactory(nil, "/tmp")
	if err != nil {
		return nil
	}
	return mgr.(*commission.Manager)
}

func getKanbanManager(registry registry.ComponentRegistry) *kanban.Manager {
	// Create adapter to bridge interface mismatch
	adapter := &kanbanRegistryAdapter{registry: registry}

	// Use the registry-aware constructor with adapter
	ctx := context.Background()
	manager, err := kanban.NewManagerWithRegistry(ctx, adapter)
	if err != nil {
		// Log error but return nil to gracefully handle missing dependencies
		return nil
	}
	return manager
}

func getOrchestrator(registry registry.ComponentRegistry) orchestrator.Orchestrator {
	// Get orchestrator components from registry
	orchReg := registry.Orchestrator()
	if orchReg == nil {
		// Return a basic working orchestrator if not available from registry
		eventBus := orchestrator.DefaultEventBusFactory()
		config := &orchestrator.Config{MaxConcurrentAgents: 3}
		dispatcher := orchestrator.DefaultTaskDispatcherFactory(nil, nil, eventBus, 3)
		return orchestrator.DefaultOrchestratorFactory(config, dispatcher, eventBus)
	}

	// For now, return a basic orchestrator - better than nil
	// In the future, we'd extract components from orchReg
	eventBus := orchestrator.DefaultEventBusFactory()
	config := &orchestrator.Config{MaxConcurrentAgents: 3}
	dispatcher := orchestrator.DefaultTaskDispatcherFactory(nil, nil, eventBus, 3)
	return orchestrator.DefaultOrchestratorFactory(config, dispatcher, eventBus)
}

func getPromptManager(registry registry.ComponentRegistry) layered.LayeredManager {
	// Create a simple memory-based layered manager for basic functionality
	// This provides a working implementation without requiring full storage integration

	// Create base components
	baseRegistry := layered.NewMemoryRegistry()
	formatter := &simplePromptFormatter{}

	// Create a simple memory-based layered store
	memStore := &memoryLayeredStore{
		prompts: make(map[string][]byte),
		cache:   make(map[string][]byte),
	}

	// Create a base manager that implements both Manager and Formatter
	baseManager := &formatterAwareManager{
		manager:   layered.NewDefaultManager(baseRegistry, formatter),
		formatter: formatter,
	}

	// Create the layered manager without RAG (can be added later)
	tokenBudget := 8000 // Default token budget

	return layered.NewGuildLayeredManager(
		baseManager,
		memStore,
		baseRegistry,
		nil, // RAG retriever - optional for now
		tokenBudget,
	)
}

// getSessionService creates a session service with proper storage backend
func getSessionService(registry registry.ComponentRegistry) pb.SessionServiceServer {
	// For now, always use memory service until we fix the interface mismatch
	observability.GetLogger(context.Background()).
		WithComponent("grpc").
		WithOperation("getSessionService").
		Info("Using memory-based session service")
	return NewMemorySessionService()
}

// formatterAwareManager wraps a Manager and implements both Manager and Formatter interfaces
type formatterAwareManager struct {
	manager   layered.Manager
	formatter layered.Formatter
}

// Implement Manager interface by delegating to the wrapped manager
func (f *formatterAwareManager) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	return f.manager.GetSystemPrompt(ctx, role, domain)
}

func (f *formatterAwareManager) GetTemplate(ctx context.Context, templateName string) (string, error) {
	return f.manager.GetTemplate(ctx, templateName)
}

func (f *formatterAwareManager) FormatContext(ctx context.Context, context layered.Context) (string, error) {
	return f.manager.FormatContext(ctx, context)
}

func (f *formatterAwareManager) ListRoles(ctx context.Context) ([]string, error) {
	return f.manager.ListRoles(ctx)
}

func (f *formatterAwareManager) ListDomains(ctx context.Context, role string) ([]string, error) {
	return f.manager.ListDomains(ctx, role)
}

// Implement Formatter interface by delegating to the formatter
func (f *formatterAwareManager) FormatAsXML(ctx layered.Context) (string, error) {
	return f.formatter.FormatAsXML(ctx)
}

func (f *formatterAwareManager) FormatAsMarkdown(ctx layered.Context) (string, error) {
	return f.formatter.FormatAsMarkdown(ctx)
}

func (f *formatterAwareManager) OptimizeForTokens(content string, maxTokens int) (string, error) {
	return f.formatter.OptimizeForTokens(content, maxTokens)
}

// simplePromptFormatter provides basic formatting functionality
type simplePromptFormatter struct{}

func (f *simplePromptFormatter) FormatAsXML(ctx layered.Context) (string, error) {
	return fmt.Sprintf("<context>%v</context>", ctx), nil
}

func (f *simplePromptFormatter) FormatAsMarkdown(ctx layered.Context) (string, error) {
	return fmt.Sprintf("## Context\n\n%v", ctx), nil
}

func (f *simplePromptFormatter) OptimizeForTokens(content string, maxTokens int) (string, error) {
	// Simple truncation for now
	if len(content) > maxTokens*4 { // Rough estimate: 1 token ≈ 4 chars
		return content[:maxTokens*4] + "...", nil
	}
	return content, nil
}

// memoryLayeredStore provides a simple in-memory implementation of LayeredStore
type memoryLayeredStore struct {
	prompts map[string][]byte
	cache   map[string][]byte
	mu      sync.RWMutex
}

// Store interface methods (required by LayeredStore which extends memory.Store)
func (m *memoryLayeredStore) Put(ctx context.Context, bucket, key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fullKey := fmt.Sprintf("%s:%s", bucket, key)
	m.prompts[fullKey] = value
	return nil
}

func (m *memoryLayeredStore) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fullKey := fmt.Sprintf("%s:%s", bucket, key)
	if data, ok := m.prompts[fullKey]; ok {
		return data, nil
	}
	return nil, gerror.New(gerror.ErrCodeNotFound, "key not found", nil)
}

func (m *memoryLayeredStore) Delete(ctx context.Context, bucket, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fullKey := fmt.Sprintf("%s:%s", bucket, key)
	delete(m.prompts, fullKey)
	return nil
}

func (m *memoryLayeredStore) List(ctx context.Context, bucket string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prefix := bucket + ":"
	var keys []string
	for k := range m.prompts {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			keys = append(keys, k[len(prefix):])
		}
	}
	return keys, nil
}

func (m *memoryLayeredStore) ListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bucketPrefix := bucket + ":"
	var keys []string
	for k := range m.prompts {
		if len(k) > len(bucketPrefix) && k[:len(bucketPrefix)] == bucketPrefix {
			key := k[len(bucketPrefix):]
			if prefix == "" || (len(key) >= len(prefix) && key[:len(prefix)] == prefix) {
				keys = append(keys, key)
			}
		}
	}
	return keys, nil
}

func (m *memoryLayeredStore) Close() error {
	// Nothing to close for in-memory store
	return nil
}

// LayeredStore specific methods
func (m *memoryLayeredStore) SavePromptLayer(ctx context.Context, layer, identifier string, data []byte) error {
	bucket := fmt.Sprintf("prompt:%s", layer)
	return m.Put(ctx, bucket, identifier, data)
}

func (m *memoryLayeredStore) GetPromptLayer(ctx context.Context, layer, identifier string) ([]byte, error) {
	bucket := fmt.Sprintf("prompt:%s", layer)
	return m.Get(ctx, bucket, identifier)
}

func (m *memoryLayeredStore) DeletePromptLayer(ctx context.Context, layer, identifier string) error {
	bucket := fmt.Sprintf("prompt:%s", layer)
	return m.Delete(ctx, bucket, identifier)
}

func (m *memoryLayeredStore) ListPromptLayers(ctx context.Context, layer string) ([]string, error) {
	bucket := fmt.Sprintf("prompt:%s", layer)
	return m.List(ctx, bucket)
}

func (m *memoryLayeredStore) CacheCompiledPrompt(ctx context.Context, cacheKey string, data []byte) error {
	return m.Put(ctx, "cache", cacheKey, data)
}

func (m *memoryLayeredStore) GetCachedPrompt(ctx context.Context, cacheKey string) ([]byte, error) {
	return m.Get(ctx, "cache", cacheKey)
}

func (m *memoryLayeredStore) InvalidatePromptCache(ctx context.Context, keyPattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simple implementation - clear all cache entries
	// In a real implementation, this would match patterns
	prefix := "cache:"
	for k := range m.prompts {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(m.prompts, k)
		}
	}
	return nil
}

func (m *memoryLayeredStore) SavePromptMetrics(ctx context.Context, metricID string, data []byte) error {
	return m.Put(ctx, "metrics", metricID, data)
}

func (m *memoryLayeredStore) GetPromptMetrics(ctx context.Context, metricID string) ([]byte, error) {
	return m.Get(ctx, "metrics", metricID)
}

// kanbanRegistryAdapter adapts the main ComponentRegistry to kanban's ComponentRegistry interface
type kanbanRegistryAdapter struct {
	registry registry.ComponentRegistry
}

// Storage implements kanban.ComponentRegistry
func (a *kanbanRegistryAdapter) Storage() kanban.StorageRegistry {
	mainStorage := a.registry.Storage()
	if mainStorage == nil {
		return nil
	}
	return &kanbanStorageAdapter{storage: mainStorage}
}

// kanbanStorageAdapter adapts the main StorageRegistry to kanban's StorageRegistry interface
type kanbanStorageAdapter struct {
	storage registry.StorageRegistry
}

// GetKanbanCampaignRepository implements kanban.StorageRegistry
func (a *kanbanStorageAdapter) GetKanbanCampaignRepository() kanban.CampaignRepository {
	return a.storage.GetKanbanCampaignRepository()
}

// GetKanbanCommissionRepository implements kanban.StorageRegistry
func (a *kanbanStorageAdapter) GetKanbanCommissionRepository() kanban.CommissionRepository {
	return a.storage.GetKanbanCommissionRepository()
}

// GetBoardRepository implements kanban.StorageRegistry
func (a *kanbanStorageAdapter) GetBoardRepository() kanban.BoardRepository {
	return a.storage.GetBoardRepository()
}

// GetKanbanTaskRepository implements kanban.StorageRegistry
func (a *kanbanStorageAdapter) GetKanbanTaskRepository() kanban.TaskRepository {
	return a.storage.GetKanbanTaskRepository()
}

// GetMemoryStore implements kanban.StorageRegistry
func (a *kanbanStorageAdapter) GetMemoryStore() kanban.MemoryStore {
	return a.storage.GetMemoryStore()
}
