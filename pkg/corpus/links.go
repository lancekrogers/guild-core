package corpus

import (
	"regexp"
	"strings"
)

// WikiLinkPattern is a regex to match wikilinks in markdown
var WikiLinkPattern = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// ExtractLinks extracts wikilinks from markdown content
func ExtractLinks(content string) []string {
	if content == "" {
		return []string{}
	}

	matches := WikiLinkPattern.FindAllStringSubmatch(content, -1)
	var links []string

	for _, match := range matches {
		if len(match) > 1 {
			link := strings.TrimSpace(match[1])
			// Check if the link is not empty and not already added
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
