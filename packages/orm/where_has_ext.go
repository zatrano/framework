package orm

import (
	"github.com/zatrano/framework/packages/database/query"
)

// WhereHasMorph keeps parents that have polymorphic related rows of typeValue.
func WhereHasMorph[Parent, Related any](
	q *Querier[Parent],
	morphTypeCol, morphIDCol, typeValue string,
	localKey ...string,
) *Querier[Parent] {
	return applyMorphExists[Parent, Related](q, false, morphTypeCol, morphIDCol, typeValue, nil, localKey...)
}

// WhereDoesntHaveMorph keeps parents with no polymorphic related rows of typeValue.
func WhereDoesntHaveMorph[Parent, Related any](
	q *Querier[Parent],
	morphTypeCol, morphIDCol, typeValue string,
	localKey ...string,
) *Querier[Parent] {
	return applyMorphExists[Parent, Related](q, true, morphTypeCol, morphIDCol, typeValue, nil, localKey...)
}

// WhereHasMorphFn keeps parents with morph related rows matching callback constraints.
func WhereHasMorphFn[Parent, Related any](
	q *Querier[Parent],
	morphTypeCol, morphIDCol, typeValue string,
	fn func(*Querier[Related]),
	localKey ...string,
) *Querier[Parent] {
	return applyMorphExists(q, false, morphTypeCol, morphIDCol, typeValue, fn, localKey...)
}

// WhereDoesntHaveMorphFn keeps parents with no morph related rows matching callback constraints.
func WhereDoesntHaveMorphFn[Parent, Related any](
	q *Querier[Parent],
	morphTypeCol, morphIDCol, typeValue string,
	fn func(*Querier[Related]),
	localKey ...string,
) *Querier[Parent] {
	return applyMorphExists(q, true, morphTypeCol, morphIDCol, typeValue, fn, localKey...)
}

func applyMorphExists[Parent, Related any](
	q *Querier[Parent],
	not bool,
	morphTypeCol, morphIDCol, typeValue string,
	fn func(*Querier[Related]),
	localKey ...string,
) *Querier[Parent] {
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
	sub.Where(morphTypeCol, typeValue)
	sub.WhereColumn(relatedTable+"."+morphIDCol, "=", parentTable+"."+local)
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

// WhereHasThrough keeps parents that have related models via an intermediate table.
func WhereHasThrough[Parent, Through, Related any](
	q *Querier[Parent],
	throughForeignKey, relatedForeignKey string,
	localKey ...string,
) *Querier[Parent] {
	return applyThroughExists[Parent, Through, Related](q, false, throughForeignKey, relatedForeignKey, nil, localKey...)
}

// WhereDoesntHaveThrough keeps parents with no related models via an intermediate table.
func WhereDoesntHaveThrough[Parent, Through, Related any](
	q *Querier[Parent],
	throughForeignKey, relatedForeignKey string,
	localKey ...string,
) *Querier[Parent] {
	return applyThroughExists[Parent, Through, Related](q, true, throughForeignKey, relatedForeignKey, nil, localKey...)
}

// WhereHasThroughFn keeps parents with through-related rows matching related constraints.
func WhereHasThroughFn[Parent, Through, Related any](
	q *Querier[Parent],
	throughForeignKey, relatedForeignKey string,
	fn func(*Querier[Related]),
	localKey ...string,
) *Querier[Parent] {
	return applyThroughExists[Parent, Through, Related](q, false, throughForeignKey, relatedForeignKey, fn, localKey...)
}

func applyThroughExists[Parent, Through, Related any](
	q *Querier[Parent],
	not bool,
	throughForeignKey, relatedForeignKey string,
	fn func(*Querier[Related]),
	localKey ...string,
) *Querier[Parent] {
	if q == nil {
		q = Query[Parent]()
	}
	local := defaultLocalKey[Parent](localKey...)
	throughTable := Table[Through]()
	relatedTable := Table[Related]()
	parentTable := q.table
	relatedKey := KeyName[Related]()

	db, driver := dbAndDriver[Parent]()
	sub := query.New(db, driver, throughTable)
	sub.Join(relatedTable, throughTable+"."+relatedForeignKey, "=", relatedTable+"."+relatedKey)
	sub.WhereColumn(throughTable+"."+throughForeignKey, "=", parentTable+"."+local)
	if hasSoftDeletes[Through]() {
		sub.WhereNull(throughTable + ".deleted_at")
	}
	if hasSoftDeletes[Related]() {
		sub.WhereNull(relatedTable + ".deleted_at")
	}
	if fn != nil {
		relatedQ := &Querier[Related]{
			builder:    sub,
			table:      relatedTable,
			softDelete: hasSoftDeletes[Related](),
		}
		// Callback constraints apply to the shared subquery builder (joined related).
		fn(relatedQ)
	}
	if not {
		q.builder.WhereNotExists(sub)
	} else {
		q.builder.WhereExists(sub)
	}
	return q
}
