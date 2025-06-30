// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package worktree

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// ConflictResolver resolves conflicts automatically using various strategies
type ConflictResolver struct {
	strategies []ResolutionStrategy
	ml         *MLResolver
	history    *ResolutionHistory
	manual     *ManualResolver
	mu         sync.RWMutex
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(ctx context.Context) (*ConflictResolver, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled", nil).
			WithComponent("git.worktree.resolver").
			WithOperation("NewConflictResolver")
	}

	resolver := &ConflictResolver{
		history: NewResolutionHistory(),
		manual:  NewManualResolver(),
	}

	// Initialize resolution strategies in priority order
	resolver.strategies = []ResolutionStrategy{
		&WhitespaceResolver{},
		&ImportResolver{
			languages: map[string]ImportSorter{
				"go":         &GoImportSorter{},
				"javascript": &JSImportSorter{},
				"python":     &PythonImportSorter{},
			},
		},
		&FormattingResolver{},
		&CommentResolver{},
		&SimpleLineResolver{},
	}

	// Initialize ML resolver
	mlResolver, err := NewMLResolver(ctx)
	if err != nil {
		// Log warning but continue without ML
		resolver.ml = nil
	} else {
		resolver.ml = mlResolver
	}

	return resolver, nil
}

// ResolveConflict attempts to resolve a conflict automatically
func (cr *ConflictResolver) ResolveConflict(ctx context.Context, conflict Conflict) (*Resolution, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled", nil).
			WithComponent("git.worktree.resolver").
			WithOperation("ResolveConflict")
	}

	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Try strategies in priority order
	strategies := cr.getApplicableStrategies(conflict)

	for _, strategy := range strategies {
		if strategy.CanResolve(ctx, conflict) {
			resolution, err := strategy.Resolve(ctx, conflict)
			if err == nil && resolution != nil {
				// Record successful resolution
				cr.history.Record(conflict, resolution, strategy)
				return resolution, nil
			}
		}
	}

	// Try ML-based resolution
	if cr.ml != nil {
		if mlResolution := cr.ml.Suggest(ctx, conflict); mlResolution != nil {
			if mlResolution.Confidence > 0.8 {
				cr.history.Record(conflict, mlResolution, nil)
				return mlResolution, nil
			}
		}
	}

	// Fall back to manual resolution
	return cr.manual.RequestManualResolution(ctx, conflict)
}

// getApplicableStrategies returns strategies sorted by priority
func (cr *ConflictResolver) getApplicableStrategies(conflict Conflict) []ResolutionStrategy {
	var applicable []ResolutionStrategy

	for _, strategy := range cr.strategies {
		applicable = append(applicable, strategy)
	}

	// Sort by priority (higher priority first)
	sort.Slice(applicable, func(i, j int) bool {
		return applicable[i].Priority() > applicable[j].Priority()
	})

	return applicable
}

// ResolutionStrategy defines the interface for conflict resolution strategies
type ResolutionStrategy interface {
	CanResolve(ctx context.Context, conflict Conflict) bool
	Resolve(ctx context.Context, conflict Conflict) (*Resolution, error)
	Priority() int
}

