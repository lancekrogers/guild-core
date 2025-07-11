// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package storage

import (
	"context"
	"testing"
)

func TestPreferencesRepository(t *testing.T) {
	ctx := context.Background()

	// Initialize test storage
	storageRegistry, _, err := InitializeSQLiteStorageForTests(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize test storage: %v", err)
	}

	repo := storageRegistry.GetPreferencesRepository()
	if repo == nil {
		t.Fatal("Preferences repository is nil")
	}

	// Test basic CRUD operations
	pref := &Preference{
		Scope:   "system",
		ScopeID: nil,
		Key:     "test.key",
		Value:   "test-value",
		Version: 1,
	}

	// Create preference
	err = repo.CreatePreference(ctx, pref)
	if err != nil {
		t.Errorf("Failed to create preference: %v", err)
	}

	// Get preference by key
	retrieved, err := repo.GetPreferenceByKey(ctx, "system", nil, "test.key")
	if err != nil {
		t.Errorf("Failed to get preference by key: %v", err)
	}

	if retrieved.Value != "test-value" {
		t.Errorf("Expected value 'test-value', got %v", retrieved.Value)
	}

	// Update preference
	retrieved.Value = "updated-value"
	err = repo.UpdatePreference(ctx, retrieved)
	if err != nil {
		t.Errorf("Failed to update preference: %v", err)
	}

	// Verify update
	updated, err := repo.GetPreference(ctx, retrieved.ID)
	if err != nil {
		t.Errorf("Failed to get updated preference: %v", err)
	}

	if updated.Value != "updated-value" {
		t.Errorf("Expected value 'updated-value', got %v", updated.Value)
	}

	// List preferences
	prefs, err := repo.ListPreferencesByScope(ctx, "system", nil)
	if err != nil {
		t.Errorf("Failed to list preferences: %v", err)
	}

	if len(prefs) == 0 {
		t.Error("Expected at least one preference in list")
	}

	// Delete preference
	err = repo.DeletePreference(ctx, retrieved.ID)
	if err != nil {
		t.Errorf("Failed to delete preference: %v", err)
	}

	// Verify deletion
	_, err = repo.GetPreference(ctx, retrieved.ID)
	if err == nil {
		t.Error("Expected error when getting deleted preference")
	}
}