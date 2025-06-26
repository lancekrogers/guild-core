// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commission

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/observability"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

// TaskDecomposer implements intelligent task breakdown with pattern matching
type TaskDecomposer struct {
	patterns  map[string]*TaskPattern
	llmClient providers.LLMClient
}

// TaskPattern defines a pattern for generating tasks from requirements
type TaskPattern struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Matcher     PatternMatcher  `json:"-"` // Not serialized
	Templates   []TaskTemplate  `json:"templates"`
	Priority    int             `json:"priority"`
	Category    string          `json:"category"`
}

// PatternMatcher is a function that determines if a requirement matches a pattern
type PatternMatcher func(requirement Requirement) bool

// TaskTemplate defines a template for creating tasks
type TaskTemplate struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"` // design, implementation, testing, documentation
	Description  string            `json:"description"`
	Prerequisites []string         `json:"prerequisites"`
	Complexity   int               `json:"base_complexity"`
	Metadata     map[string]string `json:"metadata"`
}

// RequirementAnalyzer analyzes commission content and extracts structured requirements
type RequirementAnalyzer struct {
	llmClient providers.LLMClient
}

// NewTaskDecomposer creates a new task decomposer with predefined patterns
func NewTaskDecomposer(llmClient providers.LLMClient) *TaskDecomposer {
	td := &TaskDecomposer{
		patterns:  make(map[string]*TaskPattern),
		llmClient: llmClient,
	}
	
	td.initializePatterns()
	return td
}

// NewRequirementAnalyzer creates a new requirement analyzer
func NewRequirementAnalyzer(llmClient providers.LLMClient) *RequirementAnalyzer {
	return &RequirementAnalyzer{
		llmClient: llmClient,
	}
}

// Decompose decomposes an analysis into actionable tasks
func (td *TaskDecomposer) Decompose(ctx context.Context, analysis *Analysis, agentResources *AgentResourceSummary) ([]*RefinedTask, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("commission.task_decomposer").
			WithOperation("Decompose")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "commission.task_decomposer")
	ctx = observability.WithOperation(ctx, "Decompose")

	logger.InfoContext(ctx, "Decomposing analysis into tasks",
		"requirements_count", len(analysis.Requirements),
		"available_agents", len(agentResources.AvailableAgents))

	var allTasks []*RefinedTask
	taskCounter := 0

	// Process each requirement through pattern matching
	for _, requirement := range analysis.Requirements {
		logger.DebugContext(ctx, "Processing requirement",
			"requirement_id", requirement.ID,
			"requirement_type", requirement.Type)

		// Find matching patterns
		matchingPatterns := td.findMatchingPatterns(requirement)
		
		if len(matchingPatterns) > 0 {
			// Use the highest priority pattern
			bestPattern := td.selectBestPattern(matchingPatterns)
			tasks := td.applyPattern(bestPattern, requirement, &taskCounter)
			allTasks = append(allTasks, tasks...)
			
			logger.DebugContext(ctx, "Applied pattern to requirement",
				"requirement_id", requirement.ID,
				"pattern_name", bestPattern.Name,
				"tasks_generated", len(tasks))
		} else {
			// Use generic decomposition
			tasks := td.genericDecompose(requirement, &taskCounter)
			allTasks = append(allTasks, tasks...)
			
			logger.DebugContext(ctx, "Applied generic decomposition",
				"requirement_id", requirement.ID,
				"tasks_generated", len(tasks))
		}
	}

	// Add dependencies between tasks
	td.addDependencies(allTasks)

	// Generate additional tasks for project setup and completion
	setupTasks := td.generateSetupTasks(&taskCounter)
	completionTasks := td.generateCompletionTasks(&taskCounter)
	
	allTasks = append(setupTasks, allTasks...)
	allTasks = append(allTasks, completionTasks...)

	logger.InfoContext(ctx, "Task decomposition completed",
		"total_tasks_generated", len(allTasks),
		"setup_tasks", len(setupTasks),
		"requirement_tasks", len(allTasks)-len(setupTasks)-len(completionTasks),
		"completion_tasks", len(completionTasks))

	return allTasks, nil
}

