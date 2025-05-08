# Guild Examples and Templates

This document provides example configurations, use cases, and templates to help you get started with the Guild framework.

## Directory Structure

Following the structure of your existing project:

```
guild-project/
├── ai_docs/                    # Agent knowledge repository
│   ├── api_docs/               # External API documentation
│   ├── architecture/           # System design documentation
│   ├── integration_guides/     # Integration documentation
│   └── patterns/               # Go patterns and best practices
├── cmd/
│   └── guild/                  # CLI implementation
├── examples/                   # Example guild configurations
├── pkg/                        # Core packages
├── specs/                      # Design specifications
└── guild.yaml                  # Main configuration file
```

## Example Configurations

### Simple Single-Agent Configuration

```yaml
# guild.yaml - Simple configuration with one agent
agents:
  - name: assistant
    provider: anthropic
    model: claude-3-sonnet
    tools:
      - file-reader
      - file-writer
      - web-search

guilds:
  - name: simple-assistant
    agents:
      - assistant
    objectives_path: objectives

costs:
  cli_tools:
    default: 0
  api_models:
    claude-3-sonnet: 30
```

### Development Team Configuration

```yaml
# guild.yaml - Development team with multiple specialized agents
agents:
  - name: planner
    provider: anthropic
    model: claude-3-opus
    tools:
      - tree2scaffold
      - search-codebase

  - name: implementer
    provider: ollama
    model: llama3-8b
    tools:
      - make
      - aider
      - file-writer

  - name: reviewer
    provider: anthropic
    model: claude-3-sonnet
    tools:
      - lint
      - test-runner
      - static-analysis

guilds:
  - name: dev-team
    agents:
      - planner
      - implementer
      - reviewer
    manager: planner
    objectives_path: objectives/dev

costs:
  cli_tools:
    default: 0
  local_models:
    llama3-8b: 1
  api_models:
    claude-3-opus: 45
    claude-3-sonnet: 30
```

### Privacy-Focused Configuration

```yaml
# guild.yaml - Offline-only configuration for sensitive data
agents:
  - name: docs-assistant
    provider: ollama
    model: llama3-70b
    tools:
      - file-reader
      - file-writer
      - local-search

  - name: formatter
    provider: ollama
    model: gemma-2b
    tools:
      - format-md
      - lint-docs

guilds:
  - name: legal-assistant
    agents:
      - docs-assistant
      - formatter
    objectives_path: objectives/legal

costs:
  cli_tools:
    default: 0
  local_models:
    llama3-70b: 3
    gemma-2b: 1
  api_models:
    default: 99999 # Effectively disable all API models
```

## Example Use Cases

### 1. Code Generation Guild

This example shows a guild that generates a complete Go web application.

#### Project Structure

```
/code-gen-project
├── guild.yaml
├── objectives/
│   ├── README.md
│   ├── backend/
│   │   ├── api.md
│   │   ├── database.md
│   │   └── server.md
│   └── frontend/
│       ├── overview.md
│       └── components.md
├── ai_docs/
│   ├── architecture/
│   │   └── system_design.md
│   └── api_docs/
│       └── rest_best_practices.md
└── tools/
    └── tree2scaffold/
        └── main.go
```

#### Example Objective File (api.md)

```markdown
# 🧠 Goal

Design and implement a RESTful API for our Go web application.

# 📂 Context

The API will serve as the interface between our frontend and database.
It should follow best practices for REST design and use Go's standard
library or a minimal framework like chi.

# 🔧 Requirements

- Create endpoints for CRUD operations on users and posts
- Implement middleware for authentication
- Use proper error handling and status codes
- Document all endpoints with OpenAPI

# 📌 Tags

- api
- backend
- golang
- rest

# 🔗 Related

- [../backend/database.md](../backend/database.md)
- [../backend/server.md](../backend/server.md)
```

#### Example AI Docs (rest_best_practices.md)

````markdown
# REST API Best Practices for Go

This document outlines the best practices for building RESTful APIs in Go.

## URL Structure

- Use plural nouns for resource collections: `/users`, `/posts`
- Use resource IDs for specific resources: `/users/123`, `/posts/456`
- Use nested resources for relationships: `/users/123/posts`
- Use query parameters for filtering: `/posts?author=123&category=tech`

## HTTP Methods

