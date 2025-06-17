// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"os"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/project"
)

func TestNewWizard(t *testing.T) {
	ctx := context.Background()
	
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "guild-setup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize project
	if err := project.InitializeProject(tempDir); err != nil {
		t.Fatalf("Failed to initialize project: %v", err)
	}

	config := &Config{
		ProjectPath: tempDir,
		QuickMode:   true,
		Force:       false,
	}

	wizard, err := NewWizard(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create wizard: %v", err)
	}

	if wizard == nil {
		t.Fatal("Wizard is nil")
	}
	if wizard.config != config {
		t.Error("Wizard config not set correctly")
	}
}

func TestNewWizardNilConfig(t *testing.T) {
	ctx := context.Background()
	
	wizard, err := NewWizard(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil config")
	}
	if wizard != nil {
		t.Fatal("Expected nil wizard for nil config")
	}
}

func TestIsProjectSetup(t *testing.T) {
	ctx := context.Background()
	
	// Test with non-existent project
	isSetup, err := IsProjectSetup(ctx, "/non/existent/path")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if isSetup {
		t.Error("Expected project to not be setup")
	}
}

func TestGetSetupStatus(t *testing.T) {
	ctx := context.Background()
	
	// Test with non-existent project
	status, err := GetSetupStatus(ctx, "/non/existent/path")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if status == nil {
		t.Fatal("Status is nil")
	}
	if status.IsConfigured {
		t.Error("Expected project to not be configured")
	}
	if len(status.Providers) != 0 {
		t.Error("Expected no providers")
	}
	if len(status.Agents) != 0 {
		t.Error("Expected no agents")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				ProjectPath: "/tmp",
				QuickMode:   false,
				Force:       false,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWizard(ctx, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWizard() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}