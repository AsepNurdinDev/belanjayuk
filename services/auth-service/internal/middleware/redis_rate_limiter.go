package middleware

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter — implementasi RateLimiter interface menggunakan Redis
// Ini adapter yang menghubungkan middleware.RateLimiter dengan redis.Client
type RedisRateLimiter struct {
	rdb *redis.Client
}

func NewRedisRateLimiter(rdb *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{rdb: rdb}
}

// IncrRateLimit — atomic increment counter dengan INCR + EXPIRE pattern
// Mengembalikan count saat ini, waktu reset, dan error
func (r *RedisRateLimiter) IncrRateLimit(ctx context.Context, key string, window time.Duration) (int64, time.Time, error) {
	pipe := r.rdb.Pipeline()
	incrCmd := pipe.Incr(ctx, "ratelimit:"+key)
	pipe.Expire(ctx, "ratelimit:"+key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("rate limit incr: %w", err)
	}

	count := incrCmd.Val()
	resetAt := time.Now().Add(window)
	return count, resetAt, nil
}

// GetRateLimit — ambil counter saat ini tanpa increment
func (r *RedisRateLimiter) GetRateLimit(ctx context.Context, key string) (int64, error) {
	val, err := r.rdb.Get(ctx, "ratelimit:"+key).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, fmt.Errorf("get rate limit: %w", err)
	}
	return val, nil
}

// ResetRateLimit — hapus counter (dipanggil setelah login berhasil)
func (r *RedisRateLimiter) ResetRateLimit(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, "ratelimit:"+key).Err()
}
