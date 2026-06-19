package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const allowIPScript = `
local count = redis.call("INCR", KEYS[1])
if count == 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return count
`

type RateLimitRepository struct {
	client *goredis.Client
	limit  int
	window time.Duration
	script *goredis.Script
}

func NewRateLimitRepository(redisURL string, maxIPRequestPerMinute int) (*RateLimitRepository, error) {
	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := goredis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &RateLimitRepository{
		client: client,
		limit:  maxIPRequestPerMinute,
		window: time.Minute,
		script: goredis.NewScript(allowIPScript),
	}, nil
}

func (r *RateLimitRepository) Close() error {
	if r.client == nil {
		return nil
	}
	return r.client.Close()
}

func (r *RateLimitRepository) AllowIP(ctx context.Context, ip string) (bool, error) {
	key := "ratelimit:ip:" + ip

	count, err := r.script.Run(ctx, r.client, []string{key}, int(r.window.Seconds())).Int64()
	if err != nil {
		return false, fmt.Errorf("check ip rate limit: %w", err)
	}

	return count <= int64(r.limit), nil
}
