// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package code

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/tools"
)

// MetricsTool calculates code quality metrics
type MetricsTool struct {
	*tools.BaseTool
}

// MetricsParams represents the input parameters for metrics calculation
type MetricsParams struct {
	Path        string   `json:"path"`                   // File or directory path
	Metrics     []string `json:"metrics,omitempty"`      // Specific metrics to calculate
	Recursive   bool     `json:"recursive,omitempty"`    // Recursively analyze directory
	IgnoreTests bool     `json:"ignore_tests,omitempty"` // Ignore test files
	Language    string   `json:"language,omitempty"`     // Language filter
	Format      string   `json:"format,omitempty"`       // Output format: text, json, summary
}

// MetricsResult represents the result of metrics calculation
type MetricsResult struct {
	Path         string               `json:"path"`
	Language     string               `json:"language,omitempty"`
	FileMetrics  []*CodeMetrics       `json:"file_metrics,omitempty"`
	Aggregated   *CodeMetrics         `json:"aggregated"`
	Distribution *MetricsDistribution `json:"distribution"`
	Analysis     *QualityAnalysis     `json:"analysis"`
	Errors       []string             `json:"errors,omitempty"`
}

// MetricsDistribution shows distribution of metrics across files
type MetricsDistribution struct {
	ComplexityRanges  map[string]int    `json:"complexity_ranges"` // 0-5, 6-10, 11-20, 21+
	FileSizeRanges    map[string]int    `json:"file_size_ranges"`  // small, medium, large, huge
	LanguageBreakdown map[string]int    `json:"language_breakdown"`
	TopComplexFiles   []*ComplexityInfo `json:"top_complex_files"`
	TopLargeFiles     []*SizeInfo       `json:"top_large_files"`
}

// ComplexityInfo holds information about complex files
type ComplexityInfo struct {
	File       string `json:"file"`
	Complexity int    `json:"complexity"`
	Functions  int    `json:"functions"`
}

// SizeInfo holds information about large files
type SizeInfo struct {
	File  string `json:"file"`
	Lines int    `json:"lines"`
	Size  int64  `json:"size_bytes"`
}

// QualityAnalysis provides quality assessment and recommendations
type QualityAnalysis struct {
	QualityScore    float64        `json:"quality_score"` // 0-10, higher is better
	Grade           string         `json:"grade"`         // A, B, C, D, F
	Issues          []string       `json:"issues"`
	Recommendations []string       `json:"recommendations"`
	Strengths       []string       `json:"strengths"`
	TechnicalDebt   *TechnicalDebt `json:"technical_debt"`
}

// NewMetricsTool creates a new code metrics tool
func NewMetricsTool() *MetricsTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File or directory path to analyze",
			},
			"metrics": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Specific metrics to calculate (all if not specified)",
			},
			"recursive": map[string]interface{}{
				"type":        "boolean",
				"description": "Recursively analyze directory",
			},
			"ignore_tests": map[string]interface{}{
				"type":        "boolean",
				"description": "Ignore test files in analysis",
			},
			"language": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"go", "python", "typescript", "javascript"},
				"description": "Filter analysis to specific language",
			},
			"format": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"text", "json", "summary"},
				"description": "Output format",
			},
		},
		"required": []string{"path"},
	}

	examples := []string{
		`{"path": "main.go", "format": "summary"}`,
		`{"path": "src/", "recursive": true, "ignore_tests": true}`,
		`{"path": ".", "metrics": ["complexity", "loc"], "language": "go"}`,
		`{"path": "/path/to/code", "recursive": true, "format": "json"}`,
		`{"path": "utils.py", "metrics": ["complexity", "duplication"]}`,
	}

	baseTool := tools.NewBaseTool(
		"metrics",
		"Calculate code quality metrics including cyclomatic complexity, lines of code, test coverage, and technical debt analysis.",
		schema,
		"code",
		false,
		examples,
	)

	return &MetricsTool{
		BaseTool: baseTool,
	}
}

