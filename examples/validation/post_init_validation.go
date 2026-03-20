// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package main demonstrates post-init validation
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lancekrogers/guild-core/internal/setup"
)

func main() {
	// Get project path from command line or use current directory
	projectPath := "."
	if len(os.Args) > 1 {
		projectPath = os.Args[1]
	}

	fmt.Println("🔍 Running post-init validation...")
	fmt.Printf("Project path: %s\n", projectPath)
	fmt.Println()

	// Create validator
	validator := setup.NewInitValidator(projectPath)

	// Run validation with context
	ctx := context.Background()
	err := validator.Validate(ctx)

	// Print results
	validator.PrintResults()

	// Exit with appropriate code
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	fmt.Println("✅ All validations passed! Guild chat is ready to use.")
}
