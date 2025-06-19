package providers

// Provider constants - Use these instead of magic strings
const (
	// Provider Names (string representations)
	ProviderNameClaude    = "claude_code"  // Used in config files
	ProviderNameClaudeAlt = "claude-code"  // Alternative format
	ProviderNameClaudeCode = "claudecode"  // Used in some places
	ProviderNameOllama    = "ollama"
	ProviderNameOpenAI    = "openai"
	ProviderNameAnthropic = "anthropic"
	ProviderNameDeepSeek  = "deepseek"
	ProviderNameDeepInfra = "deepinfra"
	ProviderNameOra       = "ora"
	
	// Default Models
	DefaultClaudeModel    = "claude-3-5-sonnet-20241022"
	DefaultOllamaModel    = "llama3.2"
	DefaultOpenAIModel    = "gpt-4o"
	DefaultAnthropicModel = "claude-3-5-sonnet-20241022"
	DefaultDeepSeekModel  = "deepseek-chat"
	DefaultDeepInfraModel = "meta-llama/Llama-3.3-70B-Instruct-Turbo"
	DefaultOraModel       = "gpt-4-turbo"
	
	// Provider Display Names
	DisplayNameClaude    = "Claude Code"
	DisplayNameOllama    = "Ollama"
	DisplayNameOpenAI    = "OpenAI"
	DisplayNameAnthropic = "Anthropic"
	DisplayNameDeepSeek  = "DeepSeek"
	DisplayNameDeepInfra = "DeepInfra"
	DisplayNameOra       = "Ora"
	
	// API Endpoints
	EndpointOpenAI    = "https://api.openai.com/v1"
	EndpointAnthropic = "https://api.anthropic.com"
	EndpointDeepSeek  = "https://api.deepseek.com/v1"
	EndpointDeepInfra = "https://api.deepinfra.com/v1/openai"
	EndpointOra       = "https://ora.ai/api"
	EndpointOllama    = "http://localhost:11434"
	
	// Environment Variable Names
	EnvOpenAIKey    = "OPENAI_API_KEY"
	EnvAnthropicKey = "ANTHROPIC_API_KEY"
	EnvDeepSeekKey  = "DEEPSEEK_API_KEY"
	EnvDeepInfraKey = "DEEPINFRA_API_KEY"
	EnvOraKey       = "ORA_API_KEY"
	
	// Provider Categories
	CategoryLocalProvider  = "local"
	CategoryCloudProvider  = "cloud"
	CategoryOpenSource     = "opensource"
	CategoryProprietary    = "proprietary"
)

// ProviderList returns all available provider names
var ProviderList = []string{
	ProviderNameClaude,
	ProviderNameOllama,
	ProviderNameOpenAI,
	ProviderNameAnthropic,
	ProviderNameDeepSeek,
	ProviderNameDeepInfra,
	ProviderNameOra,
}

// IsValidProvider checks if a provider name is valid
func IsValidProvider(provider string) bool {
	normalized := NormalizeProviderName(provider)
	for _, p := range ProviderList {
		if p == normalized {
			return true
		}
	}
	return false
}

// GetProviderDisplayName returns the display name for a provider
func GetProviderDisplayName(provider string) string {
	switch provider {
	case ProviderNameClaude, ProviderNameClaudeAlt, ProviderNameClaudeCode:
		return DisplayNameClaude
	case ProviderNameOllama:
		return DisplayNameOllama
	case ProviderNameOpenAI:
		return DisplayNameOpenAI
	case ProviderNameAnthropic:
		return DisplayNameAnthropic
	case ProviderNameDeepSeek:
		return DisplayNameDeepSeek
	case ProviderNameDeepInfra:
		return DisplayNameDeepInfra
	case ProviderNameOra:
		return DisplayNameOra
	default:
		return provider
	}
}

// GetDefaultModel returns the default model for a provider
func GetDefaultModel(provider string) string {
	switch provider {
	case ProviderNameClaude, ProviderNameClaudeAlt, ProviderNameClaudeCode:
		return DefaultClaudeModel
	case ProviderNameOllama:
		return DefaultOllamaModel
	case ProviderNameOpenAI:
		return DefaultOpenAIModel
	case ProviderNameAnthropic:
		return DefaultAnthropicModel
	case ProviderNameDeepSeek:
		return DefaultDeepSeekModel
	case ProviderNameDeepInfra:
		return DefaultDeepInfraModel
	case ProviderNameOra:
		return DefaultOraModel
	default:
		return ""
	}
}

// GetProviderEndpoint returns the API endpoint for a provider
func GetProviderEndpoint(provider string) string {
	switch provider {
	case ProviderNameOpenAI:
		return EndpointOpenAI
	case ProviderNameAnthropic:
		return EndpointAnthropic
	case ProviderNameDeepSeek:
		return EndpointDeepSeek
	case ProviderNameDeepInfra:
		return EndpointDeepInfra
	case ProviderNameOra:
		return EndpointOra
	case ProviderNameOllama:
		return EndpointOllama
	default:
		return ""
	}
}

// GetProviderEnvVar returns the environment variable name for a provider's API key
func GetProviderEnvVar(provider string) string {
	switch provider {
	case ProviderNameOpenAI:
		return EnvOpenAIKey
	case ProviderNameAnthropic, ProviderNameClaude, ProviderNameClaudeAlt, ProviderNameClaudeCode:
		return EnvAnthropicKey
	case ProviderNameDeepSeek:
		return EnvDeepSeekKey
	case ProviderNameDeepInfra:
		return EnvDeepInfraKey
	case ProviderNameOra:
		return EnvOraKey
	default:
		return ""
	}
}

// GetProviderCategory returns the category of a provider
func GetProviderCategory(provider string) string {
	switch provider {
	case ProviderNameOllama:
		return CategoryLocalProvider
	case ProviderNameOpenAI, ProviderNameAnthropic, ProviderNameClaude, ProviderNameClaudeAlt, ProviderNameClaudeCode, ProviderNameDeepSeek, ProviderNameDeepInfra, ProviderNameOra:
		return CategoryCloudProvider
	default:
		return ""
	}
}

// NormalizeProviderName converts various provider name formats to a standard format
func NormalizeProviderName(provider string) string {
	switch provider {
	case "claude_code", "claude-code", "claudecode":
		return ProviderNameClaude
	case "ollama":
		return ProviderNameOllama
	case "openai":
		return ProviderNameOpenAI
	case "anthropic":
		return ProviderNameAnthropic
	case "deepseek":
		return ProviderNameDeepSeek
	case "deepinfra":
		return ProviderNameDeepInfra
	case "ora":
		return ProviderNameOra
	default:
		return provider
	}
}

// ConvertToProviderType converts string provider names to ProviderType
func ConvertToProviderType(provider string) ProviderType {
	normalized := NormalizeProviderName(provider)
	switch normalized {
	case ProviderNameClaude:
		return ProviderClaudeCode
	case ProviderNameOllama:
		return ProviderOllama
	case ProviderNameOpenAI:
		return ProviderOpenAI
	case ProviderNameAnthropic:
		return ProviderAnthropic
	case ProviderNameDeepSeek:
		return ProviderDeepSeek
	case ProviderNameDeepInfra:
		return ProviderDeepInfra
	case ProviderNameOra:
		return ProviderOra
	default:
		return ProviderType(provider)
	}
}