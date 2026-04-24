// Package safequery, GORM sorgularında SQL Injection'a karşı merkezi koruma sağlar.
// Tüm dinamik kolon adları ve sıralama değerleri bu paket üzerinden geçirilmelidir.
// Asla ham string interpolasyonu kullanmayın: db.Where("name = " + name) → GÜVENSİZ
// Her zaman parametreli sorgu kullanın:            db.Where("name = ?", name) → GÜVENLİ
package safequery

import (
	"fmt"
	"regexp"
	"strings"
)

// allowedColumnPattern yalnızca geçerli kolon adı karakterlerine izin verir.
var allowedColumnPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)

// AllowedDirections geçerli sıralama yönleri.
var AllowedDirections = map[string]bool{
	"asc":  true,
	"desc": true,
}

// ErrInvalidColumn geçersiz kolon adı hatası.
type ErrInvalidColumn struct {
	Column string
}

func (e ErrInvalidColumn) Error() string {
	return fmt.Sprintf("safequery: geçersiz kolon adı: %q", e.Column)
}

// ErrInvalidDirection geçersiz sıralama yönü hatası.
type ErrInvalidDirection struct {
	Direction string
}

func (e ErrInvalidDirection) Error() string {
	return fmt.Sprintf("safequery: geçersiz sıralama yönü: %q (asc veya desc olmalı)", e.Direction)
}

// ValidateColumn bir kolon adının güvenli olup olmadığını doğrular.
// Yalnızca harf, rakam, alt çizgi ve nokta karakterlerine izin verir.
// SQL injection saldırılarını önlemek için tüm dinamik kolon adları bu fonksiyondan geçirilmelidir.
//
// Örnek:
//
//	col, err := safequery.ValidateColumn("user_name")      // OK
//	col, err := safequery.ValidateColumn("users.email")    // OK
//	col, err := safequery.ValidateColumn("'; DROP TABLE")  // HATA
func ValidateColumn(col string) (string, error) {
	col = strings.TrimSpace(col)
	if col == "" {
		return "", ErrInvalidColumn{Column: col}
	}
	if !allowedColumnPattern.MatchString(col) {
		return "", ErrInvalidColumn{Column: col}
	}
	return col, nil
}

// ValidateDirection sıralama yönünün güvenli olup olmadığını doğrular.
// Yalnızca "asc" ve "desc" kabul edilir.
//
// Örnek:
//
//	dir, err := safequery.ValidateDirection("asc")   // OK → "ASC"
//	dir, err := safequery.ValidateDirection("DESC")  // OK → "DESC"
//	dir, err := safequery.ValidateDirection("drop")  // HATA
func ValidateDirection(dir string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(dir))
	if !AllowedDirections[d] {
		return "", ErrInvalidDirection{Direction: dir}
	}
	return strings.ToUpper(d), nil
}

// OrderClause güvenli bir ORDER BY ifadesi oluşturur.
// col ve dir parametreleri doğrulanır; geçersizse varsayılan değerler kullanılır.
//
// Örnek:
//
//	clause := safequery.OrderClause("created_at", "desc", "id", "ASC")
//	// → "created_at DESC"
func OrderClause(col, dir, defaultCol, defaultDir string) string {
	safeCol, err := ValidateColumn(col)
	if err != nil {
		safeCol = defaultCol
	}
	safeDir, err := ValidateDirection(dir)
	if err != nil {
		safeDir = strings.ToUpper(defaultDir)
	}
	return safeCol + " " + safeDir
}

// AllowedColumns belirli bir izin listesine göre kolon adını doğrular.
// Repository'lerin SetAllowedSortColumns listesiyle birlikte kullanılır.
//
// Örnek:
//
//	allowed := []string{"id", "name", "created_at"}
//	col, err := safequery.AllowedColumns("name", allowed)    // OK
//	col, err := safequery.AllowedColumns("password", allowed) // HATA
func AllowedColumns(col string, allowed []string) (string, error) {
	col = strings.TrimSpace(strings.ToLower(col))
	for _, a := range allowed {
		if strings.ToLower(a) == col {
			return a, nil
		}
	}
	return "", ErrInvalidColumn{Column: col}
}

// SearchValue LIKE sorgularında kullanılan arama değerini temizler.
// % ve _ karakterlerini escape eder; boş string döndürmez.
//
// Örnek:
//
//	val := safequery.SearchValue("hello%world")
//	// GORM'da: db.Where("name LIKE ?", "%" + val + "%")
func SearchValue(val string) string {
	// % ve _ karakterleri LIKE'da wildcard olarak çalışır, escape et
	val = strings.ReplaceAll(val, `\`, `\\`)
	val = strings.ReplaceAll(val, `%`, `\%`)
	val = strings.ReplaceAll(val, `_`, `\_`)
	return strings.TrimSpace(val)
}

// Paginate güvenli sayfa ve limit değerleri döner.
// Negatif, sıfır veya aşırı büyük değerlere karşı koruma sağlar.
//
// Örnek:
//
//	page, limit := safequery.Paginate(-1, 999999, 20, 100)
//	// → page=1, limit=100
func Paginate(page, limit, defaultLimit, maxLimit int) (safePage, safeLimit int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}
