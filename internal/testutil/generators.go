// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package testutil

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/tools"
)

// CommissionOptions configures test commission generation
type CommissionOptions struct {
	Title      string
	Complexity string // simple, medium, complex
	Domain     string // web, api, cli, data
	NumTasks   int
}

// GenerateTestCommission creates a sample commission document
func GenerateTestCommission(opts CommissionOptions) string {
	// Set defaults
	if opts.Title == "" {
		opts.Title = "Test Commission"
	}
	if opts.Complexity == "" {
		opts.Complexity = "medium"
	}
	if opts.Domain == "" {
		opts.Domain = "api"
	}
	if opts.NumTasks == 0 {
		opts.NumTasks = 3
	}

	// Build commission based on complexity
	var commission strings.Builder
	commission.WriteString(fmt.Sprintf("# %s\n\n", opts.Title))
	commission.WriteString("## Overview\n")

	switch opts.Complexity {
	case "simple":
		commission.WriteString("A straightforward implementation task requiring basic functionality.\n\n")
	case "complex":
		commission.WriteString("A comprehensive system implementation requiring multiple components and careful architecture.\n\n")
	default:
		commission.WriteString("A moderate complexity task requiring thoughtful implementation.\n\n")
	}

	// Add domain-specific content
	commission.WriteString("## Requirements\n\n")
	switch opts.Domain {
	case "web":
		commission.WriteString("- Responsive web interface\n")
		commission.WriteString("- Modern frontend framework\n")
		commission.WriteString("- RESTful API backend\n")
	case "api":
		commission.WriteString("- RESTful API design\n")
		commission.WriteString("- Authentication and authorization\n")
		commission.WriteString("- Comprehensive error handling\n")
	case "cli":
		commission.WriteString("- Command-line interface\n")
		commission.WriteString("- Configuration file support\n")
		commission.WriteString("- Progress indicators\n")
	case "data":
		commission.WriteString("- Data processing pipeline\n")
		commission.WriteString("- ETL operations\n")
		commission.WriteString("- Performance optimization\n")
	}

	// Add tasks
	commission.WriteString("\n## Tasks\n\n")
	for i := 1; i <= opts.NumTasks; i++ {
		commission.WriteString(fmt.Sprintf("%d. Task %d: %s\n", i, i, generateTaskDescription(opts.Domain, i)))
	}

	// Add technical specifications
	commission.WriteString("\n## Technical Specifications\n\n")
	commission.WriteString("- Programming Language: Go\n")
	commission.WriteString("- Testing: Unit and integration tests required\n")
	commission.WriteString("- Documentation: API documentation and README\n")

	// Add success criteria
	commission.WriteString("\n## Success Criteria\n\n")
	commission.WriteString("- All functionality implemented and tested\n")
	commission.WriteString("- Code follows best practices\n")
	commission.WriteString("- Documentation is complete\n")
	commission.WriteString("- Performance meets requirements\n")

	return commission.String()
}

// AgentResponseOptions configures mock agent responses
type AgentResponseOptions struct {
	Type     string   // task_breakdown, implementation, review
	Tasks    []string // For task breakdown
	Code     string   // For implementation
	Feedback string   // For review
	Status   string   // success, partial, error
}

// GenerateMockAgentResponse creates a realistic agent response
func GenerateMockAgentResponse(opts AgentResponseOptions) *manager.ArtisanResponse {
	// Set defaults
	if opts.Type == "" {
		opts.Type = "task_breakdown"
	}
	if opts.Status == "" {
		opts.Status = "success"
	}

	response := &manager.ArtisanResponse{
		Metadata: map[string]interface{}{
			"agent_id":  "test-agent",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	switch opts.Type {
	case "task_breakdown":
		response.Content = generateTaskBreakdown(opts.Tasks)
	case "implementation":
		if opts.Code != "" {
			response.Content = generateImplementationResponse(opts.Code)
		} else {
			response.Content = generateDefaultImplementation()
		}
	case "review":
		if opts.Feedback != "" {
			response.Content = generateReviewResponse(opts.Feedback)
		} else {
			response.Content = generateDefaultReview()
		}
	}

	return response
}

// GenerateTestToolImplementation creates a mock tool for testing
func GenerateTestToolImplementation(name string, category string) tools.Tool {
	return &mockTool{
		name:        name,
		category:    category,
		description: fmt.Sprintf("Mock %s tool for testing", name),
	}
}

// mockTool implements the Tool interface for testing
type mockTool struct {
	name        string
	category    string
	description string
	executeFunc func(ctx context.Context, input string) (*tools.ToolResult, error)
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type":        "string",
				"description": "Input for the tool",
			},
		},
	}
}

func (m *mockTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	// Default mock execution
	return tools.NewToolResult(
		fmt.Sprintf("Mock execution of %s tool", m.name),
		map[string]string{"tool": m.name},
		nil,
		map[string]interface{}{"input": input},
	), nil
}

func (m *mockTool) Examples() []string {
	return []string{"example input"}
}

func (m *mockTool) Category() string {
	return m.category
}

func (m *mockTool) RequiresAuth() bool {
	return false
}

// CampaignConfigOptions configures test campaign generation
type CampaignConfigOptions struct {
	Name         string
	NumAgents    int
	NumProviders int
	WithTools    bool
}

