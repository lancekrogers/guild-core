package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blockhead-consulting/guild/pkg/agent"
	"github.com/blockhead-consulting/guild/pkg/kanban"
	"github.com/blockhead-consulting/guild/pkg/memory"
	"github.com/blockhead-consulting/guild/pkg/objective"
	"github.com/blockhead-consulting/guild/pkg/providers"
	"github.com/blockhead-consulting/guild/tools"
)

// AgentWrapper adds RAG capabilities to a GuildArtisan agent
type AgentWrapper struct {
	agent     agent.GuildArtisan
	retriever *Retriever
	config    RetrievalConfig
}

// NewAgentWrapper creates a new RAG agent wrapper
func NewAgentWrapper(agent agent.GuildArtisan, retriever *Retriever, config RetrievalConfig) *AgentWrapper {
	// Use default config if none provided
	if config.MaxResults == 0 {
		config = DefaultRetrievalConfig()
	}

	return &AgentWrapper{
		agent:     agent,
		retriever: retriever,
		config:    config,
	}
}

// Implement the GuildArtisan interface by delegating calls to the wrapped agent

// ID returns the agent's unique identifier
func (w *AgentWrapper) ID() string {
	return w.agent.ID()
}

// Name returns the agent's human-readable name
func (w *AgentWrapper) Name() string {
	return w.agent.Name()
}

// Type returns the agent's type
func (w *AgentWrapper) Type() string {
	return w.agent.Type()
}

// Status returns the agent's current status
func (w *AgentWrapper) Status() agent.AgentStatus {
	return w.agent.Status()
}

// CommissionWork assigns a task to the artisan
func (w *AgentWrapper) CommissionWork(ctx context.Context, task *kanban.Task) error {
	return w.agent.CommissionWork(ctx, task)
}

// CraftSolution runs the artisan's execution cycle
func (w *AgentWrapper) CraftSolution(ctx context.Context) error {
	// Get the current task and memory manager
	task := w.agent.GetState().CurrentTask
	if task == "" {
		return fmt.Errorf("no task assigned")
	}

	// Get the memory manager
	memoryManager := w.agent.GetMemoryManager()

	// Get the most recent memory chain
	chains, err := memoryManager.GetChainsByTask(ctx, task)
	if err != nil || len(chains) == 0 {
		// If there's an error or no chains, just delegate to the original agent
		return w.agent.CraftSolution(ctx)
	}

	// Get the latest chain
	chainID := chains[0].ID

	// Get the system message (prompt) from the latest chain
	var systemPrompt string
	for _, msg := range chains[0].Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			break
		}
	}

	// If no system prompt found, just delegate to the original agent
	if systemPrompt == "" {
		return w.agent.CraftSolution(ctx)
	}

	// Extract query from the task
	query := w.extractQueryFromTask(task)

	// Enhance the prompt with RAG context
	enhancedPrompt, err := w.retriever.EnhancePrompt(ctx, systemPrompt, query, w.config)
	if err != nil {
		// If there's an error enhancing the prompt, just use the original one
		return w.agent.CraftSolution(ctx)
	}

	// Replace the system message with the enhanced one
	newMessage := memory.Message{
		Role:      "system",
		Content:   enhancedPrompt,
		Timestamp: time.Now().UTC(),
	}

	// Remove the original system message
	var newMessages []memory.Message
	for i, msg := range chains[0].Messages {
		if i == 0 && msg.Role == "system" {
			// Skip the first system message
			continue
		}
		newMessages = append(newMessages, msg)
	}

	// Create a new chain with the enhanced prompt
	newChainID, err := memoryManager.CreateChain(ctx, w.agent.ID(), task)
	if err != nil {
		return fmt.Errorf("failed to create new memory chain: %w", err)
	}

	// Add the enhanced system message
	err = memoryManager.AddMessage(ctx, newChainID, newMessage)
	if err != nil {
		return fmt.Errorf("failed to add enhanced system message: %w", err)
	}

	// Copy the other messages from the original chain
	for _, msg := range newMessages {
		err = memoryManager.AddMessage(ctx, newChainID, msg)
		if err != nil {
			return fmt.Errorf("failed to copy message to new chain: %w", err)
		}
	}

	// Update the agent's state to use the new chain
	state := w.agent.GetState()
	if len(state.Memory) > 0 {
		state.Memory[len(state.Memory)-1] = newChainID
	} else {
		state.Memory = append(state.Memory, newChainID)
	}

	// Now delegate to the original agent to continue execution
	return w.agent.CraftSolution(ctx)
}

