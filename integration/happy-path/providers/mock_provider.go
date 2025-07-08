// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package providers

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/providers"
	"github.com/lancekrogers/guild/pkg/providers/interfaces"
)

// MockAIProvider implements a mock AI provider for testing
type MockAIProvider struct {
	providerType   providers.ProviderType
	capabilities   interfaces.ProviderCapabilities
	failureRate    float64
	latencyBase    time.Duration
	qualityScore   float64
	costMultiplier float64
	requestCount   int64
	failureCount   int64
}

// MockChatStream implements a mock chat stream
type MockChatStream struct {
	chunks   []interfaces.ChatStreamChunk
	index    int
	closed   bool
	provider providers.ProviderType
}

// NewMockAIProvider creates a new mock AI provider
func NewMockAIProvider(providerType providers.ProviderType) *MockAIProvider {
	capabilities := generateProviderCapabilities(providerType)

	return &MockAIProvider{
		providerType:   providerType,
		capabilities:   capabilities,
		failureRate:    getProviderFailureRate(providerType),
		latencyBase:    getProviderLatencyBase(providerType),
		qualityScore:   getProviderQualityScore(providerType),
		costMultiplier: getProviderCostMultiplier(providerType),
	}
}

// generateProviderCapabilities generates realistic capabilities for each provider
func generateProviderCapabilities(providerType providers.ProviderType) interfaces.ProviderCapabilities {
	switch providerType {
	case providers.ProviderOpenAI:
		return interfaces.ProviderCapabilities{
			MaxTokens:          4096,
			ContextWindow:      16384,
			SupportsVision:     true,
			SupportsTools:      true,
			SupportsStream:     true,
			SupportsEmbeddings: true,
			Models: []interfaces.ModelInfo{
				{
					ID:            "gpt-4",
					Name:          "GPT-4",
					ContextWindow: 8192,
					MaxOutput:     4096,
					InputCost:     30.0,
					OutputCost:    60.0,
				},
				{
					ID:            "gpt-3.5-turbo",
					Name:          "GPT-3.5 Turbo",
					ContextWindow: 4096,
					MaxOutput:     4096,
					InputCost:     1.5,
					OutputCost:    2.0,
				},
			},
		}
	case providers.ProviderAnthropic:
		return interfaces.ProviderCapabilities{
			MaxTokens:          4096,
			ContextWindow:      200000,
			SupportsVision:     true,
			SupportsTools:      true,
			SupportsStream:     true,
			SupportsEmbeddings: false,
			Models: []interfaces.ModelInfo{
				{
					ID:            "claude-3-opus",
					Name:          "Claude 3 Opus",
					ContextWindow: 200000,
					MaxOutput:     4096,
					InputCost:     15.0,
					OutputCost:    75.0,
				},
				{
					ID:            "claude-3-sonnet",
					Name:          "Claude 3 Sonnet",
					ContextWindow: 200000,
					MaxOutput:     4096,
					InputCost:     3.0,
					OutputCost:    15.0,
				},
			},
		}
	case providers.ProviderOllama:
		return interfaces.ProviderCapabilities{
			MaxTokens:          2048,
			ContextWindow:      4096,
			SupportsVision:     false,
			SupportsTools:      false,
			SupportsStream:     true,
			SupportsEmbeddings: true,
			Models: []interfaces.ModelInfo{
				{
					ID:            "llama2",
					Name:          "Llama 2",
					ContextWindow: 4096,
					MaxOutput:     2048,
					InputCost:     0.0,
					OutputCost:    0.0,
				},
				{
					ID:            "codellama",
					Name:          "Code Llama",
					ContextWindow: 4096,
					MaxOutput:     2048,
					InputCost:     0.0,
					OutputCost:    0.0,
				},
			},
		}
	case providers.ProviderDeepSeek:
		return interfaces.ProviderCapabilities{
			MaxTokens:          4096,
			ContextWindow:      32768,
			SupportsVision:     false,
			SupportsTools:      true,
			SupportsStream:     true,
			SupportsEmbeddings: true,
			Models: []interfaces.ModelInfo{
				{
					ID:            "deepseek-coder",
					Name:          "DeepSeek Coder",
					ContextWindow: 32768,
					MaxOutput:     4096,
					InputCost:     0.14,
					OutputCost:    0.28,
				},
				{
					ID:            "deepseek-chat",
					Name:          "DeepSeek Chat",
					ContextWindow: 32768,
					MaxOutput:     4096,
					InputCost:     0.14,
					OutputCost:    0.28,
				},
			},
		}
	case providers.ProviderOra:
		return interfaces.ProviderCapabilities{
			MaxTokens:          2048,
			ContextWindow:      8192,
			SupportsVision:     false,
			SupportsTools:      false,
			SupportsStream:     true,
			SupportsEmbeddings: false,
			Models: []interfaces.ModelInfo{
				{
					ID:            "ora-model",
					Name:          "Ora Model",
					ContextWindow: 8192,
					MaxOutput:     2048,
					InputCost:     1.0,
					OutputCost:    2.0,
				},
			},
		}
	default:
		return interfaces.ProviderCapabilities{
			MaxTokens:          2048,
			ContextWindow:      4096,
			SupportsVision:     false,
			SupportsTools:      false,
			SupportsStream:     false,
			SupportsEmbeddings: false,
			Models: []interfaces.ModelInfo{
				{
					ID:            "default-model",
					Name:          "Default Model",
					ContextWindow: 4096,
					MaxOutput:     2048,
					InputCost:     1.0,
					OutputCost:    1.0,
				},
			},
		}
	}
}

