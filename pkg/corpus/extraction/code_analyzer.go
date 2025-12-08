// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package extraction

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// CodeAnalyzer extracts knowledge from code changes and git commits
type CodeAnalyzer struct {
	gitClient       *GitClient
	codeParser      *CodeParser
	diffAnalyzer    *DiffAnalyzer
	patternDetector *CodePatternDetector
}

// NewCodeAnalyzer creates a new code analyzer with default components
func NewCodeAnalyzer(ctx context.Context, gitRepoPath string) (*CodeAnalyzer, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("NewCodeAnalyzer")
	}

	gitClient, err := NewGitClient(ctx, gitRepoPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create git client").
			WithComponent("corpus.extraction").
			WithOperation("NewCodeAnalyzer").
			WithDetails("repo_path", gitRepoPath)
	}

	codeParser := NewCodeParser()
	diffAnalyzer := NewDiffAnalyzer()
	patternDetector := NewCodePatternDetector()

	return &CodeAnalyzer{
		gitClient:       gitClient,
		codeParser:      codeParser,
		diffAnalyzer:    diffAnalyzer,
		patternDetector: patternDetector,
	}, nil
}

// AnalyzeCommits extracts knowledge from a series of git commits
func (ca *CodeAnalyzer) AnalyzeCommits(ctx context.Context, commits []Commit) ([]ExtractedKnowledge, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("AnalyzeCommits")
	}

	var knowledge []ExtractedKnowledge

	for _, commit := range commits {
		// Check for context cancellation periodically
		if ctx.Err() != nil {
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled during commit analysis").
				WithComponent("corpus.extraction").
				WithOperation("AnalyzeCommits").
				WithDetails("commit_sha", commit.SHA)
		}

		commitKnowledge, err := ca.analyzeCommit(ctx, commit)
		if err != nil {
			// Log error but continue with other commits
			continue
		}

		knowledge = append(knowledge, commitKnowledge...)
	}

	return knowledge, nil
}

// analyzeCommit extracts knowledge from a single commit
func (ca *CodeAnalyzer) analyzeCommit(ctx context.Context, commit Commit) ([]ExtractedKnowledge, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var knowledge []ExtractedKnowledge

	// Get the diff for this commit
	diff, err := ca.gitClient.GetDiff(ctx, commit.SHA)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get diff").
			WithComponent("corpus.extraction").
			WithOperation("analyzeCommit").
			WithDetails("commit_sha", commit.SHA)
	}

	// Analyze the changes
	analysis, err := ca.diffAnalyzer.Analyze(ctx, diff)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to analyze diff").
			WithComponent("corpus.extraction").
			WithOperation("analyzeCommit").
			WithDetails("commit_sha", commit.SHA)
	}

	// Detect refactoring patterns
	if pattern := ca.patternDetector.DetectRefactoringPattern(ctx, analysis); pattern != nil {
		k := ExtractedKnowledge{
			ID:         ca.generateKnowledgeID("pattern"),
			Type:       KnowledgePattern,
			Content:    pattern.Description,
			Confidence: pattern.Confidence,
			Source:     ca.buildCodeSource(commit, nil),
			Context:    ca.buildPatternContext(pattern),
			Metadata:   pattern.Metadata,
			Timestamp:  time.Now(),
		}
		knowledge = append(knowledge, k)
	}

	// Detect API changes
	apiChanges := ca.detectAPIChanges(ctx, analysis)
	for _, change := range apiChanges {
		k := ExtractedKnowledge{
			ID:         ca.generateKnowledgeID("constraint"),
			Type:       KnowledgeConstraint,
			Content:    change.ToKnowledgeString(),
			Confidence: 0.85, // High confidence for API changes
			Source:     ca.buildCodeSource(commit, &change),
			Context:    ca.buildAPIChangeContext(change),
			Metadata: map[string]interface{}{
				"api_change_type": change.Type,
				"function":        change.Function,
			},
			Timestamp: time.Now(),
		}
		knowledge = append(knowledge, k)
	}

	// Detect bug fixes
	if fix := ca.detectBugFix(ctx, commit, analysis); fix != nil {
		k := ExtractedKnowledge{
			ID:         ca.generateKnowledgeID("solution"),
			Type:       KnowledgeSolution,
			Content:    fix.ToKnowledgeString(),
			Confidence: 0.9, // High confidence for actual fixes
			Source:     ca.buildCodeSource(commit, fix),
			Context:    ca.buildBugFixContext(fix),
			Metadata: map[string]interface{}{
				"bug_severity":   fix.Severity,
				"affected_files": fix.Files,
			},
			Timestamp: time.Now(),
		}
		knowledge = append(knowledge, k)
	}

	// Detect architectural decisions
	if decision := ca.detectArchitecturalDecision(ctx, commit, analysis); decision != nil {
		k := ExtractedKnowledge{
			ID:         ca.generateKnowledgeID("decision"),
			Type:       KnowledgeDecision,
			Content:    decision.Content,
			Confidence: decision.Confidence,
			Source:     ca.buildCodeSource(commit, decision),
			Context:    decision.Context,
			Metadata:   decision.Metadata,
			Timestamp:  time.Now(),
		}
		knowledge = append(knowledge, k)
	}

	return knowledge, nil
}

