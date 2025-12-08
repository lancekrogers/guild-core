// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// CommunicationOrchestrator manages inter-agent communication and coordination
type CommunicationOrchestrator struct {
	messageQueue   chan AgentMessage
	routing        map[string]MessageHandler
	agents         map[string]core.Agent
	corpus         CorpusUpdater
	db             *sql.DB
	messageHistory []AgentMessage
	subscriptions  map[string][]MessageSubscription
	filters        []MessageFilter
	enrichers      []MessageEnricher
	isRunning      bool
}

// AgentMessage represents a message between agents
type AgentMessage struct {
	ID          string                 `json:"id"`
	Type        MessageType            `json:"type"`
	From        string                 `json:"from"`
	To          string                 `json:"to"` // Empty for broadcast messages
	Content     string                 `json:"content"`
	Context     MessageContext         `json:"context"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
	Priority    MessagePriority        `json:"priority"`
	Tags        []string               `json:"tags"`
	ThreadID    string                 `json:"thread_id,omitempty"` // For threaded conversations
	ReplyTo     string                 `json:"reply_to,omitempty"`  // For replies
	Attachments []MessageAttachment    `json:"attachments,omitempty"`
}

// MessageType represents different types of inter-agent messages
type MessageType string

const (
	MessageTypeTaskUpdate     MessageType = "task_update"
	MessageTypeHelpRequest    MessageType = "help_request"
	MessageTypeKnowledgeShare MessageType = "knowledge_share"
	MessageTypeStatusReport   MessageType = "status_report"
	MessageTypeAlert          MessageType = "alert"
	MessageTypeCoordination   MessageType = "coordination"
	MessageTypeHandoff        MessageType = "handoff"
	MessageTypeQuestion       MessageType = "question"
	MessageTypeAnswer         MessageType = "answer"
	MessageTypeBroadcast      MessageType = "broadcast"
)

// MessagePriority represents message priority levels
type MessagePriority string

const (
	PriorityLow      MessagePriority = "low"
	PriorityNormal   MessagePriority = "normal"
	PriorityHigh     MessagePriority = "high"
	PriorityCritical MessagePriority = "critical"
)

// MessageContext provides context for the message
type MessageContext struct {
	TaskID       string                 `json:"task_id,omitempty"`
	CommissionID string                 `json:"commission_id,omitempty"`
	AgentRole    string                 `json:"agent_role,omitempty"`
	Capabilities []string               `json:"capabilities,omitempty"`
	Workload     float64                `json:"workload,omitempty"`
	Timezone     string                 `json:"timezone,omitempty"`
	Language     string                 `json:"language,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// MessageAttachment represents files or data attached to messages
type MessageAttachment struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // file, image, data, link
	Size     int64  `json:"size"`
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
}

// MessageHandler defines how to handle specific message types
type MessageHandler interface {
	CanHandle(messageType MessageType) bool
	Handle(ctx context.Context, message AgentMessage) error
	GetName() string
}

// MessageSubscription represents an agent's subscription to certain message types
type MessageSubscription struct {
	AgentID     string        `json:"agent_id"`
	MessageType MessageType   `json:"message_type"`
	Filter      MessageFilter `json:"filter,omitempty"`
	Active      bool          `json:"active"`
	CreatedAt   time.Time     `json:"created_at"`
}

// MessageFilter defines criteria for filtering messages
type MessageFilter interface {
	ShouldInclude(message AgentMessage) bool
	GetDescription() string
}

// MessageEnricher adds context or enhances messages
type MessageEnricher interface {
	Enrich(ctx context.Context, message AgentMessage) (AgentMessage, error)
	GetName() string
}

// CorpusUpdater interface for updating the knowledge corpus
type CorpusUpdater interface {
	AddKnowledge(ctx context.Context, knowledge string, tags []string, source string) error
	UpdateContext(ctx context.Context, taskID string, context string) error
}

