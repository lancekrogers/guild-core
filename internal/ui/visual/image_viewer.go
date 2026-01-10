// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package visual

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/disintegration/imaging"
	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// ImageViewer provides comprehensive image viewing capabilities for terminals
type ImageViewer struct {
	maxWidth    int
	maxHeight   int
	style       lipgloss.Style
	headerStyle lipgloss.Style
	errorStyle  lipgloss.Style
	infoStyle   lipgloss.Style

	// Terminal capabilities
	capabilities TerminalCapabilities

	// Caching
	cache    map[string]*CachedImage
	cacheMu  sync.RWMutex
	maxCache int

	// Configuration
	config ViewerConfig
}

// TerminalCapabilities tracks what the terminal supports
type TerminalCapabilities struct {
	SixelSupport    bool
	iTerm2Support   bool
	KittySupport    bool
	TrueColor       bool
	Unicode         bool
	TerminalType    string
	TerminalProgram string
}

// ViewerConfig contains configuration options
type ViewerConfig struct {
	PreferredMethod  DisplayMethod
	AutoDetect       bool
	EnableCache      bool
	ShowMetadata     bool
	ShowAnalysis     bool
	CompactDisplay   bool
	ColorDepth       int
	ASCIICharSet     string
	UnicodeBlocks    bool
	AnimationSupport bool
	MaxFileSize      int64
}

// DisplayMethod represents different image display methods
type DisplayMethod int

const (
	MethodAuto DisplayMethod = iota
	MethodSixel
	MethodiTerm2
	MethodKitty
	MethodUnicodeBlocks
	MethodASCII
	MethodDescription
)

// CachedImage represents a cached image with multiple representations
type CachedImage struct {
	Path         string
	LastModified time.Time
	Analysis     *ImageAnalysis
	Sixel        string
	iTerm2       string
	Unicode      string
	ASCII        string
	Thumbnail    string
}

// ImageAnalysis provides detailed image analysis
type ImageAnalysis struct {
	// Basic properties
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	AspectRatio float64 `json:"aspect_ratio"`
	FileSize    int64   `json:"file_size"`
	Format      string  `json:"format"`

	// Color analysis
	DominantColors  []ColorInfo `json:"dominant_colors"`
	ColorPalette    []string    `json:"color_palette"`
	AverageColor    string      `json:"average_color"`
	ColorComplexity float64     `json:"color_complexity"`
	HasTransparency bool        `json:"has_transparency"`

	// Content analysis
	Type              ImageType `json:"type"`
	IsScreenshot      bool      `json:"is_screenshot"`
	HasText           bool      `json:"has_text"`
	EstimatedTextArea float64   `json:"estimated_text_area"`
	Brightness        float64   `json:"brightness"`
	Contrast          float64   `json:"contrast"`
	Sharpness         float64   `json:"sharpness"`

	// Technical details
	ColorSpace string `json:"color_space"`
	BitDepth   int    `json:"bit_depth"`
	DPI        int    `json:"dpi"`
	HasEXIF    bool   `json:"has_exif"`

	// Suggestions
	RecommendedMethod DisplayMethod `json:"recommended_method"`
	ViewingTips       []string      `json:"viewing_tips"`
	Warnings          []string      `json:"warnings"`
}

// ColorInfo represents color information
type ColorInfo struct {
	Hex        string  `json:"hex"`
	RGB        [3]int  `json:"rgb"`
	Percentage float64 `json:"percentage"`
	Name       string  `json:"name"`
}

// ImageType represents the type of image content
type ImageType int

const (
	TypeUnknown ImageType = iota
	TypePhoto
	TypeScreenshot
	TypeDiagram
	TypeChart
	TypeIcon
	TypeLogo
	TypeArt
	TypeDocument
)

