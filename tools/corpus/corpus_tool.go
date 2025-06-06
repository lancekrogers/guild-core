package corpus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/corpus"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools"
)

// CorpusTool provides functionality for agents to interact with the corpus system
type CorpusTool struct {
	*tools.BaseTool
	config corpus.Config
}

// Input defines the input parameters for the corpus tool
type Input struct {
	Action    string   `json:"action"`    // save, load, search, list, delete
	Title     string   `json:"title"`     // for save, load, delete
	Content   string   `json:"content"`   // for save
	Tags      []string `json:"tags"`      // for save, search
	Source    string   `json:"source"`    // for save
	Query     string   `json:"query"`     // for search
	GuildID   string   `json:"guild_id"`  // for save
	AgentID   string   `json:"agent_id"`  // for save
	Limit     int      `json:"limit"`     // for list, search
	BuildGraph bool     `json:"graph"`     // whether to build and include graph data
}

// NewCorpusTool creates a new corpus tool
func NewCorpusTool(config corpus.Config) *CorpusTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "The action to perform: save, load, search, list, or delete",
				"enum":        []string{"save", "load", "search", "list", "delete", "graph"},
			},
			"title": map[string]interface{}{
				"type":        "string",
				"description": "Document title for save, load, and delete operations",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Document content for save operations",
			},
			"tags": map[string]interface{}{
				"type":        "array",
				"description": "Document tags for save and search operations",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"source": map[string]interface{}{
				"type":        "string",
				"description": "Document source for save operations",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query for search operations",
			},
			"guild_id": map[string]interface{}{
				"type":        "string",
				"description": "Guild ID for save operations",
			},
			"agent_id": map[string]interface{}{
				"type":        "string",
				"description": "Agent ID for save operations",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return for list and search operations",
			},
			"graph": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to build and include graph data",
			},
		},
		"required": []string{"action"},
	}

	examples := []string{
		`{"action": "save", "title": "Machine Learning Basics", "content": "# Machine Learning\n\nThis is an introduction to machine learning...", "tags": ["AI", "learning"], "source": "research"}`,
		`{"action": "load", "title": "Machine Learning Basics"}`,
		`{"action": "search", "query": "machine learning", "limit": 5}`,
		`{"action": "list", "limit": 10}`,
		`{"action": "delete", "title": "Outdated Document"}`,
		`{"action": "graph"}`,
	}

	baseTool := tools.NewBaseTool(
		"corpus",
		"Store and retrieve documents in the Guild research corpus",
		schema,
		"knowledge",
		false,
		examples,
	)

	return &CorpusTool{
		BaseTool: baseTool,
		config:   config,
	}
}

// Execute runs the corpus tool with the given input
func (t *CorpusTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params Input
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "corpus_tool").WithComponent("execute").WithOperation("invalid input"), nil), nil
	}

	// Execute the appropriate action
	switch params.Action {
	case "save":
		return t.saveDocument(ctx, params)
	case "load":
		return t.loadDocument(ctx, params)
	case "search":
		return t.searchDocuments(ctx, params)
	case "list":
		return t.listDocuments(ctx, params)
	case "delete":
		return t.deleteDocument(ctx, params)
	case "graph":
		return t.getGraph(ctx, params)
	default:
		return tools.NewToolResult("", nil, gerror.New(gerror.ErrCodeInvalidInput, "corpus_tool", "execute", "unknown action: %s", params.Action), nil), nil
	}
}

// saveDocument saves a document to the corpus
func (t *CorpusTool) saveDocument(ctx context.Context, params Input) (*tools.ToolResult, error) {
	// Validate required parameters
	if params.Title == "" {
		return tools.NewToolResult("", nil, gerror.New(gerror.ErrCodeInvalidInput, "corpus_tool", nil).WithComponent("save_document").WithOperation("title is required for save action"), nil), nil
	}

	// Create a new document
	doc := corpus.CorpusDoc{
		Title:     params.Title,
		Source:    params.Source,
		Tags:      params.Tags,
		Body:      params.Content,
		GuildID:   params.GuildID,
		AgentID:   params.AgentID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save the document
	err := corpus.Save(ctx, &doc, t.config)
	if err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus_tool").WithComponent("save_document").WithOperation("failed to save document"), nil), nil
	}

	metadata := map[string]string{
		"title":    doc.Title,
		"filePath": doc.FilePath,
	}

	return tools.NewToolResult(fmt.Sprintf("Document '%s' saved successfully", doc.Title), metadata, nil, nil), nil
}

