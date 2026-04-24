package models

type Address struct {
	BaseModel

	CountryID  uint   `gorm:"index" json:"country_id"`
	CityID     uint   `gorm:"index" json:"city_id"`
	DistrictID uint   `gorm:"index" json:"district_id"`
	Detail     string `gorm:"type:varchar(255)" json:"detail"`

	Country  *Country  `gorm:"foreignKey:CountryID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"country,omitempty"`
	City     *City     `gorm:"foreignKey:CityID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"city,omitempty"`
	District *District `gorm:"foreignKey:DistrictID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"district,omitempty"`
}

func (Address) TableName() string {
	return "addresses"
}
