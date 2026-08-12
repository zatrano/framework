package orm

import (
	"fmt"
	"reflect"
)

// PrimaryKey models declare a non-default primary key column name.
type PrimaryKey interface {
	PrimaryKey() string
}

// KeyName returns the primary key column for model T (default "id").
func KeyName[T any]() string {
	return keyNameForType(reflect.TypeOf((*T)(nil)).Elem())
}

func keyNameForType(rt reflect.Type) string {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	zero := reflect.New(rt).Interface()
	if pk, ok := zero.(PrimaryKey); ok {
		if name := pk.PrimaryKey(); name != "" {
			return name
		}
	}
	if m := reflect.New(rt).MethodByName("PrimaryKey"); m.IsValid() {
		results := m.Call(nil)
		if len(results) == 1 {
			if name := results[0].String(); name != "" {
				return name
			}
		}
	}
	return "id"
}

// KeyValue returns the primary key value for a model instance.
func KeyValue(model any) (any, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}
	rv := reflect.ValueOf(model)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("model is nil")
		}
		rv = rv.Elem()
	}
	keyName := keyNameForType(rv.Type())
	fv, ok := fieldValueByColumn(rv, keyName)
	if !ok || !fv.IsValid() {
		return nil, fmt.Errorf("primary key field [%s] not found", keyName)
	}
	if isReflectZero(fv) {
		return nil, fmt.Errorf("model has no %s", keyName)
	}
	if fv.Kind() == reflect.Ptr {
		if fv.IsNil() {
			return nil, fmt.Errorf("model has no %s", keyName)
		}
		return fv.Elem().Interface(), nil
	}
	return fv.Interface(), nil
}

func defaultLocalKey[T any](override ...string) string {
	if len(override) > 0 && override[0] != "" {
		return override[0]
	}
	return KeyName[T]()
}

func fieldValueByColumn(rv reflect.Value, column string) (reflect.Value, bool) {
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fv := rv.Field(i)
		if field.Anonymous {
			if inner, ok := fieldValueByColumn(fv, column); ok {
				return inner, true
			}
			continue
		}
		if columnName(field) == column {
			return fv, true
		}
	}
	return reflect.Value{}, false
}

func fieldByName(rv reflect.Value, name string) (reflect.Value, bool) {
	fv := rv.FieldByName(name)
	if fv.IsValid() {
		return fv, true
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.Anonymous {
			if inner, ok := fieldByName(rv.Field(i), name); ok {
				return inner, true
			}
		}
	}
	return reflect.Value{}, false
}

func isReflectZero(fv reflect.Value) bool {
	if !fv.IsValid() {
		return true
	}
	switch fv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map:
		return fv.IsNil()
	case reflect.String:
		return fv.String() == ""
	default:
		return fv.IsZero()
	}
}

func isZeroAny(v any) bool {
	if v == nil {
		return true
	}
	return isReflectZero(reflect.ValueOf(v))
}

func setKeyField(rv reflect.Value, keyName string, value any) {
	fv, ok := fieldValueByColumn(rv, keyName)
	if !ok || !fv.CanSet() {
		return
	}
	_ = setField(fv, value)
}
