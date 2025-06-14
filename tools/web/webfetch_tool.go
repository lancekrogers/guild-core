package web

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/guild-ventures/guild-core/tools"
)

// WebFetchTool implements web content fetching and AI-powered analysis
type WebFetchTool struct {
	*tools.BaseTool
	client       *http.Client
	aiProvider   providers.AIProvider
	cache        *WebFetchCache
	userAgent    string
	maxBodySize  int64
}

// WebFetchRequest represents the input parameters for web fetch
type WebFetchRequest struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
}

// WebFetchResponse represents the response from web fetch
type WebFetchResponse struct {
	URL           string            `json:"url"`
	Title         string            `json:"title"`
	Content       string            `json:"content"`
	Analysis      string            `json:"analysis"`
	Metadata      WebPageMetadata   `json:"metadata"`
	ProcessingTime float64          `json:"processing_time_ms"`
	FromCache     bool              `json:"from_cache"`
	Error         string            `json:"error,omitempty"`
}

// WebPageMetadata contains extracted metadata from the web page
type WebPageMetadata struct {
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Keywords        string            `json:"keywords"`
	Author          string            `json:"author"`
	Language        string            `json:"language"`
	ContentType     string            `json:"content_type"`
	ContentLength   int               `json:"content_length"`
	LastModified    string            `json:"last_modified"`
	StatusCode      int               `json:"status_code"`
	Headers         map[string]string `json:"headers"`
	Links           []string          `json:"links"`
	Images          []string          `json:"images"`
	WordCount       int               `json:"word_count"`
	ReadingTimeMin  int               `json:"reading_time_minutes"`
}

// CacheEntry represents a cached web fetch result
type CacheEntry struct {
	Response  *WebFetchResponse
	Timestamp time.Time
	TTL       time.Duration
}

// WebFetchCache provides caching for web fetch results
type WebFetchCache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
	maxSize int
}

// NewWebFetchCache creates a new cache
func NewWebFetchCache(maxSize int) *WebFetchCache {
	cache := &WebFetchCache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
	}
	
	// Start cleanup goroutine
	go cache.cleanupExpired()
	
	return cache
}

// Get retrieves a cached entry
func (c *WebFetchCache) Get(key string) (*WebFetchResponse, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}
	
	if time.Since(entry.Timestamp) > entry.TTL {
		return nil, false
	}
	
	// Mark as from cache
	response := *entry.Response
	response.FromCache = true
	return &response, true
}

// Set stores a cache entry
func (c *WebFetchCache) Set(key string, response *WebFetchResponse, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Remove oldest entries if cache is full
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}
	
	c.entries[key] = &CacheEntry{
		Response:  response,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

// evictOldest removes the oldest cache entry
func (c *WebFetchCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range c.entries {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}
	
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// cleanupExpired removes expired entries periodically
func (c *WebFetchCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.Sub(entry.Timestamp) > entry.TTL {
				delete(c.entries, key)
			}
		}
		c.mutex.Unlock()
	}
}

// NewWebFetchTool creates a new web fetch tool
func NewWebFetchTool(aiProvider providers.AIProvider) *WebFetchTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"format":      "uri",
				"description": "The URL to fetch and analyze",
			},
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "The analysis prompt to apply to the fetched content",
			},
		},
		"required": []string{"url", "prompt"},
	}

	examples := []string{
		`{"url": "https://example.com/article", "prompt": "Summarize the main points of this article"}`,
		`{"url": "https://github.com/owner/repo", "prompt": "What is this project about and what are its key features?"}`,
		`{"url": "https://news.site.com/story", "prompt": "Extract the key facts and provide a neutral summary"}`,
		`{"url": "https://docs.example.com/api", "prompt": "Explain the API endpoints and their usage"}`,
	}

	baseTool := tools.NewBaseTool(
		"web_fetch",
		"Fetch content from a URL and analyze it using AI. Converts HTML to markdown, extracts metadata, and provides AI-powered analysis based on the provided prompt. Includes a 15-minute cache for faster responses.",
		schema,
		"web",
		false, // No auth required for basic fetching
		examples,
	)

	return &WebFetchTool{
		BaseTool:  baseTool,
		aiProvider: aiProvider,
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 10 redirects
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		cache:       NewWebFetchCache(100), // Cache up to 100 entries
		userAgent:   "Guild-Framework-WebFetch/1.0 (AI Assistant)",
		maxBodySize: 10 * 1024 * 1024, // 10MB max
	}
}

