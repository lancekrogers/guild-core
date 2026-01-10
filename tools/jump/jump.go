// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package jump

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/sahilm/fuzzy"
	_ "modernc.org/sqlite"
)

// Jump implements frecency-based directory jumping
type Jump struct {
	db *sql.DB
}

// New creates a new Jump instance with the given database path
func New(dbPath string) (*Jump, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create directory").
			WithComponent("jump").
			WithOperation("new")
	}

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to open database").
			WithComponent("jump").
			WithOperation("new")
	}

	// Create table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS visits (
		dir  TEXT PRIMARY KEY,
		freq INTEGER NOT NULL,
		last INTEGER NOT NULL
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create table").
			WithComponent("jump").
			WithOperation("new")
	}

	return &Jump{db: db}, nil
}

// Close closes the database connection
func (j *Jump) Close() error {
	if j.db != nil {
		return j.db.Close()
	}
	return nil
}

// Track records a visit to a directory
func (j *Jump) Track(dir string) error {
	// Clean and validate the path
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid directory path").
			WithComponent("jump").
			WithOperation("track")
	}

	// Check if directory exists
	if info, err := os.Stat(absPath); err != nil || !info.IsDir() {
		if os.IsNotExist(err) {
			return gerror.Newf(gerror.ErrCodeNotFound, "directory does not exist: %s", absPath).
				WithComponent("jump").
				WithOperation("track")
		}
		return gerror.Newf(gerror.ErrCodeInvalidInput, "not a directory: %s", absPath).
			WithComponent("jump").
			WithOperation("track")
	}

	// Update or insert the visit record
	// Use UnixMilli for better timestamp precision
	now := time.Now().UnixMilli()
	upsertSQL := `
	INSERT INTO visits (dir, freq, last) VALUES (?, 1, ?)
	ON CONFLICT(dir) DO UPDATE SET 
		freq = freq + 1,
		last = ?
	WHERE dir = ?;`

	if _, err := j.db.Exec(upsertSQL, absPath, now, now, absPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to track visit").
			WithComponent("jump").
			WithOperation("track")
	}

	// Clean up removed directories
	go j.cleanupRemovedDirs()

	return nil
}

// Find searches for directories matching the query and returns the best match
func (j *Jump) Find(query string) (string, error) {
	if query == "" {
		return "", gerror.New(gerror.ErrCodeInvalidInput, "query cannot be empty", nil).
			WithComponent("jump").
			WithOperation("find")
	}

	// Get all directories with their scores
	entries, err := j.getAllEntries()
	if err != nil {
		return "", err
	}

	if len(entries) == 0 {
		return "", gerror.New(gerror.ErrCodeNotFound, "no directories tracked", nil).
			WithComponent("jump").
			WithOperation("find")
	}

	// Calculate frecency scores and prepare for fuzzy matching
	var candidates []string
	scoreMap := make(map[string]float64)
	now := time.Now()

	for _, entry := range entries {
		// Convert milliseconds timestamp to time
		lastVisit := time.UnixMilli(entry.Last)
		hoursSinceLast := now.Sub(lastVisit).Hours()
		score := float64(entry.Freq) / (1 + hoursSinceLast)
		scoreMap[entry.Dir] = score
		candidates = append(candidates, entry.Dir)
	}

	// Perform fuzzy matching on base names
	baseCandidates := make([]string, len(candidates))
	for i, path := range candidates {
		baseCandidates[i] = filepath.Base(path)
	}

	matches := fuzzy.Find(query, baseCandidates)
	if len(matches) == 0 {
		// Try full path matching as fallback
		matches = fuzzy.Find(query, candidates)
		if len(matches) == 0 {
			return "", gerror.Newf(gerror.ErrCodeNotFound, "no match for query: %s", query).
				WithComponent("jump").
				WithOperation("find")
		}
		// For full path matches, use the matched string directly
		bestMatch := ""
		bestScore := -1.0
		for _, match := range matches {
			// Combine fuzzy match score with frecency score
			combinedScore := float64(match.Score) * scoreMap[match.Str]
			if combinedScore > bestScore {
				bestScore = combinedScore
				bestMatch = match.Str
			}
		}
		return bestMatch, nil
	}

	// Sort matches by combined fuzzy score and frecency score
	bestMatch := ""
	bestScore := -1.0

	// Match indices correspond to candidates array
	for _, match := range matches {
		fullPath := candidates[match.Index]
		// Combine fuzzy match score with frecency score
		combinedScore := float64(match.Score) * scoreMap[fullPath]
		if combinedScore > bestScore {
			bestScore = combinedScore
			bestMatch = fullPath
		}
	}

	return bestMatch, nil
}