// loadDocument loads a document from the corpus
func (t *CorpusTool) loadDocument(ctx context.Context, params Input) (*tools.ToolResult, error) {
	// Validate required parameters
	if params.Title == "" {
		return tools.NewToolResult("", nil, gerror.New(gerror.ErrCodeInvalidInput, "corpus_tool", nil).WithComponent("load_document").WithOperation("title is required for load action"), nil), nil
	}

	// Get all document paths
	paths, err := corpus.List(ctx, t.config)
	if err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus_tool").WithComponent("load_document").WithOperation("failed to list documents"), nil), nil
	}

	// Find the document with the matching title
	var targetDoc *corpus.CorpusDoc
	for _, path := range paths {
		doc, err := corpus.Load(ctx, path)
		if err != nil {
			continue // Skip documents that can't be loaded
		}
		if strings.EqualFold(doc.Title, params.Title) {
			targetDoc = doc
			break
		}
	}

	if targetDoc == nil {
		return tools.NewToolResult("", nil, gerror.New(gerror.ErrCodeNotFound, "corpus_tool", "load_document", "document not found: %s", params.Title), nil), nil
	}

	// We already have the loaded document
	doc := targetDoc

	// Create metadata
	metadata := map[string]string{
		"title":     doc.Title,
		"filePath":  doc.FilePath,
		"createdAt": doc.CreatedAt.Format(time.RFC3339),
	}
	if !doc.UpdatedAt.IsZero() {
		metadata["updatedAt"] = doc.UpdatedAt.Format(time.RFC3339)
	}
	if doc.Source != "" {
		metadata["source"] = doc.Source
	}
	if len(doc.Tags) > 0 {
		metadata["tags"] = strings.Join(doc.Tags, ", ")
	}

	// Prepare extra data with full document structure
	extraData := map[string]interface{}{
		"doc": doc,
	}

	return tools.NewToolResult(doc.Body, metadata, nil, extraData), nil
}

