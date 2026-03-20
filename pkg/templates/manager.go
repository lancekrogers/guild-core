// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package templates

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/storage/db"
)

// SQLTemplateManager implements TemplateManager using SQLite
type SQLTemplateManager struct {
	queries *db.Queries
}

// NewSQLTemplateManager creates a new template manager with database connection
func NewSQLTemplateManager(queries *db.Queries) TemplateManager {
	return &SQLTemplateManager{
		queries: queries,
	}
}

// NewTemplateManager creates a new template manager from project directory
func NewTemplateManager(projectDir string) (TemplateManager, error) {
	// For now, return nil - this would require database setup
	// This is a placeholder for compatibility
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "NewTemplateManager not implemented - use NewSQLTemplateManager with database connection", nil).
		WithComponent("templates").
		WithOperation("NewTemplateManager")
}

// Create creates a new template
func (m *SQLTemplateManager) Create(ctx context.Context, template *Template) error {
	if template.ID == "" {
		template.ID = uuid.New().String()
	}

	// Save variables before creating template (since FromDB will clear them)
	variables := template.Variables

	params := template.ToDB()
	dbTemplate, err := m.queries.CreateTemplate(ctx, params)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create template").
			WithComponent("templates").
			WithOperation("Create").
			WithDetails("template_name", template.Name)
	}

	// Update template with database values (this clears Variables)
	template.FromDB(dbTemplate, nil)

	// Create variables if any (use saved variables)
	if len(variables) > 0 {
		err = m.SetVariables(ctx, template.ID, variables)
		if err != nil {
			return err
		}
		// Restore variables to template
		template.Variables = variables
	}

	return nil
}

// Get retrieves a template by ID
func (m *SQLTemplateManager) Get(ctx context.Context, id string) (*Template, error) {
	dbTemplate, err := m.queries.GetTemplate(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, gerror.New(gerror.ErrCodeNotFound, "template not found", nil).
				WithComponent("templates").
				WithOperation("Get").
				WithDetails("template_id", id)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get template").
			WithComponent("templates").
			WithOperation("Get").
			WithDetails("template_id", id)
	}

	// Get variables
	dbVariables, err := m.queries.GetTemplateVariables(ctx, id)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get template variables").
			WithComponent("templates").
			WithOperation("Get").
			WithDetails("template_id", id)
	}

	template := &Template{}
	template.FromDB(dbTemplate, dbVariables)
	return template, nil
}

// GetByName retrieves a template by name
func (m *SQLTemplateManager) GetByName(ctx context.Context, name string) (*Template, error) {
	dbTemplate, err := m.queries.GetTemplateByName(ctx, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, gerror.New(gerror.ErrCodeNotFound, "template not found", nil).
				WithComponent("templates").
				WithOperation("GetByName").
				WithDetails("template_name", name)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get template by name").
			WithComponent("templates").
			WithOperation("GetByName").
			WithDetails("template_name", name)
	}

	// Get variables
	dbVariables, err := m.queries.GetTemplateVariables(ctx, dbTemplate.ID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get template variables").
			WithComponent("templates").
			WithOperation("GetByName").
			WithDetails("template_id", dbTemplate.ID)
	}

	template := &Template{}
	template.FromDB(dbTemplate, dbVariables)
	return template, nil
}

// List retrieves templates with optional filtering
func (m *SQLTemplateManager) List(ctx context.Context, filter *TemplateFilter) ([]*Template, error) {
	var dbTemplates []db.Template
	var err error

	if filter != nil && filter.Category != "" {
		dbTemplates, err = m.queries.ListTemplatesByCategory(ctx, filter.Category)
	} else {
		dbTemplates, err = m.queries.ListTemplates(ctx)
	}

	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list templates").
			WithComponent("templates").
			WithOperation("List")
	}

	templates := make([]*Template, len(dbTemplates))
	for i, dbTemplate := range dbTemplates {
		// Get variables for each template
		dbVariables, err := m.queries.GetTemplateVariables(ctx, dbTemplate.ID)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get template variables").
				WithComponent("templates").
				WithOperation("List").
				WithDetails("template_id", dbTemplate.ID)
		}

		template := &Template{}
		template.FromDB(dbTemplate, dbVariables)
		templates[i] = template
	}

	return templates, nil
}

