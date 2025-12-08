// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package templates

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/pkg/storage/db"
)

func setupTestDB(t *testing.T) (*db.Queries, func()) {
	// Create in-memory SQLite database
	database, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create tables
	_, err = database.Exec(`
		CREATE TABLE templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			category TEXT NOT NULL DEFAULT 'general',
			content TEXT NOT NULL,
			language TEXT,
			use_count INTEGER DEFAULT 0,
			is_built_in BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE template_variables (
			id TEXT PRIMARY KEY,
			template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			description TEXT,
			default_value TEXT,
			required BOOLEAN DEFAULT FALSE,
			variable_type TEXT DEFAULT 'text',
			options JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(template_id, name)
		);

		CREATE TABLE template_categories (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			icon TEXT,
			sort_order INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE template_usage (
			id TEXT PRIMARY KEY,
			template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
			campaign_id TEXT,
			used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			variables_used JSON,
			context TEXT
		);
	`)
	require.NoError(t, err)

	queries := db.New(database)

	cleanup := func() {
		database.Close()
	}

	return queries, cleanup
}

func TestSQLTemplateManager_Create(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	template := &Template{
		Name:        "Test Template",
		Description: "A test template",
		Category:    "testing",
		Content:     "Hello {{name}}!",
		Variables: []*TemplateVariable{
			{
				Name:        "name",
				Description: "Name to greet",
				Required:    true,
				Type:        VariableTypeText,
			},
		},
	}

	err := manager.Create(ctx, template)
	assert.NoError(t, err)
	assert.NotEmpty(t, template.ID)
	assert.False(t, template.CreatedAt.IsZero())
}

func TestSQLTemplateManager_Get(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	// Create template
	template := &Template{
		Name:     "Test Template",
		Category: "testing",
		Content:  "Hello {{name}}!",
		Variables: []*TemplateVariable{
			{
				Name:     "name",
				Required: true,
				Type:     VariableTypeText,
			},
		},
	}

	err := manager.Create(ctx, template)
	require.NoError(t, err)

	// Get template
	retrieved, err := manager.Get(ctx, template.ID)
	assert.NoError(t, err)
	assert.Equal(t, template.Name, retrieved.Name)
	assert.Equal(t, template.Content, retrieved.Content)
	assert.Len(t, retrieved.Variables, 1)
	assert.Equal(t, "name", retrieved.Variables[0].Name)
}

func TestSQLTemplateManager_GetByName(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	template := &Template{
		Name:     "Unique Template Name",
		Category: "testing",
		Content:  "Test content",
	}

	err := manager.Create(ctx, template)
	require.NoError(t, err)

	retrieved, err := manager.GetByName(ctx, "Unique Template Name")
	assert.NoError(t, err)
	assert.Equal(t, template.ID, retrieved.ID)
}

func TestSQLTemplateManager_List(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	// Create multiple templates
	templates := []*Template{
		{Name: "Template 1", Category: "testing", Content: "Content 1"},
		{Name: "Template 2", Category: "testing", Content: "Content 2"},
		{Name: "Template 3", Category: "production", Content: "Content 3"},
	}

	for _, tmpl := range templates {
		err := manager.Create(ctx, tmpl)
		require.NoError(t, err)
	}

	// List all templates
	allTemplates, err := manager.List(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, allTemplates, 3)

	// List by category
	testingTemplates, err := manager.List(ctx, &TemplateFilter{Category: "testing"})
	assert.NoError(t, err)
	assert.Len(t, testingTemplates, 2)
}

func TestSQLTemplateManager_Update(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	template := &Template{
		Name:     "Original Name",
		Category: "testing",
		Content:  "Original content",
	}

	err := manager.Create(ctx, template)
	require.NoError(t, err)

	// Update template
	template.Name = "Updated Name"
	template.Content = "Updated content"

	err = manager.Update(ctx, template)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := manager.Get(ctx, template.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.Name)
	assert.Equal(t, "Updated content", retrieved.Content)
}

func TestSQLTemplateManager_Delete(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	template := &Template{
		Name:     "To Delete",
		Category: "testing",
		Content:  "Will be deleted",
	}

	err := manager.Create(ctx, template)
	require.NoError(t, err)

	// Delete template
	err = manager.Delete(ctx, template.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = manager.Get(ctx, template.ID)
	assert.Error(t, err)
}

func TestSQLTemplateManager_Search(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	templates := []*Template{
		{Name: "API Documentation", Category: "docs", Content: "API guide content"},
		{Name: "Bug Report", Category: "debugging", Content: "Report bugs here"},
		{Name: "Code Review", Category: "code", Content: "Review code quality"},
	}

	for _, tmpl := range templates {
		err := manager.Create(ctx, tmpl)
		require.NoError(t, err)
	}

	// Search by name
	results, err := manager.Search(ctx, "API")
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "API Documentation", results[0].Name)

	// Search by content
	results, err = manager.Search(ctx, "bugs")
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Bug Report", results[0].Name)
}

