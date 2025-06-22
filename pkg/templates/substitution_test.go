// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariableSubstitution_BasicSubstitution(t *testing.T) {
	vs := NewVariableSubstitution()

	content := "Hello {{name}}, welcome to {{project}}!"
	variables := map[string]interface{}{
		"name":    "Alice",
		"project": "Guild Framework",
	}

	result, err := vs.SubstituteVariables(content, variables, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Hello Alice, welcome to Guild Framework!", result)
}

func TestVariableSubstitution_DefaultValues(t *testing.T) {
	vs := NewVariableSubstitution()

	content := "Hello {{name:World}}, status is {{status:OK}}!"
	variables := map[string]interface{}{
		"name": "Alice",
		// status not provided, should use default
	}

	result, err := vs.SubstituteVariables(content, variables, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Hello Alice, status is OK!", result)
}

func TestVariableSubstitution_ConditionalFormatting(t *testing.T) {
	vs := NewVariableSubstitution()

	content := "Name: {{name|upper}}, Project: {{project|title}}, Code: {{code|code}}"
	variables := map[string]interface{}{
		"name":    "alice",
		"project": "guild framework",
		"code":    "fmt.Println",
	}

	result, err := vs.SubstituteVariables(content, variables, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Name: ALICE, Project: Guild Framework, Code: `fmt.Println`", result)
}

func TestVariableSubstitution_FormatFilters(t *testing.T) {
	vs := NewVariableSubstitution()

	tests := []struct {
		format   string
		input    string
		expected string
	}{
		{"upper", "hello", "HELLO"},
		{"lower", "HELLO", "hello"},
		{"title", "hello world", "Hello World"},
		{"trim", "  hello  ", "hello"},
		{"quote", "hello", `"hello"`},
		{"code", "hello", "`hello`"},
		{"bold", "hello", "**hello**"},
		{"italic", "hello", "*hello*"},
		{"unknown", "hello", "hello"}, // Unknown format returns as-is
	}

	for _, test := range tests {
		t.Run(test.format, func(t *testing.T) {
			result := vs.applyFormat(test.input, test.format)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestVariableSubstitution_RequiredVariables(t *testing.T) {
	vs := NewVariableSubstitution()

	content := "Hello {{name}}, your score is {{score}}!"
	variables := map[string]interface{}{
		"name": "Alice",
		// score is missing but required
	}

	templateVars := []*TemplateVariable{
		{Name: "name", Required: true},
		{Name: "score", Required: true},
	}

	_, err := vs.SubstituteVariables(content, variables, templateVars)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required variable not provided")
}

func TestVariableSubstitution_OptionalVariables(t *testing.T) {
	vs := NewVariableSubstitution()

	content := "Hello {{name}}, optional: {{optional}}"
	variables := map[string]interface{}{
		"name": "Alice",
		// optional is missing but not required
	}

	templateVars := []*TemplateVariable{
		{Name: "name", Required: true},
		{Name: "optional", Required: false},
	}

	result, err := vs.SubstituteVariables(content, variables, templateVars)
	assert.NoError(t, err)
	assert.Equal(t, "Hello Alice, optional: {{optional}}", result) // Unreplaced
}

func TestVariableSubstitution_ExtractVariableReferences(t *testing.T) {
	vs := NewVariableSubstitution()

	content := `
		Basic: {{name}}
		Default: {{status:OK}}
		Conditional: {{message|upper}}
		Multiple: {{name}} and {{status:READY}}
	`

	refs := vs.extractVariableReferences(content)
	assert.Contains(t, refs, "name")
	assert.Contains(t, refs, "status")
	assert.Contains(t, refs, "message")
	assert.Len(t, refs, 3) // Should deduplicate "name" and "status"
}

func TestVariableSubstitution_GetVariableReferences(t *testing.T) {
	vs := NewVariableSubstitution()

	content := `
		Basic: {{name}}
		Default: {{status:OK}}
		Conditional: {{message|upper}}
	`

	refs := vs.GetVariableReferences(content)
	require.Len(t, refs, 3)

	// Check basic variable
	nameRef := refs["name"]
	require.NotNil(t, nameRef)
	assert.Equal(t, "name", nameRef.Name)
	assert.Equal(t, "basic", nameRef.Type)
	assert.True(t, nameRef.Required)

	// Check default variable
	statusRef := refs["status"]
	require.NotNil(t, statusRef)
	assert.Equal(t, "status", statusRef.Name)
	assert.Equal(t, "default", statusRef.Type)
	assert.Equal(t, "OK", statusRef.DefaultValue)
	assert.False(t, statusRef.Required)

	// Check conditional variable
	messageRef := refs["message"]
	require.NotNil(t, messageRef)
	assert.Equal(t, "message", messageRef.Name)
	assert.Equal(t, "conditional", messageRef.Type)
	assert.Equal(t, "upper", messageRef.Format)
	assert.False(t, messageRef.Required)
}

func TestVariableSubstitution_ValidateTemplate(t *testing.T) {
	vs := NewVariableSubstitution()

	content := "Hello {{name}}, status: {{status}}, unknown: {{missing}}"
	variables := []*TemplateVariable{
		{Name: "name", Required: true},
		{Name: "status", Required: false},
		// "missing" is not defined
	}

	err := vs.ValidateTemplate(content, variables)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "undefined variables")
	assert.Contains(t, err.Error(), "missing")
}

func TestVariableSubstitution_ValidateTemplateSuccess(t *testing.T) {
	vs := NewVariableSubstitution()

	content := "Hello {{name}}, status: {{status:OK}}"
	variables := []*TemplateVariable{
		{Name: "name", Required: true},
		{Name: "status", Required: false},
	}

	err := vs.ValidateTemplate(content, variables)
	assert.NoError(t, err)
}

func TestVariableSubstitution_PreviewSubstitution(t *testing.T) {
	vs := NewVariableSubstitution()

	content := "Hello {{name}}, status: {{status:OK}}, missing: {{missing}}"
	variables := map[string]interface{}{
		"name": "Alice",
		// status will use default, missing is not provided
	}
	templateVars := []*TemplateVariable{
		{Name: "name", Required: true},
		{Name: "status", Required: false, DefaultValue: "OK"},
		{Name: "missing", Required: false},
	}

	preview, err := vs.PreviewSubstitution(content, variables, templateVars)
	assert.NoError(t, err)
	require.NotNil(t, preview)

	assert.Equal(t, "Hello Alice, status: OK, missing: {{missing}}", preview.Result)
	assert.Contains(t, preview.UsedVariables, "name")
	assert.Equal(t, "Alice", preview.UsedVariables["name"])
	assert.Contains(t, preview.MissingVariables, "missing")
	assert.Len(t, preview.VariableReferences, 3)
}

func TestVariableSubstitution_ComplexTemplate(t *testing.T) {
	vs := NewVariableSubstitution()

	content := `
# {{title|title}}

**Description:** {{description}}

**Author:** {{author:Unknown}}

**Environment:** {{env|upper}}

**Code Example:**
` + "```{{language:bash}}" + `
{{code}}
` + "```" + `

**Notes:** {{notes:No additional notes}}
`

	variables := map[string]interface{}{
		"title":       "api documentation",
		"description": "REST API documentation for the Guild framework",
		"env":         "production",
		"language":    "go",
		"code":        "fmt.Println(\"Hello, Guild!\")",
	}

	templateVars := []*TemplateVariable{
		{Name: "title", Required: true},
		{Name: "description", Required: true},
		{Name: "author", Required: false, DefaultValue: "Unknown"},
		{Name: "env", Required: true},
		{Name: "language", Required: false, DefaultValue: "bash"},
		{Name: "code", Required: true},
		{Name: "notes", Required: false, DefaultValue: "No additional notes"},
	}

	result, err := vs.SubstituteVariables(content, variables, templateVars)
	assert.NoError(t, err)

	assert.Contains(t, result, "# Api Documentation") // title case
	assert.Contains(t, result, "REST API documentation")
	assert.Contains(t, result, "**Author:** Unknown")         // default value used
	assert.Contains(t, result, "**Environment:** PRODUCTION") // uppercase
	assert.Contains(t, result, "```go")                       // overridden language
	assert.Contains(t, result, "fmt.Println")
	assert.Contains(t, result, "No additional notes") // default value
}

func TestVariableSubstitution_EdgeCases(t *testing.T) {
	vs := NewVariableSubstitution()

	tests := []struct {
		name     string
		content  string
		vars     map[string]interface{}
		expected string
	}{
		{
			name:     "Empty content",
			content:  "",
			vars:     map[string]interface{}{"name": "test"},
			expected: "",
		},
		{
			name:     "No variables in content",
			content:  "Static content with no variables",
			vars:     map[string]interface{}{"name": "test"},
			expected: "Static content with no variables",
		},
		{
			name:     "Variable with spaces",
			content:  "Hello {{ name }}!", // Note: spaces around variable name
			vars:     map[string]interface{}{"name": "Alice"},
			expected: "Hello Alice!",
		},
		{
			name:     "Malformed variables ignored",
			content:  "Hello {name} and {{name}} and {{{name}}}",
			vars:     map[string]interface{}{"name": "Alice"},
			expected: "Hello {name} and Alice and {{{name}}}",
		},
		{
			name:     "Nested braces",
			content:  "JSON: {{json}} and regular {{name}}",
			vars:     map[string]interface{}{"json": "{\"key\": \"value\"}", "name": "test"},
			expected: "JSON: {\"key\": \"value\"} and regular test",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := vs.SubstituteVariables(test.content, test.vars, nil)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, result)
		})
	}
}
