// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.CorpusPath == "" {
		t.Error("CorpusPath should not be empty")
	}

	if cfg.ActivitiesPath == "" {
		t.Error("ActivitiesPath should not be empty")
	}

	if cfg.MaxSizeBytes != 10*1024*1024 {
		t.Errorf("Expected MaxSizeBytes to be %d, got %d", 10*1024*1024, cfg.MaxSizeBytes)
	}

	if len(cfg.DefaultTags) != 0 {
		t.Errorf("Expected 0 DefaultTags, got %d", len(cfg.DefaultTags))
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "corpus-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(tempDir); rmErr != nil {
			t.Logf("Failed to cleanup temp dir: %v", rmErr)
		}
	}()

	configPath := filepath.Join(tempDir, "corpus.yml")

	// Create a custom config
	cfg := Config{
		CorpusPath:     "/custom/corpus/path",
		ActivitiesPath: "/custom/activities/path",
		MaxSizeBytes:   1024 * 1024 * 100, // 100MB
		DefaultTags:    []string{"default", "tag"},
	}

	// Save the config
	err = SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load the config
	loadedCfg, err := LoadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify the loaded config
	if loadedCfg.CorpusPath != cfg.CorpusPath {
		t.Errorf("Expected CorpusPath %s, got %s", cfg.CorpusPath, loadedCfg.CorpusPath)
	}

	if loadedCfg.ActivitiesPath != cfg.ActivitiesPath {
		t.Errorf("Expected ActivitiesPath %s, got %s", cfg.ActivitiesPath, loadedCfg.ActivitiesPath)
	}

	if loadedCfg.MaxSizeBytes != cfg.MaxSizeBytes {
		t.Errorf("Expected MaxSizeBytes %d, got %d", cfg.MaxSizeBytes, loadedCfg.MaxSizeBytes)
	}

	if len(loadedCfg.DefaultTags) != len(cfg.DefaultTags) {
		t.Errorf("Expected %d DefaultTags, got %d", len(cfg.DefaultTags), len(loadedCfg.DefaultTags))
	}

	for i, tag := range cfg.DefaultTags {
		if loadedCfg.DefaultTags[i] != tag {
			t.Errorf("Expected DefaultTag %s, got %s", tag, loadedCfg.DefaultTags[i])
		}
	}
}

func TestLoadConfigFromNonExistentFile(t *testing.T) {
	// Create a path to a non-existent file
	nonExistentPath := filepath.Join(os.TempDir(), "non-existent-config.yml")

	// Make sure the file doesn't exist
	if rmErr := os.Remove(nonExistentPath); rmErr != nil && !os.IsNotExist(rmErr) {
		t.Logf("Failed to remove non-existent file: %v", rmErr)
	}

	// Load the config
	cfg, err := LoadConfigFromFile(nonExistentPath)
	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got %v", err)
	}

	// Verify it returned the default config
	defaultCfg := DefaultConfig()
	if cfg.MaxSizeBytes != defaultCfg.MaxSizeBytes {
		t.Errorf("Expected default MaxSizeBytes %d, got %d", defaultCfg.MaxSizeBytes, cfg.MaxSizeBytes)
	}
}
