// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration

package commission_test

import (
	"context"
	"testing"

	"github.com/lancekrogers/guild/pkg/project"
	"github.com/lancekrogers/guild/pkg/registry"
	"github.com/lancekrogers/guild/pkg/storage"
	"github.com/stretchr/testify/require"
)

// setupTestRegistry creates a fully initialized registry with SQLite storage for tests
func setupTestRegistry(t *testing.T, ctx context.Context, projCtx *project.Context) registry.ComponentRegistry {
	t.Helper()

	// Create registry
	reg := registry.NewComponentRegistry()

	// Initialize SQLite storage using test helper
	storageReg, memStore, err := storage.InitializeSQLiteStorageForTests(ctx)
	require.NoError(t, err)

	// Wrap the storage registry to implement registry.StorageRegistry
	registryStorageWrapper := &testStorageRegistryWrapper{
		storageReg: storageReg,
		memStore:   memStore,
	}

	// Set storage in registry (cast to concrete type for testing)
	if defaultReg, ok := reg.(*registry.DefaultComponentRegistry); ok {
		defaultReg.SetStorageRegistry(registryStorageWrapper, &testMemoryStoreWrapper{memStore: memStore})
	}

	return reg
}

// testStorageRegistryWrapper wraps storage.StorageRegistry to implement registry.StorageRegistry
type testStorageRegistryWrapper struct {
	storageReg storage.StorageRegistry
	memStore   interface{}
}

func (w *testStorageRegistryWrapper) RegisterTaskRepository(repo registry.TaskRepository) error {
	// Not needed for tests - repositories are already registered
	return nil
}

func (w *testStorageRegistryWrapper) GetTaskRepository() registry.TaskRepository {
	// Return nil - tests should use the storage registry directly
	return nil
}

func (w *testStorageRegistryWrapper) RegisterCampaignRepository(repo registry.CampaignRepository) error {
	return nil
}

func (w *testStorageRegistryWrapper) GetCampaignRepository() registry.CampaignRepository {
	// Create adapter
	return &testCampaignRepoAdapter{repo: w.storageReg.GetCampaignRepository()}
}

func (w *testStorageRegistryWrapper) RegisterCommissionRepository(repo registry.CommissionRepository) error {
	return nil
}

func (w *testStorageRegistryWrapper) GetCommissionRepository() registry.CommissionRepository {
	// Create adapter
	return &testCommissionRepoAdapter{repo: w.storageReg.GetCommissionRepository()}
}

func (w *testStorageRegistryWrapper) RegisterAgentRepository(repo registry.AgentRepository) error {
	return nil
}

func (w *testStorageRegistryWrapper) GetAgentRepository() registry.AgentRepository {
	// Create adapter
	return &testAgentRepoAdapter{repo: w.storageReg.GetAgentRepository()}
}

func (w *testStorageRegistryWrapper) RegisterPromptChainRepository(repo registry.PromptChainRepository) error {
	return nil
}

func (w *testStorageRegistryWrapper) GetPromptChainRepository() registry.PromptChainRepository {
	return nil
}

func (w *testStorageRegistryWrapper) GetMemoryStore() registry.MemoryStore {
	return &testMemoryStoreWrapper{memStore: w.memStore}
}

func (w *testStorageRegistryWrapper) RegisterStorageRegistry(storageReg interface{}) error {
	// This is used to register the actual storage registry
	if sr, ok := storageReg.(storage.StorageRegistry); ok {
		w.storageReg = sr
	}
	return nil
}

func (w *testStorageRegistryWrapper) GetStorageRegistry() storage.StorageRegistry {
	return w.storageReg
}

func (w *testStorageRegistryWrapper) GetKanbanTaskRepository() registry.KanbanTaskRepository {
	// Return wrapper that implements the kanban interface
	return &testKanbanTaskRepoWrapper{taskRepo: w.storageReg.GetTaskRepository()}
}

