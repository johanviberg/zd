package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONFormatter_Format(t *testing.T) {
	f := &JSONFormatter{}
	var buf bytes.Buffer

	data := map[string]interface{}{
		"id":      1,
		"subject": "Test",
		"status":  "open",
	}

	err := f.Format(&buf, data)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "Test", result["subject"])
}

func TestJSONFormatter_FormatWithFields(t *testing.T) {
	f := &JSONFormatter{fields: []string{"id", "status"}}
	var buf bytes.Buffer

	data := map[string]interface{}{
		"id":      1,
		"subject": "Test",
		"status":  "open",
	}

	err := f.Format(&buf, data)
	require.NoError(t, err)

	var result map[string]interface{}
	json.Unmarshal(buf.Bytes(), &result)

	assert.NotContains(t, result, "subject", "expected 'subject' to be filtered out")
	assert.Equal(t, "open", result["status"])
}

func TestJSONFormatter_FormatList(t *testing.T) {
	f := &JSONFormatter{}
	var buf bytes.Buffer

	items := []interface{}{
		map[string]interface{}{"id": 1, "subject": "A"},
		map[string]interface{}{"id": 2, "subject": "B"},
	}

	err := f.FormatList(&buf, items, nil)
	require.NoError(t, err)

	var result []map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestNDJSONFormatter_FormatList(t *testing.T) {
	f := &NDJSONFormatter{}
	var buf bytes.Buffer

	items := []interface{}{
		map[string]interface{}{"id": 1},
		map[string]interface{}{"id": 2},
	}

	err := f.FormatList(&buf, items, nil)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)

	for _, line := range lines {
		var obj map[string]interface{}
		err := json.Unmarshal([]byte(line), &obj)
		assert.NoError(t, err, "invalid NDJSON line")
	}
}

func TestTextFormatter_FormatList(t *testing.T) {
	f := &TextFormatter{fields: []string{"id", "subject"}}
	var buf bytes.Buffer

	items := []interface{}{
		map[string]interface{}{"id": 1, "subject": "First", "extra": "hidden"},
		map[string]interface{}{"id": 2, "subject": "Second", "extra": "hidden"},
	}

	err := f.FormatList(&buf, items, nil)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "First")
	assert.NotContains(t, output, "hidden", "expected output to not contain 'hidden' (field projection)")
}

func TestTextFormatter_NoHeaders(t *testing.T) {
	f := &TextFormatter{noHeaders: true}
	var buf bytes.Buffer

	items := []interface{}{
		map[string]interface{}{"id": 1, "subject": "Test"},
	}

	err := f.FormatList(&buf, items, []string{"id", "subject"})
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// With noHeaders, should only have data lines (no header or separator)
	assert.Len(t, lines, 1, "expected 1 line (no headers), got %d: %q", len(lines), output)
}

func TestNewFormatter_Unknown(t *testing.T) {
	_, err := NewFormatter("xml", nil, false)
	assert.Error(t, err, "expected error for unknown format")
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
	require.True(t, ok, "expected map, got %T", projected)
	assert.NotContains(t, m, "subject", "expected 'subject' to be filtered out")
	assert.Equal(t, float64(1), m["id"], "expected id 1") // JSON round-trip converts to float64
}

func TestTextFormatter_HumanTimestamps(t *testing.T) {
	f := &TextFormatter{}
	var buf bytes.Buffer

	ts := time.Now().Add(-3 * time.Hour).UTC().Format(time.RFC3339)
	data := map[string]interface{}{
		"id":         1,
		"updated_at": ts,
	}

	err := f.Format(&buf, data)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "ago", "expected humanized time containing 'ago'")
}

func TestJSONFormatter_PreservesTimestamp(t *testing.T) {
	f := &JSONFormatter{}
	var buf bytes.Buffer

	ts := time.Now().Add(-3 * time.Hour).UTC().Format(time.RFC3339)
	data := map[string]interface{}{
		"id":         1,
		"updated_at": ts,
	}

	err := f.Format(&buf, data)
	require.NoError(t, err)

	var result map[string]interface{}
	json.Unmarshal(buf.Bytes(), &result)
	assert.Equal(t, ts, result["updated_at"])
}

func TestFieldProjection_NoFields(t *testing.T) {
	data := map[string]interface{}{"id": 1, "subject": "Test"}
	projected := projectFields(data, nil)

	// Should return original data unchanged
	m, ok := projected.(map[string]interface{})
	require.True(t, ok, "expected map, got %T", projected)
	assert.Equal(t, "Test", m["subject"], "expected all fields when no projection")
}
