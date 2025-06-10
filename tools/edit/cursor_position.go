package edit

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools"
	"github.com/guild-ventures/guild-core/tools/code"
)

// CursorPositionTool provides code navigation and position management
type CursorPositionTool struct {
	*tools.BaseTool
	marks map[string]*Position // Named position bookmarks
}

// CursorParams represents the input parameters for cursor operations
type CursorParams struct {
	File      string `json:"file"`                   // Target file
	Line      int    `json:"line,omitempty"`         // Line number (1-based)
	Column    int    `json:"column,omitempty"`       // Column number (1-based)
	Symbol    string `json:"symbol,omitempty"`       // Symbol name to find
	Direction string `json:"direction,omitempty"`    // next, prev, up, down
	Target    string `json:"target,omitempty"`       // function, class, block, etc.
	SetMark   string `json:"set_mark,omitempty"`     // Save current position with name
	GoToMark  string `json:"go_to_mark,omitempty"`   // Jump to named mark
	Pattern   string `json:"pattern,omitempty"`      // Search pattern
	Scope     string `json:"scope,omitempty"`        // local, file, project
}

// CursorResult represents the result of cursor operations
type CursorResult struct {
	File            string            `json:"file"`
	Position        *Position         `json:"position"`
	Found           bool              `json:"found"`
	Context         *CodeContext      `json:"context,omitempty"`
	Suggestions     []*NavigationHint `json:"suggestions,omitempty"`
	MarkSet         string            `json:"mark_set,omitempty"`
	Errors          []string          `json:"errors,omitempty"`
	Preview         string            `json:"preview,omitempty"`
}

// Position represents a cursor position in a file
type Position struct {
	Line       int    `json:"line"`        // 1-based line number
	Column     int    `json:"column"`      // 1-based column number
	Offset     int    `json:"offset"`      // Byte offset from start of file
	Character  string `json:"character,omitempty"` // Character at position
}

// CodeContext provides context around a cursor position
type CodeContext struct {
	Function    *FunctionContext `json:"function,omitempty"`
	Class       *ClassContext    `json:"class,omitempty"`
	Block       *BlockContext    `json:"block,omitempty"`
	LineContent string           `json:"line_content"`
	Indentation int              `json:"indentation"`
	Language    string           `json:"language"`
	Scope       string           `json:"scope"` // global, function, class, block
}

// FunctionContext provides information about the containing function
type FunctionContext struct {
	Name       string    `json:"name"`
	StartLine  int       `json:"start_line"`
	EndLine    int       `json:"end_line"`
	Parameters []string  `json:"parameters,omitempty"`
	ReturnType string    `json:"return_type,omitempty"`
	Position   *Position `json:"position"`
}

// ClassContext provides information about the containing class/struct
type ClassContext struct {
	Name      string    `json:"name"`
	StartLine int       `json:"start_line"`
	EndLine   int       `json:"end_line"`
	Type      string    `json:"type"` // class, struct, interface
	Position  *Position `json:"position"`
}

// BlockContext provides information about the containing block
type BlockContext struct {
	Type      string    `json:"type"`       // if, for, while, try, etc.
	StartLine int       `json:"start_line"`
	EndLine   int       `json:"end_line"`
	Level     int       `json:"level"`      // Nesting level
	Position  *Position `json:"position"`
}

// NavigationHint provides suggestions for navigation
type NavigationHint struct {
	Type        string    `json:"type"`        // function, class, symbol, etc.
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Position    *Position `json:"position"`
	Distance    int       `json:"distance"`    // Lines away from current position
}

// NewCursorPositionTool creates a new cursor position tool
func NewCursorPositionTool() *CursorPositionTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file": map[string]interface{}{
				"type":        "string",
				"description": "File path to navigate in",
			},
			"line": map[string]interface{}{
				"type":        "integer",
				"description": "Line number to navigate to (1-based)",
			},
			"column": map[string]interface{}{
				"type":        "integer",
				"description": "Column number to navigate to (1-based)",
			},
			"symbol": map[string]interface{}{
				"type":        "string",
				"description": "Symbol name to find and navigate to",
			},
			"direction": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"next", "prev", "up", "down"},
				"description": "Direction to navigate",
			},
			"target": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"function", "class", "block", "line", "symbol"},
				"description": "Type of target to navigate to",
			},
			"set_mark": map[string]interface{}{
				"type":        "string",
				"description": "Save current position with this name",
			},
			"go_to_mark": map[string]interface{}{
				"type":        "string",
				"description": "Jump to previously saved mark",
			},
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Search pattern for navigation",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"local", "file", "project"},
				"description": "Search scope for navigation",
			},
		},
		"required": []string{"file"},
	}

	examples := []string{
		`{"file": "main.go", "line": 42, "column": 10}`,
		`{"file": "main.go", "symbol": "MyFunction"}`,
		`{"file": "main.go", "direction": "next", "target": "function"}`,
		`{"file": "main.go", "set_mark": "important_spot", "line": 100}`,
		`{"file": "main.go", "go_to_mark": "important_spot"}`,
		`{"file": "main.go", "pattern": "TODO", "direction": "next"}`,
	}

	baseTool := tools.NewBaseTool(
		"cursor_position",
		"Navigate to specific code locations, find symbols, manage position bookmarks, and get code context information.",
		schema,
		"edit",
		false,
		examples,
	)

	return &CursorPositionTool{
		BaseTool: baseTool,
		marks:    make(map[string]*Position),
	}
}

