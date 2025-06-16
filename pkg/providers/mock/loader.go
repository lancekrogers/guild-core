package mock

import (
	_ "embed"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed responses.yaml
var embeddedResponses []byte

// ResponseSet contains all mock responses
type ResponseSet struct {
	Responses []Response `yaml:"responses"`
}

// Response defines a mock response pattern
type Response struct {
	Name     string   `yaml:"name"`
	Patterns []string `yaml:"patterns"`
	Messages []string `yaml:"messages"`
	Delay    int      `yaml:"delay_ms"`
	Tokens   int      `yaml:"tokens"`
}

// loadResponses loads mock responses from file or embedded data
func loadResponses() (ResponseSet, error) {
	var responses ResponseSet

	// First try to load from external file (for easy customization)
	customPath := os.Getenv("GUILD_MOCK_RESPONSES")
	if customPath == "" {
		// Default locations
		locations := []string{
			"testdata/mock_responses.yaml",
			".guild/mock_responses.yaml",
			filepath.Join(os.Getenv("HOME"), ".guild/mock_responses.yaml"),
		}

		for _, loc := range locations {
			if _, err := os.Stat(loc); err == nil {
				customPath = loc
				break
			}
		}
	}

	// Load from file if found
	if customPath != "" {
		data, err := os.ReadFile(customPath)
		if err == nil {
			if err := yaml.Unmarshal(data, &responses); err == nil {
				return responses, nil
			}
		}
	}

	// Fall back to embedded responses
	if err := yaml.Unmarshal(embeddedResponses, &responses); err != nil {
		return responses, err
	}

	return responses, nil
}