// Resolution represents the result of conflict resolution
type Resolution struct {
	Content    string                 `json:"content"`
	Strategy   string                 `json:"strategy"`
	Confidence float64               `json:"confidence"`
	Pattern    *ResolutionPattern     `json:"pattern,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	ResolvedAt time.Time              `json:"resolved_at"`
}

// ResolutionPattern represents a learned resolution pattern
type ResolutionPattern struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Conditions  []string               `json:"conditions"`
	Actions     []string               `json:"actions"`
	Success     float64                `json:"success_rate"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// WhitespaceResolver resolves whitespace-only conflicts
type WhitespaceResolver struct{}

func (wr *WhitespaceResolver) Priority() int {
	return 100 // Highest priority
}

func (wr *WhitespaceResolver) CanResolve(ctx context.Context, conflict Conflict) bool {
	if conflict.Diff == nil {
		return false
	}

	// Check if conflict is only whitespace differences
	return wr.isWhitespaceOnly(conflict.Diff.Content1, conflict.Diff.Content2)
}

func (wr *WhitespaceResolver) Resolve(ctx context.Context, conflict Conflict) (*Resolution, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Normalize whitespace and merge
	normalized1 := wr.normalizeWhitespace(conflict.Diff.Content1)
	normalized2 := wr.normalizeWhitespace(conflict.Diff.Content2)

	if normalized1 == normalized2 {
		return &Resolution{
			Content:    normalized1,
			Strategy:   "whitespace_normalization",
			Confidence: 1.0,
			ResolvedAt: time.Now(),
			Metadata: map[string]interface{}{
				"conflict_type": "whitespace",
			},
		}, nil
	}

	return nil, gerror.New(gerror.ErrCodeConflict, "content differs beyond whitespace", nil)
}

func (wr *WhitespaceResolver) isWhitespaceOnly(content1, content2 string) bool {
	norm1 := wr.normalizeWhitespace(content1)
	norm2 := wr.normalizeWhitespace(content2)
	return norm1 == norm2
}

func (wr *WhitespaceResolver) normalizeWhitespace(content string) string {
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	
	// Normalize spaces and tabs
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Convert tabs to spaces
		line = strings.ReplaceAll(line, "\t", "    ")
		// Trim trailing whitespace
		line = strings.TrimRight(line, " ")
		lines[i] = line
	}
	
	return strings.Join(lines, "\n")
}

// ImportResolver resolves import conflicts
type ImportResolver struct {
	languages map[string]ImportSorter
}

func (ir *ImportResolver) Priority() int {
	return 90
}

func (ir *ImportResolver) CanResolve(ctx context.Context, conflict Conflict) bool {
	// Check if conflict is in import section
	lang := ir.detectLanguage(conflict.File)
	if sorter, ok := ir.languages[lang]; ok {
		return sorter.IsImportConflict(ctx, conflict)
	}
	return false
}

func (ir *ImportResolver) Resolve(ctx context.Context, conflict Conflict) (*Resolution, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	lang := ir.detectLanguage(conflict.File)
	sorter := ir.languages[lang]

	// Extract and merge imports
	imports1 := sorter.ExtractImports(ctx, conflict.Diff.Content1)
	imports2 := sorter.ExtractImports(ctx, conflict.Diff.Content2)

	merged := sorter.MergeImports(ctx, imports1, imports2)
	sorted := sorter.SortImports(ctx, merged)

	// Rebuild content with sorted imports
	resolved := sorter.ReplaceImports(ctx, conflict.Diff.Base, sorted)

	return &Resolution{
		Content:    resolved,
		Strategy:   "import_merge",
		Confidence: 0.95,
		ResolvedAt: time.Now(),
		Metadata: map[string]interface{}{
			"language":      lang,
			"imports_count": len(merged),
		},
	}, nil
}

func (ir *ImportResolver) detectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return "go"
	case ".js", ".jsx", ".ts", ".tsx":
		return "javascript"
	case ".py":
		return "python"
	default:
		return "unknown"
	}
}

// ImportSorter interface for language-specific import handling
type ImportSorter interface {
	IsImportConflict(ctx context.Context, conflict Conflict) bool
	ExtractImports(ctx context.Context, content string) []string
	MergeImports(ctx context.Context, imports1, imports2 []string) []string
	SortImports(ctx context.Context, imports []string) []string
	ReplaceImports(ctx context.Context, content string, imports []string) string
}

// GoImportSorter handles Go imports
type GoImportSorter struct{}

func (gis *GoImportSorter) IsImportConflict(ctx context.Context, conflict Conflict) bool {
	// Check if both contents have import statements
	return strings.Contains(conflict.Diff.Content1, "import") &&
		   strings.Contains(conflict.Diff.Content2, "import")
}

