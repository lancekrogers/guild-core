// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package extraction

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// DiffAnalyzer analyzes git diffs to extract structural changes
type DiffAnalyzer struct {
	patterns map[string]*regexp.Regexp
}

// NewDiffAnalyzer creates a new diff analyzer with default patterns
func NewDiffAnalyzer() *DiffAnalyzer {
	patterns := map[string]*regexp.Regexp{
		"go_function":     regexp.MustCompile(`^[\+\-]\s*func\s+(\w+)`),
		"go_type":         regexp.MustCompile(`^[\+\-]\s*type\s+(\w+)`),
		"go_import":       regexp.MustCompile(`^[\+\-]\s*import\s+.*?"([^"]+)"`),
		"go_struct_field": regexp.MustCompile(`^[\+\-]\s*(\w+)\s+\w+.*`),
		"js_function":     regexp.MustCompile(`^[\+\-]\s*(function\s+\w+|const\s+\w+\s*=|\w+\s*=\s*function)`),
		"py_function":     regexp.MustCompile(`^[\+\-]\s*def\s+(\w+)`),
		"py_class":        regexp.MustCompile(`^[\+\-]\s*class\s+(\w+)`),
		"py_import":       regexp.MustCompile(`^[\+\-]\s*(import\s+\w+|from\s+\w+\s+import)`),
		"file_header":     regexp.MustCompile(`^diff --git a/(.+) b/(.+)`),
		"hunk_header":     regexp.MustCompile(`^@@\s+\-(\d+),?(\d*)\s+\+(\d+),?(\d*)\s+@@`),
	}

	return &DiffAnalyzer{
		patterns: patterns,
	}
}

// Analyze analyzes a git diff and extracts structural information
func (da *DiffAnalyzer) Analyze(ctx context.Context, diff string) (DiffAnalysis, error) {
	if ctx.Err() != nil {
		return DiffAnalysis{}, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("DiffAnalyzer.Analyze")
	}

	analysis := DiffAnalysis{
		AffectedFiles:     []string{},
		ModifiedFunctions: []FunctionChange{},
		AddedImports:      []string{},
		RemovedImports:    []string{},
		TypeChanges:       []TypeChange{},
	}

	lines := strings.Split(diff, "\n")
	currentFile := ""

	for i, line := range lines {
		// Check for context cancellation periodically
		if i%100 == 0 && ctx.Err() != nil {
			return DiffAnalysis{}, ctx.Err()
		}

		// Parse file headers
		if fileMatch := da.patterns["file_header"].FindStringSubmatch(line); fileMatch != nil {
			currentFile = fileMatch[1]
			analysis.AffectedFiles = append(analysis.AffectedFiles, currentFile)
			continue
		}

		// Count line additions and deletions
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			analysis.AddedLines++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			analysis.RemovedLines++
		}

		// Analyze language-specific changes
		if currentFile != "" {
			da.analyzeLineChange(line, currentFile, &analysis)
		}
	}

	return analysis, nil
}

// analyzeLineChange analyzes a single line change in context
func (da *DiffAnalyzer) analyzeLineChange(line, file string, analysis *DiffAnalysis) {
	fileExt := da.getFileExtension(file)

	switch fileExt {
	case ".go":
		da.analyzeGoChange(line, file, analysis)
	case ".js", ".ts":
		da.analyzeJSChange(line, file, analysis)
	case ".py":
		da.analyzePyChange(line, file, analysis)
	}
}