// Recent returns the n most recently visited directories
func (j *Jump) Recent(n int) ([]string, error) {
	if n <= 0 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "n must be positive", nil).
			WithComponent("jump").
			WithOperation("recent")
	}

	// Explicitly include all visits, sorted by last visit time descending
	query := `SELECT dir, last FROM visits ORDER BY last DESC`
	rows, err := j.db.Query(query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to query recent directories").
			WithComponent("jump").
			WithOperation("recent")
	}
	defer rows.Close()

	var dirs []string
	count := 0
	for rows.Next() && count < n {
		var dir string
		var lastVisit int64
		if err := rows.Scan(&dir, &lastVisit); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to scan row").
				WithComponent("jump").
				WithOperation("recent")
		}
		// Only include existing directories
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			dirs = append(dirs, dir)
			count++
		}
	}

	return dirs, nil
}

// visitEntry represents a directory visit record
type visitEntry struct {
	Dir  string
	Freq int
	Last int64
}

// getAllEntries returns all visit entries from the database
func (j *Jump) getAllEntries() ([]visitEntry, error) {
	query := `SELECT dir, freq, last FROM visits`
	rows, err := j.db.Query(query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to query visits").
			WithComponent("jump").
			WithOperation("getAllEntries")
	}
	defer rows.Close()

	var entries []visitEntry
	for rows.Next() {
		var entry visitEntry
		if err := rows.Scan(&entry.Dir, &entry.Freq, &entry.Last); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to scan row").
				WithComponent("jump").
				WithOperation("getAllEntries")
		}
		// Only include existing directories
		if info, err := os.Stat(entry.Dir); err == nil && info.IsDir() {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// cleanupRemovedDirs removes entries for directories that no longer exist
func (j *Jump) cleanupRemovedDirs() {
	// Get all directories
	query := `SELECT dir FROM visits`
	rows, err := j.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()

	var toRemove []string
	for rows.Next() {
		var dir string
		if err := rows.Scan(&dir); err != nil {
			continue
		}
		// Check if directory exists
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			toRemove = append(toRemove, dir)
		}
	}

	// Remove non-existent directories
	for _, dir := range toRemove {
		j.db.Exec("DELETE FROM visits WHERE dir = ?", dir)
	}
}

// defaultJumpFactory is a variable that can be overridden for testing
var defaultJumpFactory = func() (*Jump, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get home directory").
			WithComponent("jump").
			WithOperation("getDefaultJump")
	}

	dbPath := filepath.Join(homeDir, ".guild", "jump.db")
	return New(dbPath)
}

// GetDefaultJump returns a Jump instance using the default database location
func GetDefaultJump() (*Jump, error) {
	return defaultJumpFactory()
}

// QuickTrack is a convenience function to track a directory using the default database
func QuickTrack(dir string) error {
	j, err := GetDefaultJump()
	if err != nil {
		return err
	}
	defer j.Close()
	return j.Track(dir)
}

// QuickFind is a convenience function to find a directory using the default database
func QuickFind(query string) (string, error) {
	j, err := GetDefaultJump()
	if err != nil {
		return "", err
	}
	defer j.Close()
	return j.Find(query)
}

// QuickRecent is a convenience function to get recent directories using the default database
func QuickRecent(n int) ([]string, error) {
	j, err := GetDefaultJump()
	if err != nil {
		return nil, err
	}
	defer j.Close()
	return j.Recent(n)
}
