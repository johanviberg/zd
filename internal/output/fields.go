package output

import (
	"encoding/json"
	"strings"
)

func projectFields(data interface{}, fields []string) interface{} {
	if len(fields) == 0 {
		return data
	}

	// Convert to map via JSON round-trip, preserving numeric types
	b, err := json.Marshal(data)
	if err != nil {
		return data
	}

	d := json.NewDecoder(strings.NewReader(string(b)))
	d.UseNumber()
	var m map[string]interface{}
	if err := d.Decode(&m); err != nil {
		return data
	}

	result := make(map[string]interface{})
	for _, f := range fields {
		if v, ok := m[f]; ok {
			result[f] = v
		}
	}
	return result
}
