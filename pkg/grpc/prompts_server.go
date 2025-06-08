package grpc

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
	"github.com/guild-ventures/guild-core/pkg/prompts/layered"
)

// PromptsServer implements the gRPC PromptService for Guild layered prompt management
type PromptsServer struct {
	promptspb.UnimplementedPromptServiceServer

	manager     layered.LayeredManager
	subscribers map[string]chan *promptspb.PromptUpdateEvent // For streaming updates
	subMutex    sync.RWMutex
}

// NewPromptsServer creates a new gRPC prompts server
func NewPromptsServer(manager layered.LayeredManager) *PromptsServer {
	return &PromptsServer{
		manager:     manager,
		subscribers: make(map[string]chan *promptspb.PromptUpdateEvent),
	}
}

// GetPromptLayer retrieves a specific prompt layer
func (s *PromptsServer) GetPromptLayer(ctx context.Context, req *promptspb.GetPromptLayerRequest) (*promptspb.GetPromptLayerResponse, error) {
	if s.manager == nil {
		return nil, status.Error(codes.Unimplemented, "prompt manager not available")
	}

	if req.Layer == promptspb.PromptLayer_PROMPT_LAYER_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "prompt layer must be specified")
	}

	layer := convertProtoToPromptLayer(req.Layer)
	prompt, err := s.manager.GetPromptLayer(ctx, layer, req.ArtisanId, req.SessionId)
	if err != nil {
		if err == layered.ErrLayerNotFound {
			return nil, status.Error(codes.NotFound, "prompt layer not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get prompt layer: %v", err)
	}

	return &promptspb.GetPromptLayerResponse{
		Prompt: convertSystemPromptToProto(prompt),
	}, nil
}

// SetPromptLayer sets or updates a specific prompt layer
func (s *PromptsServer) SetPromptLayer(ctx context.Context, req *promptspb.SetPromptLayerRequest) (*promptspb.SetPromptLayerResponse, error) {
	if s.manager == nil {
		return nil, status.Error(codes.Unimplemented, "prompt manager not available")
	}

	if req.Prompt == nil {
		return nil, status.Error(codes.InvalidArgument, "prompt must be provided")
	}

	if req.Prompt.Layer == promptspb.PromptLayer_PROMPT_LAYER_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "prompt layer must be specified")
	}

	if strings.TrimSpace(req.Prompt.Content) == "" {
		return nil, status.Error(codes.InvalidArgument, "prompt content cannot be empty")
	}

	prompt := convertProtoToSystemPrompt(req.Prompt)
	err := s.manager.SetPromptLayer(ctx, *prompt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set prompt layer: %v", err)
	}

	// Notify subscribers of the update
	s.notifySubscribers(&promptspb.PromptUpdateEvent{
		EventType: promptspb.PromptUpdateEvent_EVENT_TYPE_UPDATED,
		Layer:     req.Prompt.Layer,
		ArtisanId: req.Prompt.ArtisanId,
		SessionId: req.Prompt.SessionId,
		Prompt:    req.Prompt,
		Timestamp: timestamppb.Now(),
	})

	return &promptspb.SetPromptLayerResponse{
		Success: true,
		Message: "Prompt layer updated successfully",
	}, nil
}

// DeletePromptLayer removes a specific prompt layer
func (s *PromptsServer) DeletePromptLayer(ctx context.Context, req *promptspb.DeletePromptLayerRequest) (*promptspb.DeletePromptLayerResponse, error) {
	if s.manager == nil {
		return nil, status.Error(codes.Unimplemented, "prompt manager not available")
	}

	if req.Layer == promptspb.PromptLayer_PROMPT_LAYER_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "prompt layer must be specified")
	}

	layer := convertProtoToPromptLayer(req.Layer)
	err := s.manager.DeletePromptLayer(ctx, layer, req.ArtisanId, req.SessionId)
	if err != nil {
		if err == layered.ErrLayerNotFound {
			return nil, status.Error(codes.NotFound, "prompt layer not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete prompt layer: %v", err)
	}

	// Notify subscribers of the deletion
	s.notifySubscribers(&promptspb.PromptUpdateEvent{
		EventType: promptspb.PromptUpdateEvent_EVENT_TYPE_DELETED,
		Layer:     req.Layer,
		ArtisanId: req.ArtisanId,
		SessionId: req.SessionId,
		Timestamp: timestamppb.Now(),
	})

	return &promptspb.DeletePromptLayerResponse{
		Success: true,
		Message: "Prompt layer deleted successfully",
	}, nil
}

