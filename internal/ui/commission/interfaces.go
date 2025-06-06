package commission

import (
	"context"
	
	"github.com/guild-ventures/guild-core/pkg/generator"
	commissionpkg "github.com/guild-ventures/guild-core/pkg/commission"
)

// CommissionManager interface abstracts commission management operations
type CommissionManager interface {
	LoadCommissionFromFile(ctx context.Context, path string) (*commissionpkg.Commission, error)
	CreateCommission(ctx context.Context, commission *commissionpkg.Commission) error
	UpdateCommission(ctx context.Context, commission *commissionpkg.Commission) error
	DeleteCommission(ctx context.Context, id string) error
	GetCommission(ctx context.Context, id string) (*commissionpkg.Commission, error)
	ListCommissions(ctx context.Context) ([]*commissionpkg.Commission, error)
	SaveCommissionToFile(ctx context.Context, commission *commissionpkg.Commission, path string) error
}

// CommissionPlanner interface abstracts commission planning operations
type CommissionPlanner interface {
	SetCommission(ctx context.Context, commissionID string) error
	GenerateTasks(ctx context.Context) error
	GetPlanningSummary(ctx context.Context) (string, error)
	ValidatePlan(ctx context.Context) error
	UpdatePlan(ctx context.Context, updates map[string]interface{}) error
	GetCommissionID() string
}

// CommissionGenerator interface abstracts commission document generation
type CommissionGenerator interface {
	GenerateCommissionDocs(ctx context.Context, commission *commissionpkg.Commission) (*generator.GeneratedDocs, error)
	GenerateSpecs(ctx context.Context, commission *commissionpkg.Commission) (string, error)
	GenerateAIDocs(ctx context.Context, commission *commissionpkg.Commission) (string, error)
	SuggestImprovements(ctx context.Context, commission *commissionpkg.Commission) ([]string, error)
}