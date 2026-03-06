package output

import (
	"encoding/json"
	"fmt"
	"io"
)

type Formatter interface {
	Format(w io.Writer, data interface{}) error
	FormatList(w io.Writer, items []interface{}, headers []string) error
}

func NewFormatter(format string, fields []string, noHeaders bool) (Formatter, error) {
	switch format {
	case "json":
		return &JSONFormatter{fields: fields}, nil
	case "ndjson":
		return &NDJSONFormatter{fields: fields}, nil
	case "text", "":
		return &TextFormatter{fields: fields, noHeaders: noHeaders}, nil
	default:
		return nil, fmt.Errorf("unknown output format: %s", format)
	}
}

type JSONFormatter struct {
	fields []string
}

func (f *JSONFormatter) Format(w io.Writer, data interface{}) error {
	projected := projectFields(data, f.fields)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(projected)
}

func (f *JSONFormatter) FormatList(w io.Writer, items []interface{}, _ []string) error {
	projected := make([]interface{}, len(items))
	for i, item := range items {
		projected[i] = projectFields(item, f.fields)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(projected)
}

type NDJSONFormatter struct {
	fields []string
}

func (f *NDJSONFormatter) Format(w io.Writer, data interface{}) error {
	projected := projectFields(data, f.fields)
	b, err := json.Marshal(projected)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func (f *NDJSONFormatter) FormatList(w io.Writer, items []interface{}, _ []string) error {
	for _, item := range items {
		if err := f.Format(w, item); err != nil {
			return err
		}
	}
	return nil
}

