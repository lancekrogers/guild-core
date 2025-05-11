package corpus

import (
	"regexp"
	"strings"
)

// wikiLinkPattern matches Obsidian-style [[wikilinks]]
var wikiLinkPattern = regexp.MustCompile(`\[\[([^\[\]]+)\]\]`)

// ExtractLinks finds wikilinks in content
func ExtractLinks(content string) []string {
	// Extract all wikilink matches
	matches := wikiLinkPattern.FindAllStringSubmatch(content, -1)

	// Use a map to deduplicate links
	uniqueLinks := make(map[string]struct{})
	for _, match := range matches {
		if len(match) > 1 {
			// Clean the link (trim spaces, convert to lowercase)
			link := strings.TrimSpace(match[1])
			link = strings.ToLower(link)
			
			// Skip empty links
			if link == "" {
				continue
			}
			
			uniqueLinks[link] = struct{}{}
		}
	}

	// Convert map keys to slice
	links := make([]string, 0, len(uniqueLinks))
	for link := range uniqueLinks {
		links = append(links, link)
	}

	return links
}

// Autolink finds potential links in a document
func Autolink(doc *CorpusDoc) error {
	if doc == nil {
		return nil
	}

	// Extract links from document body
	links := ExtractLinks(doc.Body)
	doc.Links = links

	return nil
}

// InsertWikilink adds a wikilink to the document body at the specified position
func InsertWikilink(body string, link string, position int) string {
	if position < 0 || position > len(body) {
		return body // Invalid position
	}

	wikilink := "[[" + link + "]]"
	newBody := body[:position] + wikilink + body[position:]
	return newBody
}

// ReplaceWikilink replaces an existing wikilink with a new one
func ReplaceWikilink(body string, oldLink, newLink string) string {
	oldWikilink := "[[" + oldLink + "]]"
	newWikilink := "[[" + newLink + "]]"
	return strings.ReplaceAll(body, oldWikilink, newWikilink)
}

// AddTagsToFrontmatter adds tags to a document's frontmatter
func AddTagsToFrontmatter(doc *CorpusDoc, newTags []string) {
	if doc == nil {
		return
	}

	// Create a map of existing tags for quick lookup
	existingTags := make(map[string]struct{})
	for _, tag := range doc.Tags {
		existingTags[tag] = struct{}{}
	}

	// Add new tags if they don't already exist
	for _, tag := range newTags {
		if _, exists := existingTags[tag]; !exists {
			doc.Tags = append(doc.Tags, tag)
		}
	}
}

// RemoveTagsFromFrontmatter removes tags from a document's frontmatter
func RemoveTagsFromFrontmatter(doc *CorpusDoc, tagsToRemove []string) {
	if doc == nil {
		return
	}

	// Create a map of tags to remove for quick lookup
	removeMap := make(map[string]struct{})
	for _, tag := range tagsToRemove {
		removeMap[tag] = struct{}{}
	}

	// Create a new slice with only the tags we want to keep
	newTags := make([]string, 0, len(doc.Tags))
	for _, tag := range doc.Tags {
		if _, shouldRemove := removeMap[tag]; !shouldRemove {
			newTags = append(newTags, tag)
		}
	}

	doc.Tags = newTags
}

// FindBacklinks gets a list of documents that link to the given document
func FindBacklinks(cfg Config, docName string) ([]string, error) {
	if cfg.Location == "" {
		return nil, nil
	}

	// Get all documents in the corpus
	docs, err := List(cfg)
	if err != nil {
		return nil, err
	}

	// Check each document for links to our target
	var backlinks []string
	for _, path := range docs {
		doc, err := Load(path)
		if err != nil {
			continue // Skip documents that can't be loaded
		}

		// Check if this document links to our target
		for _, link := range doc.Links {
			if strings.ToLower(link) == strings.ToLower(docName) {
				// This document links to our target
				backlinks = append(backlinks, path)
				break
			}
		}
	}

	return backlinks, nil
}