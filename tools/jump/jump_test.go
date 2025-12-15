// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package jump

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJumpTrackAndFind(t *testing.T) {
	// Create a temporary database
	tmpDir, err := ioutil.TempDir("", "jump-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	j, err := New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()

	// Create test directories
	testDirs := []string{
		filepath.Join(tmpDir, "documents"),
		filepath.Join(tmpDir, "projects", "guild-framework"),
		filepath.Join(tmpDir, "projects", "other-project"),
		filepath.Join(tmpDir, "downloads"),
	}

	for _, dir := range testDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Track directories with different frequencies
	// Track guild-framework most frequently (but not most recently)
	for i := 0; i < 5; i++ {
		if err := j.Track(testDirs[1]); err != nil {
			t.Errorf("Failed to track %s: %v", testDirs[1], err)
		}
		time.Sleep(50 * time.Millisecond) // Ensure different timestamps
	}

	// Add a delay so guild-framework isn't the most recent
	time.Sleep(100 * time.Millisecond)

	// Track other directories
	for i, dir := range testDirs {
		if i == 1 {
			continue // Already tracked
		}
		for k := 0; k < i+1; k++ {
			if err := j.Track(dir); err != nil {
				t.Errorf("Failed to track %s: %v", dir, err)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Test finding directories
	tests := []struct {
		query    string
		expected string
	}{
		{"guild", testDirs[1]}, // Should find guild-framework
		{"doc", testDirs[0]},   // Should find documents
		{"down", testDirs[3]},  // Should find downloads
		{"frame", testDirs[1]}, // Should find guild-framework
	}

	for _, test := range tests {
		result, err := j.Find(test.query)
		if err != nil {
			t.Errorf("Failed to find %s: %v", test.query, err)
			continue
		}
		if result != test.expected {
			t.Errorf("Find(%s) = %s, want %s", test.query, result, test.expected)
		}
	}

	// Test no match
	_, err = j.Find("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent query")
	}
}

func TestJumpRecent(t *testing.T) {
	// Create a temporary database
	tmpDir, err := ioutil.TempDir("", "jump-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	j, err := New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()

	// Create and track directories in order
	testDirs := []string{
		filepath.Join(tmpDir, "first"),
		filepath.Join(tmpDir, "second"),
		filepath.Join(tmpDir, "third"),
		filepath.Join(tmpDir, "fourth"),
	}

	for i, dir := range testDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond) // Ensure different timestamps
		if err := j.Track(dir); err != nil {
			t.Errorf("Failed to track %s: %v", dir, err)
		}
		t.Logf("Tracked directory %d: %s", i, dir)
	}

	// Get recent directories
	recent, err := j.Recent(2)
	if err != nil {
		t.Fatal(err)
	}

	if len(recent) != 2 {
		t.Errorf("Recent(2) returned %d directories, want 2", len(recent))
	}

	// Should return in reverse order (most recent first)
	if len(recent) >= 2 {
		if recent[0] != testDirs[3] {
			t.Errorf("Most recent = %s, want %s", recent[0], testDirs[3])
		}
		if recent[1] != testDirs[2] {
			t.Errorf("Second most recent = %s, want %s", recent[1], testDirs[2])
		}
	}
}

func TestJumpCleanup(t *testing.T) {
	// Create a temporary database
	tmpDir, err := ioutil.TempDir("", "jump-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	j, err := New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()

	// Create and track a directory
	testDir := filepath.Join(tmpDir, "will-be-removed")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := j.Track(testDir); err != nil {
		t.Fatal(err)
	}

	// Verify it can be found
	result, err := j.Find("will-be-removed")
	if err != nil {
		t.Fatal(err)
	}
	if result != testDir {
		t.Errorf("Find returned %s, want %s", result, testDir)
	}

	// Remove the directory
	if err := os.RemoveAll(testDir); err != nil {
		t.Fatal(err)
	}

	// Wait a bit for cleanup to potentially happen
	time.Sleep(100 * time.Millisecond)

	// Should not find the directory anymore
	_, err = j.Find("will-be-removed")
	if err == nil {
		t.Error("Expected error when finding removed directory")
	}
}

func TestJumpEdgeCases(t *testing.T) {
	// Create a temporary database
	tmpDir, err := ioutil.TempDir("", "jump-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	j, err := New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()

	// Test empty query
	_, err = j.Find("")
	if err == nil {
		t.Error("Expected error for empty query")
	}

	// Test Recent with invalid n
	_, err = j.Recent(0)
	if err == nil {
		t.Error("Expected error for Recent(0)")
	}

	_, err = j.Recent(-1)
	if err == nil {
		t.Error("Expected error for Recent(-1)")
	}

	// Test tracking non-existent directory
	err = j.Track("/this/does/not/exist")
	if err == nil {
		t.Error("Expected error when tracking non-existent directory")
	}

	// Test tracking a file instead of directory
	testFile := filepath.Join(tmpDir, "notadir.txt")
	if err := ioutil.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	err = j.Track(testFile)
	if err == nil {
		t.Error("Expected error when tracking a file instead of directory")
	}
}
