// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// cmd/guild/commission.go
//
// Commission-based workflow for Guild agent coordination
// This file implements the true Guild functionality: coordinating specialized
// agents (artisans) to complete complex work through strategic commissions.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/lancekrogers/guild/internal/daemon"
	"github.com/lancekrogers/guild/pkg/agents/core"
	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/lancekrogers/guild/pkg/memory"
	"github.com/lancekrogers/guild/pkg/orchestrator"
	"github.com/lancekrogers/guild/pkg/project"
	"github.com/lancekrogers/guild/pkg/providers"
	"github.com/lancekrogers/guild/pkg/registry"
	"github.com/lancekrogers/guild/pkg/storage"
	"github.com/lancekrogers/guild/pkg/storage/promptchain"
	"github.com/lancekrogers/guild/pkg/tools"
)

var (
	// Commission flags
	assignFlag         bool   // Auto-assign agents to tasks
	dryRunFlag         bool   // Show what would be done without executing
	campaignIDFlag     string // Associate with campaign
	priorityFlag       string // Commission priority (high, medium, low)
	managerFlag        string // Override default manager agent
	commissionNoDaemon bool   // Don't auto-start the Guild server
)

// commissionCmd represents the commission command group
var commissionCmd = &cobra.Command{
	Use:   "commission [description]",
	Short: "Commission the Guild to complete strategic work",
	Long: `Commission specialized artisans to collaborate on complex tasks.

The commission system coordinates your Guild's agents, automatically:
- Decomposing work into specialized tasks
- Assigning tasks to capable artisans
- Tracking progress through the workshop board
- Coordinating agent collaboration

Examples:
  guild commission "Build a REST API for user management"
  guild commission "Research and implement caching strategy" --assign
  guild commission "Review security practices" --campaign security-audit`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// Show help when no arguments provided
			cmd.Help()
			return
		}

		description := strings.Join(args, " ")
		if err := executeCommission(cmd.Context(), description); err != nil {
			cmd.Printf("Commission failed: %v\n", err)
			return
		}
	},
}

// commissionStatusCmd shows the status of active commissions
var commissionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of active Guild commissions",
	Long:  `Display the current status of all active commissions and agent assignments.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := showCommissionStatus(cmd.Context()); err != nil {
			cmd.Printf("Failed to show status: %v\n", err)
			return
		}
	},
}

// commissionListCmd lists all commissions
var commissionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Guild commissions",
	Long:  `List all commissions with their status and progress.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listCommissions(cmd.Context()); err != nil {
			cmd.Printf("Failed to list commissions: %v\n", err)
			return
		}
	},
}

// workshopCmd represents the workshop command group (for monitoring work)
var workshopCmd = &cobra.Command{
	Use:   "workshop",
	Short: "Monitor and manage the Guild workshop",
	Long: `Monitor active work in the Guild workshop.

The workshop shows real-time status of:
- Agent assignments and progress
- Task dependencies and blockers
- Resource utilization and costs
- Collaboration coordination`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := showWorkshopStatus(cmd.Context()); err != nil {
			cmd.Printf("Workshop status failed: %v\n", err)
			return
		}
	},
}

func init() {
	// Register commission commands
	rootCmd.AddCommand(commissionCmd)
	rootCmd.AddCommand(workshopCmd)

	// Register commission subcommands
	commissionCmd.AddCommand(commissionStatusCmd)
	commissionCmd.AddCommand(commissionListCmd)

	// Add commission flags
	commissionCmd.Flags().BoolVar(&assignFlag, "assign", false, "Automatically assign agents to tasks")
	commissionCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be done without executing")
	commissionCmd.Flags().StringVar(&campaignIDFlag, "campaign", "", "Associate with campaign ID")
	commissionCmd.Flags().StringVar(&priorityFlag, "priority", "medium", "Commission priority (high, medium, low)")
	commissionCmd.Flags().StringVar(&managerFlag, "manager", "", "Override default manager agent")
	commissionCmd.Flags().BoolVar(&commissionNoDaemon, "no-daemon", false, "Don't auto-start the Guild server")
}

