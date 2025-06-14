package parsers

import (
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
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools/code"
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

// RegisterAllParsers registers parsers for all supported languages with an AST tool
func RegisterAllParsers(astTool interface{ RegisterParser(code.Language, code.Parser) }) error {
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