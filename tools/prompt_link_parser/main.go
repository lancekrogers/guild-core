// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

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

	"github.com/lancekrogers/guild-core/pkg/gerror"
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
	err = ioutil.WriteFile(outputFile, []byte(expanded), 0o644)
	if err != nil {
		fmt.Println("Error writing expanded file:", err)
		os.Exit(1)
	}

	fmt.Printf("Expanded markdown written to: %s\n", outputFile)
}

// ParseMarkdownWithLinks expands @link references recursively
func ParseMarkdownWithLinks(path string, depth int) (string, error) {
	if depth > maxDepth {
		return "", gerror.New(gerror.ErrCodeOutOfRange, "max link recursion depth exceeded", nil).
			WithComponent("prompt_link_parser").
			WithOperation("ParseMarkdownWithLinks").
			WithDetails("maxDepth", maxDepth).
			WithDetails("path", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to open file").
			WithComponent("prompt_link_parser").
			WithOperation("OpenFile").
			WithDetails("path", path)
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
					return "", gerror.Wrap(err, gerror.ErrCodeInternal, "error expanding link").
						WithComponent("prompt_link_parser").
						WithOperation("ExpandLink").
						WithDetails("linkPath", linkPath)
				}
				expandedLine = strings.ReplaceAll(expandedLine, "@"+linkPath, expandedContent)
			}
		}
		output.WriteString(expandedLine)
		output.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "error reading file").
			WithComponent("prompt_link_parser").
			WithOperation("ReadFile").
			WithDetails("path", path)
	}
	return output.String(), nil
}
