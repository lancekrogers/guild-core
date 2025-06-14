package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools"
)

// GrepTool provides regex content search functionality for agents
type GrepTool struct {
	*tools.BaseTool
	basePath string // Base path to restrict searches to
}

// GrepInput represents the input parameters for grep operations
type GrepInput struct {
	Pattern string `json:"pattern"`         // Regular expression pattern to search for
	Include string `json:"include,omitempty"` // File inclusion pattern (e.g., "*.js", "*.{ts,tsx}")
	Path    string `json:"path,omitempty"`  // Directory path to search in (defaults to current directory)
}

// GrepResult represents a single match result
type GrepResult struct {
	FilePath string `json:"file_path"`
	LineNum  int    `json:"line_num"`
	Line     string `json:"line"`
	Match    string `json:"match"`
	Column   int    `json:"column"`
}

// GrepOutput represents the complete grep operation result
type GrepOutput struct {
	Pattern     string       `json:"pattern"`
	Include     string       `json:"include,omitempty"`
	Path        string       `json:"path"`
	Results     []GrepResult `json:"results"`
	FileCount   int          `json:"file_count"`
	MatchCount  int          `json:"match_count"`
	Duration    string       `json:"duration"`
	FilesScanned int         `json:"files_scanned"`
}

// NewGrepTool creates a new grep tool
func NewGrepTool(basePath string) *GrepTool {
	if basePath == "" {
		// Default to current directory if none provided
		basePath, _ = os.Getwd()
	}

	// Ensure the base path exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		// Use current directory if base path doesn't exist
		basePath, _ = os.Getwd()
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Regular expression pattern to search for in file contents",
			},
			"include": map[string]interface{}{
				"type":        "string",
				"description": "File pattern to include in the search (e.g. \"*.js\", \"*.{ts,tsx}\")",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The directory to search in. Defaults to the current working directory.",
			},
		},
		"required": []string{"pattern"},
	}

	examples := []string{
		`{"pattern": "TODO"}`,
		`{"pattern": "function\\s+\\w+", "include": "*.js"}`,
		`{"pattern": "import.*react", "include": "*.{ts,tsx}"}`,
		`{"pattern": "error", "path": "src", "include": "*.go"}`,
		`{"pattern": "log\\..*Error", "include": "*.py"}`,
	}

	baseTool := tools.NewBaseTool(
		"grep",
		"Fast content search tool that works with any codebase size. Searches file contents using regular expressions. Supports full regex syntax and file pattern filtering.",
		schema,
		"filesystem",
		false,
		examples,
	)

	return &GrepTool{
		BaseTool: baseTool,
		basePath: basePath,
	}
}

// Execute runs the grep tool with the given input
func (t *GrepTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	startTime := time.Now()
	
	var params GrepInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("grep_tool").
			WithOperation("execute")
	}

	// Validate required parameters
	if params.Pattern == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "pattern is required", nil).
			WithComponent("grep_tool").
			WithOperation("execute")
	}

	// Set default path if not provided
	if params.Path == "" {
		params.Path = "."
	}

	// Resolve and validate the search path
	searchPath := t.resolvePath(params.Path)
	if searchPath == "" {
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "invalid or unsafe path: %s", params.Path).
			WithComponent("grep_tool").
			WithOperation("execute")
	}

	// Compile the regex pattern
	pattern, err := regexp.Compile(params.Pattern)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid regex pattern").
			WithComponent("grep_tool").
			WithOperation("execute")
	}

	// Find files to search
	files, err := t.findFiles(searchPath, params.Include)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to find files").
			WithComponent("grep_tool").
			WithOperation("execute")
	}

	// Files found for searching

	// Search files for matches
	results, err := t.searchFiles(pattern, files)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to search files").
			WithComponent("grep_tool").
			WithOperation("execute")
	}

	duration := time.Since(startTime)

	// Create output structure
	output := GrepOutput{
		Pattern:      params.Pattern,
		Include:      params.Include,
		Path:         params.Path,
		Results:      results,
		FileCount:    t.countUniqueFiles(results),
		MatchCount:   len(results),
		Duration:     duration.String(),
		FilesScanned: len(files),
	}

	// Format the output
	outputStr := t.formatOutput(output)

	metadata := map[string]string{
		"pattern":       params.Pattern,
		"include":       params.Include,
		"path":          params.Path,
		"match_count":   fmt.Sprintf("%d", output.MatchCount),
		"file_count":    fmt.Sprintf("%d", output.FileCount),
		"files_scanned": fmt.Sprintf("%d", output.FilesScanned),
		"duration":      duration.String(),
	}

	extraData := map[string]interface{}{
		"output": output,
	}

	logger.Info("Grep search completed", 
		observability.Int("matches", output.MatchCount),
		observability.Int("files_with_matches", output.FileCount),
		observability.String("duration", duration.String()))

	return tools.NewToolResult(outputStr, metadata, nil, extraData), nil
}

