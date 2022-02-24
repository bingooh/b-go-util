package rdb

import (
	"context"
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/util"
	"github.com/go-redis/redis/v8"
	"time"
)

const (
	TOO_OFTEN_PREFIX    = `to_`
	TOO_FREQUENT_PREFIX = `tf_`
)

//简单访问频率限制
type RateLimiter struct {
	tooOftenPrefix string
	tooFreqPrefix  string
	client         *redis.Client
}

func MustNewRateLimiter(prefix string, redisClient *redis.Client) *RateLimiter {
	util.AssertOk(!_string.Empty(prefix), `prefix is empty`)

	return &RateLimiter{
		TOO_OFTEN_PREFIX + prefix,
		TOO_FREQUENT_PREFIX + prefix,
		redisClient,
	}
}

/*如果key存在则返回true，否则设置超时时间*/
func (r *RateLimiter) IsTooOften(ctx context.Context, key string, expire time.Duration) bool {
	return !r.client.SetNX(ctx, r.TooOftenKey(key), nil, expire).Val()
}

/*如果key对应的value>=limit则返回true，否则+1，如果value为0则设置超时时间*/
func (r *RateLimiter) IsTooFrequent(ctx context.Context, key string, limit int, expire time.Duration) bool {
	if limit <= 0 {
		return true
	}

	key = r.TooFreqKey(key)
	if r.client.SetNX(ctx, key, 1, expire).Val() {
		return false
	}

	v, err := r.client.Incr(ctx, key).Result()

	return err != nil && err != redis.Nil || v > int64(limit)
}

func (r *RateLimiter) DelTooOften(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.tooOftenPrefix+key).Err()
}

func (r *RateLimiter) DelTooFrequent(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.tooFreqPrefix+key).Err()
}

func (r *RateLimiter) TooOftenKey(key string) string {
	return r.tooOftenPrefix + key
}

func (r *RateLimiter) TooFreqKey(key string) string {
	return r.tooFreqPrefix + key
}
