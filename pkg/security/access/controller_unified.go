// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package access

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild-core/pkg/events"
	"github.com/lancekrogers/guild-core/pkg/observability"
	"github.com/lancekrogers/guild-core/pkg/security/permissions"
)

// UnifiedAccessController wraps AccessController to use the unified event system
type UnifiedAccessController struct {
	*AccessController
	unifiedEventBus events.EventBus
}

// NewUnifiedAccessController creates a new access controller with unified event bus
func NewUnifiedAccessController(ctx context.Context, permissionModel *permissions.PermissionModel, auditor AuditLogger, unifiedEventBus events.EventBus) *UnifiedAccessController {
	// Create base controller without legacy event bus
	baseController := NewAccessController(ctx, permissionModel, auditor, nil)

	return &UnifiedAccessController{
		AccessController: baseController,
		unifiedEventBus:  unifiedEventBus,
	}
}

// CheckAccess overrides to publish events to unified bus
func (uac *UnifiedAccessController) CheckAccess(ctx context.Context, req AccessRequest) (*AccessDecision, error) {
	// Use base implementation for the core logic
	decision, err := uac.AccessController.CheckAccess(ctx, req)
	if err != nil {
		return nil, err
	}

	// Publish access denied event to unified bus if denied
	if !decision.Allowed && uac.unifiedEventBus != nil {
		event := events.NewBaseEvent(
			uuid.New().String(),
			"access.denied",
			"security-controller",
			map[string]interface{}{
				"agent_id":  req.AgentID,
				"resource":  decision.Resource,
				"action":    req.Action,
				"reason":    decision.Reason,
				"timestamp": time.Now(),
			},
		)

		// Add request metadata
		if req.RequestID != "" {
			event.WithData("request_id", req.RequestID)
		}
		if req.SessionID != "" {
			event.WithData("session_id", req.SessionID)
		}
		if req.IPAddress != "" {
			event.WithData("ip_address", req.IPAddress)
		}

		// Publish event
		if publishErr := uac.unifiedEventBus.Publish(ctx, event); publishErr != nil {
			// Log error but don't fail the access check
			logger := observability.GetLogger(ctx).
				WithComponent("UnifiedAccessController").
				WithOperation("CheckAccess")
			logger.WithError(publishErr).Warn("Failed to publish access denied event")
		}
	}

	return decision, nil
}

// PublishSecurityAlert publishes a security alert to the unified event bus
func (uac *UnifiedAccessController) PublishSecurityAlert(ctx context.Context, alert SecurityAlert) error {
	if uac.unifiedEventBus == nil {
		return nil
	}

	event := events.NewBaseEvent(
		alert.ID,
		"security.alert",
		"security-controller",
		map[string]interface{}{
			"type":        alert.Type,
			"severity":    alert.Severity.String(),
			"title":       alert.Title,
			"description": alert.Description,
			"timestamp":   alert.Timestamp,
			"resolved":    alert.Resolved,
		},
	)

	// Add optional fields
	if alert.AgentID != "" {
		event.WithData("agent_id", alert.AgentID)
	}
	if alert.Resource != "" {
		event.WithData("resource", alert.Resource)
	}
	if alert.Action != "" {
		event.WithData("action", alert.Action)
	}

	// Add metadata
	for k, v := range alert.Metadata {
		event.WithData(k, v)
	}

	return uac.unifiedEventBus.Publish(ctx, event)
}

// PublishAuditEvent publishes an audit event to the unified event bus
func (uac *UnifiedAccessController) PublishAuditEvent(ctx context.Context, entry AuditEntry) error {
	if uac.unifiedEventBus == nil {
		return nil
	}

	eventType := "audit." + entry.Result // e.g., "audit.allowed", "audit.denied"

	event := events.NewBaseEvent(
		entry.ID,
		eventType,
		"security-controller",
		map[string]interface{}{
			"agent_id":  entry.AgentID,
			"resource":  entry.Resource,
			"action":    entry.Action,
			"result":    entry.Result,
			"timestamp": entry.Timestamp,
		},
	)

	// Add optional fields
	if entry.UserID != "" {
		event.WithData("user_id", entry.UserID)
	}
	if entry.Reason != "" {
		event.WithData("reason", entry.Reason)
	}
	if entry.Duration > 0 {
		event.WithData("duration_ms", entry.Duration.Milliseconds())
	}
	if entry.IPAddress != "" {
		event.WithData("ip_address", entry.IPAddress)
	}
	if entry.SessionID != "" {
		event.WithData("session_id", entry.SessionID)
	}
	if entry.RequestID != "" {
		event.WithData("request_id", entry.RequestID)
	}

	// Add metadata
	for k, v := range entry.Metadata {
		event.WithData(k, v)
	}

	return uac.unifiedEventBus.Publish(ctx, event)
}

// UnifiedEventBusAdapter implements the EventBus interface using unified events
type UnifiedEventBusAdapter struct {
	eventBus events.EventBus
}

// NewUnifiedEventBusAdapter creates a new adapter for the EventBus interface
func NewUnifiedEventBusAdapter(eventBus events.EventBus) EventBus {
	return &UnifiedEventBusAdapter{
		eventBus: eventBus,
	}
}

// PublishAccessEvent publishes an access control event
func (a *UnifiedEventBusAdapter) PublishAccessEvent(ctx context.Context, event AccessEvent) error {
	if a.eventBus == nil {
		return nil
	}

	unifiedEvent := events.NewBaseEvent(
		uuid.New().String(),
		event.Type,
		"security-adapter",
		map[string]interface{}{
			"agent_id":  event.AgentID,
			"resource":  event.Resource,
			"action":    event.Action,
			"reason":    event.Reason,
			"timestamp": event.Timestamp,
		},
	)

	// Add optional fields
	if event.RequestID != "" {
		unifiedEvent.WithData("request_id", event.RequestID)
	}

	// Add metadata
	for k, v := range event.Metadata {
		unifiedEvent.WithData(k, v)
	}

	return a.eventBus.Publish(ctx, unifiedEvent)
}

// PublishSecurityAlert publishes a security alert
func (a *UnifiedEventBusAdapter) PublishSecurityAlert(ctx context.Context, alert SecurityAlert) error {
	if a.eventBus == nil {
		return nil
	}

	unifiedEvent := events.NewBaseEvent(
		alert.ID,
		"security.alert",
		"security-adapter",
		map[string]interface{}{
			"type":        alert.Type,
			"severity":    alert.Severity.String(),
			"title":       alert.Title,
			"description": alert.Description,
			"timestamp":   alert.Timestamp,
			"resolved":    alert.Resolved,
		},
	)

	// Add optional fields
	if alert.AgentID != "" {
		unifiedEvent.WithData("agent_id", alert.AgentID)
	}
	if alert.Resource != "" {
		unifiedEvent.WithData("resource", alert.Resource)
	}
	if alert.Action != "" {
		unifiedEvent.WithData("action", alert.Action)
	}

	// Add metadata
	for k, v := range alert.Metadata {
		unifiedEvent.WithData(k, v)
	}

	return a.eventBus.Publish(ctx, unifiedEvent)
}
