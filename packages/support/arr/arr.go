package arr

import (
	"cmp"
	"fmt"
	"math/rand"
	"sort"
)

// First returns the first element or fallback.
func First[T any](items []T, fallback ...T) T {
	if len(items) == 0 {
		var zero T
		if len(fallback) > 0 {
			return fallback[0]
		}
		return zero
	}
	return items[0]
}

// Last returns the last element or fallback.
func Last[T any](items []T, fallback ...T) T {
	if len(items) == 0 {
		var zero T
		if len(fallback) > 0 {
			return fallback[0]
		}
		return zero
	}
	return items[len(items)-1]
}

// Contains reports whether value exists in items.
func Contains[T comparable](items []T, value T) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

// DoesntContain reports whether value is absent from items.
func DoesntContain[T comparable](items []T, value T) bool {
	return !Contains(items, value)
}

// ContainsAll reports whether every value exists in items.
func ContainsAll[T comparable](items []T, values ...T) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if !Contains(items, value) {
			return false
		}
	}
	return true
}

// ContainsAny reports whether at least one value exists in items.
func ContainsAny[T comparable](items []T, values ...T) bool {
	for _, value := range values {
		if Contains(items, value) {
			return true
		}
	}
	return false
}

// DoesntContainAny reports whether none of the values exist in items.
func DoesntContainAny[T comparable](items []T, values ...T) bool {
	return !ContainsAny(items, values...)
}

// DoesntContainAll reports whether at least one value is absent from items.
func DoesntContainAll[T comparable](items []T, values ...T) bool {
	return !ContainsAll(items, values...)
}

// IndexOf returns the first index of value, or -1 when absent.
func IndexOf[T comparable](items []T, value T) int {
	for i, item := range items {
		if item == value {
			return i
		}
	}
	return -1
}

// LastIndexOf returns the last index of value, or -1 when absent.
func LastIndexOf[T comparable](items []T, value T) int {
	for i := len(items) - 1; i >= 0; i-- {
		if items[i] == value {
			return i
		}
	}
	return -1
}

// FindIndex returns the first index matching pred, or -1 when none match.
func FindIndex[T any](items []T, pred func(T) bool) int {
	for i, item := range items {
		if pred(item) {
			return i
		}
	}
	return -1
}

// FindLastIndex returns the last index matching pred, or -1 when none match.
func FindLastIndex[T any](items []T, pred func(T) bool) int {
	for i := len(items) - 1; i >= 0; i-- {
		if pred(items[i]) {
			return i
		}
	}
	return -1
}

// HasKey reports whether key exists in input.
func HasKey[K comparable, V any](input map[K]V, key K) bool {
	_, ok := input[key]
	return ok
}

// HasAnyKey reports whether at least one of the keys exists in input.
func HasAnyKey[K comparable, V any](input map[K]V, keys ...K) bool {
	for _, key := range keys {
		if HasKey(input, key) {
			return true
		}
	}
	return false
}

// HasAllKeys reports whether every key exists in input.
func HasAllKeys[K comparable, V any](input map[K]V, keys ...K) bool {
	if len(keys) == 0 {
		return true
	}
	for _, key := range keys {
		if !HasKey(input, key) {
			return false
		}
	}
	return true
}

// MissingKey reports whether key is absent from input.
func MissingKey[K comparable, V any](input map[K]V, key K) bool {
	return !HasKey(input, key)
}

// MissingAnyKey reports whether at least one key is absent from input.
func MissingAnyKey[K comparable, V any](input map[K]V, keys ...K) bool {
	for _, key := range keys {
		if MissingKey(input, key) {
			return true
		}
	}
	return false
}

// MissingAllKeys reports whether every key is absent from input.
func MissingAllKeys[K comparable, V any](input map[K]V, keys ...K) bool {
	if len(keys) == 0 {
		return true
	}
	for _, key := range keys {
		if HasKey(input, key) {
			return false
		}
	}
	return true
}

// IsEmpty reports whether items has no elements.
func IsEmpty[T any](items []T) bool {
	return len(items) == 0
}

// IsNotEmpty reports whether items has at least one element.
func IsNotEmpty[T any](items []T) bool {
	return len(items) > 0
}

