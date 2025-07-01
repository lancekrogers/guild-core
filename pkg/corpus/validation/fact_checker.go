// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package validation

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/corpus/extraction"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// FactChecker provides fact-checking capabilities for knowledge validation
type FactChecker struct {
	checkers []FactCheckRule
}

// FactCheckRule interface for different fact-checking methods
type FactCheckRule interface {
	Check(ctx context.Context, knowledge extraction.ExtractedKnowledge) FactCheckResult
	GetType() string
	IsApplicable(knowledge extraction.ExtractedKnowledge) bool
}

// NewFactChecker creates a new fact checker with default rules
func NewFactChecker(ctx context.Context) (*FactChecker, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.validation").
			WithOperation("NewFactChecker")
	}

	checker := &FactChecker{
		checkers: []FactCheckRule{
			&TechnicalFactChecker{},
			&VersionChecker{},
			&SyntaxChecker{},
			&LogicalConsistencyChecker{},
			&DateTimeChecker{},
		},
	}

	return checker, nil
}

// Check performs fact-checking on a piece of knowledge
func (fc *FactChecker) Check(ctx context.Context, knowledge extraction.ExtractedKnowledge) (FactCheckResult, error) {
	if ctx.Err() != nil {
		return FactCheckResult{}, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.validation").
			WithOperation("Check")
	}

	result := FactCheckResult{
		Verified:   true,
		Confidence: 1.0,
		CheckedAt:  time.Now(),
	}

	var explanations []string
	var sources []string

	// Apply applicable fact-checking rules
	for _, checker := range fc.checkers {
		if !checker.IsApplicable(knowledge) {
			continue
		}

		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		checkResult := checker.Check(ctx, knowledge)
		
		if !checkResult.Verified {
			result.Verified = false
			if checkResult.Explanation != "" {
				explanations = append(explanations, fmt.Sprintf("[%s] %s", checker.GetType(), checkResult.Explanation))
			}
		}

		// Adjust confidence based on individual check results
		result.Confidence *= checkResult.Confidence

		if checkResult.Source != "" {
			sources = append(sources, checkResult.Source)
		}
	}

	// Compile explanations and sources
	if len(explanations) > 0 {
		result.Explanation = strings.Join(explanations, "; ")
	}
	if len(sources) > 0 {
		result.Source = strings.Join(sources, ", ")
	}

	// Ensure minimum confidence
	if result.Confidence < 0.1 {
		result.Confidence = 0.1
	}

	return result, nil
}

// TechnicalFactChecker validates technical claims and statements
type TechnicalFactChecker struct{}

func (tfc *TechnicalFactChecker) GetType() string { return "technical" }

func (tfc *TechnicalFactChecker) IsApplicable(knowledge extraction.ExtractedKnowledge) bool {
	// Apply to technical knowledge types
	return knowledge.Type == extraction.KnowledgeSolution ||
		   knowledge.Type == extraction.KnowledgePattern ||
		   knowledge.Type == extraction.KnowledgeConstraint
}

func (tfc *TechnicalFactChecker) Check(ctx context.Context, knowledge extraction.ExtractedKnowledge) FactCheckResult {
	result := FactCheckResult{
		Verified:   true,
		Confidence: 1.0,
		CheckedAt:  time.Now(),
	}

	content := strings.ToLower(knowledge.Content)

	// Check for common technical inaccuracies and unrealistic claims
	inaccuracies := []struct {
		pattern string
		message string
	}{
		{"always works", "Absolute claims about technical solutions are often inaccurate"},
		{"never fails", "Absolute reliability claims are typically unrealistic"},
		{"100% secure", "Absolute security claims are misleading"},
		{"zero overhead", "Claims of zero performance impact are usually incorrect"},
		{"infinitely scalable", "Infinite scalability claims are technically impossible"},
		{"no bugs", "Claims of bug-free software are unrealistic"},
		{"this solution always works", "Absolute technical claims are unrealistic"},
		{"never fails", "Claims of infallibility are unrealistic"},
	}

	for _, check := range inaccuracies {
		if strings.Contains(content, check.pattern) {
			result.Verified = false
			result.Confidence = 0.3
			result.Explanation = check.message
			return result
		}
	}

	// Check for overly generic technical claims
	vagueClaims := []string{
		"best practice", "optimal solution", "perfect approach",
		"always use", "never use", "industry standard",
	}

	vagueCount := 0
	for _, claim := range vagueClaims {
		if strings.Contains(content, claim) {
			vagueCount++
		}
	}

	if vagueCount > 2 {
		result.Verified = false
		result.Confidence = 0.4
		result.Explanation = "Contains multiple vague or absolute technical claims"
	}

	return result
}