// Update updates an existing template
func (m *SQLTemplateManager) Update(ctx context.Context, template *Template) error {
	params := db.UpdateTemplateParams{
		Name:     template.Name,
		Category: template.Category,
		Content:  template.Content,
		ID:       template.ID,
	}

	if template.Description != "" {
		params.Description = &template.Description
	}
	if template.Language != "" {
		params.Language = &template.Language
	}

	dbTemplate, err := m.queries.UpdateTemplate(ctx, params)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to update template").
			WithComponent("templates").
			WithOperation("Update").
			WithDetails("template_id", template.ID)
	}

	// Update template with database values
	template.FromDB(dbTemplate, nil)

	// Update variables if any
	if len(template.Variables) > 0 {
		return m.SetVariables(ctx, template.ID, template.Variables)
	}

	return nil
}

// Delete removes a template and its variables
func (m *SQLTemplateManager) Delete(ctx context.Context, id string) error {
	// Delete variables first (foreign key constraint)
	err := m.queries.DeleteTemplateVariablesByTemplate(ctx, id)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete template variables").
			WithComponent("templates").
			WithOperation("Delete").
			WithDetails("template_id", id)
	}

	// Delete template
	err = m.queries.DeleteTemplate(ctx, id)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete template").
			WithComponent("templates").
			WithOperation("Delete").
			WithDetails("template_id", id)
	}

	return nil
}

// Search searches templates by query string
func (m *SQLTemplateManager) Search(ctx context.Context, query string) ([]*Template, error) {
	dbTemplates, err := m.queries.SearchTemplates(ctx, &query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to search templates").
			WithComponent("templates").
			WithOperation("Search").
			WithDetails("query", query)
	}

	templates := make([]*Template, len(dbTemplates))
	for i, dbTemplate := range dbTemplates {
		// Get variables for each template
		dbVariables, err := m.queries.GetTemplateVariables(ctx, dbTemplate.ID)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get template variables").
				WithComponent("templates").
				WithOperation("Search").
				WithDetails("template_id", dbTemplate.ID)
		}

		template := &Template{}
		template.FromDB(dbTemplate, dbVariables)
		templates[i] = template
	}

	return templates, nil
}

// GetByCategory retrieves templates by category
func (m *SQLTemplateManager) GetByCategory(ctx context.Context, category string) ([]*Template, error) {
	return m.List(ctx, &TemplateFilter{Category: category})
}

// GetMostUsed retrieves most used templates
func (m *SQLTemplateManager) GetMostUsed(ctx context.Context, limit int) ([]*Template, error) {
	dbTemplateRows, err := m.queries.GetMostUsedTemplates(ctx, int64(limit))
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get most used templates").
			WithComponent("templates").
			WithOperation("GetMostUsed")
	}

	templates := make([]*Template, len(dbTemplateRows))
	for i, row := range dbTemplateRows {
		// Convert GetMostUsedTemplatesRow to Template struct
		dbTemplate := db.Template{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			Category:    row.Category,
			Content:     row.Content,
			Language:    row.Language,
			UseCount:    row.UseCount,
			IsBuiltIn:   row.IsBuiltIn,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}

		// Get variables for each template
		dbVariables, err := m.queries.GetTemplateVariables(ctx, row.ID)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get template variables").
				WithComponent("templates").
				WithOperation("GetMostUsed").
				WithDetails("template_id", row.ID)
		}

		template := &Template{}
		template.FromDB(dbTemplate, dbVariables)
		templates[i] = template
	}

	return templates, nil
}