// Execute fetches and analyzes web content
func (t *WebFetchTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var req WebFetchRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to parse web fetch request"), nil), nil
	}

	// Validate inputs
	if req.URL == "" {
		return tools.NewToolResult("", nil, gerror.New(gerror.ErrCodeInvalidInput, "URL is required", nil), nil), nil
	}

	if req.Prompt == "" {
		return tools.NewToolResult("", nil, gerror.New(gerror.ErrCodeInvalidInput, "prompt is required", nil), nil), nil
	}

	// Validate URL format
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid URL format"), nil), nil
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return tools.NewToolResult("", nil, gerror.New(gerror.ErrCodeInvalidInput, "URL must use http or https scheme", nil), nil), nil
	}

	startTime := time.Now()

	// Check cache first
	cacheKey := t.generateCacheKey(req.URL, req.Prompt)
	if cachedResponse, found := t.cache.Get(cacheKey); found {
		// Update processing time for cache hit
		cachedResponse.ProcessingTime = float64(time.Since(startTime).Nanoseconds()) / 1e6
		
		outputJSON, err := json.MarshalIndent(cachedResponse, "", "  ")
		if err != nil {
			return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal cached response"), nil), nil
		}

		metadata := map[string]string{
			"url":             req.URL,
			"from_cache":      "true",
			"processing_time": fmt.Sprintf("%.2fms", cachedResponse.ProcessingTime),
		}

		return tools.NewToolResult(string(outputJSON), metadata, nil, nil), nil
	}

	// Fetch content
	content, metadata, err := t.fetchContent(ctx, req.URL)
	if err != nil {
		response := &WebFetchResponse{
			URL:            req.URL,
			Error:          err.Error(),
			ProcessingTime: float64(time.Since(startTime).Nanoseconds()) / 1e6,
		}

		outputJSON, _ := json.MarshalIndent(response, "", "  ")
		return tools.NewToolResult(string(outputJSON), nil, err, nil), nil
	}

	// Analyze content with AI
	analysis, err := t.analyzeContent(ctx, content, req.Prompt)
	if err != nil {
		response := &WebFetchResponse{
			URL:            req.URL,
			Content:        content,
			Metadata:       *metadata,
			Error:          fmt.Sprintf("Analysis failed: %v", err),
			ProcessingTime: float64(time.Since(startTime).Nanoseconds()) / 1e6,
		}

		outputJSON, _ := json.MarshalIndent(response, "", "  ")
		return tools.NewToolResult(string(outputJSON), nil, err, nil), nil
	}

	// Create response
	response := &WebFetchResponse{
		URL:            req.URL,
		Title:          metadata.Title,
		Content:        content,
		Analysis:       analysis,
		Metadata:       *metadata,
		ProcessingTime: float64(time.Since(startTime).Nanoseconds()) / 1e6,
		FromCache:      false,
	}

	// Cache the response for 15 minutes
	t.cache.Set(cacheKey, response, 15*time.Minute)

	outputJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal response"), nil), nil
	}

	resultMetadata := map[string]string{
		"url":             req.URL,
		"title":           metadata.Title,
		"content_type":    metadata.ContentType,
		"word_count":      fmt.Sprintf("%d", metadata.WordCount),
		"processing_time": fmt.Sprintf("%.2fms", response.ProcessingTime),
		"from_cache":      "false",
	}

	return tools.NewToolResult(string(outputJSON), resultMetadata, nil, nil), nil
}

