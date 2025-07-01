// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"regexp"
	"strings"
)

// WikiLinkPattern is a regex to match wikilinks in markdown
var WikiLinkPattern = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// MarkdownLinkPattern is a regex to match regular markdown links
var MarkdownLinkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// ExtractLinks extracts both wikilinks and regular markdown links from content
func ExtractLinks(content string) []string {
	if content == "" {
		return []string{}
	}

	var links []string

	// Extract WikiLinks
	wikiMatches := WikiLinkPattern.FindAllStringSubmatch(content, -1)
	for _, match := range wikiMatches {
		if len(match) > 1 {
			link := strings.TrimSpace(match[1])
			if link != "" && !contains(links, link) {
				links = append(links, link)
			}
		}
	}

	// Extract regular markdown links
	markdownMatches := MarkdownLinkPattern.FindAllStringSubmatch(content, -1)
	for _, match := range markdownMatches {
		if len(match) > 1 {
			link := strings.TrimSpace(match[1]) // Use link text, not URL
			if link != "" && !contains(links, link) {
				links = append(links, link)
			}
		}
	}

	return links
}

// AddLinks adds wikilinks to a document
func AddLinks(doc *CorpusDoc, links []string) {
	for _, link := range links {
		if !contains(doc.Links, link) {
			doc.Links = append(doc.Links, link)
		}
	}
}

// SetDocumentLinks replaces all links in a document
func SetDocumentLinks(doc *CorpusDoc, links []string) {
	doc.Links = links
}

// ReplaceLinks replaces wikilinks in content with a new title
func ReplaceLinks(content, oldTitle, newTitle string) string {
	return strings.ReplaceAll(
		content,
		"[["+oldTitle+"]]",
		"[["+newTitle+"]]",
	)
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// UpdateDocumentLinks extracts and updates wikilinks in a document
func UpdateDocumentLinks(doc *CorpusDoc) {
	// Extract links from content
	links := ExtractLinks(doc.Body)

	// Replace all links
	SetDocumentLinks(doc, links)
}
