package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blockhead-consulting/guild/pkg/memory"
	"github.com/blockhead-consulting/guild/pkg/memory/vector"
	"github.com/blockhead-consulting/guild/pkg/providers"
)

// ChainAwareRAG extends the RAG system with prompt chain awareness
type ChainAwareRAG struct {
	*RAGSystem
	chainManager memory.ChainManager
}

// NewChainAwareRAG creates a new chain-aware RAG system
func NewChainAwareRAG(rag *RAGSystem, chainManager memory.ChainManager) *ChainAwareRAG {
	return &ChainAwareRAG{
		RAGSystem:    rag,
		chainManager: chainManager,
	}
}

// EnhanceChainContext enhances a prompt chain with relevant context
func (r *ChainAwareRAG) EnhanceChainContext(ctx context.Context, chainID string, query string) error {
	// Get the chain
	chain, err := r.chainManager.GetChain(ctx, chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain: %w", err)
	}

	// Retrieve relevant documents
	results, err := r.vectorStore.QueryByText(ctx, query, r.config.MaxRetrievalResults)
	if err != nil {
		return fmt.Errorf("failed to retrieve documents: %w", err)
	}

	if len(results) == 0 {
		// No relevant documents found
		return nil
	}

	// Build context from retrieved documents
	var contextBuilder strings.Builder
	contextBuilder.WriteString("## Relevant Context\n\n")
	
	for i, result := range results {
		doc := result.Document

		contextBuilder.WriteString(fmt.Sprintf("[Document %d] (Relevance: %.2f)\n", i+1, result.Score))
		contextBuilder.WriteString(doc.Content)
		contextBuilder.WriteString("\n\n")

		if r.config.IncludeMetadata && len(doc.Metadata) > 0 {
			contextBuilder.WriteString("Source:\n")
			for k, v := range doc.Metadata {
				if k == "source" || k == "title" || k == "url" {
					contextBuilder.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
				}
			}
			contextBuilder.WriteString("\n")
		}
	}

	// Add retrieved context to the chain
	ragMessage := memory.Message{
		Role:      "system",
		Content:   contextBuilder.String(),
		Timestamp: time.Now().UTC(),
	}

	return r.chainManager.AddMessage(ctx, chainID, ragMessage)
}

// ProcessWithRAG enhances a prompt with RAG and gets a completion
func (r *ChainAwareRAG) ProcessWithRAG(ctx context.Context, chainID, query string) (string, error) {
	// Enhance the chain with relevant context
	if err := r.EnhanceChainContext(ctx, chainID, query); err != nil {
		return "", fmt.Errorf("failed to enhance chain context: %w", err)
	}

	// Get the updated chain with context
	messages, err := r.chainManager.BuildContext(ctx, "", "", 0)
	if err != nil {
		return "", fmt.Errorf("failed to build context: %w", err)
	}

	// Build prompt from messages
	var promptBuilder strings.Builder
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			promptBuilder.WriteString("System: " + msg.Content + "\n\n")
		case "user":
			promptBuilder.WriteString("User: " + msg.Content + "\n\n")
		case "assistant":
			promptBuilder.WriteString("Assistant: " + msg.Content + "\n\n")
		case "tool":
			promptBuilder.WriteString("Tool " + msg.Name + ": " + msg.Content + "\n\n")
		}
	}

	promptBuilder.WriteString("User: " + query + "\n\n")
	promptBuilder.WriteString("Assistant: ")

	// Generate answer
	req := &providers.CompletionRequest{
		Prompt:      promptBuilder.String(),
		MaxTokens:   r.config.DefaultMaxTokens,
		Temperature: r.config.DefaultTemperature,
	}

	if r.config.DefaultModel != "" {
		req.Options = map[string]string{
			"model": r.config.DefaultModel,
		}
	}

	resp, err := r.llmClient.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}

	// Add the answer to the chain
	answerMessage := memory.Message{
		Role:      "assistant",
		Content:   resp.Text,
		Timestamp: time.Now().UTC(),
		TokenUsage: resp.TokensUsed,
	}

	if err := r.chainManager.AddMessage(ctx, chainID, answerMessage); err != nil {
		return "", fmt.Errorf("failed to add answer to chain: %w", err)
	}

	return resp.Text, nil
}

// CreateVectorStoreFromChain creates a vector store from a prompt chain
func CreateVectorStoreFromChain(ctx context.Context, chain *memory.PromptChain, embedder providers.LLMClient, embedModel string, vectorConfig *vector.StoreConfig) error {
	if chain == nil {
		return fmt.Errorf("chain cannot be nil")
	}

	if len(chain.Messages) == 0 {
		return fmt.Errorf("chain has no messages")
	}

	if vectorConfig == nil {
		return fmt.Errorf("vector store config cannot be nil")
	}

	// Create vector store
	vectorStore, err := vector.NewVectorStore(ctx, vectorConfig)
	if err != nil {
		return fmt.Errorf("failed to create vector store: %w", err)
	}

	// Extract content from messages
	docs := []*vector.Document{}
	for i, msg := range chain.Messages {
		if msg.Role == "system" || msg.Role == "assistant" {
			// Only index system and assistant messages
			doc := &vector.Document{
				ID:      fmt.Sprintf("%s-%d", chain.ID, i),
				Content: msg.Content,
				Metadata: map[string]string{
					"chain_id":  chain.ID,
					"role":      msg.Role,
					"timestamp": msg.Timestamp.Format(time.RFC3339),
				},
			}
			docs = append(docs, doc)
		}
	}

	// Store documents
	if err := vectorStore.StoreMany(ctx, docs); err != nil {
		return fmt.Errorf("failed to store documents: %w", err)
	}

	return nil
}