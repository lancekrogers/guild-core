// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package init_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/lancekrogers/guild-core/internal/teatest"

	"github.com/lancekrogers/guild-core/internal/setup"
	uiinit "github.com/lancekrogers/guild-core/internal/ui/init"
	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// Mock implementations for testing

type mockConfigManager struct {
	createPhase0Err    error
	integrateErr       error
	createReferenceErr error
	callCount          map[string]int
}

func newMockConfigManager() *mockConfigManager {
	return &mockConfigManager{
		callCount: make(map[string]int),
	}
}

func (m *mockConfigManager) EstablishGuildFoundation(ctx context.Context, projectPath, campaignName, projectName string) error {
	m.callCount["EstablishGuildFoundation"]++
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "mock cancelled")
	}
	return m.createPhase0Err
}

func (m *mockConfigManager) FinalizeGuildCharter(ctx context.Context, projectPath, campaignName, projectName string) error {
	m.callCount["FinalizeGuildCharter"]++
	return m.integrateErr
}

func (m *mockConfigManager) CreateCampaignReference(ctx context.Context, projectPath, campaignName, projectName string) error {
	m.callCount["CreateCampaignReference"]++
	return m.createReferenceErr
}

type mockProjectInit struct {
	initErr       error
	isInitialized bool
}

func (m *mockProjectInit) InitializeProject(ctx context.Context, projectPath string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "mock cancelled")
	}
	return m.initErr
}

func (m *mockProjectInit) IsProjectInitialized(projectPath string) bool {
	return m.isInitialized
}

type mockDemoGen struct {
	generateErr error
	types       []setup.DemoCommissionType
}

func (m *mockDemoGen) GenerateCommission(ctx context.Context, demoType setup.DemoCommissionType) (string, error) {
	return "# Mock Demo Commission", m.generateErr
}

func (m *mockDemoGen) GetAvailableTypes() []setup.DemoCommissionType {
	if len(m.types) > 0 {
		return m.types
	}
	return []setup.DemoCommissionType{
		setup.DemoTypeAPIService,
		setup.DemoTypeWebApp,
	}
}

func (m *mockDemoGen) GetDemoDescription(demoType setup.DemoCommissionType) string {
	return "Mock description for " + string(demoType)
}

type mockValidator struct {
	validateErr error
	hasFailures bool
	results     []uiinit.ValidationResult
}

func (m *mockValidator) Validate(ctx context.Context) error {
	return m.validateErr
}

func (m *mockValidator) HasFailures() bool {
	return m.hasFailures
}

func (m *mockValidator) GetResults() []uiinit.ValidationResult {
	if len(m.results) > 0 {
		return m.results
	}
	return []uiinit.ValidationResult{
		{Name: "Test Check", Passed: true, Message: "All good"},
	}
}

type mockDaemonManager struct {
	saveErr error
}

func (m *mockDaemonManager) SaveSocketRegistry(projectPath, campaignName string) error {
	return m.saveErr
}

// Test cases

func TestNewInitTUIModelV2(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		config  uiinit.Config
		wantErr bool
		errCode gerror.ErrorCode
	}{
		{
			name: "success",
			ctx:  context.Background(),
			config: uiinit.Config{
				ProjectPath: ".",
				QuickMode:   false,
			},
			wantErr: false,
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			config: uiinit.Config{
				ProjectPath: ".",
			},
			wantErr: true,
			errCode: gerror.ErrCodeCancelled,
		},
		{
			name: "invalid path",
			ctx:  context.Background(),
			config: uiinit.Config{
				ProjectPath: "/nonexistent/path/that/definitely/does/not/exist/and/cannot/be/resolved",
			},
			wantErr: false, // filepath.Abs() will still work on non-existent paths
			errCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := uiinit.InitDependencies{
				ConfigManager: newMockConfigManager(),
				ProjectInit:   &mockProjectInit{},
				DemoGen:       &mockDemoGen{},
				Validator:     &mockValidator{},
				DaemonManager: &mockDaemonManager{},
			}

			model, err := uiinit.NewInitTUIModelV2(tt.ctx, tt.config, deps, true)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errCode != "" && !gerror.Is(err, tt.errCode) {
					t.Errorf("expected error code %v, got %v", tt.errCode, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if model == nil {
					t.Error("expected model but got nil")
				}
			}
		})
	}
}

