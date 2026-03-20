// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package setup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild-core/pkg/commission"
	"github.com/lancekrogers/guild-core/pkg/project"
)

// TestDemoCommissionEndToEnd tests the complete flow from generation to usage
func TestDemoCommissionEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary project directory
	tempDir, err := os.MkdirTemp("", "demo-commission-e2e-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Initialize project structure
	err = project.InitializeProject(tempDir)
	require.NoError(t, err)

	// Create demo commission generator
	generator := NewDemoCommissionGenerator()
	ctx := context.Background()

	// Test each demo type creates a valid commission in the right location
	commissionsDir := filepath.Join(tempDir, ".campaign", "objectives", "refined")
	err = os.MkdirAll(commissionsDir, 0o755)
	require.NoError(t, err)

	// Generate and save API demo commission
	content, err := generator.GenerateCommission(ctx, DemoTypeAPIService)
	require.NoError(t, err)

	// Save to the commissions directory
	commissionPath := filepath.Join(commissionsDir, "demo-api.md")
	err = os.WriteFile(commissionPath, []byte(content), 0o644)
	require.NoError(t, err)

	// Verify the file was created
	assert.FileExists(t, commissionPath)

	// Verify content structure
	savedContent, err := os.ReadFile(commissionPath)
	require.NoError(t, err)

	contentStr := string(savedContent)
	assert.Contains(t, contentStr, "RESTful Task Management API")
	assert.Contains(t, contentStr, "Project Objective")
	assert.Contains(t, contentStr, "## ")
	assert.Contains(t, contentStr, "- [ ]")

	// Test commission can be loaded by commission manager
	parser := commission.NewMarkdownParser(commission.DefaultParseOptions())
	parsedCommission, err := parser.ParseFile(commissionPath)
	require.NoError(t, err)

	assert.NotNil(t, parsedCommission)
	assert.NotEmpty(t, parsedCommission.Title)
	assert.NotEmpty(t, parsedCommission.Parts)
}

// TestDemoCommissionRecommendationFlow tests the recommendation system
func TestDemoCommissionRecommendationFlow(t *testing.T) {
	generator := NewDemoCommissionGenerator()
	ctx := context.Background()

	tests := []struct {
		name           string
		projectPath    string
		projectFiles   map[string]string
		expectedType   DemoCommissionType
		expectedReason string
	}{
		{
			name:        "Go API project",
			projectPath: "/test/user-api",
			projectFiles: map[string]string{
				"main.go":   "package main",
				"go.mod":    "module user-api",
				"README.md": "User API Service",
			},
			expectedType:   DemoTypeAPIService,
			expectedReason: "API service",
		},
		{
			name:        "React web project",
			projectPath: "/test/dashboard-app",
			projectFiles: map[string]string{
				"package.json": `{"name": "dashboard-app", "dependencies": {"react": "^18.0.0"}}`,
				"src/App.js":   "import React from 'react'",
			},
			expectedType:   DemoTypeWebApp,
			expectedReason: "React framework",
		},
		{
			name:        "Python data project",
			projectPath: "/test/analytics-tool",
			projectFiles: map[string]string{
				"requirements.txt": "pandas\nnumpy\njupyter",
				"analysis.py":      "import pandas as pd",
			},
			expectedType:   DemoTypeDataAnalysis,
			expectedReason: "data analysis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create project info based on test case
			projectInfo := map[string]interface{}{
				"project_name": filepath.Base(tt.projectPath),
			}

			// Add detected tech if applicable
			if _, hasPython := tt.projectFiles["requirements.txt"]; hasPython {
				if strings.Contains(tt.projectFiles["requirements.txt"], "pandas") {
					projectInfo["detected_tech"] = []string{"pandas", "jupyter"}
				}
			}
			if _, hasReact := tt.projectFiles["package.json"]; hasReact {
				if strings.Contains(tt.projectFiles["package.json"], "react") {
					projectInfo["detected_tech"] = []string{"React"}
				}
			}

			// Get recommendation
			recommendedType, reason := generator.GetRecommendedDemo(ctx, projectInfo)

			assert.Equal(t, tt.expectedType, recommendedType)
			assert.Contains(t, reason, tt.expectedReason)

			// Verify we can generate the recommended commission
			content, err := generator.GenerateCommission(ctx, recommendedType)
			require.NoError(t, err)
			assert.NotEmpty(t, content)
		})
	}
}

// TestDemoCommissionSelection tests the selection logic for different scenarios
func TestDemoCommissionSelection(t *testing.T) {
	generator := NewDemoCommissionGenerator()

	// Test that all demo types are available
	availableTypes := generator.GetAvailableTypes()
	assert.GreaterOrEqual(t, len(availableTypes), 6, "Should have at least 6 demo types")

	// Verify each type has a description
	for _, demoType := range availableTypes {
		desc := generator.GetDemoDescription(demoType)
		assert.NotEmpty(t, desc)
		assert.NotEqual(t, "Unknown demo type", desc)
	}

	// Test that each type generates unique content
	ctx := context.Background()
	contentMap := make(map[string]DemoCommissionType)

	for _, demoType := range availableTypes {
		content, err := generator.GenerateCommission(ctx, demoType)
		require.NoError(t, err)

		// Check for duplicate content
		if existingType, exists := contentMap[content]; exists {
			t.Errorf("Demo type %s has identical content to %s", demoType, existingType)
		}
		contentMap[content] = demoType
	}
}
