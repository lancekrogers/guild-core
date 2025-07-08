// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"regexp"
	"strconv"
	"strings"
)

var thinkingRegex = regexp.MustCompile(`(?s)<thinking>(.*?)</thinking>`)
var confidenceRegex = regexp.MustCompile(`(?i)confidence:\s*([\d.]+)`)

// AgentResponse represents a structured response from an agent execution
type AgentResponse struct {
	Content     string                 `json:"content"`
	Reasoning   string                 `json:"reasoning,omitempty"`
	Confidence  float64                `json:"confidence,omitempty"`
	ToolsUsed   []string               `json:"tools_used,omitempty"`
	Cost        float64                `json:"cost,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ExtractReasoning extracts thinking blocks and confidence from agent response
func ExtractReasoning(content string) (cleanContent, reasoning string, confidence float64) {
	// Extract thinking blocks
	matches := thinkingRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return content, "", 0.5
	}
	
	// Combine all thinking blocks
	var reasoningParts []string
	for _, match := range matches {
		if len(match) > 1 {
			reasoningParts = append(reasoningParts, strings.TrimSpace(match[1]))
		}
	}
	reasoning = strings.Join(reasoningParts, "\n\n")
	
	// Extract confidence
	if confMatch := confidenceRegex.FindStringSubmatch(reasoning); len(confMatch) > 1 {
		if conf, err := strconv.ParseFloat(confMatch[1], 64); err == nil {
			confidence = conf
		}
	}
	if confidence == 0 {
		confidence = 0.5 // default
	}
	
	// Remove thinking blocks from content
	cleanContent = thinkingRegex.ReplaceAllString(content, "")
	cleanContent = strings.TrimSpace(cleanContent)
	
	return
}