// NewImageViewer creates a new enhanced image viewer
func NewImageViewer() *ImageViewer {
	viewer := &ImageViewer{
		maxWidth:  80,
		maxHeight: 40,
		cache:     make(map[string]*CachedImage),
		maxCache:  50,

		style: lipgloss.NewStyle().Margin(1, 2),

		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			Underline(true),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),

		infoStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")),

		config: ViewerConfig{
			PreferredMethod:  MethodAuto,
			AutoDetect:       true,
			EnableCache:      true,
			ShowMetadata:     true,
			ShowAnalysis:     false,
			CompactDisplay:   false,
			ColorDepth:       256,
			ASCIICharSet:     "@%#*+=-:. ",
			UnicodeBlocks:    true,
			AnimationSupport: false,
			MaxFileSize:      50 * 1024 * 1024, // 50MB
		},
	}

	// Detect terminal capabilities
	viewer.capabilities = viewer.detectTerminalCapabilities()

	return viewer
}

// Display displays an image using the best available method
func (v *ImageViewer) Display(path string) error {
	// Validate file
	if err := v.validateImageFile(path); err != nil {
		return err
	}

	// Get or create cached image
	cached, err := v.getCachedImage(path)
	if err != nil {
		return err
	}

	// Choose display method
	method := v.chooseDisplayMethod(cached)

	// Display header
	v.displayHeader(path, cached.Analysis)

	// Display image
	switch method {
	case MethodSixel:
		return v.displaySixel(cached)
	case MethodiTerm2:
		return v.displayiTerm2(cached)
	case MethodKitty:
		return v.displayKitty(cached)
	case MethodUnicodeBlocks:
		return v.displayUnicodeBlocks(cached)
	case MethodASCII:
		return v.displayASCII(cached)
	default:
		return v.displayDescription(cached)
	}
}

// detectTerminalCapabilities detects what the terminal supports
func (v *ImageViewer) detectTerminalCapabilities() TerminalCapabilities {
	caps := TerminalCapabilities{
		TerminalType:    os.Getenv("TERM"),
		TerminalProgram: os.Getenv("TERM_PROGRAM"),
	}

	// Check for Sixel support
	caps.SixelSupport = v.checkSixelSupport()

	// Check for iTerm2
	caps.iTerm2Support = caps.TerminalProgram == "iTerm.app" ||
		os.Getenv("ITERM_SESSION_ID") != ""

	// Check for Kitty
	caps.KittySupport = caps.TerminalProgram == "kitty" ||
		os.Getenv("KITTY_WINDOW_ID") != ""

	// Check for true color support
	caps.TrueColor = v.checkTrueColorSupport()

	// Unicode support (assume yes for modern terminals)
	caps.Unicode = true

	return caps
}

// checkSixelSupport checks if the terminal supports Sixel graphics
func (v *ImageViewer) checkSixelSupport() bool {
	// Query terminal for sixel support
	if v.queryTerminalCapability("4;1") {
		return true
	}

	// Check known terminals
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	sixelTerms := []string{
		"xterm-sixel",
		"mlterm",
		"yaft",
		"foot",
	}

	for _, st := range sixelTerms {
		if strings.Contains(term, st) {
			return true
		}
	}

	// Check explicit environment variable
	return os.Getenv("SIXEL_SUPPORT") == "1" || termProgram == "WezTerm"
}

// checkTrueColorSupport checks if terminal supports 24-bit color
func (v *ImageViewer) checkTrueColorSupport() bool {
	colorterm := os.Getenv("COLORTERM")
	return colorterm == "truecolor" || colorterm == "24bit"
}

// queryTerminalCapability queries terminal for specific capabilities
func (v *ImageViewer) queryTerminalCapability(query string) bool {
	// This is a simplified check - in practice, you'd send ANSI queries
	// and wait for responses, but that's complex in a CLI context
	return false
}

// validateImageFile validates that the file is a supported image
func (v *ImageViewer) validateImageFile(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "image file not found")
	}

	// Check file size
	if v.config.MaxFileSize > 0 && info.Size() > v.config.MaxFileSize {
		return gerror.New(gerror.ErrCodeValidation,
			fmt.Sprintf("image file too large: %d bytes (max: %d)",
				info.Size(), v.config.MaxFileSize), nil)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	supportedExts := []string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".tiff", ".svg"}

	supported := false
	for _, se := range supportedExts {
		if ext == se {
			supported = true
			break
		}
	}

	if !supported {
		return gerror.New(gerror.ErrCodeValidation,
			fmt.Sprintf("unsupported image format: %s", ext), nil)
	}

	return nil
}

