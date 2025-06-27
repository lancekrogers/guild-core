// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package code

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/tools"
)

// ASTTool provides abstract syntax tree parsing and analysis
type ASTTool struct {
	*tools.BaseTool
	registry *ParserRegistry
}

// ASTParams represents the input parameters for AST analysis
type ASTParams struct {
	File       string `json:"file"`
	Query      string `json:"query"`                 // What to find: functions, classes, imports, symbol:<name>
	Language   string `json:"language,omitempty"`    // Auto-detect if not specified
	IncludeDoc bool   `json:"include_doc,omitempty"` // Include documentation strings
	Recursive  bool   `json:"recursive,omitempty"`   // For directory analysis
	Filter     string `json:"filter,omitempty"`      // Additional filtering
}

// ASTResult represents the result of AST analysis
type ASTResult struct {
	File      string       `json:"file"`
	Language  string       `json:"language"`
	Query     string       `json:"query"`
	Functions []*Function  `json:"functions,omitempty"`
	Classes   []*Class     `json:"classes,omitempty"`
	Imports   []*Import    `json:"imports,omitempty"`
	Symbols   []*Symbol    `json:"symbols,omitempty"`
	Errors    []ParseError `json:"errors,omitempty"`
	Summary   *ASTSummary  `json:"summary"`
}

// ASTSummary provides a summary of the AST analysis
type ASTSummary struct {
	TotalFunctions int  `json:"total_functions"`
	TotalClasses   int  `json:"total_classes"`
	TotalImports   int  `json:"total_imports"`
	TotalSymbols   int  `json:"total_symbols"`
	HasErrors      bool `json:"has_errors"`
	LinesAnalyzed  int  `json:"lines_analyzed"`
}

// NewASTTool creates a new AST analysis tool without parsers
// Use NewASTToolWithParsers for a ready-to-use tool
func NewASTTool() *ASTTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file": map[string]interface{}{
				"type":        "string",
				"description": "File path to analyze (can be directory for recursive analysis)",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"functions", "classes", "imports", "all", "symbol"},
				"description": "What to extract from the AST",
			},
			"language": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"go", "python", "typescript", "javascript", "rust", "java", "csharp", "cpp", "ruby", "php"},
				"description": "Programming language (auto-detected if not specified)",
			},
			"include_doc": map[string]interface{}{
				"type":        "boolean",
				"description": "Include documentation strings in results",
			},
			"recursive": map[string]interface{}{
				"type":        "boolean",
				"description": "Recursively analyze directory",
			},
			"filter": map[string]interface{}{
				"type":        "string",
				"description": "Additional filter (e.g., symbol name for symbol query)",
			},
		},
		"required": []string{"file", "query"},
	}

	examples := []string{
		`{"file": "main.go", "query": "functions", "include_doc": true}`,
		`{"file": "src/", "query": "classes", "recursive": true}`,
		`{"file": "utils.py", "query": "imports"}`,
		`{"file": "app.ts", "query": "symbol", "filter": "MyClass"}`,
		`{"file": ".", "query": "all", "recursive": true}`,
	}

	baseTool := tools.NewBaseTool(
		"ast",
		"Parse and analyze code structure using Abstract Syntax Trees. Extract functions, classes, imports, and find specific symbols.",
		schema,
		"code",
		false,
		examples,
	)

	tool := &ASTTool{
		BaseTool: baseTool,
		registry: NewParserRegistry(),
	}

	// Note: Parsers must be registered externally to avoid import cycles
	// See init.go or main.go for parser registration

	return tool
}

// RegisterParser adds a parser to the AST tool
func (t *ASTTool) RegisterParser(lang Language, parser Parser) {
	t.registry.Register(lang, parser)
}

// GetRegistry returns the parser registry for external registration
func (t *ASTTool) GetRegistry() *ParserRegistry {
	return t.registry
}