// executeCommission creates and executes a commission using the orchestrator
func executeCommission(ctx context.Context, description string) error {
	fmt.Printf("📜 Commissioning Guild work: %s\n\n", description)

	// Auto-start daemon unless --no-daemon flag is set
	if !commissionNoDaemon {
		if !daemon.IsReachable(ctx) {
			fmt.Println("🚀 Starting Guild server...")
			if err := daemon.EnsureRunning(ctx); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start Guild server").
					WithComponent("cli").
					WithOperation("commission.daemon_start")
			}
			// Give the server a moment to fully initialize
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Check if server is reachable
	if !daemon.IsReachable(ctx) {
		return gerror.New(gerror.ErrCodeConnection, "Guild server is not reachable", nil).
			WithComponent("cli").
			WithOperation("commission.execute").
			WithDetails("help", "Try running 'guild serve' manually or check 'guild status'")
	}

	// Setup Guild context and components
	components, err := setupGuildComponents(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup Guild components").
			WithComponent("cli").
			WithOperation("commission.execute")
	}
	defer components.cleanup()

	// Create commission from description
	obj := commission.NewCommission("Guild Commission", description)
	obj.Priority = priorityFlag
	obj.CampaignID = campaignIDFlag

	// Save commission
	desc := obj.Description
	commission := &storage.Commission{
		ID:          obj.ID,
		CampaignID:  obj.CampaignID,
		Title:       obj.Title,
		Description: &desc,
		Status:      "active",
		CreatedAt:   time.Now(),
	}
	if err := components.commissionRepo.CreateCommission(ctx, commission); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save commission").
			WithComponent("cli").
			WithOperation("commission.execute")
	}

	fmt.Printf("✅ Commission registered: %s\n", obj.ID)

	// Set objective in orchestrator
	if err := components.orchestrator.SetCommission(obj); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to set commission in orchestrator").
			WithComponent("cli").
			WithOperation("commission.execute")
	}

	// Start orchestrator if not running
	if components.orchestrator.Status() != orchestrator.StatusRunning {
		if err := components.orchestrator.Start(ctx); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to start orchestrator").
				WithComponent("cli").
				WithOperation("commission.execute")
		}
		fmt.Printf("🚀 Guild orchestrator started\n")
	}

	// Plan and assign tasks if requested
	if assignFlag {
		if err := planAndAssignTasks(ctx, components, obj); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to plan and assign tasks").
				WithComponent("cli").
				WithOperation("commission.execute")
		}
	} else {
		fmt.Printf("💡 Use --assign to automatically decompose work and assign agents\n")
	}

	// Show initial status
	fmt.Printf("\n📊 Commission Status:\n")
	return showCommissionStatusRefine(ctx, components, obj.ID)
}

// planAndAssignTasks uses the orchestrator to decompose work and assign agents
func planAndAssignTasks(ctx context.Context, components *guildComponents, obj *commission.Commission) error {
	fmt.Printf("🧠 Manager agent planning tasks...\n")

	// Get the manager agent
	managerAgent, err := getManagerAgent(components, components.agentFactory)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeAgent, "failed to get manager agent").
			WithComponent("cli").
			WithOperation("commission.planAndAssignTasks")
	}

	// Create task planner
	planner := orchestrator.DefaultManagerTaskPlannerFactory(managerAgent, components.kanbanBoard)

	// Plan tasks
	tasks, err := planner.PlanTasks(ctx, obj, components.guildConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "task planning failed").
			WithComponent("cli").
			WithOperation("commission.planAndAssignTasks")
	}

	fmt.Printf("✅ Planned %d tasks\n", len(tasks))

	// Assign tasks to agents
	if err := planner.AssignTasks(ctx, tasks, components.guildConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "task assignment failed").
			WithComponent("cli").
			WithOperation("commission.planAndAssignTasks")
	}

	fmt.Printf("✅ Tasks assigned to agents\n")

	// Show task breakdown
	fmt.Printf("\n📋 Task Breakdown:\n")
	for i, task := range tasks {
		assignee := "unassigned"
		if task.AssignedTo != "" {
			assignee = task.AssignedTo
		}
		fmt.Printf("  %d. %s\n", i+1, task.Title)
		fmt.Printf("     Assigned to: %s\n", assignee)
		if complexity, ok := task.Metadata["complexity"]; ok {
			fmt.Printf("     Complexity: %s\n", complexity)
		}
		fmt.Printf("\n")
	}

	return nil
}

