// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// ThinkingType represents different types of thinking patterns
type ThinkingType string

const (
	ThinkingTypeAnalysis       ThinkingType = "analysis"
	ThinkingTypePlanning       ThinkingType = "planning"
	ThinkingTypeDecisionMaking ThinkingType = "decision_making"
	ThinkingTypeErrorRecovery  ThinkingType = "error_recovery"
	ThinkingTypeToolSelection  ThinkingType = "tool_selection"
	ThinkingTypeHypothesis     ThinkingType = "hypothesis"
	ThinkingTypeVerification   ThinkingType = "verification"
	ThinkingTypeOptimization   ThinkingType = "optimization"
)

// DecisionPoint represents a point where the agent made a choice
type DecisionPoint struct {
	ID            string                 `json:"id"`
	Decision      string                 `json:"decision"`
	Alternatives  []Alternative          `json:"alternatives"`
	Rationale     string                 `json:"rationale"`
	Confidence    float64                `json:"confidence"`
	Impact        ImpactLevel            `json:"impact"`
	Reversible    bool                   `json:"reversible"`
	ActualOutcome *Outcome               `json:"actual_outcome,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Alternative represents an option that was considered but not chosen
type Alternative struct {
	Option      string   `json:"option"`
	Pros        []string `json:"pros"`
	Cons        []string `json:"cons"`
	Confidence  float64  `json:"confidence"`
	WhyRejected string   `json:"why_rejected"`
}

// ImpactLevel represents the significance of a decision
type ImpactLevel string

const (
	ImpactLevelLow      ImpactLevel = "low"
	ImpactLevelMedium   ImpactLevel = "medium"
	ImpactLevelHigh     ImpactLevel = "high"
	ImpactLevelCritical ImpactLevel = "critical"
)

// Outcome represents the result of a decision
type Outcome struct {
	Success        bool                   `json:"success"`
	Description    string                 `json:"description"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
	LessonsLearned []string               `json:"lessons_learned,omitempty"`
}

// ToolContext represents tool-related thinking
type ToolContext struct {
	ToolName         string                 `json:"tool_name"`
	Purpose          string                 `json:"purpose"`
	ExpectedOutcome  string                 `json:"expected_outcome"`
	ActualOutcome    *string                `json:"actual_outcome,omitempty"`
	Parameters       map[string]interface{} `json:"parameters"`
	AlternativeTools []string               `json:"alternative_tools,omitempty"`
	Confidence       float64                `json:"confidence"`
}

