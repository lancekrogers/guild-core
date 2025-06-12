// Package gerror provides production-ready error handling for the Guild framework.
// It includes structured errors, error codes, and integration with observability.
package gerror

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"
)

// ErrorCode represents categorized error codes for the Guild framework
type ErrorCode string

const (
	// System errors (1xxx)
	ErrCodeInternal       ErrorCode = "GUILD-1000"
	ErrCodePanic          ErrorCode = "GUILD-1001"
	ErrCodeTimeout        ErrorCode = "GUILD-1002"
	ErrCodeCancelled      ErrorCode = "GUILD-1003"
	ErrCodeRateLimit      ErrorCode = "GUILD-1004"
	ErrCodeResourceLimit  ErrorCode = "GUILD-1005"
	ErrCodeNotImplemented ErrorCode = "GUILD-1006"

	// Validation errors (2xxx)
	ErrCodeValidation      ErrorCode = "GUILD-2000"
	ErrCodeInvalidInput    ErrorCode = "GUILD-2001"
	ErrCodeMissingRequired ErrorCode = "GUILD-2002"
	ErrCodeInvalidFormat   ErrorCode = "GUILD-2003"
	ErrCodeOutOfRange      ErrorCode = "GUILD-2004"
	ErrCodeConfiguration   ErrorCode = "GUILD-2005"

	// Storage errors (3xxx)
	ErrCodeStorage       ErrorCode = "GUILD-3000"
	ErrCodeNotFound      ErrorCode = "GUILD-3001"
	ErrCodeAlreadyExists ErrorCode = "GUILD-3002"
	ErrCodeTransaction   ErrorCode = "GUILD-3003"
	ErrCodeConnection    ErrorCode = "GUILD-3004"

	// Agent errors (4xxx)
	ErrCodeAgent            ErrorCode = "GUILD-4000"
	ErrCodeAgentNotFound    ErrorCode = "GUILD-4001"
	ErrCodeAgentBusy        ErrorCode = "GUILD-4002"
	ErrCodeAgentFailed      ErrorCode = "GUILD-4003"
	ErrCodeAgentTimeout     ErrorCode = "GUILD-4004"
	ErrCodeNoAvailableAgent ErrorCode = "GUILD-4005"

	// Provider errors (5xxx)
	ErrCodeProvider        ErrorCode = "GUILD-5000"
	ErrCodeProviderAPI     ErrorCode = "GUILD-5001"
	ErrCodeProviderAuth    ErrorCode = "GUILD-5002"
	ErrCodeProviderQuota   ErrorCode = "GUILD-5003"
	ErrCodeProviderTimeout ErrorCode = "GUILD-5004"

	// Task/Orchestration errors (6xxx)
	ErrCodeOrchestration      ErrorCode = "GUILD-6000"
	ErrCodeTaskFailed         ErrorCode = "GUILD-6001"
	ErrCodeInvalidTransition  ErrorCode = "GUILD-6002"
	ErrCodeDependencyFailed   ErrorCode = "GUILD-6003"
	ErrCodeCircularDependency ErrorCode = "GUILD-6004"
	
	// External service errors (7xxx)
	ErrCodeExternal        ErrorCode = "GUILD-7000"
	ErrCodeExternalTimeout ErrorCode = "GUILD-7001"
	ErrCodeExternalAuth    ErrorCode = "GUILD-7002"
	
	// I/O errors (8xxx)
	ErrCodeIO      ErrorCode = "GUILD-8000"
	ErrCodeParsing ErrorCode = "GUILD-8001"
)

// GuildError is the standard error type for the Guild framework
type GuildError struct {
	Code      ErrorCode              `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Cause     error                  `json:"-"`
	Stack     []StackFrame           `json:"stack,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	Component string                 `json:"component,omitempty"`
	Operation string                 `json:"operation,omitempty"`
	Retryable bool                   `json:"retryable"`
	UserSafe  bool                   `json:"user_safe"`
}

