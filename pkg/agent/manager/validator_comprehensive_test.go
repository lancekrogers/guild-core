package manager

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test NewDefaultValidator
func TestNewDefaultValidator(t *testing.T) {
	validator := NewDefaultValidator()
	
	assert.NotNil(t, validator)
	assert.Equal(t, 50, validator.maxFiles)
	assert.Equal(t, 50000, validator.maxFileSize)
	assert.Equal(t, []string{".md", ".txt"}, validator.allowedExtensions)
}

// Test ValidateStructure with various scenarios
func TestDefaultValidator_ValidateStructure(t *testing.T) {
	tests := []struct {
		name          string
		structure     *FileStructure
		wantErr       bool
		errContains   string
	}{
		{
			name:          "nil structure",
			structure:     nil,
			wantErr:       true,
			errContains:   "structure cannot be nil",
		},
		{
			name: "valid simple structure",
			structure: &FileStructure{
				Files: []*FileEntry{
					{
						Path:       "README.md",
						Type:       FileTypeMarkdown,
						Content:    "# Project README\n\nThis is a test project.",
						TasksCount: 1,
						Metadata:   map[string]interface{}{"size": 100},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "too many files",
			structure: &FileStructure{
				Files: make([]*FileEntry, 51),
			},
			wantErr:     true,
			errContains: "file count",
		},
		{
			name: "file too large",
			structure: &FileStructure{
				Files: []*FileEntry{
					{
						Path:     "large.md",
						Type:     FileTypeMarkdown,
						Content:  strings.Repeat("a", 50001),
						Metadata: map[string]interface{}{"size": 50001},
					},
				},
			},
			wantErr:     true,
			errContains: "content too large",
		},
		{
			name: "invalid file extension",
			structure: &FileStructure{
				Files: []*FileEntry{
					{
						Path:     "script.py",
						Type:     FileTypeMarkdown,
						Content:  "print('hello')",
						Metadata: map[string]interface{}{"size": 100},
					},
				},
			},
			wantErr:     true,
			errContains: "invalid file extension",
		},
		{
			name: "invalid path traversal",
			structure: &FileStructure{
				Files: []*FileEntry{
					{
						Path:     "../etc/passwd",
						Type:     FileTypeMarkdown,
						Content:  "content",
						Metadata: map[string]interface{}{"size": 100},
					},
				},
			},
			wantErr:     true,
			errContains: "invalid path",
		},
		{
			name: "valid directory structure",
			structure: &FileStructure{
				Files: []*FileEntry{
					{
						Path:       "README.md",
						Type:       FileTypeMarkdown,
						Content:    "# README\n\nMain readme",
						TasksCount: 1,
						Metadata:   map[string]interface{}{"size": 20},
					},
					{
						Path:       "docs/guide.md",
						Type:       FileTypeMarkdown,
						Content:    "# Guide",
						TasksCount: 1,
						Metadata:   map[string]interface{}{"size": 10},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing README",
			structure: &FileStructure{
				Files: []*FileEntry{
					{
						Path:       "test.md",
						Type:       FileTypeMarkdown,
						Content:    "# Test",
						TasksCount: 1,
						Metadata:   map[string]interface{}{"size": 6},
					},
				},
			},
			wantErr:     true,
			errContains: "must contain a README.md file",
		},
		{
			name: "empty content file",
			structure: &FileStructure{
				Files: []*FileEntry{
					{
						Path:     "empty.md",
						Type:     FileTypeMarkdown,
						Content:  "",
						Metadata: map[string]interface{}{"size": 0},
					},
					{
						Path:       "README.md",
						Type:       FileTypeMarkdown, 
						Content:    "# README",
						TasksCount: 1,
						Metadata:   map[string]interface{}{"size": 8},
					},
				},
			},
			wantErr:     true,
			errContains: "content cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewDefaultValidator()
			err := validator.ValidateStructure(tt.structure)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test validateFileCount
func TestDefaultValidator_validateFileCount(t *testing.T) {
	validator := NewDefaultValidator()
	
	tests := []struct {
		name      string
		fileCount int
		wantErr   bool
	}{
		{
			name:      "valid count",
			fileCount: 10,
			wantErr:   false,
		},
		{
			name:      "max count",
			fileCount: 50,
			wantErr:   false,
		},
		{
			name:      "exceeds max",
			fileCount: 51,
			wantErr:   true,
		},
		{
			name:      "zero files",
			fileCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structure := &FileStructure{
				Files: make([]*FileEntry, tt.fileCount),
			}
			err := validator.validateFileCount(structure)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test validateMarkdownContent
func TestDefaultValidator_validateMarkdownContent(t *testing.T) {
	validator := NewDefaultValidator()
	
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid markdown",
			content: "# Title\n\nThis is valid markdown with **bold** and *italic*.",
			wantErr: false,
		},
		{
			name:        "missing title",
			content:     "This is markdown without a title heading",
			wantErr:     true,
			errContains: "must contain at least one title",
		},
		{
			name:    "markdown with code block",
			content: "# Title\n\n```python\nprint('hello')\n```",
			wantErr: false,
		},
		{
			name:    "title with content",
			content: "# Main Title\n\nSome content here\n\n## Subtitle\n\nMore content",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateMarkdownContent(tt.content)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test NewConfigurableValidator
func TestNewConfigurableValidator(t *testing.T) {
	config := ValidationConfig{
		MaxFiles:          100,
		MaxFileSize:       100000,
		AllowedExtensions: []string{".md", ".txt", ".go"},
		RequireReadme:     true,
		RequireTasks:      true,
	}
	
	validator := NewConfigurableValidator(config)
	
	assert.NotNil(t, validator)
	assert.Equal(t, 100, validator.config.MaxFiles)
	assert.Equal(t, 100000, validator.config.MaxFileSize)
	assert.Equal(t, []string{".md", ".txt", ".go"}, validator.config.AllowedExtensions)
	assert.True(t, validator.config.RequireReadme)
	assert.True(t, validator.config.RequireTasks)
}

// Test ConfigurableValidator ValidateStructure
func TestConfigurableValidator_ValidateStructure(t *testing.T) {
	config := ValidationConfig{
		MaxFiles:          10,
		MaxFileSize:       1000,
		AllowedExtensions: []string{".md", ".go"},
		RequireReadme:     true,
		RequireTasks:      false,
	}
	
	validator := NewConfigurableValidator(config)
	
	tests := []struct {
		name        string
		structure   *FileStructure
		wantErr     bool
		errContains string
	}{
		{
			name: "valid structure",
			structure: &FileStructure{
				Files: []*FileEntry{
					{
						Path:       "README.md",
						Type:       FileTypeMarkdown,
						Content:    "# Project",
						TasksCount: 1,
						Metadata:   map[string]interface{}{"size": 10},
					},
					{
						Path:       "main.go",
						Type:       FileTypeMarkdown,
						Content:    "package main\n\nfunc main() {}",
						TasksCount: 0,
						Metadata:   map[string]interface{}{"size": 30},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing README when required",
			structure: &FileStructure{
				Files: []*FileEntry{
					{
						Path:       "main.go",
						Type:       FileTypeMarkdown,
						Content:    "# Main\n\npackage main",
						TasksCount: 1,
						Metadata:   map[string]interface{}{"size": 20},
					},
				},
			},
			wantErr:     true,
			errContains: "structure must contain a README.md file",
		},
		{
			name: "invalid extension",
			structure: &FileStructure{
				Files: []*FileEntry{
					{
						Path:     "README.md",
						Type:     FileTypeMarkdown,
						Content:  "# Project",
						Metadata: map[string]interface{}{"size": 10},
					},
					{
						Path:     "script.py",
						Type:     FileTypeMarkdown,
						Content:  "print('hello')",
						Metadata: map[string]interface{}{"size": 14},
					},
				},
			},
			wantErr:     true,
			errContains: "invalid file extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateStructure(tt.structure)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}