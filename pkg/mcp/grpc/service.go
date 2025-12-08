// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package grpc provides gRPC service implementation for MCP
package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/guild-framework/guild-core/pkg/grpc/pb/mcp/v1"
	"github.com/guild-framework/guild-core/pkg/mcp/protocol"
	"github.com/guild-framework/guild-core/pkg/mcp/server"
	"github.com/guild-framework/guild-core/pkg/mcp/tools"
)

// MCPService implements the gRPC service for MCP
type MCPService struct {
	server *server.Server
	pb.UnimplementedMCPServiceServer
}

// NewMCPService creates a new gRPC service
func NewMCPService(mcpServer *server.Server) *MCPService {
	return &MCPService{
		server: mcpServer,
	}
}

// RegisterTool registers a tool via gRPC
func (s *MCPService) RegisterTool(ctx context.Context, req *pb.ToolRegistrationRequest) (*pb.ToolRegistrationResponse, error) {
	if req.Tool == nil {
		return nil, status.Error(codes.InvalidArgument, "tool definition required")
	}

	// Convert gRPC request to protocol request
	toolDef := &protocol.ToolDefinition{
		ID:           req.Tool.Id,
		Name:         req.Tool.Name,
		Description:  req.Tool.Description,
		Capabilities: req.Tool.Capabilities,
		Parameters:   convertParameters(req.Tool.Parameters),
		Returns:      convertParameters(req.Tool.Returns),
		CostProfile:  convertCostProfile(req.Tool.CostProfile),
	}

	// Create MCP message
	payload, err := json.Marshal(&protocol.ToolRegistrationRequest{
		Tool: *toolDef,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to marshal request")
	}

	// TODO: In a real implementation, create an MCP message and send through server
	// For now, directly register with the tool registry
	_ = payload // Mark as used
	tool := tools.NewBaseTool(
		toolDef.ID,
		toolDef.Name,
		toolDef.Description,
		toolDef.Capabilities,
		toolDef.CostProfile,
		toolDef.Parameters,
		toolDef.Returns,
		nil, // Executor would be set up separately
	)

	// For this implementation, we'll directly interact with the tool registry
	// In production, this should go through the MCP server's message handling
	if err := s.server.GetToolRegistry().RegisterTool(tool); err != nil {
		return nil, status.Error(codes.AlreadyExists, err.Error())
	}

	return &pb.ToolRegistrationResponse{
		Success: true,
		ToolId:  toolDef.ID,
	}, nil
}

// DeregisterTool removes a tool via gRPC
func (s *MCPService) DeregisterTool(ctx context.Context, req *pb.ToolDeregistrationRequest) (*pb.ToolDeregistrationResponse, error) {
	if req.ToolId == "" {
		return nil, status.Error(codes.InvalidArgument, "tool ID required")
	}

	if err := s.server.GetToolRegistry().DeregisterTool(req.ToolId); err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &pb.ToolDeregistrationResponse{
		Success: true,
	}, nil
}

// DiscoverTools discovers available tools via gRPC
func (s *MCPService) DiscoverTools(ctx context.Context, req *pb.ToolDiscoveryRequest) (*pb.ToolDiscoveryResponse, error) {
	// Convert gRPC request to protocol query
	query := protocol.ToolQuery{
		RequiredCapabilities: req.RequiredCapabilities,
		MaxCost:              req.MaxCost,
		MaxLatency:           time.Duration(req.MaxLatencyMs) * time.Millisecond,
	}
	// Note: Limit is not part of ToolQuery, would need to be handled at registry level

	tools, err := s.server.GetToolRegistry().DiscoverTools(query)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert tools to gRPC response
	var grpcTools []*pb.ToolDefinition
	for _, tool := range tools {
		grpcTools = append(grpcTools, &pb.ToolDefinition{
			Id:           tool.ID(),
			Name:         tool.Name(),
			Description:  tool.Description(),
			Capabilities: tool.Capabilities(),
			Parameters:   convertParametersToGRPC(tool.GetParameters()),
			Returns:      convertParametersToGRPC(tool.GetReturns()),
			CostProfile:  convertCostProfileToGRPC(tool.GetCostProfile()),
		})
	}

	return &pb.ToolDiscoveryResponse{
		Tools: grpcTools,
	}, nil
}

// ExecuteTool executes a tool via gRPC
func (s *MCPService) ExecuteTool(ctx context.Context, req *pb.ToolExecutionRequest) (*pb.ToolExecutionResponse, error) {
	if req.ToolId == "" {
		return nil, status.Error(codes.InvalidArgument, "tool ID required")
	}

	// Get tool
	tool, err := s.server.GetToolRegistry().GetTool(req.ToolId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	// Convert parameters
	params := make(map[string]interface{})
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid parameters")
	}

	// Execute tool
	startTime := time.Now()
	result, err := tool.Execute(ctx, params)
	endTime := time.Now()

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Marshal result
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to marshal result")
	}

	return &pb.ToolExecutionResponse{
		ExecutionId: req.ExecutionId,
		Result:      resultBytes,
		StartTime:   startTime.Unix(),
		EndTime:     endTime.Unix(),
	}, nil
}