// NewCommunicationOrchestrator creates a new communication orchestrator
func NewCommunicationOrchestrator(
	agents map[string]core.Agent,
	corpus CorpusUpdater,
	db *sql.DB,
) *CommunicationOrchestrator {
	co := &CommunicationOrchestrator{
		messageQueue:   make(chan AgentMessage, 1000),
		routing:        make(map[string]MessageHandler),
		agents:         agents,
		corpus:         corpus,
		db:             db,
		messageHistory: []AgentMessage{},
		subscriptions:  make(map[string][]MessageSubscription),
		filters:        []MessageFilter{},
		enrichers:      []MessageEnricher{},
		isRunning:      false,
	}

	// Initialize default handlers
	co.initializeHandlers()

	// Initialize default enrichers
	co.initializeEnrichers()

	return co
}

// Start begins processing messages
func (co *CommunicationOrchestrator) Start(ctx context.Context) error {
	if co.isRunning {
		return gerror.New(gerror.ErrCodeValidation, "communication orchestrator already running", nil).
			WithComponent("CommunicationOrchestrator").
			WithOperation("Start")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "CommunicationOrchestrator")
	ctx = observability.WithOperation(ctx, "Start")

	logger.InfoContext(ctx, "Starting communication orchestrator")

	co.isRunning = true

	// Start message processing goroutine
	go co.processMessages(ctx)

	logger.InfoContext(ctx, "Communication orchestrator started")
	return nil
}

// Stop stops message processing
func (co *CommunicationOrchestrator) Stop(ctx context.Context) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Stopping communication orchestrator")

	co.isRunning = false
	close(co.messageQueue)

	logger.InfoContext(ctx, "Communication orchestrator stopped")
	return nil
}

// SendMessage sends a direct message between two agents
func (co *CommunicationOrchestrator) SendMessage(ctx context.Context, from, to string, content string) error {
	return co.SendMessageWithType(ctx, from, to, content, MessageTypeCoordination, PriorityNormal)
}

// SendMessageWithType sends a message with specified type and priority
func (co *CommunicationOrchestrator) SendMessageWithType(ctx context.Context, from, to, content string, msgType MessageType, priority MessagePriority) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("CommunicationOrchestrator").
			WithOperation("SendMessageWithType")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "CommunicationOrchestrator")
	ctx = observability.WithOperation(ctx, "SendMessageWithType")

	// Validate agents exist
	if _, exists := co.agents[from]; !exists {
		return gerror.New(gerror.ErrCodeValidation, "sender agent not found", nil).
			WithComponent("CommunicationOrchestrator").
			WithOperation("SendMessageWithType").
			WithDetails("from_agent", from)
	}

	if _, exists := co.agents[to]; !exists {
		return gerror.New(gerror.ErrCodeValidation, "recipient agent not found", nil).
			WithComponent("CommunicationOrchestrator").
			WithOperation("SendMessageWithType").
			WithDetails("to_agent", to)
	}

	// Create message
	message := AgentMessage{
		ID:        generateID(),
		Type:      msgType,
		From:      from,
		To:        to,
		Content:   content,
		Context:   co.gatherMessageContext(ctx, from, to),
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
		Priority:  priority,
		Tags:      []string{},
	}

	// Enrich message
	enrichedMessage, err := co.enrichMessage(ctx, message)
	if err != nil {
		logger.WarnContext(ctx, "Failed to enrich message", "error", err)
		enrichedMessage = message // Use original if enrichment fails
	}

	// Queue message for processing
	select {
	case co.messageQueue <- enrichedMessage:
		logger.DebugContext(ctx, "Message queued for processing",
			"message_id", enrichedMessage.ID,
			"from", from,
			"to", to,
			"type", msgType)
	default:
		return gerror.New(gerror.ErrCodeResourceLimit, "message queue is full", nil).
			WithComponent("CommunicationOrchestrator").
			WithOperation("SendMessageWithType")
	}

	return nil
}

// BroadcastMessage sends a message to all agents
func (co *CommunicationOrchestrator) BroadcastMessage(ctx context.Context, from string, content string) error {
	return co.BroadcastMessageWithType(ctx, from, content, MessageTypeBroadcast, PriorityNormal)
}

