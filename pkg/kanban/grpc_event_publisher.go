// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	pb "github.com/lancekrogers/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// GRPCEventPublisher publishes kanban events to the gRPC event service
type GRPCEventPublisher struct {
	eventClient pb.EventServiceClient
	ctx         context.Context
}

// NewGRPCEventPublisher creates a new gRPC event publisher
func NewGRPCEventPublisher(ctx context.Context, conn *grpc.ClientConn) *GRPCEventPublisher {
	if conn == nil {
		return nil
	}

	return &GRPCEventPublisher{
		eventClient: pb.NewEventServiceClient(conn),
		ctx:         ctx,
	}
}

// PublishTaskEvent publishes a task event to the gRPC event service
func (p *GRPCEventPublisher) PublishTaskEvent(event *BoardEvent) error {
	if p.eventClient == nil {
		return nil // No event client, skip publishing
	}

	// Convert BoardEvent to gRPC Event
	data := make(map[string]interface{})
	data["board_id"] = event.BoardID
	data["task_id"] = event.TaskID

	// Add event-specific data
	for k, v := range event.Data {
		data[k] = v
	}

	// Create struct from data
	dataStruct, err := structpb.NewStruct(data)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create event data struct").
			WithComponent("kanban").
			WithOperation("PublishTaskEvent")
	}

	// Create gRPC event
	grpcEvent := &pb.Event{
		Type:      fmt.Sprintf("task.%s", event.EventType),
		Timestamp: timestamppb.New(event.OccurredAt),
		Source:    "kanban-board",
		Data:      dataStruct,
	}

	// Publish event
	req := &pb.PublishEventRequest{
		Event: grpcEvent,
	}

	resp, err := p.eventClient.PublishEvent(p.ctx, req)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to publish event to gRPC service").
			WithComponent("kanban").
			WithOperation("PublishTaskEvent").
			WithDetails("event_type", string(event.EventType))
	}

	if !resp.Success {
		return gerror.Newf(gerror.ErrCodeInternal, "event publish failed: %s", resp.Message).
			WithComponent("kanban").
			WithOperation("PublishTaskEvent")
	}

	return nil
}

// WrapEventManager wraps an existing event manager to also publish to gRPC
func (p *GRPCEventPublisher) WrapEventManager(em *EventManager) *EventManager {
	if p == nil || em == nil {
		return em
	}

	// Subscribe to all events and republish to gRPC
	em.SubscribeAll(func(event *BoardEvent) error {
		// Publish to gRPC in background to avoid blocking
		go func() {
			if err := p.PublishTaskEvent(event); err != nil {
				observability.GetLogger(p.ctx).Warn("failed to publish kanban event to gRPC", "error", err)
			}
		}()
		return nil
	})

	return em
}
