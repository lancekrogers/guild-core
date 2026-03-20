// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lancekrogers/guild-core/pkg/corpus"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/memory/rag"
	"github.com/lancekrogers/guild-core/pkg/providers/interfaces"
)

// CorpusAgent is a specialized agent that provides an intelligent interface
// between humans and the RAG system. It queries the RAG system to find relevant
// information and generates human-readable documents on demand.
//
// The Corpus Agent implements the on-demand generation model where:
// - All information (human and agent-generated) is stored in RAG
// - Humans interact with the Corpus Agent to request specific information
// - The agent synthesizes and formats information for human consumption
// - Humans can optionally save valuable generated documents to the corpus
type CorpusAgent struct {
	// Base agent fields
	ID   string
	Name string

	// RAG system for querying knowledge
	ragSystem *rag.Retriever

	// LLM for generating responses
	llmProvider interfaces.AIProvider

	// Corpus configuration for saving documents
	corpusConfig corpus.Config

	// Conversation context for iterative refinement
	conversationHistory []Message
}

// Message represents a conversation message
type Message struct {
	Role      string // "user" or "assistant"
	Content   string
	Timestamp time.Time
}

// NewCorpusAgent creates a new Corpus Agent
func NewCorpusAgent(ragSystem *rag.Retriever, llmProvider interfaces.AIProvider, corpusConfig corpus.Config) *CorpusAgent {
	return &CorpusAgent{
		ID:                  "corpus-agent-001",
		Name:                "Corpus Knowledge Navigator",
		ragSystem:           ragSystem,
		llmProvider:         llmProvider,
		corpusConfig:        corpusConfig,
		conversationHistory: make([]Message, 0),
	}
}

// Execute handles user queries and generates documents from the RAG system
func (a *CorpusAgent) Execute(ctx context.Context, request string) (string, error) {
	// Add user message to history
	a.conversationHistory = append(a.conversationHistory, Message{
		Role:      "user",
		Content:   request,
		Timestamp: time.Now(),
	})

	// Query RAG system for relevant context
	retrievalConfig := rag.RetrievalConfig{
		MaxResults:      10,
		MinScore:        0.5,
		UseCorpus:       true,
		IncludeMetadata: true,
	}

	results, err := a.ragSystem.RetrieveContext(ctx, request, retrievalConfig)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to retrieve context from RAG system").
			WithComponent("corpus_agent").
			WithOperation("Execute").
			WithDetails("request_length", len(request)).
			WithDetails("max_results", retrievalConfig.MaxResults)
	}

	// Generate response using LLM
	response, err := a.generateResponse(ctx, request, results)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeProvider, "failed to generate response using LLM").
			WithComponent("corpus_agent").
			WithOperation("Execute").
			WithDetails("request_length", len(request)).
			WithDetails("results_count", len(results.Results))
	}

	// Add assistant message to history
	a.conversationHistory = append(a.conversationHistory, Message{
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
	})

	// Keep conversation history limited
	if len(a.conversationHistory) > 20 {
		a.conversationHistory = a.conversationHistory[len(a.conversationHistory)-20:]
	}

	return response, nil
}

