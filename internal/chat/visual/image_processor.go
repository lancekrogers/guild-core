// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package visual

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// ImageProcessor handles image detection, processing, and ASCII art rendering
type ImageProcessor struct {
	asciiWidth     int
	asciiHeight    int
	enableColor    bool
	externalViewer string
	supportedExts  map[string]bool
}

// NewImageProcessor creates a new image processor with default settings
func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{
		asciiWidth:     80,   // Default ASCII width
		asciiHeight:    40,   // Default ASCII height
		enableColor:    true, // Enable color ASCII art
		externalViewer: detectDefaultViewer(),
		supportedExts: map[string]bool{
			".png":  true,
			".jpg":  true,
			".jpeg": true,
			".gif":  true,
			".bmp":  true,
			".webp": true,
			".svg":  true,
		},
	}
}

// ImageReference represents a detected image in content
type ImageReference struct {
	Path         string
	AltText      string
	StartIndex   int
	EndIndex     int
	IsValid      bool
	IsAccessible bool
	Error        string
}

// ProcessContent detects and processes images in content
func (ip *ImageProcessor) ProcessContent(content string) (string, []ImageReference, error) {
	// Detect image references
	refs := ip.detectImageReferences(content)

	// Process each image reference
	processedRefs := make([]ImageReference, 0, len(refs))
	processedContent := content

	// Process from end to start to maintain string indices
	for i := len(refs) - 1; i >= 0; i-- {
		ref := refs[i]

		// Validate and process the image
		processedRef := ip.processImageReference(ref)
		processedRefs = append([]ImageReference{processedRef}, processedRefs...)

		// Replace in content if valid
		if processedRef.IsValid && processedRef.IsAccessible {
			replacement := ip.generateImageReplacement(processedRef)
			processedContent = processedContent[:ref.StartIndex] + replacement + processedContent[ref.EndIndex:]
		}
	}

	return processedContent, processedRefs, nil
}

// detectImageReferences finds image references in content using various patterns
func (ip *ImageProcessor) detectImageReferences(content string) []ImageReference {
	var refs []ImageReference

	// Pattern 1: Markdown image syntax ![alt](path)
	markdownRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	matches := markdownRegex.FindAllStringSubmatch(content, -1)
	indices := markdownRegex.FindAllStringIndex(content, -1)

	for i, match := range matches {
		if len(match) >= 3 {
			refs = append(refs, ImageReference{
				Path:       match[2],
				AltText:    match[1],
				StartIndex: indices[i][0],
				EndIndex:   indices[i][1],
			})
		}
	}

	// Pattern 2: HTML img tags <img src="path" alt="alt">
	htmlRegex := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["'][^>]*(?:alt=["']([^"']*)["'])?[^>]*>`)
	htmlMatches := htmlRegex.FindAllStringSubmatch(content, -1)
	htmlIndices := htmlRegex.FindAllStringIndex(content, -1)

	for i, match := range htmlMatches {
		if len(match) >= 2 {
			altText := ""
			if len(match) >= 3 {
				altText = match[2]
			}
			refs = append(refs, ImageReference{
				Path:       match[1],
				AltText:    altText,
				StartIndex: htmlIndices[i][0],
				EndIndex:   htmlIndices[i][1],
			})
		}
	}

	// Pattern 3: Direct file paths (when they look like images)
	pathRegex := regexp.MustCompile(`(?:^|\s)((?:[~/.]?[^/\s]+/)*[^/\s]+\.(?:png|jpg|jpeg|gif|bmp|webp|svg))(?:\s|$)`)
	pathMatches := pathRegex.FindAllStringSubmatch(content, -1)
	pathIndices := pathRegex.FindAllStringIndex(content, -1)

	for i, match := range pathMatches {
		if len(match) >= 2 {
			path := strings.TrimSpace(match[1])
			// Skip if already captured by other patterns
			alreadyFound := false
			for _, existing := range refs {
				if existing.Path == path {
					alreadyFound = true
					break
				}
			}
			if !alreadyFound {
				refs = append(refs, ImageReference{
					Path:       path,
					AltText:    filepath.Base(path),
					StartIndex: pathIndices[i][0] + len(match[0]) - len(match[1]),
					EndIndex:   pathIndices[i][0] + len(match[0]),
				})
			}
		}
	}

	return refs
}

// processImageReference validates and processes a single image reference
func (ip *ImageProcessor) processImageReference(ref ImageReference) ImageReference {
	// Expand path
	expandedPath := ip.expandPath(ref.Path)
	ref.Path = expandedPath

	// Check if file exists and is supported
	if _, err := os.Stat(expandedPath); err != nil {
		ref.IsValid = false
		ref.IsAccessible = false
		ref.Error = fmt.Sprintf("File not found: %s", err.Error())
		return ref
	}

	// Check extension
	ext := strings.ToLower(filepath.Ext(expandedPath))
	if !ip.supportedExts[ext] {
		ref.IsValid = false
		ref.IsAccessible = false
		ref.Error = fmt.Sprintf("Unsupported image format: %s", ext)
		return ref
	}

	ref.IsValid = true
	ref.IsAccessible = true
	return ref
}

