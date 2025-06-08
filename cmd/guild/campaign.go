package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/pkg/campaign"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/orchestrator"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

var (
	objectivePath string
	campaignName  string
	managerID     string
	campaignID    string
)

// campaignCmd represents the campaign command group
var campaignCmd = &cobra.Command{
	Use:   "campaign",
	Short: "Manage and execute campaigns",
	Long: `A campaign coordinates agents to work on an objective.

Campaigns decompose objectives into tasks, assign them to agents,
and orchestrate execution based on agent capabilities.`,
}

// createCampaignCmd creates a new campaign from an objective
var createCampaignCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new campaign from an objective",
	Long: `Create a campaign that will coordinate agents to complete an objective.

The manager agent will decompose the objective into tasks and assign
them to appropriate agents based on their capabilities.`,
	RunE: createCampaign,
}

// startCampaignCmd starts campaign execution
var startCampaignCmd = &cobra.Command{
	Use:   "start",
	Short: "Start campaign execution",
	RunE:  startCampaign,
}

// watchCampaignCmd watches campaign progress
var watchCampaignCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch campaign progress in real-time",
	RunE:  watchCampaign,
}

// listCampaignsCmd lists all campaigns
var listCampaignsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all campaigns",
	RunE:  listCampaigns,
}

// campaignStatusCmd shows campaign status
var campaignStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show campaign status",
	RunE:  campaignStatus,
}

func init() {
	// Add subcommands to campaign
	campaignCmd.AddCommand(createCampaignCmd)
	campaignCmd.AddCommand(startCampaignCmd)
	campaignCmd.AddCommand(watchCampaignCmd)
	campaignCmd.AddCommand(listCampaignsCmd)
	campaignCmd.AddCommand(campaignStatusCmd)

	// Add flags to create command
	createCampaignCmd.Flags().StringVarP(&objectivePath, "objective", "o", "", "Path to objective file (required)")
	createCampaignCmd.MarkFlagRequired("objective")
	createCampaignCmd.Flags().StringVarP(&campaignName, "name", "n", "", "Campaign name (defaults to objective title)")
	createCampaignCmd.Flags().StringVar(&managerID, "manager", "", "Override default manager agent ID")

	// Add flags to start, watch, and status commands
	startCampaignCmd.Flags().StringVar(&campaignID, "id", "", "Campaign ID (required)")
	startCampaignCmd.MarkFlagRequired("id")

	watchCampaignCmd.Flags().StringVar(&campaignID, "id", "", "Campaign ID (required)")
	watchCampaignCmd.MarkFlagRequired("id")

	campaignStatusCmd.Flags().StringVar(&campaignID, "id", "", "Campaign ID (required)")
	campaignStatusCmd.MarkFlagRequired("id")
}

func createCampaign(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get project context
	projCtx, err := project.GetContext()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get project context").
			WithComponent("cli").
			WithOperation("campaign.create")
	}

	// Load objective
	if !filepath.IsAbs(objectivePath) {
		objectivePath = filepath.Join(projCtx.GetRootPath(), objectivePath)
	}

	obj, err := commission.ParseFile(objectivePath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to parse objective").
			WithComponent("cli").
			WithOperation("campaign.create").
			WithDetails("path", objectivePath)
	}

	// Load guild configuration
	guildConfig, err := config.LoadGuildConfig(projCtx.GetRootPath())
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load guild config").
			WithComponent("cli").
			WithOperation("campaign.create")
	}

	// Override manager if specified
	if managerID != "" {
		guildConfig.Manager.Override = managerID
	}

	// Initialize registry
	reg := registry.NewComponentRegistry()
	if err := reg.Initialize(ctx, registry.Config{}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("cli").
			WithOperation("campaign.create")
	}

	// Initialize storage using registry for SQLite
	// The registry should already have SQLite storage initialized
	storageReg := reg.Storage()
	if storageReg == nil {
		return gerror.New(gerror.ErrCodeInternal, "storage registry not available", nil).
			WithComponent("cli").
			WithOperation("campaign.create")
	}

	// Create kanban board using registry
	// We need to create a kanban manager with the proper registry adapter
	kanbanRegistry := &kanbanComponentRegistry{componentReg: reg}
	kanbanMgr, err := kanban.NewManagerWithRegistry(ctx, kanbanRegistry)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create kanban manager").
			WithComponent("cli").
			WithOperation("campaign.create")
	}

	// Create kanban board
	board, err := kanbanMgr.CreateBoard(ctx, fmt.Sprintf("Campaign-%s", obj.ID), "Campaign board")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create kanban board").
			WithComponent("cli").
			WithOperation("campaign.create").
			WithDetails("campaign_id", obj.ID)
	}

	// Create guild factory
	// TODO: Get these from registry properly
	// memoryManager := &dummyMemoryManager{}
	// toolRegistryComponent := tools.DefaultToolRegistryFactory()

	// Use commission repository instead of objective manager
	commissionRepo := storageReg.GetCommissionRepository()
	if commissionRepo == nil {
		return gerror.New(gerror.ErrCodeInternal, "commission repository not available", nil).
			WithComponent("cli").
			WithOperation("campaign.create")
	}

	// Create manager agent directly from registry
	managerAgent, err := reg.Agents().GetAgent("manager")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeAgent, "failed to create manager agent").
			WithComponent("cli").
			WithOperation("campaign.create")
	}

	// Create task planner
	planner := orchestrator.DefaultManagerTaskPlannerFactory(managerAgent, board)

	// Plan tasks
	fmt.Println("Planning tasks with manager agent...")
	tasks, err := planner.PlanTasks(ctx, obj, guildConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to plan tasks").
			WithComponent("cli").
			WithOperation("campaign.create")
	}

	fmt.Printf("Created %d tasks\n", len(tasks))

	// Assign tasks
	fmt.Println("Assigning tasks to agents...")
	if err := planner.AssignTasks(ctx, tasks, guildConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to assign tasks").
			WithComponent("cli").
			WithOperation("campaign.create")
	}

	// Create campaign
	if campaignName == "" {
		campaignName = obj.Title
	}

	campaignModel := &campaign.Campaign{
		ID:              fmt.Sprintf("campaign-%d", time.Now().Unix()),
		Name:            campaignName,
		Description:     obj.Description,
		Status:          campaign.CampaignStatusReady,
		Objectives:      []string{obj.ID},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		TotalObjectives: 1,
		Metadata: map[string]interface{}{
			"guild_config":  guildConfig.Name,
			"manager_agent": guildConfig.Manager.Default,
			"board_id":      board.ID,
		},
	}

	// Save campaign (TODO: Use campaign repository)
	campaignPath := filepath.Join(projCtx.GetGuildPath(), "campaigns", campaignModel.ID+".json")
	if err := os.MkdirAll(filepath.Dir(campaignPath), 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create campaign directory").
			WithComponent("cli").
			WithOperation("campaign.create").
			WithDetails("path", filepath.Dir(campaignPath))
	}

	// For now, just print success
	fmt.Printf("\nCampaign created successfully!\n")
	fmt.Printf("ID: %s\n", campaignModel.ID)
	fmt.Printf("Name: %s\n", campaignModel.Name)
	fmt.Printf("Objective: %s\n", obj.Title)
	fmt.Printf("Tasks: %d\n", len(tasks))
	fmt.Printf("\nUse 'guild campaign start --id %s' to begin execution\n", campaignModel.ID)

	return nil
}

