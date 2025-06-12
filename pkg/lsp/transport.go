package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/observability"
)

// Transport handles JSON-RPC communication with language servers
type Transport struct {
	reader  io.Reader
	writer  io.Writer
	scanner *bufio.Scanner
	mu      sync.Mutex

	// Request tracking
	pendingRequests map[int64]chan<- *Response
	requestMu       sync.Mutex
}

// NewTransport creates a new JSON-RPC transport
func NewTransport(reader io.Reader, writer io.Writer) *Transport {
	return &Transport{
		reader:          reader,
		writer:          writer,
		scanner:         bufio.NewScanner(reader),
		pendingRequests: make(map[int64]chan<- *Response),
	}
}

// Send sends a JSON-RPC request or notification
func (t *Transport) Send(ctx context.Context, req *Request) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Marshal request
	data, err := json.Marshal(req)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal request").
			WithComponent("lsp.transport").
			WithOperation("send").
			WithDetails("method", req.Method)
	}

	// Write header and content
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := t.writer.Write([]byte(header)); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write header").
			WithComponent("lsp.transport").
			WithOperation("send")
	}

	if _, err := t.writer.Write(data); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write content").
			WithComponent("lsp.transport").
			WithOperation("send")
	}

	return nil
}

// Call sends a request and waits for response
func (t *Transport) Call(ctx context.Context, req *Request) (*Response, error) {
	if req.ID == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "request must have ID for call", nil).
			WithComponent("lsp.transport").
			WithOperation("call")
	}

	// Create response channel
	respChan := make(chan *Response, 1)

	// Register pending request
	t.requestMu.Lock()
	t.pendingRequests[req.ID.Number] = respChan
	t.requestMu.Unlock()

	// Clean up on exit
	defer func() {
		t.requestMu.Lock()
		delete(t.pendingRequests, req.ID.Number)
		t.requestMu.Unlock()
	}()

	// Send request
	if err := t.Send(ctx, req); err != nil {
		return nil, err
	}

	// Wait for response
	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, gerror.Newf(gerror.ErrCodeExternal, "LSP error %d: %s", resp.Error.Code, resp.Error.Message).
				WithComponent("lsp.transport").
				WithOperation("call").
				WithDetails("method", req.Method)
		}
		return resp, nil

	case <-ctx.Done():
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeTimeout, "request cancelled").
			WithComponent("lsp.transport").
			WithOperation("call").
			WithDetails("method", req.Method)
	}
}

// Listen starts listening for responses and notifications
func (t *Transport) Listen(ctx context.Context) error {
	logger := observability.GetLogger(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read message
			msg, err := t.readMessage()
			if err != nil {
				if err == io.EOF {
					logger.InfoContext(ctx, "LSP transport connection closed")
					return nil
				}
				logger.ErrorContext(ctx, "Failed to read LSP message",
					"error", err)
				continue
			}

			// Handle message
			if err := t.handleMessage(ctx, msg); err != nil {
				logger.ErrorContext(ctx, "Failed to handle LSP message",
					"error", err)
			}
		}
	}
}

// readMessage reads a complete JSON-RPC message
func (t *Transport) readMessage() (json.RawMessage, error) {
	// Read headers
	headers := make(map[string]string)
	for {
		line, err := t.readLine()
		if err != nil {
			return nil, err
		}

		if line == "" {
			// Empty line signals end of headers
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, gerror.Newf(gerror.ErrCodeParsing, "invalid header: %s", line).
				WithComponent("lsp.transport").
				WithOperation("read_message")
		}

		headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	// Get content length
	lengthStr, ok := headers["Content-Length"]
	if !ok {
		return nil, gerror.New(gerror.ErrCodeParsing, "missing Content-Length header", nil).
			WithComponent("lsp.transport").
			WithOperation("read_message")
	}

	contentLength, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "invalid Content-Length").
			WithComponent("lsp.transport").
			WithOperation("read_message")
	}

	// Read content
	content := make([]byte, contentLength)
	if _, err := io.ReadFull(t.reader, content); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to read content").
			WithComponent("lsp.transport").
			WithOperation("read_message")
	}

	return json.RawMessage(content), nil
}

// readLine reads a line from the reader
func (t *Transport) readLine() (string, error) {
	if t.scanner.Scan() {
		return t.scanner.Text(), nil
	}

	if err := t.scanner.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeIO, "scanner error").
			WithComponent("lsp.transport").
			WithOperation("read_line")
	}

	return "", io.EOF
}

// handleMessage handles an incoming message
func (t *Transport) handleMessage(ctx context.Context, msg json.RawMessage) error {
	logger := observability.GetLogger(ctx)

	// Try to parse as response first
	var resp Response
	if err := json.Unmarshal(msg, &resp); err == nil && resp.ID != nil {
		// This is a response
		t.requestMu.Lock()
		if ch, ok := t.pendingRequests[resp.ID.Number]; ok {
			ch <- &resp
		} else {
			logger.WarnContext(ctx, "Received response for unknown request",
				"id", resp.ID.Number)
		}
		t.requestMu.Unlock()
		return nil
	}

	// Try to parse as request/notification
	var req Request
	if err := json.Unmarshal(msg, &req); err == nil {
		// This is a request or notification from the server
		// TODO: Handle server-initiated requests/notifications
		logger.DebugContext(ctx, "Received server request/notification",
			"method", req.Method)
		return nil
	}

	return gerror.New(gerror.ErrCodeParsing, "failed to parse LSP message", nil).
		WithComponent("lsp.transport").
		WithOperation("handle_message")
}

// ClientTransport provides a higher-level interface for clients
type ClientTransport struct {
	transport *Transport
	requestID int64
	mu        sync.Mutex
}

// NewClientTransport creates a new client transport
func NewClientTransport(reader io.Reader, writer io.Writer) *ClientTransport {
	return &ClientTransport{
		transport: NewTransport(reader, writer),
		requestID: 0,
	}
}

// Request sends a request and waits for response
func (c *ClientTransport) Request(ctx context.Context, method string, params interface{}, result interface{}) error {
	c.mu.Lock()
	c.requestID++
	id := c.requestID
	c.mu.Unlock()

	req := &Request{
		JSONRPC: "2.0",
		ID:      &RequestID{Number: id},
		Method:  method,
		Params:  params,
	}

	resp, err := c.transport.Call(ctx, req)
	if err != nil {
		return err
	}

	if result != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeParsing, "failed to unmarshal response").
				WithComponent("lsp.transport").
				WithOperation("request").
				WithDetails("method", method)
		}
	}

	return nil
}

// Notify sends a notification (no response expected)
func (c *ClientTransport) Notify(ctx context.Context, method string, params interface{}) error {
	req := &Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	return c.transport.Send(ctx, req)
}

// Listen starts listening for server messages
func (c *ClientTransport) Listen(ctx context.Context) error {
	return c.transport.Listen(ctx)
}
