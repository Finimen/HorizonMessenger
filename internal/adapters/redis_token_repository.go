package adapters

import (
	"context"
	"time"

	"github.com/go-redis/redis"
)

type RedisTokenRepository struct {
	client *redis.Client
}

func NewRedisTokenRepository(client *redis.Client) *RedisTokenRepository {
	return &RedisTokenRepository{client: client}
}

func (r *RedisTokenRepository) IsRevoked(ctx context.Context, tokenHash string) (bool, error) {
	exists, err := r.client.Exists("blacklist:" + tokenHash).Result()
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

func (r *RedisTokenRepository) Revoke(ctx context.Context, tokenHash string, expiration time.Duration) error {
	return r.client.Set("blacklist:"+tokenHash, "1", expiration).Err()
}
