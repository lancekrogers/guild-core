// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// GuildArchiveWriter implements the ArchiveWriter interface with Guild safety protocols
type GuildArchiveWriter struct {
	archiveDir  string
	backupDir   string
	dryRun      bool
	permissions os.FileMode
}

// NewGuildArchiveWriter creates a new Guild Archive writer
func NewGuildArchiveWriter(archiveDir string, options ...ArchiveWriterOption) *GuildArchiveWriter {
	writer := &GuildArchiveWriter{
		archiveDir:  archiveDir,
		backupDir:   filepath.Join(archiveDir, ".guild-backups"),
		dryRun:      false,
		permissions: 0o644,
	}

	for _, option := range options {
		option(writer)
	}

	return writer
}

// ArchiveWriterOption configures the Guild Archive writer
type ArchiveWriterOption func(*GuildArchiveWriter)

// WithDryRun configures Guild dry run mode
func WithDryRun(dryRun bool) ArchiveWriterOption {
	return func(w *GuildArchiveWriter) {
		w.dryRun = dryRun
	}
}

// WithBackupArchive configures the Guild backup archive directory
func WithBackupArchive(backupDir string) ArchiveWriterOption {
	return func(w *GuildArchiveWriter) {
		w.backupDir = backupDir
	}
}

// WithGuildPermissions configures Guild Archive file permissions
func WithGuildPermissions(perm os.FileMode) ArchiveWriterOption {
	return func(w *GuildArchiveWriter) {
		w.permissions = perm
	}
}

// WriteStructure implements the ArchiveWriter interface for Guild Archives
func (w *GuildArchiveWriter) WriteStructure(ctx context.Context, refined *RefinedCommission) error {
	if refined == nil || refined.Structure == nil {
		return gerror.New(gerror.ErrCodeValidation, "refined commission or structure cannot be nil", nil).
			WithComponent("manager").
			WithOperation("WriteStructure")
	}

	// Create target Archive directory for this commission
	targetDir := filepath.Join(w.archiveDir, refined.CommissionID)

	// Validate target Archive directory
	if err := w.validateArchiveDir(targetDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid Archive directory").
			WithComponent("manager").
			WithOperation("WriteStructure").
			WithDetails("target_dir", targetDir)
	}

	// Create Guild backup if files exist
	if err := w.createGuildBackup(ctx, targetDir, refined.CommissionID); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create Guild backup").
			WithComponent("manager").
			WithOperation("WriteStructure").
			WithDetails("commission_id", refined.CommissionID)
	}

	// Write Archive files atomically
	if err := w.writeArchiveFiles(ctx, targetDir, refined.Structure); err != nil {
		// Attempt to restore Guild backup on failure
		if restoreErr := w.restoreGuildBackup(ctx, targetDir, refined.CommissionID); restoreErr != nil {
			return gerror.Wrapf(err, gerror.ErrCodeInternal, "archive write failed and Guild backup restore failed: restore error: %v", restoreErr).
				WithComponent("manager").
				WithOperation("WriteStructure").
				WithDetails("commission_id", refined.CommissionID).
				WithDetails("restore_error", restoreErr.Error())
		}
		return gerror.Wrap(err, gerror.ErrCodeInternal, "archive write failed, Guild backup restored").
			WithComponent("manager").
			WithOperation("WriteStructure").
			WithDetails("commission_id", refined.CommissionID)
	}

	// Write Guild manifest file
	if err := w.writeGuildManifest(ctx, targetDir, refined); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write Guild manifest").
			WithComponent("manager").
			WithOperation("WriteStructure").
			WithDetails("commission_id", refined.CommissionID)
	}

	return nil
}

