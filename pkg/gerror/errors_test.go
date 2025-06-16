// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package gerror

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuildError_Basic(t *testing.T) {
	t.Run("New creates error with correct fields", func(t *testing.T) {
		err := New(ErrCodeValidation, "validation failed", nil)

		assert.Equal(t, ErrCodeValidation, err.Code)
		assert.Equal(t, "validation failed", err.Message)
		assert.Nil(t, err.Cause)
		assert.NotEmpty(t, err.Stack)
		assert.NotZero(t, err.Timestamp)
		assert.False(t, err.Retryable)
		assert.True(t, err.UserSafe) // Validation errors are user-safe
	})

	t.Run("Newf formats message", func(t *testing.T) {
		err := Newf(ErrCodeInvalidInput, "invalid field: %s", "email")

		assert.Equal(t, "invalid field: email", err.Message)
	})

	t.Run("Error string includes code and message", func(t *testing.T) {
		err := New(ErrCodeInternal, "internal error", nil)

		assert.Contains(t, err.Error(), "GUILD-1000")
		assert.Contains(t, err.Error(), "internal error")
	})

	t.Run("Error string includes cause", func(t *testing.T) {
		cause := fmt.Errorf("database connection failed")
		err := New(ErrCodeStorage, "storage error", cause)

		assert.Contains(t, err.Error(), "storage error")
		assert.Contains(t, err.Error(), "database connection failed")
	})
}

func TestGuildError_Wrap(t *testing.T) {
	t.Run("Wrap nil returns nil", func(t *testing.T) {
		err := Wrap(nil, ErrCodeInternal, "should not happen")
		assert.Nil(t, err)
	})

	t.Run("Wrap standard error", func(t *testing.T) {
		cause := fmt.Errorf("original error")
		err := Wrap(cause, ErrCodeAgent, "agent failed")

		assert.Equal(t, ErrCodeAgent, err.Code)
		assert.Equal(t, "agent failed", err.Message)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("Wrap GuildError preserves context", func(t *testing.T) {
		original := New(ErrCodeTimeout, "timeout", nil)
		original.RequestID = "req-123"
		original.TraceID = "trace-456"
		original.Retryable = true

		wrapped := Wrap(original, ErrCodeAgent, "agent timeout")

		assert.Equal(t, ErrCodeAgent, wrapped.Code)
		assert.Equal(t, "req-123", wrapped.RequestID)
		assert.Equal(t, "trace-456", wrapped.TraceID)
		assert.True(t, wrapped.Retryable)
	})

	t.Run("Wrapf formats message", func(t *testing.T) {
		cause := fmt.Errorf("connection refused")
		err := Wrapf(cause, ErrCodeConnection, "failed to connect to %s", "database")

		assert.Equal(t, "failed to connect to database", err.Message)
	})
}

func TestGuildError_Details(t *testing.T) {
	t.Run("WithDetails adds details", func(t *testing.T) {
		err := New(ErrCodeValidation, "validation failed", nil).
			WithDetails("field", "email").
			WithDetails("value", "invalid@")

		assert.Equal(t, "email", err.Details["field"])
		assert.Equal(t, "invalid@", err.Details["value"])
	})

	t.Run("WithComponent sets component", func(t *testing.T) {
		err := New(ErrCodeInternal, "error", nil).
			WithComponent("agent-manager")

		assert.Equal(t, "agent-manager", err.Component)
	})

	t.Run("WithOperation sets operation", func(t *testing.T) {
		err := New(ErrCodeInternal, "error", nil).
			WithOperation("create-task")

		assert.Equal(t, "create-task", err.Operation)
	})
}

func TestGuildError_Context(t *testing.T) {
	t.Run("FromContext extracts IDs", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "request_id", "req-789")
		ctx = context.WithValue(ctx, "trace_id", "trace-101112")

		err := New(ErrCodeInternal, "error", nil).FromContext(ctx)

		assert.Equal(t, "req-789", err.RequestID)
		assert.Equal(t, "trace-101112", err.TraceID)
	})
}

func TestGuildError_Is(t *testing.T) {
	t.Run("Is matches same error code", func(t *testing.T) {
		err1 := New(ErrCodeNotFound, "not found", nil)
		err2 := New(ErrCodeNotFound, "different message", nil)

		assert.True(t, err1.Is(err2))
	})

	t.Run("Is does not match different error code", func(t *testing.T) {
		err1 := New(ErrCodeNotFound, "not found", nil)
		err2 := New(ErrCodeInternal, "internal", nil)

		assert.False(t, err1.Is(err2))
	})

	t.Run("Is matches wrapped error", func(t *testing.T) {
		cause := fmt.Errorf("original")
		err := New(ErrCodeInternal, "wrapped", cause)

		assert.True(t, errors.Is(err, cause))
	})

	t.Run("Is with nil target", func(t *testing.T) {
		err := New(ErrCodeInternal, "error", nil)

		assert.False(t, err.Is(nil))
	})
}