// analyzeGoChange analyzes Go-specific changes
func (da *DiffAnalyzer) analyzeGoChange(line, file string, analysis *DiffAnalysis) {
	// Function changes
	if funcMatch := da.patterns["go_function"].FindStringSubmatch(line); funcMatch != nil {
		funcName := funcMatch[1]
		changeType := "modified"
		if strings.HasPrefix(line, "+") {
			changeType = "added"
		} else if strings.HasPrefix(line, "-") {
			changeType = "removed"
		}

		// Look for existing function change or create new one
		existing := da.findFunctionChange(analysis.ModifiedFunctions, funcName)
		if existing != nil {
			if changeType == "removed" {
				existing.OldContent = line
			} else if changeType == "added" {
				existing.NewContent = line
			}
		} else {
			funcChange := FunctionChange{
				Name:       funcName,
				ChangeType: changeType,
			}
			if changeType == "removed" {
				funcChange.OldContent = line
			} else {
				funcChange.NewContent = line
			}
			analysis.ModifiedFunctions = append(analysis.ModifiedFunctions, funcChange)
		}
	}

	// Type changes
	if typeMatch := da.patterns["go_type"].FindStringSubmatch(line); typeMatch != nil {
		typeName := typeMatch[1]
		changeType := "modified"
		if strings.HasPrefix(line, "+") {
			changeType = "added"
		} else if strings.HasPrefix(line, "-") {
			changeType = "removed"
		}

		existing := da.findTypeChange(analysis.TypeChanges, typeName)
		if existing != nil {
			if changeType == "removed" {
				existing.OldDef = line
			} else if changeType == "added" {
				existing.NewDef = line
			}
		} else {
			typeChange := TypeChange{
				Name:       typeName,
				ChangeType: changeType,
			}
			if changeType == "removed" {
				typeChange.OldDef = line
			} else {
				typeChange.NewDef = line
			}
			analysis.TypeChanges = append(analysis.TypeChanges, typeChange)
		}
	}

	// Import changes
	if importMatch := da.patterns["go_import"].FindStringSubmatch(line); importMatch != nil {
		importPath := importMatch[1]
		if strings.HasPrefix(line, "+") {
			analysis.AddedImports = append(analysis.AddedImports, importPath)
		} else if strings.HasPrefix(line, "-") {
			analysis.RemovedImports = append(analysis.RemovedImports, importPath)
		}
	}
}

// analyzeJSChange analyzes JavaScript/TypeScript-specific changes
func (da *DiffAnalyzer) analyzeJSChange(line, file string, analysis *DiffAnalysis) {
	if funcMatch := da.patterns["js_function"].FindStringSubmatch(line); funcMatch != nil {
		// Extract function name from various JS function patterns
		funcName := da.extractJSFunctionName(funcMatch[1])
		if funcName != "" {
			changeType := "modified"
			if strings.HasPrefix(line, "+") {
				changeType = "added"
			} else if strings.HasPrefix(line, "-") {
				changeType = "removed"
			}

			existing := da.findFunctionChange(analysis.ModifiedFunctions, funcName)
			if existing != nil {
				if changeType == "removed" {
					existing.OldContent = line
				} else if changeType == "added" {
					existing.NewContent = line
				}
			} else {
				funcChange := FunctionChange{
					Name:       funcName,
					ChangeType: changeType,
				}
				if changeType == "removed" {
					funcChange.OldContent = line
				} else {
					funcChange.NewContent = line
				}
				analysis.ModifiedFunctions = append(analysis.ModifiedFunctions, funcChange)
			}
		}
	}
}

