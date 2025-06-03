package interfaces

// ConfigurableProvider is a provider that supports runtime configuration
type ConfigurableProvider interface {
	AIProvider
	
	// SetMaxTokens sets the maximum tokens for completions
	SetMaxTokens(maxTokens int)
	
	// SetTemperature sets the temperature for completions
	SetTemperature(temperature float64)
	
	// SetTopP sets the top-p value for completions
	SetTopP(topP float64)
	
	// SetModel sets the model to use
	SetModel(model string)
	
	// GetConfig returns the current configuration
	GetConfig() map[string]interface{}
}