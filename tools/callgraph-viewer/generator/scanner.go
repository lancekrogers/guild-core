package generator

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Scanner finds SVG files in the callgraph directory
type Scanner struct {
	BaseDir string // Base directory to scan (e.g., "docs/images/callgraphs")
}

// Scan walks the base directory and returns paths to all *.svg files
func (s *Scanner) Scan(ctx context.Context) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var svgFiles []string

	err := filepath.Walk(s.BaseDir, func(path string, info fs.FileInfo, err error) error {
		// Check for context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeIO, "walk failed").
				WithDetails("path", path)
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .svg files
		if strings.HasSuffix(info.Name(), ".svg") {
			relPath, err := filepath.Rel(s.BaseDir, path)
			if err != nil {
				return gerror.Wrap(err, gerror.ErrCodeIO, "failed to get relative path").
					WithDetails("path", path).
					WithDetails("baseDir", s.BaseDir)
			}
			svgFiles = append(svgFiles, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "scan failed").
			WithDetails("baseDir", s.BaseDir)
	}

	return svgFiles, nil
}