// detectAPIChanges identifies API changes in the diff analysis
func (ca *CodeAnalyzer) detectAPIChanges(ctx context.Context, analysis DiffAnalysis) []APIChange {
	var changes []APIChange

	// Check modified functions for API changes
	for _, funcChange := range analysis.ModifiedFunctions {
		change := ca.analyzeFunctionChange(funcChange)
		if change != nil {
			changes = append(changes, *change)
		}
	}

	// Check type changes for API impacts
	for _, typeChange := range analysis.TypeChanges {
		change := ca.analyzeTypeChange(typeChange)
		if change != nil {
			changes = append(changes, *change)
		}
	}

	return changes
}

// detectBugFix determines if a commit represents a bug fix
func (ca *CodeAnalyzer) detectBugFix(ctx context.Context, commit Commit, analysis DiffAnalysis) *BugFix {
	// Look for bug fix indicators in commit message
	bugIndicators := []string{"fix", "bug", "error", "issue", "crash", "exception", "null pointer", "memory leak", "race condition"}

	commitMsg := strings.ToLower(commit.Message)

	var foundIndicators []string
	for _, indicator := range bugIndicators {
		if strings.Contains(commitMsg, indicator) {
			foundIndicators = append(foundIndicators, indicator)
		}
	}

	if len(foundIndicators) == 0 {
		return nil
	}

	// Extract problem description from commit message
	problem := ca.extractProblemFromCommit(commit.Message)

	// Extract solution from the changes
	solution := ca.extractSolutionFromAnalysis(analysis)

	// Determine severity
	severity := ca.determineBugSeverity(commit.Message, foundIndicators)

	return &BugFix{
		Problem:     problem,
		Solution:    solution,
		Files:       analysis.AffectedFiles,
		Severity:    severity,
		Description: fmt.Sprintf("Bug fix: %s", strings.Join(foundIndicators, ", ")),
	}
}

// detectArchitecturalDecision identifies architectural decisions from commits
func (ca *CodeAnalyzer) detectArchitecturalDecision(ctx context.Context, commit Commit, analysis DiffAnalysis) *ArchitecturalDecision {
	architecturalIndicators := []string{
		"refactor", "restructure", "redesign", "architecture", "pattern",
		"introduce", "extract", "separate", "decouple", "modularize",
	}

	commitMsg := strings.ToLower(commit.Message)

	var foundIndicators []string
	for _, indicator := range architecturalIndicators {
		if strings.Contains(commitMsg, indicator) {
			foundIndicators = append(foundIndicators, indicator)
		}
	}

	// Look for significant structural changes
	hasStructuralChanges := len(analysis.ModifiedFunctions) > 5 ||
		len(analysis.TypeChanges) > 2 ||
		len(analysis.AffectedFiles) > 3

	if len(foundIndicators) == 0 && !hasStructuralChanges {
		return nil
	}

	decision := &ArchitecturalDecision{
		Content:    ca.extractArchitecturalDecisionContent(commit.Message, analysis),
		Confidence: ca.calculateArchitecturalDecisionConfidence(foundIndicators, analysis),
		Context: map[string]interface{}{
			"affected_files":     len(analysis.AffectedFiles),
			"modified_functions": len(analysis.ModifiedFunctions),
			"type_changes":       len(analysis.TypeChanges),
			"indicators":         foundIndicators,
		},
		Metadata: map[string]interface{}{
			"commit_message": commit.Message,
			"files_changed":  analysis.AffectedFiles,
		},
	}

	return decision
}

// Helper methods for analysis

func (ca *CodeAnalyzer) analyzeFunctionChange(funcChange FunctionChange) *APIChange {
	// Simple heuristic to detect API changes
	oldLines := strings.Split(funcChange.OldContent, "\n")
	newLines := strings.Split(funcChange.NewContent, "\n")

	// Look for signature changes
	oldSig := ca.extractFunctionSignature(oldLines)
	newSig := ca.extractFunctionSignature(newLines)

	if oldSig != newSig {
		changeType := "additive"
		if ca.isBreakingChange(oldSig, newSig) {
			changeType = "breaking"
		}

		return &APIChange{
			Type:        changeType,
			Function:    funcChange.Name,
			OldSig:      oldSig,
			NewSig:      newSig,
			Description: fmt.Sprintf("Function signature changed from '%s' to '%s'", oldSig, newSig),
		}
	}

	return nil
}

func (ca *CodeAnalyzer) analyzeTypeChange(typeChange TypeChange) *APIChange {
	if typeChange.ChangeType == "modified" {
		return &APIChange{
			Type:        "breaking",
			Function:    typeChange.Name,
			OldSig:      typeChange.OldDef,
			NewSig:      typeChange.NewDef,
			Description: fmt.Sprintf("Type definition changed: %s", typeChange.Name),
		}
	}
	return nil
}

func (ca *CodeAnalyzer) extractProblemFromCommit(message string) string {
	lines := strings.Split(message, "\n")
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		if len(firstLine) > 10 {
			return firstLine
		}
	}
	return "Problem not clearly described in commit message"
}

