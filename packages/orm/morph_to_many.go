package orm

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/zatrano/framework/packages/database/query"
)

// MorphToMany returns related models through a polymorphic pivot
// (e.g. taggables: taggable_type, taggable_id, tag_id).
func MorphToMany[Parent, Related any](
	parent *Parent,
	pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue string,
	parentKey ...string,
) ([]Related, error) {
	key := defaultLocalKey[Parent](parentKey...)
	parentID, err := attribute(parent, key)
	if err != nil {
		return nil, err
	}
	db, driver := dbAndDriver[Parent]()
	rows, err := query.New(db, driver, pivotTable).
		Where(morphTypeCol, typeValue).
		Where(morphIDCol, parentID).
		Get()
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

// MorphedByMany is the inverse of MorphToMany: related model → parents of typeValue
// (e.g. Tag → Posts via taggables where tag_id = tag.id and taggable_type = posts).
func MorphedByMany[Related, Parent any](
	related *Related,
	pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue string,
	relatedKey ...string,
) ([]Parent, error) {
	key := defaultLocalKey[Related](relatedKey...)
	relatedID, err := attribute(related, key)
	if err != nil {
		return nil, err
	}
	db, driver := dbAndDriver[Related]()
	rows, err := query.New(db, driver, pivotTable).
		Where(relatedPivotKey, relatedID).
		Where(morphTypeCol, typeValue).
		Get()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []Parent{}, nil
	}
	ids := make([]any, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row[morphIDCol])
	}
	return Query[Parent]().WhereIn(KeyName[Parent](), ids).Get()
}

// AttachMorph inserts polymorphic pivot rows for parent → related ids.
func AttachMorph[Parent any](
	parent *Parent,
	pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue string,
	relatedIDs []any,
	extra ...map[string]any,
) error {
	return AttachMorphOn(DB, parent, pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue, relatedIDs, extra...)
}

// AttachMorphOn inserts polymorphic pivot rows on the given connection/transaction.
func AttachMorphOn[Parent any](
	db query.DBTX,
	parent *Parent,
	pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue string,
	relatedIDs []any,
	extra ...map[string]any,
) error {
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
			morphTypeCol:    typeValue,
			morphIDCol:      parentID,
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

// DetachMorph removes polymorphic pivot rows for parent.
func DetachMorph[Parent any](
	parent *Parent,
	pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue string,
	relatedIDs ...any,
) (int64, error) {
	return DetachMorphOn(DB, parent, pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue, relatedIDs...)
}

// DetachMorphOn removes polymorphic pivot rows on the given connection/transaction.
func DetachMorphOn[Parent any](
	db query.DBTX,
	parent *Parent,
	pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue string,
	relatedIDs ...any,
) (int64, error) {
	if db == nil {
		db = DB
	}
	parentID, err := KeyValue(parent)
	if err != nil {
		return 0, err
	}
	q := query.New(db, Driver, pivotTable).
		Where(morphTypeCol, typeValue).
		Where(morphIDCol, parentID)
	if len(relatedIDs) > 0 {
		q.WhereIn(relatedPivotKey, relatedIDs)
	}
	return q.Delete()
}

// SyncMorph syncs polymorphic pivot ids for parent (detach missing, attach new).
func SyncMorph[Parent any](
	parent *Parent,
	pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue string,
	relatedIDs []any,
	extra ...map[string]any,
) error {
	return Transaction(func(tx *sql.Tx) error {
		if _, err := DetachMorphOn(tx, parent, pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue); err != nil {
			return err
		}
		if len(relatedIDs) == 0 {
			return nil
		}
		return AttachMorphOn(tx, parent, pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue, relatedIDs, extra...)
	})
}

// LoadMorphToMany batch-loads MorphToMany onto parents.
func LoadMorphToMany[Parent, Related any](
	parents *[]Parent,
	field, pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue string,
	parentKey ...string,
) error {
	return LoadMorphToManyFn[Parent, Related](parents, field, pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue, nil, parentKey...)
}

// LoadMorphToManyFn batch-loads MorphToMany with an optional related query constraint.
func LoadMorphToManyFn[Parent, Related any](
	parents *[]Parent,
	field, pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue string,
	constrain func(*Querier[Related]),
	parentKey ...string,
) error {
	if parents == nil || len(*parents) == 0 {
		return nil
	}
	local := defaultLocalKey[Parent](parentKey...)

	keys := make([]any, 0, len(*parents))
	keyIndex := make(map[string][]int, len(*parents))
	for i := range *parents {
		val, err := attribute(&(*parents)[i], local)
		if err != nil {
			return err
		}
		if val == nil {
			continue
		}
		k := fmt.Sprint(val)
		keyIndex[k] = append(keyIndex[k], i)
		keys = append(keys, val)
	}
	if len(keys) == 0 {
		return nil
	}

	db, driver := dbAndDriver[Parent]()
	pivotRows, err := query.New(db, driver, pivotTable).
		Where(morphTypeCol, typeValue).
		WhereIn(morphIDCol, keys).
		Get()
	if err != nil {
		return err
	}
	if len(pivotRows) == 0 {
		return nil
	}

	relatedIDs := make([]any, 0, len(pivotRows))
	parentToRelated := map[string][]any{}
	seenRelated := map[string]bool{}
	for _, row := range pivotRows {
		pk := fmt.Sprint(row[morphIDCol])
		rid := row[relatedPivotKey]
		parentToRelated[pk] = append(parentToRelated[pk], rid)
		rk := fmt.Sprint(rid)
		if !seenRelated[rk] {
			seenRelated[rk] = true
			relatedIDs = append(relatedIDs, rid)
		}
	}

	q := Query[Related]().WhereIn(KeyName[Related](), relatedIDs)
	if constrain != nil {
		constrain(q)
	}
	related, err := q.Get()
	if err != nil {
		return err
	}
	byID := map[string]Related{}
	for _, row := range related {
		id, err := KeyValue(&row)
		if err != nil {
			return err
		}
		byID[fmt.Sprint(id)] = row
	}

	relatedType := reflect.TypeOf((*Related)(nil)).Elem()
	sliceType := reflect.SliceOf(relatedType)
	for key, indices := range keyIndex {
		ids := parentToRelated[key]
		items := make([]Related, 0, len(ids))
		for _, id := range ids {
			if row, ok := byID[fmt.Sprint(id)]; ok {
				items = append(items, row)
			}
		}
		slice := reflect.MakeSlice(sliceType, len(items), len(items))
		for j, item := range items {
			slice.Index(j).Set(reflect.ValueOf(item))
		}
		for _, idx := range indices {
			if err := setRelationField(reflect.ValueOf(&(*parents)[idx]), field, slice); err != nil {
				return err
			}
		}
	}
	return nil
}

// EagerMorphToMany returns a With() loader for MorphToMany.
func EagerMorphToMany[Parent, Related any](
	field, pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue string,
	parentKey ...string,
) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadMorphToMany[Parent, Related](&parents, field, pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue, parentKey...)
	}
}

// EagerMorphToManyFn returns a With() loader for constrained MorphToMany.
func EagerMorphToManyFn[Parent, Related any](
	field, pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue string,
	constrain func(*Querier[Related]),
	parentKey ...string,
) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadMorphToManyFn(&parents, field, pivotTable, relatedPivotKey, morphTypeCol, morphIDCol, typeValue, constrain, parentKey...)
	}
}
