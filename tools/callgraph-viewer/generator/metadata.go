package generator

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// MetadataExtractor extracts metadata from SVG callgraph files
type MetadataExtractor struct {
	BaseDir   string // Base directory where SVGs are located
	OutputDir string // Output directory for the viewer
}

// Extract parses an SVG file and extracts metadata
func (e *MetadataExtractor) Extract(ctx context.Context, svgPath string) (*GraphMetadata, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	fullPath := filepath.Join(e.BaseDir, svgPath)

	// Get file stats for size and modification time
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "stat failed").
			WithDetails("path", fullPath)
	}

	// Extract ID from filename (without .svg extension)
	filename := filepath.Base(svgPath)
	id := strings.TrimSuffix(filename, ".svg")

	// SVGs will be copied to viewer/svgs/ during generation
	// Simple local path from index.html
	relPath := filepath.Join("svgs", filename)

	meta := &GraphMetadata{
		ID:           id,
		FilePath:     relPath,
		SizeBytes:    info.Size(),
		LastModified: info.ModTime(),
	}

	// Extract category (pkg vs internal)
	if strings.HasPrefix(id, "pkg-") {
		meta.Category = "pkg"
		meta.Domain = extractDomain(strings.TrimPrefix(id, "pkg-"))
	} else if strings.HasPrefix(id, "internal-") {
		meta.Category = "internal"
		meta.Domain = extractDomain(strings.TrimPrefix(id, "internal-"))
	} else {
		meta.Category = "other"
		meta.Domain = "other"
	}

	// Categorize by size
	meta.SizeCategory = categorizeSize(meta.SizeBytes)

	// Generate title from ID (convert hyphens to slashes)
	meta.Title = generateTitle(id)

	// TODO: Optionally parse SVG to count functions
	// For now, leave as 0
	meta.Functions = 0

	return meta, nil
}

// extractDomain extracts the domain from the name
// Example: "agents-core" -> "agents", "storage-memory" -> "storage"
func extractDomain(name string) string {
	parts := strings.Split(name, "-")
	if len(parts) > 0 {
		return parts[0]
	}
	return "other"
}

// categorizeSize categorizes a file size into small/medium/large
func categorizeSize(bytes int64) string {
	const (
		smallThreshold  = 50 * 1024   // 50 KB
		mediumThreshold = 500 * 1024  // 500 KB
	)

	if bytes < smallThreshold {
		return "small"
	} else if bytes < mediumThreshold {
		return "medium"
	}
	return "large"
}

// generateTitle converts an ID to a display title
// Example: "pkg-agents-core" -> "pkg/agents/core"
func generateTitle(id string) string {
	// Replace hyphens with slashes, but only if it starts with pkg- or internal-
	if strings.HasPrefix(id, "pkg-") {
		return "pkg/" + strings.ReplaceAll(strings.TrimPrefix(id, "pkg-"), "-", "/")
	} else if strings.HasPrefix(id, "internal-") {
		return "internal/" + strings.ReplaceAll(strings.TrimPrefix(id, "internal-"), "-", "/")
	}
	// For other files (like daemon-lifecycle), keep as-is
	return id
}
