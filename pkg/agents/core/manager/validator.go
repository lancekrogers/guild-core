// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"path/filepath"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// DefaultValidator implements the StructureValidator interface
type DefaultValidator struct {
	maxFiles          int
	maxFileSize       int
	allowedExtensions []string
}

// NewDefaultValidator creates a new structure validator
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{
		maxFiles:          50,    // Maximum number of files
		maxFileSize:       50000, // Maximum file size in characters
		allowedExtensions: []string{".md", ".txt"},
	}
}

// ValidateStructure implements the StructureValidator interface
func (v *DefaultValidator) ValidateStructure(structure *FileStructure) error {
	if structure == nil {
		return gerror.New(gerror.ErrCodeValidation, "structure cannot be nil", nil).
			WithComponent("manager").
			WithOperation("ValidateStructure")
	}

	if err := v.validateFileCount(structure); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "file count validation failed").
			WithComponent("DefaultValidator").
			WithOperation("validateStructure")
	}

	if err := v.validateFiles(structure.Files); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "file validation failed").
			WithComponent("DefaultValidator").
			WithOperation("validateStructure")
	}

	if err := v.validateRequiredFiles(structure.Files); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "required files validation failed").
			WithComponent("DefaultValidator").
			WithOperation("validateStructure")
	}

	return nil
}

// validateFileCount checks if the number of files is reasonable
func (v *DefaultValidator) validateFileCount(structure *FileStructure) error {
	if len(structure.Files) == 0 {
		return gerror.New(gerror.ErrCodeValidation, "structure must contain at least one file", nil).
			WithComponent("manager").
			WithOperation("validateFileCount")
	}

	if len(structure.Files) > v.maxFiles {
		return gerror.Newf(gerror.ErrCodeValidation, "too many files: %d (max: %d)", len(structure.Files), v.maxFiles).
			WithComponent("manager").
			WithOperation("validateFileCount").
			WithDetails("file_count", len(structure.Files)).
			WithDetails("max_files", v.maxFiles)
	}

	return nil
}

// validateFiles validates individual files
func (v *DefaultValidator) validateFiles(files []*FileEntry) error {
	seenPaths := make(map[string]bool)

	for i, file := range files {
		if err := v.validateFile(file, i); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeValidation, "individual file validation failed").
				WithComponent("DefaultValidator").
				WithOperation("validateFiles").
				WithDetails("file_index", i)
		}

		// Check for duplicate paths
		if seenPaths[file.Path] {
			return gerror.Newf(gerror.ErrCodeValidation, "duplicate file path: %s", file.Path).
				WithComponent("manager").
				WithOperation("validateFiles").
				WithDetails("duplicate_path", file.Path)
		}
		seenPaths[file.Path] = true
	}

	return nil
}

// validateFile validates a single file
func (v *DefaultValidator) validateFile(file *FileEntry, index int) error {
	if file == nil {
		return gerror.Newf(gerror.ErrCodeValidation, "file at index %d is nil", index).
			WithComponent("manager").
			WithOperation("validateFile").
			WithDetails("file_index", index)
	}

	// Validate path
	if err := v.validatePath(file.Path); err != nil {
		return gerror.Wrapf(err, gerror.ErrCodeValidation, "invalid path for file %d", index).
			WithComponent("manager").
			WithOperation("validateFile").
			WithDetails("file_index", index).
			WithDetails("file_path", file.Path)
	}

	// Validate content
	if err := v.validateContent(file.Content, file.Path); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid content for file").
			WithComponent("manager").
			WithOperation("validateFile").
			WithDetails("file_path", file.Path)
	}

	// Validate type
	if err := v.validateFileType(file.Type); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid type for file").
			WithComponent("manager").
			WithOperation("validateFile").
			WithDetails("file_path", file.Path)
	}

	return nil
}

// validatePath validates a file path
func (v *DefaultValidator) validatePath(path string) error {
	if path == "" {
		return gerror.New(gerror.ErrCodeValidation, "path cannot be empty", nil).
			WithComponent("manager").
			WithOperation("validatePath")
	}

	// Clean the path
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return gerror.New(gerror.ErrCodeValidation, "path should be clean", nil).
			WithComponent("manager").
			WithOperation("validatePath").
			WithDetails("original_path", path).
			WithDetails("clean_path", cleanPath)
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return gerror.New(gerror.ErrCodeValidation, "path contains parent directory references", nil).
			WithComponent("manager").
			WithOperation("validatePath").
			WithDetails("path", path)
	}

	// Check file extension
	ext := filepath.Ext(path)
	if ext != "" {
		validExt := false
		for _, allowedExt := range v.allowedExtensions {
			if ext == allowedExt {
				validExt = true
				break
			}
		}
		if !validExt {
			return gerror.Newf(gerror.ErrCodeValidation, "invalid file extension: %s (allowed: %v)", ext, v.allowedExtensions).
				WithComponent("manager").
				WithOperation("validatePath").
				WithDetails("extension", ext).
				WithDetails("allowed_extensions", v.allowedExtensions)
		}
	}

	return nil
}