// fetchContent fetches and processes web content
func (t *WebFetchTool) fetchContent(ctx context.Context, urlStr string) (string, *WebPageMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create HTTP request").
			WithComponent("web_fetch").
			WithOperation("fetchContent")
	}

	req.Header.Set("User-Agent", t.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to fetch URL").
			WithComponent("web_fetch").
			WithOperation("fetchContent").
			WithDetails("url", urlStr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, gerror.Newf(gerror.ErrCodeExternal, "HTTP request failed with status %d", resp.StatusCode).
			WithComponent("web_fetch").
			WithOperation("fetchContent").
			WithDetails("url", urlStr).
			WithDetails("status", resp.StatusCode)
	}

	// Limit body size to prevent abuse
	limitedReader := io.LimitReader(resp.Body, t.maxBodySize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to read response body").
			WithComponent("web_fetch").
			WithOperation("fetchContent")
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return "", nil, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "failed to parse HTML content").
			WithComponent("web_fetch").
			WithOperation("fetchContent")
	}

	// Extract content and metadata
	content := t.extractContent(doc)
	metadata := t.extractMetadata(doc, resp, len(body))

	return content, metadata, nil
}

// extractContent converts HTML to clean text content
func (t *WebFetchTool) extractContent(doc *goquery.Document) string {
	// Remove unwanted elements
	doc.Find("script, style, nav, header, footer, aside, .advertisement, .ads, .social-share").Remove()

	var content strings.Builder

	// Extract title
	title := doc.Find("title").Text()
	if title != "" {
		content.WriteString("# " + strings.TrimSpace(title) + "\n\n")
	}

	// Extract main content
	mainSelectors := []string{
		"main", "article", "[role='main']", ".content", ".post-content", 
		".entry-content", ".article-content", "#content", ".main-content",
	}

	var mainContent *goquery.Selection
	for _, selector := range mainSelectors {
		if selection := doc.Find(selector); selection.Length() > 0 {
			mainContent = selection.First()
			break
		}
	}

	if mainContent == nil {
		mainContent = doc.Find("body")
	}

	// Convert HTML elements to markdown-like format
	mainContent.Contents().Each(func(i int, s *goquery.Selection) {
		t.processElement(s, &content, 0)
	})

	return strings.TrimSpace(content.String())
}

// processElement recursively processes HTML elements
func (t *WebFetchTool) processElement(s *goquery.Selection, content *strings.Builder, depth int) {
	if s.Length() == 0 {
		return
	}

	// Handle text nodes
	if goquery.NodeName(s) == "#text" {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			content.WriteString(text + " ")
		}
		return
	}

	tagName := goquery.NodeName(s)
	
	switch strings.ToLower(tagName) {
	case "h1":
		content.WriteString("\n\n# " + strings.TrimSpace(s.Text()) + "\n\n")
	case "h2":
		content.WriteString("\n\n## " + strings.TrimSpace(s.Text()) + "\n\n")
	case "h3":
		content.WriteString("\n\n### " + strings.TrimSpace(s.Text()) + "\n\n")
	case "h4":
		content.WriteString("\n\n#### " + strings.TrimSpace(s.Text()) + "\n\n")
	case "h5":
		content.WriteString("\n\n##### " + strings.TrimSpace(s.Text()) + "\n\n")
	case "h6":
		content.WriteString("\n\n###### " + strings.TrimSpace(s.Text()) + "\n\n")
	case "p":
		content.WriteString("\n\n")
		s.Contents().Each(func(i int, child *goquery.Selection) {
			t.processElement(child, content, depth+1)
		})
		content.WriteString("\n\n")
	case "br":
		content.WriteString("\n")
	case "a":
		href, exists := s.Attr("href")
		text := strings.TrimSpace(s.Text())
		if exists && text != "" {
			content.WriteString(fmt.Sprintf("[%s](%s)", text, href))
		} else if text != "" {
			content.WriteString(text)
		}
	case "strong", "b":
		content.WriteString("**" + strings.TrimSpace(s.Text()) + "**")
	case "em", "i":
		content.WriteString("*" + strings.TrimSpace(s.Text()) + "*")
	case "code":
		content.WriteString("`" + strings.TrimSpace(s.Text()) + "`")
	case "pre":
		content.WriteString("\n\n```\n" + strings.TrimSpace(s.Text()) + "\n```\n\n")
	case "ul", "ol":
		content.WriteString("\n")
		s.Find("li").Each(func(i int, li *goquery.Selection) {
			prefix := "- "
			if tagName == "ol" {
				prefix = fmt.Sprintf("%d. ", i+1)
			}
			content.WriteString(prefix + strings.TrimSpace(li.Text()) + "\n")
		})
		content.WriteString("\n")
	case "blockquote":
		lines := strings.Split(strings.TrimSpace(s.Text()), "\n")
		for _, line := range lines {
			content.WriteString("> " + strings.TrimSpace(line) + "\n")
		}
		content.WriteString("\n")
	default:
		// For other elements, just process their children
		s.Contents().Each(func(i int, child *goquery.Selection) {
			t.processElement(child, content, depth+1)
		})
	}
}

