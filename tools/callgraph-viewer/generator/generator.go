package generator

import (
	"context"
	"encoding/json"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Generator orchestrates the callgraph viewer generation
type Generator struct {
	Scanner   *Scanner
	Extractor *MetadataExtractor
	Template  *template.Template
	OutputDir string
	StaticFS  io.Reader // For copying static assets
}

// Generate runs the complete generation process
func (g *Generator) Generate(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// 1. Scan for SVG files
	svgPaths, err := g.Scanner.Scan(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "scan failed")
	}

	// 2. Extract metadata for each SVG
	var graphs []GraphMetadata
	for _, path := range svgPaths {
		meta, err := g.Extractor.Extract(ctx, path)
		if err != nil {
			// Log warning but continue with other files
			continue
		}
		graphs = append(graphs, *meta)
	}

	// 3. Sort graphs alphabetically by title
	sort.Slice(graphs, func(i, j int) bool {
		return graphs[i].Title < graphs[j].Title
	})

	// 4. Group graphs by category and domain
	categories := groupByCategory(graphs)
	domains := groupByDomain(graphs)

	data := ViewerData{
		Graphs:      graphs,
		Categories:  categories,
		Domains:     domains,
		GeneratedAt: time.Now(),
	}

	// 5. Create output directory
	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create output directory").
			WithDetails("outputDir", g.OutputDir)
	}

	// 6. Copy SVG files to viewer directory
	if err := g.copySVGs(ctx, svgPaths); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "copy SVGs failed")
	}

	// 7. Render HTML template
	if err := g.renderTemplate(ctx, data); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "render failed")
	}

	// 8. Write metadata JSON
	if err := g.writeMetadataJSON(ctx, data); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "write metadata failed")
	}

	return nil
}

// renderTemplate executes the HTML template and writes output
func (g *Generator) renderTemplate(ctx context.Context, data ViewerData) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	outPath := filepath.Join(g.OutputDir, "index.html")
	outFile, err := os.Create(outPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "create output failed").
			WithDetails("path", outPath)
	}
	defer outFile.Close()

	if err := g.Template.ExecuteTemplate(outFile, "index.html", data); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "execute template failed").
			WithDetails("template", "index.html")
	}

	return nil
}

// writeMetadataJSON writes the graph metadata as JSON for client-side use
func (g *Generator) writeMetadataJSON(ctx context.Context, data ViewerData) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	outPath := filepath.Join(g.OutputDir, "data.json")
	outFile, err := os.Create(outPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "create JSON output failed").
			WithDetails("path", outPath)
	}
	defer outFile.Close()

	encoder := json.NewEncoder(outFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "encode JSON failed")
	}

	return nil
}

// groupByCategory groups graphs by their category (pkg/internal)
func groupByCategory(graphs []GraphMetadata) map[string][]GraphMetadata {
	result := make(map[string][]GraphMetadata)
	for _, g := range graphs {
		result[g.Category] = append(result[g.Category], g)
	}
	return result
}

// copySVGs copies SVG files from source directory to viewer/svgs/
func (g *Generator) copySVGs(ctx context.Context, svgPaths []string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Create svgs subdirectory
	svgsDir := filepath.Join(g.OutputDir, "svgs")
	if err := os.MkdirAll(svgsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "create svgs directory failed").
			WithDetails("svgsDir", svgsDir)
	}

	// Copy each SVG file
	for _, svgPath := range svgPaths {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		srcPath := filepath.Join(g.Scanner.BaseDir, svgPath)
		dstPath := filepath.Join(svgsDir, filepath.Base(svgPath))

		// Read source file
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeIO, "read SVG failed").
				WithDetails("path", srcPath)
		}

		// Write to destination
		if err := os.WriteFile(dstPath, content, 0644); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeIO, "write SVG failed").
				WithDetails("path", dstPath)
		}
	}

	return nil
}

// groupByDomain groups graphs by their domain
func groupByDomain(graphs []GraphMetadata) map[string][]GraphMetadata {
	result := make(map[string][]GraphMetadata)
	for _, g := range graphs {
		result[g.Domain] = append(result[g.Domain], g)
	}
	return result
}
