// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package global

// GlobalConfig represents the global Guild configuration
type GlobalConfig struct {
	// Provider settings
	Providers ProvidersConfig `yaml:"providers"`

	// Tool settings
	Tools ToolsConfig `yaml:"tools"`

	// Cache settings
	Cache CacheConfig `yaml:"cache"`

	// Logging settings
	Logging LoggingConfig `yaml:"logging"`

	// UI settings
	UI UIConfig `yaml:"ui"`

	// Security settings
	Security GlobalSecurityConfig `yaml:"security"`

	// LSP server configurations
	LSPServers map[string]LSPServerConfig `yaml:"lsp_servers,omitempty"`

	// Global corpus settings
	GlobalCorpus CorpusConfig `yaml:"global_corpus,omitempty"`
}

// ProvidersConfig contains default provider settings
type ProvidersConfig struct {
	Default  string   `yaml:"default"`
	Fallback []string `yaml:"fallback"`
}

// ToolsConfig contains global tool settings
type ToolsConfig struct {
	Enabled  []string          `yaml:"enabled"`
	Disabled []string          `yaml:"disabled"`
	Custom   map[string]string `yaml:"custom,omitempty"` // Custom tool paths
}

// CacheConfig contains cache settings
type CacheConfig struct {
	Embeddings EmbeddingsCacheConfig `yaml:"embeddings"`
}

// EmbeddingsCacheConfig contains embeddings cache settings
type EmbeddingsCacheConfig struct {
	MaxSizeGB int `yaml:"max_size_gb"`
	TTLDays   int `yaml:"ttl_days"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level     string `yaml:"level"`
	MaxSizeMB int    `yaml:"max_size_mb"`
	MaxFiles  int    `yaml:"max_files"`
}

// UIConfig contains UI settings
type UIConfig struct {
	VimMode bool   `yaml:"vim_mode"`
	Theme   string `yaml:"theme"`
}

// GlobalSecurityConfig contains security settings
type GlobalSecurityConfig struct {
	APIKeys APIKeysConfig `yaml:"api_keys"`
}

// APIKeysConfig specifies where API keys come from
type APIKeysConfig struct {
	Source string `yaml:"source"` // "environment" or "keychain"
}

// LSPServerConfig represents configuration for an LSP server
type LSPServerConfig struct {
	Name         string   `yaml:"name"`
	Command      string   `yaml:"command"`
	Args         []string `yaml:"args,omitempty"`
	Languages    []string `yaml:"languages"`
	InstallCmd   string   `yaml:"install_cmd,omitempty"`
	CheckCmd     string   `yaml:"check_cmd,omitempty"`
	RootPatterns []string `yaml:"root_patterns,omitempty"`
}

// CorpusConfig represents global corpus settings
type CorpusConfig struct {
	SharedKnowledge []string `yaml:"shared_knowledge"` // Paths to shared knowledge bases
	MaxSizeGB       int      `yaml:"max_size_gb"`
}

// TemplateConfig represents a project template
type TemplateConfig struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Agents      []TemplateAgent `yaml:"agents"`
	Tools       []string        `yaml:"tools"`
	Corpus      CorpusTemplate  `yaml:"corpus"`
	Commissions []string        `yaml:"commissions"`
}

// TemplateAgent represents an agent in a template
type TemplateAgent struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`
	Specialties []string `yaml:"specialties"`
}

// CorpusTemplate represents corpus configuration in a template
type CorpusTemplate struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}
