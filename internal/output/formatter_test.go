package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestJSONFormatter_Format(t *testing.T) {
	f := &JSONFormatter{}
	var buf bytes.Buffer

	data := map[string]interface{}{
		"id":      1,
		"subject": "Test",
		"status":  "open",
	}

	if err := f.Format(&buf, data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["subject"] != "Test" {
		t.Errorf("expected subject 'Test', got %v", result["subject"])
	}
}

func TestJSONFormatter_FormatWithFields(t *testing.T) {
	f := &JSONFormatter{fields: []string{"id", "status"}}
	var buf bytes.Buffer

	data := map[string]interface{}{
		"id":      1,
		"subject": "Test",
		"status":  "open",
	}

	if err := f.Format(&buf, data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(buf.Bytes(), &result)

	if _, ok := result["subject"]; ok {
		t.Error("expected 'subject' to be filtered out")
	}
	if result["status"] != "open" {
		t.Errorf("expected status 'open', got %v", result["status"])
	}
}

func TestJSONFormatter_FormatList(t *testing.T) {
	f := &JSONFormatter{}
	var buf bytes.Buffer

	items := []interface{}{
		map[string]interface{}{"id": 1, "subject": "A"},
		map[string]interface{}{"id": 2, "subject": "B"},
	}

	if err := f.FormatList(&buf, items, nil); err != nil {
		t.Fatalf("FormatList: %v", err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

func TestNDJSONFormatter_FormatList(t *testing.T) {
	f := &NDJSONFormatter{}
	var buf bytes.Buffer

	items := []interface{}{
		map[string]interface{}{"id": 1},
		map[string]interface{}{"id": 2},
	}

	if err := f.FormatList(&buf, items, nil); err != nil {
		t.Fatalf("FormatList: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}

	for _, line := range lines {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("invalid NDJSON line: %v", err)
		}
	}
}

func TestTextFormatter_FormatList(t *testing.T) {
	f := &TextFormatter{fields: []string{"id", "subject"}}
	var buf bytes.Buffer

	items := []interface{}{
		map[string]interface{}{"id": 1, "subject": "First", "extra": "hidden"},
		map[string]interface{}{"id": 2, "subject": "Second", "extra": "hidden"},
	}

	if err := f.FormatList(&buf, items, nil); err != nil {
		t.Fatalf("FormatList: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "First") {
		t.Error("expected output to contain 'First'")
	}
	if strings.Contains(output, "hidden") {
		t.Error("expected output to not contain 'hidden' (field projection)")
	}
}

func TestTextFormatter_NoHeaders(t *testing.T) {
	f := &TextFormatter{noHeaders: true}
	var buf bytes.Buffer

	items := []interface{}{
		map[string]interface{}{"id": 1, "subject": "Test"},
	}

	if err := f.FormatList(&buf, items, []string{"id", "subject"}); err != nil {
		t.Fatalf("FormatList: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// With noHeaders, should only have data lines (no header or separator)
	if len(lines) != 1 {
		t.Errorf("expected 1 line (no headers), got %d: %q", len(lines), output)
	}
}

func TestNewFormatter_Unknown(t *testing.T) {
	_, err := NewFormatter("xml", nil, false)
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestFieldProjection(t *testing.T) {
	data := map[string]interface{}{
		"id":      1,
		"subject": "Test",
		"status":  "open",
		"tags":    []string{"a", "b"},
	}

	projected := projectFields(data, []string{"id", "tags"})

	m, ok := projected.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", projected)
	}
	if _, ok := m["subject"]; ok {
		t.Error("expected 'subject' to be filtered out")
	}
	if m["id"] != float64(1) { // JSON round-trip converts to float64
		t.Errorf("expected id 1, got %v", m["id"])
	}
}

func TestTextFormatter_HumanTimestamps(t *testing.T) {
	f := &TextFormatter{}
	var buf bytes.Buffer

	ts := time.Now().Add(-3 * time.Hour).UTC().Format(time.RFC3339)
	data := map[string]interface{}{
		"id":         1,
		"updated_at": ts,
	}

	if err := f.Format(&buf, data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "ago") {
		t.Errorf("expected humanized time containing 'ago', got: %s", output)
	}
}

func TestJSONFormatter_PreservesTimestamp(t *testing.T) {
	f := &JSONFormatter{}
	var buf bytes.Buffer

	ts := time.Now().Add(-3 * time.Hour).UTC().Format(time.RFC3339)
	data := map[string]interface{}{
		"id":         1,
		"updated_at": ts,
	}

	if err := f.Format(&buf, data); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(buf.Bytes(), &result)
	if result["updated_at"] != ts {
		t.Errorf("expected timestamp %q preserved, got %v", ts, result["updated_at"])
	}
}

func TestFieldProjection_NoFields(t *testing.T) {
	data := map[string]interface{}{"id": 1, "subject": "Test"}
	projected := projectFields(data, nil)

	// Should return original data unchanged
	m, ok := projected.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", projected)
	}
	if m["subject"] != "Test" {
		t.Error("expected all fields when no projection")
	}
}
