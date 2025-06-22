// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package templates

import (
	"time"

	"github.com/google/uuid"
)

// GetDefaultTemplates returns a collection of built-in templates for common Guild tasks
func GetDefaultTemplates() []*Template {
	now := time.Now()

	return []*Template{
		// Code Review Templates
		{
			ID:          uuid.New().String(),
			Name:        "Code Review Request",
			Description: "Request comprehensive code review with specific focus areas",
			Category:    "code-review",
			Language:    "",
			Content: `Please review this {{language}} code for {{focus_area|title}}:

` + "```{{language}}" + `
{{code_block}}
` + "```" + `

**Review Focus:**
- {{focus_area|title}}
- Code quality and best practices
- Performance considerations
- Security implications

**Context:**
{{context:No additional context provided}}

**Specific Questions:**
{{questions:Any specific concerns or questions about the implementation}}

Please provide:
1. Overall assessment
2. Specific issues or improvements
3. Recommendations for optimization
4. Any security concerns`,
			IsBuiltIn: true,
			CreatedAt: now,
			UpdatedAt: now,
			Variables: []*TemplateVariable{
				{
					ID:          uuid.New().String(),
					Name:        "language",
					Description: "Programming language (e.g., Go, Python, JavaScript)",
					Required:    true,
					Type:        VariableTypeSelect,
					Options:     []string{"Go", "Python", "JavaScript", "TypeScript", "Rust", "Java", "C++"},
					CreatedAt:   now,
				},
				{
					ID:          uuid.New().String(),
					Name:        "code_block",
					Description: "The code to be reviewed",
					Required:    true,
					Type:        VariableTypeCode,
					CreatedAt:   now,
				},
				{
					ID:          uuid.New().String(),
					Name:        "focus_area",
					Description: "Primary focus for the review",
					Required:    true,
					Type:        VariableTypeSelect,
					Options:     []string{"performance", "security", "readability", "architecture", "testing", "error handling"},
					CreatedAt:   now,
				},
				{
					ID:           uuid.New().String(),
					Name:         "context",
					Description:  "Additional context about the code",
					DefaultValue: "No additional context provided",
					Required:     false,
					Type:         VariableTypeMultiline,
					CreatedAt:    now,
				},
				{
					ID:           uuid.New().String(),
					Name:         "questions",
					Description:  "Specific questions about the implementation",
					DefaultValue: "Any specific concerns or questions about the implementation",
					Required:     false,
					Type:         VariableTypeMultiline,
					CreatedAt:    now,
				},
			},
		},

		// Bug Investigation Template
		{
			ID:          uuid.New().String(),
			Name:        "Bug Investigation",
			Description: "Systematic approach to investigating and debugging issues",
			Category:    "debugging",
			Content: `I need help investigating a {{severity|lower}} bug in {{component}}:

**Bug Description:**
{{description}}

**Expected Behavior:**
{{expected}}

**Actual Behavior:**
{{actual}}

**Steps to Reproduce:**
{{steps:1. [Describe steps to reproduce]}}

**Environment:**
- Platform: {{platform:Unknown}}
- Version: {{version:Unknown}}
- Configuration: {{config:Default}}

**Error Messages/Logs:**
` + "```" + `
{{logs:No logs available}}
` + "```" + `

**Debugging Strategy Needed:**
1. Root cause analysis
2. Potential solutions
3. Prevention strategies
4. Testing approach

**Priority:** {{severity|upper}}`,
			IsBuiltIn: true,
			CreatedAt: now,
			UpdatedAt: now,
			Variables: []*TemplateVariable{
				{
					ID:          uuid.New().String(),
					Name:        "severity",
					Description: "Bug severity level",
					Required:    true,
					Type:        VariableTypeSelect,
					Options:     []string{"critical", "high", "medium", "low"},
					CreatedAt:   now,
				},
				{
					ID:          uuid.New().String(),
					Name:        "component",
					Description: "Component or system where bug occurs",
					Required:    true,
					Type:        VariableTypeText,
					CreatedAt:   now,
				},
				{
					ID:          uuid.New().String(),
					Name:        "description",
					Description: "Detailed description of the bug",
					Required:    true,
					Type:        VariableTypeMultiline,
					CreatedAt:   now,
				},
				{
					ID:          uuid.New().String(),
					Name:        "expected",
					Description: "What should happen",
					Required:    true,
					Type:        VariableTypeMultiline,
					CreatedAt:   now,
				},
				{
					ID:          uuid.New().String(),
					Name:        "actual",
					Description: "What actually happens",
					Required:    true,
					Type:        VariableTypeMultiline,
					CreatedAt:   now,
				},
			},
		},

		// Architecture Design Template
		{
			ID:          uuid.New().String(),
			Name:        "Architecture Design",
			Description: "Template for designing system architecture and technical specifications",
			Category:    "architecture",
			Content: `I need to design the architecture for {{system_name}}:

**System Overview:**
{{overview}}

**Requirements:**
{{requirements}}

**Scale and Performance:**
- Expected users: {{users:Unknown}}
- Expected load: {{load:Unknown}}
- Performance requirements: {{performance:Standard web application performance}}

**Technical Constraints:**
{{constraints:No specific constraints}}

**Architecture Decisions Needed:**
1. **Technology Stack**
   - Backend: {{backend_tech:To be determined}}
   - Frontend: {{frontend_tech:To be determined}}
   - Database: {{database_tech:To be determined}}

2. **System Components**
   - Core services
   - Data storage strategy
   - API design
   - Authentication/Authorization

3. **Non-Functional Requirements**
   - Security considerations
   - Scalability approach
   - Monitoring and observability
   - Deployment strategy

**Deliverables Requested:**
- High-level architecture diagram
- Component interaction flows
- Technology recommendations
- Implementation roadmap
- Risk assessment`,
			IsBuiltIn: true,
			CreatedAt: now,
			UpdatedAt: now,
			Variables: []*TemplateVariable{
				{
					ID:          uuid.New().String(),
					Name:        "system_name",
					Description: "Name of the system being designed",
					Required:    true,
					Type:        VariableTypeText,
					CreatedAt:   now,
				},
				{
					ID:          uuid.New().String(),
					Name:        "overview",
					Description: "High-level description of the system",
					Required:    true,
					Type:        VariableTypeMultiline,
					CreatedAt:   now,
				},
				{
					ID:          uuid.New().String(),
					Name:        "requirements",
					Description: "Functional requirements for the system",
					Required:    true,
					Type:        VariableTypeMultiline,
					CreatedAt:   now,
				},
			},
		},

		// API Development Template
		{
			ID:          uuid.New().String(),
			Name:        "API Development",
			Description: "Template for designing and implementing REST APIs",
			Category:    "architecture",
			Content: `I need to develop a {{api_type}} API for {{purpose}}:

**API Overview:**
{{overview}}

**Endpoints Required:**
{{endpoints}}

**Data Models:**
{{models:Please describe the data structures needed}}

**Authentication:**
{{auth:JWT token-based authentication}}

**Requirements:**
- HTTP method: {{method|upper}}
- Response format: {{format:JSON}}
- Error handling: {{error_handling:Standard HTTP status codes with error messages}}
- Rate limiting: {{rate_limit:100 requests per minute}}

**Example Request:**
` + "```http" + `
{{method|upper}} /api/{{endpoint}}
Content-Type: application/json
Authorization: Bearer {{token:your-jwt-token}}

{{request_body:{}}}
` + "```" + `

**Expected Response:**
` + "```json" + `
{{response_body:{"success": true, "data": {}}}
` + "```" + `

**Implementation Notes:**
{{notes:No additional implementation notes}}`,
			IsBuiltIn: true,
			CreatedAt: now,
			UpdatedAt: now,
			Variables: []*TemplateVariable{
				{
					ID:          uuid.New().String(),
					Name:        "api_type",
					Description: "Type of API (REST, GraphQL, etc.)",
					Required:    true,
					Type:        VariableTypeSelect,
					Options:     []string{"REST", "GraphQL", "gRPC", "WebSocket"},
					CreatedAt:   now,
				},
				{
					ID:          uuid.New().String(),
					Name:        "purpose",
					Description: "Purpose or functionality of the API",
					Required:    true,
					Type:        VariableTypeText,
					CreatedAt:   now,
				},
				{
					ID:           uuid.New().String(),
					Name:         "method",
					Description:  "HTTP method",
					DefaultValue: "GET",
					Required:     false,
					Type:         VariableTypeSelect,
					Options:      []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
					CreatedAt:    now,
				},
			},
		},

		// Documentation Template
		{
			ID:          uuid.New().String(),
			Name:        "Technical Documentation",
			Description: "Template for creating comprehensive technical documentation",
			Category:    "documentation",
			Content: `# {{title}}

## Overview
{{overview}}

## Purpose
{{purpose}}

## {{section_type|title}} Details

### Key Features
{{features}}

### {{usage_type|title}}
{{usage}}

### Configuration
{{configuration:No configuration required}}

### Examples

#### Example 1: {{example_title:Basic Usage}}
` + "```{{language:bash}}" + `
{{example_code}}
` + "```" + `

{{example_explanation:This example demonstrates basic usage}}

### API Reference
{{api_reference:No API reference available}}

### Troubleshooting
{{troubleshooting:Contact support for issues}}

## Related Documentation
{{related:No related documentation}}

---
*Last updated: {{date}}*
*Maintainer: {{maintainer:Development Team}}*`,
			IsBuiltIn: true,
			CreatedAt: now,
			UpdatedAt: now,
			Variables: []*TemplateVariable{
				{
					ID:          uuid.New().String(),
					Name:        "title",
					Description: "Document title",
					Required:    true,
					Type:        VariableTypeText,
					CreatedAt:   now,
				},
				{
					ID:           uuid.New().String(),
					Name:         "section_type",
					Description:  "Type of section (Implementation, Usage, Configuration, etc.)",
					DefaultValue: "Implementation",
					Required:     false,
					Type:         VariableTypeText,
					CreatedAt:    now,
				},
				{
					ID:           uuid.New().String(),
					Name:         "usage_type",
					Description:  "Type of usage section",
					DefaultValue: "Usage",
					Required:     false,
					Type:         VariableTypeText,
					CreatedAt:    now,
				},
			},
		},

		// Test Planning Template
		{
			ID:          uuid.New().String(),
			Name:        "Test Planning",
			Description: "Template for planning comprehensive test strategies",
			Category:    "testing",
			Content: `# Test Plan for {{feature_name}}

## Test Objectives
{{objectives}}

## Scope
**In Scope:**
{{in_scope}}

**Out of Scope:**
{{out_scope:No specific exclusions}}

## Test Strategy

### Test Types
- [ ] Unit Tests
- [ ] Integration Tests  
- [ ] End-to-End Tests
- [ ] Performance Tests
- [ ] Security Tests

### Test Environment
{{environment:Development environment}}

### Test Data
{{test_data:Standard test dataset}}

## Test Cases

### {{test_category|title}} Tests
{{test_cases}}

### Edge Cases
{{edge_cases:Standard boundary conditions}}

### Error Scenarios
{{error_scenarios:Standard error handling tests}}

## Acceptance Criteria
{{acceptance_criteria}}

## Risk Assessment
{{risks:Low risk - standard testing approach}}

## Timeline
- Test development: {{dev_timeline:TBD}}
- Test execution: {{exec_timeline:TBD}}
- Bug fixing: {{fix_timeline:TBD}}`,
			IsBuiltIn: true,
			CreatedAt: now,
			UpdatedAt: now,
			Variables: []*TemplateVariable{
				{
					ID:          uuid.New().String(),
					Name:        "feature_name",
					Description: "Name of the feature being tested",
					Required:    true,
					Type:        VariableTypeText,
					CreatedAt:   now,
				},
				{
					ID:           uuid.New().String(),
					Name:         "test_category",
					Description:  "Primary category of tests",
					DefaultValue: "functional",
					Required:     false,
					Type:         VariableTypeSelect,
					Options:      []string{"functional", "performance", "security", "usability", "compatibility"},
					CreatedAt:    now,
				},
			},
		},

		// AI Prompting Template
		{
			ID:          uuid.New().String(),
			Name:        "Effective AI Prompting",
			Description: "Template for creating effective prompts for AI assistance",
			Category:    "prompting",
			Content: `I need help with {{task_type}} for {{project_context}}.

**Context:**
{{context}}

**Specific Requirements:**
{{requirements}}

**Constraints:**
{{constraints:No specific constraints}}

**Expected Output:**
{{expected_output}}

**Success Criteria:**
{{success_criteria}}

**Additional Information:**
{{additional_info:No additional information}}

**Preferred Approach:**
{{approach:Please suggest the best approach}}

Please provide:
1. Step-by-step solution
2. Code examples (if applicable)
3. Best practices recommendations
4. Potential pitfalls to avoid
5. Testing/validation approach`,
			IsBuiltIn: true,
			CreatedAt: now,
			UpdatedAt: now,
			Variables: []*TemplateVariable{
				{
					ID:          uuid.New().String(),
					Name:        "task_type",
					Description: "Type of task (development, design, analysis, etc.)",
					Required:    true,
					Type:        VariableTypeSelect,
					Options:     []string{"development", "debugging", "design", "analysis", "optimization", "documentation", "testing"},
					CreatedAt:   now,
				},
				{
					ID:          uuid.New().String(),
					Name:        "project_context",
					Description: "Context or domain of the project",
					Required:    true,
					Type:        VariableTypeText,
					CreatedAt:   now,
				},
			},
		},

		// Quick Bug Report Template
		{
			ID:          uuid.New().String(),
			Name:        "Quick Bug Report",
			Description: "Simple template for reporting bugs quickly",
			Category:    "debugging",
			Content: `**Bug:** {{title}}

**What happened:** {{description}}

**Expected:** {{expected:Working as intended}}

**Environment:** {{environment:Default}}

{{logs:No additional logs}}`,
			IsBuiltIn: true,
			CreatedAt: now,
			UpdatedAt: now,
			Variables: []*TemplateVariable{
				{
					ID:          uuid.New().String(),
					Name:        "title",
					Description: "Brief title describing the bug",
					Required:    true,
					Type:        VariableTypeText,
					CreatedAt:   now,
				},
				{
					ID:          uuid.New().String(),
					Name:        "description",
					Description: "What happened (actual behavior)",
					Required:    true,
					Type:        VariableTypeMultiline,
					CreatedAt:   now,
				},
			},
		},

		// Feature Request Template
		{
			ID:          uuid.New().String(),
			Name:        "Feature Request",
			Description: "Template for requesting new features or enhancements",
			Category:    "general",
			Content: `# Feature Request: {{feature_name}}

## Description
{{description}}

## Use Case
{{use_case}}

## Proposed Solution
{{solution:Open to suggestions}}

## Benefits
{{benefits}}

## Priority: {{priority|upper}}

## Additional Context
{{context:No additional context}}`,
			IsBuiltIn: true,
			CreatedAt: now,
			UpdatedAt: now,
			Variables: []*TemplateVariable{
				{
					ID:          uuid.New().String(),
					Name:        "feature_name",
					Description: "Name of the requested feature",
					Required:    true,
					Type:        VariableTypeText,
					CreatedAt:   now,
				},
				{
					ID:           uuid.New().String(),
					Name:         "priority",
					Description:  "Priority level",
					DefaultValue: "medium",
					Required:     false,
					Type:         VariableTypeSelect,
					Options:      []string{"low", "medium", "high", "critical"},
					CreatedAt:    now,
				},
			},
		},
	}
}
