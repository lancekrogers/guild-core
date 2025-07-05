// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package campaign

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/orchestrator"
)

// unifiedManager implements the Manager interface using the unified event system
type unifiedManager struct {
	repo          Repository
	commissionMgr *commission.Manager
	fsm           FSM
	eventBus      events.EventBus
	handlers      map[string][]EventHandler
	subscriptions map[string]events.SubscriptionID
	mu            sync.RWMutex
}

// NewUnifiedManager creates a new campaign manager using the unified event system
func NewUnifiedManager(repo Repository, commissionMgr *commission.Manager, eventBus events.EventBus) Manager {
	mgr := &unifiedManager{
		repo:          repo,
		commissionMgr: commissionMgr,
		fsm:           NewFSM(),
		eventBus:      eventBus,
		handlers:      make(map[string][]EventHandler),
		subscriptions: make(map[string]events.SubscriptionID),
	}

	// Subscribe to commission events
	mgr.subscribeToCommissionEvents()

	return mgr
}

// Create creates a new campaign
func (m *unifiedManager) Create(ctx context.Context, campaign *Campaign) error {
	if campaign == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "campaign", nil).WithComponent("create_campaign").WithOperation("campaign cannot be nil")
	}

	// Set default status if not set
	if campaign.Status == "" {
		campaign.Status = CampaignStatusDream
	}

	// Set timestamps
	campaign.CreatedAt = time.Now()
	campaign.UpdatedAt = campaign.CreatedAt

	// Save to repository
	if m.repo != nil {
		if err := m.repo.Create(ctx, campaign); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create campaign").WithComponent("create_campaign").WithOperation("repo.Create")
		}
	}

	// Publish event
	m.publishUnifiedEvent(&CampaignEvent{
		Type:       EventCampaignCreated,
		CampaignID: campaign.ID,
		Campaign:   campaign,
		Timestamp:  time.Now(),
	})

	return nil
}

// Get retrieves a campaign by ID
func (m *unifiedManager) Get(ctx context.Context, id string) (*Campaign, error) {
	if m.repo == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "repository not available", nil).WithComponent("get_campaign").WithOperation("repo check")
	}

	campaign, err := m.repo.Get(ctx, id)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get campaign").WithComponent("get_campaign").WithOperation("repo.Get")
	}

	return campaign, nil
}

// List returns all campaigns
func (m *unifiedManager) List(ctx context.Context) ([]*Campaign, error) {
	if m.repo == nil {
		// Return empty list if no repository
		return []*Campaign{}, nil
	}

	campaigns, err := m.repo.List(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to list campaigns").WithComponent("list_campaigns").WithOperation("repo.List")
	}

	return campaigns, nil
}

// Update updates an existing campaign
func (m *unifiedManager) Update(ctx context.Context, campaign *Campaign) error {
	if campaign == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "campaign", nil).WithComponent("update_campaign").WithOperation("campaign cannot be nil")
	}

	// Update timestamp
	campaign.UpdatedAt = time.Now()

	// Save to repository
	if m.repo != nil {
		if err := m.repo.Update(ctx, campaign); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to update campaign").WithComponent("update_campaign").WithOperation("repo.Update")
		}
	}

	// Publish event
	m.publishUnifiedEvent(&CampaignEvent{
		Type:       "campaign.updated",
		CampaignID: campaign.ID,
		Campaign:   campaign,
		Timestamp:  time.Now(),
	})

	return nil
}

// Delete removes a campaign
func (m *unifiedManager) Delete(ctx context.Context, id string) error {
	// Delete from repository
	if m.repo != nil {
		if err := m.repo.Delete(ctx, id); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to delete campaign").WithComponent("delete_campaign").WithOperation("repo.Delete")
		}
	}

	// Publish event
	m.publishUnifiedEvent(&CampaignEvent{
		Type:       "campaign.deleted",
		CampaignID: id,
		Timestamp:  time.Now(),
	})

	return nil
}

