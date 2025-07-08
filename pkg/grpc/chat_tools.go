// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/lancekrogers/guild/pkg/agents/core"
	"github.com/lancekrogers/guild/pkg/gerror"
	pb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/providers/interfaces"
	"github.com/lancekrogers/guild/pkg/tools"
	"github.com/lancekrogers/guild/pkg/tools/executor"
	"github.com/lancekrogers/guild/pkg/tools/parser"
)

// executeAgentResponseWithTools extends executeAgentResponse to handle tool calls
func (s *ChatService) executeAgentResponseWithTools(ctx context.Context, ag core.Agent, msg *pb.ChatMessage, session *ChatSession, stream pb.ChatService_ChatServer) (string, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("grpc").
		WithOperation("executeAgentResponseWithTools").
		With("agent_id", ag.GetID()).
		With("session_id", session.ID)

	// Check if agent supports tools
	toolAgent, supportsTools := ag.(core.ToolAgent)
	if !supportsTools {
		// Fallback to regular execution
		logger.Debug("Agent does not support tools, using regular execution")
		return s.executeAgentResponse(ctx, ag, msg)
	}

	// Get available tools
	toolRegistry := s.registry.Tools()
	if toolRegistry == nil {
		logger.Warn("Tool registry not available, proceeding without tools")
		return s.executeAgentResponse(ctx, ag, msg)
	}

	// Try to get the underlying registry for the executor
	var toolExecutor parser.ToolExecutor

	// Check if this is a DefaultToolRegistry which has GetUnderlyingRegistry method
	type underlyingRegistryGetter interface {
		GetUnderlyingRegistry() *tools.ToolRegistry
	}

	if getter, ok := toolRegistry.(underlyingRegistryGetter); ok {
		toolExecutor = executor.NewToolExecutor(getter.GetUnderlyingRegistry())
	} else {
		// Fallback: create a minimal executor or handle differently
		logger.Warn("Tool registry does not provide underlying registry access, cannot create executor")
		return s.executeAgentResponse(ctx, ag, msg)
	}
	availableTools := toolExecutor.GetAvailableTools()

	logger.Info("Executing agent with tool support",
		"available_tools", len(availableTools),
	)

	// Convert to provider tool definitions
	providerTools := make([]interfaces.ToolDefinition, len(availableTools))
	for i, tool := range availableTools {
		providerTools[i] = interfaces.ToolDefinition{
			Type: tool.Type,
			Function: interfaces.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}

	// Execute agent with tools
	response, toolCalls, err := toolAgent.ExecuteWithTools(ctx, msg.Content, providerTools)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "agent execution with tools failed").
			WithComponent("grpc").
			WithOperation("executeAgentResponseWithTools").
			WithDetails("agent_id", ag.GetID())
	}

	// If no tool calls, return the response
	if len(toolCalls) == 0 {
		logger.Debug("No tool calls in agent response")
		return response, nil
	}

	logger.Info("Agent requested tool execution",
		"tool_count", len(toolCalls),
	)

	// Process tool calls
	for _, toolCall := range toolCalls {
		// Create tool execution record
		toolExecID := uuid.New().String()

		// Parse arguments into map[string]string
		var params map[string]string
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			logger.WithError(err).Warn("Failed to parse tool arguments, using empty parameters")
			params = make(map[string]string)
		}

		toolExec := &pb.ToolExecution{
			ToolId:     toolExecID,
			SessionId:  session.ID,
			AgentId:    ag.GetID(),
			ToolName:   toolCall.Function.Name,
			Parameters: params,
			Status:     pb.ToolExecution_AWAITING_APPROVAL,
			StartedAt:  time.Now().Unix(),
		}

		// Store in session
		session.toolsMu.Lock()
		session.toolExecutions[toolExecID] = toolExec
		session.toolsMu.Unlock()

		// Send tool approval request
		approvalReq := &pb.ChatResponse{
			Response: &pb.ChatResponse_ToolExecution{
				ToolExecution: toolExec,
			},
		}

		if err := stream.Send(approvalReq); err != nil {
			logger.WithError(err).Error("Failed to send tool approval request")
			continue
		}

		// Wait for approval (with timeout)
		approved, err := s.waitForToolApproval(ctx, session, toolExecID, 30*time.Second)
		if err != nil {
			logger.WithError(err).Warn("Tool approval failed", "tool_id", toolExecID)
			continue
		}

		if !approved {
			logger.Info("Tool execution rejected by user", "tool_name", toolCall.Function.Name)
			continue
		}

		// Execute the tool
		logger.Info("Executing approved tool", "tool_name", toolCall.Function.Name)

		// Convert to parser format for executor
		parserCall := parser.ToolCall{
			ID:   toolCall.ID,
			Type: toolCall.Type,
			Function: parser.FunctionCall{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		}

		result, err := toolExecutor.Execute(ctx, parserCall)
		if err != nil {
			logger.WithError(err).Error("Tool execution failed", "tool_name", toolCall.Function.Name)

			// Update tool execution status
			session.toolsMu.Lock()
			toolExec.Status = pb.ToolExecution_FAILED
			toolExec.Error = err.Error()
			toolExec.UpdatedAt = time.Now().Unix()
			session.toolsMu.Unlock()

			continue
		}

		// Update tool execution with result
		session.toolsMu.Lock()
		toolExec.Status = pb.ToolExecution_COMPLETED
		toolExec.Result = result.Content
		toolExec.UpdatedAt = time.Now().Unix()
		session.toolsMu.Unlock()

		// Send tool result
		resultResp := &pb.ChatResponse{
			Response: &pb.ChatResponse_ToolExecution{
				ToolExecution: toolExec,
			},
		}
		stream.Send(resultResp)

		// Execute agent again with tool result
		continuationResponse, err := toolAgent.ContinueWithToolResult(ctx, toolCall.ID, result.Content)
		if err != nil {
			logger.WithError(err).Error("Failed to continue after tool execution")
			continue
		}

		// Append continuation to response
		response += "\n\n" + continuationResponse
	}

	return response, nil
}

// waitForToolApproval waits for user approval of a tool execution
func (s *ChatService) waitForToolApproval(ctx context.Context, session *ChatSession, toolExecID string, timeout time.Duration) (bool, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-timer.C:
			return false, gerror.New(gerror.ErrCodeTimeout, "tool approval timeout", nil).
				WithComponent("grpc").
				WithOperation("waitForToolApproval")
		case <-ticker.C:
			session.toolsMu.RLock()
			toolExec, exists := session.toolExecutions[toolExecID]
			if !exists {
				session.toolsMu.RUnlock()
				return false, gerror.New(gerror.ErrCodeNotFound, "tool execution not found", nil)
			}

			status := toolExec.Status
			session.toolsMu.RUnlock()

			switch status {
			case pb.ToolExecution_EXECUTING, pb.ToolExecution_COMPLETED:
				return true, nil
			case pb.ToolExecution_CANCELLED, pb.ToolExecution_FAILED:
				return false, nil
			case pb.ToolExecution_AWAITING_APPROVAL:
				// Keep waiting
				continue
			}
		}
	}
}

// convertParserToolsToProviderFormat converts parser tool definitions to provider format
func convertParserToolsToProviderFormat(tools []parser.ToolDefinition) []interfaces.ToolDefinition {
	result := make([]interfaces.ToolDefinition, len(tools))
	for i, tool := range tools {
		result[i] = interfaces.ToolDefinition{
			Type: tool.Type,
			Function: interfaces.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}
	return result
}