// BroadcastMessageWithType broadcasts a message with specified type and priority
func (co *CommunicationOrchestrator) BroadcastMessageWithType(ctx context.Context, from, content string, msgType MessageType, priority MessagePriority) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("CommunicationOrchestrator").
			WithOperation("BroadcastMessageWithType")
	}

	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Broadcasting message",
		"from", from,
		"type", msgType,
		"priority", priority)

	// Create broadcast message
	message := AgentMessage{
		ID:        generateID(),
		Type:      msgType,
		From:      from,
		To:        "", // Empty for broadcast
		Content:   content,
		Context:   co.gatherMessageContext(ctx, from, ""),
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
		Priority:  priority,
		Tags:      []string{"broadcast"},
	}

	// Enrich message
	enrichedMessage, err := co.enrichMessage(ctx, message)
	if err != nil {
		logger.WarnContext(ctx, "Failed to enrich broadcast message", "error", err)
		enrichedMessage = message
	}

	// Queue message for processing
	select {
	case co.messageQueue <- enrichedMessage:
		logger.DebugContext(ctx, "Broadcast message queued", "message_id", enrichedMessage.ID)
	default:
		return gerror.New(gerror.ErrCodeResourceLimit, "message queue is full", nil).
			WithComponent("CommunicationOrchestrator").
			WithOperation("BroadcastMessageWithType")
	}

	return nil
}

// RequestHelp sends a help request to capable agents
func (co *CommunicationOrchestrator) RequestHelp(ctx context.Context, agentID string, taskID string, helpType string) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Processing help request",
		"requesting_agent", agentID,
		"task_id", taskID,
		"help_type", helpType)

	// Find capable agents for this help type
	capableAgents := co.findCapableAgents(helpType)
	if len(capableAgents) == 0 {
		return gerror.New(gerror.ErrCodeNotFound, "no capable agents found for help request", nil).
			WithComponent("CommunicationOrchestrator").
			WithOperation("RequestHelp").
			WithDetails("help_type", helpType)
	}

	// Create help request message
	content := fmt.Sprintf("Help requested for task %s. Type: %s", taskID, helpType)

	// Send to capable agents
	for _, capableAgentID := range capableAgents {
		if capableAgentID == agentID {
			continue // Don't send help request to self
		}

		err := co.SendMessageWithType(ctx, agentID, capableAgentID, content, MessageTypeHelpRequest, PriorityHigh)
		if err != nil {
			logger.WarnContext(ctx, "Failed to send help request",
				"to_agent", capableAgentID,
				"error", err)
		}
	}

	return nil
}

// ShareKnowledge shares knowledge with the team and updates corpus
func (co *CommunicationOrchestrator) ShareKnowledge(ctx context.Context, agentID string, knowledge string, tags []string) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Sharing knowledge",
		"sharing_agent", agentID,
		"tags", tags)

	// Update corpus with new knowledge
	if co.corpus != nil {
		err := co.corpus.AddKnowledge(ctx, knowledge, tags, agentID)
		if err != nil {
			logger.WarnContext(ctx, "Failed to update corpus with knowledge", "error", err)
		}
	}

	// Broadcast knowledge to interested agents
	content := fmt.Sprintf("Knowledge shared: %s", knowledge)
	message := AgentMessage{
		ID:        generateID(),
		Type:      MessageTypeKnowledgeShare,
		From:      agentID,
		To:        "", // Broadcast
		Content:   content,
		Context:   co.gatherMessageContext(ctx, agentID, ""),
		Metadata:  map[string]interface{}{"knowledge": knowledge},
		Timestamp: time.Now(),
		Priority:  PriorityNormal,
		Tags:      tags,
	}

	select {
	case co.messageQueue <- message:
		logger.DebugContext(ctx, "Knowledge share message queued", "message_id", message.ID)
	default:
		return gerror.New(gerror.ErrCodeResourceLimit, "message queue is full", nil).
			WithComponent("CommunicationOrchestrator").
			WithOperation("ShareKnowledge")
	}

	return nil
}