// Execute runs the AST tool with the given input
func (t *ASTTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params ASTParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("ast_tool").
			WithOperation("execute")
	}

	// Validate required parameters
	if params.File == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "file path is required", nil).
			WithComponent("ast_tool").
			WithOperation("execute")
	}

	if params.Query == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "query is required", nil).
			WithComponent("ast_tool").
			WithOperation("execute")
	}

	// Check if file/directory exists
	fileInfo, err := os.Stat(params.File)
	if os.IsNotExist(err) {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "file or directory does not exist: %s", params.File).
			WithComponent("ast_tool").
			WithOperation("execute")
	}

	var results []*ASTResult
	metadata := map[string]string{
		"file":  params.File,
		"query": params.Query,
	}

	if fileInfo.IsDir() {
		if params.Recursive {
			results, err = t.analyzeDirectory(ctx, params)
		} else {
			return nil, gerror.New(gerror.ErrCodeInvalidInput, "directory analysis requires recursive=true", nil).
				WithComponent("ast_tool").
				WithOperation("execute")
		}
	} else {
		result, analyzeErr := t.analyzeFile(ctx, params)
		if analyzeErr != nil {
			err = analyzeErr
		} else {
			results = []*ASTResult{result}
		}
	}

	if err != nil {
		return tools.NewToolResult("", metadata, err, nil), err
	}

	// Format output
	output, err := t.formatResults(results, params.Query)
	if err != nil {
		return tools.NewToolResult("", metadata, err, nil), err
	}

	extraData := map[string]interface{}{
		"results": results,
		"count":   len(results),
	}

	return tools.NewToolResult(output, metadata, nil, extraData), nil
}

// analyzeFile analyzes a single file
func (t *ASTTool) analyzeFile(ctx context.Context, params ASTParams) (*ASTResult, error) {
	// Detect language if not specified
	language := Language(params.Language)
	if language == "" {
		language = DetectLanguage(params.File)
	}

	if !language.IsSupported() {
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "unsupported language: %s", language).
			WithComponent("ast_tool").
			WithOperation("analyze_file")
	}

	// Get parser for the language
	parser, exists := t.registry.Get(language)
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "no parser available for language: %s", language).
			WithComponent("ast_tool").
			WithOperation("analyze_file")
	}

	// Read file content
	content, err := os.ReadFile(params.File)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to read file").
			WithComponent("ast_tool").
			WithOperation("analyze_file")
	}

	// Parse the file
	parseResult, err := parser.Parse(ctx, params.File, content)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse file").
			WithComponent("ast_tool").
			WithOperation("analyze_file")
	}

	// Create result
	result := &ASTResult{
		File:     params.File,
		Language: string(language),
		Query:    params.Query,
		Errors:   parseResult.Errors,
		Summary:  &ASTSummary{},
	}

	// Extract requested information based on query
	switch params.Query {
	case "functions":
		functions, err := parser.GetFunctions(parseResult)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to extract functions").
				WithComponent("ast_tool").
				WithOperation("analyze_file")
		}
		result.Functions = functions
		result.Summary.TotalFunctions = len(functions)

	case "classes":
		classes, err := parser.GetClasses(parseResult)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to extract classes").
				WithComponent("ast_tool").
				WithOperation("analyze_file")
		}
		result.Classes = classes
		result.Summary.TotalClasses = len(classes)

	case "imports":
		imports, err := parser.GetImports(parseResult)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to extract imports").
				WithComponent("ast_tool").
				WithOperation("analyze_file")
		}
		result.Imports = imports
		result.Summary.TotalImports = len(imports)

	case "all":
		// Get all information
		functions, err := parser.GetFunctions(parseResult)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to extract functions").
				WithComponent("ast_tool").
				WithOperation("analyze_file")
		}
		result.Functions = functions
		result.Summary.TotalFunctions = len(functions)

		classes, err := parser.GetClasses(parseResult)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to extract classes").
				WithComponent("ast_tool").
				WithOperation("analyze_file")
		}
		result.Classes = classes
		result.Summary.TotalClasses = len(classes)

		imports, err := parser.GetImports(parseResult)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to extract imports").
				WithComponent("ast_tool").
				WithOperation("analyze_file")
		}
		result.Imports = imports
		result.Summary.TotalImports = len(imports)

	case "symbol":
		if params.Filter == "" {
			return nil, gerror.New(gerror.ErrCodeInvalidInput, "symbol query requires filter parameter with symbol name", nil).
				WithComponent("ast_tool").
				WithOperation("analyze_file")
		}
		symbols, err := parser.FindSymbol(parseResult, params.Filter)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to find symbol").
				WithComponent("ast_tool").
				WithOperation("analyze_file")
		}
		result.Symbols = symbols
		result.Summary.TotalSymbols = len(symbols)

	default:
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "unsupported query: %s", params.Query).
			WithComponent("ast_tool").
			WithOperation("analyze_file")
	}

	// Update summary
	result.Summary.HasErrors = len(parseResult.Errors) > 0
	result.Summary.LinesAnalyzed = len(strings.Split(string(content), "\n"))

	return result, nil
}

