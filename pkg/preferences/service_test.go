// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package preferences

import (
	"context"
	"testing"

	"github.com/lancekrogers/guild/pkg/storage"
)

func TestPreferenceService(t *testing.T) {
	ctx := context.Background()
	
	// Initialize test storage
	storageRegistry, _, err := storage.InitializeSQLiteStorageForTests(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize test storage: %v", err)
	}

	repo := storageRegistry.GetPreferencesRepository()
	service := NewService(repo)

	t.Run("SystemPreferences", func(t *testing.T) {
		// Test setting and getting system preference
		key := "test.system.pref"
		value := "test-value"

		err := service.SetSystemPreference(ctx, key, value)
		if err != nil {
			t.Errorf("Failed to set system preference: %v", err)
		}

		got, err := service.GetSystemPreference(ctx, key)
		if err != nil {
			t.Errorf("Failed to get system preference: %v", err)
		}

		if got != value {
			t.Errorf("Expected %v, got %v", value, got)
		}
	})

	t.Run("UserPreferences", func(t *testing.T) {
		userID := "user-123"
		key := "ui.theme"
		value := "light"

		err := service.SetUserPreference(ctx, userID, key, value)
		if err != nil {
			t.Errorf("Failed to set user preference: %v", err)
		}

		got, err := service.GetUserPreference(ctx, userID, key)
		if err != nil {
			t.Errorf("Failed to get user preference: %v", err)
		}

		if got != value {
			t.Errorf("Expected %v, got %v", value, got)
		}
	})

	t.Run("CampaignPreferences", func(t *testing.T) {
		campaignID := "campaign-456"
		key := "guild.maxAgents"
		value := 15

		err := service.SetCampaignPreference(ctx, campaignID, key, value)
		if err != nil {
			t.Errorf("Failed to set campaign preference: %v", err)
		}

		got, err := service.GetCampaignPreference(ctx, campaignID, key)
		if err != nil {
			t.Errorf("Failed to get campaign preference: %v", err)
		}

		// Handle float64 conversion from JSON
		if gotFloat, ok := got.(float64); ok {
			if int(gotFloat) != value {
				t.Errorf("Expected %v, got %v", value, int(gotFloat))
			}
		} else if gotInt, ok := got.(int); ok {
			if gotInt != value {
				t.Errorf("Expected %v, got %v", value, gotInt)
			}
		} else {
			t.Errorf("Unexpected type: %T", got)
		}
	})

	t.Run("PreferenceInheritance", func(t *testing.T) {
		// Set preferences at different levels
		key := "agent.timeout"
		systemValue := 3600
		userID := "user-inherit"
		userValue := 7200
		campaignID := "campaign-inherit"
		campaignValue := 1800
		agentID := "agent-inherit"

		// Set system default
		err := service.SetSystemPreference(ctx, key, systemValue)
		if err != nil {
			t.Errorf("Failed to set system preference: %v", err)
		}

		// Set user override
		err = service.SetUserPreference(ctx, userID, key, userValue)
		if err != nil {
			t.Errorf("Failed to set user preference: %v", err)
		}

		// Set campaign override
		err = service.SetCampaignPreference(ctx, campaignID, key, campaignValue)
		if err != nil {
			t.Errorf("Failed to set campaign preference: %v", err)
		}

		// Resolve preference - should get campaign value (most specific)
		resolved, err := service.ResolvePreference(ctx, key, &agentID, nil, &campaignID, &userID)
		if err != nil {
			t.Errorf("Failed to resolve preference: %v", err)
		}

		// Handle float64 conversion
		if resolvedFloat, ok := resolved.(float64); ok {
			if int(resolvedFloat) != campaignValue {
				t.Errorf("Expected campaign value %v, got %v", campaignValue, int(resolvedFloat))
			}
		} else {
			t.Errorf("Unexpected type: %T", resolved)
		}

		// Resolve without campaign - should get user value
		resolved, err = service.ResolvePreference(ctx, key, &agentID, nil, nil, &userID)
		if err != nil {
			t.Errorf("Failed to resolve preference: %v", err)
		}

		if resolvedFloat, ok := resolved.(float64); ok {
			if int(resolvedFloat) != userValue {
				t.Errorf("Expected user value %v, got %v", userValue, int(resolvedFloat))
			}
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		// Get a preference that doesn't exist - should return default
		value, err := service.GetSystemPreference(ctx, "ui.theme")
		if err != nil {
			t.Errorf("Failed to get default preference: %v", err)
		}

		expectedDefault := DefaultPreferences["ui.theme"]
		if value != expectedDefault {
			t.Errorf("Expected default %v, got %v", expectedDefault, value)
		}
	})

	t.Run("PreferenceValidation", func(t *testing.T) {
		tests := []struct {
			name      string
			key       string
			value     interface{}
			wantError bool
		}{
			{"ValidTheme", "ui.theme", "dark", false},
			{"InvalidTheme", "ui.theme", "invalid-theme", true},
			{"ValidFontSize", "ui.fontSize", 16, false},
			{"FontSizeTooSmall", "ui.fontSize", 4, true},
			{"FontSizeTooLarge", "ui.fontSize", 50, true},
			{"ValidTemperature", "provider.temperature", 0.7, false},
			{"TemperatureTooHigh", "provider.temperature", 3.0, true},
			{"ValidLogLevel", "dev.logLevel", "debug", false},
			{"InvalidLogLevel", "dev.logLevel", "trace", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := service.SetSystemPreference(ctx, tt.key, tt.value)
				if (err != nil) != tt.wantError {
					t.Errorf("SetSystemPreference() error = %v, wantError %v", err, tt.wantError)
				}
			})
		}
	})

	t.Run("BulkOperations", func(t *testing.T) {
		// Set multiple preferences
		prefs := map[string]interface{}{
			"bulk.test1": "value1",
			"bulk.test2": 42,
			"bulk.test3": true,
		}

		err := service.SetPreferences(ctx, "system", nil, prefs)
		if err != nil {
			t.Errorf("Failed to set bulk preferences: %v", err)
		}

		// Get multiple preferences
		keys := []string{"bulk.test1", "bulk.test2", "bulk.test3"}
		got, err := service.GetPreferences(ctx, "system", nil, keys)
		if err != nil {
			t.Errorf("Failed to get bulk preferences: %v", err)
		}

		if len(got) != len(keys) {
			t.Errorf("Expected %d preferences, got %d", len(keys), len(got))
		}

		// Verify values
		if got["bulk.test1"] != "value1" {
			t.Errorf("Expected 'value1', got %v", got["bulk.test1"])
		}
	})

	t.Run("ExportImport", func(t *testing.T) {
		campaignID := "export-campaign"

		// Set some preferences
		prefs := map[string]interface{}{
			"export.test1": "value1",
			"export.test2": 123,
			"export.test3": []string{"a", "b", "c"},
		}

		err := service.SetPreferences(ctx, "campaign", &campaignID, prefs)
		if err != nil {
			t.Errorf("Failed to set preferences for export: %v", err)
		}

		// Export preferences
		exported, err := service.ExportPreferences(ctx, "campaign", &campaignID)
		if err != nil {
			t.Errorf("Failed to export preferences: %v", err)
		}

		// Delete preferences
		err = service.DeletePreferencesByScope(ctx, "campaign", &campaignID)
		if err != nil {
			t.Errorf("Failed to delete preferences: %v", err)
		}

		// Import back
		err = service.ImportPreferences(ctx, "campaign", &campaignID, exported)
		if err != nil {
			t.Errorf("Failed to import preferences: %v", err)
		}

		// Verify imported
		value, err := service.GetCampaignPreference(ctx, campaignID, "export.test1")
		if err != nil {
			t.Errorf("Failed to get imported preference: %v", err)
		}

		if value != "value1" {
			t.Errorf("Expected 'value1', got %v", value)
		}
	})

	t.Run("CacheEffectiveness", func(t *testing.T) {
		key := "cache.test"
		value := "cached-value"

		// Set preference
		err := service.SetSystemPreference(ctx, key, value)
		if err != nil {
			t.Errorf("Failed to set preference: %v", err)
		}

		// First get - should hit database and cache
		got1, err := service.GetSystemPreference(ctx, key)
		if err != nil {
			t.Errorf("Failed to get preference: %v", err)
		}

		// Second get - should hit cache
		got2, err := service.GetSystemPreference(ctx, key)
		if err != nil {
			t.Errorf("Failed to get preference from cache: %v", err)
		}

		if got1 != got2 {
			t.Errorf("Cache returned different value: %v vs %v", got1, got2)
		}

		// Update preference - should invalidate cache
		newValue := "new-cached-value"
		err = service.SetSystemPreference(ctx, key, newValue)
		if err != nil {
			t.Errorf("Failed to update preference: %v", err)
		}

		// Get again - should get new value
		got3, err := service.GetSystemPreference(ctx, key)
		if err != nil {
			t.Errorf("Failed to get updated preference: %v", err)
		}

		if got3 != newValue {
			t.Errorf("Expected updated value %v, got %v", newValue, got3)
		}
	})
}

func TestPreferenceTypes(t *testing.T) {
	// Verify all default preferences have type definitions
	for key := range DefaultPreferences {
		if _, hasType := PreferenceTypes[key]; !hasType {
			t.Errorf("Default preference %s has no type definition", key)
		}
	}

	// Verify all validators reference valid preference keys
	for key := range PreferenceValidators {
		if _, hasType := PreferenceTypes[key]; !hasType {
			t.Errorf("Validator for %s has no corresponding type definition", key)
		}
	}
}
