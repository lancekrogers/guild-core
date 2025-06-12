// Package testutil provides test utilities for Guild Framework integration tests
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

// TestProjectOptions configures test project setup
type TestProjectOptions struct {
	// Name for the test project
	Name string
	// SkipDatabase skips database initialization
	SkipDatabase bool
	// CustomConfig provides custom guild configuration
	CustomConfig *config.GuildConfig
	// WithCorpus creates corpus test data
	WithCorpus bool
	// WithObjectives creates test objectives
	WithObjectives bool
}

// SetupTestProject creates a complete test project environment
// Returns project context and cleanup function
func SetupTestProject(t *testing.T, opts ...TestProjectOptions) (*project.Context, func()) {
	t.Helper()

	// Apply options
	var options TestProjectOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	// Set defaults
	if options.Name == "" {
		options.Name = "test-guild-" + t.Name()
	}

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "guild-test-*")
	require.NoError(t, err, "failed to create temp directory")

	// Initialize project and get project context
	ctx := context.Background()
	projCtx, err := project.Initialize(ctx, tempDir, project.InitOptions{})
	require.NoError(t, err, "failed to initialize project")

	// Apply custom configuration if provided
	if options.CustomConfig != nil {
		guildPath := filepath.Join(projCtx.GetGuildPath(), "guild.yaml")
		data, err := yaml.Marshal(options.CustomConfig)
		require.NoError(t, err, "failed to marshal custom config")
		err = os.WriteFile(guildPath, data, 0644)
		require.NoError(t, err, "failed to write custom config")
	}

	// Create corpus test data if requested
	if options.WithCorpus {
		createTestCorpus(t, projCtx)
	}

	// Create objectives if requested
	if options.WithObjectives {
		createTestObjectives(t, projCtx)
	}

	// Cleanup function
	cleanup := func() {
		if t.Failed() {
			t.Logf("Test failed, preserving test directory: %s", tempDir)
		} else {
			os.RemoveAll(tempDir)
		}
	}

	return projCtx, cleanup
}

// CleanupTestProject ensures proper cleanup of test resources
func CleanupTestProject(t *testing.T, rootPath string) {
	t.Helper()

	if rootPath == "" || rootPath == "/" || rootPath == os.Getenv("HOME") {
		t.Fatal("refusing to cleanup dangerous path")
	}

	err := os.RemoveAll(rootPath)
	if err != nil {
		t.Logf("failed to cleanup test project: %v", err)
	}
}

// CreateTestGuildConfig generates a test guild configuration
func CreateTestGuildConfig(name string) *config.GuildConfig {
	return &config.GuildConfig{
		Name:        name,
		Description: "Test guild for " + name,
		Providers: config.ProvidersConfig{
			Ollama: config.ProviderSettings{
				BaseURL: "http://mock.local:11434",
			},
		},
		Agents: []config.AgentConfig{
			{
				ID:       "test-manager",
				Name:     "Test Manager",
				Type:     "manager",
				Provider: "mock",
				Model:    "mock-model",
				Capabilities: []string{
					"planning",
					"coordination",
					"task_breakdown",
				},
			},
			{
				ID:       "test-developer",
				Name:     "Test Developer",
				Type:     "worker",
				Provider: "mock",
				Model:    "mock-model",
				Capabilities: []string{
					"coding",
					"testing",
					"debugging",
				},
			},
			{
				ID:       "test-reviewer",
				Name:     "Test Reviewer",
				Type:     "worker",
				Provider: "mock",
				Model:    "mock-model",
				Capabilities: []string{
					"review",
					"testing",
					"documentation",
				},
			},
		},
		// Objectives are stored as files, not in config
	}
}

// InitTestDatabase sets up a test SQLite database with migrations
func InitTestDatabase(t *testing.T, projCtx *project.Context) {
	t.Helper()

	// Database is automatically initialized during project.Initialize()
	// This function is here for explicit database operations if needed
	dbPath := filepath.Join(projCtx.GetGuildPath(), "guild.db")

	// Verify database exists
	_, err := os.Stat(dbPath)
	require.NoError(t, err, "database file should exist")
}

