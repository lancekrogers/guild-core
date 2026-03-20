// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/lancekrogers/guild-core/pkg/lsp"
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// LSPContextEnhancer enhances agent context with LSP-derived information
type LSPContextEnhancer struct {
	lspManager *lsp.Manager
}

// NewLSPContextEnhancer creates a new LSP context enhancer
func NewLSPContextEnhancer(lspManager *lsp.Manager) *LSPContextEnhancer {
	return &LSPContextEnhancer{
		lspManager: lspManager,
	}
}

// EnhancedContext represents context enhanced with LSP information
type EnhancedContext struct {
	OriginalContext map[string]interface{} `json:"original_context"`
	Files           []FileContext          `json:"files"`
	Symbols         []SymbolContext        `json:"symbols"`
	Dependencies    []string               `json:"dependencies"`
	ProjectInfo     ProjectInfo            `json:"project_info"`
}

// FileContext represents context for a specific file
type FileContext struct {
	Path         string       `json:"path"`
	Language     string       `json:"language"`
	Symbols      []SymbolInfo `json:"symbols,omitempty"`
	Imports      []string     `json:"imports,omitempty"`
	RelatedFiles []string     `json:"related_files,omitempty"`
}

// SymbolContext represents context for a specific symbol
type SymbolContext struct {
	Name          string     `json:"name"`
	Kind          string     `json:"kind"`
	File          string     `json:"file"`
	Position      Position   `json:"position"`
	Type          string     `json:"type,omitempty"`
	Documentation string     `json:"documentation,omitempty"`
	References    []Location `json:"references,omitempty"`
	Definition    *Location  `json:"definition,omitempty"`
}

// SymbolInfo represents basic symbol information
type SymbolInfo struct {
	Name     string   `json:"name"`
	Kind     string   `json:"kind"`
	Position Position `json:"position"`
}

// Position represents a position in a file
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Location represents a location in a file
type Location struct {
	File     string   `json:"file"`
	Position Position `json:"position"`
}

// ProjectInfo represents project-level information
type ProjectInfo struct {
	RootPath  string   `json:"root_path"`
	Languages []string `json:"languages"`
	Type      string   `json:"type,omitempty"`
}

// EnhanceContext enhances the given context with LSP information
func (e *LSPContextEnhancer) EnhanceContext(ctx context.Context, originalContext map[string]interface{}) (*EnhancedContext, error) {
	logger := observability.GetLogger(ctx)

	enhanced := &EnhancedContext{
		OriginalContext: originalContext,
		Files:           []FileContext{},
		Symbols:         []SymbolContext{},
		Dependencies:    []string{},
	}

	// Extract file paths from context
	filePaths := e.extractFilePaths(originalContext)
	if len(filePaths) == 0 {
		return enhanced, nil
	}

	// Process each file
	for _, filePath := range filePaths {
		fileCtx, err := e.processFile(ctx, filePath)
		if err != nil {
			logger.WarnContext(ctx, "Failed to process file for LSP context",
				"file", filePath,
				"error", err)
			continue
		}
		enhanced.Files = append(enhanced.Files, fileCtx)
	}

	// Extract symbols from context
	symbols := e.extractSymbols(originalContext)
	for _, symbol := range symbols {
		symbolCtx, err := e.processSymbol(ctx, symbol)
		if err != nil {
			logger.WarnContext(ctx, "Failed to process symbol for LSP context",
				"symbol", symbol.Name,
				"error", err)
			continue
		}
		enhanced.Symbols = append(enhanced.Symbols, symbolCtx)
	}

	// Add project information
	if len(filePaths) > 0 {
		enhanced.ProjectInfo = e.gatherProjectInfo(filePaths)
	}

	return enhanced, nil
}

// extractFilePaths extracts file paths from the context
func (e *LSPContextEnhancer) extractFilePaths(context map[string]interface{}) []string {
	var paths []string
	seen := make(map[string]bool)

	// Common keys that might contain file paths
	fileKeys := []string{"file", "files", "path", "paths", "source", "target"}

	for _, key := range fileKeys {
		if value, exists := context[key]; exists {
			switch v := value.(type) {
			case string:
				if !seen[v] && e.isFilePath(v) {
					paths = append(paths, v)
					seen[v] = true
				}
			case []interface{}:
				for _, item := range v {
					if str, ok := item.(string); ok && !seen[str] && e.isFilePath(str) {
						paths = append(paths, str)
						seen[str] = true
					}
				}
			case []string:
				for _, str := range v {
					if !seen[str] && e.isFilePath(str) {
						paths = append(paths, str)
						seen[str] = true
					}
				}
			}
		}
	}

	return paths
}

// isFilePath checks if a string looks like a file path
func (e *LSPContextEnhancer) isFilePath(s string) bool {
	// Check for common file extensions
	extensions := []string{
		".go", ".js", ".ts", ".py", ".java", ".cs", ".rb", ".php",
		".c", ".cpp", ".h", ".hpp", ".rs", ".swift", ".kt",
	}

	for _, ext := range extensions {
		if strings.HasSuffix(s, ext) {
			return true
		}
	}

	return false
}

