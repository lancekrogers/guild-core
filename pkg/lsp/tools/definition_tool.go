package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/lsp"
	"github.com/guild-ventures/guild-core/tools"
)

// DefinitionTool provides "go to definition" functionality using LSP
type DefinitionTool struct {
	tools.BaseTool
	manager *lsp.Manager
}

// LocationParams represents the parameters for location-based tools
type LocationParams struct {
	File   string `json:"file" description:"The file path"`
	Line   int    `json:"line" description:"The line number (0-based)"`
	Column int    `json:"column" description:"The column number (0-based)"`
}

// LocationResult represents a location in a file
type LocationResult struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"end_line,omitempty"`
	EndColumn int    `json:"end_column,omitempty"`
	Preview   string `json:"preview,omitempty"`
}

// DefinitionResult represents the result of a definition request
type DefinitionResult struct {
	Locations []LocationResult `json:"locations"`
}

// NewDefinitionTool creates a new definition tool
func NewDefinitionTool(manager *lsp.Manager) *DefinitionTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file": map[string]interface{}{
				"type":        "string",
				"description": "The file path",
			},
			"line": map[string]interface{}{
				"type":        "integer",
				"description": "The line number (0-based)",
			},
			"column": map[string]interface{}{
				"type":        "integer",
				"description": "The column number (0-based)",
			},
		},
		"required": []string{"file", "line", "column"},
	}

	examples := []string{
		`{"file": "/path/to/main.go", "line": 10, "column": 15}`,
		`{"file": "/path/to/app.ts", "line": 25, "column": 8}`,
	}

	return &DefinitionTool{
		BaseTool: *tools.NewBaseTool(
			"lsp_definition",
			"Go to definition for a symbol at a specific position using Language Server Protocol",
			schema,
			"code",
			false,
			examples,
		),
		manager: manager,
	}
}

// Execute runs the definition tool
func (t *DefinitionTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	// Parse input parameters
	var params LocationParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid input parameters").
			WithComponent("lsp.definition_tool").
			WithOperation("execute")
	}

	// Get definition locations from LSP
	locations, err := t.manager.GetDefinition(ctx, params.File, params.Line, params.Column)
	if err != nil {
		return nil, err
	}

	// Convert to our format
	result := DefinitionResult{
		Locations: make([]LocationResult, 0, len(locations)),
	}

	for _, loc := range locations {
		// Convert URI to file path
		filePath := strings.TrimPrefix(loc.URI, "file://")
		
		result.Locations = append(result.Locations, LocationResult{
			File:      filePath,
			Line:      loc.Range.Start.Line,
			Column:    loc.Range.Start.Character,
			EndLine:   loc.Range.End.Line,
			EndColumn: loc.Range.End.Character,
		})
	}

	// Convert result to JSON
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
			WithComponent("lsp.definition_tool").
			WithOperation("execute")
	}

	metadata := map[string]string{
		"file":           params.File,
		"position":       fmt.Sprintf("%d:%d", params.Line, params.Column),
		"location_count": fmt.Sprintf("%d", len(result.Locations)),
	}

	return tools.NewToolResult(string(output), metadata, nil, nil), nil
}

// FormatDefinitionsAsText formats definition locations as human-readable text
func FormatDefinitionsAsText(result *DefinitionResult) string {
	if len(result.Locations) == 0 {
		return "No definitions found"
	}

	var builder strings.Builder
	
	if len(result.Locations) == 1 {
		loc := result.Locations[0]
		builder.WriteString(fmt.Sprintf("Definition found at:\n%s:%d:%d", 
			loc.File, loc.Line+1, loc.Column+1))
	} else {
		builder.WriteString(fmt.Sprintf("Found %d definitions:\n\n", len(result.Locations)))
		for i, loc := range result.Locations {
			builder.WriteString(fmt.Sprintf("%d. %s:%d:%d\n", 
				i+1, loc.File, loc.Line+1, loc.Column+1))
		}
	}

	return builder.String()
}