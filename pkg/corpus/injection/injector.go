// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package injection

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/lancekrogers/guild/pkg/corpus/retrieval"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// InjectionPoint defines where context should be injected
type InjectionPoint int

const (
	InjectionSystemPrompt InjectionPoint = iota
	InjectionUserMessage
	InjectionToolContext
)

// String returns the string representation of an injection point
func (ip InjectionPoint) String() string {
	switch ip {
	case InjectionSystemPrompt:
		return "system_prompt"
	case InjectionUserMessage:
		return "user_message"
	case InjectionToolContext:
		return "tool_context"
	default:
		return "unknown"
	}
}

// ContextInjector handles smart context injection into prompts
type ContextInjector struct {
	retriever    retrieval.Retriever
	formatter    *ContextFormatter
	cache        *ContextCache
	maxTokens    int
	cacheEnabled bool
}

// NewContextInjector creates a new context injector
func NewContextInjector(retriever retrieval.Retriever, maxTokens int) (*ContextInjector, error) {
	if maxTokens <= 0 {
		maxTokens = 4000 // Default token limit
	}

	formatter := NewContextFormatter()
	cache := NewContextCache(time.Hour) // 1 hour cache TTL

	return &ContextInjector{
		retriever:    retriever,
		formatter:    formatter,
		cache:        cache,
		maxTokens:    maxTokens,
		cacheEnabled: true,
	}, nil
}

// SetCacheEnabled enables or disables caching
func (ci *ContextInjector) SetCacheEnabled(enabled bool) {
	ci.cacheEnabled = enabled
}

// InjectionRequest contains the parameters for context injection
type InjectionRequest struct {
	OriginalPrompt   Prompt
	Query            retrieval.Query
	InjectionPoints  []InjectionPoint
	MaxTokens        int
	CacheKey         string
	DisableCache     bool
}

// Prompt represents a structured prompt with different sections
type Prompt struct {
	System string `json:"system"`
	User   string `json:"user"`
	Tools  string `json:"tools"`
}

// InjectedPrompt contains the result of context injection
type InjectedPrompt struct {
	Original     Prompt                       `json:"original"`
	SystemPrompt string                       `json:"system_prompt"`
	UserMessage  string                       `json:"user_message"`
	ToolContext  string                       `json:"tool_context"`
	Contexts     map[InjectionPoint]string    `json:"contexts"`
	Metadata     map[string]interface{}       `json:"metadata"`
}

// FormattedContext contains formatted context for different injection points
type FormattedContext struct {
	System string `json:"system"`
	User   string `json:"user"`
	Tools  string `json:"tools"`
}

// InjectContext performs smart context injection into prompts
func (ci *ContextInjector) InjectContext(ctx context.Context, req InjectionRequest) (*InjectedPrompt, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("ContextInjector").
		WithOperation("InjectContext")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ContextInjector").
			WithOperation("InjectContext")
	}

	// Use provided max tokens or default
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = ci.maxTokens
	}

	// Generate cache key if not provided
	cacheKey := req.CacheKey
	if cacheKey == "" {
		cacheKey = ci.generateCacheKey(req)
	}

	// Check cache if enabled and not disabled for this request
	if ci.cacheEnabled && !req.DisableCache {
		if cached := ci.cache.Get(cacheKey); cached != nil {
			logger.Debug("Context injection cache hit")
			return cached, nil
		}
	}

	// Build retrieval query from request
	query := req.Query
	if query.Text == "" {
		query.Text = req.OriginalPrompt.User
	}

	// Retrieve relevant context
	docs, err := ci.retriever.Retrieve(ctx, query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "context retrieval failed").
			WithComponent("ContextInjector").
			WithOperation("InjectContext")
	}

	logger.Debug("Context retrieved", "retrieved_docs", len(docs))

	// Format context for injection
	formattedContext := ci.formatter.Format(docs, maxTokens)

	// Inject at appropriate points
	injected := ci.inject(req.OriginalPrompt, formattedContext, req.InjectionPoints)

	// Add metadata
	injected.Metadata = map[string]interface{}{
		"injection_timestamp": time.Now(),
		"documents_used":      len(docs),
		"cache_key":          cacheKey,
		"max_tokens":         maxTokens,
		"injection_points":   req.InjectionPoints,
	}

	// Cache result if enabled
	if ci.cacheEnabled && !req.DisableCache {
		ci.cache.Set(cacheKey, injected)
		logger.Debug("Context injection result cached")
	}

	logger.Info("Context injection completed successfully", "injection_points", len(req.InjectionPoints))

	return injected, nil
}

