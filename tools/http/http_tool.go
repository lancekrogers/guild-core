// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools"
)

// HTTPTool provides HTTP request capabilities for agents
type HTTPTool struct {
	*tools.BaseTool
	client *http.Client
}

// HTTPToolInput represents the input for HTTP requests
type HTTPToolInput struct {
	Method  string            `json:"method"`            // GET, POST, PUT, DELETE, etc.
	URL     string            `json:"url"`               // Request URL
	Headers map[string]string `json:"headers,omitempty"` // Request headers
	Body    string            `json:"body,omitempty"`    // Request body
	Timeout int               `json:"timeout,omitempty"` // Timeout in seconds
}

// NewHTTPTool creates a new HTTP tool
func NewHTTPTool() *HTTPTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"method": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"},
				"description": "HTTP method to use",
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to send the request to",
			},
			"headers": map[string]interface{}{
				"type":        "object",
				"description": "HTTP headers to include in the request",
			},
			"body": map[string]interface{}{
				"type":        "string",
				"description": "Request body for POST, PUT, or PATCH requests",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Request timeout in seconds",
				"default":     30,
			},
		},
		"required": []string{"method", "url"},
	}

	examples := []string{
		`{"method": "GET", "url": "https://example.com"}`,
		`{"method": "POST", "url": "https://example.com/api", "headers": {"Content-Type": "application/json"}, "body": "{\"key\": \"value\"}"}`,
		`{"method": "GET", "url": "https://example.com", "timeout": 10}`,
	}

	baseTool := tools.NewBaseTool(
		"http",
		"Send HTTP requests to external services and APIs",
		schema,
		"web",
		false,
		examples,
	)

	return &HTTPTool{
		BaseTool: baseTool,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Execute runs the HTTP tool with the given input
func (t *HTTPTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params HTTPToolInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "http_tool").WithComponent("execute").WithOperation("invalid input")
	}

	// Validate method
	params.Method = strings.ToUpper(params.Method)
	switch params.Method {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD":
		// Valid methods
	default:
		return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "invalid HTTP method: %s", params.Method).
			WithComponent("http_tool").
			WithOperation("execute")
	}

	// Validate URL
	if params.URL == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "URL is required", nil).
			WithComponent("http_tool").
			WithOperation("execute")
	}

	// Set timeout if specified
	client := t.client
	if params.Timeout > 0 {
		client = &http.Client{Timeout: time.Duration(params.Timeout) * time.Second}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, params.Method, params.URL, strings.NewReader(params.Body))
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "http_tool").WithComponent("execute").WithOperation("failed to create request")
	}

	// Add headers
	for key, value := range params.Headers {
		req.Header.Add(key, value)
	}

	// Set default Content-Type for POST/PUT/PATCH if not provided
	if (params.Method == "POST" || params.Method == "PUT" || params.Method == "PATCH") &&
		params.Body != "" &&
		req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "http_tool").WithComponent("execute").WithOperation("request failed")
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "http_tool").WithComponent("execute").WithOperation("failed to read response")
	}

	// Prepare metadata
	metadata := map[string]string{
		"method":       params.Method,
		"url":          params.URL,
		"status_code":  fmt.Sprintf("%d", resp.StatusCode),
		"status_text":  resp.Status,
		"content_type": resp.Header.Get("Content-Type"),
	}

	// Prepare extra data
	extraData := map[string]interface{}{
		"headers": resp.Header,
	}

	// Format response
	var responseStr string
	if contentType := resp.Header.Get("Content-Type"); strings.Contains(contentType, "application/json") {
		// Pretty-print JSON
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
			if err == nil {
				responseStr = string(prettyJSON)
			} else {
				responseStr = string(body)
			}
		} else {
			responseStr = string(body)
		}
	} else {
		responseStr = string(body)
	}

	return tools.NewToolResult(responseStr, metadata, nil, extraData), nil
}
