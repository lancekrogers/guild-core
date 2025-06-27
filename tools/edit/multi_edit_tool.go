// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package edit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/tools"
)

// MultiEditTool performs atomic multi-edit operations on a single file
type MultiEditTool struct {
	*tools.BaseTool
}

// MultiEditParams represents the input parameters for multi-edit operations
type MultiEditParams struct {
	FilePath string      `json:"file_path"`          // Path to the file to edit
	Edits    []EditEntry `json:"edits"`              // Array of edits to apply
	Backup   bool        `json:"backup,omitempty"`   // Create backup before applying changes
	DryRun   bool        `json:"dry_run,omitempty"`  // Show preview without applying changes
	Validate bool        `json:"validate,omitempty"` // Validate all edits before applying any
}

// EditEntry represents a single find-and-replace operation
type EditEntry struct {
	OldString  string `json:"old_string"`            // String to find and replace
	NewString  string `json:"new_string"`            // String to replace with
	ReplaceAll bool   `json:"replace_all,omitempty"` // Replace all occurrences (default: false)
}

// MultiEditResult represents the result of a multi-edit operation
type MultiEditResult struct {
	FilePath        string          `json:"file_path"`
	Applied         bool            `json:"applied"`
	DryRun          bool            `json:"dry_run"`
	TotalEdits      int             `json:"total_edits"`
	AppliedEdits    int             `json:"applied_edits"`
	EditsDetails    []EditResult    `json:"edits_details"`
	BackupFile      string          `json:"backup_file,omitempty"`
	ValidationError string          `json:"validation_error,omitempty"`
	Errors          []string        `json:"errors,omitempty"`
	Warnings        []string        `json:"warnings,omitempty"`
	Preview         string          `json:"preview,omitempty"`
	Stats           *MultiEditStats `json:"stats"`
}

// EditResult represents the result of a single edit operation
type EditResult struct {
	Index       int    `json:"index"` // Index of the edit in the original array
	OldString   string `json:"old_string"`
	NewString   string `json:"new_string"`
	ReplaceAll  bool   `json:"replace_all"`
	Occurrences int    `json:"occurrences"` // Number of occurrences found/replaced
	Applied     bool   `json:"applied"`
	Error       string `json:"error,omitempty"`
}

// MultiEditStats provides statistics about the multi-edit operation
type MultiEditStats struct {
	TotalOccurrences    int   `json:"total_occurrences"`
	ReplacedOccurrences int   `json:"replaced_occurrences"`
	CharactersAdded     int   `json:"characters_added"`
	CharactersRemoved   int   `json:"characters_removed"`
	ProcessingTimeMs    int64 `json:"processing_time_ms"`
}

// NewMultiEditTool creates a new multi-edit tool
func NewMultiEditTool() *MultiEditTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to edit (must exist)",
			},
			"edits": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"old_string": map[string]interface{}{
							"type":        "string",
							"description": "String to find and replace",
						},
						"new_string": map[string]interface{}{
							"type":        "string",
							"description": "String to replace with",
						},
						"replace_all": map[string]interface{}{
							"type":        "boolean",
							"description": "Replace all occurrences (default: false - replace first occurrence only)",
						},
					},
					"required": []string{"old_string", "new_string"},
				},
				"minItems":    1,
				"description": "Array of find-and-replace operations to apply atomically",
			},
			"backup": map[string]interface{}{
				"type":        "boolean",
				"description": "Create backup file before applying changes",
			},
			"dry_run": map[string]interface{}{
				"type":        "boolean",
				"description": "Show preview of changes without applying them",
			},
			"validate": map[string]interface{}{
				"type":        "boolean",
				"description": "Validate all edits can be applied before making any changes",
			},
		},
		"required": []string{"file_path", "edits"},
	}

	examples := []string{
		`{"file_path": "main.go", "edits": [{"old_string": "oldFunc", "new_string": "newFunc", "replace_all": true}]}`,
		`{"file_path": "config.json", "edits": [{"old_string": "\"debug\": false", "new_string": "\"debug\": true"}, {"old_string": "\"port\": 8080", "new_string": "\"port\": 3000"}], "backup": true}`,
		`{"file_path": "script.py", "edits": [{"old_string": "import old_module", "new_string": "import new_module"}, {"old_string": "old_function()", "new_string": "new_function()"}], "dry_run": true}`,
		`{"file_path": "README.md", "edits": [{"old_string": "# Old Title", "new_string": "# New Title"}, {"old_string": "old description", "new_string": "new description", "replace_all": true}], "validate": true}`,
	}

	baseTool := tools.NewBaseTool(
		"multi_edit",
		"Perform atomic multi-edit operations on a single file. Apply multiple find-and-replace operations in sequence with atomic success/failure semantics.",
		schema,
		"edit",
		false,
		examples,
	)

	return &MultiEditTool{
		BaseTool: baseTool,
	}
}

