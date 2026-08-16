package orm

import (
	"fmt"
	"time"
)

// Upsert inserts attrs or updates listed columns when uniqueBy conflicts.
// When update is empty, all non-unique attributes are updated.
func Upsert[T any](attrs map[string]any, uniqueBy []string, update ...string) (int64, error) {
	if len(attrs) == 0 {
		return 0, fmt.Errorf("upsert attributes required")
	}
	if len(uniqueBy) == 0 {
		return 0, fmt.Errorf("upsert unique columns required")
	}
	now := time.Now()
	if _, ok := attrs["created_at"]; !ok {
		attrs["created_at"] = now
	}
	if _, ok := attrs["updated_at"]; !ok {
		attrs["updated_at"] = now
	}
	updateCols := update
	if len(updateCols) == 0 {
		unique := make(map[string]bool, len(uniqueBy))
		for _, col := range uniqueBy {
			unique[col] = true
		}
		for col := range attrs {
			if col == "created_at" || unique[col] {
				continue
			}
			updateCols = append(updateCols, col)
		}
	} else {
		hasUpdated := false
		for _, col := range updateCols {
			if col == "updated_at" {
				hasUpdated = true
				break
			}
		}
		if !hasUpdated {
			updateCols = append(updateCols, "updated_at")
		}
	}
	return tableQuery[T]().Upsert(attrs, uniqueBy, updateCols...)
}