// getCachedImage gets or creates a cached image
func (v *ImageViewer) getCachedImage(path string) (*CachedImage, error) {
	if !v.config.EnableCache {
		return v.createCachedImage(path)
	}

	v.cacheMu.RLock()
	cached, exists := v.cache[path]
	v.cacheMu.RUnlock()

	if exists {
		// Check if file has been modified
		info, err := os.Stat(path)
		if err == nil && !info.ModTime().After(cached.LastModified) {
			return cached, nil
		}
	}

	// Create new cached image
	cached, err := v.createCachedImage(path)
	if err != nil {
		return nil, err
	}

	// Store in cache
	v.cacheMu.Lock()
	v.cache[path] = cached

	// Cleanup old cache entries if needed
	if len(v.cache) > v.maxCache {
		v.cleanupCache()
	}
	v.cacheMu.Unlock()

	return cached, nil
}

// createCachedImage creates a new cached image with analysis
func (v *ImageViewer) createCachedImage(path string) (*CachedImage, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Load and analyze image
	analysis, err := v.analyzeImage(path)
	if err != nil {
		return nil, err
	}

	cached := &CachedImage{
		Path:         path,
		LastModified: info.ModTime(),
		Analysis:     analysis,
	}

	// Pre-generate representations if the image is small enough
	if info.Size() < 5*1024*1024 { // 5MB
		v.pregenerateRepresentations(cached)
	}

	return cached, nil
}

// pregenerateRepresentations pre-generates image representations
func (v *ImageViewer) pregenerateRepresentations(cached *CachedImage) {
	// This would pre-generate ASCII, Unicode, etc. representations
	// For now, we'll generate them on-demand
}

// cleanupCache removes old cache entries
func (v *ImageViewer) cleanupCache() {
	// Keep only the most recently accessed images
	type cacheEntry struct {
		path string
		time time.Time
	}

	var entries []cacheEntry
	for path, cached := range v.cache {
		entries = append(entries, cacheEntry{path, cached.LastModified})
	}

	// Sort by last modified time
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].time.After(entries[j].time)
	})

	// Keep only the newest half
	keepCount := v.maxCache / 2
	for i := keepCount; i < len(entries); i++ {
		delete(v.cache, entries[i].path)
	}
}

// chooseDisplayMethod chooses the best display method
func (v *ImageViewer) chooseDisplayMethod(cached *CachedImage) DisplayMethod {
	if v.config.PreferredMethod != MethodAuto {
		return v.config.PreferredMethod
	}

	// Use analysis recommendation if available
	if cached.Analysis != nil && cached.Analysis.RecommendedMethod != MethodAuto {
		// Check if the recommended method is supported
		if v.isMethodSupported(cached.Analysis.RecommendedMethod) {
			return cached.Analysis.RecommendedMethod
		}
	}

	// Fall back to capability-based selection
	if v.capabilities.SixelSupport {
		return MethodSixel
	}

	if v.capabilities.iTerm2Support {
		return MethodiTerm2
	}

	if v.capabilities.KittySupport {
		return MethodKitty
	}

	if v.capabilities.Unicode && v.config.UnicodeBlocks {
		return MethodUnicodeBlocks
	}

	return MethodASCII
}

// isMethodSupported checks if a display method is supported
func (v *ImageViewer) isMethodSupported(method DisplayMethod) bool {
	switch method {
	case MethodSixel:
		return v.capabilities.SixelSupport
	case MethodiTerm2:
		return v.capabilities.iTerm2Support
	case MethodKitty:
		return v.capabilities.KittySupport
	case MethodUnicodeBlocks:
		return v.capabilities.Unicode
	case MethodASCII:
		return true
	case MethodDescription:
		return true
	default:
		return false
	}
}

