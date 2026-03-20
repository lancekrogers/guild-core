// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/tools"
)

// WebSearchTool implements web search functionality with multiple backends
type WebSearchTool struct {
	*tools.BaseTool
	client       *http.Client
	apiKey       string
	searchEngine string
	userAgent    string
}

// WebSearchRequest represents the input parameters for web search
type WebSearchRequest struct {
	Query          string   `json:"query"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
	MaxResults     int      `json:"max_results,omitempty"`
	Language       string   `json:"language,omitempty"`
	SafeSearch     string   `json:"safe_search,omitempty"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Domain  string `json:"domain"`
}

// WebSearchResponse represents the search results
type WebSearchResponse struct {
	Query       string         `json:"query"`
	Results     []SearchResult `json:"results"`
	TotalCount  int            `json:"total_count"`
	SearchTime  float64        `json:"search_time_ms"`
	Engine      string         `json:"engine"`
	FilteredOut int            `json:"filtered_out"`
}

// GoogleSearchResponse represents Google Custom Search API response
type GoogleSearchResponse struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
	SearchInformation struct {
		SearchTime       float64 `json:"searchTime"`
		TotalResults     string  `json:"totalResults"`
		FormattedResults string  `json:"formattedTotalResults"`
	} `json:"searchInformation"`
}

// DuckDuckGoResponse represents DuckDuckGo Instant Answer API response
type DuckDuckGoResponse struct {
	Abstract      string `json:"Abstract"`
	AbstractURL   string `json:"AbstractURL"`
	AbstractText  string `json:"AbstractText"`
	RelatedTopics []struct {
		Text     string `json:"Text"`
		FirstURL string `json:"FirstURL"`
	} `json:"RelatedTopics"`
	Results []struct {
		Text     string `json:"Text"`
		FirstURL string `json:"FirstURL"`
	} `json:"Results"`
}

// NewWebSearchTool creates a new web search tool
func NewWebSearchTool() *WebSearchTool {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query to execute",
			},
			"allowed_domains": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Only include results from these domains (optional)",
			},
			"blocked_domains": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Exclude results from these domains (optional)",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 10, max: 50)",
				"minimum":     1,
				"maximum":     50,
			},
			"language": map[string]interface{}{
				"type":        "string",
				"description": "Language preference for results (e.g., 'en', 'es', 'fr')",
			},
			"safe_search": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"strict", "moderate", "off"},
				"description": "Safe search filter level",
			},
		},
		"required": []string{"query"},
	}

	examples := []string{
		`{"query": "artificial intelligence latest developments"}`,
		`{"query": "python tutorial", "max_results": 5}`,
		`{"query": "machine learning", "allowed_domains": ["arxiv.org", "github.com"]}`,
		`{"query": "news today", "blocked_domains": ["example.com"], "safe_search": "strict"}`,
	}

	baseTool := tools.NewBaseTool(
		"web_search",
		"Search the web using multiple search engines. Supports domain filtering, language preferences, and safe search. Different from the web scraper tool - this performs search queries rather than scraping specific URLs.",
		schema,
		"web",
		false, // No auth required for basic search
		examples,
	)

	return &WebSearchTool{
		BaseTool: baseTool,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		searchEngine: determineSearchEngine(),
		userAgent:    "Guild-Framework-WebSearch/1.0",
	}
}

// determineSearchEngine determines which search engine to use based on environment
func determineSearchEngine() string {
	if os.Getenv("GOOGLE_SEARCH_API_KEY") != "" && os.Getenv("GOOGLE_SEARCH_ENGINE_ID") != "" {
		return "google"
	}
	return "duckduckgo" // fallback to DuckDuckGo which doesn't require API key
}

