// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package testutil

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/memory/vector"
	"github.com/guild-framework/guild-core/pkg/providers/interfaces"
	"github.com/guild-framework/guild-core/tools"
)

// MockLLMProvider provides a configurable mock LLM provider for testing
type MockLLMProvider struct {
	mu              sync.RWMutex
	responses       map[string]string
	streamResponses map[string][]string
	delays          map[string]time.Duration
	errors          map[string]error
	callCount       map[string]int
	lastRequest     map[string]*interfaces.ChatRequest
}

// NewMockLLMProvider creates a new mock LLM provider
func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{
		responses:       make(map[string]string),
		streamResponses: make(map[string][]string),
		delays:          make(map[string]time.Duration),
		errors:          make(map[string]error),
		callCount:       make(map[string]int),
		lastRequest:     make(map[string]*interfaces.ChatRequest),
	}
}

// SetResponse configures a response for a specific agent/model
func (m *MockLLMProvider) SetResponse(key string, response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[key] = response
}

// SetStreamResponse configures a streaming response
func (m *MockLLMProvider) SetStreamResponse(key string, chunks []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamResponses[key] = chunks
}

// SetDelay configures a delay before responding
func (m *MockLLMProvider) SetDelay(key string, delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delays[key] = delay
}

// SetError configures an error response
func (m *MockLLMProvider) SetError(key string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[key] = err
}

// GetCallCount returns the number of calls for a key
func (m *MockLLMProvider) GetCallCount(key string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount[key]
}

// GetLastRequest returns the last request for a key
func (m *MockLLMProvider) GetLastRequest(key string) *interfaces.ChatRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastRequest[key]
}

