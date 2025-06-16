// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package interfaces

// ProviderType represents a provider type
type ProviderType string

const (
	// ProviderOpenAI represents the OpenAI provider
	ProviderOpenAI ProviderType = "openai"

	// ProviderAnthropic represents the Anthropic provider
	ProviderAnthropic ProviderType = "anthropic"

	// ProviderOllama represents the Ollama provider
	ProviderOllama ProviderType = "ollama"

	// ProviderGoogle represents the Google Gemini provider
	ProviderGoogle ProviderType = "google"

	// ProviderClaudeCode represents the Claude Code provider
	ProviderClaudeCode ProviderType = "claude-code"

	// ProviderDeepSeek represents the DeepSeek provider
	ProviderDeepSeek ProviderType = "deepseek"

	// ProviderDeepInfra represents the DeepInfra provider
	ProviderDeepInfra ProviderType = "deepinfra"

	// ProviderOra represents the Ora provider
	ProviderOra ProviderType = "ora"
)