// generateImageReplacement creates a replacement string for an image reference
func (ip *ImageProcessor) generateImageReplacement(ref ImageReference) string {
	var parts []string

	// Add ASCII art if possible
	asciiArt, err := ip.generateASCIIArt(ref.Path)
	if err == nil && asciiArt != "" {
		parts = append(parts, "📸 ASCII Preview:")
		parts = append(parts, "```")
		parts = append(parts, asciiArt)
		parts = append(parts, "```")
	} else {
		// Fallback to file info
		parts = append(parts, fmt.Sprintf("🖼️  **Image:** %s", ref.AltText))
		if ref.AltText != filepath.Base(ref.Path) {
			parts = append(parts, fmt.Sprintf("**Path:** `%s`", ref.Path))
		}
	}

	// Add view instructions
	parts = append(parts, fmt.Sprintf("*View full image: `%s %s`*", ip.externalViewer, ref.Path))

	return strings.Join(parts, "\n")
}

// generateASCIIArt creates ASCII art from an image file
func (ip *ImageProcessor) generateASCIIArt(imagePath string) (string, error) {
	// Try chafa first (better color support)
	if chafaPath, err := exec.LookPath("chafa"); err == nil {
		return ip.generateASCIIWithChafa(chafaPath, imagePath)
	}

	// Fallback to jp2a (monochrome)
	if jp2aPath, err := exec.LookPath("jp2a"); err == nil {
		return ip.generateASCIIWithJp2a(jp2aPath, imagePath)
	}

	return "", gerror.New(gerror.ErrCodeExternal, "No ASCII art tools available (chafa or jp2a required)", nil)
}

// generateASCIIWithChafa uses chafa to generate colored ASCII art
func (ip *ImageProcessor) generateASCIIWithChafa(chafaPath, imagePath string) (string, error) {
	args := []string{
		imagePath,
		"--size", fmt.Sprintf("%dx%d", ip.asciiWidth, ip.asciiHeight),
		"--format", "symbols",
	}

	if ip.enableColor {
		args = append(args, "--colors", "256")
	} else {
		args = append(args, "--colors", "2")
	}

	cmd := exec.Command(chafaPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeExternal, "chafa failed")
	}

	return strings.TrimSpace(string(output)), nil
}

// generateASCIIWithJp2a uses jp2a to generate monochrome ASCII art
func (ip *ImageProcessor) generateASCIIWithJp2a(jp2aPath, imagePath string) (string, error) {
	args := []string{
		"--width", fmt.Sprintf("%d", ip.asciiWidth),
		"--height", fmt.Sprintf("%d", ip.asciiHeight),
		imagePath,
	}

	cmd := exec.Command(jp2aPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeExternal, "jp2a failed")
	}

	return strings.TrimSpace(string(output)), nil
}

// expandPath expands ~ and relative paths
func (ip *ImageProcessor) expandPath(path string) string {
	// Expand ~
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(homeDir, path[2:])
		}
	}

	// Convert to absolute path
	if !filepath.IsAbs(path) {
		if absPath, err := filepath.Abs(path); err == nil {
			path = absPath
		}
	}

	return path
}

// detectDefaultViewer detects the default image viewer for the system
func detectDefaultViewer() string {
	// Common viewers by OS
	viewers := []string{
		"open",     // macOS
		"xdg-open", // Linux
		"start",    // Windows
		"feh",      // Linux image viewer
		"eog",      // GNOME image viewer
		"gwenview", // KDE image viewer
		"qlmanage", // macOS QuickLook
	}

	for _, viewer := range viewers {
		if _, err := exec.LookPath(viewer); err == nil {
			return viewer
		}
	}

	return "cat" // Fallback
}

// SetASCIISize sets the size for ASCII art generation
func (ip *ImageProcessor) SetASCIISize(width, height int) {
	ip.asciiWidth = width
	ip.asciiHeight = height
}

// SetExternalViewer sets the external image viewer command
func (ip *ImageProcessor) SetExternalViewer(viewer string) {
	ip.externalViewer = viewer
}

// SetColorEnabled enables or disables color ASCII art
func (ip *ImageProcessor) SetColorEnabled(enabled bool) {
	ip.enableColor = enabled
}

// GetSupportedExtensions returns a list of supported image extensions
func (ip *ImageProcessor) GetSupportedExtensions() []string {
	exts := make([]string, 0, len(ip.supportedExts))
	for ext := range ip.supportedExts {
		exts = append(exts, ext)
	}
	return exts
}
