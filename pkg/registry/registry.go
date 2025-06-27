// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/paths"
	"github.com/lancekrogers/guild/pkg/prompts/layered"
	"github.com/lancekrogers/guild/pkg/providers"
	"github.com/lancekrogers/guild/pkg/storage"
)

// layeredManagerWrapper wraps a basic manager to implement LayeredManager interface
type layeredManagerWrapper struct {
	manager layered.Manager
}

// Manager interface methods (delegate to wrapped manager)
func (w *layeredManagerWrapper) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	return w.manager.GetSystemPrompt(ctx, role, domain)
}

func (w *layeredManagerWrapper) GetTemplate(ctx context.Context, templateName string) (string, error) {
	return w.manager.GetTemplate(ctx, templateName)
}

func (w *layeredManagerWrapper) FormatContext(ctx context.Context, context layered.Context) (string, error) {
	return w.manager.FormatContext(ctx, context)
}

func (w *layeredManagerWrapper) ListRoles(ctx context.Context) ([]string, error) {
	return w.manager.ListRoles(ctx)
}

func (w *layeredManagerWrapper) ListDomains(ctx context.Context, role string) ([]string, error) {
	return w.manager.ListDomains(ctx, role)
}

// LayeredManager interface methods (stub implementations for now)
func (w *layeredManagerWrapper) BuildLayeredPrompt(ctx context.Context, artisanID, sessionID string, turnCtx layered.TurnContext) (*layered.LayeredPrompt, error) {
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "layered prompt building not yet implemented", nil).
		WithComponent("registry").
		WithOperation("BuildLayeredPrompt")
}

func (w *layeredManagerWrapper) GetPromptLayer(ctx context.Context, layer layered.PromptLayer, artisanID, sessionID string) (*layered.SystemPrompt, error) {
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "prompt layer retrieval not yet implemented", nil).
		WithComponent("registry").
		WithOperation("GetPromptLayer")
}

func (w *layeredManagerWrapper) SetPromptLayer(ctx context.Context, prompt layered.SystemPrompt) error {
	return gerror.New(gerror.ErrCodeNotImplemented, "prompt layer setting not yet implemented", nil).
		WithComponent("registry").
		WithOperation("SetPromptLayer")
}

func (w *layeredManagerWrapper) DeletePromptLayer(ctx context.Context, layer layered.PromptLayer, artisanID, sessionID string) error {
	return gerror.New(gerror.ErrCodeNotImplemented, "prompt layer deletion not yet implemented", nil).
		WithComponent("registry").
		WithOperation("DeletePromptLayer")
}

func (w *layeredManagerWrapper) ListPromptLayers(ctx context.Context, artisanID, sessionID string) ([]layered.SystemPrompt, error) {
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "prompt layer listing not yet implemented", nil).
		WithComponent("registry").
		WithOperation("ListPromptLayers")
}

func (w *layeredManagerWrapper) InvalidateCache(ctx context.Context, artisanID, sessionID string) error {
	return gerror.New(gerror.ErrCodeNotImplemented, "cache invalidation not yet implemented", nil).
		WithComponent("registry").
		WithOperation("InvalidateCache")
}

// DefaultComponentRegistry is the default implementation of ComponentRegistry
type DefaultComponentRegistry struct {
	agentRegistry        AgentRegistry
	toolRegistry         ToolRegistry
	providerRegistry     ProviderRegistry
	memoryRegistry       MemoryRegistry
	projectRegistry      ProjectRegistry
	promptRegistry       *PromptRegistry
	layeredPromptManager LayeredPromptManager
	storageRegistry      StorageRegistry
	orchestratorRegistry interface{}
	config               Config
	initialized          bool
	mu                   sync.RWMutex
}

// SQLiteStorageRegistry implements StorageRegistry for SQLite storage
type SQLiteStorageRegistry struct {
	registry    storage.StorageRegistry
	memoryStore MemoryStore

	// Kanban adapters to bridge interface{} expectations
	taskAdapter       *storage.KanbanTaskRepositoryAdapter
	boardAdapter      *storage.KanbanBoardRepositoryAdapter
	campaignAdapter   *storage.KanbanCampaignRepositoryAdapter
	commissionAdapter *storage.KanbanCommissionRepositoryAdapter
}

// campaignRepositoryAdapter adapts storage.CampaignRepository to registry.CampaignRepository
type campaignRepositoryAdapter struct {
	repo storage.CampaignRepository
}

func (a *campaignRepositoryAdapter) CreateCampaign(ctx context.Context, campaign *Campaign) error {
	// Convert registry.Campaign to storage.Campaign
	storageCampaign := &storage.Campaign{
		ID:        campaign.ID,
		Name:      campaign.Name,
		Status:    campaign.Status,
		CreatedAt: campaign.CreatedAt,
		UpdatedAt: campaign.UpdatedAt,
	}
	return a.repo.CreateCampaign(ctx, storageCampaign)
}

func (a *campaignRepositoryAdapter) GetCampaign(ctx context.Context, id string) (*Campaign, error) {
	storageCampaign, err := a.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert storage.Campaign to registry.Campaign
	return &Campaign{
		ID:        storageCampaign.ID,
		Name:      storageCampaign.Name,
		Status:    storageCampaign.Status,
		CreatedAt: storageCampaign.CreatedAt,
		UpdatedAt: storageCampaign.UpdatedAt,
	}, nil
}

func (a *campaignRepositoryAdapter) UpdateCampaignStatus(ctx context.Context, id, status string) error {
	return a.repo.UpdateCampaignStatus(ctx, id, status)
}

func (a *campaignRepositoryAdapter) DeleteCampaign(ctx context.Context, id string) error {
	return a.repo.DeleteCampaign(ctx, id)
}

