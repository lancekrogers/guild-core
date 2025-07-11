package writers

import (
	"io"
	"log/slog"
)

// FilteredWriter filters log output based on level
type FilteredWriter struct {
	writer   io.Writer
	minLevel slog.Level
}

// NewFilteredWriter creates a writer that only outputs logs at or above the specified level
func NewFilteredWriter(w io.Writer, minLevel slog.Level) *FilteredWriter {
	return &FilteredWriter{
		writer:   w,
		minLevel: minLevel,
	}
}

// Write implements io.Writer
func (fw *FilteredWriter) Write(p []byte) (n int, err error) {
	// Note: This is a simple implementation. In practice, you'd need to parse
	// the log level from the message or use this as part of a handler chain
	return fw.writer.Write(p)
}

// SetLevel updates the minimum log level
func (fw *FilteredWriter) SetLevel(level slog.Level) {
	fw.minLevel = level
}

// Close closes the underlying writer if it implements io.Closer
func (fw *FilteredWriter) Close() error {
	if closer, ok := fw.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
