package collection

import (
	"fmt"
	"sort"
)

// Collection is a fluent wrapper around a slice of any values.
type Collection[T any] struct {
	items []T
}

// Make creates a collection from items.
func Make[T any](items ...T) *Collection[T] {
	copied := make([]T, len(items))
	copy(copied, items)
	return &Collection[T]{items: copied}
}

// FromSlice creates a collection from a slice.
func FromSlice[T any](items []T) *Collection[T] {
	return Make(items...)
}

// All returns the underlying items.
func (c *Collection[T]) All() []T {
	out := make([]T, len(c.items))
	copy(out, c.items)
	return out
}

// Count returns the number of items.
func (c *Collection[T]) Count() int {
	return len(c.items)
}

// IsEmpty reports whether the collection is empty.
func (c *Collection[T]) IsEmpty() bool {
	return len(c.items) == 0
}

// IsNotEmpty reports whether the collection has items.
func (c *Collection[T]) IsNotEmpty() bool {
	return !c.IsEmpty()
}

// First returns the first item.
func (c *Collection[T]) First() (T, bool) {
	var zero T
	if len(c.items) == 0 {
		return zero, false
	}
	return c.items[0], true
}

// Last returns the last item.
func (c *Collection[T]) Last() (T, bool) {
	var zero T
	if len(c.items) == 0 {
		return zero, false
	}
	return c.items[len(c.items)-1], true
}

// Push appends an item.
func (c *Collection[T]) Push(item T) *Collection[T] {
	c.items = append(c.items, item)
	return c
}

// Filter keeps items matching the callback.
func (c *Collection[T]) Filter(fn func(T) bool) *Collection[T] {
	out := make([]T, 0)
	for _, item := range c.items {
		if fn(item) {
			out = append(out, item)
		}
	}
	return &Collection[T]{items: out}
}

// Map transforms items.
func Map[T any, R any](c *Collection[T], fn func(T) R) *Collection[R] {
	out := make([]R, 0, len(c.items))
	for _, item := range c.items {
		out = append(out, fn(item))
	}
	return &Collection[R]{items: out}
}

// Each iterates items.
func (c *Collection[T]) Each(fn func(T)) {
	for _, item := range c.items {
		fn(item)
	}
}

// Contains reports whether any item matches.
func (c *Collection[T]) Contains(fn func(T) bool) bool {
	for _, item := range c.items {
		if fn(item) {
			return true
		}
	}
	return false
}

// Take returns the first n items.
func (c *Collection[T]) Take(n int) *Collection[T] {
	if n <= 0 {
		return Make[T]()
	}
	if n > len(c.items) {
		n = len(c.items)
	}
	return FromSlice(c.items[:n])
}

// Skip skips the first n items.
func (c *Collection[T]) Skip(n int) *Collection[T] {
	if n <= 0 {
		return FromSlice(c.items)
	}
	if n >= len(c.items) {
		return Make[T]()
	}
	return FromSlice(c.items[n:])
}

// Reverse reverses the collection.
func (c *Collection[T]) Reverse() *Collection[T] {
	out := make([]T, len(c.items))
	for i, item := range c.items {
		out[len(c.items)-1-i] = item
	}
	return &Collection[T]{items: out}
}

// SortBy sorts using a less function.
func (c *Collection[T]) SortBy(less func(a, b T) bool) *Collection[T] {
	out := c.All()
	sort.SliceStable(out, func(i, j int) bool {
		return less(out[i], out[j])
	})
	return &Collection[T]{items: out}
}

// Unique keeps unique items using a key function.
func (c *Collection[T]) Unique(key func(T) any) *Collection[T] {
	seen := map[string]bool{}
	out := make([]T, 0, len(c.items))
	for _, item := range c.items {
		k := fmt.Sprint(key(item))
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, item)
	}
	return &Collection[T]{items: out}
}

// Pluck extracts values via callback.
func Pluck[T any, R any](c *Collection[T], fn func(T) R) []R {
	out := make([]R, 0, len(c.items))
	for _, item := range c.items {
		out = append(out, fn(item))
	}
	return out
}

// Chunk splits into chunks of size.
func (c *Collection[T]) Chunk(size int) []*Collection[T] {
	if size <= 0 {
		return nil
	}
	out := make([]*Collection[T], 0)
	for i := 0; i < len(c.items); i += size {
		end := i + size
		if end > len(c.items) {
			end = len(c.items)
		}
		out = append(out, FromSlice(c.items[i:end]))
	}
	return out
}

// Reduce reduces the collection to a single value.
func Reduce[T any, R any](c *Collection[T], initial R, fn func(R, T) R) R {
	acc := initial
	for _, item := range c.items {
		acc = fn(acc, item)
	}
	return acc
}

// GroupBy groups items by key.
func GroupBy[T any, K comparable](c *Collection[T], key func(T) K) map[K][]T {
	out := make(map[K][]T)
	if c == nil {
		return out
	}
	for _, item := range c.items {
		k := key(item)
		out[k] = append(out[k], item)
	}
	return out
}

// KeyBy indexes items by key (later items overwrite earlier ones).
func KeyBy[T any, K comparable](c *Collection[T], key func(T) K) map[K]T {
	out := make(map[K]T)
	if c == nil {
		return out
	}
	for _, item := range c.items {
		out[key(item)] = item
	}
	return out
}

// Partition splits items into matching and non-matching collections.
func (c *Collection[T]) Partition(fn func(T) bool) (pass, fail *Collection[T]) {
	passItems := make([]T, 0)
	failItems := make([]T, 0)
	if c != nil {
		for _, item := range c.items {
			if fn(item) {
				passItems = append(passItems, item)
			} else {
				failItems = append(failItems, item)
			}
		}
	}
	return &Collection[T]{items: passItems}, &Collection[T]{items: failItems}
}

// FirstWhere returns the first item matching the callback.
func (c *Collection[T]) FirstWhere(fn func(T) bool) (T, bool) {
	var zero T
	if c == nil {
		return zero, false
	}
	for _, item := range c.items {
		if fn(item) {
			return item, true
		}
	}
	return zero, false
}

// LastWhere returns the last item matching the callback.
func (c *Collection[T]) LastWhere(fn func(T) bool) (T, bool) {
	var zero T
	if c == nil {
		return zero, false
	}
	for i := len(c.items) - 1; i >= 0; i-- {
		if fn(c.items[i]) {
			return c.items[i], true
		}
	}
	return zero, false
}

// FlatMap maps each item to a slice and flattens the result.
func FlatMap[T any, R any](c *Collection[T], fn func(T) []R) *Collection[R] {
	out := make([]R, 0)
	if c != nil {
		for _, item := range c.items {
			out = append(out, fn(item)...)
		}
	}
	return &Collection[R]{items: out}
}

// Flatten flattens a collection of slices into a single collection.
func Flatten[T any](c *Collection[[]T]) *Collection[T] {
	return FlatMap(c, func(items []T) []T { return items })
}

// Sliding returns overlapping windows of the given size and step.
func (c *Collection[T]) Sliding(size, step int) []*Collection[T] {
	if c == nil || size <= 0 {
		return nil
	}
	if step <= 0 {
		step = 1
	}
	out := make([]*Collection[T], 0)
	for i := 0; i+size <= len(c.items); i += step {
		out = append(out, FromSlice(c.items[i:i+size]))
	}
	return out
}