// Execute runs the multi-edit tool with the given input
func (t *MultiEditTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	startTime := time.Now()

	var params MultiEditParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("multi_edit_tool").
			WithOperation("execute")
	}

	// Validate required parameters
	if params.FilePath == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "file_path is required", nil).
			WithComponent("multi_edit_tool").
			WithOperation("execute")
	}

	if len(params.Edits) == 0 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "at least one edit is required", nil).
			WithComponent("multi_edit_tool").
			WithOperation("execute")
	}

	// Validate each edit
	for i, edit := range params.Edits {
		if edit.OldString == "" {
			return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "edit[%d]: old_string cannot be empty", i).
				WithComponent("multi_edit_tool").
				WithOperation("execute")
		}
		if edit.OldString == edit.NewString {
			return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "edit[%d]: old_string and new_string cannot be identical", i).
				WithComponent("multi_edit_tool").
				WithOperation("execute")
		}
	}

	// Set defaults
	if params.Validate && !params.DryRun {
		params.Validate = true // Always validate when not in dry run mode for safety
	}

	// Check if file exists
	if _, err := os.Stat(params.FilePath); os.IsNotExist(err) {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "file does not exist: %s", params.FilePath).
			WithComponent("multi_edit_tool").
			WithOperation("execute")
	}

	// Perform multi-edit operation
	result, err := t.performMultiEdit(ctx, params, startTime)
	if err != nil {
		return nil, err
	}

	// Format output
	output := t.formatResult(result)

	metadata := map[string]string{
		"file_path":            params.FilePath,
		"applied":              fmt.Sprintf("%t", result.Applied),
		"dry_run":              fmt.Sprintf("%t", result.DryRun),
		"total_edits":          fmt.Sprintf("%d", result.TotalEdits),
		"applied_edits":        fmt.Sprintf("%d", result.AppliedEdits),
		"total_occurrences":    fmt.Sprintf("%d", result.Stats.TotalOccurrences),
		"replaced_occurrences": fmt.Sprintf("%d", result.Stats.ReplacedOccurrences),
	}

	extraData := map[string]interface{}{
		"result": result,
	}

	return tools.NewToolResult(output, metadata, nil, extraData), nil
}

// performMultiEdit performs the actual multi-edit operation
func (t *MultiEditTool) performMultiEdit(ctx context.Context, params MultiEditParams, startTime time.Time) (*MultiEditResult, error) {
	result := &MultiEditResult{
		FilePath:   params.FilePath,
		DryRun:     params.DryRun,
		TotalEdits: len(params.Edits),
		Stats:      &MultiEditStats{},
	}

	// Read the original file
	originalContent, err := os.ReadFile(params.FilePath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to read file").
			WithComponent("multi_edit_tool").
			WithOperation("perform_multi_edit")
	}

	workingContent := string(originalContent)

	// Phase 1: Validation (if requested)
	if params.Validate {
		validationErr := t.validateEdits(params.Edits, workingContent)
		if validationErr != nil {
			result.ValidationError = validationErr.Error()
			result.Stats.ProcessingTimeMs = time.Since(startTime).Milliseconds()
			return result, nil
		}
	}

	// Phase 2: Apply edits in sequence
	for i, edit := range params.Edits {
		editResult := t.applyEdit(edit, &workingContent, i)
		result.EditsDetails = append(result.EditsDetails, editResult)

		if editResult.Applied {
			result.AppliedEdits++
			result.Stats.TotalOccurrences += editResult.Occurrences
			result.Stats.ReplacedOccurrences += editResult.Occurrences

			// Calculate character changes
			charDiff := (len(edit.NewString) - len(edit.OldString)) * editResult.Occurrences
			if charDiff > 0 {
				result.Stats.CharactersAdded += charDiff
			} else {
				result.Stats.CharactersRemoved += -charDiff
			}
		} else if editResult.Error != "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Edit %d failed: %s", i+1, editResult.Error))
		}
	}

	// Phase 3: Create backup if requested and not dry run
	if params.Backup && !params.DryRun && result.AppliedEdits > 0 {
		backupPath := params.FilePath + ".bak." + fmt.Sprintf("%d", time.Now().Unix())
		if err := os.WriteFile(backupPath, originalContent, 0644); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to create backup: %v", err))
		} else {
			result.BackupFile = backupPath
		}
	}

	// Phase 4: Apply changes or generate preview
	if params.DryRun {
		result.Preview = t.generatePreview(string(originalContent), workingContent)
	} else if result.AppliedEdits > 0 {
		// Write the modified content atomically
		err := t.writeFileAtomically(params.FilePath, []byte(workingContent))
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write modified file").
				WithComponent("multi_edit_tool").
				WithOperation("perform_multi_edit")
		}
		result.Applied = true
	}

	result.Stats.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	return result, nil
}

