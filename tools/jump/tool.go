// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package jump

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/tools"
)

// JumpTool provides directory jumping functionality for agents
type JumpTool struct {
	*tools.BaseTool
	jump *Jump
}

// JumpToolInput represents the input for jump operations
type JumpToolInput struct {
	Query  string `json:"query,omitempty"`  // Directory query for jumping or tracking
	Track  bool   `json:"track,omitempty"`  // If true, track the directory specified in query
	Recent int    `json:"recent,omitempty"` // Number of recent directories to return
}

// NewJumpTool creates a new jump tool
func NewJumpTool() (*JumpTool, error) {
	// Don't create the jump instance here - create it on demand
	// This avoids keeping a database connection open unnecessarily

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Directory query for jumping or path to track",
			},
			"track": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, track the directory specified in query",
			},
			"recent": map[string]interface{}{
				"type":        "integer",
				"description": "Number of recent directories to return (if specified, query and track are ignored)",
			},
		},
		"oneOf": []map[string]interface{}{
			{
				"required": []string{"query"},
			},
			{
				"required": []string{"recent"},
			},
		},
	}

	examples := []string{
		`{"query": "docs"}`,
		`{"query": "guild-framework"}`,
		`{"query": "/abs/path/to/project", "track": true}`,
		`{"recent": 5}`,
		`{"query": ".", "track": true}`,
	}

	baseTool := tools.NewBaseTool(
		"jump",
		"Fuzzy, frecency-based directory jumping. Find directories by partial name, track visits, or list recent directories.",
		schema,
		"navigation",
		false,
		examples,
	)

	return &JumpTool{
		BaseTool: baseTool,
		jump:     nil, // Will be created on demand
	}, nil
}

// Execute runs the jump tool with the given input
func (t *JumpTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params JumpToolInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("jump_tool").
			WithOperation("execute")
	}

	// Create jump instance on demand
	j, err := GetDefaultJump()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get jump instance").
			WithComponent("jump_tool").
			WithOperation("execute")
	}
	defer j.Close()

	// Handle recent directories request
	if params.Recent > 0 {
		dirs, err := j.Recent(params.Recent)
		if err != nil {
			return tools.NewToolResult("", nil, err, nil), err
		}

		// Return as JSON array
		output, err := json.Marshal(dirs)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
				WithComponent("jump_tool").
				WithOperation("execute")
		}

		metadata := map[string]string{
			"operation": "recent",
			"count":     fmt.Sprintf("%d", len(dirs)),
		}

		return tools.NewToolResult(string(output), metadata, nil, nil), nil
	}

	// Require query for other operations
	if params.Query == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "query is required when recent is not specified", nil).
			WithComponent("jump_tool").
			WithOperation("execute")
	}

	// Handle tracking
	if params.Track {
		// Resolve the path
		path := params.Query
		if path == "." || path == "" {
			// Use current working directory
			cwd, err := os.Getwd()
			if err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get current directory").
					WithComponent("jump_tool").
					WithOperation("execute")
			}
			path = cwd
		}

		if err := j.Track(path); err != nil {
			return tools.NewToolResult("", nil, err, nil), err
		}

		metadata := map[string]string{
			"operation": "track",
			"path":      path,
		}

		return tools.NewToolResult("ok", metadata, nil, nil), nil
	}

	// Handle directory jumping
	result, err := j.Find(params.Query)
	if err != nil {
		return tools.NewToolResult("", nil, err, nil), err
	}

	// Return the absolute path as a quoted string
	output := fmt.Sprintf("%q", result)

	metadata := map[string]string{
		"operation": "find",
		"query":     params.Query,
		"result":    result,
	}

	return tools.NewToolResult(output, metadata, nil, nil), nil
}

// Close is a no-op since we create jump instances on demand
func (t *JumpTool) Close() error {
	return nil
}

// init registers the jump tool
func init() {
	// This will be called when the tool is registered with the registry
}

// RegisterJumpTool registers the jump tool with the given registry
func RegisterJumpTool(registry interface{ RegisterTool(tool tools.Tool) error }) error {
	jumpTool, err := NewJumpTool()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create jump tool").
			WithComponent("jump_tool").
			WithOperation("register")
	}

	if err := registry.RegisterTool(jumpTool); err != nil {
		jumpTool.Close() // Clean up on failure
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register jump tool").
			WithComponent("jump_tool").
			WithOperation("register")
	}

	return nil
}
