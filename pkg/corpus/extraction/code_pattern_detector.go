// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package extraction

import (
	"context"
	"fmt"
	"strings"
)

// CodePatternDetector identifies patterns in code changes
type CodePatternDetector struct {
	refactoringPatterns map[string]RefactoringDetector
}

// RefactoringDetector interface for detecting specific refactoring patterns
type RefactoringDetector interface {
	Detect(ctx context.Context, analysis DiffAnalysis) *RefactoringPattern
}

// NewCodePatternDetector creates a new code pattern detector
func NewCodePatternDetector() *CodePatternDetector {
	return &CodePatternDetector{
		refactoringPatterns: map[string]RefactoringDetector{
			"extract_method":         &ExtractMethodDetector{},
			"introduce_interface":    &IntroduceInterfaceDetector{},
			"error_handling":         &ErrorHandlingDetector{},
			"dependency_injection":   &DependencyInjectionDetector{},
			"factory_pattern":        &FactoryPatternDetector{},
			"consolidate_duplicates": &ConsolidateDuplicatesDetector{},
		},
	}
}

// DetectRefactoringPattern detects refactoring patterns in a diff analysis
func (cpd *CodePatternDetector) DetectRefactoringPattern(ctx context.Context, analysis DiffAnalysis) *RefactoringPattern {
	if ctx.Err() != nil {
		return nil
	}

	// Try each pattern detector
	for patternType, detector := range cpd.refactoringPatterns {
		if pattern := detector.Detect(ctx, analysis); pattern != nil {
			pattern.Type = patternType
			return pattern
		}
	}

	return nil
}

// ExtractMethodDetector detects method extraction refactoring
type ExtractMethodDetector struct{}

func (emd *ExtractMethodDetector) Detect(ctx context.Context, analysis DiffAnalysis) *RefactoringPattern {
	// Look for new functions and corresponding code removal
	newFunctions := 0
	modifiedFunctions := 0

	for _, funcChange := range analysis.ModifiedFunctions {
		if funcChange.ChangeType == "added" {
			newFunctions++
		} else if funcChange.ChangeType == "modified" {
			modifiedFunctions++
		}
	}

	// Heuristic: new functions with modified existing functions suggests method extraction
	if newFunctions > 0 && modifiedFunctions > 0 && analysis.RemovedLines > 10 {
		confidence := 0.7
		if newFunctions > 1 {
			confidence += 0.1
		}
		if analysis.RemovedLines > 50 {
			confidence += 0.1
		}

		examples := []string{}
		for _, funcChange := range analysis.ModifiedFunctions {
			if funcChange.ChangeType == "added" {
				examples = append(examples, fmt.Sprintf("New function: %s", funcChange.Name))
			}
		}

		return &RefactoringPattern{
			Description: fmt.Sprintf("Extracted %d new functions, reducing complexity in existing functions", newFunctions),
			Confidence:  confidence,
			Examples:    examples,
			Metadata: map[string]interface{}{
				"new_functions":      newFunctions,
				"modified_functions": modifiedFunctions,
				"lines_removed":      analysis.RemovedLines,
			},
		}
	}

	return nil
}

// IntroduceInterfaceDetector detects interface introduction patterns
type IntroduceInterfaceDetector struct{}

func (iid *IntroduceInterfaceDetector) Detect(ctx context.Context, analysis DiffAnalysis) *RefactoringPattern {
	newInterfaces := 0
	modifiedTypes := 0

	for _, typeChange := range analysis.TypeChanges {
		if typeChange.ChangeType == "added" && strings.Contains(typeChange.NewDef, "interface") {
			newInterfaces++
		} else if typeChange.ChangeType == "modified" {
			modifiedTypes++
		}
	}

	// Look for new interface types and changes to existing code
	if newInterfaces > 0 && (modifiedTypes > 0 || len(analysis.ModifiedFunctions) > 2) {
		confidence := 0.8
		if newInterfaces > 1 {
			confidence += 0.05
		}

		examples := []string{}
		for _, typeChange := range analysis.TypeChanges {
			if typeChange.ChangeType == "added" && strings.Contains(typeChange.NewDef, "interface") {
				examples = append(examples, fmt.Sprintf("New interface: %s", typeChange.Name))
			}
		}

		return &RefactoringPattern{
			Description: fmt.Sprintf("Introduced %d new interfaces for better abstraction", newInterfaces),
			Confidence:  confidence,
			Examples:    examples,
			Metadata: map[string]interface{}{
				"new_interfaces":   newInterfaces,
				"modified_types":   modifiedTypes,
				"affected_files":   len(analysis.AffectedFiles),
			},
		}
	}

	return nil
}