// Analyze analyzes a commission and extracts structured requirements using AI
func (ra *RequirementAnalyzer) Analyze(ctx context.Context, commission *Commission) (*Analysis, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("commission.requirement_analyzer").
			WithOperation("Analyze")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "commission.requirement_analyzer")
	ctx = observability.WithOperation(ctx, "Analyze")

	logger.InfoContext(ctx, "Analyzing commission requirements",
		"commission_id", commission.ID,
		"commission_title", commission.Title)

	// Extract basic requirements from commission structure
	requirements := ra.extractBasicRequirements(commission)

	// Use AI to enhance requirement analysis if LLM client is available
	if ra.llmClient != nil {
		enhancedAnalysis, err := ra.aiEnhancedAnalysis(ctx, commission)
		if err != nil {
			logger.WarnContext(ctx, "AI-enhanced analysis failed, using basic extraction",
				"error", err.Error())
		} else {
			// Merge AI insights with basic requirements
			requirements = ra.mergeRequirements(requirements, enhancedAnalysis.Requirements)
		}
	}

	// Analyze technical stack from content
	technicalStack := ra.extractTechnicalStack(commission)

	// Determine scope and effort
	scope := ra.determineScope(commission, requirements)
	effort := ra.estimateEffort(scope, len(requirements))

	// Extract success criteria and deliverables
	successCriteria := ra.extractSuccessCriteria(commission)
	deliverables := ra.extractDeliverables(commission)

	// Identify risk factors
	riskFactors := ra.identifyRiskFactors(commission, requirements)

	analysis := &Analysis{
		Requirements:    requirements,
		TechnicalStack:  technicalStack,
		Scope:          scope,
		EstimatedEffort: effort,
		RiskFactors:     riskFactors,
		SuccessCriteria: successCriteria,
		KeyDeliverables: deliverables,
	}

	logger.InfoContext(ctx, "Commission analysis completed",
		"commission_id", commission.ID,
		"requirements_identified", len(requirements),
		"technical_stack_items", len(technicalStack),
		"scope", scope,
		"estimated_effort", effort)

	return analysis, nil
}