// ThinkingBlock represents a structured unit of agent reasoning
type ThinkingBlock struct {
	ID             string                 `json:"id"`
	Type           ThinkingType           `json:"type"`
	Content        string                 `json:"content"`
	StructuredData *StructuredThinking    `json:"structured_data,omitempty"`
	Confidence     float64                `json:"confidence"`
	Depth          int                    `json:"depth"` // Nesting level
	ParentID       *string                `json:"parent_id,omitempty"`
	ChildIDs       []string               `json:"child_ids,omitempty"`
	DecisionPoints []DecisionPoint        `json:"decision_points,omitempty"`
	ToolContext    *ToolContext           `json:"tool_context,omitempty"`
	ErrorContext   *ErrorAnalysis         `json:"error_context,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	Duration       time.Duration          `json:"duration"`
	TokenCount     int                    `json:"token_count"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// StructuredThinking provides type-specific structured data
type StructuredThinking struct {
	// For analysis type
	Subject     string   `json:"subject,omitempty"`
	Findings    []string `json:"findings,omitempty"`
	Conclusions []string `json:"conclusions,omitempty"`

	// For planning type
	Goal         string   `json:"goal,omitempty"`
	Steps        []Step   `json:"steps,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	Risks        []Risk   `json:"risks,omitempty"`

	// For hypothesis type
	Statement  string   `json:"statement,omitempty"`
	Evidence   []string `json:"evidence,omitempty"`
	TestPlan   string   `json:"test_plan,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
}

// Step represents a planning step
type Step struct {
	Order     int      `json:"order"`
	Action    string   `json:"action"`
	Purpose   string   `json:"purpose"`
	Completed bool     `json:"completed"`
	Duration  string   `json:"duration,omitempty"`
	Blockers  []string `json:"blockers,omitempty"`
}

// Risk represents a potential issue
type Risk struct {
	Description string      `json:"description"`
	Probability float64     `json:"probability"`
	Impact      ImpactLevel `json:"impact"`
	Mitigation  string      `json:"mitigation"`
}

// ErrorAnalysis provides detailed error context
type ErrorAnalysis struct {
	ErrorType     string   `json:"error_type"`
	Description   string   `json:"description"`
	RootCause     string   `json:"root_cause"`
	Impact        string   `json:"impact"`
	Recovery      string   `json:"recovery_strategy"`
	Prevention    string   `json:"prevention"`
	RelatedErrors []string `json:"related_errors,omitempty"`
}

// ThinkingBlockParser extracts structured thinking from responses
type ThinkingBlockParser struct {
	patterns           map[ThinkingType]*regexp.Regexp
	typeDetector       *ThinkingTypeDetector
	structureExtractor *StructureExtractor
	metrics            *observability.MetricsRegistry
}

// NewThinkingBlockParser creates a new parser with sophisticated pattern matching
func NewThinkingBlockParser(metrics *observability.MetricsRegistry) *ThinkingBlockParser {
	return &ThinkingBlockParser{
		patterns: map[ThinkingType]*regexp.Regexp{
			ThinkingTypeAnalysis:       regexp.MustCompile(`(?s)<thinking[^>]*>.*?(?:analyz|examin|investigat|assess).*?</thinking>`),
			ThinkingTypePlanning:       regexp.MustCompile(`(?s)<thinking[^>]*>.*?(?:plan|strateg|approach|steps?).*?</thinking>`),
			ThinkingTypeDecisionMaking: regexp.MustCompile(`(?s)<thinking[^>]*>.*?(?:decid|choos|select|option).*?</thinking>`),
			ThinkingTypeErrorRecovery:  regexp.MustCompile(`(?s)<thinking[^>]*>.*?(?:error|fail|recover|fix).*?</thinking>`),
			ThinkingTypeToolSelection:  regexp.MustCompile(`(?s)<thinking[^>]*>.*?(?:tool|function|api|call).*?</thinking>`),
			ThinkingTypeHypothesis:     regexp.MustCompile(`(?s)<thinking[^>]*>.*?(?:hypothes|theory|assume|predict).*?</thinking>`),
			ThinkingTypeVerification:   regexp.MustCompile(`(?s)<thinking[^>]*>.*?(?:verif|check|confirm|validat).*?</thinking>`),
			ThinkingTypeOptimization:   regexp.MustCompile(`(?s)<thinking[^>]*>.*?(?:optim|improv|enhanc|refin|streamlin|performanc).*?</thinking>`),
		},
		typeDetector:       NewThinkingTypeDetector(),
		structureExtractor: NewStructureExtractor(),
		metrics:            metrics,
	}
}

// ParseThinkingBlocks extracts all thinking blocks from a response
func (p *ThinkingBlockParser) ParseThinkingBlocks(ctx context.Context, response string) ([]*ThinkingBlock, error) {
	startTime := time.Now()
	defer func() {
		if p.metrics != nil {
			p.metrics.RecordHistogram("thinking_block_parsing_seconds", time.Since(startTime).Seconds())
		}
	}()

	// Find all thinking blocks
	blockPattern := regexp.MustCompile(`(?s)<thinking(?:\s+type="([^"]+)")?>(.*?)</thinking>`)
	matches := blockPattern.FindAllStringSubmatch(response, -1)

	if len(matches) == 0 {
		return nil, nil
	}

	blocks := make([]*ThinkingBlock, 0, len(matches))

	for _, match := range matches {
		if ctx.Err() != nil {
			return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCanceled, "context cancelled during parsing").
				WithComponent("thinking_parser")
		}

		block, err := p.parseBlock(ctx, match[1], match[2])
		if err != nil {
			logger := observability.GetLogger(ctx)
			logger.WithError(err).WarnContext(ctx, "Failed to parse thinking block",
				"content_preview", truncate(match[2], 100))
			continue
		}

		blocks = append(blocks, block)
	}

	// Link nested blocks
	p.linkNestedBlocks(blocks)

	// Extract decision points across blocks
	p.extractDecisionPoints(blocks)

	if p.metrics != nil {
		p.metrics.RecordGauge("thinking_blocks_parsed", float64(len(blocks)))
	}

	return blocks, nil
}

