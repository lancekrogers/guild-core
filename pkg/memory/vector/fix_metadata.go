// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package vector

// MetadataConverter converts an interface{} metadata to map[string]interface{}
func MetadataConverter(metadata interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Convert based on type
	if meta, ok := metadata.(map[string]interface{}); ok {
		// Already the right type
		return meta
	} else if meta, ok := metadata.(map[string]string); ok {
		// Convert string map to interface map
		for k, v := range meta {
			result[k] = v
		}
	}

	return result
}
