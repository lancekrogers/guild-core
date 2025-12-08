package writers

import (
	"io"
	"sync"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// MultiWriter writes to multiple destinations
type MultiWriter struct {
	writers []io.Writer
	mu      sync.RWMutex
}

// NewMultiWriter creates a writer that duplicates writes to all provided writers
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write implements io.Writer
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	mw.mu.RLock()
	defer mw.mu.RUnlock()

	var errs []error
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			errs = append(errs, err)
		}
		if n != len(p) {
			errs = append(errs, io.ErrShortWrite)
		}
	}

	if len(errs) > 0 {
		return n, gerror.New(gerror.ErrCodeInternal, "multi-writer errors", nil).
			WithDetails("errors", errs)
	}

	return len(p), nil
}

// AddWriter adds a new writer to the multi-writer
func (mw *MultiWriter) AddWriter(w io.Writer) {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	mw.writers = append(mw.writers, w)
}

// RemoveWriter removes a writer from the multi-writer
func (mw *MultiWriter) RemoveWriter(w io.Writer) {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	filtered := make([]io.Writer, 0, len(mw.writers))
	for _, writer := range mw.writers {
		if writer != w {
			filtered = append(filtered, writer)
		}
	}
	mw.writers = filtered
}

// Close closes all writers that implement io.Closer
func (mw *MultiWriter) Close() error {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	var errs []error
	for _, w := range mw.writers {
		if closer, ok := w.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return gerror.New(gerror.ErrCodeInternal, "failed to close writers", nil).
			WithDetails("errors", errs)
	}

	return nil
}
