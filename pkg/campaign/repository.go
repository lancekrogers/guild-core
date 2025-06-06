package campaign

import (
	"context"
	"encoding/json"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
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
		return gerror.New(gerror.ErrCodeInvalidInput, "campaign", nil).WithComponent("create_campaign").WithOperation("campaign cannot be nil")
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
		return gerror.New(gerror.ErrCodeAlreadyExists, "campaign", "create_campaign", "campaign with ID %s already exists", campaign.ID)
	}

	// Marshal campaign to JSON
	data, err := json.Marshal(campaign)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("create_campaign").WithOperation("failed to marshal campaign")
	}

	// Store campaign
	if err := r.store.Put(ctx, campaignBucket, key, data); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("create_campaign").WithOperation("failed to store campaign")
	}

	return nil
}

// Get retrieves a campaign by ID
func (r *repository) Get(ctx context.Context, id string) (*Campaign, error) {
	if id == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "campaign", nil).WithComponent("get_campaign").WithOperation("campaign ID cannot be empty")
	}

	key := campaignPrefix + id
	data, err := r.store.Get(ctx, campaignBucket, key)
	if err != nil {
		if err == memory.ErrNotFound {
			return nil, gerror.New(gerror.ErrCodeNotFound, "campaign", "get_campaign", "campaign %s not found", id)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("get_campaign").WithOperation("failed to get campaign")
	}

	var campaign Campaign
	if err := json.Unmarshal(data, &campaign); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("get_campaign").WithOperation("failed to unmarshal campaign")
	}

	return &campaign, nil
}

// List returns all campaigns
func (r *repository) List(ctx context.Context) ([]*Campaign, error) {
	keys, err := r.store.ListKeys(ctx, campaignBucket, campaignPrefix)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("list_campaigns").WithOperation("failed to list campaign keys")
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
		return gerror.New(gerror.ErrCodeInvalidInput, "campaign", nil).WithComponent("update_campaign").WithOperation("campaign cannot be nil")
	}
	if campaign.ID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "campaign", nil).WithComponent("update_campaign").WithOperation("campaign ID cannot be empty")
	}

	// Check if campaign exists
	key := campaignPrefix + campaign.ID
	existing, err := r.store.Get(ctx, campaignBucket, key)
	if err != nil {
		if err == memory.ErrNotFound {
			return gerror.New(gerror.ErrCodeNotFound, "campaign", "update_campaign", "campaign %s not found", campaign.ID)
		}
		return gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("update_campaign").WithOperation("failed to check campaign existence")
	}
	if existing == nil {
		return gerror.New(gerror.ErrCodeNotFound, "campaign", "update_campaign", "campaign %s not found", campaign.ID)
	}

	// Update timestamp
	campaign.UpdatedAt = time.Now()

	// Marshal campaign to JSON
	data, err := json.Marshal(campaign)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("create_campaign").WithOperation("failed to marshal campaign")
	}

	// Store updated campaign
	if err := r.store.Put(ctx, campaignBucket, key, data); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("update_campaign").WithOperation("failed to update campaign")
	}

	return nil
}

// Delete removes a campaign
func (r *repository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "campaign", nil).WithComponent("delete_campaign").WithOperation("campaign ID cannot be empty")
	}

	key := campaignPrefix + id
	if err := r.store.Delete(ctx, campaignBucket, key); err != nil {
		if err == memory.ErrNotFound {
			return gerror.New(gerror.ErrCodeNotFound, "campaign", "delete_campaign", "campaign %s not found", id)
		}
		return gerror.Wrap(err, gerror.ErrCodeInternal, "campaign").WithComponent("delete_campaign").WithOperation("failed to delete campaign")
	}

	return nil
}

// GetByObjectiveID returns campaigns containing the specified objective
func (r *repository) GetByObjectiveID(ctx context.Context, objectiveID string) ([]*Campaign, error) {
	if objectiveID == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "campaign", nil).WithComponent("get_by_objective_id").WithOperation("objective ID cannot be empty")
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