// createTestCorpus creates sample corpus files for testing
func createTestCorpus(t *testing.T, projCtx *project.Context) {
	t.Helper()

	docsPath := filepath.Join(projCtx.GetCorpusPath(), "docs")

	// Architecture document
	archDoc := `# System Architecture

## Overview
This is a test system with modular architecture.

## Components
- API Gateway
- Business Logic Layer
- Data Access Layer

## Technologies
- Go
- SQLite
- gRPC`

	err := os.WriteFile(filepath.Join(docsPath, "architecture.md"), []byte(archDoc), 0644)
	require.NoError(t, err)

	// API documentation
	apiDoc := `# API Documentation

## Endpoints

### GET /api/v1/users
Returns list of users

### POST /api/v1/users
Creates a new user

### GET /api/v1/users/:id
Returns user by ID`

	err = os.WriteFile(filepath.Join(docsPath, "api.md"), []byte(apiDoc), 0644)
	require.NoError(t, err)

	// README
	readmeDoc := `# Test Project

This is a test project for Guild Framework integration testing.

## Features
- User management
- Authentication
- API Gateway`

	err = os.WriteFile(filepath.Join(docsPath, "README.md"), []byte(readmeDoc), 0644)
	require.NoError(t, err)
}

// createTestObjectives creates sample objectives for testing
func createTestObjectives(t *testing.T, projCtx *project.Context) {
	t.Helper()

	objectivesPath := projCtx.GetObjectivesPath()

	// Sample objective
	objective := `# User Authentication System

## Objective
Implement a complete user authentication system with JWT tokens.

## Requirements
- User registration with email verification
- Login with JWT token generation
- Password reset functionality
- Role-based access control

## Technical Details
- Use bcrypt for password hashing
- JWT tokens with 24-hour expiration
- Refresh token support
- Rate limiting on auth endpoints

## Success Criteria
- All endpoints have tests
- Documentation is complete
- Security best practices followed`

	err := os.WriteFile(filepath.Join(objectivesPath, "auth-system.md"), []byte(objective), 0644)
	require.NoError(t, err)

	// Another objective
	apiObjective := `# REST API Implementation

## Objective
Create RESTful API for user management

## Endpoints Required
- CRUD operations for users
- Pagination support
- Filtering and sorting
- Proper error responses

## Standards
- Follow REST conventions
- Use proper HTTP status codes
- Include OpenAPI documentation`

	err = os.WriteFile(filepath.Join(objectivesPath, "rest-api.md"), []byte(apiObjective), 0644)
	require.NoError(t, err)
}

// WithTestProject runs a test function with a test project context
func WithTestProject(t *testing.T, name string, fn func(context.Context, *project.Context)) {
	t.Helper()

	projCtx, cleanup := SetupTestProject(t, TestProjectOptions{Name: name})
	defer cleanup()

	ctx := project.WithContext(context.Background(), projCtx)
	fn(ctx, projCtx)
}

// AssertProjectStructure verifies the project structure is correct
func AssertProjectStructure(t *testing.T, projCtx *project.Context) {
	t.Helper()

	// Check required directories exist
	dirs := []string{
		projCtx.GetGuildPath(),
		projCtx.GetCorpusPath(),
		projCtx.GetEmbeddingsPath(),
		projCtx.GetAgentsPath(),
		projCtx.GetObjectivesPath(),
		filepath.Join(projCtx.GetCorpusPath(), "docs"),
		filepath.Join(projCtx.GetCorpusPath(), ".activities"),
	}

	for _, dir := range dirs {
		info, err := os.Stat(dir)
		require.NoError(t, err, "directory should exist: %s", dir)
		require.True(t, info.IsDir(), "should be a directory: %s", dir)
	}

	// Check required files exist
	files := []string{
		filepath.Join(projCtx.GetGuildPath(), "guild.yaml"),
		filepath.Join(projCtx.GetGuildPath(), "config.yaml"),
		filepath.Join(projCtx.GetGuildPath(), "guild.db"),
		filepath.Join(projCtx.GetGuildPath(), "README.md"),
		filepath.Join(projCtx.GetGuildPath(), ".gitignore"),
	}

	for _, file := range files {
		_, err := os.Stat(file)
		require.NoError(t, err, "file should exist: %s", file)
	}
}
