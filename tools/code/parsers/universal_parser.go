// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parsers

import (
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/tools/code"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/cue"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"github.com/smacker/go-tree-sitter/elixir"
	"github.com/smacker/go-tree-sitter/elm"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/groovy"
	"github.com/smacker/go-tree-sitter/hcl"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/lua"
	markdown "github.com/smacker/go-tree-sitter/markdown/tree-sitter-markdown"
	"github.com/smacker/go-tree-sitter/ocaml"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/protobuf"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/scala"
	"github.com/smacker/go-tree-sitter/sql"
	"github.com/smacker/go-tree-sitter/svelte"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/toml"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/yaml"
)

// languageGrammars maps language types to their tree-sitter grammars
var languageGrammars = map[code.Language]*sitter.Language{
	code.LanguageBash:       bash.GetLanguage(),
	code.LanguageC:          c.GetLanguage(),
	code.LanguageCpp:        cpp.GetLanguage(),
	code.LanguageCSharp:     csharp.GetLanguage(),
	code.LanguageCSS:        css.GetLanguage(),
	code.LanguageCue:        cue.GetLanguage(),
	code.LanguageDockerfile: dockerfile.GetLanguage(),
	code.LanguageElixir:     elixir.GetLanguage(),
	code.LanguageElm:        elm.GetLanguage(),
	code.LanguageGo:         golang.GetLanguage(),
	code.LanguageGroovy:     groovy.GetLanguage(),
	code.LanguageHCL:        hcl.GetLanguage(),
	code.LanguageHTML:       html.GetLanguage(),
	code.LanguageJava:       java.GetLanguage(),
	code.LanguageJavaScript: javascript.GetLanguage(),
	code.LanguageKotlin:     kotlin.GetLanguage(),
	code.LanguageLua:        lua.GetLanguage(),
	code.LanguageMarkdown:   markdown.GetLanguage(),
	code.LanguageOCaml:      ocaml.GetLanguage(),
	code.LanguagePhp:        php.GetLanguage(),
	code.LanguageProtobuf:   protobuf.GetLanguage(),
	code.LanguagePython:     python.GetLanguage(),
	code.LanguageRuby:       ruby.GetLanguage(),
	code.LanguageRust:       rust.GetLanguage(),
	code.LanguageScala:      scala.GetLanguage(),
	code.LanguageSQL:        sql.GetLanguage(),
	code.LanguageSvelte:     svelte.GetLanguage(),
	code.LanguageSwift:      swift.GetLanguage(),
	code.LanguageTOML:       toml.GetLanguage(),
	code.LanguageTypeScript: typescript.GetLanguage(),
	code.LanguageYAML:       yaml.GetLanguage(),
}

// languageExtensions maps languages to their file extensions
var languageExtensions = map[code.Language][]string{
	code.LanguageBash:       {".sh", ".bash"},
	code.LanguageC:          {".c", ".h"},
	code.LanguageCpp:        {".cpp", ".cc", ".cxx", ".hpp", ".h++"},
	code.LanguageCSharp:     {".cs"},
	code.LanguageCSS:        {".css", ".scss", ".sass", ".less"},
	code.LanguageCue:        {".cue"},
	code.LanguageDockerfile: {"Dockerfile", ".dockerfile"},
	code.LanguageElixir:     {".ex", ".exs"},
	code.LanguageElm:        {".elm"},
	code.LanguageGo:         {".go"},
	code.LanguageGroovy:     {".groovy", ".gvy", ".gy", ".gsh"},
	code.LanguageHCL:        {".hcl", ".tf"},
	code.LanguageHTML:       {".html", ".htm", ".xhtml"},
	code.LanguageJava:       {".java"},
	code.LanguageJavaScript: {".js", ".jsx", ".mjs"},
	code.LanguageKotlin:     {".kt", ".kts"},
	code.LanguageLua:        {".lua"},
	code.LanguageMarkdown:   {".md", ".markdown", ".mdown", ".mkd"},
	code.LanguageOCaml:      {".ml", ".mli"},
	code.LanguagePhp:        {".php", ".php3", ".php4", ".php5", ".phtml"},
	code.LanguageProtobuf:   {".proto"},
	code.LanguagePython:     {".py", ".pyw"},
	code.LanguageRuby:       {".rb"},
	code.LanguageRust:       {".rs"},
	code.LanguageScala:      {".scala", ".sc"},
	code.LanguageSQL:        {".sql"},
	code.LanguageSvelte:     {".svelte"},
	code.LanguageSwift:      {".swift"},
	code.LanguageTOML:       {".toml"},
	code.LanguageTypeScript: {".ts", ".tsx"},
	code.LanguageYAML:       {".yaml", ".yml"},
}

// UniversalParser is a generic tree-sitter parser that can handle any supported language
type UniversalParser struct {
	*TreeSitterParser
}

// NewUniversalParser creates a parser for the specified language
func NewUniversalParser(language code.Language) (*UniversalParser, error) {
	grammar, ok := languageGrammars[language]
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInvalidInput,
			"unsupported language: "+string(language), nil).
			WithComponent("universal_parser").
			WithOperation("new_parser")
	}

	extensions, ok := languageExtensions[language]
	if !ok {
		extensions = []string{} // Use empty extensions for unknown languages
	}

	return &UniversalParser{
		TreeSitterParser: NewTreeSitterParser(language, extensions, grammar),
	}, nil
}