// getProviderFailureRate returns realistic failure rates for each provider
func getProviderFailureRate(providerType providers.ProviderType) float64 {
	switch providerType {
	case providers.ProviderOpenAI:
		return 0.02 // 2% failure rate
	case providers.ProviderAnthropic:
		return 0.01 // 1% failure rate
	case providers.ProviderOllama:
		return 0.05 // 5% failure rate (local, more variable)
	case providers.ProviderDeepSeek:
		return 0.03 // 3% failure rate
	case providers.ProviderOra:
		return 0.04 // 4% failure rate
	default:
		return 0.05 // 5% default failure rate
	}
}

// getProviderLatencyBase returns base latency for each provider
func getProviderLatencyBase(providerType providers.ProviderType) time.Duration {
	switch providerType {
	case providers.ProviderOpenAI:
		return 800 * time.Millisecond
	case providers.ProviderAnthropic:
		return 600 * time.Millisecond
	case providers.ProviderOllama:
		return 200 * time.Millisecond // Local, faster
	case providers.ProviderDeepSeek:
		return 1200 * time.Millisecond
	case providers.ProviderOra:
		return 1000 * time.Millisecond
	default:
		return 1000 * time.Millisecond
	}
}

// getProviderQualityScore returns quality score for each provider
func getProviderQualityScore(providerType providers.ProviderType) float64 {
	switch providerType {
	case providers.ProviderOpenAI:
		return 0.95
	case providers.ProviderAnthropic:
		return 0.97
	case providers.ProviderOllama:
		return 0.85
	case providers.ProviderDeepSeek:
		return 0.90
	case providers.ProviderOra:
		return 0.88
	default:
		return 0.80
	}
}

// getProviderCostMultiplier returns cost multiplier for each provider
func getProviderCostMultiplier(providerType providers.ProviderType) float64 {
	switch providerType {
	case providers.ProviderOpenAI:
		return 1.0 // Baseline
	case providers.ProviderAnthropic:
		return 0.8 // Slightly cheaper
	case providers.ProviderOllama:
		return 0.0 // Free (local)
	case providers.ProviderDeepSeek:
		return 0.1 // Much cheaper
	case providers.ProviderOra:
		return 0.5 // Moderately cheaper
	default:
		return 1.0
	}
}

