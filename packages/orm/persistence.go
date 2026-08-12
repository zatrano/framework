package orm

import (
	"database/sql"
	"fmt"
	"reflect"
)

// FirstOrCreate finds a model matching attrs or creates one.
// Optional values are merged into attrs when creating. Returns (model, created, error).
func FirstOrCreate[T any](attrs map[string]any, values ...map[string]any) (*T, bool, error) {
	if len(attrs) == 0 {
		return nil, false, fmt.Errorf("first or create requires attributes")
	}
	q := Query[T]()
	for column, value := range attrs {
		q.Where(column, value)
	}
	model, err := q.First()
	if err == nil {
		return model, false, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}
	merged := mergeAttrs(attrs, values...)
	created, err := Create[T](merged)
	if err != nil {
		return nil, false, err
	}
	return created, true, nil
}

// FirstOrNew finds a model matching attrs or returns an unsaved instance
// hydrated from attrs merged with optional values. Returns (model, exists, error).
func FirstOrNew[T any](attrs map[string]any, values ...map[string]any) (*T, bool, error) {
	if len(attrs) == 0 {
		return nil, false, fmt.Errorf("first or new requires attributes")
	}
	q := Query[T]()
	for column, value := range attrs {
		q.Where(column, value)
	}
	model, err := q.First()
	if err == nil {
		return model, true, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}
	merged := mergeAttrs(attrs, values...)
	inst, err := mapToModel[T](merged)
	if err != nil {
		return nil, false, err
	}
	return inst, false, nil
}

// UpdateOrCreate finds a model matching attrs and updates it with values,
// or creates a new model from attrs merged with values. Returns (model, created, error).
func UpdateOrCreate[T any](attrs map[string]any, values map[string]any) (*T, bool, error) {
	if len(attrs) == 0 {
		return nil, false, fmt.Errorf("update or create requires attributes")
	}
	q := Query[T]()
	for column, value := range attrs {
		q.Where(column, value)
	}
	model, err := q.First()
	if err == nil {
		if len(values) > 0 {
			rv := reflect.ValueOf(model).Elem()
			for column, value := range values {
				if fv, ok := fieldValueByColumn(rv, column); ok && fv.CanSet() {
					_ = setField(fv, value)
				}
			}
			if err := Save(model); err != nil {
				return nil, false, err
			}
			return model, false, nil
		}
		return model, false, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, err
	}
	merged := mergeAttrs(attrs, values)
	created, err := Create[T](merged)
	if err != nil {
		return nil, false, err
	}
	return created, true, nil
}

func mergeAttrs(attrs map[string]any, values ...map[string]any) map[string]any {
	merged := make(map[string]any, len(attrs)+8)
	for k, v := range attrs {
		merged[k] = v
	}
	if len(values) > 0 {
		for k, v := range values[0] {
			merged[k] = v
		}
	}
	return merged
}

func modelID[T any](model *T) (any, error) {
	return KeyValue(model)
}
