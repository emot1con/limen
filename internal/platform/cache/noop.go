package cache

import (
	"context"
	"time"
)

type noopCache struct{}

func NewNoop() Cache {
	return &noopCache{}
}

func (n *noopCache) Get(ctx context.Context, key string, dest any) (bool, error) {
	return false, nil
}

func (n *noopCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return nil
}

func (n *noopCache) Delete(ctx context.Context, keys ...string) error {
	return nil
}

func (n *noopCache) GetVersion(ctx context.Context, group string) (int64, error) {
	return 0, nil
}

func (n *noopCache) BumpVersion(ctx context.Context, group string) (int64, error) {
	return 0, nil
}
