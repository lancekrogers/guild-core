package corpus

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"gopkg.in/yaml.v3"
)

// ConfigKeys defines the environment variable keys for corpus configuration
const (
	EnvCorpusPath     = "GUILD_CORPUS_PATH"
	EnvActivitiesPath = "GUILD_CORPUS_ACTIVITIES_PATH"
	EnvMaxSizeBytes   = "GUILD_CORPUS_MAX_SIZE_BYTES"
	EnvDefaultTags    = "GUILD_CORPUS_DEFAULT_TAGS"
)

// ConfigKey is a key for the corpus configuration
type ConfigKey string

// Configuration keys
const (
	KeyCorpusPath     ConfigKey = "corpusPath"
	KeyActivitiesPath ConfigKey = "activitiesPath"
	KeyMaxSizeBytes   ConfigKey = "maxSizeBytes"
	KeyDefaultTags    ConfigKey = "defaultTags"
)

// DefaultConfig returns the default configuration for the corpus
func DefaultConfig() Config {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	corpusPath := filepath.Join(cwd, "corpus")
	maxSizeBytes := int64(10 * 1024 * 1024) // 10MB default size limit
	return Config{
		CorpusPath:      corpusPath,
		ActivitiesPath:  filepath.Join(cwd, "corpus", ".activities"),
		MaxSizeBytes:    maxSizeBytes,
		DefaultTags:     []string{},
		DefaultCategory: "general",
		Location:        corpusPath, // Set Location as an alias to CorpusPath
		MaxSizeMB:       maxSizeBytes / 1024 / 1024, // Convert bytes to MB
	}
}

// LoadConfig loads the corpus configuration from the config loader
func LoadConfig(loader *config.ConfigLoader) (Config, error) {
	// Start with default configuration
	cfg := DefaultConfig()

	// Try to load from config file
	if loader != nil {
		configData, err := loader.LoadConfig("corpus.yml")
		if err == nil {
			// Override with configuration from file
			if corpusPath, ok := configData[string(KeyCorpusPath)].(string); ok && corpusPath != "" {
				cfg.CorpusPath = corpusPath
			}
			if activitiesPath, ok := configData[string(KeyActivitiesPath)].(string); ok && activitiesPath != "" {
				cfg.ActivitiesPath = activitiesPath
			}
			if maxSizeBytes, ok := configData[string(KeyMaxSizeBytes)].(int64); ok && maxSizeBytes > 0 {
				cfg.MaxSizeBytes = maxSizeBytes
			}
			if defaultTags, ok := configData[string(KeyDefaultTags)].([]string); ok {
				cfg.DefaultTags = defaultTags
			}
		}
	}

	// Override with environment variables
	envVars := config.LoadFromEnv("GUILD_CORPUS")
	
	if path, ok := envVars[EnvCorpusPath]; ok && path != "" {
		cfg.CorpusPath = path
	}
	
	if path, ok := envVars[EnvActivitiesPath]; ok && path != "" {
		cfg.ActivitiesPath = path
	}
	
	if sizeStr, ok := envVars[EnvMaxSizeBytes]; ok && sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && size > 0 {
			cfg.MaxSizeBytes = size
		}
	}
	
	if tagsStr, ok := envVars[EnvDefaultTags]; ok && tagsStr != "" {
		// Parse comma-separated tags
		cfg.DefaultTags = filepath.SplitList(tagsStr)
	}

	// Ensure the corpus directory exists
	if err := os.MkdirAll(cfg.CorpusPath, 0755); err != nil {
		return cfg, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("new_config").WithOperation("error creating corpus directory")
	}

	// Ensure the activities directory exists
	if err := os.MkdirAll(cfg.ActivitiesPath, 0755); err != nil {
		return cfg, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("new_config").WithOperation("error creating activities directory")
	}

	// Set backward compatibility fields
	cfg.Location = cfg.CorpusPath
	cfg.MaxSizeMB = cfg.MaxSizeBytes / 1024 / 1024

	// Set default category if not specified
	if cfg.DefaultCategory == "" {
		cfg.DefaultCategory = "general"
	}

	return cfg, nil
}

// SaveConfig saves the corpus configuration to a YAML file
func SaveConfig(cfg Config, path string) error {
	// Create the configuration directory if it doesn't exist
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("save_config").WithOperation("error creating config directory")
	}

	// Marshal the configuration to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("save_config").WithOperation("error marshaling config")
	}

	// Write the configuration to the file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("save_config").WithOperation("error writing config file")
	}

	return nil
}

// LoadConfigFromFile loads the corpus configuration from a YAML file
func LoadConfigFromFile(path string) (Config, error) {
	// Start with default configuration
	cfg := DefaultConfig()

	// Check if the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return default configuration if the file doesn't exist
		return cfg, nil
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("load_config").WithOperation("error reading config file")
	}

	// Unmarshal the YAML
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("load_config").WithOperation("error unmarshaling config")
	}

	// Set backward compatibility fields
	cfg.Location = cfg.CorpusPath
	cfg.MaxSizeMB = cfg.MaxSizeBytes / 1024 / 1024

	// Set default category if not specified
	if cfg.DefaultCategory == "" {
		cfg.DefaultCategory = "general"
	}

	return cfg, nil
}