package http

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func (r *Request) jsonInput() map[string]string {
	r.applyPendingInputTransforms()
	r.ensureJSONParsed()
	return r.jsonData
}

func (r *Request) ensureJSONParsed() {
	if r == nil {
		return
	}
	if r.jsonRead {
		if r.jsonData == nil {
			r.jsonData = map[string]string{}
		}
		return
	}
	r.jsonRead = true
	r.jsonData = map[string]string{}
	r.jsonRaw = map[string]any{}
	if r.raw == nil || r.raw.Body == nil || !r.IsJSON() {
		return
	}
	raw, err := r.readBody()
	if err != nil {
		return
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return
	}
	r.jsonRaw = payload
	for key, value := range payload {
		r.jsonData[key] = stringifyJSON(value)
	}
	flattenJSON("", payload, r.jsonData)
}

func stringifyJSON(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return ""
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(raw)
	}
}