// ChatCompletion implements the AIProvider interface
func (m *MockLLMProvider) ChatCompletion(ctx context.Context, req interfaces.ChatRequest) (*interfaces.ChatResponse, error) {
	m.mu.Lock()
	key := req.Model
	if key == "" {
		key = "default"
	}
	m.callCount[key]++
	m.lastRequest[key] = &req
	m.mu.Unlock()

	// Check for configured delay
	if delay, ok := m.delays[key]; ok {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Check for configured error
	if err, ok := m.errors[key]; ok {
		return nil, err
	}

	// Return configured response
	if response, ok := m.responses[key]; ok {
		return &interfaces.ChatResponse{
			Model: req.Model,
			Choices: []interfaces.ChatChoice{{
				Message: interfaces.ChatMessage{
					Role:    "assistant",
					Content: response,
				},
				FinishReason: "stop",
			}},
			Usage: interfaces.UsageInfo{
				PromptTokens:     len(strings.Fields(req.Messages[0].Content)) * 2,
				CompletionTokens: len(strings.Fields(response)) * 2,
				TotalTokens:      len(strings.Fields(req.Messages[0].Content))*2 + len(strings.Fields(response))*2,
			},
		}, nil
	}

	// Default response
	defaultContent := "Mock response for: " + req.Messages[0].Content
	return &interfaces.ChatResponse{
		Model: req.Model,
		Choices: []interfaces.ChatChoice{{
			Message: interfaces.ChatMessage{
				Role:    "assistant",
				Content: defaultContent,
			},
			FinishReason: "stop",
		}},
		Usage: interfaces.UsageInfo{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

// StreamChatCompletion implements the AIProvider interface for streaming
func (m *MockLLMProvider) StreamChatCompletion(ctx context.Context, req interfaces.ChatRequest) (interfaces.ChatStream, error) {
	m.mu.Lock()
	key := req.Model
	if key == "" {
		key = "default"
	}
	m.callCount[key]++
	m.lastRequest[key] = &req
	chunks := m.streamResponses[key]
	delay := m.delays[key]
	err := m.errors[key]
	m.mu.Unlock()

	// Check for configured error
	if err != nil {
		return nil, err
	}

	// Default chunks if not configured
	if len(chunks) == 0 {
		chunks = []string{"Mock ", "streaming ", "response ", "for: ", req.Messages[0].Content}
	}

	return &mockChatStream{
		ctx:    ctx,
		chunks: chunks,
		delay:  delay,
		index:  0,
	}, nil
}

// CreateEmbedding implements the AIProvider interface
func (m *MockLLMProvider) CreateEmbedding(ctx context.Context, req interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	// Simple mock embedding
	embeddings := make([]interfaces.Embedding, len(req.Input))
	for i, input := range req.Input {
		embed := make([]float64, 768)
		for j := range embed {
			embed[j] = float64(len(input)%10) / 10.0
		}
		embeddings[i] = interfaces.Embedding{
			Index:     i,
			Embedding: embed,
		}
	}

	return &interfaces.EmbeddingResponse{
		Model:      req.Model,
		Embeddings: embeddings,
		Usage: interfaces.UsageInfo{
			PromptTokens:     len(req.Input) * 10,
			CompletionTokens: 0,
			TotalTokens:      len(req.Input) * 10,
		},
	}, nil
}

// GetCapabilities implements the AIProvider interface
func (m *MockLLMProvider) GetCapabilities() interfaces.ProviderCapabilities {
	return interfaces.ProviderCapabilities{
		MaxTokens:          4096,
		ContextWindow:      8192,
		SupportsVision:     false,
		SupportsTools:      true,
		SupportsStream:     true,
		SupportsEmbeddings: true,
		Models: []interfaces.ModelInfo{
			{
				ID:            "mock-model",
				Name:          "Mock Model",
				ContextWindow: 8192,
				MaxOutput:     4096,
			},
		},
	}
}

// Complete implements the LLMClient interface for legacy compatibility
func (m *MockLLMProvider) Complete(ctx context.Context, prompt string) (string, error) {
	// Convert to ChatRequest
	req := interfaces.ChatRequest{
		Model: "default",
		Messages: []interfaces.ChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Use ChatCompletion
	resp, err := m.ChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response choices")
}

// mockChatStream implements the ChatStream interface
type mockChatStream struct {
	ctx    context.Context
	chunks []string
	delay  time.Duration
	index  int
}

func (s *mockChatStream) Next() (interfaces.ChatStreamChunk, error) {
	if s.index >= len(s.chunks) {
		return interfaces.ChatStreamChunk{}, fmt.Errorf("stream ended")
	}

	// Apply delay if configured
	if s.delay > 0 && s.index > 0 {
		select {
		case <-time.After(s.delay / time.Duration(len(s.chunks))):
		case <-s.ctx.Done():
			return interfaces.ChatStreamChunk{}, s.ctx.Err()
		}
	}

	chunk := interfaces.ChatStreamChunk{
		Delta: interfaces.ChatMessage{
			Role:    "assistant",
			Content: s.chunks[s.index],
		},
	}

	s.index++
	if s.index >= len(s.chunks) {
		chunk.FinishReason = "stop"
	}

	return chunk, nil
}

func (s *mockChatStream) Close() error {
	return nil
}

// MockToolRegistry provides a test tool registry with pre-configured tools
type MockToolRegistry struct {
	mu          sync.RWMutex
	tools       map[string]tools.Tool
	execResults map[string]interface{}
	execErrors  map[string]error
	callCount   map[string]int
}

// NewMockToolRegistry creates a new mock tool registry
func NewMockToolRegistry() *MockToolRegistry {
	registry := &MockToolRegistry{
		tools:       make(map[string]tools.Tool),
		execResults: make(map[string]interface{}),
		execErrors:  make(map[string]error),
		callCount:   make(map[string]int),
	}

	// Add default test tools
	registry.RegisterTool("file", &mockFileTool{registry: registry})
	registry.RegisterTool("shell", &mockShellTool{registry: registry})
	registry.RegisterTool("http", &mockHTTPTool{registry: registry})

	return registry
}

// RegisterTool adds a tool to the registry
func (r *MockToolRegistry) RegisterTool(name string, tool tools.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = tool
	return nil
}

// GetTool retrieves a tool by name
func (r *MockToolRegistry) GetTool(name string) (tools.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// ListTools returns all registered tools
func (r *MockToolRegistry) ListTools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// SetExecutionResult configures the result for a tool execution
func (r *MockToolRegistry) SetExecutionResult(toolName string, result interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.execResults[toolName] = result
}

// SetExecutionError configures an error for a tool execution
func (r *MockToolRegistry) SetExecutionError(toolName string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.execErrors[toolName] = err
}

// GetCallCount returns the execution count for a tool
func (r *MockToolRegistry) GetCallCount(toolName string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.callCount[toolName]
}

// Mock tool implementations

type mockFileTool struct {
	registry *MockToolRegistry
}

func (t *mockFileTool) Name() string {
	return "file"
}

func (t *mockFileTool) Description() string {
	return "Mock file operations"
}

func (t *mockFileTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}

func (t *mockFileTool) Category() string {
	return "filesystem"
}

func (t *mockFileTool) RequiresAuth() bool {
	return false
}

func (t *mockFileTool) Examples() []string {
	return []string{"example"}
}

func (t *mockFileTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	t.registry.mu.Lock()
	t.registry.callCount["file"]++
	result := t.registry.execResults["file"]
	err := t.registry.execErrors["file"]
	t.registry.mu.Unlock()

	if err != nil {
		return nil, err
	}
	if result != nil {
		// Convert to ToolResult if not already
		if tr, ok := result.(*tools.ToolResult); ok {
			return tr, nil
		}
	}

	// Default behavior
	return tools.NewToolResult(
		"Mock file operation completed",
		map[string]string{"action": "file operation"},
		nil,
		map[string]interface{}{"input": input},
	), nil
}

func (t *mockFileTool) HealthCheck() error {
	return nil
}

type mockShellTool struct {
	registry *MockToolRegistry
}

func (t *mockShellTool) Name() string {
	return "shell"
}

func (t *mockShellTool) Description() string {
	return "Mock shell commands"
}

func (t *mockShellTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}

func (t *mockShellTool) Category() string {
	return "shell"
}

func (t *mockShellTool) RequiresAuth() bool {
	return false
}

func (t *mockShellTool) Examples() []string {
	return []string{"example"}
}

func (t *mockShellTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	t.registry.mu.Lock()
	t.registry.callCount["shell"]++
	result := t.registry.execResults["shell"]
	err := t.registry.execErrors["shell"]
	t.registry.mu.Unlock()

	if err != nil {
		return nil, err
	}
	if result != nil {
		// Convert to ToolResult if not already
		if tr, ok := result.(*tools.ToolResult); ok {
			return tr, nil
		}
	}

	// Default behavior
	return tools.NewToolResult(
		"Mock shell output",
		map[string]string{"command": "mock"},
		nil,
		map[string]interface{}{"input": input},
	), nil
}

func (t *mockShellTool) HealthCheck() error {
	return nil
}

type mockHTTPTool struct {
	registry *MockToolRegistry
}

func (t *mockHTTPTool) Name() string {
	return "http"
}

func (t *mockHTTPTool) Description() string {
	return "Mock HTTP requests"
}

func (t *mockHTTPTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}

func (t *mockHTTPTool) Category() string {
	return "network"
}

func (t *mockHTTPTool) RequiresAuth() bool {
	return false
}

func (t *mockHTTPTool) Examples() []string {
	return []string{"example"}
}

func (t *mockHTTPTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	t.registry.mu.Lock()
	t.registry.callCount["http"]++
	result := t.registry.execResults["http"]
	err := t.registry.execErrors["http"]
	t.registry.mu.Unlock()

	if err != nil {
		return nil, err
	}
	if result != nil {
		// Convert to ToolResult if not already
		if tr, ok := result.(*tools.ToolResult); ok {
			return tr, nil
		}
	}

	// Default behavior
	return tools.NewToolResult(
		`{"message": "Mock HTTP response"}`,
		map[string]string{"status": "200"},
		nil,
		map[string]interface{}{"input": input},
	), nil
}

func (t *mockHTTPTool) HealthCheck() error {
	return nil
}

// MockEventBus provides a test event bus for testing event-driven flows
type MockEventBus struct {
	mu        sync.RWMutex
	events    []interface{}
	handlers  map[string][]func(interface{})
	blockSend bool
	sendDelay time.Duration
}

// NewMockEventBus creates a new mock event bus
func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		events:   make([]interface{}, 0),
		handlers: make(map[string][]func(interface{})),
	}
}

// Publish sends an event
func (b *MockEventBus) Publish(eventType string, event interface{}) error {
	b.mu.Lock()
	b.events = append(b.events, event)
	handlers := b.handlers[eventType]
	delay := b.sendDelay
	blocked := b.blockSend
	b.mu.Unlock()

	if blocked {
		return fmt.Errorf("event bus blocked")
	}

	if delay > 0 {
		time.Sleep(delay)
	}

	// Call handlers
	for _, handler := range handlers {
		go handler(event)
	}

	return nil
}

// Subscribe registers an event handler
func (b *MockEventBus) Subscribe(eventType string, handler func(interface{})) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// GetEvents returns all published events
func (b *MockEventBus) GetEvents() []interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()
	events := make([]interface{}, len(b.events))
	copy(events, b.events)
	return events
}

// SetBlocked blocks event sending
func (b *MockEventBus) SetBlocked(blocked bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.blockSend = blocked
}

// SetDelay adds delay to event sending
func (b *MockEventBus) SetDelay(delay time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sendDelay = delay
}

// Clear removes all events
func (b *MockEventBus) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = b.events[:0]
}

// MockVectorStore provides a test vector store for RAG testing
type MockVectorStore struct {
	mu         sync.RWMutex
	documents  map[string]*vector.Document
	embeddings map[string][]float32
	searchFunc func(query []float32, k int) ([]*vector.Document, error)
}

// NewMockVectorStore creates a new mock vector store
func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		documents:  make(map[string]*vector.Document),
		embeddings: make(map[string][]float32),
	}
}

