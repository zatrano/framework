package orm

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	originalsMu sync.Mutex
	originals   = map[uintptr]map[string]any{}
	lastChanges = map[uintptr]map[string]any{}
)

// SyncOriginal stores the current attribute snapshot for dirty tracking.
func SyncOriginal[T any](model *T) {
	if model == nil {
		return
	}
	key := modelKey(model)
	rv := reflect.ValueOf(model).Elem()
	originalsMu.Lock()
	originals[key] = modelToMap(rv)
	originalsMu.Unlock()
}

// GetOriginal returns the original value for a column (or all originals when column is empty).
func GetOriginal[T any](model *T, column ...string) any {
	if model == nil {
		return nil
	}
	key := modelKey(model)
	originalsMu.Lock()
	snap := originals[key]
	originalsMu.Unlock()
	if snap == nil {
		return nil
	}
	if len(column) == 0 || column[0] == "" {
		out := make(map[string]any, len(snap))
		for k, v := range snap {
			out[k] = v
		}
		return out
	}
	return snap[column[0]]
}

// IsDirty reports whether any (or the given) attributes differ from the original snapshot.
func IsDirty[T any](model *T, columns ...string) bool {
	if model == nil {
		return false
	}
	key := modelKey(model)
	originalsMu.Lock()
	snap := originals[key]
	originalsMu.Unlock()
	if snap == nil {
		return true
	}
	current := modelToMap(reflect.ValueOf(model).Elem())
	if len(columns) == 0 {
		if len(current) != len(snap) {
			return true
		}
		for k, v := range current {
			if fmt.Sprint(snap[k]) != fmt.Sprint(v) {
				return true
			}
		}
		return false
	}
	for _, col := range columns {
		if fmt.Sprint(snap[col]) != fmt.Sprint(current[col]) {
			return true
		}
	}
	return false
}

// WasChanged reports whether attributes changed during the last successful Save/Create.
// Unlike IsDirty (compares against SyncOriginal snapshot), WasChanged uses the last save diff.
func WasChanged[T any](model *T, columns ...string) bool {
	if model == nil {
		return false
	}
	key := modelKey(model)
	originalsMu.Lock()
	changes := lastChanges[key]
	originalsMu.Unlock()
	if len(changes) == 0 {
		return false
	}
	if len(columns) == 0 {
		return true
	}
	for _, col := range columns {
		if _, ok := changes[col]; ok {
			return true
		}
	}
	return false
}

// GetChanges returns attributes that changed during the last successful Save/Create.
func GetChanges[T any](model *T) map[string]any {
	if model == nil {
		return nil
	}
	key := modelKey(model)
	originalsMu.Lock()
	changes := lastChanges[key]
	originalsMu.Unlock()
	if changes == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(changes))
	for k, v := range changes {
		out[k] = v
	}
	return out
}

func recordChanges[T any](model *T, before map[string]any) {
	if model == nil {
		return
	}
	current := modelToMap(reflect.ValueOf(model).Elem())
	changes := map[string]any{}
	if before == nil {
		for k, v := range current {
			changes[k] = v
		}
	} else {
		for k, v := range current {
			if fmt.Sprint(before[k]) != fmt.Sprint(v) {
				changes[k] = v
			}
		}
		for k := range before {
			if _, ok := current[k]; !ok {
				changes[k] = nil
			}
		}
	}
	key := modelKey(model)
	originalsMu.Lock()
	lastChanges[key] = changes
	originalsMu.Unlock()
}

func modelKey[T any](model *T) uintptr {
	return reflect.ValueOf(model).Pointer()
}
