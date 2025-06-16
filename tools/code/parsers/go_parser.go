// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parsers

import (
	"context"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/scanner"
	"go/token"
	"strconv"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools/code"
)

// GoParser implements the Parser interface for Go language
type GoParser struct{}

// NewGoParser creates a new Go parser
func NewGoParser() *GoParser {
	return &GoParser{}
}

// Parse parses Go source code and returns the AST
func (p *GoParser) Parse(ctx context.Context, filename string, content []byte) (*code.ParseResult, error) {
	fset := token.NewFileSet()

	// Parse the Go file
	file, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		// Parse errors are collected and returned as part of the result
		var parseErrors []code.ParseError
		if list, ok := err.(scanner.ErrorList); ok {
			for _, e := range list {
				parseErrors = append(parseErrors, code.ParseError{
					Line:     e.Pos.Line,
					Column:   e.Pos.Column,
					Message:  e.Msg,
					Severity: "error",
				})
			}
		} else {
			parseErrors = append(parseErrors, code.ParseError{
				Line:     1,
				Column:   1,
				Message:  err.Error(),
				Severity: "error",
			})
		}

		return &code.ParseResult{
			Language: code.LanguageGo,
			Filename: filename,
			Content:  content,
			AST:      file, // May be partial
			FileSet:  fset,
			Errors:   parseErrors,
		}, nil
	}

	return &code.ParseResult{
		Language: code.LanguageGo,
		Filename: filename,
		Content:  content,
		AST:      file,
		FileSet:  fset,
		Errors:   nil,
		Metadata: map[string]interface{}{
			"package": file.Name.Name,
		},
	}, nil
}

// Language returns the language this parser supports
func (p *GoParser) Language() code.Language {
	return code.LanguageGo
}

// Extensions returns the file extensions this parser handles
func (p *GoParser) Extensions() []string {
	return []string{".go"}
}

// GetFunctions extracts function definitions from the parsed Go AST
func (p *GoParser) GetFunctions(result *code.ParseResult) ([]*code.Function, error) {
	file, ok := result.AST.(*ast.File)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInternal, "invalid AST type for Go parser", nil).
			WithComponent("go_parser").
			WithOperation("get_functions")
	}

	var functions []*code.Function

	// Create a doc package for extracting documentation
	pkg := &ast.Package{
		Name:  file.Name.Name,
		Files: map[string]*ast.File{result.Filename: file},
	}
	docPkg := doc.New(pkg, "", doc.AllDecls)

	// Create a map of function docs
	funcDocs := make(map[string]string)
	for _, f := range docPkg.Funcs {
		funcDocs[f.Name] = f.Doc
	}

	// Walk through the AST to find functions
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			fn := p.extractFunction(node, result.FileSet, funcDocs)
			if fn != nil {
				fn.Package = file.Name.Name
				functions = append(functions, fn)
			}
		}
		return true
	})

	return functions, nil
}

// GetClasses extracts class-like definitions (structs with methods) from Go AST
func (p *GoParser) GetClasses(result *code.ParseResult) ([]*code.Class, error) {
	file, ok := result.AST.(*ast.File)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInternal, "invalid AST type for Go parser", nil).
			WithComponent("go_parser").
			WithOperation("get_classes")
	}

	var classes []*code.Class
	structMethods := make(map[string][]*code.Function)

	// First pass: collect all methods and their receivers
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				// This is a method
				receiverType := p.extractReceiverType(funcDecl.Recv.List[0].Type)
				if receiverType != "" {
					method := p.extractFunction(funcDecl, result.FileSet, nil)
					if method != nil {
						method.IsMethod = true
						structMethods[receiverType] = append(structMethods[receiverType], method)
					}
				}
			}
		}
		return true
	})

	// Second pass: find struct definitions and combine with methods
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			if node.Tok == token.TYPE {
				for _, spec := range node.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							class := p.extractStruct(typeSpec, structType, result.FileSet, file.Name.Name)
							if class != nil {
								// Add methods if any exist for this struct
								if methods, exists := structMethods[class.Name]; exists {
									class.Methods = methods
								}
								classes = append(classes, class)
							}
						}
					}
				}
			}
		}
		return true
	})

	return classes, nil
}

