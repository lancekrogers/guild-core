package prompts

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// PromptMetadata contains metadata for a prompt template
type PromptMetadata struct {
	ID                 string                 `yaml:"id"`
	Version            string                 `yaml:"version"`
	Category           string                 `yaml:"category"`
	Complexity         int                    `yaml:"complexity"`
	Tags               []string               `yaml:"tags"`
	Variables          VariableConfig         `yaml:"variables"`
	Created            time.Time              `yaml:"created"`
	Updated            time.Time              `yaml:"updated"`
	ModelCompatibility []string               `yaml:"model_compatibility"`
	EvaluationCriteria []string               `yaml:"evaluation_criteria"`
	Extra              map[string]interface{} `yaml:",inline"` // For future extensibility
}

// VariableConfig defines required and optional variables for a prompt
type VariableConfig struct {
	Required []string `yaml:"required"`
	Optional []string `yaml:"optional"`
}

// PromptTemplate wraps a template with its metadata
type PromptTemplate struct {
	Name     string
	Content  string
	Metadata *PromptMetadata
}

// ParsePromptWithMetadata extracts YAML frontmatter and content from a prompt file
func ParsePromptWithMetadata(content string) (*PromptTemplate, error) {
	// Check if the content starts with frontmatter delimiter
	if !strings.HasPrefix(content, "---\n") {
		return &PromptTemplate{
			Content:  content,
			Metadata: nil, // No metadata, treat as legacy prompt
		}, nil
	}

	// Find the closing delimiter
	parts := strings.SplitN(content[4:], "\n---\n", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid frontmatter format: missing closing delimiter")
	}

	// Parse YAML metadata
	var metadata PromptMetadata
	if err := yaml.Unmarshal([]byte(parts[0]), &metadata); err != nil {
		return nil, fmt.Errorf("error parsing frontmatter: %w", err)
	}

	// Validate metadata
	if err := metadata.Validate(); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	return &PromptTemplate{
		Content:  strings.TrimSpace(parts[1]),
		Metadata: &metadata,
	}, nil
}

// Validate checks if the metadata is valid
func (m *PromptMetadata) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("missing required field: id")
	}
	if m.Version == "" {
		return fmt.Errorf("missing required field: version")
	}
	if m.Category == "" {
		return fmt.Errorf("missing required field: category")
	}
	if m.Complexity < 1 || m.Complexity > 10 {
		return fmt.Errorf("complexity must be between 1 and 10, got %d", m.Complexity)
	}
	return nil
}

// CalculateComplexity analyzes prompt content to calculate complexity score
func CalculateComplexity(content string) int {
	complexity := 1

	// Factor 1: Length (tokens approximation)
	tokenCount := len(strings.Fields(content))
	if tokenCount > 500 {
		complexity += 2
	} else if tokenCount > 200 {
		complexity += 1
	}

	// Factor 2: Conditional blocks
	ifBlockCount := strings.Count(content, "<if_block")
	complexity += ifBlockCount

	// Factor 3: Variable count
	varCount := strings.Count(content, "{{.")
	if varCount > 10 {
		complexity += 2
	} else if varCount > 5 {
		complexity += 1
	}

	// Factor 4: Result placeholders
	resultCount := strings.Count(content, "<result")
	complexity += resultCount

	// Factor 5: Nested structure depth (simple heuristic)
	maxHeaderDepth := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "#") {
			depth := len(line) - len(strings.TrimLeft(line, "#"))
			if depth > maxHeaderDepth {
				maxHeaderDepth = depth
			}
		}
	}
	if maxHeaderDepth > 3 {
		complexity += 1
	}

	// Cap at 10
	if complexity > 10 {
		complexity = 10
	}

	return complexity
}

// IsCompatibleWithModel checks if a prompt is compatible with a given model
func (m *PromptMetadata) IsCompatibleWithModel(model string) bool {
	if len(m.ModelCompatibility) == 0 {
		// If no compatibility list, assume it works with all models
		return true
	}

	for _, compatible := range m.ModelCompatibility {
		if strings.EqualFold(compatible, model) {
			return true
		}
	}
	return false
}

// HasRequiredVariables checks if all required variables are present in the data
func (m *PromptMetadata) HasRequiredVariables(data map[string]interface{}) error {
	for _, required := range m.Variables.Required {
		if _, exists := data[required]; !exists {
			return fmt.Errorf("missing required variable: %s", required)
		}
	}
	return nil
}