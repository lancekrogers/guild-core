// Package grpc provides gRPC service implementation for MCP
package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
	"github.com/guild-ventures/guild-core/pkg/mcp/server"
	"github.com/guild-ventures/guild-core/pkg/mcp/tools"
)

// MCPService implements the gRPC service for MCP
type MCPService struct {
	server *server.Server
	UnimplementedMCPServiceServer
}

// NewMCPService creates a new gRPC service
func NewMCPService(mcpServer *server.Server) *MCPService {
	return &MCPService{
		server: mcpServer,
	}
}

// RegisterTool registers a tool via gRPC
func (s *MCPService) RegisterTool(ctx context.Context, req *ToolRegistrationRequest) (*ToolRegistrationResponse, error) {
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

	msg := &protocol.MCPMessage{
		ID:          generateID(),
		Version:     "1.0",
		MessageType: protocol.RequestMessage,
		Method:      "tool.register",
		Timestamp:   time.Now(),
		Payload:     json.RawMessage(payload),
	}

	// Handle via MCP server (this is a simplified approach)
	// In a real implementation, you'd have a proper bridge
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

	return &ToolRegistrationResponse{
		Success: true,
		ToolId:  toolDef.ID,
	}, nil
}

// DeregisterTool removes a tool via gRPC
func (s *MCPService) DeregisterTool(ctx context.Context, req *ToolDeregistrationRequest) (*ToolDeregistrationResponse, error) {
	if req.ToolId == "" {
		return nil, status.Error(codes.InvalidArgument, "tool ID required")
	}

	if err := s.server.GetToolRegistry().DeregisterTool(req.ToolId); err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &ToolDeregistrationResponse{
		Success: true,
	}, nil
}

// DiscoverTools discovers available tools via gRPC
func (s *MCPService) DiscoverTools(ctx context.Context, req *ToolDiscoveryRequest) (*ToolDiscoveryResponse, error) {
	// Convert gRPC request to protocol query
	query := protocol.ToolQuery{
		RequiredCapabilities: req.RequiredCapabilities,
		MaxCost:              req.MaxCost,
		MaxLatency:           time.Duration(req.MaxLatencyMs) * time.Millisecond,
		Limit:                int(req.Limit),
	}

	tools, err := s.server.GetToolRegistry().DiscoverTools(query)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert tools to gRPC response
	var grpcTools []*ToolDefinition
	for _, tool := range tools {
		grpcTools = append(grpcTools, &ToolDefinition{
			Id:           tool.ID(),
			Name:         tool.Name(),
			Description:  tool.Description(),
			Capabilities: tool.Capabilities(),
			Parameters:   convertParametersToGRPC(tool.GetParameters()),
			Returns:      convertParametersToGRPC(tool.GetReturns()),
			CostProfile:  convertCostProfileToGRPC(tool.GetCostProfile()),
		})
	}

	return &ToolDiscoveryResponse{
		Tools: grpcTools,
	}, nil
}

// ExecuteTool executes a tool via gRPC
func (s *MCPService) ExecuteTool(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error) {
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

	return &ToolExecutionResponse{
		ExecutionId: req.ExecutionId,
		Result:      resultBytes,
		StartTime:   startTime.Unix(),
		EndTime:     endTime.Unix(),
	}, nil
}

// CheckToolHealth checks tool health via gRPC
func (s *MCPService) CheckToolHealth(ctx context.Context, req *ToolHealthRequest) (*ToolHealthResponse, error) {
	if req.ToolId == "" {
		return nil, status.Error(codes.InvalidArgument, "tool ID required")
	}

	tool, err := s.server.GetToolRegistry().GetTool(req.ToolId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	healthy := tool.HealthCheck() == nil

	return &ToolHealthResponse{
		ToolId:  req.ToolId,
		Healthy: healthy,
	}, nil
}

// ReportCost reports cost via gRPC
func (s *MCPService) ReportCost(ctx context.Context, req *CostReportRequest) (*CostReportResponse, error) {
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

	return &CostReportResponse{
		Success: true,
	}, nil
}

// QueryCosts queries cost analysis via gRPC
func (s *MCPService) QueryCosts(ctx context.Context, req *CostQueryRequest) (*CostQueryResponse, error) {
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
	var breakdown []*CostReport
	for _, cost := range analysis.Breakdown {
		breakdown = append(breakdown, &CostReport{
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

	return &CostQueryResponse{
		TotalCost: &CostReport{
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
func (s *MCPService) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	return &PingResponse{
		Timestamp: time.Now().Unix(),
		ServerId:  s.server.GetConfig().ServerID,
	}, nil
}

// GetSystemInfo gets system information via gRPC
func (s *MCPService) GetSystemInfo(ctx context.Context, req *SystemInfoRequest) (*SystemInfoResponse, error) {
	config := s.server.GetConfig()
	
	return &SystemInfoResponse{
		ServerId:   config.ServerID,
		ServerName: config.ServerName,
		Version:    config.Version,
		Features: &SystemFeatures{
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

func convertParameters(params []*ToolParameter) []protocol.ToolParameter {
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

func convertParametersToGRPC(params []protocol.ToolParameter) []*ToolParameter {
	var result []*ToolParameter
	for _, param := range params {
		result = append(result, &ToolParameter{
			Name:        param.Name,
			Type:        param.Type,
			Description: param.Description,
			Required:    param.Required,
			Default:     param.Default,
		})
	}
	return result
}

func convertCostProfile(profile *CostProfile) protocol.CostProfile {
	if profile == nil {
		return protocol.CostProfile{}
	}
	return protocol.CostProfile{
		ComputeCost:   profile.ComputeCost,
		MemoryCost:    profile.MemoryCost,
		LatencyCost:   time.Duration(profile.LatencyMs) * time.Millisecond,
		TokensCost:    int(profile.TokensCost),
		APICallsCost:  int(profile.ApiCallsCost),
		FinancialCost: profile.FinancialCost,
	}
}

func convertCostProfileToGRPC(profile protocol.CostProfile) *CostProfile {
	return &CostProfile{
		ComputeCost:   profile.ComputeCost,
		MemoryCost:    profile.MemoryCost,
		LatencyMs:     profile.LatencyCost.Milliseconds(),
		TokensCost:    int64(profile.TokensCost),
		ApiCallsCost:  int64(profile.APICallsCost),
		FinancialCost: profile.FinancialCost,
	}
}