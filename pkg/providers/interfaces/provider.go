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
)