func (gis *GoImportSorter) ExtractImports(ctx context.Context, content string) []string {
	var imports []string
	lines := strings.Split(content, "\n")
	
	inImportBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		if strings.HasPrefix(trimmed, "import (") {
			inImportBlock = true
			continue
		}
		
		if inImportBlock && trimmed == ")" {
			inImportBlock = false
			continue
		}
		
		if inImportBlock || strings.HasPrefix(trimmed, "import ") {
			if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
				imports = append(imports, trimmed)
			}
		}
	}
	
	return imports
}

func (gis *GoImportSorter) MergeImports(ctx context.Context, imports1, imports2 []string) []string {
	importSet := make(map[string]struct{})
	var merged []string
	
	// Add all unique imports
	for _, imp := range append(imports1, imports2...) {
		cleaned := strings.TrimSpace(imp)
		if cleaned != "" && cleaned != "import (" && cleaned != ")" {
			if _, exists := importSet[cleaned]; !exists {
				importSet[cleaned] = struct{}{}
				merged = append(merged, cleaned)
			}
		}
	}
	
	return merged
}

func (gis *GoImportSorter) SortImports(ctx context.Context, imports []string) []string {
	// Separate standard library, third-party, and local imports
	var stdlib, thirdparty, local []string
	
	for _, imp := range imports {
		// Remove import prefix if present
		imp = strings.TrimPrefix(imp, "import ")
		imp = strings.TrimSpace(imp)
		
		if strings.Contains(imp, ".") {
			if strings.Contains(imp, "github.com") || strings.Contains(imp, "gitlab.com") {
				thirdparty = append(thirdparty, imp)
			} else {
				local = append(local, imp)
			}
		} else {
			stdlib = append(stdlib, imp)
		}
	}
	
	// Sort each group
	sort.Strings(stdlib)
	sort.Strings(thirdparty)
	sort.Strings(local)
	
	// Combine with proper formatting
	var sorted []string
	sorted = append(sorted, stdlib...)
	if len(thirdparty) > 0 {
		sorted = append(sorted, thirdparty...)
	}
	if len(local) > 0 {
		sorted = append(sorted, local...)
	}
	
	return sorted
}

func (gis *GoImportSorter) ReplaceImports(ctx context.Context, content string, imports []string) string {
	lines := strings.Split(content, "\n")
	var result []string
	
	inImportBlock := false
	importAdded := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		if strings.HasPrefix(trimmed, "import (") {
			inImportBlock = true
			result = append(result, "import (")
			for _, imp := range imports {
				result = append(result, "\t"+imp)
			}
			result = append(result, ")")
			importAdded = true
			continue
		}
		
		if inImportBlock && trimmed == ")" {
			inImportBlock = false
			continue
		}
		
		if inImportBlock {
			continue // Skip original import lines
		}
		
		if strings.HasPrefix(trimmed, "import ") && !importAdded {
			// Single import line, replace with block
			result = append(result, "import (")
			for _, imp := range imports {
				result = append(result, "\t"+imp)
			}
			result = append(result, ")")
			importAdded = true
			continue
		}
		
		result = append(result, line)
	}
	
	return strings.Join(result, "\n")
}

// JSImportSorter handles JavaScript/TypeScript imports
type JSImportSorter struct{}

func (jis *JSImportSorter) IsImportConflict(ctx context.Context, conflict Conflict) bool {
	return strings.Contains(conflict.Diff.Content1, "import") &&
		   strings.Contains(conflict.Diff.Content2, "import")
}

func (jis *JSImportSorter) ExtractImports(ctx context.Context, content string) []string {
	var imports []string
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") && strings.Contains(trimmed, "from") {
			imports = append(imports, trimmed)
		}
	}
	
	return imports
}

