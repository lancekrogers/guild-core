// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// pkg/objective/context.go
package commission

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ContextHandler manages document references
type ContextHandler struct {
	session *PlanningSession
	baseDir string
}

// newContextHandler creates a new context handler (private constructor)
func newContextHandler(session *PlanningSession, baseDir string) *ContextHandler {
	return &ContextHandler{
		session: session,
		baseDir: baseDir,
	}
}

// DefaultContextHandlerFactory creates a context handler factory for registry use
func DefaultContextHandlerFactory(session *PlanningSession, baseDir string) *ContextHandler {
	return newContextHandler(session, baseDir)
}

// ProcessInput handles user input and extracts document references
func (ch *ContextHandler) ProcessInput(input string) (string, map[string]string, error) {
	// Regex to match @spec/path or @ai_docs/path references
	re := regexp.MustCompile(`@(spec|ai_docs)/([^\s]+)`)
	matches := re.FindAllStringSubmatch(input, -1)

	// Document map to track loaded documents
	loadedDocs := make(map[string]string)

	// Process each match
	for _, match := range matches {
		docType := match[1] // "spec" or "ai_docs"
		path := match[2]    // path component

		// Full document path
		fullPath := filepath.Join(ch.baseDir, docType, path)

		// Check if file exists
		if _, err := os.Stat(fullPath); err == nil {
			// Load document content
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return input, loadedDocs, err
			}

			// Add to loaded documents
			docRef := "@" + docType + "/" + path
			loadedDocs[docRef] = string(content)

			// Store in session
			ch.session.Documents[docRef] = string(content)
		}
	}

	return input, loadedDocs, nil
}

// GetDocumentContext formats all referenced documents for LLM prompts
func (ch *ContextHandler) GetDocumentContext() string {
	var sb strings.Builder

	for path, content := range ch.session.Documents {
		sb.WriteString("## Document: " + path + "\n\n")
		sb.WriteString(content)
		sb.WriteString("\n\n---\n\n")
	}

	return sb.String()
}
