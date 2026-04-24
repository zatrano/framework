package models

import (
	"context"
	"reflect"
	"time"

	"github.com/zatrano/framework/packages/currentuser"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type BaseModel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	CreatedBy uint  `gorm:"column:created_by;index" json:"created_by"`
	UpdatedBy uint  `gorm:"column:updated_by;index" json:"updated_by"`
	DeletedBy *uint `gorm:"column:deleted_by;index" json:"deleted_by,omitempty"`
	IsActive  bool  `gorm:"default:true;index" json:"is_active"`
}

func getCurrentUserID(ctx context.Context) uint {
	cu := currentuser.FromContext(ctx)
	if cu.ID != 0 {
		return cu.ID
	}
	return 0
}

func applyToReflectValue(ctx context.Context, rv reflect.Value, f *schema.Field, value interface{}) {
	rv = reflect.Indirect(rv)
	if rv.Kind() == reflect.Slice {
		for i := 0; i < rv.Len(); i++ {
			applyToReflectValue(ctx, rv.Index(i), f, value)
		}
		return
	}

	if rv.Kind() == reflect.Struct {
		_ = f.Set(ctx, rv, value)
	}
}

func applyIsActiveSafe(ctx context.Context, rv reflect.Value, f *schema.Field) {
	rv = reflect.Indirect(rv)
	if rv.Kind() == reflect.Slice {
		for i := 0; i < rv.Len(); i++ {
			applyIsActiveSafe(ctx, rv.Index(i), f)
		}
		return
	}

	if rv.Kind() == reflect.Struct {
		fieldVal := rv.FieldByName(f.Name)
		if fieldVal.IsValid() && fieldVal.Kind() == reflect.Bool && !fieldVal.Bool() {
			_ = f.Set(ctx, rv, true)
		}
	}
}

func RegisterBaseModelCallbacks(db *gorm.DB) {
	db.Callback().Create().Before("gorm:create").Register("base_model:before_create", func(tx *gorm.DB) {
		cuID := getCurrentUserID(tx.Statement.Context)
		if tx.Statement.Schema == nil {
			return
		}

		rv := tx.Statement.ReflectValue

		if f := tx.Statement.Schema.LookUpField("created_by"); f != nil {
			applyToReflectValue(tx.Statement.Context, rv, f, cuID)
		}
		if f := tx.Statement.Schema.LookUpField("updated_by"); f != nil {
			applyToReflectValue(tx.Statement.Context, rv, f, cuID)
		}
		if f := tx.Statement.Schema.LookUpField("is_active"); f != nil {
			applyIsActiveSafe(tx.Statement.Context, rv, f)
		}
	})

	db.Callback().Update().Before("gorm:update").Register("base_model:before_update", func(tx *gorm.DB) {
		cuID := getCurrentUserID(tx.Statement.Context)
		if tx.Statement.Schema == nil {
			return
		}

		rv := tx.Statement.ReflectValue
		if f := tx.Statement.Schema.LookUpField("updated_by"); f != nil {
			applyToReflectValue(tx.Statement.Context, rv, f, cuID)
		}
	})

	db.Callback().Delete().Before("gorm:delete").Register("base_model:before_delete", func(tx *gorm.DB) {
		cuID := getCurrentUserID(tx.Statement.Context)
		if tx.Statement.Schema == nil {
			return
		}

		rv := tx.Statement.ReflectValue
		if f := tx.Statement.Schema.LookUpField("deleted_by"); f != nil {
			applyToReflectValue(tx.Statement.Context, rv, f, cuID)
		}
		if f := tx.Statement.Schema.LookUpField("updated_by"); f != nil {
			applyToReflectValue(tx.Statement.Context, rv, f, cuID)
		}
	})
}