- GET: Retrieve resources
- POST: Create resources
- PUT: Update resources (full replacement)
- PATCH: Update resources (partial update)
- DELETE: Remove resources

## Status Codes

- 200 OK: Success
- 201 Created: Resource created
- 204 No Content: Success with no response body
- 400 Bad Request: Invalid input
- 401 Unauthorized: Authentication required
- 403 Forbidden: Permission denied
- 404 Not Found: Resource not found
- 500 Internal Server Error: Server error

## Common Patterns

```go
// Resource struct
type User struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

// Handler pattern
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    // Extract ID from URL
    id := chi.URLParam(r, "id")

    // Get user from database
    user, err := db.GetUser(id)
    if err != nil {
        if errors.Is(err, db.ErrNotFound) {
            http.Error(w, "User not found", http.StatusNotFound)
            return
        }
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Return user as JSON
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}
```
````

````

#### Execution

```bash
# Run the code generation guild
guild run code-gen-project

# The guild will:
# 1. Break down the objectives into tasks
# 2. Assign tasks to agents based on their roles
# 3. Generate the code according to the specifications
# 4. Run tests and validation
# 5. Provide a summary of the generated application
````

### 2. Content Creation Guild

This example shows a guild that creates blog content.

#### Project Structure

```
/content-project
├── guild.yaml
├── objectives/
│   ├── README.md
│   ├── blog-posts/
│   │   ├── golang-concurrency.md
│   │   ├── vector-databases.md
│   │   └── llm-frameworks.md
│   └── images/
│       ├── requirements.md
│       └── style-guide.md
├── ai_docs/
│   └── patterns/
│       └── technical_writing.md
└── output/
    ├── posts/
    └── images/
```

#### Example Objective File (golang-concurrency.md)

```markdown
# 🧠 Goal

Create an in-depth blog post explaining Go's concurrency model for intermediate developers.

# 📂 Context

This blog post will be the first in a series about Go programming. It should
explain goroutines, channels, and the "share memory by communicating" philosophy.

# 🔧 Requirements

- Explain goroutines vs. threads
- Provide examples of channel usage
- Compare to other concurrency models
- Include practical examples
- Target length: 1500-2000 words

# 📌 Tags

- golang
- concurrency
- blog-post
- technical

# 🔗 Related

- [../images/requirements.md](../images/requirements.md)
```

#### Execution

```bash
# Run the content creation guild
guild run content-project

# The guild will:
# 1. Create tasks for each blog post
# 2. Generate the content
# 3. Create supplementary images
# 4. Store everything in the output directory
```

### 3. Data Analysis Guild

This example shows a guild that performs data analysis tasks.

#### Project Structure

```
/data-analysis-project
├── guild.yaml
├── objectives/
│   ├── README.md
│   ├── analysis/
│   │   ├── exploratory.md
│   │   ├── statistical.md
│   │   └── visualization.md
│   └── reporting/
│       ├── executive-summary.md
│       └── detailed-report.md
├── ai_docs/
│   └── patterns/
│       └── data_analysis_workflows.md
├── data/
│   ├── sales.csv
│   └── customers.csv
└── output/
    ├── reports/
    └── visualizations/
```

#### Example Objective File (exploratory.md)

```markdown
# 🧠 Goal

Perform exploratory data analysis on the sales and customer datasets.

# 📂 Context

We have two datasets: sales.csv and customers.csv. We need to understand
the basic structure and relationships within this data before proceeding
with detailed analysis.

# 🔧 Requirements

- Load and clean both datasets
- Identify missing values and outliers
- Calculate basic statistics
- Explore relationships between key variables
- Create summary visualizations

# 📌 Tags

- eda
- data-analysis
- python
- pandas

# 🔗 Related

- [../analysis/statistical.md](../analysis/statistical.md)
- [../analysis/visualization.md](../analysis/visualization.md)
```

#### Example Tool Configuration

```yaml
# tools.yaml
- name: pandas-processor
  cmd: "python scripts/pandas_processor.py"
  context_description: "Process CSV data using pandas"
  args:
    input: "{{input_file}}"
    output: "{{output_file}}"

- name: data-visualizer
  cmd: "python scripts/visualizer.py"
  context_description: "Generate data visualizations"
  working_dir: "output/visualizations"
```

#### Execution

```bash
# Run the data analysis guild
guild run data-analysis-project

