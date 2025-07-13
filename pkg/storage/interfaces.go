// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package storage

import (
	"context"
	"time"
)

// Domain models - following Guild's naming conventions
type Task struct {
	ID              string                 `json:"id"`
	BoardID         *string                `json:"board_id,omitempty"` // Nullable for new schema
	AssignedAgentID *string                `json:"assigned_agent_id,omitempty"`
	Title           string                 `json:"title"`
	Description     *string                `json:"description,omitempty"`
	Status          string                 `json:"status"`
	Column          string                 `json:"column"` // Kanban column (backlog, in_progress, etc.)
	StoryPoints     int32                  `json:"story_points"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	AgentName       *string                `json:"agent_name,omitempty"` // For joined queries
	AgentType       *string                `json:"agent_type,omitempty"` // For joined queries

	// DEPRECATED: Use BoardID instead - kept for backward compatibility
	CommissionID string `json:"commission_id,omitempty"`
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

// Preference represents a configuration preference at any scope level
type Preference struct {
	ID        string                 `json:"id"`
	Scope     string                 `json:"scope"`    // "system", "user", "campaign", "guild", "agent"
	ScopeID   *string                `json:"scope_id"` // NULL for system scope
	Key       string                 `json:"key"`
	Value     interface{}            `json:"value"` // Flexible JSON value
	Version   int                    `json:"version"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// PreferenceScope defines a scope for preference resolution
type PreferenceScope struct {
	Scope   string  // "system", "user", "campaign", "guild", "agent"
	ScopeID *string // ID of the scoped entity (nil for system)
}

// PreferenceInheritance defines the inheritance relationship between scopes
type PreferenceInheritance struct {
	ID            string    `json:"id"`
	ChildScope    string    `json:"child_scope"`
	ChildScopeID  *string   `json:"child_scope_id"`
	ParentScope   string    `json:"parent_scope"`
	ParentScopeID *string   `json:"parent_scope_id"`
	Priority      int       `json:"priority"` // Higher priority overrides lower
	CreatedAt     time.Time `json:"created_at"`
}

type Board struct {
	ID           string    `json:"id"`
	CommissionID string    `json:"commission_id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Agent struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Provider      *string                `json:"provider,omitempty"`
	Model         *string                `json:"model,omitempty"`
	Capabilities  map[string]interface{} `json:"capabilities,omitempty"`
	Tools         map[string]interface{} `json:"tools,omitempty"`
	CostMagnitude int32                  `json:"cost_magnitude"`
	CreatedAt     time.Time              `json:"created_at"`
}

type TaskEvent struct {
	ID        int64     `json:"id"`
	TaskID    string    `json:"task_id"`
	AgentID   *string   `json:"agent_id,omitempty"`
	EventType string    `json:"event_type"`
	OldValue  *string   `json:"old_value,omitempty"`
	NewValue  *string   `json:"new_value,omitempty"`
	Reason    *string   `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type AgentWorkload struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	TaskCount   int64  `json:"task_count"`
	ActiveTasks int64  `json:"active_tasks"`
}

type PromptChain struct {
	ID        string                `json:"id"`
	AgentID   string                `json:"agent_id"`
	TaskID    *string               `json:"task_id,omitempty"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
	Messages  []*PromptChainMessage `json:"messages,omitempty"`
}

type PromptChainMessage struct {
	ID         int64     `json:"id"`
	ChainID    string    `json:"chain_id"`
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	Name       *string   `json:"name,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	TokenUsage int32     `json:"token_usage"`
}

// Repository interfaces - following Guild's context-first pattern
type TaskRepository interface {
	CreateTask(ctx context.Context, task *Task) error
	GetTask(ctx context.Context, id string) (*Task, error)
	UpdateTask(ctx context.Context, task *Task) error
	DeleteTask(ctx context.Context, id string) error
	ListTasks(ctx context.Context) ([]*Task, error)
	ListTasksByStatus(ctx context.Context, status string) ([]*Task, error)
	ListTasksByBoard(ctx context.Context, boardID string) ([]*Task, error)
	ListTasksByCommission(ctx context.Context, commissionID string) ([]*Task, error)
	ListTasksForKanban(ctx context.Context, boardID string) ([]*Task, error)
	AssignTask(ctx context.Context, taskID, agentID string) error
	UpdateTaskStatus(ctx context.Context, taskID, status string) error
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

type BoardRepository interface {
	CreateBoard(ctx context.Context, board *Board) error
	GetBoard(ctx context.Context, id string) (*Board, error)
	GetBoardByCommission(ctx context.Context, commissionID string) (*Board, error)
	UpdateBoard(ctx context.Context, board *Board) error
	DeleteBoard(ctx context.Context, id string) error
	ListBoards(ctx context.Context) ([]*Board, error)
}

type AgentRepository interface {
	CreateAgent(ctx context.Context, agent *Agent) error
	GetAgent(ctx context.Context, id string) (*Agent, error)
	UpdateAgent(ctx context.Context, agent *Agent) error
	DeleteAgent(ctx context.Context, id string) error
	ListAgents(ctx context.Context) ([]*Agent, error)
	ListAgentsByType(ctx context.Context, agentType string) ([]*Agent, error)
}

type PromptChainRepository interface {
	CreateChain(ctx context.Context, chain *PromptChain) error
	GetChain(ctx context.Context, id string) (*PromptChain, error)
	AddMessage(ctx context.Context, chainID string, message *PromptChainMessage) error
	GetChainsByAgent(ctx context.Context, agentID string) ([]*PromptChain, error)
	GetChainsByTask(ctx context.Context, taskID string) ([]*PromptChain, error)
	DeleteChain(ctx context.Context, id string) error
}

// ChatSession represents a chat conversation for repository operations
type ChatSession struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	CampaignID *string                `json:"campaign_id,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ChatMessage represents a message in a chat session
type ChatMessage struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	CreatedAt time.Time              `json:"created_at"`
	ToolCalls map[string]interface{} `json:"tool_calls,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SessionRepository provides chat session persistence
type SessionRepository interface {
	// Session operations
	CreateSession(ctx context.Context, session *ChatSession) error
	GetSession(ctx context.Context, id string) (*ChatSession, error)
	ListSessions(ctx context.Context, limit, offset int32) ([]*ChatSession, error)
	ListSessionsByCampaign(ctx context.Context, campaignID string) ([]*ChatSession, error)
	UpdateSession(ctx context.Context, session *ChatSession) error
	DeleteSession(ctx context.Context, id string) error
	CountSessions(ctx context.Context) (int64, error)

	// Message operations
	SaveMessage(ctx context.Context, message *ChatMessage) error
	GetMessage(ctx context.Context, id string) (*ChatMessage, error)
	GetMessages(ctx context.Context, sessionID string) ([]*ChatMessage, error)
	GetMessagesPaginated(ctx context.Context, sessionID string, limit, offset int32) ([]*ChatMessage, error)
	GetMessagesAfter(ctx context.Context, sessionID string, after time.Time) ([]*ChatMessage, error)
	CountMessages(ctx context.Context, sessionID string) (int64, error)
	DeleteMessage(ctx context.Context, id string) error

	// Streaming operations for daemon
	StreamMessages(ctx context.Context, sessionID string, since time.Time) (<-chan *ChatMessage, error)
}

// PreferencesRepository handles preference storage and retrieval with hierarchical resolution
type PreferencesRepository interface {
	// Basic CRUD operations
	CreatePreference(ctx context.Context, pref *Preference) error
	GetPreference(ctx context.Context, id string) (*Preference, error)
	UpdatePreference(ctx context.Context, pref *Preference) error
	DeletePreference(ctx context.Context, id string) error

	// Scope-based queries
	GetPreferenceByKey(ctx context.Context, scope string, scopeID *string, key string) (*Preference, error)
	ListPreferencesByScope(ctx context.Context, scope string, scopeID *string) ([]*Preference, error)
	ListPreferencesByKey(ctx context.Context, key string) ([]*Preference, error)

	// Hierarchical resolution - resolves preference value through inheritance chain
	ResolvePreference(ctx context.Context, key string, scopes []PreferenceScope) (*Preference, error)

	// Bulk operations
	GetPreferencesByKeys(ctx context.Context, scope string, scopeID *string, keys []string) ([]*Preference, error)
	SetPreferences(ctx context.Context, scope string, scopeID *string, prefs map[string]interface{}) error
	DeletePreferencesByScope(ctx context.Context, scope string, scopeID *string) error

	// Inheritance management
	CreateInheritance(ctx context.Context, inheritance *PreferenceInheritance) error
	GetInheritanceChain(ctx context.Context, scope string, scopeID *string) ([]*PreferenceInheritance, error)
	DeleteInheritance(ctx context.Context, id string) error

	// Import/Export
	ExportPreferences(ctx context.Context, scope string, scopeID *string) (map[string]interface{}, error)
	ImportPreferences(ctx context.Context, scope string, scopeID *string, prefs map[string]interface{}) error
}

// StorageRegistry follows Guild's registry pattern
type StorageRegistry interface {
	RegisterTaskRepository(repo TaskRepository)
	RegisterCampaignRepository(repo CampaignRepository)
	RegisterCommissionRepository(repo CommissionRepository)
	RegisterBoardRepository(repo BoardRepository)
	RegisterAgentRepository(repo AgentRepository)
	RegisterPromptChainRepository(repo PromptChainRepository)
	RegisterSessionRepository(repo SessionRepository)
	RegisterPreferencesRepository(repo PreferencesRepository)
	RegisterMemoryStore(store interface{})
	RegisterOptimizationManager(manager interface{})

	GetTaskRepository() TaskRepository
	GetCampaignRepository() CampaignRepository
	GetCommissionRepository() CommissionRepository
	GetBoardRepository() BoardRepository
	GetAgentRepository() AgentRepository
	GetPromptChainRepository() PromptChainRepository
	GetSessionRepository() SessionRepository
	GetPreferencesRepository() PreferencesRepository
	GetMemoryStore() interface{}
	GetOptimizationManager() interface{}
}