// Add stores a document
func (s *MockVectorStore) Add(ctx context.Context, doc *vector.Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.documents[doc.ID] = doc
	// Generate mock embedding
	s.embeddings[doc.ID] = generateMockEmbedding(doc.Content)
	return nil
}

// Search performs similarity search
func (s *MockVectorStore) Search(ctx context.Context, query []float32, k int, filters map[string]interface{}) ([]*vector.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.searchFunc != nil {
		return s.searchFunc(query, k)
	}

	// Default behavior: return first k documents
	results := make([]*vector.Document, 0, k)
	for _, doc := range s.documents {
		if len(results) >= k {
			break
		}
		// Apply filters if any
		if filters != nil && !matchFilters(doc, filters) {
			continue
		}
		results = append(results, doc)
	}

	return results, nil
}

// Delete removes a document
func (s *MockVectorStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.documents, id)
	delete(s.embeddings, id)
	return nil
}

// GetDocument retrieves a document by ID
func (s *MockVectorStore) GetDocument(id string) *vector.Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.documents[id]
}

// SetSearchFunction allows custom search behavior
func (s *MockVectorStore) SetSearchFunction(fn func(query []float32, k int) ([]*vector.Document, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searchFunc = fn
}

// Count returns the number of documents
func (s *MockVectorStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.documents)
}