// GenerateCampaignConfig creates a test campaign configuration
func GenerateCampaignConfig(opts CampaignConfigOptions) *config.GuildConfig {
	// Set defaults
	if opts.Name == "" {
		opts.Name = "test-campaign"
	}
	if opts.NumAgents == 0 {
		opts.NumAgents = 3
	}
	if opts.NumProviders == 0 {
		opts.NumProviders = 1
	}

	cfg := &config.GuildConfig{
		Name:        opts.Name,
		Description: fmt.Sprintf("Test campaign configuration for %s", opts.Name),
		Agents:      make([]config.AgentConfig, 0, opts.NumAgents),
	}

	// Configure mock provider in providers config
	cfg.Providers = config.ProvidersConfig{
		Ollama: config.ProviderSettings{
			BaseURL: "http://mock.local:11434",
		},
	}

	// Add agents with different roles
	agentRoles := []struct {
		role         string
		agentType    string
		capabilities []string
	}{
		{"manager", "manager", []string{"planning", "coordination", "task_breakdown"}},
		{"developer", "worker", []string{"coding", "testing", "debugging"}},
		{"reviewer", "worker", []string{"review", "documentation", "quality_assurance"}},
		{"architect", "specialist", []string{"design", "architecture", "technical_decisions"}},
		{"tester", "worker", []string{"testing", "test_automation", "quality_assurance"}},
	}

	for i := 0; i < opts.NumAgents && i < len(agentRoles); i++ {
		role := agentRoles[i]
		cfg.Agents = append(cfg.Agents, config.AgentConfig{
			ID:           fmt.Sprintf("test-%s", role.role),
			Name:         fmt.Sprintf("Test %s", strings.Title(role.role)),
			Type:         role.agentType,
			Provider:     "ollama", // Use mock provider
			Model:        "mock-model",
			Capabilities: role.capabilities,
		})
	}

	// Add tools to agent configurations if requested
	if opts.WithTools {
		for i := range cfg.Agents {
			cfg.Agents[i].Tools = []string{"file", "shell", "http"}
		}
	}

	return cfg
}

// Helper functions

func generateTaskDescription(domain string, taskNum int) string {
	descriptions := map[string][]string{
		"api": {
			"Design and implement the API schema",
			"Implement authentication middleware",
			"Add request validation and error handling",
			"Write comprehensive API tests",
			"Create API documentation",
		},
		"web": {
			"Set up the frontend framework",
			"Implement the user interface components",
			"Connect frontend to backend API",
			"Add responsive design",
			"Implement user authentication flow",
		},
		"cli": {
			"Design command structure",
			"Implement core commands",
			"Add configuration file support",
			"Implement progress indicators",
			"Write command documentation",
		},
		"data": {
			"Design data pipeline architecture",
			"Implement data ingestion",
			"Add data transformation logic",
			"Implement data validation",
			"Add monitoring and logging",
		},
	}

	domainTasks := descriptions[domain]
	if taskNum <= len(domainTasks) {
		return domainTasks[taskNum-1]
	}
	return fmt.Sprintf("Implement feature %d", taskNum)
}

func generateTaskBreakdown(tasks []string) string {
	if len(tasks) == 0 {
		tasks = []string{
			"Set up project structure",
			"Implement core functionality",
			"Add tests",
			"Write documentation",
		}
	}

	var content strings.Builder
	content.WriteString("Based on the commission, I'll break this down into the following tasks:\n\n")

	for i, task := range tasks {
		content.WriteString(fmt.Sprintf("## Task %d: %s\n", i+1, task))
		content.WriteString("**Priority**: High\n")
		content.WriteString("**Estimated effort**: 2-4 hours\n")
		content.WriteString("**Dependencies**: ")
		if i == 0 {
			content.WriteString("None\n")
		} else {
			content.WriteString(fmt.Sprintf("Task %d\n", i))
		}
		content.WriteString("\n")
	}

	content.WriteString("## Execution Plan\n")
	content.WriteString("I recommend executing these tasks in sequence, with regular reviews after each task.\n")

	return content.String()
}

func generateImplementationResponse(code string) string {
	return fmt.Sprintf(`I've implemented the requested functionality:

## Implementation

%s

## Explanation
The implementation follows best practices and includes proper error handling.

## Next Steps
- Run tests to verify functionality
- Review code for improvements
- Update documentation`, code)
}

func generateDefaultImplementation() string {
	code := `package main

import (
    "fmt"
    "log"
)

// ExampleFunction demonstrates the implementation
func ExampleFunction(input string) (string, error) {
    if input == "" {
        return "", fmt.Errorf("input cannot be empty")
    }
    
    result := fmt.Sprintf("Processed: %s", input)
    log.Printf("Processing completed for: %s", input)
    
    return result, nil
}

func main() {
    result, err := ExampleFunction("test input")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)
}`

	return generateImplementationResponse(code)
}

func generateReviewResponse(feedback string) string {
	return fmt.Sprintf(`## Code Review

I've reviewed the implementation and have the following feedback:

%s

## Summary
The code is generally well-structured but could benefit from the improvements mentioned above.

## Recommendation
After addressing these points, the code will be ready for production use.`, feedback)
}

func generateDefaultReview() string {
	feedback := `### Positive Aspects
- Clean code structure
- Good error handling
- Follows Go conventions

### Areas for Improvement
- Add more comprehensive tests
- Consider edge cases
- Improve documentation

### Suggestions
1. Add unit tests for error cases
2. Include examples in documentation
3. Consider performance optimizations for large inputs`

	return generateReviewResponse(feedback)
}
