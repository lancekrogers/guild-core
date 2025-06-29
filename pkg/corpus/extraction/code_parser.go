// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package extraction

import (
	"context"
	"regexp"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// CodeParser provides language-specific parsing capabilities
type CodeParser struct {
	goPatterns map[string]*regexp.Regexp
	jsPatterns map[string]*regexp.Regexp
	pyPatterns map[string]*regexp.Regexp
}

// NewCodeParser creates a new code parser with language-specific patterns
func NewCodeParser() *CodeParser {
	return &CodeParser{
		goPatterns: map[string]*regexp.Regexp{
			"function":     regexp.MustCompile(`func\s+(\w+)\s*\(`),
			"method":       regexp.MustCompile(`func\s+\(\w+\s+\*?\w+\)\s+(\w+)\s*\(`),
			"struct":       regexp.MustCompile(`type\s+(\w+)\s+struct`),
			"interface":    regexp.MustCompile(`type\s+(\w+)\s+interface`),
			"const":        regexp.MustCompile(`const\s+(\w+)`),
			"var":          regexp.MustCompile(`var\s+(\w+)`),
			"import":       regexp.MustCompile(`import\s+.*?"([^"]+)"`),
			"package":      regexp.MustCompile(`package\s+(\w+)`),
			"comment":      regexp.MustCompile(`//\s*(.*)`),
			"block_comment": regexp.MustCompile(`/\*[\s\S]*?\*/`),
		},
		jsPatterns: map[string]*regexp.Regexp{
			"function":     regexp.MustCompile(`function\s+(\w+)\s*\(`),
			"arrow_func":   regexp.MustCompile(`const\s+(\w+)\s*=\s*\([^)]*\)\s*=>`),
			"class":        regexp.MustCompile(`class\s+(\w+)`),
			"method":       regexp.MustCompile(`(\w+)\s*\([^)]*\)\s*\{`),
			"const":        regexp.MustCompile(`const\s+(\w+)`),
			"let":          regexp.MustCompile(`let\s+(\w+)`),
			"var":          regexp.MustCompile(`var\s+(\w+)`),
			"import":       regexp.MustCompile(`import\s+.*?from\s+['"]([^'"]+)['"]`),
			"export":       regexp.MustCompile(`export\s+(?:default\s+)?(\w+)`),
		},
		pyPatterns: map[string]*regexp.Regexp{
			"function":     regexp.MustCompile(`def\s+(\w+)\s*\(`),
			"class":        regexp.MustCompile(`class\s+(\w+)`),
			"method":       regexp.MustCompile(`\s+def\s+(\w+)\s*\(`),
			"variable":     regexp.MustCompile(`(\w+)\s*=`),
			"import":       regexp.MustCompile(`import\s+(\w+)`),
			"from_import":  regexp.MustCompile(`from\s+(\w+)\s+import`),
			"decorator":    regexp.MustCompile(`@(\w+)`),
		},
	}
}

// ParseGoCode analyzes Go source code and extracts structural information
func (cp *CodeParser) ParseGoCode(ctx context.Context, content string) (*CodeStructure, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("ParseGoCode")
	}

	structure := &CodeStructure{
		Language:   "go",
		Functions:  []FunctionInfo{},
		Types:      []TypeInfo{},
		Imports:    []string{},
		Variables:  []VariableInfo{},
		Comments:   []string{},
	}

	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		// Check for context cancellation periodically
		if i%50 == 0 && ctx.Err() != nil {
			return nil, ctx.Err()
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse package declaration
		if match := cp.goPatterns["package"].FindStringSubmatch(line); match != nil {
			structure.Package = match[1]
		}

		// Parse imports
		if match := cp.goPatterns["import"].FindStringSubmatch(line); match != nil {
			structure.Imports = append(structure.Imports, match[1])
		}

		// Parse functions
		if match := cp.goPatterns["function"].FindStringSubmatch(line); match != nil {
			funcInfo := FunctionInfo{
				Name:       match[1],
				LineNumber: i + 1,
				Signature:  line,
				Type:       "function",
			}
			structure.Functions = append(structure.Functions, funcInfo)
		}

		// Parse methods
		if match := cp.goPatterns["method"].FindStringSubmatch(line); match != nil {
			funcInfo := FunctionInfo{
				Name:       match[1],
				LineNumber: i + 1,
				Signature:  line,
				Type:       "method",
			}
			structure.Functions = append(structure.Functions, funcInfo)
		}

		// Parse structs
		if match := cp.goPatterns["struct"].FindStringSubmatch(line); match != nil {
			typeInfo := TypeInfo{
				Name:       match[1],
				Type:       "struct",
				LineNumber: i + 1,
				Definition: line,
			}
			structure.Types = append(structure.Types, typeInfo)
		}

		// Parse interfaces
		if match := cp.goPatterns["interface"].FindStringSubmatch(line); match != nil {
			typeInfo := TypeInfo{
				Name:       match[1],
				Type:       "interface",
				LineNumber: i + 1,
				Definition: line,
			}
			structure.Types = append(structure.Types, typeInfo)
		}

		// Parse constants
		if match := cp.goPatterns["const"].FindStringSubmatch(line); match != nil {
			varInfo := VariableInfo{
				Name:       match[1],
				Type:       "const",
				LineNumber: i + 1,
			}
			structure.Variables = append(structure.Variables, varInfo)
		}

		// Parse variables
		if match := cp.goPatterns["var"].FindStringSubmatch(line); match != nil {
			varInfo := VariableInfo{
				Name:       match[1],
				Type:       "var",
				LineNumber: i + 1,
			}
			structure.Variables = append(structure.Variables, varInfo)
		}

		// Parse comments
		if match := cp.goPatterns["comment"].FindStringSubmatch(line); match != nil {
			structure.Comments = append(structure.Comments, match[1])
		}
	}

	return structure, nil
}

