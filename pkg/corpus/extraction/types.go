// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package extraction

import (
	"fmt"
	"time"

	"github.com/guild-framework/guild-core/pkg/storage/db"
)

// KnowledgeType represents different categories of extracted knowledge
type KnowledgeType int

const (
	KnowledgeDecision KnowledgeType = iota
	KnowledgeSolution
	KnowledgePattern
	KnowledgePreference
	KnowledgeConstraint
	KnowledgeContext
)

// String returns the string representation of the knowledge type
func (kt KnowledgeType) String() string {
	switch kt {
	case KnowledgeDecision:
		return "decision"
	case KnowledgeSolution:
		return "solution"
	case KnowledgePattern:
		return "pattern"
	case KnowledgePreference:
		return "preference"
	case KnowledgeConstraint:
		return "constraint"
	case KnowledgeContext:
		return "context"
	default:
		return "unknown"
	}
}

// ExtractedKnowledge represents a piece of knowledge extracted from conversations or code
type ExtractedKnowledge struct {
	ID         string                 `json:"id"`
	Type       KnowledgeType          `json:"type"`
	Content    string                 `json:"content"`
	Source     Source                 `json:"source"`
	Confidence float64                `json:"confidence"`
	Entities   []Entity               `json:"entities,omitempty"`
	Relations  []Relation             `json:"relations,omitempty"`
	Validation ValidationStatus       `json:"validation"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// Source represents where the knowledge was extracted from
type Source struct {
	Type         string    `json:"type"` // "chat", "code", "commit", etc.
	MessageIDs   []string  `json:"message_ids,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	CommitSHA    string    `json:"commit_sha,omitempty"`
	Files        []string  `json:"files,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	Participants []string  `json:"participants,omitempty"`
}

// Entity represents a named entity extracted from content
type Entity struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"` // "person", "technology", "concept", etc.
	Confidence float64 `json:"confidence"`
	Position   int     `json:"position,omitempty"` // Character position in source text
}

// Relation represents a relationship between entities
type Relation struct {
	Subject    string  `json:"subject"`
	Predicate  string  `json:"predicate"` // "uses", "depends_on", "replaces", etc.
	Object     string  `json:"object"`
	Confidence float64 `json:"confidence"`
}

// ValidationStatus represents the validation state of extracted knowledge
type ValidationStatus struct {
	Valid       bool      `json:"valid"`
	Confidence  float64   `json:"confidence"`
	Issues      []string  `json:"issues,omitempty"`
	ValidatedAt time.Time `json:"validated_at,omitempty"`
}

// Exchange represents a logical conversation unit
type Exchange struct {
	Messages  []db.ChatMessage `json:"messages"`
	StartTime time.Time        `json:"start_time"`
	EndTime   time.Time        `json:"end_time"`
}

// GetMessageIDs returns a slice of message IDs in the exchange
func (e Exchange) GetMessageIDs() []string {
	ids := make([]string, len(e.Messages))
	for i, msg := range e.Messages {
		ids[i] = msg.ID
	}
	return ids
}

// GetParticipants returns a slice of unique participants in the exchange
func (e Exchange) GetParticipants() []string {
	participants := make(map[string]bool)
	for _, msg := range e.Messages {
		participants[msg.Role] = true
	}

	var result []string
	for role := range participants {
		result = append(result, role)
	}
	return result
}

// RefactoringPattern represents a detected code refactoring pattern
type RefactoringPattern struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Confidence  float64                `json:"confidence"`
	Examples    []string               `json:"examples,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FunctionChange represents a modification to a function
type FunctionChange struct {
	Name       string `json:"name"`
	OldContent string `json:"old_content"`
	NewContent string `json:"new_content"`
	ChangeType string `json:"change_type"` // "added", "modified", "removed"
}

// TypeChange represents a modification to a type definition
type TypeChange struct {
	Name       string `json:"name"`
	OldDef     string `json:"old_definition"`
	NewDef     string `json:"new_definition"`
	ChangeType string `json:"change_type"`
}

// Commit represents a git commit for analysis
type Commit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
	Files     []string  `json:"files"`
}

// DiffAnalysis contains the results of analyzing a code diff
type DiffAnalysis struct {
	AffectedFiles     []string         `json:"affected_files"`
	AddedLines        int              `json:"added_lines"`
	RemovedLines      int              `json:"removed_lines"`
	ModifiedFunctions []FunctionChange `json:"modified_functions"`
	AddedImports      []string         `json:"added_imports"`
	RemovedImports    []string         `json:"removed_imports"`
	TypeChanges       []TypeChange     `json:"type_changes"`
}

// APIChange represents a change to an API
type APIChange struct {
	Type        string `json:"type"` // "breaking", "additive", "deprecation"
	Function    string `json:"function"`
	OldSig      string `json:"old_signature"`
	NewSig      string `json:"new_signature"`
	Description string `json:"description"`
}

// ToKnowledgeString converts an API change to a knowledge string
func (ac APIChange) ToKnowledgeString() string {
	return fmt.Sprintf("API Change (%s): %s - %s", ac.Type, ac.Function, ac.Description)
}

// BugFix represents a detected bug fix
type BugFix struct {
	Problem     string   `json:"problem"`
	Solution    string   `json:"solution"`
	Files       []string `json:"files"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
}

// ToKnowledgeString converts a bug fix to a knowledge string
func (bf BugFix) ToKnowledgeString() string {
	return fmt.Sprintf("Bug Fix: %s\nSolution: %s\nFiles: %v", bf.Problem, bf.Solution, bf.Files)
}

// ValidationIssue represents an issue found during knowledge validation
type ValidationIssue struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// ConflictInfo represents a conflict between pieces of knowledge
type ConflictInfo struct {
	Description            string `json:"description"`
	Severity               string `json:"severity"`
	ConflictingKnowledgeID string `json:"conflicting_knowledge_id"`
}

// FactCheckResult represents the result of fact-checking knowledge
type FactCheckResult struct {
	Verified    bool    `json:"verified"`
	Confidence  float64 `json:"confidence"`
	Source      string  `json:"source,omitempty"`
	Explanation string  `json:"explanation,omitempty"`
}
