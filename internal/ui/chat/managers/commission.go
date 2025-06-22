package managers

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// MinimalCommissionManager is a placeholder commission manager for suggestion system
type MinimalCommissionManager struct{}

func (m *MinimalCommissionManager) CreateCommission(ctx context.Context, commission commission.Commission) (*commission.Commission, error) {
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "commission creation not implemented", nil)
}

func (m *MinimalCommissionManager) GetCommission(ctx context.Context, id string) (*commission.Commission, error) {
	return nil, gerror.New(gerror.ErrCodeNotFound, "commission not found", nil)
}

func (m *MinimalCommissionManager) UpdateCommission(ctx context.Context, commission commission.Commission) error {
	return gerror.New(gerror.ErrCodeNotImplemented, "commission update not implemented", nil)
}

func (m *MinimalCommissionManager) DeleteCommission(ctx context.Context, id string) error {
	return gerror.New(gerror.ErrCodeNotImplemented, "commission deletion not implemented", nil)
}

func (m *MinimalCommissionManager) ListCommissions(ctx context.Context) ([]*commission.Commission, error) {
	return []*commission.Commission{}, nil
}

func (m *MinimalCommissionManager) SaveCommission(ctx context.Context, commission *commission.Commission) error {
	return gerror.New(gerror.ErrCodeNotImplemented, "commission save not implemented", nil)
}

func (m *MinimalCommissionManager) LoadCommissionFromFile(ctx context.Context, path string) (*commission.Commission, error) {
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "commission load not implemented", nil)
}

func (m *MinimalCommissionManager) GetCommissionsByTag(ctx context.Context, tag string) ([]*commission.Commission, error) {
	return []*commission.Commission{}, nil
}

func (m *MinimalCommissionManager) SetCommission(ctx context.Context, commissionID string) error {
	return gerror.New(gerror.ErrCodeNotImplemented, "set commission not implemented", nil)
}