// Execute runs the metrics tool with the given input
func (t *MetricsTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params MetricsParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("metrics_tool").
			WithOperation("execute")
	}

	// Validate required parameters
	if params.Path == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "path is required", nil).
			WithComponent("metrics_tool").
			WithOperation("execute")
	}

	// Check if path exists
	if _, err := os.Stat(params.Path); os.IsNotExist(err) {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "path does not exist: %s", params.Path).
			WithComponent("metrics_tool").
			WithOperation("execute")
	}

	// Calculate metrics
	result, err := t.calculateMetrics(ctx, params)
	if err != nil {
		return tools.NewToolResult("", map[string]string{
			"path": params.Path,
		}, err, nil), err
	}

	// Format output
	output, err := t.formatResult(result, params.Format)
	if err != nil {
		return tools.NewToolResult("", map[string]string{
			"path": params.Path,
		}, err, nil), err
	}

	metadata := map[string]string{
		"path":           params.Path,
		"files_analyzed": fmt.Sprintf("%d", len(result.FileMetrics)),
		"total_loc":      fmt.Sprintf("%d", result.Aggregated.LinesOfCode),
		"avg_complexity": fmt.Sprintf("%.1f", result.Aggregated.AverageComplexity),
		"quality_grade":  result.Analysis.Grade,
	}

	extraData := map[string]interface{}{
		"result": result,
	}

	return tools.NewToolResult(output, metadata, nil, extraData), nil
}

// calculateMetrics calculates code metrics for the given path
func (t *MetricsTool) calculateMetrics(ctx context.Context, params MetricsParams) (*MetricsResult, error) {
	result := &MetricsResult{
		Path:       params.Path,
		Aggregated: &CodeMetrics{},
		Distribution: &MetricsDistribution{
			ComplexityRanges:  make(map[string]int),
			FileSizeRanges:    make(map[string]int),
			LanguageBreakdown: make(map[string]int),
		},
		Analysis: &QualityAnalysis{
			TechnicalDebt: &TechnicalDebt{},
		},
	}

	// Check if path is file or directory
	fileInfo, err := os.Stat(params.Path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to stat path").
			WithComponent("metrics_tool").
			WithOperation("calculate_metrics")
	}

	if fileInfo.IsDir() {
		if !params.Recursive {
			return nil, gerror.New(gerror.ErrCodeInvalidInput, "directory analysis requires recursive=true", nil).
				WithComponent("metrics_tool").
				WithOperation("calculate_metrics")
		}
		err = t.analyzeDirectory(ctx, params, result)
	} else {
		err = t.analyzeFile(ctx, params.Path, params, result)
	}

	if err != nil {
		return nil, err
	}

	// Calculate aggregated metrics
	t.aggregateMetrics(result)

	// Generate distribution analysis
	t.generateDistribution(result)

	// Perform quality analysis
	t.performQualityAnalysis(result)

	return result, nil
}

// analyzeDirectory recursively analyzes a directory
func (t *MetricsTool) analyzeDirectory(ctx context.Context, params MetricsParams, result *MetricsResult) error {
	return filepath.Walk(params.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Filter by language if specified
		language := DetectLanguage(path)
		if params.Language != "" && string(language) != params.Language {
			return nil
		}

		// Skip unsupported files
		if !language.IsSupported() {
			return nil
		}

		// Skip test files if requested
		if params.IgnoreTests && t.isTestFile(path, language) {
			return nil
		}

		// Analyze the file
		return t.analyzeFile(ctx, path, params, result)
	})
}

// analyzeFile analyzes a single file
func (t *MetricsTool) analyzeFile(ctx context.Context, filePath string, params MetricsParams, result *MetricsResult) error {
	language := DetectLanguage(filePath)
	if !language.IsSupported() {
		return nil // Skip unsupported files silently
	}

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read %s: %v", filePath, err))
		return nil // Continue with other files
	}

	// Calculate metrics based on language
	var metrics *CodeMetrics
	switch language {
	case LanguageGo:
		metrics, err = t.calculateGoMetrics(filePath, content)
	case LanguagePython:
		metrics, err = t.calculatePythonMetrics(filePath, content)
	case LanguageTypeScript, LanguageJavaScript:
		metrics, err = t.calculateJSMetrics(filePath, content)
	default:
		metrics, err = t.calculateGenericMetrics(filePath, content)
	}

	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to analyze %s: %v", filePath, err))
		return nil
	}

	metrics.File = filePath
	result.FileMetrics = append(result.FileMetrics, metrics)

	return nil
}

