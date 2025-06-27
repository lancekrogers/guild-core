// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package evaluation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// ContainsAssertion checks if the output contains a substring
type ContainsAssertion struct {
	Substring string
}

func (a *ContainsAssertion) Assert(output string) error {
	if !strings.Contains(output, a.Substring) {
		return gerror.Newf(gerror.ErrCodeInvalidInput, "output does not contain '%s'", a.Substring).WithComponent("prompts").WithOperation("contains_assertion")
	}
	return nil
}

func (a *ContainsAssertion) Description() string {
	return fmt.Sprintf("contains '%s'", a.Substring)
}

// RegexAssertion checks if the output matches a regex pattern
type RegexAssertion struct {
	Pattern string
	regex   *regexp.Regexp
}

func NewRegexAssertion(pattern string) (*RegexAssertion, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "prompts").WithComponent("regex_assertion").WithOperation("invalid regex pattern")
	}
	return &RegexAssertion{Pattern: pattern, regex: regex}, nil
}

func (a *RegexAssertion) Assert(output string) error {
	if !a.regex.MatchString(output) {
		return gerror.Newf(gerror.ErrCodeInvalidInput, "output does not match pattern '%s'", a.Pattern).WithComponent("prompts").WithOperation("regex_assertion")
	}
	return nil
}

func (a *RegexAssertion) Description() string {
	return fmt.Sprintf("matches regex '%s'", a.Pattern)
}

// LengthAssertion checks if the output meets length requirements
type LengthAssertion struct {
	MinLength int
	MaxLength int
}

func (a *LengthAssertion) Assert(output string) error {
	length := len(output)
	if a.MinLength > 0 && length < a.MinLength {
		return gerror.Newf(gerror.ErrCodeInvalidInput, "output too short: %d < %d", length, a.MinLength).WithComponent("prompts").WithOperation("length_assertion")
	}
	if a.MaxLength > 0 && length > a.MaxLength {
		return gerror.Newf(gerror.ErrCodeInvalidInput, "output too long: %d > %d", length, a.MaxLength).WithComponent("prompts").WithOperation("length_assertion")
	}
	return nil
}

func (a *LengthAssertion) Description() string {
	if a.MinLength > 0 && a.MaxLength > 0 {
		return fmt.Sprintf("length between %d and %d", a.MinLength, a.MaxLength)
	} else if a.MinLength > 0 {
		return fmt.Sprintf("length at least %d", a.MinLength)
	} else {
		return fmt.Sprintf("length at most %d", a.MaxLength)
	}
}

// StructureAssertion checks if the output has expected structure
type StructureAssertion struct {
	RequiredSections []string
}

func (a *StructureAssertion) Assert(output string) error {
	for _, section := range a.RequiredSections {
		if !strings.Contains(output, section) {
			return gerror.Newf(gerror.ErrCodeInvalidInput, "missing required section: %s", section).WithComponent("prompts").WithOperation("structure_assertion")
		}
	}
	return nil
}

func (a *StructureAssertion) Description() string {
	return fmt.Sprintf("contains sections: %v", a.RequiredSections)
}

// MultiAssertion combines multiple assertions
type MultiAssertion struct {
	Assertions []PromptAssertion
	RequireAll bool
}

func (a *MultiAssertion) Assert(output string) error {
	var errors []string

	for _, assertion := range a.Assertions {
		err := assertion.Assert(output)
		if err != nil {
			if a.RequireAll {
				errors = append(errors, err.Error())
			}
		} else if !a.RequireAll {
			// If we don't require all and one passes, we're good
			return nil
		}
	}

	if a.RequireAll && len(errors) > 0 {
		return gerror.Newf(gerror.ErrCodeInvalidInput, "multiple assertion failures: %s", strings.Join(errors, "; ")).WithComponent("prompts").WithOperation("composite_assertion")
	}

	if !a.RequireAll && len(errors) == len(a.Assertions) {
		return gerror.Newf(gerror.ErrCodeInvalidInput, "all assertions failed: %s", strings.Join(errors, "; ")).WithComponent("prompts").WithOperation("composite_assertion")
	}

	return nil
}

func (a *MultiAssertion) Description() string {
	var descriptions []string
	for _, assertion := range a.Assertions {
		descriptions = append(descriptions, assertion.Description())
	}
	if a.RequireAll {
		return "all of: " + strings.Join(descriptions, ", ")
	}
	return "any of: " + strings.Join(descriptions, ", ")
}

// QualityAssertion checks for quality indicators
type QualityAssertion struct {
	MinSentences      int
	RequireExamples   bool
	RequireFormatting bool
}

func (a *QualityAssertion) Assert(output string) error {
	// Count sentences (simple approximation)
	sentences := strings.Count(output, ".") + strings.Count(output, "!") + strings.Count(output, "?")
	if sentences < a.MinSentences {
		return gerror.Newf(gerror.ErrCodeInvalidInput, "insufficient detail: only %d sentences (min %d)", sentences, a.MinSentences).WithComponent("prompts").WithOperation("quality_assertion")
	}

	if a.RequireExamples {
		// Look for common example indicators
		hasExamples := strings.Contains(output, "example") ||
			strings.Contains(output, "Example") ||
			strings.Contains(output, "e.g.") ||
			strings.Contains(output, "for instance")
		if !hasExamples {
			return gerror.New(gerror.ErrCodeInvalidInput, "no examples found in output", nil).WithComponent("prompts").WithOperation("quality_assertion")
		}
	}

	if a.RequireFormatting {
		// Check for markdown formatting
		hasFormatting := strings.Contains(output, "#") ||
			strings.Contains(output, "**") ||
			strings.Contains(output, "- ") ||
			strings.Contains(output, "```")
		if !hasFormatting {
			return gerror.New(gerror.ErrCodeInvalidInput, "no formatting found in output", nil).WithComponent("prompts").WithOperation("quality_assertion")
		}
	}

	return nil
}

func (a *QualityAssertion) Description() string {
	parts := []string{}
	if a.MinSentences > 0 {
		parts = append(parts, fmt.Sprintf("at least %d sentences", a.MinSentences))
	}
	if a.RequireExamples {
		parts = append(parts, "includes examples")
	}
	if a.RequireFormatting {
		parts = append(parts, "properly formatted")
	}
	return strings.Join(parts, ", ")
}