// GenerateDocument creates a formatted document from a query
func (a *CorpusAgent) GenerateDocument(ctx context.Context, query string, title string) (*corpus.CorpusDoc, error) {
	// Get response from RAG
	response, err := a.Execute(ctx, query)
	if err != nil {
		return nil, err
	}

	// Create corpus document
	doc := &corpus.CorpusDoc{
		Title:     title,
		Body:      response,
		Tags:      a.extractTags(query, response),
		Source:    "corpus-agent",
		GuildID:   "corpus",
		AgentID:   a.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return doc, nil
}

// SaveGeneratedDocument saves a generated document to the corpus
func (a *CorpusAgent) SaveGeneratedDocument(ctx context.Context, doc *corpus.CorpusDoc) error {
	// Validate document
	if doc.Title == "" {
		return gerror.New(gerror.ErrCodeValidation, "document title is required", nil).
			WithComponent("corpus_agent").
			WithOperation("SaveGeneratedDocument")
	}

	if doc.Body == "" {
		return gerror.New(gerror.ErrCodeValidation, "document body is required", nil).
			WithComponent("corpus_agent").
			WithOperation("SaveGeneratedDocument")
	}

	// Save to corpus
	return corpus.Save(ctx, doc, a.corpusConfig)
}

// generateResponse creates a response using the LLM and retrieved context
func (a *CorpusAgent) generateResponse(ctx context.Context, query string, results *rag.SearchResults) (string, error) {
	// Build the prompt with context
	prompt := a.buildPrompt(query, results)

	// Create chat request
	messages := []interfaces.ChatMessage{
		{
			Role:    "system",
			Content: a.getSystemPrompt(),
		},
	}

	// Add limited conversation history for context
	historyStart := len(a.conversationHistory) - 4
	if historyStart < 0 {
		historyStart = 0
	}
	for i := historyStart; i < len(a.conversationHistory); i++ {
		msg := a.conversationHistory[i]
		messages = append(messages, interfaces.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add current prompt
	messages = append(messages, interfaces.ChatMessage{
		Role:    "user",
		Content: prompt,
	})

	// Get available models
	capabilities := a.llmProvider.GetCapabilities()
	model := ""
	if len(capabilities.Models) > 0 {
		model = capabilities.Models[0].ID
	}

	// Create chat request
	req := interfaces.ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	// Get response from LLM
	resp, err := a.llmProvider.ChatCompletion(ctx, req)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeProvider, "failed to get LLM chat completion response").
			WithComponent("corpus_agent").
			WithOperation("generateResponse").
			WithDetails("model", req.Model).
			WithDetails("message_count", len(req.Messages))
	}

	if len(resp.Choices) == 0 {
		return "", gerror.New(gerror.ErrCodeProvider, "no response choices from LLM provider", nil).
			WithComponent("corpus_agent").
			WithOperation("generateResponse").
			WithDetails("model", req.Model)
	}

	return resp.Choices[0].Message.Content, nil
}

// buildPrompt creates a prompt with retrieved context
func (a *CorpusAgent) buildPrompt(query string, results *rag.SearchResults) string {
	var builder strings.Builder

	builder.WriteString("Based on the following context from our knowledge base, ")
	builder.WriteString("please provide a comprehensive response to this query:\n\n")

	// Add retrieved context
	if len(results.Results) > 0 {
		builder.WriteString("## Relevant Context:\n\n")
		for i, result := range results.Results {
			builder.WriteString(fmt.Sprintf("### Source %d (Relevance: %.2f)\n", i+1, result.Score))
			builder.WriteString(result.Content)
			builder.WriteString("\n\n")

			// Add metadata if available
			if len(result.Metadata) > 0 {
				builder.WriteString("Metadata: ")
				for k, v := range result.Metadata {
					builder.WriteString(fmt.Sprintf("%s=%v ", k, v))
				}
				builder.WriteString("\n\n")
			}
		}
	} else {
		builder.WriteString("(No specific context found in knowledge base)\n\n")
	}

	builder.WriteString("## Query:\n")
	builder.WriteString(query)
	builder.WriteString("\n\n")
	builder.WriteString("Please synthesize the available information into a clear, well-structured response.")

	return builder.String()
}

// getSystemPrompt returns the system prompt for the Corpus Agent
func (a *CorpusAgent) getSystemPrompt() string {
	return `You are the Corpus Knowledge Navigator, an intelligent agent that helps users
explore and understand information stored in the Guild Framework's knowledge base.

Your role is to:
1. Search through the RAG system to find relevant information
2. Synthesize multiple sources into coherent, human-readable responses
3. Format information clearly with appropriate structure and markdown
4. Acknowledge when information is limited or unavailable
5. Suggest related topics the user might find interesting

When generating responses:
- Use clear headings and structure
- Cite sources when specific information comes from particular documents
- Acknowledge uncertainty when context is limited
- Provide comprehensive yet concise answers
- Use markdown formatting for readability

Remember: You are the bridge between the comprehensive RAG system and human understanding.
Make information accessible, accurate, and actionable.`
}

// extractTags generates relevant tags for the document
func (a *CorpusAgent) extractTags(query, response string) []string {
	// Simple tag extraction - in production, this could use NLP
	tags := []string{"generated", "corpus-agent"}

	// Add some basic keyword extraction
	keywords := []string{
		"api", "function", "method", "class", "interface", "implementation",
		"design", "architecture", "pattern", "system", "component", "module",
	}

	lowerQuery := strings.ToLower(query)
	lowerResponse := strings.ToLower(response)
	combined := lowerQuery + " " + lowerResponse

	for _, keyword := range keywords {
		if strings.Contains(combined, keyword) {
			tags = append(tags, keyword)
		}
	}

	// Limit to reasonable number of tags
	if len(tags) > 10 {
		tags = tags[:10]
	}

	return tags
}

// ClearHistory clears the conversation history
func (a *CorpusAgent) ClearHistory() {
	a.conversationHistory = make([]Message, 0)
}

// GetID returns the agent's ID
func (a *CorpusAgent) GetID() string {
	return a.ID
}

// GetName returns the agent's name
func (a *CorpusAgent) GetName() string {
	return a.Name
}