// analyzePyChange analyzes Python-specific changes
func (da *DiffAnalyzer) analyzePyChange(line, file string, analysis *DiffAnalysis) {
	// Function changes
	if funcMatch := da.patterns["py_function"].FindStringSubmatch(line); funcMatch != nil {
		funcName := funcMatch[1]
		changeType := "modified"
		if strings.HasPrefix(line, "+") {
			changeType = "added"
		} else if strings.HasPrefix(line, "-") {
			changeType = "removed"
		}

		existing := da.findFunctionChange(analysis.ModifiedFunctions, funcName)
		if existing != nil {
			if changeType == "removed" {
				existing.OldContent = line
			} else if changeType == "added" {
				existing.NewContent = line
			}
		} else {
			funcChange := FunctionChange{
				Name:       funcName,
				ChangeType: changeType,
			}
			if changeType == "removed" {
				funcChange.OldContent = line
			} else {
				funcChange.NewContent = line
			}
			analysis.ModifiedFunctions = append(analysis.ModifiedFunctions, funcChange)
		}
	}

	// Class changes
	if classMatch := da.patterns["py_class"].FindStringSubmatch(line); classMatch != nil {
		className := classMatch[1]
		changeType := "modified"
		if strings.HasPrefix(line, "+") {
			changeType = "added"
		} else if strings.HasPrefix(line, "-") {
			changeType = "removed"
		}

		existing := da.findTypeChange(analysis.TypeChanges, className)
		if existing != nil {
			if changeType == "removed" {
				existing.OldDef = line
			} else if changeType == "added" {
				existing.NewDef = line
			}
		} else {
			typeChange := TypeChange{
				Name:       className,
				ChangeType: changeType,
			}
			if changeType == "removed" {
				typeChange.OldDef = line
			} else {
				typeChange.NewDef = line
			}
			analysis.TypeChanges = append(analysis.TypeChanges, typeChange)
		}
	}

	// Import changes
	if da.patterns["py_import"].MatchString(line) {
		importStmt := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "+"), "-"))
		if strings.HasPrefix(line, "+") {
			analysis.AddedImports = append(analysis.AddedImports, importStmt)
		} else if strings.HasPrefix(line, "-") {
			analysis.RemovedImports = append(analysis.RemovedImports, importStmt)
		}
	}
}

// Helper methods

func (da *DiffAnalyzer) getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return "." + parts[len(parts)-1]
	}
	return ""
}

func (da *DiffAnalyzer) extractJSFunctionName(funcDecl string) string {
	// Handle different JS function patterns
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`function\s+(\w+)`),
		regexp.MustCompile(`const\s+(\w+)\s*=`),
		regexp.MustCompile(`(\w+)\s*=\s*function`),
		regexp.MustCompile(`(\w+)\s*:\s*function`),
	}

	for _, pattern := range patterns {
		if match := pattern.FindStringSubmatch(funcDecl); match != nil && len(match) > 1 {
			return match[1]
		}
	}

	return ""
}

func (da *DiffAnalyzer) findFunctionChange(changes []FunctionChange, name string) *FunctionChange {
	for i := range changes {
		if changes[i].Name == name {
			return &changes[i]
		}
	}
	return nil
}

func (da *DiffAnalyzer) findTypeChange(changes []TypeChange, name string) *TypeChange {
	for i := range changes {
		if changes[i].Name == name {
			return &changes[i]
		}
	}
	return nil
}

// CalculateComplexity estimates the complexity of changes in a diff
func (da *DiffAnalyzer) CalculateComplexity(analysis DiffAnalysis) DiffComplexity {
	complexity := DiffComplexity{
		Score:       0,
		Level:       "low",
		Factors:     make(map[string]int),
		Description: "",
	}

	// File count factor
	fileCount := len(analysis.AffectedFiles)
	complexity.Factors["files"] = fileCount
	complexity.Score += fileCount * 2

	// Function changes factor
	funcCount := len(analysis.ModifiedFunctions)
	complexity.Factors["functions"] = funcCount
	complexity.Score += funcCount * 5

	// Type changes factor (higher weight as they can be breaking)
	typeCount := len(analysis.TypeChanges)
	complexity.Factors["types"] = typeCount
	complexity.Score += typeCount * 8

	// Import changes factor
	importCount := len(analysis.AddedImports) + len(analysis.RemovedImports)
	complexity.Factors["imports"] = importCount
	complexity.Score += importCount * 3

	// Line changes factor
	lineChanges := analysis.AddedLines + analysis.RemovedLines
	complexity.Factors["lines"] = lineChanges
	complexity.Score += lineChanges / 10 // Scale down line changes

	// Determine complexity level
	if complexity.Score < 20 {
		complexity.Level = "low"
		complexity.Description = "Minor changes with limited scope"
	} else if complexity.Score < 50 {
		complexity.Level = "medium"
		complexity.Description = "Moderate changes affecting multiple components"
	} else if complexity.Score < 100 {
		complexity.Level = "high"
		complexity.Description = "Significant changes with broad impact"
	} else {
		complexity.Level = "critical"
		complexity.Description = "Major refactoring or architectural changes"
	}

	return complexity
}

