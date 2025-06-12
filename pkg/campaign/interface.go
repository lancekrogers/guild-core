package campaign

import (
	"context"
)

// Repository defines the interface for campaign storage operations
type Repository interface {
	// Create stores a new campaign
	Create(ctx context.Context, campaign *Campaign) error

	// Get retrieves a campaign by ID
	Get(ctx context.Context, id string) (*Campaign, error)

	// List returns all campaigns
	List(ctx context.Context) ([]*Campaign, error)

	// Update modifies an existing campaign
	Update(ctx context.Context, campaign *Campaign) error

	// Delete removes a campaign
	Delete(ctx context.Context, id string) error

	// GetByCommissionID returns campaigns containing the specified commission
	GetByCommissionID(ctx context.Context, commissionID string) ([]*Campaign, error)
}

// Manager defines the interface for campaign business logic
type Manager interface {
	// Campaign CRUD operations
	Create(ctx context.Context, campaign *Campaign) error
	Get(ctx context.Context, id string) (*Campaign, error)
	List(ctx context.Context) ([]*Campaign, error)
	Update(ctx context.Context, campaign *Campaign) error
	Delete(ctx context.Context, id string) error

	// Commission management
	AddCommission(ctx context.Context, campaignID, commissionID string) error
	RemoveCommission(ctx context.Context, campaignID, commissionID string) error
	GetCommissions(ctx context.Context, campaignID string) ([]string, error)

	// State transitions
	StartPlanning(ctx context.Context, campaignID string) error
	MarkReady(ctx context.Context, campaignID string) error
	Start(ctx context.Context, campaignID string) error
	Pause(ctx context.Context, campaignID string) error
	Resume(ctx context.Context, campaignID string) error
	Complete(ctx context.Context, campaignID string) error
	Cancel(ctx context.Context, campaignID string) error

	// Progress tracking
	UpdateProgress(ctx context.Context, campaignID string) error
	GetProgress(ctx context.Context, campaignID string) (*CampaignProgress, error)

	// Event subscription
	Subscribe(eventType string, handler EventHandler) error
	Unsubscribe(eventType string, handler EventHandler) error
}

// EventHandler is a function that handles campaign events
type EventHandler func(ctx context.Context, event CampaignEvent) error

// FSM defines the interface for campaign state machine
type FSM interface {
	// CanTransition checks if a transition is valid
	CanTransition(from, to CampaignStatus) bool

	// Transition performs a state transition
	Transition(ctx context.Context, campaign *Campaign, to CampaignStatus) error

	// GetValidTransitions returns valid transitions from current state
	GetValidTransitions(from CampaignStatus) []CampaignStatus
}

// ProgressCalculator computes campaign progress based on commissions
type ProgressCalculator interface {
	// Calculate computes progress for a campaign
	Calculate(ctx context.Context, campaign *Campaign) (*CampaignProgress, error)
}
