// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package campaign

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/orchestrator"
)

// manager implements the Manager interface
type manager struct {
	repo          Repository
	commissionMgr *commission.Manager
	fsm           FSM
	eventBus      orchestrator.EventBus
	handlers      map[string][]EventHandler
	mu            sync.RWMutex
}

// NewManager creates a new campaign manager
func NewManager(repo Repository, commissionMgr *commission.Manager, eventBus orchestrator.EventBus) Manager {
	mgr := &manager{
		repo:          repo,
		commissionMgr: commissionMgr,
		fsm:           NewFSM(),
		eventBus:      eventBus,
		handlers:      make(map[string][]EventHandler),
	}

	// Subscribe to commission events
	mgr.subscribeToCommissionEvents()

	return mgr
}

// Create creates a new campaign
func (m *manager) Create(ctx context.Context, campaign *Campaign) error {
	if campaign == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "campaign", nil).WithComponent("create_campaign").WithOperation("campaign cannot be nil")
	}

	// Set default status if not set
	if campaign.Status == "" {
		campaign.Status = CampaignStatusPlanning
	}

	// Validate status
	if err := ValidateStatus(campaign.Status); err != nil {
		return err
	}

	// Create campaign in repository
	if err := m.repo.Create(ctx, campaign); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("create_campaign").WithOperation("failed to create campaign")
	}

	// Publish created event
	m.publishEvent(ctx, CampaignEvent{
		ID:         generateID(),
		Type:       EventCampaignCreated,
		CampaignID: campaign.ID,
		Campaign:   campaign,
		Timestamp:  time.Now(),
	})

	return nil
}

// Get retrieves a campaign by ID
func (m *manager) Get(ctx context.Context, id string) (*Campaign, error) {
	return m.repo.Get(ctx, id)
}

// List returns all campaigns
func (m *manager) List(ctx context.Context) ([]*Campaign, error) {
	return m.repo.List(ctx)
}

// Update modifies an existing campaign
func (m *manager) Update(ctx context.Context, campaign *Campaign) error {
	if campaign == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "campaign", nil).WithComponent("create_campaign").WithOperation("campaign cannot be nil")
	}

	// Get existing campaign to check for changes
	existing, err := m.repo.Get(ctx, campaign.ID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("update_campaign").WithOperation("failed to get existing campaign")
	}

	// Update progress if commissions changed
	if len(campaign.Commissions) != len(existing.Commissions) {
		if err := m.UpdateProgress(ctx, campaign.ID); err != nil {
			// Log error but don't fail the update
			fmt.Printf("Warning: failed to update progress: %v\n", err)
		}
	}

	// Update campaign in repository
	if err := m.repo.Update(ctx, campaign); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("update_campaign").WithOperation("failed to update campaign")
	}

	return nil
}

// Delete removes a campaign
func (m *manager) Delete(ctx context.Context, id string) error {
	// Get campaign first to check status
	campaign, err := m.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	// Don't delete active campaigns
	if campaign.Status == CampaignStatusActive {
		return gerror.New(gerror.ErrCodeInvalidInput, "campaign", nil).WithComponent("delete_campaign").WithOperation("cannot delete active campaign")
	}

	return m.repo.Delete(ctx, id)
}

// AddCommission adds a commission to a campaign
func (m *manager) AddCommission(ctx context.Context, campaignID, commissionID string) error {
	// Get campaign
	campaign, err := m.repo.Get(ctx, campaignID)
	if err != nil {
		return err
	}

	// Check if commission exists
	if m.commissionMgr != nil {
		if _, err := m.commissionMgr.GetCommission(ctx, commissionID); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeNotFound, fmt.Sprintf("commission %s not found", commissionID))
		}
	}

	// Check if commission already in campaign
	for _, id := range campaign.Commissions {
		if id == commissionID {
			return gerror.New(gerror.ErrCodeAlreadyExists, fmt.Sprintf("commission %s already in campaign", commissionID), nil)
		}
	}

	// Add commission
	campaign.Commissions = append(campaign.Commissions, commissionID)
	campaign.TotalCommissions = len(campaign.Commissions)

	// Update campaign
	if err := m.repo.Update(ctx, campaign); err != nil {
		return err
	}

	// Update progress
	if err := m.UpdateProgress(ctx, campaignID); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to update progress: %v\n", err)
	}

	// Publish event
	m.publishEvent(ctx, CampaignEvent{
		ID:         generateID(),
		Type:       EventCommissionAdded,
		CampaignID: campaignID,
		Timestamp:  time.Now(),
		Data: map[string]interface{}{
			"commission_id": commissionID,
		},
	})

	return nil
}

