// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package extraction

import (
	"regexp"
	"strings"
)

// PatternMatcher provides regex-based pattern detection for knowledge extraction
type PatternMatcher struct {
	patterns map[string]*regexp.Regexp
}

// NewPatternMatcher creates a new pattern matcher with predefined patterns
func NewPatternMatcher() *PatternMatcher {
	patterns := map[string]*regexp.Regexp{
		"decision": regexp.MustCompile(`(?i)(decided to|will use|going with|chose|selected|let's go with|we'll use|I recommend|the best option is|should use)`),
		"solution": regexp.MustCompile(`(?i)(fixed by|solved with|works when|solution is|to fix this|the fix is|resolved by|you can solve|try this|try starting|try running|here's how|here's the|typically occurs|usually happens)`),
		"preference": regexp.MustCompile(`(?i)(prefer|always|never|should|shouldn't|best practice|avoid|recommend|suggest|better to|it's better|I like)`),
		"problem": regexp.MustCompile(`(?i)(error|issue|problem|failing|broken|not working|doesn't work|won't work|bug|exception|crash)`),
		"certainty": regexp.MustCompile(`(?i)(definitely|certainly|absolutely|sure|confirmed|verified|tested|proven|guaranteed)`),
		"uncertainty": regexp.MustCompile(`(?i)(maybe|might|could|probably|possibly|not sure|uncertain|think|believe|seems)`),
		"question": regexp.MustCompile(`\?`),
		"code_block": regexp.MustCompile("```[\\s\\S]*?```"),
		"technical_term": regexp.MustCompile(`(?i)(API|database|server|client|function|method|class|interface|library|framework|service|endpoint)`),
	}

	return &PatternMatcher{
		patterns: patterns,
	}
}

// MatchesDecision checks if content contains decision indicators
func (pm *PatternMatcher) MatchesDecision(content string) bool {
	return pm.patterns["decision"].MatchString(content)
}

// MatchesSolution checks if content contains solution indicators
func (pm *PatternMatcher) MatchesSolution(content string) bool {
	return pm.patterns["solution"].MatchString(content)
}

// MatchesPreference checks if content contains preference indicators
func (pm *PatternMatcher) MatchesPreference(content string) bool {
	return pm.patterns["preference"].MatchString(content)
}

// MatchesProblem checks if content contains problem indicators
func (pm *PatternMatcher) MatchesProblem(content string) bool {
	return pm.patterns["problem"].MatchString(content)
}

// MatchesCertainty checks if content contains certainty indicators
func (pm *PatternMatcher) MatchesCertainty(content string) bool {
	return pm.patterns["certainty"].MatchString(content)
}

// MatchesUncertainty checks if content contains uncertainty indicators
func (pm *PatternMatcher) MatchesUncertainty(content string) bool {
	return pm.patterns["uncertainty"].MatchString(content)
}

// MatchesQuestion checks if content contains question indicators
func (pm *PatternMatcher) MatchesQuestion(content string) bool {
	return pm.patterns["question"].MatchString(content)
}

// ExtractCodeBlocks extracts code blocks from content
func (pm *PatternMatcher) ExtractCodeBlocks(content string) []string {
	return pm.patterns["code_block"].FindAllString(content, -1)
}

// ExtractTechnicalTerms extracts technical terms from content
func (pm *PatternMatcher) ExtractTechnicalTerms(content string) []string {
	matches := pm.patterns["technical_term"].FindAllString(content, -1)
	
	// Deduplicate and normalize
	termMap := make(map[string]bool)
	for _, match := range matches {
		term := strings.ToLower(strings.TrimSpace(match))
		if term != "" {
			termMap[term] = true
		}
	}
	
	var terms []string
	for term := range termMap {
		terms = append(terms, term)
	}
	
	return terms
}

// FindSentenceWithPattern finds the first sentence containing a specific pattern
func (pm *PatternMatcher) FindSentenceWithPattern(content string, patternName string) string {
	pattern, exists := pm.patterns[patternName]
	if !exists {
		return ""
	}
	
	sentences := pm.splitIntoSentences(content)
	for _, sentence := range sentences {
		if pattern.MatchString(sentence) {
			return strings.TrimSpace(sentence)
		}
	}
	
	return ""
}

