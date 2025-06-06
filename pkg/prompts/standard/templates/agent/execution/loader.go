package execution

import (
	"embed"
	"fmt"
	"path/filepath"
)

//go:embed *.md
var promptFiles embed.FS

// Loader loads prompt templates from embedded files
type Loader struct {
	fs embed.FS
}

// NewLoader creates a new prompt loader
func NewLoader() *Loader {
	return &Loader{
		fs: promptFiles,
	}
}

// LoadPrompt loads a prompt template by name
func (l *Loader) LoadPrompt(name string) (string, error) {
	// If name includes path, use it directly
	// Otherwise, assume it's just the filename
	filename := name
	if filepath.Ext(name) == "" {
		filename = name + ".md"
	}

	content, err := l.fs.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file %s: %w", filename, err)
	}

	return string(content), nil
}