// Combine builds a map by pairing keys with values (uses the shorter length).
func Combine[K comparable, V any](keys []K, values []V) map[K]V {
	n := len(keys)
	if len(values) < n {
		n = len(values)
	}
	out := make(map[K]V, n)
	for i := 0; i < n; i++ {
		out[keys[i]] = values[i]
	}
	return out
}

// Compact removes zero-value elements, preserving order.
func Compact[T comparable](items []T) []T {
	var zero T
	out := make([]T, 0, len(items))
	for _, item := range items {
		if item != zero {
			out = append(out, item)
		}
	}
	return out
}

// Sole returns the only element when len(items)==1.
func Sole[T any](items []T) (T, bool) {
	var zero T
	if len(items) != 1 {
		return zero, false
	}
	return items[0], true
}

// Unique returns unique values preserving order.
func Unique[T comparable](items []T) []T {
	seen := make(map[T]struct{}, len(items))
	out := make([]T, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

// UniqueBy returns items unique by key, preserving first occurrence order.
func UniqueBy[T any, K comparable](items []T, key func(T) K) []T {
	seen := make(map[K]struct{}, len(items))
	out := make([]T, 0, len(items))
	for _, item := range items {
		k := key(item)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, item)
	}
	return out
}

// Filter returns items matching predicate.
func Filter[T any](items []T, pred func(T) bool) []T {
	out := make([]T, 0, len(items))
	for _, item := range items {
		if pred(item) {
			out = append(out, item)
		}
	}
	return out
}

// Map transforms items.
func Map[T any, R any](items []T, fn func(T) R) []R {
	out := make([]R, len(items))
	for i, item := range items {
		out[i] = fn(item)
	}
	return out
}

// FlatMap maps each item to a slice and flattens the results.
func FlatMap[T any, R any](items []T, fn func(T) []R) []R {
	out := make([]R, 0, len(items))
	for _, item := range items {
		out = append(out, fn(item)...)
	}
	return out
}

// FilterMap maps items, keeping only results where ok is true.
func FilterMap[T any, R any](items []T, fn func(T) (R, bool)) []R {
	out := make([]R, 0, len(items))
	for _, item := range items {
		if v, ok := fn(item); ok {
			out = append(out, v)
		}
	}
	return out
}

// Pluck extracts string keys from map slices.
func Pluck(items []map[string]any, key string) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		if v, ok := item[key]; ok {
			out = append(out, v)
		}
	}
	return out
}

// Only keeps the given keys from a map.
func Only(input map[string]any, keys ...string) map[string]any {
	out := make(map[string]any, len(keys))
	for _, key := range keys {
		if v, ok := input[key]; ok {
			out[key] = v
		}
	}
	return out
}

// Except removes the given keys from a map.
func Except(input map[string]any, keys ...string) map[string]any {
	skip := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		skip[key] = struct{}{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		if _, ok := skip[key]; ok {
			continue
		}
		out[key] = value
	}
	return out
}

// Keys returns map keys.
func Keys[K comparable, V any](input map[K]V) []K {
	out := make([]K, 0, len(input))
	for key := range input {
		out = append(out, key)
	}
	return out
}

// Values returns map values.
func Values[K comparable, V any](input map[K]V) []V {
	out := make([]V, 0, len(input))
	for _, value := range input {
		out = append(out, value)
	}
	return out
}

// Chunk splits a slice into chunks of size.
func Chunk[T any](items []T, size int) [][]T {
	if size <= 0 {
		return nil
	}
	out := make([][]T, 0)
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		out = append(out, items[i:end])
	}
	return out
}

// SortStrings returns a sorted copy of strings.
func SortStrings(items []string) []string {
	out := append([]string{}, items...)
	sort.Strings(out)
	return out
}

// Wrap ensures value is a slice.
func Wrap(value any) []any {
	switch v := value.(type) {
	case nil:
		return []any{}
	case []any:
		return v
	case []string:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out
	default:
		return []any{v}
	}
}

// Get retrieves a nested map value using dot notation.
func Get(input map[string]any, key string, fallback ...any) any {
	parts := splitDot(key)
	var current any = input
	for _, part := range parts {
		asMap, ok := current.(map[string]any)
		if !ok {
			if len(fallback) > 0 {
				return fallback[0]
			}
			return nil
		}
		next, exists := asMap[part]
		if !exists {
			if len(fallback) > 0 {
				return fallback[0]
			}
			return nil
		}
		current = next
	}
	return current
}

