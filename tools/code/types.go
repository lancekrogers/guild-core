package code

import (
	"context"
	"go/token"
	"strings"
)

// Language represents a programming language
type Language string

const (
	// Primary languages
	LanguageGo         Language = "go"
	LanguagePython     Language = "python"
	LanguageTypeScript Language = "typescript"
	LanguageJavaScript Language = "javascript"
	LanguageRust       Language = "rust"
	LanguageJava       Language = "java"
	LanguageCSharp     Language = "csharp"
	LanguageCpp        Language = "cpp"
	LanguageC          Language = "c"
	LanguageRuby       Language = "ruby"
	LanguagePhp        Language = "php"
	
	// Additional languages
	LanguageBash       Language = "bash"
	LanguageCSS        Language = "css"
	LanguageCue        Language = "cue"
	LanguageDockerfile Language = "dockerfile"
	LanguageElixir     Language = "elixir"
	LanguageElm        Language = "elm"
	LanguageGroovy     Language = "groovy"
	LanguageHCL        Language = "hcl"
	LanguageHTML       Language = "html"
	LanguageKotlin     Language = "kotlin"
	LanguageLua        Language = "lua"
	LanguageMarkdown   Language = "markdown"
	LanguageOCaml      Language = "ocaml"
	LanguageProtobuf   Language = "protobuf"
	LanguageScala      Language = "scala"
	LanguageSQL        Language = "sql"
	LanguageSvelte     Language = "svelte"
	LanguageSwift      Language = "swift"
	LanguageTOML       Language = "toml"
	LanguageYAML       Language = "yaml"
	
	LanguageUnknown    Language = "unknown"
)

// Parser defines the interface for language-specific parsers
type Parser interface {
	// Parse parses a file and returns the AST
	Parse(ctx context.Context, filename string, content []byte) (*ParseResult, error)

	// Language returns the language this parser supports
	Language() Language

	// Extensions returns the file extensions this parser handles
	Extensions() []string

	// GetFunctions extracts function definitions from the parsed content
	GetFunctions(result *ParseResult) ([]*Function, error)

	// GetClasses extracts class definitions from the parsed content
	GetClasses(result *ParseResult) ([]*Class, error)

	// GetImports extracts import/dependency information
	GetImports(result *ParseResult) ([]*Import, error)

	// FindSymbol finds a specific symbol in the AST
	FindSymbol(result *ParseResult, symbolName string) ([]*Symbol, error)
}

// ParseResult contains the result of parsing a file
type ParseResult struct {
	Language Language
	Filename string
	Content  []byte
	AST      interface{}    // Language-specific AST node
	FileSet  *token.FileSet // For Go files
	Errors   []ParseError
	Metadata map[string]interface{}
}

// ParseError represents a parsing error
type ParseError struct {
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // error, warning, info
}