// ListPromptLayers returns all layers for an artisan/session
func (s *PromptsServer) ListPromptLayers(ctx context.Context, req *promptspb.ListPromptLayersRequest) (*promptspb.ListPromptLayersResponse, error) {
	if s.manager == nil {
		return nil, status.Error(codes.Unimplemented, "prompt manager not available")
	}

	layers, err := s.manager.ListPromptLayers(ctx, req.ArtisanId, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list prompt layers: %v", err)
	}

	var protoPrompts []*promptspb.SystemPrompt
	for _, layer := range layers {
		protoPrompts = append(protoPrompts, convertSystemPromptToProto(&layer))
	}

	return &promptspb.ListPromptLayersResponse{
		Prompts: protoPrompts,
	}, nil
}

// BuildLayeredPrompt assembles a complete layered prompt
func (s *PromptsServer) BuildLayeredPrompt(ctx context.Context, req *promptspb.BuildLayeredPromptRequest) (*promptspb.BuildLayeredPromptResponse, error) {
	if s.manager == nil {
		return nil, status.Error(codes.Unimplemented, "prompt manager not available")
	}

	if req.ArtisanId == "" {
		return nil, status.Error(codes.InvalidArgument, "artisan ID must be specified")
	}

	var turnCtx layered.TurnContext
	if req.TurnContext != nil {
		turnCtx = convertProtoToTurnContext(req.TurnContext)
	}

	layeredPrompt, err := s.manager.BuildLayeredPrompt(ctx, req.ArtisanId, req.SessionId, turnCtx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to build layered prompt: %v", err)
	}

	return &promptspb.BuildLayeredPromptResponse{
		Prompt: convertLayeredPromptToProto(layeredPrompt),
	}, nil
}

// InvalidateCache clears the layered prompt cache
func (s *PromptsServer) InvalidateCache(ctx context.Context, req *promptspb.InvalidateCacheRequest) (*promptspb.InvalidateCacheResponse, error) {
	if s.manager == nil {
		return nil, status.Error(codes.Unimplemented, "prompt manager not available")
	}

	err := s.manager.InvalidateCache(ctx, req.ArtisanId, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to invalidate cache: %v", err)
	}

	// Notify subscribers of cache invalidation
	s.notifySubscribers(&promptspb.PromptUpdateEvent{
		EventType: promptspb.PromptUpdateEvent_EVENT_TYPE_CACHE_INVALIDATED,
		ArtisanId: req.ArtisanId,
		SessionId: req.SessionId,
		Timestamp: timestamppb.Now(),
	})

	return &promptspb.InvalidateCacheResponse{
		Success: true,
		Message: "Cache invalidated successfully",
	}, nil
}

// GetLayerStats returns statistics for a specific layer
func (s *PromptsServer) GetLayerStats(ctx context.Context, req *promptspb.GetLayerStatsRequest) (*promptspb.GetLayerStatsResponse, error) {
	if req.Layer == promptspb.PromptLayer_PROMPT_LAYER_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "prompt layer must be specified")
	}

	// For now, return mock statistics
	// TODO: Implement actual statistics collection in the manager
	stats := &promptspb.LayerStats{
		Layer:         req.Layer,
		PromptCount:   42,   // Mock data
		AverageTokens: 1247, // Mock data
		LastUpdated:   timestamppb.Now(),
	}

	return &promptspb.GetLayerStatsResponse{
		Stats: stats,
	}, nil
}

