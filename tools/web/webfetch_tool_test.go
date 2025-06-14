package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/guild-ventures/guild-core/tools"
)

// MockAIProvider implements the AIProvider interface for testing
type MockAIProvider struct {
	mock.Mock
}

func (m *MockAIProvider) ChatCompletion(ctx context.Context, req interfaces.ChatRequest) (*interfaces.ChatResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*interfaces.ChatResponse), args.Error(1)
}

func (m *MockAIProvider) StreamChatCompletion(ctx context.Context, req interfaces.ChatRequest) (interfaces.ChatStream, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(interfaces.ChatStream), args.Error(1)
}

func (m *MockAIProvider) CreateEmbedding(ctx context.Context, req interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*interfaces.EmbeddingResponse), args.Error(1)
}

func (m *MockAIProvider) GetCapabilities() interfaces.ProviderCapabilities {
	args := m.Called()
	return args.Get(0).(interfaces.ProviderCapabilities)
}

func TestWebFetchTool_Interface(t *testing.T) {
	mockProvider := &MockAIProvider{}
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	
	// Ensure it implements the Tool interface
	var _ tools.Tool = tool
	
	assert.Equal(t, "web_fetch", tool.Name())
	assert.Equal(t, "web", tool.Category())
	assert.False(t, tool.RequiresAuth())
	assert.NotEmpty(t, tool.Description())
	assert.NotEmpty(t, tool.Examples())
	assert.NotNil(t, tool.Schema())
}

func TestWebFetchTool_Schema(t *testing.T) {
	mockProvider := &MockAIProvider{}
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	schema := tool.Schema()
	
	assert.Equal(t, "object", schema["type"])
	
	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)
	
	// Check required fields
	url, ok := properties["url"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "string", url["type"])
	assert.Equal(t, "uri", url["format"])
	
	prompt, ok := properties["prompt"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "string", prompt["type"])
	
	// Check required fields
	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "url")
	assert.Contains(t, required, "prompt")
}

func TestWebFetchTool_Execute_InvalidInput(t *testing.T) {
	mockProvider := &MockAIProvider{}
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	ctx := context.Background()
	
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid JSON",
			input: `{"invalid": json}`,
		},
		{
			name:  "missing URL",
			input: `{"prompt": "analyze this"}`,
		},
		{
			name:  "missing prompt",
			input: `{"url": "https://example.com"}`,
		},
		{
			name:  "empty URL",
			input: `{"url": "", "prompt": "analyze"}`,
		},
		{
			name:  "empty prompt",
			input: `{"url": "https://example.com", "prompt": ""}`,
		},
		{
			name:  "invalid URL format",
			input: `{"url": "not-a-url", "prompt": "analyze"}`,
		},
		{
			name:  "unsupported URL scheme",
			input: `{"url": "ftp://example.com", "prompt": "analyze"}`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.input)
			assert.NoError(t, err) // Tool should handle errors gracefully
			assert.NotNil(t, result)
			assert.False(t, result.Success)
			assert.NotEmpty(t, result.Error)
		})
	}
}

func TestWebFetchTool_Execute_ValidInput(t *testing.T) {
	// Create mock HTTP server
	mockHTML := `
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<title>Test Page</title>
			<meta name="description" content="This is a test page">
			<meta name="author" content="Test Author">
		</head>
		<body>
			<h1>Main Title</h1>
			<p>This is a test paragraph with some content.</p>
			<a href="https://example.com">Example Link</a>
			<img src="test.jpg" alt="Test Image">
		</body>
		</html>
	`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()
	
	// Create mock AI provider
	mockProvider := &MockAIProvider{}
	mockProvider.On("ChatCompletion", mock.Anything, mock.Anything).Return(
		&interfaces.ChatResponse{
			Choices: []interfaces.ChatChoice{
				{
					Message: interfaces.ChatMessage{
						Content: "This is a test analysis of the web page content.",
					},
				},
			},
		}, nil)
	
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	ctx := context.Background()
	
	input := fmt.Sprintf(`{"url": "%s", "prompt": "Summarize this page"}`, server.URL)
	
	result, err := tool.Execute(ctx, input)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	
	// Parse the response
	var response WebFetchResponse
	err = json.Unmarshal([]byte(result.Output), &response)
	assert.NoError(t, err)
	
	assert.Equal(t, server.URL, response.URL)
	assert.Equal(t, "Test Page", response.Title)
	assert.Contains(t, response.Content, "Main Title")
	assert.Contains(t, response.Content, "test paragraph")
	assert.Equal(t, "This is a test analysis of the web page content.", response.Analysis)
	assert.Equal(t, "Test Page", response.Metadata.Title)
	assert.Equal(t, "This is a test page", response.Metadata.Description)
	assert.Equal(t, "Test Author", response.Metadata.Author)
	assert.Equal(t, "en", response.Metadata.Language)
	assert.GreaterOrEqual(t, response.ProcessingTime, 0.0)
	assert.False(t, response.FromCache)
	
	mockProvider.AssertExpectations(t)
}

