// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lancekrogers/guild/pkg/gerror"
	pb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild/pkg/registry"
	"go.uber.org/zap"
)

// PerformanceOptimizationServiceServer implements performance optimization gRPC services
// This extends the existing SessionService with performance optimization capabilities
type PerformanceOptimizationServiceServer struct {
	pb.UnimplementedSessionServiceServer

	registry        *registry.PerformanceOptimizationRegistry
	logger          *zap.Logger

	// Service health tracking with proper observability
	healthy         bool
	lastHealth      time.Time
	healthCheck     context.CancelFunc
	shutdownChan    chan struct{}
	mu              sync.RWMutex
}

// NewPerformanceOptimizationServiceServer creates a new performance optimization service server with proper lifecycle management
func NewPerformanceOptimizationServiceServer(perfOptRegistry *registry.PerformanceOptimizationRegistry, logger *zap.Logger) *PerformanceOptimizationServiceServer {
	if perfOptRegistry == nil {
		panic("PerformanceOptimizationRegistry cannot be nil")
	}
	if logger == nil {
		panic("logger cannot be nil")
	}

	// Create cancellable context for health monitoring
	healthCtx, healthCancel := context.WithCancel(context.Background())

	s := &PerformanceOptimizationServiceServer{
		registry:     perfOptRegistry,
		logger:       logger.Named("performance-optimization-grpc-service"),
		healthy:      true,
		lastHealth:   time.Now(),
		healthCheck:  healthCancel,
		shutdownChan: make(chan struct{}),
	}

	// Start production-grade health monitoring with proper lifecycle
	go s.healthMonitor(healthCtx)

	return s
}

// healthMonitor provides production-grade health monitoring with proper lifecycle management
func (s *PerformanceOptimizationServiceServer) healthMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	logger := s.logger.With(zap.String("component", "health-monitor"))
	logger.Info("Health monitoring started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Health monitoring shutting down", zap.Error(ctx.Err()))
			close(s.shutdownChan)
			return

		case <-ticker.C:
			// Create a timeout context for the health check
			healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			
			// Perform health check with proper error handling
			startTime := time.Now()
			err := s.registry.HealthCheck(healthCtx)
			duration := time.Since(startTime)
			cancel()

			s.mu.Lock()
			s.healthy = err == nil
			s.lastHealth = time.Now()
			s.mu.Unlock()

			if err != nil {
				logger.Warn("Health check failed",
					zap.Error(err),
					zap.Duration("check_duration", duration),
					zap.Time("check_time", s.lastHealth))
			} else {
				logger.Debug("Health check passed",
					zap.Duration("check_duration", duration),
					zap.Time("check_time", s.lastHealth))
			}
		}
	}
}

// CreateSession implements session creation with comprehensive validation and error handling
func (s *PerformanceOptimizationServiceServer) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.Session, error) {
	// Comprehensive input validation
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}

	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name is required")
	}

	// Check for context cancellation early
	if err := ctx.Err(); err != nil {
		return nil, status.Errorf(codes.Canceled, "request cancelled: %v", err)
	}

	// Create context-aware logger with request correlation
	campaignID := ""
	if req.CampaignId != nil {
		campaignID = *req.CampaignId
	}
	logger := s.logger.With(
		zap.String("session_name", req.Name),
		zap.String("campaign_id", campaignID),
		zap.String("operation", "CreateSession"),
		zap.String("grpc_method", "CreateSession"),
		zap.Time("request_start", time.Now()),
	)

	logger.Info("Processing session creation request")

	// Check service health before processing
	if !s.GetHealth() {
		logger.Error("Service unhealthy, rejecting request")
		return nil, status.Errorf(codes.Unavailable, "service is currently unhealthy")
	}

	// Get session manager with validation
	sessionManager := s.registry.GetSessionManager()
	if sessionManager == nil {
		logger.Error("Session manager not available")
		return nil, status.Errorf(codes.Internal, "session manager not configured")
	}

	// Create session with proper error handling - use name as userID for now
	startTime := time.Now()
	session, err := sessionManager.CreateSession(ctx, req.Name, campaignID)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("Failed to create session",
			zap.Error(err),
			zap.Duration("operation_duration", duration),
			zap.String("failure_stage", "session_creation"))
		return nil, convertGuildErrorToGRPCStatus(err)
	}

	// Final context check before returning
	if err := ctx.Err(); err != nil {
		logger.Warn("Request cancelled after session creation",
			zap.Error(err),
			zap.String("session_id", session.ID))
		return nil, status.Errorf(codes.Canceled, "request cancelled during response preparation")
	}

	logger.Info("Session created successfully",
		zap.String("session_id", session.ID),
		zap.Duration("operation_duration", duration))

	return &pb.Session{
		Id: session.ID,
		Name: req.Name,
		CampaignId: req.CampaignId,
		CreatedAt: timestamppb.New(session.CreatedAt),
		UpdatedAt: timestamppb.New(session.UpdatedAt),
		Metadata: convertSessionMetadataToProto(session.Metadata),
	}, nil
}