# The guild will:
# 1. Perform the exploratory analysis
# 2. Run statistical tests
# 3. Generate visualizations
# 4. Create reports
```

## Template Library

### 1. Go API Server Template

```markdown
# 🧠 Goal

Create a Go API server template with standard endpoints and middleware.

# 📂 Context

This template provides a starting point for Go API servers using the chi router.
It includes common middleware, error handling, and basic endpoints.

# 🔧 Requirements

- Chi router configuration
- CORS middleware
- JWT authentication
- Structured logging
- Health check endpoint
- Rate limiting
- Graceful shutdown

# 📌 Tags

- template
- golang
- api
- server

# 📑 Template Components

- `main.go`: Entry point with server configuration
- `routes/routes.go`: Route definitions
- `middleware/auth.go`: Authentication middleware
- `middleware/logging.go`: Logging middleware
- `handlers/health.go`: Health check handler
- `config/config.go`: Configuration loading
```

### 2. Documentation Template

```markdown
# 🧠 Goal

Generate comprehensive documentation for a software project.

# 📂 Context

This template creates a documentation site using Markdown files organized into
sections. It follows best practices for technical documentation.

# 🔧 Requirements

- Project overview
- Installation instructions
- Getting started guide
- API documentation
- Troubleshooting section
- FAQ

# 📌 Tags

- template
- documentation
- markdown
- technical-writing

# 📑 Template Components

- `README.md`: Project overview
- `installation.md`: Setup instructions
- `getting-started.md`: Quick start guide
- `api/`: API documentation
- `troubleshooting.md`: Common issues and solutions
- `faq.md`: Frequently asked questions
```

### 3. Python Data Analysis Template

```markdown
# 🧠 Goal

Create a Python data analysis notebook template.

# 📂 Context

This template provides a structured Jupyter notebook for data analysis tasks,
following best practices for reproducible data science.

# 🔧 Requirements

- Data loading and cleaning section
- Exploratory data analysis
- Statistical testing
- Visualization
- Results interpretation
- Export functionality

# 📌 Tags

- template
- python
- data-analysis
- jupyter

# 📑 Template Components

- `analysis.ipynb`: Main Jupyter notebook
- `requirements.txt`: Python dependencies
- `utils.py`: Helper functions
- `config.yaml`: Configuration settings
```

## CLI Commands and Workflow

### Initializing a New Project

```bash
# Create a new Guild project
guild init my-project

# Create with a specific template
guild init my-api-project --template dev

# Create for offline use only
guild init private-project --offline
```

### Adding Agents and Tools

```bash
# Add a planner agent
guild agent add planner --provider anthropic --model claude-3-opus

# Add a code implementation agent with local model
guild agent add coder --provider ollama --model deepseek-coder-33b --tools aider,tree2scaffold

# Add a reviewer agent with personality
guild agent add reviewer --provider anthropic --model claude-3-sonnet --persona "Senior Go Developer with 10+ years experience"

# Add a custom tool
guild tool add format-md --cmd "pandoc -f markdown -t markdown" --desc "Standardize markdown formatting"
```

### Creating Objectives

```bash
# Add a new objective
guild objective add "Create Login API" --path objectives/backend/login-api.md

# Add with more details
guild objective add "User Authentication System" \
  --path objectives/auth/system.md \
  --description "Design and implement a secure authentication system" \
  --template api-feature
```

### Running a Guild

```bash
# Run a guild with all configured agents
guild run my-project

# Run with specific objective focus
guild run my-project --focus objectives/backend/login-api.md

# Run with a time limit
guild run my-project --timeout 30m

# Run with a cost budget
guild run my-project --budget 100
```

### Monitoring Progress

```bash
# View the Kanban board
guild kanban

# Show task status
guild task list

# View detailed task information
guild task show task-123

# View logs
guild logs
```

### Interacting with Tasks

```bash
# List all tasks
guild task list

# View a specific task
guild task show task-123

# Unblock a task that needs human input
guild task unblock task-123 --input "Use PostgreSQL instead of MySQL"

# Move a task to a different status
guild task move task-123 --status done
```

## Best Practices

### Cost Optimization

Configure costs to prioritize local tools and models:

```yaml
costs:
  cli_tools:
    default: 0

  local_models:
    gemma-2b: 1
    llama3-70b: 3

  api_models:
    openai-gpt-4: 60
    claude-3-opus: 45
    claude-3-sonnet: 30
