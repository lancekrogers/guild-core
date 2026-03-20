package generator

import "time"

// GraphMetadata represents metadata for a single callgraph SVG file
type GraphMetadata struct {
	ID           string    `json:"id"`            // Sanitized filename (e.g., "pkg-agents-core")
	Title        string    `json:"title"`         // Display name (e.g., "pkg/agents/core")
	FilePath     string    `json:"filePath"`      // Relative path from viewer/index.html
	Category     string    `json:"category"`      // "pkg" or "internal"
	Domain       string    `json:"domain"`        // Extracted from path (e.g., "agents", "storage")
	SizeBytes    int64     `json:"sizeBytes"`     // File size in bytes
	SizeCategory string    `json:"sizeCategory"`  // "small" (<50KB), "medium" (<500KB), "large" (>=500KB)
	Functions    int       `json:"functions"`     // Count of functions (optional, from SVG nodes)
	LastModified time.Time `json:"lastModified"`  // File modification time
}

// ViewerData represents the complete data structure passed to HTML templates
type ViewerData struct {
	Graphs      []GraphMetadata            `json:"graphs"`      // All graphs, sorted alphabetically
	Categories  map[string][]GraphMetadata `json:"categories"`  // Graphs grouped by category (pkg/internal)
	Domains     map[string][]GraphMetadata `json:"domains"`     // Graphs grouped by domain
	GeneratedAt time.Time                  `json:"generatedAt"` // Timestamp when viewer was generated
}
