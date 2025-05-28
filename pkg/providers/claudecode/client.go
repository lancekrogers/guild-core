package claudecode

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

// Latest Claude Code capabilities as of 2025
var SupportedFeatures = map[string]FeatureInfo{
	// Core Claude Code Features
	"code-generation":    {Name: "code-generation", Type: "coding", Description: "Advanced code generation and completion"},
	"code-review":        {Name: "code-review", Type: "coding", Description: "Code analysis and review capabilities"},
	"debugging":          {Name: "debugging", Type: "coding", Description: "Debug code and identify issues"},
	"refactoring":        {Name: "refactoring", Type: "coding", Description: "Code refactoring and optimization"},
	
	// MCP Integration
	"mcp-tools":          {Name: "mcp-tools", Type: "integration", Description: "Model Context Protocol tool integration"},
	"file-processing":    {Name: "file-processing", Type: "integration", Description: "File system operations"},
	"git-integration":    {Name: "git-integration", Type: "integration", Description: "Git repository operations"},
	
	// Advanced Features
	"multi-turn":         {Name: "multi-turn", Type: "conversation", Description: "Multi-turn conversation support"},
	"session-management": {Name: "session-management", Type: "conversation", Description: "Session persistence and resumption"},
	"streaming":          {Name: "streaming", Type: "output", Description: "Real-time streaming responses"},
	"custom-prompts":     {Name: "custom-prompts", Type: "customization", Description: "Custom system prompts"},
}

// FeatureInfo contains information about a Claude Code feature
type FeatureInfo struct {
	Name        string // Feature name
	Type        string // Feature type: coding, integration, conversation, output, customization
	Description string // Feature description
}

// Client implements the LLMClient interface for Claude Code
type Client struct {
	claudeClient *claude.ClaudeClient
	binPath      string
	defaultOpts  *claude.RunOptions
}

// NewClient creates a new Claude Code client
func NewClient(binPath, model string) *Client {
	// Use default path if none specified
	if binPath == "" {
		binPath = "claude" // Assumes claude is in PATH
	}
	
	// Create claude client
	claudeClient := claude.NewClient(binPath)
	
	// Set up default options
	defaultOpts := &claude.RunOptions{
		Format:   claude.TextOutput,
		MaxTurns: 10,
		Verbose:  false,
	}
	
	// Configure model-specific options if specified
	if model != "" {
		// Claude Code doesn't use traditional models, but we can use this for system prompts
		switch model {
		case "coding-focused":
			defaultOpts.SystemPrompt = "You are an expert software engineer focused on writing clean, efficient code."
		case "debugging-focused":
			defaultOpts.SystemPrompt = "You are an expert debugger focused on identifying and fixing code issues."
		case "review-focused":
			defaultOpts.SystemPrompt = "You are an expert code reviewer focused on best practices and quality."
		}
	}

	return &Client{
		claudeClient: claudeClient,
		binPath:      binPath,
		defaultOpts:  defaultOpts,
	}
}

// Complete generates a completion for the given prompt
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	result, err := c.claudeClient.RunPrompt(prompt, c.defaultOpts)
	if err != nil {
		return "", fmt.Errorf("claude code execution failed: %w", err)
	}
	
	return result.Result, nil
}

// CompleteWithOptions generates a completion with custom options
func (c *Client) CompleteWithOptions(ctx context.Context, prompt string, opts *claude.RunOptions) (*claude.ClaudeResult, error) {
	if opts == nil {
		opts = c.defaultOpts
	}
	
	return c.claudeClient.RunPrompt(prompt, opts)
}

// StreamComplete generates a streaming completion
func (c *Client) StreamComplete(ctx context.Context, prompt string) (<-chan string, <-chan error) {
	messageChan, errorChan := c.claudeClient.StreamPrompt(ctx, prompt, c.defaultOpts)
	
	// Convert Message channel to string channel
	stringChan := make(chan string)
	go func() {
		defer close(stringChan)
		for msg := range messageChan {
			if msg.Result != "" {
				stringChan <- msg.Result
			}
		}
	}()
	
	return stringChan, errorChan
}

// ContinueConversation continues an existing conversation
func (c *Client) ContinueConversation(ctx context.Context, sessionID, prompt string) (*claude.ClaudeResult, error) {
	opts := *c.defaultOpts // Copy default options
	opts.ResumeID = sessionID
	opts.Continue = true
	
	return c.claudeClient.RunPrompt(prompt, &opts)
}

