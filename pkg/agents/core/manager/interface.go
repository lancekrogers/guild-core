// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package manager implements the Guild Master agent refinement capabilities
package manager

import (
	"context"
)

// CommissionRefiner handles the refinement of commissions into hierarchical file structures
type CommissionRefiner interface {
	// RefineCommission takes a high-level commission and creates a hierarchical file structure
	RefineCommission(ctx context.Context, commission Commission) (*RefinedCommission, error)
}

// ArchiveWriter handles writing the refined structure to the Archives (filesystem)
type ArchiveWriter interface {
	// WriteStructure writes the refined commission structure to the Archives
	WriteStructure(ctx context.Context, refined *RefinedCommission) error
}

// StructureValidator validates the generated file structure for Guild standards
type StructureValidator interface {
	// ValidateStructure checks if the structure meets Guild requirements
	ValidateStructure(structure *FileStructure) error
}

// Commission represents a high-level guild commission to be refined
type Commission struct {
	ID          string
	Title       string
	Description string
	Domain      string // web-app, cli-tool, library, microservice
	Context     map[string]interface{}
}

// RefinedCommission represents the output of the refinement process
type RefinedCommission struct {
	CommissionID string
	Structure    *FileStructure
	Metadata     map[string]interface{}
}

// FileStructure represents the hierarchical file structure
type FileStructure struct {
	RootDir string
	Files   []*FileEntry
}

// FileEntry represents a file in the structure
type FileEntry struct {
	Path       string
	Content    string
	Type       FileType
	TasksCount int
	Metadata   map[string]interface{}
}

// FileType represents the type of file
type FileType string

const (
	FileTypeMarkdown FileType = "markdown"
	FileTypeManifest FileType = "manifest"
)

// ArtisanRequest represents a request to an Artisan (LLM) for refinement
type ArtisanRequest struct {
	SystemPrompt string
	UserPrompt   string
	Temperature  float32
	MaxTokens    int
}

// ArtisanResponse represents the Artisan's response
type ArtisanResponse struct {
	Content  string
	Metadata map[string]interface{}
}

// ArtisanClient interface for interacting with Guild Artisans (language models)
type ArtisanClient interface {
	Complete(ctx context.Context, request ArtisanRequest) (*ArtisanResponse, error)
}

// ResponseParser parses Artisan responses into file structures for the Archives
type ResponseParser interface {
	// ParseResponse parses an Artisan response into a file structure
	ParseResponse(response *ArtisanResponse) (*FileStructure, error)
	// ParseResponseWithContext parses an Artisan response with context support
	ParseResponseWithContext(ctx context.Context, response *ArtisanResponse) (*FileStructure, error)
}