// CheckToolHealth checks tool health via gRPC
func (s *MCPService) CheckToolHealth(ctx context.Context, req *pb.ToolHealthRequest) (*pb.ToolHealthResponse, error) {
	if req.ToolId == "" {
		return nil, status.Error(codes.InvalidArgument, "tool ID required")
	}

	tool, err := s.server.GetToolRegistry().GetTool(req.ToolId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	healthy := tool.HealthCheck() == nil

	return &pb.ToolHealthResponse{
		ToolId:  req.ToolId,
		Healthy: healthy,
	}, nil
}

// ReportCost reports cost via gRPC
func (s *MCPService) ReportCost(ctx context.Context, req *pb.CostReportRequest) (*pb.CostReportResponse, error) {
	if req.Cost == nil {
		return nil, status.Error(codes.InvalidArgument, "cost report required")
	}

	// Convert gRPC cost to protocol cost
	cost := protocol.CostReport{
		OperationID:   req.Cost.OperationId,
		StartTime:     time.Unix(req.Cost.StartTime, 0),
		EndTime:       time.Unix(req.Cost.EndTime, 0),
		ComputeCost:   req.Cost.ComputeCost,
		MemoryCost:    req.Cost.MemoryCost,
		LatencyCost:   time.Duration(req.Cost.LatencyMs) * time.Millisecond,
		TokensCost:    int(req.Cost.TokensCost),
		APICallsCost:  int(req.Cost.ApiCallsCost),
		FinancialCost: req.Cost.FinancialCost,
	}

	s.server.GetCostObserver().RecordCost(ctx, cost.OperationID, cost)

	return &pb.CostReportResponse{
		Success: true,
	}, nil
}

// QueryCosts queries cost analysis via gRPC
func (s *MCPService) QueryCosts(ctx context.Context, req *pb.CostQueryRequest) (*pb.CostQueryResponse, error) {
	// Convert gRPC query to protocol query
	query := protocol.CostQuery{
		OperationIDs: req.OperationIds,
		StartTime:    time.Unix(req.StartTime, 0),
		EndTime:      time.Unix(req.EndTime, 0),
		GroupBy:      req.GroupBy,
		Limit:        int(req.Limit),
	}

	analysis, err := s.server.GetCostObserver().Analyze(ctx, query)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert analysis to gRPC response
	var breakdown []*pb.CostReport
	for _, cost := range analysis.Breakdown {
		breakdown = append(breakdown, &pb.CostReport{
			OperationId:   cost.OperationID,
			StartTime:     cost.StartTime.Unix(),
			EndTime:       cost.EndTime.Unix(),
			ComputeCost:   cost.ComputeCost,
			MemoryCost:    cost.MemoryCost,
			LatencyMs:     cost.LatencyCost.Milliseconds(),
			TokensCost:    int64(cost.TokensCost),
			ApiCallsCost:  int64(cost.APICallsCost),
			FinancialCost: cost.FinancialCost,
		})
	}

	return &pb.CostQueryResponse{
		TotalCost: &pb.CostReport{
			OperationId:   analysis.TotalCost.OperationID,
			StartTime:     analysis.TotalCost.StartTime.Unix(),
			EndTime:       analysis.TotalCost.EndTime.Unix(),
			ComputeCost:   analysis.TotalCost.ComputeCost,
			MemoryCost:    analysis.TotalCost.MemoryCost,
			LatencyMs:     analysis.TotalCost.LatencyCost.Milliseconds(),
			TokensCost:    int64(analysis.TotalCost.TokensCost),
			ApiCallsCost:  int64(analysis.TotalCost.APICallsCost),
			FinancialCost: analysis.TotalCost.FinancialCost,
		},
		BreakdownBy:     analysis.BreakdownBy,
		Breakdown:       breakdown,
		Recommendations: analysis.Recommendations,
	}, nil
}

// Ping sends a ping via gRPC
func (s *MCPService) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{
		Timestamp: time.Now().Unix(),
		ServerId:  s.server.GetConfig().ServerID,
	}, nil
}

