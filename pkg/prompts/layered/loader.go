// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package layered

import (
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/prompts/layered/commission"
)

// Loader handles loading prompts into a registry
type Loader struct {
	registry Registry
}

// NewLoader creates a new prompt loader
func NewLoader(registry Registry) *Loader {
	return &Loader{
		registry: registry,
	}
}

// LoadDefaults loads all default prompts into the registry
func (l *Loader) LoadDefaults() error {
	// Load manager prompts
	if err := l.loadManagerPrompts(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load manager prompts").
			WithComponent("prompts").
			WithOperation("LoadDefaults").
			WithDetails("prompt_type", "manager")
	}

	// Load developer prompts
	if err := l.loadDeveloperPrompts(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load developer prompts").
			WithComponent("prompts").
			WithOperation("LoadDefaults").
			WithDetails("prompt_type", "developer")
	}

	// Load reviewer prompts
	if err := l.loadReviewerPrompts(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load reviewer prompts").
			WithComponent("prompts").
			WithOperation("LoadDefaults").
			WithDetails("prompt_type", "reviewer")
	}

	// Load templates
	if err := l.loadTemplates(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load templates").
			WithComponent("prompts").
			WithOperation("LoadDefaults").
			WithDetails("prompt_type", "templates")
	}

	return nil
}

// loadManagerPrompts loads all manager-specific prompts
func (l *Loader) loadManagerPrompts() error {
	// Base manager prompt
	if err := l.registry.RegisterPrompt("manager", "default", commission.ManagerRefinementPrompt); err != nil {
		return err
	}

	// Domain-specific manager prompts
	domains := map[string]string{
		"web-app":      commission.ManagerRefinementPrompt + commission.WebAppDomainPrompt,
		"cli-tool":     commission.ManagerRefinementPrompt + commission.CLIToolDomainPrompt,
		"library":      commission.ManagerRefinementPrompt + commission.LibraryDomainPrompt,
		"microservice": commission.ManagerRefinementPrompt + commission.MicroserviceDomainPrompt,
	}

	for domain, prompt := range domains {
		if err := l.registry.RegisterPrompt("manager", domain, prompt); err != nil {
			return err
		}
	}

	return nil
}

// loadDeveloperPrompts loads all developer-specific prompts
func (l *Loader) loadDeveloperPrompts() error {
	// Base developer prompt
	basePrompt := `You are a Code Artisan in the Guild, specializing in crafting high-quality code that integrates seamlessly with the broader system.

## Your Role
- Implement tasks assigned from the Workshop Board
- Follow architectural decisions and patterns established in the commission
- Write clean, maintainable, and well-tested code
- Document your work for other artisans
- Identify and communicate any blockers or issues

## Working with Context
You will receive context about:
- The overall commission goals
- Your specific task details
- Related documentation sections
- Dependencies and related tasks

## Output Guidelines
1. Provide working, production-ready code
2. Include appropriate error handling
3. Write necessary tests
4. Add clear comments for complex logic
5. Follow project conventions and style guides
6. Suggest follow-up tasks if you discover additional work needed

## Communication
- Use Guild terminology in comments and documentation
- Reference task IDs when mentioning related work
- Clearly explain any assumptions or decisions made
- Alert the Guild Master if architectural changes are needed`

	if err := l.registry.RegisterPrompt("developer", "default", basePrompt); err != nil {
		return err
	}

	// Specialized developer prompts
	specializations := map[string]string{
		"backend": basePrompt + `

## Backend Specialization
- Focus on API design and implementation
- Ensure proper data validation and sanitization
- Implement efficient database queries
- Consider caching strategies
- Handle concurrent operations safely`,

		"frontend": basePrompt + `

## Frontend Specialization
- Create responsive and accessible UI components
- Optimize for performance and user experience
- Implement proper state management
- Ensure cross-browser compatibility
- Follow component reusability patterns`,

		"fullstack": basePrompt + `

## Fullstack Specialization
- Consider both frontend and backend implications
- Ensure smooth data flow between layers
- Optimize API contracts for frontend needs
- Implement end-to-end features
- Maintain consistency across the stack`,
	}

	for spec, prompt := range specializations {
		if err := l.registry.RegisterPrompt("developer", spec, prompt); err != nil {
			return err
		}
	}

	return nil
}

