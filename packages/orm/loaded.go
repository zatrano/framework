package orm

import (
	"reflect"
	"sync"
)

var (
	relationsMu   sync.Mutex
	relations     = map[uintptr]map[string]bool{}
	countsBag     = map[uintptr]map[string]int64{}
	pivotAttrsBag = map[uintptr]map[string]any{}
)

// RelationLoaded reports whether a relation field was populated by Load*/With/Nested.
// An empty slice still counts as loaded once marked.
func RelationLoaded[T any](model *T, field string) bool {
	if model == nil || field == "" {
		return false
	}
	key := modelKey(model)
	relationsMu.Lock()
	defer relationsMu.Unlock()
	return relations[key][field]
}

// MarkRelationLoaded records that field is loaded (including empty results).
func MarkRelationLoaded[T any](model *T, field string) {
	if model == nil || field == "" {
		return
	}
	markRelationLoadedPtr(reflect.ValueOf(model).Pointer(), field)
}

func markRelationLoadedPtr(ptr uintptr, field string) {
	if ptr == 0 || field == "" {
		return
	}
	relationsMu.Lock()
	defer relationsMu.Unlock()
	bag, ok := relations[ptr]
	if !ok {
		bag = map[string]bool{}
		relations[ptr] = bag
	}
	bag[field] = true
}

// ClearRelationLoaded clears loaded markers for a model (all fields or named ones).
func ClearRelationLoaded[T any](model *T, fields ...string) {
	if model == nil {
		return
	}
	key := modelKey(model)
	relationsMu.Lock()
	defer relationsMu.Unlock()
	if len(fields) == 0 {
		delete(relations, key)
		delete(countsBag, key)
		delete(existsBag, key)
		delete(aggregatesBag, key)
		delete(pivotAttrsBag, key)
		return
	}
	bag := relations[key]
	if bag != nil {
		for _, f := range fields {
			delete(bag, f)
		}
		if len(bag) == 0 {
			delete(relations, key)
		}
	}
	cb := countsBag[key]
	if cb != nil {
		for _, f := range fields {
			delete(cb, f)
		}
		if len(cb) == 0 {
			delete(countsBag, key)
		}
	}
	eb := existsBag[key]
	if eb != nil {
		for _, f := range fields {
			delete(eb, f)
		}
		if len(eb) == 0 {
			delete(existsBag, key)
		}
	}
	ab := aggregatesBag[key]
	if ab != nil {
		for _, f := range fields {
			delete(ab, f)
		}
		if len(ab) == 0 {
			delete(aggregatesBag, key)
		}
	}
}

// MarkRelationCount stores a hydrated relation aggregate on the model (e.g. withCount).
func MarkRelationCount[T any](model *T, name string, n int64) {
	if model == nil || name == "" {
		return
	}
	key := modelKey(model)
	relationsMu.Lock()
	defer relationsMu.Unlock()
	bag, ok := countsBag[key]
	if !ok {
		bag = map[string]int64{}
		countsBag[key] = bag
	}
	bag[name] = n
}

// RelationCount returns a previously hydrated withCount value (0 if missing).
func RelationCount[T any](model *T, name string) int64 {
	if model == nil || name == "" {
		return 0
	}
	key := modelKey(model)
	relationsMu.Lock()
	defer relationsMu.Unlock()
	return countsBag[key][name]
}

// LoadMissing runs loader only for models whose relation field is not yet loaded.
// After a successful load (or empty result), models are marked loaded for field.
func LoadMissing[T any](models *[]T, field string, loader func([]T) error) error {
	if models == nil || len(*models) == 0 || loader == nil || field == "" {
		return nil
	}
	missingIdx := make([]int, 0, len(*models))
	for i := range *models {
		if !RelationLoaded(&(*models)[i], field) {
			missingIdx = append(missingIdx, i)
		}
	}
	if len(missingIdx) == 0 {
		return nil
	}

	subset := make([]T, len(missingIdx))
	for j, idx := range missingIdx {
		subset[j] = (*models)[idx]
	}
	if err := loader(subset); err != nil {
		return err
	}
	for j, idx := range missingIdx {
		(*models)[idx] = subset[j]
		MarkRelationLoaded(&(*models)[idx], field)
	}
	return nil
}

// EagerMissing wraps a With() loader so it only runs for models missing field.
func EagerMissing[T any](field string, loader func([]T) error) func([]T) error {
	return func(models []T) error {
		return LoadMissing(&models, field, loader)
	}
}

// MarkPivot stores pivot attributes on a related model (belongs-to-many withPivot).
func MarkPivot[T any](model *T, attrs map[string]any) {
	if model == nil {
		return
	}
	key := modelKey(model)
	relationsMu.Lock()
	defer relationsMu.Unlock()
	if attrs == nil {
		delete(pivotAttrsBag, key)
		return
	}
	cp := make(map[string]any, len(attrs))
	for k, v := range attrs {
		cp[k] = v
	}
	pivotAttrsBag[key] = cp
}

// Pivot returns previously hydrated pivot attributes (nil if none).
func Pivot[T any](model *T) map[string]any {
	if model == nil {
		return nil
	}
	key := modelKey(model)
	relationsMu.Lock()
	defer relationsMu.Unlock()
	src := pivotAttrsBag[key]
	if src == nil {
		return nil
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
