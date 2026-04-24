package models

type District struct {
	BaseModel

	CountryID uint   `gorm:"index;not null" json:"country_id"`
	CityID    uint   `gorm:"index;not null" json:"city_id"`
	Name      string `gorm:"type:varchar(100);not null;index" json:"name"`

	Country *Country `gorm:"foreignKey:CountryID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"country,omitempty"`
	City    *City    `gorm:"foreignKey:CityID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"city,omitempty"`
}

func (District) TableName() string {
	return "districts"
}
