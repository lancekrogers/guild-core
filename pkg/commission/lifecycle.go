// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commission

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// LifecycleManager handles the commission lifecycle operations
type LifecycleManager struct {
	manager         *Manager
	commissionsPath string
	aiDocsPath      string
	specsPath       string
	guildReadyFile  string
}

// newLifecycleManager creates a new lifecycle manager (private constructor)
func newLifecycleManager(manager *Manager, basePath string) *LifecycleManager {
	// If basePath is empty, default to current directory
	if basePath == "" {
		var err error
		basePath, err = os.Getwd()
		if err != nil {
			// Rather than failing, default to temporary directory
			basePath = os.TempDir()
		}
	}

	return &LifecycleManager{
		manager:         manager,
		commissionsPath: filepath.Join(basePath, "commissions"),
		aiDocsPath:      filepath.Join(basePath, "ai_docs"),
		specsPath:       filepath.Join(basePath, "specs"),
		guildReadyFile:  ".guildready",
	}
}

// DefaultLifecycleManagerFactory creates a lifecycle manager factory for registry use
func DefaultLifecycleManagerFactory(manager *Manager, basePath string) *LifecycleManager {
	return newLifecycleManager(manager, basePath)
}

// CreateCommissionFromDescription creates a new commission from a natural language description
func (l *LifecycleManager) CreateCommissionFromDescription(ctx context.Context, description string) (*Commission, error) {
	// First, ensure commissions directory exists
	if err := os.MkdirAll(l.commissionsPath, 0755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create commissions directory").WithComponent("commission").WithOperation("CreateCommissionFromDescription")
	}

	// Create a new commission with a title derived from the description
	title := deriveTitle(description)
	fileName := sanitizeFilename(title) + ".md"
	filePath := filepath.Join(l.commissionsPath, fileName)

	// Create the commission object
	obj := NewCommission(title, description)
	obj.Status = StatusDraft
	obj.FilePath = filePath
	obj.Goal = description

	// Generate initial markdown content
	content := formatCommissionMarkdown(obj)

	// Write to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write commission file").WithComponent("commission").WithOperation("CreateCommissionFromDescription")
	}

	// Save to manager
	obj.Source = filePath
	obj.Content = content
	obj.CalculateCompletion()
	if err := l.manager.SaveCommission(ctx, obj); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save commission").WithComponent("commission").WithOperation("CreateCommissionFromDescription")
	}

	return obj, nil
}

// AddContext adds context to an commission and updates its lifecycle state
func (l *LifecycleManager) AddContext(ctx context.Context, commissionID, context string) error {
	// Get the commission
	obj, err := l.manager.GetCommission(ctx, commissionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get commission").WithComponent("commission").WithOperation("AddContext")
	}

	// Parse the context for any document references
	context, refs := parseDocumentReferences(context)

	// Add the context to the commission
	if obj.Context == nil {
		obj.Context = []string{context}
	} else {
		obj.Context = append(obj.Context, context)
	}

	// Add any document references
	for _, ref := range refs {
		if !containsString(obj.Context, ref) {
			obj.Context = append(obj.Context, ref)
		}
	}

	// Update status if appropriate
	if obj.Status == StatusEmpty {
		obj.Status = StatusDraft
	} else if obj.Status == StatusDraft {
		obj.Status = StatusInProgress
	}

	// Increment iteration counter
	obj.IncrementIteration()

	// Calculate completion
	obj.CalculateCompletion()

	// Update the file content
	content := formatCommissionMarkdown(obj)
	obj.Content = content

	// Save changes to file
	if err := os.WriteFile(obj.FilePath, []byte(content), 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update commission file").WithComponent("commission").WithOperation("AddContext")
	}

	// Save to manager
	if err := l.manager.SaveCommission(ctx, obj); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save commission").WithComponent("commission").WithOperation("AddContext")
	}

	return nil
}