// validateEdits validates that all edits can be applied to the content
func (t *MultiEditTool) validateEdits(edits []EditEntry, content string) error {
	workingContent := content

	for i, edit := range edits {
		if !strings.Contains(workingContent, edit.OldString) {
			return gerror.Newf(gerror.ErrCodeValidation, "edit[%d]: old_string '%s' not found in content", i, edit.OldString).
				WithComponent("multi_edit_tool").
				WithOperation("validate_edits")
		}

		// Simulate the edit to check for conflicts with subsequent edits
		if edit.ReplaceAll {
			workingContent = strings.ReplaceAll(workingContent, edit.OldString, edit.NewString)
		} else {
			workingContent = strings.Replace(workingContent, edit.OldString, edit.NewString, 1)
		}
	}

	return nil
}

// applyEdit applies a single edit to the content
func (t *MultiEditTool) applyEdit(edit EditEntry, content *string, index int) EditResult {
	result := EditResult{
		Index:      index,
		OldString:  edit.OldString,
		NewString:  edit.NewString,
		ReplaceAll: edit.ReplaceAll,
	}

	// Check if the old string exists in the content
	if !strings.Contains(*content, edit.OldString) {
		result.Error = fmt.Sprintf("old_string '%s' not found in content", edit.OldString)
		return result
	}

	// Count occurrences
	if edit.ReplaceAll {
		result.Occurrences = strings.Count(*content, edit.OldString)
		*content = strings.ReplaceAll(*content, edit.OldString, edit.NewString)
	} else {
		if strings.Contains(*content, edit.OldString) {
			result.Occurrences = 1
			*content = strings.Replace(*content, edit.OldString, edit.NewString, 1)
		}
	}

	result.Applied = result.Occurrences > 0
	return result
}

// writeFileAtomically writes content to a file atomically by writing to a temp file first
func (t *MultiEditTool) writeFileAtomically(filePath string, content []byte) error {
	dir := filepath.Dir(filePath)
	tempFile, err := os.CreateTemp(dir, ".multi_edit_tmp_*")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create temporary file").
			WithComponent("multi_edit_tool").
			WithOperation("write_file_atomically")
	}

	tempPath := tempFile.Name()
	defer func() {
		tempFile.Close()
		os.Remove(tempPath) // Clean up temp file if something goes wrong
	}()

	// Write content to temp file
	if _, err := tempFile.Write(content); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write to temporary file").
			WithComponent("multi_edit_tool").
			WithOperation("write_file_atomically")
	}

	// Sync to disk
	if err := tempFile.Sync(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to sync temporary file").
			WithComponent("multi_edit_tool").
			WithOperation("write_file_atomically")
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to close temporary file").
			WithComponent("multi_edit_tool").
			WithOperation("write_file_atomically")
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to rename temporary file").
			WithComponent("multi_edit_tool").
			WithOperation("write_file_atomically")
	}

	return nil
}

