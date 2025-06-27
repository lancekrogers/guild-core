// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package search

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// FileResult represents a file search result with fuzzy matching score
type FileResult struct {
	Path        string    `json:"path"`         // Relative file path
	AbsPath     string    `json:"abs_path"`     // Absolute file path
	Name        string    `json:"name"`         // File name only
	Score       int       `json:"score"`        // Fuzzy match score (lower is better)
	IsRecent    bool      `json:"is_recent"`    // Whether file was recently accessed
	LastAccess  time.Time `json:"last_access"`  // Last access time
	Size        int64     `json:"size"`         // File size in bytes
	IsDirectory bool      `json:"is_directory"` // Whether this is a directory
}

// FuzzyFinder provides fuzzy file finding capabilities
type FuzzyFinder struct {
	workingDir      string
	excludePatterns []string
	recentFiles     map[string]time.Time
	maxResults      int
	indexCache      []string
	lastIndexTime   time.Time
	indexDuration   time.Duration
}

// FuzzyFinderConfig configures the fuzzy finder
type FuzzyFinderConfig struct {
	WorkingDir      string        `json:"working_dir"`      // Working directory to search
	ExcludePatterns []string      `json:"exclude_patterns"` // Patterns to exclude
	MaxResults      int           `json:"max_results"`      // Maximum results to return
	IndexTimeout    time.Duration `json:"index_timeout"`    // Timeout for indexing
}

// DefaultExcludePatterns returns common patterns to exclude from file searches
func DefaultExcludePatterns() []string {
	return []string{
		".git/**",
		".git/*",
		"node_modules/**",
		"node_modules/*",
		"vendor/**",
		"vendor/*",
		".vscode/**",
		".vscode/*",
		".idea/**",
		".idea/*",
		"*.tmp",
		"*.temp",
		"*.log",
		"*.swp",
		"*.swo",
		"*~",
		".DS_Store",
		"Thumbs.db",
		"*.exe",
		"*.dll",
		"*.so",
		"*.dylib",
		"*.o",
		"*.a",
		"__pycache__/**",
		"__pycache__/*",
		"*.pyc",
		"*.pyo",
		".pytest_cache/**",
		".pytest_cache/*",
		"target/**", // Rust
		"target/*",
		"dist/**", // JavaScript/TypeScript
		"dist/*",
		"build/**", // Various build outputs
		"build/*",
		"bin/**",
		"bin/*",
		".next/**", // Next.js
		".next/*",
		".nuxt/**", // Nuxt.js
		".nuxt/*",
		"coverage/**", // Test coverage
		"coverage/*",
		".nyc_output/**",
		".nyc_output/*",
	}
}

// NewFuzzyFinder creates a new fuzzy file finder
func NewFuzzyFinder(config FuzzyFinderConfig) *FuzzyFinder {
	if config.WorkingDir == "" {
		config.WorkingDir, _ = os.Getwd()
	}
	if config.MaxResults == 0 {
		config.MaxResults = 50
	}
	if config.IndexTimeout == 0 {
		config.IndexTimeout = 5 * time.Second
	}
	if len(config.ExcludePatterns) == 0 {
		config.ExcludePatterns = DefaultExcludePatterns()
	}

	return &FuzzyFinder{
		workingDir:      config.WorkingDir,
		excludePatterns: config.ExcludePatterns,
		recentFiles:     make(map[string]time.Time),
		maxResults:      config.MaxResults,
		indexDuration:   config.IndexTimeout,
	}
}

