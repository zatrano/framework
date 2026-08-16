package orm

import (
	"database/sql"
	"fmt"

	"github.com/zatrano/framework/packages/database/query"
)

// Attach inserts pivot rows linking parent to related ids.
func Attach[Parent any](parent *Parent, pivotTable, foreignPivotKey, relatedPivotKey string, relatedIDs []any, extra ...map[string]any) error {
	return AttachOn(DB, parent, pivotTable, foreignPivotKey, relatedPivotKey, relatedIDs, extra...)
}

// AttachOn inserts pivot rows using the given connection/transaction.
func AttachOn[Parent any](db query.DBTX, parent *Parent, pivotTable, foreignPivotKey, relatedPivotKey string, relatedIDs []any, extra ...map[string]any) error {
	if db == nil {
		db = DB
	}
	parentID, err := KeyValue(parent)
	if err != nil {
		return err
	}
	if parentID == nil {
		return fmt.Errorf("parent has no primary key")
	}
	extras := map[string]any{}
	if len(extra) > 0 && extra[0] != nil {
		extras = extra[0]
	}
	for _, relatedID := range relatedIDs {
		attrs := map[string]any{
			foreignPivotKey: parentID,
			relatedPivotKey: relatedID,
		}
		for k, v := range extras {
			attrs[k] = v
		}
		if _, err := query.New(db, Driver, pivotTable).Insert(attrs); err != nil {
			return err
		}
	}
	return nil
}

// Detach removes pivot rows for the given related ids (or all when ids empty).
func Detach[Parent any](parent *Parent, pivotTable, foreignPivotKey, relatedPivotKey string, relatedIDs ...any) (int64, error) {
	return DetachOn(DB, parent, pivotTable, foreignPivotKey, relatedPivotKey, relatedIDs...)
}

// DetachOn removes pivot rows using the given connection/transaction.
func DetachOn[Parent any](db query.DBTX, parent *Parent, pivotTable, foreignPivotKey, relatedPivotKey string, relatedIDs ...any) (int64, error) {
	if db == nil {
		db = DB
	}
	parentID, err := KeyValue(parent)
	if err != nil {
		return 0, err
	}
	q := query.New(db, Driver, pivotTable).Where(foreignPivotKey, parentID)
	if len(relatedIDs) > 0 {
		q.WhereIn(relatedPivotKey, relatedIDs)
	}
	return q.Delete()
}

// Sync detaches missing related ids and attaches new ones inside a transaction.
// Optional extra attributes are applied to newly attached pivot rows.
func Sync[Parent any](parent *Parent, pivotTable, foreignPivotKey, relatedPivotKey string, relatedIDs []any, extra ...map[string]any) error {
	return Transaction(func(tx *sql.Tx) error {
		return syncOn(tx, parent, pivotTable, foreignPivotKey, relatedPivotKey, relatedIDs, false, extra...)
	})
}

// SyncWithoutDetaching attaches missing related ids without removing existing ones.
func SyncWithoutDetaching[Parent any](parent *Parent, pivotTable, foreignPivotKey, relatedPivotKey string, relatedIDs []any, extra ...map[string]any) error {
	return Transaction(func(tx *sql.Tx) error {
		return syncOn(tx, parent, pivotTable, foreignPivotKey, relatedPivotKey, relatedIDs, true, extra...)
	})
}

// Toggle attaches missing ids and detaches existing ones for the given related ids.
func Toggle[Parent any](parent *Parent, pivotTable, foreignPivotKey, relatedPivotKey string, relatedIDs []any, extra ...map[string]any) error {
	return Transaction(func(tx *sql.Tx) error {
		parentID, err := KeyValue(parent)
		if err != nil {
			return err
		}
		rows, err := query.New(tx, Driver, pivotTable).Where(foreignPivotKey, parentID).Get()
		if err != nil {
			return err
		}
		existing := map[string]bool{}
		for _, row := range rows {
			existing[fmt.Sprint(row[relatedPivotKey])] = true
		}
		var attachIDs, detachIDs []any
		for _, id := range relatedIDs {
			if existing[fmt.Sprint(id)] {
				detachIDs = append(detachIDs, id)
			} else {
				attachIDs = append(attachIDs, id)
			}
		}
		if len(detachIDs) > 0 {
			if _, err := DetachOn(tx, parent, pivotTable, foreignPivotKey, relatedPivotKey, detachIDs...); err != nil {
				return err
			}
		}
		if len(attachIDs) > 0 {
			return AttachOn(tx, parent, pivotTable, foreignPivotKey, relatedPivotKey, attachIDs, extra...)
		}
		return nil
	})
}

