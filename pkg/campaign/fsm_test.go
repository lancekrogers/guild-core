package campaign

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFSMCanTransition(t *testing.T) {
	fsm := NewFSM()

	tests := []struct {
		name     string
		from     CampaignStatus
		to       CampaignStatus
		expected bool
	}{
		// Dream transitions
		{
			name:     "dream to planning",
			from:     CampaignStatusDream,
			to:       CampaignStatusPlanning,
			expected: true,
		},
		{
			name:     "dream to cancelled",
			from:     CampaignStatusDream,
			to:       CampaignStatusCancelled,
			expected: true,
		},
		{
			name:     "dream to ready",
			from:     CampaignStatusDream,
			to:       CampaignStatusReady,
			expected: false,
		},
		// Planning transitions
		{
			name:     "planning to ready",
			from:     CampaignStatusPlanning,
			to:       CampaignStatusReady,
			expected: true,
		},
		{
			name:     "planning to cancelled",
			from:     CampaignStatusPlanning,
			to:       CampaignStatusCancelled,
			expected: true,
		},
		{
			name:     "planning to active",
			from:     CampaignStatusPlanning,
			to:       CampaignStatusActive,
			expected: false,
		},
		// Ready transitions
		{
			name:     "ready to active",
			from:     CampaignStatusReady,
			to:       CampaignStatusActive,
			expected: true,
		},
		{
			name:     "ready to cancelled",
			from:     CampaignStatusReady,
			to:       CampaignStatusCancelled,
			expected: true,
		},
		{
			name:     "ready to completed",
			from:     CampaignStatusReady,
			to:       CampaignStatusCompleted,
			expected: false,
		},
		// Active transitions
		{
			name:     "active to paused",
			from:     CampaignStatusActive,
			to:       CampaignStatusPaused,
			expected: true,
		},
		{
			name:     "active to completed",
			from:     CampaignStatusActive,
			to:       CampaignStatusCompleted,
			expected: true,
		},
		{
			name:     "active to cancelled",
			from:     CampaignStatusActive,
			to:       CampaignStatusCancelled,
			expected: true,
		},
		{
			name:     "active to planning",
			from:     CampaignStatusActive,
			to:       CampaignStatusPlanning,
			expected: false,
		},
		// Paused transitions
		{
			name:     "paused to active",
			from:     CampaignStatusPaused,
			to:       CampaignStatusActive,
			expected: true,
		},
		{
			name:     "paused to cancelled",
			from:     CampaignStatusPaused,
			to:       CampaignStatusCancelled,
			expected: true,
		},
		{
			name:     "paused to completed",
			from:     CampaignStatusPaused,
			to:       CampaignStatusCompleted,
			expected: false,
		},
		// Terminal state transitions
		{
			name:     "completed to any",
			from:     CampaignStatusCompleted,
			to:       CampaignStatusActive,
			expected: false,
		},
		{
			name:     "cancelled to any",
			from:     CampaignStatusCancelled,
			to:       CampaignStatusActive,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fsm.CanTransition(tt.from, tt.to)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFSMTransition(t *testing.T) {
	ctx := context.Background()
	fsm := NewFSM()

	t.Run("valid transition", func(t *testing.T) {
		campaign := NewCampaign("Test", "Test Campaign")
		campaign.Status = CampaignStatusReady

		err := fsm.Transition(ctx, campaign, CampaignStatusActive)
		require.NoError(t, err)
		assert.Equal(t, CampaignStatusActive, campaign.Status)
		assert.NotNil(t, campaign.StartedAt)
		assert.False(t, campaign.UpdatedAt.IsZero())
	})

	t.Run("invalid transition", func(t *testing.T) {
		campaign := NewCampaign("Test", "Test Campaign")
		campaign.Status = CampaignStatusPlanning

		err := fsm.Transition(ctx, campaign, CampaignStatusCompleted)
		assert.Error(t, err)
		assert.Equal(t, CampaignStatusPlanning, campaign.Status)
	})

	t.Run("nil campaign", func(t *testing.T) {
		err := fsm.Transition(ctx, nil, CampaignStatusActive)
		assert.Error(t, err)
	})

	t.Run("transition to completed", func(t *testing.T) {
		campaign := NewCampaign("Test", "Test Campaign")
		campaign.Status = CampaignStatusActive

		err := fsm.Transition(ctx, campaign, CampaignStatusCompleted)
		require.NoError(t, err)
		assert.Equal(t, CampaignStatusCompleted, campaign.Status)
		assert.NotNil(t, campaign.CompletedAt)
		assert.Equal(t, 1.0, campaign.Progress)
	})

	t.Run("transition to cancelled", func(t *testing.T) {
		campaign := NewCampaign("Test", "Test Campaign")
		campaign.Status = CampaignStatusActive

		err := fsm.Transition(ctx, campaign, CampaignStatusCancelled)
		require.NoError(t, err)
		assert.Equal(t, CampaignStatusCancelled, campaign.Status)
		assert.NotNil(t, campaign.CompletedAt)
	})

	t.Run("resume from paused", func(t *testing.T) {
		campaign := NewCampaign("Test", "Test Campaign")
		campaign.Status = CampaignStatusActive

		// First pause
		err := fsm.Transition(ctx, campaign, CampaignStatusPaused)
		require.NoError(t, err)

		// Then resume
		err = fsm.Transition(ctx, campaign, CampaignStatusActive)
		require.NoError(t, err)
		assert.Equal(t, CampaignStatusActive, campaign.Status)
	})
}

func TestFSMGetValidTransitions(t *testing.T) {
	fsm := NewFSM()

	tests := []struct {
		name     string
		from     CampaignStatus
		expected []CampaignStatus
	}{
		{
			name:     "from dream",
			from:     CampaignStatusDream,
			expected: []CampaignStatus{CampaignStatusPlanning, CampaignStatusCancelled},
		},
		{
			name:     "from planning",
			from:     CampaignStatusPlanning,
			expected: []CampaignStatus{CampaignStatusReady, CampaignStatusCancelled},
		},
		{
			name:     "from ready",
			from:     CampaignStatusReady,
			expected: []CampaignStatus{CampaignStatusActive, CampaignStatusCancelled},
		},
		{
			name:     "from active",
			from:     CampaignStatusActive,
			expected: []CampaignStatus{CampaignStatusPaused, CampaignStatusCompleted, CampaignStatusCancelled},
		},
		{
			name:     "from paused",
			from:     CampaignStatusPaused,
			expected: []CampaignStatus{CampaignStatusActive, CampaignStatusCancelled},
		},
		{
			name:     "from completed",
			from:     CampaignStatusCompleted,
			expected: []CampaignStatus{},
		},
		{
			name:     "from cancelled",
			from:     CampaignStatusCancelled,
			expected: []CampaignStatus{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transitions := fsm.GetValidTransitions(tt.from)
			assert.ElementsMatch(t, tt.expected, transitions)
		})
	}
}

func TestValidateStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  CampaignStatus
		wantErr bool
	}{
		{
			name:    "valid planning",
			status:  CampaignStatusPlanning,
			wantErr: false,
		},
		{
			name:    "valid active",
			status:  CampaignStatusActive,
			wantErr: false,
		},
		{
			name:    "valid paused",
			status:  CampaignStatusPaused,
			wantErr: false,
		},
		{
			name:    "valid completed",
			status:  CampaignStatusCompleted,
			wantErr: false,
		},
		{
			name:    "valid cancelled",
			status:  CampaignStatusCancelled,
			wantErr: false,
		},
		{
			name:    "invalid status",
			status:  CampaignStatus("invalid"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStatus(tt.status)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
