package orm

import (
	"fmt"
	"reflect"

	"github.com/zatrano/framework/packages/database/query"
)

// LoadHasMany batch-loads a has-many relation onto parents.
func LoadHasMany[Parent, Related any](parents *[]Parent, field, foreignKey string, localKey ...string) error {
	return LoadHasManyFn[Parent, Related](parents, field, foreignKey, nil, localKey...)
}

// LoadHasManyFn batch-loads a has-many relation with an optional related query constraint.
func LoadHasManyFn[Parent, Related any](parents *[]Parent, field, foreignKey string, constrain func(*Querier[Related]), localKey ...string) error {
	if parents == nil || len(*parents) == 0 {
		return nil
	}
	local := defaultLocalKey[Parent](localKey...)

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

	q := Query[Related]().WhereIn(foreignKey, keys)
	if constrain != nil {
		constrain(q)
	}
	related, err := q.Get()
	if err != nil {
		return err
	}

	grouped := make(map[string][]Related)
	for _, row := range related {
		fk, err := attribute(&row, foreignKey)
		if err != nil {
			return err
		}
		grouped[fmt.Sprint(fk)] = append(grouped[fmt.Sprint(fk)], row)
	}

	relatedType := reflect.TypeOf((*Related)(nil)).Elem()
	sliceType := reflect.SliceOf(relatedType)

	for key, indices := range keyIndex {
		items := grouped[key]
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

// LoadHasOne batch-loads a has-one relation onto parents.
func LoadHasOne[Parent, Related any](parents *[]Parent, field, foreignKey string, localKey ...string) error {
	return LoadHasOneFn[Parent, Related](parents, field, foreignKey, nil, localKey...)
}

// LoadHasOneFn batch-loads a has-one relation with an optional related query constraint.
func LoadHasOneFn[Parent, Related any](parents *[]Parent, field, foreignKey string, constrain func(*Querier[Related]), localKey ...string) error {
	if parents == nil || len(*parents) == 0 {
		return nil
	}
	local := defaultLocalKey[Parent](localKey...)

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

	q := Query[Related]().WhereIn(foreignKey, keys)
	if constrain != nil {
		constrain(q)
	}
	related, err := q.Get()
	if err != nil {
		return err
	}

	grouped := make(map[string]Related)
	for _, row := range related {
		fk, err := attribute(&row, foreignKey)
		if err != nil {
			return err
		}
		k := fmt.Sprint(fk)
		if _, exists := grouped[k]; !exists {
			grouped[k] = row
		}
	}

	relatedType := reflect.TypeOf((*Related)(nil)).Elem()
	for key, indices := range keyIndex {
		row, ok := grouped[key]
		if !ok {
			continue
		}
		val := reflect.ValueOf(row)
		for _, idx := range indices {
			parentVal := reflect.ValueOf(&(*parents)[idx])
			fv, found := fieldByName(parentVal.Elem(), field)
			if !found {
				return fmt.Errorf("field [%s] not found", field)
			}
			if fv.Kind() == reflect.Ptr {
				ptr := reflect.New(relatedType)
				ptr.Elem().Set(val)
				if err := setRelationField(parentVal, field, ptr); err != nil {
					return err
				}
			} else {
				if err := setRelationField(parentVal, field, val); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// LoadBelongsTo batch-loads a belongs-to relation onto children.
func LoadBelongsTo[Child, Parent any](children *[]Child, field, foreignKey string, ownerKey ...string) error {
	return LoadBelongsToFn[Child, Parent](children, field, foreignKey, nil, ownerKey...)
}

// LoadBelongsToFn batch-loads a belongs-to relation with an optional parent query constraint.
func LoadBelongsToFn[Child, Parent any](children *[]Child, field, foreignKey string, constrain func(*Querier[Parent]), ownerKey ...string) error {
	if children == nil || len(*children) == 0 {
		return nil
	}
	owner := defaultLocalKey[Parent](ownerKey...)

	keys := make([]any, 0, len(*children))
	keyIndex := make(map[string][]int, len(*children))
	for i := range *children {
		val, err := attribute(&(*children)[i], foreignKey)
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

	q := Query[Parent]().WhereIn(owner, keys)
	if constrain != nil {
		constrain(q)
	}
	parents, err := q.Get()
	if err != nil {
		return err
	}

	grouped := make(map[string]Parent)
	for _, row := range parents {
		id, err := attribute(&row, owner)
		if err != nil {
			return err
		}
		grouped[fmt.Sprint(id)] = row
	}

	parentType := reflect.TypeOf((*Parent)(nil)).Elem()
	for key, indices := range keyIndex {
		row, ok := grouped[key]
		if !ok {
			continue
		}
		val := reflect.ValueOf(row)
		for _, idx := range indices {
			parentVal := reflect.ValueOf(&(*children)[idx])
			fv, found := fieldByName(parentVal.Elem(), field)
			if !found {
				return fmt.Errorf("field [%s] not found", field)
			}
			if fv.Kind() == reflect.Ptr {
				ptr := reflect.New(parentType)
				ptr.Elem().Set(val)
				if err := setRelationField(parentVal, field, ptr); err != nil {
					return err
				}
			} else {
				if err := setRelationField(parentVal, field, val); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// EagerHasMany returns a With() loader for a has-many relation.
func EagerHasMany[Parent, Related any](field, foreignKey string, localKey ...string) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadHasMany[Parent, Related](&parents, field, foreignKey, localKey...)
	}
}

// EagerHasManyFn returns a With() loader for a constrained has-many relation.
func EagerHasManyFn[Parent, Related any](field, foreignKey string, constrain func(*Querier[Related]), localKey ...string) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadHasManyFn(&parents, field, foreignKey, constrain, localKey...)
	}
}

// EagerHasOne returns a With() loader for a has-one relation.
func EagerHasOne[Parent, Related any](field, foreignKey string, localKey ...string) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadHasOne[Parent, Related](&parents, field, foreignKey, localKey...)
	}
}

// EagerHasOneFn returns a With() loader for a constrained has-one relation.
func EagerHasOneFn[Parent, Related any](field, foreignKey string, constrain func(*Querier[Related]), localKey ...string) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadHasOneFn(&parents, field, foreignKey, constrain, localKey...)
	}
}

// EagerBelongsTo returns a With() loader for a belongs-to relation.
func EagerBelongsTo[Child, Parent any](field, foreignKey string, ownerKey ...string) func([]Child) error {
	return func(children []Child) error {
		return LoadBelongsTo[Child, Parent](&children, field, foreignKey, ownerKey...)
	}
}

// EagerBelongsToFn returns a With() loader for a constrained belongs-to relation.
func EagerBelongsToFn[Child, Parent any](field, foreignKey string, constrain func(*Querier[Parent]), ownerKey ...string) func([]Child) error {
	return func(children []Child) error {
		return LoadBelongsToFn(&children, field, foreignKey, constrain, ownerKey...)
	}
}

// LoadBelongsToMany batch-loads a belongs-to-many relation onto parents.
func LoadBelongsToMany[Parent, Related any](
	parents *[]Parent,
	field, pivotTable, foreignPivotKey, relatedPivotKey string,
	parentKey ...string,
) error {
	return LoadBelongsToManyFn[Parent, Related](parents, field, pivotTable, foreignPivotKey, relatedPivotKey, nil, parentKey...)
}

// LoadBelongsToManyFn batch-loads belongs-to-many with an optional related query constraint.
func LoadBelongsToManyFn[Parent, Related any](
	parents *[]Parent,
	field, pivotTable, foreignPivotKey, relatedPivotKey string,
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
	parentToRelated := map[string][]any{}
	seenRelated := map[string]bool{}
	for _, row := range pivotRows {
		pk := fmt.Sprint(row[foreignPivotKey])
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

// EagerBelongsToMany returns a With() loader for belongs-to-many.
func EagerBelongsToMany[Parent, Related any](
	field, pivotTable, foreignPivotKey, relatedPivotKey string,
	parentKey ...string,
) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadBelongsToMany[Parent, Related](&parents, field, pivotTable, foreignPivotKey, relatedPivotKey, parentKey...)
	}
}

// EagerBelongsToManyFn returns a With() loader for constrained belongs-to-many.
func EagerBelongsToManyFn[Parent, Related any](
	field, pivotTable, foreignPivotKey, relatedPivotKey string,
	constrain func(*Querier[Related]),
	parentKey ...string,
) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadBelongsToManyFn(&parents, field, pivotTable, foreignPivotKey, relatedPivotKey, constrain, parentKey...)
	}
}

// LoadMorphMany batch-loads a morph-many relation onto parents.
func LoadMorphMany[Parent, Related any](parents *[]Parent, field, morphTypeCol, morphIDCol, typeValue string, localKey ...string) error {
	return LoadMorphManyFn[Parent, Related](parents, field, morphTypeCol, morphIDCol, typeValue, nil, localKey...)
}

// LoadMorphManyFn batch-loads morph-many with an optional related query constraint.
func LoadMorphManyFn[Parent, Related any](
	parents *[]Parent,
	field, morphTypeCol, morphIDCol, typeValue string,
	constrain func(*Querier[Related]),
	localKey ...string,
) error {
	if parents == nil || len(*parents) == 0 {
		return nil
	}
	local := defaultLocalKey[Parent](localKey...)

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

	q := Query[Related]().Where(morphTypeCol, typeValue).WhereIn(morphIDCol, keys)
	if constrain != nil {
		constrain(q)
	}
	related, err := q.Get()
	if err != nil {
		return err
	}
	grouped := make(map[string][]Related)
	for _, row := range related {
		fk, err := attribute(&row, morphIDCol)
		if err != nil {
			return err
		}
		grouped[fmt.Sprint(fk)] = append(grouped[fmt.Sprint(fk)], row)
	}

	relatedType := reflect.TypeOf((*Related)(nil)).Elem()
	sliceType := reflect.SliceOf(relatedType)
	for key, indices := range keyIndex {
		items := grouped[key]
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

// EagerMorphMany returns a With() loader for morph-many.
func EagerMorphMany[Parent, Related any](field, morphTypeCol, morphIDCol, typeValue string, localKey ...string) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadMorphMany[Parent, Related](&parents, field, morphTypeCol, morphIDCol, typeValue, localKey...)
	}
}

// EagerMorphManyFn returns a With() loader for constrained morph-many.
func EagerMorphManyFn[Parent, Related any](
	field, morphTypeCol, morphIDCol, typeValue string,
	constrain func(*Querier[Related]),
	localKey ...string,
) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadMorphManyFn(&parents, field, morphTypeCol, morphIDCol, typeValue, constrain, localKey...)
	}
}

