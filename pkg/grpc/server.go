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
	"github.com/guild-ventures/guild-core/pkg/objective"
	"github.com/guild-ventures/guild-core/pkg/orchestrator"
	"github.com/guild-ventures/guild-core/pkg/prompts"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// Server implements the Guild gRPC service
type Server struct {
	pb.UnimplementedGuildServer
	
	campaignMgr   campaign.Manager
	objectiveMgr  *objective.Manager
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

// NewServer creates a new gRPC server
func NewServer(
	campaignMgr campaign.Manager,
	objectiveMgr *objective.Manager,
	kanbanMgr *kanban.Manager,
	agentReg registry.AgentRegistry,
	orchestrator *orchestrator.Orchestrator,
	promptManager prompts.LayeredManager,
) *Server {
	promptServer := NewPromptsServer(promptManager)
	
	return &Server{
		campaignMgr:   campaignMgr,
		objectiveMgr:  objectiveMgr,
		kanbanMgr:     kanbanMgr,
		agentReg:      agentReg,
		orchestrator:  orchestrator,
		promptManager: promptManager,
		frameBuilder:  NewFrameBuilder(campaignMgr, objectiveMgr, kanbanMgr, agentReg),
		watchers:      make(map[string]*watcher),
		promptServer:  promptServer,
	}
}

// Start starts the gRPC server
func (s *Server) Start(ctx context.Context, address string) error {
	var err error
	s.listener, err = net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.grpcServer = grpc.NewServer()
	pb.RegisterGuildServer(s.grpcServer, s)
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
	campaign, err := s.campaignMgr.Get(context.Background(), w.campaignID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	// Build frame
	frame, metadata := s.frameBuilder.BuildFrame(campaign, w.options)

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

// AddObjectiveToCampaign adds an objective to a campaign
func (s *Server) AddObjectiveToCampaign(ctx context.Context, req *pb.AddObjectiveRequest) (*pb.Campaign, error) {
	if err := s.campaignMgr.AddObjective(ctx, req.CampaignId, req.ObjectiveId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add objective: %v", err)
	}

	c, err := s.campaignMgr.Get(ctx, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	return campaignToProto(c), nil
}

// RemoveObjectiveFromCampaign removes an objective from a campaign
func (s *Server) RemoveObjectiveFromCampaign(ctx context.Context, req *pb.RemoveObjectiveRequest) (*pb.Campaign, error) {
	if err := s.campaignMgr.RemoveObjective(ctx, req.CampaignId, req.ObjectiveId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove objective: %v", err)
	}

	c, err := s.campaignMgr.Get(ctx, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	return campaignToProto(c), nil
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
		Id:                  c.ID,
		Name:                c.Name,
		Description:         c.Description,
		Status:              string(c.Status),
		ObjectiveIds:        c.Objectives,
		Tags:                c.Tags,
		Progress:            c.Progress,
		TotalObjectives:     int32(c.TotalObjectives),
		CompletedObjectives: int32(c.CompletedObjectives),
		CreatedAt:           c.CreatedAt.Unix(),
		UpdatedAt:           c.UpdatedAt.Unix(),
		Metadata:            metadata,
	}

	if c.StartedAt != nil {
		proto.StartedAt = c.StartedAt.Unix()
	}
	if c.CompletedAt != nil {
		proto.CompletedAt = c.CompletedAt.Unix()
	}

	return proto
}