// GetVariables retrieves variables for a template
func (m *SQLTemplateManager) GetVariables(ctx context.Context, templateID string) ([]*TemplateVariable, error) {
	dbVariables, err := m.queries.GetTemplateVariables(ctx, templateID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get template variables").
			WithComponent("templates").
			WithOperation("GetVariables").
			WithDetails("template_id", templateID)
	}

	variables := make([]*TemplateVariable, len(dbVariables))
	for i, dbVar := range dbVariables {
		variable := &TemplateVariable{
			ID:         dbVar.ID,
			TemplateID: dbVar.TemplateID,
			Name:       dbVar.Name,
		}
		if dbVar.Description != nil {
			variable.Description = *dbVar.Description
		}
		if dbVar.DefaultValue != nil {
			variable.DefaultValue = *dbVar.DefaultValue
		}
		if dbVar.Required != nil {
			variable.Required = *dbVar.Required
		}
		if dbVar.VariableType != nil {
			variable.Type = VariableType(*dbVar.VariableType)
		}
		if dbVar.CreatedAt != nil {
			variable.CreatedAt = *dbVar.CreatedAt
		}

		// Parse options JSON if present
		if dbVar.Options != nil {
			if optionsBytes, ok := dbVar.Options.([]byte); ok {
				var options []string
				if err := json.Unmarshal(optionsBytes, &options); err == nil {
					variable.Options = options
				}
			}
		}

		variables[i] = variable
	}

	return variables, nil
}

// SetVariables sets variables for a template (replaces existing)
func (m *SQLTemplateManager) SetVariables(ctx context.Context, templateID string, variables []*TemplateVariable) error {
	// Delete existing variables
	err := m.queries.DeleteTemplateVariablesByTemplate(ctx, templateID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to delete existing template variables").
			WithComponent("templates").
			WithOperation("SetVariables").
			WithDetails("template_id", templateID)
	}

	// Create new variables
	for _, variable := range variables {
		if variable.ID == "" {
			variable.ID = uuid.New().String()
		}
		variable.TemplateID = templateID

		params := db.CreateTemplateVariableParams{
			ID:         variable.ID,
			TemplateID: templateID,
			Name:       variable.Name,
			Required:   &variable.Required,
		}

		if variable.Description != "" {
			params.Description = &variable.Description
		}
		if variable.DefaultValue != "" {
			params.DefaultValue = &variable.DefaultValue
		}
		if variable.Type != "" {
			varType := string(variable.Type)
			params.VariableType = &varType
		}
		if len(variable.Options) > 0 {
			optionsBytes, err := json.Marshal(variable.Options)
			if err == nil {
				params.Options = optionsBytes
			}
		}

		_, err := m.queries.CreateTemplateVariable(ctx, params)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create template variable").
				WithComponent("templates").
				WithOperation("SetVariables").
				WithDetails("template_id", templateID).
				WithDetails("variable_name", variable.Name)
		}
	}

	return nil
}

// Render renders a template with variable substitution
func (m *SQLTemplateManager) Render(ctx context.Context, templateID string, variables map[string]interface{}) (string, error) {
	template, err := m.Get(ctx, templateID)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get template for rendering").
			WithComponent("templates").
			WithOperation("Render").
			WithDetails("template_id", templateID)
	}

	rendered, err := m.RenderContent(ctx, template.Content, variables)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to render template content").
			WithComponent("templates").
			WithOperation("Render").
			WithDetails("template_id", templateID)
	}

	return rendered, nil
}

// RenderContent renders template content with variable substitution
func (m *SQLTemplateManager) RenderContent(ctx context.Context, content string, variables map[string]interface{}) (string, error) {
	result := content

	// Simple variable substitution: {{variable_name}}
	for name, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", name)
		valueStr := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, valueStr)
	}

	return result, nil
}

