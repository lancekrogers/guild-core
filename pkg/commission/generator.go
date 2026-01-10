// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commission

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/lancekrogers/guild-core/pkg/agents/core/elena"
	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// Generator is responsible for generating objectives
type Generator struct {
	manager   *Manager
	templates map[string]*template.Template
}

// newGenerator creates a new objective generator (private constructor)
func newGenerator(manager *Manager) *Generator {
	g := &Generator{
		manager:   manager,
		templates: make(map[string]*template.Template),
	}
	g.initTemplates()
	return g
}

// DefaultGeneratorFactory creates a generator factory for registry use
func DefaultGeneratorFactory(manager *Manager) *Generator {
	return newGenerator(manager)
}

// GenerateFromPrompt generates an objective from a user prompt
func (g *Generator) GenerateFromPrompt(ctx context.Context, prompt string) (*Commission, error) {
	// In a real implementation, this would use an LLM to generate a commission
	// For now, we'll create a simple commission based on the prompt
	obj := &Commission{
		Title:       prompt,
		Description: "Auto-generated objective",
		Status:      CommissionStatusDraft,
		Parts:       []*CommissionPart{},
	}

	return obj, nil
}

// GenerateFromDialogue generates a commission from Elena's planning dialogue
func (g *Generator) GenerateFromDialogue(ctx context.Context, dialogue *elena.PlanningDialogue) (*Commission, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("commission.generator").
			WithOperation("GenerateFromDialogue")
	}

	if !dialogue.IsComplete() {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "dialogue is not complete", nil).
			WithComponent("commission.generator").
			WithOperation("GenerateFromDialogue")
	}

	// Get responses and context
	responses := dialogue.GetResponses()
	dialogueCtx := dialogue.GetContext()

	// Determine project type for template selection
	projectType := g.determineProjectType(responses, dialogueCtx)

	// Select appropriate template
	tmpl := g.selectTemplate(projectType)

	// Prepare template data
	templateData := g.prepareTemplateData(responses, dialogueCtx)

	// Generate commission content
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to execute template").
			WithComponent("commission.generator").
			WithOperation("GenerateFromDialogue")
	}

	// Create commission structure
	commission := &Commission{
		ID:          g.generateCommissionID(),
		Title:       g.generateTitle(responses),
		Description: g.generateDescription(responses),
		Status:      CommissionStatusDraft,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Tags:        g.generateTags(responses, dialogueCtx),
		Content:     buf.String(),
		Metadata: map[string]string{
			"created_via": "chat",
			"dialogue_id": dialogue.ID,
			"agent":       "elena-guild-master",
		},
		Goal:         g.extractGoal(responses),
		Requirements: g.extractRequirements(responses),
		Priority:     g.determinePriority(responses),
		Owner:        "user", // Set by chat context
		Parts:        g.generateParts(responses, dialogueCtx),
	}

	return commission, nil
}

// SaveGeneratedCommission saves a generated commission
func (g *Generator) SaveGeneratedCommission(ctx context.Context, obj *Commission) error {
	if obj == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "commission is nil", nil).
			WithComponent("commission.generator").
			WithOperation("SaveGeneratedCommission")
	}

	// Save the commission using the manager
	return g.manager.SaveCommission(ctx, obj)
}

// initTemplates initializes commission templates
func (g *Generator) initTemplates() {
	// Software project template
	g.templates["software"] = template.Must(template.New("software").Parse(softwareCommissionTemplate))

	// Research project template
	g.templates["research"] = template.Must(template.New("research").Parse(researchCommissionTemplate))

	// Default template
	g.templates["default"] = template.Must(template.New("default").Parse(defaultCommissionTemplate))
}

// Helper methods

func (g *Generator) determineProjectType(responses map[string]string, ctx map[string]interface{}) string {
	purpose := responses["project_purpose"]

	if strings.Contains(strings.ToLower(purpose), "build") || strings.Contains(strings.ToLower(purpose), "software") {
		return "software"
	} else if strings.Contains(strings.ToLower(purpose), "research") {
		return "research"
	}

	return "default"
}

func (g *Generator) selectTemplate(projectType string) *template.Template {
	if tmpl, ok := g.templates[projectType]; ok {
		return tmpl
	}
	return g.templates["default"]
}