// AddCommission adds a commission to a campaign
func (m *unifiedManager) AddCommission(ctx context.Context, campaignID, commissionID string) error {
	campaign, err := m.Get(ctx, campaignID)
	if err != nil {
		return err
	}

	// Check if commission already exists
	for _, cid := range campaign.Commissions {
		if cid == commissionID {
			return nil // Already added
		}
	}

	// Add commission
	campaign.Commissions = append(campaign.Commissions, commissionID)
	campaign.TotalCommissions++

	// Update campaign
	if err := m.Update(ctx, campaign); err != nil {
		return err
	}

	// Publish event
	m.publishUnifiedEvent(&CampaignEvent{
		Type:       EventCommissionAdded,
		CampaignID: campaignID,
		Campaign:   campaign,
		Timestamp:  time.Now(),
		Data: map[string]interface{}{
			"commission_id": commissionID,
		},
	})

	return nil
}

// RemoveCommission removes a commission from a campaign
func (m *unifiedManager) RemoveCommission(ctx context.Context, campaignID, commissionID string) error {
	campaign, err := m.Get(ctx, campaignID)
	if err != nil {
		return err
	}

	// Remove commission
	newCommissions := make([]string, 0, len(campaign.Commissions))
	found := false
	for _, cid := range campaign.Commissions {
		if cid != commissionID {
			newCommissions = append(newCommissions, cid)
		} else {
			found = true
		}
	}

	if !found {
		return nil // Not found, nothing to do
	}

	campaign.Commissions = newCommissions
	campaign.TotalCommissions--
	if campaign.CompletedCommissions > campaign.TotalCommissions {
		campaign.CompletedCommissions = campaign.TotalCommissions
	}

	// Update campaign
	if err := m.Update(ctx, campaign); err != nil {
		return err
	}

	// Publish event
	m.publishUnifiedEvent(&CampaignEvent{
		Type:       EventCommissionRemoved,
		CampaignID: campaignID,
		Campaign:   campaign,
		Timestamp:  time.Now(),
		Data: map[string]interface{}{
			"commission_id": commissionID,
		},
	})

	return nil
}

// GetCommissions retrieves commission IDs for a campaign
func (m *unifiedManager) GetCommissions(ctx context.Context, campaignID string) ([]string, error) {
	campaign, err := m.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	return campaign.Commissions, nil
}

// State transition methods...
func (m *unifiedManager) StartPlanning(ctx context.Context, campaignID string) error {
	return m.transitionState(ctx, campaignID, CampaignStatusPlanning, EventCampaignPlanningStarted)
}

func (m *unifiedManager) MarkReady(ctx context.Context, campaignID string) error {
	return m.transitionState(ctx, campaignID, CampaignStatusReady, EventCampaignMarkedReady)
}

func (m *unifiedManager) Start(ctx context.Context, campaignID string) error {
	return m.transitionState(ctx, campaignID, CampaignStatusActive, EventCampaignStarted)
}

func (m *unifiedManager) Pause(ctx context.Context, campaignID string) error {
	return m.transitionState(ctx, campaignID, CampaignStatusPaused, EventCampaignPaused)
}

func (m *unifiedManager) Resume(ctx context.Context, campaignID string) error {
	return m.transitionState(ctx, campaignID, CampaignStatusActive, EventCampaignResumed)
}

func (m *unifiedManager) Complete(ctx context.Context, campaignID string) error {
	return m.transitionState(ctx, campaignID, CampaignStatusCompleted, EventCampaignCompleted)
}

func (m *unifiedManager) Cancel(ctx context.Context, campaignID string) error {
	return m.transitionState(ctx, campaignID, CampaignStatusCancelled, EventCampaignCancelled)
}

// transitionState performs a state transition
func (m *unifiedManager) transitionState(ctx context.Context, campaignID string, newStatus CampaignStatus, eventType string) error {
	campaign, err := m.Get(ctx, campaignID)
	if err != nil {
		return err
	}

	// Validate transition
	if !m.fsm.CanTransition(campaign.Status, newStatus) {
		return gerror.New(gerror.ErrCodeInvalidInput, "invalid state transition", nil).
			WithComponent("unifiedManager").
			WithOperation("transitionState").
			WithDetails("from", string(campaign.Status)).
			WithDetails("to", string(newStatus))
	}

	// Update status
	oldStatus := campaign.Status
	campaign.Status = newStatus

	// Update timestamps
	now := time.Now()
	campaign.UpdatedAt = now
	switch newStatus {
	case CampaignStatusActive:
		if campaign.StartedAt == nil {
			campaign.StartedAt = &now
		}
	case CampaignStatusCompleted, CampaignStatusCancelled:
		campaign.CompletedAt = &now
	}

	// Save
	if err := m.Update(ctx, campaign); err != nil {
		return err
	}

	// Publish event with additional data
	event := &CampaignEvent{
		Type:       eventType,
		CampaignID: campaignID,
		Campaign:   campaign,
		Timestamp:  now,
		Data: map[string]interface{}{
			"old_status": string(oldStatus),
			"new_status": string(newStatus),
		},
	}
	m.publishUnifiedEvent(event)

	return nil
}

