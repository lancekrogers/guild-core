// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package lsp

// Common types used across LSP tools

// LocationParams represents parameters for location-based tools
type LocationParams struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// Note: Additional common types can be added here as needed when
// the LSP manager is extended with the new operations