// ChatCompletion implements chat completion
func (m *MockAIProvider) ChatCompletion(ctx context.Context, req interfaces.ChatRequest) (*interfaces.ChatResponse, error) {
	m.requestCount++

	// Simulate latency
	latency := m.calculateLatency()
	select {
	case <-time.After(latency):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Simulate failures
	if m.shouldFail() {
		m.failureCount++
		return nil, m.generateError()
	}

	// Generate mock response
	response := m.generateChatResponse(req)
	return response, nil
}

// StreamChatCompletion implements streaming chat completion
func (m *MockAIProvider) StreamChatCompletion(ctx context.Context, req interfaces.ChatRequest) (interfaces.ChatStream, error) {
	m.requestCount++

	// Simulate initial latency
	latency := m.calculateLatency() / 4 // Streaming starts faster
	select {
	case <-time.After(latency):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Simulate failures
	if m.shouldFail() {
		m.failureCount++
		return nil, m.generateError()
	}

	// Create mock stream
	stream := m.generateChatStream(req)
	return stream, nil
}

// CreateEmbedding implements embedding creation
func (m *MockAIProvider) CreateEmbedding(ctx context.Context, req interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	if !m.capabilities.SupportsEmbeddings {
		return nil, &interfaces.ProviderError{
			Provider:   string(m.providerType),
			StatusCode: 400,
			Type:       interfaces.ErrorTypeValidation,
			Message:    "provider does not support embeddings",
			Retryable:  false,
		}
	}

	m.requestCount++

	// Simulate latency
	latency := m.calculateLatency() / 2 // Embeddings are typically faster
	select {
	case <-time.After(latency):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Simulate failures
	if m.shouldFail() {
		m.failureCount++
		return nil, m.generateError()
	}

	// Generate mock embeddings
	response := m.generateEmbeddingResponse(req)
	return response, nil
}

// GetCapabilities returns provider capabilities
func (m *MockAIProvider) GetCapabilities() interfaces.ProviderCapabilities {
	return m.capabilities
}

// calculateLatency calculates realistic latency with variation
func (m *MockAIProvider) calculateLatency() time.Duration {
	// Add random variation (±30%)
	variation := 0.7 + rand.Float64()*0.6 // 0.7 to 1.3
	return time.Duration(float64(m.latencyBase) * variation)
}

// shouldFail determines if this request should fail
func (m *MockAIProvider) shouldFail() bool {
	return rand.Float64() < m.failureRate
}

// generateError generates a realistic error
func (m *MockAIProvider) generateError() error {
	errorTypes := []struct {
		errorType interfaces.ProviderError
		weight    float64
	}{
		{
			errorType: interfaces.ProviderError{
				Provider:   string(m.providerType),
				StatusCode: 500,
				Type:       interfaces.ErrorTypeServer,
				Message:    "internal server error",
				Retryable:  true,
			},
			weight: 0.4,
		},
		{
			errorType: interfaces.ProviderError{
				Provider:   string(m.providerType),
				StatusCode: 429,
				Type:       interfaces.ErrorTypeRateLimit,
				Message:    "rate limit exceeded",
				Retryable:  true,
			},
			weight: 0.3,
		},
		{
			errorType: interfaces.ProviderError{
				Provider:   string(m.providerType),
				StatusCode: 401,
				Type:       interfaces.ErrorTypeAuth,
				Message:    "authentication failed",
				Retryable:  false,
			},
			weight: 0.2,
		},
		{
			errorType: interfaces.ProviderError{
				Provider:   string(m.providerType),
				StatusCode: 400,
				Type:       interfaces.ErrorTypeValidation,
				Message:    "invalid request parameters",
				Retryable:  false,
			},
			weight: 0.1,
		},
	}

	// Select error type based on weights
	r := rand.Float64()
	cumWeight := 0.0
	for _, errorType := range errorTypes {
		cumWeight += errorType.weight
		if r <= cumWeight {
			return &errorType.errorType
		}
	}

	// Fallback
	return &errorTypes[0].errorType
}

// generateChatResponse generates a mock chat response
func (m *MockAIProvider) generateChatResponse(req interfaces.ChatRequest) *interfaces.ChatResponse {
	// Generate response based on provider characteristics
	content := m.generateResponseContent(req)

	response := &interfaces.ChatResponse{
		ID:    fmt.Sprintf("resp-%s-%d", m.providerType, time.Now().UnixNano()),
		Model: req.Model,
		Choices: []interfaces.ChatChoice{
			{
				Index: 0,
				Message: interfaces.ChatMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: interfaces.UsageInfo{
			PromptTokens:     m.estimateTokens(req.Messages),
			CompletionTokens: m.estimateTokens([]interfaces.ChatMessage{{Content: content}}),
		},
		FinishReason: "stop",
	}

	response.Usage.TotalTokens = response.Usage.PromptTokens + response.Usage.CompletionTokens
	return response
}

// generateChatStream generates a mock chat stream
func (m *MockAIProvider) generateChatStream(req interfaces.ChatRequest) *MockChatStream {
	content := m.generateResponseContent(req)

	// Split content into chunks for streaming
	words := []string{"Hello", "there!", "I'm", "a", "mock", "response", "from", string(m.providerType) + ".", "This", "simulates", "streaming", "behavior."}
	if len(content) > 100 {
		words = append(words, "Here's", "some", "additional", "content", "to", "make", "it", "more", "realistic.")
	}

	chunks := make([]interfaces.ChatStreamChunk, 0, len(words)+1)

	for i, word := range words {
		chunk := interfaces.ChatStreamChunk{
			Delta: interfaces.ChatMessage{
				Role:    "assistant",
				Content: word + " ",
			},
		}
		if i == len(words)-1 {
			chunk.FinishReason = "stop"
		}
		chunks = append(chunks, chunk)
	}

	// Add final chunk
	chunks = append(chunks, interfaces.ChatStreamChunk{
		Delta:        interfaces.ChatMessage{},
		FinishReason: "stop",
	})

	return &MockChatStream{
		chunks:   chunks,
		provider: m.providerType,
	}
}

// generateResponseContent generates response content based on provider type
func (m *MockAIProvider) generateResponseContent(req interfaces.ChatRequest) string {
	baseResponses := map[providers.ProviderType]string{
		providers.ProviderOpenAI:    "I'm an OpenAI-powered assistant. I can help you with a wide variety of tasks including writing, analysis, coding, and creative projects.",
		providers.ProviderAnthropic: "Hello! I'm Claude, an AI assistant created by Anthropic. I'm designed to be helpful, harmless, and honest in my interactions.",
		providers.ProviderOllama:    "I'm running locally via Ollama. I may have different capabilities than cloud-based models but can process your requests privately.",
		providers.ProviderDeepSeek:  "I'm DeepSeek, an AI model particularly strong in coding and technical tasks. How can I assist you today?",
		providers.ProviderOra:       "Hello! I'm an AI assistant powered by Ora. I'm here to help you with your questions and tasks.",
	}

	baseResponse, exists := baseResponses[m.providerType]
	if !exists {
		baseResponse = "I'm an AI assistant. How can I help you today?"
	}

	// Add quality variation based on provider score
	if m.qualityScore > 0.9 {
		baseResponse += " I strive to provide accurate and detailed responses."
	} else if m.qualityScore < 0.8 {
		baseResponse += " I'll do my best to help."
	}

	// Simulate different response styles
	if req.Messages != nil && len(req.Messages) > 0 {
		lastMessage := req.Messages[len(req.Messages)-1]
		if len(lastMessage.Content) > 100 {
			baseResponse += " I notice you've provided a detailed question. Let me address the key points."
		}
	}

	return baseResponse
}

// estimateTokens estimates token count for messages
func (m *MockAIProvider) estimateTokens(messages []interfaces.ChatMessage) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Content) + len(msg.Role) + 10 // Add overhead
	}
	// Rough estimate: 4 characters per token
	return totalChars / 4
}

// generateEmbeddingResponse generates a mock embedding response
func (m *MockAIProvider) generateEmbeddingResponse(req interfaces.EmbeddingRequest) *interfaces.EmbeddingResponse {
	embeddings := make([]interfaces.Embedding, len(req.Input))

	for i, input := range req.Input {
		// Generate a mock embedding vector (384 dimensions)
		vector := make([]float64, 384)
		for j := range vector {
			vector[j] = rand.Float64()*2 - 1 // Random values between -1 and 1
		}

		embeddings[i] = interfaces.Embedding{
			Index:     i,
			Embedding: vector,
		}
	}

	totalTokens := 0
	for _, input := range req.Input {
		totalTokens += len(input) / 4 // Rough token estimate
	}

	return &interfaces.EmbeddingResponse{
		Model:      req.Model,
		Embeddings: embeddings,
		Usage: interfaces.UsageInfo{
			PromptTokens:     totalTokens,
			CompletionTokens: 0,
			TotalTokens:      totalTokens,
		},
	}
}

// MockChatStream methods

// Next returns the next chunk in the stream
func (s *MockChatStream) Next() (interfaces.ChatStreamChunk, error) {
	if s.closed {
		return interfaces.ChatStreamChunk{}, io.EOF
	}

	if s.index >= len(s.chunks) {
		s.closed = true
		return interfaces.ChatStreamChunk{}, io.EOF
	}

	// Simulate streaming delay
	time.Sleep(50 * time.Millisecond)

	chunk := s.chunks[s.index]
	s.index++

	return chunk, nil
}

// Close closes the stream
func (s *MockChatStream) Close() error {
	s.closed = true
	return nil
}

// GetStats returns provider statistics
func (m *MockAIProvider) GetStats() map[string]interface{} {
	successRate := 1.0
	if m.requestCount > 0 {
		successRate = float64(m.requestCount-m.failureCount) / float64(m.requestCount)
	}

	return map[string]interface{}{
		"provider":        string(m.providerType),
		"request_count":   m.requestCount,
		"failure_count":   m.failureCount,
		"success_rate":    successRate,
		"failure_rate":    m.failureRate,
		"quality_score":   m.qualityScore,
		"cost_multiplier": m.costMultiplier,
		"avg_latency_ms":  m.latencyBase.Milliseconds(),
	}
}

// SetFailureRate sets the failure rate for testing
func (m *MockAIProvider) SetFailureRate(rate float64) {
	m.failureRate = rate
}

// SetLatencyBase sets the base latency for testing
func (m *MockAIProvider) SetLatencyBase(latency time.Duration) {
	m.latencyBase = latency
}

// SetQualityScore sets the quality score for testing
func (m *MockAIProvider) SetQualityScore(score float64) {
	m.qualityScore = score
}

// Reset resets the provider statistics
func (m *MockAIProvider) Reset() {
	m.requestCount = 0
	m.failureCount = 0
}
