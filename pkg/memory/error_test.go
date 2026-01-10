// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package memory_test

import (
	"testing"

	"github.com/lancekrogers/guild-core/pkg/memory"
)

// TestStoreError tests the StoreError implementation
func TestStoreError(t *testing.T) {
	// Test creating a StoreError
	errMsg := "test error message"
	err := memory.StoreError{Message: errMsg}

	// Test Error() method
	if err.Error() != errMsg {
		t.Errorf("Expected error message '%s', got '%s'", errMsg, err.Error())
	}

	// Test ErrNotFound
	if memory.ErrNotFound.Error() != "item not found" {
		t.Errorf("Expected ErrNotFound message 'item not found', got '%s'", memory.ErrNotFound.Error())
	}
}