// ParseJavaScriptCode analyzes JavaScript/TypeScript source code
func (cp *CodeParser) ParseJavaScriptCode(ctx context.Context, content string) (*CodeStructure, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	structure := &CodeStructure{
		Language:  "javascript",
		Functions: []FunctionInfo{},
		Types:     []TypeInfo{},
		Imports:   []string{},
		Variables: []VariableInfo{},
		Comments:  []string{},
	}

	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		if i%50 == 0 && ctx.Err() != nil {
			return nil, ctx.Err()
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse imports
		if match := cp.jsPatterns["import"].FindStringSubmatch(line); match != nil {
			structure.Imports = append(structure.Imports, match[1])
		}

		// Parse functions
		if match := cp.jsPatterns["function"].FindStringSubmatch(line); match != nil {
			funcInfo := FunctionInfo{
				Name:       match[1],
				LineNumber: i + 1,
				Signature:  line,
				Type:       "function",
			}
			structure.Functions = append(structure.Functions, funcInfo)
		}

		// Parse arrow functions
		if match := cp.jsPatterns["arrow_func"].FindStringSubmatch(line); match != nil {
			funcInfo := FunctionInfo{
				Name:       match[1],
				LineNumber: i + 1,
				Signature:  line,
				Type:       "arrow_function",
			}
			structure.Functions = append(structure.Functions, funcInfo)
		}

		// Parse classes
		if match := cp.jsPatterns["class"].FindStringSubmatch(line); match != nil {
			typeInfo := TypeInfo{
				Name:       match[1],
				Type:       "class",
				LineNumber: i + 1,
				Definition: line,
			}
			structure.Types = append(structure.Types, typeInfo)
		}

		// Parse variables
		for _, pattern := range []string{"const", "let", "var"} {
			if match := cp.jsPatterns[pattern].FindStringSubmatch(line); match != nil {
				varInfo := VariableInfo{
					Name:       match[1],
					Type:       pattern,
					LineNumber: i + 1,
				}
				structure.Variables = append(structure.Variables, varInfo)
			}
		}
	}

	return structure, nil
}

// ParsePythonCode analyzes Python source code
func (cp *CodeParser) ParsePythonCode(ctx context.Context, content string) (*CodeStructure, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	structure := &CodeStructure{
		Language:  "python",
		Functions: []FunctionInfo{},
		Types:     []TypeInfo{},
		Imports:   []string{},
		Variables: []VariableInfo{},
		Comments:  []string{},
	}

	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		if i%50 == 0 && ctx.Err() != nil {
			return nil, ctx.Err()
		}

		originalLine := line
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse imports
		if match := cp.pyPatterns["import"].FindStringSubmatch(line); match != nil {
			structure.Imports = append(structure.Imports, match[1])
		}
		if match := cp.pyPatterns["from_import"].FindStringSubmatch(line); match != nil {
			structure.Imports = append(structure.Imports, match[1])
		}

		// Parse functions (top-level)
		if match := cp.pyPatterns["function"].FindStringSubmatch(line); match != nil && !strings.HasPrefix(originalLine, " ") {
			funcInfo := FunctionInfo{
				Name:       match[1],
				LineNumber: i + 1,
				Signature:  line,
				Type:       "function",
			}
			structure.Functions = append(structure.Functions, funcInfo)
		}

		// Parse methods (indented)
		if match := cp.pyPatterns["method"].FindStringSubmatch(line); match != nil && strings.HasPrefix(originalLine, " ") {
			funcInfo := FunctionInfo{
				Name:       match[1],
				LineNumber: i + 1,
				Signature:  line,
				Type:       "method",
			}
			structure.Functions = append(structure.Functions, funcInfo)
		}

		// Parse classes
		if match := cp.pyPatterns["class"].FindStringSubmatch(line); match != nil {
			typeInfo := TypeInfo{
				Name:       match[1],
				Type:       "class",
				LineNumber: i + 1,
				Definition: line,
			}
			structure.Types = append(structure.Types, typeInfo)
		}

		// Parse top-level variables
		if match := cp.pyPatterns["variable"].FindStringSubmatch(line); match != nil && !strings.HasPrefix(originalLine, " ") {
			varInfo := VariableInfo{
				Name:       match[1],
				Type:       "variable",
				LineNumber: i + 1,
			}
			structure.Variables = append(structure.Variables, varInfo)
		}
	}

	return structure, nil
}

