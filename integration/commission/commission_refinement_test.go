// +build integration

package commission_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/orchestrator"
	"github.com/guild-ventures/guild-core/pkg/providers/mock"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommissionRefinementOnly tests just the commission refinement step
func TestCommissionRefinementOnly(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-refinement-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fmt.Printf("🧪 Testing commission refinement only\n")

	// Setup minimal registry
	reg := registry.NewComponentRegistry()
	ctx := context.Background()
	
	registryConfig := registry.Config{
		Providers: registry.ProviderConfig{
			DefaultProvider: "mock",
		},
	}
	err = reg.Initialize(ctx, registryConfig)
	require.NoError(t, err)

	// Setup mock provider with simple response that should validate
	mockProvider := mock.NewProvider()
	mockProvider.SetResponse("You are the Guild Master", `# Commission Refined

This is a simple refined commission with proper markdown structure.

<task>
<id>task-1</id>
<title>Simple Task</title>
<description>A simple task description</description>
<priority>medium</priority>
<estimate>2h</estimate>
<category>general</category>
<dependencies></dependencies>
</task>

## Implementation Plan

This is the implementation plan for the commission.`)

	err = reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	// Create integration service
	service, err := orchestrator.NewCommissionIntegrationService(reg)
	require.NoError(t, err)

	// Test just the commission refinement part
	fmt.Printf("Testing commission refinement...\n")
	
	// For now, let's test that the service was created successfully
	assert.NotNil(t, service)
	
	fmt.Printf("✅ Commission refinement service created successfully\n")
}

// TestSimpleResponseParsing tests the response parser with simple content
func TestSimpleResponseParsing(t *testing.T) {
	fmt.Printf("🧪 Testing response parsing\n")
	
	parser := manager.NewResponseParser()
	
	// Test with file structure format that includes README.md
	response := &manager.ArtisanResponse{
		Content: `## File: commission_refined.md

# Test Document

This is a simple test document.

<task>
<id>test-task</id>
<title>Test Task</title>
<description>A test task</description>
<priority>medium</priority>
<estimate>1h</estimate>
<category>testing</category>
<dependencies></dependencies>
</task>

## Implementation

Implementation details here.

## File: README.md

# Test Commission

This is a test commission for validation.

## Overview
This commission demonstrates the basic structure.`,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}
	
	structure, err := parser.ParseResponse(response)
	require.NoError(t, err)
	require.NotNil(t, structure)
	
	assert.Equal(t, 2, len(structure.Files), "Should create exactly 2 files")
	
	// Find the commission file
	var commissionFile, readmeFile *manager.FileEntry
	for _, file := range structure.Files {
		if file.Path == "commission_refined.md" {
			commissionFile = file
		} else if file.Path == "README.md" {
			readmeFile = file
		}
	}
	
	// Check the commission file
	require.NotNil(t, commissionFile, "Should have commission_refined.md")
	assert.Contains(t, commissionFile.Content, "# Test Document")
	assert.Equal(t, manager.FileTypeMarkdown, commissionFile.Type)
	
	// Check the README file
	require.NotNil(t, readmeFile, "Should have README.md")
	assert.Contains(t, readmeFile.Content, "# Test Commission")
	assert.Equal(t, manager.FileTypeMarkdown, readmeFile.Type)
	
	// Validate the content
	validator := manager.NewDefaultValidator()
	err = validator.ValidateStructure(structure)
	require.NoError(t, err, "Structure should validate successfully")
	
	fmt.Printf("✅ Response parsing works correctly\n")
}

// TestBasicTaskExtraction tests task extraction from content
func TestBasicTaskExtraction(t *testing.T) {
	fmt.Printf("🧪 Testing task extraction\n")
	
	parser := manager.NewResponseParser()
	
	content := `## File: commission_refined.md

# Test Commission

## Task Breakdown

- BACKEND-001: Design Database Schema (priority: high, estimate: 4h)
- FRONTEND-002: Create Product Listing UI (priority: medium, estimate: 2h, depends: BACKEND-001)

## Summary

Two tasks have been defined.

## File: README.md

# Test Commission

This is a test commission for validation purposes.`

	// Parse the response
	response := &manager.ArtisanResponse{Content: content}
	structure, err := parser.ParseResponse(response)
	require.NoError(t, err)
	
	// Debug: print structure details
	fmt.Printf("Found %d files:\n", len(structure.Files))
	for i, file := range structure.Files {
		fmt.Printf("  File %d: %s (tasks: %d)\n", i, file.Path, file.TasksCount)
		if tasks, ok := file.Metadata["tasks"]; ok {
			if taskList, ok := tasks.([]manager.TaskInfo); ok {
				fmt.Printf("    Tasks: %d items\n", len(taskList))
				for j, task := range taskList {
					fmt.Printf("      %d. %s (%s)\n", j+1, task.Title, task.ID)
				}
			}
		}
	}
	
	// Validate structure
	validator := manager.NewDefaultValidator()
	err = validator.ValidateStructure(structure)
	require.NoError(t, err)
	
	// Find the commission file that should contain tasks
	var commissionFile *manager.FileEntry
	for _, file := range structure.Files {
		if file.Path == "commission_refined.md" {
			commissionFile = file
			break
		}
	}
	require.NotNil(t, commissionFile, "Should have commission_refined.md file")
	
	// Check tasks were extracted from commission file
	tasks, ok := commissionFile.Metadata["tasks"]
	require.True(t, ok, "Tasks should be extracted")
	
	tasksList, ok := tasks.([]manager.TaskInfo)
	require.True(t, ok, "Tasks should be TaskInfo slice")
	assert.Len(t, tasksList, 2, "Should extract 2 tasks")
	
	// Verify task details (note: tasks may be in different order)
	taskByID := make(map[string]manager.TaskInfo)
	for _, task := range tasksList {
		taskByID[task.ID] = task
	}
	
	// Check BACKEND-001 task
	backendTask, exists := taskByID["BACKEND-001"]
	assert.True(t, exists, "Should have BACKEND-001 task")
	if exists {
		assert.Equal(t, "Design Database Schema", backendTask.Title)
		assert.Equal(t, "BACKEND", backendTask.Category)
		assert.Equal(t, "high", backendTask.Priority)
		assert.Equal(t, "4h", backendTask.Estimate)
	}
	
	// Check FRONTEND-002 task  
	frontendTask, exists := taskByID["FRONTEND-002"]
	assert.True(t, exists, "Should have FRONTEND-002 task")
	if exists {
		assert.Equal(t, "Create Product Listing UI", frontendTask.Title)
		assert.Equal(t, "FRONTEND", frontendTask.Category)
		assert.Equal(t, "medium", frontendTask.Priority)
		assert.Equal(t, "2h", frontendTask.Estimate)
		assert.Equal(t, []string{"BACKEND-001"}, frontendTask.Dependencies)
	}
	
	fmt.Printf("✅ Task extraction works correctly\n")
}