// generatePreview generates a preview of the changes
func (t *MultiEditTool) generatePreview(original, modified string) string {
	var preview strings.Builder

	preview.WriteString("Multi-Edit Preview:\n")
	preview.WriteString("==================\n\n")

	if original == modified {
		preview.WriteString("No changes would be made to the file.\n")
		return preview.String()
	}

	// Simple line-by-line comparison
	originalLines := strings.Split(original, "\n")
	modifiedLines := strings.Split(modified, "\n")

	preview.WriteString("Changes that would be applied:\n\n")

	maxLines := len(originalLines)
	if len(modifiedLines) > maxLines {
		maxLines = len(modifiedLines)
	}

	changesFound := false
	for i := 0; i < maxLines; i++ {
		var origLine, modLine string

		if i < len(originalLines) {
			origLine = originalLines[i]
		}
		if i < len(modifiedLines) {
			modLine = modifiedLines[i]
		}

		if origLine != modLine {
			changesFound = true
			preview.WriteString(fmt.Sprintf("Line %d:\n", i+1))
			if origLine != "" {
				preview.WriteString(fmt.Sprintf("- %s\n", origLine))
			}
			if modLine != "" {
				preview.WriteString(fmt.Sprintf("+ %s\n", modLine))
			}
			preview.WriteString("\n")
		}
	}

	if !changesFound {
		preview.WriteString("No line-level changes detected (changes may be within lines).\n")
	}

	return preview.String()
}

// formatResult formats the multi-edit result for output
func (t *MultiEditTool) formatResult(result *MultiEditResult) string {
	var output strings.Builder

	// Header
	if result.DryRun {
		output.WriteString("Multi-Edit Preview\n")
	} else if result.Applied {
		output.WriteString("Multi-Edit Applied Successfully\n")
	} else {
		output.WriteString("Multi-Edit Completed\n")
	}

	output.WriteString("===================\n\n")
	output.WriteString(fmt.Sprintf("Target File: %s\n", result.FilePath))

	// Summary statistics
	output.WriteString(fmt.Sprintf("Total Edits: %d\n", result.TotalEdits))
	output.WriteString(fmt.Sprintf("Applied Edits: %d\n", result.AppliedEdits))
	output.WriteString(fmt.Sprintf("Total Occurrences: %d\n", result.Stats.TotalOccurrences))
	output.WriteString(fmt.Sprintf("Replaced Occurrences: %d\n", result.Stats.ReplacedOccurrences))

	if result.Stats.CharactersAdded > 0 || result.Stats.CharactersRemoved > 0 {
		output.WriteString(fmt.Sprintf("Characters Added: %d\n", result.Stats.CharactersAdded))
		output.WriteString(fmt.Sprintf("Characters Removed: %d\n", result.Stats.CharactersRemoved))
	}

	output.WriteString(fmt.Sprintf("Processing Time: %dms\n", result.Stats.ProcessingTimeMs))

	// Backup info
	if result.BackupFile != "" {
		output.WriteString(fmt.Sprintf("Backup Created: %s\n", result.BackupFile))
	}

	// Validation error
	if result.ValidationError != "" {
		output.WriteString(fmt.Sprintf("\nValidation Failed: %s\n", result.ValidationError))
	}

	// Edit details
	if len(result.EditsDetails) > 0 {
		output.WriteString("\nEdit Details:\n")
		for _, edit := range result.EditsDetails {
			status := "✓"
			if !edit.Applied {
				status = "✗"
			}

			output.WriteString(fmt.Sprintf("  %s Edit %d: '%s' → '%s' (%d occurrences",
				status, edit.Index+1, edit.OldString, edit.NewString, edit.Occurrences))

			if edit.ReplaceAll {
				output.WriteString(", replace_all")
			}
			output.WriteString(")\n")

			if edit.Error != "" {
				output.WriteString(fmt.Sprintf("    Error: %s\n", edit.Error))
			}
		}
	}

	// Warnings
	if len(result.Warnings) > 0 {
		output.WriteString("\nWarnings:\n")
		for _, warning := range result.Warnings {
			output.WriteString(fmt.Sprintf("- %s\n", warning))
		}
	}

	// Errors
	if len(result.Errors) > 0 {
		output.WriteString("\nErrors:\n")
		for _, err := range result.Errors {
			output.WriteString(fmt.Sprintf("- %s\n", err))
		}
	}

	// Preview
	if result.DryRun && result.Preview != "" {
		output.WriteString("\n")
		output.WriteString(result.Preview)
	}

	return output.String()
}
