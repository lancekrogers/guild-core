// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commission

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// MarkdownParser implements ObjectiveParser for markdown files
type MarkdownParser struct {
	options ParseOptions
	// Map of section type identifiers to their type names
	sectionTypes map[string]string
}

// newMarkdownParser creates a new markdown parser with the given options (private constructor)
func newMarkdownParser(options ParseOptions) *MarkdownParser {
	if options.TagPrefixes == nil {
		options = DefaultParseOptions()
	}

	// Default section type mapping
	sectionTypes := map[string]string{
		"context":        "context",
		"background":     "context",
		"goal":           "goal",
		"goals":          "goal",
		"objective":      "goal",
		"acceptance":     "acceptance",
		"criteria":       "acceptance",
		"implementation": "implementation",
		"approach":       "implementation",
		"plan":           "implementation",
		"tasks":          "tasks",
		"task list":      "tasks",
		"timeline":       "timeline",
		"schedule":       "timeline",
		"resources":      "resources",
		"references":     "resources",
		"links":          "resources",
		"notes":          "notes",
		"considerations": "notes",
	}

	return &MarkdownParser{
		options:      options,
		sectionTypes: sectionTypes,
	}
}

// NewMarkdownParser creates a new markdown parser with the given options
func NewMarkdownParser(options ParseOptions) *MarkdownParser {
	return newMarkdownParser(options)
}

// DefaultMarkdownParserFactory creates a markdown parser for registry use
func DefaultMarkdownParserFactory(options ParseOptions) *MarkdownParser {
	return newMarkdownParser(options)
}

// Parse parses an objective from markdown content
func (p *MarkdownParser) Parse(content, source string) (*Commission, error) {
	// Process the content into sections
	sections, err := p.extractSections(content)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidFormat, "failed to extract sections").
			WithComponent("commission.parser").
			WithOperation("Parse")
	}

	// Extract main title and description
	title, description := p.extractTitleAndDescription(sections)

	// Create new commission with default values
	commission := NewCommission(title, description)
	commission.Source = source
	commission.Content = content
	commission.Status = p.options.DefaultStatus

	// Extract metadata and tags from the content
	metadata, tags := p.extractMetadataAndTags(content)
	commission.Metadata = metadata
	commission.Tags = tags

	// Set priority if found in metadata
	if priority, ok := metadata["priority"]; ok {
		commission.Priority = priority
	} else {
		commission.Priority = p.options.DefaultPriority
	}

	// Set owner if found in metadata
	if owner, ok := metadata["owner"]; ok {
		commission.Owner = owner
	} else {
		commission.Owner = p.options.DefaultOwner
	}

	// Set assignees if found in metadata
	if assignees, ok := metadata["assignees"]; ok {
		commission.Assignees = strings.Split(assignees, ",")
		// Trim spaces
		for i, a := range commission.Assignees {
			commission.Assignees[i] = strings.TrimSpace(a)
		}
	}

	// Process sections into objective parts
	commission.Parts = p.processSectionsIntoParts(sections)

	// Extract tasks if present
	commission.Tasks = p.extractTasks(sections)

	return commission, nil
}

// ParseFile parses an objective from a markdown file
func (p *MarkdownParser) ParseFile(path string) (*Commission, error) {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read commission file").
			WithComponent("commission.parser").
			WithOperation("ParseFile").
			WithDetails("file_path", path)
	}

	// Parse the content
	return p.Parse(string(content), path)
}

// ParseFile is a standalone function to parse a markdown file into an Objective
func ParseFile(path string) (*Commission, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to open commission file").
			WithComponent("commission.parser").
			WithOperation("ParseFile").
			WithDetails("file_path", path)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log the close error
			_ = gerror.Wrap(closeErr, gerror.ErrCodeStorage, "failed to close commission file").
				WithComponent("commission.parser").
				WithOperation("ParseFile").
				WithDetails("file_path", path)
		}
	}()

	scanner := bufio.NewScanner(file)
	content := &strings.Builder{}
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}

	parser := DefaultMarkdownParserFactory(ParseOptions{})
	return parser.Parse(content.String(), path)
}

// extractSections extracts sections from markdown content
func (p *MarkdownParser) extractSections(content string) ([]*SectionInfo, error) {
	var sections []*SectionInfo

	// Regular expression to match headings
	// This matches headings like "## Title" and extracts the level and title
	headingRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

	// Process content line by line
	scanner := bufio.NewScanner(strings.NewReader(content))

	var currentSection *SectionInfo
	var sectionContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this line is a heading
		matches := headingRegex.FindStringSubmatch(line)

		if len(matches) > 0 {
			// We found a heading, which means the end of the previous section
			// and the start of a new one

			// Save the previous section if it exists
			if currentSection != nil {
				currentSection.Content = sectionContent.String()
				sections = append(sections, currentSection)
				sectionContent.Reset()
			}

			// Create a new section
			level := len(matches[1]) // Number of # characters
			title := strings.TrimSpace(matches[2])
			sectionType := p.determineSectionType(title)

			currentSection = &SectionInfo{
				Title:    title,
				Level:    level,
				Type:     sectionType,
				MetaTags: make(map[string]string),
			}
		} else if currentSection != nil {
			// Add the line to the current section content
			sectionContent.WriteString(line + "\n")
		} else {
			// This is content before the first heading, treat it as a preamble
			sectionContent.WriteString(line + "\n")
		}
	}

	// Don't forget to save the last section
	if currentSection != nil {
		currentSection.Content = sectionContent.String()
		sections = append(sections, currentSection)
	} else if sectionContent.Len() > 0 {
		// If there were no headings at all, create a default section
		sections = append(sections, &SectionInfo{
			Title:    "Untitled",
			Level:    1,
			Content:  sectionContent.String(),
			Type:     "main",
			MetaTags: make(map[string]string),
		})
	}

	if len(sections) == 0 {
		return nil, gerror.New(gerror.ErrCodeInvalidFormat, "no sections found in commission content", nil).
			WithComponent("commission.parser").
			WithOperation("extractSections")
	}

	return sections, nil
}

