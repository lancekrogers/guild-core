// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package init

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// doInitialization performs the main initialization with proper context handling
func (m *InitTUIModelV2) doInitialization() tea.Cmd {
	return func() tea.Msg {
		// Create a sub-context with timeout for the entire operation
		ctx, cancel := context.WithTimeout(m.ctx, 5*time.Minute)
		defer cancel()
		
		// Check context at start
		if err := ctx.Err(); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeCancelled, "initialization cancelled").
				WithComponent("InitTUIV2").
				WithOperation("doInitialization")}
		}
		
		// Step 1: Check existing campaign
		if err := m.checkExistingCampaign(ctx); err != nil {
			return errMsg{err: err}
		}
		
		// Report progress
		select {
		case <-ctx.Done():
			return errMsg{err: gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "cancelled during campaign check")}
		default:
			// Continue
		}
		
		// Step 2: Initialize project structure
		if !m.projectInit.IsProjectInitialized(m.config.ProjectPath) {
			if err := m.projectInit.InitializeProject(ctx, m.config.ProjectPath); err != nil {
				return errMsg{err: gerror.Wrap(err, gerror.ErrCodeStorage, "failed to initialize project").
					WithComponent("InitTUIV2").
					WithOperation("doInitialization").
					WithDetails("path", m.config.ProjectPath)}
			}
		}
		
		// Check context after I/O operation
		if err := ctx.Err(); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeCancelled, "cancelled after project init")}
		}
		
		// Step 3: Create Phase 0 configuration
		if err := m.configManager.CreatePhase0Configuration(ctx, m.config.ProjectPath, m.campaignName, m.projectName); err != nil {
			return errMsg{err: err}
		}
		
		// Check context after configuration
		if err := ctx.Err(); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeCancelled, "cancelled after config creation")}
		}
		
		// Complete initialization
		return initProgressMsg{
			phase:   "complete",
			percent: 1.0,
			message: "Initialization complete",
		}
	}
}

// createDemoCommission creates a demo with proper error handling
func (m *InitTUIModelV2) createDemoCommission() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
		defer cancel()
		
		// Create commission directory
		commissionsDir := filepath.Join(m.config.ProjectPath, ".campaign", "commissions")
		if err := os.MkdirAll(commissionsDir, 0755); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create commissions directory").
				WithComponent("InitTUIV2").
				WithOperation("createDemoCommission").
				WithDetails("dir", commissionsDir)}
		}
		
		// Check context after I/O
		if err := ctx.Err(); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeCancelled, "cancelled during demo creation")}
		}
		
		// Generate commission content
		content, err := m.demoGen.GenerateCommission(ctx, m.demoType)
		if err != nil {
			// Don't fail entire init for demo
			return warnMsg{message: fmt.Sprintf("Could not generate demo commission: %v", err)}
		}
		
		// Write commission file
		fileName := fmt.Sprintf("demo-%s.md", string(m.demoType))
		commissionPath := filepath.Join(commissionsDir, fileName)
		
		// Write with context awareness
		if err := writeFileWithContext(ctx, commissionPath, []byte(content), 0644); err != nil {
			return warnMsg{message: fmt.Sprintf("Could not save demo commission: %v", err)}
		}
		
		return successMsg{message: fmt.Sprintf("Created demo commission: %s", fileName)}
	}
}

// doValidation performs validation with proper context handling
func (m *InitTUIModelV2) doValidation() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 2*time.Minute)
		defer cancel()
		
		// Phase 0 integration
		if err := m.configManager.IntegrateWithPhase0Config(ctx, m.config.ProjectPath, m.campaignName, m.projectName); err != nil {
			return errMsg{err: err}
		}
		
		// Check context between operations
		if err := ctx.Err(); err != nil {
			return errMsg{err: gerror.Wrap(err, gerror.ErrCodeCancelled, "cancelled during config integration")}
		}
		
		// Campaign reference already created in CreatePhase0Configuration
		
		// Socket registry
		if err := m.daemonManager.SaveSocketRegistry(m.config.ProjectPath, m.campaignName); err != nil {
			// Non-fatal but log it
			return warnMsg{message: fmt.Sprintf("Could not save socket registry: %v", err)}
		}
		
		// Run validation
		if err := m.validator.Validate(ctx); err != nil {
			// Validation errors are not fatal
			results := m.validator.GetResults()
			return validationResultsMsg{
				results: results,
				failed:  true,
			}
		}
		
		return validationResultsMsg{
			results: m.validator.GetResults(),
			failed:  false,
		}
	}
}

// Helper functions

func (m *InitTUIModelV2) checkExistingCampaign(ctx context.Context) error {
	// This would check for existing campaign
	// For now, just a placeholder
	return nil
}

// writeFileWithContext writes a file with context cancellation support
func writeFileWithContext(ctx context.Context, path string, data []byte, perm os.FileMode) error {
	// Create a channel to signal completion
	done := make(chan error, 1)
	
	go func() {
		done <- os.WriteFile(path, data, perm)
	}()
	
	select {
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "file write cancelled").
			WithDetails("path", path)
	case err := <-done:
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write file").
				WithDetails("path", path)
		}
		return nil
	}
}

// Message types with better structure

type initProgressMsg struct {
	phase   string
	percent float64
	message string
}

type successMsg struct {
	message string
}

type warnMsg struct {
	message string
}

type errMsg struct {
	err error
}

type validationResultsMsg struct {
	results []ValidationResult
	failed  bool
}