func (a *campaignRepositoryAdapter) ListCampaigns(ctx context.Context) ([]*Campaign, error) {
	storageCampaigns, err := a.repo.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}

	campaigns := make([]*Campaign, 0, len(storageCampaigns))
	for _, sc := range storageCampaigns {
		campaigns = append(campaigns, &Campaign{
			ID:        sc.ID,
			Name:      sc.Name,
			Status:    sc.Status,
			CreatedAt: sc.CreatedAt,
			UpdatedAt: sc.UpdatedAt,
		})
	}
	return campaigns, nil
}

// commissionRepositoryAdapter adapts storage.CommissionRepository to registry.CommissionRepository
type commissionRepositoryAdapter struct {
	repo storage.CommissionRepository
}

func (a *commissionRepositoryAdapter) CreateCommission(ctx context.Context, commission *Commission) error {
	// Convert registry.Commission to storage.Commission
	storageCommission := &storage.Commission{
		ID:          commission.ID,
		CampaignID:  commission.CampaignID,
		Title:       commission.Title,
		Description: commission.Description,
		Domain:      commission.Domain,
		Context:     commission.Context,
		Status:      commission.Status,
		CreatedAt:   commission.CreatedAt,
	}
	return a.repo.CreateCommission(ctx, storageCommission)
}

