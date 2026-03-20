// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package manager

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// ArchiveParser implements the ResponseParser interface for Guild Archives
type ArchiveParser struct {
	taskPattern *regexp.Regexp
	filePattern *regexp.Regexp
}

// NewArchiveParser creates a new Archive parser for Guild responses
func NewArchiveParser() *ArchiveParser {
	// Pattern to match file sections in the response
	filePattern := regexp.MustCompile(`(?m)^## File: (.+)$\n((?:.*\n?)*)(?=^## File: |$)`)

	// Pattern to match tasks in content (Workshop Board tasks)
	taskPattern := regexp.MustCompile(`(?m)^\*\*Tasks Generated\*\*:\s*\n((?:^- \w+-\d+:.*\n(?:^\s+- .*\n)*)+)`)

	return &ArchiveParser{
		taskPattern: taskPattern,
		filePattern: filePattern,
	}
}

// ParseResponse implements the ResponseParser interface for Guild Archives
func (p *ArchiveParser) ParseResponse(response *ArtisanResponse) (*FileStructure, error) {
	content := response.Content

	// First, try to parse as structured response with file sections
	if files := p.parseStructuredResponse(content); len(files) > 0 {
		return &FileStructure{
			RootDir: ".",
			Files:   files,
		}, nil
	}

	// Fall back to parsing as a single README if no structure found
	return p.parseSingleFile(content)
}

// parseStructuredResponse parses Guild Master responses with "## File: path" sections for Archives
func (p *ArchiveParser) parseStructuredResponse(content string) []*FileEntry {
	matches := p.filePattern.FindAllStringSubmatch(content, -1)
	var files []*FileEntry

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		filePath := strings.TrimSpace(match[1])
		fileContent := strings.TrimSpace(match[2])

		// Clean up the file path
		filePath = p.cleanFilePath(filePath)

		// Count Workshop Board tasks in this file
		taskCount := p.countWorkshopTasks(fileContent)

		files = append(files, &FileEntry{
			Path:       filePath,
			Content:    fileContent,
			Type:       FileTypeMarkdown,
			TasksCount: taskCount,
			Metadata: map[string]interface{}{
				"source":      "guild_master_response",
				"archived_by": "guild_archive_parser",
			},
		})
	}

	return files
}

// parseSingleFile parses a Guild Master response as a single README file for Archives
func (p *ArchiveParser) parseSingleFile(content string) (*FileStructure, error) {
	// Clean up the content
	content = strings.TrimSpace(content)

	if content == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "empty Guild Master response content", nil).
			WithComponent("manager").
			WithOperation("parseSingleFile")
	}

	// Count Workshop Board tasks
	taskCount := p.countWorkshopTasks(content)

	// Create a single README file for the Guild Archives
	file := &FileEntry{
		Path:       "README.md",
		Content:    content,
		Type:       FileTypeMarkdown,
		TasksCount: taskCount,
		Metadata: map[string]interface{}{
			"source":            "guild_master_response",
			"archived_by":       "guild_archive_parser",
			"single_file":       true,
			"commission_format": "unified",
		},
	}

	return &FileStructure{
		RootDir: ".",
		Files:   []*FileEntry{file},
	}, nil
}

// cleanFilePath cleans and validates file paths for Guild Archives
func (p *ArchiveParser) cleanFilePath(path string) string {
	// Remove any leading/trailing whitespace
	path = strings.TrimSpace(path)

	// Remove any markdown formatting that might have leaked in
	path = strings.TrimPrefix(path, "`")
	path = strings.TrimSuffix(path, "`")

	// Clean the path
	path = filepath.Clean(path)

	// Ensure it ends with .md if it's not a directory
	if !strings.HasSuffix(path, "/") && !strings.HasSuffix(path, ".md") {
		path += ".md"
	}

	return path
}

// countWorkshopTasks counts the number of Workshop Board tasks in the content
func (p *ArchiveParser) countWorkshopTasks(content string) int {
	matches := p.taskPattern.FindAllString(content, -1)
	count := 0

	for _, match := range matches {
		// Count individual task lines (- CATEGORY-NUMBER: title)
		lines := strings.Split(match, "\n")
		for _, line := range lines {
			if regexp.MustCompile(`^\s*- \w+-\d+:`).MatchString(line) {
				count++
			}
		}
	}

	return count
}

// ExtractWorkshopTasksFromFile extracts Workshop Board tasks from a specific Archive file
func (p *ArchiveParser) ExtractWorkshopTasksFromFile(content string) []WorkshopTaskInfo {
	var tasks []WorkshopTaskInfo

	matches := p.taskPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		taskSection := match[1]
		tasks = append(tasks, p.parseWorkshopTaskSection(taskSection)...)
	}

	return tasks
}

// parseWorkshopTaskSection parses a Workshop Board tasks section into individual artisan tasks
func (p *ArchiveParser) parseWorkshopTaskSection(section string) []WorkshopTaskInfo {
	var tasks []WorkshopTaskInfo
	lines := strings.Split(section, "\n")

	var currentTask *WorkshopTaskInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this is a Workshop Board task line (- CATEGORY-NUMBER: title)
		if matches := regexp.MustCompile(`^- (\w+)-(\d+): (.+)$`).FindStringSubmatch(line); len(matches) == 4 {
			// Save previous task if exists
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
			}

			// Start new artisan task
			currentTask = &WorkshopTaskInfo{
				ID:       matches[1] + "-" + matches[2],
				Category: matches[1],
				Number:   matches[2],
				Title:    matches[3],
			}
		} else if currentTask != nil {
			// Parse task properties
			if matches := regexp.MustCompile(`^\s*- Priority: (.+)$`).FindStringSubmatch(line); len(matches) == 2 {
				currentTask.Priority = strings.TrimSpace(matches[1])
			} else if matches := regexp.MustCompile(`^\s*- Estimate: (.+)$`).FindStringSubmatch(line); len(matches) == 2 {
				currentTask.Estimate = strings.TrimSpace(matches[1])
			} else if matches := regexp.MustCompile(`^\s*- Dependencies: (.+)$`).FindStringSubmatch(line); len(matches) == 2 {
				deps := strings.TrimSpace(matches[1])
				if deps != "none" && deps != "" {
					currentTask.Dependencies = strings.Split(deps, ",")
					for i := range currentTask.Dependencies {
						currentTask.Dependencies[i] = strings.TrimSpace(currentTask.Dependencies[i])
					}
				}
			} else if matches := regexp.MustCompile(`^\s*- Capabilities: (.+)$`).FindStringSubmatch(line); len(matches) == 2 {
				caps := strings.TrimSpace(matches[1])
				if caps != "" {
					// Parse artisan capabilities required for this task
					currentTask.ArtisanCapabilities = strings.Split(caps, ",")
					for i := range currentTask.ArtisanCapabilities {
						currentTask.ArtisanCapabilities[i] = strings.TrimSpace(currentTask.ArtisanCapabilities[i])
					}
				}
			} else if matches := regexp.MustCompile(`^\s*- Description: (.+)$`).FindStringSubmatch(line); len(matches) == 2 {
				currentTask.Description = strings.TrimSpace(matches[1])
			}
		}
	}

	// Don't forget the last task
	if currentTask != nil {
		tasks = append(tasks, *currentTask)
	}

	return tasks
}

// WorkshopTaskInfo represents a parsed Workshop Board task for artisans
type WorkshopTaskInfo struct {
	ID                  string
	Category            string
	Number              string
	Title               string
	Priority            string
	Estimate            string
	Dependencies        []string
	ArtisanCapabilities []string // Required artisan capabilities
	Description         string
}