// analyzeDirectory recursively analyzes a directory
func (t *ASTTool) analyzeDirectory(ctx context.Context, params ASTParams) ([]*ASTResult, error) {
	var results []*ASTResult

	err := filepath.Walk(params.File, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file has supported extension
		language := DetectLanguage(path)
		if !language.IsSupported() {
			return nil // Skip unsupported files
		}

		// Create params for this file
		fileParams := params
		fileParams.File = path

		// Analyze the file
		result, err := t.analyzeFile(ctx, fileParams)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("Warning: failed to analyze %s: %v\n", path, err)
			return nil
		}

		results = append(results, result)
		return nil
	})

	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to walk directory").
			WithComponent("ast_tool").
			WithOperation("analyze_directory")
	}

	return results, nil
}

// formatResults formats the analysis results for output
func (t *ASTTool) formatResults(results []*ASTResult, query string) (string, error) {
	if len(results) == 0 {
		return "No results found.", nil
	}

	var output strings.Builder

	// Write summary
	totalFunctions := 0
	totalClasses := 0
	totalImports := 0
	totalSymbols := 0
	totalErrors := 0

	for _, result := range results {
		totalFunctions += result.Summary.TotalFunctions
		totalClasses += result.Summary.TotalClasses
		totalImports += result.Summary.TotalImports
		totalSymbols += result.Summary.TotalSymbols
		if result.Summary.HasErrors {
			totalErrors += len(result.Errors)
		}
	}

	output.WriteString(fmt.Sprintf("AST Analysis Results (%d files analyzed)\n", len(results)))
	output.WriteString(fmt.Sprintf("Query: %s\n", query))
	output.WriteString(fmt.Sprintf("Total Functions: %d\n", totalFunctions))
	output.WriteString(fmt.Sprintf("Total Classes: %d\n", totalClasses))
	output.WriteString(fmt.Sprintf("Total Imports: %d\n", totalImports))
	output.WriteString(fmt.Sprintf("Total Symbols: %d\n", totalSymbols))
	if totalErrors > 0 {
		output.WriteString(fmt.Sprintf("Parse Errors: %d\n", totalErrors))
	}
	output.WriteString("\n")

	// Write detailed results for each file
	for _, result := range results {
		output.WriteString(fmt.Sprintf("File: %s (%s)\n", result.File, result.Language))

		if len(result.Functions) > 0 {
			output.WriteString(fmt.Sprintf("  Functions (%d):\n", len(result.Functions)))
			for _, fn := range result.Functions {
				visibility := ""
				if fn.Visibility != "" && fn.Visibility != "public" {
					visibility = fmt.Sprintf(" [%s]", fn.Visibility)
				}
				output.WriteString(fmt.Sprintf("    - %s%s (line %d)\n", fn.Signature, visibility, fn.StartLine))
				if fn.DocString != "" {
					output.WriteString(fmt.Sprintf("      %s\n", strings.TrimSpace(fn.DocString)))
				}
			}
		}

		if len(result.Classes) > 0 {
			output.WriteString(fmt.Sprintf("  Classes (%d):\n", len(result.Classes)))
			for _, cls := range result.Classes {
				output.WriteString(fmt.Sprintf("    - %s (line %d)\n", cls.Name, cls.StartLine))
				if cls.DocString != "" {
					output.WriteString(fmt.Sprintf("      %s\n", strings.TrimSpace(cls.DocString)))
				}
				if len(cls.Methods) > 0 {
					output.WriteString(fmt.Sprintf("      Methods: %d\n", len(cls.Methods)))
				}
			}
		}

		if len(result.Imports) > 0 {
			output.WriteString(fmt.Sprintf("  Imports (%d):\n", len(result.Imports)))
			for _, imp := range result.Imports {
				alias := ""
				if imp.Alias != "" {
					alias = fmt.Sprintf(" as %s", imp.Alias)
				}
				output.WriteString(fmt.Sprintf("    - %s%s (line %d)\n", imp.Path, alias, imp.Line))
			}
		}

		if len(result.Symbols) > 0 {
			output.WriteString(fmt.Sprintf("  Symbols (%d):\n", len(result.Symbols)))
			for _, sym := range result.Symbols {
				output.WriteString(fmt.Sprintf("    - %s [%s] (line %d)\n", sym.Name, sym.Type, sym.StartLine))
			}
		}

		if len(result.Errors) > 0 {
			output.WriteString(fmt.Sprintf("  Parse Errors (%d):\n", len(result.Errors)))
			for _, err := range result.Errors {
				output.WriteString(fmt.Sprintf("    - Line %d: %s\n", err.Line, err.Message))
			}
		}

		output.WriteString("\n")
	}

	return output.String(), nil
}