// SaveEmbedding stores a vector embedding (implements VectorStore interface)
func (s *MockVectorStore) SaveEmbedding(ctx context.Context, embedding vector.Embedding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Convert Embedding to Document for storage
	doc := &vector.Document{
		ID:       embedding.ID,
		Content:  embedding.Text,
		Metadata: embedding.Metadata,
	}
	s.documents[doc.ID] = doc
	s.embeddings[doc.ID] = embedding.Vector
	return nil
}

// QueryEmbeddings performs a similarity search (implements VectorStore interface)
func (s *MockVectorStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]vector.EmbeddingMatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// For mock, return first N documents as matches
	matches := make([]vector.EmbeddingMatch, 0, limit)
	for id, doc := range s.documents {
		if len(matches) >= limit {
			break
		}
		// Convert metadata if needed
		metadata := make(map[string]interface{})
		if doc.Metadata != nil {
			if m, ok := doc.Metadata.(map[string]interface{}); ok {
				metadata = m
			}
		}

		matches = append(matches, vector.EmbeddingMatch{
			ID:        id,
			Text:      doc.Content,
			Source:    fmt.Sprintf("mock:%s", id),
			Score:     0.95, // Mock high similarity score
			Timestamp: time.Now(),
			Metadata:  metadata,
		})
	}

	return matches, nil
}

// QueryCollection performs a similarity search on a specific collection
func (s *MockVectorStore) QueryCollection(ctx context.Context, collectionName, query string, limit int) ([]vector.EmbeddingMatch, error) {
	// For simplicity, just delegate to QueryEmbeddings
	return s.QueryEmbeddings(ctx, query, limit)
}

// DeleteEmbedding removes an embedding by ID
func (s *MockVectorStore) DeleteEmbedding(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.documents, id)
	delete(s.embeddings, id)
	return nil
}

// Close closes the vector store (implements VectorStore interface)
func (s *MockVectorStore) Close() error {
	// Nothing to close in mock implementation
	return nil
}

// Helper functions

func generateMockEmbedding(content string) []float32 {
	// Simple mock embedding based on content length
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = float32(len(content)%10) / 10.0
	}
	return embedding
}

func matchFilters(doc *vector.Document, filters map[string]interface{}) bool {
	// Check if metadata is a map
	metadata, ok := doc.Metadata.(map[string]interface{})
	if !ok {
		return false
	}

	for key, value := range filters {
		if docValue, ok := metadata[key]; !ok || docValue != value {
			return false
		}
	}
	return true
}
