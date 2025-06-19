// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

// ConfiguredProvider represents a configured provider with selected models
type ConfiguredProvider struct {
	Name     string
	Type     string
	Models   []ModelInfo
	Settings map[string]string
}