// Search performs fuzzy search for files matching the given pattern
func (ff *FuzzyFinder) Search(ctx context.Context, pattern string) ([]FileResult, error) {
	// Rebuild index if necessary
	if err := ff.rebuildIndexIfNeeded(ctx); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to rebuild file index").
			WithComponent("fuzzy_finder").
			WithOperation("search")
	}

	// If pattern is empty, return recent files
	if strings.TrimSpace(pattern) == "" {
		return ff.getRecentFiles(), nil
	}

	// Perform fuzzy matching
	var results []FileResult
	pattern = strings.ToLower(pattern)

	for _, filePath := range ff.indexCache {
		score := ff.fuzzyScore(filePath, pattern)
		if score >= 0 {
			result, err := ff.createFileResult(filePath, score)
			if err != nil {
				continue // Skip files that can't be accessed
			}
			results = append(results, result)
		}
	}

	// Sort by score (lower is better), then by recent access
	sort.Slice(results, func(i, j int) bool {
		if results[i].IsRecent && !results[j].IsRecent {
			return true
		}
		if !results[i].IsRecent && results[j].IsRecent {
			return false
		}
		if results[i].Score != results[j].Score {
			return results[i].Score < results[j].Score
		}
		return results[i].LastAccess.After(results[j].LastAccess)
	})

	// Limit results
	if len(results) > ff.maxResults {
		results = results[:ff.maxResults]
	}

	return results, nil
}

// MarkRecentFile marks a file as recently accessed
func (ff *FuzzyFinder) MarkRecentFile(filePath string) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return
	}

	relPath, err := filepath.Rel(ff.workingDir, absPath)
	if err != nil {
		return
	}

	ff.recentFiles[relPath] = time.Now()

	// Keep only the last 20 recent files
	if len(ff.recentFiles) > 20 {
		// Find oldest file and remove it
		oldestPath := ""
		oldestTime := time.Now()
		for path, accessTime := range ff.recentFiles {
			if accessTime.Before(oldestTime) {
				oldestTime = accessTime
				oldestPath = path
			}
		}
		if oldestPath != "" {
			delete(ff.recentFiles, oldestPath)
		}
	}
}

// GetRecentFiles returns recently accessed files
func (ff *FuzzyFinder) GetRecentFiles() []FileResult {
	return ff.getRecentFiles()
}

// rebuildIndexIfNeeded rebuilds the file index if it's stale
func (ff *FuzzyFinder) rebuildIndexIfNeeded(ctx context.Context) error {
	// Check if index needs rebuilding (older than 30 seconds)
	if time.Since(ff.lastIndexTime) < 30*time.Second && len(ff.indexCache) > 0 {
		return nil
	}

	// Create context with timeout
	indexCtx, cancel := context.WithTimeout(ctx, ff.indexDuration)
	defer cancel()

	var files []string
	err := filepath.WalkDir(ff.workingDir, func(path string, d fs.DirEntry, err error) error {
		// Check for cancellation
		select {
		case <-indexCtx.Done():
			return indexCtx.Err()
		default:
		}

		if err != nil {
			return nil // Skip files with errors
		}

		// Get relative path
		relPath, err := filepath.Rel(ff.workingDir, path)
		if err != nil {
			return nil
		}

		// Check if file should be excluded
		if ff.shouldExclude(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		files = append(files, relPath)
		return nil
	})

	if err != nil && err != context.DeadlineExceeded {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to index files").
			WithComponent("fuzzy_finder").
			WithOperation("rebuild_index")
	}

	ff.indexCache = files
	ff.lastIndexTime = time.Now()
	return nil
}

// shouldExclude checks if a file path should be excluded based on patterns
func (ff *FuzzyFinder) shouldExclude(path string) bool {
	for _, pattern := range ff.excludePatterns {
		// Simple pattern matching - can be enhanced with proper glob matching
		if strings.Contains(pattern, "**") {
			// Handle recursive patterns
			cleanPattern := strings.ReplaceAll(pattern, "**", "*")
			cleanPattern = strings.ReplaceAll(cleanPattern, "/*", "")
			if strings.HasPrefix(path, cleanPattern) {
				return true
			}
		} else if strings.Contains(pattern, "*") {
			// Handle wildcard patterns
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				return true
			}
			if matched, _ := filepath.Match(pattern, path); matched {
				return true
			}
		} else {
			// Exact match or prefix match
			if path == pattern || strings.HasPrefix(path, pattern+"/") {
				return true
			}
		}
	}
	return false
}

