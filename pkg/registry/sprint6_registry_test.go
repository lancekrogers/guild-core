// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"context"
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// Test suite for PerformanceOptimizationRegistry with comprehensive coverage
func TestPerformanceOptimizationRegistry_HealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		setupRegistry  func(*PerformanceOptimizationRegistry)
		contextTimeout time.Duration
		wantErr        bool
		expectedCode   gerror.ErrorCode
		description    string
	}{
		{
			name: "healthy_registry_with_all_required_components",
			setupRegistry: func(r *PerformanceOptimizationRegistry) {
				// Register all required components
				r.RegisterSessionManager(&mockSessionManager{})
				r.RegisterSessionResumer(&mockSessionResumer{})
				r.RegisterSessionAnalytics(&mockSessionAnalytics{})
				r.RegisterPerformanceProfiler(&mockPerformanceProfiler{})
				r.RegisterCacheManager(&mockCacheManager{})
				r.RegisterMemoryOptimizer(&mockMemoryOptimizer{})
				r.RegisterPerformanceMonitor(&mockPerformanceMonitor{})
				r.RegisterAlertManager(&mockAlertManager{})
			},
			contextTimeout: 5 * time.Second,
			wantErr:        false,
			description:    "Should pass when all required components are registered",
		},
		{
			name: "unhealthy_registry_missing_required_components",
			setupRegistry: func(r *PerformanceOptimizationRegistry) {
				// Only register some components, leaving required ones missing
				r.RegisterSessionManager(&mockSessionManager{})
			},
			contextTimeout: 5 * time.Second,
			wantErr:        true,
			expectedCode:   gerror.ErrCodeConfiguration,
			description:    "Should fail when required components are missing",
		},
		{
			name: "context_cancellation_during_health_check",
			setupRegistry: func(r *PerformanceOptimizationRegistry) {
				r.RegisterSessionManager(&mockSessionManager{})
			},
			contextTimeout: 1 * time.Nanosecond, // Immediate cancellation
			wantErr:        true,
			expectedCode:   gerror.ErrCodeCancelled,
			description:    "Should handle context cancellation gracefully",
		},
		{
			name: "healthy_registry_with_optional_components",
			setupRegistry: func(r *PerformanceOptimizationRegistry) {
				// Register required components
				r.RegisterSessionManager(&mockSessionManager{})
				r.RegisterSessionResumer(&mockSessionResumer{})
				r.RegisterSessionAnalytics(&mockSessionAnalytics{})
				r.RegisterPerformanceProfiler(&mockPerformanceProfiler{})
				r.RegisterCacheManager(&mockCacheManager{})
				r.RegisterMemoryOptimizer(&mockMemoryOptimizer{})
				r.RegisterPerformanceMonitor(&mockPerformanceMonitor{})
				r.RegisterAlertManager(&mockAlertManager{})
				
				// Also register optional components
				r.RegisterSessionExporter(&mockSessionExporter{})
				r.RegisterSLOMonitor(&mockSLOMonitor{})
				r.RegisterDashboardRenderer(&mockDashboardRenderer{})
			},
			contextTimeout: 5 * time.Second,
			wantErr:        false,
			description:    "Should pass when both required and optional components are registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test logger that captures logs for verification
			logger := zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))
			
			// Create mock event bus and component registry
			eventBus := &mockEventBus{}
			componentRegistry := &mockComponentRegistry{}
			
			// Create registry under test
			registry := NewPerformanceOptimizationRegistry(eventBus, componentRegistry, logger)
			
			// Setup registry according to test case
			if tt.setupRegistry != nil {
				tt.setupRegistry(registry)
			}

			// Create context with timeout for the test
			ctx, cancel := context.WithTimeout(context.Background(), tt.contextTimeout)
			defer cancel()

			// Execute health check
			err := registry.HealthCheck(ctx)

			// Verify results
			if tt.wantErr {
				if err == nil {
					t.Errorf("HealthCheck() expected error but got none. Test: %s", tt.description)
					return
				}

				// Verify error type if specified
				if tt.expectedCode != "" {
					if !gerror.Is(err, tt.expectedCode) {
						t.Errorf("HealthCheck() expected error code %v, got %v. Test: %s", 
							tt.expectedCode, gerror.GetCode(err), tt.description)
					}
				}

				// Verify error has proper context information for observability
				var guildErr *gerror.GuildError
				if gerror.As(err, &guildErr) {
					if guildErr.Component == "" {
						t.Errorf("HealthCheck() error missing component information for observability")
					}
					if guildErr.Operation == "" {
						t.Errorf("HealthCheck() error missing operation information for observability")
					}
				}
			} else {
				if err != nil {
					t.Errorf("HealthCheck() unexpected error: %v. Test: %s", err, tt.description)
				}
			}
		})
	}
}