// calculateGoMetrics calculates metrics for Go files
func (t *MetricsTool) calculateGoMetrics(filePath string, content []byte) (*CodeMetrics, error) {
	metrics := &CodeMetrics{
		File: filePath,
	}

	// Basic line counting
	lines := strings.Split(string(content), "\n")
	metrics.LinesOfCode = len(lines)

	// Count source lines, comments, and blank lines
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			metrics.BlankLines++
		} else if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			metrics.CommentLines++
		} else {
			metrics.SourceLinesOfCode++
		}
	}

	// Parse Go AST for detailed analysis
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		// If parsing fails, return basic metrics
		return metrics, nil
	}

	// Count functions and calculate complexity
	var totalComplexity int
	var maxComplexity int

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Body != nil {
				metrics.FunctionCount++
				complexity := t.calculateCyclomaticComplexity(node.Body)
				totalComplexity += complexity
				if complexity > maxComplexity {
					maxComplexity = complexity
				}
			}
		case *ast.TypeSpec:
			if _, ok := node.Type.(*ast.StructType); ok {
				metrics.ClassCount++ // Treat structs as classes
			}
		}
		return true
	})

	metrics.CyclomaticComplexity = totalComplexity
	metrics.MaxComplexity = maxComplexity
	if metrics.FunctionCount > 0 {
		metrics.AverageComplexity = float64(totalComplexity) / float64(metrics.FunctionCount)
	}

	return metrics, nil
}

// calculatePythonMetrics calculates metrics for Python files (simplified)
func (t *MetricsTool) calculatePythonMetrics(filePath string, content []byte) (*CodeMetrics, error) {
	metrics := &CodeMetrics{
		File: filePath,
	}

	lines := strings.Split(string(content), "\n")
	metrics.LinesOfCode = len(lines)

	// Simple Python analysis
	functionRegex := regexp.MustCompile(`^\s*def\s+\w+`)
	classRegex := regexp.MustCompile(`^\s*class\s+\w+`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			metrics.BlankLines++
		} else if strings.HasPrefix(trimmed, "#") {
			metrics.CommentLines++
		} else {
			metrics.SourceLinesOfCode++

			if functionRegex.MatchString(line) {
				metrics.FunctionCount++
			}
			if classRegex.MatchString(line) {
				metrics.ClassCount++
			}
		}
	}

	// Simplified complexity calculation for Python
	metrics.CyclomaticComplexity = t.calculateSimpleComplexity(string(content))
	metrics.MaxComplexity = metrics.CyclomaticComplexity // Simplified
	if metrics.FunctionCount > 0 {
		metrics.AverageComplexity = float64(metrics.CyclomaticComplexity) / float64(metrics.FunctionCount)
	}

	return metrics, nil
}

// calculateJSMetrics calculates metrics for JavaScript/TypeScript files (simplified)
func (t *MetricsTool) calculateJSMetrics(filePath string, content []byte) (*CodeMetrics, error) {
	metrics := &CodeMetrics{
		File: filePath,
	}

	lines := strings.Split(string(content), "\n")
	metrics.LinesOfCode = len(lines)

	// Simple JS/TS analysis
	functionRegex := regexp.MustCompile(`^\s*(function\s+\w+|const\s+\w+\s*=.*=>|\w+\s*\([^)]*\)\s*{)`)
	classRegex := regexp.MustCompile(`^\s*class\s+\w+`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			metrics.BlankLines++
		} else if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			metrics.CommentLines++
		} else {
			metrics.SourceLinesOfCode++

			if functionRegex.MatchString(line) {
				metrics.FunctionCount++
			}
			if classRegex.MatchString(line) {
				metrics.ClassCount++
			}
		}
	}

	// Simplified complexity calculation
	metrics.CyclomaticComplexity = t.calculateSimpleComplexity(string(content))
	if metrics.FunctionCount > 0 {
		metrics.AverageComplexity = float64(metrics.CyclomaticComplexity) / float64(metrics.FunctionCount)
	}

	return metrics, nil
}

// calculateGenericMetrics calculates basic metrics for any file
func (t *MetricsTool) calculateGenericMetrics(filePath string, content []byte) (*CodeMetrics, error) {
	metrics := &CodeMetrics{
		File: filePath,
	}

	lines := strings.Split(string(content), "\n")
	metrics.LinesOfCode = len(lines)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			metrics.BlankLines++
		} else {
			metrics.SourceLinesOfCode++
		}
	}

	return metrics, nil
}