// WhereHas keeps parents that have matching related rows.
func WhereHas[Parent, Related any](q *Querier[Parent], foreignKey string, localKey ...string) *Querier[Parent] {
	return applyRelationExists[Parent, Related](q, false, foreignKey, nil, localKey...)
}

// WhereDoesntHave keeps parents that have no matching related rows.
func WhereDoesntHave[Parent, Related any](q *Querier[Parent], foreignKey string, localKey ...string) *Querier[Parent] {
	return applyRelationExists[Parent, Related](q, true, foreignKey, nil, localKey...)
}

// WhereHasFn keeps parents with related rows matching callback constraints.
func WhereHasFn[Parent, Related any](q *Querier[Parent], foreignKey string, fn func(*Querier[Related]), localKey ...string) *Querier[Parent] {
	return applyRelationExists(q, false, foreignKey, fn, localKey...)
}

// WhereDoesntHaveFn keeps parents with no related rows matching callback constraints.
func WhereDoesntHaveFn[Parent, Related any](q *Querier[Parent], foreignKey string, fn func(*Querier[Related]), localKey ...string) *Querier[Parent] {
	return applyRelationExists(q, true, foreignKey, fn, localKey...)
}

// Has is an alias for WhereHas starting a fresh query.
func Has[Parent, Related any](foreignKey string, localKey ...string) *Querier[Parent] {
	return WhereHas[Parent, Related](Query[Parent](), foreignKey, localKey...)
}

// DoesntHave is an alias for WhereDoesntHave starting a fresh query.
func DoesntHave[Parent, Related any](foreignKey string, localKey ...string) *Querier[Parent] {
	return WhereDoesntHave[Parent, Related](Query[Parent](), foreignKey, localKey...)
}

func applyRelationExists[Parent, Related any](q *Querier[Parent], not bool, foreignKey string, fn func(*Querier[Related]), localKey ...string) *Querier[Parent] {
	if q == nil {
		q = Query[Parent]()
	}
	local := defaultLocalKey[Parent](localKey...)
	relatedTable := Table[Related]()
	parentTable := q.table

	db, driver := dbAndDriver[Parent]()
	subQ := &Querier[Related]{
		builder:    query.New(db, driver, relatedTable),
		table:      relatedTable,
		softDelete: hasSoftDeletes[Related](),
	}
	if fn != nil {
		fn(subQ)
	}
	sub := subQ.builder
	sub.WhereColumn(relatedTable+"."+foreignKey, "=", parentTable+"."+local)
	if subQ.softDelete {
		sub.WhereNull(relatedTable + ".deleted_at")
	}
	if not {
		q.builder.WhereNotExists(sub)
	} else {
		q.builder.WhereExists(sub)
	}
	return q
}

func syncOn[Parent any](
	db query.DBTX,
	parent *Parent,
	pivotTable, foreignPivotKey, relatedPivotKey string,
	relatedIDs []any,
	withoutDetaching bool,
	extra ...map[string]any,
) error {
	parentID, err := KeyValue(parent)
	if err != nil {
		return err
	}
	rows, err := query.New(db, Driver, pivotTable).Where(foreignPivotKey, parentID).Get()
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for _, row := range rows {
		existing[fmt.Sprint(row[relatedPivotKey])] = true
	}
	wanted := map[string]bool{}
	for _, id := range relatedIDs {
		wanted[fmt.Sprint(id)] = true
	}
	if !withoutDetaching {
		var detachIDs []any
		for _, row := range rows {
			id := row[relatedPivotKey]
			if !wanted[fmt.Sprint(id)] {
				detachIDs = append(detachIDs, id)
			}
		}
		if len(detachIDs) > 0 {
			if _, err := DetachOn(db, parent, pivotTable, foreignPivotKey, relatedPivotKey, detachIDs...); err != nil {
				return err
			}
		}
	}
	var attachIDs []any
	for _, id := range relatedIDs {
		if !existing[fmt.Sprint(id)] {
			attachIDs = append(attachIDs, id)
		}
	}
	if len(attachIDs) > 0 {
		return AttachOn(db, parent, pivotTable, foreignPivotKey, relatedPivotKey, attachIDs, extra...)
	}
	return nil
}
