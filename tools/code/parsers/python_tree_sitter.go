// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parsers

import (
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools/code"
	sitter "github.com/smacker/go-tree-sitter"
	python "github.com/smacker/go-tree-sitter/python"
)

// PythonTreeSitterParser implements Python parsing using tree-sitter
type PythonTreeSitterParser struct {
	*TreeSitterParser
}

// NewPythonParser creates a new Python parser
func NewPythonParser() *PythonTreeSitterParser {
	return &PythonTreeSitterParser{
		TreeSitterParser: NewTreeSitterParser(
			code.LanguagePython,
			[]string{".py", ".pyw"},
			python.GetLanguage(),
		),
	}
}

// GetFunctions extracts all functions from the parse result
func (p *PythonTreeSitterParser) GetFunctions(result *code.ParseResult) ([]*code.Function, error) {
	tree, ok := result.AST.(*sitter.Tree)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInternal, "invalid AST type for Python parser", nil).
			WithComponent("python_parser").
			WithOperation("get_functions")
	}

	// Python function query
	queryStr := `
(function_definition
	name: (identifier) @name
	parameters: (parameters) @params
	return_type: (type)? @return_type
	body: (block) @body) @function
	`

	query, err := p.LoadQuery(QueryFunctions, queryStr)
	if err != nil {
		return nil, err
	}

	var functions []*code.Function
	qc := sitter.NewQueryCursor()
	qc.Exec(query, tree.RootNode())

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		fn := &code.Function{}

		for _, c := range m.Captures {
			node := c.Node
			switch query.CaptureNameForId(c.Index) {
			case "name":
				fn.Name = p.nodeText(node, result.Content)
			case "params":
				fn.Parameters = p.extractParameters(node, result.Content)
			case "return_type":
				if node.ChildCount() > 0 {
					// Extract the identifier from the type node
					fn.ReturnType = p.nodeText(node.Child(0), result.Content)
				} else {
					fn.ReturnType = p.nodeText(node, result.Content)
				}
			case "function":
				fn.StartLine = int(node.StartPoint().Row) + 1
				fn.EndLine = int(node.EndPoint().Row) + 1

				// Check if function is async
				if node.Child(0) != nil && node.Child(0).Type() == "async" {
					fn.IsStatic = true // Using IsStatic to indicate async
				}

				// Extract docstring
				if body := node.ChildByFieldName("body"); body != nil {
					if firstStmt := body.Child(0); firstStmt != nil && firstStmt.Type() == "expression_statement" {
						if strNode := firstStmt.Child(0); strNode != nil && strNode.Type() == "string" {
							fn.DocString = p.cleanDocstring(p.nodeText(strNode, result.Content))
						}
					}
				}

				// Extract decorators
				if parent := node.Parent(); parent != nil && parent.Type() == "decorated_definition" {
					fn.Decorators = p.extractDecorators(parent, result.Content)
				}
			}
		}

		// Build signature
		fn.Signature = p.buildFunctionSignature(fn)
		functions = append(functions, fn)
	}

	return functions, nil
}

// GetClasses extracts all classes from the parse result
func (p *PythonTreeSitterParser) GetClasses(result *code.ParseResult) ([]*code.Class, error) {
	tree, ok := result.AST.(*sitter.Tree)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInternal, "invalid AST type for Python parser", nil).
			WithComponent("python_parser").
			WithOperation("get_classes")
	}

	queryStr := `
	(class_definition
		name: (identifier) @name
		superclasses: (argument_list)? @bases
		body: (block) @body) @class
	`

	query, err := p.LoadQuery(QueryClasses, queryStr)
	if err != nil {
		return nil, err
	}

	var classes []*code.Class
	qc := sitter.NewQueryCursor()
	qc.Exec(query, tree.RootNode())

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		class := &code.Class{
			Methods: []*code.Function{},
			Fields:  []*code.Field{},
		}

		for _, c := range m.Captures {
			node := c.Node
			switch query.CaptureNameForId(c.Index) {
			case "name":
				class.Name = p.nodeText(node, result.Content)
			case "bases":
				class.BaseClasses = p.extractBaseClasses(node, result.Content)
			case "class":
				class.StartLine = int(node.StartPoint().Row) + 1
				class.EndLine = int(node.EndPoint().Row) + 1
				class.Visibility = "public" // Python doesn't have explicit visibility

				// Extract docstring and methods
				if body := node.ChildByFieldName("body"); body != nil {
					class.Methods = p.extractMethods(body, result.Content)

					// Check for docstring
					if firstStmt := body.Child(0); firstStmt != nil && firstStmt.Type() == "expression_statement" {
						if strNode := firstStmt.Child(0); strNode != nil && strNode.Type() == "string" {
							class.DocString = p.cleanDocstring(p.nodeText(strNode, result.Content))
						}
					}
				}

				// Extract decorators
				if parent := node.Parent(); parent != nil && parent.Type() == "decorated_definition" {
					class.Decorators = p.extractDecorators(parent, result.Content)
				}
			}
		}

		classes = append(classes, class)
	}

	return classes, nil
}