// parseBlock parses a single thinking block
func (p *ThinkingBlockParser) parseBlock(ctx context.Context, typeHint, content string) (*ThinkingBlock, error) {
	blockID := uuid.New().String()
	startTime := time.Now()

	// Clean content
	content = strings.TrimSpace(content)

	// Determine type
	var thinkingType ThinkingType
	if typeHint != "" {
		thinkingType = ThinkingType(typeHint)
	} else {
		thinkingType = p.typeDetector.DetectType(content)
	}

	// Extract confidence
	confidence := p.extractConfidence(content)

	// Extract structured data based on type
	structuredData, err := p.structureExtractor.Extract(ctx, thinkingType, content)
	if err != nil {
		logger := observability.GetLogger(ctx)
		logger.WithError(err).DebugContext(ctx, "Failed to extract structured data",
			"type", thinkingType)
	}

	// Extract tool context if present
	toolContext := p.extractToolContext(content)

	// Extract error context if present
	errorContext := p.extractErrorContext(content)

	// Count tokens (simplified - in production, use proper tokenizer)
	tokenCount := len(strings.Fields(content)) * 4 / 3

	block := &ThinkingBlock{
		ID:             blockID,
		Type:           thinkingType,
		Content:        content,
		StructuredData: structuredData,
		Confidence:     confidence,
		Depth:          0, // Will be set by linkNestedBlocks
		ToolContext:    toolContext,
		ErrorContext:   errorContext,
		Timestamp:      startTime,
		Duration:       time.Since(startTime),
		TokenCount:     tokenCount,
		Metadata: map[string]interface{}{
			"parser_version": "1.0",
			"detected_type":  typeHint == "",
		},
	}

	return block, nil
}

// extractConfidence extracts confidence level from content
func (p *ThinkingBlockParser) extractConfidence(content string) float64 {
	// Multiple patterns for flexibility
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`[Cc]onfidence:\s*(\d+(?:\.\d+)?)`),
		regexp.MustCompile(`(\d+(?:\.\d+)?)\s*confidence`),
		regexp.MustCompile(`[Cc]onfidence\s*level:\s*(\d+(?:\.\d+)?)`),
		regexp.MustCompile(`(\d+)%\s*(?:confident|sure|certain)`),
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(content); len(matches) > 1 {
			var conf float64
			fmt.Sscanf(matches[1], "%f", &conf)

			// Handle percentage
			if strings.Contains(content, "%") && conf > 1 {
				conf = conf / 100
			}

			// Ensure valid range
			if conf >= 0 && conf <= 1 {
				return conf
			}
		}
	}

	// Default based on certain keywords
	if strings.Contains(strings.ToLower(content), "uncertain") {
		return 0.3
	} else if strings.Contains(strings.ToLower(content), "likely") {
		return 0.7
	} else if strings.Contains(strings.ToLower(content), "definitely") {
		return 0.9
	}

	return 0.5 // Default medium confidence
}