func TestInitTUIModelV2_QuickMode(t *testing.T) {
	ctx := context.Background()
	configMgr := newMockConfigManager()

	deps := uiinit.InitDependencies{
		ConfigManager: configMgr,
		ProjectInit:   &mockProjectInit{isInitialized: false},
		DemoGen:       &mockDemoGen{},
		Validator:     &mockValidator{},
		DaemonManager: &mockDaemonManager{},
	}

	config := uiinit.Config{
		ProjectPath: ".",
		QuickMode:   true,
	}

	model, err := uiinit.NewInitTUIModelV2(ctx, config, deps, true)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	// Initialize the model
	cmd := model.Init()
	if cmd == nil {
		t.Fatal("expected initialization command")
	}

	// Quick mode should immediately start initialization
	// We can't easily test the full flow without a real tea.Program,
	// but we can verify the initial state
	if model.GetError() != nil {
		t.Errorf("unexpected error: %v", model.GetError())
	}
}

func TestInitTUIModelV2_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	deps := uiinit.InitDependencies{
		ConfigManager: newMockConfigManager(),
		ProjectInit:   &mockProjectInit{},
		DemoGen:       &mockDemoGen{},
		Validator:     &mockValidator{},
		DaemonManager: &mockDaemonManager{},
	}

	config := uiinit.Config{
		ProjectPath: ".",
		QuickMode:   false,
	}

	model, err := uiinit.NewInitTUIModelV2(ctx, config, deps, true)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	// Cancel context
	cancel()

	// Update should detect cancellation
	_, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if model.GetError() == nil {
		t.Error("expected error for cancelled context")
	} else if !gerror.Is(model.GetError(), gerror.ErrCodeCancelled) {
		t.Errorf("expected cancelled error, got: %v", model.GetError())
	}
}

func TestInitTUIModelV2_ErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		setupDeps func() uiinit.InitDependencies
		wantErr   bool
	}{
		{
			name: "config manager error",
			setupDeps: func() uiinit.InitDependencies {
				configMgr := newMockConfigManager()
				configMgr.createPhase0Err = errors.New("config error")

				return uiinit.InitDependencies{
					ConfigManager: configMgr,
					ProjectInit:   &mockProjectInit{},
					DemoGen:       &mockDemoGen{},
					Validator:     &mockValidator{},
					DaemonManager: &mockDaemonManager{},
				}
			},
			wantErr: true,
		},
		{
			name: "project init error",
			setupDeps: func() uiinit.InitDependencies {
				return uiinit.InitDependencies{
					ConfigManager: newMockConfigManager(),
					ProjectInit:   &mockProjectInit{initErr: errors.New("init error")},
					DemoGen:       &mockDemoGen{},
					Validator:     &mockValidator{},
					DaemonManager: &mockDaemonManager{},
				}
			},
			wantErr: true,
		},
		{
			name: "daemon error non-fatal",
			setupDeps: func() uiinit.InitDependencies {
				return uiinit.InitDependencies{
					ConfigManager: newMockConfigManager(),
					ProjectInit:   &mockProjectInit{},
					DemoGen:       &mockDemoGen{},
					Validator:     &mockValidator{},
					DaemonManager: &mockDaemonManager{saveErr: errors.New("daemon error")},
				}
			},
			wantErr: false, // Daemon errors should be warnings, not fatal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			deps := tt.setupDeps()

			config := uiinit.Config{
				ProjectPath: ".",
				QuickMode:   true, // Use quick mode to trigger initialization
			}

			model, err := uiinit.NewInitTUIModelV2(ctx, config, deps, true)
			if err != nil {
				t.Fatalf("failed to create model: %v", err)
			}

			// We would need to run through the full tea.Program to test error propagation
			// For now, we're testing that the model is created correctly
			if model == nil {
				t.Error("expected model but got nil")
			}
		})
	}
}

