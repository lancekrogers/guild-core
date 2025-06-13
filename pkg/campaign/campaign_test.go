package campaign

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampaignStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   CampaignStatus
		terminal bool
		valid    bool
	}{
		{
			name:     "dream status",
			status:   CampaignStatusDream,
			terminal: false,
			valid:    true,
		},
		{
			name:     "planning status",
			status:   CampaignStatusPlanning,
			terminal: false,
			valid:    true,
		},
		{
			name:     "ready status",
			status:   CampaignStatusReady,
			terminal: false,
			valid:    true,
		},
		{
			name:     "active status",
			status:   CampaignStatusActive,
			terminal: false,
			valid:    true,
		},
		{
			name:     "paused status",
			status:   CampaignStatusPaused,
			terminal: false,
			valid:    true,
		},
		{
			name:     "completed status",
			status:   CampaignStatusCompleted,
			terminal: true,
			valid:    true,
		},
		{
			name:     "cancelled status",
			status:   CampaignStatusCancelled,
			terminal: true,
			valid:    true,
		},
		{
			name:     "invalid status",
			status:   CampaignStatus("invalid"),
			terminal: false,
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.terminal, tt.status.IsTerminal())
			assert.Equal(t, tt.valid, tt.status.IsValid())
		})
	}
}

func TestNewCampaign(t *testing.T) {
	name := "Test Campaign"
	description := "Test Description"

	campaign := NewCampaign(name, description)

	assert.NotEmpty(t, campaign.ID)
	assert.Equal(t, name, campaign.Name)
	assert.Equal(t, description, campaign.Description)
	assert.Equal(t, CampaignStatusDream, campaign.Status) // Now starts in dream status
	assert.Empty(t, campaign.Commissions)
	assert.Empty(t, campaign.Tags)
	assert.NotNil(t, campaign.Metadata)
	assert.False(t, campaign.CreatedAt.IsZero())
	assert.False(t, campaign.UpdatedAt.IsZero())
	assert.Nil(t, campaign.StartedAt)
	assert.Nil(t, campaign.CompletedAt)
	assert.Equal(t, 0.0, campaign.Progress)
}

func TestCampaignEvent(t *testing.T) {
	campaign := NewCampaign("Test", "Test Campaign")
	event := CampaignEvent{
		ID:         generateID(),
		Type:       EventCampaignCreated,
		CampaignID: campaign.ID,
		Campaign:   campaign,
		Timestamp:  time.Now(),
		Data: map[string]interface{}{
			"test": "data",
		},
	}

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, EventCampaignCreated, event.Type)
	assert.Equal(t, campaign.ID, event.CampaignID)
	assert.Equal(t, campaign, event.Campaign)
	assert.False(t, event.Timestamp.IsZero())
	assert.Equal(t, "data", event.Data["test"])
}

func TestCampaignProgress(t *testing.T) {
	progress := &CampaignProgress{
		CampaignID:          "test-123",
		TotalCommissions:     10,
		CompletedCommissions: 3,
		ActiveCommissions:    2,
		PendingCommissions:   5,
		Progress:            0.3,
		UpdatedAt:           time.Now(),
	}

	assert.Equal(t, "test-123", progress.CampaignID)
	assert.Equal(t, 10, progress.TotalCommissions)
	assert.Equal(t, 3, progress.CompletedCommissions)
	assert.Equal(t, 2, progress.ActiveCommissions)
	assert.Equal(t, 5, progress.PendingCommissions)
	assert.Equal(t, 0.3, progress.Progress)
	assert.False(t, progress.UpdatedAt.IsZero())
}

// Mock implementation of memory.Store for testing
type mockStore struct {
	data map[string]map[string][]byte
}

func newMockStore() *mockStore {
	return &mockStore{
		data: make(map[string]map[string][]byte),
	}
}

func (m *mockStore) Put(ctx context.Context, bucket, key string, value []byte) error {
	if m.data[bucket] == nil {
		m.data[bucket] = make(map[string][]byte)
	}
	m.data[bucket][key] = value
	return nil
}

func (m *mockStore) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	if m.data[bucket] == nil || m.data[bucket][key] == nil {
		return nil, fmt.Errorf("not found")
	}
	return m.data[bucket][key], nil
}

func (m *mockStore) Delete(ctx context.Context, bucket, key string) error {
	if m.data[bucket] != nil {
		delete(m.data[bucket], key)
	}
	return nil
}

func (m *mockStore) List(ctx context.Context, bucket string) ([]string, error) {
	if m.data[bucket] == nil {
		return []string{}, nil
	}
	keys := make([]string, 0, len(m.data[bucket]))
	for k := range m.data[bucket] {
		keys = append(keys, k)
	}
	return keys, nil
}

func (m *mockStore) ListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	if m.data[bucket] == nil {
		return []string{}, nil
	}
	keys := make([]string, 0)
	for k := range m.data[bucket] {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (m *mockStore) Close() error {
	return nil
}

func TestRepository(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	repo := NewRepository(store)

	t.Run("Create", func(t *testing.T) {
		campaign := NewCampaign("Test Campaign", "Description")
		err := repo.Create(ctx, campaign)
		require.NoError(t, err)
		assert.NotEmpty(t, campaign.ID)
	})

	t.Run("Get", func(t *testing.T) {
		campaign := NewCampaign("Test Campaign", "Description")
		err := repo.Create(ctx, campaign)
		require.NoError(t, err)

		retrieved, err := repo.Get(ctx, campaign.ID)
		require.NoError(t, err)
		assert.Equal(t, campaign.ID, retrieved.ID)
		assert.Equal(t, campaign.Name, retrieved.Name)
	})

	t.Run("List", func(t *testing.T) {
		// Create multiple campaigns
		for i := 0; i < 3; i++ {
			campaign := NewCampaign(fmt.Sprintf("Campaign %d", i), "Description")
			err := repo.Create(ctx, campaign)
			require.NoError(t, err)
		}

		campaigns, err := repo.List(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(campaigns), 3)
	})

	t.Run("Update", func(t *testing.T) {
		campaign := NewCampaign("Test Campaign", "Description")
		err := repo.Create(ctx, campaign)
		require.NoError(t, err)

		campaign.Name = "Updated Campaign"
		campaign.Description = "Updated Description"
		err = repo.Update(ctx, campaign)
		require.NoError(t, err)

		retrieved, err := repo.Get(ctx, campaign.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Campaign", retrieved.Name)
		assert.Equal(t, "Updated Description", retrieved.Description)
	})

	t.Run("Delete", func(t *testing.T) {
		campaign := NewCampaign("Test Campaign", "Description")
		err := repo.Create(ctx, campaign)
		require.NoError(t, err)

		err = repo.Delete(ctx, campaign.ID)
		require.NoError(t, err)

		_, err = repo.Get(ctx, campaign.ID)
		assert.Error(t, err)
	})

	t.Run("GetByObjectiveID", func(t *testing.T) {
		// Create campaigns with objectives
		campaign1 := NewCampaign("Campaign 1", "Description")
		campaign1.Commissions = []string{"obj1", "obj2"}
		err := repo.Create(ctx, campaign1)
		require.NoError(t, err)

		campaign2 := NewCampaign("Campaign 2", "Description")
		campaign2.Commissions = []string{"obj2", "obj3"}
		err = repo.Create(ctx, campaign2)
		require.NoError(t, err)

		// Find campaigns with obj2
		campaigns, err := repo.GetByCommissionID(ctx, "obj2")
		require.NoError(t, err)
		assert.Len(t, campaigns, 2)
	})
}
