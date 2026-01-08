// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package view

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// String returns the best-effort string representation of a Bubble Tea view.
func String(v tea.View) string {
	if v.Content == nil {
		return ""
	}
	if s, ok := v.Content.(fmt.Stringer); ok {
		return s.String()
	}
	return fmt.Sprintf("%v", v.Content)
}
