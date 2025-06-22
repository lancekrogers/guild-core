// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package templates

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// VariableSubstitution provides advanced variable substitution capabilities
type VariableSubstitution struct {
	// Pattern matching for variables: {{variable}}, {{variable:default}}, {{variable|filter}}
	basicPattern       *regexp.Regexp
	defaultPattern     *regexp.Regexp
	conditionalPattern *regexp.Regexp
}

// NewVariableSubstitution creates a new substitution engine
func NewVariableSubstitution() *VariableSubstitution {
	return &VariableSubstitution{
		basicPattern:       regexp.MustCompile(`\{\{\s*([^}:|\s]+)\s*\}\}`),
		defaultPattern:     regexp.MustCompile(`\{\{\s*([^}:|\s]+)\s*:\s*([^}|]*)\s*\}\}`),
		conditionalPattern: regexp.MustCompile(`\{\{\s*([^}:|\s]+)\s*\|\s*([^}]*)\s*\}\}`),
	}
}

// SubstituteVariables performs variable substitution with advanced features
func (vs *VariableSubstitution) SubstituteVariables(content string, variables map[string]interface{}, templateVars []*TemplateVariable) (string, error) {
	result := content
	var err error

	// Build variable map with defaults from template definition
	varMap := make(map[string]interface{})
	for _, templateVar := range templateVars {
		if templateVar.DefaultValue != "" {
			varMap[templateVar.Name] = templateVar.DefaultValue
		}
	}

	// Override with provided variables
	for k, v := range variables {
		varMap[k] = v
	}

	// 1. Handle variables with defaults: {{name:default_value}}
	result = vs.defaultPattern.ReplaceAllStringFunc(result, func(match string) string {
		matches := vs.defaultPattern.FindStringSubmatch(match)
		if len(matches) == 3 {
			varName := strings.TrimSpace(matches[1])
			defaultValue := strings.TrimSpace(matches[2])

			if value, exists := varMap[varName]; exists && value != nil {
				return fmt.Sprintf("%v", value)
			}
			return defaultValue
		}
		return match
	})

	// 2. Handle conditional variables: {{name|format}}
	result = vs.conditionalPattern.ReplaceAllStringFunc(result, func(match string) string {
		matches := vs.conditionalPattern.FindStringSubmatch(match)
		if len(matches) == 3 {
			varName := strings.TrimSpace(matches[1])
			format := strings.TrimSpace(matches[2])

			if value, exists := varMap[varName]; exists && value != nil {
				return vs.applyFormat(fmt.Sprintf("%v", value), format)
			}
			return "" // Empty if variable not found
		}
		return match
	})

	// 3. Handle basic variables: {{name}}
	result = vs.basicPattern.ReplaceAllStringFunc(result, func(match string) string {
		matches := vs.basicPattern.FindStringSubmatch(match)
		if len(matches) == 2 {
			varName := strings.TrimSpace(matches[1])

			if value, exists := varMap[varName]; exists && value != nil {
				return fmt.Sprintf("%v", value)
			}

			// Check if this is a required variable
			for _, templateVar := range templateVars {
				if templateVar.Name == varName && templateVar.Required {
					err = gerror.New(gerror.ErrCodeInvalidInput, "required variable not provided", nil).
						WithComponent("templates").
						WithOperation("SubstituteVariables").
						WithDetails("variable_name", varName)
					return match
				}
			}

			return match // Leave unreplaced if not required
		}
		return match
	})

	return result, err
}

// applyFormat applies formatting to variable values
func (vs *VariableSubstitution) applyFormat(value, format string) string {
	switch strings.ToLower(format) {
	case "upper", "uppercase":
		return strings.ToUpper(value)
	case "lower", "lowercase":
		return strings.ToLower(value)
	case "title", "titlecase":
		return strings.Title(value)
	case "trim":
		return strings.TrimSpace(value)
	case "quote":
		return fmt.Sprintf(`"%s"`, value)
	case "code":
		return fmt.Sprintf("`%s`", value)
	case "bold":
		return fmt.Sprintf("**%s**", value)
	case "italic":
		return fmt.Sprintf("*%s*", value)
	default:
		// If format is not recognized, return as-is
		return value
	}
}

