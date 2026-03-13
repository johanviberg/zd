package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dustin/go-humanize"
)

type TextFormatter struct {
	fields    []string
	noHeaders bool
}

func (f *TextFormatter) Format(w io.Writer, data interface{}) error {
	// Single item: print key-value pairs
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	var keys []string
	if len(f.fields) > 0 {
		for _, field := range f.fields {
			if _, ok := m[field]; ok {
				keys = append(keys, field)
			}
		}
	} else {
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	for _, k := range keys {
		fmt.Fprintf(tw, "%s:\t%v\n", k, formatValue(m[k]))
	}
	return tw.Flush()
}

func (f *TextFormatter) FormatList(w io.Writer, items []interface{}, headers []string) error {
	if len(items) == 0 {
		return nil
	}

	if len(f.fields) > 0 {
		headers = f.fields
	}

	if len(headers) == 0 {
		// Derive headers from first item
		b, err := json.Marshal(items[0])
		if err != nil {
			return fmt.Errorf("formatting: %w", err)
		}
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			return fmt.Errorf("formatting: %w", err)
		}
		for k := range m {
			headers = append(headers, k)
		}
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	if !f.noHeaders {
		fmt.Fprintln(tw, strings.Join(headers, "\t"))
		dashes := make([]string, len(headers))
		for i, h := range headers {
			dashes[i] = strings.Repeat("-", len(h))
		}
		fmt.Fprintln(tw, strings.Join(dashes, "\t"))
	}

	for _, item := range items {
		b, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("formatting: %w", err)
		}
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			return fmt.Errorf("formatting: %w", err)
		}

		vals := make([]string, len(headers))
		for i, h := range headers {
			vals[i] = fmt.Sprintf("%v", formatValue(m[h]))
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}

	return tw.Flush()
}

func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return humanize.Time(t)
		}
		return val
	case []interface{}:
		strs := make([]string, len(val))
		for i, s := range val {
			strs[i] = fmt.Sprintf("%v", s)
		}
		return strings.Join(strs, ",")
	default:
		return fmt.Sprintf("%v", v)
	}
}