// showCommissionStatus displays the status of active commissions
func showCommissionStatus(ctx context.Context) error {
	// Auto-start daemon unless --no-daemon flag is set
	if !commissionNoDaemon {
		if !daemon.IsReachable(ctx) {
			fmt.Println("🚀 Starting Guild server...")
			if err := daemon.EnsureRunning(ctx); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start Guild server").
					WithComponent("cli").
					WithOperation("commission.status.daemon_start")
			}
			// Give the server a moment to fully initialize
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Check if server is reachable
	if !daemon.IsReachable(ctx) {
		return gerror.New(gerror.ErrCodeConnection, "Guild server is not reachable", nil).
			WithComponent("cli").
			WithOperation("commission.status").
			WithDetails("help", "Try running 'guild serve' manually or check 'guild status'")
	}

	components, err := setupGuildComponents(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup Guild components").
			WithComponent("cli").
			WithOperation("commission.showStatus")
	}
	defer components.cleanup()

	fmt.Printf("🏰 Guild Workshop Status\n\n")

	// Show orchestrator status
	fmt.Printf("Orchestrator: %s\n", components.orchestrator.Status())

	// List active commissions
	// List active commissions by campaign (use empty campaign ID to get all)
	commissions, err := components.commissionRepo.ListCommissionsByCampaign(ctx, "")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list commissions").
			WithComponent("cli").
			WithOperation("commission.showStatus")
	}

	if len(commissions) == 0 {
		fmt.Printf("No active commissions\n")
		return nil
	}

	fmt.Printf("\nActive Commissions:\n")
	for i, commission := range commissions {
		fmt.Printf("  %d. %s (%s)\n", i+1, commission.Title, commission.Status)
		if commission.Description != nil {
			fmt.Printf("     %s\n", *commission.Description)
		}
		if commission.CampaignID != "" {
			fmt.Printf("     Campaign: %s\n", commission.CampaignID)
		}
		fmt.Printf("\n")
	}

	return nil
}

// listCommissions lists all commissions
func listCommissions(ctx context.Context) error {
	// Auto-start daemon unless --no-daemon flag is set
	if !commissionNoDaemon {
		if !daemon.IsReachable(ctx) {
			fmt.Println("🚀 Starting Guild server...")
			if err := daemon.EnsureRunning(ctx); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start Guild server").
					WithComponent("cli").
					WithOperation("commission.list.daemon_start")
			}
			// Give the server a moment to fully initialize
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Check if server is reachable
	if !daemon.IsReachable(ctx) {
		return gerror.New(gerror.ErrCodeConnection, "Guild server is not reachable", nil).
			WithComponent("cli").
			WithOperation("commission.list").
			WithDetails("help", "Try running 'guild serve' manually or check 'guild status'")
	}

	components, err := setupGuildComponents(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup Guild components").
			WithComponent("cli").
			WithOperation("commission.list")
	}
	defer components.cleanup()

	// List active commissions by campaign (use empty campaign ID to get all)
	commissions, err := components.commissionRepo.ListCommissionsByCampaign(ctx, "")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list commissions").
			WithComponent("cli").
			WithOperation("commission.list")
	}

	if len(commissions) == 0 {
		fmt.Printf("No commissions found\n")
		return nil
	}

	fmt.Printf("Guild Commissions:\n\n")
	for _, commission := range commissions {
		fmt.Printf("📜 %s (%s)\n", commission.Title, commission.Status)
		if commission.Description != nil {
			fmt.Printf("   %s\n", *commission.Description)
		}
		fmt.Printf("   Created: %s\n", commission.CreatedAt.Format("2006-01-02 15:04"))
		if commission.CampaignID != "" {
			fmt.Printf("   Campaign: %s\n", commission.CampaignID)
		}
		fmt.Printf("\n")
	}

	return nil
}

