// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package edit

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

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/tools"
	"github.com/lancekrogers/guild/tools/code"
)

// MultiFileRefactorTool provides coordinated refactoring across multiple files
type MultiFileRefactorTool struct {
	*tools.BaseTool
}

// RefactorParams represents the input parameters for multi-file refactoring
type RefactorParams struct {
	Type        string           `json:"type"`                  // rename, extract, move, inline
	Target      *RefactorTarget  `json:"target"`                // What to refactor
	NewName     string           `json:"new_name,omitempty"`    // New name for rename operations
	Destination string           `json:"destination,omitempty"` // Destination file for move operations
	Preview     bool             `json:"preview,omitempty"`     // Show preview before applying
	Scope       string           `json:"scope,omitempty"`       // file, package, project
	Language    string           `json:"language,omitempty"`    // Target language
	Options     *RefactorOptions `json:"options,omitempty"`     // Additional options
}

// RefactorTarget specifies what to refactor
type RefactorTarget struct {
	File      string `json:"file"`                 // Source file
	Line      int    `json:"line,omitempty"`       // Line number
	Column    int    `json:"column,omitempty"`     // Column number
	Symbol    string `json:"symbol,omitempty"`     // Symbol name
	StartLine int    `json:"start_line,omitempty"` // Start of selection
	EndLine   int    `json:"end_line,omitempty"`   // End of selection
	Type      string `json:"type,omitempty"`       // function, variable, type, etc.
}

// RefactorOptions provides additional refactoring options
type RefactorOptions struct {
	UpdateReferences bool     `json:"update_references"`       // Update all references
	UpdateImports    bool     `json:"update_imports"`          // Update import statements
	UpdateComments   bool     `json:"update_comments"`         // Update comments and docs
	IncludeTests     bool     `json:"include_tests"`           // Include test files
	BackupFiles      bool     `json:"backup_files"`            // Create backups
	FilePatterns     []string `json:"file_patterns,omitempty"` // File patterns to search
}

// RefactorResult represents the result of a multi-file refactoring operation
type RefactorResult struct {
	Type              string              `json:"type"`
	Applied           bool                `json:"applied"`
	Preview           bool                `json:"preview"`
	FilesChanged      []*FileChange       `json:"files_changed"`
	ReferencesFound   int                 `json:"references_found"`
	ReferencesUpdated int                 `json:"references_updated"`
	ImportsUpdated    []*ImportChange     `json:"imports_updated,omitempty"`
	MovedCode         *CodeMove           `json:"moved_code,omitempty"`
	Conflicts         []*RefactorConflict `json:"conflicts,omitempty"`
	Warnings          []string            `json:"warnings,omitempty"`
	Errors            []string            `json:"errors,omitempty"`
	Summary           string              `json:"summary"`
}

// FileChange represents changes made to a single file
type FileChange struct {
	File       string        `json:"file"`
	ChangeType string        `json:"change_type"` // modified, created, deleted
	Changes    []*LineChange `json:"changes"`
	BackupFile string        `json:"backup_file,omitempty"`
	NewContent string        `json:"new_content,omitempty"`
	Preview    string        `json:"preview,omitempty"`
}