func TestWebFetchTool_Execute_HTTPError(t *testing.T) {
	// Create server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()
	
	mockProvider := &MockAIProvider{}
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	ctx := context.Background()
	
	input := fmt.Sprintf(`{"url": "%s", "prompt": "Analyze this"}`, server.URL)
	
	result, err := tool.Execute(ctx, input)
	assert.NoError(t, err) // Tool handles errors gracefully
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	
	// Parse the response
	var response WebFetchResponse
	err = json.Unmarshal([]byte(result.Output), &response)
	assert.NoError(t, err)
	
	assert.Equal(t, server.URL, response.URL)
	assert.Contains(t, response.Error, "HTTP request failed with status 404")
}

func TestWebFetchTool_Execute_AIProviderError(t *testing.T) {
	// Create mock HTTP server
	mockHTML := `<html><body><h1>Test</h1></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()
	
	// Create mock AI provider that returns error
	mockProvider := &MockAIProvider{}
	mockProvider.On("ChatCompletion", mock.Anything, mock.Anything).Return(
		(*interfaces.ChatResponse)(nil), fmt.Errorf("AI service unavailable"))
	
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	ctx := context.Background()
	
	input := fmt.Sprintf(`{"url": "%s", "prompt": "Analyze this"}`, server.URL)
	
	result, err := tool.Execute(ctx, input)
	assert.NoError(t, err) // Tool handles errors gracefully
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	
	// Parse the response
	var response WebFetchResponse
	err = json.Unmarshal([]byte(result.Output), &response)
	assert.NoError(t, err)
	
	assert.Equal(t, server.URL, response.URL)
	assert.Contains(t, response.Error, "Analysis failed")
	assert.Contains(t, response.Error, "AI service unavailable")
	
	mockProvider.AssertExpectations(t)
}

func TestWebFetchTool_ContentExtraction(t *testing.T) {
	mockProvider := &MockAIProvider{}
	
	tests := []struct {
		name     string
		html     string
		expected map[string]bool // Map of expected strings and whether they should be present
	}{
		{
			name: "basic HTML structure",
			html: `
				<html>
				<head><title>Test Title</title></head>
				<body>
					<h1>Main Heading</h1>
					<h2>Sub Heading</h2>
					<p>This is a paragraph.</p>
					<strong>Bold text</strong>
					<em>Italic text</em>
				</body>
				</html>
			`,
			expected: map[string]bool{
				"# Test Title":      true,
				"# Main Heading":    true,
				"## Sub Heading":    true,
				"This is a paragraph": true,
				"**Bold text**":     true,
				"*Italic text*":     true,
			},
		},
		{
			name: "lists and links",
			html: `
				<html>
				<body>
					<ul>
						<li>Item 1</li>
						<li>Item 2</li>
					</ul>
					<ol>
						<li>First</li>
						<li>Second</li>
					</ol>
					<a href="https://example.com">Link Text</a>
				</body>
				</html>
			`,
			expected: map[string]bool{
				"- Item 1":        true,
				"- Item 2":        true,
				"1. First":        true,
				"2. Second":       true,
				"[Link Text](https://example.com)": true,
			},
		},
		{
			name: "code and blockquotes",
			html: `
				<html>
				<body>
					<pre>code block</pre>
					<code>inline code</code>
					<blockquote>This is a quote</blockquote>
				</body>
				</html>
			`,
			expected: map[string]bool{
				"```\ncode block\n```": true,
				"`inline code`":        true,
				"> This is a quote":     true,
			},
		},
		{
			name: "unwanted elements removed",
			html: `
				<html>
				<head>
					<script>alert('test');</script>
					<style>body { color: red; }</style>
				</head>
				<body>
					<p>Visible content</p>
					<nav>Navigation</nav>
					<header>Header</header>
					<footer>Footer</footer>
					<aside>Sidebar</aside>
					<div class="advertisement">Ad content</div>
				</body>
				</html>
			`,
			expected: map[string]bool{
				"Visible content": true,
				"alert('test')":   false,
				"color: red":      false,
				"Navigation":      false,
				"Header":          false,
				"Footer":          false,
				"Sidebar":         false,
				"Ad content":      false,
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create server with test HTML
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.html))
			}))
			defer server.Close()
			
			// Mock AI provider
			mockProvider.On("ChatCompletion", mock.Anything, mock.Anything).Return(
				&interfaces.ChatResponse{
					Choices: []interfaces.ChatChoice{
						{Message: interfaces.ChatMessage{Content: "Test analysis"}},
					},
				}, nil).Once()
			
			testTool := NewWebFetchTool(mockProvider)
			defer testTool.Close()
			ctx := context.Background()
			
			input := fmt.Sprintf(`{"url": "%s", "prompt": "Analyze"}`, server.URL)
			result, err := testTool.Execute(ctx, input)
			
			assert.NoError(t, err)
			assert.True(t, result.Success)
			
			var response WebFetchResponse
			err = json.Unmarshal([]byte(result.Output), &response)
			assert.NoError(t, err)
			
			content := response.Content
			for expectedText, shouldBePresent := range tt.expected {
				if shouldBePresent {
					assert.Contains(t, content, expectedText, "Expected content to contain: %s", expectedText)
				} else {
					assert.NotContains(t, content, expectedText, "Expected content to NOT contain: %s", expectedText)
				}
			}
		})
	}
}

func TestWebFetchTool_MetadataExtraction(t *testing.T) {
	mockHTML := `
		<!DOCTYPE html>
		<html lang="fr">
		<head>
			<title>Page Title</title>
			<meta name="description" content="Page description">
			<meta name="keywords" content="keyword1, keyword2">
			<meta name="author" content="Page Author">
		</head>
		<body>
			<p>This is test content with multiple words for counting.</p>
			<a href="https://example.com">Link 1</a>
			<a href="/relative">Link 2</a>
			<img src="image1.jpg" alt="Image 1">
			<img src="image2.png" alt="Image 2">
		</body>
		</html>
	`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()
	
	mockProvider := &MockAIProvider{}
	mockProvider.On("ChatCompletion", mock.Anything, mock.Anything).Return(
		&interfaces.ChatResponse{
			Choices: []interfaces.ChatChoice{
				{Message: interfaces.ChatMessage{Content: "Analysis"}},
			},
		}, nil)
	
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	ctx := context.Background()
	
	input := fmt.Sprintf(`{"url": "%s", "prompt": "Analyze"}`, server.URL)
	result, err := tool.Execute(ctx, input)
	
	assert.NoError(t, err)
	assert.True(t, result.Success)
	
	var response WebFetchResponse
	err = json.Unmarshal([]byte(result.Output), &response)
	assert.NoError(t, err)
	
	metadata := response.Metadata
	assert.Equal(t, "Page Title", metadata.Title)
	assert.Equal(t, "Page description", metadata.Description)
	assert.Equal(t, "keyword1, keyword2", metadata.Keywords)
	assert.Equal(t, "Page Author", metadata.Author)
	assert.Equal(t, "fr", metadata.Language)
	assert.Equal(t, "text/html; charset=utf-8", metadata.ContentType)
	assert.Equal(t, "Wed, 21 Oct 2015 07:28:00 GMT", metadata.LastModified)
	assert.Equal(t, 200, metadata.StatusCode)
	assert.Greater(t, metadata.ContentLength, 0)
	assert.Greater(t, metadata.WordCount, 0)
	assert.GreaterOrEqual(t, metadata.ReadingTimeMin, 1)
	assert.Len(t, metadata.Links, 2)
	assert.Len(t, metadata.Images, 2)
	assert.Contains(t, metadata.Links, "https://example.com")
	assert.Contains(t, metadata.Links, "/relative")
	assert.Contains(t, metadata.Images, "image1.jpg")
	assert.Contains(t, metadata.Images, "image2.png")
}

func TestWebFetchTool_Cache(t *testing.T) {
	mockHTML := `<html><body><h1>Test</h1></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()
	
	mockProvider := &MockAIProvider{}
	// First call should hit the AI provider
	mockProvider.On("ChatCompletion", mock.Anything, mock.Anything).Return(
		&interfaces.ChatResponse{
			Choices: []interfaces.ChatChoice{
				{Message: interfaces.ChatMessage{Content: "Cached analysis"}},
			},
		}, nil).Once()
	
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	ctx := context.Background()
	
	input := fmt.Sprintf(`{"url": "%s", "prompt": "Analyze this"}`, server.URL)
	
	// First execution - should call AI provider
	result1, err := tool.Execute(ctx, input)
	assert.NoError(t, err)
	assert.True(t, result1.Success)
	
	var response1 WebFetchResponse
	err = json.Unmarshal([]byte(result1.Output), &response1)
	assert.NoError(t, err)
	assert.False(t, response1.FromCache)
	assert.Equal(t, "Cached analysis", response1.Analysis)
	
	// Second execution - should use cache (no additional AI provider call)
	result2, err := tool.Execute(ctx, input)
	assert.NoError(t, err)
	assert.True(t, result2.Success)
	
	var response2 WebFetchResponse
	err = json.Unmarshal([]byte(result2.Output), &response2)
	assert.NoError(t, err)
	assert.True(t, response2.FromCache)
	assert.Equal(t, "Cached analysis", response2.Analysis)
	
	mockProvider.AssertExpectations(t)
}

