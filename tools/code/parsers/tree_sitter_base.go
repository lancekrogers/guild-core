package parsers

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools/code"
)

// TreeSitterParser provides base functionality for all tree-sitter based parsers
type TreeSitterParser struct {
	language   code.Language
	extensions []string
	parser     *sitter.Parser
	grammar    *sitter.Language
	queryCache map[string]*sitter.Query
}

// Common queries used across languages
const (
	QueryFunctions = "functions"
	QueryClasses   = "classes"
	QueryImports   = "imports"
	QuerySymbols   = "symbols"
)

// NewTreeSitterParser creates a new tree-sitter based parser
func NewTreeSitterParser(lang code.Language, extensions []string, grammar *sitter.Language) *TreeSitterParser {
	parser := sitter.NewParser()
	parser.SetLanguage(grammar)

	return &TreeSitterParser{
		language:   lang,
		extensions: extensions,
		parser:     parser,
		grammar:    grammar,
		queryCache: make(map[string]*sitter.Query),
	}
}

// Parse implements the Parser interface
func (p *TreeSitterParser) Parse(ctx context.Context, filename string, content []byte) (*code.ParseResult, error) {
	tree, err := p.parser.ParseCtx(ctx, nil, content)
	if err != nil {
		return nil, gerror.Newf(gerror.ErrCodeInternal, "failed to parse file: %s", filename).
			WithComponent("tree_sitter_parser").
			WithOperation("parse").
			WithDetails("language", p.language.String())
	}

	return &code.ParseResult{
		Language: p.language,
		Filename: filename,
		Content:  content,
		AST:      tree,
		Errors:   p.extractErrors(tree),
		Metadata: map[string]interface{}{
			"parser": "tree-sitter",
		},
	}, nil
}

// Language returns the language this parser handles
func (p *TreeSitterParser) Language() code.Language {
	return p.language
}

// Extensions returns the file extensions this parser handles
func (p *TreeSitterParser) Extensions() []string {
	return p.extensions
}

// LoadQuery loads and caches a query for the language
func (p *TreeSitterParser) LoadQuery(name, queryString string) (*sitter.Query, error) {
	if query, exists := p.queryCache[name]; exists {
		return query, nil
	}

	query, err := sitter.NewQuery([]byte(queryString), p.grammar)
	if err != nil {
		return nil, gerror.Newf(gerror.ErrCodeInternal, "failed to create query %s: %v", name, err).
			WithComponent("tree_sitter_parser").
			WithOperation("load_query").
			WithDetails("language", p.language.String()).
			WithDetails("query", queryString)
	}

	p.queryCache[name] = query
	return query, nil
}

// extractErrors extracts parse errors from the tree
func (p *TreeSitterParser) extractErrors(tree *sitter.Tree) []code.ParseError {
	var errors []code.ParseError

	// Tree-sitter marks error nodes in the AST
	cursor := sitter.NewTreeCursor(tree.RootNode())
	defer cursor.Close()

	var visitNode func(*sitter.TreeCursor) bool
	visitNode = func(c *sitter.TreeCursor) bool {
		node := c.CurrentNode()
		if node.IsError() || node.IsMissing() {
			errors = append(errors, code.ParseError{
				Line:     int(node.StartPoint().Row) + 1,
				Column:   int(node.StartPoint().Column) + 1,
				Message:  fmt.Sprintf("Syntax error: %s", node.Type()),
				Severity: "error",
			})
		}

		if c.GoToFirstChild() {
			for {
				visitNode(c)
				if !c.GoToNextSibling() {
					break
				}
			}
			c.GoToParent()
		}
		return true
	}

	visitNode(cursor)
	return errors
}

// Helper to extract text from a node
func (p *TreeSitterParser) nodeText(node *sitter.Node, content []byte) string {
	return string(content[node.StartByte():node.EndByte()])
}

// GetFunctions provides default implementation (can be overridden)
func (p *TreeSitterParser) GetFunctions(result *code.ParseResult) ([]*code.Function, error) {
	// Default implementation returns empty list
	// Languages should override this method with their specific implementation
	return []*code.Function{}, nil
}

// GetClasses provides default implementation (can be overridden)
func (p *TreeSitterParser) GetClasses(result *code.ParseResult) ([]*code.Class, error) {
	// Default implementation returns empty list
	// Languages should override this method with their specific implementation
	return []*code.Class{}, nil
}

// GetImports provides default implementation (can be overridden)
func (p *TreeSitterParser) GetImports(result *code.ParseResult) ([]*code.Import, error) {
	// Default implementation returns empty list
	// Languages should override this method with their specific implementation
	return []*code.Import{}, nil
}

// FindSymbol provides default implementation (can be overridden)
func (p *TreeSitterParser) FindSymbol(result *code.ParseResult, symbolName string) ([]*code.Symbol, error) {
	// Default implementation returns empty list
	// Languages should override this method with their specific implementation
	return []*code.Symbol{}, nil
}