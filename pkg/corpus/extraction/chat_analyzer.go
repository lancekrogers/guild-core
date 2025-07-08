// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package extraction

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/storage/db"
)

// ChatAnalyzer extracts knowledge from chat conversations
type ChatAnalyzer struct {
	nlp              *NLPProcessor
	patternMatcher   *PatternMatcher
	knowledgeBuilder *KnowledgeBuilder
}

// NewChatAnalyzer creates a new chat analyzer with default components
func NewChatAnalyzer(ctx context.Context) (*ChatAnalyzer, error) {
	nlp, err := NewNLPProcessor(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create NLP processor").
			WithComponent("corpus.extraction").
			WithOperation("NewChatAnalyzer")
	}

	patternMatcher := NewPatternMatcher()
	knowledgeBuilder := NewKnowledgeBuilder()

	return &ChatAnalyzer{
		nlp:              nlp,
		patternMatcher:   patternMatcher,
		knowledgeBuilder: knowledgeBuilder,
	}, nil
}

// AnalyzeConversation extracts knowledge from a sequence of chat messages
func (ca *ChatAnalyzer) AnalyzeConversation(ctx context.Context, messages []db.ChatMessage) ([]ExtractedKnowledge, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.extraction").
			WithOperation("AnalyzeConversation")
	}

	if len(messages) == 0 {
		return []ExtractedKnowledge{}, nil
	}

	var knowledge []ExtractedKnowledge

	// Group messages into meaningful exchanges
	exchanges, err := ca.groupIntoExchanges(ctx, messages)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to group messages into exchanges").
			WithComponent("corpus.extraction").
			WithOperation("AnalyzeConversation")
	}

	for _, exchange := range exchanges {
		// Check for context cancellation periodically
		if ctx.Err() != nil {
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled during analysis").
				WithComponent("corpus.extraction").
				WithOperation("AnalyzeConversation")
		}

		// Extract different types of knowledge from the exchange
		if ca.isDecisionPoint(ctx, exchange) {
			k, err := ca.extractDecision(ctx, exchange)
			if err == nil {
				knowledge = append(knowledge, k)
			}
		}

		if ca.isSolutionPattern(ctx, exchange) {
			k, err := ca.extractSolution(ctx, exchange)
			if err == nil {
				knowledge = append(knowledge, k)
			}
		}

		if ca.isPreferenceStatement(ctx, exchange) {
			k, err := ca.extractPreference(ctx, exchange)
			if err == nil {
				knowledge = append(knowledge, k)
			}
		}

		// Extract entities and relations using NLP
		entities, err := ca.nlp.ExtractEntities(ctx, exchange)
		if err != nil {
			continue // Log but don't fail the entire analysis
		}

		relations, err := ca.nlp.ExtractRelations(ctx, exchange)
		if err != nil {
			continue // Log but don't fail the entire analysis
		}

		if len(entities) > 0 || len(relations) > 0 {
			k := ExtractedKnowledge{
				ID:         ca.generateKnowledgeID(),
				Type:       KnowledgeContext,
				Entities:   entities,
				Relations:  relations,
				Source:     ca.buildSource(exchange),
				Confidence: 0.7, // Moderate confidence for entity/relation extraction
				Timestamp:  time.Now(),
			}
			knowledge = append(knowledge, k)
		}
	}

	return knowledge, nil
}

// groupIntoExchanges organizes messages into logical conversation units
func (ca *ChatAnalyzer) groupIntoExchanges(ctx context.Context, messages []db.ChatMessage) ([]Exchange, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var exchanges []Exchange
	var currentExchange Exchange

	for i, msg := range messages {
		// Check for context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Start new exchange on user messages (role = "user")
		if msg.Role == "user" && len(currentExchange.Messages) > 0 {
			exchanges = append(exchanges, currentExchange)
			currentExchange = Exchange{Messages: []db.ChatMessage{}}
		}

		currentExchange.Messages = append(currentExchange.Messages, msg)

		// Set exchange metadata
		if i == 0 || msg.Role == "user" {
			if msg.CreatedAt != nil {
				currentExchange.StartTime = *msg.CreatedAt
			}
		}
		if msg.CreatedAt != nil {
			currentExchange.EndTime = *msg.CreatedAt
		}
	}

	// Add the last exchange if it has messages
	if len(currentExchange.Messages) > 0 {
		exchanges = append(exchanges, currentExchange)
	}

	return exchanges, nil
}

// isDecisionPoint detects if an exchange contains a decision
func (ca *ChatAnalyzer) isDecisionPoint(ctx context.Context, exchange Exchange) bool {
	if ctx.Err() != nil {
		return false
	}

	// Look for decision patterns in assistant responses
	for _, msg := range exchange.Messages {
		if msg.Role == "assistant" && ca.patternMatcher.MatchesDecision(msg.Content) {
			// Additional validation: check if there was a question that led to this decision
			if ca.hasUserQuestion(exchange) {
				return true
			}
		}
	}
	return false
}