// With registers eager loaders executed after Get() or First().
func (q *Querier[T]) With(loaders ...func([]T) error) *Querier[T] {
	q.loaders = append(q.loaders, loaders...)
	return q
}

// Without clears all registered eager loaders.
func (q *Querier[T]) Without() *Querier[T] {
	q.loaders = nil
	return q
}

// WithOnly replaces eager loaders with the given set (drops any previously registered).
func (q *Querier[T]) WithOnly(loaders ...func([]T) error) *Querier[T] {
	q.loaders = append([]func([]T) error{}, loaders...)
	return q
}

// Load is a fluent alias for running loaders on an already-fetched slice.
func Load[T any](models *[]T, loaders ...func([]T) error) error {
	if models == nil || len(*models) == 0 {
		return nil
	}
	for _, loader := range loaders {
		if loader == nil {
			continue
		}
		if err := loader(*models); err != nil {
			return err
		}
	}
	return nil
}

func (q *Querier[T]) runLoaders(items []T) error {
	for _, loader := range q.loaders {
		if loader == nil {
			continue
		}
		if err := loader(items); err != nil {
			return err
		}
	}
	return nil
}

func setRelationField(parent reflect.Value, name string, value reflect.Value) error {
	if parent.Kind() == reflect.Ptr {
		parent = parent.Elem()
	}
	fv, ok := fieldByName(parent, name)
	if !ok {
		return fmt.Errorf("field [%s] not found", name)
	}
	if !fv.CanSet() {
		return fmt.Errorf("field [%s] cannot be set", name)
	}
	fv.Set(value)
	// Track loaded relations for LoadMissing / RelationLoaded.
	if parent.CanAddr() {
		markRelationLoadedPtr(parent.Addr().Pointer(), name)
	}
	return nil
}
