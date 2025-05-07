# 🧐 Guild Research Corpus System

Guild Feature Spec
Created: 2025-05-07

## 🌟 Purpose

Enable Guild agents to store and organize research findings, summaries, and generated insights in a structured, human-navigable format. This corpus should be explorable via plain text editors (e.g. Neovim), note-taking apps (e.g. Obsidian), or a simple local web viewer.

---

## 🧹 Responsibilities

- Accept `CorpusDoc` objects from tools (e.g. YouTube ingestion, web scraper, code summaries).
- Store content in a persistent, readable Markdown format.
- Link related documents using:

  - Obsidian-style `[[wikilinks]]`
  - Consistent naming and directory hierarchy

- Track user exploration of documents
- Restrict agent access to corpus unless explicitly specified in task objective
- Block agent tasks that depend on a corpus until it is built
- Ensure all documents include parsable metadata (e.g. `guild1:agent1`)
- Enforce user-defined max corpus size and location
- Prevent writes that would exceed disk usage limits

---

## 📁 Directory Layout (Example)

```
guild_memory/corpus/
├── ai/
│   └── open-weights.md
├── youtube/
│   └── future-of-llms.md
├── devtools/
│   └── llama-cpp-summary.md
├── _graph/
│   └── links.json      # Link map for graph navigation
├── _viewlog/
│   └── user_activity.json  # Tracks user document views
```

---

## 📝 Markdown Format

Each file should follow this format:

```markdown
---
title: "The Future of LLMs"
source: "YouTube"
tags: ["llm", "open weights", "research"]
created: 2025-05-07
author: guild1:agent1
---

# The Future of LLMs

## 🔗 Source

https://youtube.com/watch?v=abc123

## 📌 Summary

- Open weight models are accelerating.
- Closed APIs are facing pressure from performance parity.
- [[open-weights-vs-closed]] [[mistral-roadmap]]

## 📜 Transcript / Notes

[...content...]
```

---

## 🔧 Go Package: `pkg/corpus`

```go
type CorpusDoc struct {
    Title    string
    Source   string
    Tags     []string
    Body     string
    Links    []string
    GuildID  string
    AgentID  string
}

type Config struct {
    Location   string
    MaxSizeMB  int
}

func Save(doc CorpusDoc, cfg Config) error
func Autolink(doc *CorpusDoc) error
func TrackUserView(user, docPath string) error
```

The `Save()` function must:

- Check total corpus size against `MaxSizeMB`
- Check free disk space
- Compare document delta before writing

---

## 🔄 Integration with Guild

- Agent tasks specify if they require corpus access
- Corpus-linked tasks are blocked until corpus is generated
- Agents only read from corpus if required info is not in memory
- Users can explore and improve corpus content via command interface
- Agent additions must include `guildID:agentID` in metadata

---

## 🧪 Future Enhancements

- Embedding-based auto-linking for semantically related docs
- Export full Obsidian-compatible vault
- Git-style history tracking for documents
- Graph visualization via web or Obsidian plug-in
- Search, filter, and navigation tools (local and web-based)

---

## Example graph package for obsidian style graph views

## pkg/corpus/graph.go

```go
package corpus

import (
 "encoding/json"
 "errors"
 "io/fs"
 "os"
 "path/filepath"
 "regexp"
 "strings"
)

type Graph struct {
 Nodes map[string][]string `json:"nodes"` // map[doc] -> list of linked docs
}

func BuildGraph(corpusDir, outputPath string) error {
 graph := Graph{Nodes: make(map[string][]string)}

 err := filepath.WalkDir(corpusDir, func(path string, d fs.DirEntry, err error) error {
  if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
   return nil
  }

  content, err := os.ReadFile(path)
  if err != nil {
   return err
  }

  docName := strings.TrimSuffix(filepath.Base(path), ".md")
  links := extractLinks(string(content))
  graph.Nodes[docName] = links

  return nil
 })
 if err != nil {
  return err
 }

 out, err := json.MarshalIndent(graph, "", "  ")
 if err != nil {
  return err
 }
 return os.WriteFile(outputPath, out, 0644)
}

func extractLinks(content string) []string {
 re := regexp.MustCompile(`$begin:math:display$\\[([^\\[$end:math:display$]+)\]\]`)
 matches := re.FindAllStringSubmatch(content, -1)

 unique := map[string]bool{}
 for _, m := range matches {
  unique[m[1]] = true
 }

 links := make([]string, 0, len(unique))
 for link := range unique {
  links = append(links, link)
 }
 return links
}
```