func (ca *CodeAnalyzer) extractSolutionFromAnalysis(analysis DiffAnalysis) string {
	var solutions []string

	if len(analysis.ModifiedFunctions) > 0 {
		solutions = append(solutions, fmt.Sprintf("Modified %d functions", len(analysis.ModifiedFunctions)))
	}

	if len(analysis.AddedImports) > 0 {
		solutions = append(solutions, fmt.Sprintf("Added imports: %s", strings.Join(analysis.AddedImports, ", ")))
	}

	if len(solutions) == 0 {
		return fmt.Sprintf("Code changes across %d files", len(analysis.AffectedFiles))
	}

	return strings.Join(solutions, "; ")
}

func (ca *CodeAnalyzer) determineBugSeverity(message string, indicators []string) string {
	severityMap := map[string]string{
		"crash":          "critical",
		"exception":      "critical",
		"null pointer":   "high",
		"memory leak":    "high",
		"race condition": "high",
		"security":       "high",
		"performance":    "medium",
		"bug":            "medium",
		"fix":            "medium",
	}

	highestSeverity := "low"
	severityOrder := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	msgLower := strings.ToLower(message)
	for term, severity := range severityMap {
		if strings.Contains(msgLower, term) {
			if severityOrder[severity] > severityOrder[highestSeverity] {
				highestSeverity = severity
			}
		}
	}

	return highestSeverity
}

func (ca *CodeAnalyzer) extractArchitecturalDecisionContent(message string, analysis DiffAnalysis) string {
	lines := strings.Split(message, "\n")
	content := ""

	if len(lines) > 0 {
		content = strings.TrimSpace(lines[0])
	}

	// Add analysis summary
	content += fmt.Sprintf("\n\nImpact: %d files, %d functions, %d type changes",
		len(analysis.AffectedFiles),
		len(analysis.ModifiedFunctions),
		len(analysis.TypeChanges))

	return content
}

func (ca *CodeAnalyzer) calculateArchitecturalDecisionConfidence(indicators []string, analysis DiffAnalysis) float64 {
	confidence := 0.5

	// Increase confidence based on indicators
	confidence += float64(len(indicators)) * 0.1

	// Increase confidence based on scope of changes
	if len(analysis.AffectedFiles) > 5 {
		confidence += 0.2
	}
	if len(analysis.ModifiedFunctions) > 10 {
		confidence += 0.2
	}

	// Cap confidence
	if confidence > 0.9 {
		confidence = 0.9
	}

	return confidence
}

func (ca *CodeAnalyzer) extractFunctionSignature(lines []string) string {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "func ") {
			return line
		}
	}
	return ""
}

func (ca *CodeAnalyzer) isBreakingChange(oldSig, newSig string) bool {
	// Simple heuristic: if parameters are removed or return type changes, it's breaking
	oldParamCount := strings.Count(oldSig, ",") + 1
	newParamCount := strings.Count(newSig, ",") + 1

	if newParamCount < oldParamCount {
		return true
	}

	// Check if return type changed (simplified)
	oldReturnStart := strings.LastIndex(oldSig, ") ")
	newReturnStart := strings.LastIndex(newSig, ") ")

	if oldReturnStart > 0 && newReturnStart > 0 {
		oldReturn := oldSig[oldReturnStart:]
		newReturn := newSig[newReturnStart:]
		return oldReturn != newReturn
	}

	return false
}

// Builder methods for context and sources

func (ca *CodeAnalyzer) buildCodeSource(commit Commit, item interface{}) Source {
	source := Source{
		Type:      "code",
		CommitSHA: commit.SHA,
		Files:     commit.Files,
		Timestamp: commit.Timestamp,
	}

	if commit.Author != "" {
		source.Participants = []string{commit.Author}
	}

	return source
}

func (ca *CodeAnalyzer) buildPatternContext(pattern *RefactoringPattern) map[string]interface{} {
	return map[string]interface{}{
		"pattern_type": pattern.Type,
		"confidence":   pattern.Confidence,
		"examples":     len(pattern.Examples),
	}
}

func (ca *CodeAnalyzer) buildAPIChangeContext(change APIChange) map[string]interface{} {
	return map[string]interface{}{
		"change_type":       change.Type,
		"function_name":     change.Function,
		"is_breaking":       change.Type == "breaking",
		"signature_changed": change.OldSig != change.NewSig,
	}
}

func (ca *CodeAnalyzer) buildBugFixContext(fix *BugFix) map[string]interface{} {
	return map[string]interface{}{
		"severity":        fix.Severity,
		"files_affected":  len(fix.Files),
		"has_description": fix.Description != "",
	}
}

func (ca *CodeAnalyzer) generateKnowledgeID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// ArchitecturalDecision represents a detected architectural decision
type ArchitecturalDecision struct {
	Content    string                 `json:"content"`
	Confidence float64                `json:"confidence"`
	Context    map[string]interface{} `json:"context"`
	Metadata   map[string]interface{} `json:"metadata"`
}
