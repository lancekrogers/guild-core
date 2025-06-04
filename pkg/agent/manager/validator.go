package manager

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DefaultValidator implements the StructureValidator interface
type DefaultValidator struct {
	maxFiles       int
	maxFileSize    int
	allowedExtensions []string
}

// NewDefaultValidator creates a new structure validator
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{
		maxFiles:    50,  // Maximum number of files
		maxFileSize: 50000, // Maximum file size in characters
		allowedExtensions: []string{".md", ".txt"},
	}
}

// ValidateStructure implements the StructureValidator interface
func (v *DefaultValidator) ValidateStructure(structure *FileStructure) error {
	if structure == nil {
		return fmt.Errorf("structure cannot be nil")
	}

	if err := v.validateFileCount(structure); err != nil {
		return err
	}

	if err := v.validateFiles(structure.Files); err != nil {
		return err
	}

	if err := v.validateRequiredFiles(structure.Files); err != nil {
		return err
	}

	return nil
}

// validateFileCount checks if the number of files is reasonable
func (v *DefaultValidator) validateFileCount(structure *FileStructure) error {
	if len(structure.Files) == 0 {
		return fmt.Errorf("structure must contain at least one file")
	}

	if len(structure.Files) > v.maxFiles {
		return fmt.Errorf("too many files: %d (max: %d)", len(structure.Files), v.maxFiles)
	}

	return nil
}

// validateFiles validates individual files
func (v *DefaultValidator) validateFiles(files []*FileEntry) error {
	seenPaths := make(map[string]bool)

	for i, file := range files {
		if err := v.validateFile(file, i); err != nil {
			return err
		}

		// Check for duplicate paths
		if seenPaths[file.Path] {
			return fmt.Errorf("duplicate file path: %s", file.Path)
		}
		seenPaths[file.Path] = true
	}

	return nil
}

// validateFile validates a single file
func (v *DefaultValidator) validateFile(file *FileEntry, index int) error {
	if file == nil {
		return fmt.Errorf("file at index %d is nil", index)
	}

	// Validate path
	if err := v.validatePath(file.Path); err != nil {
		return fmt.Errorf("invalid path for file %d: %w", index, err)
	}

	// Validate content
	if err := v.validateContent(file.Content, file.Path); err != nil {
		return fmt.Errorf("invalid content for file %s: %w", file.Path, err)
	}

	// Validate type
	if err := v.validateFileType(file.Type); err != nil {
		return fmt.Errorf("invalid type for file %s: %w", file.Path, err)
	}

	return nil
}

// validatePath validates a file path
func (v *DefaultValidator) validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return fmt.Errorf("path should be clean: %s -> %s", path, cleanPath)
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains parent directory references: %s", path)
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
			return fmt.Errorf("invalid file extension: %s (allowed: %v)", ext, v.allowedExtensions)
		}
	}

	return nil
}

// validateContent validates file content
func (v *DefaultValidator) validateContent(content, path string) error {
	if content == "" {
		return fmt.Errorf("content cannot be empty")
	}

	if len(content) > v.maxFileSize {
		return fmt.Errorf("content too large: %d characters (max: %d)", len(content), v.maxFileSize)
	}

	// Basic markdown validation for .md files
	if strings.HasSuffix(path, ".md") {
		if err := v.validateMarkdownContent(content); err != nil {
			return err
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
		return fmt.Errorf("markdown file must contain at least one title (# heading)")
	}

	return nil
}

// validateFileType validates the file type
func (v *DefaultValidator) validateFileType(fileType FileType) error {
	switch fileType {
	case FileTypeMarkdown, FileTypeManifest:
		return nil
	default:
		return fmt.Errorf("unsupported file type: %s", fileType)
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
		return fmt.Errorf("structure must contain a README.md file")
	}

	if totalTasks == 0 {
		return fmt.Errorf("structure must contain at least one task")
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
		maxFiles:       v.config.MaxFiles,
		maxFileSize:    v.config.MaxFileSize,
		allowedExtensions: v.config.AllowedExtensions,
	}

	if err := validator.ValidateStructure(structure); err != nil {
		return err
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
		return fmt.Errorf("README.md is required")
	}

	if v.config.RequireTasks && totalTasks == 0 {
		return fmt.Errorf("at least one task is required")
	}

	return nil
}