// CoordinateAgents facilitates communication between specific agents
func (co *CommunicationOrchestrator) CoordinateAgents(ctx context.Context, from, to string, message string) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Coordinating agents",
		"from", from,
		"to", to)

	// Elena mediates the communication by enriching the message
	enrichedContent := co.enrichMessageContent(message, from, to)

	return co.SendMessageWithType(ctx, from, to, enrichedContent, MessageTypeCoordination, PriorityNormal)
}

// processMessages processes messages from the queue
func (co *CommunicationOrchestrator) processMessages(ctx context.Context) {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Starting message processing")

	for co.isRunning {
		select {
		case message, ok := <-co.messageQueue:
			if !ok {
				logger.InfoContext(ctx, "Message queue closed, stopping processing")
				return
			}

			if err := co.processMessage(ctx, message); err != nil {
				logger.WarnContext(ctx, "Failed to process message",
					"message_id", message.ID,
					"error", err)
			}

		case <-ctx.Done():
			logger.InfoContext(ctx, "Context cancelled, stopping message processing")
			return
		}
	}
}

// processMessage processes a single message
func (co *CommunicationOrchestrator) processMessage(ctx context.Context, message AgentMessage) error {
	logger := observability.GetLogger(ctx)
	logger.DebugContext(ctx, "Processing message",
		"message_id", message.ID,
		"type", message.Type,
		"from", message.From,
		"to", message.To)

	// Add to history
	co.messageHistory = append(co.messageHistory, message)

	// Route message to appropriate handlers
	return co.routeMessage(ctx, message)
}

// routeMessage routes a message to appropriate handlers
func (co *CommunicationOrchestrator) routeMessage(ctx context.Context, message AgentMessage) error {
	logger := observability.GetLogger(ctx)

	// Find handlers for this message type
	handlers := co.findHandlers(message.Type)
	if len(handlers) == 0 {
		logger.WarnContext(ctx, "No handlers found for message type", "type", message.Type)
		return nil
	}

	// Apply handlers
	for _, handler := range handlers {
		if err := handler.Handle(ctx, message); err != nil {
			logger.WarnContext(ctx, "Handler failed to process message",
				"handler", handler.GetName(),
				"message_id", message.ID,
				"error", err)
		}
	}

	// Store message in database
	if err := co.storeMessage(ctx, message); err != nil {
		logger.WarnContext(ctx, "Failed to store message", "message_id", message.ID, "error", err)
	}

	return nil
}

// Helper methods

func (co *CommunicationOrchestrator) gatherMessageContext(ctx context.Context, from, to string) MessageContext {
	context := MessageContext{
		Language: "en",
		Timezone: "UTC",
		Context:  make(map[string]interface{}),
	}

	// Add agent role information
	if _, exists := co.agents[from]; exists {
		// Would extract role and capabilities from agent
		context.AgentRole = "agent" // Placeholder
	}

	return context
}

func (co *CommunicationOrchestrator) enrichMessage(ctx context.Context, message AgentMessage) (AgentMessage, error) {
	enriched := message

	// Apply all enrichers
	for _, enricher := range co.enrichers {
		var err error
		enriched, err = enricher.Enrich(ctx, enriched)
		if err != nil {
			return message, gerror.Wrap(err, gerror.ErrCodeInternal, "message enrichment failed").
				WithComponent("CommunicationOrchestrator").
				WithOperation("enrichMessage").
				WithDetails("enricher", enricher.GetName())
		}
	}

	return enriched, nil
}

func (co *CommunicationOrchestrator) enrichMessageContent(message, from, to string) string {
	// Elena adds context and improves clarity
	return fmt.Sprintf("[From %s to %s] %s", from, to, message)
}

func (co *CommunicationOrchestrator) findCapableAgents(helpType string) []string {
	var capable []string

	// Simple capability matching - would be more sophisticated in practice
	for agentID := range co.agents {
		// Would check agent capabilities against help type
		capable = append(capable, agentID)
	}

	return capable
}

func (co *CommunicationOrchestrator) findHandlers(messageType MessageType) []MessageHandler {
	var handlers []MessageHandler

	for _, handler := range co.routing {
		if handler.CanHandle(messageType) {
			handlers = append(handlers, handler)
		}
	}

	return handlers
}