func (a *commissionRepositoryAdapter) GetCommission(ctx context.Context, id string) (*Commission, error) {
	storageCommission, err := a.repo.GetCommission(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert storage.Commission to registry.Commission
	return &Commission{
		ID:          storageCommission.ID,
		CampaignID:  storageCommission.CampaignID,
		Title:       storageCommission.Title,
		Description: storageCommission.Description,
		Domain:      storageCommission.Domain,
		Context:     storageCommission.Context,
		Status:      storageCommission.Status,
		CreatedAt:   storageCommission.CreatedAt,
	}, nil
}

func (a *commissionRepositoryAdapter) UpdateCommissionStatus(ctx context.Context, id, status string) error {
	return a.repo.UpdateCommissionStatus(ctx, id, status)
}

func (a *commissionRepositoryAdapter) DeleteCommission(ctx context.Context, id string) error {
	return a.repo.DeleteCommission(ctx, id)
}

func (a *commissionRepositoryAdapter) ListCommissionsByCampaign(ctx context.Context, campaignID string) ([]*Commission, error) {
	storageCommissions, err := a.repo.ListCommissionsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	commissions := make([]*Commission, 0, len(storageCommissions))
	for _, sc := range storageCommissions {
		commissions = append(commissions, &Commission{
			ID:          sc.ID,
			CampaignID:  sc.CampaignID,
			Title:       sc.Title,
			Description: sc.Description,
			Domain:      sc.Domain,
			Context:     sc.Context,
			Status:      sc.Status,
			CreatedAt:   sc.CreatedAt,
		})
	}
	return commissions, nil
}

// promptChainRepositoryAdapter adapts storage.PromptChainRepository to registry.PromptChainRepository
type promptChainRepositoryAdapter struct {
	repo storage.PromptChainRepository
}

func (a *promptChainRepositoryAdapter) CreateChain(ctx context.Context, chain *PromptChain) error {
	// Convert registry.PromptChain to storage.PromptChain
	storageChain := &storage.PromptChain{
		ID:        chain.ID,
		AgentID:   chain.AgentID,
		TaskID:    chain.TaskID,
		CreatedAt: chain.CreatedAt,
		UpdatedAt: chain.UpdatedAt,
	}
	return a.repo.CreateChain(ctx, storageChain)
}

func (a *promptChainRepositoryAdapter) GetChain(ctx context.Context, id string) (*PromptChain, error) {
	storageChain, err := a.repo.GetChain(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert storage.PromptChain to registry.PromptChain
	chain := &PromptChain{
		ID:        storageChain.ID,
		AgentID:   storageChain.AgentID,
		TaskID:    storageChain.TaskID,
		CreatedAt: storageChain.CreatedAt,
		UpdatedAt: storageChain.UpdatedAt,
		Messages:  make([]*PromptChainMessage, 0, len(storageChain.Messages)),
	}

	// Convert messages
	for _, msg := range storageChain.Messages {
		chain.Messages = append(chain.Messages, &PromptChainMessage{
			ID:         msg.ID,
			ChainID:    msg.ChainID,
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			Timestamp:  msg.Timestamp,
			TokenUsage: msg.TokenUsage,
		})
	}

	return chain, nil
}

func (a *promptChainRepositoryAdapter) AddMessage(ctx context.Context, chainID string, message *PromptChainMessage) error {
	// Convert registry.PromptChainMessage to storage.PromptChainMessage
	storageMsg := &storage.PromptChainMessage{
		ID:         message.ID,
		ChainID:    message.ChainID,
		Role:       message.Role,
		Content:    message.Content,
		Name:       message.Name,
		Timestamp:  message.Timestamp,
		TokenUsage: message.TokenUsage,
	}
	return a.repo.AddMessage(ctx, chainID, storageMsg)
}

func (a *promptChainRepositoryAdapter) GetChainsByAgent(ctx context.Context, agentID string) ([]*PromptChain, error) {
	storageChains, err := a.repo.GetChainsByAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}

	chains := make([]*PromptChain, 0, len(storageChains))
	for _, sc := range storageChains {
		chains = append(chains, &PromptChain{
			ID:        sc.ID,
			AgentID:   sc.AgentID,
			TaskID:    sc.TaskID,
			CreatedAt: sc.CreatedAt,
			UpdatedAt: sc.UpdatedAt,
		})
	}
	return chains, nil
}

func (a *promptChainRepositoryAdapter) GetChainsByTask(ctx context.Context, taskID string) ([]*PromptChain, error) {
	storageChains, err := a.repo.GetChainsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	chains := make([]*PromptChain, 0, len(storageChains))
	for _, sc := range storageChains {
		chains = append(chains, &PromptChain{
			ID:        sc.ID,
			AgentID:   sc.AgentID,
			TaskID:    sc.TaskID,
			CreatedAt: sc.CreatedAt,
			UpdatedAt: sc.UpdatedAt,
		})
	}
	return chains, nil
}

func (a *promptChainRepositoryAdapter) DeleteChain(ctx context.Context, id string) error {
	return a.repo.DeleteChain(ctx, id)
}

// taskRepositoryAdapter adapts storage.TaskRepository to registry.TaskRepository
type taskRepositoryAdapter struct {
	repo storage.TaskRepository
}

func (a *taskRepositoryAdapter) CreateTask(ctx context.Context, task *StorageTask) error {
	// Convert registry.StorageTask to storage.Task
	storageTask := &storage.Task{
		ID:              task.ID,
		CommissionID:    task.CommissionID,
		AssignedAgentID: task.AssignedAgentID,
		Title:           task.Title,
		Description:     task.Description,
		Status:          task.Status,
		Column:          task.Column,
		StoryPoints:     task.StoryPoints,
		Metadata:        task.Metadata,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
	}
	return a.repo.CreateTask(ctx, storageTask)
}

func (a *taskRepositoryAdapter) GetTask(ctx context.Context, id string) (*StorageTask, error) {
	storageTask, err := a.repo.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert storage.Task to registry.StorageTask
	return &StorageTask{
		ID:              storageTask.ID,
		CommissionID:    storageTask.CommissionID,
		AssignedAgentID: storageTask.AssignedAgentID,
		Title:           storageTask.Title,
		Description:     storageTask.Description,
		Status:          storageTask.Status,
		Column:          storageTask.Column,
		StoryPoints:     storageTask.StoryPoints,
		Metadata:        storageTask.Metadata,
		CreatedAt:       storageTask.CreatedAt,
		UpdatedAt:       storageTask.UpdatedAt,
		AgentName:       storageTask.AgentName,
		AgentType:       storageTask.AgentType,
	}, nil
}

func (a *taskRepositoryAdapter) UpdateTask(ctx context.Context, task *StorageTask) error {
	storageTask := &storage.Task{
		ID:              task.ID,
		CommissionID:    task.CommissionID,
		AssignedAgentID: task.AssignedAgentID,
		Title:           task.Title,
		Description:     task.Description,
		Status:          task.Status,
		Column:          task.Column,
		StoryPoints:     task.StoryPoints,
		Metadata:        task.Metadata,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
	}
	return a.repo.UpdateTask(ctx, storageTask)
}

func (a *taskRepositoryAdapter) DeleteTask(ctx context.Context, id string) error {
	return a.repo.DeleteTask(ctx, id)
}

func (a *taskRepositoryAdapter) ListTasks(ctx context.Context) ([]*StorageTask, error) {
	storageTasks, err := a.repo.ListTasks(ctx)
	if err != nil {
		return nil, err
	}

	tasks := make([]*StorageTask, 0, len(storageTasks))
	for _, st := range storageTasks {
		tasks = append(tasks, &StorageTask{
			ID:              st.ID,
			CommissionID:    st.CommissionID,
			AssignedAgentID: st.AssignedAgentID,
			Title:           st.Title,
			Description:     st.Description,
			Status:          st.Status,
			Column:          st.Column,
			StoryPoints:     st.StoryPoints,
			Metadata:        st.Metadata,
			CreatedAt:       st.CreatedAt,
			UpdatedAt:       st.UpdatedAt,
			AgentName:       st.AgentName,
			AgentType:       st.AgentType,
		})
	}
	return tasks, nil
}

func (a *taskRepositoryAdapter) ListTasksByStatus(ctx context.Context, status string) ([]*StorageTask, error) {
	storageTasks, err := a.repo.ListTasksByStatus(ctx, status)
	if err != nil {
		return nil, err
	}

	tasks := make([]*StorageTask, 0, len(storageTasks))
	for _, st := range storageTasks {
		tasks = append(tasks, &StorageTask{
			ID:              st.ID,
			CommissionID:    st.CommissionID,
			AssignedAgentID: st.AssignedAgentID,
			Title:           st.Title,
			Description:     st.Description,
			Status:          st.Status,
			Column:          st.Column,
			StoryPoints:     st.StoryPoints,
			Metadata:        st.Metadata,
			CreatedAt:       st.CreatedAt,
			UpdatedAt:       st.UpdatedAt,
			AgentName:       st.AgentName,
			AgentType:       st.AgentType,
		})
	}
	return tasks, nil
}

func (a *taskRepositoryAdapter) ListTasksByCommission(ctx context.Context, commissionID string) ([]*StorageTask, error) {
	storageTasks, err := a.repo.ListTasksByCommission(ctx, commissionID)
	if err != nil {
		return nil, err
	}

	tasks := make([]*StorageTask, 0, len(storageTasks))
	for _, st := range storageTasks {
		tasks = append(tasks, &StorageTask{
			ID:              st.ID,
			CommissionID:    st.CommissionID,
			AssignedAgentID: st.AssignedAgentID,
			Title:           st.Title,
			Description:     st.Description,
			Status:          st.Status,
			Column:          st.Column,
			StoryPoints:     st.StoryPoints,
			Metadata:        st.Metadata,
			CreatedAt:       st.CreatedAt,
			UpdatedAt:       st.UpdatedAt,
			AgentName:       st.AgentName,
			AgentType:       st.AgentType,
		})
	}
	return tasks, nil
}

func (a *taskRepositoryAdapter) ListTasksForKanban(ctx context.Context, commissionID string) ([]*StorageTask, error) {
	storageTasks, err := a.repo.ListTasksForKanban(ctx, commissionID)
	if err != nil {
		return nil, err
	}

	tasks := make([]*StorageTask, 0, len(storageTasks))
	for _, st := range storageTasks {
		tasks = append(tasks, &StorageTask{
			ID:              st.ID,
			CommissionID:    st.CommissionID,
			AssignedAgentID: st.AssignedAgentID,
			Title:           st.Title,
			Description:     st.Description,
			Status:          st.Status,
			Column:          st.Column,
			StoryPoints:     st.StoryPoints,
			Metadata:        st.Metadata,
			CreatedAt:       st.CreatedAt,
			UpdatedAt:       st.UpdatedAt,
			AgentName:       st.AgentName,
			AgentType:       st.AgentType,
		})
	}
	return tasks, nil
}

func (a *taskRepositoryAdapter) AssignTask(ctx context.Context, taskID, agentID string) error {
	return a.repo.AssignTask(ctx, taskID, agentID)
}

func (a *taskRepositoryAdapter) UpdateTaskStatus(ctx context.Context, taskID, status string) error {
	return a.repo.UpdateTaskStatus(ctx, taskID, status)
}

func (a *taskRepositoryAdapter) UpdateTaskColumn(ctx context.Context, taskID, column string) error {
	// The storage package doesn't have UpdateTaskColumn, so we'll need to do a full update
	task, err := a.repo.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	task.Column = column
	return a.repo.UpdateTask(ctx, task)
}

func (a *taskRepositoryAdapter) RecordTaskEvent(ctx context.Context, event *TaskEvent) error {
	storageEvent := &storage.TaskEvent{
		ID:        event.ID,
		TaskID:    event.TaskID,
		AgentID:   event.AgentID,
		EventType: event.EventType,
		OldValue:  event.OldValue,
		NewValue:  event.NewValue,
		Reason:    event.Reason,
		CreatedAt: event.CreatedAt,
	}
	return a.repo.RecordTaskEvent(ctx, storageEvent)
}

func (a *taskRepositoryAdapter) GetTaskHistory(ctx context.Context, taskID string) ([]*TaskEvent, error) {
	storageEvents, err := a.repo.GetTaskHistory(ctx, taskID)
	if err != nil {
		return nil, err
	}

	events := make([]*TaskEvent, 0, len(storageEvents))
	for _, se := range storageEvents {
		events = append(events, &TaskEvent{
			ID:        se.ID,
			TaskID:    se.TaskID,
			AgentID:   se.AgentID,
			EventType: se.EventType,
			OldValue:  se.OldValue,
			NewValue:  se.NewValue,
			Reason:    se.Reason,
			CreatedAt: se.CreatedAt,
		})
	}
	return events, nil
}

func (a *taskRepositoryAdapter) GetAgentWorkload(ctx context.Context) ([]*AgentWorkload, error) {
	storageWorkloads, err := a.repo.GetAgentWorkload(ctx)
	if err != nil {
		return nil, err
	}

	workloads := make([]*AgentWorkload, 0, len(storageWorkloads))
	for _, sw := range storageWorkloads {
		workloads = append(workloads, &AgentWorkload{
			ID:          sw.ID,
			Name:        sw.Name,
			TaskCount:   sw.TaskCount,
			ActiveTasks: sw.ActiveTasks,
		})
	}
	return workloads, nil
}

// agentRepositoryAdapter adapts storage.AgentRepository to registry.AgentRepository
type agentRepositoryAdapter struct {
	repo storage.AgentRepository
}

func (a *agentRepositoryAdapter) CreateAgent(ctx context.Context, agent *StorageAgent) error {
	// Convert registry.StorageAgent to storage.Agent
	storageAgent := &storage.Agent{
		ID:            agent.ID,
		Name:          agent.Name,
		Type:          agent.Type,
		Provider:      agent.Provider,
		Model:         agent.Model,
		Capabilities:  agent.Capabilities,
		Tools:         agent.Tools,
		CostMagnitude: agent.CostMagnitude,
		CreatedAt:     agent.CreatedAt,
	}
	return a.repo.CreateAgent(ctx, storageAgent)
}

func (a *agentRepositoryAdapter) GetAgent(ctx context.Context, id string) (*StorageAgent, error) {
	storageAgent, err := a.repo.GetAgent(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert storage.Agent to registry.StorageAgent
	return &StorageAgent{
		ID:            storageAgent.ID,
		Name:          storageAgent.Name,
		Type:          storageAgent.Type,
		Provider:      storageAgent.Provider,
		Model:         storageAgent.Model,
		Capabilities:  storageAgent.Capabilities,
		Tools:         storageAgent.Tools,
		CostMagnitude: storageAgent.CostMagnitude,
		CreatedAt:     storageAgent.CreatedAt,
	}, nil
}

func (a *agentRepositoryAdapter) UpdateAgent(ctx context.Context, agent *StorageAgent) error {
	storageAgent := &storage.Agent{
		ID:            agent.ID,
		Name:          agent.Name,
		Type:          agent.Type,
		Provider:      agent.Provider,
		Model:         agent.Model,
		Capabilities:  agent.Capabilities,
		Tools:         agent.Tools,
		CostMagnitude: agent.CostMagnitude,
		CreatedAt:     agent.CreatedAt,
	}
	return a.repo.UpdateAgent(ctx, storageAgent)
}

func (a *agentRepositoryAdapter) DeleteAgent(ctx context.Context, id string) error {
	return a.repo.DeleteAgent(ctx, id)
}

func (a *agentRepositoryAdapter) ListAgents(ctx context.Context) ([]*StorageAgent, error) {
	storageAgents, err := a.repo.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	agents := make([]*StorageAgent, 0, len(storageAgents))
	for _, sa := range storageAgents {
		agents = append(agents, &StorageAgent{
			ID:            sa.ID,
			Name:          sa.Name,
			Type:          sa.Type,
			Provider:      sa.Provider,
			Model:         sa.Model,
			Capabilities:  sa.Capabilities,
			Tools:         sa.Tools,
			CostMagnitude: sa.CostMagnitude,
			CreatedAt:     sa.CreatedAt,
		})
	}
	return agents, nil
}

func (a *agentRepositoryAdapter) ListAgentsByType(ctx context.Context, agentType string) ([]*StorageAgent, error) {
	storageAgents, err := a.repo.ListAgentsByType(ctx, agentType)
	if err != nil {
		return nil, err
	}

	agents := make([]*StorageAgent, 0, len(storageAgents))
	for _, sa := range storageAgents {
		agents = append(agents, &StorageAgent{
			ID:            sa.ID,
			Name:          sa.Name,
			Type:          sa.Type,
			Provider:      sa.Provider,
			Model:         sa.Model,
			Capabilities:  sa.Capabilities,
			Tools:         sa.Tools,
			CostMagnitude: sa.CostMagnitude,
			CreatedAt:     sa.CreatedAt,
		})
	}
	return agents, nil
}

func (s *SQLiteStorageRegistry) RegisterTaskRepository(repo TaskRepository) error {
	// Not needed for SQLite - repositories are created internally
	return nil
}

func (s *SQLiteStorageRegistry) GetTaskRepository() TaskRepository {
	// Get the task repository from the underlying storage registry
	if s.registry != nil {
		storageRepo := s.registry.GetTaskRepository()
		if storageRepo != nil {
			// Wrap the storage repository with an adapter
			return &taskRepositoryAdapter{repo: storageRepo}
		}
	}
	return nil
}

func (s *SQLiteStorageRegistry) RegisterCampaignRepository(repo CampaignRepository) error {
	// Not needed for SQLite - repositories are created internally
	return nil
}

func (s *SQLiteStorageRegistry) GetCampaignRepository() CampaignRepository {
	// Get the campaign repository from the underlying storage registry
	if s.registry != nil {
		storageRepo := s.registry.GetCampaignRepository()
		if storageRepo != nil {
			// Wrap the storage repository with an adapter
			return &campaignRepositoryAdapter{repo: storageRepo}
		}
	}
	return nil
}

func (s *SQLiteStorageRegistry) RegisterCommissionRepository(repo CommissionRepository) error {
	// Not needed for SQLite - repositories are created internally
	return nil
}

func (s *SQLiteStorageRegistry) GetCommissionRepository() CommissionRepository {
	// Get the commission repository from the underlying storage registry
	if s.registry != nil {
		storageRepo := s.registry.GetCommissionRepository()
		if storageRepo != nil {
			// Wrap the storage repository with an adapter
			return &commissionRepositoryAdapter{repo: storageRepo}
		}
	}
	return nil
}

func (s *SQLiteStorageRegistry) RegisterAgentRepository(repo AgentRepository) error {
	// Not needed for SQLite - repositories are created internally
	return nil
}

func (s *SQLiteStorageRegistry) GetAgentRepository() AgentRepository {
	// Get the agent repository from the underlying storage registry
	if s.registry != nil {
		storageRepo := s.registry.GetAgentRepository()
		if storageRepo != nil {
			// Wrap the storage repository with an adapter
			return &agentRepositoryAdapter{repo: storageRepo}
		}
	}
	return nil
}

func (s *SQLiteStorageRegistry) RegisterPromptChainRepository(repo PromptChainRepository) error {
	// Not needed for SQLite - repositories are created internally
	return nil
}

func (s *SQLiteStorageRegistry) GetPromptChainRepository() PromptChainRepository {
	// Get the prompt chain repository from the underlying storage registry
	if s.registry != nil {
		storageRepo := s.registry.GetPromptChainRepository()
		if storageRepo != nil {
			// Wrap the storage repository with an adapter
			return &promptChainRepositoryAdapter{repo: storageRepo}
		}
	}
	return nil
}

func (s *SQLiteStorageRegistry) GetMemoryStore() MemoryStore {
	return s.memoryStore
}

func (s *SQLiteStorageRegistry) SetMemoryStore(store MemoryStore) {
	s.memoryStore = store
}

// GetStorageRegistry returns the underlying storage.StorageRegistry for components that need it
func (s *SQLiteStorageRegistry) GetStorageRegistry() storage.StorageRegistry {
	return s.registry
}

// Kanban repository adapter methods
func (s *SQLiteStorageRegistry) GetBoardRepository() KanbanBoardRepository {
	return s.boardAdapter
}

func (s *SQLiteStorageRegistry) GetKanbanTaskRepository() KanbanTaskRepository {
	return s.taskAdapter
}

func (s *SQLiteStorageRegistry) GetKanbanCampaignRepository() KanbanCampaignRepository {
	return s.campaignAdapter
}

func (s *SQLiteStorageRegistry) GetKanbanCommissionRepository() KanbanCommissionRepository {
	return s.commissionAdapter
}

// SetRegistry sets the underlying storage registry (for testing)
func (s *SQLiteStorageRegistry) SetRegistry(registry storage.StorageRegistry) {
	s.registry = registry
}

// SetAdapters sets the kanban adapters (for testing)
func (s *SQLiteStorageRegistry) SetAdapters(taskAdapter *storage.KanbanTaskRepositoryAdapter, boardAdapter *storage.KanbanBoardRepositoryAdapter, campaignAdapter *storage.KanbanCampaignRepositoryAdapter, commissionAdapter *storage.KanbanCommissionRepositoryAdapter) {
	s.taskAdapter = taskAdapter
	s.boardAdapter = boardAdapter
	s.campaignAdapter = campaignAdapter
	s.commissionAdapter = commissionAdapter
}

// NewComponentRegistry creates a new ComponentRegistry instance
func NewComponentRegistry() ComponentRegistry {
	return &DefaultComponentRegistry{
		agentRegistry:        NewAgentRegistry(),
		toolRegistry:         NewToolRegistry(),
		providerRegistry:     NewProviderRegistry(),
		memoryRegistry:       NewMemoryRegistry(),
		projectRegistry:      NewProjectRegistry(),
		promptRegistry:       NewPromptRegistry(),
		storageRegistry:      NewStorageRegistry(),
		orchestratorRegistry: nil, // Will be initialized when needed
	}
}

// Agents returns the agent registry
func (r *DefaultComponentRegistry) Agents() AgentRegistry {
	return r.agentRegistry
}

// Tools returns the tool registry
func (r *DefaultComponentRegistry) Tools() ToolRegistry {
	return r.toolRegistry
}

// Providers returns the provider registry
func (r *DefaultComponentRegistry) Providers() ProviderRegistry {
	return r.providerRegistry
}

// Memory returns the memory registry
func (r *DefaultComponentRegistry) Memory() MemoryRegistry {
	return r.memoryRegistry
}

// Project returns the project registry
func (r *DefaultComponentRegistry) Project() ProjectRegistry {
	return r.projectRegistry
}

// Prompts returns the prompt registry
func (r *DefaultComponentRegistry) Prompts() *PromptRegistry {
	return r.promptRegistry
}

// GetPromptManager returns the configured layered prompt manager
func (r *DefaultComponentRegistry) GetPromptManager() (LayeredPromptManager, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.layeredPromptManager == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "layered prompt manager not initialized", nil).
			WithComponent("registry").
			WithOperation("GetPromptManager")
	}

	return r.layeredPromptManager, nil
}

