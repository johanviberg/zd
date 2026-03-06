package output

import (
	"encoding/json"
)

func projectFields(data interface{}, fields []string) interface{} {
	if len(fields) == 0 {
		return data
	}

	// Convert to map via JSON round-trip
	b, err := json.Marshal(data)
	if err != nil {
		return data
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
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