// ValidateTemplate checks if a template content has valid variable syntax
func (vs *VariableSubstitution) ValidateTemplate(content string, variables []*TemplateVariable) error {
	// Find all variable references in content
	varRefs := vs.extractVariableReferences(content)

	// Build map of defined variables
	definedVars := make(map[string]*TemplateVariable)
	for _, templateVar := range variables {
		definedVars[templateVar.Name] = templateVar
	}

	// Check for undefined variables
	var undefinedVars []string
	for _, varRef := range varRefs {
		if _, exists := definedVars[varRef]; !exists {
			undefinedVars = append(undefinedVars, varRef)
		}
	}

	if len(undefinedVars) > 0 {
		message := fmt.Sprintf("template contains undefined variables: %s", strings.Join(undefinedVars, ", "))
		return gerror.New(gerror.ErrCodeInvalidInput, message, nil).
			WithComponent("templates").
			WithOperation("ValidateTemplate").
			WithDetails("undefined_variables", strings.Join(undefinedVars, ", "))
	}

	return nil
}

// extractVariableReferences extracts all variable names from template content
func (vs *VariableSubstitution) extractVariableReferences(content string) []string {
	var refs []string
	seen := make(map[string]bool)

	// Basic variables: {{name}}
	matches := vs.basicPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 2 {
			varName := strings.TrimSpace(match[1])
			if !seen[varName] {
				refs = append(refs, varName)
				seen[varName] = true
			}
		}
	}

	// Variables with defaults: {{name:default}}
	matches = vs.defaultPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			varName := strings.TrimSpace(match[1])
			if !seen[varName] {
				refs = append(refs, varName)
				seen[varName] = true
			}
		}
	}

	// Conditional variables: {{name|format}}
	matches = vs.conditionalPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			varName := strings.TrimSpace(match[1])
			if !seen[varName] {
				refs = append(refs, varName)
				seen[varName] = true
			}
		}
	}

	return refs
}

// GetVariableReferences returns all variable references found in template content
func (vs *VariableSubstitution) GetVariableReferences(content string) map[string]*VariableReference {
	refs := make(map[string]*VariableReference)

	// Basic variables: {{name}}
	matches := vs.basicPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 2 {
			varName := strings.TrimSpace(match[1])
			refs[varName] = &VariableReference{
				Name:     varName,
				Type:     "basic",
				Required: true,
			}
		}
	}

	// Variables with defaults: {{name:default}}
	matches = vs.defaultPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			varName := strings.TrimSpace(match[1])
			defaultValue := strings.TrimSpace(match[2])
			refs[varName] = &VariableReference{
				Name:         varName,
				Type:         "default",
				DefaultValue: defaultValue,
				Required:     false,
			}
		}
	}

	// Conditional variables: {{name|format}}
	matches = vs.conditionalPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			varName := strings.TrimSpace(match[1])
			format := strings.TrimSpace(match[2])
			refs[varName] = &VariableReference{
				Name:     varName,
				Type:     "conditional",
				Format:   format,
				Required: false,
			}
		}
	}

	return refs
}

// VariableReference represents a variable reference found in template content
type VariableReference struct {
	Name         string `json:"name"`
	Type         string `json:"type"` // "basic", "default", "conditional"
	DefaultValue string `json:"default_value,omitempty"`
	Format       string `json:"format,omitempty"`
	Required     bool   `json:"required"`
}

// PreviewSubstitution shows what the template would look like with given variables
func (vs *VariableSubstitution) PreviewSubstitution(content string, variables map[string]interface{}, templateVars []*TemplateVariable) (*SubstitutionPreview, error) {
	result, err := vs.SubstituteVariables(content, variables, templateVars)
	if err != nil {
		return nil, err
	}

	// Find variables that were used vs not used
	varRefs := vs.GetVariableReferences(content)
	usedVars := make(map[string]interface{})
	missingVars := make([]string, 0)

	for varName := range varRefs {
		if value, exists := variables[varName]; exists {
			usedVars[varName] = value
		} else {
			// Check if template variable has default
			hasDefault := false
			for _, templateVar := range templateVars {
				if templateVar.Name == varName && templateVar.DefaultValue != "" {
					hasDefault = true
					break
				}
			}
			if !hasDefault {
				missingVars = append(missingVars, varName)
			}
		}
	}

	return &SubstitutionPreview{
		Result:             result,
		UsedVariables:      usedVars,
		MissingVariables:   missingVars,
		VariableReferences: varRefs,
	}, nil
}

// SubstitutionPreview provides a preview of template substitution
type SubstitutionPreview struct {
	Result             string                        `json:"result"`
	UsedVariables      map[string]interface{}        `json:"used_variables"`
	MissingVariables   []string                      `json:"missing_variables"`
	VariableReferences map[string]*VariableReference `json:"variable_references"`
}