// Storage returns the storage registry
func (r *DefaultComponentRegistry) Storage() StorageRegistry {
	return r.storageRegistry
}

// SetStorageRegistry sets the storage registry (used for testing)
func (r *DefaultComponentRegistry) SetStorageRegistry(storageReg StorageRegistry, memoryStore MemoryStore) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.storageRegistry = storageReg

	// Also set the memory store in the storage registry if it's the default type
	if defaultStorageReg, ok := r.storageRegistry.(*DefaultStorageRegistry); ok {
		defaultStorageReg.SetMemoryStore(memoryStore)
	}
}

// Orchestrator returns the orchestrator registry
func (r *DefaultComponentRegistry) Orchestrator() interface{} {
	return r.orchestratorRegistry
}

// Initialize sets up all registries with the provided configuration
func (r *DefaultComponentRegistry) Initialize(ctx context.Context, config Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.config = config

	// Initialize project context first as other components may depend on it
	ctx, err := r.initializeProject(ctx)
	if err != nil {
		// Project initialization is optional - just log and continue
		// This allows the system to work without a project
		_ = err // Suppress error, project is optional
	}

	// Initialize each registry
	if err := r.initializeAgents(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize agents").
			WithComponent("registry").
			WithOperation("Initialize")
	}

	if err := r.initializeTools(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize tools").
			WithComponent("registry").
			WithOperation("Initialize")
	}

	if err := r.initializeProviders(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize providers").
			WithComponent("registry").
			WithOperation("Initialize")
	}

	if err := r.initializeMemory(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize memory").
			WithComponent("registry").
			WithOperation("Initialize")
	}

	if err := r.initializeStorage(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize storage").
			WithComponent("registry").
			WithOperation("Initialize")
	}

	if err := r.initializePrompts(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize prompts").
			WithComponent("registry").
			WithOperation("Initialize")
	}

	r.initialized = true
	return nil
}

