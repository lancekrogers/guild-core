// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package protocol

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

const JSONRPCVersion = "2.0"

// IDGenerator generates unique IDs for JSON-RPC requests
type IDGenerator interface {
	NextID() interface{}
}

// NumericIDGenerator generates numeric IDs
type NumericIDGenerator struct {
	counter uint64
}

// NextID returns the next numeric ID
func (g *NumericIDGenerator) NextID() interface{} {
	return atomic.AddUint64(&g.counter, 1)
}

// StringIDGenerator generates string IDs
type StringIDGenerator struct {
	prefix  string
	counter uint64
}

// NextID returns the next string ID
func (g *StringIDGenerator) NextID() interface{} {
	count := atomic.AddUint64(&g.counter, 1)
	return fmt.Sprintf("%s-%d", g.prefix, count)
}

// Codec handles JSON-RPC message encoding/decoding
type Codec struct {
	idGen IDGenerator
	mu    sync.RWMutex
}

// NewCodec creates a new JSON-RPC codec
func NewCodec(idGen IDGenerator) *Codec {
	if idGen == nil {
		idGen = &NumericIDGenerator{}
	}
	return &Codec{
		idGen: idGen,
	}
}

// EncodeRequest encodes a method call as a JSON-RPC request
func (c *Codec) EncodeRequest(method string, params interface{}) (*Request, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal params").
			WithComponent("mcp_protocol").
			WithOperation("EncodeRequest")
	}

	return &Request{
		JSONRPC: JSONRPCVersion,
		ID:      c.idGen.NextID(),
		Method:  method,
		Params:  paramsJSON,
	}, nil
}

// EncodeResponse encodes a result as a JSON-RPC response
func (c *Codec) EncodeResponse(id interface{}, result interface{}) (*Response, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
			WithComponent("mcp_protocol").
			WithOperation("EncodeResponse")
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  resultJSON,
	}, nil
}

// EncodeError encodes an error as a JSON-RPC error response
func (c *Codec) EncodeError(id interface{}, code int, message string, data interface{}) (*Response, error) {
	var dataJSON json.RawMessage
	if data != nil {
		var err error
		dataJSON, err = json.Marshal(data)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal error data").
				WithComponent("mcp_protocol").
				WithOperation("EncodeError")
		}
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    dataJSON,
		},
	}, nil
}

// EncodeNotification encodes a notification
func (c *Codec) EncodeNotification(method string, params interface{}) (*Notification, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal params").
			WithComponent("mcp_protocol").
			WithOperation("EncodeNotification")
	}

	return &Notification{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  paramsJSON,
	}, nil
}

// DecodeMessage decodes a JSON-RPC message
func (c *Codec) DecodeMessage(data []byte) (interface{}, error) {
	// First, try to determine the message type
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, &Error{
			Code:    ErrorCodeParse,
			Message: "Parse error",
		}
	}

	// Check JSON-RPC version
	var version string
	if v, ok := raw["jsonrpc"]; ok {
		if err := json.Unmarshal(v, &version); err != nil || version != JSONRPCVersion {
			return nil, &Error{
				Code:    ErrorCodeInvalidRequest,
				Message: "Invalid JSON-RPC version",
			}
		}
	} else {
		return nil, &Error{
			Code:    ErrorCodeInvalidRequest,
			Message: "Missing JSON-RPC version",
		}
	}

	// Determine message type
	_, hasID := raw["id"]
	_, hasMethod := raw["method"]
	_, hasResult := raw["result"]
	_, hasError := raw["error"]

	switch {
	case hasMethod && hasID:
		// Request
		var req Request
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, &Error{
				Code:    ErrorCodeParse,
				Message: "Failed to parse request",
			}
		}
		return &req, nil

	case hasMethod && !hasID:
		// Notification
		var notif Notification
		if err := json.Unmarshal(data, &notif); err != nil {
			return nil, &Error{
				Code:    ErrorCodeParse,
				Message: "Failed to parse notification",
			}
		}
		return &notif, nil

	case (hasResult || hasError) && hasID:
		// Response
		var resp Response
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, &Error{
				Code:    ErrorCodeParse,
				Message: "Failed to parse response",
			}
		}
		return &resp, nil

	default:
		return nil, &Error{
			Code:    ErrorCodeInvalidRequest,
			Message: "Invalid message structure",
		}
	}
}

// Batch represents a batch of JSON-RPC messages
type Batch struct {
	Messages []json.RawMessage
}

// EncodeBatch encodes multiple messages as a batch
func (c *Codec) EncodeBatch(messages ...interface{}) ([]byte, error) {
	batch := make([]json.RawMessage, 0, len(messages))

	for _, msg := range messages {
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal message").
				WithComponent("mcp_protocol").
				WithOperation("EncodeBatch")
		}
		batch = append(batch, msgJSON)
	}

	return json.Marshal(batch)
}

// DecodeBatch decodes a batch of JSON-RPC messages
func (c *Codec) DecodeBatch(data []byte) ([]interface{}, error) {
	var batch []json.RawMessage
	if err := json.Unmarshal(data, &batch); err != nil {
		// Not a batch, try single message
		msg, err := c.DecodeMessage(data)
		if err != nil {
			return nil, err
		}
		return []interface{}{msg}, nil
	}

	messages := make([]interface{}, 0, len(batch))
	for _, raw := range batch {
		msg, err := c.DecodeMessage(raw)
		if err != nil {
			// Include parse errors in results
			messages = append(messages, err)
		} else {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}
