package output

import (
	"encoding/json"
	"testing"
)

func TestJSONFormatter(t *testing.T) {
	f := NewJSONFormatter(false)

	data := &AggregatedData{
		Timestamp: "2026-01-01T00:00:00Z",
		Hostname:  "test-pc",
		OS:        "windows amd64",
		IsAdmin:   false,
		Sections: map[string]any{
			"cpu": map[string]any{
				"model": "Test CPU",
				"cores": 8,
			},
		},
		Warnings: []Warning{{
			Section: "storage",
			Message: "smartctl not found",
			OSHint:  "install smartmontools",
		}},
	}

	result := f.Format(data)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if parsed["hostname"] != "test-pc" {
		t.Errorf("expected hostname 'test-pc', got %v", parsed["hostname"])
	}

	sections := parsed["sections"].(map[string]any)
	cpu := sections["cpu"].(map[string]any)
	if cpu["model"] != "Test CPU" {
		t.Errorf("expected cpu model 'Test CPU', got %v", cpu["model"])
	}

	warnings := parsed["warnings"].([]any)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestWarning(t *testing.T) {
	w := Warning{
		Section: "cpu",
		Message: "test warning",
		OSHint:  "run as admin",
	}

	if w.Section != "cpu" {
		t.Errorf("expected section 'cpu', got %s", w.Section)
	}
	if w.Message != "test warning" {
		t.Errorf("expected message 'test warning', got %s", w.Message)
	}
}

func TestFormatMB(t *testing.T) {
	tests := []struct {
		mb       float64
		unit     string
		expected string
	}{
		{500, "mb", "500 MB"},
		{1024, "mb", "1024 MB"},
		{2048, "gb", "2.0 GB"},
		{2560, "gb", "2.5 GB"},
		{500, "auto", "500 MB"},
		{1536, "auto", "1.5 GB"},
	}

	for _, tt := range tests {
		result := FormatMB(tt.mb, tt.unit)
		if result != tt.expected {
			t.Errorf("FormatMB(%f, %q) = %q; want %q", tt.mb, tt.unit, result, tt.expected)
		}
	}
}
