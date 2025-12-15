package scaffoldlite

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	Exists(path string) bool
	IsDir(path string) bool
	Remove(path string) error
	RemoveAll(path string) error
}

type OSFileSystem struct {
	basePath string
	mu       sync.RWMutex
}

func NewOSFileSystem(basePath string) (*OSFileSystem, error) {
	if basePath != "" {
		absPath, err := filepath.Abs(basePath)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get absolute path").WithDetails("basePath", basePath)
		}
		basePath = absPath
	}
	return &OSFileSystem{basePath: basePath}, nil
}

func (osfs *OSFileSystem) ReadFile(path string) ([]byte, error) {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(safePath)
	if err != nil {
		return nil, ErrFileRead(path, err)
	}
	return data, nil
}

func (osfs *OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return err
	}
	if osfs.Exists(path) {
		return ErrFileExists(path)
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := osfs.MkdirAll(dir, 0o755); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create directory").WithDetails("dir", dir).WithDetails("file", path)
		}
	}
	return osfs.writeFileAtomic(safePath, data, perm)
}

func (osfs *OSFileSystem) writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".tmp-scaffold-*")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create temporary file").WithDetails("directory", dir)
	}
	tmpPath := tmpFile.Name()
	cleanup := func() { tmpFile.Close(); os.Remove(tmpPath) }
	if _, err := tmpFile.Write(data); err != nil {
		cleanup()
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write to temporary file").WithDetails("tempFile", tmpPath)
	}
	if err := tmpFile.Sync(); err != nil {
		cleanup()
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to sync temporary file").WithDetails("tempFile", tmpPath)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to close temporary file").WithDetails("tempFile", tmpPath)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		os.Remove(tmpPath)
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to set file permissions").WithDetails("tempFile", tmpPath).WithDetails("permissions", perm)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return ErrFileWrite(path, err)
	}
	return nil
}

func (osfs *OSFileSystem) Stat(path string) (os.FileInfo, error) {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(safePath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to stat file").WithDetails("path", path)
	}
	return info, nil
}

func (osfs *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if (path == "." || path == "") && osfs.basePath != "" {
		if err := os.MkdirAll(osfs.basePath, perm); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create base directory").WithDetails("path", osfs.basePath).WithDetails("permissions", perm)
		}
		return nil
	}
	safePath, err := osfs.safePath(path)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to validate path for MkdirAll").WithDetails("input_path", path).WithDetails("base_path", osfs.basePath)
	}
	if err := os.MkdirAll(safePath, perm); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create directories").WithDetails("path", path).WithDetails("permissions", perm)
	}
	return nil
}

func (osfs *OSFileSystem) Exists(path string) bool {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return false
	}
	_, err = os.Stat(safePath)
	return err == nil
}

func (osfs *OSFileSystem) IsDir(path string) bool {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return false
	}
	info, err := os.Stat(safePath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (osfs *OSFileSystem) Remove(path string) error {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return err
	}
	if err := os.Remove(safePath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to remove file").WithDetails("path", path)
	}
	return nil
}

func (osfs *OSFileSystem) RemoveAll(path string) error {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(safePath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to remove directory").WithDetails("path", path)
	}
	return nil
}

func (osfs *OSFileSystem) safePath(path string) (string, error) {
	osfs.mu.RLock()
	basePath := osfs.basePath
	osfs.mu.RUnlock()
	cleanPath := filepath.Clean(path)
	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, string(filepath.Separator)+"..") {
		return "", ErrInvalidPath(path, "path traversal not allowed")
	}
	if basePath == "" {
		if filepath.IsAbs(cleanPath) {
			return cleanPath, nil
		}
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to resolve path").WithDetails("path", path)
		}
		return absPath, nil
	}
	if filepath.IsAbs(cleanPath) {
		return "", ErrInvalidPath(path, "absolute paths not allowed when base path is set")
	}
	fullPath := filepath.Join(basePath, cleanPath)
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to resolve full path").WithDetails("path", path)
	}
	baseWithSep := basePath
	if !strings.HasSuffix(baseWithSep, string(filepath.Separator)) {
		baseWithSep = basePath + string(filepath.Separator)
	}
	if absFullPath != basePath && !strings.HasPrefix(absFullPath, baseWithSep) {
		return "", gerror.New(gerror.ErrCodeInvalidInput, "path escapes base directory", nil).WithDetails("path", path).WithDetails("cleanPath", cleanPath).WithDetails("absFullPath", absFullPath).WithDetails("basePath", basePath)
	}
	return absFullPath, nil
}

