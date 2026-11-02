package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string, dest any) (bool, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	GetVersion(ctx context.Context, group string) (int64, error)
	BumpVersion(ctx context.Context, group string) (int64, error)
}

func Key(group string, version int64, scope string) string {
	return fmt.Sprintf("%s:v%d:%s", group, version, scope)
}

func KeyWithHash(group string, version int64, params string) string {
	sum := sha256.Sum256([]byte(params))
	return Key(group, version, hex.EncodeToString(sum[:8]))
}

func jitter(ttl time.Duration) time.Duration {
	percent := 0.1
	jitterRange := float64(ttl) * percent
	jitterAmount := (rand.Float64()*2 - 1) * jitterRange
	return time.Duration(float64(ttl) + jitterAmount)
}