// RemoveCommission removes a commission from a campaign
func (m *manager) RemoveCommission(ctx context.Context, campaignID, commissionID string) error {
	// Get campaign
	campaign, err := m.repo.Get(ctx, campaignID)
	if err != nil {
		return err
	}

	// Remove commission
	newCommissions := make([]string, 0, len(campaign.Commissions))
	found := false
	for _, id := range campaign.Commissions {
		if id != commissionID {
			newCommissions = append(newCommissions, id)
		} else {
			found = true
		}
	}

	if !found {
		return gerror.New(gerror.ErrCodeNotFound, fmt.Sprintf("commission %s not found", commissionID), nil)
	}

	campaign.Commissions = newCommissions
	campaign.TotalCommissions = len(campaign.Commissions)

	// Update campaign
	if err := m.repo.Update(ctx, campaign); err != nil {
		return err
	}

	// Update progress
	if err := m.UpdateProgress(ctx, campaignID); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to update progress: %v\n", err)
	}

	// Publish event
	m.publishEvent(ctx, CampaignEvent{
		ID:         generateID(),
		Type:       EventCommissionRemoved,
		CampaignID: campaignID,
		Timestamp:  time.Now(),
		Data: map[string]interface{}{
			"commission_id": commissionID,
		},
	})

	return nil
}

// GetCommissions returns all commission IDs for a campaign
func (m *manager) GetCommissions(ctx context.Context, campaignID string) ([]string, error) {
	campaign, err := m.repo.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	return campaign.Commissions, nil
}

// StartPlanning transitions a campaign from dream to planning status
func (m *manager) StartPlanning(ctx context.Context, campaignID string) error {
	return m.transitionStatus(ctx, campaignID, CampaignStatusPlanning)
}

// MarkReady transitions a campaign from planning to ready status
func (m *manager) MarkReady(ctx context.Context, campaignID string) error {
	return m.transitionStatus(ctx, campaignID, CampaignStatusReady)
}

// Start transitions a campaign to active status
func (m *manager) Start(ctx context.Context, campaignID string) error {
	return m.transitionStatus(ctx, campaignID, CampaignStatusActive)
}

// Pause transitions a campaign to paused status
func (m *manager) Pause(ctx context.Context, campaignID string) error {
	return m.transitionStatus(ctx, campaignID, CampaignStatusPaused)
}

// Resume transitions a campaign back to active status
func (m *manager) Resume(ctx context.Context, campaignID string) error {
	return m.transitionStatus(ctx, campaignID, CampaignStatusActive)
}

// Complete transitions a campaign to completed status
func (m *manager) Complete(ctx context.Context, campaignID string) error {
	return m.transitionStatus(ctx, campaignID, CampaignStatusCompleted)
}

// Cancel transitions a campaign to cancelled status
func (m *manager) Cancel(ctx context.Context, campaignID string) error {
	return m.transitionStatus(ctx, campaignID, CampaignStatusCancelled)
}

// transitionStatus performs a state transition
func (m *manager) transitionStatus(ctx context.Context, campaignID string, newStatus CampaignStatus) error {
	// Get campaign
	campaign, err := m.repo.Get(ctx, campaignID)
	if err != nil {
		return err
	}

	// Use FSM to transition
	if err := m.fsm.Transition(ctx, campaign, newStatus); err != nil {
		return err
	}

	// Update campaign
	if err := m.repo.Update(ctx, campaign); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("transition_state").WithOperation("failed to update campaign after transition")
	}

	// Publish appropriate event
	eventType := ""
	switch newStatus {
	case CampaignStatusPlanning:
		eventType = EventCampaignPlanningStarted
	case CampaignStatusReady:
		eventType = EventCampaignMarkedReady
	case CampaignStatusActive:
		if campaign.StartedAt != nil && campaign.StartedAt.Equal(campaign.UpdatedAt) {
			eventType = EventCampaignStarted
		} else {
			eventType = EventCampaignResumed
		}
	case CampaignStatusPaused:
		eventType = EventCampaignPaused
	case CampaignStatusCompleted:
		eventType = EventCampaignCompleted
	case CampaignStatusCancelled:
		eventType = EventCampaignCancelled
	}

	if eventType != "" {
		m.publishEvent(ctx, CampaignEvent{
			ID:         generateID(),
			Type:       eventType,
			CampaignID: campaignID,
			Campaign:   campaign,
			Timestamp:  time.Now(),
		})
	}

	return nil
}

// UpdateProgress updates the progress of a campaign
func (m *manager) UpdateProgress(ctx context.Context, campaignID string) error {
	campaign, err := m.repo.Get(ctx, campaignID)
	if err != nil {
		return err
	}

	if m.commissionMgr == nil {
		// Can't calculate progress without commission manager
		return nil
	}

	completedCount := 0
	for _, commID := range campaign.Commissions {
		comm, err := m.commissionMgr.GetCommission(ctx, commID)
		if err != nil {
			continue
		}
		if comm.Status == commission.CommissionStatusCompleted {
			completedCount++
		}
	}

	campaign.CompletedCommissions = completedCount
	campaign.TotalCommissions = len(campaign.Commissions)
	if campaign.TotalCommissions > 0 {
		campaign.Progress = float64(completedCount) / float64(campaign.TotalCommissions)
	} else {
		campaign.Progress = 0.0
	}

	// Update campaign
	if err := m.repo.Update(ctx, campaign); err != nil {
		return err
	}

	// Publish progress event
	m.publishEvent(ctx, CampaignEvent{
		ID:         generateID(),
		Type:       EventCampaignProgress,
		CampaignID: campaignID,
		Timestamp:  time.Now(),
		Data: map[string]interface{}{
			"progress":              campaign.Progress,
			"completed_commissions": campaign.CompletedCommissions,
			"total_commissions":     campaign.TotalCommissions,
		},
	})

	// Check if campaign should be auto-completed
	if campaign.CompletedCommissions == campaign.TotalCommissions &&
		campaign.TotalCommissions > 0 &&
		campaign.Status == CampaignStatusActive {
		// Auto-complete campaign
		_ = m.Complete(ctx, campaignID)
	}

	return nil
}