func TestPerformanceOptimizationRegistry_InitializeComponents(t *testing.T) {
	tests := []struct {
		name           string
		config         *PerformanceOptimizationConfig
		contextTimeout time.Duration
		wantErr        bool
		expectedCode   gerror.ErrorCode
		description    string
	}{
		{
			name:           "nil_config_should_fail",
			config:         nil,
			contextTimeout: 5 * time.Second,
			wantErr:        true,
			expectedCode:   gerror.ErrCodeInvalidInput,
			description:    "Should fail with proper error when config is nil",
		},
		{
			name: "valid_config_should_succeed",
			config: &PerformanceOptimizationConfig{
				Session: SessionConfig{
					Database: DatabaseConfig{Path: "/tmp/test.db"},
					EventBus: EventBusConfig{BufferSize: 100},
				},
				Performance: PerformanceConfig{
					Profiling: ProfilingConfig{Enabled: true},
					Caching:   CachingConfig{Enabled: true},
				},
				Monitoring: MonitoringConfig{
					Metrics:  MetricsConfig{Enabled: true},
					Alerting: AlertingConfig{Enabled: true},
				},
			},
			contextTimeout: 5 * time.Second,
			wantErr:        false,
			description:    "Should succeed with valid configuration",
		},
		{
			name: "context_cancellation_should_cleanup",
			config: &PerformanceOptimizationConfig{
				Session:     SessionConfig{},
				Performance: PerformanceConfig{},
				Monitoring:  MonitoringConfig{},
			},
			contextTimeout: 1 * time.Nanosecond, // Immediate cancellation
			wantErr:        true,
			expectedCode:   gerror.ErrCodeCancelled,
			description:    "Should handle context cancellation with proper cleanup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			eventBus := &mockEventBus{}
			componentRegistry := &mockComponentRegistry{}
			
			registry := NewPerformanceOptimizationRegistry(eventBus, componentRegistry, logger)

			ctx, cancel := context.WithTimeout(context.Background(), tt.contextTimeout)
			defer cancel()

			err := registry.InitializeComponents(ctx, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("InitializeComponents() expected error but got none. Test: %s", tt.description)
					return
				}

				if tt.expectedCode != "" && !gerror.Is(err, tt.expectedCode) {
					t.Errorf("InitializeComponents() expected error code %v, got %v", 
						tt.expectedCode, gerror.GetCode(err))
				}
			} else {
				if err != nil {
					t.Errorf("InitializeComponents() unexpected error: %v. Test: %s", err, tt.description)
				}

				// Verify registry is marked as initialized on success
				if !registry.initialized {
					t.Errorf("InitializeComponents() should mark registry as initialized on success")
				}
			}
		})
	}
}