// Shutdown cleanly shuts down all registries and their components
func (r *DefaultComponentRegistry) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errors []error

	// Shutdown each registry
	if err := r.shutdownMemory(ctx); err != nil {
		errors = append(errors, gerror.Wrap(err, gerror.ErrCodeInternal, "memory shutdown error").
			WithComponent("registry").
			WithOperation("Shutdown"))
	}

	if err := r.shutdownProviders(ctx); err != nil {
		errors = append(errors, gerror.Wrap(err, gerror.ErrCodeInternal, "providers shutdown error").
			WithComponent("registry").
			WithOperation("Shutdown"))
	}

	if err := r.shutdownTools(ctx); err != nil {
		errors = append(errors, gerror.Wrap(err, gerror.ErrCodeInternal, "tools shutdown error").
			WithComponent("registry").
			WithOperation("Shutdown"))
	}

	if err := r.shutdownAgents(ctx); err != nil {
		errors = append(errors, gerror.Wrap(err, gerror.ErrCodeInternal, "agents shutdown error").
			WithComponent("registry").
			WithOperation("Shutdown"))
	}

	r.initialized = false

	if len(errors) > 0 {
		return gerror.Newf(gerror.ErrCodeInternal, "shutdown errors: %v", errors).
			WithComponent("registry").
			WithOperation("Shutdown")
	}

	return nil
}