func (jis *JSImportSorter) MergeImports(ctx context.Context, imports1, imports2 []string) []string {
	importSet := make(map[string]struct{})
	var merged []string
	
	for _, imp := range append(imports1, imports2...) {
		if _, exists := importSet[imp]; !exists {
			importSet[imp] = struct{}{}
			merged = append(merged, imp)
		}
	}
	
	return merged
}

func (jis *JSImportSorter) SortImports(ctx context.Context, imports []string) []string {
	sort.Strings(imports)
	return imports
}

func (jis *JSImportSorter) ReplaceImports(ctx context.Context, content string, imports []string) string {
	lines := strings.Split(content, "\n")
	var result []string
	
	// Skip existing import lines and add sorted imports at the top
	for _, imp := range imports {
		result = append(result, imp)
	}
	
	// Add non-import lines
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "import ") || !strings.Contains(trimmed, "from") {
			result = append(result, line)
		}
	}
	
	return strings.Join(result, "\n")
}

// PythonImportSorter handles Python imports
type PythonImportSorter struct{}

func (pis *PythonImportSorter) IsImportConflict(ctx context.Context, conflict Conflict) bool {
	return (strings.Contains(conflict.Diff.Content1, "import ") || strings.Contains(conflict.Diff.Content1, "from ")) &&
		   (strings.Contains(conflict.Diff.Content2, "import ") || strings.Contains(conflict.Diff.Content2, "from "))
}

func (pis *PythonImportSorter) ExtractImports(ctx context.Context, content string) []string {
	var imports []string
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") {
			imports = append(imports, trimmed)
		}
	}
	
	return imports
}

func (pis *PythonImportSorter) MergeImports(ctx context.Context, imports1, imports2 []string) []string {
	importSet := make(map[string]struct{})
	var merged []string
	
	for _, imp := range append(imports1, imports2...) {
		if _, exists := importSet[imp]; !exists {
			importSet[imp] = struct{}{}
			merged = append(merged, imp)
		}
	}
	
	return merged
}

func (pis *PythonImportSorter) SortImports(ctx context.Context, imports []string) []string {
	// Separate standard library, third-party, and local imports
	var stdlib, thirdparty, local []string
	
	stdlibModules := map[string]bool{
		"os": true, "sys": true, "re": true, "json": true, "time": true,
		"datetime": true, "collections": true, "itertools": true,
	}
	
	for _, imp := range imports {
		if strings.HasPrefix(imp, "from ") {
			parts := strings.Fields(imp)
			if len(parts) >= 2 {
				module := parts[1]
				if stdlibModules[module] {
					stdlib = append(stdlib, imp)
				} else if strings.Contains(module, ".") {
					local = append(local, imp)
				} else {
					thirdparty = append(thirdparty, imp)
				}
			}
		} else if strings.HasPrefix(imp, "import ") {
			module := strings.TrimPrefix(imp, "import ")
			module = strings.Fields(module)[0]
			if stdlibModules[module] {
				stdlib = append(stdlib, imp)
			} else if strings.Contains(module, ".") {
				local = append(local, imp)
			} else {
				thirdparty = append(thirdparty, imp)
			}
		}
	}
	
	sort.Strings(stdlib)
	sort.Strings(thirdparty)
	sort.Strings(local)
	
	var sorted []string
	sorted = append(sorted, stdlib...)
	sorted = append(sorted, thirdparty...)
	sorted = append(sorted, local...)
	
	return sorted
}

func (pis *PythonImportSorter) ReplaceImports(ctx context.Context, content string, imports []string) string {
	lines := strings.Split(content, "\n")
	var result []string
	
	// Add sorted imports at the top
	for _, imp := range imports {
		result = append(result, imp)
	}
	
	// Add a blank line after imports
	if len(imports) > 0 {
		result = append(result, "")
	}
	
	// Add non-import lines
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "import ") && !strings.HasPrefix(trimmed, "from ") {
			result = append(result, line)
		}
	}
	
	return strings.Join(result, "\n")
}

