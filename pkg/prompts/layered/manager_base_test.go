package layered_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/internal/prompts/layered"
)

// TestManagerInterface ensures the interface is properly defined
func TestManagerInterface(t *testing.T) {
	// This test ensures that our interface contracts are maintained
	var _ layered.Manager = (*testManager)(nil)
	var _ layered.Context = (*testContext)(nil)
	var _ layered.Formatter = (*testFormatter)(nil)
	var _ layered.Registry = (*testRegistry)(nil)
}

// Mock implementations for testing

type testManager struct{}

func (m *testManager) GetSystemPrompt(ctx context.Context, role string, domain string) (string, error) {
	if role == "manager" && domain == "web-app" {
		return "You are a Guild Master for web applications...", nil
	}
	return "", layered.ErrPromptNotFound
}

func (m *testManager) GetTemplate(ctx context.Context, templateName string) (string, error) {
	if templateName == "task-format" {
		return "**Tasks Generated**:\n- {ID}: {Title}", nil
	}
	return "", layered.ErrTemplateNotFound
}

func (m *testManager) FormatContext(ctx context.Context, context layered.Context) (string, error) {
	return "<context>formatted</context>", nil
}

func (m *testManager) ListRoles(ctx context.Context) ([]string, error) {
	return []string{"manager", "developer", "reviewer"}, nil
}

func (m *testManager) ListDomains(ctx context.Context, role string) ([]string, error) {
	if role == "manager" {
		return []string{"web-app", "cli-tool", "library"}, nil
	}
	return []string{}, nil
}

type testContext struct {
	commissionID    string
	commissionTitle string
	currentTask     layered.TaskContext
	sections        []layered.Section
	relatedTasks    []layered.TaskContext
}

func (c *testContext) GetCommissionID() string {
	return c.commissionID
}

func (c *testContext) GetCommissionTitle() string {
	return c.commissionTitle
}

func (c *testContext) GetCurrentTask() layered.TaskContext {
	return c.currentTask
}

func (c *testContext) GetRelevantSections() []layered.Section {
	return c.sections
}

func (c *testContext) GetRelatedTasks() []layered.TaskContext {
	return c.relatedTasks
}

type testFormatter struct{}

func (f *testFormatter) FormatAsXML(ctx layered.Context) (string, error) {
	return "<context>xml</context>", nil
}

func (f *testFormatter) FormatAsMarkdown(ctx layered.Context) (string, error) {
	return "# Context\nmarkdown", nil
}

func (f *testFormatter) OptimizeForTokens(content string, maxTokens int) (string, error) {
	if len(content) > maxTokens*4 { // Rough approximation
		return content[:maxTokens*4], nil
	}
	return content, nil
}

type testRegistry struct {
	prompts   map[string]string
	templates map[string]string
}

func newTestRegistry() *testRegistry {
	return &testRegistry{
		prompts:   make(map[string]string),
		templates: make(map[string]string),
	}
}

func (r *testRegistry) RegisterPrompt(role, domain, prompt string) error {
	key := role + ":" + domain
	r.prompts[key] = prompt
	return nil
}

func (r *testRegistry) RegisterTemplate(name, template string) error {
	r.templates[name] = template
	return nil
}

func (r *testRegistry) GetPrompt(role, domain string) (string, error) {
	key := role + ":" + domain
	if prompt, ok := r.prompts[key]; ok {
		return prompt, nil
	}
	return "", layered.ErrPromptNotFound
}

func (r *testRegistry) GetTemplate(name string) (string, error) {
	if template, ok := r.templates[name]; ok {
		return template, nil
	}
	return "", layered.ErrTemplateNotFound
}

// Actual tests

