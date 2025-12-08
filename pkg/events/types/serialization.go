// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package types

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/guild-framework/guild-core/pkg/events"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// SerializationFormat defines the format for event serialization
type SerializationFormat string

const (
	// FormatJSON serializes events as JSON
	FormatJSON SerializationFormat = "json"

	// FormatBinary serializes events as binary using gob
	FormatBinary SerializationFormat = "binary"

	// FormatCompressed serializes events as compressed binary
	FormatCompressed SerializationFormat = "compressed"
)

// Serializer handles event serialization and deserialization
type Serializer struct {
	format     SerializationFormat
	registry   *EventRegistry
	compressor Compressor
}

// Compressor defines the interface for compression
type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
}

// GzipCompressor implements gzip compression
type GzipCompressor struct {
	level int
}

// NewSerializer creates a new event serializer
func NewSerializer(format SerializationFormat, registry *EventRegistry) *Serializer {
	s := &Serializer{
		format:   format,
		registry: registry,
	}

	if format == FormatCompressed {
		s.compressor = &GzipCompressor{level: gzip.BestSpeed}
	}

	return s
}

// Serialize serializes an event to bytes
func (s *Serializer) Serialize(ctx context.Context, event events.CoreEvent) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// Create serializable event
	se := &SerializableEvent{
		ID:        event.GetID(),
		Type:      event.GetType(),
		Source:    event.GetSource(),
		Timestamp: event.GetTimestamp(),
		Data:      event.GetData(),
		Metadata:  event.GetMetadata(),
	}

	// Extract correlation and parent IDs from metadata if available
	if metadata := event.GetMetadata(); metadata != nil {
		if corrID, ok := metadata["correlation_id"].(string); ok {
			se.CorrelationID = corrID
		}
		if parentID, ok := metadata["parent_id"].(string); ok {
			se.ParentID = parentID
		}
	}

	var data []byte
	var err error

	switch s.format {
	case FormatJSON:
		data, err = json.Marshal(se)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal event to JSON")
		}

	case FormatBinary, FormatCompressed:
		var buf bytes.Buffer
		encoder := gob.NewEncoder(&buf)
		if err := encoder.Encode(se); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to encode event to binary")
		}
		data = buf.Bytes()

		// Compress if needed
		if s.format == FormatCompressed && s.compressor != nil {
			data, err = s.compressor.Compress(data)
			if err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to compress event")
			}
		}

	default:
		return nil, gerror.New(gerror.ErrCodeValidation, "unsupported serialization format", nil).
			WithDetails("format", string(s.format))
	}

	return data, nil
}

// Deserialize deserializes an event from bytes
func (s *Serializer) Deserialize(ctx context.Context, data []byte) (events.CoreEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	var se SerializableEvent

	switch s.format {
	case FormatJSON:
		if err := json.Unmarshal(data, &se); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal JSON event")
		}

	case FormatBinary, FormatCompressed:
		// Decompress if needed
		if s.format == FormatCompressed && s.compressor != nil {
			var err error
			data, err = s.compressor.Decompress(data)
			if err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to decompress event")
			}
		}

		decoder := gob.NewDecoder(bytes.NewReader(data))
		if err := decoder.Decode(&se); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to decode binary event")
		}

	default:
		return nil, gerror.New(gerror.ErrCodeValidation, "unsupported serialization format", nil).
			WithDetails("format", string(s.format))
	}

	// Create event from serializable data
	event := events.NewBaseEvent(se.ID, se.Type, se.Source, se.Data)

	// Add metadata
	if se.Metadata != nil {
		for k, v := range se.Metadata {
			event.WithMetadata(k, v)
		}
	}

	// Set correlation and parent IDs in metadata if available
	if se.CorrelationID != "" {
		event.WithMetadata("correlation_id", se.CorrelationID)
	}
	if se.ParentID != "" {
		event.WithMetadata("parent_id", se.ParentID)
	}

	// Validate against schema if registry is available
	if s.registry != nil {
		if err := s.registry.ValidateEvent(ctx, event); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "deserialized event failed validation")
		}
	}

	return event, nil
}

