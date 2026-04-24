package services_test

// Not: Cache testleri gerçek Redis bağlantısı gerektirir.
// CI/CD'de redis:alpine servisi ile çalıştırılmalıdır.
// Bu dosya derleme kontrolü için yapı testini içerir.

import (
	"testing"

	"github.com/zatrano/framework/packages/cache"
)

func TestCacheTTLConstants(t *testing.T) {
	if cache.TTL1Min <= 0 {
		t.Error("TTL1Min sıfırdan büyük olmalı")
	}
	if cache.TTL1Day <= cache.TTL1Hour {
		t.Error("TTL1Day, TTL1Hour'dan büyük olmalı")
	}
	if cache.TTL1Hour <= cache.TTL15Min {
		t.Error("TTL1Hour, TTL15Min'den büyük olmalı")
	}
}
