package http

import (
	"encoding/json"
	"sync/atomic"
)

// jsonFn is an optional replacement for encoding/json on JSON and BindJSON.
// The default path does not load it.
type jsonFn struct {
	marshal   func(any) ([]byte, error)
	unmarshal func([]byte, any) error
}

var (
	jsonFnOn  atomic.Bool
	jsonFnVal atomic.Value
)

// SetJSONCodec replaces encoding/json for JSON, Request.JSON, and BindJSON.
// A nil marshal or unmarshal restores encoding/json. The default is encoding/json.
// Call it before serving. It is process-wide.
func SetJSONCodec(marshal func(any) ([]byte, error), unmarshal func([]byte, any) error) {
	if marshal == nil || unmarshal == nil {
		jsonFnOn.Store(false)
		return
	}
	jsonFnVal.Store(jsonFn{marshal: marshal, unmarshal: unmarshal})
	jsonFnOn.Store(true)
}

func marshalJSON(v any) ([]byte, error) {
	if jsonFnOn.Load() {
		return jsonFnVal.Load().(jsonFn).marshal(v)
	}
	switch m := v.(type) {
	case map[string]string:
		if b, ok := marshalMapStringASCII(m); ok {
			return b, nil
		}
	case map[string]any:
		if b, ok := marshalMapAnyASCII(m); ok {
			return b, nil
		}
	}
	return json.Marshal(v)
}

func unmarshalJSON(data []byte, v any) error {
	if jsonFnOn.Load() {
		return jsonFnVal.Load().(jsonFn).unmarshal(data, v)
	}
	if dest, ok := v.(*map[string]any); ok && fillStringObject(dest, data) {
		return nil
	}
	return json.Unmarshal(data, v)
}

// fillStringObject decodes {} or a one-key object whose value is a string
// with no escapes. Anything else returns false so encoding/json handles it.
func fillStringObject(dest *map[string]any, data []byte) bool {
	if dest == nil {
		return false
	}
	i := skipJSONSpace(data, 0)
	if i >= len(data) || data[i] != '{' {
		return false
	}
	i = skipJSONSpace(data, i+1)
	if i < len(data) && data[i] == '}' {
		if skipJSONSpace(data, i+1) != len(data) {
			return false
		}
		if *dest == nil {
			*dest = map[string]any{}
		} else {
			clear(*dest)
		}
		return true
	}
	key, i, ok := rawJSONString(data, i)
	if !ok {
		return false
	}
	i = skipJSONSpace(data, i)
	if i >= len(data) || data[i] != ':' {
		return false
	}
	val, i, ok := rawJSONString(data, skipJSONSpace(data, i+1))
	if !ok {
		return false
	}
	i = skipJSONSpace(data, i)
	if i >= len(data) || data[i] != '}' || skipJSONSpace(data, i+1) != len(data) {
		return false
	}
	if *dest == nil {
		*dest = map[string]any{key: val}
	} else {
		clear(*dest)
		(*dest)[key] = val
	}
	return true
}

func skipJSONSpace(data []byte, i int) int {
	for i < len(data) {
		switch data[i] {
		case ' ', '\n', '\r', '\t':
			i++
		default:
			return i
		}
	}
	return i
}

func rawJSONString(data []byte, i int) (string, int, bool) {
	if i >= len(data) || data[i] != '"' {
		return "", i, false
	}
	i++
	start := i
	for i < len(data) {
		c := data[i]
		if c == '\\' || c < 0x20 || c >= 0x80 {
			return "", i, false
		}
		if c == '"' {
			return string(data[start:i]), i + 1, true
		}
		i++
	}
	return "", i, false
}

func marshalMapAnyASCII(m map[string]any) ([]byte, bool) {
	if len(m) == 0 {
		return append([]byte(nil), '{', '}'), true
	}
	if len(m) != 1 {
		return nil, false
	}
	var key string
	var val any
	for key, val = range m {
	}
	s, ok := val.(string)
	if !ok {
		return nil, false
	}
	return marshalOneStringASCII(key, s)
}

// marshalMapStringASCII encodes a one-key or empty map without reflection.
// Bytes match encoding/json, including the HTML escape of <, >, and &.
// Any non-ASCII byte falls back to encoding/json. More than one key does too,
// because encoding/json sorts keys and that sort allocates.
func marshalMapStringASCII(m map[string]string) ([]byte, bool) {
	if len(m) > 1 {
		return nil, false
	}
	if len(m) == 0 {
		return append([]byte(nil), '{', '}'), true
	}
	var key, val string
	for key, val = range m {
	}
	return marshalOneStringASCII(key, val)
}

func marshalOneStringASCII(key, val string) ([]byte, bool) {
	ks, ok := jsonASCIISize(key)
	if !ok {
		return nil, false
	}
	vs, ok := jsonASCIISize(val)
	if !ok {
		return nil, false
	}
	buf := make([]byte, 0, ks+vs+7)
	buf = append(buf, '{', '"')
	buf = appendJSONASCII(buf, key)
	buf = append(buf, '"', ':', '"')
	buf = appendJSONASCII(buf, val)
	buf = append(buf, '"', '}')
	return buf, true
}

func jsonASCIISize(s string) (int, bool) {
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x80 {
			return 0, false
		}
		switch c {
		case '"', '\\', '\b', '\f', '\n', '\r', '\t':
			n += 2
		case '&', '<', '>':
			n += 6
		default:
			if c < 0x20 {
				n += 6
			} else {
				n++
			}
		}
	}
	return n, true
}

func appendJSONASCII(dst []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"', '\\':
			dst = append(dst, '\\', c)
		case '\b':
			dst = append(dst, '\\', 'b')
		case '\f':
			dst = append(dst, '\\', 'f')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		case '&':
			dst = append(dst, '\\', 'u', '0', '0', '2', '6')
		case '<':
			dst = append(dst, '\\', 'u', '0', '0', '3', 'c')
		case '>':
			dst = append(dst, '\\', 'u', '0', '0', '3', 'e')
		default:
			if c < 0x20 {
				const hex = "0123456789abcdef"
				dst = append(dst, '\\', 'u', '0', '0', hex[c>>4], hex[c&0xF])
			} else {
				dst = append(dst, c)
			}
		}
	}
	return dst
}
