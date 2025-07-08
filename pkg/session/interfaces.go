// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"context"
	"time"

	"github.com/lancekrogers/guild/pkg/storage"
)

// SessionManagerInterface defines the high-level session management contract
type SessionManagerInterface interface {
	// Session lifecycle
	CreateSession(ctx context.Context, userID, campaignID string) (*Session, error)
	LoadSession(ctx context.Context, sessionID string) (*Session, error)
	SaveSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, sessionID string) error
	ListSessions(ctx context.Context, options ListOptions) ([]*Session, error)

	// Session operations
	AddMessage(ctx context.Context, sessionID string, message *Message) error
	GetMessages(ctx context.Context, sessionID string, limit int) ([]*Message, error)
	UpdateSessionState(ctx context.Context, sessionID string, state SessionState) error

	// Auto-save and backup
	StartAutoSave(ctx context.Context, session *Session)
	StopAutoSave(ctx context.Context, sessionID string)
	CreateBackup(ctx context.Context, sessionID string) error
}

// SessionResumerInterface defines the session restoration contract
type SessionResumerInterface interface {
	// Resume operations
	ResumeSession(ctx context.Context, sessionID string) error
	CanResumeSession(ctx context.Context, sessionID string) (bool, error)
	GetResumableSessions(ctx context.Context, userID string) ([]*Session, error)

	// Recovery operations
	RecoverFromCrash(ctx context.Context) (*Session, error)
	CreateRecoveryPoint(ctx context.Context, session *Session) error
}

// SessionExporterInterface defines the export/import contract
type SessionExporterInterface interface {
	// Export operations
	ExportSession(session *Session, format ExportFormat, options ExportOptions) ([]byte, error)
	ExportSessions(sessions []*Session, format ExportFormat, options ExportOptions) ([]byte, error)

	// Import operations
	ImportSession(ctx context.Context, data []byte, format ExportFormat) (*Session, error)
	ValidateImportData(data []byte, format ExportFormat) error

	// Format support
	GetSupportedFormats() []ExportFormat
	GetFormatCapabilities(format ExportFormat) FormatCapabilities
}

// SessionAnalyticsInterface defines the analytics contract
type SessionAnalyticsInterface interface {
	// Analytics operations
	AnalyzeSession(ctx context.Context, session *Session) (*AnalyticsData, error)
	GenerateReport(ctx context.Context, period TimePeriod) (*AnalyticsReport, error)
	GetSessionMetrics(ctx context.Context, sessionID string) (*SessionMetrics, error)

	// Insights
	GenerateInsights(ctx context.Context, userID string, period TimePeriod) ([]Insight, error)
	GetProductivityScore(ctx context.Context, sessionID string) (float64, error)

	// Tracking
	TrackEvent(ctx context.Context, sessionID string, event AnalyticsEvent) error
	GetUsagePatterns(ctx context.Context, userID string) (*UsagePatterns, error)
}

// OrchestratorIntegration defines how session management integrates with the orchestrator
type OrchestratorIntegration interface {
	// Agent management
	RegisterAgentStateHandler(handler AgentStateHandler)
	NotifyAgentStateChange(ctx context.Context, sessionID, agentID string, state AgentState) error
	GetActiveAgents(ctx context.Context, sessionID string) ([]AgentInterface, error)

	// Task management
	RegisterTaskHandler(handler TaskHandler)
	NotifyTaskStateChange(ctx context.Context, sessionID, taskID string, status TaskStatus) error
	GetActiveTasks(ctx context.Context, sessionID string) ([]Task, error)

	// Event handling
	RegisterEventHandler(handler SessionEventHandler)
	PublishSessionEvent(ctx context.Context, event SessionEvent) error
}

// UIIntegration defines how session management integrates with the UI
type UIIntegration interface {
	// UI state management
	RestoreUIState(ctx context.Context, sessionID string, state SessionState) error
	CaptureUIState(ctx context.Context, sessionID string) (SessionState, error)

	// Message handling
	DisplayMessage(ctx context.Context, message *Message) error
	DisplayNotification(ctx context.Context, notification Notification) error

	// User interactions
	ShowResumeDialog(ctx context.Context, sessions []*Session) (*Session, error)
	ShowExportDialog(ctx context.Context, session *Session) (ExportOptions, error)
	ShowAnalyticsDashboard(ctx context.Context, analytics *AnalyticsReport) error
}

// Agent state handling
type AgentStateHandler interface {
	OnAgentConnected(ctx context.Context, sessionID, agentID string) error
	OnAgentDisconnected(ctx context.Context, sessionID, agentID string) error
	OnAgentStateChanged(ctx context.Context, sessionID, agentID string, oldState, newState AgentState) error
}