// DiffComplexity represents the complexity analysis of a diff
type DiffComplexity struct {
	Score       int            `json:"score"`
	Level       string         `json:"level"`
	Factors     map[string]int `json:"factors"`
	Description string         `json:"description"`
}

// AnalyzeDiffPatterns identifies patterns in the diff that might indicate specific types of changes
func (da *DiffAnalyzer) AnalyzeDiffPatterns(analysis DiffAnalysis) []DiffPattern {
	var patterns []DiffPattern

	// Refactoring patterns
	if da.isRefactoringPattern(analysis) {
		patterns = append(patterns, DiffPattern{
			Type:        "refactoring",
			Confidence:  0.8,
			Description: "Code refactoring detected",
			Evidence:    da.getRefactoringEvidence(analysis),
		})
	}

	// API change patterns
	if da.isAPIChangePattern(analysis) {
		patterns = append(patterns, DiffPattern{
			Type:        "api_change",
			Confidence:  0.9,
			Description: "API changes detected",
			Evidence:    da.getAPIChangeEvidence(analysis),
		})
	}

	// Bug fix patterns
	if da.isBugFixPattern(analysis) {
		patterns = append(patterns, DiffPattern{
			Type:        "bug_fix",
			Confidence:  0.7,
			Description: "Bug fix pattern detected",
			Evidence:    da.getBugFixEvidence(analysis),
		})
	}

	return patterns
}

// Pattern detection methods
func (da *DiffAnalyzer) isRefactoringPattern(analysis DiffAnalysis) bool {
	// High function change to file ratio
	if len(analysis.AffectedFiles) > 0 {
		ratio := float64(len(analysis.ModifiedFunctions)) / float64(len(analysis.AffectedFiles))
		return ratio > 2.0
	}
	return false
}

func (da *DiffAnalyzer) isAPIChangePattern(analysis DiffAnalysis) bool {
	// Type changes or significant function signature changes
	return len(analysis.TypeChanges) > 0 || len(analysis.ModifiedFunctions) > 5
}

func (da *DiffAnalyzer) isBugFixPattern(analysis DiffAnalysis) bool {
	// Small, focused changes
	return len(analysis.AffectedFiles) <= 3 && len(analysis.ModifiedFunctions) <= 2
}

func (da *DiffAnalyzer) getRefactoringEvidence(analysis DiffAnalysis) []string {
	evidence := []string{
		fmt.Sprintf("%d functions modified", len(analysis.ModifiedFunctions)),
		fmt.Sprintf("%d files affected", len(analysis.AffectedFiles)),
	}
	if len(analysis.TypeChanges) > 0 {
		evidence = append(evidence, fmt.Sprintf("%d type changes", len(analysis.TypeChanges)))
	}
	return evidence
}

func (da *DiffAnalyzer) getAPIChangeEvidence(analysis DiffAnalysis) []string {
	evidence := []string{}
	if len(analysis.TypeChanges) > 0 {
		evidence = append(evidence, fmt.Sprintf("%d type definitions changed", len(analysis.TypeChanges)))
	}
	if len(analysis.ModifiedFunctions) > 0 {
		evidence = append(evidence, fmt.Sprintf("%d function signatures modified", len(analysis.ModifiedFunctions)))
	}
	return evidence
}

func (da *DiffAnalyzer) getBugFixEvidence(analysis DiffAnalysis) []string {
	return []string{
		fmt.Sprintf("Small scope: %d files", len(analysis.AffectedFiles)),
		fmt.Sprintf("Focused changes: %d functions", len(analysis.ModifiedFunctions)),
	}
}

// DiffPattern represents a detected pattern in a diff
type DiffPattern struct {
	Type        string   `json:"type"`
	Confidence  float64  `json:"confidence"`
	Description string   `json:"description"`
	Evidence    []string `json:"evidence"`
}