// Comprehensive benchmark to ensure performance meets enterprise standards
func BenchmarkPerformanceOptimizationRegistry_HealthCheck(b *testing.B) {
	logger := zap.NewNop() // No-op logger for benchmarks
	eventBus := &mockEventBus{}
	componentRegistry := &mockComponentRegistry{}
	
	registry := NewPerformanceOptimizationRegistry(eventBus, componentRegistry, logger)
	
	// Setup with all components for worst-case scenario
	registry.RegisterSessionManager(&mockSessionManager{})
	registry.RegisterSessionResumer(&mockSessionResumer{})
	registry.RegisterSessionAnalytics(&mockSessionAnalytics{})
	registry.RegisterPerformanceProfiler(&mockPerformanceProfiler{})
	registry.RegisterCacheManager(&mockCacheManager{})
	registry.RegisterMemoryOptimizer(&mockMemoryOptimizer{})
	registry.RegisterPerformanceMonitor(&mockPerformanceMonitor{})
	registry.RegisterAlertManager(&mockAlertManager{})

	ctx := context.Background()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := registry.HealthCheck(ctx)
		if err != nil {
			b.Fatalf("HealthCheck failed: %v", err)
		}
	}
}

// Mock implementations for testing
type mockEventBus struct{}

func (m *mockEventBus) Publish(event interface{})                              {}
func (m *mockEventBus) Subscribe(eventType string, handler interface{}) error { return nil }
func (m *mockEventBus) Unsubscribe(eventType string, handler interface{}) error { return nil }

type mockComponentRegistry struct{}

func (m *mockComponentRegistry) Agents() AgentRegistry                                    { return nil }
func (m *mockComponentRegistry) Tools() ToolRegistry                                     { return nil }
func (m *mockComponentRegistry) Providers() ProviderRegistry                             { return nil }
func (m *mockComponentRegistry) Memory() MemoryRegistry                                  { return nil }
func (m *mockComponentRegistry) Project() ProjectRegistry                                { return nil }
func (m *mockComponentRegistry) Prompts() *PromptRegistry                                { return nil }
func (m *mockComponentRegistry) GetPromptManager() (LayeredPromptManager, error)         { return nil, nil }
func (m *mockComponentRegistry) Storage() StorageRegistry                                { return nil }
func (m *mockComponentRegistry) Orchestrator() interface{}                               { return nil }
func (m *mockComponentRegistry) Initialize(ctx context.Context, config Config) error    { return nil }
func (m *mockComponentRegistry) Shutdown(ctx context.Context) error                      { return nil }
func (m *mockComponentRegistry) GetAgentsByCost(maxCost int) []AgentInfo                 { return []AgentInfo{} }
func (m *mockComponentRegistry) GetCheapestAgentByCapability(capability string) (*AgentInfo, error) { return nil, nil }
func (m *mockComponentRegistry) GetToolsByCost(maxCost int) []ToolInfo                   { return []ToolInfo{} }
func (m *mockComponentRegistry) GetCheapestToolByCapability(capability string) (*ToolInfo, error) { return nil, nil }
func (m *mockComponentRegistry) GetAgentsByCapability(capability string) []AgentInfo     { return []AgentInfo{} }

type mockSessionManager struct{}

func (m *mockSessionManager) CreateSession(ctx context.Context, userID, campaignID string) (*SessionData, error) {
	return &SessionData{ID: "test-session", UserID: userID, CampaignID: campaignID}, nil
}
func (m *mockSessionManager) LoadSession(ctx context.Context, sessionID string) (*SessionData, error) {
	return &SessionData{ID: sessionID}, nil
}
func (m *mockSessionManager) SaveSession(ctx context.Context, session *SessionData) error   { return nil }
func (m *mockSessionManager) DeleteSession(ctx context.Context, sessionID string) error    { return nil }
func (m *mockSessionManager) ListSessions(ctx context.Context, userID string) ([]*SessionData, error) {
	return []*SessionData{}, nil
}

type mockSessionResumer struct{}

func (m *mockSessionResumer) ResumeSession(ctx context.Context, sessionID string) error { return nil }
func (m *mockSessionResumer) GetRestorableState(ctx context.Context, sessionID string) (*RestorableState, error) {
	return &RestorableState{}, nil
}
func (m *mockSessionResumer) RestoreUIState(ctx context.Context, sessionID string, state map[string]interface{}) error {
	return nil
}