func TestWebFetchTool_LargeContent(t *testing.T) {
	// Create large HTML content
	largeContent := strings.Repeat("This is a long paragraph with lots of content. ", 1000)
	mockHTML := fmt.Sprintf(`<html><body><p>%s</p></body></html>`, largeContent)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()
	
	mockProvider := &MockAIProvider{}
	mockProvider.On("ChatCompletion", mock.Anything, mock.MatchedBy(func(req interfaces.ChatRequest) bool {
		// Verify that content is truncated for AI analysis
		content := req.Messages[0].Content
		return len(content) < 15000 // Should be truncated
	})).Return(
		&interfaces.ChatResponse{
			Choices: []interfaces.ChatChoice{
				{Message: interfaces.ChatMessage{Content: "Analysis of truncated content"}},
			},
		}, nil)
	
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	ctx := context.Background()
	
	input := fmt.Sprintf(`{"url": "%s", "prompt": "Analyze"}`, server.URL)
	result, err := tool.Execute(ctx, input)
	
	assert.NoError(t, err)
	assert.True(t, result.Success)
	
	var response WebFetchResponse
	err = json.Unmarshal([]byte(result.Output), &response)
	assert.NoError(t, err)
	
	assert.Equal(t, "Analysis of truncated content", response.Analysis)
	
	mockProvider.AssertExpectations(t)
}