// Execute runs the cursor position tool with the given input
func (t *CursorPositionTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params CursorParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("cursor_position_tool").
			WithOperation("execute")
	}

	// Validate required parameters
	if params.File == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "file is required", nil).
			WithComponent("cursor_position_tool").
			WithOperation("execute")
	}

	// Check if file exists
	if _, err := os.Stat(params.File); os.IsNotExist(err) {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "file does not exist: %s", params.File).
			WithComponent("cursor_position_tool").
			WithOperation("execute")
	}

	// Perform navigation
	result, err := t.navigate(ctx, params)
	if err != nil {
		return tools.NewToolResult("", map[string]string{
			"file": params.File,
		}, err, nil), err
	}

	// Format output
	output := t.formatResult(result, params)

	metadata := map[string]string{
		"file":  params.File,
		"found": fmt.Sprintf("%t", result.Found),
	}

	if result.Position != nil {
		metadata["line"] = fmt.Sprintf("%d", result.Position.Line)
		metadata["column"] = fmt.Sprintf("%d", result.Position.Column)
	}

	extraData := map[string]interface{}{
		"result": result,
	}

	return tools.NewToolResult(output, metadata, nil, extraData), nil
}

// navigate performs the navigation operation
func (t *CursorPositionTool) navigate(ctx context.Context, params CursorParams) (*CursorResult, error) {
	result := &CursorResult{
		File: params.File,
	}

	// Handle mark operations first
	if params.SetMark != "" {
		position := &Position{
			Line:   params.Line,
			Column: params.Column,
		}
		if params.Line == 0 {
			position.Line = 1
		}
		if params.Column == 0 {
			position.Column = 1
		}
		
		t.marks[params.SetMark] = position
		result.MarkSet = params.SetMark
		result.Position = position
		result.Found = true
		
		// Still get context for the position
		context, err := t.getCodeContext(params.File, position)
		if err == nil {
			result.Context = context
		}
		
		return result, nil
	}

	if params.GoToMark != "" {
		if mark, exists := t.marks[params.GoToMark]; exists {
			result.Position = mark
			result.Found = true
			
			context, err := t.getCodeContext(params.File, mark)
			if err == nil {
				result.Context = context
			}
			
			return result, nil
		} else {
			return nil, gerror.Newf(gerror.ErrCodeNotFound, "mark '%s' not found", params.GoToMark).
				WithComponent("cursor_position_tool").
				WithOperation("navigate")
		}
	}

	// Handle direct line/column navigation
	if params.Line > 0 || params.Column > 0 {
		position := &Position{
			Line:   params.Line,
			Column: params.Column,
		}
		if position.Line == 0 {
			position.Line = 1
		}
		if position.Column == 0 {
			position.Column = 1
		}
		
		// Validate position is within file bounds
		valid, err := t.validatePosition(params.File, position)
		if err != nil {
			return nil, err
		}
		
		result.Position = position
		result.Found = valid
		
		if valid {
			context, err := t.getCodeContext(params.File, position)
			if err == nil {
				result.Context = context
			}
		}
		
		return result, nil
	}

	// Handle symbol search
	if params.Symbol != "" {
		return t.findSymbol(params)
	}

	// Handle pattern search
	if params.Pattern != "" {
		return t.searchPattern(params)
	}

	// Handle directional navigation
	if params.Direction != "" && params.Target != "" {
		return t.navigateDirection(params)
	}

	return nil, gerror.New(gerror.ErrCodeInvalidInput, "no navigation operation specified").
		WithComponent("cursor_position_tool").
		WithOperation("navigate")
}

// validatePosition checks if a position is valid within a file
func (t *CursorPositionTool) validatePosition(filename string, pos *Position) (bool, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return false, err
	}
	
	lines := strings.Split(string(content), "\n")
	
	if pos.Line < 1 || pos.Line > len(lines) {
		return false, nil
	}
	
	if pos.Column < 1 {
		return false, nil
	}
	
	line := lines[pos.Line-1]
	if pos.Column > len(line)+1 { // +1 for end of line position
		return false, nil
	}
	
	// Calculate offset
	offset := 0
	for i := 0; i < pos.Line-1; i++ {
		offset += len(lines[i]) + 1 // +1 for newline
	}
	offset += pos.Column - 1
	pos.Offset = offset
	
	// Get character at position
	if pos.Column <= len(line) {
		pos.Character = string(line[pos.Column-1])
	}
	
	return true, nil
}

