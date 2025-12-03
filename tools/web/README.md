# Guild Framework Web Tools

This package provides web-related tools for the Guild Framework, including web search and content fetching capabilities.

## Tools Implemented

### 1. WebSearch Tool (`web_search`)

A comprehensive web search tool that integrates with multiple search backends:

**Features:**

- **Multiple Search Engines**: Supports Google Custom Search API and DuckDuckGo Instant Answer API
- **Domain Filtering**: Allows filtering results by allowed/blocked domains
- **Configurable Results**: Control maximum number of results (1-50)
- **Language Support**: Specify preferred language for results
- **Safe Search**: Configurable safe search filtering
- **Automatic Fallback**: Falls back to DuckDuckGo if Google API credentials are not available

**Usage:**

```json
{
  "query": "artificial intelligence latest developments",
  "max_results": 10,
  "allowed_domains": ["arxiv.org", "github.com"],
  "blocked_domains": ["spam.com"],
  "language": "en",
  "safe_search": "moderate"
}
```

**Configuration:**

- Set `GOOGLE_SEARCH_API_KEY` and `GOOGLE_SEARCH_ENGINE_ID` environment variables to use Google Custom Search
- Without these, the tool automatically uses DuckDuckGo (no API key required)

### 2. WebFetch Tool (`web_fetch`)

An intelligent web content fetching and analysis tool with AI-powered processing:

**Features:**

- **Content Extraction**: Converts HTML to clean, structured text
- **AI Analysis**: Uses LLM providers to analyze content based on custom prompts
- **Metadata Extraction**: Extracts comprehensive page metadata (title, description, author, etc.)
- **Caching**: 15-minute cache for faster repeated requests
- **Content Processing**: Removes unwanted elements (ads, navigation, scripts)
- **Reading Time Estimation**: Calculates estimated reading time
- **Link and Image Extraction**: Catalogs all links and images on the page

**Usage:**

```json
{
  "url": "https://example.com/article",
  "prompt": "Summarize the main points and key takeaways from this article"
}
```

**Response includes:**

- Original URL and extracted title
- Clean text content in markdown format
- AI-powered analysis based on the prompt
- Comprehensive metadata (word count, reading time, links, images, etc.)
- Processing time and cache status

## Installation & Registration

### Basic Registration

```go
import "github.com/guild-ventures/guild-core/tools/web"

// Register with basic tool registry
toolRegistry := tools.NewToolRegistry()
err := web.RegisterWebTools(toolRegistry, aiProvider)

// Register with cost-aware registry
registry := registry.NewToolRegistry()
err := web.RegisterWebToolsWithRegistry(registry, aiProvider)
```

### Individual Tool Creation

```go
// Create WebSearch tool
searchTool := web.NewWebSearchTool()

// Create WebFetch tool (requires AI provider)
fetchTool := web.NewWebFetchTool(aiProvider)
```

## Cost Information

The tools are registered with the following cost levels in the Guild framework:

- **WebSearch**: Cost Level 1 (Low) - Basic web search functionality
- **WebFetch**: Cost Level 2 (Medium) - Involves AI analysis of content

## Dependencies

- **HTTP Client**: Built-in Go `net/http`
- **HTML Parsing**: `github.com/PuerkitoBio/goquery` (for WebFetch)
- **AI Provider**: Guild's provider interface (for WebFetch)
- **Error Handling**: Guild's `gerror` package
- **External APIs**:
  - Google Custom Search API (optional)
  - DuckDuckGo Instant Answer API

## Architecture

Both tools follow Guild framework patterns:

- **Interface Compliance**: Implement the standard `tools.Tool` interface
- **Error Handling**: Use structured Guild errors with proper categorization
- **Logging**: Include component and operation context for observability
- **Testing**: Comprehensive test suites with mocks and benchmarks
- **Configuration**: Environment-based configuration for external services

## Testing

Run tests with:

```bash
go test ./tools/web/...
```

The test suite includes:

- Interface compliance tests
- Input validation tests
- Mock server tests for HTTP interactions
- Mock AI provider tests
- Domain filtering tests
- Content extraction tests
- Cache functionality tests
- Error handling tests
- Performance benchmarks

## Security Considerations

- **Input Validation**: All inputs are validated and sanitized
- **Content Size Limits**: WebFetch limits response size to prevent abuse (10MB max)
- **Timeout Protection**: All HTTP requests have configurable timeouts
- **Domain Filtering**: Supports blocking malicious or unwanted domains
- **Safe Search**: Configurable content filtering for search results

## Performance

- **Caching**: WebFetch includes intelligent caching to reduce API calls
- **Concurrent Safe**: All operations are safe for concurrent use
- **Memory Efficient**: Streaming and size-limited content processing
- **Timeout Handling**: Proper context-aware timeout handling

## Examples

See `example_test.go` for comprehensive usage examples and integration patterns.

## Contributing

When contributing to the web tools:

1. Follow Guild framework patterns and interfaces
2. Include comprehensive tests with mocks
3. Use structured error handling with proper error codes
4. Add benchmarks for performance-critical operations
5. Update documentation for new features

## Future Enhancements

Potential future improvements:

- Additional search engine backends (Bing, Yahoo, etc.)
- Enhanced content extraction for specific site types
- Bulk URL processing capabilities
- Advanced caching strategies
- Rate limiting for external API calls
- Content summarization without requiring AI prompts
