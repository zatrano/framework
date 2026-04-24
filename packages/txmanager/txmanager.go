// Package txmanager — servis katmanında güvenli DB transaction yönetimi.
// Repository'lerin doğrudan transaction başlatması yerine bu paket kullanılır.
// Böylece servis katmanı birden fazla repository operasyonunu atomik yapar.
package txmanager

import (
	"context"
	"fmt"

	"github.com/zatrano/framework/configs/databaseconfig"

	"gorm.io/gorm"
)

type contextKey string

const txKey contextKey = "db_tx"

// WithTransaction — verilen fonksiyonu bir DB transaction içinde çalıştırır.
// fn başarılı dönerse COMMIT, hata dönerse ROLLBACK yapılır.
// context.Context üzerinden transaction iletilir; tüm repository'ler
// FromContext() ile mevcut transaction'a katılabilir.
func WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	db := databaseconfig.GetDB()
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("transaction başlatılamadı: %w", tx.Error)
	}

	txCtx := context.WithValue(ctx, txKey, tx)

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback().Error; rbErr != nil {
			return fmt.Errorf("rollback hatası: %v (orijinal hata: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit hatası: %w", err)
	}
	return nil
}

// FromContext — context'ten aktif transaction'ı alır.
// Transaction yoksa normal DB bağlantısı döner.
// Repository'ler her zaman bu fonksiyonu kullanmalıdır.
func FromContext(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return databaseconfig.GetDB().WithContext(ctx)
}

// InTransaction — context'in içinde aktif transaction olup olmadığını kontrol eder.
func InTransaction(ctx context.Context) bool {
	_, ok := ctx.Value(txKey).(*gorm.DB)
	return ok
}