// Display methods

// displayHeader displays image information header
func (v *ImageViewer) displayHeader(path string, analysis *ImageAnalysis) {
	header := fmt.Sprintf("🖼️  %s", filepath.Base(path))
	fmt.Println(v.headerStyle.Render(header))

	if v.config.ShowMetadata && analysis != nil {
		info := []string{
			fmt.Sprintf("📐 %dx%d", analysis.Width, analysis.Height),
			fmt.Sprintf("📄 %s", strings.ToUpper(analysis.Format)),
			v.formatFileSize(analysis.FileSize),
		}

		if analysis.ColorComplexity > 0 {
			info = append(info, fmt.Sprintf("🎨 %.0f%% color complexity", analysis.ColorComplexity*100))
		}

		fmt.Println(v.infoStyle.Render(strings.Join(info, " • ")))
	}
}

// displaySixel displays image using Sixel graphics
func (v *ImageViewer) displaySixel(cached *CachedImage) error {
	// Generate Sixel if not cached
	if cached.Sixel == "" {
		sixel, err := v.generateSixel(cached.Path)
		if err != nil {
			return err
		}
		cached.Sixel = sixel
	}

	fmt.Print(cached.Sixel)
	return nil
}

// displayiTerm2 displays image using iTerm2 inline images
func (v *ImageViewer) displayiTerm2(cached *CachedImage) error {
	// Generate iTerm2 sequence if not cached
	if cached.iTerm2 == "" {
		iterm2, err := v.generateiTerm2(cached.Path)
		if err != nil {
			return err
		}
		cached.iTerm2 = iterm2
	}

	fmt.Print(cached.iTerm2)
	return nil
}

// displayKitty displays image using Kitty graphics protocol
func (v *ImageViewer) displayKitty(cached *CachedImage) error {
	// For now, fall back to iTerm2 method
	return v.displayiTerm2(cached)
}

// displayUnicodeBlocks displays image using Unicode block characters
func (v *ImageViewer) displayUnicodeBlocks(cached *CachedImage) error {
	// Generate Unicode representation if not cached
	if cached.Unicode == "" {
		unicode, err := v.generateUnicodeBlocks(cached.Path)
		if err != nil {
			return err
		}
		cached.Unicode = unicode
	}

	fmt.Print(cached.Unicode)
	return nil
}

// displayASCII displays image using ASCII art
func (v *ImageViewer) displayASCII(cached *CachedImage) error {
	// Generate ASCII if not cached
	if cached.ASCII == "" {
		ascii, err := v.generateASCII(cached.Path)
		if err != nil {
			return err
		}
		cached.ASCII = ascii
	}

	fmt.Print(cached.ASCII)
	return nil
}

// displayDescription displays image description and metadata
func (v *ImageViewer) displayDescription(cached *CachedImage) error {
	if cached.Analysis == nil {
		return gerror.New(gerror.ErrCodeInternal, "no image analysis available", nil)
	}

	analysis := cached.Analysis

	fmt.Printf("📸 Image: %s\n", filepath.Base(cached.Path))
	fmt.Printf("📐 Dimensions: %dx%d (%.2f:1)\n",
		analysis.Width, analysis.Height, analysis.AspectRatio)
	fmt.Printf("📄 Format: %s\n", strings.ToUpper(analysis.Format))
	fmt.Printf("💾 Size: %s\n", v.formatFileSize(analysis.FileSize))

	if len(analysis.DominantColors) > 0 {
		fmt.Printf("🎨 Dominant colors: %s\n", v.formatColorList(analysis.DominantColors))
	}

	if analysis.Type != TypeUnknown {
		fmt.Printf("🏷️  Type: %s\n", v.formatImageType(analysis.Type))
	}

	if len(analysis.ViewingTips) > 0 {
		fmt.Printf("💡 Tips: %s\n", strings.Join(analysis.ViewingTips, ", "))
	}

	return nil
}

// Generation methods

