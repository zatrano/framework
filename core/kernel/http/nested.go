package http

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// JSONMap returns the decoded JSON object body (nested maps preserved).
func (r *Request) JSONMap() map[string]any {
	_ = r.jsonInput()
	if r.jsonRaw == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(r.jsonRaw))
	for k, v := range r.jsonRaw {
		out[k] = v
	}
	return out
}

// InputAny returns a nested JSON value by dotted path (e.g. "user.profile.name").
// Falls back to string Input for form/query keys when path has no dots or JSON miss.
func (r *Request) InputAny(path string, fallback ...any) any {
	if path == "" {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return nil
	}
	_ = r.jsonInput()
	if r.jsonRaw != nil {
		if v, ok := digMap(r.jsonRaw, path); ok {
			return v
		}
	}
	if !strings.Contains(path, ".") {
		if r.Has(path) {
			return r.Input(path)
		}
	} else if flat, ok := r.jsonData[path]; ok {
		return flat
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return nil
}

// Dot returns a nested value as string via dotted path.
func (r *Request) Dot(path string, fallback ...string) string {
	v := r.InputAny(path)
	if v == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	case json.Number:
		return x.String()
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprint(x)
		}
		return string(b)
	}
}

// HasNested reports whether a dotted JSON path exists.
func (r *Request) HasNested(path string) bool {
	_ = r.jsonInput()
	if r.jsonRaw == nil {
		return false
	}
	_, ok := digMap(r.jsonRaw, path)
	return ok
}

func digMap(data map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var cur any = data
	for _, part := range parts {
		switch m := cur.(type) {
		case map[string]any:
			v, ok := m[part]
			if !ok {
				return nil, false
			}
			cur = v
		default:
			return nil, false
		}
	}
	return cur, true
}

func flattenJSON(prefix string, data map[string]any, out map[string]string) {
	for key, value := range data {
		full := key
		if prefix != "" {
			full = prefix + "." + key
		}
		switch v := value.(type) {
		case map[string]any:
			flattenJSON(full, v, out)
		default:
			out[full] = stringifyJSON(value)
		}
	}
}
