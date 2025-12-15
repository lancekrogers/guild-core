// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Graph represents document connections
type Graph struct {
	// Nodes maps from document names to their outgoing links
	Nodes map[string][]string `json:"nodes"`

	// Backlinks maps from document names to documents that link to them
	Backlinks map[string][]string `json:"backlinks"`

	// Tags maps from tag names to documents that have that tag
	Tags map[string][]string `json:"tags"`

	// TagLinks maps from tags to other related tags
	TagLinks map[string][]string `json:"tag_links"`

	// Edges is a flat list of all connections for visualization
	Edges []GraphEdge `json:"edges"`
}

// GraphEdge represents a directed edge in the graph
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// NewGraph creates a new empty graph
func NewGraph() *Graph {
	return &Graph{
		Nodes:     make(map[string][]string),
		Backlinks: make(map[string][]string),
		Tags:      make(map[string][]string),
		TagLinks:  make(map[string][]string),
		Edges:     []GraphEdge{},
	}
}

// GetBacklinks returns a list of documents that link to the specified document
func (g *Graph) GetBacklinks(docName string) []string {
	return g.Backlinks[docName]
}

// GetDocumentsWithTag returns a list of documents that have the specified tag
func (g *Graph) GetDocumentsWithTag(tag string) []string {
	return g.Tags[tag]
}

// BuildGraph creates a graph from corpus documents
func BuildGraph(ctx context.Context, cfg Config) (*Graph, error) {
	if cfg.CorpusPath == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "corpus", nil).WithComponent("build_graph").WithOperation("corpus location not specified")
	}

	graph := NewGraph()

	// Get all documents
	docs, err := List(ctx, cfg)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("build_graph").WithOperation("failed to list corpus documents")
	}

	// First pass: collect all nodes and their outgoing links
	for _, path := range docs {
		doc, err := Load(ctx, path)
		if err != nil {
			continue // Skip documents that can't be loaded
		}

		// Extract the document name (without extension)
		docName := filepath.Base(path)
		docName = strings.TrimSuffix(docName, filepath.Ext(docName))
		docName = strings.ToLower(docName)

		// Add node with its outgoing links
		graph.Nodes[docName] = doc.Links

		// Add tags
		for _, tag := range doc.Tags {
			if graph.Tags[tag] == nil {
				graph.Tags[tag] = []string{}
			}
			graph.Tags[tag] = append(graph.Tags[tag], docName)
		}
	}

	// Second pass: build backlinks
	for docName, links := range graph.Nodes {
		for _, link := range links {
			// Add this document as a backlink to the linked document
			if graph.Backlinks[link] == nil {
				graph.Backlinks[link] = []string{}
			}
			graph.Backlinks[link] = append(graph.Backlinks[link], docName)
		}
	}

	return graph, nil
}

// SaveGraph serializes and saves the graph to a JSON file
func SaveGraph(ctx context.Context, graph *Graph, cfg Config) error {
	if graph == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "corpus", nil).WithComponent("save_graph").WithOperation("graph cannot be nil")
	}

	// Ensure the graph directory exists
	graphDir := filepath.Join(cfg.CorpusPath, GraphDirName)
	if err := os.MkdirAll(graphDir, 0o755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("save_graph").WithOperation("failed to create graph directory")
	}

	// Serialize the graph to JSON
	graphPath := filepath.Join(graphDir, "links.json")
	data, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("save_graph").WithOperation("failed to serialize graph")
	}

	// Write to file
	if err := os.WriteFile(graphPath, data, 0o644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("save_graph").WithOperation("failed to save graph")
	}

	return nil
}

// LoadGraph loads a graph from JSON file
func LoadGraph(ctx context.Context, cfg Config) (*Graph, error) {
	// Check for graph file
	graphPath := filepath.Join(cfg.CorpusPath, GraphDirName, "links.json")
	data, err := os.ReadFile(graphPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If the graph doesn't exist, build a new one
			return BuildGraph(ctx, cfg)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("load_graph").WithOperation("failed to read graph file")
	}

	// Deserialize the graph
	var graph Graph
	if err := json.Unmarshal(data, &graph); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("load_graph").WithOperation("failed to parse graph")
	}

	return &graph, nil
}

// UpdateGraph rebuilds and saves the graph
func UpdateGraph(ctx context.Context, cfg Config) error {
	graph, err := BuildGraph(ctx, cfg)
	if err != nil {
		return err
	}

	return SaveGraph(ctx, graph, cfg)
}

// FindOrphanNodes returns documents that have no incoming links
func FindOrphanNodes(graph *Graph) []string {
	if graph == nil {
		return nil
	}

	var orphans []string
	for node := range graph.Nodes {
		if len(graph.Backlinks[node]) == 0 {
			orphans = append(orphans, node)
		}
	}

	return orphans
}

// FindStronglyConnected returns groups of documents that form clusters
func FindStronglyConnected(graph *Graph, minSize int) [][]string {
	if graph == nil {
		return nil
	}

	// This is a simplified implementation for terminal display
	// A real implementation would use Tarjan's algorithm or similar

	// For now, we'll just group documents by common tags
	clusters := make(map[string][]string)

	// Group by tags
	for tag, docs := range graph.Tags {
		if len(docs) >= minSize {
			clusters[tag] = docs
		}
	}

	// Convert to slice of slices
	result := make([][]string, 0, len(clusters))
	for _, docs := range clusters {
		result = append(result, docs)
	}

	return result
}

// ExportGraphDOT generates a DOT format representation for Graphviz
func ExportGraphDOT(ctx context.Context, graph *Graph, cfg Config) error {
	if graph == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "corpus", nil).WithComponent("save_graph").WithOperation("graph cannot be nil")
	}

	// Create DOT file content
	var sb strings.Builder
	sb.WriteString("digraph CorpusGraph {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=filled, fillcolor=lightskyblue];\n\n")

	// Add nodes and edges
	for node, links := range graph.Nodes {
		// Clean node name for DOT
		cleanNode := strings.ReplaceAll(node, "-", "_")
		cleanNode = strings.ReplaceAll(cleanNode, " ", "_")

		for _, link := range links {
			// Clean link name for DOT
			cleanLink := strings.ReplaceAll(link, "-", "_")
			cleanLink = strings.ReplaceAll(cleanLink, " ", "_")

			sb.WriteString(fmt.Sprintf("  %s -> %s;\n", cleanNode, cleanLink))
		}
	}

	sb.WriteString("}\n")

	// Write to file
	graphDir := filepath.Join(cfg.CorpusPath, GraphDirName)
	if err := os.MkdirAll(graphDir, 0o755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("save_graph").WithOperation("failed to create graph directory")
	}

	dotPath := filepath.Join(graphDir, "corpus.dot")
	if err := os.WriteFile(dotPath, []byte(sb.String()), 0o644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("export_to_dot").WithOperation("failed to save DOT file")
	}

	return nil
}