// Execute performs the web search
func (t *WebSearchTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var req WebSearchRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to parse search request"), nil), nil
	}

	// Validate and set defaults
	if req.Query == "" {
		return tools.NewToolResult("", nil, gerror.New(gerror.ErrCodeInvalidInput, "query is required", nil), nil), nil
	}

	if req.MaxResults <= 0 {
		req.MaxResults = 10
	}
	if req.MaxResults > 50 {
		req.MaxResults = 50
	}

	if req.SafeSearch == "" {
		req.SafeSearch = "moderate"
	}

	startTime := time.Now()

	// Perform search based on configured engine
	var response *WebSearchResponse
	var err error

	switch t.searchEngine {
	case "google":
		response, err = t.searchGoogle(ctx, req)
	case "duckduckgo":
		response, err = t.searchDuckDuckGo(ctx, req)
	default:
		return tools.NewToolResult("", nil, gerror.Newf(gerror.ErrCodeInternal, "unsupported search engine: %s", t.searchEngine), nil), nil
	}

	if err != nil {
		return tools.NewToolResult("", nil, err, nil), nil
	}

	response.SearchTime = float64(time.Since(startTime).Nanoseconds()) / 1e6 // Convert to milliseconds

	// Apply domain filtering
	response = t.applyDomainFiltering(response, req.AllowedDomains, req.BlockedDomains)

	// Convert to JSON for output
	outputJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return tools.NewToolResult("", nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal search results"), nil), nil
	}

	metadata := map[string]string{
		"engine":       response.Engine,
		"query":        response.Query,
		"result_count": fmt.Sprintf("%d", len(response.Results)),
		"search_time":  fmt.Sprintf("%.2fms", response.SearchTime),
	}

	return tools.NewToolResult(string(outputJSON), metadata, nil, nil), nil
}

// searchGoogle performs search using Google Custom Search API
func (t *WebSearchTool) searchGoogle(ctx context.Context, req WebSearchRequest) (*WebSearchResponse, error) {
	apiKey := os.Getenv("GOOGLE_SEARCH_API_KEY")
	engineID := os.Getenv("GOOGLE_SEARCH_ENGINE_ID")

	if apiKey == "" || engineID == "" {
		return nil, gerror.New(gerror.ErrCodeConfiguration, "Google Search API key or engine ID not configured", nil).
			WithComponent("web_search").
			WithOperation("searchGoogle")
	}

	baseURL := "https://www.googleapis.com/customsearch/v1"
	params := url.Values{}
	params.Set("key", apiKey)
	params.Set("cx", engineID)
	params.Set("q", req.Query)
	params.Set("num", fmt.Sprintf("%d", req.MaxResults))

	if req.Language != "" {
		params.Set("lr", "lang_"+req.Language)
	}

	if req.SafeSearch != "off" {
		if req.SafeSearch == "strict" {
			params.Set("safe", "active")
		}
	}

	searchURL := baseURL + "?" + params.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create search request").
			WithComponent("web_search").
			WithOperation("searchGoogle")
	}

	httpReq.Header.Set("User-Agent", t.userAgent)

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to execute search request").
			WithComponent("web_search").
			WithOperation("searchGoogle")
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, gerror.Newf(gerror.ErrCodeExternal, "Google Search API returned status %d: %s", resp.StatusCode, string(body)).
			WithComponent("web_search").
			WithOperation("searchGoogle")
	}

	var googleResp GoogleSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "failed to decode Google Search response").
			WithComponent("web_search").
			WithOperation("searchGoogle")
	}

	// Convert to our format
	results := make([]SearchResult, 0, len(googleResp.Items))
	for _, item := range googleResp.Items {
		parsedURL, _ := url.Parse(item.Link)
		domain := ""
		if parsedURL != nil {
			domain = parsedURL.Hostname()
		}

		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
			Domain:  domain,
		})
	}

	return &WebSearchResponse{
		Query:      req.Query,
		Results:    results,
		TotalCount: len(results),
		Engine:     "google",
	}, nil
}