func TestMockManager(t *testing.T) {
	ctx := context.Background()
	manager := &testManager{}

	t.Run("GetSystemPrompt", func(t *testing.T) {
		// Test existing prompt
		prompt, err := manager.GetSystemPrompt(ctx, "manager", "web-app")
		require.NoError(t, err)
		assert.Contains(t, prompt, "Guild Master")

		// Test non-existent prompt
		_, err = manager.GetSystemPrompt(ctx, "unknown", "unknown")
		assert.ErrorIs(t, err, layered.ErrPromptNotFound)
	})

	t.Run("GetTemplate", func(t *testing.T) {
		// Test existing template
		template, err := manager.GetTemplate(ctx, "task-format")
		require.NoError(t, err)
		assert.Contains(t, template, "Tasks Generated")

		// Test non-existent template
		_, err = manager.GetTemplate(ctx, "unknown")
		assert.ErrorIs(t, err, layered.ErrTemplateNotFound)
	})

	t.Run("ListRoles", func(t *testing.T) {
		roles, err := manager.ListRoles(ctx)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"manager", "developer", "reviewer"}, roles)
	})

	t.Run("ListDomains", func(t *testing.T) {
		domains, err := manager.ListDomains(ctx, "manager")
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"web-app", "cli-tool", "library"}, domains)

		// Test role with no domains
		domains, err = manager.ListDomains(ctx, "unknown")
		require.NoError(t, err)
		assert.Empty(t, domains)
	})
}

func TestContext(t *testing.T) {
	ctx := &testContext{
		commissionID:    "test-commission",
		commissionTitle: "Build Test System",
		currentTask: layered.TaskContext{
			ID:          "TASK-001",
			Title:       "Implement feature",
			Description: "Implement the test feature",
		},
		sections: []layered.Section{
			{
				Level:   1,
				Path:    "1",
				Title:   "Overview",
				Content: "System overview content",
			},
		},
		relatedTasks: []layered.TaskContext{
			{
				ID:    "TASK-002",
				Title: "Related task",
			},
		},
	}

	assert.Equal(t, "test-commission", ctx.GetCommissionID())
	assert.Equal(t, "Build Test System", ctx.GetCommissionTitle())
	assert.Equal(t, "TASK-001", ctx.GetCurrentTask().ID)
	assert.Len(t, ctx.GetRelevantSections(), 1)
	assert.Len(t, ctx.GetRelatedTasks(), 1)
}

func TestFormatter(t *testing.T) {
	formatter := &testFormatter{}
	ctx := &testContext{}

	t.Run("FormatAsXML", func(t *testing.T) {
		result, err := formatter.FormatAsXML(ctx)
		require.NoError(t, err)
		assert.Equal(t, "<context>xml</context>", result)
	})

	t.Run("FormatAsMarkdown", func(t *testing.T) {
		result, err := formatter.FormatAsMarkdown(ctx)
		require.NoError(t, err)
		assert.Contains(t, result, "# Context")
	})

	t.Run("OptimizeForTokens", func(t *testing.T) {
		// Test content that fits
		content := "Short content"
		result, err := formatter.OptimizeForTokens(content, 100)
		require.NoError(t, err)
		assert.Equal(t, content, result)

		// Test content that needs truncation
		longContent := string(make([]byte, 1000))
		result, err = formatter.OptimizeForTokens(longContent, 100)
		require.NoError(t, err)
		assert.Len(t, result, 400) // 100 * 4
	})
}

func TestRegistry(t *testing.T) {
	registry := newTestRegistry()

	t.Run("RegisterAndGetPrompt", func(t *testing.T) {
		// Register a prompt
		err := registry.RegisterPrompt("manager", "web-app", "Test prompt")
		require.NoError(t, err)

		// Get the prompt
		prompt, err := registry.GetPrompt("manager", "web-app")
		require.NoError(t, err)
		assert.Equal(t, "Test prompt", prompt)

		// Try to get non-existent prompt
		_, err = registry.GetPrompt("unknown", "unknown")
		assert.ErrorIs(t, err, layered.ErrPromptNotFound)
	})

	t.Run("RegisterAndGetTemplate", func(t *testing.T) {
		// Register a template
		err := registry.RegisterTemplate("test-template", "Template content")
		require.NoError(t, err)

		// Get the template
		template, err := registry.GetTemplate("test-template")
		require.NoError(t, err)
		assert.Equal(t, "Template content", template)

		// Try to get non-existent template
		_, err = registry.GetTemplate("unknown")
		assert.ErrorIs(t, err, layered.ErrTemplateNotFound)
	})
}