func (g *Generator) prepareTemplateData(responses map[string]string, ctx map[string]interface{}) map[string]interface{} {
	data := make(map[string]interface{})

	// Copy all responses
	for k, v := range responses {
		data[k] = v
	}

	// Add context
	data["context"] = ctx

	// Add formatted sections
	data["formatted_requirements"] = g.formatRequirements(responses["requirements"])
	data["formatted_technology"] = g.formatTechnology(responses["technology"])
	data["formatted_constraints"] = g.formatConstraints(responses["constraints"])

	return data
}

func (g *Generator) generateCommissionID() string {
	// Generate a unique ID based on timestamp and random component
	timestamp := time.Now().Unix()
	return fmt.Sprintf("comm_%d_%s", timestamp, g.generateRandomSuffix(6))
}

func (g *Generator) generateRandomSuffix(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

func (g *Generator) generateTitle(responses map[string]string) string {
	desc := responses["project_description"]

	// Extract key words for title
	words := strings.Fields(desc)
	if len(words) > 8 {
		return strings.Join(words[:8], " ")
	}

	return desc
}

func (g *Generator) generateDescription(responses map[string]string) string {
	purpose := responses["project_purpose"]
	projectType := responses["project_type"]

	return fmt.Sprintf("%s - %s", purpose, projectType)
}

func (g *Generator) generateTags(responses map[string]string, ctx map[string]interface{}) []string {
	tags := []string{}

	// Add purpose tag
	if purpose, ok := ctx["purpose_category"].(string); ok {
		switch purpose {
		case "build_software":
			tags = append(tags, "software", "development")
		case "deep_research":
			tags = append(tags, "research", "analysis")
		case "improve_existing":
			tags = append(tags, "refactoring", "optimization")
		}
	}

	// Add technology tags
	tech := strings.ToLower(responses["technology"])
	if strings.Contains(tech, "go") || strings.Contains(tech, "golang") {
		tags = append(tags, "golang")
	}
	if strings.Contains(tech, "python") {
		tags = append(tags, "python")
	}
	if strings.Contains(tech, "typescript") || strings.Contains(tech, "javascript") {
		tags = append(tags, "javascript")
	}
	if strings.Contains(tech, "api") {
		tags = append(tags, "api")
	}
	if strings.Contains(tech, "web") {
		tags = append(tags, "web")
	}

	return tags
}

func (g *Generator) extractGoal(responses map[string]string) string {
	return responses["project_description"]
}

func (g *Generator) extractRequirements(responses map[string]string) []string {
	reqs := responses["requirements"]
	lines := strings.Split(reqs, "\n")

	requirements := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		if line != "" {
			requirements = append(requirements, line)
		}
	}

	return requirements
}

func (g *Generator) determinePriority(responses map[string]string) string {
	timeline := strings.ToLower(responses["timeline"])

	if strings.Contains(timeline, "urgent") || strings.Contains(timeline, "asap") || strings.Contains(timeline, "week") {
		return "high"
	} else if strings.Contains(timeline, "month") {
		return "medium"
	}

	return "low"
}

func (g *Generator) generateParts(responses map[string]string, ctx map[string]interface{}) []*CommissionPart {
	parts := []*CommissionPart{}

	// Context part
	parts = append(parts, &CommissionPart{
		Title: "Context",
		Content: fmt.Sprintf(`The user wants to %s. They described their vision as: "%s"

Project Type: %s
Team Size: %s
Timeline: %s`,
			responses["project_purpose"],
			responses["project_description"],
			responses["project_type"],
			responses["team_size"],
			responses["timeline"]),
	})

	// Requirements part
	if reqs := responses["requirements"]; reqs != "" {
		parts = append(parts, &CommissionPart{
			Title:   "Requirements",
			Content: g.formatRequirements(reqs),
		})
	}

	// Technology part
	if tech := responses["technology"]; tech != "" {
		parts = append(parts, &CommissionPart{
			Title:   "Technology Stack",
			Content: g.formatTechnology(tech),
		})
	}

	// Constraints part
	if constraints := responses["constraints"]; constraints != "" {
		parts = append(parts, &CommissionPart{
			Title:   "Constraints & Considerations",
			Content: g.formatConstraints(constraints),
		})
	}

	return parts
}