// showWorkshopStatus displays the workshop status with agent activity
func showWorkshopStatus(ctx context.Context) error {
	components, err := setupGuildComponents(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to setup Guild components").
			WithComponent("cli").
			WithOperation("workshop.status")
	}
	defer components.cleanup()

	fmt.Printf("🔨 Guild Workshop Status\n\n")

	// Show available agents
	fmt.Printf("Available Artisans:\n")
	for i, agent := range components.guildConfig.Agents {
		fmt.Printf("  %d. %s (%s)\n", i+1, agent.Name, agent.ID)
		fmt.Printf("     Type: %s | Provider: %s\n", agent.Type, agent.Provider)
		fmt.Printf("     Capabilities: %s\n", strings.Join(agent.Capabilities, ", "))
		fmt.Printf("\n")
	}

	// Show current tasks
	tasks, err := components.kanbanBoard.GetAllTasks(ctx)
	if err != nil {
		fmt.Printf("Warning: Could not load workshop tasks: %v\n", err)
	} else if len(tasks) > 0 {
		fmt.Printf("Active Tasks:\n")
		for _, task := range tasks {
			assignee := "unassigned"
			if task.AssignedTo != "" {
				assignee = task.AssignedTo
			}
			fmt.Printf("  • %s [%s] - %s\n", task.Title, task.Status, assignee)
		}
		fmt.Printf("\n")
	}

	return nil
}

// showCommissionStatus shows detailed status for a specific commission
func showCommissionStatusRefine(ctx context.Context, components *guildComponents, objectiveID string) error {
	commission, err := components.commissionRepo.GetCommission(ctx, objectiveID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get commission").
			WithComponent("cli").
			WithOperation("commission.showCommissionStatus").
			WithDetails("objective_id", objectiveID)
	}

	fmt.Printf("Commission: %s\n", commission.Title)
	fmt.Printf("Status: %s\n", commission.Status)
	fmt.Printf("Priority: high\n") // Default priority for now
	if commission.CampaignID != "" {
		fmt.Printf("Campaign: %s\n", commission.CampaignID)
	}

	// TODO: Calculate and show completion based on tasks
	fmt.Printf("Completion: N/A\n")

	return nil
}

// guildComponents holds all the components needed for Guild operations
type guildComponents struct {
	guildConfig    *config.GuildConfig
	commissionRepo storage.CommissionRepository
	kanbanBoard    *kanban.Board
	orchestrator   orchestrator.Orchestrator
	eventBus       orchestrator.EventBus
	registry       registry.ComponentRegistry
	agentFactory   *guildAgentFactory
	store          interface{} // Keep reference for cleanup
}

// cleanup cleans up resources
func (gc *guildComponents) cleanup() {
	// Close any resources if needed
}