// validateArchiveDir validates the target Archive directory path for Guild security
func (w *GuildArchiveWriter) validateArchiveDir(targetDir string) error {
	// Check if path is within Guild Archive directory
	absBase, err := filepath.Abs(w.archiveDir)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get absolute base path").
			WithComponent("manager").
			WithOperation("validateArchiveDir")
	}

	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get absolute target path").
			WithComponent("manager").
			WithOperation("validateArchiveDir").
			WithDetails("target_dir", targetDir)
	}

	if !strings.HasPrefix(absTarget, absBase) {
		return gerror.Newf(gerror.ErrCodeValidation, "target directory %s is outside Guild Archive directory %s", absTarget, absBase).
			WithComponent("manager").
			WithOperation("validateArchiveDir").
			WithDetails("target_dir", absTarget).
			WithDetails("base_dir", absBase)
	}

	return nil
}

// createGuildBackup creates a Guild backup of existing files
func (w *GuildArchiveWriter) createGuildBackup(ctx context.Context, targetDir, commissionID string) error {
	// Check if target directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return nil // No backup needed
	}

	if w.dryRun {
		fmt.Printf("DRY RUN: Would create Guild backup of %s\n", targetDir)
		return nil
	}

	// Create Guild backup directory
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(w.backupDir, fmt.Sprintf("%s-%s", commissionID, timestamp))

	if err := os.MkdirAll(w.backupDir, 0o755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create backup directory").
			WithComponent("manager").
			WithOperation("createGuildBackup").
			WithDetails("backup_dir", w.backupDir)
	}

	// Copy existing files to backup
	if err := copyDir(targetDir, backupPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to copy files to backup").
			WithComponent("manager").
			WithOperation("createGuildBackup").
			WithDetails("source", targetDir).
			WithDetails("backup_path", backupPath)
	}

	return nil
}

// writeArchiveFiles writes all files in the structure to Guild Archives
func (w *GuildArchiveWriter) writeArchiveFiles(ctx context.Context, targetDir string, structure *FileStructure) error {
	// Create target directory
	if !w.dryRun {
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create target directory").
				WithComponent("manager").
				WithOperation("writeArchiveFiles").
				WithDetails("target_dir", targetDir)
		}
	}

	// Write each file to Archives
	for _, file := range structure.Files {
		if err := w.writeArchiveFile(ctx, targetDir, file); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to write Archive file %s", file.Path).
				WithComponent("manager").
				WithOperation("writeArchiveFiles").
				WithDetails("file_path", file.Path)
		}
	}

	return nil
}

// writeArchiveFile writes a single file to Guild Archives
func (w *GuildArchiveWriter) writeArchiveFile(ctx context.Context, targetDir string, file *FileEntry) error {
	filePath := filepath.Join(targetDir, file.Path)

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if !w.dryRun {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to create directory %s", dir).
				WithComponent("manager").
				WithOperation("writeArchiveFile").
				WithDetails("directory", dir)
		}
	}

	if w.dryRun {
		fmt.Printf("DRY RUN: Would write file %s (%d characters, %d tasks)\n",
			filePath, len(file.Content), file.TasksCount)
		return nil
	}

	// Write file atomically (write to temp file, then rename)
	tempFile := filePath + ".tmp"

	if err := os.WriteFile(tempFile, []byte(file.Content), w.permissions); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write temporary file").
			WithComponent("manager").
			WithOperation("writeArchiveFile").
			WithDetails("temp_file", tempFile)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		// Clean up temp file on error
		os.Remove(tempFile)
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to rename temporary file").
			WithComponent("manager").
			WithOperation("writeArchiveFile").
			WithDetails("temp_file", tempFile).
			WithDetails("target_file", filePath)
	}

	return nil
}