// determineSectionType determines the type of a section based on its title
func (p *MarkdownParser) determineSectionType(title string) string {
	// Convert to lowercase for case-insensitive matching
	titleLower := strings.ToLower(title)

	// Check if the title directly matches any of our known section types
	for key, sectionType := range p.sectionTypes {
		if strings.Contains(titleLower, key) {
			return sectionType
		}
	}

	// Default to "other" if no match is found
	return "other"
}

// extractTitleAndDescription extracts the title and description from sections
func (p *MarkdownParser) extractTitleAndDescription(sections []*SectionInfo) (string, string) {
	if len(sections) == 0 {
		return "Untitled Objective", ""
	}

	// Use the first section's title as the objective title
	title := sections[0].Title

	// Extract description from the first section's content
	description := ""
	if len(sections) > 0 {
		// Split the content by lines and use non-empty lines for description
		lines := strings.Split(sections[0].Content, "\n")
		var descLines []string

		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine != "" {
				descLines = append(descLines, trimmedLine)
			}
		}

		if len(descLines) > 0 {
			// Join the first few lines as description
			maxDescLines := 3
			if len(descLines) < maxDescLines {
				maxDescLines = len(descLines)
			}
			description = strings.Join(descLines[:maxDescLines], " ")
		}
	}

	return title, description
}

// extractMetadataAndTags extracts metadata and tags from the content
func (p *MarkdownParser) extractMetadataAndTags(content string) (map[string]string, []string) {
	metadata := make(map[string]string)
	var tags []string

	// Process content line by line
	scanner := bufio.NewScanner(strings.NewReader(content))

	// Regular expressions for metadata and tags
	metaRegex := make([]*regexp.Regexp, 0, len(p.options.MetaPrefixes))
	for _, prefix := range p.options.MetaPrefixes {
		metaRegex = append(metaRegex, regexp.MustCompile(`(?i)`+regexp.QuoteMeta(prefix)+`(\w+):\s*(.+)`))
	}

	tagRegex := make([]*regexp.Regexp, 0, len(p.options.TagPrefixes))
	for _, prefix := range p.options.TagPrefixes {
		tagRegex = append(tagRegex, regexp.MustCompile(`(?i)`+regexp.QuoteMeta(prefix)+`(\w+)`))
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Check for metadata
		for _, re := range metaRegex {
			matches := re.FindStringSubmatch(line)
			if len(matches) > 2 {
				key := strings.ToLower(matches[1])
				value := strings.TrimSpace(matches[2])
				metadata[key] = value
			}
		}

		// Check for tags
		for _, re := range tagRegex {
			matches := re.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) > 1 {
					tag := strings.ToLower(match[1])
					if !containsString(tags, tag) {
						tags = append(tags, tag)
					}
				}
			}
		}
	}

	return metadata, tags
}

// processSectionsIntoParts converts sections into objective parts
func (p *MarkdownParser) processSectionsIntoParts(sections []*SectionInfo) []*CommissionPart {
	var parts []*CommissionPart

	// Skip the first section if it's the title/description section
	startIdx := 0
	if len(sections) > 1 {
		startIdx = 1
	}

	// Process each section into a part
	for i, section := range sections[startIdx:] {
		part := NewCommissionPart(
			section.Title,
			section.Content,
			section.Type,
			i,
		)

		// Add any metadata from the section
		for k, v := range section.MetaTags {
			part.Metadata[k] = v
		}

		parts = append(parts, part)
	}

	return parts
}

// extractTasks extracts tasks from sections
func (p *MarkdownParser) extractTasks(sections []*SectionInfo) []*CommissionTask {
	var tasks []*CommissionTask

	// Find sections that might contain tasks
	for _, section := range sections {
		if section.Type == "tasks" || section.Type == "implementation" {
			// Extract tasks from this section
			sectionTasks := p.extractTasksFromSection(section)
			tasks = append(tasks, sectionTasks...)
		}
	}

	return tasks
}

// extractTasksFromSection extracts tasks from a single section
func (p *MarkdownParser) extractTasksFromSection(section *SectionInfo) []*CommissionTask {
	var tasks []*CommissionTask

	// Regular expressions for task lists
	// Matches Markdown task lists like "- [ ] Task description"
	taskRegex := regexp.MustCompile(`^\s*[-*]\s*\[\s*([xX ])\s*\]\s*(.+)$`)

	// Matches numbered lists like "1. Task description"
	numberedTaskRegex := regexp.MustCompile(`^\s*(\d+)\.[\s]+(.*?)$`)

	// Process section content line by line
	scanner := bufio.NewScanner(strings.NewReader(section.Content))

	taskIndex := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Check for markdown task list items
		matches := taskRegex.FindStringSubmatch(line)
		if len(matches) > 2 {
			status := "todo"
			if strings.ToLower(matches[1]) == "x" {
				status = "done"
			}

			description := matches[2]
			task := NewCommissionTask(
				description,
				"", // No detailed description for now
				taskIndex,
			)
			task.Status = status

			tasks = append(tasks, task)
			taskIndex++
			continue
		}

		// Check for numbered list items
		matches = numberedTaskRegex.FindStringSubmatch(line)
		if len(matches) > 2 {
			description := matches[2]
			task := NewCommissionTask(
				description,
				"", // No detailed description for now
				taskIndex,
			)

			tasks = append(tasks, task)
			taskIndex++
		}
	}

	return tasks
}