// StackFrame represents a single frame in the error stack trace
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// Error implements the error interface
func (e *GuildError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap implements the errors.Unwrap interface
func (e *GuildError) Unwrap() error {
	return e.Cause
}

// Is implements errors.Is interface
func (e *GuildError) Is(target error) bool {
	if target == nil {
		return false
	}

	// Check if target is a GuildError with the same code
	if gerr, ok := target.(*GuildError); ok {
		return e.Code == gerr.Code
	}

	// Check wrapped error
	return errors.Is(e.Cause, target)
}

// WithDetails adds details to the error
func (e *GuildError) WithDetails(key string, value interface{}) *GuildError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithComponent sets the component that generated the error
func (e *GuildError) WithComponent(component string) *GuildError {
	e.Component = component
	return e
}

// WithOperation sets the operation that was being performed
func (e *GuildError) WithOperation(operation string) *GuildError {
	e.Operation = operation
	return e
}

// IsRetryable returns whether the error is retryable
func (e *GuildError) IsRetryable() bool {
	return e.Retryable
}

// IsUserSafe returns whether the error message is safe to show to users
func (e *GuildError) IsUserSafe() bool {
	return e.UserSafe
}

// MarshalJSON implements json.Marshaler
func (e *GuildError) MarshalJSON() ([]byte, error) {
	type Alias GuildError
	return json.Marshal(&struct {
		*Alias
		Error string `json:"error"`
	}{
		Alias: (*Alias)(e),
		Error: e.Error(),
	})
}

// New creates a new GuildError
func New(code ErrorCode, message string, cause error) *GuildError {
	err := &GuildError{
		Code:      code,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
		Stack:     captureStack(2), // Skip New and the caller
	}

	// Set retryable based on error code
	switch code {
	case ErrCodeTimeout, ErrCodeCancelled, ErrCodeRateLimit,
		ErrCodeConnection, ErrCodeAgentTimeout, ErrCodeProviderTimeout:
		err.Retryable = true
	}

	// Set user-safe for certain error types
	switch code {
	case ErrCodeValidation, ErrCodeInvalidInput, ErrCodeMissingRequired,
		ErrCodeInvalidFormat, ErrCodeOutOfRange, ErrCodeConfiguration, ErrCodeNotFound:
		err.UserSafe = true
	}

	return err
}

// Newf creates a new GuildError with formatted message
func Newf(code ErrorCode, format string, args ...interface{}) *GuildError {
	return New(code, fmt.Sprintf(format, args...), nil)
}

// Wrap wraps an existing error with a GuildError
func Wrap(err error, code ErrorCode, message string) *GuildError {
	if err == nil {
		return nil
	}

	// If already a GuildError, preserve some information
	if gerr, ok := err.(*GuildError); ok {
		newErr := New(code, message, err)
		// Preserve request and trace IDs
		newErr.RequestID = gerr.RequestID
		newErr.TraceID = gerr.TraceID
		// Preserve retryable status if the new error doesn't override it
		if !newErr.Retryable && gerr.Retryable {
			newErr.Retryable = true
		}
		return newErr
	}

	return New(code, message, err)
}

// Wrapf wraps an existing error with a formatted message
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *GuildError {
	return Wrap(err, code, fmt.Sprintf(format, args...))
}

// FromContext extracts request and trace IDs from context and adds them to the error
func (e *GuildError) FromContext(ctx context.Context) *GuildError {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		e.RequestID = requestID
	}
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		e.TraceID = traceID
	}
	return e
}

// captureStack captures the current stack trace
func captureStack(skip int) []StackFrame {
	var frames []StackFrame

	// Capture up to 10 frames
	pcs := make([]uintptr, 10)
	n := runtime.Callers(skip+1, pcs)

	if n > 0 {
		frames = make([]StackFrame, 0, n)
		callFrames := runtime.CallersFrames(pcs[:n])

		for {
			frame, more := callFrames.Next()
			frames = append(frames, StackFrame{
				Function: frame.Function,
				File:     frame.File,
				Line:     frame.Line,
			})

			if !more {
				break
			}
		}
	}

	return frames
}

// Sentinel errors for common cases
var (
	// Storage errors
	ErrNotFound      = New(ErrCodeNotFound, "resource not found", nil)
	ErrAlreadyExists = New(ErrCodeAlreadyExists, "resource already exists", nil)

	// Agent errors
	ErrNoAgentAvailable = New(ErrCodeNoAvailableAgent, "no agent available for task", nil)
	ErrAgentNotFound    = New(ErrCodeAgentNotFound, "agent not found", nil)

	// System errors
	ErrTimeout   = New(ErrCodeTimeout, "operation timed out", nil)
	ErrCancelled = New(ErrCodeCancelled, "operation cancelled", nil)
)

// Is checks if an error matches a target error or error code
func Is(err error, target interface{}) bool {
	if err == nil {
		return false
	}

	// Check if target is an error
	if targetErr, ok := target.(error); ok {
		return errors.Is(err, targetErr)
	}

	// Check if target is an error code
	if code, ok := target.(ErrorCode); ok {
		var gerr *GuildError
		if errors.As(err, &gerr) {
			return gerr.Code == code
		}
	}

	return false
}

// As finds the first error in err's chain that matches target
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// GetCode extracts the error code from an error
func GetCode(err error) ErrorCode {
	var gerr *GuildError
	if errors.As(err, &gerr) {
		return gerr.Code
	}
	return ErrCodeInternal
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	var gerr *GuildError
	if errors.As(err, &gerr) {
		return gerr.IsRetryable()
	}
	return false
}

// IsUserSafe checks if an error is safe to show to users
func IsUserSafe(err error) bool {
	var gerr *GuildError
	if errors.As(err, &gerr) {
		return gerr.IsUserSafe()
	}
	return false
}