// generateSixel generates Sixel graphics representation
func (v *ImageViewer) generateSixel(path string) (string, error) {
	// Try using img2sixel if available
	if imgPath, err := exec.LookPath("img2sixel"); err == nil {
		return v.generateSixelWithImg2sixel(imgPath, path)
	}

	// Try using ImageMagick
	if convertPath, err := exec.LookPath("convert"); err == nil {
		return v.generateSixelWithImageMagick(convertPath, path)
	}

	// Fall back to native Go implementation (basic)
	return v.generateSixelNative(path)
}

// generateSixelWithImg2sixel uses img2sixel tool
func (v *ImageViewer) generateSixelWithImg2sixel(toolPath, imagePath string) (string, error) {
	args := []string{
		"-w", strconv.Itoa(v.maxWidth),
		"-h", strconv.Itoa(v.maxHeight),
		imagePath,
	}

	cmd := exec.Command(toolPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeExternal, "img2sixel failed")
	}

	return string(output), nil
}

// generateSixelWithImageMagick uses ImageMagick to generate Sixel
func (v *ImageViewer) generateSixelWithImageMagick(convertPath, imagePath string) (string, error) {
	args := []string{
		imagePath,
		"-resize", fmt.Sprintf("%dx%d>", v.maxWidth, v.maxHeight),
		"sixel:-",
	}

	cmd := exec.Command(convertPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeExternal, "ImageMagick convert failed")
	}

	return string(output), nil
}

// generateSixelNative generates Sixel using native Go (basic implementation)
func (v *ImageViewer) generateSixelNative(path string) (string, error) {
	// This is a simplified Sixel generator
	// A full implementation would be quite complex
	return "", gerror.New(gerror.ErrCodeNotImplemented,
		"native Sixel generation not yet implemented", nil)
}

// generateiTerm2 generates iTerm2 inline image sequence
func (v *ImageViewer) generateiTerm2(path string) (string, error) {
	// Load and resize image
	img, err := v.loadAndResize(path)
	if err != nil {
		return "", err
	}

	// Encode as PNG
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, img, imaging.PNG); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to encode image")
	}

	// Base64 encode
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Generate iTerm2 sequence
	// Format: \033]1337;File=inline=1;size=SIZE;width=WIDTHpx;height=HEIGHTpx:[base64 data]\a
	bounds := img.Bounds()
	sequence := fmt.Sprintf("\033]1337;File=inline=1;size=%d;width=%dpx;height=%dpx:%s\a",
		buf.Len(), bounds.Dx(), bounds.Dy(), encoded)

	return sequence, nil
}

// generateUnicodeBlocks generates Unicode block character representation
func (v *ImageViewer) generateUnicodeBlocks(path string) (string, error) {
	img, err := v.loadAndResize(path)
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	var result strings.Builder

	// Unicode block characters for different intensities
	blocks := []rune{' ', '░', '▒', '▓', '█'}

	// Process image in 2x1 blocks (each character represents 2 vertical pixels)
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get average intensity for this block
			intensity := v.getBlockIntensity(img, x, y, x+1, y+2)

			// Map to block character
			blockIndex := int(intensity * float64(len(blocks)-1))
			if blockIndex >= len(blocks) {
				blockIndex = len(blocks) - 1
			}

			result.WriteRune(blocks[blockIndex])
		}
		result.WriteRune('\n')
	}

	return result.String(), nil
}

// generateASCII generates ASCII art representation
func (v *ImageViewer) generateASCII(path string) (string, error) {
	img, err := v.loadAndResize(path)
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	var result strings.Builder

	// ASCII characters from dark to light
	ascii := v.config.ASCIICharSet

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 { // Skip every other row for aspect ratio
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get grayscale intensity
			gray := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			intensity := float64(gray.Y) / 255.0

			// Map to ASCII character
			charIndex := int(intensity * float64(len(ascii)-1))
			if charIndex >= len(ascii) {
				charIndex = len(ascii) - 1
			}

			result.WriteByte(ascii[charIndex])
		}
		result.WriteRune('\n')
	}

	return result.String(), nil
}