// searchDocuments searches for documents in the corpus
func (t *CorpusTool) searchDocuments(ctx context.Context, params Input) (*tools.ToolResult, error) {
	// Get all document paths
	paths, err := corpus.List(ctx, t.config)
	if err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus_tool").WithComponent("load_document").WithOperation("failed to list documents"), nil), nil
	}

	// Apply search criteria
	var results []corpus.CorpusDoc
	for _, path := range paths {
		// Load the document
		doc, err := corpus.Load(ctx, path)
		if err != nil {
			continue // Skip documents that can't be loaded
		}

		// Search by query in title or body
		if params.Query != "" {
			if !(strings.Contains(strings.ToLower(doc.Title), strings.ToLower(params.Query)) ||
				strings.Contains(strings.ToLower(doc.Body), strings.ToLower(params.Query))) {
				continue
			}
		}

		// Filter by tags if provided
		if len(params.Tags) > 0 {
			// Check if the document has at least one of the requested tags
			hasTag := false
			for _, tag := range params.Tags {
				for _, docTag := range doc.Tags {
					if strings.EqualFold(tag, docTag) {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		// Add the document to the results
		results = append(results, *doc)
	}

	// Apply limit
	if params.Limit > 0 && len(results) > params.Limit {
		results = results[:params.Limit]
	}

	// Load full content for each result
	for i, doc := range results {
		fullDoc, err := corpus.Load(ctx, doc.FilePath)
		if err != nil {
			// Skip documents that can't be loaded
			continue
		}
		results[i] = *fullDoc
	}

	// Build graph if requested
	var graph *corpus.Graph
	if params.BuildGraph {
		graph, _ = corpus.BuildGraph(ctx, t.config)
	}

	// Create a summary of the results
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Found %d documents matching the search criteria:\n\n", len(results)))
	for i, doc := range results {
		summary.WriteString(fmt.Sprintf("%d. %s\n", i+1, doc.Title))
		if len(doc.Tags) > 0 {
			summary.WriteString(fmt.Sprintf("   Tags: %s\n", strings.Join(doc.Tags, ", ")))
		}
		summary.WriteString("\n")
	}

	// Prepare extra data with full search results
	extraData := map[string]interface{}{
		"results": results,
	}
	if graph != nil {
		extraData["graph"] = graph
	}

	return tools.NewToolResult(summary.String(), nil, nil, extraData), nil
}

// listDocuments lists documents in the corpus
func (t *CorpusTool) listDocuments(ctx context.Context, params Input) (*tools.ToolResult, error) {
	// Get all document paths
	paths, err := corpus.List(ctx, t.config)
	if err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus_tool").WithComponent("load_document").WithOperation("failed to list documents"), nil), nil
	}

	// Apply limit to paths
	if params.Limit > 0 && len(paths) > params.Limit {
		paths = paths[:params.Limit]
	}

	// Load documents
	var docs []corpus.CorpusDoc
	for _, path := range paths {
		doc, err := corpus.Load(ctx, path)
		if err != nil {
			continue // Skip documents that can't be loaded
		}
		docs = append(docs, *doc)
	}

	// Build graph if requested
	var graph *corpus.Graph
	if params.BuildGraph {
		graph, _ = corpus.BuildGraph(ctx, t.config)
	}

	// Create a summary of the documents
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Corpus contains %d documents:\n\n", len(docs)))
	for i, doc := range docs {
		summary.WriteString(fmt.Sprintf("%d. %s\n", i+1, doc.Title))
		if len(doc.Tags) > 0 {
			summary.WriteString(fmt.Sprintf("   Tags: %s\n", strings.Join(doc.Tags, ", ")))
		}
		summary.WriteString(fmt.Sprintf("   Created: %s\n", doc.CreatedAt.Format("2006-01-02 15:04:05")))
		if !doc.UpdatedAt.IsZero() && doc.UpdatedAt != doc.CreatedAt {
			summary.WriteString(fmt.Sprintf("   Updated: %s\n", doc.UpdatedAt.Format("2006-01-02 15:04:05")))
		}
		summary.WriteString("\n")
	}

	// Prepare extra data with full document list
	extraData := map[string]interface{}{
		"docs": docs,
	}
	if graph != nil {
		extraData["graph"] = graph
	}

	return tools.NewToolResult(summary.String(), nil, nil, extraData), nil
}

// deleteDocument deletes a document from the corpus
func (t *CorpusTool) deleteDocument(ctx context.Context, params Input) (*tools.ToolResult, error) {
	// Validate required parameters
	if params.Title == "" {
		return tools.NewToolResult("", nil, gerror.New(gerror.ErrCodeInvalidInput, "corpus_tool", nil).WithComponent("delete_document").WithOperation("title is required for delete action"), nil), nil
	}

	// Get all document paths
	paths, err := corpus.List(ctx, t.config)
	if err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus_tool").WithComponent("load_document").WithOperation("failed to list documents"), nil), nil
	}

	// Find the document with the matching title
	var targetDoc *corpus.CorpusDoc
	for _, path := range paths {
		doc, err := corpus.Load(ctx, path)
		if err != nil {
			continue // Skip documents that can't be loaded
		}
		if strings.EqualFold(doc.Title, params.Title) {
			targetDoc = doc
			break
		}
	}

	if targetDoc == nil {
		return tools.NewToolResult("", nil, gerror.New(gerror.ErrCodeNotFound, "corpus_tool", "load_document", "document not found: %s", params.Title), nil), nil
	}

	// Delete the document
	err = corpus.Delete(ctx, targetDoc.FilePath)
	if err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus_tool").WithComponent("delete_document").WithOperation("failed to delete document"), nil), nil
	}

	return tools.NewToolResult(fmt.Sprintf("Document '%s' deleted successfully", params.Title), nil, nil, nil), nil
}

// getGraph builds and returns the document relationship graph
func (t *CorpusTool) getGraph(ctx context.Context, params Input) (*tools.ToolResult, error) {
	// Build the graph
	graph, err := corpus.BuildGraph(ctx, t.config)
	if err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus_tool").WithComponent("get_graph").WithOperation("failed to build graph"), nil), nil
	}

	// Create a summary of the graph
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Corpus Graph: %d documents, %d connections\n\n", len(graph.Nodes), len(graph.Edges)))
	
	// Show most connected documents
	type docConnection struct {
		title string
		count int
	}
	
	connections := make(map[string]int)
	for _, edge := range graph.Edges {
		connections[edge.From]++
		connections[edge.To]++
	}
	
	var sorted []docConnection
	for title, count := range connections {
		sorted = append(sorted, docConnection{title, count})
	}
	
	// Sort by connection count (simple bubble sort)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].count < sorted[j+1].count {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	// Show top 5 most connected documents
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	
	if limit > 0 {
		summary.WriteString("Most connected documents:\n")
		for i := 0; i < limit; i++ {
			summary.WriteString(fmt.Sprintf("%d. %s (%d connections)\n", i+1, sorted[i].title, sorted[i].count))
		}
	}

	// Prepare extra data with full graph
	extraData := map[string]interface{}{
		"graph": graph,
	}

	return tools.NewToolResult(summary.String(), nil, nil, extraData), nil
}