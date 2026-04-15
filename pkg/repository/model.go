package repository

import (
	"time"

	"gorm.io/gorm"
)

// Model is the standard base model for all ZATRANO entities.
// Embed this in every model to get ID, timestamps, and soft-delete support.
//
//	type User struct {
//	    repository.Model
//	    Name  string
//	    Email string
//	}
type Model struct {
	ID        uint           `gorm:"primarykey"                  json:"id"`
	CreatedAt time.Time      `                                   json:"created_at"`
	UpdatedAt time.Time      `                                   json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"       json:"-"        swaggerignore:"true"`
}

// IsDeleted returns true if the record has been soft-deleted.
func (m *Model) IsDeleted() bool {
	return m.DeletedAt.Valid
}