// inject performs the actual context injection into the prompt
func (ci *ContextInjector) inject(prompt Prompt, context FormattedContext, points []InjectionPoint) *InjectedPrompt {
	result := &InjectedPrompt{
		Original: prompt,
		Contexts: make(map[InjectionPoint]string),
	}

	// Copy original prompt as defaults
	result.SystemPrompt = prompt.System
	result.UserMessage = prompt.User
	result.ToolContext = prompt.Tools

	for _, point := range points {
		switch point {
		case InjectionSystemPrompt:
			result.SystemPrompt = ci.injectIntoSystem(prompt.System, context.System)
			result.Contexts[point] = context.System

		case InjectionUserMessage:
			result.UserMessage = ci.injectIntoUser(prompt.User, context.User)
			result.Contexts[point] = context.User

		case InjectionToolContext:
			result.ToolContext = context.Tools
			result.Contexts[point] = context.Tools
		}
	}

	return result
}

// injectIntoSystem injects context into the system prompt
func (ci *ContextInjector) injectIntoSystem(original, context string) string {
	if context == "" {
		return original
	}

	if original == "" {
		return context
	}

	// Insert context before the main system instruction
	return context + "\n\n" + original
}

// injectIntoUser injects context into the user message
func (ci *ContextInjector) injectIntoUser(original, context string) string {
	if context == "" {
		return original
	}

	if original == "" {
		return context
	}

	// Add context as a prefix to the user message
	return context + "\n\n---\n\n" + original
}