// Has reports whether a nested key exists using dot notation.
func Has(input map[string]any, key string) bool {
	parts := splitDot(key)
	var current any = input
	for _, part := range parts {
		asMap, ok := current.(map[string]any)
		if !ok {
			return false
		}
		next, exists := asMap[part]
		if !exists {
			return false
		}
		current = next
	}
	return true
}

// Set assigns a nested value using dot notation (creates maps as needed).
func Set(input map[string]any, key string, value any) map[string]any {
	if input == nil {
		input = map[string]any{}
	}
	parts := splitDot(key)
	current := input
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return input
		}
		next, ok := current[part].(map[string]any)
		if !ok || next == nil {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	return input
}

// Dot flattens a nested map into dot-notated keys.
func Dot(input map[string]any, prefix ...string) map[string]any {
	pre := ""
	if len(prefix) > 0 {
		pre = prefix[0]
	}
	out := make(map[string]any)
	for key, value := range input {
		full := key
		if pre != "" {
			full = pre + "." + key
		}
		if nested, ok := value.(map[string]any); ok {
			for k, v := range Dot(nested, full) {
				out[k] = v
			}
			continue
		}
		out[full] = value
	}
	return out
}

// Undot expands flat dot-notated keys into a nested map.
func Undot(input map[string]any) map[string]any {
	out := make(map[string]any)
	for key, value := range input {
		Set(out, key, value)
	}
	return out
}

// Flip swaps keys and string values.
func Flip(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[value] = key
	}
	return out
}

// Random returns one random element.
func Random[T any](items []T) (T, bool) {
	var zero T
	if len(items) == 0 {
		return zero, false
	}
	return items[rand.Intn(len(items))], true
}

// TakeRandom returns up to n distinct random elements (order is shuffled).
func TakeRandom[T any](items []T, n int) []T {
	if n <= 0 || len(items) == 0 {
		return []T{}
	}
	shuffled := Shuffle(items)
	if n >= len(shuffled) {
		return shuffled
	}
	return shuffled[:n]
}

// Where filters map rows where key equals value.
func Where(items []map[string]any, key string, value any) []map[string]any {
	out := make([]map[string]any, 0)
	for _, item := range items {
		if v, ok := item[key]; ok && v == value {
			out = append(out, item)
		}
	}
	return out
}

// WhereNot filters map rows where key is missing or not equal to value.
func WhereNot(items []map[string]any, key string, value any) []map[string]any {
	out := make([]map[string]any, 0)
	for _, item := range items {
		v, ok := item[key]
		if !ok || v != value {
			out = append(out, item)
		}
	}
	return out
}

// Collapse flattens one level of nested slices into a single slice.
func Collapse[T any](items [][]T) []T {
	return Flatten(items)
}

func splitDot(key string) []string {
	out := make([]string, 0)
	start := 0
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			out = append(out, key[start:i])
			start = i + 1
		}
	}
	out = append(out, key[start:])
	return out
}

// Join joins values with a separator.
func Join(items []any, sep string) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += sep
		}
		out += fmt.Sprint(item)
	}
	return out
}

// Reverse returns a reversed copy of items.
func Reverse[T any](items []T) []T {
	out := make([]T, len(items))
	for i, item := range items {
		out[len(items)-1-i] = item
	}
	return out
}

// Diff returns items in a that are not in b.
func Diff[T comparable](a, b []T) []T {
	set := make(map[T]struct{}, len(b))
	for _, item := range b {
		set[item] = struct{}{}
	}
	out := make([]T, 0, len(a))
	for _, item := range a {
		if _, ok := set[item]; !ok {
			out = append(out, item)
		}
	}
	return out
}

