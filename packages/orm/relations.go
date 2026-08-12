package orm

import (
	"fmt"
	"reflect"

	"github.com/zatrano/framework/packages/database/query"
)

// HasMany returns related models using a foreign key.
func HasMany[Parent any, Related any](parent *Parent, foreignKey string, localKey ...string) ([]Related, error) {
	local := defaultLocalKey[Parent](localKey...)
	parentID, err := attribute(parent, local)
	if err != nil {
		return nil, err
	}
	return Where[Related](foreignKey, parentID).Get()
}

// CountRelated counts related models for a parent.
func CountRelated[Parent any, Related any](parent *Parent, foreignKey string, localKey ...string) (int64, error) {
	local := defaultLocalKey[Parent](localKey...)
	parentID, err := attribute(parent, local)
	if err != nil {
		return 0, err
	}
	return Where[Related](foreignKey, parentID).Count()
}

// WithCount returns a map of parent local-key values to related counts (single batched query).
func WithCount[Parent any, Related any](parents []Parent, foreignKey string, localKey ...string) (map[any]int64, error) {
	local := defaultLocalKey[Parent](localKey...)
	out := make(map[any]int64, len(parents))
	if len(parents) == 0 {
		return out, nil
	}

	keys := make([]any, 0, len(parents))
	seen := map[string]any{}
	for i := range parents {
		id, err := attribute(&parents[i], local)
		if err != nil {
			return nil, err
		}
		if id == nil {
			continue
		}
		k := fmt.Sprint(id)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = id
		keys = append(keys, id)
		out[id] = 0
	}
	if len(keys) == 0 {
		return out, nil
	}

	relatedTable := Table[Related]()
	rows, err := query.New(DB, Driver, relatedTable).
		SelectRaw(foreignKey+", COUNT(*) as aggregate").
		WhereIn(foreignKey, keys).
		GroupBy(foreignKey).
		Get()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		fk := row[foreignKey]
		n, _ := toInt64(row["aggregate"])
		out[fk] = n
		// Also key by string form for callers using Sprint keys inconsistently.
		if id, ok := seen[fmt.Sprint(fk)]; ok {
			out[id] = n
		}
	}
	return out, nil
}

// HasOne returns a related model using a foreign key.
func HasOne[Parent any, Related any](parent *Parent, foreignKey string, localKey ...string) (*Related, error) {
	items, err := HasMany[Parent, Related](parent, foreignKey, localKey...)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

// BelongsTo returns the parent model for a child.
func BelongsTo[Child any, Parent any](child *Child, foreignKey string, ownerKey ...string) (*Parent, error) {
	owner := defaultLocalKey[Parent](ownerKey...)
	fk, err := attribute(child, foreignKey)
	if err != nil {
		return nil, err
	}
	if fk == nil {
		return nil, nil
	}
	return Where[Parent](owner, fk).First()
}

// BelongsToMany returns related models through a pivot table.
func BelongsToMany[Parent any, Related any](
	parent *Parent,
	pivotTable, foreignPivotKey, relatedPivotKey string,
	parentKey ...string,
) ([]Related, error) {
	key := defaultLocalKey[Parent](parentKey...)
	parentID, err := attribute(parent, key)
	if err != nil {
		return nil, err
	}

	rows, err := query.New(DB, Driver, pivotTable).Where(foreignPivotKey, parentID).Get()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []Related{}, nil
	}

	ids := make([]any, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row[relatedPivotKey])
	}
	return Query[Related]().WhereIn(KeyName[Related](), ids).Get()
}

func attribute(model any, name string) (any, error) {
	rv := reflect.ValueOf(model)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	row := modelToMap(rv)
	if value, ok := row[name]; ok {
		return value, nil
	}
	if value, ok := row[toSnake(name)]; ok {
		return value, nil
	}
	return nil, fmt.Errorf("attribute [%s] not found", name)
}