// validateContent validates file content
func (v *DefaultValidator) validateContent(content, path string) error {
	if content == "" {
		return gerror.New(gerror.ErrCodeValidation, "content cannot be empty", nil).
			WithComponent("manager").
			WithOperation("validateContent")
	}

	if len(content) > v.maxFileSize {
		return gerror.New(gerror.ErrCodeValidation, "content too large", nil).
			WithComponent("manager").
			WithOperation("validateContent").
			WithDetails("content_size", len(content)).
			WithDetails("max_size", v.maxFileSize)
	}

	// Basic markdown validation for .md files
	if strings.HasSuffix(path, ".md") {
		if err := v.validateMarkdownContent(content); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeValidation, "markdown content validation failed").
				WithComponent("DefaultValidator").
				WithOperation("validateContent").
				WithDetails("file_path", path)
		}
	}

	return nil
}

// validateMarkdownContent performs basic markdown validation
func (v *DefaultValidator) validateMarkdownContent(content string) error {
	lines := strings.Split(content, "\n")

	hasTitle := false
	for _, line := range lines {
		// Check for at least one title
		if strings.HasPrefix(line, "# ") {
			hasTitle = true
			break
		}
	}

	if !hasTitle {
		return gerror.New(gerror.ErrCodeValidation, "markdown file must contain at least one title (# heading)", nil).
			WithComponent("manager").
			WithOperation("validateMarkdownContent")
	}

	return nil
}

// validateFileType validates the file type
func (v *DefaultValidator) validateFileType(fileType FileType) error {
	switch fileType {
	case FileTypeMarkdown, FileTypeManifest:
		return nil
	default:
		return gerror.Newf(gerror.ErrCodeValidation, "unsupported file type: %s", fileType).
			WithComponent("manager").
			WithOperation("validateFileType").
			WithDetails("file_type", string(fileType))
	}
}

// validateRequiredFiles checks for required files
func (v *DefaultValidator) validateRequiredFiles(files []*FileEntry) error {
	hasReadme := false
	totalTasks := 0

	for _, file := range files {
		// Check for README
		if file.Path == "README.md" || strings.HasSuffix(file.Path, "/README.md") {
			hasReadme = true
		}

		// Count total tasks
		totalTasks += file.TasksCount
	}

	if !hasReadme {
		return gerror.New(gerror.ErrCodeValidation, "structure must contain a README.md file", nil).
			WithComponent("manager").
			WithOperation("validateRequiredFiles")
	}

	if totalTasks == 0 {
		return gerror.New(gerror.ErrCodeValidation, "structure must contain at least one task", nil).
			WithComponent("manager").
			WithOperation("validateRequiredFiles").
			WithDetails("total_tasks", totalTasks)
	}

	return nil
}

// ValidationConfig allows customizing validation parameters
type ValidationConfig struct {
	MaxFiles          int
	MaxFileSize       int
	AllowedExtensions []string
	RequireReadme     bool
	RequireTasks      bool
}

// ConfigurableValidator allows custom validation rules
type ConfigurableValidator struct {
	config ValidationConfig
}

// NewConfigurableValidator creates a validator with custom config
func NewConfigurableValidator(config ValidationConfig) *ConfigurableValidator {
	// Set defaults
	if config.MaxFiles == 0 {
		config.MaxFiles = 50
	}
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 50000
	}
	if len(config.AllowedExtensions) == 0 {
		config.AllowedExtensions = []string{".md", ".txt"}
	}

	return &ConfigurableValidator{
		config: config,
	}
}

// ValidateStructure implements validation with custom rules
func (v *ConfigurableValidator) ValidateStructure(structure *FileStructure) error {
	// Use similar logic to DefaultValidator but with custom config
	validator := &DefaultValidator{
		maxFiles:          v.config.MaxFiles,
		maxFileSize:       v.config.MaxFileSize,
		allowedExtensions: v.config.AllowedExtensions,
	}

	if err := validator.ValidateStructure(structure); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "structure validation failed with custom config").
			WithComponent("ConfigurableValidator").
			WithOperation("ValidateStructure")
	}

	// Additional custom validations
	if v.config.RequireReadme || v.config.RequireTasks {
		return v.validateCustomRequirements(structure)
	}

	return nil
}

// validateCustomRequirements validates custom requirements
func (v *ConfigurableValidator) validateCustomRequirements(structure *FileStructure) error {
	hasReadme := false
	totalTasks := 0

	for _, file := range structure.Files {
		if file.Path == "README.md" {
			hasReadme = true
		}
		totalTasks += file.TasksCount
	}

	if v.config.RequireReadme && !hasReadme {
		return gerror.New(gerror.ErrCodeValidation, "README.md is required", nil).
			WithComponent("manager").
			WithOperation("validateCustomRequirements")
	}

	if v.config.RequireTasks && totalTasks == 0 {
		return gerror.New(gerror.ErrCodeValidation, "at least one task is required", nil).
			WithComponent("manager").
			WithOperation("validateCustomRequirements").
			WithDetails("total_tasks", totalTasks)
	}

	return nil
}
