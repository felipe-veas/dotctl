package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrinterSuccess(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{w: &buf, jsonOut: false}

	p.Success("done %s", "ok")
	got := buf.String()
	if !strings.Contains(got, "✓") || !strings.Contains(got, "done ok") {
		t.Errorf("Success output = %q, want checkmark + message", got)
	}
}

func TestPrinterSuppressedInJSON(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{w: &buf, jsonOut: true}

	p.Success("should not appear")
	p.Error("should not appear")
	p.Warn("should not appear")
	p.Info("should not appear")
	p.Header("should not appear")
	p.Field("key", "value")

	if buf.Len() != 0 {
		t.Errorf("expected no output in JSON mode, got: %q", buf.String())
	}
}

func TestPrinterJSON(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{w: &buf, jsonOut: true}

	data := map[string]string{"profile": "test"}
	if err := p.JSON(data); err != nil {
		t.Fatalf("JSON: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result["profile"] != "test" {
		t.Errorf("profile = %q, want %q", result["profile"], "test")
	}
}

func TestPrinterField(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{w: &buf, jsonOut: false}

	p.Field("Profile", "macstudio")
	got := buf.String()
	if !strings.Contains(got, "Profile:") || !strings.Contains(got, "macstudio") {
		t.Errorf("Field output = %q, want key-value pair", got)
	}
}

func TestStatusLine(t *testing.T) {
	tests := []struct {
		total, ok, broken, drift int
		want                     string
	}{
		{12, 12, 0, 0, "12 total: 12 ok"},
		{12, 10, 1, 1, "12 total: 10 ok, 1 broken, 1 drift"},
		{5, 3, 2, 0, "5 total: 3 ok, 2 broken"},
		{5, 4, 0, 1, "5 total: 4 ok, 1 drift"},
	}

	for _, tt := range tests {
		got := StatusLine(tt.total, tt.ok, tt.broken, tt.drift)
		if got != tt.want {
			t.Errorf("StatusLine(%d,%d,%d,%d) = %q, want %q",
				tt.total, tt.ok, tt.broken, tt.drift, got, tt.want)
		}
	}
}

func TestIsJSON(t *testing.T) {
	p := New(true)
	if !p.IsJSON() {
		t.Error("IsJSON should return true")
	}
	p = New(false)
	if p.IsJSON() {
		t.Error("IsJSON should return false")
	}
}