type mockSessionExporter struct{}

func (m *mockSessionExporter) ExportSession(session *SessionData, format string, options *ExportOptions) ([]byte, error) {
	return []byte("exported"), nil
}
func (m *mockSessionExporter) GetSupportedFormats() []string                                        { return []string{"json"} }
func (m *mockSessionExporter) ValidateExportOptions(format string, options *ExportOptions) error { return nil }

type mockSessionAnalytics struct{}

func (m *mockSessionAnalytics) GetSessionMetrics(ctx context.Context, sessionID string) (*SessionMetrics, error) {
	return &SessionMetrics{}, nil
}
func (m *mockSessionAnalytics) AnalyzeSessionPatterns(ctx context.Context, userID string) (*SessionPatterns, error) {
	return &SessionPatterns{}, nil
}
func (m *mockSessionAnalytics) RecordInteraction(ctx context.Context, sessionID string, interaction *Interaction) error {
	return nil
}

type mockPerformanceProfiler struct{}

func (m *mockPerformanceProfiler) ProfileApplication(ctx context.Context, duration time.Duration) (*PerformanceReport, error) {
	return &PerformanceReport{}, nil
}
func (m *mockPerformanceProfiler) GetActiveProfiles(ctx context.Context) ([]*ProfileInfo, error) {
	return []*ProfileInfo{}, nil
}
func (m *mockPerformanceProfiler) StopProfiling(ctx context.Context, profileID string) error { return nil }

type mockCacheManager struct{}

func (m *mockCacheManager) GetMetrics(ctx context.Context, cacheName string) (*CacheMetrics, error) {
	return &CacheMetrics{}, nil
}
func (m *mockCacheManager) InvalidateCache(ctx context.Context, cacheName string) error { return nil }
func (m *mockCacheManager) OptimizeCache(ctx context.Context, cacheName string) error   { return nil }

type mockMemoryOptimizer struct{}

func (m *mockMemoryOptimizer) OptimizeMemory(ctx context.Context) (*MemoryOptimizationReport, error) {
	return &MemoryOptimizationReport{}, nil
}
func (m *mockMemoryOptimizer) GetMemoryUsage(ctx context.Context) (*MemoryUsage, error) {
	return &MemoryUsage{}, nil
}

type mockPerformanceMonitor struct{}

func (m *mockPerformanceMonitor) GetCurrentMetrics(ctx context.Context, component string) (*SystemMetrics, error) {
	return &SystemMetrics{}, nil
}
func (m *mockPerformanceMonitor) StartMonitoring(ctx context.Context, component string) error { return nil }
func (m *mockPerformanceMonitor) StopMonitoring(ctx context.Context, component string) error  { return nil }

type mockAlertManager struct{}

func (m *mockAlertManager) GetActiveAlerts(ctx context.Context, severity string) ([]*Alert, error) {
	return []*Alert{}, nil
}
func (m *mockAlertManager) CreateAlert(ctx context.Context, alert *Alert) error     { return nil }
func (m *mockAlertManager) ResolveAlert(ctx context.Context, alertID string) error { return nil }

type mockSLOMonitor struct{}

func (m *mockSLOMonitor) CheckSLO(ctx context.Context, sloName string) (*SLOStatus, error) {
	return &SLOStatus{}, nil
}
func (m *mockSLOMonitor) UpdateSLO(ctx context.Context, sloName string, target float64) error { return nil }

type mockDashboardRenderer struct{}

func (m *mockDashboardRenderer) RenderDashboard(ctx context.Context, config *DashboardConfig) ([]byte, error) {
	return []byte("dashboard"), nil
}
func (m *mockDashboardRenderer) GetAvailableWidgets() []string { return []string{"widget1"} }