// isSolutionPattern detects if an exchange contains a problem-solution pattern
func (ca *ChatAnalyzer) isSolutionPattern(ctx context.Context, exchange Exchange) bool {
	if ctx.Err() != nil {
		return false
	}

	hasProblem := false
	hasSolution := false

	for _, msg := range exchange.Messages {
		if ca.patternMatcher.MatchesProblem(msg.Content) {
			hasProblem = true
		}
		if ca.patternMatcher.MatchesSolution(msg.Content) {
			hasSolution = true
		}
	}

	return hasProblem && hasSolution
}

// isPreferenceStatement detects if an exchange contains preference statements
func (ca *ChatAnalyzer) isPreferenceStatement(ctx context.Context, exchange Exchange) bool {
	if ctx.Err() != nil {
		return false
	}

	for _, msg := range exchange.Messages {
		if ca.patternMatcher.MatchesPreference(msg.Content) {
			return true
		}
	}
	return false
}

// extractDecision extracts decision knowledge from an exchange
func (ca *ChatAnalyzer) extractDecision(ctx context.Context, exchange Exchange) (ExtractedKnowledge, error) {
	if ctx.Err() != nil {
		return ExtractedKnowledge{}, ctx.Err()
	}

	decision := ExtractedKnowledge{
		ID:        ca.generateKnowledgeID(),
		Type:      KnowledgeDecision,
		Timestamp: time.Now(),
	}

	// Find the decision content
	for _, msg := range exchange.Messages {
		if msg.Role == "assistant" && ca.patternMatcher.MatchesDecision(msg.Content) {
			decision.Content = ca.extractDecisionContent(msg.Content)
			decision.Confidence = ca.calculateConfidence(ctx, exchange)
			break
		}
	}

	// Extract context from the entire exchange
	decision.Source = ca.buildSource(exchange)
	decision.Context = ca.extractExchangeContext(ctx, exchange)

	return decision, nil
}

// extractSolution extracts solution knowledge from an exchange
func (ca *ChatAnalyzer) extractSolution(ctx context.Context, exchange Exchange) (ExtractedKnowledge, error) {
	if ctx.Err() != nil {
		return ExtractedKnowledge{}, ctx.Err()
	}

	solution := ExtractedKnowledge{
		ID:        ca.generateKnowledgeID(),
		Type:      KnowledgeSolution,
		Timestamp: time.Now(),
	}

	// Extract problem and solution statements
	problem := ca.extractProblemStatement(ctx, exchange)
	solutionSteps := ca.extractSolutionSteps(ctx, exchange)

	solution.Content = fmt.Sprintf("Problem: %s\n\nSolution: %s", problem, solutionSteps)

	// Add code snippets if present
	codeBlocks := ca.extractCodeBlocks(ctx, exchange)
	if len(codeBlocks) > 0 {
		if solution.Metadata == nil {
			solution.Metadata = make(map[string]interface{})
		}
		solution.Metadata["code_snippets"] = codeBlocks
	}

	solution.Confidence = ca.assessSolutionConfidence(ctx, exchange)
	solution.Source = ca.buildSource(exchange)
	solution.Context = ca.extractExchangeContext(ctx, exchange)

	return solution, nil
}

// extractPreference extracts preference knowledge from an exchange
func (ca *ChatAnalyzer) extractPreference(ctx context.Context, exchange Exchange) (ExtractedKnowledge, error) {
	if ctx.Err() != nil {
		return ExtractedKnowledge{}, ctx.Err()
	}

	preference := ExtractedKnowledge{
		ID:        ca.generateKnowledgeID(),
		Type:      KnowledgePreference,
		Timestamp: time.Now(),
	}

	// Find preference statements
	for _, msg := range exchange.Messages {
		if ca.patternMatcher.MatchesPreference(msg.Content) {
			preference.Content = ca.extractPreferenceContent(msg.Content)
			preference.Confidence = 0.8 // High confidence for explicit preferences
			break
		}
	}

	preference.Source = ca.buildSource(exchange)
	preference.Context = ca.extractExchangeContext(ctx, exchange)

	return preference, nil
}

// Helper methods for content extraction

func (ca *ChatAnalyzer) hasUserQuestion(exchange Exchange) bool {
	for _, msg := range exchange.Messages {
		if msg.Role == "user" && strings.Contains(msg.Content, "?") {
			return true
		}
	}
	return false
}

func (ca *ChatAnalyzer) extractDecisionContent(content string) string {
	// Extract the key decision from the content
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if ca.patternMatcher.MatchesDecision(line) {
			return line
		}
	}
	return content[:min(200, len(content))] // Fallback to first 200 chars
}