// extractQueryFromTask extracts a search query from a task
func (w *AgentWrapper) extractQueryFromTask(taskID string) string {
	// Get task details from memory
	task, ok := w.agent.GetState().CurrentTask, true
	if !ok || task == "" {
		return ""
	}

	// Combine task title and description
	var queryParts []string

	// Get the task object
	memoryManager := w.agent.GetMemoryManager()
	ctx := context.Background()
	chains, err := memoryManager.GetChainsByTask(ctx, taskID)
	if err != nil || len(chains) == 0 {
		return ""
	}

	// Extract title and description from messages
	for _, chain := range chains {
		for _, msg := range chain.Messages {
			if msg.Role == "system" {
				// Extract title from prompt
				if titleStart := strings.Index(msg.Content, "Title:"); titleStart != -1 {
					titleEnd := strings.Index(msg.Content[titleStart:], "\n")
					if titleEnd != -1 {
						title := msg.Content[titleStart+7 : titleStart+titleEnd]
						queryParts = append(queryParts, strings.TrimSpace(title))
					}
				}

				// Extract description from prompt
				if descStart := strings.Index(msg.Content, "Description:"); descStart != -1 {
					descEnd := strings.Index(msg.Content[descStart:], "\n\n")
					if descEnd != -1 {
						desc := msg.Content[descStart+12 : descStart+descEnd]
						queryParts = append(queryParts, strings.TrimSpace(desc))
					} else {
						// If no double newline, take until end of message
						desc := msg.Content[descStart+12:]
						queryParts = append(queryParts, strings.TrimSpace(desc))
					}
				}
			}
		}
	}

	// Join the parts to form a query
	return strings.Join(queryParts, " ")
}

// Stop gracefully stops the agent's execution
func (w *AgentWrapper) Stop(ctx context.Context) error {
	return w.agent.Stop(ctx)
}

// CleanSlate resets the artisan to its initial state
func (w *AgentWrapper) CleanSlate(ctx context.Context) error {
	return w.agent.CleanSlate(ctx)
}

// SaveState saves the artisan's current state
func (w *AgentWrapper) SaveState(ctx context.Context) error {
	return w.agent.SaveState(ctx)
}

// GetAvailableTools returns the list of tools available to the agent
func (w *AgentWrapper) GetAvailableTools() []tools.Tool {
	return w.agent.GetAvailableTools()
}

// GetConfig returns the agent's configuration
func (w *AgentWrapper) GetConfig() *agent.AgentConfig {
	return w.agent.GetConfig()
}

// GetState returns the agent's current state
func (w *AgentWrapper) GetState() *agent.AgentState {
	return w.agent.GetState()
}

// GetMemoryManager returns the agent's memory manager
func (w *AgentWrapper) GetMemoryManager() memory.ChainManager {
	return w.agent.GetMemoryManager()
}

// SetCostBudget sets the budget for a specific cost type
func (w *AgentWrapper) SetCostBudget(costType agent.CostType, amount float64) {
	w.agent.SetCostBudget(costType, amount)
}

// GetCostReport returns a report of all costs incurred by the agent
func (w *AgentWrapper) GetCostReport() map[string]interface{} {
	return w.agent.GetCostReport()
}

// RagAgentOptions contains options for creating a RAG agent
type RagAgentOptions struct {
	// MaxResults is the maximum number of results to return
	MaxResults int

	// MinScore is the minimum similarity score required (0-1)
	MinScore float32

	// IncludeCorpus determines whether to include documents from the corpus
	IncludeCorpus bool

	// ChunkSize is the size of chunks to break documents into
	ChunkSize int

	// ChunkOverlap is the overlap between chunks
	ChunkOverlap int

	// ChunkStrategy is the strategy for chunking documents
	ChunkStrategy ChunkStrategy
}

// DefaultRagAgentOptions returns default options for RAG agents
func DefaultRagAgentOptions() RagAgentOptions {
	return RagAgentOptions{
		MaxResults:    5,
		MinScore:      0.7,
		IncludeCorpus: true,
		ChunkSize:     1000,
		ChunkOverlap:  100,
		ChunkStrategy: ChunkByParagraph,
	}
}

// NewRagAgent creates a new agent with RAG capabilities
func NewRagAgent(
	config *agent.AgentConfig,
	llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry *tools.ToolRegistry,
	objectiveMgr *objective.Manager,
	retriever *Retriever,
	options RagAgentOptions,
) agent.GuildArtisan {
	// Create a base craftsman agent
	baseAgent := agent.NewCraftsman(config, llmClient, memoryManager, toolRegistry, objectiveMgr)

	// Map options to retrieval config
	retrievalConfig := RetrievalConfig{
		MaxResults:    options.MaxResults,
		MinScore:      options.MinScore,
		IncludeCorpus: options.IncludeCorpus,
		ChunkSize:     options.ChunkSize,
		ChunkOverlap:  options.ChunkOverlap,
		ChunkStrategy: options.ChunkStrategy,
	}

	// Wrap in RAG capabilities
	return NewAgentWrapper(baseAgent, retriever, retrievalConfig)
}