// extractToolContext extracts tool-related information
func (p *ThinkingBlockParser) extractToolContext(content string) *ToolContext {
	toolPattern := regexp.MustCompile(`(?i)(?:use|call|invoke|execute)\s+(?:tool|function):\s*(\w+)`)
	purposePattern := regexp.MustCompile(`(?i)(?:purpose|to|for):\s*([^.]+)`)

	toolMatches := toolPattern.FindStringSubmatch(content)
	if len(toolMatches) < 2 {
		return nil
	}

	tc := &ToolContext{
		ToolName:   toolMatches[1],
		Confidence: p.extractConfidence(content),
	}

	if purposeMatches := purposePattern.FindStringSubmatch(content); len(purposeMatches) > 1 {
		tc.Purpose = strings.TrimSpace(purposeMatches[1])
	}

	// Extract expected outcome
	outcomePattern := regexp.MustCompile(`(?i)expect(?:ed)?:\s*([^.]+)`)
	if outcomeMatches := outcomePattern.FindStringSubmatch(content); len(outcomeMatches) > 1 {
		tc.ExpectedOutcome = strings.TrimSpace(outcomeMatches[1])
	}

	return tc
}

// extractErrorContext extracts error analysis information
func (p *ThinkingBlockParser) extractErrorContext(content string) *ErrorAnalysis {
	if !strings.Contains(strings.ToLower(content), "error") &&
		!strings.Contains(strings.ToLower(content), "fail") {
		return nil
	}

	ea := &ErrorAnalysis{}

	// Extract error type
	typePattern := regexp.MustCompile(`(?i)error\s+type:\s*([^.]+)`)
	if matches := typePattern.FindStringSubmatch(content); len(matches) > 1 {
		ea.ErrorType = strings.TrimSpace(matches[1])
	}

	// Extract root cause
	causePattern := regexp.MustCompile(`(?i)(?:root\s+)?cause:\s*([^.]+)`)
	if matches := causePattern.FindStringSubmatch(content); len(matches) > 1 {
		ea.RootCause = strings.TrimSpace(matches[1])
	}

	// Extract recovery strategy
	recoveryPattern := regexp.MustCompile(`(?i)recover(?:y)?:\s*([^.]+)`)
	if matches := recoveryPattern.FindStringSubmatch(content); len(matches) > 1 {
		ea.Recovery = strings.TrimSpace(matches[1])
	}

	return ea
}

// linkNestedBlocks establishes parent-child relationships
func (p *ThinkingBlockParser) linkNestedBlocks(blocks []*ThinkingBlock) {
	// Simple heuristic: blocks that reference other blocks or appear to be sub-thoughts
	for i, block := range blocks {
		// Check if this block references a previous block
		for j := 0; j < i; j++ {
			if strings.Contains(block.Content, "building on") ||
				strings.Contains(block.Content, "following from") ||
				strings.Contains(block.Content, "based on the above") {
				parentID := blocks[j].ID
				block.ParentID = &parentID
				block.Depth = blocks[j].Depth + 1
				blocks[j].ChildIDs = append(blocks[j].ChildIDs, block.ID)
				break
			}
		}
	}
}

// extractDecisionPoints finds decision points across all blocks
func (p *ThinkingBlockParser) extractDecisionPoints(blocks []*ThinkingBlock) {
	decisionPattern := regexp.MustCompile(`(?i)(?:decide|choose|select).*?(?:between|from|option)`)

	for _, block := range blocks {
		if !decisionPattern.MatchString(block.Content) {
			continue
		}

		// Extract alternatives
		alternativePattern := regexp.MustCompile(`(?i)(?:option|alternative|choice)\s*\d*:\s*([^.]+)`)
		alternatives := alternativePattern.FindAllStringSubmatch(block.Content, -1)

		if len(alternatives) > 1 {
			dp := DecisionPoint{
				ID:           uuid.New().String(),
				Confidence:   block.Confidence,
				Timestamp:    block.Timestamp,
				Alternatives: make([]Alternative, 0, len(alternatives)),
			}

			for _, alt := range alternatives {
				dp.Alternatives = append(dp.Alternatives, Alternative{
					Option: strings.TrimSpace(alt[1]),
				})
			}

			block.DecisionPoints = append(block.DecisionPoints, dp)
		}
	}
}