// generateCacheKey creates a cache key for the injection request
func (ci *ContextInjector) generateCacheKey(req InjectionRequest) string {
	// Create a hash based on the request content
	h := md5.New()
	h.Write([]byte(req.OriginalPrompt.System))
	h.Write([]byte(req.OriginalPrompt.User))
	h.Write([]byte(req.OriginalPrompt.Tools))
	h.Write([]byte(req.Query.Text))
	
	// Include injection points in the hash
	for _, point := range req.InjectionPoints {
		h.Write([]byte(point.String()))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// GenerateCacheKey generates a cache key for the given request
func (req *InjectionRequest) GenerateCacheKey() string {
	return req.CacheKey
}

// ContextFormatter handles formatting of retrieved context for injection
type ContextFormatter struct {
	templates map[string]*template.Template
}

// NewContextFormatter creates a new context formatter with default templates
func NewContextFormatter() *ContextFormatter {
	cf := &ContextFormatter{
		templates: make(map[string]*template.Template),
	}

	// Initialize default templates
	cf.initDefaultTemplates()
	return cf
}

// Format formats retrieved documents into context suitable for injection
func (cf *ContextFormatter) Format(docs []retrieval.RankedDocument, maxTokens int) FormattedContext {
	var system, user, tools strings.Builder
	tokenCount := 0

	// Group documents by type
	grouped := cf.groupByType(docs)

	// Format system context (high-level docs, architecture)
	for _, doc := range grouped["system"] {
		formatted := cf.formatSystemDoc(doc)
		tokens := cf.estimateTokens(formatted)

		if tokenCount+tokens > maxTokens {
			break
		}

		system.WriteString(formatted)
		system.WriteString("\n\n")
		tokenCount += tokens
	}

	// Format user context (specific examples, code snippets)
	for _, doc := range grouped["example"] {
		formatted := cf.formatExampleDoc(doc)
		tokens := cf.estimateTokens(formatted)

		if tokenCount+tokens > maxTokens {
			break
		}

		user.WriteString(formatted)
		user.WriteString("\n\n")
		tokenCount += tokens
	}

	// Format tool context (API docs, schemas)
	for _, doc := range grouped["tool"] {
		formatted := cf.formatToolDoc(doc)
		tokens := cf.estimateTokens(formatted)

		if tokenCount+tokens > maxTokens {
			break
		}

		tools.WriteString(formatted)
		tools.WriteString("\n\n")
		tokenCount += tokens
	}

	return FormattedContext{
		System: system.String(),
		User:   user.String(),
		Tools:  tools.String(),
	}
}

// groupByType groups documents by their intended usage type
func (cf *ContextFormatter) groupByType(docs []retrieval.RankedDocument) map[string][]retrieval.RankedDocument {
	grouped := map[string][]retrieval.RankedDocument{
		"system":  make([]retrieval.RankedDocument, 0),
		"example": make([]retrieval.RankedDocument, 0),
		"tool":    make([]retrieval.RankedDocument, 0),
	}

	for _, doc := range docs {
		docType := cf.inferDocumentType(doc)
		if group, exists := grouped[docType]; exists {
			grouped[docType] = append(group, doc)
		} else {
			// Default to example if unknown type
			grouped["example"] = append(grouped["example"], doc)
		}
	}

	return grouped
}

// inferDocumentType determines the best injection context for a document
func (cf *ContextFormatter) inferDocumentType(doc retrieval.RankedDocument) string {
	// Check metadata for explicit type
	if docType, ok := doc.Metadata["type"].(string); ok {
		switch strings.ToLower(docType) {
		case "architecture":
			return "system"
		case "api":
			return "tool"
		case "example":
			return "example"
		default:
			// Unknown types default to example
			return "example"
		}
	}

	// If no metadata type, default to example
	return "example"
}

// formatSystemDoc formats a document for system context injection
func (cf *ContextFormatter) formatSystemDoc(doc retrieval.RankedDocument) string {
	title := cf.getDocumentTitle(doc)
	return "## " + title + "\n" + doc.Content
}

// formatExampleDoc formats a document for user context injection
func (cf *ContextFormatter) formatExampleDoc(doc retrieval.RankedDocument) string {
	title := cf.getDocumentTitle(doc)
	score := doc.FinalScore
	return "### Example: " + title + " (Relevance: " + formatFloat(score) + ")\n" + doc.Content
}

// formatToolDoc formats a document for tool context injection
func (cf *ContextFormatter) formatToolDoc(doc retrieval.RankedDocument) string {
	title := cf.getDocumentTitle(doc)
	return "#### " + title + "\n```\n" + doc.Content + "\n```"
}

// getDocumentTitle extracts or generates a title for the document
func (cf *ContextFormatter) getDocumentTitle(doc retrieval.RankedDocument) string {
	if title, ok := doc.Metadata["title"].(string); ok && title != "" {
		return title
	}

	if source, ok := doc.Metadata["source"].(string); ok && source != "" {
		return source
	}

	return "Document " + doc.ID
}

// estimateTokens provides a rough estimate of token count for text
func (cf *ContextFormatter) estimateTokens(text string) int {
	// Simple word-based estimation for testing consistency
	words := strings.Fields(text)
	return len(words)
}

// initDefaultTemplates initializes default formatting templates
func (cf *ContextFormatter) initDefaultTemplates() {
	// System prompt template
	systemTemplate := `# Relevant Documentation

The following documentation may help you understand the context and requirements:

{{.Content}}

---
`

	// User message template
	userTemplate := `## Reference Examples

Here are some relevant examples that might help:

{{.Content}}

`

	// Tool context template
	toolTemplate := `### API Reference

{{.Content}}

`

	cf.templates["system"] = template.Must(template.New("system").Parse(systemTemplate))
	cf.templates["user"] = template.Must(template.New("user").Parse(userTemplate))
	cf.templates["tool"] = template.Must(template.New("tool").Parse(toolTemplate))
}

// formatFloat formats a float64 to 2 decimal places
func formatFloat(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
}