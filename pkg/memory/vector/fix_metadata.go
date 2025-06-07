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