// GetImports extracts import information from Go AST
func (p *GoParser) GetImports(result *code.ParseResult) ([]*code.Import, error) {
	file, ok := result.AST.(*ast.File)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInternal, "invalid AST type for Go parser", nil).
			WithComponent("go_parser").
			WithOperation("get_imports")
	}

	var imports []*code.Import

	for _, imp := range file.Imports {
		importPath, _ := strconv.Unquote(imp.Path.Value)

		goImport := &code.Import{
			Path: importPath,
			Line: result.FileSet.Position(imp.Pos()).Line,
		}

		// Handle import aliases
		if imp.Name != nil {
			switch imp.Name.Name {
			case "_":
				// Blank import
				goImport.Names = []string{"_"}
			case ".":
				// Dot import
				goImport.IsWildcard = true
			default:
				// Named import
				goImport.Alias = imp.Name.Name
			}
		}

		imports = append(imports, goImport)
	}

	return imports, nil
}

// FindSymbol finds a specific symbol in the Go AST
func (p *GoParser) FindSymbol(result *code.ParseResult, symbolName string) ([]*code.Symbol, error) {
	file, ok := result.AST.(*ast.File)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInternal, "invalid AST type for Go parser", nil).
			WithComponent("go_parser").
			WithOperation("find_symbol")
	}

	var symbols []*code.Symbol

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Name.Name == symbolName {
				pos := result.FileSet.Position(node.Pos())
				end := result.FileSet.Position(node.End())

				symbol := &code.Symbol{
					Name:      node.Name.Name,
					Type:      "function",
					Package:   file.Name.Name,
					StartLine: pos.Line,
					EndLine:   end.Line,
					StartCol:  pos.Column,
					EndCol:    end.Column,
				}

				if node.Doc != nil {
					symbol.DocString = node.Doc.Text()
				}

				// Determine visibility
				if ast.IsExported(node.Name.Name) {
					symbol.Visibility = "public"
				} else {
					symbol.Visibility = "private"
				}

				symbols = append(symbols, symbol)
			}

		case *ast.TypeSpec:
			if node.Name.Name == symbolName {
				pos := result.FileSet.Position(node.Pos())
				end := result.FileSet.Position(node.End())

				symbolType := "type"
				if _, ok := node.Type.(*ast.StructType); ok {
					symbolType = "struct"
				} else if _, ok := node.Type.(*ast.InterfaceType); ok {
					symbolType = "interface"
				}

				symbol := &code.Symbol{
					Name:      node.Name.Name,
					Type:      symbolType,
					Package:   file.Name.Name,
					StartLine: pos.Line,
					EndLine:   end.Line,
					StartCol:  pos.Column,
					EndCol:    end.Column,
				}

				if ast.IsExported(node.Name.Name) {
					symbol.Visibility = "public"
				} else {
					symbol.Visibility = "private"
				}

				symbols = append(symbols, symbol)
			}

		case *ast.ValueSpec:
			for _, name := range node.Names {
				if name.Name == symbolName {
					pos := result.FileSet.Position(name.Pos())
					end := result.FileSet.Position(name.End())

					symbol := &code.Symbol{
						Name:      name.Name,
						Type:      "variable",
						Package:   file.Name.Name,
						StartLine: pos.Line,
						EndLine:   end.Line,
						StartCol:  pos.Column,
						EndCol:    end.Column,
					}

					if ast.IsExported(name.Name) {
						symbol.Visibility = "public"
					} else {
						symbol.Visibility = "private"
					}

					symbols = append(symbols, symbol)
				}
			}
		}
		return true
	})

	return symbols, nil
}