// setupGuildComponents initializes all Guild components
func setupGuildComponents(ctx context.Context) (*guildComponents, error) {
	// Get project context
	projCtx, err := project.GetContext()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get project context").
			WithComponent("cli").
			WithOperation("commission.setupGuildComponents")
	}

	// Load guild configuration
	guildConfig, err := config.LoadGuildConfig(ctx, projCtx.GetRootPath())
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load guild config").
			WithComponent("cli").
			WithOperation("commission.setupGuildComponents")
	}

	// Setup data directory
	dataDir := filepath.Join(projCtx.GetRootPath(), ".guild")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create data directory").
			WithComponent("cli").
			WithOperation("commission.setupGuildComponents").
			WithDetails("path", dataDir)
	}

	// Initialize registry with SQLite storage
	reg := registry.NewComponentRegistry()
	registryConfig := registry.Config{
		Agents: registry.AgentConfigYaml{
			DefaultType: "worker",
		},
	}
	if err := reg.Initialize(ctx, registryConfig); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("cli").
			WithOperation("commission.setupGuildComponents")
	}

	// Get storage registry for commission repository
	storageReg := reg.Storage()
	if storageReg == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "storage registry not available", nil).
			WithComponent("cli").
			WithOperation("commission.setupGuildComponents")
	}

	// Use commission repository instead of objective manager
	regCommissionRepo := storageReg.GetCommissionRepository()
	if regCommissionRepo == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "commission repository not available", nil).
			WithComponent("cli").
			WithOperation("commission.setupGuildComponents")
	}

	// Wrap the registry commission repo to match storage interface
	commissionRepo := &commissionRepoAdapter{repo: regCommissionRepo}

	// Create kanban manager using registry
	kanbanRegistry := &kanbanComponentRegistry{componentReg: reg}
	kanbanMgr, err := kanban.NewManagerWithRegistry(ctx, kanbanRegistry)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create kanban manager").
			WithComponent("cli").
			WithOperation("commission.setupGuildComponents")
	}

	// Create or get kanban board
	kanbanBoard, err := kanbanMgr.CreateBoard(ctx, "main-board", "Guild Main Workshop Board")
	if err != nil {
		// Try to get existing board
		boards, err := kanbanMgr.ListBoards(ctx)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list boards").
				WithComponent("cli").
				WithOperation("commission.setupGuildComponents")
		}
		for _, board := range boards {
			if board.ID == "main-board" {
				kanbanBoard = board
				break
			}
		}
		if kanbanBoard == nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create or find kanban board").
				WithComponent("cli").
				WithOperation("commission.setupGuildComponents")
		}
	}

	// Initialize event bus
	eventBus := orchestrator.DefaultEventBusFactory()

	// Initialize orchestrator
	orchestratorConfig := &orchestrator.Config{
		MaxConcurrentAgents: 5,
		ManagerAgentID:      getManagerAgentID(guildConfig),
		KanbanBoardID:       "main-board",
		ExecutionMode:       "managed",
	}

	// Registry already initialized above

	// Initialize provider factory
	providerFactory := providers.NewFactory()

	// Initialize memory manager with SQLite chain manager
	// For SQLite storage, we need to get the underlying storage registry
	var memoryManager memory.ChainManager
	if sqliteReg, ok := storageReg.(*registry.SQLiteStorageRegistry); ok {
		// Get the underlying storage registry
		underlyingStorageReg := sqliteReg.GetStorageRegistry()
		if underlyingStorageReg != nil {
			promptChainRepo := underlyingStorageReg.GetPromptChainRepository()
			if promptChainRepo != nil {
				// Use the promptchain package's SQLite chain manager
				memoryManager = promptchain.NewSQLiteChainManager(promptChainRepo)
			}
		}
	}

	if memoryManager == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "failed to create memory manager", nil).
			WithComponent("cli").
			WithOperation("commission.setupGuildComponents")
	}

	// Initialize tool registry via registry pattern
	toolRegistryComponent := reg.Tools()

	// Create a tools.Registry adapter if needed
	var toolsRegistry tools.Registry
	if toolRegistryComponent != nil {
		// Create adapter from registry.ToolRegistry to tools.Registry
		toolsRegistry = &toolsRegistryAdapter{registry: toolRegistryComponent}
	} else {
		// Fallback to direct creation
		toolsRegistry = tools.DefaultToolRegistryFactory()
	}

	// Create guild agent factory
	agentFactory := &guildAgentFactory{
		registry:        reg,
		guildConfig:     guildConfig,
		agentInstances:  make(map[string]core.Agent),
		providerFactory: providerFactory,
		memoryManager:   memoryManager,
		toolRegistry:    toolsRegistry,
		commissionRepo:  commissionRepo,
	}

	// Register all agents from guild config into the registry
	if err := registerGuildAgents(reg, guildConfig); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register guild agents").
			WithComponent("cli").
			WithOperation("commission.setupGuildComponents")
	}

	// Create kanban manager adapter
	kanbanManager := &kanbanManagerAdapter{board: kanbanBoard}

	dispatcher := orchestrator.DefaultTaskDispatcherFactory(kanbanManager, agentFactory, eventBus, orchestratorConfig.MaxConcurrentAgents)

	orch := orchestrator.DefaultOrchestratorFactory(orchestratorConfig, dispatcher, eventBus)

	return &guildComponents{
		guildConfig:    guildConfig,
		commissionRepo: commissionRepo,
		kanbanBoard:    kanbanBoard,
		orchestrator:   orch,
		eventBus:       eventBus,
		registry:       reg,
		agentFactory:   agentFactory,
	}, nil
}

