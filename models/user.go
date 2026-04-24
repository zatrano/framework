package models

import "time"

type User struct {
	BaseModel

	Name              string `gorm:"size:100;not null;index" json:"name"`
	Email             string `gorm:"size:100;unique;not null" json:"email"`
	Password          string `gorm:"size:255;not null" json:"-"`
	UserTypeID        uint   `gorm:"index" json:"user_type_id,omitempty"`
	ResetToken            string     `gorm:"size:255;index" json:"-"`
	ResetTokenExpiresAt   *time.Time `gorm:"index" json:"-"`
	EmailVerified         bool       `gorm:"default:false;index" json:"email_verified"`
	VerificationToken     string     `gorm:"size:255;index" json:"-"`
	VerificationExpiresAt *time.Time `gorm:"index" json:"-"`
	Provider          string `gorm:"size:50;index" json:"provider,omitempty"`
	ProviderID        string `gorm:"size:100;index" json:"provider_id,omitempty"`

	UserType UserType `gorm:"foreignKey:UserTypeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"user_type,omitempty"`
}

func (User) TableName() string {
	return "users"
}
