// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"testing"
)

// TestDeprecatedMethods ensures backward compatibility for deprecated methods
func TestDeprecatedMethods(t *testing.T) {
	ctx := context.Background()

	t.Run("DetectProviders backward compatibility", func(t *testing.T) {
		detectors, err := NewDetectors(ctx, "/tmp/test")
		if err != nil {
			t.Fatalf("Failed to create detectors: %v", err)
		}

		// Test that both old and new methods work
		result1, err1 := detectors.DetectProviders(ctx)
		result2, err2 := detectors.Providers(ctx)

		if err1 != nil || err2 != nil {
			t.Errorf("Methods returned different errors: %v vs %v", err1, err2)
		}

		// Results should be the same (both methods call the same implementation)
		if (result1 == nil) != (result2 == nil) {
			t.Error("Methods returned different nil states")
		}
	})

	t.Run("GetProviderRecommendations backward compatibility", func(t *testing.T) {
		config, err := NewProviderConfig(ctx, "/tmp/test")
		if err != nil {
			t.Fatalf("Failed to create provider config: %v", err)
		}

		providers := []DetectedProvider{
			{Name: "openai", HasCredentials: true, IsLocal: false},
		}

		// Test that both old and new methods work
		rec1, err1 := config.GetProviderRecommendations(ctx, providers)
		rec2, err2 := config.ProviderRecommendations(ctx, providers)

		if err1 != nil || err2 != nil {
			t.Errorf("Methods returned different errors: %v vs %v", err1, err2)
		}

		// Results should be identical
		if rec1.Primary != rec2.Primary {
			t.Errorf("Methods returned different primary recommendations: %s vs %s", rec1.Primary, rec2.Primary)
		}
	})

	// Note: getOllamaModels method was removed as it's no longer needed
	// The Ollama integration now works through the standard provider interface
}
