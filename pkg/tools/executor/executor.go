// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/tools"
	"github.com/lancekrogers/guild/pkg/tools/parser"
)

// toolExecutor implements parser.ToolExecutor with proper context handling
type toolExecutor struct {
	registry *tools.ToolRegistry
	mu       sync.RWMutex
}

// NewToolExecutor creates a new tool executor with the given registry
func NewToolExecutor(registry *tools.ToolRegistry) parser.ToolExecutor {
	if registry == nil {
		panic("tool registry cannot be nil")
	}

	return &toolExecutor{
		registry: registry,
	}
}

// Execute runs a single tool call with full context support
func (e *toolExecutor) Execute(ctx context.Context, call parser.ToolCall) (*parser.ToolResult, error) {
	// Check context at the start
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCanceled, "context canceled before execution").
			WithComponent("ToolExecutor").
			WithOperation("Execute").
			WithDetails("tool", call.Function.Name)
	}

	start := time.Now()

	result := &parser.ToolResult{
		ID:      call.ID,
		Success: false,
	}

	// Get the tool from registry
	e.mu.RLock()
	tool, err := e.registry.GetTool(call.Function.Name)
	e.mu.RUnlock()

	if err != nil {
		errMsg := fmt.Sprintf("tool not found: %s", call.Function.Name)
		result.Error = errMsg
		result.Duration = time.Since(start)
		return result, gerror.Wrap(err, gerror.ErrCodeNotFound, errMsg).
			WithComponent("ToolExecutor").
			WithOperation("Execute").
			WithDetails("tool", call.Function.Name).
			WithDetails("availableTools", e.getToolNames())
	}

	// Convert arguments to string for tool execution
	argString := string(call.Function.Arguments)
	if argString == "" {
		argString = "{}"
	}

	// Validate arguments are valid JSON
	var argCheck interface{}
	if err := json.Unmarshal([]byte(argString), &argCheck); err != nil {
		errMsg := "invalid tool arguments"
		result.Error = errMsg
		result.Duration = time.Since(start)
		return result, gerror.Wrap(err, gerror.ErrCodeValidation, errMsg).
			WithComponent("ToolExecutor").
			WithOperation("Execute").
			WithDetails("tool", call.Function.Name).
			WithDetails("arguments", argString)
	}

	// Execute the tool with timeout from context
	execCtx := ctx
	if deadline, ok := ctx.Deadline(); ok {
		// Create a slightly shorter timeout to allow for cleanup
		timeout := time.Until(deadline) - 100*time.Millisecond
		if timeout > 0 {
			var cancel context.CancelFunc
			execCtx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
	}

	// Execute in a goroutine to respect context cancellation
	type execResult struct {
		result *tools.ToolResult
		err    error
	}
	execCh := make(chan execResult, 1)

	go func() {
		toolResult, err := tool.Execute(execCtx, argString)
		execCh <- execResult{result: toolResult, err: err}
	}()

	// Wait for execution or context cancellation
	select {
	case <-ctx.Done():
		result.Error = "execution canceled"
		result.Duration = time.Since(start)
		return result, gerror.Wrap(ctx.Err(), gerror.ErrCodeCanceled, "tool execution canceled").
			WithComponent("ToolExecutor").
			WithOperation("Execute").
			WithDetails("tool", call.Function.Name).
			WithDetails("duration", time.Since(start).String())

	case exec := <-execCh:
		if exec.err != nil {
			result.Error = exec.err.Error()
			result.Duration = time.Since(start)
			// Don't propagate error - include it in result for the agent to handle
			return result, nil
		}

		// Convert tool result to our format
		result.Success = true
		result.Content = exec.result.Output
		if exec.result.ExtraData != nil {
			result.Output = exec.result.ExtraData
		}
		result.Duration = time.Since(start)

		return result, nil
	}
}

// ExecuteBatch runs multiple tool calls with proper error handling
func (e *toolExecutor) ExecuteBatch(ctx context.Context, calls []parser.ToolCall) ([]*parser.ToolResult, error) {
	if len(calls) == 0 {
		return nil, nil
	}

	// Check context at the start
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCanceled, "context canceled before batch execution").
			WithComponent("ToolExecutor").
			WithOperation("ExecuteBatch").
			WithDetails("toolCount", len(calls))
	}

	results := make([]*parser.ToolResult, len(calls))
	var wg sync.WaitGroup

	// Execute tools in parallel with proper synchronization
	for i, call := range calls {
		wg.Add(1)
		go func(idx int, toolCall parser.ToolCall) {
			defer wg.Done()

			result, err := e.Execute(ctx, toolCall)
			if err != nil {
				// Create error result
				result = &parser.ToolResult{
					ID:       toolCall.ID,
					Success:  false,
					Error:    err.Error(),
					Duration: 0,
				}
			}
			results[idx] = result
		}(i, call)
	}

	// Wait for all executions to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for completion or context cancellation
	select {
	case <-ctx.Done():
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCanceled, "batch execution canceled").
			WithComponent("ToolExecutor").
			WithOperation("ExecuteBatch").
			WithDetails("toolCount", len(calls))
	case <-done:
		return results, nil
	}
}

// GetAvailableTools returns all registered tools as standardized definitions
func (e *toolExecutor) GetAvailableTools() []parser.ToolDefinition {
	e.mu.RLock()
	defer e.mu.RUnlock()

	definitions := make([]parser.ToolDefinition, 0)

	// Get all tools from registry
	toolNames := e.registry.ListTools()
	for _, name := range toolNames {
		tool, err := e.registry.GetTool(name)
		if err != nil {
			continue
		}

		// Convert to standard definition
		def := parser.ToolDefinition{
			Type: "function",
			Function: parser.FunctionDefinition{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  e.extractParameters(tool),
			},
		}

		definitions = append(definitions, def)
	}

	return definitions
}

// extractParameters extracts parameter schema from a tool with error handling
func (e *toolExecutor) extractParameters(tool tools.Tool) json.RawMessage {
	// Default empty schema
	emptySchema := json.RawMessage(`{"type":"object","properties":{}}`)

	// Get the schema from the tool - it returns map[string]interface{}
	schema := tool.Schema()
	if schema == nil || len(schema) == 0 {
		return emptySchema
	}

	// Convert map to JSON
	data, err := json.Marshal(schema)
	if err != nil {
		return emptySchema
	}
	return data
}

// getToolNames returns a list of available tool names for error messages
func (e *toolExecutor) getToolNames() []string {
	return e.registry.ListTools()
}
