package orm

import (
	"reflect"
)

// DeleteModel soft-deletes or hard-deletes a model instance by its id.
func DeleteModel[T any](model *T) (int64, error) {
	id, err := modelID(model)
	if err != nil {
		return 0, err
	}
	if err := dispatchModel("deleting", model); err != nil {
		return 0, err
	}
	n, err := Query[T]().Where(KeyName[T](), id).Delete()
	if err != nil {
		return 0, err
	}
	_ = dispatchModel("deleted", model)
	return n, nil
}

// ForceDeleteModel permanently deletes a model instance.
func ForceDeleteModel[T any](model *T) (int64, error) {
	id, err := modelID(model)
	if err != nil {
		return 0, err
	}
	if err := dispatchModel("deleting", model); err != nil {
		return 0, err
	}
	n, err := Query[T]().WithTrashed().Where(KeyName[T](), id).ForceDelete()
	if err != nil {
		return 0, err
	}
	_ = dispatchModel("forceDeleted", model)
	return n, nil
}

// RestoreModel restores a soft-deleted model instance.
func RestoreModel[T any](model *T) (int64, error) {
	id, err := modelID(model)
	if err != nil {
		return 0, err
	}
	_ = dispatchModel("restoring", model)
	n, err := Query[T]().OnlyTrashed().Where(KeyName[T](), id).Restore()
	if err != nil {
		return 0, err
	}
	_ = dispatchModel("restored", model)
	return n, nil
}

// Replicate returns a copy of the model without primary key and timestamps.
func Replicate[T any](model *T) *T {
	if model == nil {
		return nil
	}
	src := reflect.ValueOf(model).Elem()
	dst := reflect.New(src.Type()).Elem()
	dst.Set(src)

	keyName := KeyName[T]()
	zeroColumn(dst, keyName)
	if id := dst.FieldByName("ID"); id.IsValid() && id.CanSet() {
		id.SetZero()
	}
	for _, name := range []string{"CreatedAt", "UpdatedAt", "DeletedAt"} {
		if f := dst.FieldByName(name); f.IsValid() && f.CanSet() {
			f.SetZero()
		}
	}
	for i := 0; i < dst.NumField(); i++ {
		sf := dst.Type().Field(i)
		if sf.Anonymous && sf.Type == reflect.TypeOf(SoftDeletes{}) {
			if del := dst.Field(i).FieldByName("DeletedAt"); del.IsValid() && del.CanSet() {
				del.SetZero()
			}
		}
		if sf.Anonymous && sf.Type == reflect.TypeOf(Model{}) {
			emb := dst.Field(i)
			if id := emb.FieldByName("ID"); id.IsValid() && id.CanSet() {
				id.SetZero()
			}
			for _, name := range []string{"CreatedAt", "UpdatedAt"} {
				if f := emb.FieldByName(name); f.IsValid() && f.CanSet() {
					f.SetZero()
				}
			}
		}
	}
	out := dst.Interface().(T)
	_ = dispatchModel("replicating", model)
	return &out
}

func zeroColumn(dst reflect.Value, column string) {
	if column == "" {
		return
	}
	for i := 0; i < dst.NumField(); i++ {
		sf := dst.Type().Field(i)
		tag := sf.Tag.Get("db")
		if tag == column || toSnake(sf.Name) == column {
			if dst.Field(i).CanSet() {
				dst.Field(i).SetZero()
			}
			return
		}
		if sf.Anonymous {
			zeroColumn(dst.Field(i), column)
		}
	}
}

// Fresh returns a newly loaded instance from the database.
func Fresh[T any](model *T) (*T, error) {
	id, err := modelID(model)
	if err != nil {
		return nil, err
	}
	return Find[T](id)
}