func startCampaign(cmd *cobra.Command, args []string) error {
	// TODO: Load campaign from storage
	// TODO: Initialize orchestrator
	// TODO: Start execution

	fmt.Printf("Starting campaign %s...\n", campaignID)
	fmt.Println("Campaign execution not yet implemented")

	return nil
}

func watchCampaign(cmd *cobra.Command, args []string) error {
	// TODO: Connect to campaign gRPC stream
	// TODO: Display real-time updates

	fmt.Printf("Watching campaign %s...\n", campaignID)
	fmt.Println("Campaign watching not yet implemented")

	return nil
}

func listCampaigns(cmd *cobra.Command, args []string) error {
	// TODO: List campaigns from storage

	fmt.Println("Campaign listing not yet implemented")

	return nil
}

func campaignStatus(cmd *cobra.Command, args []string) error {
	// TODO: Load campaign and show status

	fmt.Printf("Campaign %s status:\n", campaignID)
	fmt.Println("Campaign status not yet implemented")

	return nil
}

// Temporary dummy memory manager
type dummyMemoryManager struct{}

func (d *dummyMemoryManager) CreateChain(ctx context.Context, agentID, taskID string) (string, error) {
	chainID := fmt.Sprintf("chain-%d", time.Now().Unix())
	return chainID, nil
}

func (d *dummyMemoryManager) GetChain(ctx context.Context, chainID string) (*memory.PromptChain, error) {
	return nil, nil
}

func (d *dummyMemoryManager) AddMessage(ctx context.Context, chainID string, message memory.Message) error {
	return nil
}

func (d *dummyMemoryManager) GetChainsByAgent(ctx context.Context, agentID string) ([]*memory.PromptChain, error) {
	return []*memory.PromptChain{}, nil
}

func (d *dummyMemoryManager) GetChainsByTask(ctx context.Context, taskID string) ([]*memory.PromptChain, error) {
	return []*memory.PromptChain{}, nil
}

func (d *dummyMemoryManager) BuildContext(ctx context.Context, agentID, taskID string, maxTokens int) ([]memory.Message, error) {
	return []memory.Message{}, nil
}

func (d *dummyMemoryManager) DeleteChain(ctx context.Context, chainID string) error {
	return nil
}

// kanbanComponentRegistry adapts registry.ComponentRegistry to kanban.ComponentRegistry
type kanbanComponentRegistry struct {
	componentReg registry.ComponentRegistry
}

func (k *kanbanComponentRegistry) Storage() kanban.StorageRegistry {
	// Return a kanban storage registry adapter
	return &kanbanStorageAdapter{storageReg: k.componentReg.Storage()}
}

// kanbanStorageAdapter adapts registry.StorageRegistry to kanban.StorageRegistry
type kanbanStorageAdapter struct {
	storageReg registry.StorageRegistry
}

func (k *kanbanStorageAdapter) GetKanbanCampaignRepository() kanban.CampaignRepository {
	// Return the campaign repository from registry
	return k.storageReg.GetKanbanCampaignRepository()
}

func (k *kanbanStorageAdapter) GetKanbanCommissionRepository() kanban.CommissionRepository {
	return k.storageReg.GetKanbanCommissionRepository()
}

func (k *kanbanStorageAdapter) GetBoardRepository() kanban.BoardRepository {
	return k.storageReg.GetBoardRepository()
}

func (k *kanbanStorageAdapter) GetKanbanTaskRepository() kanban.TaskRepository {
	return k.storageReg.GetKanbanTaskRepository()
}

func (k *kanbanStorageAdapter) GetMemoryStore() kanban.MemoryStore {
	return k.storageReg.GetMemoryStore()
}