// VersionChecker validates version numbers and compatibility claims
type VersionChecker struct{}

func (vc *VersionChecker) GetType() string { return "version" }

func (vc *VersionChecker) IsApplicable(knowledge extraction.ExtractedKnowledge) bool {
	content := strings.ToLower(knowledge.Content)
	return strings.Contains(content, "version") ||
		   strings.Contains(content, "v1.") ||
		   strings.Contains(content, "v2.") ||
		   regexp.MustCompile(`\d+\.\d+`).MatchString(content)
}

func (vc *VersionChecker) Check(ctx context.Context, knowledge extraction.ExtractedKnowledge) FactCheckResult {
	result := FactCheckResult{
		Verified:   true,
		Confidence: 1.0,
		CheckedAt:  time.Now(),
	}

	content := knowledge.Content

	// Extract version numbers
	versionPattern := regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`)
	versions := versionPattern.FindAllString(content, -1)

	if len(versions) > 0 {
		// Check if versions mentioned are reasonable
		for _, version := range versions {
			parts := strings.Split(version, ".")
			if len(parts) >= 2 {
				major := parts[0]
				minor := parts[1]

				// Basic sanity checks
				if len(major) > 2 || len(minor) > 3 {
					result.Verified = false
					result.Confidence = 0.4
					result.Explanation = fmt.Sprintf("Version number '%s' appears unusual", version)
					return result
				}
			}
		}

		// Check for version currency (if source is old)
		if time.Since(knowledge.Source.Timestamp) > 365*24*time.Hour {
			result.Confidence = 0.8
			result.Explanation = "Version information may be outdated"
		}
	}

	return result
}

// SyntaxChecker validates code syntax and examples
type SyntaxChecker struct{}

func (sc *SyntaxChecker) GetType() string { return "syntax" }

func (sc *SyntaxChecker) IsApplicable(knowledge extraction.ExtractedKnowledge) bool {
	content := knowledge.Content
	return strings.Contains(content, "```") ||
		   strings.Contains(content, "function") ||
		   strings.Contains(content, "class") ||
		   strings.Contains(content, "import") ||
		   strings.Contains(content, "package")
}

func (sc *SyntaxChecker) Check(ctx context.Context, knowledge extraction.ExtractedKnowledge) FactCheckResult {
	result := FactCheckResult{
		Verified:   true,
		Confidence: 1.0,
		CheckedAt:  time.Now(),
	}

	content := knowledge.Content

	// Extract code blocks
	codeBlockPattern := regexp.MustCompile("```[\\s\\S]*?```")
	codeBlocks := codeBlockPattern.FindAllString(content, -1)

	for _, block := range codeBlocks {
		// Basic syntax validation
		if sc.hasBasicSyntaxErrors(block) {
			result.Verified = false
			result.Confidence = 0.3
			result.Explanation = "Code examples contain syntax errors"
			return result
		}
	}

	// Check for common syntax mistakes in inline code
	commonMistakes := []string{
		"import(", // Missing space in Go
		"func()",  // Invalid function declaration
		"if (",    // Missing condition
		"for (",   // Missing loop parts
	}

	for _, mistake := range commonMistakes {
		if strings.Contains(content, mistake) {
			result.Confidence = 0.8
			result.Explanation = "Potential syntax issues detected in code examples"
			break
		}
	}

	return result
}

