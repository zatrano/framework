package orm

import (
	"reflect"
	"strings"
	"sync"
)

// ScopeFunc customizes a querier (local scope).
type ScopeFunc[T any] func(q *Querier[T]) *Querier[T]

type globalScopeEntry struct {
	name string
	fn   any
}

var (
	globalScopeMu sync.RWMutex
	globalScopes  = map[reflect.Type][]globalScopeEntry{}
	localScopeMu  sync.RWMutex
	localScopes   = map[reflect.Type]map[string]any{}
)

// AddGlobalScope registers a named global scope for model T.
func AddGlobalScope[T any](name string, fn ScopeFunc[T]) {
	var zero T
	key := reflect.TypeOf(zero)
	globalScopeMu.Lock()
	defer globalScopeMu.Unlock()
	list := globalScopes[key]
	for i, entry := range list {
		if entry.name == name {
			list[i].fn = fn
			globalScopes[key] = list
			return
		}
	}
	globalScopes[key] = append(list, globalScopeEntry{name: name, fn: fn})
}

// RegisterScope registers a named local scope for model T.
func RegisterScope[T any](name string, fn ScopeFunc[T]) {
	name = strings.TrimSpace(name)
	if name == "" || fn == nil {
		return
	}
	var zero T
	key := reflect.TypeOf(zero)
	localScopeMu.Lock()
	defer localScopeMu.Unlock()
	if localScopes[key] == nil {
		localScopes[key] = map[string]any{}
	}
	localScopes[key][name] = fn
}

// HasScope reports whether a named local scope exists for model T.
func HasScope[T any](name string) bool {
	var zero T
	key := reflect.TypeOf(zero)
	localScopeMu.RLock()
	defer localScopeMu.RUnlock()
	_, ok := localScopes[key][name]
	return ok
}

// ClearLocalScopes removes named local scopes for model T (tests).
func ClearLocalScopes[T any]() {
	var zero T
	key := reflect.TypeOf(zero)
	localScopeMu.Lock()
	defer localScopeMu.Unlock()
	delete(localScopes, key)
}

// ClearGlobalScopes removes all global scopes for model T (mainly for tests).
func ClearGlobalScopes[T any]() {
	var zero T
	key := reflect.TypeOf(zero)
	globalScopeMu.Lock()
	defer globalScopeMu.Unlock()
	delete(globalScopes, key)
}

// Scope applies a local scope.
func (q *Querier[T]) Scope(fn ScopeFunc[T]) *Querier[T] {
	if fn == nil {
		return q
	}
	return fn(q)
}

// NamedScope applies a previously registered named local scope.
func (q *Querier[T]) NamedScope(name string) *Querier[T] {
	var zero T
	key := reflect.TypeOf(zero)
	localScopeMu.RLock()
	raw, ok := localScopes[key][name]
	localScopeMu.RUnlock()
	if !ok {
		return q
	}
	fn, _ := raw.(ScopeFunc[T])
	return q.Scope(fn)
}

// Scopes applies multiple local scopes.
func (q *Querier[T]) Scopes(fns ...ScopeFunc[T]) *Querier[T] {
	for _, fn := range fns {
		q = q.Scope(fn)
	}
	return q
}

// When applies a scope only when condition is true.
func (q *Querier[T]) When(condition bool, fn ScopeFunc[T]) *Querier[T] {
	if condition {
		return q.Scope(fn)
	}
	return q
}

// Unless applies a scope only when condition is false.
func (q *Querier[T]) Unless(condition bool, fn ScopeFunc[T]) *Querier[T] {
	return q.When(!condition, fn)
}

// Tap runs fn with the querier and returns it for chaining.
func (q *Querier[T]) Tap(fn func(*Querier[T])) *Querier[T] {
	if fn != nil {
		fn(q)
	}
	return q
}

// GetBindings returns the bound arguments for the compiled SQL.
func (q *Querier[T]) GetBindings() []any {
	q.prepare()
	return q.builder.GetBindings()
}

// ToSQL returns the compiled SQL and bindings.
func (q *Querier[T]) ToSQL() (string, []any) {
	q.prepare()
	return q.builder.ToSQL()
}

// WithoutGlobalScopes disables all global scopes for this query.
func (q *Querier[T]) WithoutGlobalScopes() *Querier[T] {
	q.skipGlobalScopes = true
	q.removedScopes = nil
	return q
}

// WithoutGlobalScope disables one named global scope.
func (q *Querier[T]) WithoutGlobalScope(name string) *Querier[T] {
	if q.removedScopes == nil {
		q.removedScopes = map[string]bool{}
	}
	q.removedScopes[name] = true
	return q
}

func (q *Querier[T]) applyGlobalScopes() {
	if q.skipGlobalScopes || q.globalsApplied {
		return
	}
	q.globalsApplied = true

	var zero T
	key := reflect.TypeOf(zero)
	globalScopeMu.RLock()
	list := append([]globalScopeEntry{}, globalScopes[key]...)
	globalScopeMu.RUnlock()

	for _, entry := range list {
		if q.removedScopes != nil && q.removedScopes[entry.name] {
			continue
		}
		if fn, ok := entry.fn.(ScopeFunc[T]); ok && fn != nil {
			_ = fn(q)
		}
	}
}

func (q *Querier[T]) prepare() {
	q.applyGlobalScopes()
	q.applySoftDelete()
}