func (w *testStorageRegistryWrapper) GetBoardRepository() registry.KanbanBoardRepository {
	// Return wrapper that implements the kanban board interface
	return &testKanbanBoardRepoWrapper{boardRepo: w.storageReg.GetBoardRepository()}
}

func (w *testStorageRegistryWrapper) GetKanbanCampaignRepository() registry.KanbanCampaignRepository {
	// Return wrapper that implements the kanban campaign interface
	return &testKanbanCampaignRepoWrapper{campaignRepo: w.storageReg.GetCampaignRepository()}
}

func (w *testStorageRegistryWrapper) GetKanbanCommissionRepository() registry.KanbanCommissionRepository {
	// Return wrapper that implements the kanban commission interface
	return &testKanbanCommissionRepoWrapper{commissionRepo: w.storageReg.GetCommissionRepository()}
}

// Adapter implementations
type testCampaignRepoAdapter struct {
	repo storage.CampaignRepository
}

func (a *testCampaignRepoAdapter) CreateCampaign(ctx context.Context, campaign *registry.Campaign) error {
	storageCampaign := &storage.Campaign{
		ID:        campaign.ID,
		Name:      campaign.Name,
		Status:    campaign.Status,
		CreatedAt: campaign.CreatedAt,
		UpdatedAt: campaign.UpdatedAt,
	}
	return a.repo.CreateCampaign(ctx, storageCampaign)
}

func (a *testCampaignRepoAdapter) GetCampaign(ctx context.Context, id string) (*registry.Campaign, error) {
	storageCampaign, err := a.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	return &registry.Campaign{
		ID:        storageCampaign.ID,
		Name:      storageCampaign.Name,
		Status:    storageCampaign.Status,
		CreatedAt: storageCampaign.CreatedAt,
		UpdatedAt: storageCampaign.UpdatedAt,
	}, nil
}

func (a *testCampaignRepoAdapter) UpdateCampaignStatus(ctx context.Context, id, status string) error {
	return a.repo.UpdateCampaignStatus(ctx, id, status)
}

func (a *testCampaignRepoAdapter) DeleteCampaign(ctx context.Context, id string) error {
	return a.repo.DeleteCampaign(ctx, id)
}

func (a *testCampaignRepoAdapter) ListCampaigns(ctx context.Context) ([]*registry.Campaign, error) {
	storageCampaigns, err := a.repo.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}

	campaigns := make([]*registry.Campaign, len(storageCampaigns))
	for i, sc := range storageCampaigns {
		campaigns[i] = &registry.Campaign{
			ID:        sc.ID,
			Name:      sc.Name,
			Status:    sc.Status,
			CreatedAt: sc.CreatedAt,
			UpdatedAt: sc.UpdatedAt,
		}
	}
	return campaigns, nil
}

type testCommissionRepoAdapter struct {
	repo storage.CommissionRepository
}

