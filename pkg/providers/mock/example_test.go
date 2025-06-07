package mock_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/guild-ventures/guild-core/pkg/providers/mock"
)

// Example shows basic mock provider usage
func Example() {
	// Create mock provider
	provider := mock.NewProvider()

	// Set up responses
	provider.SetResponse("Hello", "Hi there!")
	provider.SetResponse("What's 2+2?", "4")
	provider.SetDefaultResponse("I don't understand")

	// Use the provider
	ctx := context.Background()
	req := interfaces.ChatRequest{
		Model: "mock-model",
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, _ := provider.ChatCompletion(ctx, req)
	fmt.Println(resp.Choices[0].Message.Content)
	// Output: Hi there!
}

// Example_builder shows using the builder pattern
func Example_builder() {
	provider := mock.NewBuilder().
		WithResponse("test", "test response").
		WithDefaultResponse("default response").
		WithDelay(100 * time.Millisecond).
		Build()

	// Use the provider...
	_ = provider
}

// Example: Testing error conditions
func TestErrorConditions(t *testing.T) {
	provider := mock.NewBuilder().
		WithError("error prompt", fmt.Errorf("simulated error")).
		Build()

	ctx := context.Background()
	req := interfaces.ChatRequest{
		Model: "mock-model",
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "error prompt"},
		},
	}

	_, err := provider.ChatCompletion(ctx, req)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// Example: Verifying API calls
func TestAPICallVerification(t *testing.T) {
	provider := mock.NewProvider()
	ctx := context.Background()

	// Make some calls
	requests := []string{"first", "second", "third"}
	for _, content := range requests {
		req := interfaces.ChatRequest{
			Model: "mock-model",
			Messages: []interfaces.ChatMessage{
				{Role: "user", Content: content},
			},
		}
		provider.ChatCompletion(ctx, req)
	}

	// Verify calls
	calls := provider.GetCalls()
	if len(calls) != 3 {
		t.Errorf("Expected 3 calls, got %d", len(calls))
	}

	// Check call details
	for i, call := range calls {
		if call.Method != "ChatCompletion" {
			t.Errorf("Call %d: wrong method %s", i, call.Method)
		}

		req := call.Request.(interfaces.ChatRequest)
		if req.Messages[0].Content != requests[i] {
			t.Errorf("Call %d: wrong content", i)
		}
	}
}

// Example: Testing streaming
func TestStreaming(t *testing.T) {
	provider := mock.NewBuilder().
		WithResponse("stream test", "This is a streaming response").
		Build()

	ctx := context.Background()
	req := interfaces.ChatRequest{
		Model: "mock-model",
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "stream test"},
		},
	}

	stream, err := provider.StreamChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}
	defer stream.Close()

	// Collect chunks
	var chunks []string
	for {
		chunk, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
		chunks = append(chunks, chunk.Delta.Content)
	}

	// Verify we got chunks
	if len(chunks) == 0 {
		t.Error("No chunks received")
	}

	// Reconstruct message
	full := strings.Join(chunks, "")
	if full != "This is a streaming response" {
		t.Errorf("Wrong streamed content: %s", full)
	}
}

// Example: Testing embeddings
func TestEmbeddings(t *testing.T) {
	provider := mock.NewProvider()
	ctx := context.Background()

	req := interfaces.EmbeddingRequest{
		Model: "mock-model",
		Input: []string{"test1", "test2", "test3"},
	}

	resp, err := provider.CreateEmbedding(ctx, req)
	if err != nil {
		t.Fatalf("CreateEmbedding failed: %v", err)
	}

	if len(resp.Embeddings) != 3 {
		t.Errorf("Expected 3 embeddings, got %d", len(resp.Embeddings))
	}

	// Verify embeddings have consistent dimensions
	dim := len(resp.Embeddings[0].Embedding)
	for i, emb := range resp.Embeddings {
		if len(emb.Embedding) != dim {
			t.Errorf("Embedding %d has wrong dimension: %d", i, len(emb.Embedding))
		}
	}
}