// GetConfidenceMultiplier returns a confidence multiplier based on pattern strength
func (pm *PatternMatcher) GetConfidenceMultiplier(content string) float64 {
	multiplier := 1.0
	
	// Increase confidence for certainty indicators
	if pm.MatchesCertainty(content) {
		multiplier += 0.2
	}
	
	// Decrease confidence for uncertainty indicators
	if pm.MatchesUncertainty(content) {
		multiplier -= 0.3
	}
	
	// Increase confidence for technical content
	if len(pm.ExtractTechnicalTerms(content)) > 2 {
		multiplier += 0.1
	}
	
	// Ensure multiplier stays in reasonable range
	if multiplier < 0.1 {
		multiplier = 0.1
	} else if multiplier > 1.5 {
		multiplier = 1.5
	}
	
	return multiplier
}

// AnalyzePatterns provides a comprehensive analysis of patterns in content
func (pm *PatternMatcher) AnalyzePatterns(content string) PatternAnalysis {
	analysis := PatternAnalysis{
		Content: content,
		Patterns: make(map[string]bool),
		Confidence: 0.5, // Base confidence
	}
	
	// Check all patterns
	for name, pattern := range pm.patterns {
		if pattern.MatchString(content) {
			analysis.Patterns[name] = true
		}
	}
	
	// Calculate confidence based on patterns found
	analysis.Confidence = pm.calculatePatternConfidence(analysis.Patterns)
	
	// Extract additional information
	analysis.TechnicalTerms = pm.ExtractTechnicalTerms(content)
	analysis.CodeBlocks = pm.ExtractCodeBlocks(content)
	analysis.SentenceCount = len(pm.splitIntoSentences(content))
	
	return analysis
}

// calculatePatternConfidence calculates confidence based on detected patterns
func (pm *PatternMatcher) calculatePatternConfidence(patterns map[string]bool) float64 {
	confidence := 0.5 // Base confidence
	
	// High-confidence patterns
	highConfidencePatterns := []string{"decision", "solution", "certainty"}
	for _, pattern := range highConfidencePatterns {
		if patterns[pattern] {
			confidence += 0.15
		}
	}
	
	// Medium-confidence patterns
	mediumConfidencePatterns := []string{"preference", "problem", "technical_term"}
	for _, pattern := range mediumConfidencePatterns {
		if patterns[pattern] {
			confidence += 0.1
		}
	}
	
	// Adjust for uncertainty
	if patterns["uncertainty"] {
		confidence -= 0.2
	}
	
	// Cap confidence
	if confidence > 0.95 {
		confidence = 0.95
	} else if confidence < 0.1 {
		confidence = 0.1
	}
	
	return confidence
}

// splitIntoSentences splits content into sentences
func (pm *PatternMatcher) splitIntoSentences(content string) []string {
	// Simple sentence splitting - can be enhanced with more sophisticated NLP
	sentences := strings.Split(content, ".")
	
	var result []string
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 0 {
			result = append(result, sentence)
		}
	}
	
	return result
}

// PatternAnalysis contains the results of pattern analysis
type PatternAnalysis struct {
	Content        string            `json:"content"`
	Patterns       map[string]bool   `json:"patterns"`
	Confidence     float64           `json:"confidence"`
	TechnicalTerms []string          `json:"technical_terms"`
	CodeBlocks     []string          `json:"code_blocks"`
	SentenceCount  int               `json:"sentence_count"`
}

// HasPattern checks if a specific pattern was detected
func (pa PatternAnalysis) HasPattern(pattern string) bool {
	return pa.Patterns[pattern]
}

// GetPatternCount returns the number of patterns detected
func (pa PatternAnalysis) GetPatternCount() int {
	count := 0
	for _, found := range pa.Patterns {
		if found {
			count++
		}
	}
	return count
}

// IsHighConfidence returns true if the analysis has high confidence
func (pa PatternAnalysis) IsHighConfidence() bool {
	return pa.Confidence >= 0.8
}

// AddCustomPattern allows adding custom patterns to the matcher
func (pm *PatternMatcher) AddCustomPattern(name, pattern string) error {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	
	pm.patterns[name] = compiled
	return nil
}

// RemovePattern removes a pattern from the matcher
func (pm *PatternMatcher) RemovePattern(name string) {
	delete(pm.patterns, name)
}

// GetAvailablePatterns returns a list of available pattern names
func (pm *PatternMatcher) GetAvailablePatterns() []string {
	var patterns []string
	for name := range pm.patterns {
		patterns = append(patterns, name)
	}
	return patterns
}