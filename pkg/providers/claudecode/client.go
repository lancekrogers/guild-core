// Package claudecode provides a client for Claude Code CLI, which is included with Claude Max subscription.
//
// Claude Code is a powerful CLI tool that comes bundled with the Claude Max plan, offering:
//   - Higher usage limits compared to using the API directly
//   - Access to all Claude models including Opus 4 and Sonnet 4
//   - Built-in MCP (Model Context Protocol) support
//   - Local session management and conversation history
//
// To get started with Claude Max:
//   1. Sign up for Claude Max using this affiliate link: https://t.co/54ylwq0OPh
//   2. After signing up, configure your Max plan via the Claude console
//   3. Use /logout and /login commands in Claude Code to authenticate
//   4. Enjoy higher usage limits and all premium features
//
// The Claude Max plan is ideal for developers who need:
//   - Extended conversation limits beyond API quotas
//   - Access to the latest models without separate API billing
//   - Integrated development tools like Claude Code
//   - Priority access to new features and models
package claudecode

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

// Claude 4 model constants (Released May 2025)
const (
	// Claude 4 models - hybrid models with near-instant and extended thinking modes
	ClaudeOpus4  = "claude-opus-4"   // $15/$75 per M tokens, 32K max output
	ClaudeSonnet4 = "claude-sonnet-4" // $3/$15 per M tokens, 64K max output
	
	// Claude 3.7 models
	ClaudeSonnet37 = "claude-3.7-sonnet"
	
	// Claude 3.5 models
	ClaudeSonnet35 = "claude-3-5-sonnet-20241022"
	ClaudeHaiku35  = "claude-3-5-haiku-20241022"
	
	// Claude 3 models
	ClaudeOpus3 = "claude-3-opus-20240229"
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

// NewClient creates a new Claude Code client.
//
// Claude Code CLI is included with the Claude Max subscription plan.
// To use this client, you need:
//   1. An active Claude Max subscription (sign up at: https://t.co/54ylwq0OPh)
//   2. Claude Code CLI installed and authenticated
//
// Authentication steps:
//   - Run `claude /logout` to clear any existing sessions
//   - Run `claude /login` to authenticate with your Claude Max account
//   - The CLI will open a browser for authentication
//
// Benefits of Claude Max over direct API usage:
//   - Higher usage limits and priority access
//   - No per-token billing - unlimited usage within Max plan limits
//   - Access to all models including Claude 4 Opus and Sonnet
//   - Integrated MCP tools and file system access
//   - Persistent conversation history
//
// Parameters:
//   - binPath: Path to claude binary (defaults to "claude" in PATH)
//   - model: Model to use (e.g., "claude-opus-4", "claude-sonnet-4")
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
	
	// Configure model if specified
	if model != "" {
		// Claude Code now supports model selection via --model flag
		defaultOpts.Model = model
		
		// Also configure model-specific system prompts based on common model names
		switch model {
		// Claude 4 models (Released May 2025)
		case "claude-opus-4", "opus-4", "claude-4-opus":
			defaultOpts.SystemPrompt = "You are Claude Opus 4, Anthropic's most powerful model and the best coding model in the world. You have exceptional reasoning and can work autonomously for extended periods."
		case "claude-sonnet-4", "sonnet-4", "claude-4-sonnet":
			defaultOpts.SystemPrompt = "You are Claude Sonnet 4, delivering an optimal mix of capability and practicality with improvements in coding and math."
		// Claude 3.7 models
		case "claude-3.7-sonnet", "claude-3-7-sonnet":
			defaultOpts.SystemPrompt = "You are Claude 3.7 Sonnet, a powerful AI assistant with strong reasoning capabilities."
		// Claude 3.5 models
		case "claude-3-5-sonnet-20241022", "claude-3-sonnet", "sonnet":
			defaultOpts.SystemPrompt = "You are Claude, an AI assistant created by Anthropic. You are helpful, harmless, and honest."
		case "claude-3-5-haiku-20241022", "claude-3-haiku", "haiku":
			defaultOpts.SystemPrompt = "You are Claude, an efficient AI assistant. Be concise and direct in your responses."
		// Claude 3 models
		case "claude-3-opus-20240229", "claude-3-opus":
			defaultOpts.SystemPrompt = "You are Claude, an advanced AI assistant with deep reasoning capabilities."
		// Task-focused modes
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

// Complete generates a completion for the given prompt.
//
// This method uses Claude Code CLI which is included with Claude Max subscription.
// Usage limits with Claude Max are significantly higher than direct API usage:
//   - No per-token charges - covered by your Max subscription
//   - Higher daily/monthly limits compared to API tier
//   - Priority processing during high-demand periods
//
// If you hit usage limits, you can:
//   1. Wait for the limit reset (usually daily)
//   2. Upgrade your Claude Max plan
//   3. Use the API directly as a fallback (separate billing)
//
// Get Claude Max: https://t.co/54ylwq0OPh
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	result, err := c.claudeClient.RunPrompt(prompt, c.defaultOpts)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeProviderAPI, "claude code execution failed").
			WithComponent("providers").
			WithOperation("Complete").
			WithDetails("provider", "claudecode")
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

// GetModel returns the currently configured model
func (c *Client) GetModel() string {
	return c.defaultOpts.Model
}

// SetModel sets the model to use for completions
func (c *Client) SetModel(model string) {
	c.defaultOpts.Model = model
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
		return nil, gerror.Wrap(err, gerror.ErrCodeProviderAPI, "claude code execution failed").
			WithComponent("providers").
			WithOperation("CreateCompletion").
			WithDetails("provider", "claudecode")
	}

	// Determine the model used
	modelUsed := "claude-code"
	if c.defaultOpts.Model != "" {
		modelUsed = c.defaultOpts.Model
	}
	
	return &interfaces.CompletionResponse{
		Text:         result.Result,
		TokensUsed:   0, // Claude Code doesn't report token usage
		TokensInput:  0,
		TokensOutput: 0,
		ModelUsed:    modelUsed,
		Metadata:     map[string]string{
			"cost_usd": fmt.Sprintf("%.4f", result.CostUSD),
			"model": modelUsed,
		},
	}, nil
}

// CreateEmbedding generates an embedding (not supported by Claude Code)
func (c *Client) CreateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	return nil, gerror.New(gerror.ErrCodeProvider, "embedding generation not supported by Claude Code provider", nil).
		WithComponent("providers").
		WithOperation("CreateEmbedding").
		WithDetails("provider", "claudecode").
		WithDetails("capability", "embeddings")
}

// CreateEmbeddings generates embeddings (not supported by Claude Code)
func (c *Client) CreateEmbeddings(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	return nil, gerror.New(gerror.ErrCodeProvider, "embedding generation not supported by Claude Code provider", nil).
		WithComponent("providers").
		WithOperation("CreateEmbeddings").
		WithDetails("provider", "claudecode").
		WithDetails("capability", "embeddings")
}