// Intersect returns items present in both a and b (order of a).
func Intersect[T comparable](a, b []T) []T {
	set := make(map[T]struct{}, len(b))
	for _, item := range b {
		set[item] = struct{}{}
	}
	out := make([]T, 0)
	seen := make(map[T]struct{})
	for _, item := range a {
		if _, ok := set[item]; !ok {
			continue
		}
		if _, dup := seen[item]; dup {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

// Union returns unique items from all sets, preserving first-seen order.
func Union[T comparable](sets ...[]T) []T {
	seen := make(map[T]struct{})
	out := make([]T, 0)
	for _, set := range sets {
		for _, item := range set {
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			out = append(out, item)
		}
	}
	return out
}

// Concat concatenates slices into one (order preserved, duplicates kept).
func Concat[T any](slices ...[]T) []T {
	n := 0
	for _, s := range slices {
		n += len(s)
	}
	out := make([]T, 0, n)
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

// Every reports whether every item matches pred (true when empty).
func Every[T any](items []T, pred func(T) bool) bool {
	for _, item := range items {
		if !pred(item) {
			return false
		}
	}
	return true
}

// Some reports whether any item matches pred.
func Some[T any](items []T, pred func(T) bool) bool {
	for _, item := range items {
		if pred(item) {
			return true
		}
	}
	return false
}

// None reports whether no item matches pred (true when empty).
func None[T any](items []T, pred func(T) bool) bool {
	return !Some(items, pred)
}

// FirstWhere returns the first item matching pred, or optional fallback / zero.
func FirstWhere[T any](items []T, pred func(T) bool, fallback ...T) T {
	for _, item := range items {
		if pred(item) {
			return item
		}
	}
	var zero T
	if len(fallback) > 0 {
		return fallback[0]
	}
	return zero
}

// LastWhere returns the last item matching pred, or optional fallback / zero.
func LastWhere[T any](items []T, pred func(T) bool, fallback ...T) T {
	for i := len(items) - 1; i >= 0; i-- {
		if pred(items[i]) {
			return items[i]
		}
	}
	var zero T
	if len(fallback) > 0 {
		return fallback[0]
	}
	return zero
}

// Shuffle returns a randomly shuffled copy of items.
func Shuffle[T any](items []T) []T {
	out := append([]T{}, items...)
	rand.Shuffle(len(out), func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}

// Flatten flattens one level of nested slices.
func Flatten[T any](items [][]T) []T {
	total := 0
	for _, chunk := range items {
		total += len(chunk)
	}
	out := make([]T, 0, total)
	for _, chunk := range items {
		out = append(out, chunk...)
	}
	return out
}

// Reduce reduces items to a single value.
func Reduce[T any, R any](items []T, initial R, fn func(R, T) R) R {
	acc := initial
	for _, item := range items {
		acc = fn(acc, item)
	}
	return acc
}

// GroupBy groups map rows by the string form of key.
func GroupBy(items []map[string]any, key string) map[string][]map[string]any {
	out := make(map[string][]map[string]any)
	for _, item := range items {
		k := fmt.Sprint(item[key])
		out[k] = append(out[k], item)
	}
	return out
}

// KeyBy indexes map rows by the string form of key (later duplicates win).
func KeyBy(items []map[string]any, key string) map[string]map[string]any {
	out := make(map[string]map[string]any, len(items))
	for _, item := range items {
		out[fmt.Sprint(item[key])] = item
	}
	return out
}

// SortBy returns a stably sorted copy using less.
func SortBy[T any](items []T, less func(a, b T) bool) []T {
	out := append([]T{}, items...)
	sort.SliceStable(out, func(i, j int) bool {
		return less(out[i], out[j])
	})
	return out
}

// Partition splits items into matching and non-matching slices.
func Partition[T any](items []T, pred func(T) bool) (matched, rejected []T) {
	matched = make([]T, 0)
	rejected = make([]T, 0)
	for _, item := range items {
		if pred(item) {
			matched = append(matched, item)
		} else {
			rejected = append(rejected, item)
		}
	}
	return matched, rejected
}

// Take returns the first n items.
func Take[T any](items []T, n int) []T {
	if n <= 0 {
		return []T{}
	}
	if n >= len(items) {
		return append([]T{}, items...)
	}
	return append([]T{}, items[:n]...)
}

// Skip returns items after skipping the first n.
func Skip[T any](items []T, n int) []T {
	if n <= 0 {
		return append([]T{}, items...)
	}
	if n >= len(items) {
		return []T{}
	}
	return append([]T{}, items[n:]...)
}

// TakeWhile returns leading items while pred is true.
func TakeWhile[T any](items []T, pred func(T) bool) []T {
	out := make([]T, 0, len(items))
	for _, item := range items {
		if !pred(item) {
			break
		}
		out = append(out, item)
	}
	return out
}

// SkipWhile skips leading items while pred is true, then returns the rest.
func SkipWhile[T any](items []T, pred func(T) bool) []T {
	i := 0
	for i < len(items) && pred(items[i]) {
		i++
	}
	return append([]T{}, items[i:]...)
}

// TakeUntil returns leading items until pred is true (exclusive).
func TakeUntil[T any](items []T, pred func(T) bool) []T {
	return TakeWhile(items, func(item T) bool { return !pred(item) })
}

// SkipUntil skips items until pred is true, then returns from that item onward.
func SkipUntil[T any](items []T, pred func(T) bool) []T {
	return SkipWhile(items, func(item T) bool { return !pred(item) })
}

// Nth returns every step-th item starting at optional offset.
func Nth[T any](items []T, step int, offset ...int) []T {
	if step <= 0 {
		return []T{}
	}
	start := 0
	if len(offset) > 0 && offset[0] > 0 {
		start = offset[0]
	}
	out := make([]T, 0)
	for i := start; i < len(items); i += step {
		out = append(out, items[i])
	}
	return out
}

// Sliding returns overlapping windows of the given size (optional step, default 1).
func Sliding[T any](items []T, size int, step ...int) [][]T {
	if size <= 0 || len(items) == 0 {
		return nil
	}
	stride := 1
	if len(step) > 0 && step[0] > 0 {
		stride = step[0]
	}
	out := make([][]T, 0)
	for i := 0; i+size <= len(items); i += stride {
		out = append(out, append([]T{}, items[i:i+size]...))
	}
	return out
}

// Reject returns items that do not match pred.
func Reject[T any](items []T, pred func(T) bool) []T {
	return Filter(items, func(item T) bool {
		return !pred(item)
	})
}

// Prepend returns a new slice with values inserted at the front.
func Prepend[T any](items []T, values ...T) []T {
	out := make([]T, 0, len(items)+len(values))
	out = append(out, values...)
	out = append(out, items...)
	return out
}

// Push returns a new slice with values appended.
func Push[T any](items []T, values ...T) []T {
	out := make([]T, 0, len(items)+len(values))
	out = append(out, items...)
	out = append(out, values...)
	return out
}

// Pop returns the last item and the remaining slice.
func Pop[T any](items []T) (T, []T) {
	var zero T
	if len(items) == 0 {
		return zero, []T{}
	}
	return items[len(items)-1], append([]T{}, items[:len(items)-1]...)
}

// Shift returns the first item and the remaining slice.
func Shift[T any](items []T) (T, []T) {
	var zero T
	if len(items) == 0 {
		return zero, []T{}
	}
	return items[0], append([]T{}, items[1:]...)
}

// Forget removes keys from a map (dot notation supported).
func Forget(input map[string]any, keys ...string) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	for _, key := range keys {
		parts := splitDot(key)
		if len(parts) == 0 {
			continue
		}
		if len(parts) == 1 {
			delete(input, parts[0])
			continue
		}
		current := input
		ok := true
		for i := 0; i < len(parts)-1; i++ {
			next, exists := current[parts[i]].(map[string]any)
			if !exists || next == nil {
				ok = false
				break
			}
			current = next
		}
		if ok {
			delete(current, parts[len(parts)-1])
		}
	}
	return input
}

// Pull returns a value by key (dot notation) and removes it from the map.
func Pull(input map[string]any, key string, fallback ...any) any {
	value := Get(input, key, fallback...)
	Forget(input, key)
	return value
}

// Range returns integers from start to end inclusive.
// Optional step defaults to 1, or -1 when start > end.
func Range(start, end int, step ...int) []int {
	inc := 1
	if len(step) > 0 {
		inc = step[0]
	} else if start > end {
		inc = -1
	}
	if inc == 0 {
		return []int{}
	}
	out := make([]int, 0)
	if inc > 0 {
		for i := start; i <= end; i += inc {
			out = append(out, i)
		}
	} else {
		for i := start; i >= end; i += inc {
			out = append(out, i)
		}
	}
	return out
}

// Times builds a slice by invoking fn for indexes 0..n-1.
func Times[T any](n int, fn func(i int) T) []T {
	if n <= 0 || fn == nil {
		return []T{}
	}
	out := make([]T, n)
	for i := 0; i < n; i++ {
		out[i] = fn(i)
	}
	return out
}

// Pair holds a zipped left/right value.
type Pair[A, B any] struct {
	First  A `json:"first"`
	Second B `json:"second"`
}

// Zip pairs elements from two slices until the shorter ends.
func Zip[A, B any](left []A, right []B) []Pair[A, B] {
	n := len(left)
	if len(right) < n {
		n = len(right)
	}
	out := make([]Pair[A, B], n)
	for i := 0; i < n; i++ {
		out[i] = Pair[A, B]{First: left[i], Second: right[i]}
	}
	return out
}

// Pad appends value until the slice reaches size (no-op when already longer).
func Pad[T any](items []T, size int, value T) []T {
	out := append([]T{}, items...)
	for len(out) < size {
		out = append(out, value)
	}
	return out
}

// Duplicates returns values that appear more than once, preserving first-seen order.
func Duplicates[T comparable](items []T) []T {
	counts := make(map[T]int, len(items))
	for _, item := range items {
		counts[item]++
	}
	out := make([]T, 0)
	seen := make(map[T]struct{})
	for _, item := range items {
		if counts[item] < 2 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

// CrossJoin returns the cartesian product of the given sets.
func CrossJoin[T any](sets ...[]T) [][]T {
	if len(sets) == 0 {
		return nil
	}
	result := [][]T{{}}
	for _, set := range sets {
		if len(set) == 0 {
			return [][]T{}
		}
		next := make([][]T, 0, len(result)*len(set))
		for _, prefix := range result {
			for _, item := range set {
				row := make([]T, len(prefix)+1)
				copy(row, prefix)
				row[len(prefix)] = item
				next = append(next, row)
			}
		}
		result = next
	}
	return result
}

// Min returns the smallest element.
func Min[T cmp.Ordered](items []T) (T, bool) {
	var zero T
	if len(items) == 0 {
		return zero, false
	}
	best := items[0]
	for _, item := range items[1:] {
		if item < best {
			best = item
		}
	}
	return best, true
}

// Max returns the largest element.
func Max[T cmp.Ordered](items []T) (T, bool) {
	var zero T
	if len(items) == 0 {
		return zero, false
	}
	best := items[0]
	for _, item := range items[1:] {
		if item > best {
			best = item
		}
	}
	return best, true
}

// Sum returns the total of numeric items.
func Sum[T interface{ ~int | ~int64 | ~float64 }](items []T) T {
	var total T
	for _, item := range items {
		total += item
	}
	return total
}

// Avg returns the arithmetic mean (0 when empty).
func Avg[T interface{ ~int | ~int64 | ~float64 }](items []T) float64 {
	if len(items) == 0 {
		return 0
	}
	var total float64
	for _, item := range items {
		total += float64(item)
	}
	return total / float64(len(items))
}

// Median returns the middle value after sorting (average of two middles when even).
func Median[T interface{ ~int | ~int64 | ~float64 }](items []T) (float64, bool) {
	if len(items) == 0 {
		return 0, false
	}
	sorted := append([]T(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return float64(sorted[mid]), true
	}
	return (float64(sorted[mid-1]) + float64(sorted[mid])) / 2, true
}

// Mode returns every value that appears most often (stable order of first occurrence).
func Mode[T comparable](items []T) []T {
	if len(items) == 0 {
		return nil
	}
	counts := make(map[T]int, len(items))
	max := 0
	for _, item := range items {
		counts[item]++
		if counts[item] > max {
			max = counts[item]
		}
	}
	out := make([]T, 0)
	seen := make(map[T]struct{})
	for _, item := range items {
		if counts[item] != max {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

// CountBy returns occurrence counts keyed by value.
func CountBy[T comparable](items []T) map[T]int {
	out := make(map[T]int, len(items))
	for _, item := range items {
		out[item]++
	}
	return out
}

// Product returns the product of numeric items (1 when empty).
func Product[T interface{ ~int | ~int64 | ~float64 }](items []T) T {
	var product T = 1
	for _, item := range items {
		product *= item
	}
	return product
}