func (r *DefaultComponentRegistry) initializeAgents(ctx context.Context) error {
	// Register default agent types
	if agentReg, ok := r.agentRegistry.(*DefaultAgentRegistry); ok {
		// Create the agent factory with dependencies
		agentFactory, err := r.createAgentFactory(ctx)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agent factory").
				WithComponent("registry").
				WithOperation("initializeAgents")
		}

		// Register default agent type factory
		if r.config.Agents.DefaultType != "" {
			err = agentReg.RegisterAgentType(r.config.Agents.DefaultType, agentFactory)
			if err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register default agent type").
					WithComponent("registry").
					WithOperation("initializeAgentRegistry")
			}
		}

		// Load agents from guild configuration if available
		if err := r.loadGuildAgents(ctx, agentReg); err != nil {
			// Log warning but don't fail - guild config is optional
			_ = err // Suppress error for now
		}
	}

	return nil
}

func (r *DefaultComponentRegistry) initializeTools(ctx context.Context) error {
	// Initialize enabled tools based on configuration
	for _, toolName := range r.config.Tools.EnabledTools {
		// Here you would create and register the actual tool instances
		// This is where you'd integrate with your existing tool implementations
		_ = toolName // Suppress unused variable warning
	}

	// Register basic tools with cost information
	if toolReg, ok := r.toolRegistry.(*DefaultToolRegistry); ok {
		if err := toolReg.RegisterBasicTools(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register basic tools").
				WithComponent("registry").
				WithOperation("initializeTools")
		}
	}

	// Register filesystem tools
	if err := RegisterFSTools(r.toolRegistry); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register filesystem tools").
			WithComponent("registry").
			WithOperation("initializeTools")
	}

	// Register code tools
	if err := RegisterCodeTools(r.toolRegistry); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register code tools").
			WithComponent("registry").
			WithOperation("initializeTools")
	}

	// Register jump tools
	if err := RegisterJumpTools(r.toolRegistry); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register jump tools").
			WithComponent("registry").
			WithOperation("initializeTools")
	}

	// Register dev tools - skip for now due to interface mismatch
	// TODO: Fix RegisterDevTools to work with current provider interfaces
	// if defaultProvider, err := r.providerRegistry.GetDefaultProvider(); err == nil {
	//	if err := RegisterDevTools(r.toolRegistry, defaultProvider); err != nil {
	//		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register dev tools").
	//			WithComponent("registry").
	//			WithOperation("initializeTools")
	//	}
	// }

	return nil
}

