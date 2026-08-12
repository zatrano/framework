package orm

import (
	"fmt"
	"sync"
)

var (
	morphMu  sync.RWMutex
	morphMap = map[string]func(id any) (any, error){}
)

// RegisterMorph registers a morph type name to a finder used by MorphTo.
func RegisterMorph(typeName string, finder func(id any) (any, error)) {
	morphMu.Lock()
	defer morphMu.Unlock()
	morphMap[typeName] = finder
}

// MorphMany returns polymorphic related models for a parent.
func MorphMany[Parent, Related any](
	parent *Parent,
	morphTypeCol, morphIDCol, typeValue string,
) ([]Related, error) {
	parentKey, err := KeyValue(parent)
	if err != nil {
		return nil, err
	}
	return Query[Related]().
		Where(morphTypeCol, typeValue).
		Where(morphIDCol, parentKey).
		Get()
}

// MorphOne returns a single polymorphic related model for a parent.
func MorphOne[Parent, Related any](
	parent *Parent,
	morphTypeCol, morphIDCol, typeValue string,
) (*Related, error) {
	items, err := MorphMany[Parent, Related](parent, morphTypeCol, morphIDCol, typeValue)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

// MorphToByTable resolves a morph-to parent when the stored type matches expectedType.
func MorphToByTable[Child, Parent any](
	child *Child,
	morphTypeCol, morphIDCol, expectedType string,
) (*Parent, error) {
	morphType, err := attribute(child, morphTypeCol)
	if err != nil {
		return nil, err
	}
	if fmt.Sprint(morphType) != expectedType {
		return nil, nil
	}
	morphID, err := attribute(child, morphIDCol)
	if err != nil {
		return nil, err
	}
	if morphID == nil {
		return nil, nil
	}
	return Find[Parent](morphID)
}

// MorphTo resolves a morph parent using RegisterMorph.
func MorphTo[Child any](child *Child, morphTypeCol, morphIDCol string) (any, error) {
	morphType, err := attribute(child, morphTypeCol)
	if err != nil {
		return nil, err
	}
	morphID, err := attribute(child, morphIDCol)
	if err != nil {
		return nil, err
	}
	if morphID == nil {
		return nil, nil
	}
	morphMu.RLock()
	finder := morphMap[fmt.Sprint(morphType)]
	morphMu.RUnlock()
	if finder == nil {
		return nil, fmt.Errorf("morph type [%v] is not registered", morphType)
	}
	return finder(morphID)
}
