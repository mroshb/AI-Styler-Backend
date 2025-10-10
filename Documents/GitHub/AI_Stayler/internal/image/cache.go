package image

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache
func NewRedisCache() *RedisCache {
	// In a real implementation, you would get the Redis client from dependency injection
	// For now, we'll create a mock implementation
	return &RedisCache{}
}

// Get retrieves a value from cache
func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	// Mock implementation - in production, use actual Redis client
	return "", fmt.Errorf("not implemented")
}

// Set stores a value in cache
func (c *RedisCache) Set(ctx context.Context, key string, value string, ttl int64) error {
	// Mock implementation - in production, use actual Redis client
	return nil
}

// Delete removes a value from cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	// Mock implementation - in production, use actual Redis client
	return nil
}

// DeletePattern removes values matching a pattern from cache
func (c *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	// Mock implementation - in production, use actual Redis client
	return nil
}

// CacheImage caches an image
func (c *RedisCache) CacheImage(ctx context.Context, imageID string, image Image) error {
	key := fmt.Sprintf("image:%s", imageID)
	data, err := json.Marshal(image)
	if err != nil {
		return fmt.Errorf("failed to marshal image: %w", err)
	}
	return c.Set(ctx, key, string(data), 3600) // Cache for 1 hour
}

// GetCachedImage retrieves a cached image
func (c *RedisCache) GetCachedImage(ctx context.Context, imageID string) (Image, error) {
	key := fmt.Sprintf("image:%s", imageID)
	data, err := c.Get(ctx, key)
	if err != nil {
		return Image{}, err
	}

	var image Image
	err = json.Unmarshal([]byte(data), &image)
	if err != nil {
		return Image{}, fmt.Errorf("failed to unmarshal image: %w", err)
	}

	return image, nil
}

// CacheSignedURL caches a signed URL
func (c *RedisCache) CacheSignedURL(ctx context.Context, imageID string, url string, ttl int64) error {
	key := fmt.Sprintf("signed_url:%s", imageID)
	return c.Set(ctx, key, url, ttl)
}

// GetCachedSignedURL retrieves a cached signed URL
func (c *RedisCache) GetCachedSignedURL(ctx context.Context, imageID string) (string, error) {
	key := fmt.Sprintf("signed_url:%s", imageID)
	return c.Get(ctx, key)
}