// UpdateProgress updates campaign progress based on commission statuses
func (m *unifiedManager) UpdateProgress(ctx context.Context, campaignID string) error {
	campaign, err := m.Get(ctx, campaignID)
	if err != nil {
		return err
	}

	// Calculate progress based on commissions
	if m.commissionMgr != nil && len(campaign.Commissions) > 0 {
		completedCount := 0
		totalWork := 0.0
		completedWork := 0.0

		for _, commissionID := range campaign.Commissions {
			commissionObj, err := m.commissionMgr.GetCommission(ctx, commissionID)
			if err != nil {
				continue
			}

			// The commission manager returns interface{}, we need to handle it properly
			// For now, we'll use a simple progress calculation
			// TODO: Create proper interface for commission progress
			_ = commissionObj // Suppress unused variable warning
			
			// For now, just count commissions
			totalWork += 100.0
			// Assume 50% progress for active commissions
			completedWork += 50.0
		}

		campaign.CompletedCommissions = completedCount
		if totalWork > 0 {
			campaign.Progress = completedWork / totalWork
		}
	}

	// Save updated campaign
	return m.Update(ctx, campaign)
}

// GetProgress retrieves campaign progress
func (m *unifiedManager) GetProgress(ctx context.Context, campaignID string) (*CampaignProgress, error) {
	campaign, err := m.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	return &CampaignProgress{
		CampaignID:           campaign.ID,
		Progress:             campaign.Progress,
		TotalCommissions:     campaign.TotalCommissions,
		CompletedCommissions: campaign.CompletedCommissions,
		UpdatedAt:            campaign.UpdatedAt,
		// TODO: Calculate active and pending commissions
		ActiveCommissions:    0,
		PendingCommissions:   campaign.TotalCommissions - campaign.CompletedCommissions,
	}, nil
}

// Subscribe registers an event handler
func (m *unifiedManager) Subscribe(eventType string, handler EventHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.handlers[eventType] = append(m.handlers[eventType], handler)

	// Also subscribe to unified event bus
	ctx := context.Background()
	subID, err := m.eventBus.Subscribe(ctx, eventType, func(ctx context.Context, event events.CoreEvent) error {
		// Convert unified event to campaign event
		campaignEvent := m.convertFromUnifiedEvent(event)
		if campaignEvent != nil {
			// Call all handlers
			m.mu.RLock()
			handlers := m.handlers[eventType]
			m.mu.RUnlock()

			for _, h := range handlers {
				h(ctx, *campaignEvent)
			}
		}
		return nil
	})

	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to subscribe to unified event bus").
			WithComponent("unifiedManager").
			WithOperation("Subscribe")
	}

	m.subscriptions[eventType] = subID
	return nil
}

// Unsubscribe removes an event handler
func (m *unifiedManager) Unsubscribe(eventType string, handler EventHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	handlers, exists := m.handlers[eventType]
	if !exists {
		return nil
	}

	// Remove handler
	newHandlers := make([]EventHandler, 0, len(handlers))
	for _, h := range handlers {
		if fmt.Sprintf("%p", h) != fmt.Sprintf("%p", handler) {
			newHandlers = append(newHandlers, h)
		}
	}
	m.handlers[eventType] = newHandlers

	// If no more handlers, unsubscribe from unified bus
	if len(newHandlers) == 0 {
		if subID, ok := m.subscriptions[eventType]; ok {
			ctx := context.Background()
			if err := m.eventBus.Unsubscribe(ctx, subID); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unsubscribe from unified event bus").
					WithComponent("unifiedManager").
					WithOperation("Unsubscribe")
			}
			delete(m.subscriptions, eventType)
		}
	}

	return nil
}