func TestGuildError_As(t *testing.T) {
	t.Run("As extracts GuildError", func(t *testing.T) {
		original := New(ErrCodeAgent, "agent error", nil)
		wrapped := fmt.Errorf("wrapped: %w", original)

		var gerr *GuildError
		require.True(t, errors.As(wrapped, &gerr))
		assert.Equal(t, ErrCodeAgent, gerr.Code)
	})
}

func TestGuildError_Retryable(t *testing.T) {
	tests := []struct {
		code      ErrorCode
		retryable bool
	}{
		{ErrCodeTimeout, true},
		{ErrCodeCancelled, true},
		{ErrCodeRateLimit, true},
		{ErrCodeConnection, true},
		{ErrCodeAgentTimeout, true},
		{ErrCodeProviderTimeout, true},
		{ErrCodeInternal, false},
		{ErrCodeValidation, false},
		{ErrCodeNotFound, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := New(tt.code, "test", nil)
			assert.Equal(t, tt.retryable, err.Retryable)
		})
	}
}

func TestGuildError_UserSafe(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		userSafe bool
	}{
		{ErrCodeValidation, true},
		{ErrCodeInvalidInput, true},
		{ErrCodeMissingRequired, true},
		{ErrCodeInvalidFormat, true},
		{ErrCodeOutOfRange, true},
		{ErrCodeNotFound, true},
		{ErrCodeInternal, false},
		{ErrCodePanic, false},
		{ErrCodeStorage, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := New(tt.code, "test", nil)
			assert.Equal(t, tt.userSafe, err.UserSafe)
		})
	}
}

func TestGuildError_Stack(t *testing.T) {
	t.Run("Stack captures call frames", func(t *testing.T) {
		err := New(ErrCodeInternal, "error", nil)

		require.NotEmpty(t, err.Stack)

		// Check that stack contains this test function
		found := false
		for _, frame := range err.Stack {
			if frame.Function == "github.com/guild-ventures/guild-core/pkg/gerror.TestGuildError_Stack.func1" {
				found = true
				break
			}
		}
		assert.True(t, found, "Stack should contain test function")
	})
}

func TestGuildError_JSON(t *testing.T) {
	t.Run("MarshalJSON includes all fields", func(t *testing.T) {
		err := New(ErrCodeValidation, "validation failed", nil).
			WithDetails("field", "email").
			WithComponent("validator").
			WithOperation("validate-user")
		err.RequestID = "req-123"

		data, jsonErr := err.MarshalJSON()
		require.NoError(t, jsonErr)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"code":"GUILD-2000"`)
		assert.Contains(t, jsonStr, `"message":"validation failed"`)
		assert.Contains(t, jsonStr, `"error":"[GUILD-2000] validation failed"`)
		assert.Contains(t, jsonStr, `"component":"validator"`)
		assert.Contains(t, jsonStr, `"operation":"validate-user"`)
		assert.Contains(t, jsonStr, `"request_id":"req-123"`)
		assert.Contains(t, jsonStr, `"details":{"field":"email"}`)
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("Is with error code", func(t *testing.T) {
		err := New(ErrCodeNotFound, "not found", nil)
		wrapped := fmt.Errorf("wrapped: %w", err)

		assert.True(t, Is(wrapped, ErrCodeNotFound))
		assert.False(t, Is(wrapped, ErrCodeInternal))
	})

	t.Run("Is with sentinel error", func(t *testing.T) {
		wrapped := fmt.Errorf("wrapped: %w", ErrNotFound)

		assert.True(t, Is(wrapped, ErrNotFound))
		assert.False(t, Is(wrapped, ErrTimeout))
	})

	t.Run("GetCode extracts error code", func(t *testing.T) {
		err := New(ErrCodeAgent, "agent error", nil)
		wrapped := fmt.Errorf("wrapped: %w", err)

		assert.Equal(t, ErrCodeAgent, GetCode(wrapped))
	})

	t.Run("GetCode returns internal for non-GuildError", func(t *testing.T) {
		err := fmt.Errorf("standard error")

		assert.Equal(t, ErrCodeInternal, GetCode(err))
	})

	t.Run("IsRetryable checks retryable status", func(t *testing.T) {
		retryable := New(ErrCodeTimeout, "timeout", nil)
		notRetryable := New(ErrCodeValidation, "validation", nil)

		assert.True(t, IsRetryable(retryable))
		assert.False(t, IsRetryable(notRetryable))
	})

	t.Run("IsUserSafe checks user-safe status", func(t *testing.T) {
		userSafe := New(ErrCodeValidation, "validation", nil)
		notUserSafe := New(ErrCodeInternal, "internal", nil)

		assert.True(t, IsUserSafe(userSafe))
		assert.False(t, IsUserSafe(notUserSafe))
	})
}