// FormattingResolver resolves formatting conflicts
type FormattingResolver struct{}

func (fr *FormattingResolver) Priority() int {
	return 80
}

func (fr *FormattingResolver) CanResolve(ctx context.Context, conflict Conflict) bool {
	// Check if conflict is only formatting differences
	return fr.isFormattingOnly(conflict.Diff.Content1, conflict.Diff.Content2)
}

func (fr *FormattingResolver) Resolve(ctx context.Context, conflict Conflict) (*Resolution, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Apply consistent formatting and choose the better formatted version
	formatted1 := fr.normalizeFormatting(conflict.Diff.Content1)
	formatted2 := fr.normalizeFormatting(conflict.Diff.Content2)

	// Choose the version with better formatting
	content := formatted1
	if fr.hasConsistentFormatting(formatted2) {
		content = formatted2
	}

	return &Resolution{
		Content:    content,
		Strategy:   "formatting_normalization",
		Confidence: 0.9,
		ResolvedAt: time.Now(),
	}, nil
}

func (fr *FormattingResolver) isFormattingOnly(content1, content2 string) bool {
	// Remove all whitespace and compare
	clean1 := regexp.MustCompile(`\s+`).ReplaceAllString(content1, " ")
	clean2 := regexp.MustCompile(`\s+`).ReplaceAllString(content2, " ")
	return clean1 == clean2
}

func (fr *FormattingResolver) normalizeFormatting(content string) string {
	// Basic formatting normalization
	lines := strings.Split(content, "\n")
	var formatted []string
	
	for _, line := range lines {
		// Trim trailing whitespace
		line = strings.TrimRight(line, " \t")
		formatted = append(formatted, line)
	}
	
	return strings.Join(formatted, "\n")
}

func (fr *FormattingResolver) hasConsistentFormatting(content string) bool {
	lines := strings.Split(content, "\n")
	
	// Check for consistent indentation
	var indentSize int
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))
			if indentSize == 0 && leadingSpaces > 0 {
				indentSize = leadingSpaces
			}
			if leadingSpaces > 0 && leadingSpaces%indentSize != 0 {
				return false
			}
		}
	}
	
	return true
}

// CommentResolver resolves comment conflicts
type CommentResolver struct{}

func (cr *CommentResolver) Priority() int {
	return 70
}

func (cr *CommentResolver) CanResolve(ctx context.Context, conflict Conflict) bool {
	// Check if conflict is only in comments
	return cr.isCommentOnly(conflict.Diff.Content1, conflict.Diff.Content2)
}

func (cr *CommentResolver) Resolve(ctx context.Context, conflict Conflict) (*Resolution, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Merge comments from both versions
	merged := cr.mergeComments(conflict.Diff.Content1, conflict.Diff.Content2)

	return &Resolution{
		Content:    merged,
		Strategy:   "comment_merge",
		Confidence: 0.85,
		ResolvedAt: time.Now(),
	}, nil
}

func (cr *CommentResolver) isCommentOnly(content1, content2 string) bool {
	// Remove comments and compare
	clean1 := cr.removeComments(content1)
	clean2 := cr.removeComments(content2)
	return clean1 == clean2
}

func (cr *CommentResolver) removeComments(content string) string {
	// Remove single-line comments
	lines := strings.Split(content, "\n")
	var clean []string
	
	for _, line := range lines {
		// Remove // comments
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		// Remove # comments
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		clean = append(clean, strings.TrimSpace(line))
	}
	
	return strings.Join(clean, "\n")
}

func (cr *CommentResolver) mergeComments(content1, content2 string) string {
	// Simple comment merging - take content1 as base and add unique comments from content2
	return content1 // Simplified implementation
}

// SimpleLineResolver resolves simple line-based conflicts
type SimpleLineResolver struct{}