// RecordUsage records template usage for analytics
func (m *SQLTemplateManager) RecordUsage(ctx context.Context, templateID string, campaignID *string, variables map[string]interface{}, context string) error {
	usageID := uuid.New().String()

	// Serialize variables
	var variablesBytes []byte
	var err error
	if variables != nil {
		variablesBytes, err = json.Marshal(variables)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to serialize variables").
				WithComponent("templates").
				WithOperation("RecordUsage").
				WithDetails("template_id", templateID)
		}
	}

	params := db.CreateTemplateUsageParams{
		ID:            usageID,
		TemplateID:    templateID,
		CampaignID:    campaignID,
		VariablesUsed: variablesBytes,
	}

	if context != "" {
		params.Context = &context
	}

	_, err = m.queries.CreateTemplateUsage(ctx, params)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to record template usage").
			WithComponent("templates").
			WithOperation("RecordUsage").
			WithDetails("template_id", templateID)
	}

	// Increment template use count
	err = m.queries.IncrementTemplateUsage(ctx, templateID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to increment template use count").
			WithComponent("templates").
			WithOperation("RecordUsage").
			WithDetails("template_id", templateID)
	}

	return nil
}

// GetUsageStats retrieves usage statistics for a template
func (m *SQLTemplateManager) GetUsageStats(ctx context.Context, templateID string) (*UsageStats, error) {
	// Get total usage count
	useCount, err := m.queries.GetTemplateUsageCount(ctx, templateID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get template usage count").
			WithComponent("templates").
			WithOperation("GetUsageStats").
			WithDetails("template_id", templateID)
	}

	var totalUsage int64
	if useCount != nil {
		totalUsage = *useCount
	}

	stats := &UsageStats{
		TemplateID: templateID,
		TotalUsage: totalUsage,
		// TODO: Add more detailed stats queries
		RecentUsage:    0,
		AveragePerWeek: 0,
	}

	return stats, nil
}

// ListCategories retrieves all template categories
func (m *SQLTemplateManager) ListCategories(ctx context.Context) ([]*TemplateCategory, error) {
	dbCategories, err := m.queries.ListTemplateCategories(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list template categories").
			WithComponent("templates").
			WithOperation("ListCategories")
	}

	categories := make([]*TemplateCategory, len(dbCategories))
	for i, dbCat := range dbCategories {
		category := &TemplateCategory{
			ID:   dbCat.ID,
			Name: dbCat.Name,
		}
		if dbCat.Description != nil {
			category.Description = *dbCat.Description
		}
		if dbCat.Icon != nil {
			category.Icon = *dbCat.Icon
		}
		if dbCat.SortOrder != nil {
			category.SortOrder = *dbCat.SortOrder
		}
		if dbCat.CreatedAt != nil {
			category.CreatedAt = *dbCat.CreatedAt
		}
		categories[i] = category
	}

	return categories, nil
}

// CreateCategory creates a new template category
func (m *SQLTemplateManager) CreateCategory(ctx context.Context, category *TemplateCategory) error {
	if category.ID == "" {
		category.ID = uuid.New().String()
	}

	params := db.CreateTemplateCategoryParams{
		ID:   category.ID,
		Name: category.Name,
	}

	if category.Description != "" {
		params.Description = &category.Description
	}
	if category.Icon != "" {
		params.Icon = &category.Icon
	}
	params.SortOrder = &category.SortOrder

	dbCategory, err := m.queries.CreateTemplateCategory(ctx, params)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create template category").
			WithComponent("templates").
			WithOperation("CreateCategory").
			WithDetails("category_name", category.Name)
	}

	// Update category with database values
	if dbCategory.CreatedAt != nil {
		category.CreatedAt = *dbCategory.CreatedAt
	}

	return nil
}

