// Package sanitizer, kullanıcı girdilerini XSS saldırılarına karşı merkezi olarak temizler.
// Tüm kullanıcıdan gelen string veriler template'e render edilmeden önce bu paketten geçirilmelidir.
//
// ZATRANO'da iki XSS senaryosu vardır:
//  1. HTML template render → html/template otomatik escape eder (güvenli)
//  2. JSON API response  → ham string dönülürse tehlikeli olabilir
//  3. Veritabanına yazılmadan önce → temizlenmiş veri saklanmalı
//
// Bu paket 3. senaryo için merkezi temizleme sağlar.
package sanitizer

import (
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Config sanitizer davranışını özelleştirir.
type Config struct {
	// MaxLength maksimum karakter uzunluğu (0 = sınırsız).
	MaxLength int
	// StripHTML tüm HTML tag'lerini kaldırır.
	StripHTML bool
	// TrimSpace baştaki ve sondaki boşlukları kaldırır.
	TrimSpace bool
}

// DefaultConfig standart konfigürasyon: HTML strip + trim + 10000 karakter.
var DefaultConfig = Config{
	MaxLength: 10000,
	StripHTML: true,
	TrimSpace: true,
}

// NameConfig isim alanları için konfigürasyon: 200 karakter, HTML yok.
var NameConfig = Config{
	MaxLength: 200,
	StripHTML: true,
	TrimSpace: true,
}

// HTMLContentConfig zengin metin alanları için: HTML tag'leri korunur ama tehlikeliler temizlenir.
var HTMLContentConfig = Config{
	MaxLength: 100000,
	StripHTML: false,
	TrimSpace: true,
}

// htmlTagPattern tüm HTML tag'lerini yakalar.
var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// dangerousTagPattern tehlikeli HTML tag'lerini yakalar.
var dangerousTagPattern = regexp.MustCompile(
	`(?i)<\s*(script|iframe|object|embed|form|input|button|link|meta|style|base|applet|svg|math)[^>]*>.*?</\s*\1\s*>|` +
		`(?i)<\s*(script|iframe|object|embed|form|input|button|link|meta|style|base|applet|svg|math)[^>]*/?>`,
)

// eventHandlerPattern JS event handler attribute'larını yakalar.
var eventHandlerPattern = regexp.MustCompile(`(?i)\s*on\w+\s*=\s*["'][^"']*["']|\s*on\w+\s*=\s*[^\s>]+`)

// javascriptProtocolPattern javascript: protokolünü yakalar.
var javascriptProtocolPattern = regexp.MustCompile(`(?i)javascript\s*:`)

// dataProtocolPattern data: protokolünü yakalar (XSS vektörü olabilir).
var dataProtocolPattern = regexp.MustCompile(`(?i)data\s*:[^,]*base64`)

// Text kullanıcı girdisini verilen konfigürasyona göre temizler.
// Temel kullanım: her form alanı bu fonksiyondan geçirilmeli.
//
// Örnek:
//
//	clean := sanitizer.Text(c.FormValue("name"), sanitizer.NameConfig)
func Text(input string, cfg Config) string {
	if cfg.TrimSpace {
		input = strings.TrimSpace(input)
	}

	if cfg.StripHTML {
		// Tüm HTML tag'lerini kaldır
		input = htmlTagPattern.ReplaceAllString(input, "")
		// HTML entity'lerini decode et sonra tekrar encode et (double encoding önlemi)
		input = html.UnescapeString(input)
		input = html.EscapeString(input)
	} else {
		// Sadece tehlikeli tag'leri ve attribute'ları kaldır
		input = dangerousTagPattern.ReplaceAllString(input, "")
		input = eventHandlerPattern.ReplaceAllString(input, "")
		input = javascriptProtocolPattern.ReplaceAllString(input, "blocked:")
		input = dataProtocolPattern.ReplaceAllString(input, "blocked:")
	}

	// Maksimum uzunluk kontrolü (unicode-aware)
	if cfg.MaxLength > 0 && utf8.RuneCountInString(input) > cfg.MaxLength {
		runes := []rune(input)
		input = string(runes[:cfg.MaxLength])
	}

	return input
}

// Name isim alanları için kısayol (200 karakter, HTML yok).
//
// Örnek:
//
//	clean := sanitizer.Name(req.Name)
func Name(input string) string {
	return Text(input, NameConfig)
}

// Plain genel metin alanları için kısayol (10000 karakter, HTML yok).
//
// Örnek:
//
//	clean := sanitizer.Plain(req.Description)
func Plain(input string) string {
	return Text(input, DefaultConfig)
}

// Email e-posta adresi temizleme ve temel format doğrulama.
// Tehlikeli karakterleri kaldırır, lowercase'e çevirir.
//
// Örnek:
//
//	clean := sanitizer.Email(req.Email)
func Email(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	// HTML temizle
	input = htmlTagPattern.ReplaceAllString(input, "")
	// 254 karakter RFC 5321 limiti
	if utf8.RuneCountInString(input) > 254 {
		runes := []rune(input)
		input = string(runes[:254])
	}
	return input
}

// Slug URL slug değerlerini temizler (sadece harf, rakam, tire).
//
// Örnek:
//
//	clean := sanitizer.Slug(req.Slug)
func Slug(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	// Yalnızca güvenli karakterler bırak
	var b strings.Builder
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// StripAllHTML tüm HTML etiketlerini kaldırır, entity'leri düz metne çevirir.
// Zengin metin editörü çıktısını düz metne dönüştürmek için kullanılır.
func StripAllHTML(input string) string {
	input = htmlTagPattern.ReplaceAllString(input, " ")
	input = html.UnescapeString(input)
	// Çift boşlukları temizle
	spacePattern := regexp.MustCompile(`\s+`)
	input = spacePattern.ReplaceAllString(input, " ")
	return strings.TrimSpace(input)
}

// Map bir map içindeki tüm string değerleri temizler.
// Fiber form değerlerini toplu temizlemek için kullanılır.
//
// Örnek:
//
//	clean := sanitizer.Map(c.Request().PostForm(), sanitizer.DefaultConfig)
func Map(data map[string][]string, cfg Config) map[string]string {
	result := make(map[string]string, len(data))
	for key, values := range data {
		if len(values) > 0 {
			result[key] = Text(values[0], cfg)
		}
	}
	return result
}