func (co *CommunicationOrchestrator) storeMessage(ctx context.Context, message AgentMessage) error {
	// Placeholder - would store message in database
	return nil
}

func (co *CommunicationOrchestrator) initializeHandlers() {
	// Initialize default message handlers
	co.routing["task_update"] = &TaskUpdateHandler{}
	co.routing["help_request"] = &HelpRequestHandler{}
	co.routing["knowledge_share"] = &KnowledgeShareHandler{}
	co.routing["status_report"] = &StatusReportHandler{}
}

func (co *CommunicationOrchestrator) initializeEnrichers() {
	// Initialize default message enrichers
	co.enrichers = []MessageEnricher{
		&ContextEnricher{},
		&TimestampEnricher{},
		&PriorityEnricher{},
	}
}

// Default Handler Implementations

// TaskUpdateHandler handles task update messages
type TaskUpdateHandler struct{}

func (h *TaskUpdateHandler) CanHandle(messageType MessageType) bool {
	return messageType == MessageTypeTaskUpdate
}

func (h *TaskUpdateHandler) Handle(ctx context.Context, message AgentMessage) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Handling task update message", "message_id", message.ID)

	// Would update task status and notify interested parties
	return nil
}

func (h *TaskUpdateHandler) GetName() string { return "TaskUpdateHandler" }

// HelpRequestHandler handles help request messages
type HelpRequestHandler struct{}

func (h *HelpRequestHandler) CanHandle(messageType MessageType) bool {
	return messageType == MessageTypeHelpRequest
}

func (h *HelpRequestHandler) Handle(ctx context.Context, message AgentMessage) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Handling help request message", "message_id", message.ID)

	// Would route help request to capable agents
	return nil
}

func (h *HelpRequestHandler) GetName() string { return "HelpRequestHandler" }

// KnowledgeShareHandler handles knowledge sharing messages
type KnowledgeShareHandler struct{}

func (h *KnowledgeShareHandler) CanHandle(messageType MessageType) bool {
	return messageType == MessageTypeKnowledgeShare
}

func (h *KnowledgeShareHandler) Handle(ctx context.Context, message AgentMessage) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Handling knowledge share message", "message_id", message.ID)

	// Would update knowledge base and notify relevant agents
	return nil
}

func (h *KnowledgeShareHandler) GetName() string { return "KnowledgeShareHandler" }

// StatusReportHandler handles status report messages
type StatusReportHandler struct{}

func (h *StatusReportHandler) CanHandle(messageType MessageType) bool {
	return messageType == MessageTypeStatusReport
}

func (h *StatusReportHandler) Handle(ctx context.Context, message AgentMessage) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Handling status report message", "message_id", message.ID)

	// Would process status report and update dashboards
	return nil
}

func (h *StatusReportHandler) GetName() string { return "StatusReportHandler" }

// Default Enricher Implementations

// ContextEnricher adds contextual information to messages
type ContextEnricher struct{}

func (e *ContextEnricher) Enrich(ctx context.Context, message AgentMessage) (AgentMessage, error) {
	// Add contextual information
	message.Context.Context["enriched_at"] = time.Now().Format(time.RFC3339)
	return message, nil
}

func (e *ContextEnricher) GetName() string { return "ContextEnricher" }

// TimestampEnricher ensures accurate timestamps
type TimestampEnricher struct{}

func (e *TimestampEnricher) Enrich(ctx context.Context, message AgentMessage) (AgentMessage, error) {
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}
	return message, nil
}

func (e *TimestampEnricher) GetName() string { return "TimestampEnricher" }

// PriorityEnricher adjusts message priority based on content
type PriorityEnricher struct{}

func (e *PriorityEnricher) Enrich(ctx context.Context, message AgentMessage) (AgentMessage, error) {
	// Analyze content for priority indicators
	content := strings.ToLower(message.Content)

	if strings.Contains(content, "urgent") || strings.Contains(content, "critical") {
		message.Priority = PriorityCritical
	} else if strings.Contains(content, "important") || strings.Contains(content, "asap") {
		message.Priority = PriorityHigh
	}

	return message, nil
}

func (e *PriorityEnricher) GetName() string { return "PriorityEnricher" }