// GetBinPath returns the Claude Code binary path
func (c *Client) GetBinPath() string {
	return c.binPath
}

// GetDefaultOptions returns the default run options
func (c *Client) GetDefaultOptions() *claude.RunOptions {
	return c.defaultOpts
}

// SetSystemPrompt sets a custom system prompt
func (c *Client) SetSystemPrompt(prompt string) {
	c.defaultOpts.SystemPrompt = prompt
}

// EnableMCP enables MCP with the specified config path
func (c *Client) EnableMCP(configPath string) {
	c.defaultOpts.MCPConfigPath = configPath
}

// SetAllowedTools sets the allowed tools list
func (c *Client) SetAllowedTools(tools []string) {
	c.defaultOpts.AllowedTools = tools
}

// SetDisallowedTools sets the disallowed tools list
func (c *Client) SetDisallowedTools(tools []string) {
	c.defaultOpts.DisallowedTools = tools
}

// ListSupportedFeatures returns all supported Claude Code features
func ListSupportedFeatures() map[string]FeatureInfo {
	return SupportedFeatures
}

// GetFeaturesByType returns features of a specific type
func GetFeaturesByType(featureType string) map[string]FeatureInfo {
	filtered := make(map[string]FeatureInfo)
	for name, info := range SupportedFeatures {
		if info.Type == featureType {
			filtered[name] = info
		}
	}
	return filtered
}

// GetRecommendedConfiguration returns recommended configuration for a use case
func GetRecommendedConfiguration(useCase string) *claude.RunOptions {
	baseOpts := &claude.RunOptions{
		Format:   claude.TextOutput,
		MaxTurns: 10,
		Verbose:  false,
	}

	switch useCase {
	case "coding":
		baseOpts.SystemPrompt = "You are an expert software engineer. Write clean, efficient, well-documented code. Follow best practices and explain your approach."
		baseOpts.MaxTurns = 20 // Allow longer conversations for complex coding tasks
		
	case "debugging":
		baseOpts.SystemPrompt = "You are an expert debugger. Analyze code carefully, identify issues, and provide clear explanations and fixes."
		baseOpts.Format = claude.JSONOutput // Structured output for debugging info
		
	case "code-review":
		baseOpts.SystemPrompt = "You are an expert code reviewer. Focus on code quality, best practices, security, and maintainability."
		baseOpts.MaxTurns = 15
		
	case "refactoring":
		baseOpts.SystemPrompt = "You are an expert at code refactoring. Improve code structure, readability, and performance while maintaining functionality."
		
	case "architecture":
		baseOpts.SystemPrompt = "You are a software architect. Design scalable, maintainable systems and provide architectural guidance."
		baseOpts.MaxTurns = 25 // Architecture discussions can be lengthy
		
	case "learning":
		baseOpts.SystemPrompt = "You are a patient programming teacher. Explain concepts clearly with examples and help learners understand."
		baseOpts.MaxTurns = 30 // Learning conversations can be long
		
	default:
		baseOpts.SystemPrompt = "You are a helpful AI assistant with expertise in software development."
	}

	return baseOpts
}

// CreateCompletion is a lower-level method to create a completion (for interface compatibility)
func (c *Client) CreateCompletion(ctx context.Context, req *interfaces.CompletionRequest) (*interfaces.CompletionResponse, error) {
	result, err := c.claudeClient.RunPrompt(req.Prompt, c.defaultOpts)
	if err != nil {
		return nil, fmt.Errorf("claude code execution failed: %w", err)
	}

	return &interfaces.CompletionResponse{
		Text:         result.Result,
		TokensUsed:   0, // Claude Code doesn't report token usage
		TokensInput:  0,
		TokensOutput: 0,
		ModelUsed:    "claude-code",
		Metadata:     map[string]string{
			"cost_usd": fmt.Sprintf("%.4f", result.CostUSD),
		},
	}, nil
}

// CreateEmbedding generates an embedding (not supported by Claude Code)
func (c *Client) CreateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	return nil, fmt.Errorf("embedding generation not supported by Claude Code provider")
}

// CreateEmbeddings generates embeddings (not supported by Claude Code)
func (c *Client) CreateEmbeddings(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	return nil, fmt.Errorf("embedding generation not supported by Claude Code provider")
}