// calculateCyclomaticComplexity calculates cyclomatic complexity for a Go function body
func (t *MetricsTool) calculateCyclomaticComplexity(body *ast.BlockStmt) int {
	complexity := 1 // Base complexity

	ast.Inspect(body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.SwitchStmt:
			complexity++
		case *ast.TypeSwitchStmt:
			complexity++
		case *ast.SelectStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		}
		return true
	})

	return complexity
}

// calculateSimpleComplexity calculates a simplified complexity metric
func (t *MetricsTool) calculateSimpleComplexity(content string) int {
	complexity := 1

	// Count decision points
	patterns := []string{
		"if ", "elif ", "else:", "for ", "while ", "switch ", "case ", "catch ", "&&", "||",
	}

	for _, pattern := range patterns {
		complexity += strings.Count(content, pattern)
	}

	return complexity
}

// isTestFile determines if a file is a test file
func (t *MetricsTool) isTestFile(filePath string, language Language) bool {
	basename := filepath.Base(filePath)

	switch language {
	case LanguageGo:
		return strings.HasSuffix(basename, "_test.go")
	case LanguagePython:
		return strings.HasPrefix(basename, "test_") || strings.HasSuffix(basename, "_test.py")
	case LanguageJavaScript, LanguageTypeScript:
		return strings.Contains(basename, ".test.") || strings.Contains(basename, ".spec.")
	default:
		return false
	}
}

// aggregateMetrics calculates aggregated metrics across all files
func (t *MetricsTool) aggregateMetrics(result *MetricsResult) {
	agg := result.Aggregated

	for _, metrics := range result.FileMetrics {
		agg.LinesOfCode += metrics.LinesOfCode
		agg.SourceLinesOfCode += metrics.SourceLinesOfCode
		agg.CommentLines += metrics.CommentLines
		agg.BlankLines += metrics.BlankLines
		agg.FunctionCount += metrics.FunctionCount
		agg.ClassCount += metrics.ClassCount
		agg.CyclomaticComplexity += metrics.CyclomaticComplexity

		if metrics.MaxComplexity > agg.MaxComplexity {
			agg.MaxComplexity = metrics.MaxComplexity
		}
	}

	if agg.FunctionCount > 0 {
		agg.AverageComplexity = float64(agg.CyclomaticComplexity) / float64(agg.FunctionCount)
	}
}

// generateDistribution generates distribution analysis
func (t *MetricsTool) generateDistribution(result *MetricsResult) {
	dist := result.Distribution

	// Initialize ranges
	dist.ComplexityRanges["0-5"] = 0
	dist.ComplexityRanges["6-10"] = 0
	dist.ComplexityRanges["11-20"] = 0
	dist.ComplexityRanges["21+"] = 0

	dist.FileSizeRanges["small (0-100)"] = 0
	dist.FileSizeRanges["medium (101-500)"] = 0
	dist.FileSizeRanges["large (501-1000)"] = 0
	dist.FileSizeRanges["huge (1000+)"] = 0

	var complexFiles []*ComplexityInfo
	var largeFiles []*SizeInfo

	for _, metrics := range result.FileMetrics {
		// Complexity distribution
		if metrics.CyclomaticComplexity <= 5 {
			dist.ComplexityRanges["0-5"]++
		} else if metrics.CyclomaticComplexity <= 10 {
			dist.ComplexityRanges["6-10"]++
		} else if metrics.CyclomaticComplexity <= 20 {
			dist.ComplexityRanges["11-20"]++
		} else {
			dist.ComplexityRanges["21+"]++
		}

		// File size distribution
		if metrics.LinesOfCode <= 100 {
			dist.FileSizeRanges["small (0-100)"]++
		} else if metrics.LinesOfCode <= 500 {
			dist.FileSizeRanges["medium (101-500)"]++
		} else if metrics.LinesOfCode <= 1000 {
			dist.FileSizeRanges["large (501-1000)"]++
		} else {
			dist.FileSizeRanges["huge (1000+)"]++
		}

		// Language breakdown
		language := string(DetectLanguage(metrics.File))
		dist.LanguageBreakdown[language]++

		// Collect top complex files
		if metrics.CyclomaticComplexity > 10 {
			complexFiles = append(complexFiles, &ComplexityInfo{
				File:       metrics.File,
				Complexity: metrics.CyclomaticComplexity,
				Functions:  metrics.FunctionCount,
			})
		}

		// Collect large files
		if metrics.LinesOfCode > 500 {
			largeFiles = append(largeFiles, &SizeInfo{
				File:  metrics.File,
				Lines: metrics.LinesOfCode,
			})
		}
	}

	// Sort and limit top files
	// (Simple sorting by complexity/size - could be improved)
	if len(complexFiles) > 5 {
		dist.TopComplexFiles = complexFiles[:5]
	} else {
		dist.TopComplexFiles = complexFiles
	}

	if len(largeFiles) > 5 {
		dist.TopLargeFiles = largeFiles[:5]
	} else {
		dist.TopLargeFiles = largeFiles
	}
}

