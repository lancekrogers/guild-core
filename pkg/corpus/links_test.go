package corpus

import (
	"reflect"
	"testing"
)

func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "No links",
			content:  "This is a document with no links.",
			expected: []string{},
		},
		{
			name:     "Single link",
			content:  "This document links to [[Another Document]].",
			expected: []string{"Another Document"},
		},
		{
			name:     "Multiple links",
			content:  "This document links to [[Document 1]] and [[Document 2]].",
			expected: []string{"Document 1", "Document 2"},
		},
		{
			name:     "Link with special characters",
			content:  "This links to [[Document with spaces & special-chars]].",
			expected: []string{"Document with spaces & special-chars"},
		},
		{
			name:     "Duplicate links",
			content:  "This links to [[Same Document]] twice: [[Same Document]].",
			expected: []string{"Same Document"},
		},
		{
			name:     "Links across lines",
			content:  "This links to [[Document 1]].\nAnd this links to [[Document 2]].",
			expected: []string{"Document 1", "Document 2"},
		},
		{
			name:     "Mixed case links",
			content:  "Links to [[Document]] and [[DOCUMENT]] are case sensitive.",
			expected: []string{"Document", "DOCUMENT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractLinks(tt.content)

			// Special handling for comparing empty slices
			if (len(result) == 0 && len(tt.expected) == 0) {
				// Both are empty, test passes
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ExtractLinks() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAddLinks(t *testing.T) {
	tests := []struct {
		name     string
		doc      *CorpusDoc
		links    []string
		expected []string
	}{
		{
			name: "Add to empty links",
			doc: &CorpusDoc{
				Links: []string{},
			},
			links:    []string{"Document 1", "Document 2"},
			expected: []string{"Document 1", "Document 2"},
		},
		{
			name: "Add to existing links",
			doc: &CorpusDoc{
				Links: []string{"Existing Document"},
			},
			links:    []string{"Document 1", "Document 2"},
			expected: []string{"Existing Document", "Document 1", "Document 2"},
		},
		{
			name: "Add duplicate links",
			doc: &CorpusDoc{
				Links: []string{"Existing Document"},
			},
			links:    []string{"Existing Document", "Document 1"},
			expected: []string{"Existing Document", "Document 1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddLinks(tt.doc, tt.links)
			if !reflect.DeepEqual(tt.doc.Links, tt.expected) {
				t.Errorf("AddLinks() = %v, want %v", tt.doc.Links, tt.expected)
			}
		})
	}
}

func TestReplaceLinks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		oldTitle string
		newTitle string
		expected string
	}{
		{
			name:     "No links",
			content:  "This is a document with no links.",
			oldTitle: "Old Document",
			newTitle: "New Document",
			expected: "This is a document with no links.",
		},
		{
			name:     "Replace single link",
			content:  "This document links to [[Old Document]].",
			oldTitle: "Old Document",
			newTitle: "New Document",
			expected: "This document links to [[New Document]].",
		},
		{
			name:     "Replace multiple links",
			content:  "This document links to [[Old Document]] twice: [[Old Document]].",
			oldTitle: "Old Document",
			newTitle: "New Document",
			expected: "This document links to [[New Document]] twice: [[New Document]].",
		},
		{
			name:     "Replace mixed with others",
			content:  "Links to [[Old Document]] and [[Another Document]].",
			oldTitle: "Old Document",
			newTitle: "New Document",
			expected: "Links to [[New Document]] and [[Another Document]].",
		},
		{
			name:     "Case sensitive replacement",
			content:  "Links to [[old document]] and [[Old Document]].",
			oldTitle: "Old Document",
			newTitle: "New Document",
			expected: "Links to [[old document]] and [[New Document]].",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplaceLinks(tt.content, tt.oldTitle, tt.newTitle)
			if result != tt.expected {
				t.Errorf("ReplaceLinks() = %v, want %v", result, tt.expected)
			}
		})
	}
}