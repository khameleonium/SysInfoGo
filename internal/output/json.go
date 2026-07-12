package output

import (
	"encoding/json"
)

type JSONFormatter struct {
	Indent bool
}

func NewJSONFormatter(indent bool) *JSONFormatter {
	return &JSONFormatter{Indent: indent}
}

func (f *JSONFormatter) Format(data *AggregatedData) string {
	var b []byte
	var err error
	if f.Indent {
		b, err = json.MarshalIndent(data, "", "  ")
	} else {
		b, err = json.Marshal(data)
	}
	if err != nil {
		return `{"error": "failed to marshal json"}`
	}
	return string(b) + "\n"
}
