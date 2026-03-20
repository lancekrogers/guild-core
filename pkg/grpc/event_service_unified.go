// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lancekrogers/guild-core/pkg/events"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	pb "github.com/lancekrogers/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// UnifiedEventService implements the gRPC EventService with unified event bus integration
type UnifiedEventService struct {
	pb.UnimplementedEventServiceServer

	eventBus events.EventBus

	// Stream management
	streams   map[string]*unifiedEventStream
	streamsMu sync.RWMutex
}

// unifiedEventStream represents an active event stream subscription
type unifiedEventStream struct {
	id              string
	eventTypes      []string
	subscribeAll    bool
	stream          pb.EventService_StreamEventsServer
	done            chan struct{}
	subscriptionIDs []events.SubscriptionID
}

// NewUnifiedEventService creates a new event service using the unified event bus
func NewUnifiedEventService(eventBus events.EventBus) pb.EventServiceServer {
	return &UnifiedEventService{
		eventBus: eventBus,
		streams:  make(map[string]*unifiedEventStream),
	}
}

// StreamEvents implements event streaming with pattern matching
func (s *UnifiedEventService) StreamEvents(req *pb.StreamEventsRequest, stream pb.EventService_StreamEventsServer) error {
	ctx := stream.Context()
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("StreamEvents")

	streamID := uuid.New().String()
	es := &unifiedEventStream{
		id:              streamID,
		eventTypes:      req.EventTypes,
		subscribeAll:    req.SubscribeAll,
		stream:          stream,
		done:            make(chan struct{}),
		subscriptionIDs: make([]events.SubscriptionID, 0),
	}

	// Register stream
	s.streamsMu.Lock()
	s.streams[streamID] = es
	s.streamsMu.Unlock()

	defer func() {
		// Unsubscribe all subscriptions
		for _, subID := range es.subscriptionIDs {
			if err := s.eventBus.Unsubscribe(ctx, subID); err != nil {
				logger.WithError(err).Warn("Failed to unsubscribe",
					"subscription_id", subID,
					"stream_id", streamID,
				)
			}
		}

		s.streamsMu.Lock()
		delete(s.streams, streamID)
		s.streamsMu.Unlock()
		close(es.done)
	}()

	logger.Info("Event stream started",
		"stream_id", streamID,
		"event_types", req.EventTypes,
		"subscribe_all", req.SubscribeAll,
	)

	// Create event handler that converts unified events to proto
	handler := func(ctx context.Context, event events.CoreEvent) error {
		// Check if stream is still active
		select {
		case <-es.done:
			return nil
		default:
		}

		// Convert to proto event
		protoEvent, err := s.unifiedEventToProto(event)
		if err != nil {
			logger.WithError(err).Warn("Failed to convert event to proto",
				"event_id", event.GetID(),
				"event_type", event.GetType(),
			)
			return nil
		}

		// Check timestamp filter
		if req.Since != nil && protoEvent.Timestamp.AsTime().Before(req.Since.AsTime()) {
			return nil
		}

		// Send event to stream
		if err := stream.Send(protoEvent); err != nil {
			logger.WithError(err).Warn("Failed to send event to stream",
				"stream_id", streamID,
				"event_id", event.GetID(),
			)
			// Stream error will cause cleanup in main loop
			return err
		}

		return nil
	}

	// Subscribe to events based on request
	if req.SubscribeAll {
		subID, err := s.eventBus.SubscribeAll(ctx, handler)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to subscribe to all events").
				WithComponent("grpc").
				WithOperation("StreamEvents")
		}
		es.subscriptionIDs = append(es.subscriptionIDs, subID)
	} else {
		// Subscribe to each event type pattern
		for _, pattern := range req.EventTypes {
			subID, err := s.eventBus.Subscribe(ctx, pattern, handler)
			if err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to subscribe to event type").
					WithComponent("grpc").
					WithOperation("StreamEvents").
					WithDetails("event_type", pattern)
			}
			es.subscriptionIDs = append(es.subscriptionIDs, subID)
		}
	}

	// Keep stream alive until context is cancelled
	<-ctx.Done()
	logger.Info("Event stream ended", "stream_id", streamID)
	return nil
}

// PublishEvent publishes an event to the system event bus.
// Automatically generates event ID and timestamp if not provided.
//
// Errors:
//   - ErrCodeCancelled: Context was cancelled
//   - ErrCodeInvalidInput: Event is nil or invalid
func (s *UnifiedEventService) PublishEvent(ctx context.Context, req *pb.PublishEventRequest) (*pb.PublishEventResponse, error) {
	// Check context cancellation early
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("grpc").
			WithOperation("PublishEvent")
	}

	if req.Event == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "event is required", nil).
			WithComponent("grpc").
			WithOperation("PublishEvent")
	}

	event := req.Event

	// Generate ID if not provided
	if event.Id == "" {
		event.Id = uuid.New().String()
	}

	// Set timestamp if not provided
	if event.Timestamp == nil {
		event.Timestamp = timestamppb.Now()
	}

	// Convert proto event to unified event
	unifiedEvent, err := s.protoEventToUnified(event)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event data: %v", err)
	}

	// Publish to event bus
	if err := s.eventBus.Publish(ctx, unifiedEvent); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to publish event").
			WithComponent("grpc").
			WithOperation("PublishEvent").
			WithDetails("event_id", event.Id)
	}

	observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("PublishEvent").
		Info("Event published",
			"event_id", event.Id,
			"event_type", event.Type,
			"source", event.Source,
		)

	return &pb.PublishEventResponse{
		Success: true,
		Message: fmt.Sprintf("Event %s published successfully", event.Id),
	}, nil
}

// Helper methods

func (s *UnifiedEventService) unifiedEventToProto(event events.CoreEvent) (*pb.Event, error) {
	// Convert map[string]interface{} to structpb.Struct
	dataStruct, err := structpb.NewStruct(event.GetData())
	if err != nil {
		return nil, fmt.Errorf("failed to convert event data to struct: %w", err)
	}

	return &pb.Event{
		Id:        event.GetID(),
		Type:      event.GetType(),
		Timestamp: timestamppb.New(event.GetTimestamp()),
		Source:    event.GetSource(),
		Data:      dataStruct,
	}, nil
}

func (s *UnifiedEventService) protoEventToUnified(event *pb.Event) (events.CoreEvent, error) {
	// Convert structpb.Struct to map[string]interface{}
	data := make(map[string]interface{})
	if event.Data != nil {
		data = event.Data.AsMap()
	}

	unifiedEvent := events.NewBaseEvent(
		event.Id,
		event.Type,
		event.Source,
		data,
	)

	// Set timestamp if provided
	if event.Timestamp != nil {
		unifiedEvent.Timestamp = event.Timestamp.AsTime()
	}

	return unifiedEvent, nil
}

// matchesPatternUnified checks if an event type matches a subscription pattern
func matchesPatternUnified(eventType, pattern string) bool {
	// Support wildcard patterns like "task.*"
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(eventType, prefix)
	}
	return eventType == pattern
}
