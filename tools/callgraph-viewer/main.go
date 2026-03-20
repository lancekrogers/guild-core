package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/lancekrogers/guild-core/tools/callgraph-viewer/generator"
	"github.com/lancekrogers/guild-core/tools/callgraph-viewer/static"
	"github.com/lancekrogers/guild-core/tools/callgraph-viewer/templates"
)

func main() {
	ctx := context.Background()

	// Paths relative to guild-core root
	svgDir := filepath.Join("docs", "images", "callgraphs")
	outputDir := filepath.Join("docs", "callgraphs", "viewer")

	// Load embedded templates
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"groupByDomain": groupByDomain,
	}).ParseFS(templates.FS, "*.html", "partials/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	// Create generator
	gen := &generator.Generator{
		Scanner: &generator.Scanner{
			BaseDir: svgDir,
		},
		Extractor: &generator.MetadataExtractor{
			BaseDir:   svgDir,
			OutputDir: outputDir,
		},
		Template:  tmpl,
		OutputDir: outputDir,
	}

	// Run generation
	if err := gen.Generate(ctx); err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	// Copy static assets
	if err := copyStaticAssets(outputDir); err != nil {
		log.Fatalf("Failed to copy static assets: %v", err)
	}

	fmt.Printf("✓ Callgraph viewer generated at: %s/index.html\n", outputDir)
}

// groupByDomain is a template function to group graphs by domain
func groupByDomain(graphs []generator.GraphMetadata) map[string][]generator.GraphMetadata {
	result := make(map[string][]generator.GraphMetadata)
	for _, g := range graphs {
		result[g.Domain] = append(result[g.Domain], g)
	}
	return result
}

// copyStaticAssets copies CSS/JS files from embedded FS to output directory
func copyStaticAssets(outputDir string) error {
	assetsDir := filepath.Join(outputDir, "assets")

	// Walk the embedded static filesystem
	return walkEmbedFS(static.FS, ".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Create directory in output
			destDir := filepath.Join(assetsDir, path)
			return os.MkdirAll(destDir, 0755)
		}

		// Copy file
		content, err := static.FS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded file %s: %w", path, err)
		}

		destPath := filepath.Join(assetsDir, path)
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("write file %s: %w", destPath, err)
		}

		return nil
	})
}

// walkEmbedFS walks an embedded filesystem (simplified version of filepath.Walk)
func walkEmbedFS(fsys interface{ ReadDir(string) ([]os.DirEntry, error) }, root string, fn func(string, os.FileInfo, error) error) error {
	entries, err := fsys.ReadDir(root)
	if err != nil {
		return fn(root, nil, err)
	}

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		info, err := entry.Info()
		if err != nil {
			if err := fn(path, nil, err); err != nil {
				return err
			}
			continue
		}

		if err := fn(path, info, nil); err != nil {
			return err
		}

		if entry.IsDir() {
			if err := walkEmbedFS(fsys, path, fn); err != nil {
				return err
			}
		}
	}

	return nil
}
