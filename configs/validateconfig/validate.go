package validateconfig

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Rule — tek bir env değişkeni kuralı.
type Rule struct {
	Key      string
	Required bool
	MinInt   *int
	MaxInt   *int
	OneOf    []string
}

// Validator — startup'ta tüm env değişkenlerini doğrular.
// Eksik veya geçersiz değerler varsa uygulama başlamaz.
type Validator struct {
	rules  []Rule
	errors []string
}

func New() *Validator { return &Validator{} }

func (v *Validator) Require(key string) *Validator {
	v.rules = append(v.rules, Rule{Key: key, Required: true})
	return v
}

func (v *Validator) RequireOneOf(key string, values ...string) *Validator {
	v.rules = append(v.rules, Rule{Key: key, Required: true, OneOf: values})
	return v
}

func (v *Validator) RequirePort(key string) *Validator {
	min, max := 1, 65535
	v.rules = append(v.rules, Rule{Key: key, Required: true, MinInt: &min, MaxInt: &max})
	return v
}

func (v *Validator) Optional(key string, oneOf ...string) *Validator {
	v.rules = append(v.rules, Rule{Key: key, Required: false, OneOf: oneOf})
	return v
}

// Validate — tüm kuralları kontrol eder. Hata varsa birleşik hata döner.
func (v *Validator) Validate() error {
	v.errors = nil
	for _, r := range v.rules {
		val := strings.TrimSpace(os.Getenv(r.Key))

		if r.Required && val == "" {
			v.errors = append(v.errors, fmt.Sprintf("  [EKSIK] %s zorunlu ama tanımlı değil", r.Key))
			continue
		}
		if val == "" {
			continue
		}

		if len(r.OneOf) > 0 {
			found := false
			for _, allowed := range r.OneOf {
				if strings.EqualFold(val, allowed) {
					found = true
					break
				}
			}
			if !found {
				v.errors = append(v.errors, fmt.Sprintf(
					"  [GEÇERSİZ] %s=%q; izin verilen değerler: %s",
					r.Key, val, strings.Join(r.OneOf, ", ")))
			}
		}

		if r.MinInt != nil || r.MaxInt != nil {
			n, err := strconv.Atoi(val)
			if err != nil {
				v.errors = append(v.errors, fmt.Sprintf("  [SAYI DEĞİL] %s=%q sayısal olmalı", r.Key, val))
				continue
			}
			if r.MinInt != nil && n < *r.MinInt {
				v.errors = append(v.errors, fmt.Sprintf("  [ARALIK] %s=%d; min %d olmalı", r.Key, n, *r.MinInt))
			}
			if r.MaxInt != nil && n > *r.MaxInt {
				v.errors = append(v.errors, fmt.Sprintf("  [ARALIK] %s=%d; max %d olmalı", r.Key, n, *r.MaxInt))
			}
		}
	}

	if len(v.errors) > 0 {
		return errors.New("Ortam değişkeni doğrulama hatası:\n" + strings.Join(v.errors, "\n"))
	}
	return nil
}

// ValidateAll — production için kullanılan tam kural seti.
// Uygulama başlamadan önce main() içinde çağrılır.
func ValidateAll() error {
	return New().
		Require("APP_ENV").
		Require("APP_HOST").
		RequirePort("APP_PORT").
		Require("APP_BASE_URL").
		Require("DB_HOST").
		RequirePort("DB_PORT").
		Require("DB_USERNAME").
		Require("DB_DATABASE").
		RequireOneOf("DB_SSL_MODE", "disable", "require", "verify-ca", "verify-full").
		Require("REDIS_HOST").
		RequirePort("REDIS_PORT").
		Require("SESSION_COOKIE_NAME").
		RequireOneOf("APP_ENV", "development", "staging", "production").
		Validate()
}

// ValidateProduction — production'a özgü ek kurallar.
func ValidateProduction() error {
	return New().
		Require("APP_BASE_URL").
		Require("COOKIE_DOMAIN").
		Require("DB_PASSWORD").
		Validate()
}