// StreamPromptUpdates provides real-time prompt update notifications
func (s *PromptsServer) StreamPromptUpdates(req *promptspb.StreamPromptUpdatesRequest, stream promptspb.PromptService_StreamPromptUpdatesServer) error {
	// Create subscriber channel
	subscriberID := fmt.Sprintf("%s:%s:%d", req.ArtisanId, req.SessionId, time.Now().UnixNano())
	updateChan := make(chan *promptspb.PromptUpdateEvent, 100)

	s.subMutex.Lock()
	s.subscribers[subscriberID] = updateChan
	s.subMutex.Unlock()

	// Clean up when done
	defer func() {
		s.subMutex.Lock()
		delete(s.subscribers, subscriberID)
		close(updateChan)
		s.subMutex.Unlock()
	}()

	// Send initial connection confirmation
	welcomeEvent := &promptspb.PromptUpdateEvent{
		EventType: promptspb.PromptUpdateEvent_EVENT_TYPE_UNSPECIFIED,
		Timestamp: timestamppb.Now(),
		Metadata: map[string]string{
			"subscriber_id": subscriberID,
			"message":       "Connected to Guild prompt update stream",
		},
	}
	if err := stream.Send(welcomeEvent); err != nil {
		return err
	}

	// Stream updates
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case event, ok := <-updateChan:
			if !ok {
				return nil
			}

			// Filter by requested layers if specified
			if len(req.Layers) > 0 {
				found := false
				for _, layer := range req.Layers {
					if event.Layer == layer {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			// Filter by artisan/session if specified
			if req.ArtisanId != "" && event.ArtisanId != req.ArtisanId {
				continue
			}
			if req.SessionId != "" && event.SessionId != req.SessionId {
				continue
			}

			if err := stream.Send(event); err != nil {
				return err
			}
		}
	}
}

// Helper methods for conversion between proto and domain types

func convertProtoToPromptLayer(layer promptspb.PromptLayer) layered.PromptLayer {
	switch layer {
	case promptspb.PromptLayer_PROMPT_LAYER_PLATFORM:
		return layered.LayerPlatform
	case promptspb.PromptLayer_PROMPT_LAYER_GUILD:
		return layered.LayerGuild
	case promptspb.PromptLayer_PROMPT_LAYER_ROLE:
		return layered.LayerRole
	case promptspb.PromptLayer_PROMPT_LAYER_DOMAIN:
		return layered.LayerDomain
	case promptspb.PromptLayer_PROMPT_LAYER_SESSION:
		return layered.LayerSession
	case promptspb.PromptLayer_PROMPT_LAYER_TURN:
		return layered.LayerTurn
	default:
		return layered.LayerPlatform
	}
}

func convertPromptLayerToProto(layer layered.PromptLayer) promptspb.PromptLayer {
	switch layer {
	case layered.LayerPlatform:
		return promptspb.PromptLayer_PROMPT_LAYER_PLATFORM
	case layered.LayerGuild:
		return promptspb.PromptLayer_PROMPT_LAYER_GUILD
	case layered.LayerRole:
		return promptspb.PromptLayer_PROMPT_LAYER_ROLE
	case layered.LayerDomain:
		return promptspb.PromptLayer_PROMPT_LAYER_DOMAIN
	case layered.LayerSession:
		return promptspb.PromptLayer_PROMPT_LAYER_SESSION
	case layered.LayerTurn:
		return promptspb.PromptLayer_PROMPT_LAYER_TURN
	default:
		return promptspb.PromptLayer_PROMPT_LAYER_PLATFORM
	}
}

func convertProtoToSystemPrompt(proto *promptspb.SystemPrompt) *layered.SystemPrompt {
	metadata := make(map[string]interface{})
	for k, v := range proto.Metadata {
		metadata[k] = v
	}

	return &layered.SystemPrompt{
		Layer:     convertProtoToPromptLayer(proto.Layer),
		ArtisanID: proto.ArtisanId,
		SessionID: proto.SessionId,
		Content:   proto.Content,
		Version:   int(proto.Version),
		Priority:  int(proto.Priority),
		Updated:   proto.Updated.AsTime(),
		Metadata:  metadata,
	}
}

func convertSystemPromptToProto(prompt *layered.SystemPrompt) *promptspb.SystemPrompt {
	metadata := make(map[string]string)
	for k, v := range prompt.Metadata {
		if str, ok := v.(string); ok {
			metadata[k] = str
		} else {
			metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	return &promptspb.SystemPrompt{
		Layer:     convertPromptLayerToProto(prompt.Layer),
		ArtisanId: prompt.ArtisanID,
		SessionId: prompt.SessionID,
		Content:   prompt.Content,
		Version:   int32(prompt.Version),
		Priority:  int32(prompt.Priority),
		Updated:   timestamppb.New(prompt.Updated),
		Metadata:  metadata,
	}
}

func convertProtoToTurnContext(proto *promptspb.TurnContext) layered.TurnContext {
	metadata := make(map[string]interface{})
	for k, v := range proto.Metadata {
		metadata[k] = v
	}

	return layered.TurnContext{
		UserMessage:  proto.UserMessage,
		TaskID:       proto.TaskId,
		CommissionID: proto.CommissionId,
		Urgency:      proto.Urgency,
		Instructions: proto.Instructions,
		Metadata:     metadata,
	}
}

func convertLayeredPromptToProto(layered *layered.LayeredPrompt) *promptspb.LayeredPrompt {
	var protoLayers []*promptspb.SystemPrompt
	for _, layer := range layered.Layers {
		protoLayers = append(protoLayers, convertSystemPromptToProto(&layer))
	}

	metadata := make(map[string]string)
	for k, v := range layered.Metadata {
		if str, ok := v.(string); ok {
			metadata[k] = str
		} else {
			metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	return &promptspb.LayeredPrompt{
		Layers:      protoLayers,
		Compiled:    layered.Compiled,
		TokenCount:  int32(layered.TokenCount),
		Truncated:   layered.Truncated,
		CacheKey:    layered.CacheKey,
		ArtisanId:   layered.ArtisanID,
		SessionId:   layered.SessionID,
		AssembledAt: timestamppb.New(layered.AssembledAt),
		Metadata:    metadata,
	}
}

// notifySubscribers sends updates to all stream subscribers
func (s *PromptsServer) notifySubscribers(event *promptspb.PromptUpdateEvent) {
	s.subMutex.RLock()
	defer s.subMutex.RUnlock()

	for _, ch := range s.subscribers {
		select {
		case ch <- event:
		default:
			// Channel is full, skip this subscriber
		}
	}
}
