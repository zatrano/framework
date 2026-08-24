package orm

import (
	"fmt"

	"github.com/zatrano/framework/packages/database/query"
)

// WithRelationSubquery adds a correlated subquery select alias onto the parent query.
func (q *Querier[T]) WithRelationSubquery(alias string, sub *query.Builder) *Querier[T] {
	if q == nil || sub == nil || alias == "" {
		return q
	}
	sqlStr, bindings := sub.ToSQL()
	q.builder.SelectRaw(fmt.Sprintf("(%s) as %s", sqlStr, alias), bindings...)
	return q
}

// OrderByRelationSubquery orders by a correlated subquery expression.
func (q *Querier[T]) OrderByRelationSubquery(sub *query.Builder, direction ...string) *Querier[T] {
	if q == nil || sub == nil {
		return q
	}
	dir := "asc"
	if len(direction) > 0 && direction[0] != "" {
		dir = direction[0]
	}
	sqlStr, bindings := sub.ToSQL()
	q.builder.OrderByRaw(fmt.Sprintf("(%s) %s", sqlStr, dir), bindings...)
	return q
}