func (r *DefaultComponentRegistry) initializeProviders(ctx context.Context) error {
	// Create provider factory
	factory := providers.NewFactory()

	// Register all configured providers
	err := factory.RegisterProvidersWithRegistry(r.providerRegistry, r.config.Providers.Providers)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register providers").
			WithComponent("registry").
			WithOperation("initializeProviders")
	}

	// Set default provider if configured
	if r.config.Providers.DefaultProvider != "" {
		err := r.providerRegistry.SetDefaultProvider(r.config.Providers.DefaultProvider)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to set default provider").
				WithComponent("registry").
				WithOperation("initializeProviders")
		}
	}

	return nil
}

func (r *DefaultComponentRegistry) initializeMemory(ctx context.Context) error {
	// Initialize memory stores based on configuration
	for storeName, storeConfig := range r.config.Memory.Stores {
		// Here you would create and register memory store instances
		_ = storeName
		_ = storeConfig
	}
	return nil
}

func (r *DefaultComponentRegistry) initializeStorage(ctx context.Context) error {
	// Get project context to load guild configuration
	projectCtx, err := r.projectRegistry.GetCurrentContext(ctx)
	if err != nil {
		// No project context available, use default SQLite backend
		return r.initializeSQLiteStorage(ctx, filepath.Join(paths.DefaultCampaignDir, "memory.db"))
	}

	// Load guild configuration to determine storage backend
	guildConfig, err := config.LoadGuildConfig(ctx, (*projectCtx).GetRootPath())
	if err != nil {
		// If guild config fails to load, default to SQLite
		return r.initializeSQLiteStorage(ctx, filepath.Join((*projectCtx).GetGuildPath(), "memory.db"))
	}

	// Initialize SQLite storage (BoltDB no longer supported)
	dbPath := filepath.Join((*projectCtx).GetGuildPath(), guildConfig.GetEffectiveSQLitePath())
	// Make path relative to guild directory
	if !filepath.IsAbs(guildConfig.GetEffectiveSQLitePath()) {
		dbPath = filepath.Join((*projectCtx).GetGuildPath(), guildConfig.GetEffectiveSQLitePath())
	} else {
		dbPath = guildConfig.GetEffectiveSQLitePath()
	}
	return r.initializeSQLiteStorage(ctx, dbPath)
}

func (r *DefaultComponentRegistry) initializeSQLiteStorage(ctx context.Context, dbPath string) error {

	// Initialize SQLite storage using the storage package's initialization function
	storageReg, memoryStoreAdapter, err := storage.InitializeSQLiteStorageForRegistry(ctx, dbPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize SQLite storage").
			WithComponent("registry").
			WithOperation("initializeSQLiteStorage").
			WithDetails("dbPath", dbPath)
	}

	// Replace the placeholder storage registry with the real one
	// Cast to storage.StorageRegistry first, then wrap with our own interface
	if sqliteReg, ok := storageReg.(storage.StorageRegistry); ok {
		// Create kanban adapters
		taskAdapter := storage.NewKanbanTaskRepositoryAdapter(sqliteReg.GetTaskRepository())
		boardAdapter := storage.NewKanbanBoardRepositoryAdapter(sqliteReg.GetBoardRepository())
		campaignAdapter := storage.NewKanbanCampaignRepositoryAdapter(sqliteReg.GetCampaignRepository())
		commissionAdapter := storage.NewKanbanCommissionRepositoryAdapter(sqliteReg.GetCommissionRepository())

		r.storageRegistry = &SQLiteStorageRegistry{
			registry:          sqliteReg,
			memoryStore:       memoryStoreAdapter.(MemoryStore),
			taskAdapter:       taskAdapter,
			boardAdapter:      boardAdapter,
			campaignAdapter:   campaignAdapter,
			commissionAdapter: commissionAdapter,
		}
	} else {
		return gerror.New(gerror.ErrCodeInternal, "unexpected storage registry type", nil).
			WithComponent("registry").
			WithOperation("initializeSQLiteStorage")
	}

	// SQLite storage registry already has the memory store set in the struct above

	return nil
}