// analyzeImage performs comprehensive image analysis
func (v *ImageViewer) analyzeImage(path string) (*ImageAnalysis, error) {
	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Load image
	img, format, err := v.loadImageWithFormat(path)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	analysis := &ImageAnalysis{
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		AspectRatio: float64(bounds.Dx()) / float64(bounds.Dy()),
		FileSize:    info.Size(),
		Format:      format,
	}

	// Analyze colors
	analysis.DominantColors = v.analyzeDominantColors(img)
	analysis.AverageColor = v.calculateAverageColor(img)
	analysis.ColorComplexity = v.calculateColorComplexity(img)
	analysis.HasTransparency = v.checkTransparency(img)

	// Content analysis
	analysis.Type = v.detectImageType(img, path)
	analysis.IsScreenshot = v.detectScreenshot(img, path)
	analysis.Brightness = v.calculateBrightness(img)
	analysis.Contrast = v.calculateContrast(img)

	// Determine recommended display method
	analysis.RecommendedMethod = v.recommendDisplayMethod(analysis)

	// Generate viewing tips
	analysis.ViewingTips = v.generateViewingTips(analysis)

	return analysis, nil
}

// Helper methods

// loadAndResize loads and resizes an image
func (v *ImageViewer) loadAndResize(path string) (image.Image, error) {
	img, _, err := v.loadImageWithFormat(path)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()

	// Resize if needed
	if bounds.Dx() > v.maxWidth || bounds.Dy() > v.maxHeight {
		img = imaging.Fit(img, v.maxWidth, v.maxHeight, imaging.Lanczos)
	}

	return img, nil
}

// loadImageWithFormat loads an image and returns the format
func (v *ImageViewer) loadImageWithFormat(path string) (image.Image, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return nil, "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to decode image")
	}

	return img, format, nil
}

// getBlockIntensity calculates average intensity for a block region
func (v *ImageViewer) getBlockIntensity(img image.Image, x1, y1, x2, y2 int) float64 {
	bounds := img.Bounds()

	// Clamp coordinates
	if x2 > bounds.Max.X {
		x2 = bounds.Max.X
	}
	if y2 > bounds.Max.Y {
		y2 = bounds.Max.Y
	}

	var total float64
	var count int

	for y := y1; y < y2; y++ {
		for x := x1; x < x2; x++ {
			if x >= bounds.Min.X && x < bounds.Max.X && y >= bounds.Min.Y && y < bounds.Max.Y {
				gray := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
				total += float64(gray.Y) / 255.0
				count++
			}
		}
	}

	if count == 0 {
		return 0
	}

	return total / float64(count)
}

// analyzeDominantColors analyzes dominant colors in the image
func (v *ImageViewer) analyzeDominantColors(img image.Image) []ColorInfo {
	// This is a simplified implementation
	// A full implementation would use k-means clustering or similar

	bounds := img.Bounds()
	colorCounts := make(map[string]int)

	// Sample colors (don't check every pixel for performance)
	step := max(1, bounds.Dx()/100)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert to 8-bit
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

			hex := fmt.Sprintf("#%02x%02x%02x", r8, g8, b8)
			colorCounts[hex]++
		}
	}

	// Sort by frequency
	type colorCount struct {
		hex   string
		count int
	}

	var colors []colorCount
	for hex, count := range colorCounts {
		colors = append(colors, colorCount{hex, count})
	}

	sort.Slice(colors, func(i, j int) bool {
		return colors[i].count > colors[j].count
	})

	// Return top colors
	var result []ColorInfo
	total := len(colors)

	for i, cc := range colors {
		if i >= 5 { // Top 5 colors
			break
		}

		// Parse hex color
		var r, g, b int
		fmt.Sscanf(cc.hex, "#%02x%02x%02x", &r, &g, &b)

		result = append(result, ColorInfo{
			Hex:        cc.hex,
			RGB:        [3]int{r, g, b},
			Percentage: float64(cc.count) / float64(total) * 100,
			Name:       v.getColorName(r, g, b),
		})
	}

	return result
}