// GetProgress returns the progress of a campaign
func (m *manager) GetProgress(ctx context.Context, campaignID string) (*CampaignProgress, error) {
	campaign, err := m.repo.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	// Count commission statuses
	activeCount := 0
	pendingCount := 0
	completedCount := 0

	if m.commissionMgr != nil {
		for _, commID := range campaign.Commissions {
			comm, err := m.commissionMgr.GetCommission(ctx, commID)
			if err != nil {
				pendingCount++
				continue
			}

			switch comm.Status {
			case commission.CommissionStatusActive:
				activeCount++
			case commission.CommissionStatusCompleted:
				completedCount++
			default:
				pendingCount++
			}
		}
	} else {
		// If no commission manager, use campaign's stored values
		completedCount = campaign.CompletedCommissions
		pendingCount = campaign.TotalCommissions - completedCount
	}

	return &CampaignProgress{
		CampaignID:           campaign.ID,
		TotalCommissions:     campaign.TotalCommissions,
		CompletedCommissions: completedCount,
		ActiveCommissions:    activeCount,
		PendingCommissions:   pendingCount,
		Progress:             campaign.Progress,
		UpdatedAt:            time.Now(),
	}, nil
}

// Subscribe registers an event handler
func (m *manager) Subscribe(eventType string, handler EventHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.handlers[eventType] = append(m.handlers[eventType], handler)
	return nil
}

// Unsubscribe removes an event handler
func (m *manager) Unsubscribe(eventType string, handler EventHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	handlers := m.handlers[eventType]
	for i, h := range handlers {
		// Compare function pointers
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			m.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
	return nil
}

// publishEvent publishes an event to handlers and event bus
func (m *manager) publishEvent(ctx context.Context, event CampaignEvent) {
	// Notify local handlers
	m.mu.RLock()
	handlers := append([]EventHandler{}, m.handlers[event.Type]...)
	wildcardHandlers := append([]EventHandler{}, m.handlers["*"]...)
	m.mu.RUnlock()

	// Call type-specific handlers
	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h(ctx, event); err != nil {
				fmt.Printf("Error in campaign event handler: %v\n", err)
			}
		}(handler)
	}

	// Call wildcard handlers
	for _, handler := range wildcardHandlers {
		go func(h EventHandler) {
			if err := h(ctx, event); err != nil {
				fmt.Printf("Error in campaign wildcard handler: %v\n", err)
			}
		}(handler)
	}

	// Publish to orchestrator event bus if available
	if m.eventBus != nil {
		orchEvent := orchestrator.Event{
			ID:        event.ID,
			Type:      orchestrator.EventType(event.Type),
			Timestamp: event.Timestamp,
			Source:    "campaign_manager",
			Data: map[string]interface{}{
				"campaign_id": event.CampaignID,
				"campaign":    event.Campaign,
			},
		}
		for k, v := range event.Data {
			orchEvent.Data[k] = v
		}
		m.eventBus.Publish(orchEvent)
	}
}

// subscribeToCommissionEvents subscribes to commission events to update campaign progress
func (m *manager) subscribeToCommissionEvents() {
	if m.eventBus == nil {
		return
	}

	// Subscribe to commission completion
	m.eventBus.Subscribe(orchestrator.EventCommissionCompleted, func(e orchestrator.Event) {
		commissionID, ok := e.Data["commission_id"].(string)
		if !ok {
			return
		}

		// Find campaigns containing this commission
		ctx := context.Background()
		campaigns, err := m.repo.GetByCommissionID(ctx, commissionID)
		if err != nil {
			return
		}

		// Update progress for each campaign
		for _, campaign := range campaigns {
			_ = m.UpdateProgress(ctx, campaign.ID)
		}
	})

	// Subscribe to commission status changes
	m.eventBus.Subscribe(orchestrator.EventCommissionStatusChanged, func(e orchestrator.Event) {
		commissionID, ok := e.Data["commission_id"].(string)
		if !ok {
			return
		}

		// Find campaigns containing this commission
		ctx := context.Background()
		campaigns, err := m.repo.GetByCommissionID(ctx, commissionID)
		if err != nil {
			return
		}

		// Update progress for each campaign
		for _, campaign := range campaigns {
			_ = m.UpdateProgress(ctx, campaign.ID)
		}
	})
}