func (slr *SimpleLineResolver) Priority() int {
	return 60
}

func (slr *SimpleLineResolver) CanResolve(ctx context.Context, conflict Conflict) bool {
	// Can resolve if conflict involves only a few lines
	if conflict.Diff == nil {
		return false
	}
	return len(conflict.Diff.ConflictLines) <= 3
}

func (slr *SimpleLineResolver) Resolve(ctx context.Context, conflict Conflict) (*Resolution, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// For simple conflicts, try to merge by taking the longer or more specific version
	content1 := conflict.Diff.Content1
	content2 := conflict.Diff.Content2

	var resolvedContent string
	if len(content1) > len(content2) {
		resolvedContent = content1
	} else {
		resolvedContent = content2
	}

	return &Resolution{
		Content:    resolvedContent,
		Strategy:   "simple_line_merge",
		Confidence: 0.6,
		ResolvedAt: time.Now(),
	}, nil
}

// MLResolver provides ML-based conflict resolution
type MLResolver struct {
	model    *ConflictModel
	features *FeatureExtractor
}

// NewMLResolver creates a new ML resolver
func NewMLResolver(ctx context.Context) (*MLResolver, error) {
	// Placeholder for ML model initialization
	return &MLResolver{
		model:    &ConflictModel{},
		features: &FeatureExtractor{},
	}, nil
}

// Suggest provides ML-based resolution suggestions
func (mlr *MLResolver) Suggest(ctx context.Context, conflict Conflict) *Resolution {
	if ctx.Err() != nil {
		return nil
	}

	// Extract features from the conflict
	features := mlr.features.Extract(ctx, conflict)

	// Get similar historical resolutions
	similar := mlr.model.FindSimilar(ctx, features, 10)

	if len(similar) == 0 {
		return nil
	}

	// Apply learned patterns
	resolution := mlr.applyPattern(ctx, conflict, similar[0].Pattern)

	return &Resolution{
		Content:    resolution,
		Strategy:   "ml_pattern",
		Confidence: similar[0].Similarity,
		Pattern:    similar[0].Pattern,
		ResolvedAt: time.Now(),
	}
}

func (mlr *MLResolver) applyPattern(ctx context.Context, conflict Conflict, pattern *ResolutionPattern) string {
	// Simplified pattern application
	return conflict.Diff.Content1 // Placeholder
}

// ConflictModel represents an ML model for conflict resolution
type ConflictModel struct{}

// SimilarResolution represents a similar historical resolution
type SimilarResolution struct {
	Pattern    *ResolutionPattern `json:"pattern"`
	Similarity float64            `json:"similarity"`
}

func (cm *ConflictModel) FindSimilar(ctx context.Context, features map[string]interface{}, limit int) []SimilarResolution {
	// Placeholder for ML model inference
	return []SimilarResolution{}
}

// FeatureExtractor extracts features from conflicts for ML
type FeatureExtractor struct{}

func (fe *FeatureExtractor) Extract(ctx context.Context, conflict Conflict) map[string]interface{} {
	features := make(map[string]interface{})
	
	// Basic features
	features["file_extension"] = filepath.Ext(conflict.File)
	features["conflict_type"] = conflict.Type
	features["severity"] = conflict.Severity
	
	if conflict.Diff != nil {
		features["content1_length"] = len(conflict.Diff.Content1)
		features["content2_length"] = len(conflict.Diff.Content2)
		features["conflict_lines"] = len(conflict.Diff.ConflictLines)
	}
	
	return features
}

// ResolutionHistory tracks resolution history for learning
type ResolutionHistory struct {
	resolutions []HistoricalResolution
	mu          sync.RWMutex
}

// HistoricalResolution represents a past resolution
type HistoricalResolution struct {
	Conflict   Conflict           `json:"conflict"`
	Resolution *Resolution        `json:"resolution"`
	Strategy   ResolutionStrategy `json:"-"`
	Timestamp  time.Time          `json:"timestamp"`
	Success    bool               `json:"success"`
}