func (ca *ChatAnalyzer) extractProblemStatement(ctx context.Context, exchange Exchange) string {
	for _, msg := range exchange.Messages {
		if ca.patternMatcher.MatchesProblem(msg.Content) {
			return ca.extractRelevantSentence(msg.Content, ca.patternMatcher.patterns["problem"])
		}
	}
	return "Problem statement not clearly identified"
}

func (ca *ChatAnalyzer) extractSolutionSteps(ctx context.Context, exchange Exchange) string {
	var solutions []string
	for _, msg := range exchange.Messages {
		if ca.patternMatcher.MatchesSolution(msg.Content) {
			solution := ca.extractRelevantSentence(msg.Content, ca.patternMatcher.patterns["solution"])
			if solution != "" {
				solutions = append(solutions, solution)
			}
		}
	}
	return strings.Join(solutions, "\n")
}

func (ca *ChatAnalyzer) extractPreferenceContent(content string) string {
	return ca.extractRelevantSentence(content, ca.patternMatcher.patterns["preference"])
}

func (ca *ChatAnalyzer) extractRelevantSentence(content string, pattern *regexp.Regexp) string {
	sentences := strings.Split(content, ". ")
	for _, sentence := range sentences {
		if pattern.MatchString(sentence) {
			return strings.TrimSpace(sentence)
		}
	}
	return content[:min(150, len(content))] // Fallback
}

func (ca *ChatAnalyzer) extractCodeBlocks(ctx context.Context, exchange Exchange) []string {
	var codeBlocks []string
	codePattern := regexp.MustCompile("```[\\s\\S]*?```")

	for _, msg := range exchange.Messages {
		matches := codePattern.FindAllString(msg.Content, -1)
		codeBlocks = append(codeBlocks, matches...)
	}

	return codeBlocks
}

func (ca *ChatAnalyzer) extractExchangeContext(ctx context.Context, exchange Exchange) map[string]interface{} {
	context := make(map[string]interface{})

	// Extract participants
	participants := make(map[string]bool)
	for _, msg := range exchange.Messages {
		participants[msg.Role] = true
	}

	var roleList []string
	for role := range participants {
		roleList = append(roleList, role)
	}
	context["participants"] = roleList

	// Extract message count
	context["message_count"] = len(exchange.Messages)

	// Extract time span
	if !exchange.StartTime.IsZero() && !exchange.EndTime.IsZero() {
		duration := exchange.EndTime.Sub(exchange.StartTime)
		context["duration_seconds"] = duration.Seconds()
	}

	return context
}

// calculateConfidence estimates confidence based on exchange characteristics
func (ca *ChatAnalyzer) calculateConfidence(ctx context.Context, exchange Exchange) float64 {
	confidence := 0.5 // Base confidence

	// Increase confidence for longer exchanges
	if len(exchange.Messages) > 3 {
		confidence += 0.2
	}

	// Increase confidence if there's clear back-and-forth
	hasUserQuestion := ca.hasUserQuestion(exchange)
	if hasUserQuestion {
		confidence += 0.2
	}

	// Check for certainty indicators
	for _, msg := range exchange.Messages {
		if ca.patternMatcher.MatchesCertainty(msg.Content) {
			confidence += 0.1
			break
		}
	}

	// Cap confidence at 0.95
	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}

func (ca *ChatAnalyzer) assessSolutionConfidence(ctx context.Context, exchange Exchange) float64 {
	confidence := 0.6 // Base confidence for solutions

	// Check if solution was tested or verified
	for _, msg := range exchange.Messages {
		content := strings.ToLower(msg.Content)
		if strings.Contains(content, "works") || strings.Contains(content, "tested") ||
			strings.Contains(content, "verified") || strings.Contains(content, "fixed") {
			confidence += 0.25
			break
		}
	}

	// Check for code examples
	if len(ca.extractCodeBlocks(ctx, exchange)) > 0 {
		confidence += 0.1
	}

	if confidence > 0.95 {
		confidence = 0.95
	}
	return confidence
}

func (ca *ChatAnalyzer) buildSource(exchange Exchange) Source {
	messageIDs := make([]string, len(exchange.Messages))
	for i, msg := range exchange.Messages {
		messageIDs[i] = msg.ID
	}

	participants := make(map[string]bool)
	for _, msg := range exchange.Messages {
		participants[msg.Role] = true
	}

	var participantList []string
	for role := range participants {
		participantList = append(participantList, role)
	}

	return Source{
		Type:         "chat",
		MessageIDs:   messageIDs,
		Timestamp:    exchange.StartTime,
		Participants: participantList,
		SessionID:    getSessionID(exchange),
	}
}

func (ca *ChatAnalyzer) generateKnowledgeID() string {
	return fmt.Sprintf("knowledge_%d", time.Now().UnixNano())
}

func getSessionID(exchange Exchange) string {
	if len(exchange.Messages) > 0 {
		return exchange.Messages[0].SessionID
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
