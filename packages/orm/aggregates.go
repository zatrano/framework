package orm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/zatrano/framework/packages/database/query"
)

var (
	existsBag     = map[uintptr]map[string]bool{}
	aggregatesBag = map[uintptr]map[string]float64{}
)

// MarkRelationExists stores a hydrated withExists flag on the model.
func MarkRelationExists[T any](model *T, name string, ok bool) {
	if model == nil || name == "" {
		return
	}
	key := modelKey(model)
	relationsMu.Lock()
	defer relationsMu.Unlock()
	bag, found := existsBag[key]
	if !found {
		bag = map[string]bool{}
		existsBag[key] = bag
	}
	bag[name] = ok
}

// RelationExists returns a previously hydrated withExists value.
func RelationExists[T any](model *T, name string) bool {
	if model == nil || name == "" {
		return false
	}
	key := modelKey(model)
	relationsMu.Lock()
	defer relationsMu.Unlock()
	return existsBag[key][name]
}

// MarkRelationAggregate stores a hydrated withMax/Min/Avg/Sum value.
func MarkRelationAggregate[T any](model *T, name string, n float64) {
	if model == nil || name == "" {
		return
	}
	key := modelKey(model)
	relationsMu.Lock()
	defer relationsMu.Unlock()
	bag, found := aggregatesBag[key]
	if !found {
		bag = map[string]float64{}
		aggregatesBag[key] = bag
	}
	bag[name] = n
}

// RelationAggregate returns a previously hydrated aggregate (0 if missing).
func RelationAggregate[T any](model *T, name string) float64 {
	if model == nil || name == "" {
		return 0
	}
	key := modelKey(model)
	relationsMu.Lock()
	defer relationsMu.Unlock()
	return aggregatesBag[key][name]
}

// WithExists returns whether each parent has at least one related row (batched).
func WithExists[Parent, Related any](parents []Parent, foreignKey string, localKey ...string) (map[any]bool, error) {
	local := defaultLocalKey[Parent](localKey...)
	out := make(map[any]bool, len(parents))
	if len(parents) == 0 {
		return out, nil
	}
	keys, seen, err := collectParentKeys(parents, local)
	if err != nil {
		return nil, err
	}
	for _, id := range keys {
		out[id] = false
	}
	if len(keys) == 0 {
		return out, nil
	}

	relatedTable := Table[Related]()
	db, driver := dbAndDriver[Related]()
	rows, err := query.New(db, driver, relatedTable).
		Select(foreignKey).
		WhereIn(foreignKey, keys).
		GroupBy(foreignKey).
		Get()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		fk := row[foreignKey]
		out[fk] = true
		if id, ok := seen[fmt.Sprint(fk)]; ok {
			out[id] = true
		}
	}
	return out, nil
}

// LoadExists hydrates related existence onto a bool field and RelationExists bag.
func LoadExists[Parent, Related any](parents *[]Parent, field, foreignKey string, localKey ...string) error {
	if parents == nil || len(*parents) == 0 {
		return nil
	}
	flags, err := WithExists[Parent, Related](*parents, foreignKey, localKey...)
	if err != nil {
		return err
	}
	local := defaultLocalKey[Parent](localKey...)
	for i := range *parents {
		id, err := attribute(&(*parents)[i], local)
		if err != nil {
			return err
		}
		ok := false
		if id != nil {
			if v, found := flags[id]; found {
				ok = v
			} else if v, found := flags[fmt.Sprint(id)]; found {
				ok = v
			}
		}
		MarkRelationExists(&(*parents)[i], field, ok)
		_ = setBoolField(reflect.ValueOf(&(*parents)[i]), field, ok)
	}
	return nil
}

// EagerExists returns a With() loader for withExists hydration.
func EagerExists[Parent, Related any](field, foreignKey string, localKey ...string) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadExists[Parent, Related](&parents, field, foreignKey, localKey...)
	}
}

