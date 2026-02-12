package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Printer handles formatted output to the user.
type Printer struct {
	w       io.Writer
	jsonOut bool
}

func (p *Printer) writef(format string, args ...any) {
	_, _ = fmt.Fprintf(p.w, format, args...)
}

// New creates a Printer. If jsonMode is true, Print calls are suppressed
// and only JSON output is emitted via JSON().
func New(jsonMode bool) *Printer {
	return &Printer{
		w:       os.Stdout,
		jsonOut: jsonMode,
	}
}

// IsJSON returns whether the printer is in JSON mode.
func (p *Printer) IsJSON() bool {
	return p.jsonOut
}

// Success prints a green checkmark line (suppressed in JSON mode).
func (p *Printer) Success(format string, args ...any) {
	if p.jsonOut {
		return
	}
	p.writef("✓ %s\n", fmt.Sprintf(format, args...))
}

// Error prints a red cross line (suppressed in JSON mode).
func (p *Printer) Error(format string, args ...any) {
	if p.jsonOut {
		return
	}
	p.writef("✗ %s\n", fmt.Sprintf(format, args...))
}

// Warn prints a warning line (suppressed in JSON mode).
func (p *Printer) Warn(format string, args ...any) {
	if p.jsonOut {
		return
	}
	p.writef("! %s\n", fmt.Sprintf(format, args...))
}

// Info prints an info line (suppressed in JSON mode).
func (p *Printer) Info(format string, args ...any) {
	if p.jsonOut {
		return
	}
	p.writef("%s\n", fmt.Sprintf(format, args...))
}

// Header prints a section header (suppressed in JSON mode).
func (p *Printer) Header(title string) {
	if p.jsonOut {
		return
	}
	p.writef("\n%s\n", title)
}

// Field prints a key-value pair (suppressed in JSON mode).
func (p *Printer) Field(key, value string) {
	if p.jsonOut {
		return
	}
	p.writef("%-12s %s\n", key+":", value)
}

// JSON outputs v as indented JSON to stdout. Returns error if marshaling fails.
func (p *Printer) JSON(v any) error {
	enc := json.NewEncoder(p.w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// StatusLine builds a one-line status string (e.g. "12 total: 10 ok, 1 broken, 1 drift").
func StatusLine(total, ok, broken, drift int) string {
	parts := []string{fmt.Sprintf("%d ok", ok)}
	if broken > 0 {
		parts = append(parts, fmt.Sprintf("%d broken", broken))
	}
	if drift > 0 {
		parts = append(parts, fmt.Sprintf("%d drift", drift))
	}
	return fmt.Sprintf("%d total: %s", total, strings.Join(parts, ", "))
}