func (a *testCommissionRepoAdapter) CreateCommission(ctx context.Context, commission *registry.Commission) error {
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

func (a *testCommissionRepoAdapter) GetCommission(ctx context.Context, id string) (*registry.Commission, error) {
	storageCommission, err := a.repo.GetCommission(ctx, id)
	if err != nil {
		return nil, err
	}
	return &registry.Commission{
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

func (a *testCommissionRepoAdapter) UpdateCommissionStatus(ctx context.Context, id, status string) error {
	return a.repo.UpdateCommissionStatus(ctx, id, status)
}

func (a *testCommissionRepoAdapter) DeleteCommission(ctx context.Context, id string) error {
	return a.repo.DeleteCommission(ctx, id)
}

func (a *testCommissionRepoAdapter) ListCommissionsByCampaign(ctx context.Context, campaignID string) ([]*registry.Commission, error) {
	storageCommissions, err := a.repo.ListCommissionsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	commissions := make([]*registry.Commission, len(storageCommissions))
	for i, sc := range storageCommissions {
		commissions[i] = &registry.Commission{
			ID:          sc.ID,
			CampaignID:  sc.CampaignID,
			Title:       sc.Title,
			Description: sc.Description,
			Domain:      sc.Domain,
			Context:     sc.Context,
			Status:      sc.Status,
			CreatedAt:   sc.CreatedAt,
		}
	}
	return commissions, nil
}

type testAgentRepoAdapter struct {
	repo storage.AgentRepository
}

func (a *testAgentRepoAdapter) CreateAgent(ctx context.Context, agent *registry.StorageAgent) error {
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

func (a *testAgentRepoAdapter) GetAgent(ctx context.Context, id string) (*registry.StorageAgent, error) {
	storageAgent, err := a.repo.GetAgent(ctx, id)
	if err != nil {
		return nil, err
	}
	return &registry.StorageAgent{
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

func (a *testAgentRepoAdapter) UpdateAgent(ctx context.Context, agent *registry.StorageAgent) error {
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

func (a *testAgentRepoAdapter) DeleteAgent(ctx context.Context, id string) error {
	return a.repo.DeleteAgent(ctx, id)
}

func (a *testAgentRepoAdapter) ListAgents(ctx context.Context) ([]*registry.StorageAgent, error) {
	storageAgents, err := a.repo.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	agents := make([]*registry.StorageAgent, len(storageAgents))
	for i, sa := range storageAgents {
		agents[i] = &registry.StorageAgent{
			ID:            sa.ID,
			Name:          sa.Name,
			Type:          sa.Type,
			Provider:      sa.Provider,
			Model:         sa.Model,
			Capabilities:  sa.Capabilities,
			Tools:         sa.Tools,
			CostMagnitude: sa.CostMagnitude,
			CreatedAt:     sa.CreatedAt,
		}
	}
	return agents, nil
}

func (a *testAgentRepoAdapter) ListAgentsByType(ctx context.Context, agentType string) ([]*registry.StorageAgent, error) {
	storageAgents, err := a.repo.ListAgentsByType(ctx, agentType)
	if err != nil {
		return nil, err
	}

	agents := make([]*registry.StorageAgent, len(storageAgents))
	for i, sa := range storageAgents {
		agents[i] = &registry.StorageAgent{
			ID:            sa.ID,
			Name:          sa.Name,
			Type:          sa.Type,
			Provider:      sa.Provider,
			Model:         sa.Model,
			Capabilities:  sa.Capabilities,
			Tools:         sa.Tools,
			CostMagnitude: sa.CostMagnitude,
			CreatedAt:     sa.CreatedAt,
		}
	}
	return agents, nil
}

type testMemoryStoreWrapper struct {
	memStore interface{}
}

func (w *testMemoryStoreWrapper) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	if ms, ok := w.memStore.(interface {
		Get(context.Context, string, string) ([]byte, error)
	}); ok {
		return ms.Get(ctx, bucket, key)
	}
	return nil, nil
}

func (w *testMemoryStoreWrapper) Put(ctx context.Context, bucket, key string, value []byte) error {
	if ms, ok := w.memStore.(interface {
		Put(context.Context, string, string, []byte) error
	}); ok {
		return ms.Put(ctx, bucket, key, value)
	}
	return nil
}

func (w *testMemoryStoreWrapper) Delete(ctx context.Context, bucket, key string) error {
	if ms, ok := w.memStore.(interface {
		Delete(context.Context, string, string) error
	}); ok {
		return ms.Delete(ctx, bucket, key)
	}
	return nil
}

func (w *testMemoryStoreWrapper) List(ctx context.Context, bucket string) ([]string, error) {
	if ms, ok := w.memStore.(interface {
		List(context.Context, string) ([]string, error)
	}); ok {
		return ms.List(ctx, bucket)
	}
	return nil, nil
}

func (w *testMemoryStoreWrapper) ListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	if ms, ok := w.memStore.(interface {
		ListKeys(context.Context, string, string) ([]string, error)
	}); ok {
		return ms.ListKeys(ctx, bucket, prefix)
	}
	return nil, nil
}

func (w *testMemoryStoreWrapper) Close() error {
	if closer, ok := w.memStore.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// testKanbanTaskRepoWrapper wraps storage.TaskRepository to implement kanban interface
type testKanbanTaskRepoWrapper struct {
	taskRepo storage.TaskRepository
}

func (w *testKanbanTaskRepoWrapper) CreateTask(ctx context.Context, task interface{}) error {
	if t, ok := task.(*storage.Task); ok {
		return w.taskRepo.CreateTask(ctx, t)
	}
	return nil
}

func (w *testKanbanTaskRepoWrapper) UpdateTask(ctx context.Context, task interface{}) error {
	if t, ok := task.(*storage.Task); ok {
		return w.taskRepo.UpdateTask(ctx, t)
	}
	return nil
}

func (w *testKanbanTaskRepoWrapper) GetTask(ctx context.Context, id string) (interface{}, error) {
	return w.taskRepo.GetTask(ctx, id)
}

func (w *testKanbanTaskRepoWrapper) DeleteTask(ctx context.Context, id string) error {
	return w.taskRepo.DeleteTask(ctx, id)
}

func (w *testKanbanTaskRepoWrapper) ListTasksByBoard(ctx context.Context, boardID string) ([]interface{}, error) {
	tasks, err := w.taskRepo.ListTasksByBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(tasks))
	for i, task := range tasks {
		result[i] = task
	}
	return result, nil
}

func (w *testKanbanTaskRepoWrapper) RecordTaskEvent(ctx context.Context, event interface{}) error {
	if e, ok := event.(*storage.TaskEvent); ok {
		return w.taskRepo.RecordTaskEvent(ctx, e)
	}
	return nil
}

// Kanban repository wrappers
type testKanbanBoardRepoWrapper struct {
	boardRepo storage.BoardRepository
}

func (w *testKanbanBoardRepoWrapper) CreateBoard(ctx context.Context, board interface{}) error {
	if b, ok := board.(*storage.Board); ok {
		return w.boardRepo.CreateBoard(ctx, b)
	}
	return nil
}

func (w *testKanbanBoardRepoWrapper) GetBoard(ctx context.Context, id string) (interface{}, error) {
	return w.boardRepo.GetBoard(ctx, id)
}

func (w *testKanbanBoardRepoWrapper) GetBoardByCommission(ctx context.Context, commissionID string) (interface{}, error) {
	return w.boardRepo.GetBoardByCommission(ctx, commissionID)
}

func (w *testKanbanBoardRepoWrapper) UpdateBoard(ctx context.Context, board interface{}) error {
	if b, ok := board.(*storage.Board); ok {
		return w.boardRepo.UpdateBoard(ctx, b)
	}
	return nil
}

func (w *testKanbanBoardRepoWrapper) ListBoards(ctx context.Context) ([]interface{}, error) {
	boards, err := w.boardRepo.ListBoards(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]interface{}, len(boards))
	for i, board := range boards {
		result[i] = board
	}
	return result, nil
}

func (w *testKanbanBoardRepoWrapper) DeleteBoard(ctx context.Context, id string) error {
	// Not implemented in storage layer
	return nil
}

type testKanbanCampaignRepoWrapper struct {
	campaignRepo storage.CampaignRepository
}

func (w *testKanbanCampaignRepoWrapper) CreateCampaign(ctx context.Context, campaign interface{}) error {
	if c, ok := campaign.(*storage.Campaign); ok {
		return w.campaignRepo.CreateCampaign(ctx, c)
	}
	return nil
}

type testKanbanCommissionRepoWrapper struct {
	commissionRepo storage.CommissionRepository
}

func (w *testKanbanCommissionRepoWrapper) CreateCommission(ctx context.Context, commission interface{}) error {
	if c, ok := commission.(*storage.Commission); ok {
		return w.commissionRepo.CreateCommission(ctx, c)
	}
	return nil
}

func (w *testKanbanCommissionRepoWrapper) GetCommission(ctx context.Context, id string) (interface{}, error) {
	return w.commissionRepo.GetCommission(ctx, id)
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	return &s
}
