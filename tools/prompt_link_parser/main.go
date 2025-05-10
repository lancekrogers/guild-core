/*
Prompt Link Parse

A simple tool for expanding @link references inside Markdown files.

Usage:

	prompt-link-parse <inputfile.md>

Behavior:
- Scans the input markdown file for any '@relative/path/to/file.md' references.
- Replaces each @link with the full contents of the referenced markdown file.
- Recursively processes links inside linked files (up to a maximum depth of 5).
- Outputs a new file alongside the input, named "<original_filename>_links_inserted.md".

Intended Use:
- Preparing Markdown prompt files for direct use with LLMs.
- Building final documents with modular prompt linking.
- Optional: can be imported as a library function (ParseMarkdownWithLinks) for in-memory expansion without writing to disk.

Limitations:
- Links must match the pattern: '@relative/path/to/file.md'.
- Maximum recursion depth is hardcoded to prevent infinite loops (currently set to 5).
- Assumes UTF-8 encoding for all files.

Example:

Input file 'commands/plan.md':

	# Plan
	Here's the overall plan:
	@common/templates/overview.md

Expanded output 'commands/plan_links_inserted.md' will have the full content from 'common/templates/overview.md' in place.
*/
package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

const maxDepth = 5

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: prompt-link-parse <inputfile>")
		os.Exit(1)
	}
	inputFile := os.Args[1]

	expanded, err := ParseMarkdownWithLinks(inputFile, 0)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Output new file with _links_inserted suffix
	outputFile := strings.TrimSuffix(inputFile, ".md") + "_links_inserted.md"
	err = ioutil.WriteFile(outputFile, []byte(expanded), 0644)
	if err != nil {
		fmt.Println("Error writing expanded file:", err)
		os.Exit(1)
	}

	fmt.Printf("Expanded markdown written to: %s\n", outputFile)
}

// ParseMarkdownWithLinks expands @link references recursively
func ParseMarkdownWithLinks(path string, depth int) (string, error) {
	if depth > maxDepth {
		return "", fmt.Errorf("max link recursion depth (%d) exceeded at %s", maxDepth, path)
	}

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	var output strings.Builder
	scanner := bufio.NewScanner(file)

	atLinkRegex := regexp.MustCompile(`@([\w./_-]+\.md)`) // e.g., @some/path/to/snippet.md

	for scanner.Scan() {
		line := scanner.Text()
		matches := atLinkRegex.FindAllStringSubmatch(line, -1)

		if matches == nil {
			output.WriteString(line)
			output.WriteString("\n")
			continue
		}

		expandedLine := line
		for _, match := range matches {
			if len(match) == 2 {
				linkPath := match[1]
				expandedContent, err := ParseMarkdownWithLinks(linkPath, depth+1)
				if err != nil {
					return "", fmt.Errorf("error expanding link %s: %w", linkPath, err)
				}
				expandedLine = strings.ReplaceAll(expandedLine, "@"+linkPath, expandedContent)
			}
		}
		output.WriteString(expandedLine)
		output.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file %s: %w", path, err)
	}
	return output.String(), nil
}
