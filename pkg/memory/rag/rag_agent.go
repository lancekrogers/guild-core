package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/tools"
)

// AgentWrapper adds RAG capabilities to a GuildArtisan agent
type AgentWrapper struct {
	agent     agent.GuildArtisan
	retriever *Retriever
	config    Config
}

// NewAgentWrapper creates a new RAG agent wrapper
func NewAgentWrapper(agent agent.GuildArtisan, retriever *Retriever, config Config) *AgentWrapper {
	return &AgentWrapper{
		agent:     agent,
		retriever: retriever,
		config:    config,
	}
}

// Implement the Agent interface

// Execute runs a task with RAG enhancement
func (w *AgentWrapper) Execute(ctx context.Context, request string) (string, error) {
	// First, enhance the request with relevant context
	enhancedRequest, err := w.enhanceRequestWithRAG(ctx, request)
	if err != nil {
		// If there's an error enhancing the request, just use the original
		return w.agent.Execute(ctx, request)
	}
	
	// Execute with enhanced request
	return w.agent.Execute(ctx, enhancedRequest)
}

// GetID returns the agent's ID
func (w *AgentWrapper) GetID() string {
	return w.agent.GetID()
}

// GetName returns the agent's name
func (w *AgentWrapper) GetName() string {
	return w.agent.GetName()
}

// Implement the GuildArtisan interface

// GetToolRegistry returns the tool registry
func (w *AgentWrapper) GetToolRegistry() *tools.ToolRegistry {
	return w.agent.GetToolRegistry()
}

// GetObjectiveManager returns the objective manager
func (w *AgentWrapper) GetObjectiveManager() *objective.Manager {
	return w.agent.GetObjectiveManager()
}

// GetLLMClient returns the LLM client
func (w *AgentWrapper) GetLLMClient() providers.LLMClient {
	return w.agent.GetLLMClient()
}

// GetMemoryManager returns the memory manager
func (w *AgentWrapper) GetMemoryManager() memory.ChainManager {
	return w.agent.GetMemoryManager()
}

// enhanceRequestWithRAG enhances a request with relevant context from the RAG system
func (w *AgentWrapper) enhanceRequestWithRAG(ctx context.Context, request string) (string, error) {
	// If no retriever, return the original request
	if w.retriever == nil {
		return request, nil
	}
	
	// Define retrieval configuration
	retrievalConfig := RetrievalConfig{
		Query:           request,
		MaxResults:      w.config.MaxResults,
		MinScore:        0.7, // Default minimum score
		IncludeMetadata: true,
		UseCorpus:       true,
	}
	
	// Retrieve relevant context
	results, err := w.retriever.RetrieveContext(ctx, request, retrievalConfig)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve context: %w", err)
	}
	
	// If no results found, return the original request
	if len(results.Results) == 0 {
		return request, nil
	}
	
	// Build enhanced request with context
	var enhancedRequest strings.Builder
	enhancedRequest.WriteString("Based on the following relevant context:\n\n")
	
	for i, result := range results.Results {
		enhancedRequest.WriteString(fmt.Sprintf("Context %d (Score: %.3f, Source: %s):\n", i+1, result.Score, result.Source))
		enhancedRequest.WriteString(result.Content)
		enhancedRequest.WriteString("\n\n")
	}
	
	enhancedRequest.WriteString("Now, please respond to this request:\n")
	enhancedRequest.WriteString(request)
	
	return enhancedRequest.String(), nil
}

// EnhancePrompt enhances a prompt with relevant context from the RAG system
// This is a public method that can be used for more fine-grained control
func (w *AgentWrapper) EnhancePrompt(ctx context.Context, prompt, query string, config RetrievalConfig) (string, error) {
	// Retrieve relevant context
	results, err := w.retriever.RetrieveContext(ctx, query, config)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve context: %w", err)
	}
	
	// If no results found, return the original prompt
	if len(results.Results) == 0 {
		return prompt, nil
	}
	
	// Build enhanced prompt with context
	var enhancedPrompt strings.Builder
	enhancedPrompt.WriteString(prompt)
	enhancedPrompt.WriteString("\n\n## Relevant Context\n\n")
	
	for i, result := range results.Results {
		enhancedPrompt.WriteString(fmt.Sprintf("### Context %d (Score: %.3f)\n", i+1, result.Score))
		if result.Source != "" {
			enhancedPrompt.WriteString(fmt.Sprintf("Source: %s\n", result.Source))
		}
		enhancedPrompt.WriteString(result.Content)
		enhancedPrompt.WriteString("\n\n")
	}
	
	return enhancedPrompt.String(), nil
}