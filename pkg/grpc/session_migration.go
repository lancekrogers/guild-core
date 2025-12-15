// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/guild-framework/guild-core/pkg/gerror"
	v1 "github.com/guild-framework/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// LegacySession represents the old session format for migration
type LegacySession struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Campaign  string                 `json:"campaign,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Messages  []LegacyMessage        `json:"messages"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LegacyMessage represents the old message format for migration
type LegacyMessage struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	ToolCalls []LegacyToolCall       `json:"tool_calls,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LegacyToolCall represents the old tool call format
type LegacyToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function LegacyFunction         `json:"function"`
	Result   map[string]interface{} `json:"result,omitempty"`
}

// LegacyFunction represents the old function call format
type LegacyFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// SessionMigrationService provides legacy session import capabilities
type SessionMigrationService struct {
	sessionService v1.SessionServiceServer
}

// NewSessionMigrationService creates a new migration service
func NewSessionMigrationService(sessionService v1.SessionServiceServer) *SessionMigrationService {
	return &SessionMigrationService{
		sessionService: sessionService,
	}
}

// ImportLegacySessions imports sessions from legacy format files
func (s *SessionMigrationService) ImportLegacySessions(ctx context.Context, legacyDir string) (*ImportResult, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("ImportLegacySessions")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("grpc").
			WithOperation("ImportLegacySessions")
	}

	logger.Info("Starting legacy session import", "legacy_dir", legacyDir)
	startTime := time.Now()

	result := &ImportResult{
		StartedAt: startTime,
	}

	// Find all legacy session files
	sessionFiles, err := s.findLegacySessionFiles(legacyDir)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to find legacy session files").
			WithComponent("grpc").
			WithOperation("ImportLegacySessions")
	}

	logger.Info("Found legacy session files", "count", len(sessionFiles))
	result.TotalFiles = len(sessionFiles)

	// Import each session file
	for _, sessionFile := range sessionFiles {
		if err := ctx.Err(); err != nil {
			return result, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during import").
				WithComponent("grpc").
				WithOperation("ImportLegacySessions")
		}

		sessionResult, err := s.importSessionFile(ctx, sessionFile)
		result.Sessions = append(result.Sessions, sessionResult)

		if err != nil {
			logger.WithError(err).Warn("Failed to import session file",
				"file", sessionFile,
			)
			result.Errors = append(result.Errors, ImportError{
				File:  sessionFile,
				Error: err.Error(),
			})
		} else {
			result.SuccessfulImports++
			logger.Info("Successfully imported session",
				"file", sessionFile,
				"session_id", sessionResult.NewSessionID,
			)
		}
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	logger.Info("Legacy session import completed",
		"total_files", result.TotalFiles,
		"successful_imports", result.SuccessfulImports,
		"errors", len(result.Errors),
		"duration_ms", result.Duration.Milliseconds(),
	)

	return result, nil
}

// ImportResult contains the results of a legacy session import operation
type ImportResult struct {
	StartedAt         time.Time       `json:"started_at"`
	CompletedAt       time.Time       `json:"completed_at"`
	Duration          time.Duration   `json:"duration"`
	TotalFiles        int             `json:"total_files"`
	SuccessfulImports int             `json:"successful_imports"`
	Sessions          []SessionResult `json:"sessions"`
	Errors            []ImportError   `json:"errors"`
}

// SessionResult contains the result of importing a single session
type SessionResult struct {
	LegacySessionID string    `json:"legacy_session_id"`
	NewSessionID    string    `json:"new_session_id"`
	MessageCount    int       `json:"message_count"`
	ImportedAt      time.Time `json:"imported_at"`
}

// ImportError contains details about an import error
type ImportError struct {
	File  string `json:"file"`
	Error string `json:"error"`
}

// findLegacySessionFiles finds all JSON files that could contain legacy sessions
func (s *SessionMigrationService) findLegacySessionFiles(dir string) ([]string, error) {
	var sessionFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for JSON files that might contain sessions
		if filepath.Ext(path) == ".json" && !info.IsDir() {
			sessionFiles = append(sessionFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return sessionFiles, nil
}

// importSessionFile imports a single legacy session file
func (s *SessionMigrationService) importSessionFile(ctx context.Context, filePath string) (SessionResult, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("importSessionFile")

	// Read and parse the legacy session file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return SessionResult{}, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read session file").
			WithComponent("grpc").
			WithOperation("importSessionFile").
			WithDetails("file", filePath)
	}

	var legacySession LegacySession
	if err := json.Unmarshal(data, &legacySession); err != nil {
		return SessionResult{}, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to parse session JSON").
			WithComponent("grpc").
			WithOperation("importSessionFile").
			WithDetails("file", filePath)
	}

	logger.Info("Importing legacy session",
		"legacy_id", legacySession.ID,
		"name", legacySession.Name,
		"message_count", len(legacySession.Messages),
	)

	// Convert legacy session to new format
	newSession, err := s.convertLegacySession(legacySession)
	if err != nil {
		return SessionResult{}, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to convert legacy session").
			WithComponent("grpc").
			WithOperation("importSessionFile")
	}

	// Create the new session
	createdSession, err := s.sessionService.CreateSession(ctx, newSession)
	if err != nil {
		return SessionResult{}, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create session").
			WithComponent("grpc").
			WithOperation("importSessionFile")
	}

	// Import messages
	messageCount := 0
	for _, legacyMsg := range legacySession.Messages {
		newMessage, err := s.convertLegacyMessage(legacyMsg, createdSession.Id)
		if err != nil {
			logger.WithError(err).Warn("Skipping invalid legacy message",
				"message_id", legacyMsg.ID,
			)
			continue
		}

		_, err = s.sessionService.SaveMessage(ctx, &v1.SaveMessageRequest{
			Message: newMessage,
		})
		if err != nil {
			logger.WithError(err).Warn("Failed to save message",
				"message_id", legacyMsg.ID,
			)
			continue
		}

		messageCount++
	}

	return SessionResult{
		LegacySessionID: legacySession.ID,
		NewSessionID:    createdSession.Id,
		MessageCount:    messageCount,
		ImportedAt:      time.Now(),
	}, nil
}

// convertLegacySession converts a legacy session to the new gRPC format
func (s *SessionMigrationService) convertLegacySession(legacy LegacySession) (*v1.CreateSessionRequest, error) {
	// Validate legacy session
	if legacy.Name == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "legacy session missing name", nil).
			WithComponent("grpc").
			WithOperation("convertLegacySession")
	}

	req := &v1.CreateSessionRequest{
		Name:     fmt.Sprintf("[IMPORTED] %s", legacy.Name),
		Metadata: make(map[string]string),
	}

	// Add campaign if present
	if legacy.Campaign != "" {
		req.CampaignId = &legacy.Campaign
	}

	// Convert metadata
	if legacy.Metadata != nil {
		for k, v := range legacy.Metadata {
			if str, ok := v.(string); ok {
				req.Metadata[k] = str
			} else {
				// Convert non-string values to JSON
				jsonVal, _ := json.Marshal(v)
				req.Metadata[k] = string(jsonVal)
			}
		}
	}

	// Add import metadata
	req.Metadata["imported_from"] = "legacy"
	req.Metadata["original_id"] = legacy.ID
	req.Metadata["import_timestamp"] = time.Now().Format(time.RFC3339)
	req.Metadata["original_created_at"] = legacy.CreatedAt.Format(time.RFC3339)

	return req, nil
}

// convertLegacyMessage converts a legacy message to the new gRPC format
func (s *SessionMigrationService) convertLegacyMessage(legacy LegacyMessage, sessionID string) (*v1.Message, error) {
	// Convert role
	role, err := s.convertLegacyRole(legacy.Role)
	if err != nil {
		return nil, err
	}

	msg := &v1.Message{
		SessionId: sessionID,
		Role:      role,
		Content:   legacy.Content,
		Metadata:  make(map[string]string),
	}

	// Generate new ID if empty
	if legacy.ID == "" {
		msg.Id = uuid.New().String()
	} else {
		msg.Id = legacy.ID
	}

	// Convert metadata
	if legacy.Metadata != nil {
		for k, v := range legacy.Metadata {
			if str, ok := v.(string); ok {
				msg.Metadata[k] = str
			} else {
				jsonVal, _ := json.Marshal(v)
				msg.Metadata[k] = string(jsonVal)
			}
		}
	}

	// Add import metadata
	msg.Metadata["imported_from"] = "legacy"
	msg.Metadata["original_timestamp"] = legacy.Timestamp.Format(time.RFC3339)

	// Handle tool calls if present
	if len(legacy.ToolCalls) > 0 {
		toolCallsJSON, _ := json.Marshal(legacy.ToolCalls)
		msg.Metadata["legacy_tool_calls"] = string(toolCallsJSON)
	}

	return msg, nil
}

// convertLegacyRole converts legacy role strings to the new enum
func (s *SessionMigrationService) convertLegacyRole(legacyRole string) (v1.Message_MessageRole, error) {
	switch legacyRole {
	case "system":
		return v1.Message_SYSTEM, nil
	case "user":
		return v1.Message_USER, nil
	case "assistant":
		return v1.Message_ASSISTANT, nil
	case "tool":
		return v1.Message_TOOL, nil
	default:
		// Default to user for unknown roles
		return v1.Message_USER, nil
	}
}

// ValidateLegacySession validates a legacy session before import
func (s *SessionMigrationService) ValidateLegacySession(legacy LegacySession) error {
	if legacy.ID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "legacy session missing ID", nil)
	}

	if legacy.Name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "legacy session missing name", nil)
	}

	// Validate messages
	for i, msg := range legacy.Messages {
		if msg.Role == "" {
			return gerror.New(gerror.ErrCodeInvalidInput,
				fmt.Sprintf("message %d missing role", i), nil)
		}

		if len(msg.Content) > maxContentLength {
			return gerror.New(gerror.ErrCodeInvalidInput,
				fmt.Sprintf("message %d content too long: %d > %d", i, len(msg.Content), maxContentLength), nil)
		}
	}

	return nil
}