// getManagerAgentID returns the manager agent ID from guild config
func getManagerAgentID(guild *config.GuildConfig) string {
	if managerFlag != "" {
		return managerFlag
	}

	if guild.Manager.Override != "" {
		return guild.Manager.Override
	}

	if guild.Manager.Default != "" {
		return guild.Manager.Default
	}

	// Find first manager type agent
	for _, agent := range guild.Agents {
		if agent.Type == "manager" {
			return agent.ID
		}
	}

	// Fallback to first agent
	if len(guild.Agents) > 0 {
		return guild.Agents[0].ID
	}

	return "default-manager"
}

// kanbanManagerAdapter adapts kanban.Board to orchestrator.KanbanManager interface
type kanbanManagerAdapter struct {
	board *kanban.Board
}

// ListTasksByStatus implements orchestrator.KanbanManager
func (k *kanbanManagerAdapter) ListTasksByStatus(ctx context.Context, boardID string, status kanban.TaskStatus) ([]*kanban.Task, error) {
	return k.board.GetTasksByStatus(ctx, status)
}

// UpdateTaskStatus implements orchestrator.KanbanManager
func (k *kanbanManagerAdapter) UpdateTaskStatus(ctx context.Context, taskID, status, assignee, comment string) error {
	return k.board.UpdateTaskStatus(ctx, taskID, kanban.TaskStatus(status), assignee, comment)
}

// GetTask implements orchestrator.KanbanManager
func (k *kanbanManagerAdapter) GetTask(ctx context.Context, taskID string) (*kanban.Task, error) {
	return k.board.GetTask(ctx, taskID)
}

// CreateTask implements orchestrator.KanbanManager
func (k *kanbanManagerAdapter) CreateTask(ctx context.Context, title, description string) (*kanban.Task, error) {
	return k.board.CreateTask(ctx, title, description)
}

// toolsRegistryAdapter adapts registry.ToolRegistry to tools.Registry interface
type toolsRegistryAdapter struct {
	registry registry.ToolRegistry
}

// RegisterTool implements tools.Registry
func (t *toolsRegistryAdapter) RegisterTool(name string, tool tools.Tool) error {
	if t.registry == nil {
		return gerror.New(gerror.ErrCodeInternal, "registry not initialized", nil).
			WithComponent("adapter").
			WithOperation("RegisterTool")
	}
	// Convert tools.Tool to registry.Tool interface
	return t.registry.RegisterTool(name, tool)
}

// GetTool implements tools.Registry
func (t *toolsRegistryAdapter) GetTool(name string) (tools.Tool, error) {
	if t.registry == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "registry not initialized", nil).
			WithComponent("adapter").
			WithOperation("GetTool")
	}
	return t.registry.GetTool(name)
}

// ListTools implements tools.Registry
func (t *toolsRegistryAdapter) ListTools() []string {
	if t.registry == nil {
		return []string{}
	}
	return t.registry.ListTools()
}

// Clear implements tools.Registry
func (t *toolsRegistryAdapter) Clear() {
	// Not supported by registry.ToolRegistry, so do nothing
}

// HasTool implements tools.Registry
func (t *toolsRegistryAdapter) HasTool(name string) bool {
	if t.registry == nil {
		return false
	}
	return t.registry.HasTool(name)
}

// UnregisterTool implements tools.Registry
func (t *toolsRegistryAdapter) UnregisterTool(name string) error {
	// Not supported by registry.ToolRegistry, return nil
	return nil
}

// UpdateTask implements orchestrator.KanbanManager
func (k *kanbanManagerAdapter) UpdateTask(ctx context.Context, task *kanban.Task) error {
	return k.board.UpdateTask(ctx, task)
}

// guildAgentFactory implements AgentFactory interface
type guildAgentFactory struct {
	registry        registry.ComponentRegistry
	guildConfig     *config.GuildConfig
	agentInstances  map[string]core.Agent // Cache for created agents
	providerFactory *providers.Factory
	memoryManager   memory.ChainManager
	toolRegistry    tools.Registry
	commissionRepo  storage.CommissionRepository
}