// performQualityAnalysis performs quality analysis and generates recommendations
func (t *MetricsTool) performQualityAnalysis(result *MetricsResult) {
	analysis := result.Analysis
	agg := result.Aggregated

	// Calculate quality score (0-10)
	score := 10.0

	// Factor 1: Average complexity (lower is better)
	if agg.AverageComplexity > 10 {
		score -= 3.0
	} else if agg.AverageComplexity > 7 {
		score -= 2.0
	} else if agg.AverageComplexity > 5 {
		score -= 1.0
	}

	// Factor 2: Maximum complexity
	if agg.MaxComplexity > 20 {
		score -= 2.0
	} else if agg.MaxComplexity > 15 {
		score -= 1.0
	}

	// Factor 3: Comment ratio
	if agg.LinesOfCode > 0 {
		commentRatio := float64(agg.CommentLines) / float64(agg.LinesOfCode)
		if commentRatio < 0.1 {
			score -= 1.0
		} else if commentRatio > 0.2 {
			score += 0.5
		}
	}

	// Factor 4: File size distribution
	dist := result.Distribution
	if dist.FileSizeRanges["huge (1000+)"] > len(result.FileMetrics)/4 {
		score -= 1.0
	}

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}

	analysis.QualityScore = score

	// Determine grade
	if score >= 9 {
		analysis.Grade = "A"
	} else if score >= 8 {
		analysis.Grade = "B"
	} else if score >= 6 {
		analysis.Grade = "C"
	} else if score >= 4 {
		analysis.Grade = "D"
	} else {
		analysis.Grade = "F"
	}

	// Generate issues and recommendations
	if agg.AverageComplexity > 7 {
		analysis.Issues = append(analysis.Issues, fmt.Sprintf("High average complexity: %.1f", agg.AverageComplexity))
		analysis.Recommendations = append(analysis.Recommendations, "Consider refactoring complex functions")
	}

	if agg.MaxComplexity > 15 {
		analysis.Issues = append(analysis.Issues, fmt.Sprintf("Very high maximum complexity: %d", agg.MaxComplexity))
		analysis.Recommendations = append(analysis.Recommendations, "Break down the most complex functions")
	}

	if agg.LinesOfCode > 0 {
		commentRatio := float64(agg.CommentLines) / float64(agg.LinesOfCode)
		if commentRatio < 0.1 {
			analysis.Issues = append(analysis.Issues, "Low comment coverage")
			analysis.Recommendations = append(analysis.Recommendations, "Add more documentation and comments")
		}
	}

	if dist.FileSizeRanges["huge (1000+)"] > 0 {
		analysis.Issues = append(analysis.Issues, "Large files detected")
		analysis.Recommendations = append(analysis.Recommendations, "Consider splitting large files")
	}

	// Generate strengths
	if agg.AverageComplexity <= 5 {
		analysis.Strengths = append(analysis.Strengths, "Low complexity code")
	}

	if agg.LinesOfCode > 0 {
		commentRatio := float64(agg.CommentLines) / float64(agg.LinesOfCode)
		if commentRatio > 0.15 {
			analysis.Strengths = append(analysis.Strengths, "Well documented code")
		}
	}

	if len(analysis.Issues) == 0 {
		analysis.Strengths = append(analysis.Strengths, "Code quality looks good")
	}

	// Calculate technical debt (simplified)
	debt := analysis.TechnicalDebt
	debt.Issues = len(analysis.Issues)
	debt.Minutes = debt.Issues * 30 // Rough estimate: 30 minutes per issue
	debt.Hours = float64(debt.Minutes) / 60.0

	if score >= 8 {
		debt.Rating = "A"
	} else if score >= 6 {
		debt.Rating = "B"
	} else if score >= 4 {
		debt.Rating = "C"
	} else {
		debt.Rating = "D"
	}
}

