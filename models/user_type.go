package models

type UserType struct {
	BaseModel `json:"-"`

	Name string `gorm:"size:50;unique;not null;index" json:"name"`
}

func (UserType) TableName() string {
	return "user_types"
}
