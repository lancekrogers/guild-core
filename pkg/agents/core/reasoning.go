// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	thinkingRegex   = regexp.MustCompile(`(?s)<thinking>(.*?)</thinking>`)
	confidenceRegex = regexp.MustCompile(`(?i)confidence:\s*([\d.]+)`)
)

// AgentResponse represents a structured response from an agent execution
type AgentResponse struct {
	Content    string                 `json:"content"`
	Reasoning  string                 `json:"reasoning,omitempty"`
	Confidence float64                `json:"confidence,omitempty"`
	ToolsUsed  []string               `json:"tools_used,omitempty"`
	Cost       float64                `json:"cost,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ExtractReasoning extracts thinking blocks and confidence from agent response
func ExtractReasoning(content string) (cleanContent, reasoning string, confidence float64) {
	// Default confidence
	confidence = 0.5

	// Extract thinking blocks
	matches := thinkingRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return strings.TrimSpace(content), "", confidence
	}

	// Combine all thinking blocks
	var reasoningParts []string
	for _, match := range matches {
		if len(match) > 1 {
			reasoningParts = append(reasoningParts, strings.TrimSpace(match[1]))
		}
	}
	reasoning = strings.Join(reasoningParts, "\n\n")

	// Extract confidence from the last reasoning block (most recent confidence)
	if confMatch := confidenceRegex.FindAllStringSubmatch(reasoning, -1); len(confMatch) > 0 {
		lastMatch := confMatch[len(confMatch)-1]
		if len(lastMatch) > 1 {
			if conf, err := strconv.ParseFloat(lastMatch[1], 64); err == nil {
				// Clamp confidence to [0, 1]
				if conf > 1.0 {
					confidence = 1.0
				} else if conf < 0.0 {
					confidence = 0.0
				} else {
					confidence = conf
				}
			}
		}
	}

	// Remove thinking blocks from content
	cleanContent = thinkingRegex.ReplaceAllString(content, "")
	// Clean up extra whitespace
	cleanContent = strings.TrimSpace(cleanContent)
	// Replace multiple newlines with double newlines
	cleanContent = strings.ReplaceAll(cleanContent, "\n\n\n", "\n\n")

	return
}
