// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package templates

import (
	"context"
	"time"

	"github.com/lancekrogers/guild/pkg/storage/db"
)

// TemplateManager provides template management capabilities
type TemplateManager interface {
	// Template CRUD operations
	Create(ctx context.Context, template *Template) error
	Get(ctx context.Context, id string) (*Template, error)
	GetByName(ctx context.Context, name string) (*Template, error)
	List(ctx context.Context, filter *TemplateFilter) ([]*Template, error)
	Update(ctx context.Context, template *Template) error
	Delete(ctx context.Context, id string) error

	// Template search and discovery
	Search(ctx context.Context, query string) ([]*Template, error)
	GetByCategory(ctx context.Context, category string) ([]*Template, error)
	GetMostUsed(ctx context.Context, limit int) ([]*Template, error)

	// Variable management
	GetVariables(ctx context.Context, templateID string) ([]*TemplateVariable, error)
	SetVariables(ctx context.Context, templateID string, variables []*TemplateVariable) error

	// Template rendering with variable substitution
	Render(ctx context.Context, templateID string, variables map[string]interface{}) (string, error)
	RenderContent(ctx context.Context, content string, variables map[string]interface{}) (string, error)

	// Usage tracking
	RecordUsage(ctx context.Context, templateID string, campaignID *string, variables map[string]interface{}, context string) error
	GetUsageStats(ctx context.Context, templateID string) (*UsageStats, error)

	// Categories
	ListCategories(ctx context.Context) ([]*TemplateCategory, error)
	CreateCategory(ctx context.Context, category *TemplateCategory) error

	// Import/Export
	Export(ctx context.Context, templateIDs []string) ([]byte, error)
	Import(ctx context.Context, data []byte, overwrite bool) (*ImportResult, error)

	// Built-in templates
	InstallBuiltInTemplates(ctx context.Context) error
	GetBuiltInTemplates() []*Template

	// Extended functionality for content formatting
	GetContextualSuggestions(context map[string]interface{}) ([]*Template, error)
	RenderTemplate(templateID string, variables map[string]interface{}) (string, error)
	SearchTemplates(query string, limit int) ([]*TemplateSearchResult, error)
}

// Template represents a reusable template with variables
type Template struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Category    string              `json:"category"`
	Content     string              `json:"content"`
	Language    string              `json:"language,omitempty"`
	UseCount    int64               `json:"use_count"`
	IsBuiltIn   bool                `json:"is_built_in"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Variables   []*TemplateVariable `json:"variables,omitempty"`
}

// TemplateVariable represents a variable placeholder in a template
type TemplateVariable struct {
	ID           string       `json:"id"`
	TemplateID   string       `json:"template_id"`
	Name         string       `json:"name"`
	Description  string       `json:"description,omitempty"`
	DefaultValue string       `json:"default_value,omitempty"`
	Required     bool         `json:"required"`
	Type         VariableType `json:"type"`
	Options      []string     `json:"options,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
}

// TemplateCategory represents a template category for organization
type TemplateCategory struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	SortOrder   int64     `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

// VariableType defines the type of template variable
type VariableType string

const (
	VariableTypeText      VariableType = "text"
	VariableTypeCode      VariableType = "code"
	VariableTypeMultiline VariableType = "multiline"
	VariableTypeSelect    VariableType = "select"
)

// TemplateFilter provides filtering options for template listing
type TemplateFilter struct {
	Category  string `json:"category,omitempty"`
	Language  string `json:"language,omitempty"`
	IsBuiltIn *bool  `json:"is_built_in,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

// UsageStats provides usage statistics for a template
type UsageStats struct {
	TemplateID     string     `json:"template_id"`
	TotalUsage     int64      `json:"total_usage"`
	RecentUsage    int64      `json:"recent_usage_30d"`
	LastUsed       *time.Time `json:"last_used,omitempty"`
	AveragePerWeek float64    `json:"average_per_week"`
}

// ImportResult provides information about template import operation
type ImportResult struct {
	ImportedCount int      `json:"imported_count"`
	SkippedCount  int      `json:"skipped_count"`
	ErrorCount    int      `json:"error_count"`
	ImportedIDs   []string `json:"imported_ids"`
	Errors        []string `json:"errors,omitempty"`
}

// TemplateSearchResult represents a search result for templates
type TemplateSearchResult struct {
	Template    *Template `json:"template"`
	Relevance   float64   `json:"relevance"`
	MatchedTags []string  `json:"matched_tags,omitempty"`
	Matches     []string  `json:"matches,omitempty"`
}

// ConvertFromDB converts database model to domain model
func (t *Template) FromDB(dbTemplate db.Template, variables []db.TemplateVariable) {
	t.ID = dbTemplate.ID
	t.Name = dbTemplate.Name
	if dbTemplate.Description != nil {
		t.Description = *dbTemplate.Description
	}
	t.Category = dbTemplate.Category
	t.Content = dbTemplate.Content
	if dbTemplate.Language != nil {
		t.Language = *dbTemplate.Language
	}
	if dbTemplate.UseCount != nil {
		t.UseCount = *dbTemplate.UseCount
	}
	if dbTemplate.IsBuiltIn != nil {
		t.IsBuiltIn = *dbTemplate.IsBuiltIn
	}
	if dbTemplate.CreatedAt != nil {
		t.CreatedAt = *dbTemplate.CreatedAt
	}
	if dbTemplate.UpdatedAt != nil {
		t.UpdatedAt = *dbTemplate.UpdatedAt
	}

	// Convert variables
	t.Variables = make([]*TemplateVariable, len(variables))
	for i, dbVar := range variables {
		t.Variables[i] = &TemplateVariable{
			ID:         dbVar.ID,
			TemplateID: dbVar.TemplateID,
			Name:       dbVar.Name,
		}
		if dbVar.Description != nil {
			t.Variables[i].Description = *dbVar.Description
		}
		if dbVar.DefaultValue != nil {
			t.Variables[i].DefaultValue = *dbVar.DefaultValue
		}
		if dbVar.Required != nil {
			t.Variables[i].Required = *dbVar.Required
		}
		if dbVar.VariableType != nil {
			t.Variables[i].Type = VariableType(*dbVar.VariableType)
		}
		if dbVar.CreatedAt != nil {
			t.Variables[i].CreatedAt = *dbVar.CreatedAt
		}
	}
}

// ToDB converts domain model to database parameters
func (t *Template) ToDB() db.CreateTemplateParams {
	params := db.CreateTemplateParams{
		ID:       t.ID,
		Name:     t.Name,
		Category: t.Category,
		Content:  t.Content,
	}

	if t.Description != "" {
		params.Description = &t.Description
	}
	if t.Language != "" {
		params.Language = &t.Language
	}
	if t.IsBuiltIn {
		params.IsBuiltIn = &t.IsBuiltIn
	}

	return params
}
