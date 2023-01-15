package cache

import "context"

type CacheInterface interface {
	Get(ctx context.Context, key string) (string, error)
}

type Cache struct{}

func New() *Cache {
	return &Cache{}
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (c *Cache) Set(ctx context.Context, key string, value string) error {
	return nil
}
