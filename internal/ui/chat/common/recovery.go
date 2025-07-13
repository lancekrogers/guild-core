// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package common

// RecoveryCommandMsg is sent when the user responds to a recovery prompt
type RecoveryCommandMsg struct {
	Recover bool // true to recover, false to start fresh
}
