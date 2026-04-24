package models

type Country struct {
	BaseModel

	Name string `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`

	Cities []City `gorm:"foreignKey:CountryID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"cities,omitempty"`
}

func (Country) TableName() string {
	return "countries"
}