// ThinkingTypeDetector intelligently detects thinking type from content
type ThinkingTypeDetector struct {
	keywordMap map[ThinkingType][]string
}

// NewThinkingTypeDetector creates a new type detector
func NewThinkingTypeDetector() *ThinkingTypeDetector {
	return &ThinkingTypeDetector{
		keywordMap: map[ThinkingType][]string{
			ThinkingTypeAnalysis:       {"analyze", "examine", "investigate", "assess", "evaluate", "review"},
			ThinkingTypePlanning:       {"plan", "strategy", "approach", "steps", "roadmap", "outline"},
			ThinkingTypeDecisionMaking: {"decide", "choose", "select", "option", "alternative", "preference"},
			ThinkingTypeErrorRecovery:  {"error", "fail", "recover", "fix", "resolve", "troubleshoot"},
			ThinkingTypeToolSelection:  {"tool", "function", "api", "call", "invoke", "execute"},
			ThinkingTypeHypothesis:     {"hypothesis", "theory", "assume", "predict", "suppose", "expect"},
			ThinkingTypeVerification:   {"verify", "check", "confirm", "validate", "ensure", "test"},
			ThinkingTypeOptimization:   {"optimize", "improve", "enhance", "refine", "streamline", "performance"},
		},
	}
}

// DetectType determines the thinking type from content
func (d *ThinkingTypeDetector) DetectType(content string) ThinkingType {
	contentLower := strings.ToLower(content)
	scores := make(map[ThinkingType]int)

	// Score each type based on keyword matches
	for thinkingType, keywords := range d.keywordMap {
		for _, keyword := range keywords {
			if strings.Contains(contentLower, keyword) {
				scores[thinkingType]++
			}
		}
	}

	// Find highest scoring type
	var bestType ThinkingType = ThinkingTypeAnalysis // default
	maxScore := 0

	for thinkingType, score := range scores {
		if score > maxScore {
			maxScore = score
			bestType = thinkingType
		}
	}

	return bestType
}

// StructureExtractor extracts structured data based on thinking type
type StructureExtractor struct {
	extractors map[ThinkingType]func(string) (*StructuredThinking, error)
}

// NewStructureExtractor creates a new structure extractor
func NewStructureExtractor() *StructureExtractor {
	se := &StructureExtractor{
		extractors: make(map[ThinkingType]func(string) (*StructuredThinking, error)),
	}

	// Register type-specific extractors
	se.extractors[ThinkingTypeAnalysis] = se.extractAnalysis
	se.extractors[ThinkingTypePlanning] = se.extractPlanning
	se.extractors[ThinkingTypeHypothesis] = se.extractHypothesis

	return se
}

// Extract extracts structured data based on type
func (se *StructureExtractor) Extract(ctx context.Context, thinkingType ThinkingType, content string) (*StructuredThinking, error) {
	extractor, exists := se.extractors[thinkingType]
	if !exists {
		return nil, nil // No specific extractor, return nil
	}

	return extractor(content)
}

