package chat

import (
	"fmt"
	"time"
)

// getTestHelp returns help text for test commands
func (m ChatModel) getTestHelp() string {
	return `🧪 Test Commands:
  /test markdown   - Test markdown rendering
  /test code      - Test syntax highlighting
  /test mixed     - Test mixed content
  /test agents    - Test agent animations
  /test completion - Test auto-completion`
}

// testMarkdownRendering tests markdown rendering capabilities
func (m ChatModel) testMarkdownRendering() string {
	testContent := `# Markdown Test

This is a **bold** statement and this is *italic*.

## Features

- First item
- Second item with ` + "`inline code`" + `
- Third item

### Code Block

` + "```go" + `
func TestMarkdown() {
    fmt.Println("Syntax highlighting!")
}
` + "```" + `

> This is a blockquote with [a link](https://example.com).`

	// If markdown renderer is available, use it
	if m.contentFormatter != nil {
		return m.contentFormatter.FormatMarkdown(testContent)
	}

	// Fallback to plain text
	return testContent
}

// testCodeHighlighting tests code syntax highlighting
func (m ChatModel) testCodeHighlighting() string {
	codeExamples := map[string]string{
		"go": `package main

import "fmt"

func main() {
    // Medieval-themed example
    fmt.Println("Welcome to the Guild!")

    agents := []string{"manager", "developer", "reviewer"}
    for _, agent := range agents {
        fmt.Printf("Agent @%s is ready\n", agent)
    }
}`,
		"python": `def process_commission(commission):
    """Process a guild commission"""
    agents = ["manager", "developer", "reviewer"]

    for agent in agents:
        print(f"Assigning task to @{agent}")

    return "Commission completed!"`,
		"javascript": `class GuildAgent {
    constructor(id, capabilities) {
        this.id = id;
        this.capabilities = capabilities;
    }

    async processTask(task) {
        console.log(` + "`Agent @${this.id} processing: ${task}`" + `);
        // Simulate work
        await new Promise(resolve => setTimeout(resolve, 1000));
        return "Task completed!";
    }
}`,
	}

	var result string
	for lang, code := range codeExamples {
		if m.contentFormatter != nil {
			result += m.contentFormatter.FormatCodeBlock(code, lang)
			result += "\n\n"
		} else {
			result += fmt.Sprintf("```%s\n%s\n```\n\n", lang, code)
		}
	}

	return result
}

// testMixedContent tests rendering of mixed markdown and code
func (m ChatModel) testMixedContent() string {
	mixedContent := `# Guild Framework Demo

The **Guild Framework** orchestrates AI agents like a medieval guild, where each agent has specialized skills.

## Agent Types

1. **Guild Master** (@manager) - Coordinates and plans
2. **Code Artisan** (@developer) - Implements features
3. **Review Artisan** (@reviewer) - Ensures quality

## Example Commission

` + "```markdown" + `
# E-Commerce API Commission

## Objective
Build a REST API for product management

## Requirements
- CRUD operations for products
- Authentication and authorization
- Rate limiting
- API documentation

## Assigned Agents
- @manager: Planning and coordination
- @developer: Implementation
- @reviewer: Code review and testing
` + "```" + `

## Code Example

Here's how an agent might implement a simple endpoint:

` + "```go" + `
func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
    var product Product
    if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Validate product
    if err := h.validator.Validate(product); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Save to database
    if err := h.db.Create(&product); err != nil {
        http.Error(w, "Failed to create product", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(product)
}
` + "```" + `

> 💡 **Tip**: Use @all to broadcast tasks to all available agents!`

	if m.contentFormatter != nil {
		return m.contentFormatter.FormatMarkdown(mixedContent)
	}

	return mixedContent
}

// testAgentAnimations simulates agent activity animations
func (m *ChatModel) testAgentAnimations() {
	if m.agentIndicators == nil {
		m.addSystemMessage("Agent indicators not initialized")
		return
	}

	// Simulate various agent activities
	agents := []string{"manager", "developer", "reviewer", "tester"}

	// Start thinking animations
	for _, agent := range agents {
		m.agentIndicators.SetThinkingAnimation(agent)
		m.addSystemMessage(fmt.Sprintf("@%s is thinking...", agent))
	}

	// After 2 seconds, switch to working
	go func() {
		time.Sleep(2 * time.Second)
		for i, agent := range agents {
			contexts := []string{"planning", "coding", "reviewing", "testing"}
			m.agentIndicators.SetWorkingAnimation(agent, contexts[i])
			m.addSystemMessage(fmt.Sprintf("@%s is now %s", agent, contexts[i]))
		}

		// After another 3 seconds, clear animations
		time.Sleep(3 * time.Second)
		for _, agent := range agents {
			m.agentIndicators.ClearAnimation(agent)
			m.addSystemMessage(fmt.Sprintf("@%s completed their task", agent))
		}
	}()
}

// testCompletionSystem tests the auto-completion functionality
func (m *ChatModel) testCompletionSystem() {
	if m.completionEng == nil {
		m.addSystemMessage("Completion engine not initialized")
		return
	}

	// Test various completion scenarios
	testCases := []struct {
		input    string
		expected string
	}{
		{"/he", "Should complete to /help"},
		{"@man", "Should complete to @manager"},
		{"/prompt li", "Should complete to /prompt list"},
		{"@dev", "Should complete to @developer"},
		{"/tools st", "Should complete to /tools status"},
	}

	m.addSystemMessage("Testing auto-completion system:")

	for _, tc := range testCases {
		completions := m.completionEng.Complete(tc.input, len(tc.input))
		if len(completions) > 0 {
			m.addSystemMessage(fmt.Sprintf("✅ '%s' → '%s' (%s)",
				tc.input, completions[0].Content, tc.expected))
		} else {
			m.addSystemMessage(fmt.Sprintf("❌ '%s' → No completions (%s)",
				tc.input, tc.expected))
		}
	}

	// Test command registration
	m.addSystemMessage("\nRegistered commands:")
	commands := m.completionEng.GetAllCommands()
	for _, cmd := range commands {
		m.addSystemMessage(fmt.Sprintf("  • %s", cmd))
	}

	// Test agent registration
	m.addSystemMessage("\nRegistered agents:")
	agents := m.completionEng.GetAllAgents()
	for _, agent := range agents {
		m.addSystemMessage(fmt.Sprintf("  • %s", agent))
	}
}

// handleTestRichContent handles test rich content messages
func (m *ChatModel) handleTestRichContent(msg testRichContentMsg) {
	// Add the rich content message
	m.messages = append(m.messages, Message{
		Type:      msgAgent,
		Content:   msg.content,
		AgentID:   msg.agentID,
		Timestamp: time.Now(),
		Metadata:  msg.metadata,
	})

	// If we have a content formatter, the View will handle rendering
	m.updateMessagesView()
}

// handleCompletionResult handles completion result messages
func (m *ChatModel) handleCompletionResult(msg completionResultMsg) {
	m.completionResults = msg.results
	m.showingCompletion = len(msg.results) > 0
	m.completionIndex = 0
}