// WithAggregate returns batched aggregates (sum/avg/min/max) keyed by parent local key.
func WithAggregate[Parent, Related any](parents []Parent, foreignKey, column, fn string, localKey ...string) (map[any]float64, error) {
	fn = strings.ToLower(strings.TrimSpace(fn))
	switch fn {
	case "sum", "avg", "min", "max":
	default:
		return nil, fmt.Errorf("orm: unsupported aggregate [%s]", fn)
	}
	local := defaultLocalKey[Parent](localKey...)
	out := make(map[any]float64, len(parents))
	if len(parents) == 0 {
		return out, nil
	}
	keys, seen, err := collectParentKeys(parents, local)
	if err != nil {
		return nil, err
	}
	for _, id := range keys {
		out[id] = 0
	}
	if len(keys) == 0 {
		return out, nil
	}

	expr := strings.ToUpper(fn) + "(" + column + ") as aggregate"
	relatedTable := Table[Related]()
	db, driver := dbAndDriver[Related]()
	rows, err := query.New(db, driver, relatedTable).
		SelectRaw(foreignKey+", "+expr).
		WhereIn(foreignKey, keys).
		GroupBy(foreignKey).
		Get()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		fk := row[foreignKey]
		n, _ := toFloat64(row["aggregate"])
		out[fk] = n
		if id, ok := seen[fmt.Sprint(fk)]; ok {
			out[id] = n
		}
	}
	return out, nil
}

// LoadAggregate hydrates sum/avg/min/max onto field and RelationAggregate bag.
func LoadAggregate[Parent, Related any](parents *[]Parent, field, foreignKey, column, fn string, localKey ...string) error {
	if parents == nil || len(*parents) == 0 {
		return nil
	}
	values, err := WithAggregate[Parent, Related](*parents, foreignKey, column, fn, localKey...)
	if err != nil {
		return err
	}
	local := defaultLocalKey[Parent](localKey...)
	for i := range *parents {
		id, err := attribute(&(*parents)[i], local)
		if err != nil {
			return err
		}
		n := float64(0)
		if id != nil {
			if v, ok := values[id]; ok {
				n = v
			} else if v, ok := values[fmt.Sprint(id)]; ok {
				n = v
			}
		}
		MarkRelationAggregate(&(*parents)[i], field, n)
		_ = setNumericField(reflect.ValueOf(&(*parents)[i]), field, n)
	}
	return nil
}

// EagerAggregate returns a With() loader for withMax/Min/Avg/Sum hydration.
func EagerAggregate[Parent, Related any](field, foreignKey, column, fn string, localKey ...string) func([]Parent) error {
	return func(parents []Parent) error {
		return LoadAggregate[Parent, Related](&parents, field, foreignKey, column, fn, localKey...)
	}
}

// EagerMax / EagerMin / EagerAvg / EagerSum are convenience With() loaders.
func EagerMax[Parent, Related any](field, foreignKey, column string, localKey ...string) func([]Parent) error {
	return EagerAggregate[Parent, Related](field, foreignKey, column, "max", localKey...)
}
func EagerMin[Parent, Related any](field, foreignKey, column string, localKey ...string) func([]Parent) error {
	return EagerAggregate[Parent, Related](field, foreignKey, column, "min", localKey...)
}
func EagerAvg[Parent, Related any](field, foreignKey, column string, localKey ...string) func([]Parent) error {
	return EagerAggregate[Parent, Related](field, foreignKey, column, "avg", localKey...)
}
func EagerSum[Parent, Related any](field, foreignKey, column string, localKey ...string) func([]Parent) error {
	return EagerAggregate[Parent, Related](field, foreignKey, column, "sum", localKey...)
}

func collectParentKeys[Parent any](parents []Parent, local string) (keys []any, seen map[string]any, err error) {
	seen = map[string]any{}
	keys = make([]any, 0, len(parents))
	for i := range parents {
		id, err := attribute(&parents[i], local)
		if err != nil {
			return nil, nil, err
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
	}
	return keys, seen, nil
}

func setBoolField(parent reflect.Value, name string, ok bool) error {
	if parent.Kind() == reflect.Ptr {
		parent = parent.Elem()
	}
	fv, found := fieldByName(parent, name)
	if !found || !fv.CanSet() {
		return fmt.Errorf("field [%s] not found", name)
	}
	if fv.Kind() != reflect.Bool {
		return fmt.Errorf("field [%s] must be bool", name)
	}
	fv.SetBool(ok)
	return nil
}

func setNumericField(parent reflect.Value, name string, n float64) error {
	if parent.Kind() == reflect.Ptr {
		parent = parent.Elem()
	}
	fv, found := fieldByName(parent, name)
	if !found || !fv.CanSet() {
		return fmt.Errorf("field [%s] not found", name)
	}
	switch fv.Kind() {
	case reflect.Float32, reflect.Float64:
		fv.SetFloat(n)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fv.SetInt(int64(n))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if n < 0 {
			n = 0
		}
		fv.SetUint(uint64(n))
	default:
		return fmt.Errorf("field [%s] must be numeric", name)
	}
	return nil
}
