package models

type Definition struct {
	BaseModel

	Key         string `gorm:"type:varchar(120);not null;uniqueIndex" json:"key"`
	Value       string `gorm:"type:text;not null;default:''" json:"value"`
	Description string `gorm:"type:varchar(255)" json:"description"`
}

func (Definition) TableName() string {
	return "definitions"
}