func TestInitTUIModelV2_Styling(t *testing.T) {
	// Test that styles are properly initialized
	styles := uiinit.NewStyles()

	if styles == nil {
		t.Fatal("expected styles but got nil")
	}

	// Test some key style properties
	if styles.Title.GetBold() != true {
		t.Error("expected title to be bold")
	}

	// Test helper methods
	header := styles.RenderHeader("Test Title", "Test Subtitle")
	if header == "" {
		t.Error("expected header content")
	}

	success := styles.RenderSuccess("Success message")
	if success == "" {
		t.Error("expected success message")
	}
}

func TestDemoRenderer(t *testing.T) {
	styles := uiinit.NewStyles()
	renderer, err := uiinit.NewDemoRenderer(80, styles)
	if err != nil {
		t.Fatalf("failed to create demo renderer: %v", err)
	}

	// Test demo info
	demos := uiinit.GetDemoInfo()
	if len(demos) == 0 {
		t.Error("expected demo info")
	}

	// Test rendering
	output := renderer.RenderDemoSelection(demos, 0)
	if output == "" {
		t.Error("expected demo selection output")
	}

	// Test validation results rendering
	results := []uiinit.ValidationResult{
		{Name: "Test Pass", Passed: true, Message: "All good"},
		{Name: "Test Fail", Passed: false, Message: "Something wrong"},
	}

	validationOutput := renderer.RenderValidationResults(results)
	if validationOutput == "" {
		t.Error("expected validation output")
	}
}

// Benchmarks to ensure performance

func BenchmarkNewInitTUIModelV2(b *testing.B) {
	ctx := context.Background()
	deps := uiinit.InitDependencies{
		ConfigManager: newMockConfigManager(),
		ProjectInit:   &mockProjectInit{},
		DemoGen:       &mockDemoGen{},
		Validator:     &mockValidator{},
		DaemonManager: &mockDaemonManager{},
	}

	config := uiinit.Config{
		ProjectPath: ".",
		QuickMode:   false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := uiinit.NewInitTUIModelV2(ctx, config, deps, true)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkDemoRendering(b *testing.B) {
	styles := uiinit.NewStyles()
	renderer, _ := uiinit.NewDemoRenderer(80, styles)
	demos := uiinit.GetDemoInfo()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = renderer.RenderDemoSelection(demos, 0)
	}
}

// Integration tests using teatest

func TestInitTUIModel_Integration_QuickMode(t *testing.T) {
	// Create mocks that simulate quick successful initialization
	configMgr := newMockConfigManager()

	deps := uiinit.InitDependencies{
		ConfigManager: configMgr,
		ProjectInit:   &mockProjectInit{isInitialized: false},
		DemoGen:       &mockDemoGen{},
		Validator:     &mockValidator{},
		DaemonManager: &mockDaemonManager{},
	}

	config := uiinit.Config{
		ProjectPath: ".",
		QuickMode:   true, // This should complete automatically
	}

	// Create model
	model, err := uiinit.NewInitTUIModelV2(context.Background(), config, deps, true)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	// Use teatest to run the program
	tm := teatest.NewTestModel(
		t,
		model,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for the app to reach completion state, then send quit
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			// Look for completion indicators
			return contains(bts, "✅") || contains(bts, "complete") || contains(bts, "success")
		},
		teatest.WithCheckInterval(100*time.Millisecond),
		teatest.WithDuration(3*time.Second),
	)

	// Send quit to exit the program since it doesn't auto-quit
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEscape})

	// Now it should finish
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Check final output
	finalOutput := tm.FinalOutput(t)
	outputBytes, err := io.ReadAll(finalOutput)
	if err != nil {
		t.Fatalf("failed to read final output: %v", err)
	}

	t.Logf("Final output: %s", string(outputBytes))

	// Check that there were no errors
	finalModel := tm.FinalModel(t)
	if initModel, ok := finalModel.(*uiinit.InitTUIModelV2); ok {
		if initModel.GetError() != nil {
			t.Errorf("Expected no error but got: %v", initModel.GetError())
		}
	}
}