// GenerateProjectStructure generates ai_docs and specs from an commission
func (l *LifecycleManager) GenerateProjectStructure(ctx context.Context, commissionID string) error {
	// Get the commission
	obj, err := l.manager.GetCommission(ctx, commissionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get commission").WithComponent("commission").WithOperation("GenerateProjectStructure")
	}

	// Create project directory using commission title
	projectName := sanitizeFilename(obj.Title)
	projectDir := filepath.Join(filepath.Dir(obj.FilePath), projectName)

	// Create the ai_docs and specs directories
	aiDocsDir := filepath.Join(projectDir, "ai_docs")
	specsDir := filepath.Join(projectDir, "specs")

	if err := os.MkdirAll(aiDocsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create ai_docs directory").WithComponent("commission").WithOperation("GenerateProjectStructure")
	}

	if err := os.MkdirAll(specsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create specs directory").WithComponent("commission").WithOperation("GenerateProjectStructure")
	}

	// TODO: In a real implementation, this would generate actual ai_docs and specs
	// based on the commission using the LLM generators.
	// For now, we'll just create stub files

	// Create README.md in ai_docs
	aiDocsReadme := filepath.Join(aiDocsDir, "README.md")
	aiDocsContent := fmt.Sprintf("# AI Docs for %s\n\nGenerated from commission: %s\n\n## Overview\n\n%s\n\n## Related Specs\n\n@spec/README.md\n",
		obj.Title,
		filepath.Base(obj.FilePath),
		obj.Description)

	if err := os.WriteFile(aiDocsReadme, []byte(aiDocsContent), 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write ai_docs README").WithComponent("commission").WithOperation("GenerateProjectStructure")
	}

	// Create README.md in specs
	specsReadme := filepath.Join(specsDir, "README.md")
	specsContent := fmt.Sprintf("# Specifications for %s\n\nGenerated from commission: %s\n\n## Requirements\n\n",
		obj.Title,
		filepath.Base(obj.FilePath))

	// Add requirements
	for _, req := range obj.Requirements {
		specsContent += fmt.Sprintf("- %s\n", req)
	}

	if err := os.WriteFile(specsReadme, []byte(specsContent), 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write specs README").WithComponent("commission").WithOperation("GenerateProjectStructure")
	}

	// Update commission with references to the new files
	obj.AIDocs = []string{aiDocsReadme}
	obj.Specs = []string{specsReadme}

	// Update status
	obj.Status = StatusInProgress
	obj.IncrementIteration()
	obj.CalculateCompletion()

	// Save to manager
	if err := l.manager.SaveCommission(ctx, obj); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save commission").WithComponent("commission").WithOperation("GenerateProjectStructure")
	}

	return nil
}

// MarkCommissionReady marks an commission as ready for implementation
func (l *LifecycleManager) MarkCommissionReady(ctx context.Context, commissionID string) error {
	// Get the commission
	obj, err := l.manager.GetCommission(ctx, commissionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get commission").WithComponent("commission").WithOperation("MarkCommissionReady")
	}

	// Check if the commission has been processed (has ai_docs and specs)
	if len(obj.AIDocs) == 0 || len(obj.Specs) == 0 {
		return gerror.New(gerror.ErrCodeValidation, "commission must have generated ai_docs and specs before being marked as ready", nil).WithComponent("commission").WithOperation("MarkCommissionReady")
	}

	// Update status
	obj.Status = StatusReady
	obj.IncrementIteration()
	obj.CalculateCompletion()

	// Create .guildready file in the project directory
	projectName := sanitizeFilename(obj.Title)
	projectDir := filepath.Join(filepath.Dir(obj.FilePath), projectName)
	readyFile := filepath.Join(projectDir, l.guildReadyFile)

	// Write current time to the ready file
	readyContent := fmt.Sprintf("Commission marked ready at: %s\n", time.Now().Format(time.RFC3339))
	if err := os.WriteFile(readyFile, []byte(readyContent), 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create ready file").WithComponent("commission").WithOperation("MarkCommissionReady")
	}

	// Save to manager
	if err := l.manager.SaveCommission(ctx, obj); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save commission").WithComponent("commission").WithOperation("MarkCommissionReady")
	}

	return nil
}

// MarkCommissionImplementing marks an commission as being implemented
func (l *LifecycleManager) MarkCommissionImplementing(ctx context.Context, commissionID string) error {
	// Get the commission
	obj, err := l.manager.GetCommission(ctx, commissionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get commission").WithComponent("commission").WithOperation("MarkCommissionImplementing")
	}

	// Check if the commission is ready
	if obj.Status != StatusReady {
		return gerror.New(gerror.ErrCodeValidation, "commission must be ready before being marked as implementing", nil).WithComponent("commission").WithOperation("MarkCommissionImplementing")
	}

	// Update status
	obj.Status = StatusImplementing
	obj.IncrementIteration()

	// Save to manager
	if err := l.manager.SaveCommission(ctx, obj); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save commission").WithComponent("commission").WithOperation("MarkCommissionImplementing")
	}

	return nil
}