// CreateAgent creates an agent from guild configuration
func (f *guildAgentFactory) CreateAgent(agentID, name string, options ...interface{}) (core.Agent, error) {
	// Check if agent already exists in cache
	if existingAgent, exists := f.agentInstances[agentID]; exists {
		return existingAgent, nil
	}

	// Find agent config by ID
	var agentConfig *config.AgentConfig
	for _, cfg := range f.guildConfig.Agents {
		if cfg.ID == agentID {
			agentConfig = &cfg
			break
		}
	}

	if agentConfig == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "agent configuration not found", nil).
			WithComponent("cli").
			WithOperation("commission.guildAgentFactory.CreateAgent").
			WithDetails("agent_id", agentID)
	}

	// Map provider name to ProviderType
	var providerType providers.ProviderType
	switch agentConfig.Provider {
	case "openai":
		providerType = providers.ProviderOpenAI
	case "anthropic":
		providerType = providers.ProviderAnthropic
	case "ollama":
		providerType = providers.ProviderOllama
	case "claudecode":
		providerType = providers.ProviderClaudeCode
	case "deepseek":
		providerType = providers.ProviderDeepSeek
	case "deepinfra":
		providerType = providers.ProviderDeepInfra
	case "ora":
		providerType = providers.ProviderOra
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unknown provider", nil).
			WithComponent("cli").
			WithOperation("commission.guildAgentFactory.CreateAgent").
			WithDetails("provider", agentConfig.Provider)
	}

	// Get API key from guild configuration (with environment variable fallback)
	apiKey := f.guildConfig.GetProviderAPIKey(agentConfig.Provider)

	// Check if API key is required and missing for this provider
	if apiKey == "" && requiresAPIKey(agentConfig.Provider) {
		return nil, gerror.New(gerror.ErrCodeProviderAuth, "API key required", nil).
			WithComponent("cli").
			WithOperation("commission.guildAgentFactory.CreateAgent").
			WithDetails("provider", agentConfig.Provider).
			WithDetails("env_var", getEnvVarName(agentConfig.Provider))
	}

	// Create LLM client for this agent
	llmClient, err := f.providerFactory.CreateClient(providerType, apiKey, agentConfig.Model)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeProvider, "failed to create LLM client").
			WithComponent("cli").
			WithOperation("commission.guildAgentFactory.CreateAgent").
			WithDetails("agent_id", agentID).
			WithDetails("provider", agentConfig.Provider)
	}

	// Create agent factory
	// Use the DefaultFactoryFactory
	costManager := core.DefaultCostManagerFactory()
	agentFactory := core.DefaultFactoryFactory(llmClient, f.memoryManager, f.toolRegistry, nil, costManager)

	// Create agent based on type
	var newAgent core.Agent
	switch agentConfig.Type {
	case "manager":
		newAgent, err = agentFactory.CreateManagerAgent(context.Background(), agentConfig.ID, agentConfig.Name)
	case "worker", "specialist":
		newAgent, err = agentFactory.CreateWorkerAgent(context.Background(), agentConfig.ID, agentConfig.Name)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unknown agent type", nil).
			WithComponent("cli").
			WithOperation("commission.guildAgentFactory.CreateAgent").
			WithDetails("agent_type", agentConfig.Type)
	}

	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeAgent, "failed to create agent").
			WithComponent("cli").
			WithOperation("commission.guildAgentFactory.CreateAgent").
			WithDetails("agent_id", agentID)
	}

	// Cache the agent instance
	f.agentInstances[agentID] = newAgent

	return newAgent, nil
}

// getManagerAgent returns the manager agent instance
func getManagerAgent(components *guildComponents, agentFactory *guildAgentFactory) (core.Agent, error) {
	managerID := getManagerAgentID(components.guildConfig)

	// Try to get from orchestrator first
	if agent, exists := components.orchestrator.GetAgent(managerID); exists {
		return agent, nil
	}

	// Create manager agent using the factory
	managerAgent, err := agentFactory.CreateAgent(managerID, "")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeAgent, "failed to create manager agent").
			WithComponent("cli").
			WithOperation("commission.getManagerAgent").
			WithDetails("manager_id", managerID)
	}

	return managerAgent, nil
}

