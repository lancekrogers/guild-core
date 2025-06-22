package providers

import (
	"testing"
)

func TestProviderConstants(t *testing.T) {
	// Test IsValidProvider
	tests := []struct {
		provider string
		expected bool
	}{
		{"claude_code", true},
		{"claude-code", true},
		{"claudecode", true},
		{"ollama", true},
		{"openai", true},
		{"anthropic", true},
		{"deepseek", true},
		{"deepinfra", true},
		{"ora", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := IsValidProvider(tt.provider); got != tt.expected {
				t.Errorf("IsValidProvider(%q) = %v, want %v", tt.provider, got, tt.expected)
			}
		})
	}
}

func TestNormalizeProviderName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"claude_code", ProviderNameClaude},
		{"claude-code", ProviderNameClaude},
		{"claudecode", ProviderNameClaude},
		{"ollama", ProviderNameOllama},
		{"openai", ProviderNameOpenAI},
		{"anthropic", ProviderNameAnthropic},
		{"deepseek", ProviderNameDeepSeek},
		{"deepinfra", ProviderNameDeepInfra},
		{"ora", ProviderNameOra},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeProviderName(tt.input); got != tt.expected {
				t.Errorf("NormalizeProviderName(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestConvertToProviderType(t *testing.T) {
	tests := []struct {
		input    string
		expected ProviderType
	}{
		{"claude_code", ProviderClaudeCode},
		{"claude-code", ProviderClaudeCode},
		{"claudecode", ProviderClaudeCode},
		{"ollama", ProviderOllama},
		{"openai", ProviderOpenAI},
		{"anthropic", ProviderAnthropic},
		{"deepseek", ProviderDeepSeek},
		{"deepinfra", ProviderDeepInfra},
		{"ora", ProviderOra},
		{"unknown", ProviderType("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ConvertToProviderType(tt.input); got != tt.expected {
				t.Errorf("ConvertToProviderType(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetProviderDisplayName(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{ProviderNameClaude, DisplayNameClaude},
		{ProviderNameClaudeAlt, DisplayNameClaude},
		{ProviderNameClaudeCode, DisplayNameClaude},
		{ProviderNameOllama, DisplayNameOllama},
		{ProviderNameOpenAI, DisplayNameOpenAI},
		{ProviderNameAnthropic, DisplayNameAnthropic},
		{ProviderNameDeepSeek, DisplayNameDeepSeek},
		{ProviderNameDeepInfra, DisplayNameDeepInfra},
		{ProviderNameOra, DisplayNameOra},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := GetProviderDisplayName(tt.provider); got != tt.expected {
				t.Errorf("GetProviderDisplayName(%q) = %v, want %v", tt.provider, got, tt.expected)
			}
		})
	}
}

func TestGetProviderEnvVar(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{ProviderNameOpenAI, EnvOpenAIKey},
		{ProviderNameAnthropic, EnvAnthropicKey},
		{ProviderNameClaude, EnvAnthropicKey},
		{ProviderNameClaudeAlt, EnvAnthropicKey},
		{ProviderNameClaudeCode, EnvAnthropicKey},
		{ProviderNameDeepSeek, EnvDeepSeekKey},
		{ProviderNameDeepInfra, EnvDeepInfraKey},
		{ProviderNameOra, EnvOraKey},
		{ProviderNameOllama, ""},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := GetProviderEnvVar(tt.provider); got != tt.expected {
				t.Errorf("GetProviderEnvVar(%q) = %v, want %v", tt.provider, got, tt.expected)
			}
		})
	}
}

func TestGetProviderCategory(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{ProviderNameOllama, CategoryLocalProvider},
		{ProviderNameOpenAI, CategoryCloudProvider},
		{ProviderNameAnthropic, CategoryCloudProvider},
		{ProviderNameClaude, CategoryCloudProvider},
		{ProviderNameClaudeAlt, CategoryCloudProvider},
		{ProviderNameClaudeCode, CategoryCloudProvider},
		{ProviderNameDeepSeek, CategoryCloudProvider},
		{ProviderNameDeepInfra, CategoryCloudProvider},
		{ProviderNameOra, CategoryCloudProvider},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := GetProviderCategory(tt.provider); got != tt.expected {
				t.Errorf("GetProviderCategory(%q) = %v, want %v", tt.provider, got, tt.expected)
			}
		})
	}
}