// GetImports extracts all imports from the parse result
func (p *PythonTreeSitterParser) GetImports(result *code.ParseResult) ([]*code.Import, error) {
	tree, ok := result.AST.(*sitter.Tree)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInternal, "invalid AST type for Python parser", nil).
			WithComponent("python_parser").
			WithOperation("get_imports")
	}

	queryStr := `
(import_statement
	name: (dotted_name) @name) @import

(import_statement
	name: (aliased_import
		name: (dotted_name) @name
		alias: (identifier) @alias)) @import

(import_from_statement
	module_name: (dotted_name)? @module
	name: (dotted_name) @name) @import

(import_from_statement
	module_name: (dotted_name)? @module
	name: (aliased_import
		name: (dotted_name) @name
		alias: (identifier) @alias)) @import
	`

	query, err := p.LoadQuery(QueryImports, queryStr)
	if err != nil {
		return nil, err
	}

	var imports []*code.Import
	qc := sitter.NewQueryCursor()
	qc.Exec(query, tree.RootNode())

	// Track processed imports to avoid duplicates
	seen := make(map[string]bool)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		imp := &code.Import{}
		var module, name, alias string

		for _, c := range m.Captures {
			node := c.Node
			switch query.CaptureNameForId(c.Index) {
			case "module":
				module = p.nodeText(node, result.Content)
			case "name":
				name = p.nodeText(node, result.Content)
			case "alias":
				alias = p.nodeText(node, result.Content)
			}
		}

		// Build import path
		if module != "" {
			imp.Path = module + "." + name
		} else {
			imp.Path = name
		}

		imp.Alias = alias
		imp.Line = int(m.Captures[0].Node.StartPoint().Row) + 1

		// Avoid duplicates
		key := imp.Path + "|" + imp.Alias
		if !seen[key] {
			seen[key] = true
			imports = append(imports, imp)
		}
	}

	return imports, nil
}

// FindSymbol finds all occurrences of a symbol
func (p *PythonTreeSitterParser) FindSymbol(result *code.ParseResult, symbolName string) ([]*code.Symbol, error) {
	tree, ok := result.AST.(*sitter.Tree)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInternal, "invalid AST type for Python parser", nil).
			WithComponent("python_parser").
			WithOperation("find_symbol")
	}

	// Query for identifiers matching the symbol name
	queryStr := `(identifier) @id`

	query, err := p.LoadQuery(QuerySymbols, queryStr)
	if err != nil {
		return nil, err
	}

	var symbols []*code.Symbol
	qc := sitter.NewQueryCursor()
	qc.Exec(query, tree.RootNode())

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		for _, c := range m.Captures {
			node := c.Node
			if p.nodeText(node, result.Content) == symbolName {
				symbol := &code.Symbol{
					Name:      symbolName,
					Type:      p.determineSymbolType(node),
					StartLine: int(node.StartPoint().Row) + 1,
					EndLine:   int(node.EndPoint().Row) + 1,
					StartCol:  int(node.StartPoint().Column) + 1,
					EndCol:    int(node.EndPoint().Column) + 1,
				}
				symbols = append(symbols, symbol)
			}
		}
	}

	return symbols, nil
}

// Helper methods