// Task handling
type TaskHandler interface {
	OnTaskCreated(ctx context.Context, sessionID string, task Task) error
	OnTaskStarted(ctx context.Context, sessionID, taskID string) error
	OnTaskCompleted(ctx context.Context, sessionID, taskID string, result TaskResult) error
	OnTaskFailed(ctx context.Context, sessionID, taskID string, error error) error
}

// Session event handling
type SessionEventHandler interface {
	OnSessionCreated(ctx context.Context, session *Session) error
	OnSessionResumed(ctx context.Context, session *Session) error
	OnSessionClosed(ctx context.Context, sessionID string) error
	OnMessageAdded(ctx context.Context, sessionID string, message *Message) error
}

// Additional types for integration

// FormatCapabilities describes what a format supports
type FormatCapabilities struct {
	SupportsMetadata    bool     `json:"supports_metadata"`
	SupportsAttachments bool     `json:"supports_attachments"`
	SupportsFormatting  bool     `json:"supports_formatting"`
	MaxFileSize         int64    `json:"max_file_size"`
	Extensions          []string `json:"extensions"`
}

// SessionMetrics contains key metrics for a session
type SessionMetrics struct {
	Duration          time.Duration `json:"duration"`
	MessageCount      int           `json:"message_count"`
	AgentCount        int           `json:"agent_count"`
	TaskCount         int           `json:"task_count"`
	CompletionRate    float64       `json:"completion_rate"`
	ProductivityScore float64       `json:"productivity_score"`
	LastActivity      time.Time     `json:"last_activity"`
}

// AnalyticsEvent represents an event to be tracked
type AnalyticsEvent struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	SessionID string                 `json:"session_id"`
	UserID    string                 `json:"user_id"`
	Data      map[string]interface{} `json:"data"`
}

// UsagePatterns contains user usage pattern analysis
type UsagePatterns struct {
	MostActiveHours    []int                  `json:"most_active_hours"`
	PreferredAgents    []string               `json:"preferred_agents"`
	CommonCommands     []string               `json:"common_commands"`
	AverageSessionTime time.Duration          `json:"average_session_time"`
	Productivity       ProductivityPattern    `json:"productivity"`
	Preferences        map[string]interface{} `json:"preferences"`
}

// ProductivityPattern analyzes productivity trends
type ProductivityPattern struct {
	BestHours        []int    `json:"best_hours"`
	ProductiveAgents []string `json:"productive_agents"`
	TrendDirection   string   `json:"trend_direction"`
	Recommendations  []string `json:"recommendations"`
}

// TaskResult represents the result of a completed task
type TaskResult struct {
	Success  bool                   `json:"success"`
	Output   string                 `json:"output"`
	Error    string                 `json:"error,omitempty"`
	Duration time.Duration          `json:"duration"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Notification represents a UI notification
type Notification struct {
	Type     NotificationType     `json:"type"`
	Title    string               `json:"title"`
	Message  string               `json:"message"`
	Duration time.Duration        `json:"duration"`
	Actions  []NotificationAction `json:"actions,omitempty"`
}

// NotificationType categorizes notifications
type NotificationType string

const (
	NotificationInfo    NotificationType = "info"
	NotificationWarning NotificationType = "warning"
	NotificationError   NotificationType = "error"
	NotificationSuccess NotificationType = "success"
)

// NotificationAction represents an action that can be taken from a notification
type NotificationAction struct {
	Label   string `json:"label"`
	Action  string `json:"action"`
	Primary bool   `json:"primary"`
}

// SessionEvent represents events that occur during a session
type SessionEvent struct {
	ID        string                 `json:"id"`
	Type      SessionEventType       `json:"type"`
	SessionID string                 `json:"session_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// SessionEventType categorizes session events
type SessionEventType string

const (
	SessionEventCreated        SessionEventType = "session_created"
	SessionEventResumed        SessionEventType = "session_resumed"
	SessionEventClosed         SessionEventType = "session_closed"
	SessionEventMessageAdded   SessionEventType = "message_added"
	SessionEventAgentConnected SessionEventType = "agent_connected"
	SessionEventTaskStarted    SessionEventType = "task_started"
	SessionEventTaskCompleted  SessionEventType = "task_completed"
	SessionEventStateChanged   SessionEventType = "state_changed"
)

// StorageIntegration defines how session management integrates with storage systems
type StorageIntegration interface {
	// Session storage
	GetSessionRepository() storage.SessionRepository

	// Analytics storage
	GetAnalyticsStore() AnalyticsStore

	// Backup storage
	CreateBackup(ctx context.Context, sessionID string, data []byte) error
	RestoreBackup(ctx context.Context, sessionID string) ([]byte, error)
	ListBackups(ctx context.Context, sessionID string) ([]BackupInfo, error)
}

// BackupInfo contains information about a session backup
type BackupInfo struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size"`
	Type      string    `json:"type"`
}