func TestInitTUIModel_Integration_QuickModeFullFlow(t *testing.T) {
	// Create mocks that simulate successful initialization and validation
	configMgr := newMockConfigManager()
	validator := &mockValidator{
		hasFailures: false,
		results: []uiinit.ValidationResult{
			{Name: "Database Check", Passed: true, Message: "SQLite database initialized"},
			{Name: "Config Check", Passed: true, Message: "Guild configuration valid"},
		},
	}

	deps := uiinit.InitDependencies{
		ConfigManager: configMgr,
		ProjectInit:   &mockProjectInit{isInitialized: false},
		DemoGen:       &mockDemoGen{},
		Validator:     validator,
		DaemonManager: &mockDaemonManager{},
	}

	config := uiinit.Config{
		ProjectPath: ".",
		QuickMode:   true,
	}

	// Create model
	model, err := uiinit.NewInitTUIModelV2(context.Background(), config, deps, true)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	// Use teatest to run the program with longer timeout for full flow
	tm := teatest.NewTestModel(
		t,
		model,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for the app to complete all phases and reach final completion
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			// Look for completion indicators in the final success screen
			return (contains(bts, "Guild Successfully Established") ||
				contains(bts, "Press Enter to exit") ||
				contains(bts, "guild chat"))
		},
		teatest.WithCheckInterval(100*time.Millisecond),
		teatest.WithDuration(5*time.Second), // Longer timeout for full flow
	)

	// Send quit to exit the program
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Should finish quickly now
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Check final output for success indicators
	finalOutput := tm.FinalOutput(t)
	outputBytes, err := io.ReadAll(finalOutput)
	if err != nil {
		t.Fatalf("failed to read final output: %v", err)
	}

	outputStr := string(outputBytes)
	t.Logf("Final full flow output: %s", outputStr)

	// Verify no errors and successful completion
	finalModel := tm.FinalModel(t)
	if initModel, ok := finalModel.(*uiinit.InitTUIModelV2); ok {
		if initModel.GetError() != nil {
			t.Errorf("Expected no error but got: %v", initModel.GetError())
		}
	}

	// Verify that all mock methods were called (indicating full flow)
	if configMgr.callCount["EstablishGuildFoundation"] == 0 {
		t.Error("Expected EstablishGuildFoundation to be called")
	}
	if configMgr.callCount["FinalizeGuildCharter"] == 0 {
		t.Error("Expected FinalizeGuildCharter to be called")
	}
	if configMgr.callCount["CreateCampaignReference"] == 0 {
		t.Error("Expected CreateCampaignReference to be called")
	}
}

func TestInitTUIModel_Integration_InteractiveCancel(t *testing.T) {
	t.Skip("Interactive test - demonstrates pattern but skipped for CI")

	// This test demonstrates how we would test interactive behavior
	deps := uiinit.InitDependencies{
		ConfigManager: newMockConfigManager(),
		ProjectInit:   &mockProjectInit{isInitialized: false},
		DemoGen:       &mockDemoGen{},
		Validator:     &mockValidator{},
		DaemonManager: &mockDaemonManager{},
	}

	config := uiinit.Config{
		ProjectPath: ".",
		QuickMode:   false, // Interactive mode
	}

	model, err := uiinit.NewInitTUIModelV2(context.Background(), config, deps, true)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	tm := teatest.NewTestModel(
		t,
		model,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for initial render
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			return contains(bts, "Guild") // Wait for welcome screen
		},
		teatest.WithCheckInterval(100*time.Millisecond),
		teatest.WithDuration(time.Second),
	)

	// Send quit key
	tm.Send(tea.KeyPressMsg{
		Text: "q",
		Code: 'q',
	})

	// Program should exit
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	finalOutput := tm.FinalOutput(t)
	outputBytes, err := io.ReadAll(finalOutput)
	if err != nil {
		t.Fatalf("failed to read final output: %v", err)
	}
	t.Logf("Final output: %s", string(outputBytes))
}

// Helper function to check if bytes contain a string
func contains(data []byte, s string) bool {
	return bytes.Contains(data, []byte(s))
}
