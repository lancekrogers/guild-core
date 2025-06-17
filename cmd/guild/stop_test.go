// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStopCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		// Test that the command is properly configured
		assert.Equal(t, "stop", stopCmd.Use)
		assert.NotEmpty(t, stopCmd.Short)
		assert.NotEmpty(t, stopCmd.Long)
		
		// Check flags
		flag := stopCmd.Flag("campaign")
		require.NotNil(t, flag)
		assert.Equal(t, "string", flag.Value.Type())
		
		flag = stopCmd.Flag("all")
		require.NotNil(t, flag)
		assert.Equal(t, "bool", flag.Value.Type())
		
		flag = stopCmd.Flag("session")
		require.NotNil(t, flag)
		assert.Equal(t, "int", flag.Value.Type())
		
		flag = stopCmd.Flag("force")
		require.NotNil(t, flag)
		assert.Equal(t, "bool", flag.Value.Type())
		assert.Equal(t, "f", flag.Shorthand)
		
		flag = stopCmd.Flag("timeout")
		require.NotNil(t, flag)
		assert.Equal(t, "duration", flag.Value.Type())
	})
}

func TestStatusCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		// Test that the command is properly configured
		assert.Equal(t, "status", statusCmd.Use)
		assert.NotEmpty(t, statusCmd.Short)
		assert.NotEmpty(t, statusCmd.Long)
		
		// Check flags
		flag := statusCmd.Flag("all")
		require.NotNil(t, flag)
		assert.Equal(t, "bool", flag.Value.Type())
	})
}