// Export exports templates to JSON format
func (m *SQLTemplateManager) Export(ctx context.Context, templateIDs []string) ([]byte, error) {
	templates := make([]*Template, len(templateIDs))
	for i, id := range templateIDs {
		template, err := m.Get(ctx, id)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get template for export").
				WithComponent("templates").
				WithOperation("Export").
				WithDetails("template_id", id)
		}
		templates[i] = template
	}

	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal templates").
			WithComponent("templates").
			WithOperation("Export")
	}

	return data, nil
}

// Import imports templates from JSON format
func (m *SQLTemplateManager) Import(ctx context.Context, data []byte, overwrite bool) (*ImportResult, error) {
	var templates []*Template
	err := json.Unmarshal(data, &templates)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to unmarshal templates").
			WithComponent("templates").
			WithOperation("Import")
	}

	result := &ImportResult{
		ImportedIDs: make([]string, 0, len(templates)),
		Errors:      make([]string, 0),
	}

	for _, template := range templates {
		// Check if template exists
		existing, err := m.GetByName(ctx, template.Name)
		if err == nil && existing != nil {
			if !overwrite {
				result.SkippedCount++
				continue
			}
			// Update existing template
			template.ID = existing.ID
			err = m.Update(ctx, template)
		} else {
			// Create new template
			template.ID = uuid.New().String()
			err = m.Create(ctx, template)
		}

		if err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, fmt.Sprintf("Template '%s': %v", template.Name, err))
		} else {
			result.ImportedCount++
			result.ImportedIDs = append(result.ImportedIDs, template.ID)
		}
	}

	return result, nil
}

// InstallBuiltInTemplates installs default templates
func (m *SQLTemplateManager) InstallBuiltInTemplates(ctx context.Context) error {
	builtInTemplates := m.GetBuiltInTemplates()

	for _, template := range builtInTemplates {
		// Check if template already exists
		existing, err := m.GetByName(ctx, template.Name)
		if err == nil && existing != nil {
			continue // Skip if already exists
		}

		template.IsBuiltIn = true
		err = m.Create(ctx, template)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to install built-in template").
				WithComponent("templates").
				WithOperation("InstallBuiltInTemplates").
				WithDetails("template_name", template.Name)
		}
	}

	return nil
}

// GetBuiltInTemplates returns the default built-in templates
func (m *SQLTemplateManager) GetBuiltInTemplates() []*Template {
	return GetDefaultTemplates()
}

// GetContextualSuggestions returns contextual template suggestions
func (m *SQLTemplateManager) GetContextualSuggestions(contextMap map[string]interface{}) ([]*Template, error) {
	// Simple implementation - return most used templates
	ctx, ok := contextMap["ctx"].(context.Context)
	if !ok {
		ctx = context.Background()
	}
	return m.GetMostUsed(ctx, 5)
}

// RenderTemplate renders a template with variables (convenience method)
func (m *SQLTemplateManager) RenderTemplate(templateID string, variables map[string]interface{}) (string, error) {
	ctx := context.Background()
	return m.Render(ctx, templateID, variables)
}

// SearchTemplates searches for templates matching a query
func (m *SQLTemplateManager) SearchTemplates(query string, limit int) ([]*TemplateSearchResult, error) {
	ctx := context.Background()
	templates, err := m.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	// Convert to search results with basic relevance scoring
	results := make([]*TemplateSearchResult, 0, len(templates))
	for i, template := range templates {
		if limit > 0 && i >= limit {
			break
		}

		// Simple matching to populate Matches field
		var matches []string
		queryLower := strings.ToLower(query)
		if strings.Contains(strings.ToLower(template.Name), queryLower) {
			matches = append(matches, "name")
		}
		if strings.Contains(strings.ToLower(template.Description), queryLower) {
			matches = append(matches, "description")
		}
		if strings.Contains(strings.ToLower(template.Category), queryLower) {
			matches = append(matches, "category")
		}

		results = append(results, &TemplateSearchResult{
			Template:  template,
			Relevance: 1.0 - (float64(i) / float64(len(templates))), // Simple relevance scoring
			Matches:   matches,
		})
	}

	return results, nil
}