// findSymbol finds a symbol in the file
func (t *CursorPositionTool) findSymbol(params CursorParams) (*CursorResult, error) {
	result := &CursorResult{
		File: params.File,
	}

	// Detect language
	language := code.DetectLanguage(params.File)
	
	switch language {
	case code.LanguageGo:
		return t.findGoSymbol(params, result)
	case code.LanguagePython:
		return t.findPythonSymbol(params, result)
	default:
		return t.findGenericSymbol(params, result)
	}
}

// findGoSymbol finds a symbol in a Go file using AST
func (t *CursorPositionTool) findGoSymbol(params CursorParams, result *CursorResult) (*CursorResult, error) {
	content, err := os.ReadFile(params.File)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, params.File, content, parser.ParseComments)
	if err != nil {
		// Fallback to generic search if parsing fails
		return t.findGenericSymbol(params, result)
	}

	// Search for the symbol in the AST
	var foundPos token.Pos
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Name.Name == params.Symbol {
				foundPos = node.Name.Pos()
				return false
			}
		case *ast.TypeSpec:
			if node.Name.Name == params.Symbol {
				foundPos = node.Name.Pos()
				return false
			}
		case *ast.ValueSpec:
			for _, name := range node.Names {
				if name.Name == params.Symbol {
					foundPos = name.Pos()
					return false
				}
			}
		}
		return true
	})

	if foundPos != token.NoPos {
		pos := fset.Position(foundPos)
		result.Position = &Position{
			Line:   pos.Line,
			Column: pos.Column,
			Offset: pos.Offset,
		}
		result.Found = true

		context, err := t.getCodeContext(params.File, result.Position)
		if err == nil {
			result.Context = context
		}
	}

	return result, nil
}

// findPythonSymbol finds a symbol in a Python file (simplified)
func (t *CursorPositionTool) findPythonSymbol(params CursorParams, result *CursorResult) (*CursorResult, error) {
	content, err := os.ReadFile(params.File)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	
	// Simple regex-based search for Python symbols
	functionRegex := regexp.MustCompile(`^\s*def\s+` + regexp.QuoteMeta(params.Symbol) + `\s*\(`)
	classRegex := regexp.MustCompile(`^\s*class\s+` + regexp.QuoteMeta(params.Symbol) + `\s*[\(:]`)

	for i, line := range lines {
		if functionRegex.MatchString(line) || classRegex.MatchString(line) {
			// Find the position of the symbol name in the line
			symbolPos := strings.Index(line, params.Symbol)
			if symbolPos >= 0 {
				result.Position = &Position{
					Line:   i + 1,
					Column: symbolPos + 1,
				}
				result.Found = true

				context, err := t.getCodeContext(params.File, result.Position)
				if err == nil {
					result.Context = context
				}
				break
			}
		}
	}

	return result, nil
}

// findGenericSymbol finds a symbol using simple text search
func (t *CursorPositionTool) findGenericSymbol(params CursorParams, result *CursorResult) (*CursorResult, error) {
	content, err := os.ReadFile(params.File)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	
	for i, line := range lines {
		if strings.Contains(line, params.Symbol) {
			symbolPos := strings.Index(line, params.Symbol)
			result.Position = &Position{
				Line:   i + 1,
				Column: symbolPos + 1,
			}
			result.Found = true

			context, err := t.getCodeContext(params.File, result.Position)
			if err == nil {
				result.Context = context
			}
			break
		}
	}

	return result, nil
}

// searchPattern searches for a pattern in the file
func (t *CursorPositionTool) searchPattern(params CursorParams) (*CursorResult, error) {
	result := &CursorResult{
		File: params.File,
	}

	content, err := os.ReadFile(params.File)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	
	// Simple text search for now
	for i, line := range lines {
		if strings.Contains(line, params.Pattern) {
			patternPos := strings.Index(line, params.Pattern)
			result.Position = &Position{
				Line:   i + 1,
				Column: patternPos + 1,
			}
			result.Found = true

			context, err := t.getCodeContext(params.File, result.Position)
			if err == nil {
				result.Context = context
			}
			break
		}
	}

	return result, nil
}

// navigateDirection performs directional navigation
func (t *CursorPositionTool) navigateDirection(params CursorParams) (*CursorResult, error) {
	result := &CursorResult{
		File: params.File,
	}

	// This would implement navigation like "next function", "previous class", etc.
	// For now, return a placeholder implementation
	result.Errors = append(result.Errors, "Directional navigation not yet fully implemented")
	
	return result, nil
}

