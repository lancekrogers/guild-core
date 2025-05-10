## Implement Corpus System

@context
@lore_conventions

This command guides you through implementing the Guild corpus system, which stores and organizes research findings, summaries, and generated insights in a structured, human-navigable format.

### 1. Check Existing Implementation

```bash
# Check for existing corpus-related files
find . -type f -name "*.go" | xargs grep -l "corpus" | grep -v "_test.go"

# Check for existing Bubble Tea UI components
find ./pkg -type f -name "*.go" | xargs grep -l "bubbletea\|tea\." | grep -v "_test.go"
```

### 2. Review Specification Documents

```
cat specs/features/corpus_system.md
```

### 3. Implementation Process

#### A. Core Corpus Package

1. **Define Models**

   - Create `pkg/corpus/models.go` with:

     ```go
     // CorpusDoc represents a document in the research corpus
     type CorpusDoc struct {
         Title    string
         Source   string
         Tags     []string
         Body     string
         Links    []string
         GuildID  string
         AgentID  string
     }

     // Config represents corpus configuration
     type Config struct {
         Location   string
         MaxSizeMB  int
     }
     ```

2. **Implement Storage**

   - Create `pkg/corpus/storage.go` with:

     ```go
     // Save stores a CorpusDoc to the filesystem
     func Save(doc CorpusDoc, cfg Config) error

     // Load retrieves a CorpusDoc from the filesystem
     func Load(path string) (*CorpusDoc, error)

     // List returns all corpus documents
     func List(cfg Config) ([]string, error)
     ```

3. **Add Link Management**

   - Create `pkg/corpus/links.go` with:

     ```go
     // Autolink finds potential links in a document
     func Autolink(doc *CorpusDoc) error

     // ExtractLinks finds wikilinks in content
     func ExtractLinks(content string) []string
     ```

4. **Implement Graph Building**

   - Create `pkg/corpus/graph.go` with:

     ```go
     // Graph represents document connections
     type Graph struct {
         Nodes map[string][]string
     }

     // BuildGraph creates a graph from corpus documents
     func BuildGraph(corpusDir, outputPath string) error
     ```

5. **Add User Activity Tracking**

   - Create `pkg/corpus/activity.go` with:

     ```go
     // TrackUserView records document views
     func TrackUserView(user, docPath string) error

     // GetUserActivity retrieves user viewing history
     func GetUserActivity(user string) ([]string, error)
     ```

#### B. UI Components

1. **Create Base UI Components**

   - Create `pkg/ui/corpus/model.go` with:

     ```go
     // Model represents corpus UI state
     type Model struct {
         docs         []corpus.CorpusDoc
         currentDoc   *corpus.CorpusDoc
         mode         string // "list", "view", "search"
         width, height int
         list         list.Model
         viewport     viewport.Model
         input        textinput.Model
         help         help.Model
     }

     // NewModel creates a new corpus UI model
     func NewModel() Model
     ```

2. **Implement View Logic**

   - Create `pkg/ui/corpus/view.go` with rendering functions:

     ```go
     // View renders the current UI state
     func (m Model) View() string

     // renderList displays the document list
     func (m Model) renderList() string

     // renderDocument displays a single document
     func (m Model) renderDocument() string

     // renderSearch displays search interface
     func (m Model) renderSearch() string
     ```

3. **Add Event Handling**

   - Create `pkg/ui/corpus/update.go` with event handlers:

     ```go
     // Update handles messages and events
     func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)

     // handleKeyMsg processes keyboard input
     func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd)
     ```

4. **Create Graph Visualization**

   - Create `pkg/ui/corpus/graph.go` with:

     ```go
     // GraphModel represents graph visualization state
     type GraphModel struct {
         graph       corpus.Graph
         nodes       []Node
         edges       []Edge
         selected    string
         zoom        float64
         offsetX, offsetY int
         width, height int
     }

     // RenderGraph draws the graph visualization
     func (m GraphModel) RenderGraph() string
     ```

#### C. CLI Commands

1. **Implement Corpus Command**

   - Create `cmd/guild/corpus_cmd.go` with:
     ```go
     // corpusCmd represents the corpus command
     var corpusCmd = &cobra.Command{
         Use:   "corpus",
         Short: "Manage research corpus",
         Long:  `View, search, and manage the Guild research corpus.`,
         Run: func(cmd *cobra.Command, args []string) {
             // Launch corpus browser UI
             model := corpus_ui.NewModel()
             p := tea.NewProgram(model, tea.WithAltScreen())
             if _, err := p.Run(); err != nil {
                 fmt.Println("Error running corpus UI:", err)
                 os.Exit(1)
             }
         },
     }
     ```

