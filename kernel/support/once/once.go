package once

import "sync"

var named sync.Map

// Do runs fn at most once for the given key (process-wide).
func Do(key string, fn func()) {
	actual, _ := named.LoadOrStore(key, &sync.Once{})
	actual.(*sync.Once).Do(fn)
}

// Reset clears a named once key (tests).
func Reset(key string) {
	named.Delete(key)
}

// Value returns a function that computes fn once and caches the result.
func Value[T any](fn func() T) func() T {
	var (
		once  sync.Once
		value T
	)
	return func() T {
		once.Do(func() {
			value = fn()
		})
		return value
	}
}

// Memo returns a memoized version of fn keyed by argument.
func Memo[K comparable, V any](fn func(K) V) func(K) V {
	var (
		mu    sync.Mutex
		cache = make(map[K]V)
	)
	return func(key K) V {
		mu.Lock()
		defer mu.Unlock()
		if v, ok := cache[key]; ok {
			return v
		}
		v := fn(key)
		cache[key] = v
		return v
	}
}