// LineChange represents a change to a specific line
type LineChange struct {
	Line       int    `json:"line"`
	Type       string `json:"type"` // replace, insert, delete
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

// ImportChange represents changes to import statements
type ImportChange struct {
	File      string `json:"file"`
	OldImport string `json:"old_import,omitempty"`
	NewImport string `json:"new_import,omitempty"`
	Action    string `json:"action"` // add, remove, update
}

// CodeMove represents moved code information
type CodeMove struct {
	SourceFile      string `json:"source_file"`
	DestinationFile string `json:"destination_file"`
	MovedSymbol     string `json:"moved_symbol"`
	SourceLines     string `json:"source_lines"`
	InsertionPoint  int    `json:"insertion_point"`
}

// RefactorConflict represents a conflict during refactoring
type RefactorConflict struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Type        string `json:"type"` // name_collision, syntax_error, etc.
	Description string `json:"description"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// NewMultiFileRefactorTool creates a new multi-file refactoring tool
func NewMultiFileRefactorTool() *MultiFileRefactorTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"rename", "extract", "move", "inline"},
				"description": "Type of refactoring operation",
			},
			"target": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file": map[string]interface{}{
						"type":        "string",
						"description": "Source file containing the code to refactor",
					},
					"symbol": map[string]interface{}{
						"type":        "string",
						"description": "Symbol name to refactor",
					},
					"line": map[string]interface{}{
						"type":        "integer",
						"description": "Line number for position-based refactoring",
					},
					"start_line": map[string]interface{}{
						"type":        "integer",
						"description": "Start line for range-based refactoring",
					},
					"end_line": map[string]interface{}{
						"type":        "integer",
						"description": "End line for range-based refactoring",
					},
				},
				"required": []string{"file"},
			},
			"new_name": map[string]interface{}{
				"type":        "string",
				"description": "New name for rename operations",
			},
			"destination": map[string]interface{}{
				"type":        "string",
				"description": "Destination file for move operations",
			},
			"preview": map[string]interface{}{
				"type":        "boolean",
				"description": "Show preview of changes without applying",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"file", "package", "project"},
				"description": "Scope of the refactoring operation",
			},
			"options": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"update_references": map[string]interface{}{
						"type":        "boolean",
						"description": "Update all references to the symbol",
					},
					"update_imports": map[string]interface{}{
						"type":        "boolean",
						"description": "Update import statements",
					},
					"backup_files": map[string]interface{}{
						"type":        "boolean",
						"description": "Create backup files before changes",
					},
				},
			},
		},
		"required": []string{"type", "target"},
	}

	examples := []string{
		`{"type": "rename", "target": {"file": "main.go", "symbol": "oldName"}, "new_name": "newName", "preview": true}`,
		`{"type": "move", "target": {"file": "utils.go", "symbol": "MyFunction"}, "destination": "helpers.go"}`,
		`{"type": "extract", "target": {"file": "main.go", "start_line": 10, "end_line": 20}, "new_name": "extractedMethod"}`,
		`{"type": "rename", "target": {"file": "types.go", "symbol": "OldStruct"}, "new_name": "NewStruct", "scope": "project"}`,
	}

	baseTool := tools.NewBaseTool(
		"multi_refactor",
		"Perform coordinated refactoring operations across multiple files including rename, extract method, move code, and update all references.",
		schema,
		"edit",
		false,
		examples,
	)

	return &MultiFileRefactorTool{
		BaseTool: baseTool,
	}
}

// Execute runs the multi-file refactoring tool with the given input
func (t *MultiFileRefactorTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params RefactorParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("multi_refactor_tool").
			WithOperation("execute")
	}

	// Validate required parameters
	if params.Type == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "refactor type is required", nil).
			WithComponent("multi_refactor_tool").
			WithOperation("execute")
	}

	if params.Target == nil || params.Target.File == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "target file is required", nil).
			WithComponent("multi_refactor_tool").
			WithOperation("execute")
	}

	// Set defaults
	if params.Options == nil {
		params.Options = &RefactorOptions{
			UpdateReferences: true,
			UpdateImports:    true,
			UpdateComments:   false,
			IncludeTests:     true,
			BackupFiles:      false,
		}
	}
	if params.Scope == "" {
		params.Scope = "package"
	}

	// Check if source file exists
	if _, err := os.Stat(params.Target.File); os.IsNotExist(err) {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "source file does not exist: %s", params.Target.File).
			WithComponent("multi_refactor_tool").
			WithOperation("execute")
	}

	// Perform refactoring
	result, err := t.performRefactoring(ctx, params)
	if err != nil {
		return nil, err
	}

	// Format output
	output := t.formatResult(result)

	metadata := map[string]string{
		"type":               params.Type,
		"source":             params.Target.File,
		"applied":            fmt.Sprintf("%t", result.Applied),
		"files_changed":      fmt.Sprintf("%d", len(result.FilesChanged)),
		"references_found":   fmt.Sprintf("%d", result.ReferencesFound),
		"references_updated": fmt.Sprintf("%d", result.ReferencesUpdated),
	}

	extraData := map[string]interface{}{
		"result": result,
	}

	return tools.NewToolResult(output, metadata, nil, extraData), nil
}

// performRefactoring performs the refactoring operation
func (t *MultiFileRefactorTool) performRefactoring(ctx context.Context, params RefactorParams) (*RefactorResult, error) {
	result := &RefactorResult{
		Type:    params.Type,
		Preview: params.Preview,
	}

	switch params.Type {
	case "rename":
		return t.performRename(ctx, params, result)
	case "extract":
		return t.performExtract(ctx, params, result)
	case "move":
		return t.performMove(ctx, params, result)
	case "inline":
		return t.performInline(ctx, params, result)
	default:
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "unsupported refactor type: %s", params.Type).
			WithComponent("multi_refactor_tool").
			WithOperation("perform_refactoring")
	}
}

// performRename performs a rename refactoring operation
func (t *MultiFileRefactorTool) performRename(ctx context.Context, params RefactorParams, result *RefactorResult) (*RefactorResult, error) {
	if params.NewName == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "new_name is required for rename operation", nil).
			WithComponent("multi_refactor_tool").
			WithOperation("perform_rename")
	}

	if params.Target.Symbol == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "target symbol is required for rename operation", nil).
			WithComponent("multi_refactor_tool").
			WithOperation("perform_rename")
	}

	// Find files to search based on scope
	files, err := t.findFilesInScope(params)
	if err != nil {
		return nil, err
	}

	// Find all references to the symbol
	references, err := t.findReferences(params.Target.File, params.Target.Symbol, files)
	if err != nil {
		return nil, err
	}

	result.ReferencesFound = len(references)

	// Check for naming conflicts
	conflicts := t.checkNamingConflicts(params.NewName, references, files)
	result.Conflicts = conflicts

	if len(conflicts) > 0 && !params.Preview {
		result.Errors = append(result.Errors, "Cannot proceed due to naming conflicts")
		return result, nil
	}

	// Apply renames
	if !params.Preview || len(conflicts) == 0 {
		err = t.applyRenames(references, params.Target.Symbol, params.NewName, params.Options.BackupFiles, result)
		if err != nil {
			return nil, err
		}

		if !params.Preview {
			result.Applied = true
		}
		result.ReferencesUpdated = len(references)
	}

	result.Summary = fmt.Sprintf("Renamed '%s' to '%s' (%d references in %d files)",
		params.Target.Symbol, params.NewName, result.ReferencesFound, len(result.FilesChanged))

	return result, nil
}

// performExtract performs an extract method refactoring
func (t *MultiFileRefactorTool) performExtract(ctx context.Context, params RefactorParams, result *RefactorResult) (*RefactorResult, error) {
	if params.NewName == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "new_name is required for extract operation", nil).
			WithComponent("multi_refactor_tool").
			WithOperation("perform_extract")
	}

	if params.Target.StartLine == 0 || params.Target.EndLine == 0 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "start_line and end_line are required for extract operation", nil).
			WithComponent("multi_refactor_tool").
			WithOperation("perform_extract")
	}

	// Read source file
	content, err := os.ReadFile(params.Target.File)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	if params.Target.StartLine > len(lines) || params.Target.EndLine > len(lines) {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "line numbers out of range", nil).
			WithComponent("multi_refactor_tool").
			WithOperation("perform_extract")
	}

	// Extract the code block
	extractedLines := lines[params.Target.StartLine-1 : params.Target.EndLine]
	extractedCode := strings.Join(extractedLines, "\n")

	// Analyze the extracted code for parameters and return values
	parameters, returnValues := t.analyzeExtractedCode(extractedCode, params.Target.File)

	// Generate new method
	newMethod := t.generateExtractedMethod(params.NewName, parameters, returnValues, extractedCode)

	// Create method call to replace extracted code
	methodCall := t.generateMethodCall(params.NewName, parameters, returnValues)

	// Apply changes
	fileChange := &FileChange{
		File:       params.Target.File,
		ChangeType: "modified",
	}

	// Replace extracted code with method call
	for i := params.Target.StartLine - 1; i < params.Target.EndLine; i++ {
		if i == params.Target.StartLine-1 {
			fileChange.Changes = append(fileChange.Changes, &LineChange{
				Line:       i + 1,
				Type:       "replace",
				OldContent: lines[i],
				NewContent: methodCall,
			})
		} else {
			fileChange.Changes = append(fileChange.Changes, &LineChange{
				Line:       i + 1,
				Type:       "delete",
				OldContent: lines[i],
			})
		}
	}

	// Add new method (simplified - would need proper insertion logic)
	fileChange.Changes = append(fileChange.Changes, &LineChange{
		Line:       len(lines) + 1,
		Type:       "insert",
		NewContent: newMethod,
	})

	result.FilesChanged = append(result.FilesChanged, fileChange)

	if !params.Preview {
		err = t.applyFileChanges(fileChange, params.Options.BackupFiles)
		if err != nil {
			return nil, err
		}
		result.Applied = true
	}

	result.Summary = fmt.Sprintf("Extracted %d lines into method '%s'",
		params.Target.EndLine-params.Target.StartLine+1, params.NewName)

	return result, nil
}

// performMove performs a move refactoring operation
func (t *MultiFileRefactorTool) performMove(ctx context.Context, params RefactorParams, result *RefactorResult) (*RefactorResult, error) {
	if params.Destination == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "destination file is required for move operation", nil).
			WithComponent("multi_refactor_tool").
			WithOperation("perform_move")
	}

	// This is a simplified implementation
	result.Errors = append(result.Errors, "Move refactoring not fully implemented yet")
	result.Summary = "Move refactoring is a complex operation that requires deeper AST analysis"

	return result, nil
}

// performInline performs an inline refactoring operation
func (t *MultiFileRefactorTool) performInline(ctx context.Context, params RefactorParams, result *RefactorResult) (*RefactorResult, error) {
	// This is a simplified implementation
	result.Errors = append(result.Errors, "Inline refactoring not fully implemented yet")
	result.Summary = "Inline refactoring requires complex dependency analysis"

	return result, nil
}

// findFilesInScope finds files to search based on the specified scope
func (t *MultiFileRefactorTool) findFilesInScope(params RefactorParams) ([]string, error) {
	var files []string

	switch params.Scope {
	case "file":
		files = []string{params.Target.File}

	case "package":
		// Find files in the same package/directory
		dir := filepath.Dir(params.Target.File)
		language := code.DetectLanguage(params.Target.File)

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && code.DetectLanguage(path) == language {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

	case "project":
		// Find files in the entire project
		language := code.DetectLanguage(params.Target.File)

		err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && code.DetectLanguage(path) == language {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

	default:
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "unsupported scope: %s", params.Scope).
			WithComponent("multi_refactor_tool").
			WithOperation("find_files_in_scope")
	}

	return files, nil
}

// findReferences finds all references to a symbol in the specified files
func (t *MultiFileRefactorTool) findReferences(sourceFile, symbol string, files []string) ([]*SymbolReference, error) {
	var references []*SymbolReference

	for _, file := range files {
		refs, err := t.findSymbolReferences(file, symbol)
		if err != nil {
			// Log error but continue with other files
			continue
		}
		references = append(references, refs...)
	}

	return references, nil
}

// findSymbolReferences finds references to a symbol in a single file
func (t *MultiFileRefactorTool) findSymbolReferences(filename, symbol string) ([]*SymbolReference, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	language := code.DetectLanguage(filename)

	if language == code.LanguageGo {
		return t.findGoReferences(filename, symbol, content)
	}

	// Fallback to simple text search
	return t.findTextReferences(filename, symbol, content), nil
}

// findGoReferences finds references using Go AST
func (t *MultiFileRefactorTool) findGoReferences(filename, symbol string, content []byte) ([]*SymbolReference, error) {
	var references []*SymbolReference

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		// Fallback to text search if parsing fails
		return t.findTextReferences(filename, symbol, content), nil
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.Ident:
			if node.Name == symbol {
				pos := fset.Position(node.Pos())
				references = append(references, &SymbolReference{
					File:   filename,
					Line:   pos.Line,
					Column: pos.Column,
				})
			}
		}
		return true
	})

	return references, nil
}

// findTextReferences finds references using simple text search
func (t *MultiFileRefactorTool) findTextReferences(filename, symbol string, content []byte) []*SymbolReference {
	var references []*SymbolReference

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		// Use word boundary regex to find whole word matches
		pattern := `\b` + regexp.QuoteMeta(symbol) + `\b`
		regex := regexp.MustCompile(pattern)

		matches := regex.FindAllStringIndex(line, -1)
		for _, match := range matches {
			references = append(references, &SymbolReference{
				File:   filename,
				Line:   i + 1,
				Column: match[0] + 1,
			})
		}
	}

	return references
}

// SymbolReference represents a reference to a symbol
type SymbolReference struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// checkNamingConflicts checks for potential naming conflicts
func (t *MultiFileRefactorTool) checkNamingConflicts(newName string, references []*SymbolReference, files []string) []*RefactorConflict {
	var conflicts []*RefactorConflict

	// Check if new name already exists in any of the files
	for _, file := range files {
		// Read the file content
		content, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files that can't be read
		}

		// Find existing references to the new name
		existingRefs := t.findTextReferences(file, newName, content)
		if len(existingRefs) > 0 {
			conflicts = append(conflicts, &RefactorConflict{
				File:        file,
				Type:        "name_collision",
				Description: fmt.Sprintf("Name '%s' already exists in %s", newName, file),
				Suggestion:  "Consider using a different name",
			})
		}
	}

	return conflicts
}

// applyRenames applies rename changes to all files
func (t *MultiFileRefactorTool) applyRenames(references []*SymbolReference, oldName, newName string, createBackups bool, result *RefactorResult) error {
	fileChanges := make(map[string]*FileChange)

	// Group references by file
	for _, ref := range references {
		if _, exists := fileChanges[ref.File]; !exists {
			fileChanges[ref.File] = &FileChange{
				File:       ref.File,
				ChangeType: "modified",
			}
		}

		fileChanges[ref.File].Changes = append(fileChanges[ref.File].Changes, &LineChange{
			Line:       ref.Line,
			Type:       "replace",
			OldContent: oldName,
			NewContent: newName,
		})
	}

	// Apply changes to each file
	for _, fileChange := range fileChanges {
		err := t.applyFileChanges(fileChange, createBackups)
		if err != nil {
			return err
		}
		result.FilesChanged = append(result.FilesChanged, fileChange)
	}

	return nil
}

// applyFileChanges applies changes to a single file
func (t *MultiFileRefactorTool) applyFileChanges(fileChange *FileChange, createBackup bool) error {
	// Read original content
	content, err := os.ReadFile(fileChange.File)
	if err != nil {
		return err
	}

	// Create backup if requested
	if createBackup {
		backupFile := fileChange.File + ".bak"
		err = os.WriteFile(backupFile, content, 0644)
		if err != nil {
			return err
		}
		fileChange.BackupFile = backupFile
	}

	// Apply changes (simplified implementation)
	lines := strings.Split(string(content), "\n")

	for _, change := range fileChange.Changes {
		if change.Line > 0 && change.Line <= len(lines) {
			switch change.Type {
			case "replace":
				lines[change.Line-1] = strings.ReplaceAll(lines[change.Line-1], change.OldContent, change.NewContent)
			case "insert":
				// Insert new line
				newLines := make([]string, len(lines)+1)
				copy(newLines[:change.Line-1], lines[:change.Line-1])
				newLines[change.Line-1] = change.NewContent
				copy(newLines[change.Line:], lines[change.Line-1:])
				lines = newLines
			case "delete":
				// Delete line
				newLines := make([]string, len(lines)-1)
				copy(newLines[:change.Line-1], lines[:change.Line-1])
				copy(newLines[change.Line-1:], lines[change.Line:])
				lines = newLines
			}
		}
	}

	// Write modified content
	newContent := strings.Join(lines, "\n")
	fileChange.NewContent = newContent

	return os.WriteFile(fileChange.File, []byte(newContent), 0644)
}

// analyzeExtractedCode analyzes extracted code to determine parameters and return values
func (t *MultiFileRefactorTool) analyzeExtractedCode(code, filename string) ([]string, []string) {
	// Simplified analysis - in a real implementation, this would use AST analysis
	// to determine variables used, defined, and returned

	var parameters []string
	var returnValues []string

	// Simple heuristic: look for variable usage patterns
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		// This is a very simplified approach
		if strings.Contains(line, "return") {
			// Extract return value
			parts := strings.Split(line, "return")
			if len(parts) > 1 {
				returnVal := strings.TrimSpace(parts[1])
				if returnVal != "" && returnVal != "nil" {
					returnValues = append(returnValues, returnVal)
				}
			}
		}
	}

	// For now, return empty parameters - real implementation would analyze scope
	return parameters, returnValues
}

// generateExtractedMethod generates the code for an extracted method
func (t *MultiFileRefactorTool) generateExtractedMethod(name string, params, returns []string, code string) string {
	var method strings.Builder

	method.WriteString(fmt.Sprintf("func %s(", name))
	method.WriteString(strings.Join(params, ", "))
	method.WriteString(")")

	if len(returns) > 0 {
		if len(returns) == 1 {
			method.WriteString(" " + returns[0])
		} else {
			method.WriteString(" (" + strings.Join(returns, ", ") + ")")
		}
	}

	method.WriteString(" {\n")

	// Indent the extracted code
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			method.WriteString("\t" + line + "\n")
		}
	}

	method.WriteString("}")

	return method.String()
}

// generateMethodCall generates a method call to replace extracted code
func (t *MultiFileRefactorTool) generateMethodCall(name string, params, returns []string) string {
	call := name + "(" + strings.Join(params, ", ") + ")"

	if len(returns) > 0 {
		if len(returns) == 1 {
			return returns[0] + " := " + call
		} else {
			return strings.Join(returns, ", ") + " := " + call
		}
	}

	return call
}

// formatResult formats the multi-file refactoring result for output
func (t *MultiFileRefactorTool) formatResult(result *RefactorResult) string {
	var output strings.Builder

	// Header
	if result.Preview {
		output.WriteString(fmt.Sprintf("Multi-File Refactoring Preview (%s)\n", strings.Title(result.Type)))
	} else if result.Applied {
		output.WriteString(fmt.Sprintf("Multi-File Refactoring Applied (%s)\n", strings.Title(result.Type)))
	} else {
		output.WriteString(fmt.Sprintf("Multi-File Refactoring Failed (%s)\n", strings.Title(result.Type)))
	}

	// Summary
	output.WriteString(fmt.Sprintf("%s\n", result.Summary))

	// Statistics
	if result.ReferencesFound > 0 {
		output.WriteString(fmt.Sprintf("References found: %d\n", result.ReferencesFound))
		output.WriteString(fmt.Sprintf("References updated: %d\n", result.ReferencesUpdated))
	}

	output.WriteString(fmt.Sprintf("Files changed: %d\n", len(result.FilesChanged)))

	// File changes
	if len(result.FilesChanged) > 0 {
		output.WriteString("\nFiles modified:\n")
		for _, fileChange := range result.FilesChanged {
			output.WriteString(fmt.Sprintf("  %s (%s, %d changes)\n",
				fileChange.File, fileChange.ChangeType, len(fileChange.Changes)))

			if fileChange.BackupFile != "" {
				output.WriteString(fmt.Sprintf("    Backup: %s\n", fileChange.BackupFile))
			}
		}
	}

	// Conflicts
	if len(result.Conflicts) > 0 {
		output.WriteString(fmt.Sprintf("\nConflicts (%d):\n", len(result.Conflicts)))
		for _, conflict := range result.Conflicts {
			output.WriteString(fmt.Sprintf("  %s: %s\n", conflict.Type, conflict.Description))
			if conflict.Suggestion != "" {
				output.WriteString(fmt.Sprintf("    Suggestion: %s\n", conflict.Suggestion))
			}
		}
	}

	// Warnings and errors
	if len(result.Warnings) > 0 {
		output.WriteString("\nWarnings:\n")
		for _, warning := range result.Warnings {
			output.WriteString(fmt.Sprintf("- %s\n", warning))
		}
	}

	if len(result.Errors) > 0 {
		output.WriteString("\nErrors:\n")
		for _, err := range result.Errors {
			output.WriteString(fmt.Sprintf("- %s\n", err))
		}
	}

	return output.String()
}
