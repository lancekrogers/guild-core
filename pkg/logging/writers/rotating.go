package writers

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// RotatingWriter implements log rotation based on size and age
type RotatingWriter struct {
	baseDir    string
	baseName   string
	maxSize    int64 // Max size per file in bytes
	maxAge     int   // Days to keep
	maxBackups int   // Number of backups
	compress   bool  // Gzip old logs

	mu          sync.Mutex
	currentFile *os.File
	currentSize int64
}

// NewRotatingWriter creates a new rotating log writer
func NewRotatingWriter(filePath string, maxSize int64, maxAge, maxBackups int, compress bool) (*RotatingWriter, error) {
	if filePath == "" {
		return nil, gerror.New(gerror.ErrCodeMissingRequired, "file path required", nil)
	}

	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)

	// Create directory if needed
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create log directory").
			WithDetails("dir", dir)
	}

	w := &RotatingWriter{
		baseDir:    dir,
		baseName:   base,
		maxSize:    maxSize,
		maxAge:     maxAge,
		maxBackups: maxBackups,
		compress:   compress,
	}

	// Open initial file
	if err := w.openFile(); err != nil {
		return nil, err
	}

	// Start cleanup goroutine
	go w.cleanupLoop()

	return w, nil
}

// Write implements io.Writer
func (w *RotatingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if rotation is needed
	if w.currentSize+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to rotate log")
		}
	}

	// Write to current file
	n, err = w.currentFile.Write(p)
	if err != nil {
		return n, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write log")
	}

	w.currentSize += int64(n)
	return n, nil
}

// Close closes the current log file
func (w *RotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentFile != nil {
		return w.currentFile.Close()
	}
	return nil
}

// openFile opens a new log file
func (w *RotatingWriter) openFile() error {
	path := filepath.Join(w.baseDir, w.baseName)

	// Check if file exists and get its size
	if info, err := os.Stat(path); err == nil {
		w.currentSize = info.Size()
	}

	// Open or create file
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to open log file").
			WithDetails("path", path)
	}

	w.currentFile = file
	return nil
}

// rotate performs log rotation
func (w *RotatingWriter) rotate() error {
	// Close current file
	if w.currentFile != nil {
		if err := w.currentFile.Close(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to close current log")
		}
	}

	// Generate rotation name with timestamp
	timestamp := time.Now().Format("20060102-150405")
	ext := filepath.Ext(w.baseName)
	nameWithoutExt := w.baseName[:len(w.baseName)-len(ext)]
	rotatedName := fmt.Sprintf("%s-%s%s", nameWithoutExt, timestamp, ext)
	rotatedPath := filepath.Join(w.baseDir, rotatedName)

	// Rename current file
	currentPath := filepath.Join(w.baseDir, w.baseName)
	if err := os.Rename(currentPath, rotatedPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to rotate log file").
			WithDetails("from", currentPath).
			WithDetails("to", rotatedPath)
	}

	// Compress if enabled
	if w.compress {
		go w.compressFile(rotatedPath)
	}

	// Open new file
	if err := w.openFile(); err != nil {
		return err
	}

	// Reset size
	w.currentSize = 0

	// Cleanup old files
	go w.cleanup()

	return nil
}

// compressFile compresses a log file
func (w *RotatingWriter) compressFile(path string) {
	gzPath := path + ".gz"

	// Open source file
	src, err := os.Open(path)
	if err != nil {
		return
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(gzPath)
	if err != nil {
		return
	}
	defer dst.Close()

	// Create gzip writer
	gz := gzip.NewWriter(dst)
	gz.Name = filepath.Base(path)
	gz.ModTime = time.Now()
	defer gz.Close()

	// Copy data
	if _, err := io.Copy(gz, src); err != nil {
		os.Remove(gzPath)
		return
	}

	// Remove original file
	os.Remove(path)
}

// cleanup removes old log files
func (w *RotatingWriter) cleanup() {
	// Get all log files
	pattern := w.baseName[:len(w.baseName)-len(filepath.Ext(w.baseName))] + "-*"
	matches, err := filepath.Glob(filepath.Join(w.baseDir, pattern))
	if err != nil {
		return
	}

	// Include compressed files
	gzPattern := pattern + ".gz"
	gzMatches, err := filepath.Glob(filepath.Join(w.baseDir, gzPattern))
	if err == nil {
		matches = append(matches, gzMatches...)
	}

	// Sort by modification time
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	files := make([]fileInfo, 0, len(matches))
	cutoff := time.Now().AddDate(0, 0, -w.maxAge)

	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		// Remove files older than maxAge
		if w.maxAge > 0 && info.ModTime().Before(cutoff) {
			os.Remove(match)
			continue
		}

		files = append(files, fileInfo{
			path:    match,
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	// Remove excess backups
	if w.maxBackups > 0 && len(files) > w.maxBackups {
		for i := w.maxBackups; i < len(files); i++ {
			os.Remove(files[i].path)
		}
	}
}

// cleanupLoop runs periodic cleanup
func (w *RotatingWriter) cleanupLoop() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		w.cleanup()
	}
}