func (sc *SyntaxChecker) hasBasicSyntaxErrors(codeBlock string) bool {
	// Check for obvious syntax errors like incomplete function declarations
	content := strings.ToLower(codeBlock)
	
	// Look for common syntax error patterns
	errorPatterns := []string{
		"func( {",     // Missing function name and parameters
		"func() {",    // Missing function name
		"return\n}",  // Missing return value
		"}\n```",     // Incomplete code block
	}
	
	for _, pattern := range errorPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	
	// Check for unbalanced brackets across the entire block
	openBrackets := strings.Count(codeBlock, "{") + strings.Count(codeBlock, "(") + strings.Count(codeBlock, "[")
	closeBrackets := strings.Count(codeBlock, "}") + strings.Count(codeBlock, ")") + strings.Count(codeBlock, "]")
	
	// Significant imbalance suggests syntax errors
	return absInt(openBrackets-closeBrackets) > 1
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// LogicalConsistencyChecker validates logical consistency within knowledge
type LogicalConsistencyChecker struct{}

func (lcc *LogicalConsistencyChecker) GetType() string { return "logical" }

func (lcc *LogicalConsistencyChecker) IsApplicable(knowledge extraction.ExtractedKnowledge) bool {
	// Apply to all knowledge types
	return true
}

func (lcc *LogicalConsistencyChecker) Check(ctx context.Context, knowledge extraction.ExtractedKnowledge) FactCheckResult {
	result := FactCheckResult{
		Verified:   true,
		Confidence: 1.0,
		CheckedAt:  time.Now(),
	}

	content := strings.ToLower(knowledge.Content)

	// Check for logical contradictions within the same content
	contradictions := [][]string{
		{"always", "sometimes"},
		{"never", "occasionally"},
		{"required", "optional"},
		{"mandatory", "suggested"},
		{"must", "may"},
		{"secure", "vulnerable"},
		{"fast", "slow"},
		{"efficient", "wasteful"},
	}

	for _, pair := range contradictions {
		if strings.Contains(content, pair[0]) && strings.Contains(content, pair[1]) {
			result.Verified = false
			result.Confidence = 0.6
			result.Explanation = fmt.Sprintf("Logical inconsistency: contains both '%s' and '%s'", pair[0], pair[1])
			return result
		}
	}

	// Check for overly complex or confusing statements
	sentences := strings.Split(content, ".")
	for _, sentence := range sentences {
		wordCount := len(strings.Fields(sentence))
		if wordCount > 50 {
			result.Confidence = 0.9
			result.Explanation = "Contains very complex sentences that may be unclear"
			break
		}
	}

	return result
}

// DateTimeChecker validates date and time related claims
type DateTimeChecker struct{}

func (dtc *DateTimeChecker) GetType() string { return "datetime" }

func (dtc *DateTimeChecker) IsApplicable(knowledge extraction.ExtractedKnowledge) bool {
	content := strings.ToLower(knowledge.Content)
	return strings.Contains(content, "date") ||
		   strings.Contains(content, "time") ||
		   strings.Contains(content, "year") ||
		   strings.Contains(content, "month") ||
		   regexp.MustCompile(`\d{4}`).MatchString(content)
}

func (dtc *DateTimeChecker) Check(ctx context.Context, knowledge extraction.ExtractedKnowledge) FactCheckResult {
	result := FactCheckResult{
		Verified:   true,
		Confidence: 1.0,
		CheckedAt:  time.Now(),
	}

	content := knowledge.Content

	// Check for future dates that might be unrealistic
	yearPattern := regexp.MustCompile(`20(\d{2})`)
	years := yearPattern.FindAllString(content, -1)

	currentYear := time.Now().Year()
	for _, yearStr := range years {
		if len(yearStr) == 4 {
			year := int(yearStr[0]-'0')*1000 + int(yearStr[1]-'0')*100 + int(yearStr[2]-'0')*10 + int(yearStr[3]-'0')
			
			if int(year) > currentYear+10 {
				result.Verified = false
				result.Confidence = 0.4
				result.Explanation = fmt.Sprintf("Contains unrealistic future year: %s", yearStr)
				return result
			}
			
			if int(year) < 1990 && strings.Contains(strings.ToLower(content), "current") {
				result.Confidence = 0.7
				result.Explanation = "Contains old dates in context suggesting currency"
			}
		}
	}

	return result
}

// AddFactChecker adds a custom fact checking rule
func (fc *FactChecker) AddFactChecker(checker FactCheckRule) {
	fc.checkers = append(fc.checkers, checker)
}

// RemoveFactChecker removes a fact checking rule by type
func (fc *FactChecker) RemoveFactChecker(checkerType string) {
	var filteredCheckers []FactCheckRule
	for _, checker := range fc.checkers {
		if checker.GetType() != checkerType {
			filteredCheckers = append(filteredCheckers, checker)
		}
	}
	fc.checkers = filteredCheckers
}