func TestSQLTemplateManager_Render(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	template := &Template{
		Name:     "Greeting Template",
		Category: "testing",
		Content:  "Hello {{name}}, welcome to {{project}}!",
		Variables: []*TemplateVariable{
			{
				Name:     "name",
				Required: true,
				Type:     VariableTypeText,
			},
			{
				Name:     "project",
				Required: true,
				Type:     VariableTypeText,
			},
		},
	}

	err := manager.Create(ctx, template)
	require.NoError(t, err)

	// Render template
	variables := map[string]interface{}{
		"name":    "Alice",
		"project": "Guild Framework",
	}

	result, err := manager.Render(ctx, template.ID, variables)
	assert.NoError(t, err)
	assert.Equal(t, "Hello Alice, welcome to Guild Framework!", result)
}

func TestSQLTemplateManager_RecordUsage(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	template := &Template{
		Name:     "Usage Test",
		Category: "testing",
		Content:  "Test content",
	}

	err := manager.Create(ctx, template)
	require.NoError(t, err)

	variables := map[string]interface{}{
		"test": "value",
	}

	err = manager.RecordUsage(ctx, template.ID, nil, variables, "test context")
	assert.NoError(t, err)

	// Verify usage stats
	stats, err := manager.GetUsageStats(ctx, template.ID)
	assert.NoError(t, err)
	assert.Equal(t, template.ID, stats.TemplateID)
	assert.Equal(t, int64(1), stats.TotalUsage)
}

func TestSQLTemplateManager_ImportExport(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	// Create templates to export
	templates := []*Template{
		{Name: "Export Test 1", Category: "testing", Content: "Content 1"},
		{Name: "Export Test 2", Category: "testing", Content: "Content 2"},
	}

	var templateIDs []string
	for _, tmpl := range templates {
		err := manager.Create(ctx, tmpl)
		require.NoError(t, err)
		templateIDs = append(templateIDs, tmpl.ID)
	}

	// Export templates
	exportData, err := manager.Export(ctx, templateIDs)
	assert.NoError(t, err)
	assert.NotEmpty(t, exportData)

	// Import templates to new manager (simulating different database)
	queries2, cleanup2 := setupTestDB(t)
	defer cleanup2()
	manager2 := NewSQLTemplateManager(queries2)

	result, err := manager2.Import(ctx, exportData, false)
	assert.NoError(t, err)
	assert.Equal(t, 2, result.ImportedCount)
	assert.Equal(t, 0, result.ErrorCount)
	assert.Len(t, result.ImportedIDs, 2)

	// Verify imported templates
	imported, err := manager2.List(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, imported, 2)
}

func TestSQLTemplateManager_InstallBuiltInTemplates(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	err := manager.InstallBuiltInTemplates(ctx)
	assert.NoError(t, err)

	// Verify templates were installed
	templates, err := manager.List(ctx, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, templates)

	// Check that they're marked as built-in
	for _, tmpl := range templates {
		assert.True(t, tmpl.IsBuiltIn, "Template %s should be marked as built-in", tmpl.Name)
	}

	// Installing again should not create duplicates
	err = manager.InstallBuiltInTemplates(ctx)
	assert.NoError(t, err)

	templatesAfter, err := manager.List(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, len(templates), len(templatesAfter))
}

func TestSQLTemplateManager_Categories(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	category := &TemplateCategory{
		Name:        "Test Category",
		Description: "A test category",
		Icon:        "🧪",
		SortOrder:   10,
	}

	err := manager.CreateCategory(ctx, category)
	assert.NoError(t, err)
	assert.NotEmpty(t, category.ID)

	categories, err := manager.ListCategories(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, categories)

	// Find our category
	var found *TemplateCategory
	for _, cat := range categories {
		if cat.Name == "Test Category" {
			found = cat
			break
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "A test category", found.Description)
	assert.Equal(t, "🧪", found.Icon)
	assert.Equal(t, int64(10), found.SortOrder)
}

func TestSQLTemplateManager_Variables(t *testing.T) {
	queries, cleanup := setupTestDB(t)
	defer cleanup()

	manager := NewSQLTemplateManager(queries)
	ctx := context.Background()

	template := &Template{
		Name:     "Variable Test",
		Category: "testing",
		Content:  "Test with {{var1}} and {{var2}}",
	}

	err := manager.Create(ctx, template)
	require.NoError(t, err)

	variables := []*TemplateVariable{
		{
			Name:         "var1",
			Description:  "First variable",
			Required:     true,
			Type:         VariableTypeText,
			DefaultValue: "default1",
		},
		{
			Name:        "var2",
			Description: "Second variable",
			Required:    false,
			Type:        VariableTypeSelect,
			Options:     []string{"option1", "option2", "option3"},
		},
	}

	err = manager.SetVariables(ctx, template.ID, variables)
	assert.NoError(t, err)

	retrievedVars, err := manager.GetVariables(ctx, template.ID)
	assert.NoError(t, err)
	assert.Len(t, retrievedVars, 2)

	// Check first variable
	var1 := retrievedVars[0]
	if var1.Name != "var1" {
		var1 = retrievedVars[1]
	}
	assert.Equal(t, "var1", var1.Name)
	assert.Equal(t, "First variable", var1.Description)
	assert.True(t, var1.Required)
	assert.Equal(t, VariableTypeText, var1.Type)

	// Check second variable
	var2 := retrievedVars[1]
	if var2.Name != "var2" {
		var2 = retrievedVars[0]
	}
	assert.Equal(t, "var2", var2.Name)
	assert.Equal(t, VariableTypeSelect, var2.Type)
	assert.Len(t, var2.Options, 3)
}