// extractFunction converts an ast.FuncDecl to a code.Function
func (p *GoParser) extractFunction(funcDecl *ast.FuncDecl, fset *token.FileSet, funcDocs map[string]string) *code.Function {
	if funcDecl.Name == nil {
		return nil
	}

	pos := fset.Position(funcDecl.Pos())
	end := fset.Position(funcDecl.End())

	fn := &code.Function{
		Name:      funcDecl.Name.Name,
		StartLine: pos.Line,
		EndLine:   end.Line,
	}

	// Extract documentation
	if funcDecl.Doc != nil {
		fn.DocString = funcDecl.Doc.Text()
	} else if doc, exists := funcDocs[fn.Name]; exists {
		fn.DocString = doc
	}

	// Determine visibility
	if ast.IsExported(funcDecl.Name.Name) {
		fn.Visibility = "public"
	} else {
		fn.Visibility = "private"
	}

	// Extract parameters
	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			paramType := p.typeToString(field.Type)
			if len(field.Names) == 0 {
				// Anonymous parameter
				fn.Parameters = append(fn.Parameters, &code.Parameter{
					Name: "",
					Type: paramType,
				})
			} else {
				for _, name := range field.Names {
					param := &code.Parameter{
						Name: name.Name,
						Type: paramType,
					}
					// Check for variadic
					if strings.HasPrefix(paramType, "...") {
						param.IsVariadic = true
						param.Type = strings.TrimPrefix(paramType, "...")
					}
					fn.Parameters = append(fn.Parameters, param)
				}
			}
		}
	}

	// Extract return type
	if funcDecl.Type.Results != nil {
		var returnTypes []string
		for _, field := range funcDecl.Type.Results.List {
			returnTypes = append(returnTypes, p.typeToString(field.Type))
		}
		if len(returnTypes) == 1 {
			fn.ReturnType = returnTypes[0]
		} else if len(returnTypes) > 1 {
			fn.ReturnType = "(" + strings.Join(returnTypes, ", ") + ")"
		}
	}

	// Build signature
	fn.Signature = p.buildFunctionSignature(fn)

	// Check if it's a method
	if funcDecl.Recv != nil {
		fn.IsMethod = true
	}

	return fn
}

// extractStruct converts struct information to a code.Class
func (p *GoParser) extractStruct(typeSpec *ast.TypeSpec, structType *ast.StructType, fset *token.FileSet, packageName string) *code.Class {
	pos := fset.Position(typeSpec.Pos())
	end := fset.Position(typeSpec.End())

	class := &code.Class{
		Name:      typeSpec.Name.Name,
		Package:   packageName,
		StartLine: pos.Line,
		EndLine:   end.Line,
	}

	// Determine visibility
	if ast.IsExported(typeSpec.Name.Name) {
		class.Visibility = "public"
	} else {
		class.Visibility = "private"
	}

	// Extract fields
	if structType.Fields != nil {
		for _, field := range structType.Fields.List {
			fieldType := p.typeToString(field.Type)

			if len(field.Names) == 0 {
				// Embedded field
				class.Fields = append(class.Fields, &code.Field{
					Name:       "",
					Type:       fieldType,
					Visibility: "embedded",
					Line:       fset.Position(field.Pos()).Line,
				})
			} else {
				for _, name := range field.Names {
					fieldVisibility := "private"
					if ast.IsExported(name.Name) {
						fieldVisibility = "public"
					}

					fieldObj := &code.Field{
						Name:       name.Name,
						Type:       fieldType,
						Visibility: fieldVisibility,
						Line:       fset.Position(name.Pos()).Line,
					}

					// Extract field tag if present
					if field.Tag != nil {
						tag, _ := strconv.Unquote(field.Tag.Value)
						if tag != "" {
							fieldObj.DocString = "Tag: " + tag
						}
					}

					class.Fields = append(class.Fields, fieldObj)
				}
			}
		}
	}

	return class
}

// extractReceiverType extracts the receiver type from a method declaration
func (p *GoParser) extractReceiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

// typeToString converts an AST type expression to a string
func (p *GoParser) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.typeToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + p.typeToString(t.Elt)
		}
		return "[...]" + p.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + p.typeToString(t.Key) + "]" + p.typeToString(t.Value)
	case *ast.ChanType:
		prefix := "chan"
		if t.Dir == ast.RECV {
			prefix = "<-chan"
		} else if t.Dir == ast.SEND {
			prefix = "chan<-"
		}
		return prefix + " " + p.typeToString(t.Value)
	case *ast.FuncType:
		return "func(...)"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.SelectorExpr:
		return p.typeToString(t.X) + "." + t.Sel.Name
	case *ast.Ellipsis:
		return "..." + p.typeToString(t.Elt)
	default:
		return "unknown"
	}
}

// buildFunctionSignature builds a readable function signature
func (p *GoParser) buildFunctionSignature(fn *code.Function) string {
	var sig strings.Builder

	sig.WriteString("func ")
	sig.WriteString(fn.Name)
	sig.WriteString("(")

	for i, param := range fn.Parameters {
		if i > 0 {
			sig.WriteString(", ")
		}
		if param.Name != "" {
			sig.WriteString(param.Name)
			sig.WriteString(" ")
		}
		if param.IsVariadic {
			sig.WriteString("...")
		}
		sig.WriteString(param.Type)
	}

	sig.WriteString(")")

	if fn.ReturnType != "" {
		sig.WriteString(" ")
		sig.WriteString(fn.ReturnType)
	}

	return sig.String()
}