// fuzzyScore calculates a fuzzy matching score for a file path
// Returns -1 if no match, otherwise returns score (lower is better)
func (ff *FuzzyFinder) fuzzyScore(filePath, pattern string) int {
	fileName := strings.ToLower(filepath.Base(filePath))
	fullPath := strings.ToLower(filePath)

	// Exact match on filename gets highest priority
	if fileName == pattern {
		return 0
	}

	// Prefix match on filename
	if strings.HasPrefix(fileName, pattern) {
		return 1
	}

	// Substring match on filename
	if strings.Contains(fileName, pattern) {
		return 2
	}

	// Exact match on full path
	if fullPath == pattern {
		return 3
	}

	// Prefix match on full path
	if strings.HasPrefix(fullPath, pattern) {
		return 4
	}

	// Substring match on full path
	if strings.Contains(fullPath, pattern) {
		return 5
	}

	// Fuzzy character sequence matching
	score := ff.fuzzySequenceScore(fileName, pattern)
	if score >= 0 {
		return 10 + score
	}

	// Try fuzzy matching on full path
	score = ff.fuzzySequenceScore(fullPath, pattern)
	if score >= 0 {
		return 20 + score
	}

	return -1 // No match
}

// fuzzySequenceScore calculates fuzzy sequence matching score
func (ff *FuzzyFinder) fuzzySequenceScore(text, pattern string) int {
	if len(pattern) == 0 {
		return 0
	}

	textIdx := 0
	patternIdx := 0
	score := 0
	lastMatch := -1

	for patternIdx < len(pattern) && textIdx < len(text) {
		if text[textIdx] == pattern[patternIdx] {
			if lastMatch >= 0 {
				// Add distance between matches to score
				score += textIdx - lastMatch - 1
			}
			lastMatch = textIdx
			patternIdx++
		}
		textIdx++
	}

	// If we didn't match all pattern characters, no match
	if patternIdx < len(pattern) {
		return -1
	}

	return score
}

// createFileResult creates a FileResult from a file path
func (ff *FuzzyFinder) createFileResult(relPath string, score int) (FileResult, error) {
	absPath := filepath.Join(ff.workingDir, relPath)

	stat, err := os.Stat(absPath)
	if err != nil {
		return FileResult{}, err
	}

	lastAccess := stat.ModTime()
	if recentTime, exists := ff.recentFiles[relPath]; exists {
		lastAccess = recentTime
	}

	return FileResult{
		Path:        relPath,
		AbsPath:     absPath,
		Name:        filepath.Base(relPath),
		Score:       score,
		IsRecent:    ff.isRecentFile(relPath),
		LastAccess:  lastAccess,
		Size:        stat.Size(),
		IsDirectory: stat.IsDir(),
	}, nil
}

// isRecentFile checks if a file is in the recent files list
func (ff *FuzzyFinder) isRecentFile(path string) bool {
	_, exists := ff.recentFiles[path]
	return exists
}

// getRecentFiles returns recent files as FileResults
func (ff *FuzzyFinder) getRecentFiles() []FileResult {
	var results []FileResult

	for path, accessTime := range ff.recentFiles {
		result, err := ff.createFileResult(path, 0)
		if err != nil {
			continue
		}
		result.LastAccess = accessTime
		result.IsRecent = true
		results = append(results, result)
	}

	// Sort by last access time (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].LastAccess.After(results[j].LastAccess)
	})

	return results
}

// RefreshIndex forces a refresh of the file index
func (ff *FuzzyFinder) RefreshIndex(ctx context.Context) error {
	ff.lastIndexTime = time.Time{} // Force rebuild
	return ff.rebuildIndexIfNeeded(ctx)
}

// GetIndexStats returns statistics about the file index
func (ff *FuzzyFinder) GetIndexStats() map[string]interface{} {
	return map[string]interface{}{
		"total_files":   len(ff.indexCache),
		"recent_files":  len(ff.recentFiles),
		"last_indexed":  ff.lastIndexTime,
		"working_dir":   ff.workingDir,
		"exclude_count": len(ff.excludePatterns),
	}
}
