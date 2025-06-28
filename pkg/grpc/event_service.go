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

	"github.com/lancekrogers/guild/pkg/gerror"
	pb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/orchestrator"
	"github.com/lancekrogers/guild/pkg/orchestrator/interfaces"
)

// EventService implements the gRPC EventService with orchestrator event bus integration
type EventService struct {
	pb.UnimplementedEventServiceServer

	eventBus orchestrator.EventBus

	// Stream management
	streams   map[string]*eventStream
	streamsMu sync.RWMutex
}

// eventStream represents an active event stream subscription
type eventStream struct {
	id           string
	eventTypes   []string
	subscribeAll bool
	stream       pb.EventService_StreamEventsServer
	done         chan struct{}
}

// NewEventService creates a new event service
func NewEventService(eventBus orchestrator.EventBus) pb.EventServiceServer {
	return &EventService{
		eventBus: eventBus,
		streams:  make(map[string]*eventStream),
	}
}

// StreamEvents implements event streaming with pattern matching
func (s *EventService) StreamEvents(req *pb.StreamEventsRequest, stream pb.EventService_StreamEventsServer) error {
	ctx := stream.Context()
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("StreamEvents")

	streamID := uuid.New().String()
	es := &eventStream{
		id:           streamID,
		eventTypes:   req.EventTypes,
		subscribeAll: req.SubscribeAll,
		stream:       stream,
		done:         make(chan struct{}),
	}

	// Register stream
	s.streamsMu.Lock()
	s.streams[streamID] = es
	s.streamsMu.Unlock()

	defer func() {
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

	// Create event handler that converts orchestrator events to proto
	handler := func(event interfaces.Event) {
		// Check if stream is still active
		select {
		case <-es.done:
			return
		default:
		}

		// Convert to proto event
		protoEvent, err := s.orchestratorEventToProto(event)
		if err != nil {
			logger.WithError(err).Warn("Failed to convert event to proto",
				"event_id", event.ID,
				"event_type", event.Type,
			)
			return
		}

		// Check timestamp filter
		if req.Since != nil && protoEvent.Timestamp.AsTime().Before(req.Since.AsTime()) {
			return
		}

		// Send event to stream
		if err := stream.Send(protoEvent); err != nil {
			logger.WithError(err).Warn("Failed to send event to stream",
				"stream_id", streamID,
				"event_id", event.ID,
			)
			// Stream error will cause cleanup in main loop
			return
		}
	}

	// Subscribe to events based on request
	if req.SubscribeAll {
		s.eventBus.SubscribeAll(handler)
		defer s.eventBus.Unsubscribe("*", handler) // Unsubscribe from all
	} else {
		// Subscribe to each event type pattern
		for _, pattern := range req.EventTypes {
			eventType := interfaces.EventType(pattern)
			s.eventBus.Subscribe(eventType, handler)
			defer s.eventBus.Unsubscribe(eventType, handler)
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
func (s *EventService) PublishEvent(ctx context.Context, req *pb.PublishEventRequest) (*pb.PublishEventResponse, error) {
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

	// Convert proto event to orchestrator event
	orchEvent, err := s.protoEventToOrchestrator(event)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event data: %v", err)
	}

	// Publish to event bus
	s.eventBus.Publish(orchEvent)

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

func (s *EventService) orchestratorEventToProto(event interfaces.Event) (*pb.Event, error) {
	// Convert map[string]interface{} to structpb.Struct
	dataStruct, err := structpb.NewStruct(event.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert event data to struct: %w", err)
	}

	return &pb.Event{
		Id:        event.ID,
		Type:      string(event.Type),
		Timestamp: timestamppb.New(event.Timestamp),
		Source:    event.Source,
		Data:      dataStruct,
	}, nil
}

func (s *EventService) protoEventToOrchestrator(event *pb.Event) (interfaces.Event, error) {
	// Convert structpb.Struct to map[string]interface{}
	data := make(map[string]interface{})
	if event.Data != nil {
		data = event.Data.AsMap()
	}

	return interfaces.Event{
		ID:        event.Id,
		Type:      interfaces.EventType(event.Type),
		Timestamp: event.Timestamp.AsTime(),
		Source:    event.Source,
		Data:      data,
	}, nil
}

// matchesPattern checks if an event type matches a subscription pattern
func matchesPattern(eventType, pattern string) bool {
	// Support wildcard patterns like "task.*"
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(eventType, prefix)
	}
	return eventType == pattern
}