// getCodeContext gets the code context for a position
func (t *CursorPositionTool) getCodeContext(filename string, pos *Position) (*CodeContext, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	if pos.Line < 1 || pos.Line > len(lines) {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "position out of bounds").
			WithComponent("cursor_position_tool").
			WithOperation("get_code_context")
	}

	context := &CodeContext{
		Language:    string(code.DetectLanguage(filename)),
		LineContent: lines[pos.Line-1],
		Indentation: len(lines[pos.Line-1]) - len(strings.TrimLeft(lines[pos.Line-1], " \t")),
		Scope:       "global",
	}

	// Try to get more detailed context based on language
	if context.Language == "go" {
		t.getGoContext(filename, pos, context)
	}

	return context, nil
}

// getGoContext gets Go-specific context information
func (t *CursorPositionTool) getGoContext(filename string, pos *Position, context *CodeContext) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		return
	}

	// Find the containing function and type
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			start := fset.Position(node.Pos())
			end := fset.Position(node.End())
			
			if pos.Line >= start.Line && pos.Line <= end.Line {
				context.Function = &FunctionContext{
					Name:      node.Name.Name,
					StartLine: start.Line,
					EndLine:   end.Line,
					Position: &Position{
						Line:   start.Line,
						Column: start.Column,
					},
				}
				context.Scope = "function"
				
				// Extract parameters
				if node.Type.Params != nil {
					for _, field := range node.Type.Params.List {
						for _, name := range field.Names {
							context.Function.Parameters = append(context.Function.Parameters, name.Name)
						}
					}
				}
			}
			
		case *ast.TypeSpec:
			if structType, ok := node.Type.(*ast.StructType); ok {
				start := fset.Position(node.Pos())
				end := fset.Position(structType.End())
				
				if pos.Line >= start.Line && pos.Line <= end.Line {
					context.Class = &ClassContext{
						Name:      node.Name.Name,
						Type:      "struct",
						StartLine: start.Line,
						EndLine:   end.Line,
						Position: &Position{
							Line:   start.Line,
							Column: start.Column,
						},
					}
					if context.Scope == "global" {
						context.Scope = "struct"
					}
				}
			}
		}
		return true
	})
}

// formatResult formats the cursor position result for output
func (t *CursorPositionTool) formatResult(result *CursorResult, params CursorParams) string {
	var output strings.Builder
	
	if result.MarkSet != "" {
		output.WriteString(fmt.Sprintf("Mark '%s' set in %s\n", result.MarkSet, result.File))
	} else if params.GoToMark != "" {
		output.WriteString(fmt.Sprintf("Jumped to mark '%s' in %s\n", params.GoToMark, result.File))
	} else {
		output.WriteString(fmt.Sprintf("Navigation in %s\n", result.File))
	}

	if result.Position != nil {
		output.WriteString(fmt.Sprintf("Position: Line %d, Column %d", 
			result.Position.Line, result.Position.Column))
		
		if result.Position.Character != "" {
			output.WriteString(fmt.Sprintf(" ('%s')", result.Position.Character))
		}
		output.WriteString("\n")
	}

	if !result.Found {
		output.WriteString("Target not found\n")
	}

	// Context information
	if result.Context != nil {
		ctx := result.Context
		output.WriteString(fmt.Sprintf("Context: %s (%s)\n", ctx.Scope, ctx.Language))
		
		if ctx.Function != nil {
			output.WriteString(fmt.Sprintf("Function: %s (lines %d-%d)\n", 
				ctx.Function.Name, ctx.Function.StartLine, ctx.Function.EndLine))
			if len(ctx.Function.Parameters) > 0 {
				output.WriteString(fmt.Sprintf("Parameters: %s\n", strings.Join(ctx.Function.Parameters, ", ")))
			}
		}
		
		if ctx.Class != nil {
			output.WriteString(fmt.Sprintf("%s: %s (lines %d-%d)\n", 
				strings.Title(ctx.Class.Type), ctx.Class.Name, ctx.Class.StartLine, ctx.Class.EndLine))
		}
		
		if ctx.LineContent != "" {
			output.WriteString(fmt.Sprintf("Line content: %s\n", strings.TrimSpace(ctx.LineContent)))
		}
	}

	// Navigation suggestions
	if len(result.Suggestions) > 0 {
		output.WriteString("\nNavigation suggestions:\n")
		for _, suggestion := range result.Suggestions {
			output.WriteString(fmt.Sprintf("- %s: %s (line %d)\n", 
				suggestion.Type, suggestion.Name, suggestion.Position.Line))
		}
	}

	// Errors
	if len(result.Errors) > 0 {
		output.WriteString("\nWarnings:\n")
		for _, err := range result.Errors {
			output.WriteString(fmt.Sprintf("- %s\n", err))
		}
	}

	return output.String()
}