// searchDuckDuckGo performs search using DuckDuckGo Instant Answer API
func (t *WebSearchTool) searchDuckDuckGo(ctx context.Context, req WebSearchRequest) (*WebSearchResponse, error) {
	baseURL := "https://api.duckduckgo.com/"
	params := url.Values{}
	params.Set("q", req.Query)
	params.Set("format", "json")
	params.Set("no_redirect", "1")
	params.Set("no_html", "1")
	params.Set("skip_disambig", "1")

	searchURL := baseURL + "?" + params.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create search request").
			WithComponent("web_search").
			WithOperation("searchDuckDuckGo")
	}

	httpReq.Header.Set("User-Agent", t.userAgent)

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to execute search request").
			WithComponent("web_search").
			WithOperation("searchDuckDuckGo")
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, gerror.Newf(gerror.ErrCodeExternal, "DuckDuckGo API returned status %d: %s", resp.StatusCode, string(body)).
			WithComponent("web_search").
			WithOperation("searchDuckDuckGo")
	}

	// Read the entire response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to read DuckDuckGo response").
			WithComponent("web_search").
			WithOperation("searchDuckDuckGo")
	}

	// Check if response is empty
	if len(body) == 0 {
		// Return empty results for queries with no matches
		return &WebSearchResponse{
			Query:      req.Query,
			Results:    []SearchResult{},
			TotalCount: 0,
			Engine:     "duckduckgo",
		}, nil
	}

	var ddgResp DuckDuckGoResponse
	if err := json.Unmarshal(body, &ddgResp); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "failed to decode DuckDuckGo response").
			WithComponent("web_search").
			WithOperation("searchDuckDuckGo")
	}

	// Convert to our format
	var results []SearchResult

	// Add abstract if available
	if ddgResp.Abstract != "" && ddgResp.AbstractURL != "" {
		parsedURL, _ := url.Parse(ddgResp.AbstractURL)
		domain := ""
		if parsedURL != nil {
			domain = parsedURL.Hostname()
		}

		results = append(results, SearchResult{
			Title:   extractTitle(ddgResp.Abstract),
			URL:     ddgResp.AbstractURL,
			Snippet: ddgResp.Abstract,
			Domain:  domain,
		})
	}

	// Add direct results
	for _, result := range ddgResp.Results {
		if result.FirstURL != "" {
			parsedURL, _ := url.Parse(result.FirstURL)
			domain := ""
			if parsedURL != nil {
				domain = parsedURL.Hostname()
			}

			results = append(results, SearchResult{
				Title:   extractTitle(result.Text),
				URL:     result.FirstURL,
				Snippet: result.Text,
				Domain:  domain,
			})
		}
	}

	// Add related topics
	for i, topic := range ddgResp.RelatedTopics {
		if i >= req.MaxResults-len(results) {
			break
		}
		if topic.FirstURL != "" {
			parsedURL, _ := url.Parse(topic.FirstURL)
			domain := ""
			if parsedURL != nil {
				domain = parsedURL.Hostname()
			}

			results = append(results, SearchResult{
				Title:   extractTitle(topic.Text),
				URL:     topic.FirstURL,
				Snippet: topic.Text,
				Domain:  domain,
			})
		}
	}

	// Limit results
	if len(results) > req.MaxResults {
		results = results[:req.MaxResults]
	}

	return &WebSearchResponse{
		Query:      req.Query,
		Results:    results,
		TotalCount: len(results),
		Engine:     "duckduckgo",
	}, nil
}

// applyDomainFiltering filters results based on allowed and blocked domains
func (t *WebSearchTool) applyDomainFiltering(response *WebSearchResponse, allowedDomains, blockedDomains []string) *WebSearchResponse {
	if len(allowedDomains) == 0 && len(blockedDomains) == 0 {
		return response
	}

	var filteredResults []SearchResult
	filteredCount := 0

	for _, result := range response.Results {
		domain := result.Domain

		// Check blocked domains first
		if len(blockedDomains) > 0 {
			blocked := false
			for _, blockedDomain := range blockedDomains {
				if strings.Contains(domain, blockedDomain) || domain == blockedDomain {
					blocked = true
					break
				}
			}
			if blocked {
				filteredCount++
				continue
			}
		}

		// Check allowed domains
		if len(allowedDomains) > 0 {
			allowed := false
			for _, allowedDomain := range allowedDomains {
				if strings.Contains(domain, allowedDomain) || domain == allowedDomain {
					allowed = true
					break
				}
			}
			if !allowed {
				filteredCount++
				continue
			}
		}

		filteredResults = append(filteredResults, result)
	}

	response.Results = filteredResults
	response.FilteredOut = filteredCount
	return response
}

// extractTitle extracts a title from text, taking the first part before a dash or period
func extractTitle(text string) string {
	// Trim whitespace first
	text = strings.TrimSpace(text)

	if len(text) == 0 {
		return "Untitled"
	}

	// Try to extract title from text
	parts := strings.Split(text, " - ")
	if len(parts) > 1 {
		title := strings.TrimSpace(parts[0])
		if title != "" {
			return title
		}
	}

	parts = strings.Split(text, ". ")
	if len(parts) > 1 {
		title := strings.TrimSpace(parts[0])
		if len(title) > 5 { // Ensure it's a meaningful title
			return title
		}
	}

	// Fallback to first 100 characters
	if len(text) > 100 {
		return text[:100] + "..."
	}

	return text
}
