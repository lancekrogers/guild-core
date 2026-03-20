// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"context"

	"github.com/lancekrogers/guild-core/pkg/corpus"
)

// CorpusManager interface abstracts corpus operations for UI
type CorpusManager interface {
	LoadCorpus(ctx context.Context, config corpus.Config) error
	ScanDocuments(ctx context.Context) ([]corpus.CorpusDoc, error)
	GetDocument(ctx context.Context, path string) (*corpus.CorpusDoc, error)
	SearchDocuments(ctx context.Context, query string) ([]corpus.CorpusDoc, error)
	GetTags(ctx context.Context) ([]string, error)
	GetDocumentsByTag(ctx context.Context, tag string) ([]corpus.CorpusDoc, error)
	GetGraph(ctx context.Context) (corpus.Graph, error)
	RefreshCorpus(ctx context.Context) error
	GetBacklinks(ctx context.Context, documentPath string) ([]string, error)
}

// CorpusConfig interface abstracts configuration for corpus operations
type CorpusConfig interface {
	GetCorpusPath() string
	GetIncludePatterns() []string
	GetExcludePatterns() []string
	GetMaxDepth() int
	IsRecursive() bool
	GetUser() string
}
