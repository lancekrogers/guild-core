// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package providers includes example usage of the auto-detection system.
// This file demonstrates how to use the provider auto-detection capabilities.
package providers

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleAutoDetection demonstrates how to use the provider auto-detection system
func ExampleAutoDetection() {
	fmt.Println("🏰 Guild Framework Provider Auto-Detection Demo")
	fmt.Println("===============================================")

	// Create auto-detector with reasonable timeout
	detector := NewAutoDetector(5 * time.Second)

	ctx := context.Background()

	// Detect all available providers
	fmt.Println("\n🔍 Scanning for available providers...")
	results, err := detector.DetectAll(ctx)
	if err != nil {
		log.Printf("Detection failed: %v", err)
		return
	}

	// Display results
	fmt.Printf("\n📊 Found %d provider configurations:\n\n", len(results))
	for _, result := range results {
		displayProviderResult(result)
	}

	// Find the best provider with preferences
	fmt.Println("\n🎯 Finding best provider (preferring Claude Code, then Ollama)...")
	preferences := []ProviderType{ProviderClaudeCode, ProviderOllama}

	best, err := detector.GetBestProvider(ctx, preferences)
	if err != nil {
		fmt.Printf("❌ No suitable provider found: %v\n", err)
	} else {
		fmt.Printf("✅ Best provider: %s (confidence: %.1f%%)\n",
			GetProviderDisplayName(string(best.Provider)), best.Confidence*100)
		if best.Path != "" {
			fmt.Printf("   📍 Path: %s\n", best.Path)
		}
		if best.Endpoint != "" {
			fmt.Printf("   🌐 Endpoint: %s\n", best.Endpoint)
		}
		fmt.Printf("   🔧 Capabilities: %v\n", best.Capabilities)
	}

	// Validate specific providers
	fmt.Println("\n🔒 Validating provider availability...")

	for _, providerType := range []ProviderType{ProviderClaudeCode, ProviderOllama} {
		fmt.Printf("Checking %s... ", GetProviderDisplayName(string(providerType)))

		err := detector.ValidateProvider(ctx, providerType)
		if err != nil {
			fmt.Printf("❌ Not available (%v)\n", err)
		} else {
			fmt.Printf("✅ Available\n")
		}
	}

	fmt.Println("\n🏰 Detection complete!")
}

// displayProviderResult formats and displays a detection result
func displayProviderResult(result DetectionResult) {
	displayName := GetProviderDisplayName(string(result.Provider))

	if result.Available {
		fmt.Printf("✅ %s\n", displayName)
		fmt.Printf("   🔧 Version: %s\n", getVersionDisplay(result.Version))
		fmt.Printf("   📈 Confidence: %.1f%%\n", result.Confidence*100)

		if result.Path != "" {
			fmt.Printf("   📍 Binary: %s\n", result.Path)
		}
		if result.Endpoint != "" {
			fmt.Printf("   🌐 Service: %s\n", result.Endpoint)
		}

		if len(result.Capabilities) > 0 {
			fmt.Printf("   🎯 Features: %s\n", formatCapabilities(result.Capabilities))
		}
	} else {
		fmt.Printf("❌ %s - %s\n", displayName, result.Error)
	}
	fmt.Println()
}

// getVersionDisplay formats version for display
func getVersionDisplay(version string) string {
	if version == "" || version == "unknown" {
		return "Unknown"
	}
	return version
}

// formatCapabilities formats capability list for display
func formatCapabilities(capabilities []string) string {
	if len(capabilities) == 0 {
		return "None"
	}

	// Limit display to first few capabilities
	if len(capabilities) <= 3 {
		return fmt.Sprintf("%v", capabilities)
	}

	return fmt.Sprintf("%v... (%d total)", capabilities[:3], len(capabilities))
}

// ExampleQuickSetup demonstrates a quick setup pattern for auto-detection
func ExampleQuickSetup() {
	fmt.Println("🚀 Quick Setup Example")
	fmt.Println("======================")

	ctx := context.Background()
	detector := NewAutoDetector(3 * time.Second)

	// Try to get Claude Code first, fallback to Ollama
	preferences := []ProviderType{ProviderClaudeCode, ProviderOllama}

	provider, err := detector.GetBestProvider(ctx, preferences)
	if err != nil {
		fmt.Printf("❌ Setup failed: %v\n", err)
		fmt.Println("💡 Try installing Claude Code CLI or running Ollama service")
		return
	}

	fmt.Printf("✅ Using %s provider\n", GetProviderDisplayName(string(provider.Provider)))

	// Note: In real usage, you would use the factory to create a client:
	// factory := NewFactory()
	// client, err := factory.CreateClient(provider.Provider, apiKey, model)

	fmt.Println("🎉 Setup complete!")
}

// ExampleAdvancedDetection shows advanced detection features
func ExampleAdvancedDetection() {
	fmt.Println("🔬 Advanced Detection Example")
	fmt.Println("=============================")

	ctx := context.Background()
	detector := NewAutoDetector(2 * time.Second)

	// Detect Claude Code specifically
	fmt.Println("🔍 Checking Claude Code CLI installation...")
	claudeResult, err := detector.DetectClaudeCode(ctx)
	if err != nil {
		fmt.Printf("❌ Claude Code detection failed: %v\n", err)
	} else {
		fmt.Printf("Claude Code Status: ")
		if claudeResult.Available {
			fmt.Printf("✅ Found at %s (v%s)\n", claudeResult.Path, claudeResult.Version)
		} else {
			fmt.Printf("❌ Not found - %s\n", claudeResult.Error)
		}
	}

	// Detect Ollama specifically
	fmt.Println("\n🔍 Checking Ollama service...")
	ollamaResult, err := detector.DetectOllama(ctx)
	if err != nil {
		fmt.Printf("❌ Ollama detection failed: %v\n", err)
	} else {
		fmt.Printf("Ollama Status: ")
		if ollamaResult.Available {
			fmt.Printf("✅ Running at %s (v%s)\n", ollamaResult.Endpoint, ollamaResult.Version)
		} else {
			fmt.Printf("❌ Not running - %s\n", ollamaResult.Error)
		}
	}

	fmt.Println("\n🔬 Advanced detection complete!")
}

// ExampleIntegrationWithGuild shows how to integrate with Guild's provider system
func ExampleIntegrationWithGuild() {
	fmt.Println("🏰 Guild Integration Example")
	fmt.Println("===========================")

	ctx := context.Background()
	detector := NewAutoDetector(5 * time.Second)

	// This is how you might integrate auto-detection with Guild's existing systems
	fmt.Println("1. 🔍 Auto-detecting available providers...")

	results, err := detector.DetectAll(ctx)
	if err != nil {
		fmt.Printf("❌ Detection failed: %v\n", err)
		return
	}

	// Filter to available providers only
	var available []DetectionResult
	for _, result := range results {
		if result.Available {
			available = append(available, result)
		}
	}

	if len(available) == 0 {
		fmt.Println("❌ No providers available")
		fmt.Println("💡 Install Claude Code CLI or start Ollama service")
		return
	}

	fmt.Printf("2. ✅ Found %d available provider(s)\n", len(available))

	// In a real Guild integration, you would:
	// - Register detected providers with the registry
	// - Set up fallback chains based on detection results
	// - Update configuration with auto-detected settings

	fmt.Println("3. 🔧 Integration points:")
	fmt.Println("   - Register with provider registry")
	fmt.Println("   - Configure fallback chains")
	fmt.Println("   - Update runtime configuration")
	fmt.Println("   - Enable hot-swapping based on availability")

	fmt.Println("\n🏰 Integration planning complete!")
}
