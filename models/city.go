package models

type City struct {
	BaseModel

	CountryID uint   `gorm:"index;not null" json:"country_id"`
	Name      string `gorm:"type:varchar(100);not null;index" json:"name"`

	Country   *Country   `gorm:"foreignKey:CountryID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"country,omitempty"`
	Districts []District `gorm:"foreignKey:CityID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"districts,omitempty"`
}

func (City) TableName() string {
	return "cities"
}