// ErrorHandlingDetector detects improvements to error handling
type ErrorHandlingDetector struct{}

func (ehd *ErrorHandlingDetector) Detect(ctx context.Context, analysis DiffAnalysis) *RefactoringPattern {
	errorHandlingIndicators := 0
	
	// Look for error-related imports
	for _, imp := range analysis.AddedImports {
		if strings.Contains(imp, "error") || strings.Contains(imp, "gerror") {
			errorHandlingIndicators++
		}
	}

	// Look for error-related function changes
	for _, funcChange := range analysis.ModifiedFunctions {
		if strings.Contains(strings.ToLower(funcChange.Name), "error") ||
		   strings.Contains(funcChange.NewContent, "error") ||
		   strings.Contains(funcChange.NewContent, "gerror") {
			errorHandlingIndicators++
		}
	}

	if errorHandlingIndicators > 0 && len(analysis.ModifiedFunctions) > 1 {
		confidence := 0.85
		if errorHandlingIndicators > 2 {
			confidence += 0.05
		}

		return &RefactoringPattern{
			Description: "Improved error handling with proper error wrapping and context",
			Confidence:  confidence,
			Examples: []string{
				fmt.Sprintf("%d error-related changes detected", errorHandlingIndicators),
			},
			Metadata: map[string]interface{}{
				"error_indicators": errorHandlingIndicators,
				"functions_changed": len(analysis.ModifiedFunctions),
			},
		}
	}

	return nil
}

// DependencyInjectionDetector detects dependency injection patterns
type DependencyInjectionDetector struct{}

func (did *DependencyInjectionDetector) Detect(ctx context.Context, analysis DiffAnalysis) *RefactoringPattern {
	injectionIndicators := 0

	// Look for constructor-like patterns
	for _, funcChange := range analysis.ModifiedFunctions {
		if strings.HasPrefix(funcChange.Name, "New") && 
		   (funcChange.ChangeType == "added" || funcChange.ChangeType == "modified") {
			injectionIndicators++
		}
	}

	// Look for interface-related changes
	interfaceChanges := 0
	for _, typeChange := range analysis.TypeChanges {
		if strings.Contains(typeChange.NewDef, "interface") {
			interfaceChanges++
		}
	}

	if injectionIndicators > 0 && interfaceChanges > 0 && len(analysis.ModifiedFunctions) > 2 {
		confidence := 0.75
		if injectionIndicators > 2 {
			confidence += 0.1
		}

		return &RefactoringPattern{
			Description: "Introduced dependency injection for better testability and modularity",
			Confidence:  confidence,
			Examples: []string{
				fmt.Sprintf("%d constructor functions", injectionIndicators),
				fmt.Sprintf("%d interface definitions", interfaceChanges),
			},
			Metadata: map[string]interface{}{
				"constructors": injectionIndicators,
				"interfaces":   interfaceChanges,
			},
		}
	}

	return nil
}

// FactoryPatternDetector detects factory pattern introduction
type FactoryPatternDetector struct{}

func (fpd *FactoryPatternDetector) Detect(ctx context.Context, analysis DiffAnalysis) *RefactoringPattern {
	factoryIndicators := 0

	for _, funcChange := range analysis.ModifiedFunctions {
		funcName := strings.ToLower(funcChange.Name)
		if strings.Contains(funcName, "factory") || 
		   strings.Contains(funcName, "create") ||
		   strings.Contains(funcName, "build") {
			factoryIndicators++
		}
	}

	// Look for multiple new types (products of the factory)
	newTypes := 0
	for _, typeChange := range analysis.TypeChanges {
		if typeChange.ChangeType == "added" {
			newTypes++
		}
	}

	if factoryIndicators > 0 && newTypes > 1 {
		confidence := 0.8
		if factoryIndicators > 1 {
			confidence += 0.1
		}

		return &RefactoringPattern{
			Description: fmt.Sprintf("Introduced factory pattern with %d factory methods for %d types", factoryIndicators, newTypes),
			Confidence:  confidence,
			Examples: []string{
				fmt.Sprintf("%d factory methods", factoryIndicators),
				fmt.Sprintf("%d new types", newTypes),
			},
			Metadata: map[string]interface{}{
				"factory_methods": factoryIndicators,
				"new_types":       newTypes,
			},
		}
	}

	return nil
}