// publishUnifiedEvent publishes an event to the unified event bus
func (m *unifiedManager) publishUnifiedEvent(event *CampaignEvent) {
	if m.eventBus != nil {
		ctx := context.Background()
		logger := observability.GetLogger(ctx).
			WithComponent("unifiedManager").
			WithOperation("publishUnifiedEvent")

		// Convert to unified event
		eventData := map[string]interface{}{
			"campaign_id": event.CampaignID,
			"timestamp":   event.Timestamp,
		}
		
		// Check if commission_id is in event data
		if commissionID, ok := event.Data["commission_id"]; ok {
			eventData["commission_id"] = commissionID
		}
		
		unifiedEvent := events.NewBaseEvent(
			uuid.New().String(),
			string(event.Type),
			"campaign-service",
			eventData,
		)

		// Add campaign data if present
		if event.Campaign != nil {
			unifiedEvent.WithData("campaign_name", event.Campaign.Name)
			unifiedEvent.WithData("campaign_status", string(event.Campaign.Status))
			unifiedEvent.WithData("campaign_progress", event.Campaign.Progress)
		}

		// Add additional data
		for k, v := range event.Data {
			unifiedEvent.WithData(k, v)
		}

		// Publish
		if err := m.eventBus.Publish(ctx, unifiedEvent); err != nil {
			logger.WithError(err).Warn("Failed to publish campaign event",
				"event_type", event.Type,
				"campaign_id", event.CampaignID,
			)
		}
	}
}

// convertFromUnifiedEvent converts a unified event to a campaign event
func (m *unifiedManager) convertFromUnifiedEvent(event events.CoreEvent) *CampaignEvent {
	data := event.GetData()
	
	campaignEvent := &CampaignEvent{
		Type:      event.GetType(),
		Timestamp: event.GetTimestamp(),
		Data:      make(map[string]interface{}),
	}

	// Extract campaign ID
	if campaignID, ok := data["campaign_id"].(string); ok {
		campaignEvent.CampaignID = campaignID
	}

	// Extract commission ID and put it in data
	if commissionID, ok := data["commission_id"].(string); ok {
		campaignEvent.Data["commission_id"] = commissionID
	}

	// Copy other data
	for k, v := range data {
		if k != "campaign_id" && k != "commission_id" && k != "timestamp" {
			campaignEvent.Data[k] = v
		}
	}

	return campaignEvent
}

// subscribeToCommissionEvents subscribes to commission events to update campaign progress
func (m *unifiedManager) subscribeToCommissionEvents() {
	if m.eventBus == nil {
		return
	}

	ctx := context.Background()

	// Subscribe to commission completion
	m.eventBus.Subscribe(ctx, orchestrator.EventCommissionCompleted, func(ctx context.Context, e events.CoreEvent) error {
		data := e.GetData()
		commissionID, ok := data["commission_id"].(string)
		if !ok {
			return nil
		}

		// Find campaigns containing this commission
		campaigns, err := m.List(ctx)
		if err != nil {
			return nil
		}

		for _, campaign := range campaigns {
			for _, cid := range campaign.Commissions {
				if cid == commissionID {
					// Update progress
					if err := m.UpdateProgress(ctx, campaign.ID); err == nil {
						// Check if all commissions are complete
						if campaign.CompletedCommissions == campaign.TotalCommissions && campaign.Status == CampaignStatusActive {
							// Auto-complete campaign
							m.Complete(ctx, campaign.ID)
						}
					}
					break
				}
			}
		}
		return nil
	})

	// Subscribe to commission status changes
	m.eventBus.Subscribe(ctx, orchestrator.EventCommissionStatusChanged, func(ctx context.Context, e events.CoreEvent) error {
		data := e.GetData()
		commissionID, ok := data["commission_id"].(string)
		if !ok {
			return nil
		}

		// Find campaigns containing this commission
		campaigns, err := m.List(ctx)
		if err != nil {
			return nil
		}

		for _, campaign := range campaigns {
			for _, cid := range campaign.Commissions {
				if cid == commissionID {
					// Update progress
					m.UpdateProgress(ctx, campaign.ID)
					break
				}
			}
		}
		return nil
	})
}