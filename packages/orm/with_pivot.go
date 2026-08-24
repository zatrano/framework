package orm

import (
	"fmt"
	"reflect"

	"github.com/zatrano/framework/packages/database/query"
)

// LoadBelongsToManyWithPivot batch-loads belongs-to-many and hydrates pivot columns
// onto each related model via Pivot()/MarkPivot.
func LoadBelongsToManyWithPivot[Parent, Related any](
	parents *[]Parent,
	field, pivotTable, foreignPivotKey, relatedPivotKey string,
	pivotColumns []string,
	parentKey ...string,
) error {
	return LoadBelongsToManyWithPivotFn[Parent, Related](parents, field, pivotTable, foreignPivotKey, relatedPivotKey, pivotColumns, nil, parentKey...)
}

// LoadBelongsToManyWithPivotFn is LoadBelongsToManyWithPivot with an optional related constraint.
func LoadBelongsToManyWithPivotFn[Parent, Related any](
	parents *[]Parent,
	field, pivotTable, foreignPivotKey, relatedPivotKey string,
	pivotColumns []string,
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
	pivotRows, err := query.New(db, driver, pivotTable).WhereIn(foreignPivotKey, keys).Get()
	if err != nil {
		return err
	}
	if len(pivotRows) == 0 {
		return nil
	}

	relatedIDs := make([]any, 0, len(pivotRows))
	type pivotLink struct {
		parentKey string
		relatedID any
		attrs     map[string]any
	}
	links := make([]pivotLink, 0, len(pivotRows))
	seenRelated := map[string]bool{}
	for _, row := range pivotRows {
		pk := fmt.Sprint(row[foreignPivotKey])
		rid := row[relatedPivotKey]
		attrs := map[string]any{
			foreignPivotKey: row[foreignPivotKey],
			relatedPivotKey: rid,
		}
		for _, col := range pivotColumns {
			if col == "" || col == foreignPivotKey || col == relatedPivotKey {
				continue
			}
			if v, ok := row[col]; ok {
				attrs[col] = v
			}
		}
		links = append(links, pivotLink{parentKey: pk, relatedID: rid, attrs: attrs})
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

	type tagged struct {
		model Related
		attrs map[string]any
	}
	grouped := map[string][]tagged{}
	for _, link := range links {
		row, ok := byID[fmt.Sprint(link.relatedID)]
		if !ok {
			continue
		}
		grouped[link.parentKey] = append(grouped[link.parentKey], tagged{model: row, attrs: link.attrs})
	}

	relatedType := reflect.TypeOf((*Related)(nil)).Elem()
	sliceType := reflect.SliceOf(relatedType)
	for key, indices := range keyIndex {
		items := grouped[key]
		slice := reflect.MakeSlice(sliceType, len(items), len(items))
		for j, item := range items {
			slice.Index(j).Set(reflect.ValueOf(item.model))
		}
		for _, idx := range indices {
			parentVal := reflect.ValueOf(&(*parents)[idx])
			if err := setRelationField(parentVal, field, slice); err != nil {
				return err
			}
			fv, ok := fieldByName(parentVal.Elem(), field)
			if !ok || fv.Kind() != reflect.Slice {
				continue
			}
			for j := 0; j < fv.Len() && j < len(items); j++ {
				elem := fv.Index(j)
				if !elem.CanAddr() {
					continue
				}
				ptr, ok := elem.Addr().Interface().(*Related)
				if !ok {
					continue
				}
				MarkPivot(ptr, items[j].attrs)
			}
		}
	}
	return nil
}

// EagerBelongsToManyWithPivot returns a With() loader that hydrates pivot columns.
func EagerBelongsToManyWithPivot[Parent, Related any](
	field, pivotTable, foreignPivotKey, relatedPivotKey string,
	pivotColumns []string,
	parentKey ...string,
) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadBelongsToManyWithPivot[Parent, Related](&parents, field, pivotTable, foreignPivotKey, relatedPivotKey, pivotColumns, parentKey...)
	}
}
