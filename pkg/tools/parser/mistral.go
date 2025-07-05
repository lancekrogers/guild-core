// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

// mistralParser handles Mistral's tool call format
type mistralParser struct {
	*openAIParser // Mistral uses OpenAI-compatible format
}

// NewMistralParser creates a new Mistral format parser
func NewMistralParser() ToolCallParser {
	return &mistralParser{
		openAIParser: &openAIParser{},
	}
}

// SupportedFormat returns the Mistral format
func (p *mistralParser) SupportedFormat() ProviderFormat {
	return FormatMistral
}

// mistralFormatter formats tools for Mistral
type mistralFormatter struct {
	*openAIFormatter // Mistral uses OpenAI-compatible format
}

// NewMistralFormatter creates a new Mistral formatter
func NewMistralFormatter() ToolFormatter {
	return &mistralFormatter{
		openAIFormatter: &openAIFormatter{},
	}
}

// SupportedFormat returns the Mistral format
func (f *mistralFormatter) SupportedFormat() ProviderFormat {
	return FormatMistral
}