// writeGuildManifest writes a Guild manifest file with metadata
func (w *GuildArchiveWriter) writeGuildManifest(ctx context.Context, targetDir string, refined *RefinedCommission) error {
	manifest := &GuildManifest{
		CommissionID: refined.CommissionID,
		CreatedAt:    time.Now(),
		FileCount:    len(refined.Structure.Files),
		TotalTasks:   w.countTotalTasks(refined.Structure),
		Files:        make([]FileManifestEntry, 0, len(refined.Structure.Files)),
		Metadata:     refined.Metadata,
	}

	// Add file entries to manifest
	for _, file := range refined.Structure.Files {
		manifest.Files = append(manifest.Files, FileManifestEntry{
			Path:       file.Path,
			Type:       string(file.Type),
			TasksCount: file.TasksCount,
			Size:       len(file.Content),
		})
	}

	manifestPath := filepath.Join(targetDir, ".guild-manifest.json")

	if w.dryRun {
		fmt.Printf("DRY RUN: Would write Guild manifest %s\n", manifestPath)
		return nil
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal Guild manifest").
			WithComponent("manager").
			WithOperation("writeGuildManifest").
			WithDetails("commission_id", refined.CommissionID)
	}

	// Write Guild manifest file
	if err := os.WriteFile(manifestPath, data, w.permissions); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write Guild manifest file").
			WithComponent("manager").
			WithOperation("writeGuildManifest").
			WithDetails("manifest_path", manifestPath)
	}

	return nil
}

// restoreGuildBackup restores from the most recent Guild backup
func (w *GuildArchiveWriter) restoreGuildBackup(ctx context.Context, targetDir, commissionID string) error {
	if w.dryRun {
		fmt.Printf("DRY RUN: Would restore Guild backup for %s\n", commissionID)
		return nil
	}

	// Find most recent Guild backup
	backupPath, err := w.findLatestGuildBackup(commissionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to find Guild backup").
			WithComponent("manager").
			WithOperation("restoreGuildBackup").
			WithDetails("commission_id", commissionID)
	}

	if backupPath == "" {
		return gerror.Newf(gerror.ErrCodeNotFound, "no Guild backup found for commission %s", commissionID).
			WithComponent("manager").
			WithOperation("restoreGuildBackup").
			WithDetails("commission_id", commissionID)
	}

	// Remove current directory
	if err := os.RemoveAll(targetDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to remove current directory").
			WithComponent("manager").
			WithOperation("restoreGuildBackup").
			WithDetails("target_dir", targetDir)
	}

	// Restore from Guild backup
	if err := copyDir(backupPath, targetDir); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to restore from Guild backup").
			WithComponent("manager").
			WithOperation("restoreGuildBackup").
			WithDetails("backup_path", backupPath).
			WithDetails("target_dir", targetDir)
	}

	return nil
}

// findLatestGuildBackup finds the most recent Guild backup for a commission
func (w *GuildArchiveWriter) findLatestGuildBackup(commissionID string) (string, error) {
	entries, err := os.ReadDir(w.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No backups exist
		}
		return "", err
	}

	var latestBackup string
	var latestTime time.Time

	prefix := commissionID + "-"
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		// Extract timestamp
		timestamp := strings.TrimPrefix(name, prefix)
		t, err := time.Parse("20060102-150405", timestamp)
		if err != nil {
			continue // Skip invalid timestamps
		}

		if t.After(latestTime) {
			latestTime = t
			latestBackup = filepath.Join(w.backupDir, name)
		}
	}

	return latestBackup, nil
}

// countTotalTasks counts total Workshop Board tasks across all Archive files
func (w *GuildArchiveWriter) countTotalTasks(structure *FileStructure) int {
	total := 0
	for _, file := range structure.Files {
		total += file.TasksCount
	}
	return total
}

// GuildManifest represents the metadata file for a refined commission in Guild Archives
type GuildManifest struct {
	CommissionID string                 `json:"commission_id"`
	CreatedAt    time.Time              `json:"created_at"`
	FileCount    int                    `json:"file_count"`
	TotalTasks   int                    `json:"total_tasks"`
	Files        []FileManifestEntry    `json:"files"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// FileManifestEntry represents a file in the manifest
type FileManifestEntry struct {
	Path       string `json:"path"`
	Type       string `json:"type"`
	TasksCount int    `json:"tasks_count"`
	Size       int    `json:"size"`
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}
