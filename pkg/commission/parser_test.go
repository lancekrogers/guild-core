package commission

import (
	"strings"
	"testing"
)

func TestMarkdownParser_Parse(t *testing.T) {
	testCases := []struct {
		name          string
		content       string
		expectedTitle string
		expectedParts int
		expectedTasks int
	}{
		{
			name: "Basic objective",
			content: `# Test Objective
			
This is a description of the test objective.

## Context

This is the context section.

## Goal

This is the goal section.

## Implementation

- [ ] Task 1
- [x] Task 2
- [ ] Task 3

## Notes

Some notes about the objective.
`,
			expectedTitle: "Test Objective",
			expectedParts: 4, // Context, Goal, Implementation, Notes
			expectedTasks: 3,
		},
		{
			name: "Objective with metadata",
			content: `# Feature Implementation
			
@priority: high
@owner: alice
@tag:important @tag:feature

Implement a new feature in the application.

## Background

This feature is needed because...

## Acceptance Criteria

- Must support X
- Must be compatible with Y

## Tasks

1. Design the API
2. Implement the backend
3. Implement the frontend
4. Write tests
5. Document the feature
`,
			expectedTitle: "Feature Implementation",
			expectedParts: 3, // Background, Acceptance Criteria, Tasks
			expectedTasks: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewMarkdownParser(DefaultParseOptions())
			objective, err := parser.Parse(tc.content, "test.md")
			
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			
			if objective.Title != tc.expectedTitle {
				t.Errorf("Expected title '%s', got '%s'", tc.expectedTitle, objective.Title)
			}
			
			if len(objective.Parts) != tc.expectedParts {
				t.Errorf("Expected %d parts, got %d", tc.expectedParts, len(objective.Parts))
				for i, part := range objective.Parts {
					t.Logf("Part %d: Title=%s, Type=%s", i, part.Title, part.Type)
				}
			}
			
			if len(objective.Tasks) != tc.expectedTasks {
				t.Errorf("Expected %d tasks, got %d", tc.expectedTasks, len(objective.Tasks))
			}
			
			// Additional tests for other objective properties could go here
		})
	}
}

func TestMarkdownParser_ParseFile(t *testing.T) {
	parser := NewMarkdownParser(DefaultParseOptions())
	objective, err := parser.ParseFile("testdata/sample_commission.md")

	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Test basic properties
	if objective.Title != "Implement Objective System" {
		t.Errorf("Expected title 'Implement Objective System', got '%s'", objective.Title)
	}

	if objective.Priority != "high" {
		t.Errorf("Expected priority 'high', got '%s'", objective.Priority)
	}

	if objective.Owner != "guild-team" {
		t.Errorf("Expected owner 'guild-team', got '%s'", objective.Owner)
	}

	// Test tags
	expectedTags := []string{"core", "feature"}
	if len(objective.Tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(objective.Tags))
	}

	// Test parts
	if len(objective.Parts) < 4 {
		t.Errorf("Expected at least 4 parts, got %d", len(objective.Parts))
	}

	// Test tasks
	if len(objective.Tasks) != 6 {
		t.Errorf("Expected 6 tasks, got %d", len(objective.Tasks))
	}
}

func TestExtractSections(t *testing.T) {
	parser := NewMarkdownParser(DefaultParseOptions())
	
	content := `# Main Title
	
This is the main description.

## Section 1
	
This is section 1 content.

## Section 2
	
This is section 2 content.
`
	
	sections, err := parser.extractSections(content)
	if err != nil {
		t.Fatalf("extractSections failed: %v", err)
	}
	
	if len(sections) != 3 {
		t.Errorf("Expected 3 sections, got %d", len(sections))
	}
	
	if sections[0].Title != "Main Title" {
		t.Errorf("Expected first section title 'Main Title', got '%s'", sections[0].Title)
	}
	
	if sections[1].Title != "Section 1" {
		t.Errorf("Expected second section title 'Section 1', got '%s'", sections[1].Title)
	}
	
	if !strings.Contains(sections[1].Content, "This is section 1 content.") {
		t.Errorf("Expected section 1 to contain its content, got '%s'", sections[1].Content)
	}
}

func TestExtractTasks(t *testing.T) {
	parser := NewMarkdownParser(DefaultParseOptions())
	
	section := &SectionInfo{
		Title: "Tasks",
		Type:  "tasks",
		Content: `
- [ ] Task 1
- [x] Task 2
1. Numbered task 1
2. Numbered task 2
`,
	}
	
	tasks := parser.extractTasksFromSection(section)
	
	if len(tasks) != 4 {
		t.Errorf("Expected 4 tasks, got %d", len(tasks))
	}
	
	if tasks[0].Title != "Task 1" {
		t.Errorf("Expected first task title 'Task 1', got '%s'", tasks[0].Title)
	}
	
	if tasks[0].Status != "todo" {
		t.Errorf("Expected first task status 'todo', got '%s'", tasks[0].Status)
	}
	
	if tasks[1].Status != "done" {
		t.Errorf("Expected second task status 'done', got '%s'", tasks[1].Status)
	}
}