func TestWebFetchTool_Redirects(t *testing.T) {
	// Create server that redirects
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Final Page</h1></body></html>`))
	}))
	defer redirectServer.Close()
	
	mainServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectServer.URL, http.StatusMovedPermanently)
	}))
	defer mainServer.Close()
	
	mockProvider := &MockAIProvider{}
	mockProvider.On("ChatCompletion", mock.Anything, mock.Anything).Return(
		&interfaces.ChatResponse{
			Choices: []interfaces.ChatChoice{
				{Message: interfaces.ChatMessage{Content: "Analysis of redirected content"}},
			},
		}, nil)
	
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	ctx := context.Background()
	
	input := fmt.Sprintf(`{"url": "%s", "prompt": "Analyze"}`, mainServer.URL)
	result, err := tool.Execute(ctx, input)
	
	assert.NoError(t, err)
	assert.True(t, result.Success)
	
	var response WebFetchResponse
	err = json.Unmarshal([]byte(result.Output), &response)
	assert.NoError(t, err)
	
	assert.Contains(t, response.Content, "Final Page")
	assert.Equal(t, "Analysis of redirected content", response.Analysis)
	
	mockProvider.AssertExpectations(t)
}

func TestWebFetchCache_Operations(t *testing.T) {
	cache := NewWebFetchCache(3) // Small cache for testing
	defer cache.Stop()
	
	response1 := &WebFetchResponse{URL: "http://example1.com", Analysis: "Analysis 1"}
	response2 := &WebFetchResponse{URL: "http://example2.com", Analysis: "Analysis 2"}
	response3 := &WebFetchResponse{URL: "http://example3.com", Analysis: "Analysis 3"}
	response4 := &WebFetchResponse{URL: "http://example4.com", Analysis: "Analysis 4"}
	
	// Test setting and getting
	cache.Set("key1", response1, 1*time.Hour)
	cached, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "Analysis 1", cached.Analysis)
	assert.True(t, cached.FromCache)
	
	// Test cache miss
	_, found = cache.Get("nonexistent")
	assert.False(t, found)
	
	// Test cache eviction (fill cache and add one more)
	cache.Set("key2", response2, 1*time.Hour)
	cache.Set("key3", response3, 1*time.Hour)
	cache.Set("key4", response4, 1*time.Hour) // Should evict oldest
	
	// key1 should be evicted
	_, found = cache.Get("key1")
	assert.False(t, found)
	
	// Others should still be there
	_, found = cache.Get("key2")
	assert.True(t, found)
	_, found = cache.Get("key3")
	assert.True(t, found)
	_, found = cache.Get("key4")
	assert.True(t, found)
}

func TestWebFetchCache_Expiration(t *testing.T) {
	cache := NewWebFetchCache(10)
	defer cache.Stop()
	
	response := &WebFetchResponse{URL: "http://example.com", Analysis: "Analysis"}
	
	// Set with very short TTL
	cache.Set("key", response, 1*time.Millisecond)
	
	// Should be available immediately
	_, found := cache.Get("key")
	assert.True(t, found)
	
	// Wait for expiration
	time.Sleep(2 * time.Millisecond)
	
	// Should be expired
	_, found = cache.Get("key")
	assert.False(t, found)
}

func BenchmarkWebFetchTool_Execute(b *testing.B) {
	mockHTML := `<html><body><h1>Benchmark Test</h1><p>Content</p></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()
	
	mockProvider := &MockAIProvider{}
	mockProvider.On("ChatCompletion", mock.Anything, mock.Anything).Return(
		&interfaces.ChatResponse{
			Choices: []interfaces.ChatChoice{
				{Message: interfaces.ChatMessage{Content: "Benchmark analysis"}},
			},
		}, nil)
	
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	ctx := context.Background()
	input := fmt.Sprintf(`{"url": "%s", "prompt": "Analyze"}`, server.URL)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tool.Execute(ctx, input)
		if err != nil {
			b.Fatalf("Execute failed: %v", err)
		}
	}
}