// Function represents a function/method definition
type Function struct {
	Name       string                 `json:"name"`
	Package    string                 `json:"package,omitempty"`
	Class      string                 `json:"class,omitempty"`
	Signature  string                 `json:"signature"`
	Parameters []*Parameter           `json:"parameters"`
	ReturnType string                 `json:"return_type,omitempty"`
	DocString  string                 `json:"doc_string,omitempty"`
	StartLine  int                    `json:"start_line"`
	EndLine    int                    `json:"end_line"`
	Visibility string                 `json:"visibility"` // public, private, protected
	IsMethod   bool                   `json:"is_method"`
	IsStatic   bool                   `json:"is_static"`
	Decorators []string               `json:"decorators,omitempty"`
	Complexity int                    `json:"complexity,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Parameter represents a function parameter
type Parameter struct {
	Name         string `json:"name"`
	Type         string `json:"type,omitempty"`
	DefaultValue string `json:"default_value,omitempty"`
	IsOptional   bool   `json:"is_optional"`
	IsVariadic   bool   `json:"is_variadic"`
}

// Class represents a class definition
type Class struct {
	Name        string                 `json:"name"`
	Package     string                 `json:"package,omitempty"`
	BaseClasses []string               `json:"base_classes,omitempty"`
	Interfaces  []string               `json:"interfaces,omitempty"`
	Methods     []*Function            `json:"methods"`
	Fields      []*Field               `json:"fields"`
	DocString   string                 `json:"doc_string,omitempty"`
	StartLine   int                    `json:"start_line"`
	EndLine     int                    `json:"end_line"`
	Visibility  string                 `json:"visibility"`
	IsAbstract  bool                   `json:"is_abstract"`
	Decorators  []string               `json:"decorators,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Field represents a class field/property
type Field struct {
	Name         string `json:"name"`
	Type         string `json:"type,omitempty"`
	DefaultValue string `json:"default_value,omitempty"`
	Visibility   string `json:"visibility"`
	IsStatic     bool   `json:"is_static"`
	IsConstant   bool   `json:"is_constant"`
	DocString    string `json:"doc_string,omitempty"`
	Line         int    `json:"line"`
}

// Import represents an import/dependency
type Import struct {
	Path       string   `json:"path"`
	Alias      string   `json:"alias,omitempty"`
	Names      []string `json:"names,omitempty"` // Specific names imported
	IsWildcard bool     `json:"is_wildcard"`
	Line       int      `json:"line"`
	Module     string   `json:"module,omitempty"`
}

// Symbol represents any symbol in the code (function, class, variable, etc.)
type Symbol struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // function, class, variable, interface, etc.
	Package    string                 `json:"package,omitempty"`
	Class      string                 `json:"class,omitempty"`
	Signature  string                 `json:"signature,omitempty"`
	DocString  string                 `json:"doc_string,omitempty"`
	StartLine  int                    `json:"start_line"`
	EndLine    int                    `json:"end_line"`
	StartCol   int                    `json:"start_col"`
	EndCol     int                    `json:"end_col"`
	Visibility string                 `json:"visibility"`
	References []*SymbolReference     `json:"references,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SymbolReference represents a reference to a symbol
type SymbolReference struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Context string `json:"context"` // usage context (call, assignment, etc.)
}

// Dependency represents a project dependency
type Dependency struct {
	Name           string                 `json:"name"`
	Version        string                 `json:"version"`
	LatestVersion  string                 `json:"latest_version,omitempty"`
	Type           string                 `json:"type"`   // direct, transitive
	Source         string                 `json:"source"` // registry, git, local
	License        string                 `json:"license,omitempty"`
	Description    string                 `json:"description,omitempty"`
	IsOutdated     bool                   `json:"is_outdated"`
	SecurityIssues []*SecurityIssue       `json:"security_issues,omitempty"`
	UsageCount     int                    `json:"usage_count"`            // How many files use this
	Size           int64                  `json:"size,omitempty"`         // Size in bytes
	Dependencies   []*Dependency          `json:"dependencies,omitempty"` // Sub-dependencies
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityIssue represents a security vulnerability in a dependency
type SecurityIssue struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"` // critical, high, medium, low
	CVSS        float64 `json:"cvss,omitempty"`
	URL         string  `json:"url,omitempty"`
	FixedIn     string  `json:"fixed_in,omitempty"`
}

