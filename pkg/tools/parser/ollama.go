// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

// ollamaParser handles Ollama's tool call format (similar to OpenAI)
type ollamaParser struct {
	*openAIParser // Ollama uses similar format to OpenAI
}

// NewOllamaParser creates a new Ollama format parser
func NewOllamaParser() ToolCallParser {
	return &ollamaParser{
		openAIParser: &openAIParser{},
	}
}

// SupportedFormat returns the Ollama format
func (p *ollamaParser) SupportedFormat() ProviderFormat {
	return FormatOllama
}

// ollamaFormatter formats tools for Ollama
type ollamaFormatter struct {
	*openAIFormatter // Ollama uses similar format to OpenAI
}

// NewOllamaFormatter creates a new Ollama formatter
func NewOllamaFormatter() ToolFormatter {
	return &ollamaFormatter{
		openAIFormatter: &openAIFormatter{},
	}
}

// SupportedFormat returns the Ollama format
func (f *ollamaFormatter) SupportedFormat() ProviderFormat {
	return FormatOllama
}