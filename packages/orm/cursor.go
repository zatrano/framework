package orm

import "fmt"

// Cursor iterates models in primary-key order using batched windows (memory-friendly).
// Return a non-nil error from fn to stop early.
func (q *Querier[T]) Cursor(fn func(*T) error) error {
	if fn == nil {
		return fmt.Errorf("orm: cursor callback required")
	}
	key := KeyName[T]()
	const batch = 1000
	var after any
	for {
		cq := q.cloneForCursor()
		cq.OrderBy(key)
		if after != nil {
			cq.Where(key, ">", after)
		}
		cq.Limit(batch)
		rows, err := cq.Get()
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for i := range rows {
			if err := fn(&rows[i]); err != nil {
				return err
			}
			after, err = KeyValue(&rows[i])
			if err != nil {
				return err
			}
		}
		if len(rows) < batch {
			return nil
		}
	}
}

// Lazy is an alias for Cursor.
func (q *Querier[T]) Lazy(fn func(*T) error) error {
	return q.Cursor(fn)
}

func (q *Querier[T]) cloneForCursor() *Querier[T] {
	nq := &Querier[T]{
		builder:          q.builder.Clone(),
		table:            q.table,
		softDelete:       q.softDelete,
		softApplied:      false,
		skipGlobalScopes: q.skipGlobalScopes,
		globalsApplied:   false,
		removedScopes:    copyRemoved(q.removedScopes),
		loaders:          append([]func([]T) error{}, q.loaders...),
	}
	return nq
}

func copyRemoved(in map[string]bool) map[string]bool {
	if in == nil {
		return nil
	}
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
