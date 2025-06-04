package storage

import (
	"context"
	"time"
)

// Domain models - following Guild's naming conventions
type Task struct {
	ID                string                 `json:"id"`
	CommissionID      string                 `json:"commission_id"`
	AssignedAgentID   *string                `json:"assigned_agent_id,omitempty"`
	Title             string                 `json:"title"`
	Description       *string                `json:"description,omitempty"`
	Status            string                 `json:"status"`
	Column            string                 `json:"column"`
	StoryPoints       int32                  `json:"story_points"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	AgentName         *string                `json:"agent_name,omitempty"`  // For joined queries
	AgentType         *string                `json:"agent_type,omitempty"` // For joined queries
}

type Campaign struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Commission struct {
	ID          string                 `json:"id"`
	CampaignID  string                 `json:"campaign_id"`
	Title       string                 `json:"title"`
	Description *string                `json:"description,omitempty"`
	Domain      *string                `json:"domain,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
}

type Agent struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Provider       *string                `json:"provider,omitempty"`
	Model          *string                `json:"model,omitempty"`
	Capabilities   map[string]interface{} `json:"capabilities,omitempty"`
	Tools          map[string]interface{} `json:"tools,omitempty"`
	CostMagnitude  int32                  `json:"cost_magnitude"`
	CreatedAt      time.Time              `json:"created_at"`
}

type TaskEvent struct {
	ID         int64     `json:"id"`
	TaskID     string    `json:"task_id"`
	AgentID    *string   `json:"agent_id,omitempty"`
	EventType  string    `json:"event_type"`
	OldValue   *string   `json:"old_value,omitempty"`
	NewValue   *string   `json:"new_value,omitempty"`
	Reason     *string   `json:"reason,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type AgentWorkload struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	TaskCount   int64  `json:"task_count"`
	ActiveTasks int64  `json:"active_tasks"`
}

// Repository interfaces - following Guild's context-first pattern
type TaskRepository interface {
	CreateTask(ctx context.Context, task *Task) error
	GetTask(ctx context.Context, id string) (*Task, error)
	UpdateTask(ctx context.Context, task *Task) error
	DeleteTask(ctx context.Context, id string) error
	ListTasks(ctx context.Context) ([]*Task, error)
	ListTasksByStatus(ctx context.Context, status string) ([]*Task, error)
	ListTasksByCommission(ctx context.Context, commissionID string) ([]*Task, error)
	ListTasksForKanban(ctx context.Context, commissionID string) ([]*Task, error)
	AssignTask(ctx context.Context, taskID, agentID string) error
	UpdateTaskStatus(ctx context.Context, taskID, status string) error
	UpdateTaskColumn(ctx context.Context, taskID, column string) error
	RecordTaskEvent(ctx context.Context, event *TaskEvent) error
	GetTaskHistory(ctx context.Context, taskID string) ([]*TaskEvent, error)
	GetAgentWorkload(ctx context.Context) ([]*AgentWorkload, error)
}

type CampaignRepository interface {
	CreateCampaign(ctx context.Context, campaign *Campaign) error
	GetCampaign(ctx context.Context, id string) (*Campaign, error)
	UpdateCampaignStatus(ctx context.Context, id, status string) error
	DeleteCampaign(ctx context.Context, id string) error
	ListCampaigns(ctx context.Context) ([]*Campaign, error)
}

type CommissionRepository interface {
	CreateCommission(ctx context.Context, commission *Commission) error
	GetCommission(ctx context.Context, id string) (*Commission, error)
	UpdateCommissionStatus(ctx context.Context, id, status string) error
	DeleteCommission(ctx context.Context, id string) error
	ListCommissionsByCampaign(ctx context.Context, campaignID string) ([]*Commission, error)
}

type AgentRepository interface {
	CreateAgent(ctx context.Context, agent *Agent) error
	GetAgent(ctx context.Context, id string) (*Agent, error)
	UpdateAgent(ctx context.Context, agent *Agent) error
	DeleteAgent(ctx context.Context, id string) error
	ListAgents(ctx context.Context) ([]*Agent, error)
	ListAgentsByType(ctx context.Context, agentType string) ([]*Agent, error)
}

// StorageRegistry follows Guild's registry pattern
type StorageRegistry interface {
	RegisterTaskRepository(repo TaskRepository)
	RegisterCampaignRepository(repo CampaignRepository)
	RegisterCommissionRepository(repo CommissionRepository)
	RegisterAgentRepository(repo AgentRepository)
	
	GetTaskRepository() TaskRepository
	GetCampaignRepository() CampaignRepository
	GetCommissionRepository() CommissionRepository
	GetAgentRepository() AgentRepository
}