```

Set a budget for your Guild runs:

```bash
# Run with a cost budget
guild run my-project --budget 100
```

Monitor usage over time:

```bash
# View cost report
guild cost report --period 30d
```

### Objective Writing

Structure your objectives with clear, actionable requirements:

1. **Be specific** - Clearly state what should be accomplished
2. **Provide context** - Explain why this objective matters
3. **List requirements** - Enumerate specific deliverables
4. **Add tags** - Use consistent tagging for searchability
5. **Link related objectives** - Connect dependencies

Example objective template:

```markdown
# 🧠 Goal

[One sentence describing the outcome]

# 📂 Context

[Background information, approximately 2-3 paragraphs]

# 🔧 Requirements

- [Specific requirement 1]
- [Specific requirement 2]
- [...]

# 📌 Tags

- [tag1]
- [tag2]
- [...]

# 🔗 Related

- [./related-objective-1.md](./related-objective-1.md)
- [../other-category/related-objective-2.md](../other-category/related-objective-2.md)
```

### Guild Composition

Design guilds with specialized agents:

1. **Planning agent** - High-capability model for task decomposition

   ```yaml
   - name: planner
     provider: anthropic
     model: claude-3-opus
     tools: [tree2scaffold, search-codebase]
     personality:
       persona: "Senior Software Architect"
       style: "analytical"
       expertise: ["system design", "technical planning"]
   ```

2. **Implementation agents** - May use local models for specific tasks

   ```yaml
   - name: backend-dev
     provider: ollama
     model: deepseek-coder-33b
     tools: [aider, go-compiler]
   ```

3. **Reviewers** - Quality control and validation

   ```yaml
   - name: code-reviewer
     provider: anthropic
     model: claude-3-sonnet
     tools: [lint, test-runner, security-scan]
     personality:
       persona: "Senior Code Reviewer"
       style: "critical but constructive"
       expertise: ["security", "performance", "code quality"]
   ```

4. **Manager** - Coordinates and assigns tasks
   ```yaml
   guilds:
     - name: dev-team
       agents: [planner, backend-dev, frontend-dev, code-reviewer]
       manager: planner
   ```

### Tool Integration

Create tools for repetitive tasks:

1. **Identify patterns** - Look for repetitive LLM prompts
2. **Implement as CLI** - Create simple command-line tools
3. **Document usage** - Provide clear descriptions
4. **Test thoroughly** - Ensure tools work consistently

Example tool configuration:

```yaml
# tools.yaml
- name: format-code
  cmd: "prettier --write {{file}}"
  context_description: "Format code files using Prettier"
  working_dir: "{{project_dir}}"

- name: generate-docs
  cmd: "godoc -http=:6060"
  context_description: "Generate and serve Go documentation"

- name: scaffold-component
  cmd: "./scripts/scaffold.sh {{component_name}} {{component_type}}"
  context_description: "Create a new component from a template"
  args:
    component_name: ""
    component_type: "basic"
```

## Extensions and Plugin Ideas

### Editor Integrations

Example VS Code integration:

```json
// .vscode/settings.json
{
  "guild.enableCodeAssistant": true,
  "guild.defaultAgent": "coder",
  "guild.objectives.path": "./objectives",
  "guild.tools.path": "./tools",
  "guild.autoSave": true
}
```

Command palette actions:

- `Guild: Initialize Project`
- `Guild: Add Objective`
- `Guild: Run Guild`
- `Guild: Show Kanban Board`
- `Guild: Rewrite Selection`
- `Guild: Explain Code`

### CI/CD Integration

GitHub Actions workflow:

```yaml
# .github/workflows/guild.yml
name: Guild CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  guild:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Guild
        uses: guild-ai/setup-guild@v1

      - name: Run Guild
        run: guild run ci-guild --focus objectives/ci

      - name: Generate Reports
        run: guild report generate

      - name: Upload Artifacts
        uses: actions/upload-artifact@v3
        with:
          name: guild-reports
          path: output/reports
```

### Custom Agent Templates

Creating specialized agent templates:

```yaml
# templates/agents/security-reviewer.yaml
name: security-reviewer
provider: anthropic
model: claude-3-opus
tools:
  - security-scan
  - vulnerability-check
  - license-audit
personality:
  persona: "Security Specialist"
  style: "thorough and security-focused"
  expertise: ["application security", "vulnerability assessment"]
