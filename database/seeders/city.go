package seeders

import (
	"time"

	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/models"

	"gorm.io/gorm"
)

func SeedCities(db *gorm.DB) error {
	logconfig.SLog.Info("Şehirler yükleniyor...")

	cityNames := []string{
		"Adana", "Adıyaman", "Afyonkarahisar", "Ağrı", "Amasya", "Ankara", "Antalya", "Artvin",
		"Aydın", "Balıkesir", "Bilecik", "Bingöl", "Bitlis", "Bolu", "Burdur", "Bursa",
		"Çanakkale", "Çankırı", "Çorum", "Denizli", "Diyarbakır", "Edirne", "Elâzığ", "Erzincan",
		"Erzurum", "Eskişehir", "Gaziantep", "Giresun", "Gümüşhane", "Hakkâri", "Hatay", "Isparta",
		"Mersin", "İstanbul", "İzmir", "Kars", "Kastamonu", "Kayseri", "Kırklareli", "Kırşehir",
		"Kocaeli", "Konya", "Kütahya", "Malatya", "Manisa", "Kahramanmaraş", "Mardin", "Muğla",
		"Muş", "Nevşehir", "Niğde", "Ordu", "Rize", "Sakarya", "Samsun", "Siirt", "Sinop", "Sivas",
		"Tekirdağ", "Tokat", "Trabzon", "Tunceli", "Şanlıurfa", "Uşak", "Van", "Yozgat", "Zonguldak",
		"Aksaray", "Bayburt", "Karaman", "Kırıkkale", "Batman", "Şırnak", "Bartın", "Ardahan",
		"Iğdır", "Yalova", "Karabük", "Kilis", "Osmaniye", "Düzce",
	}

	for _, name := range cityNames {
		city := models.City{
			Name:      name,
			CountryID: 1,
			BaseModel: models.BaseModel{
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				CreatedBy: 1,
				UpdatedBy: 1,
			},
		}

		if err := db.Create(&city).Error; err != nil {
			logconfig.SLog.Error("Şehir eklenirken hata: "+city.Name, err)
			return err
		}

	}

	logconfig.SLog.Info("Tüm şehirler başarıyla yüklendi.")
	return nil
}
