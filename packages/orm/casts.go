package orm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Castable models declare attribute cast types keyed by DB column.
type Castable interface {
	Casts() map[string]string
}

// CastValue applies a named cast to a raw database/input value.
func CastValue(cast string, value any) (any, error) {
	return castIncoming(cast, value)
}

func castsOf(rv reflect.Value) map[string]string {
	if !rv.IsValid() {
		return nil
	}
	if rv.CanAddr() {
		if c, ok := rv.Addr().Interface().(Castable); ok {
			return c.Casts()
		}
	}
	if c, ok := rv.Interface().(Castable); ok {
		return c.Casts()
	}
	return nil
}

func castIncoming(cast string, value any) (any, error) {
	cast = strings.ToLower(strings.TrimSpace(cast))
	if cast == "" || value == nil {
		return value, nil
	}
	switch cast {
	case "bool", "boolean":
		return toBool(value), nil
	case "int", "integer":
		n, err := toInt64(value)
		return n, err
	case "float", "double", "real":
		n, err := toFloat64(value)
		return n, err
	case "string":
		return fmt.Sprint(value), nil
	case "json", "array", "object":
		return toJSONValue(value)
	case "datetime", "timestamp", "date":
		return toTime(value)
	case "encrypted":
		return castEncryptedIn(value)
	default:
		if h, ok := lookupCustomCast(cast); ok && h.In != nil {
			return h.In(value)
		}
		return value, nil
	}
}

func castOutgoing(cast string, value any) any {
	cast = strings.ToLower(strings.TrimSpace(cast))
	if cast == "" || value == nil {
		return value
	}
	switch cast {
	case "json", "array", "object":
		switch v := value.(type) {
		case string:
			return v
		case []byte:
			return string(v)
		default:
			raw, err := json.Marshal(v)
			if err != nil {
				return fmt.Sprint(v)
			}
			return string(raw)
		}
	case "bool", "boolean":
		if toBool(value) {
			return 1
		}
		return 0
	case "encrypted":
		return castEncryptedOut(value)
	default:
		if h, ok := lookupCustomCast(cast); ok && h.Out != nil {
			return h.Out(value)
		}
		return value
	}
}

func toBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int() != 0
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(v).Uint() != 0
	case float32, float64:
		return reflect.ValueOf(v).Float() != 0
	case []byte:
		s := strings.ToLower(strings.TrimSpace(string(v)))
		return s == "1" || s == "true" || s == "yes" || s == "on"
	default:
		s := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
		return s == "1" || s == "true" || s == "yes" || s == "on"
	}
}

func toInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case []byte:
		return strconv.ParseInt(strings.TrimSpace(string(v)), 10, 64)
	case string:
		return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	default:
		return strconv.ParseInt(strings.TrimSpace(fmt.Sprint(v)), 10, 64)
	}
}

func toFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case []byte:
		return strconv.ParseFloat(strings.TrimSpace(string(v)), 64)
	case string:
		return strconv.ParseFloat(strings.TrimSpace(v), 64)
	default:
		return strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(v)), 64)
	}
}

func toJSONValue(value any) (any, error) {
	switch v := value.(type) {
	case map[string]any, []any:
		return v, nil
	case string:
		var dest any
		if err := json.Unmarshal([]byte(v), &dest); err != nil {
			return v, nil
		}
		return dest, nil
	case []byte:
		var dest any
		if err := json.Unmarshal(v, &dest); err != nil {
			return string(v), nil
		}
		return dest, nil
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var dest any
		if err := json.Unmarshal(raw, &dest); err != nil {
			return nil, err
		}
		return dest, nil
	}
}

func toTime(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case *time.Time:
		if v == nil {
			return time.Time{}, fmt.Errorf("nil time")
		}
		return *v, nil
	case string:
		return parseTime(v)
	case []byte:
		return parseTime(string(v))
	default:
		return parseTime(fmt.Sprint(v))
	}
}