func (g *Generator) formatRequirements(reqs string) string {
	if reqs == "" {
		return "No specific requirements defined yet."
	}

	lines := strings.Split(reqs, "\n")
	var formatted []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			if !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "*") && !strings.HasPrefix(line, "•") {
				line = "• " + line
			}
			formatted = append(formatted, line)
		}
	}

	return strings.Join(formatted, "\n")
}

func (g *Generator) formatTechnology(tech string) string {
	if tech == "" {
		return "Technology stack to be determined."
	}

	// Similar formatting to requirements
	return g.formatRequirements(tech)
}

func (g *Generator) formatConstraints(constraints string) string {
	if constraints == "" {
		return "No specific constraints identified."
	}

	return g.formatRequirements(constraints)
}

// Commission Templates

const softwareCommissionTemplate = `# 🛠️ {{.project_description}}

> **Project Commission**: {{.project_purpose}}

---

## 🎯 Project Overview

{{.project_description}}

This commission establishes the requirements and constraints for developing a {{.project_type}} solution.

---

## 🛠 Technical Stack

{{.formatted_technology}}

---

## 📋 Core Requirements

{{.formatted_requirements}}

---

## ⚠️ Constraints & Considerations

{{.formatted_constraints}}

---

## 👥 Team & Timeline

- **Team Size**: {{.team_size}}
- **Timeline**: {{.timeline}}
- **Development Approach**: Iterative with regular milestones

---

## 🎯 Success Criteria

The project will be considered successful when:

1. All core requirements have been implemented and tested
2. The solution meets performance and security constraints
3. Documentation is complete and comprehensive
4. The system is deployed and operational

---

## 📁 Project Structure

The project will be organized with clear separation of concerns and follow best practices for {{.technology}} development.

---

## 🚀 Next Steps

1. Review and refine this commission document
2. Break down requirements into specific tasks
3. Set up development environment
4. Begin implementation of core features

---

> **💡 Note**: This commission was created through interactive planning with Elena, the Guild Master. It serves as the foundation for project execution and can be refined as needed.
`

const researchCommissionTemplate = `# 🔍 {{.project_description}}

> **Research Commission**: {{.project_purpose}}

---

## 🎯 Research Objective

{{.project_description}}

This commission guides a systematic investigation into {{.project_type}}.

---

## 📊 Research Scope

### Areas of Investigation

{{.formatted_requirements}}

### Research Methodology

{{.formatted_technology}}

---

## 🔬 Research Constraints

{{.formatted_constraints}}

---

## 📅 Timeline & Resources

- **Team Size**: {{.team_size}}
- **Duration**: {{.timeline}}
- **Approach**: Systematic analysis with documented findings

---

## 📈 Expected Deliverables

1. Comprehensive research report
2. Technical analysis and recommendations
3. Proof of concepts (if applicable)
4. Decision matrix for stakeholders
5. Implementation roadmap

---

## 🎯 Success Metrics

The research will be considered complete when:

1. All research questions have been thoroughly investigated
2. Findings are documented with supporting evidence
3. Recommendations are clear and actionable
4. Stakeholders have sufficient information for decision-making

---

## 📚 Documentation Plan

All research findings will be documented in:
- Executive summary
- Detailed technical analysis
- Comparison matrices
- Recommendation report

---

> **💡 Note**: This research commission was created through interactive planning with Elena, the Guild Master. It provides structure for systematic investigation.
`

const defaultCommissionTemplate = `# 📋 {{.project_description}}

> **Commission**: {{.project_purpose}}

---

## 🎯 Goal

{{.project_description}}

---

## 📂 Context

{{if .context}}
### Background
The user wants to {{.project_purpose}}. They described their vision as: "{{.project_description}}"

- **Project Type**: {{.project_type}}
- **Team Size**: {{.team_size}}
- **Timeline**: {{.timeline}}
{{end}}

---

## 🔧 Requirements

{{.formatted_requirements}}

---

## 🛠 Technology & Tools

{{.formatted_technology}}

---

## ⚠️ Constraints

{{.formatted_constraints}}

---

## 📌 Tags

{{range .Tags}}{{.}} {{end}}

---

## 🔗 Related

_No related documents yet_

---

## 🚀 Next Steps

1. Review this commission for completeness
2. Identify any missing requirements
3. Break down into actionable tasks
4. Begin implementation planning

---

> **💡 Note**: This commission was created through interactive planning with Elena, the Guild Master. It can be refined and expanded as the project evolves.
`