// GetSession loads and returns a session
func (s *PerformanceOptimizationServiceServer) GetSession(ctx context.Context, req *pb.GetSessionRequest) (*pb.Session, error) {
	// Comprehensive input validation
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}

	if req.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "session id is required")
	}

	// Check for context cancellation early
	if err := ctx.Err(); err != nil {
		return nil, status.Errorf(codes.Canceled, "request cancelled: %v", err)
	}

	// Create context-aware logger with request correlation
	logger := s.logger.With(
		zap.String("session_id", req.Id),
		zap.String("operation", "GetSession"),
		zap.String("grpc_method", "GetSession"),
		zap.Time("request_start", time.Now()),
	)

	logger.Info("Processing session get request")

	// Check service health before processing
	if !s.GetHealth() {
		logger.Error("Service unhealthy, rejecting request")
		return nil, status.Errorf(codes.Unavailable, "service is currently unhealthy")
	}

	// Get session manager with validation
	sessionManager := s.registry.GetSessionManager()
	if sessionManager == nil {
		logger.Error("Session manager not available")
		return nil, status.Errorf(codes.Internal, "session manager not configured")
	}

	// Load session with proper error handling
	startTime := time.Now()
	session, err := sessionManager.LoadSession(ctx, req.Id)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("Failed to load session",
			zap.Error(err),
			zap.Duration("operation_duration", duration),
			zap.String("failure_stage", "session_load"))
		return nil, convertGuildErrorToGRPCStatus(err)
	}

	// Final context check before returning
	if err := ctx.Err(); err != nil {
		logger.Warn("Request cancelled after session load",
			zap.Error(err),
			zap.String("session_id", session.ID))
		return nil, status.Errorf(codes.Canceled, "request cancelled during response preparation")
	}

	logger.Info("Session loaded successfully",
		zap.String("session_id", session.ID),
		zap.Duration("operation_duration", duration))

	return &pb.Session{
		Id: session.ID,
		Name: session.ID, // Use ID as name for now
		CreatedAt: timestamppb.New(session.CreatedAt),
		UpdatedAt: timestamppb.New(session.UpdatedAt),
		Metadata: convertSessionMetadataToProto(session.Metadata),
	}, nil
}

// Shutdown gracefully shuts down the service server
func (s *PerformanceOptimizationServiceServer) Shutdown(ctx context.Context) error {
	logger := s.logger.With(zap.String("operation", "Shutdown"))
	logger.Info("Starting graceful shutdown")

	// Cancel health monitoring
	if s.healthCheck != nil {
		s.healthCheck()
	}

	// Wait for health monitor to finish or timeout
	select {
	case <-s.shutdownChan:
		logger.Info("Health monitor shut down successfully")
	case <-time.After(5 * time.Second):
		logger.Warn("Health monitor shutdown timed out")
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "shutdown cancelled").
			FromContext(ctx).
			WithComponent("performance-optimization-grpc-service").
			WithOperation("Shutdown")
	}

	s.mu.Lock()
	s.healthy = false
	s.mu.Unlock()

	logger.Info("Service server shutdown completed")
	return nil
}

// Helper functions
func convertGuildErrorToGRPCStatus(err error) error {
	if gErr, ok := err.(*gerror.GuildError); ok {
		switch gErr.Code {
		case gerror.ErrCodeNotFound:
			return status.Error(codes.NotFound, gErr.Message)
		case gerror.ErrCodeValidation, gerror.ErrCodeInvalidInput:
			return status.Error(codes.InvalidArgument, gErr.Message)
		case gerror.ErrCodeTimeout:
			return status.Error(codes.DeadlineExceeded, gErr.Message)
		case gerror.ErrCodePermissionDenied:
			return status.Error(codes.PermissionDenied, gErr.Message)
		case gerror.ErrCodeAlreadyExists:
			return status.Error(codes.AlreadyExists, gErr.Message)
		case gerror.ErrCodeNotImplemented:
			return status.Error(codes.Unimplemented, gErr.Message)
		default:
			return status.Error(codes.Internal, gErr.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}

func convertSessionMetadataToProto(metadata map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range metadata {
		if str, ok := value.(string); ok {
			result[key] = str
		}
	}
	return result
}

// GetHealth provides a health check endpoint
func (s *PerformanceOptimizationServiceServer) GetHealth() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthy
}

// GetLastHealthCheck returns the last health check time
func (s *PerformanceOptimizationServiceServer) GetLastHealthCheck() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastHealth
}