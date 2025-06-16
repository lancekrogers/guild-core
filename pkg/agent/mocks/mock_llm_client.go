// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"context"
)

// MockLLMClient implements the providers.LLMClient interface for testing
type MockLLMClient struct {
	// Response to return for Complete calls
	Response string
	// Error to return
	Error error
	// Track calls for assertions
	CallCount  int
	LastPrompt string
}

// NewMockLLMClient creates a new mock LLM client
func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		Response: "Mock response",
	}
}

// Complete implements the LLMClient.Complete method
func (m *MockLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue execution
	}

	// Track the call
	m.CallCount++
	m.LastPrompt = prompt

	// Return configured error if set
	if m.Error != nil {
		return "", m.Error
	}

	// Return configured response
	return m.Response, nil
}

// WithResponse configures the response
func (m *MockLLMClient) WithResponse(response string) *MockLLMClient {
	m.Response = response
	return m
}

// WithError configures an error response
func (m *MockLLMClient) WithError(err error) *MockLLMClient {
	m.Error = err
	return m
}

// Reset resets the mock state
func (m *MockLLMClient) Reset() {
	m.CallCount = 0
	m.LastPrompt = ""
}