// CodeMetrics represents various code quality metrics
type CodeMetrics struct {
	File                 string                 `json:"file,omitempty"`
	Directory            string                 `json:"directory,omitempty"`
	LinesOfCode          int                    `json:"lines_of_code"`
	SourceLinesOfCode    int                    `json:"source_lines_of_code"`
	CommentLines         int                    `json:"comment_lines"`
	BlankLines           int                    `json:"blank_lines"`
	FunctionCount        int                    `json:"function_count"`
	ClassCount           int                    `json:"class_count"`
	CyclomaticComplexity int                    `json:"cyclomatic_complexity"`
	CognitiveComplexity  int                    `json:"cognitive_complexity"`
	MaxComplexity        int                    `json:"max_complexity"`
	AverageComplexity    float64                `json:"average_complexity"`
	TestCoverage         float64                `json:"test_coverage,omitempty"`
	Duplication          float64                `json:"duplication,omitempty"`
	TechnicalDebt        *TechnicalDebt         `json:"technical_debt,omitempty"`
	Issues               []*CodeIssue           `json:"issues,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
}

// TechnicalDebt represents technical debt metrics
type TechnicalDebt struct {
	Minutes   int     `json:"minutes"` // Time to fix
	Hours     float64 `json:"hours"`
	DebtRatio float64 `json:"debt_ratio"` // Percentage
	Rating    string  `json:"rating"`     // A, B, C, D, E
	Issues    int     `json:"issues"`     // Number of issues
}

// CodeIssue represents a code quality issue
type CodeIssue struct {
	Type       string `json:"type"`     // complexity, duplication, style, etc.
	Severity   string `json:"severity"` // error, warning, info
	Message    string `json:"message"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Column     int    `json:"column"`
	Rule       string `json:"rule,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// DetectLanguage attempts to detect the programming language from a filename
func DetectLanguage(filename string) Language {
	if filename == "" {
		return LanguageUnknown
	}

	// Get file extension
	ext := ""
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			ext = filename[i:]
			break
		}
	}

	switch ext {
	// Go
	case ".go":
		return LanguageGo
	// Python
	case ".py", ".pyx", ".pyw":
		return LanguagePython
	// TypeScript
	case ".ts", ".tsx":
		return LanguageTypeScript
	// JavaScript
	case ".js", ".jsx", ".mjs":
		return LanguageJavaScript
	// Rust
	case ".rs":
		return LanguageRust
	// Java
	case ".java":
		return LanguageJava
	// C#
	case ".cs":
		return LanguageCSharp
	// C++
	case ".cpp", ".cc", ".cxx", ".hpp", ".h++":
		return LanguageCpp
	// C
	case ".c", ".h":
		return LanguageC
	// Ruby
	case ".rb":
		return LanguageRuby
	// PHP
	case ".php", ".php3", ".php4", ".php5", ".phtml":
		return LanguagePhp
	// Bash
	case ".sh", ".bash":
		return LanguageBash
	// CSS
	case ".css", ".scss", ".sass", ".less":
		return LanguageCSS
	// CUE
	case ".cue":
		return LanguageCue
	// Elixir
	case ".ex", ".exs":
		return LanguageElixir
	// Elm
	case ".elm":
		return LanguageElm
	// Groovy
	case ".groovy", ".gvy", ".gy", ".gsh":
		return LanguageGroovy
	// HCL
	case ".hcl", ".tf":
		return LanguageHCL
	// HTML
	case ".html", ".htm", ".xhtml":
		return LanguageHTML
	// Kotlin
	case ".kt", ".kts":
		return LanguageKotlin
	// Lua
	case ".lua":
		return LanguageLua
	// Markdown
	case ".md", ".markdown", ".mdown", ".mkd":
		return LanguageMarkdown
	// OCaml
	case ".ml", ".mli":
		return LanguageOCaml
	// Protocol Buffers
	case ".proto":
		return LanguageProtobuf
	// Scala
	case ".scala", ".sc":
		return LanguageScala
	// SQL
	case ".sql":
		return LanguageSQL
	// Svelte
	case ".svelte":
		return LanguageSvelte
	// Swift
	case ".swift":
		return LanguageSwift
	// TOML
	case ".toml":
		return LanguageTOML
	// YAML
	case ".yaml", ".yml":
		return LanguageYAML
	default:
		// Check for Dockerfile
		if filename == "Dockerfile" || strings.HasSuffix(filename, ".dockerfile") {
			return LanguageDockerfile
		}
		return LanguageUnknown
	}
}

// IsSupported returns true if the language is supported by the code tools
func (l Language) IsSupported() bool {
	// All languages with tree-sitter support are now supported
	switch l {
	case LanguageGo, LanguagePython, LanguageTypeScript, LanguageJavaScript,
		LanguageRust, LanguageJava, LanguageCSharp, LanguageCpp, LanguageC,
		LanguageRuby, LanguagePhp, LanguageBash, LanguageCSS, LanguageCue,
		LanguageDockerfile, LanguageElixir, LanguageElm, LanguageGroovy,
		LanguageHCL, LanguageHTML, LanguageKotlin, LanguageLua, LanguageMarkdown,
		LanguageOCaml, LanguageProtobuf, LanguageScala, LanguageSQL, LanguageSvelte,
		LanguageSwift, LanguageTOML, LanguageYAML:
		return true
	default:
		return false
	}
}

// String returns the string representation of the language
func (l Language) String() string {
	return string(l)
}
