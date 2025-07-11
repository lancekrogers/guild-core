package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// Formatter interface for custom log formatting
type Formatter interface {
	Format(record *slog.Record) ([]byte, error)
}

// PrettyHandlerOptions configures the pretty handler
type PrettyHandlerOptions struct {
	Level      slog.Level
	AddSource  bool
	UseColor   bool
	Multiline  bool
	TimeFormat string
}

// PrettyHandler provides human-readable colored output
type PrettyHandler struct {
	opts   PrettyHandlerOptions
	writer io.Writer
	mu     sync.Mutex
	attrs  []slog.Attr
	groups []string
}

// NewPrettyHandler creates a new pretty handler
func NewPrettyHandler(w io.Writer, opts *PrettyHandlerOptions) *PrettyHandler {
	if opts == nil {
		opts = &PrettyHandlerOptions{
			Level:      slog.LevelInfo,
			UseColor:   true,
			TimeFormat: "15:04:05.000",
		}
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = "15:04:05.000"
	}
	return &PrettyHandler{
		opts:   *opts,
		writer: w,
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *PrettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level
}

// Handle formats and writes the log record
func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var b strings.Builder

	// Time
	if h.opts.UseColor {
		b.WriteString("\x1b[90m") // Gray
	}
	b.WriteString(r.Time.Format(h.opts.TimeFormat))
	if h.opts.UseColor {
		b.WriteString("\x1b[0m")
	}
	b.WriteString(" ")

	// Level with color
	levelStr := strings.ToUpper(r.Level.String())
	if h.opts.UseColor {
		switch r.Level {
		case slog.LevelDebug:
			b.WriteString("\x1b[36m") // Cyan
		case slog.LevelInfo:
			b.WriteString("\x1b[32m") // Green
		case slog.LevelWarn:
			b.WriteString("\x1b[33m") // Yellow
		case slog.LevelError:
			b.WriteString("\x1b[31m") // Red
		}
	}
	b.WriteString(fmt.Sprintf("%-5s", levelStr))
	if h.opts.UseColor {
		b.WriteString("\x1b[0m")
	}
	b.WriteString(" ")

	// Source
	if h.opts.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if h.opts.UseColor {
			b.WriteString("\x1b[90m") // Gray
		}
		// Shorten the file path
		file := f.File
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			if idx2 := strings.LastIndex(file[:idx], "/"); idx2 >= 0 {
				file = file[idx2+1:]
			}
		}
		b.WriteString(fmt.Sprintf("%s:%d", file, f.Line))
		if h.opts.UseColor {
			b.WriteString("\x1b[0m")
		}
		b.WriteString(" ")
	}

	// Message
	if h.opts.UseColor {
		b.WriteString("\x1b[97m") // Bright white
	}
	b.WriteString(r.Message)
	if h.opts.UseColor {
		b.WriteString("\x1b[0m")
	}

	// Attributes
	if r.NumAttrs() > 0 || len(h.attrs) > 0 {
		if h.opts.Multiline {
			b.WriteString("\n")
			h.formatAttrsMultiline(&b, h.attrs)
			r.Attrs(func(a slog.Attr) bool {
				h.formatAttrMultiline(&b, a, 1)
				return true
			})
		} else {
			b.WriteString(" ")
			h.formatAttrsInline(&b, h.attrs)
			r.Attrs(func(a slog.Attr) bool {
				b.WriteString(" ")
				h.formatAttrInline(&b, a)
				return true
			})
		}
	}

	b.WriteString("\n")

	_, err := h.writer.Write([]byte(b.String()))
	return err
}

