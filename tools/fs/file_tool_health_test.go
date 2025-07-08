// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package fs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileTool_HealthCheck(t *testing.T) {
	t.Run("healthy with valid base path", func(t *testing.T) {
		// Create a temporary directory
		tmpDir, err := ioutil.TempDir("", "file_tool_health_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create file tool with temp directory
		tool := NewFileTool(tmpDir)

		// Health check should pass
		err = tool.HealthCheck()
		assert.NoError(t, err, "Health check should pass with valid directory")
	})

	t.Run("unhealthy with non-existent base path", func(t *testing.T) {
		// Create tool with non-existent path
		nonExistentPath := filepath.Join(os.TempDir(), "non_existent_dir_12345")
		tool := NewFileTool(nonExistentPath)

		// Remove the directory if it was created by NewFileTool
		os.RemoveAll(nonExistentPath)

		// Manually set the base path to ensure it doesn't exist
		tool.basePath = nonExistentPath

		// Health check should fail
		err := tool.HealthCheck()
		assert.Error(t, err, "Health check should fail with non-existent directory")
		assert.Contains(t, err.Error(), "base path not accessible")
	})

	t.Run("unhealthy with file instead of directory", func(t *testing.T) {
		// Create a temporary file
		tmpFile, err := ioutil.TempFile("", "file_tool_health_test")
		require.NoError(t, err)
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		// Create tool with file path instead of directory
		tool := NewFileTool(tmpFile.Name())

		// Manually set the base path to the file
		tool.basePath = tmpFile.Name()

		// Health check should fail
		err = tool.HealthCheck()
		assert.Error(t, err, "Health check should fail when base path is a file")
		assert.Contains(t, err.Error(), "base path is not a directory")
	})

	t.Run("unhealthy with no read permissions", func(t *testing.T) {
		// Skip this test on Windows as permission handling is different
		if os.Getenv("GOOS") == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		// Create a temporary directory
		tmpDir, err := ioutil.TempDir("", "file_tool_health_test")
		require.NoError(t, err)
		defer func() {
			// Restore permissions before cleanup
			os.Chmod(tmpDir, 0755)
			os.RemoveAll(tmpDir)
		}()

		// Create file tool
		tool := NewFileTool(tmpDir)

		// Remove read permissions
		err = os.Chmod(tmpDir, 0000)
		require.NoError(t, err)

		// Health check should fail
		err = tool.HealthCheck()
		assert.Error(t, err, "Health check should fail without read permissions")
	})
}
