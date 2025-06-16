// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package interfaces

import "context"

// LLMClient defines the interface for LLM clients
type LLMClient interface {
	// Complete generates completions for the given prompt
	Complete(ctx context.Context, prompt string) (string, error)
}
