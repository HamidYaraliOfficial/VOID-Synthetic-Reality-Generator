package export

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func sampleRecords() []Record {
	return []Record{
		{"id": "1", "name": "Alex"},
		{"id": "2", "name": "Sara"},
	}
}

func TestJSONExportRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := ToWriter(&buf, FormatJSON, "users", sampleRecords()); err != nil {
		t.Fatal(err)
	}
	var out []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 records, got %d", len(out))
	}
}

func TestCSVExportHasHeader(t *testing.T) {
	var buf bytes.Buffer
	if err := ToWriter(&buf, FormatCSV, "users", sampleRecords()); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 { // header + 2 rows
		t.Fatalf("expected 3 lines (header+2 rows), got %d: %v", len(lines), lines)
	}
}

func TestJSONLExportOneObjectPerLine(t *testing.T) {
	var buf bytes.Buffer
	if err := ToWriter(&buf, FormatJSONL, "users", sampleRecords()); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSONL lines, got %d", len(lines))
	}
}