func (p *PythonTreeSitterParser) extractParameters(node *sitter.Node, content []byte) []*code.Parameter {
	var params []*code.Parameter

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		switch child.Type() {
		case "identifier":
			param := &code.Parameter{
				Name: p.nodeText(child, content),
			}
			params = append(params, param)

		case "typed_parameter", "typed_default_parameter", "default_parameter":
			param := &code.Parameter{}

			// Find the name and type
			for i := 0; i < int(child.ChildCount()); i++ {
				subchild := child.Child(i)
				switch subchild.Type() {
				case "identifier":
					if param.Name == "" {
						param.Name = p.nodeText(subchild, content)
					}
				case "type":
					// Extract the type identifier
					if subchild.ChildCount() > 0 {
						param.Type = p.nodeText(subchild.Child(0), content)
					} else {
						param.Type = p.nodeText(subchild, content)
					}
				case "=":
					// Mark as having default value
					param.IsOptional = true
				}
			}

			params = append(params, param)

		case "list_splat_pattern":
			params = append(params, &code.Parameter{
				Name:       p.nodeText(child, content),
				IsVariadic: true,
			})
		case "dictionary_splat_pattern":
			params = append(params, &code.Parameter{
				Name:       p.nodeText(child, content),
				IsVariadic: true,
			})
		}
	}

	return params
}

func (p *PythonTreeSitterParser) extractDecorators(node *sitter.Node, content []byte) []string {
	var decorators []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "decorator" {
			decorators = append(decorators, p.nodeText(child, content))
		}
	}

	return decorators
}

func (p *PythonTreeSitterParser) extractMethods(classBody *sitter.Node, content []byte) []*code.Function {
	var methods []*code.Function

	for i := 0; i < int(classBody.ChildCount()); i++ {
		child := classBody.Child(i)

		if child.Type() == "function_definition" {
			method := &code.Function{
				Name:      p.nodeText(child.ChildByFieldName("name"), content),
				StartLine: int(child.StartPoint().Row) + 1,
				EndLine:   int(child.EndPoint().Row) + 1,
				IsMethod:  true,
			}

			// Check if method is async
			if child.Child(0) != nil && child.Child(0).Type() == "async" {
				method.IsStatic = true // Using IsStatic to indicate async
			}

			if params := child.ChildByFieldName("parameters"); params != nil {
				method.Parameters = p.extractParameters(params, content)
			}

			// Build signature
			method.Signature = p.buildFunctionSignature(method)
			methods = append(methods, method)
		}
	}

	return methods
}

func (p *PythonTreeSitterParser) extractBaseClasses(node *sitter.Node, content []byte) []string {
	var bases []string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" || child.Type() == "attribute" {
			bases = append(bases, p.nodeText(child, content))
		}
	}

	return bases
}

func (p *PythonTreeSitterParser) cleanDocstring(doc string) string {
	// Remove quotes
	doc = strings.Trim(doc, `"'`)

	// Handle triple quotes
	if strings.HasPrefix(doc, `""`) || strings.HasPrefix(doc, `''`) {
		doc = doc[2 : len(doc)-2]
	}

	return strings.TrimSpace(doc)
}

func (p *PythonTreeSitterParser) determineSymbolType(node *sitter.Node) string {
	parent := node.Parent()
	if parent == nil {
		return "variable"
	}

	switch parent.Type() {
	case "function_definition", "async_function_definition":
		if parent.ChildByFieldName("name") == node {
			return "function"
		}
	case "class_definition":
		if parent.ChildByFieldName("name") == node {
			return "class"
		}
	case "import_statement", "import_from_statement":
		return "import"
	}

	return "variable"
}

func (p *PythonTreeSitterParser) buildFunctionSignature(fn *code.Function) string {
	var sig strings.Builder

	if fn.IsStatic {
		sig.WriteString("async ")
	}
	sig.WriteString("def ")
	sig.WriteString(fn.Name)
	sig.WriteString("(")

	for i, param := range fn.Parameters {
		if i > 0 {
			sig.WriteString(", ")
		}
		sig.WriteString(param.Name)
		if param.Type != "" {
			sig.WriteString(": ")
			sig.WriteString(param.Type)
		}
	}

	sig.WriteString(")")

	if fn.ReturnType != "" {
		sig.WriteString(" -> ")
		sig.WriteString(fn.ReturnType)
	}

	return sig.String()
}
