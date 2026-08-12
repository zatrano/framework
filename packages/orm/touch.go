package orm

import (
	"fmt"
	"reflect"
	"time"
)

// Touch updates updated_at (and optional extra columns) to now.
func Touch[T any](model *T, columns ...string) error {
	if model == nil {
		return fmt.Errorf("model is nil")
	}
	keyVal, err := KeyValue(model)
	if err != nil {
		return err
	}
	now := time.Now()
	attrs := map[string]any{"updated_at": now}
	for _, col := range columns {
		if col == "" || col == "updated_at" {
			continue
		}
		attrs[col] = now
	}
	_, err = Query[T]().Where(KeyName[T](), keyVal).Update(attrs)
	if err != nil {
		return err
	}
	rv := reflect.ValueOf(model).Elem()
	setTimeField(rv, "UpdatedAt", now)
	return nil
}

// TouchByID updates updated_at for a row by id.
func TouchByID[T any](id any) error {
	_, err := Query[T]().Where(KeyName[T](), id).Update(map[string]any{
		"updated_at": time.Now(),
	})
	return err
}

// Refresh reloads the model attributes from the database.
func Refresh[T any](model *T) error {
	if model == nil {
		return fmt.Errorf("model is nil")
	}
	keyVal, err := KeyValue(model)
	if err != nil {
		return err
	}
	fresh, err := Find[T](keyVal)
	if err != nil {
		return err
	}
	if fresh == nil {
		return fmt.Errorf("model not found")
	}
	reflect.ValueOf(model).Elem().Set(reflect.ValueOf(fresh).Elem())
	return nil
}