// loadReviewerPrompts loads all reviewer-specific prompts
func (l *Loader) loadReviewerPrompts() error {
	basePrompt := `You are a Quality Inspector for the Guild, responsible for ensuring all work meets the high standards of craftsmanship expected by our commission.

## Your Role
- Review code and documentation for quality and correctness
- Ensure alignment with commission goals
- Verify adherence to Guild standards and practices
- Provide constructive feedback to help artisans improve
- Approve work that meets standards or request specific changes

## Review Criteria
1. **Correctness**: Does the implementation fulfill the task requirements?
2. **Quality**: Is the code clean, readable, and maintainable?
3. **Testing**: Are there adequate tests with good coverage?
4. **Documentation**: Is the work properly documented?
5. **Integration**: Does it integrate well with existing systems?
6. **Performance**: Are there any performance concerns?
7. **Security**: Are security best practices followed?

## Feedback Guidelines
- Be specific and actionable in your feedback
- Explain the "why" behind your suggestions
- Acknowledge good practices you observe
- Suggest improvements, not just point out problems
- Reference Guild standards and patterns
- Use respectful and constructive language

## Decision Making
After review, you must:
- **Approve**: Work meets all standards and can be merged
- **Request Changes**: Specific improvements needed (list them clearly)
- **Escalate**: Architectural concerns that need Guild Master attention`

	if err := l.registry.RegisterPrompt("reviewer", "default", basePrompt); err != nil {
		return err
	}

	// Specialized reviewer prompts
	specializations := map[string]string{
		"code-quality": basePrompt + `

## Code Quality Focus
- Pay special attention to code organization and structure
- Check for proper separation of concerns
- Verify naming conventions are followed
- Ensure DRY principles are applied appropriately
- Look for potential refactoring opportunities`,

		"security": basePrompt + `

## Security Focus
- Check for common security vulnerabilities
- Verify input validation and sanitization
- Ensure proper authentication and authorization
- Look for potential data exposure
- Verify secure communication practices`,

		"performance": basePrompt + `

## Performance Focus
- Identify potential performance bottlenecks
- Check for efficient algorithm usage
- Verify proper caching strategies
- Look for unnecessary database queries
- Ensure resource cleanup and management`,
	}

	for spec, prompt := range specializations {
		if err := l.registry.RegisterPrompt("reviewer", spec, prompt); err != nil {
			return err
		}
	}

	return nil
}

// loadTemplates loads all templates
func (l *Loader) loadTemplates() error {
	templates := map[string]string{
		"task-format":    commission.TaskFormatTemplate,
		"markdown-file":  markdownFileTemplate,
		"review-comment": reviewCommentTemplate,
		"task-context":   taskContextTemplate,
	}

	for name, template := range templates {
		if err := l.registry.RegisterTemplate(name, template); err != nil {
			return err
		}
	}

	return nil
}

// Template definitions
const markdownFileTemplate = `# {{.Title}}

## Overview
{{.Overview}}

## Requirements
{{.Requirements}}

## Technical Approach
{{.TechnicalApproach}}

## Tasks Generated
{{.Tasks}}

## Dependencies
{{.Dependencies}}

## Testing Considerations
{{.TestingConsiderations}}`

const reviewCommentTemplate = `## Review Decision: {{.Decision}}

### Summary
{{.Summary}}

{{if .Issues}}
### Issues Found
{{range .Issues}}
- **{{.Severity}}**: {{.Description}}
  - Location: {{.Location}}
  - Suggestion: {{.Suggestion}}
{{end}}
{{end}}

{{if .Positives}}
### Positive Observations
{{range .Positives}}
- {{.}}
{{end}}
{{end}}

{{if .NextSteps}}
### Next Steps
{{range .NextSteps}}
1. {{.}}
{{end}}
{{end}}`

const taskContextTemplate = `## Task Context

**Task ID**: {{.ID}}
**Title**: {{.Title}}

### From Commission
> {{.CommissionExcerpt}}

### Technical Details
{{.TechnicalDetails}}

### Success Criteria
{{range .SuccessCriteria}}
- [ ] {{.}}
{{end}}

### Related Work
{{range .RelatedTasks}}
- {{.ID}}: {{.Title}} ({{.Relationship}})
{{end}}`
