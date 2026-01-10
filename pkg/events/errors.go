// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package events

import (
	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// Event validation errors
var (
	ErrMissingEventID        = gerror.New(gerror.ErrCodeValidation, "event ID is required", nil)
	ErrMissingEventType      = gerror.New(gerror.ErrCodeValidation, "event type is required", nil)
	ErrMissingEventSource    = gerror.New(gerror.ErrCodeValidation, "event source is required", nil)
	ErrMissingEventTimestamp = gerror.New(gerror.ErrCodeValidation, "event timestamp is required", nil)
)

// EventBus operation errors
var (
	ErrEventBusNotInitialized = gerror.New(gerror.ErrCodeInternal, "event bus not initialized", nil)
	ErrInvalidSubscriptionID  = gerror.New(gerror.ErrCodeValidation, "invalid subscription ID", nil)
	ErrHandlerPanic           = gerror.New(gerror.ErrCodeInternal, "event handler panicked", nil)
	ErrEventBusClosed         = gerror.New(gerror.ErrCodeInternal, "event bus is closed", nil)
)

// Event publishing errors
var (
	ErrInvalidEventType = gerror.New(gerror.ErrCodeValidation, "invalid event type", nil)
	ErrEventTooLarge    = gerror.New(gerror.ErrCodeValidation, "event payload too large", nil)
	ErrPublishTimeout   = gerror.New(gerror.ErrCodeTimeout, "event publish timeout", nil)
)

// Event subscription errors
var (
	ErrInvalidHandler       = gerror.New(gerror.ErrCodeValidation, "invalid event handler", nil)
	ErrSubscriptionNotFound = gerror.New(gerror.ErrCodeNotFound, "subscription not found", nil)
	ErrTooManySubscriptions = gerror.New(gerror.ErrCodeResourceExhausted, "too many subscriptions", nil)
)

// Event conversion errors
var (
	ErrEventConversion    = gerror.New(gerror.ErrCodeInternal, "event conversion failed", nil)
	ErrInvalidJSON        = gerror.New(gerror.ErrCodeValidation, "invalid JSON event", nil)
	ErrUnknownEventFormat = gerror.New(gerror.ErrCodeValidation, "unknown event format", nil)
)