func BenchmarkWebFetchTool_ContentExtraction(b *testing.B) {
	mockProvider := &MockAIProvider{}
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	
	// Create complex HTML for benchmarking
	complexHTML := `
		<html>
		<head><title>Complex Page</title></head>
		<body>
			<h1>Main Title</h1>
			<div class="content">
				<h2>Section 1</h2>
				<p>Paragraph with <strong>bold</strong> and <em>italic</em> text.</p>
				<ul>
					<li>List item 1</li>
					<li>List item 2</li>
				</ul>
				<blockquote>Quote content</blockquote>
				<pre><code>Code block</code></pre>
			</div>
		</body>
		</html>
	`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(complexHTML))
	}))
	defer server.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := tool.fetchContent(context.Background(), server.URL)
		if err != nil {
			b.Fatalf("fetchContent failed: %v", err)
		}
	}
}

func TestWebFetchTool_Timeout(t *testing.T) {
	// Create a slow server to test timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wait for the context to be cancelled, which will happen after 1 second
		select {
		case <-r.Context().Done():
			// Client disconnected due to timeout - this is expected
			return
		case <-time.After(10 * time.Second):
			// Fallback - write response if context doesn't get cancelled
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer func() {
		// Close client connections first to avoid blocking
		server.CloseClientConnections()
		server.Close()
	}()
	
	mockProvider := &MockAIProvider{}
	tool := NewWebFetchTool(mockProvider)
	defer tool.Close()
	
	// Create a context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	input := fmt.Sprintf(`{"url": "%s", "prompt": "Analyze"}`, server.URL)
	
	result, err := tool.Execute(ctx, input)
	assert.NoError(t, err) // Tool handles errors gracefully
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	
	var response WebFetchResponse
	err = json.Unmarshal([]byte(result.Output), &response)
	assert.NoError(t, err)
	
	assert.Contains(t, response.Error, "context deadline exceeded")
}