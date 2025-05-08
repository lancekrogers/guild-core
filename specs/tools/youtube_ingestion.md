# 📺 YouTube Transcript Ingestion + Summarization

Guild Feature Spec  
Created: 2025-05-07

## 🎯 Purpose

Enable agents to ingest YouTube videos as knowledge sources via automatic transcript retrieval and summarization, producing structured raw data suitable for research use.

## 🧩 Responsibilities

- Accept one or more YouTube URLs.
- Retrieve available transcripts.
- Summarize transcripts using pluggable LLMs (Ollama, Claude, etc.).
- Return:
  - Raw transcript text
  - Markdown-formatted summary
  - Optional metadata (title, timestamps, etc.)

## 🔁 Workflow

1. Parse and validate YouTube URL
2. Fetch transcript via:
   - YouTube captions API
   - Fallback: `youtube-transcript-api` clone
3. Summarize in chunks using agent LLM tool
4. Return:
   - Markdown-formatted string
   - Struct for corpus storage

## 🔧 Example Output

````md
# Summary - The Future of LLMs

🔗 https://youtube.com/watch?v=abc123

## Key Points

- Open weights models are gaining ground
- Transformer architecture is evolving

## Transcript

[...]

## Package:pkg/youtube

```go
func FetchTranscript(videoID string) (string, error)
func SummarizeTranscript(text string) (string, error)
```
````

Returns: TranscriptDoc struct for corpus insertion