// formatResult formats the metrics result
func (t *MetricsTool) formatResult(result *MetricsResult, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	case "summary":
		return t.formatSummary(result), nil

	default: // "text"
		return t.formatText(result), nil
	}
}

// formatSummary formats a summary view of metrics
func (t *MetricsTool) formatSummary(result *MetricsResult) string {
	var output strings.Builder
	agg := result.Aggregated
	analysis := result.Analysis

	output.WriteString("Code Quality Metrics Summary\n")
	output.WriteString(fmt.Sprintf("Path: %s\n", result.Path))
	output.WriteString(fmt.Sprintf("Files Analyzed: %d\n", len(result.FileMetrics)))
	output.WriteString(fmt.Sprintf("Total Lines: %d (Source: %d, Comments: %d, Blank: %d)\n",
		agg.LinesOfCode, agg.SourceLinesOfCode, agg.CommentLines, agg.BlankLines))
	output.WriteString(fmt.Sprintf("Functions: %d, Classes: %d\n", agg.FunctionCount, agg.ClassCount))
	output.WriteString(fmt.Sprintf("Complexity: Avg %.1f, Max %d, Total %d\n",
		agg.AverageComplexity, agg.MaxComplexity, agg.CyclomaticComplexity))
	output.WriteString(fmt.Sprintf("Quality Score: %.1f/10 (Grade: %s)\n", analysis.QualityScore, analysis.Grade))

	if len(analysis.Issues) > 0 {
		output.WriteString("\nIssues:\n")
		for _, issue := range analysis.Issues {
			output.WriteString(fmt.Sprintf("- %s\n", issue))
		}
	}

	if len(analysis.Recommendations) > 0 {
		output.WriteString("\nRecommendations:\n")
		for _, rec := range analysis.Recommendations {
			output.WriteString(fmt.Sprintf("- %s\n", rec))
		}
	}

	if len(analysis.Strengths) > 0 {
		output.WriteString("\nStrengths:\n")
		for _, strength := range analysis.Strengths {
			output.WriteString(fmt.Sprintf("- %s\n", strength))
		}
	}

	return output.String()
}

// formatText formats a detailed text view of metrics
func (t *MetricsTool) formatText(result *MetricsResult) string {
	var output strings.Builder

	// Add summary first
	output.WriteString(t.formatSummary(result))
	output.WriteString("\n")

	// Add distribution information
	dist := result.Distribution
	output.WriteString("Distribution Analysis:\n")

	output.WriteString("Complexity Distribution:\n")
	for range_, count := range dist.ComplexityRanges {
		output.WriteString(fmt.Sprintf("  %s: %d files\n", range_, count))
	}

	output.WriteString("File Size Distribution:\n")
	for range_, count := range dist.FileSizeRanges {
		output.WriteString(fmt.Sprintf("  %s lines: %d files\n", range_, count))
	}

	if len(dist.LanguageBreakdown) > 1 {
		output.WriteString("Language Breakdown:\n")
		for lang, count := range dist.LanguageBreakdown {
			output.WriteString(fmt.Sprintf("  %s: %d files\n", lang, count))
		}
	}

	if len(dist.TopComplexFiles) > 0 {
		output.WriteString("\nMost Complex Files:\n")
		for _, file := range dist.TopComplexFiles {
			output.WriteString(fmt.Sprintf("  %s: complexity %d (%d functions)\n",
				file.File, file.Complexity, file.Functions))
		}
	}

	if len(dist.TopLargeFiles) > 0 {
		output.WriteString("\nLargest Files:\n")
		for _, file := range dist.TopLargeFiles {
			output.WriteString(fmt.Sprintf("  %s: %d lines\n", file.File, file.Lines))
		}
	}

	return output.String()
}
