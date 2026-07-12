package watch

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"time"
)

type CSVLogger struct {
	file   *os.File
	writer *csv.Writer
	header []string
	sep    rune
}

func NewLogger(path string, appendMode bool) (*CSVLogger, error) {
	flag := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	f, err := os.OpenFile(path, flag, 0644)
	if err != nil {
		return nil, err
	}

	w := csv.NewWriter(f)
	w.Comma = ';'
	w.UseCRLF = true

	return &CSVLogger{file: f, writer: w, sep: ';'}, nil
}

func (l *CSVLogger) WriteRow(ts time.Time, data map[string]any) error {
	flat := make(map[string]string)
	for section, val := range data {
		flatten(section, val, flat)
	}

	row := make(map[string]string)
	row["timestamp"] = ts.UTC().Format(time.RFC3339)
	for k, v := range flat {
		row[k] = v
	}

	if l.header == nil {
		l.header = sortedKeys(row)
		if err := l.writer.Write(l.header); err != nil {
			return err
		}
	}

	values := make([]string, len(l.header))
	for i, k := range l.header {
		values[i] = row[k]
	}

	if err := l.writer.Write(values); err != nil {
		return err
	}
	l.writer.Flush()
	return nil
}

func (l *CSVLogger) Close() error {
	l.writer.Flush()
	return l.file.Close()
}

func flatten(prefix string, v any, out map[string]string) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return
	}
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	flattenValue(prefix, rv, out)
}

func flattenValue(prefix string, rv reflect.Value, out map[string]string) {
	flattenValueDepth(prefix, rv, out, 0)
}

func flattenValueDepth(prefix string, rv reflect.Value, out map[string]string, depth int) {
	if depth > 3 {
		return
	}
	switch rv.Kind() {
	case reflect.Struct:
		t := rv.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			name := jsonName(field)
			if name == "-" || name == "" {
				continue
			}
			key := prefix + "." + name
			fv := rv.Field(i)
			if fv.Kind() == reflect.Ptr && !fv.IsNil() {
				fv = fv.Elem()
			}
			if isNumeric(fv) {
				out[key] = formatFloat(fv)
			} else if fv.Kind() == reflect.Struct || fv.Kind() == reflect.Slice || fv.Kind() == reflect.Array {
				flattenValueDepth(key, fv, out, depth+1)
			}
		}
	case reflect.Slice, reflect.Array:
		limit := 20
		n := rv.Len()
		if n > limit {
			n = limit
		}
		for i := 0; i < n; i++ {
			flattenValueDepth(fmt.Sprintf("%s.%d", prefix, i), rv.Index(i), out, depth+1)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		out[prefix] = fmt.Sprintf("%d", rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		out[prefix] = fmt.Sprintf("%d", rv.Uint())
	case reflect.Float32, reflect.Float64:
		out[prefix] = fmt.Sprintf("%.2f", rv.Float())
	}
}

func jsonName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "" {
		return field.Name
	}
	return parts[0]
}

func isNumeric(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func formatFloat(rv reflect.Value) string {
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.2f", rv.Float())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", rv.Uint())
	}
	return ""
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