// calculateAverageColor calculates the average color of the image
func (v *ImageViewer) calculateAverageColor(img image.Image) string {
	bounds := img.Bounds()
	var totalR, totalG, totalB uint64
	var count uint64

	step := max(1, bounds.Dx()/50) // Sample for performance

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			r, g, b, _ := img.At(x, y).RGBA()
			totalR += uint64(r >> 8)
			totalG += uint64(g >> 8)
			totalB += uint64(b >> 8)
			count++
		}
	}

	if count == 0 {
		return "#000000"
	}

	avgR := uint8(totalR / count)
	avgG := uint8(totalG / count)
	avgB := uint8(totalB / count)

	return fmt.Sprintf("#%02x%02x%02x", avgR, avgG, avgB)
}

// calculateColorComplexity calculates color complexity (0-1)
func (v *ImageViewer) calculateColorComplexity(img image.Image) float64 {
	// Simplified: count unique colors in a sample
	bounds := img.Bounds()
	colors := make(map[string]bool)

	step := max(1, bounds.Dx()/50)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			r, g, b, _ := img.At(x, y).RGBA()
			hex := fmt.Sprintf("%02x%02x%02x", r>>8, g>>8, b>>8)
			colors[hex] = true
		}
	}

	// Normalize by sample size
	sampleSize := ((bounds.Dy() / step) + 1) * ((bounds.Dx() / step) + 1)
	complexity := float64(len(colors)) / float64(sampleSize)

	if complexity > 1.0 {
		complexity = 1.0
	}

	return complexity
}

// checkTransparency checks if the image has transparency
func (v *ImageViewer) checkTransparency(img image.Image) bool {
	bounds := img.Bounds()

	// Check a sample of pixels
	step := max(1, bounds.Dx()/20)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 65535 { // Not fully opaque
				return true
			}
		}
	}

	return false
}

// detectImageType detects the type of image content
func (v *ImageViewer) detectImageType(img image.Image, path string) ImageType {
	bounds := img.Bounds()

	// Simple heuristics based on filename and dimensions
	filename := strings.ToLower(filepath.Base(path))

	if strings.Contains(filename, "screenshot") || strings.Contains(filename, "screen") {
		return TypeScreenshot
	}

	if strings.Contains(filename, "icon") || (bounds.Dx() <= 128 && bounds.Dy() <= 128) {
		return TypeIcon
	}

	if strings.Contains(filename, "logo") {
		return TypeLogo
	}

	if strings.Contains(filename, "chart") || strings.Contains(filename, "graph") {
		return TypeChart
	}

	// Aspect ratio heuristics
	aspectRatio := float64(bounds.Dx()) / float64(bounds.Dy())

	if aspectRatio > 2.0 || aspectRatio < 0.5 {
		return TypeDiagram
	}

	if bounds.Dx() > 800 && bounds.Dy() > 600 {
		return TypePhoto
	}

	return TypeUnknown
}

// detectScreenshot detects if image is likely a screenshot
func (v *ImageViewer) detectScreenshot(img image.Image, path string) bool {
	filename := strings.ToLower(filepath.Base(path))
	return strings.Contains(filename, "screenshot") ||
		strings.Contains(filename, "screen") ||
		strings.Contains(filename, "capture")
}

// calculateBrightness calculates average brightness (0-1)
func (v *ImageViewer) calculateBrightness(img image.Image) float64 {
	bounds := img.Bounds()
	var total float64
	var count int

	step := max(1, bounds.Dx()/30)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			gray := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			total += float64(gray.Y) / 255.0
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / float64(count)
}

// calculateContrast calculates image contrast (simplified)
func (v *ImageViewer) calculateContrast(img image.Image) float64 {
	bounds := img.Bounds()
	var values []float64

	step := max(1, bounds.Dx()/30)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			gray := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			values = append(values, float64(gray.Y)/255.0)
		}
	}

	if len(values) < 2 {
		return 0
	}

	// Calculate standard deviation as a measure of contrast
	var mean float64
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance)
}