// MarkCommissionCompleted marks an commission as completed
func (l *LifecycleManager) MarkCommissionCompleted(ctx context.Context, commissionID string) error {
	// Get the commission
	obj, err := l.manager.GetCommission(ctx, commissionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get commission").WithComponent("commission").WithOperation("MarkCommissionCompleted")
	}

	// Update status
	obj.Status = StatusCompleted
	obj.IncrementIteration()
	obj.Completion = 1.0 // Fully complete

	// Add completion timestamp to the .guildready file
	projectName := sanitizeFilename(obj.Title)
	projectDir := filepath.Join(filepath.Dir(obj.FilePath), projectName)
	readyFile := filepath.Join(projectDir, l.guildReadyFile)

	// Append completion time to the ready file
	completionContent := fmt.Sprintf("Commission completed at: %s\n", time.Now().Format(time.RFC3339))
	f, err := os.OpenFile(readyFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to open ready file").WithComponent("commission").WithOperation("MarkCommissionCompleted")
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			// Log the close error but don't override the main error
			_ = gerror.Wrap(closeErr, gerror.ErrCodeStorage, "failed to close ready file").
				WithComponent("commission").
				WithOperation("MarkCommissionCompleted")
		}
	}()

	if _, err := f.WriteString(completionContent); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update ready file").WithComponent("commission").WithOperation("MarkCommissionCompleted")
	}

	// Save to manager
	if err := l.manager.SaveCommission(ctx, obj); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save commission").WithComponent("commission").WithOperation("MarkCommissionCompleted")
	}

	return nil
}

// Helper functions

// deriveTitle derives a title from a description
func deriveTitle(description string) string {
	// Extract the first line as title
	lines := strings.Split(description, "\n")
	title := lines[0]

	// If title is too long, truncate it
	maxTitleLength := 50
	if len(title) > maxTitleLength {
		// Truncate at a word boundary if possible
		words := strings.Fields(title[:maxTitleLength])
		if len(words) > 0 {
			title = strings.Join(words[:len(words)-1], " ") + "..."
		} else {
			title = title[:maxTitleLength-3] + "..."
		}
	}

	return title
}

// formatCommissionMarkdown formats an commission as markdown
func formatCommissionMarkdown(obj *Commission) string {
	var content strings.Builder

	// Title
	content.WriteString(fmt.Sprintf("# 🧠 Goal\n\n%s\n\n", obj.Goal))

	// Context
	content.WriteString("# 📂 Context\n\n")
	if len(obj.Context) > 0 {
		for _, ctx := range obj.Context {
			content.WriteString(ctx + "\n\n")
		}
	} else {
		content.WriteString("No context provided yet.\n\n")
	}

	// Requirements
	content.WriteString("# 🔧 Requirements\n\n")
	if len(obj.Requirements) > 0 {
		for _, req := range obj.Requirements {
			content.WriteString(fmt.Sprintf("- %s\n", req))
		}
		content.WriteString("\n")
	} else {
		content.WriteString("No specific requirements defined yet.\n\n")
	}

	// Tags
	content.WriteString("# 📌 Tags\n\n")
	if len(obj.Tags) > 0 {
		for _, tag := range obj.Tags {
			content.WriteString(fmt.Sprintf("- %s\n", tag))
		}
		content.WriteString("\n")
	} else {
		content.WriteString("No tags defined yet.\n\n")
	}

	// Related
	content.WriteString("# 🔗 Related\n\n")
	if len(obj.Related) > 0 {
		for _, rel := range obj.Related {
			content.WriteString(fmt.Sprintf("- %s\n", rel))
		}
		content.WriteString("\n")
	} else {
		content.WriteString("No related commissions defined yet.\n\n")
	}

	return content.String()
}

// parseDocumentReferences parses document references in the form @path/to/file
func parseDocumentReferences(content string) (string, []string) {
	// Regular expression to match document references like @ai_docs/... or @spec/...
	referenceRegex := regexp.MustCompile(`@(ai_docs|spec)/[a-zA-Z0-9_\.\-\/]+`)

	// Extract all references
	matches := referenceRegex.FindAllString(content, -1)

	// Return the content and the list of references
	return content, matches
}
