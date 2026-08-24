package orm

import (
	"fmt"
	"reflect"
)

// Nested runs child loaders on models already present in parent.field (slice or singular).
// Pair with Eager* loaders so nested relations load in batch (Eloquent "a.b" without N+1).
//
//	Query[User]().With(
//	  Then(
//	    EagerHasMany[User, Post]("Posts", "user_id"),
//	    "Posts",
//	    EagerHasMany[Post, Comment]("Comments", "post_id"),
//	  ),
//	)
func Nested[Parent, Child any](field string, loaders ...func([]Child) error) func([]Parent) error {
	return func(parents []Parent) error {
		if len(parents) == 0 || len(loaders) == 0 {
			return nil
		}
		collected, writeBack, err := collectRelation[Parent, Child](parents, field)
		if err != nil {
			return err
		}
		if len(collected) == 0 {
			return nil
		}
		for _, loader := range loaders {
			if loader == nil {
				continue
			}
			if err := loader(collected); err != nil {
				return err
			}
		}
		return writeBack(collected)
	}
}

// Then runs parentLoader, then Nested child loaders on field — the common "a.b" eager pattern.
func Then[Parent, Child any](
	parentLoader func([]Parent) error,
	field string,
	childLoaders ...func([]Child) error,
) func([]Parent) error {
	return func(parents []Parent) error {
		if parentLoader != nil {
			if err := parentLoader(parents); err != nil {
				return err
			}
		}
		return Nested[Parent, Child](field, childLoaders...)(parents)
	}
}

type relationLoc struct {
	parentIdx int
	kind      int // 0 = slice element, 1 = singular field
	childIdx  int
}

func collectRelation[Parent, Child any](parents []Parent, field string) ([]Child, func([]Child) error, error) {
	childType := reflect.TypeOf((*Child)(nil)).Elem()
	collected := make([]Child, 0)
	locs := make([]relationLoc, 0)

	for i := range parents {
		pv := reflect.ValueOf(&parents[i]).Elem()
		fv, ok := fieldByName(pv, field)
		if !ok {
			return nil, nil, fmt.Errorf("field [%s] not found", field)
		}
		switch fv.Kind() {
		case reflect.Slice:
			for j := 0; j < fv.Len(); j++ {
				item, err := valueAsChild[Child](fv.Index(j), childType)
				if err != nil {
					return nil, nil, err
				}
				collected = append(collected, item)
				locs = append(locs, relationLoc{parentIdx: i, kind: 0, childIdx: j})
			}
		case reflect.Ptr:
			if fv.IsNil() {
				continue
			}
			item, err := valueAsChild[Child](fv, childType)
			if err != nil {
				return nil, nil, err
			}
			collected = append(collected, item)
			locs = append(locs, relationLoc{parentIdx: i, kind: 1})
		case reflect.Struct:
			item, err := valueAsChild[Child](fv, childType)
			if err != nil {
				return nil, nil, err
			}
			collected = append(collected, item)
			locs = append(locs, relationLoc{parentIdx: i, kind: 1})
		default:
			return nil, nil, fmt.Errorf("field [%s] is not a relation container", field)
		}
	}

	writeBack := func(updated []Child) error {
		if len(updated) != len(locs) {
			return fmt.Errorf("orm: nested write-back length mismatch")
		}
		for k, loc := range locs {
			pv := reflect.ValueOf(&parents[loc.parentIdx]).Elem()
			fv, ok := fieldByName(pv, field)
			if !ok {
				return fmt.Errorf("field [%s] not found", field)
			}
			val := reflect.ValueOf(updated[k])
			switch loc.kind {
			case 0:
				elem := fv.Index(loc.childIdx)
				if err := assignRelationValue(elem, val, childType); err != nil {
					return err
				}
			case 1:
				if err := assignRelationValue(fv, val, childType); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return collected, writeBack, nil
}

func valueAsChild[Child any](v reflect.Value, childType reflect.Type) (Child, error) {
	var zero Child
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return zero, fmt.Errorf("orm: nil relation value")
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return zero, fmt.Errorf("orm: nil relation pointer")
		}
		v = v.Elem()
	}
	if !v.IsValid() || v.Type() != childType {
		return zero, fmt.Errorf("orm: relation type mismatch: got %v want %v", v.Type(), childType)
	}
	return v.Interface().(Child), nil
}

func assignRelationValue(dest, src reflect.Value, childType reflect.Type) error {
	if !dest.CanSet() {
		return fmt.Errorf("orm: relation field cannot be set")
	}
	switch dest.Kind() {
	case reflect.Ptr:
		if dest.IsNil() {
			ptr := reflect.New(childType)
			ptr.Elem().Set(src)
			dest.Set(ptr)
			return nil
		}
		dest.Elem().Set(src)
		return nil
	default:
		dest.Set(src)
		return nil
	}
}