// GetSystemInfo gets system information via gRPC
func (s *MCPService) GetSystemInfo(ctx context.Context, req *pb.SystemInfoRequest) (*pb.SystemInfoResponse, error) {
	config := s.server.GetConfig()

	return &pb.SystemInfoResponse{
		ServerId:   config.ServerID,
		ServerName: config.ServerName,
		Version:    config.Version,
		Features: &pb.SystemFeatures{
			Tls:          config.EnableTLS,
			Auth:         config.EnableAuth,
			Metrics:      config.EnableMetrics,
			Tracing:      config.EnableTracing,
			CostTracking: config.EnableCostTracking,
		},
		ToolsCount: int64(len(s.server.GetToolRegistry().ListTools())),
	}, nil
}

// Helper functions

func generateID() string {
	return fmt.Sprintf("grpc-%d", time.Now().UnixNano())
}

func convertParameters(params []*pb.ToolParameter) []protocol.ToolParameter {
	var result []protocol.ToolParameter
	for _, param := range params {
		result = append(result, protocol.ToolParameter{
			Name:        param.Name,
			Type:        param.Type,
			Description: param.Description,
			Required:    param.Required,
			Default:     param.Default,
		})
	}
	return result
}

func convertParametersToGRPC(params []protocol.ToolParameter) []*pb.ToolParameter {
	var result []*pb.ToolParameter
	for _, param := range params {
		// Convert default value to string
		var defaultStr string
		if param.Default != nil {
			// Marshal to JSON string for complex types
			if bytes, err := json.Marshal(param.Default); err == nil {
				defaultStr = string(bytes)
			}
		}

		result = append(result, &pb.ToolParameter{
			Name:        param.Name,
			Type:        param.Type,
			Description: param.Description,
			Required:    param.Required,
			Default:     defaultStr,
		})
	}
	return result
}

func convertCostProfile(profile *pb.CostProfile) protocol.CostProfile {
	if profile == nil {
		return protocol.CostProfile{}
	}
	return protocol.CostProfile{
		ComputeCost:   profile.ComputeCost,
		MemoryCost:    profile.MemoryCost,
		LatencyCost:   time.Duration(profile.LatencyMs) * time.Millisecond,
		FinancialCost: profile.FinancialCost,
	}
}

func convertCostProfileToGRPC(profile protocol.CostProfile) *pb.CostProfile {
	return &pb.CostProfile{
		ComputeCost:   profile.ComputeCost,
		MemoryCost:    profile.MemoryCost,
		LatencyMs:     profile.LatencyCost.Milliseconds(),
		TokensCost:    0, // Not in protocol.CostProfile
		ApiCallsCost:  0, // Not in protocol.CostProfile
		FinancialCost: profile.FinancialCost,
	}
}
