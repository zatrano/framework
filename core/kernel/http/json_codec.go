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
	return json.Marshal(v)
}

func unmarshalJSON(data []byte, v any) error {
	if jsonFnOn.Load() {
		return jsonFnVal.Load().(jsonFn).unmarshal(data, v)
	}
	return json.Unmarshal(data, v)
}
