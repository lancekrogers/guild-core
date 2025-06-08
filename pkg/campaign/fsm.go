package campaign

import (
	"context"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// fsm implements the FSM interface for campaign state transitions
type fsm struct {
	transitions map[CampaignStatus][]CampaignStatus
	mu          sync.RWMutex
}

// NewFSM creates a new campaign state machine
func NewFSM() FSM {
	return &fsm{
		transitions: map[CampaignStatus][]CampaignStatus{
			CampaignStatusDream:     {CampaignStatusPlanning, CampaignStatusCancelled},
			CampaignStatusPlanning:  {CampaignStatusReady, CampaignStatusCancelled},
			CampaignStatusReady:     {CampaignStatusActive, CampaignStatusCancelled},
			CampaignStatusActive:    {CampaignStatusPaused, CampaignStatusCompleted, CampaignStatusCancelled},
			CampaignStatusPaused:    {CampaignStatusActive, CampaignStatusCancelled},
			CampaignStatusCompleted: {}, // Terminal state
			CampaignStatusCancelled: {}, // Terminal state
		},
	}
}

// CanTransition checks if a transition from one status to another is valid
func (f *fsm) CanTransition(from, to CampaignStatus) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	validTransitions, exists := f.transitions[from]
	if !exists {
		return false
	}

	for _, status := range validTransitions {
		if status == to {
			return true
		}
	}
	return false
}

// Transition performs a state transition on a campaign
func (f *fsm) Transition(ctx context.Context, campaign *Campaign, to CampaignStatus) error {
	if campaign == nil {
		return gerror.New(gerror.ErrCodeValidation, "campaign cannot be nil", nil).
			WithComponent("campaign").
			WithOperation("Transition")
	}

	// Check if transition is valid
	if !f.CanTransition(campaign.Status, to) {
		return gerror.New(gerror.ErrCodeValidation, "invalid state transition", nil).
			WithComponent("campaign").
			WithOperation("Transition").
			WithDetails("from_status", string(campaign.Status)).
			WithDetails("to_status", string(to))
	}

	// Store previous status
	previousStatus := campaign.Status

	// Update status
	campaign.Status = to
	campaign.UpdatedAt = time.Now()

	// Handle specific transition side effects
	switch to {
	case CampaignStatusActive:
		if previousStatus == CampaignStatusReady {
			now := time.Now()
			campaign.StartedAt = &now
		}
	case CampaignStatusCompleted:
		now := time.Now()
		campaign.CompletedAt = &now
		campaign.Progress = 1.0
	case CampaignStatusCancelled:
		now := time.Now()
		campaign.CompletedAt = &now
	}

	return nil
}

// GetValidTransitions returns all valid transitions from the given status
func (f *fsm) GetValidTransitions(from CampaignStatus) []CampaignStatus {
	f.mu.RLock()
	defer f.mu.RUnlock()

	transitions, exists := f.transitions[from]
	if !exists {
		return []CampaignStatus{}
	}

	// Return a copy to prevent external modification
	result := make([]CampaignStatus, len(transitions))
	copy(result, transitions)
	return result
}

// ValidateStatus checks if a status is valid
func ValidateStatus(status CampaignStatus) error {
	if !status.IsValid() {
		return gerror.New(gerror.ErrCodeValidation, "invalid campaign status", nil).
			WithComponent("campaign").
			WithOperation("ValidateStatus").
			WithDetails("status", string(status))
	}
	return nil
}
