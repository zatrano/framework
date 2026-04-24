// Package eagerload, GORM sorgularında N+1 sorununu önlemek için
// merkezi Preload ve Join yönetimi sağlar.
//
// N+1 Sorunu Nedir?
//
//	// YANLIŞ: Her user için ayrı sorgu (100 user = 101 sorgu!)
//	users, _ := repo.GetAll(ctx)
//	for _, u := range users {
//	    u.Posts, _ = postRepo.GetByUserID(u.ID) // N sorgu daha!
//	}
//
//	// DOĞRU: Tek sorguda JOIN veya 2 sorguda Preload
//	db.Preload("Posts").Find(&users) // 2 sorgu: users + posts
//	db.Joins("LEFT JOIN posts ON posts.user_id = users.id").Find(&users) // 1 sorgu
//
// Bu paket hangi ilişkilerin preload edileceğini merkezi olarak yönetir.
package eagerload

import (
	"strings"

	"gorm.io/gorm"
)

// PreloadOption bir preload yapılandırmasını temsil eder.
type PreloadOption struct {
	// Association ilişki adı (GORM struct field adı, örn: "Posts", "User.Profile")
	Association string
	// Condition opsiyonel WHERE koşulu (parametreli olmalı!)
	// Örn: "is_active = ?", true
	Condition []interface{}
}

// Opt hızlı PreloadOption oluşturur (koşulsuz).
//
// Örnek:
//
//	eagerload.Opt("Posts")
//	eagerload.Opt("User")
//	eagerload.Opt("Category.Parent")
func Opt(association string) PreloadOption {
	return PreloadOption{Association: association}
}

// OptWhere koşullu PreloadOption oluşturur.
// Condition parametreli GORM sorgusu olmalı (SQL injection riski!).
//
// Örnek:
//
//	eagerload.OptWhere("Posts", "is_active = ?", true)
//	eagerload.OptWhere("Comments", "status = ?", "approved")
func OptWhere(association string, condition string, args ...interface{}) PreloadOption {
	return PreloadOption{
		Association: association,
		Condition:   append([]interface{}{condition}, args...),
	}
}

// Apply verilen GORM db'ye preload'ları uygular.
// Repository'lerin GetAll ve GetByID fonksiyonlarında kullanılır.
//
// Örnek:
//
//	db = eagerload.Apply(db, eagerload.Opt("User"), eagerload.Opt("Category"))
//	db.Find(&products)
func Apply(db *gorm.DB, opts ...PreloadOption) *gorm.DB {
	for _, opt := range opts {
		if opt.Association == "" {
			continue
		}
		if len(opt.Condition) > 0 {
			db = db.Preload(opt.Association, opt.Condition...)
		} else {
			db = db.Preload(opt.Association)
		}
	}
	return db
}

// ApplyJoins LEFT JOIN ile ilişkileri tek sorguda çeker.
// Preload'dan farkı: filtreleme/sıralama için ilişki kolonlarını kullanabilirsiniz.
// Dikkat: JOIN büyük veri setlerinde bellek kullanımını artırabilir.
//
// Örnek:
//
//	db = eagerload.ApplyJoins(db, "LEFT JOIN categories ON categories.id = products.category_id")
//	db.Where("categories.is_active = ?", true).Find(&products)
func ApplyJoins(db *gorm.DB, joins ...string) *gorm.DB {
	for _, join := range joins {
		// Güvenlik: ham join string'lerini doğrula (sadece SELECT güvenli, mutation değil)
		upper := strings.ToUpper(strings.TrimSpace(join))
		if strings.Contains(upper, "DROP") ||
			strings.Contains(upper, "DELETE") ||
			strings.Contains(upper, "UPDATE") ||
			strings.Contains(upper, "INSERT") ||
			strings.Contains(upper, "TRUNCATE") {
			// Tehlikeli JOIN ifadesi — atla ve logla
			continue
		}
		db = db.Joins(join)
	}
	return db
}

// SelectFields güvenli alan seçimi yapar.
// Tüm alanları çekmek yerine sadece ihtiyaç duyulanları seçer (performans + güvenlik).
// Kolon adları safequery.ValidateColumn ile doğrulanmalıdır.
//
// Örnek:
//
//	db = eagerload.SelectFields(db, "id", "name", "email", "created_at")
//	db.Find(&users) // password_hash, refresh_token vs. gelmez
func SelectFields(db *gorm.DB, fields ...string) *gorm.DB {
	if len(fields) == 0 {
		return db
	}
	// Alanları doğrula (harf, rakam, alt çizgi, nokta)
	safe := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if isValidIdentifier(f) {
			safe = append(safe, f)
		}
	}
	if len(safe) == 0 {
		return db
	}
	return db.Select(safe)
}

// isValidIdentifier bir SQL identifier'ının güvenli olup olmadığını kontrol eder.
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !isLetter(r) && r != '_' {
				return false
			}
		} else {
			if !isLetter(r) && !isDigit(r) && r != '_' && r != '.' {
				return false
			}
		}
	}
	return true
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// -----------------------------------------------------------------------
// Hazır Preload Setleri — sık kullanılan ilişki grupları
// -----------------------------------------------------------------------

// UserWithRelations kullanıcı sorgularında standart ilişkileri yükler.
var UserWithRelations = []PreloadOption{
	Opt("UserType"),
}

// ProductWithRelations ürün sorgularında standart ilişkileri yükler.
// Projenize göre özelleştirin.
var ProductWithRelations = []PreloadOption{
	Opt("Category"),
	Opt("Images"),
}