// resolvePath resolves and validates the search path
func (t *GrepTool) resolvePath(path string) string {
	// Convert to absolute path
	var absPath string
	if filepath.IsAbs(path) {
		absPath = path
	} else {
		absPath = filepath.Join(t.basePath, path)
	}

	// Clean the path
	absPath = filepath.Clean(absPath)

	// Ensure the path is within the base path for security
	if !strings.HasPrefix(absPath, t.basePath) {
		return ""
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return ""
	}

	return absPath
}

// findFiles finds all files matching the include pattern in the given directory
func (t *GrepTool) findFiles(searchPath, includePattern string) ([]string, error) {
	var files []string
	var patterns []string

	// Parse include pattern
	if includePattern == "" {
		// Default to all files
		patterns = []string{"*"}
	} else if strings.Contains(includePattern, "{") && strings.Contains(includePattern, "}") {
		// Handle patterns like "*.{ts,tsx,js}"
		patterns = t.expandBracePattern(includePattern)
	} else {
		patterns = []string{includePattern}
	}

	// Walk the directory tree
	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip very large files (>10MB) to avoid memory issues
		if info.Size() > 10*1024*1024 {
			return nil
		}

		// Check if file matches any of the patterns
		for _, pattern := range patterns {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err != nil {
				continue
			}
			if matched {
				files = append(files, path)
				break
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort files by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		infoI, errI := os.Stat(files[i])
		infoJ, errJ := os.Stat(files[j])
		if errI != nil || errJ != nil {
			return files[i] < files[j] // fallback to alphabetical
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return files, nil
}

// expandBracePattern expands patterns like "*.{ts,tsx}" to ["*.ts", "*.tsx"]
func (t *GrepTool) expandBracePattern(pattern string) []string {
	// Find the brace section
	start := strings.Index(pattern, "{")
	end := strings.Index(pattern, "}")
	
	if start == -1 || end == -1 || end <= start {
		return []string{pattern}
	}

	prefix := pattern[:start]
	suffix := pattern[end+1:]
	braceContent := pattern[start+1 : end]

	// Split by comma and create patterns
	parts := strings.Split(braceContent, ",")
	var expanded []string
	
	for _, part := range parts {
		expanded = append(expanded, prefix+strings.TrimSpace(part)+suffix)
	}

	return expanded
}

// searchFiles searches for the pattern in all provided files
func (t *GrepTool) searchFiles(pattern *regexp.Regexp, files []string) ([]GrepResult, error) {
	var results []GrepResult

	for _, filePath := range files {
		fileResults, err := t.searchFile(pattern, filePath)
		if err != nil {
			// Log the error but continue with other files
			continue
		}
		results = append(results, fileResults...)
	}

	return results, nil
}

// searchFile searches for the pattern in a single file
func (t *GrepTool) searchFile(pattern *regexp.Regexp, filePath string) ([]GrepResult, error) {
	// Check if file is likely to be text
	if !t.isTextFile(filePath) {
		return nil, nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Split content into lines
	lines := strings.Split(string(content), "\n")
	var results []GrepResult

	for lineNum, line := range lines {
		matches := pattern.FindAllStringIndex(line, -1)
		for _, match := range matches {
			result := GrepResult{
				FilePath: filePath,
				LineNum:  lineNum + 1, // 1-based line numbers
				Line:     line,
				Match:    line[match[0]:match[1]],
				Column:   match[0] + 1, // 1-based column numbers
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// isTextFile performs a simple heuristic to determine if a file is likely to be text
func (t *GrepTool) isTextFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	// Common text file extensions
	textExts := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".py": true, ".js": true,
		".ts": true, ".tsx": true, ".jsx": true, ".html": true, ".css": true,
		".scss": true, ".sass": true, ".json": true, ".xml": true, ".yaml": true,
		".yml": true, ".toml": true, ".ini": true, ".cfg": true, ".conf": true,
		".sh": true, ".bash": true, ".zsh": true, ".fish": true, ".sql": true,
		".java": true, ".c": true, ".cpp": true, ".h": true, ".hpp": true,
		".cs": true, ".php": true, ".rb": true, ".swift": true, ".kt": true,
		".rs": true, ".elm": true, ".hs": true, ".clj": true, ".scala": true,
		".r": true, ".matlab": true, ".m": true, ".pl": true, ".pm": true,
		".dockerfile": true, ".makefile": true, ".cmake": true, ".gradle": true,
		".vue": true, ".svelte": true, ".astro": true, ".proto": true,
	}

	if textExts[ext] {
		return true
	}

	// Check for files without extensions or common config files
	baseName := strings.ToLower(filepath.Base(filePath))
	textFiles := map[string]bool{
		"readme": true, "license": true, "changelog": true, "authors": true,
		"contributors": true, "dockerfile": true, "makefile": true, "rakefile": true,
		"gemfile": true, "pipfile": true, "requirements": true,
	}

	if textFiles[baseName] {
		return true
	}

	// As a final check, read the first few bytes to look for binary markers
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return false
	}

	// Check for common binary markers
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return false // Found null byte, likely binary
		}
	}

	return true
}

// countUniqueFiles counts the number of unique files in the results
func (t *GrepTool) countUniqueFiles(results []GrepResult) int {
	files := make(map[string]bool)
	for _, result := range results {
		files[result.FilePath] = true
	}
	return len(files)
}

// formatOutput formats the grep results for display
func (t *GrepTool) formatOutput(output GrepOutput) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("Grep Results for pattern: %s\n", output.Pattern))
	if output.Include != "" {
		sb.WriteString(fmt.Sprintf("File filter: %s\n", output.Include))
	}
	sb.WriteString(fmt.Sprintf("Search path: %s\n", output.Path))
	sb.WriteString(fmt.Sprintf("Found %d matches in %d files (scanned %d files)\n", 
		output.MatchCount, output.FileCount, output.FilesScanned))
	sb.WriteString(fmt.Sprintf("Search completed in %s\n\n", output.Duration))

	if len(output.Results) == 0 {
		sb.WriteString("No matches found.\n")
		return sb.String()
	}

	// Group results by file
	fileGroups := make(map[string][]GrepResult)
	for _, result := range output.Results {
		fileGroups[result.FilePath] = append(fileGroups[result.FilePath], result)
	}

	// Get sorted file paths
	var filePaths []string
	for filePath := range fileGroups {
		filePaths = append(filePaths, filePath)
	}
	sort.Strings(filePaths)

	// Output results grouped by file
	for _, filePath := range filePaths {
		results := fileGroups[filePath]
		sb.WriteString(fmt.Sprintf("%s:\n", filePath))
		
		for _, result := range results {
			sb.WriteString(fmt.Sprintf("  Line %d:%d: %s\n", 
				result.LineNum, result.Column, strings.TrimSpace(result.Line)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}