// Memory FS (only needed for info/validation convenience)
type MemoryFileSystem struct {
	files map[string][]byte
	dirs  map[string]bool
	mu    sync.RWMutex
}

func NewMemoryFileSystem() *MemoryFileSystem {
	return &MemoryFileSystem{files: make(map[string][]byte), dirs: make(map[string]bool)}
}

func (mfs *MemoryFileSystem) ReadFile(path string) ([]byte, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()
	cleanPath := filepath.Clean(path)
	data, exists := mfs.files[cleanPath]
	if !exists {
		return nil, ErrFileRead(path, fs.ErrNotExist)
	}
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

func (mfs *MemoryFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()
	cleanPath := filepath.Clean(path)
	if _, exists := mfs.files[cleanPath]; exists {
		return ErrFileExists(path)
	}
	dir := filepath.Dir(cleanPath)
	if dir != "." && dir != "/" {
		mfs.dirs[dir] = true
	}
	fileCopy := make([]byte, len(data))
	copy(fileCopy, data)
	mfs.files[cleanPath] = fileCopy
	return nil
}

func (mfs *MemoryFileSystem) Stat(path string) (os.FileInfo, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()
	cleanPath := filepath.Clean(path)
	if data, exists := mfs.files[cleanPath]; exists {
		return &memoryFileInfo{name: filepath.Base(cleanPath), size: int64(len(data)), isDir: false}, nil
	}
	if _, exists := mfs.dirs[cleanPath]; exists {
		return &memoryFileInfo{name: filepath.Base(cleanPath), size: 0, isDir: true}, nil
	}
	return nil, fs.ErrNotExist
}

func (mfs *MemoryFileSystem) MkdirAll(path string, perm os.FileMode) error {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || cleanPath == "/" {
		return nil
	}
	parts := strings.Split(cleanPath, string(filepath.Separator))
	currentPath := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = filepath.Join(currentPath, part)
		}
		mfs.dirs[currentPath] = true
	}
	return nil
}

func (mfs *MemoryFileSystem) Exists(path string) bool {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()
	cleanPath := filepath.Clean(path)
	if _, ok := mfs.files[cleanPath]; ok {
		return true
	}
	if _, ok := mfs.dirs[cleanPath]; ok {
		return true
	}
	return false
}

func (mfs *MemoryFileSystem) IsDir(path string) bool {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()
	cleanPath := filepath.Clean(path)
	_, isDir := mfs.dirs[cleanPath]
	return isDir
}

func (mfs *MemoryFileSystem) Remove(path string) error {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()
	cleanPath := filepath.Clean(path)
	delete(mfs.files, cleanPath)
	delete(mfs.dirs, cleanPath)
	return nil
}
func (mfs *MemoryFileSystem) RemoveAll(path string) error { return mfs.Remove(path) }

type memoryFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (fi *memoryFileInfo) Name() string { return fi.name }
func (fi *memoryFileInfo) Size() int64  { return fi.size }
func (fi *memoryFileInfo) Mode() os.FileMode {
	if fi.isDir {
		return os.ModeDir | 0o755
	}
	return 0o644
}
func (fi *memoryFileInfo) ModTime() (t time.Time) { return }
func (fi *memoryFileInfo) IsDir() bool            { return fi.isDir }
func (fi *memoryFileInfo) Sys() any               { return nil }

// Convenience wrapper to satisfy any lingering interface uses
func WriteFile(ctx context.Context, fs FileSystem, path string, data []byte, perm os.FileMode) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before writing file")
	}
	dir := filepath.Dir(path)
	if err := fs.MkdirAll(dir, 0o755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create directory").WithDetails("directory", dir)
	}
	return fs.WriteFile(path, data, perm)
}
