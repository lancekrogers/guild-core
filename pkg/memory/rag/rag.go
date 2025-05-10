package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/blockhead-consulting/guild/pkg/memory/vector"
	"github.com/blockhead-consulting/guild/pkg/providers"
	"github.com/google/uuid"
)

// RAGSystem represents a Retrieval Augmented Generation system
type RAGSystem struct {
	vectorStore vector.VectorStore
	llmClient   providers.LLMClient
	config      *Config
}

// Config contains configuration for the RAG system
type Config struct {
	// MaxRetrievalResults is the maximum number of results to retrieve
	MaxRetrievalResults int

	// IncludeMetadata determines whether to include document metadata in the prompt
	IncludeMetadata bool

	// PromptTemplate is the template for the RAG prompt
	PromptTemplate string

	// DefaultModel is the default model to use for the LLM
	DefaultModel string

	// DefaultTemperature is the default temperature to use for the LLM
	DefaultTemperature float64

	// DefaultMaxTokens is the default max tokens to use for the LLM
	DefaultMaxTokens int
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxRetrievalResults: 5,
		IncludeMetadata:     true,
		PromptTemplate: `
You are a helpful knowledge-based assistant. 
Answer the question based on the context provided below. 
If the answer is not in the context, say "I don't have enough information to answer this question."

Context:
{{.Context}}

Question: {{.Question}}
`,
		DefaultModel:       "",
		DefaultTemperature: 0.0,
		DefaultMaxTokens:   1000,
	}
}

// NewRAGSystem creates a new RAG system
func NewRAGSystem(vectorStore vector.VectorStore, llmClient providers.LLMClient, config *Config) *RAGSystem {
	if config == nil {
		config = DefaultConfig()
	}

	return &RAGSystem{
		vectorStore: vectorStore,
		llmClient:   llmClient,
		config:      config,
	}
}

// AnswerQuestion answers a question based on retrieved context
func (r *RAGSystem) AnswerQuestion(ctx context.Context, question string) (string, error) {
	// Retrieve relevant documents
	results, err := r.vectorStore.QueryByText(ctx, question, r.config.MaxRetrievalResults)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve documents: %w", err)
	}

	if len(results) == 0 {
		return "I don't have enough information to answer this question.", nil
	}

	// Build context from retrieved documents
	var contextBuilder strings.Builder
	for i, result := range results {
		doc := result.Document

		contextBuilder.WriteString(fmt.Sprintf("[Document %d] (Score: %.4f)\n", i+1, result.Score))
		contextBuilder.WriteString(doc.Content)
		contextBuilder.WriteString("\n\n")

		if r.config.IncludeMetadata && len(doc.Metadata) > 0 {
			contextBuilder.WriteString("Metadata:\n")
			for k, v := range doc.Metadata {
				contextBuilder.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
			}
			contextBuilder.WriteString("\n")
		}
	}

	// Build prompt
	prompt := r.config.PromptTemplate
	prompt = strings.Replace(prompt, "{{.Context}}", contextBuilder.String(), -1)
	prompt = strings.Replace(prompt, "{{.Question}}", question, -1)

	// Generate answer
	req := &providers.CompletionRequest{
		Prompt:      prompt,
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

	return resp.Text, nil
}

// AddDocument adds a document to the vector store
func (r *RAGSystem) AddDocument(ctx context.Context, content string, metadata map[string]string) (string, error) {
	id := uuid.New().String()

	doc := &vector.Document{
		ID:       id,
		Content:  content,
		Metadata: metadata,
	}

	if err := r.vectorStore.Store(ctx, doc); err != nil {
		return "", fmt.Errorf("failed to store document: %w", err)
	}

	return id, nil
}

// AddDocuments adds multiple documents to the vector store
func (r *RAGSystem) AddDocuments(ctx context.Context, contents []string, metadatas []map[string]string) ([]string, error) {
	if len(contents) == 0 {
		return []string{}, nil
	}

	if metadatas != nil && len(metadatas) != len(contents) {
		return nil, fmt.Errorf("number of metadatas must match number of contents")
	}

	docs := make([]*vector.Document, len(contents))
	ids := make([]string, len(contents))

	for i, content := range contents {
		id := uuid.New().String()
		ids[i] = id

		var metadata map[string]string
		if metadatas != nil {
			metadata = metadatas[i]
		} else {
			metadata = make(map[string]string)
		}

		docs[i] = &vector.Document{
			ID:       id,
			Content:  content,
			Metadata: metadata,
		}
	}

	if err := r.vectorStore.StoreMany(ctx, docs); err != nil {
		return nil, fmt.Errorf("failed to store documents: %w", err)
	}

	return ids, nil
}

// GetDocument retrieves a document by ID
func (r *RAGSystem) GetDocument(ctx context.Context, id string) (*vector.Document, error) {
	return r.vectorStore.Retrieve(ctx, id)
}

// DeleteDocument deletes a document by ID
func (r *RAGSystem) DeleteDocument(ctx context.Context, id string) error {
	return r.vectorStore.Delete(ctx, id)
}

// DeleteDocuments deletes multiple documents by ID
func (r *RAGSystem) DeleteDocuments(ctx context.Context, ids []string) error {
	return r.vectorStore.DeleteMany(ctx, ids)
}