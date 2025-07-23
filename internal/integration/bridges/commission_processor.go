// Package bridges provides integration bridges between different components
package bridges

import (
	"context"

	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/orchestrator"
	"github.com/lancekrogers/guild/pkg/registry"
)

// CommissionProcessorBridge provides commission processing functionality to other bridges
type CommissionProcessorBridge struct {
	registry              registry.ComponentRegistry
	commissionIntegration *orchestrator.CommissionIntegrationService
	guildConfig           *config.GuildConfig
	logger                observability.Logger
}

// NewCommissionProcessorBridge creates a new commission processor bridge
func NewCommissionProcessorBridge(
	registry registry.ComponentRegistry,
	logger observability.Logger,
) *CommissionProcessorBridge {
	return &CommissionProcessorBridge{
		registry: registry,
		logger:   logger.WithComponent("CommissionProcessorBridge"),
	}
}

// Initialize sets up the commission integration service
func (b *CommissionProcessorBridge) Initialize(ctx context.Context) error {
	b.logger.InfoContext(ctx, "Initializing commission processor bridge")

	// Get orchestrator registry
	orchRegistry := b.registry.Orchestrator()
	if orchRegistry == nil {
		return gerror.New(gerror.ErrCodeInternal, "orchestrator registry not available", nil).
			WithComponent("CommissionProcessorBridge")
	}

	// Get commission integration service from orchestrator registry
	if orchReg, ok := orchRegistry.(interface{ GetCommissionIntegrationService() *orchestrator.CommissionIntegrationService }); ok {
		b.commissionIntegration = orchReg.GetCommissionIntegrationService()
		if b.commissionIntegration == nil {
			b.logger.WarnContext(ctx, "Commission integration service not available in registry")
		}
	}

	// Load guild config from project
	// Try to get project context from registry
	projectReg := b.registry.Project()
	if projectReg != nil {
		projCtx, err := projectReg.GetCurrentContext(ctx)
		if err == nil && projCtx != nil {
			// Load guild config from project path
			guildConfig, err := config.LoadGuildConfig(ctx, (*projCtx).GetRootPath())
			if err == nil {
				b.guildConfig = guildConfig
			} else {
				b.logger.WithError(err).WarnContext(ctx, "Failed to load guild config from project")
			}
		}
	}

	if b.guildConfig == nil {
		// Create minimal guild config
		b.guildConfig = &config.GuildConfig{
			Agents: []config.AgentConfig{},
		}
		b.logger.WarnContext(ctx, "Guild config not available, using empty config")
	}

	b.logger.InfoContext(ctx, "Commission processor bridge initialized",
		"has_integration_service", b.commissionIntegration != nil,
		"has_guild_config", b.guildConfig != nil)

	return nil
}

// CreateProcessCommissionFunc returns a function that processes commissions
func (b *CommissionProcessorBridge) CreateProcessCommissionFunc() ProcessCommissionFunc {
	return func(ctx context.Context, commissionID string) error {
		return b.ProcessCommission(ctx, commissionID)
	}
}

// ProcessCommission processes a single commission into tasks
func (b *CommissionProcessorBridge) ProcessCommission(ctx context.Context, commissionID string) error {
	b.logger.InfoContext(ctx, "Processing commission", "commission_id", commissionID)

	// Check if we have the integration service
	if b.commissionIntegration == nil {
		b.logger.ErrorContext(ctx, "Commission integration service not available")
		return gerror.New(gerror.ErrCodeInternal, "commission integration service not available", nil).
			WithComponent("CommissionProcessorBridge").
			WithDetails("commission_id", commissionID)
	}

	// Process the commission to tasks
	result, err := b.commissionIntegration.ProcessCommissionToTasksByID(ctx, commissionID, b.guildConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to process commission").
			WithComponent("CommissionProcessorBridge").
			WithDetails("commission_id", commissionID)
	}

	b.logger.InfoContext(ctx, "Commission processed successfully",
		"commission_id", commissionID,
		"task_count", len(result.Tasks),
		"assigned_artisans", len(result.AssignedArtisans))

	return nil
}

// WireCommissionProcessing connects the commission processor to the orchestrator campaign bridge
func WireCommissionProcessing(
	campaignBridge *OrchestratorCampaignBridge,
	processorBridge *CommissionProcessorBridge,
) error {
	if campaignBridge == nil {
		return gerror.New(gerror.ErrCodeValidation, "campaign bridge is nil", nil).
			WithComponent("CommissionProcessorBridge")
	}

	if processorBridge == nil {
		return gerror.New(gerror.ErrCodeValidation, "processor bridge is nil", nil).
			WithComponent("CommissionProcessorBridge")
	}

	// Set the process commission function on the campaign bridge
	campaignBridge.SetProcessCommissionFunc(processorBridge.CreateProcessCommissionFunc())

	return nil
}