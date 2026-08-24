package orm

import "fmt"

var preventLazyLoading bool

// PreventLazyLoading enables strict mode: lazy relation helpers panic/error
// when called without a prior eager Load*/With (Eloquent preventLazyLoading).
func PreventLazyLoading(enabled bool) {
	preventLazyLoading = enabled
}

// LazyLoadingPrevented reports whether strict lazy-loading mode is on.
func LazyLoadingPrevented() bool { return preventLazyLoading }

func guardLazy[T any](model *T, field string) error {
	if !preventLazyLoading || model == nil {
		return nil
	}
	if field != "" && RelationLoaded(model, field) {
		return nil
	}
	return fmt.Errorf("orm: lazy loading is prevented (attempted relation %q)", field)
}

// LazyHasMany is HasMany with PreventLazyLoading guard (pass field name used for RelationLoaded).
func LazyHasMany[Parent, Related any](parent *Parent, field, foreignKey string, localKey ...string) ([]Related, error) {
	if err := guardLazy(parent, field); err != nil {
		return nil, err
	}
	items, err := HasMany[Parent, Related](parent, foreignKey, localKey...)
	if err != nil {
		return nil, err
	}
	if field != "" {
		MarkRelationLoaded(parent, field)
	}
	return items, nil
}

// LazyBelongsTo is BelongsTo with PreventLazyLoading guard.
func LazyBelongsTo[Child, Parent any](child *Child, field, foreignKey string, ownerKey ...string) (*Parent, error) {
	if err := guardLazy(child, field); err != nil {
		return nil, err
	}
	item, err := BelongsTo[Child, Parent](child, foreignKey, ownerKey...)
	if err != nil {
		return nil, err
	}
	if field != "" {
		MarkRelationLoaded(child, field)
	}
	return item, nil
}
