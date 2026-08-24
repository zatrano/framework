package orm

import (
	"encoding/json"
	"reflect"
)

// Hidden models omit listed attribute names from ToMap/ToJSON.
type Hidden interface {
	Hidden() []string
}

// Visible models include only listed attribute names in ToMap/ToJSON
// (takes precedence over Hidden when both are implemented).
type Visible interface {
	Visible() []string
}

// ToMap serializes model attributes honouring Visible/Hidden interfaces.
// Relation fields tagged `db:"-"` are included when non-zero unless hidden.
func ToMap[T any](model *T) map[string]any {
	if model == nil {
		return nil
	}
	rv := reflect.ValueOf(model).Elem()
	raw := modelToMapIncludingRelations(rv)
	raw = applyAccessors(model, raw)

	ptr := any(model)
	if visible, ok := ptr.(Visible); ok {
		allowed := map[string]bool{}
		for _, col := range visible.Visible() {
			allowed[col] = true
			allowed[toSnake(col)] = true
		}
		out := make(map[string]any, len(allowed))
		for k, v := range raw {
			if allowed[k] {
				out[k] = v
			}
		}
		return out
	}
	if hidden, ok := ptr.(Hidden); ok {
		blocked := map[string]bool{}
		for _, col := range hidden.Hidden() {
			blocked[col] = true
			blocked[toSnake(col)] = true
		}
		out := make(map[string]any, len(raw))
		for k, v := range raw {
			if !blocked[k] {
				out[k] = v
			}
		}
		return out
	}
	return raw
}

// ToJSON marshals ToMap output.
func ToJSON[T any](model *T) ([]byte, error) {
	return json.Marshal(ToMap(model))
}

// ToArray is an alias for ToMap (serialization parity naming).
func ToArray[T any](model *T) map[string]any {
	return ToMap(model)
}

func modelToMapIncludingRelations(rv reflect.Value) map[string]any {
	out := modelToMap(rv)
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			continue
		}
		tag := field.Tag.Get("db")
		if tag != "-" {
			continue
		}
		name := field.Tag.Get("json")
		if name == "" || name == "-" {
			name = toSnake(field.Name)
		} else if comma := indexByte(name, ','); comma >= 0 {
			name = name[:comma]
		}
		fv := rv.Field(i)
		if isReflectZero(fv) {
			continue
		}
		out[name] = fv.Interface()
	}
	return out
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
