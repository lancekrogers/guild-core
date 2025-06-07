package commission

import (
	"context"
)

// CommissionManager defines the interface for commission management operations
type CommissionManager interface {
	// CreateCommission creates a new commission
	CreateCommission(ctx context.Context, commission Commission) (*Commission, error)

	// GetCommission retrieves a commission by ID
	GetCommission(ctx context.Context, id string) (*Commission, error)

	// UpdateCommission updates an existing commission
	UpdateCommission(ctx context.Context, commission Commission) error

	// DeleteCommission removes a commission
	DeleteCommission(ctx context.Context, id string) error

	// ListCommissions returns all commissions matching the filter
	ListCommissions(ctx context.Context) ([]*Commission, error)

	// SaveCommission persists a commission to storage
	SaveCommission(ctx context.Context, commission *Commission) error

	// LoadCommissionFromFile loads a commission from a file path
	LoadCommissionFromFile(ctx context.Context, path string) (*Commission, error)

	// GetCommissionsByTag retrieves commissions with a specific tag
	GetCommissionsByTag(ctx context.Context, tag string) ([]*Commission, error)

	// SetCommission updates the current active commission
	SetCommission(ctx context.Context, commissionID string) error
}