// recommendDisplayMethod recommends the best display method for an image
func (v *ImageViewer) recommendDisplayMethod(analysis *ImageAnalysis) DisplayMethod {
	// High resolution images: prefer Sixel or iTerm2
	if analysis.Width > 1000 || analysis.Height > 1000 {
		if v.capabilities.SixelSupport {
			return MethodSixel
		}
		if v.capabilities.iTerm2Support {
			return MethodiTerm2
		}
	}

	// Screenshots: prefer accurate representation
	if analysis.IsScreenshot {
		if v.capabilities.SixelSupport {
			return MethodSixel
		}
		if v.capabilities.iTerm2Support {
			return MethodiTerm2
		}
		return MethodUnicodeBlocks
	}

	// Simple images: ASCII might be sufficient
	if analysis.ColorComplexity < 0.3 && analysis.Type == TypeIcon {
		return MethodASCII
	}

	// Default: use best available
	return MethodAuto
}

// generateViewingTips generates helpful viewing tips
func (v *ImageViewer) generateViewingTips(analysis *ImageAnalysis) []string {
	var tips []string

	if analysis.Width > v.maxWidth || analysis.Height > v.maxHeight {
		tips = append(tips, "Image will be resized to fit terminal")
	}

	if analysis.HasTransparency && !v.capabilities.TrueColor {
		tips = append(tips, "Transparency may not display correctly")
	}

	if analysis.ColorComplexity > 0.8 {
		tips = append(tips, "High color complexity - consider viewing externally")
	}

	if analysis.IsScreenshot {
		tips = append(tips, "Screenshot detected - text may be hard to read")
	}

	return tips
}

// formatFileSize formats file size in human readable format
func (v *ImageViewer) formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	suffixes := []string{"B", "KB", "MB", "GB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), suffixes[exp])
}

// formatColorList formats a list of colors for display
func (v *ImageViewer) formatColorList(colors []ColorInfo) string {
	var parts []string
	for _, color := range colors {
		if len(parts) >= 3 { // Limit display
			break
		}
		parts = append(parts, fmt.Sprintf("%s (%.1f%%)", color.Hex, color.Percentage))
	}
	return strings.Join(parts, ", ")
}

// formatImageType formats image type for display
func (v *ImageViewer) formatImageType(imageType ImageType) string {
	types := map[ImageType]string{
		TypePhoto:      "Photo",
		TypeScreenshot: "Screenshot",
		TypeDiagram:    "Diagram",
		TypeChart:      "Chart",
		TypeIcon:       "Icon",
		TypeLogo:       "Logo",
		TypeArt:        "Art",
		TypeDocument:   "Document",
	}

	if name, ok := types[imageType]; ok {
		return name
	}
	return "Unknown"
}

// getColorName returns a human-readable color name (simplified)
func (v *ImageViewer) getColorName(r, g, b int) string {
	// Simplified color naming - a full implementation would use a color database
	if r > 200 && g > 200 && b > 200 {
		return "Light"
	}
	if r < 50 && g < 50 && b < 50 {
		return "Dark"
	}
	if r > g && r > b {
		return "Red"
	}
	if g > r && g > b {
		return "Green"
	}
	if b > r && b > g {
		return "Blue"
	}
	return "Mixed"
}

// Configuration methods

// SetMaxSize sets the maximum display size
func (v *ImageViewer) SetMaxSize(width, height int) {
	v.maxWidth = width
	v.maxHeight = height
}

// SetPreferredMethod sets the preferred display method
func (v *ImageViewer) SetPreferredMethod(method DisplayMethod) {
	v.config.PreferredMethod = method
}

// SetConfig updates the viewer configuration
func (v *ImageViewer) SetConfig(config ViewerConfig) {
	v.config = config
}

// GetCapabilities returns the detected terminal capabilities
func (v *ImageViewer) GetCapabilities() TerminalCapabilities {
	return v.capabilities
}

// ClearCache clears the image cache
func (v *ImageViewer) ClearCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()

	v.cache = make(map[string]*CachedImage)
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
