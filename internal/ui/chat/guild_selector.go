// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"

	"github.com/guild-ventures/guild-core/internal/ui/chat/selector"
)

// RunGuildSelector runs the guild selector UI and returns the selected guild name
func RunGuildSelector(ctx context.Context) (string, error) {
	return selector.RunGuildSelector(ctx)
}