// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"context"
	"encoding/json"

	"github.com/guild-ventures/guild-core/tools"
)

// MockTool is a mock implementation of the Tool interface for testing
type MockTool struct {
	NameValue       string
	DescValue       string
	SchemaValue     map[string]interface{}
	CategoryValue   string
	NeedsAuthValue  bool
	ExamplesValue   []string
	ResultValue     *tools.ToolResult
	ErrorValue      error
	ExecuteCount    int
	LastInputValue  string
	LastParamsValue map[string]interface{}
}

// NewMockTool creates a new mock tool
func NewMockTool(name, description string) *MockTool {
	return &MockTool{
		NameValue:     name,
		DescValue:     description,
		CategoryValue: "test",
		SchemaValue:   make(map[string]interface{}),
		ExamplesValue: []string{"example input"},
		ResultValue: &tools.ToolResult{
			Output:  "Mock tool result",
			Success: true,
		},
	}
}

// WithResult sets the result to return from Execute
func (t *MockTool) WithResult(result *tools.ToolResult) *MockTool {
	t.ResultValue = result
	return t
}

// WithError sets an error to return from Execute
func (t *MockTool) WithError(err error) *MockTool {
	t.ErrorValue = err
	return t
}

// WithCategory sets the category of the tool
func (t *MockTool) WithCategory(category string) *MockTool {
	t.CategoryValue = category
	return t
}

// WithSchema sets the input schema for the tool
func (t *MockTool) WithSchema(schema map[string]interface{}) *MockTool {
	t.SchemaValue = schema
	return t
}

// WithExamples sets example inputs for the tool
func (t *MockTool) WithExamples(examples []string) *MockTool {
	t.ExamplesValue = examples
	return t
}

// WithRequiresAuth sets whether the tool requires authentication
func (t *MockTool) WithRequiresAuth(needsAuth bool) *MockTool {
	t.NeedsAuthValue = needsAuth
	return t
}

// Name returns the name of the tool
func (t *MockTool) Name() string {
	return t.NameValue
}

// Description returns a description of what the tool does
func (t *MockTool) Description() string {
	return t.DescValue
}

// Schema returns the JSON schema for the tool's input parameters
func (t *MockTool) Schema() map[string]interface{} {
	return t.SchemaValue
}

// Execute runs the tool with the given input and returns the result
func (t *MockTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue execution
	}

	t.ExecuteCount++
	t.LastInputValue = input

	// Try to parse the input as JSON params
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(input), &params); err == nil {
		t.LastParamsValue = params
	}

	if t.ErrorValue != nil {
		return nil, t.ErrorValue
	}

	return t.ResultValue, nil
}

// Examples returns a list of example inputs for the tool
func (t *MockTool) Examples() []string {
	return t.ExamplesValue
}

// Category returns the category of the tool
func (t *MockTool) Category() string {
	return t.CategoryValue
}

// RequiresAuth returns whether the tool requires authentication
func (t *MockTool) RequiresAuth() bool {
	return t.NeedsAuthValue
}