// ConfigurationProvider defines configuration management for sessions
type ConfigurationProvider interface {
	// Session configuration
	GetSessionConfig() SessionConfig
	GetAutoSaveConfig() AutoSaveConfig
	GetEncryptionConfig() EncryptionConfig

	// Analytics configuration
	GetAnalyticsConfig() AnalyticsConfig

	// Export configuration
	GetExportConfig() ExportConfig
}

// SessionConfig contains session management configuration
type SessionConfig struct {
	MaxSessionDuration    time.Duration `json:"max_session_duration"`
	MaxMessagesPerSession int           `json:"max_messages_per_session"`
	CleanupInterval       time.Duration `json:"cleanup_interval"`
	RetentionPeriod       time.Duration `json:"retention_period"`
}

// AutoSaveConfig contains auto-save configuration
type AutoSaveConfig struct {
	Enabled         bool          `json:"enabled"`
	Interval        time.Duration `json:"interval"`
	MaxChangeBuffer int           `json:"max_change_buffer"`
	SaveOnIdle      bool          `json:"save_on_idle"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
}

// EncryptionConfig contains encryption configuration
type EncryptionConfig struct {
	Enabled   bool   `json:"enabled"`
	KeySource string `json:"key_source"`
	Algorithm string `json:"algorithm"`
	KeySize   int    `json:"key_size"`
}

// AnalyticsConfig contains analytics configuration
type AnalyticsConfig struct {
	Enabled            bool          `json:"enabled"`
	TrackProductivity  bool          `json:"track_productivity"`
	TrackUsagePatterns bool          `json:"track_usage_patterns"`
	RetentionPeriod    time.Duration `json:"retention_period"`
	GenerateInsights   bool          `json:"generate_insights"`
}

// ExportConfig contains export configuration
type ExportConfig struct {
	DefaultFormat        ExportFormat   `json:"default_format"`
	MaxExportSize        int64          `json:"max_export_size"`
	AllowedFormats       []ExportFormat `json:"allowed_formats"`
	IncludeMetadata      bool           `json:"include_metadata_by_default"`
	CompressLargeExports bool           `json:"compress_large_exports"`
}

// Registry integration for dependency injection
type SessionRegistry interface {
	// Core components
	RegisterSessionManager(manager SessionManagerInterface)
	RegisterSessionResumer(resumer SessionResumerInterface)
	RegisterSessionExporter(exporter SessionExporterInterface)
	RegisterSessionAnalytics(analytics SessionAnalyticsInterface)

	// Integrations
	RegisterOrchestratorIntegration(integration OrchestratorIntegration)
	RegisterUIIntegration(integration UIIntegration)
	RegisterStorageIntegration(integration StorageIntegration)

	// Configuration
	RegisterConfigurationProvider(provider ConfigurationProvider)

	// Getters
	GetSessionManager() SessionManagerInterface
	GetSessionResumer() SessionResumerInterface
	GetSessionExporter() SessionExporterInterface
	GetSessionAnalytics() SessionAnalyticsInterface
}

// Middleware interfaces for extensibility
type SessionMiddleware interface {
	BeforeSessionCreate(ctx context.Context, session *Session) error
	AfterSessionCreate(ctx context.Context, session *Session) error
	BeforeSessionLoad(ctx context.Context, sessionID string) error
	AfterSessionLoad(ctx context.Context, session *Session) error
	BeforeSessionSave(ctx context.Context, session *Session) error
	AfterSessionSave(ctx context.Context, session *Session) error
}

type MessageMiddleware interface {
	BeforeMessageAdd(ctx context.Context, sessionID string, message *Message) error
	AfterMessageAdd(ctx context.Context, sessionID string, message *Message) error
	ProcessMessage(ctx context.Context, message *Message) (*Message, error)
}

type AnalyticsMiddleware interface {
	BeforeAnalytics(ctx context.Context, session *Session) error
	AfterAnalytics(ctx context.Context, analytics *AnalyticsData) error
	ProcessInsight(ctx context.Context, insight *Insight) (*Insight, error)
}

// Error types for better error handling
type SessionError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *SessionError) Error() string {
	return e.Message
}

// Common error codes
const (
	ErrSessionNotFound      = "SESSION_NOT_FOUND"
	ErrSessionAlreadyExists = "SESSION_ALREADY_EXISTS"
	ErrInvalidSessionState  = "INVALID_SESSION_STATE"
	ErrResumeFailed         = "RESUME_FAILED"
	ErrExportFailed         = "EXPORT_FAILED"
	ErrImportFailed         = "IMPORT_FAILED"
	ErrAnalyticsFailed      = "ANALYTICS_FAILED"
	ErrStorageFailed        = "STORAGE_FAILED"
	ErrEncryptionFailed     = "ENCRYPTION_FAILED"
)