// extractSymbols extracts symbol information from the context
func (e *LSPContextEnhancer) extractSymbols(context map[string]interface{}) []SymbolContext {
	var symbols []SymbolContext

	// Look for symbol-related keys
	symbolKeys := []string{"symbol", "symbols", "function", "method", "class", "variable"}

	for _, key := range symbolKeys {
		if value, exists := context[key]; exists {
			switch v := value.(type) {
			case string:
				symbols = append(symbols, SymbolContext{
					Name: v,
					Kind: key,
				})
			case map[string]interface{}:
				if name, ok := v["name"].(string); ok {
					symbol := SymbolContext{
						Name: name,
						Kind: key,
					}
					if file, ok := v["file"].(string); ok {
						symbol.File = file
					}
					symbols = append(symbols, symbol)
				}
			}
		}
	}

	return symbols
}

// processFile processes a file to extract LSP context
func (e *LSPContextEnhancer) processFile(ctx context.Context, filePath string) (FileContext, error) {
	fileCtx := FileContext{
		Path:     filePath,
		Language: lsp.DetectLanguage(filePath),
	}

	// Get language server for the file
	server, err := e.lspManager.GetServerForFile(ctx, filePath)
	if err != nil {
		return fileCtx, err
	}

	fileCtx.Language = server.Language

	// TODO: Extract symbols from file using textDocument/documentSymbol
	// TODO: Extract imports/dependencies
	// TODO: Find related files (same package/module)

	return fileCtx, nil
}

// processSymbol processes a symbol to extract LSP context
func (e *LSPContextEnhancer) processSymbol(ctx context.Context, symbol SymbolContext) (SymbolContext, error) {
	if symbol.File == "" {
		return symbol, nil // Can't process without file context
	}

	// TODO: Find symbol position in file
	// TODO: Get hover information for type/documentation
	// TODO: Find references
	// TODO: Find definition

	return symbol, nil
}

// gatherProjectInfo gathers project-level information
func (e *LSPContextEnhancer) gatherProjectInfo(filePaths []string) ProjectInfo {
	info := ProjectInfo{
		Languages: []string{},
	}

	// Find common root path
	if len(filePaths) > 0 {
		info.RootPath = e.findCommonRoot(filePaths)
	}

	// Collect unique languages
	langMap := make(map[string]bool)
	for _, path := range filePaths {
		if lang := lsp.DetectLanguage(path); lang != "" {
			langMap[lang] = true
		}
	}

	for lang := range langMap {
		info.Languages = append(info.Languages, lang)
	}

	// Detect project type based on root markers
	info.Type = e.detectProjectType(info.RootPath)

	return info
}

// findCommonRoot finds the common root directory of multiple file paths
func (e *LSPContextEnhancer) findCommonRoot(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	if len(paths) == 1 {
		return filepath.Dir(paths[0])
	}

	// Split all paths into components
	var splitPaths [][]string
	for _, path := range paths {
		absPath, _ := filepath.Abs(path)
		splitPaths = append(splitPaths, strings.Split(filepath.Dir(absPath), string(filepath.Separator)))
	}

	// Find common prefix
	var common []string
	for i := 0; i < len(splitPaths[0]); i++ {
		component := splitPaths[0][i]
		allMatch := true

		for j := 1; j < len(splitPaths); j++ {
			if i >= len(splitPaths[j]) || splitPaths[j][i] != component {
				allMatch = false
				break
			}
		}

		if allMatch {
			common = append(common, component)
		} else {
			break
		}
	}

	if len(common) == 0 {
		return "/"
	}

	return strings.Join(common, string(filepath.Separator))
}

// detectProjectType detects the type of project based on files in the root
func (e *LSPContextEnhancer) detectProjectType(rootPath string) string {
	if rootPath == "" {
		return ""
	}

	// Check for project type indicators
	indicators := map[string]string{
		"go.mod":           "go",
		"package.json":     "node",
		"requirements.txt": "python",
		"Cargo.toml":       "rust",
		"pom.xml":          "java/maven",
		"build.gradle":     "java/gradle",
		"*.csproj":         "dotnet",
		"Gemfile":          "ruby",
		"composer.json":    "php",
	}

	for file, projectType := range indicators {
		if _, err := filepath.Glob(filepath.Join(rootPath, file)); err == nil {
			return projectType
		}
	}

	return "unknown"
}

// GetMinimalContext returns minimal context needed for a specific task
func (e *LSPContextEnhancer) GetMinimalContext(ctx context.Context, taskDescription string, currentFile string) (map[string]interface{}, error) {
	context := make(map[string]interface{})

	if currentFile == "" {
		return context, nil
	}

	// Add current file information
	context["current_file"] = currentFile
	context["language"] = lsp.DetectLanguage(currentFile)

	// Parse task to determine what context is needed
	if strings.Contains(taskDescription, "implement") || strings.Contains(taskDescription, "create") {
		// For implementation tasks, get interface/type definitions
		// TODO: Use LSP to find relevant interfaces/types
	} else if strings.Contains(taskDescription, "fix") || strings.Contains(taskDescription, "debug") {
		// For bug fixes, get error locations and related code
		// TODO: Use LSP diagnostics
	} else if strings.Contains(taskDescription, "refactor") {
		// For refactoring, get all references to the target
		// TODO: Use LSP references
	}

	return context, nil
}