// NewResolutionHistory creates a new resolution history
func NewResolutionHistory() *ResolutionHistory {
	return &ResolutionHistory{
		resolutions: make([]HistoricalResolution, 0),
	}
}

// Record records a resolution in history
func (rh *ResolutionHistory) Record(conflict Conflict, resolution *Resolution, strategy ResolutionStrategy) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	
	rh.resolutions = append(rh.resolutions, HistoricalResolution{
		Conflict:   conflict,
		Resolution: resolution,
		Strategy:   strategy,
		Timestamp:  time.Now(),
		Success:    true, // Assume success for now
	})
}

// ManualResolver handles manual conflict resolution
type ManualResolver struct {
	ui       *ResolutionUI
	reviewer *CodeReviewer
}

// NewManualResolver creates a new manual resolver
func NewManualResolver() *ManualResolver {
	return &ManualResolver{
		ui:       NewResolutionUI(),
		reviewer: NewCodeReviewer(),
	}
}

// RequestManualResolution requests manual intervention for conflict resolution
func (mr *ManualResolver) RequestManualResolution(ctx context.Context, conflict Conflict) (*Resolution, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Create resolution request
	request := &ResolutionRequest{
		ID:        mr.generateRequestID(),
		Conflict:  conflict,
		CreatedAt: time.Now(),
	}

	// Show in UI (placeholder)
	mr.ui.ShowConflict(ctx, request)

	// Wait for resolution with timeout
	resolution, err := mr.waitForResolution(ctx, request.ID, 30*time.Minute)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeTimeout, "manual resolution timeout", nil).
			WithComponent("git.worktree.resolver").
			WithOperation("RequestManualResolution")
	}

	// Validate resolution
	if err := mr.reviewer.ValidateResolution(ctx, conflict, resolution); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidArgument, "resolution validation failed", nil).
			WithComponent("git.worktree.resolver").
			WithOperation("RequestManualResolution")
	}

	return resolution, nil
}

func (mr *ManualResolver) generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func (mr *ManualResolver) waitForResolution(ctx context.Context, requestID string, timeout time.Duration) (*Resolution, error) {
	// Placeholder for waiting for manual resolution
	// In practice, this would wait for user input through the UI
	
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, gerror.New(gerror.ErrCodeTimeout, "manual resolution timeout", nil)
	}
}

// ResolutionRequest represents a manual resolution request
type ResolutionRequest struct {
	ID        string    `json:"id"`
	Conflict  Conflict  `json:"conflict"`
	CreatedAt time.Time `json:"created_at"`
}

// ResolutionUI handles the user interface for manual resolution
type ResolutionUI struct{}

func NewResolutionUI() *ResolutionUI {
	return &ResolutionUI{}
}

func (rui *ResolutionUI) ShowConflict(ctx context.Context, request *ResolutionRequest) {
	// Placeholder for UI display
	fmt.Printf("Manual resolution required for conflict: %s\n", request.Conflict.ID)
}

// CodeReviewer validates manual resolutions
type CodeReviewer struct{}

func NewCodeReviewer() *CodeReviewer {
	return &CodeReviewer{}
}

func (cr *CodeReviewer) ValidateResolution(ctx context.Context, conflict Conflict, resolution *Resolution) error {
	// Basic validation of manual resolution
	if resolution.Content == "" {
		return gerror.New(gerror.ErrCodeInvalidArgument, "resolution content is empty", nil)
	}
	
	// Check for remaining conflict markers
	if strings.Contains(resolution.Content, "<<<<<<<") ||
	   strings.Contains(resolution.Content, "=======") ||
	   strings.Contains(resolution.Content, ">>>>>>>") {
		return gerror.New(gerror.ErrCodeInvalidArgument, "resolution contains conflict markers", nil)
	}
	
	return nil
}