2. **Add View Subcommand**

   - Add to `corpus_cmd.go`:
     ```go
     var corpusViewCmd = &cobra.Command{
         Use:   "view [docPath]",
         Short: "View corpus document",
         Run: func(cmd *cobra.Command, args []string) {
             // Implementation...
         },
     }
     ```

3. **Add Search Subcommand**

   - Add to `corpus_cmd.go`:
     ```go
     var corpusSearchCmd = &cobra.Command{
         Use:   "search <query>",
         Short: "Search corpus content",
         Run: func(cmd *cobra.Command, args []string) {
             // Implementation...
         },
     }
     ```

4. **Add Graph Subcommand**
   - Add to `corpus_cmd.go`:
     ```go
     var corpusGraphCmd = &cobra.Command{
         Use:   "graph",
         Short: "Show corpus graph visualization",
         Run: func(cmd *cobra.Command, args []string) {
             // Implementation...
         },
     }
     ```

#### D. Config Integration

1. **Add Corpus Configuration**

   - Update `pkg/config/config.go` with corpus settings:
     ```go
     type CorpusConfig struct {
         Location  string `yaml:"location"`
         MaxSizeMB int    `yaml:"max_size_mb"`
     }
     ```

2. **Add Configuration Command**
   - Add to `cmd/guild/config_cmd.go`:
     ```go
     var configCorpusCmd = &cobra.Command{
         Use:   "corpus",
         Short: "Configure corpus settings",
         Run: func(cmd *cobra.Command, args []string) {
             // Implementation...
         },
     }
     ```

#### E. Tool Integration

1. **Add Tool Support**
   - Update tools to support corpus:
     ```go
     // tools/scraper/scraper.go
     func (s *Scraper) SaveToCorpus(result *ScrapeResult) error {
         doc := corpus.CorpusDoc{
             Title:   result.Title,
             Source:  result.URL,
             Tags:    result.Tags,
             Body:    result.Content,
             GuildID: s.GuildID,
             AgentID: s.AgentID,
         }

         return corpus.Save(doc, s.CorpusConfig)
     }
     ```

### 4. Testing Strategy

1. **Unit Tests**

   - Create `pkg/corpus/storage_test.go`:

     ```go
     func TestSave(t *testing.T) {
         // Test saving documents
     }

     func TestLoad(t *testing.T) {
         // Test loading documents
     }
     ```

   - Create `pkg/corpus/graph_test.go`:

     ```go
     func TestBuildGraph(t *testing.T) {
         // Test graph generation
     }

     func TestExtractLinks(t *testing.T) {
         // Test link extraction
     }
     ```

2. **UI Tests**

   - Create `pkg/ui/corpus/model_test.go`:
     ```go
     func TestUpdate(t *testing.T) {
         // Test UI state transitions
     }
     ```

3. **Integration Tests**
   - Create `test/integration/corpus_test.go`:
     ```go
     func TestCorpusWorkflow(t *testing.T) {
         // Test end-to-end workflow
     }
     ```

### 5. Implementation Tips

1. **Markdown Handling**

   - Use a robust markdown parser like Goldmark
   - Preserve YAML frontmatter for metadata
   - Handle wikilinks properly for navigation

2. **Size Management**

   - Check document size before saving
   - Implement incremental updates where possible
   - Use efficient storage for large corpora

3. **UI Performance**

   - Load documents on demand in the UI
   - Implement pagination for large document lists
   - Cache rendered markdown for performance

4. **Graph Visualization**
   - Use ASCII/ANSI art for graph in terminal
   - Consider force-directed layout for node positioning
   - Enable filtering by tag or relevance

### Key Deliverables

1. Complete corpus package in `pkg/corpus/`
2. Bubble Tea UI components in `pkg/ui/corpus/`
3. CLI commands in `cmd/guild/corpus_cmd.go`
4. Configuration integration
5. Tool support for corpus additions
6. Comprehensive tests

### Corpus Directory Structure

The implementation should create this structure:

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

### Document Format

Each document should follow this format:

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