// ParseCode automatically detects language and parses accordingly
func (cp *CodeParser) ParseCode(ctx context.Context, content, filename string) (*CodeStructure, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	ext := cp.getFileExtension(filename)
	
	switch ext {
	case ".go":
		return cp.ParseGoCode(ctx, content)
	case ".js", ".ts", ".jsx", ".tsx":
		return cp.ParseJavaScriptCode(ctx, content)
	case ".py":
		return cp.ParsePythonCode(ctx, content)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported file type", nil).
			WithComponent("corpus.extraction").
			WithOperation("ParseCode").
			WithDetails("extension", ext).
			WithDetails("filename", filename)
	}
}

// AnalyzeComplexity calculates complexity metrics for the code structure
func (cp *CodeParser) AnalyzeComplexity(structure *CodeStructure) ComplexityMetrics {
	metrics := ComplexityMetrics{
		CyclomaticComplexity: 1, // Base complexity
		LinesOfCode:          0,
		Functions:            len(structure.Functions),
		Types:                len(structure.Types),
		Variables:            len(structure.Variables),
		Imports:              len(structure.Imports),
	}

	// Simple LOC calculation (excluding empty lines and comments)
	for _, comment := range structure.Comments {
		if strings.TrimSpace(comment) != "" {
			metrics.LinesOfCode++
		}
	}

	// Estimate cyclomatic complexity based on function count
	// In a real implementation, you'd parse control flow statements
	metrics.CyclomaticComplexity = 1 + (metrics.Functions * 2)

	// Calculate maintainability index (simplified)
	if metrics.LinesOfCode > 0 {
		metrics.MaintainabilityIndex = 100 - float64(metrics.CyclomaticComplexity)*2 - float64(metrics.LinesOfCode)/10
		if metrics.MaintainabilityIndex < 0 {
			metrics.MaintainabilityIndex = 0
		}
	} else {
		metrics.MaintainabilityIndex = 100
	}

	return metrics
}

func (cp *CodeParser) getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return "." + parts[len(parts)-1]
	}
	return ""
}

// Supporting types

// CodeStructure represents the parsed structure of a source code file
type CodeStructure struct {
	Language  string         `json:"language"`
	Package   string         `json:"package,omitempty"`
	Functions []FunctionInfo `json:"functions"`
	Types     []TypeInfo     `json:"types"`
	Imports   []string       `json:"imports"`
	Variables []VariableInfo `json:"variables"`
	Comments  []string       `json:"comments"`
}

// FunctionInfo represents information about a function or method
type FunctionInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"` // "function", "method", "arrow_function"
	Signature  string `json:"signature"`
	LineNumber int    `json:"line_number"`
	Parameters []string `json:"parameters,omitempty"`
	ReturnType string `json:"return_type,omitempty"`
}

// TypeInfo represents information about a type definition
type TypeInfo struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"` // "struct", "interface", "class"
	Definition string   `json:"definition"`
	LineNumber int      `json:"line_number"`
	Fields     []string `json:"fields,omitempty"`
	Methods    []string `json:"methods,omitempty"`
}

// VariableInfo represents information about a variable or constant
type VariableInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"` // "var", "const", "let"
	LineNumber int    `json:"line_number"`
	DataType   string `json:"data_type,omitempty"`
}

// ComplexityMetrics represents code complexity measurements
type ComplexityMetrics struct {
	CyclomaticComplexity  int     `json:"cyclomatic_complexity"`
	LinesOfCode          int     `json:"lines_of_code"`
	Functions            int     `json:"functions"`
	Types                int     `json:"types"`
	Variables            int     `json:"variables"`
	Imports              int     `json:"imports"`
	MaintainabilityIndex float64 `json:"maintainability_index"`
}