// initializePatterns sets up the predefined task patterns
func (td *TaskDecomposer) initializePatterns() {
	// API Endpoint Pattern
	td.patterns["api_endpoint"] = &TaskPattern{
		Name:        "API Endpoint",
		Description: "Pattern for REST API endpoint development",
		Matcher:     td.apiEndpointMatcher,
		Priority:    10,
		Category:    "backend",
		Templates: []TaskTemplate{
			{
				Name:        "Design {endpoint} API schema",
				Type:        "design",
				Description: "Design the request/response schema and API specification",
				Complexity:  3,
				Metadata:    map[string]string{"phase": "design"},
			},
			{
				Name:        "Implement {endpoint} handler",
				Type:        "implementation",
				Description: "Implement the API endpoint handler and business logic",
				Prerequisites: []string{"Design {endpoint} API schema"},
				Complexity:  5,
				Metadata:    map[string]string{"phase": "implementation"},
			},
			{
				Name:        "Write {endpoint} tests",
				Type:        "testing",
				Description: "Write unit and integration tests for the endpoint",
				Prerequisites: []string{"Implement {endpoint} handler"},
				Complexity:  3,
				Metadata:    map[string]string{"phase": "testing"},
			},
			{
				Name:        "Document {endpoint} API",
				Type:        "documentation",
				Description: "Create API documentation and usage examples",
				Prerequisites: []string{"Write {endpoint} tests"},
				Complexity:  2,
				Metadata:    map[string]string{"phase": "documentation"},
			},
		},
	}

	// Database Schema Pattern
	td.patterns["database_schema"] = &TaskPattern{
		Name:        "Database Schema",
		Description: "Pattern for database design and implementation",
		Matcher:     td.databaseSchemaMatcher,
		Priority:    9,
		Category:    "backend",
		Templates: []TaskTemplate{
			{
				Name:        "Design {entity} database schema",
				Type:        "design",
				Description: "Design database tables, relationships, and constraints",
				Complexity:  5,
				Metadata:    map[string]string{"phase": "design"},
			},
			{
				Name:        "Create {entity} migrations",
				Type:        "implementation",
				Description: "Create database migration scripts",
				Prerequisites: []string{"Design {entity} database schema"},
				Complexity:  3,
				Metadata:    map[string]string{"phase": "implementation"},
			},
			{
				Name:        "Implement {entity} repository",
				Type:        "implementation",
				Description: "Implement data access layer and repository pattern",
				Prerequisites: []string{"Create {entity} migrations"},
				Complexity:  5,
				Metadata:    map[string]string{"phase": "implementation"},
			},
		},
	}

	// User Interface Pattern
	td.patterns["user_interface"] = &TaskPattern{
		Name:        "User Interface",
		Description: "Pattern for UI component development",
		Matcher:     td.userInterfaceMatcher,
		Priority:    8,
		Category:    "frontend",
		Templates: []TaskTemplate{
			{
				Name:        "Design {component} mockups",
				Type:        "design",
				Description: "Create wireframes and visual mockups",
				Complexity:  3,
				Metadata:    map[string]string{"phase": "design"},
			},
			{
				Name:        "Implement {component} component",
				Type:        "implementation",
				Description: "Develop the UI component with styling",
				Prerequisites: []string{"Design {component} mockups"},
				Complexity:  5,
				Metadata:    map[string]string{"phase": "implementation"},
			},
			{
				Name:        "Test {component} interactions",
				Type:        "testing",
				Description: "Write interaction and accessibility tests",
				Prerequisites: []string{"Implement {component} component"},
				Complexity:  3,
				Metadata:    map[string]string{"phase": "testing"},
			},
		},
	}

	// Authentication Pattern
	td.patterns["authentication"] = &TaskPattern{
		Name:        "Authentication System",
		Description: "Pattern for user authentication implementation",
		Matcher:     td.authenticationMatcher,
		Priority:    9,
		Category:    "security",
		Templates: []TaskTemplate{
			{
				Name:        "Design authentication architecture",
				Type:        "design",
				Description: "Design authentication flow and security measures",
				Complexity:  5,
				Metadata:    map[string]string{"phase": "design", "security": "high"},
			},
			{
				Name:        "Implement user registration",
				Type:        "implementation",
				Description: "Implement user registration with validation",
				Prerequisites: []string{"Design authentication architecture"},
				Complexity:  5,
				Metadata:    map[string]string{"phase": "implementation"},
			},
			{
				Name:        "Implement user login",
				Type:        "implementation",
				Description: "Implement login with session management",
				Prerequisites: []string{"Implement user registration"},
				Complexity:  5,
				Metadata:    map[string]string{"phase": "implementation"},
			},
			{
				Name:        "Add password security",
				Type:        "implementation",
				Description: "Implement password hashing and security features",
				Prerequisites: []string{"Implement user login"},
				Complexity:  3,
				Metadata:    map[string]string{"phase": "implementation", "security": "high"},
			},
		},
	}

	// Testing Pattern
	td.patterns["testing_framework"] = &TaskPattern{
		Name:        "Testing Framework",
		Description: "Pattern for comprehensive testing setup",
		Matcher:     td.testingFrameworkMatcher,
		Priority:    7,
		Category:    "testing",
		Templates: []TaskTemplate{
			{
				Name:        "Setup testing framework",
				Type:        "implementation",
				Description: "Configure testing tools and environment",
				Complexity:  3,
				Metadata:    map[string]string{"phase": "setup"},
			},
			{
				Name:        "Write unit test suite",
				Type:        "testing",
				Description: "Create comprehensive unit tests",
				Prerequisites: []string{"Setup testing framework"},
				Complexity:  8,
				Metadata:    map[string]string{"phase": "testing"},
			},
			{
				Name:        "Setup integration tests",
				Type:        "testing",
				Description: "Create integration test framework",
				Prerequisites: []string{"Setup testing framework"},
				Complexity:  5,
				Metadata:    map[string]string{"phase": "testing"},
			},
		},
	}
}