// extractAnalysis extracts analysis-specific structure
func (se *StructureExtractor) extractAnalysis(content string) (*StructuredThinking, error) {
	st := &StructuredThinking{
		Findings:    []string{},
		Conclusions: []string{},
	}

	// Extract subject
	subjectPattern := regexp.MustCompile(`(?i)analyz(?:ing|e)\s+([^.]+)`)
	if matches := subjectPattern.FindStringSubmatch(content); len(matches) > 1 {
		st.Subject = strings.TrimSpace(matches[1])
	}

	// Extract findings (bullet points or numbered items)
	findingPattern := regexp.MustCompile(`(?m)^[\s-*•]+(.+)$`)
	findings := findingPattern.FindAllStringSubmatch(content, -1)
	for _, finding := range findings {
		st.Findings = append(st.Findings, strings.TrimSpace(finding[1]))
	}

	// Extract conclusions
	conclusionPattern := regexp.MustCompile(`(?i)(?:conclud|therefore|thus|result).*?:?\s*([^.]+)`)
	conclusions := conclusionPattern.FindAllStringSubmatch(content, -1)
	for _, conclusion := range conclusions {
		st.Conclusions = append(st.Conclusions, strings.TrimSpace(conclusion[1]))
	}

	return st, nil
}

// extractPlanning extracts planning-specific structure
func (se *StructureExtractor) extractPlanning(content string) (*StructuredThinking, error) {
	st := &StructuredThinking{
		Steps: []Step{},
		Risks: []Risk{},
	}

	// Extract goal
	goalPattern := regexp.MustCompile(`(?i)goal:\s*([^.]+)`)
	if matches := goalPattern.FindStringSubmatch(content); len(matches) > 1 {
		st.Goal = strings.TrimSpace(matches[1])
	}

	// Extract steps
	stepPattern := regexp.MustCompile(`(?i)(?:step\s*)?(\d+)[\s:.)]+(.+)`)
	steps := stepPattern.FindAllStringSubmatch(content, -1)
	for _, step := range steps {
		order := 0
		fmt.Sscanf(step[1], "%d", &order)
		st.Steps = append(st.Steps, Step{
			Order:  order,
			Action: strings.TrimSpace(step[2]),
		})
	}

	// Extract risks
	riskPattern := regexp.MustCompile(`(?i)risk:\s*([^.]+)`)
	risks := riskPattern.FindAllStringSubmatch(content, -1)
	for _, risk := range risks {
		st.Risks = append(st.Risks, Risk{
			Description: strings.TrimSpace(risk[1]),
			Impact:      ImpactLevelMedium, // Default
		})
	}

	return st, nil
}

// extractHypothesis extracts hypothesis-specific structure
func (se *StructureExtractor) extractHypothesis(content string) (*StructuredThinking, error) {
	st := &StructuredThinking{
		Evidence: []string{},
	}

	// Extract hypothesis statement
	hypPattern := regexp.MustCompile(`(?i)hypothes[ie]s:\s*([^.]+)`)
	if matches := hypPattern.FindStringSubmatch(content); len(matches) > 1 {
		st.Statement = strings.TrimSpace(matches[1])
	}

	// Extract evidence
	evidencePattern := regexp.MustCompile(`(?i)evidence:\s*([^.]+)`)
	evidence := evidencePattern.FindAllStringSubmatch(content, -1)
	for _, ev := range evidence {
		st.Evidence = append(st.Evidence, strings.TrimSpace(ev[1]))
	}

	// Extract confidence
	st.Confidence = se.extractConfidenceFromContent(content)

	return st, nil
}

// extractConfidenceFromContent is a helper to extract confidence
func (se *StructureExtractor) extractConfidenceFromContent(content string) float64 {
	confPattern := regexp.MustCompile(`(?i)confidence:\s*(\d+(?:\.\d+)?)`)
	if matches := confPattern.FindStringSubmatch(content); len(matches) > 1 {
		var conf float64
		fmt.Sscanf(matches[1], "%f", &conf)
		if conf > 1 {
			conf = conf / 100
		}
		return conf
	}
	return 0.5
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// MarshalJSON implements custom JSON marshaling
func (tb *ThinkingBlock) MarshalJSON() ([]byte, error) {
	type Alias ThinkingBlock
	return json.Marshal(&struct {
		*Alias
		Duration string `json:"duration"`
	}{
		Alias:    (*Alias)(tb),
		Duration: tb.Duration.String(),
	})
}