// ConsolidateDuplicatesDetector detects duplicate code elimination
type ConsolidateDuplicatesDetector struct{}

func (cdd *ConsolidateDuplicatesDetector) Detect(ctx context.Context, analysis DiffAnalysis) *RefactoringPattern {
	// High line removal with fewer new lines suggests consolidation
	if analysis.RemovedLines > analysis.AddedLines*2 && analysis.RemovedLines > 20 {
		newFunctions := 0
		for _, funcChange := range analysis.ModifiedFunctions {
			if funcChange.ChangeType == "added" {
				newFunctions++
			}
		}

		confidence := 0.7
		removalRatio := float64(analysis.RemovedLines) / float64(analysis.AddedLines+1)
		if removalRatio > 3.0 {
			confidence += 0.1
		}

		return &RefactoringPattern{
			Description: fmt.Sprintf("Consolidated duplicate code, removed %d lines and added %d lines", 
				analysis.RemovedLines, analysis.AddedLines),
			Confidence: confidence,
			Examples: []string{
				fmt.Sprintf("Removal ratio: %.1f:1", removalRatio),
				fmt.Sprintf("%d files affected", len(analysis.AffectedFiles)),
			},
			Metadata: map[string]interface{}{
				"lines_removed":  analysis.RemovedLines,
				"lines_added":    analysis.AddedLines,
				"removal_ratio":  removalRatio,
				"new_functions":  newFunctions,
			},
		}
	}

	return nil
}

// DetectArchitecturalPatterns detects higher-level architectural patterns
func (cpd *CodePatternDetector) DetectArchitecturalPatterns(ctx context.Context, analysis DiffAnalysis) []ArchitecturalPattern {
	if ctx.Err() != nil {
		return nil
	}

	var patterns []ArchitecturalPattern

	// Microservice pattern
	if cpd.isMicroservicePattern(analysis) {
		patterns = append(patterns, ArchitecturalPattern{
			Type:        "microservice_decomposition",
			Confidence:  0.8,
			Description: "Code reorganization suggesting microservice decomposition",
			Impact:      "high",
		})
	}

	// Layered architecture pattern
	if cpd.isLayeredArchitecturePattern(analysis) {
		patterns = append(patterns, ArchitecturalPattern{
			Type:        "layered_architecture",
			Confidence:  0.75,
			Description: "Introduction of layered architecture patterns",
			Impact:      "medium",
		})
	}

	// Event-driven pattern
	if cpd.isEventDrivenPattern(analysis) {
		patterns = append(patterns, ArchitecturalPattern{
			Type:        "event_driven",
			Confidence:  0.85,
			Description: "Introduction of event-driven architecture components",
			Impact:      "high",
		})
	}

	return patterns
}

func (cpd *CodePatternDetector) isMicroservicePattern(analysis DiffAnalysis) bool {
	// Look for service-related naming and multiple independent modules
	serviceIndicators := 0
	for _, file := range analysis.AffectedFiles {
		if strings.Contains(file, "service") || strings.Contains(file, "api") {
			serviceIndicators++
		}
	}
	return serviceIndicators > 2 && len(analysis.AffectedFiles) > 5
}

func (cpd *CodePatternDetector) isLayeredArchitecturePattern(analysis DiffAnalysis) bool {
	// Look for layer-related directory structure
	layers := map[string]bool{}
	layerNames := []string{"controller", "service", "repository", "model", "handler", "middleware"}
	
	for _, file := range analysis.AffectedFiles {
		for _, layer := range layerNames {
			if strings.Contains(file, layer) {
				layers[layer] = true
			}
		}
	}
	return len(layers) >= 3
}

func (cpd *CodePatternDetector) isEventDrivenPattern(analysis DiffAnalysis) bool {
	// Look for event-related components
	eventIndicators := 0
	eventTerms := []string{"event", "message", "queue", "publish", "subscribe", "listener"}
	
	for _, funcChange := range analysis.ModifiedFunctions {
		for _, term := range eventTerms {
			if strings.Contains(strings.ToLower(funcChange.Name), term) {
				eventIndicators++
				break
			}
		}
	}
	
	return eventIndicators > 2
}

// ArchitecturalPattern represents a detected architectural pattern
type ArchitecturalPattern struct {
	Type        string  `json:"type"`
	Confidence  float64 `json:"confidence"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"` // "low", "medium", "high"
}