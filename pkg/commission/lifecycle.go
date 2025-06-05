package commission

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// LifecycleManager handles the objective lifecycle operations
type LifecycleManager struct {
	manager         *Manager
	objectivesPath  string
	aiDocsPath      string
	specsPath       string
	guildReadyFile  string
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(manager *Manager, basePath string) *LifecycleManager {
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
		manager:        manager,
		objectivesPath: filepath.Join(basePath, "objectives"),
		aiDocsPath:     filepath.Join(basePath, "ai_docs"),
		specsPath:      filepath.Join(basePath, "specs"),
		guildReadyFile: ".guildready",
	}
}

// CreateObjectiveFromDescription creates a new objective from a natural language description
func (l *LifecycleManager) CreateObjectiveFromDescription(ctx context.Context, description string) (*Commission, error) {
	// First, ensure objectives directory exists
	if err := os.MkdirAll(l.objectivesPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create objectives directory: %w", err)
	}

	// Create a new objective with a title derived from the description
	title := deriveTitle(description)
	fileName := sanitizeFilename(title) + ".md"
	filePath := filepath.Join(l.objectivesPath, fileName)

	// Create the objective object
	obj := NewCommission(title, description)
	obj.Status = StatusDraft
	obj.FilePath = filePath
	obj.Goal = description

	// Generate initial markdown content
	content := formatObjectiveMarkdown(obj)
	
	// Write to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write objective file: %w", err)
	}

	// Save to manager
	obj.Source = filePath
	obj.Content = content
	obj.CalculateCompletion()
	if err := l.manager.SaveObjective(ctx, obj); err != nil {
		return nil, fmt.Errorf("failed to save objective: %w", err)
	}

	return obj, nil
}

// AddContext adds context to an objective and updates its lifecycle state
func (l *LifecycleManager) AddContext(ctx context.Context, objectiveID, context string) error {
	// Get the objective
	obj, err := l.manager.GetObjective(ctx, objectiveID)
	if err != nil {
		return fmt.Errorf("failed to get objective: %w", err)
	}

	// Parse the context for any document references
	context, refs := parseDocumentReferences(context)
	
	// Add the context to the objective
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
	content := formatObjectiveMarkdown(obj)
	obj.Content = content
	
	// Save changes to file
	if err := os.WriteFile(obj.FilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to update objective file: %w", err)
	}

	// Save to manager
	if err := l.manager.SaveObjective(ctx, obj); err != nil {
		return fmt.Errorf("failed to save objective: %w", err)
	}

	return nil
}

// GenerateProjectStructure generates ai_docs and specs from an objective
func (l *LifecycleManager) GenerateProjectStructure(ctx context.Context, objectiveID string) error {
	// Get the objective
	obj, err := l.manager.GetObjective(ctx, objectiveID)
	if err != nil {
		return fmt.Errorf("failed to get objective: %w", err)
	}

	// Create project directory using objective title
	projectName := sanitizeFilename(obj.Title)
	projectDir := filepath.Join(filepath.Dir(obj.FilePath), projectName)
	
	// Create the ai_docs and specs directories
	aiDocsDir := filepath.Join(projectDir, "ai_docs")
	specsDir := filepath.Join(projectDir, "specs")
	
	if err := os.MkdirAll(aiDocsDir, 0755); err != nil {
		return fmt.Errorf("failed to create ai_docs directory: %w", err)
	}
	
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		return fmt.Errorf("failed to create specs directory: %w", err)
	}

	// TODO: In a real implementation, this would generate actual ai_docs and specs
	// based on the objective using the LLM generators.
	// For now, we'll just create stub files

	// Create README.md in ai_docs
	aiDocsReadme := filepath.Join(aiDocsDir, "README.md")
	aiDocsContent := fmt.Sprintf("# AI Docs for %s\n\nGenerated from objective: %s\n\n## Overview\n\n%s\n\n## Related Specs\n\n@spec/README.md\n", 
		obj.Title, 
		filepath.Base(obj.FilePath),
		obj.Description)
	
	if err := os.WriteFile(aiDocsReadme, []byte(aiDocsContent), 0644); err != nil {
		return fmt.Errorf("failed to write ai_docs README: %w", err)
	}

	// Create README.md in specs
	specsReadme := filepath.Join(specsDir, "README.md")
	specsContent := fmt.Sprintf("# Specifications for %s\n\nGenerated from objective: %s\n\n## Requirements\n\n", 
		obj.Title, 
		filepath.Base(obj.FilePath))
	
	// Add requirements
	for _, req := range obj.Requirements {
		specsContent += fmt.Sprintf("- %s\n", req)
	}
	
	if err := os.WriteFile(specsReadme, []byte(specsContent), 0644); err != nil {
		return fmt.Errorf("failed to write specs README: %w", err)
	}

	// Update objective with references to the new files
	obj.AIDocs = []string{aiDocsReadme}
	obj.Specs = []string{specsReadme}
	
	// Update status
	obj.Status = StatusInProgress
	obj.IncrementIteration()
	obj.CalculateCompletion()

	// Save to manager
	if err := l.manager.SaveObjective(ctx, obj); err != nil {
		return fmt.Errorf("failed to save objective: %w", err)
	}

	return nil
}

