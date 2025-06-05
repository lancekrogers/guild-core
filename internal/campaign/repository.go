package campaign

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/guild-ventures/guild-core/pkg/memory"
)

const (
	campaignBucket = "campaigns"
	campaignPrefix = "campaign:"
)

// repository implements the Repository interface using memory.Store
type repository struct {
	store memory.Store
}

// NewRepository creates a new campaign repository
func NewRepository(store memory.Store) Repository {
	return &repository{
		store: store,
	}
}

// Create stores a new campaign
func (r *repository) Create(ctx context.Context, campaign *Campaign) error {
	if campaign == nil {
		return fmt.Errorf("campaign cannot be nil")
	}

	// Set creation timestamp if not set
	if campaign.CreatedAt.IsZero() {
		campaign.CreatedAt = time.Now()
	}
	campaign.UpdatedAt = time.Now()

	// Generate ID if not set
	if campaign.ID == "" {
		campaign.ID = generateID()
	}

	// Check if campaign already exists
	key := campaignPrefix + campaign.ID
	existing, err := r.store.Get(ctx, campaignBucket, key)
	if err == nil && existing != nil {
		return fmt.Errorf("campaign with ID %s already exists", campaign.ID)
	}

	// Marshal campaign to JSON
	data, err := json.Marshal(campaign)
	if err != nil {
		return fmt.Errorf("failed to marshal campaign: %w", err)
	}

	// Store campaign
	if err := r.store.Put(ctx, campaignBucket, key, data); err != nil {
		return fmt.Errorf("failed to store campaign: %w", err)
	}

	return nil
}

// Get retrieves a campaign by ID
func (r *repository) Get(ctx context.Context, id string) (*Campaign, error) {
	if id == "" {
		return nil, fmt.Errorf("campaign ID cannot be empty")
	}

	key := campaignPrefix + id
	data, err := r.store.Get(ctx, campaignBucket, key)
	if err != nil {
		if err == memory.ErrNotFound {
			return nil, fmt.Errorf("campaign %s not found", id)
		}
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}

	var campaign Campaign
	if err := json.Unmarshal(data, &campaign); err != nil {
		return nil, fmt.Errorf("failed to unmarshal campaign: %w", err)
	}

	return &campaign, nil
}

// List returns all campaigns
func (r *repository) List(ctx context.Context) ([]*Campaign, error) {
	keys, err := r.store.ListKeys(ctx, campaignBucket, campaignPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list campaign keys: %w", err)
	}

	campaigns := make([]*Campaign, 0, len(keys))
	for _, key := range keys {
		data, err := r.store.Get(ctx, campaignBucket, key)
		if err != nil {
			// Skip campaigns that can't be loaded
			continue
		}

		var campaign Campaign
		if err := json.Unmarshal(data, &campaign); err != nil {
			// Skip malformed campaigns
			continue
		}

		campaigns = append(campaigns, &campaign)
	}

	return campaigns, nil
}

// Update modifies an existing campaign
func (r *repository) Update(ctx context.Context, campaign *Campaign) error {
	if campaign == nil {
		return fmt.Errorf("campaign cannot be nil")
	}
	if campaign.ID == "" {
		return fmt.Errorf("campaign ID cannot be empty")
	}

	// Check if campaign exists
	key := campaignPrefix + campaign.ID
	existing, err := r.store.Get(ctx, campaignBucket, key)
	if err != nil {
		if err == memory.ErrNotFound {
			return fmt.Errorf("campaign %s not found", campaign.ID)
		}
		return fmt.Errorf("failed to check campaign existence: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("campaign %s not found", campaign.ID)
	}

	// Update timestamp
	campaign.UpdatedAt = time.Now()

	// Marshal campaign to JSON
	data, err := json.Marshal(campaign)
	if err != nil {
		return fmt.Errorf("failed to marshal campaign: %w", err)
	}

	// Store updated campaign
	if err := r.store.Put(ctx, campaignBucket, key, data); err != nil {
		return fmt.Errorf("failed to update campaign: %w", err)
	}

	return nil
}

// Delete removes a campaign
func (r *repository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("campaign ID cannot be empty")
	}

	key := campaignPrefix + id
	if err := r.store.Delete(ctx, campaignBucket, key); err != nil {
		if err == memory.ErrNotFound {
			return fmt.Errorf("campaign %s not found", id)
		}
		return fmt.Errorf("failed to delete campaign: %w", err)
	}

	return nil
}

// GetByObjectiveID returns campaigns containing the specified objective
func (r *repository) GetByObjectiveID(ctx context.Context, objectiveID string) ([]*Campaign, error) {
	if objectiveID == "" {
		return nil, fmt.Errorf("objective ID cannot be empty")
	}

	// Get all campaigns
	campaigns, err := r.List(ctx)
	if err != nil {
		return nil, err
	}

	// Filter campaigns containing the objective
	var result []*Campaign
	for _, campaign := range campaigns {
		for _, objID := range campaign.Objectives {
			if objID == objectiveID {
				result = append(result, campaign)
				break
			}
		}
	}

	return result, nil
}