// WithAttrs returns a new handler with additional attributes
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	return &PrettyHandler{
		opts:   h.opts,
		writer: h.writer,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

// WithGroup returns a new handler with the given group name
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	return &PrettyHandler{
		opts:   h.opts,
		writer: h.writer,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

// formatAttrInline formats an attribute for inline display
func (h *PrettyHandler) formatAttrInline(b *strings.Builder, a slog.Attr) {
	if h.opts.UseColor {
		b.WriteString("\x1b[94m") // Blue
	}
	b.WriteString(a.Key)
	if h.opts.UseColor {
		b.WriteString("\x1b[0m")
	}
	b.WriteString("=")
	h.formatValue(b, a.Value)
}

// formatAttrMultiline formats an attribute for multiline display
func (h *PrettyHandler) formatAttrMultiline(b *strings.Builder, a slog.Attr, indent int) {
	prefix := strings.Repeat("  ", indent)
	b.WriteString(prefix)
	if h.opts.UseColor {
		b.WriteString("\x1b[94m") // Blue
	}
	b.WriteString(a.Key)
	if h.opts.UseColor {
		b.WriteString("\x1b[0m")
	}
	b.WriteString(": ")

	switch a.Value.Kind() {
	case slog.KindGroup:
		b.WriteString("\n")
		for _, ga := range a.Value.Group() {
			h.formatAttrMultiline(b, ga, indent+1)
		}
	default:
		h.formatValue(b, a.Value)
		b.WriteString("\n")
	}
}

// formatAttrsInline formats multiple attributes inline
func (h *PrettyHandler) formatAttrsInline(b *strings.Builder, attrs []slog.Attr) {
	for i, a := range attrs {
		if i > 0 {
			b.WriteString(" ")
		}
		h.formatAttrInline(b, a)
	}
}

// formatAttrsMultiline formats multiple attributes multiline
func (h *PrettyHandler) formatAttrsMultiline(b *strings.Builder, attrs []slog.Attr) {
	for _, a := range attrs {
		h.formatAttrMultiline(b, a, 1)
	}
}

// formatValue formats a slog value
func (h *PrettyHandler) formatValue(b *strings.Builder, v slog.Value) {
	switch v.Kind() {
	case slog.KindString:
		if h.opts.UseColor {
			b.WriteString("\x1b[93m") // Yellow
		}
		fmt.Fprintf(b, "%q", v.String())
		if h.opts.UseColor {
			b.WriteString("\x1b[0m")
		}
	case slog.KindInt64:
		if h.opts.UseColor {
			b.WriteString("\x1b[96m") // Cyan
		}
		b.WriteString(fmt.Sprintf("%d", v.Int64()))
		if h.opts.UseColor {
			b.WriteString("\x1b[0m")
		}
	case slog.KindFloat64:
		if h.opts.UseColor {
			b.WriteString("\x1b[96m") // Cyan
		}
		b.WriteString(fmt.Sprintf("%g", v.Float64()))
		if h.opts.UseColor {
			b.WriteString("\x1b[0m")
		}
	case slog.KindBool:
		if h.opts.UseColor {
			b.WriteString("\x1b[95m") // Magenta
		}
		b.WriteString(fmt.Sprintf("%t", v.Bool()))
		if h.opts.UseColor {
			b.WriteString("\x1b[0m")
		}
	case slog.KindDuration:
		if h.opts.UseColor {
			b.WriteString("\x1b[96m") // Cyan
		}
		b.WriteString(v.Duration().String())
		if h.opts.UseColor {
			b.WriteString("\x1b[0m")
		}
	case slog.KindTime:
		if h.opts.UseColor {
			b.WriteString("\x1b[96m") // Cyan
		}
		b.WriteString(v.Time().Format(time.RFC3339))
		if h.opts.UseColor {
			b.WriteString("\x1b[0m")
		}
	case slog.KindAny:
		if err, ok := v.Any().(error); ok {
			if h.opts.UseColor {
				b.WriteString("\x1b[91m") // Bright red
			}
			// Handle gerror specially
			if gErr, ok := err.(*gerror.GuildError); ok {
				b.WriteString(gErr.Error())
				details := gErr.Details
				if len(details) > 0 {
					b.WriteString(" {")
					i := 0
					for k, v := range details {
						if i > 0 {
							b.WriteString(", ")
						}
						fmt.Fprintf(b, "%s: %v", k, v)
						i++
					}
					b.WriteString("}")
				}
			} else {
				b.WriteString(err.Error())
			}
			if h.opts.UseColor {
				b.WriteString("\x1b[0m")
			}
		} else {
			fmt.Fprintf(b, "%v", v.Any())
		}
	default:
		fmt.Fprintf(b, "%v", v.Any())
	}
}

// TextFormatter provides plain text output with configurable delimiter
type TextFormatter struct {
	Delimiter  string
	QuoteEmpty bool
	TimeFormat string
}

// NewTextFormatter creates a new text formatter
func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		Delimiter:  " | ",
		QuoteEmpty: true,
		TimeFormat: time.RFC3339,
	}
}

// Format formats a log record as delimited text
func (f *TextFormatter) Format(r *slog.Record) ([]byte, error) {
	var b strings.Builder

	// Time
	b.WriteString(r.Time.Format(f.TimeFormat))
	b.WriteString(f.Delimiter)

	// Level
	b.WriteString(r.Level.String())
	b.WriteString(f.Delimiter)

	// Message
	b.WriteString(r.Message)

	// Attributes
	r.Attrs(func(a slog.Attr) bool {
		b.WriteString(f.Delimiter)
		b.WriteString(a.Key)
		b.WriteString("=")
		f.formatValue(&b, a.Value)
		return true
	})

	b.WriteString("\n")
	return []byte(b.String()), nil
}

// formatValue formats a value for text output
func (f *TextFormatter) formatValue(b *strings.Builder, v slog.Value) {
	switch v.Kind() {
	case slog.KindString:
		s := v.String()
		if s == "" && f.QuoteEmpty {
			b.WriteString(`""`)
		} else {
			b.WriteString(s)
		}
	default:
		fmt.Fprintf(b, "%v", v.Any())
	}
}