prompt_template: |
  You are a security reviewer agent. Your job is to review code for security vulnerabilities.
  Look for:
  - SQL injection
  - XSS vulnerabilities
  - Authentication issues
  - Authorization flaws
  - Dependency vulnerabilities
```

Import with:

```bash
guild agent import templates/agents/security-reviewer.yaml
```

## Example Integrations

### Aider Integration

Using Aider with Guild:

```yaml
# tools.yaml
- name: aider
  cmd: "aider --yes --message '{{task}}'"
  context_description: "Refactor code with scoped assistant."
  working_dir: "{{project_dir}}"
```

Example task using Aider:

```yaml
# Example task that uses Aider
task:
  id: "task-improve-auth"
  title: "Improve authentication system"
  description: "Refactor the authentication system to use JWT and add refresh token support"
  agent_id: "backend-dev"
  tools: ["aider"]
  files: ["pkg/auth/auth.go", "pkg/auth/token.go"]
```

### ZeroMQ Event Subscription

Subscribing to Guild events from your own applications:

```go
// monitor.go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/zeromq/goczmq"
)

type Event struct {
	Type      string      `json:"type"`
	TaskID    string      `json:"task_id"`
	AgentID   string      `json:"agent_id"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

func main() {
	// Create a ZeroMQ subscriber socket
	subscriber, err := goczmq.NewSub("tcp://localhost:5555", "")
	if err != nil {
		log.Fatal(err)
	}
	defer subscriber.Destroy()

	fmt.Println("Listening for Guild events...")

	for {
		// Receive a message
		message, err := subscriber.RecvMessage()
		if err != nil {
			log.Printf("Error receiving message: %v", err)
			continue
		}

		if len(message) < 2 {
			continue
		}

		// Parse the event
		var event Event
		if err := json.Unmarshal([]byte(message[1]), &event); err != nil {
			log.Printf("Error parsing event: %v", err)
			continue
		}

		// Process the event
		fmt.Printf("Event: %s - Task: %s - Agent: %s\n", event.Type, event.TaskID, event.AgentID)
	}
}
```

### BoltDB Inspection

Custom tool for inspecting the BoltDB database:

```go
// cmd/guild/commands/inspect.go
package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

// InspectCmd returns the database inspection command
func InspectCmd() *cobra.Command {
	var (
		dbPath  string
		bucket  string
		verbose bool
	)

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect the Guild database",
		Run: func(cmd *cobra.Command, args []string) {
			if dbPath == "" {
				dbPath = "guild.db"
			}

			// Open the database
			db, err := bbolt.Open(dbPath, 0600, nil)
			if err != nil {
				log.Fatalf("Failed to open database: %v", err)
			}
			defer db.Close()

			// List buckets if no specific bucket provided
			if bucket == "" {
				fmt.Println("Buckets:")
				err = db.View(func(tx *bbolt.Tx) error {
					return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
						fmt.Printf("- %s\n", name)
						return nil
					})
				})
				return
			}

			// Inspect specific bucket
			fmt.Printf("Inspecting bucket: %s\n", bucket)
			err = db.View(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte(bucket))
				if b == nil {
					return fmt.Errorf("bucket not found: %s", bucket)
				}

				count := 0
				err := b.ForEach(func(k, v []byte) error {
					count++
					if verbose {
						if len(v) > 100 {
							fmt.Printf("%s: %s...\n", k, v[:100])
						} else {
							fmt.Printf("%s: %s\n", k, v)
						}
					}
					return nil
				})

				fmt.Printf("Total entries: %d\n", count)
				return err
			})

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&dbPath, "db", "d", "", "Database path (default: guild.db)")
	cmd.Flags().StringVarP(&bucket, "bucket", "b", "", "Bucket to inspect")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show all entries")

	return cmd
}
```

## Conclusion

Guild provides a flexible framework for orchestrating AI agents to complete complex tasks. By combining the power of modern LLMs with a structured workflow and cost-aware optimization, Guild enables you to automate a wide range of tasks while maintaining control over the process.

To get started:

1. **Initialize a project**: `guild init my-project`
2. **Add agents**: `guild agent add assistant --provider anthropic --model claude-3-sonnet`
3. **Define objectives**: Create markdown files in the `objectives/` directory
4. **Run the guild**: `guild run my-project`

Remember to follow the best practices for writing objectives, composing guilds, and integrating tools to get the most out of the framework.