// Pattern matchers

func (td *TaskDecomposer) apiEndpointMatcher(req Requirement) bool {
	content := strings.ToLower(req.Description)
	return containsAny(content, []string{"api", "endpoint", "rest", "http", "json", "route"})
}

func (td *TaskDecomposer) databaseSchemaMatcher(req Requirement) bool {
	content := strings.ToLower(req.Description)
	return containsAny(content, []string{"database", "table", "schema", "sql", "data model", "entity"})
}

func (td *TaskDecomposer) userInterfaceMatcher(req Requirement) bool {
	content := strings.ToLower(req.Description)
	return containsAny(content, []string{"ui", "interface", "component", "page", "form", "button", "frontend"})
}

func (td *TaskDecomposer) authenticationMatcher(req Requirement) bool {
	content := strings.ToLower(req.Description)
	return containsAny(content, []string{"auth", "login", "registration", "user", "password", "session", "security"})
}

func (td *TaskDecomposer) testingFrameworkMatcher(req Requirement) bool {
	content := strings.ToLower(req.Description)
	return containsAny(content, []string{"test", "testing", "qa", "quality", "validation", "coverage"})
}

// Core decomposition methods

func (td *TaskDecomposer) findMatchingPatterns(requirement Requirement) []*TaskPattern {
	var matches []*TaskPattern
	
	for _, pattern := range td.patterns {
		if pattern.Matcher(requirement) {
			matches = append(matches, pattern)
		}
	}
	
	return matches
}

func (td *TaskDecomposer) selectBestPattern(patterns []*TaskPattern) *TaskPattern {
	if len(patterns) == 0 {
		return nil
	}
	
	// Return the pattern with highest priority
	bestPattern := patterns[0]
	for _, pattern := range patterns[1:] {
		if pattern.Priority > bestPattern.Priority {
			bestPattern = pattern
		}
	}
	
	return bestPattern
}

func (td *TaskDecomposer) applyPattern(pattern *TaskPattern, requirement Requirement, taskCounter *int) []*RefinedTask {
	var tasks []*RefinedTask
	
	// Extract entity name from requirement
	entityName := td.extractEntityName(requirement.Description)
	
	for _, template := range pattern.Templates {
		task := td.createTaskFromTemplate(template, requirement, entityName, taskCounter)
		tasks = append(tasks, task)
	}
	
	return tasks
}