// SerializeBatch serializes multiple events efficiently
func (s *Serializer) SerializeBatch(ctx context.Context, events []events.CoreEvent) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	batch := &EventBatch{
		Format:    string(s.format),
		Count:     len(events),
		Timestamp: time.Now(),
		Events:    make([]SerializableEvent, len(events)),
	}

	for i, event := range events {
		se := SerializableEvent{
			ID:        event.GetID(),
			Type:      event.GetType(),
			Source:    event.GetSource(),
			Timestamp: event.GetTimestamp(),
			Data:      event.GetData(),
			Metadata:  event.GetMetadata(),
		}

		// Extract correlation and parent IDs from metadata if available
		if metadata := event.GetMetadata(); metadata != nil {
			if corrID, ok := metadata["correlation_id"].(string); ok {
				se.CorrelationID = corrID
			}
			if parentID, ok := metadata["parent_id"].(string); ok {
				se.ParentID = parentID
			}
		}

		batch.Events[i] = se
	}

	var data []byte
	var err error

	switch s.format {
	case FormatJSON:
		data, err = json.Marshal(batch)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal batch to JSON")
		}

	case FormatBinary, FormatCompressed:
		var buf bytes.Buffer
		encoder := gob.NewEncoder(&buf)
		if err := encoder.Encode(batch); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to encode batch to binary")
		}
		data = buf.Bytes()

		if s.format == FormatCompressed && s.compressor != nil {
			data, err = s.compressor.Compress(data)
			if err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to compress batch")
			}
		}

	default:
		return nil, gerror.New(gerror.ErrCodeValidation, "unsupported serialization format", nil).
			WithDetails("format", string(s.format))
	}

	return data, nil
}

// DeserializeBatch deserializes multiple events
func (s *Serializer) DeserializeBatch(ctx context.Context, data []byte) ([]events.CoreEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	var batch EventBatch

	switch s.format {
	case FormatJSON:
		if err := json.Unmarshal(data, &batch); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal JSON batch")
		}

	case FormatBinary, FormatCompressed:
		if s.format == FormatCompressed && s.compressor != nil {
			var err error
			data, err = s.compressor.Decompress(data)
			if err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to decompress batch")
			}
		}

		decoder := gob.NewDecoder(bytes.NewReader(data))
		if err := decoder.Decode(&batch); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to decode binary batch")
		}

	default:
		return nil, gerror.New(gerror.ErrCodeValidation, "unsupported serialization format", nil).
			WithDetails("format", string(s.format))
	}

	// Convert to events
	result := make([]events.CoreEvent, len(batch.Events))
	for i, se := range batch.Events {
		event := events.NewBaseEvent(se.ID, se.Type, se.Source, se.Data)

		if se.Metadata != nil {
			for k, v := range se.Metadata {
				event.WithMetadata(k, v)
			}
		}

		if se.CorrelationID != "" {
			event.WithMetadata("correlation_id", se.CorrelationID)
		}
		if se.ParentID != "" {
			event.WithMetadata("parent_id", se.ParentID)
		}

		result[i] = event
	}

	return result, nil
}

// SerializableEvent is a serializable representation of an event
type SerializableEvent struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Source        string                 `json:"source"`
	Timestamp     time.Time              `json:"timestamp"`
	Data          map[string]interface{} `json:"data"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	ParentID      string                 `json:"parent_id,omitempty"`
}

// EventBatch represents a batch of events
type EventBatch struct {
	Format    string              `json:"format"`
	Count     int                 `json:"count"`
	Timestamp time.Time           `json:"timestamp"`
	Events    []SerializableEvent `json:"events"`
}

// Compress compresses data using gzip
func (c *GzipCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, c.level)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to write compressed data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// Decompress decompresses gzip data
func (c *GzipCompressor) Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}

	return decompressed, nil
}

// EventStream provides streaming serialization/deserialization
type EventStream struct {
	serializer *Serializer
	writer     io.Writer
	reader     io.Reader
	encoder    *json.Encoder
	decoder    *json.Decoder
}

// NewEventStream creates a new event stream
func NewEventStream(serializer *Serializer, rw io.ReadWriter) *EventStream {
	return &EventStream{
		serializer: serializer,
		writer:     rw,
		reader:     rw,
		encoder:    json.NewEncoder(rw),
		decoder:    json.NewDecoder(rw),
	}
}

// WriteEvent writes a single event to the stream
func (es *EventStream) WriteEvent(ctx context.Context, event events.CoreEvent) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	data, err := es.serializer.Serialize(ctx, event)
	if err != nil {
		return err
	}

	// Write length prefix
	length := uint32(len(data))
	if err := es.writeUint32(length); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write event length")
	}

	// Write data
	if _, err := es.writer.Write(data); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write event data")
	}

	return nil
}

// ReadEvent reads a single event from the stream
func (es *EventStream) ReadEvent(ctx context.Context) (events.CoreEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// Read length prefix
	length, err := es.readUint32()
	if err != nil {
		if err == io.EOF {
			return nil, err
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to read event length")
	}

	// Read data
	data := make([]byte, length)
	if _, err := io.ReadFull(es.reader, data); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to read event data")
	}

	return es.serializer.Deserialize(ctx, data)
}

// writeUint32 writes a uint32 in big-endian format
func (es *EventStream) writeUint32(n uint32) error {
	buf := []byte{
		byte(n >> 24),
		byte(n >> 16),
		byte(n >> 8),
		byte(n),
	}
	_, err := es.writer.Write(buf)
	return err
}

// readUint32 reads a uint32 in big-endian format
func (es *EventStream) readUint32() (uint32, error) {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(es.reader, buf); err != nil {
		return 0, err
	}
	return uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3]), nil
}
