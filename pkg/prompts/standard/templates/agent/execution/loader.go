// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package execution

import (
	"embed"
	"path/filepath"

	"github.com/guild-framework/guild-core/pkg/gerror"
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
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to read prompt file").
			WithComponent("prompts").
			WithOperation("LoadPrompt").
			WithDetails("filename", filename)
	}

	return string(content), nil
}