// extractMetadata extracts metadata from the HTML document and HTTP response
func (t *WebFetchTool) extractMetadata(doc *goquery.Document, resp *http.Response, contentLength int) *WebPageMetadata {
	metadata := &WebPageMetadata{
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: contentLength,
		LastModified:  resp.Header.Get("Last-Modified"),
		StatusCode:    resp.StatusCode,
		Headers:       make(map[string]string),
	}

	// Extract basic headers
	for key, values := range resp.Header {
		if len(values) > 0 {
			metadata.Headers[key] = values[0]
		}
	}

	// Extract title
	metadata.Title = strings.TrimSpace(doc.Find("title").Text())

	// Extract meta description
	if desc, exists := doc.Find("meta[name='description']").Attr("content"); exists {
		metadata.Description = strings.TrimSpace(desc)
	}

	// Extract meta keywords
	if keywords, exists := doc.Find("meta[name='keywords']").Attr("content"); exists {
		metadata.Keywords = strings.TrimSpace(keywords)
	}

	// Extract author
	if author, exists := doc.Find("meta[name='author']").Attr("content"); exists {
		metadata.Author = strings.TrimSpace(author)
	}

	// Extract language
	if lang, exists := doc.Find("html").Attr("lang"); exists {
		metadata.Language = strings.TrimSpace(lang)
	}

	// Extract links
	var links []string
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists && href != "" {
			links = append(links, href)
		}
	})
	metadata.Links = links

	// Extract images
	var images []string
	doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && src != "" {
			images = append(images, src)
		}
	})
	metadata.Images = images

	// Calculate word count and reading time
	text := doc.Find("body").Text()
	words := strings.Fields(text)
	metadata.WordCount = len(words)
	metadata.ReadingTimeMin = (len(words) + 199) / 200 // ~200 words per minute

	return metadata
}

// analyzeContent uses AI to analyze the fetched content
func (t *WebFetchTool) analyzeContent(ctx context.Context, content, prompt string) (string, error) {
	if t.aiProvider == nil {
		return "", gerror.New(gerror.ErrCodeConfiguration, "AI provider not configured", nil).
			WithComponent("web_fetch").
			WithOperation("analyzeContent")
	}

	// Prepare the analysis prompt
	analysisPrompt := fmt.Sprintf(
		"Please analyze the following web content based on this request: %s\n\nWeb Content:\n%s",
		prompt, content,
	)

	// Limit content length to avoid token limits
	if len(analysisPrompt) > 12000 { // Leave room for response
		truncatedContent := content[:8000] + "\n\n[Content truncated due to length...]"
		analysisPrompt = fmt.Sprintf(
			"Please analyze the following web content based on this request: %s\n\nWeb Content:\n%s",
			prompt, truncatedContent,
		)
	}

	// Create chat request
	chatReq := interfaces.ChatRequest{
		Model: "gpt-3.5-turbo", // Default model, provider will use their best available
		Messages: []interfaces.ChatMessage{
			{
				Role:    "user",
				Content: analysisPrompt,
			},
		},
		MaxTokens:   1000,
		Temperature: 0.3, // Lower temperature for more focused analysis
	}

	// Get analysis from AI
	response, err := t.aiProvider.ChatCompletion(ctx, chatReq)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeExternal, "failed to analyze content with AI").
			WithComponent("web_fetch").
			WithOperation("analyzeContent")
	}

	if len(response.Choices) == 0 {
		return "", gerror.New(gerror.ErrCodeExternal, "AI provider returned no response choices", nil).
			WithComponent("web_fetch").
			WithOperation("analyzeContent")
	}

	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}

// generateCacheKey generates a cache key from URL and prompt
func (t *WebFetchTool) generateCacheKey(url, prompt string) string {
	combined := url + "|" + prompt
	hash := md5.Sum([]byte(combined))
	return fmt.Sprintf("%x", hash)
}