func (td *TaskDecomposer) createTaskFromTemplate(template TaskTemplate, requirement Requirement, entityName string, taskCounter *int) *RefinedTask {
	*taskCounter++
	
	// Replace placeholders in template
	title := strings.ReplaceAll(template.Name, "{endpoint}", entityName)
	title = strings.ReplaceAll(title, "{entity}", entityName)
	title = strings.ReplaceAll(title, "{component}", entityName)
	
	description := strings.ReplaceAll(template.Description, "{endpoint}", entityName)
	description = strings.ReplaceAll(description, "{entity}", entityName)
	description = strings.ReplaceAll(description, "{component}", entityName)
	
	task := &RefinedTask{
		ID:            fmt.Sprintf("task-%03d", *taskCounter),
		Title:         title,
		Description:   description,
		Type:          template.Type,
		Status:        "todo",
		Complexity:    template.Complexity,
		Dependencies:  make([]string, 0),
		Prerequisites: template.Prerequisites,
		Metadata:      make(map[string]string),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	// Copy template metadata
	for k, v := range template.Metadata {
		task.Metadata[k] = v
	}
	
	// Add requirement metadata
	task.Metadata["requirement_id"] = requirement.ID
	task.Metadata["requirement_type"] = requirement.Type
	task.Metadata["requirement_priority"] = requirement.Priority
	
	return task
}

func (td *TaskDecomposer) genericDecompose(requirement Requirement, taskCounter *int) []*RefinedTask {
	*taskCounter++
	
	// Create a generic implementation task
	task := &RefinedTask{
		ID:          fmt.Sprintf("task-%03d", *taskCounter),
		Title:       fmt.Sprintf("Implement %s", requirement.Description),
		Description: fmt.Sprintf("Implement the requirement: %s", requirement.Description),
		Type:        "implementation",
		Status:      "todo",
		Complexity:  3, // Default complexity
		Dependencies: make([]string, 0),
		Metadata:    map[string]string{
			"requirement_id":       requirement.ID,
			"requirement_type":     requirement.Type,
			"requirement_priority": requirement.Priority,
			"decomposition_type":   "generic",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	return []*RefinedTask{task}
}

func (td *TaskDecomposer) addDependencies(tasks []*RefinedTask) {
	// Create a map for quick task lookup
	taskMap := make(map[string]*RefinedTask)
	for _, task := range tasks {
		taskMap[task.Title] = task
	}
	
	// Process prerequisites into dependencies
	for _, task := range tasks {
		for _, prereq := range task.Prerequisites {
			// Find matching task by title pattern
			for _, otherTask := range tasks {
				if strings.Contains(prereq, otherTask.Title) || 
				   strings.Contains(otherTask.Title, prereq) {
					task.Dependencies = append(task.Dependencies, otherTask.ID)
					break
				}
			}
		}
		
		// Clear prerequisites as they're now converted to dependencies
		task.Prerequisites = []string{}
	}
}

func (td *TaskDecomposer) generateSetupTasks(taskCounter *int) []*RefinedTask {
	tasks := []*RefinedTask{}
	
	*taskCounter++
	setupTask := &RefinedTask{
		ID:          fmt.Sprintf("task-%03d", *taskCounter),
		Title:       "Project setup and initialization",
		Description: "Set up development environment, dependencies, and project structure",
		Type:        "implementation",
		Status:      "todo",
		Complexity:  2,
		Dependencies: make([]string, 0),
		Metadata: map[string]string{
			"phase": "setup",
			"category": "infrastructure",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	tasks = append(tasks, setupTask)
	return tasks
}

func (td *TaskDecomposer) generateCompletionTasks(taskCounter *int) []*RefinedTask {
	tasks := []*RefinedTask{}
	
	*taskCounter++
	deployTask := &RefinedTask{
		ID:          fmt.Sprintf("task-%03d", *taskCounter),
		Title:       "Deployment and final validation",
		Description: "Deploy the solution and perform final integration testing",
		Type:        "implementation",
		Status:      "todo",
		Complexity:  3,
		Dependencies: make([]string, 0), // Will depend on all other tasks
		Metadata: map[string]string{
			"phase": "completion",
			"category": "deployment",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	tasks = append(tasks, deployTask)
	return tasks
}

// Requirement analysis methods

func (ra *RequirementAnalyzer) extractBasicRequirements(commission *Commission) []Requirement {
	requirements := []Requirement{}
	reqCounter := 0
	
	// Extract from commission description
	if commission.Description != "" {
		reqCounter++
		req := Requirement{
			ID:          fmt.Sprintf("req-%03d", reqCounter),
			Type:        "functional",
			Priority:    "high",
			Description: commission.Description,
			Acceptance:  []string{},
			Metadata:    map[string]string{"source": "description"},
		}
		requirements = append(requirements, req)
	}
	
	// Extract from commission parts
	for _, part := range commission.Parts {
		if part.Type == "goal" || part.Type == "acceptance" || part.Type == "implementation" {
			reqCounter++
			req := Requirement{
				ID:          fmt.Sprintf("req-%03d", reqCounter),
				Type:        ra.determineRequirementType(part.Content),
				Priority:    ra.determinePriority(part.Content),
				Description: part.Content,
				Acceptance:  ra.extractAcceptanceCriteria(part.Content),
				Metadata: map[string]string{
					"source": "part",
					"part_type": part.Type,
					"part_title": part.Title,
				},
			}
			requirements = append(requirements, req)
		}
	}
	
	// Extract from existing commission requirements field
	for _, reqText := range commission.Requirements {
		reqCounter++
		req := Requirement{
			ID:          fmt.Sprintf("req-%03d", reqCounter),
			Type:        ra.determineRequirementType(reqText),
			Priority:    ra.determinePriority(reqText),
			Description: reqText,
			Acceptance:  []string{},
			Metadata:    map[string]string{"source": "requirements_field"},
		}
		requirements = append(requirements, req)
	}
	
	return requirements
}

func (ra *RequirementAnalyzer) aiEnhancedAnalysis(ctx context.Context, commission *Commission) (*Analysis, error) {
	// This would use the LLM client to perform AI-enhanced analysis
	// For now, return nil to indicate AI enhancement is not implemented
	// TODO: Implement AI-enhanced requirement analysis
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "AI-enhanced analysis not yet implemented", nil)
}

func (ra *RequirementAnalyzer) mergeRequirements(basic, enhanced []Requirement) []Requirement {
	// Simple merge - just append enhanced requirements
	// In production, this would deduplicate and merge intelligently
	return append(basic, enhanced...)
}

func (ra *RequirementAnalyzer) extractTechnicalStack(commission *Commission) []string {
	stack := []string{}
	content := strings.ToLower(commission.Description + " " + commission.Goal)
	
	// Add commission content from all parts
	for _, part := range commission.Parts {
		content += " " + strings.ToLower(part.Content)
	}
	
	// Common technology keywords
	technologies := map[string]string{
		"go":         "Go",
		"golang":     "Go",
		"javascript": "JavaScript",
		"react":      "React",
		"vue":        "Vue.js",
		"angular":    "Angular",
		"node":       "Node.js",
		"python":     "Python",
		"java":       "Java",
		"sql":        "SQL",
		"postgres":   "PostgreSQL",
		"mysql":      "MySQL",
		"mongodb":    "MongoDB",
		"redis":      "Redis",
		"docker":     "Docker",
		"kubernetes": "Kubernetes",
		"aws":        "AWS",
		"gcp":        "Google Cloud",
		"azure":      "Azure",
	}
	
	for keyword, tech := range technologies {
		if strings.Contains(content, keyword) {
			stack = append(stack, tech)
		}
	}
	
	return stack
}

func (ra *RequirementAnalyzer) determineScope(commission *Commission, requirements []Requirement) string {
	// Simple heuristic based on requirement count and content
	reqCount := len(requirements)
	contentLength := len(commission.Description)
	
	if reqCount <= 3 && contentLength < 500 {
		return "small"
	} else if reqCount <= 8 && contentLength < 1500 {
		return "medium"
	} else {
		return "large"
	}
}

func (ra *RequirementAnalyzer) estimateEffort(scope string, reqCount int) string {
	switch scope {
	case "small":
		return "1-3 days"
	case "medium":
		return "1-2 weeks"
	case "large":
		return "2-4 weeks"
	default:
		return "2-4 weeks"
	}
}

func (ra *RequirementAnalyzer) extractSuccessCriteria(commission *Commission) []string {
	criteria := []string{}
	
	// Look for acceptance criteria in parts
	for _, part := range commission.Parts {
		if part.Type == "acceptance" || strings.Contains(strings.ToLower(part.Title), "acceptance") {
			criteria = append(criteria, part.Content)
		}
	}
	
	// Default criteria if none found
	if len(criteria) == 0 {
		criteria = append(criteria, "All functional requirements implemented and tested")
		criteria = append(criteria, "Code passes quality review and testing")
		criteria = append(criteria, "Documentation is complete and accurate")
	}
	
	return criteria
}

func (ra *RequirementAnalyzer) extractDeliverables(commission *Commission) []string {
	deliverables := []string{}
	
	// Extract from goal and description
	content := strings.ToLower(commission.Goal + " " + commission.Description)
	
	commonDeliverables := map[string]string{
		"api":           "REST API implementation",
		"database":      "Database schema and data layer",
		"ui":            "User interface components",
		"documentation": "Technical documentation",
		"tests":         "Test suite and quality assurance",
		"deployment":    "Deployment configuration",
	}
	
	for keyword, deliverable := range commonDeliverables {
		if strings.Contains(content, keyword) {
			deliverables = append(deliverables, deliverable)
		}
	}
	
	// Default deliverables
	if len(deliverables) == 0 {
		deliverables = append(deliverables, "Working software implementation")
		deliverables = append(deliverables, "Source code and documentation")
	}
	
	return deliverables
}

func (ra *RequirementAnalyzer) identifyRiskFactors(commission *Commission, requirements []Requirement) []string {
	risks := []string{}
	
	// Analyze complexity
	if len(requirements) > 10 {
		risks = append(risks, "High complexity - many requirements to coordinate")
	}
	
	// Check for technology risks
	content := strings.ToLower(commission.Description)
	if strings.Contains(content, "new") || strings.Contains(content, "experimental") {
		risks = append(risks, "Technology risk - new or experimental technologies")
	}
	
	// Check for integration risks
	if strings.Contains(content, "integrate") || strings.Contains(content, "third-party") {
		risks = append(risks, "Integration risk - external dependencies")
	}
	
	// Check for security requirements
	if strings.Contains(content, "security") || strings.Contains(content, "auth") {
		risks = append(risks, "Security requirements - additional validation needed")
	}
	
	return risks
}

func (ra *RequirementAnalyzer) determineRequirementType(content string) string {
	contentLower := strings.ToLower(content)
	
	if containsAny(contentLower, []string{"performance", "scalability", "security", "reliability"}) {
		return "non-functional"
	} else if containsAny(contentLower, []string{"technology", "framework", "database", "architecture"}) {
		return "technical"
	} else {
		return "functional"
	}
}

func (ra *RequirementAnalyzer) determinePriority(content string) string {
	contentLower := strings.ToLower(content)
	
	if containsAny(contentLower, []string{"critical", "essential", "must", "required"}) {
		return "high"
	} else if containsAny(contentLower, []string{"should", "important", "preferred"}) {
		return "medium"
	} else {
		return "low"
	}
}

func (ra *RequirementAnalyzer) extractAcceptanceCriteria(content string) []string {
	criteria := []string{}
	
	// Look for bullet points or numbered lists
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") || 
		   regexp.MustCompile(`^\d+\.`).MatchString(trimmed) {
			criteria = append(criteria, strings.TrimSpace(strings.TrimPrefix(
				strings.TrimPrefix(trimmed, "-"), "*")))
		}
	}
	
	return criteria
}

func (td *TaskDecomposer) extractEntityName(description string) string {
	// Simple entity name extraction - look for key nouns
	words := strings.Fields(strings.ToLower(description))
	
	// Common entity patterns
	entities := []string{"user", "product", "order", "payment", "account", "task", "project"}
	
	for _, word := range words {
		for _, entity := range entities {
			if strings.Contains(word, entity) {
				return entity
			}
		}
	}
	
	// Default to "item" if no entity found
	return "item"
}

// Utility functions

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}