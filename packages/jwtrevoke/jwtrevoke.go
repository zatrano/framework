package jwtrevoke

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/zatrano/framework/configs/redisconfig"
)

const revokeKeyPrefix = "jwt:revoked"

func tokenKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return redisconfig.GetPrefixedKey(revokeKeyPrefix, hex.EncodeToString(sum[:]))
}

// RevokeToken token'ı belirtilen süre boyunca blacklist'te tutar.
func RevokeToken(ctx context.Context, token string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = time.Minute
	}
	return redisconfig.GetClient().Set(ctx, tokenKey(token), "1", ttl).Err()
}

// IsRevoked token blacklist'te mi kontrol eder.
func IsRevoked(ctx context.Context, token string) (bool, error) {
	exists, err := redisconfig.GetClient().Exists(ctx, tokenKey(token)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}