// CreateParser is a factory function that creates the appropriate parser for a language
func CreateParser(language code.Language) (code.Parser, error) {
	// Special cases for languages with custom implementations
	switch language {
	case code.LanguageGo:
		// Use the existing Go parser with AST support
		return NewGoParser(), nil
	case code.LanguagePython:
		// Use the custom Python parser
		return NewPythonParser(), nil
	default:
		// Use the universal parser for all other languages
		return NewUniversalParser(language)
	}
}

// GetSupportedLanguages returns all languages supported by tree-sitter
func GetSupportedLanguages() []code.Language {
	languages := make([]code.Language, 0, len(languageGrammars))
	for lang := range languageGrammars {
		languages = append(languages, lang)
	}
	return languages
}

// GetLanguageExtensions returns the file extensions for a language
func GetLanguageExtensions(language code.Language) []string {
	return languageExtensions[language]
}

// GetFunctions extracts functions based on the language
func (p *UniversalParser) GetFunctions(result *code.ParseResult) ([]*code.Function, error) {
	if result.AST == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "no AST provided", nil).
			WithComponent("universal_parser").
			WithOperation("get_functions")
	}

	tree, ok := result.AST.(*sitter.Tree)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "invalid AST type", nil).
			WithComponent("universal_parser").
			WithOperation("get_functions")
	}

	var functions []*code.Function

	// Define queries based on language
	var queryString string
	switch p.language {
	case code.LanguageJavaScript, code.LanguageTypeScript:
		queryString = `
			(function_declaration
				name: (identifier) @name) @function
			(method_definition
				name: (property_identifier) @name) @function
			(variable_declarator
				name: (identifier) @name
				value: (arrow_function)) @function
		`
	case code.LanguagePython:
		queryString = `
			(function_definition
				name: (identifier) @name) @function
		`
	case code.LanguageRust:
		queryString = `
			(function_item
				name: (identifier) @name) @function
		`
	case code.LanguageGo:
		queryString = `
			(function_declaration
				name: (identifier) @name) @function
			(method_declaration
				name: (field_identifier) @name) @function
		`
	default:
		// Return empty for unsupported languages
		return functions, nil
	}

	query, err := p.LoadQuery(QueryFunctions, queryString)
	if err != nil {
		return nil, err
	}

	cursor := sitter.NewQueryCursor()
	cursor.Exec(query, tree.RootNode())

	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		var funcNode, nameNode *sitter.Node
		for _, capture := range match.Captures {
			switch query.CaptureNameForId(capture.Index) {
			case "function":
				funcNode = capture.Node
			case "name":
				nameNode = capture.Node
			}
		}

		if funcNode != nil && nameNode != nil {
			function := &code.Function{
				Name:      p.nodeText(nameNode, result.Content),
				StartLine: int(funcNode.StartPoint().Row) + 1,
				EndLine:   int(funcNode.EndPoint().Row) + 1,
				Signature: p.nodeText(funcNode, result.Content),
			}
			functions = append(functions, function)
		}
	}

	return functions, nil
}

// GetClasses extracts classes based on the language
func (p *UniversalParser) GetClasses(result *code.ParseResult) ([]*code.Class, error) {
	if result.AST == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "no AST provided", nil).
			WithComponent("universal_parser").
			WithOperation("get_classes")
	}

	tree, ok := result.AST.(*sitter.Tree)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "invalid AST type", nil).
			WithComponent("universal_parser").
			WithOperation("get_classes")
	}

	var classes []*code.Class

	// Define queries based on language
	var queryString string
	switch p.language {
	case code.LanguageJavaScript, code.LanguageTypeScript:
		queryString = `
			(class_declaration
				name: (identifier) @name) @class
		`
	case code.LanguagePython:
		queryString = `
			(class_definition
				name: (identifier) @name) @class
		`
	case code.LanguageRust:
		queryString = `
			(struct_item
				name: (type_identifier) @name) @class
		`
	case code.LanguageJava, code.LanguageCSharp:
		queryString = `
			(class_declaration
				name: (identifier) @name) @class
		`
	default:
		// Return empty for unsupported languages
		return classes, nil
	}

	query, err := p.LoadQuery(QueryClasses, queryString)
	if err != nil {
		return nil, err
	}

	cursor := sitter.NewQueryCursor()
	cursor.Exec(query, tree.RootNode())

	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		var classNode, nameNode *sitter.Node
		for _, capture := range match.Captures {
			switch query.CaptureNameForId(capture.Index) {
			case "class":
				classNode = capture.Node
			case "name":
				nameNode = capture.Node
			}
		}

		if classNode != nil && nameNode != nil {
			class := &code.Class{
				Name:      p.nodeText(nameNode, result.Content),
				StartLine: int(classNode.StartPoint().Row) + 1,
				EndLine:   int(classNode.EndPoint().Row) + 1,
			}
			classes = append(classes, class)
		}
	}

	return classes, nil
}

// RegisterAllParsers registers parsers for all supported languages with an AST tool
func RegisterAllParsers(astTool interface {
	RegisterParser(code.Language, code.Parser)
},
) error {
	for language := range languageGrammars {
		parser, err := CreateParser(language)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create parser").
				WithComponent("universal_parser").
				WithOperation("register_all").
				WithDetails("language", string(language))
		}
		astTool.RegisterParser(language, parser)
	}
	return nil
}