// registerGuildAgents registers all agents from guild config into the component registry
func registerGuildAgents(reg registry.ComponentRegistry, guildConfig *config.GuildConfig) error {
	// Get the agent registry from the component registry
	agentRegistry := reg.Agents()

	// Register each agent configuration
	for _, agentConfig := range guildConfig.Agents {
		guildAgentConfig := registry.GuildAgentConfig{
			ID:            agentConfig.ID,
			Name:          agentConfig.Name,
			Type:          agentConfig.Type,
			Provider:      agentConfig.Provider,
			Model:         agentConfig.Model,
			Capabilities:  agentConfig.Capabilities,
			Tools:         agentConfig.Tools,
			CostMagnitude: agentConfig.CostMagnitude,
			ContextWindow: agentConfig.ContextWindow,
		}

		// Only register if not already registered
		if err := agentRegistry.RegisterGuildAgent(guildAgentConfig); err != nil {
			// Ignore "already registered" errors
			if !strings.Contains(err.Error(), "already registered") {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register agent").
					WithComponent("cli").
					WithOperation("commission.registerGuildAgents").
					WithDetails("agent_id", agentConfig.ID)
			}
		}
	}

	return nil
}

// requiresAPIKey checks if a provider requires an API key
func requiresAPIKey(provider string) bool {
	switch provider {
	case "ollama", "claudecode":
		return false // Local providers don't need API keys
	case "openai", "anthropic", "deepseek", "deepinfra", "ora":
		return true // Cloud providers require API keys
	default:
		return true // Default to requiring API key for unknown providers
	}
}

// getEnvVarName returns the standard environment variable name for a provider
func getEnvVarName(provider string) string {
	switch provider {
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "deepseek":
		return "DEEPSEEK_API_KEY"
	case "deepinfra":
		return "DEEPINFRA_API_KEY"
	case "ora":
		return "ORA_API_KEY"
	default:
		return strings.ToUpper(provider) + "_API_KEY"
	}
}

// commissionRepoAdapter adapts registry.CommissionRepository to storage.CommissionRepository
type commissionRepoAdapter struct {
	repo registry.CommissionRepository
}

func (c *commissionRepoAdapter) CreateCommission(ctx context.Context, commission *storage.Commission) error {
	// Convert storage.Commission to registry.Commission
	regCommission := &registry.Commission{
		ID:          commission.ID,
		CampaignID:  commission.CampaignID,
		Title:       commission.Title,
		Description: commission.Description,
		Domain:      commission.Domain,
		Context:     commission.Context,
		Status:      commission.Status,
		CreatedAt:   commission.CreatedAt,
	}
	return c.repo.CreateCommission(ctx, regCommission)
}

func (c *commissionRepoAdapter) GetCommission(ctx context.Context, id string) (*storage.Commission, error) {
	regCommission, err := c.repo.GetCommission(ctx, id)
	if err != nil {
		return nil, err
	}
	// Convert registry.Commission to storage.Commission
	return &storage.Commission{
		ID:          regCommission.ID,
		CampaignID:  regCommission.CampaignID,
		Title:       regCommission.Title,
		Description: regCommission.Description,
		Domain:      regCommission.Domain,
		Context:     regCommission.Context,
		Status:      regCommission.Status,
		CreatedAt:   regCommission.CreatedAt,
	}, nil
}

func (c *commissionRepoAdapter) UpdateCommissionStatus(ctx context.Context, id, status string) error {
	return c.repo.UpdateCommissionStatus(ctx, id, status)
}

func (c *commissionRepoAdapter) DeleteCommission(ctx context.Context, id string) error {
	return c.repo.DeleteCommission(ctx, id)
}

func (c *commissionRepoAdapter) ListCommissionsByCampaign(ctx context.Context, campaignID string) ([]*storage.Commission, error) {
	regCommissions, err := c.repo.ListCommissionsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	// Convert slice
	result := make([]*storage.Commission, len(regCommissions))
	for i, rc := range regCommissions {
		result[i] = &storage.Commission{
			ID:          rc.ID,
			CampaignID:  rc.CampaignID,
			Title:       rc.Title,
			Description: rc.Description,
			Domain:      rc.Domain,
			Context:     rc.Context,
			Status:      rc.Status,
			CreatedAt:   rc.CreatedAt,
		}
	}
	return result, nil
}