// MarkObjectiveReady marks an objective as ready for implementation
func (l *LifecycleManager) MarkObjectiveReady(ctx context.Context, objectiveID string) error {
	// Get the objective
	obj, err := l.manager.GetObjective(ctx, objectiveID)
	if err != nil {
		return fmt.Errorf("failed to get objective: %w", err)
	}

	// Check if the objective has been processed (has ai_docs and specs)
	if len(obj.AIDocs) == 0 || len(obj.Specs) == 0 {
		return fmt.Errorf("objective must have generated ai_docs and specs before being marked as ready")
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
	readyContent := fmt.Sprintf("Objective marked ready at: %s\n", time.Now().Format(time.RFC3339))
	if err := os.WriteFile(readyFile, []byte(readyContent), 0644); err != nil {
		return fmt.Errorf("failed to create ready file: %w", err)
	}

	// Save to manager
	if err := l.manager.SaveObjective(ctx, obj); err != nil {
		return fmt.Errorf("failed to save objective: %w", err)
	}

	return nil
}

// MarkObjectiveImplementing marks an objective as being implemented
func (l *LifecycleManager) MarkObjectiveImplementing(ctx context.Context, objectiveID string) error {
	// Get the objective
	obj, err := l.manager.GetObjective(ctx, objectiveID)
	if err != nil {
		return fmt.Errorf("failed to get objective: %w", err)
	}

	// Check if the objective is ready
	if obj.Status != StatusReady {
		return fmt.Errorf("objective must be ready before being marked as implementing")
	}

	// Update status
	obj.Status = StatusImplementing
	obj.IncrementIteration()

	// Save to manager
	if err := l.manager.SaveObjective(ctx, obj); err != nil {
		return fmt.Errorf("failed to save objective: %w", err)
	}

	return nil
}

// MarkObjectiveCompleted marks an objective as completed
func (l *LifecycleManager) MarkObjectiveCompleted(ctx context.Context, objectiveID string) error {
	// Get the objective
	obj, err := l.manager.GetObjective(ctx, objectiveID)
	if err != nil {
		return fmt.Errorf("failed to get objective: %w", err)
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
	completionContent := fmt.Sprintf("Objective completed at: %s\n", time.Now().Format(time.RFC3339))
	f, err := os.OpenFile(readyFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open ready file: %w", err)
	}
	defer f.Close()
	
	if _, err := f.WriteString(completionContent); err != nil {
		return fmt.Errorf("failed to update ready file: %w", err)
	}

	// Save to manager
	if err := l.manager.SaveObjective(ctx, obj); err != nil {
		return fmt.Errorf("failed to save objective: %w", err)
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

// formatObjectiveMarkdown formats an objective as markdown
func formatObjectiveMarkdown(obj *Commission) string {
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
		content.WriteString("No related objectives defined yet.\n\n")
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