func (r *DefaultComponentRegistry) shutdownAgents(ctx context.Context) error {
	// Shutdown agents if needed
	return nil
}

func (r *DefaultComponentRegistry) shutdownTools(ctx context.Context) error {
	// Shutdown tools if needed
	return nil
}

func (r *DefaultComponentRegistry) shutdownProviders(ctx context.Context) error {
	// Shutdown providers if needed
	return nil
}

func (r *DefaultComponentRegistry) shutdownMemory(ctx context.Context) error {
	// Shutdown memory stores if needed
	return nil
}

func (r *DefaultComponentRegistry) initializeProject(ctx context.Context) (context.Context, error) {
	// Try to add project context to the context
	return r.projectRegistry.WithProjectContext(ctx)
}

func (r *DefaultComponentRegistry) initializePrompts(ctx context.Context) error {
	// Register default prompt provider
	defaultProvider, err := NewDefaultPromptProvider()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "error creating default prompt provider").
			WithComponent("registry").
			WithOperation("initializePrompts")
	}

	if err := r.promptRegistry.Register("default", defaultProvider); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "error registering default prompt provider").
			WithComponent("registry").
			WithOperation("initializePrompts")
	}

	// Initialize layered prompt manager
	// TODO: Create a proper layered prompt manager with Guild storage and dependencies
	// For now, create a basic manager and wrap it to implement LayeredManager interface
	basicManager := layered.NewLayeredPromptManager()
	r.layeredPromptManager = &layeredManagerWrapper{
		manager: basicManager,
	}

	// TODO: Add any other prompt providers from config

	return nil
}

// loadGuildAgents loads agents from guild configuration
func (r *DefaultComponentRegistry) loadGuildAgents(ctx context.Context, agentReg *DefaultAgentRegistry) error {
	// Try to get project context and load guild configuration
	projectCtx, err := r.projectRegistry.GetCurrentContext(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get project context").
			WithComponent("registry").
			WithOperation("loadGuildAgents")
	}

	// Load guild configuration
	guildConfig, err := config.LoadGuildConfig(ctx, (*projectCtx).GetRootPath())
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load guild config").
			WithComponent("registry").
			WithOperation("loadGuildAgents")
	}

	// Register each agent with the registry
	for _, agent := range guildConfig.Agents {
		guildAgent := GuildAgentConfig{
			ID:            agent.ID,
			Name:          agent.Name,
			Type:          agent.Type,
			Provider:      agent.Provider,
			Model:         agent.Model,
			SystemPrompt:  agent.SystemPrompt,
			Capabilities:  agent.Capabilities,
			Tools:         agent.Tools,
			CostMagnitude: agent.CostMagnitude,
			ContextWindow: agent.ContextWindow,
		}

		if err := agentReg.RegisterGuildAgent(guildAgent); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to register agent %s", agent.ID).
				WithComponent("registry").
				WithOperation("loadGuildAgents").
				WithDetails("agentID", agent.ID)
		}
	}

	return nil
}

// GetAgentsByCost provides access to cost-based agent selection
func (r *DefaultComponentRegistry) GetAgentsByCost(maxCost int) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if agentReg, ok := r.agentRegistry.(*DefaultAgentRegistry); ok {
		return agentReg.GetAgentsByCost(maxCost)
	}
	return nil
}

// GetCheapestAgentByCapability provides access to cost-optimal agent selection
func (r *DefaultComponentRegistry) GetCheapestAgentByCapability(capability string) (*AgentInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if agentReg, ok := r.agentRegistry.(*DefaultAgentRegistry); ok {
		return agentReg.GetCheapestAgentByCapability(capability)
	}
	return nil, gerror.New(gerror.ErrCodeInternal, "agent registry not properly initialized", nil).
		WithComponent("registry").
		WithOperation("GetCheapestAgentByCapability")
}

// GetToolsByCost provides access to cost-based tool selection
func (r *DefaultComponentRegistry) GetToolsByCost(maxCost int) []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if toolReg, ok := r.toolRegistry.(*DefaultToolRegistry); ok {
		return toolReg.GetToolsByCost(maxCost)
	}
	return nil
}

// GetCheapestToolByCapability provides access to cost-optimal tool selection
func (r *DefaultComponentRegistry) GetCheapestToolByCapability(capability string) (*ToolInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if toolReg, ok := r.toolRegistry.(*DefaultToolRegistry); ok {
		return toolReg.GetCheapestToolByCapability(capability)
	}
	return nil, gerror.New(gerror.ErrCodeInternal, "tool registry not properly initialized", nil).
		WithComponent("registry").
		WithOperation("GetCheapestToolByCapability")
}

// GetAgentsByCapability provides access to capability-based agent selection
func (r *DefaultComponentRegistry) GetAgentsByCapability(capability string) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if agentReg, ok := r.agentRegistry.(*DefaultAgentRegistry); ok {
		return agentReg.GetAgentsByCapability(capability)
	}
	return nil
}
