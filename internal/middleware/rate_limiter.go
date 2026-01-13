package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimitRule struct {
	Requests  int
	Window    time.Duration
	BurstSize int
}

var DefaultRateLimits = map[string]RateLimitRule{
	"guest_api": {
		Requests:  30,
		Window:    time.Minute,
		BurstSize: 10,
	},
	"user_api": {
		Requests:  60,
		Window:    time.Minute,
		BurstSize: 20,
	},
	"premium_api": {
		Requests:  300,
		Window:    time.Minute,
		BurstSize: 50,
	},
	"upload": {
		Requests:  5,
		Window:    time.Hour,
		BurstSize: 2,
	},
	"search": {
		Requests:  30,
		Window:    time.Minute,
		BurstSize: 10,
	},
	"streaming": {
		Requests:  300,
		Window:    time.Minute,
		BurstSize: 100,
	},
	"auth": {
		Requests:  10,
		Window:    time.Minute,
		BurstSize: 5,
	},
}

type RateLimiter struct {
	redis *redis.Client
	rules map[string]RateLimitRule
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis: redisClient,
		rules: DefaultRateLimits,
	}
}

func (rl *RateLimiter) SetRule(name string, rule RateLimitRule) {
	rl.rules[name] = rule
}

func (rl *RateLimiter) Allow(ctx context.Context, key string, ruleName string) (bool, int64, time.Duration, error) {
	rule, exists := rl.rules[ruleName]
	if !exists {
		rule = rl.rules["guest_api"]
	}

	now := time.Now()
	windowStart := now.Truncate(rule.Window)
	windowKey := fmt.Sprintf("ratelimit:%s:%s:%d", ruleName, key, windowStart.Unix())

	pipe := rl.redis.Pipeline()
	incrCmd := pipe.Incr(ctx, windowKey)
	pipe.Expire(ctx, windowKey, rule.Window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return true, 0, 0, nil
	}

	count := incrCmd.Val()
	limit := int64(rule.Requests)
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	resetTime := windowStart.Add(rule.Window).Sub(now)

	if count > limit+int64(rule.BurstSize) {
		return false, remaining, resetTime, nil
	}

	return true, remaining, resetTime, nil
}

func (rl *RateLimiter) GetRetryAfter(ctx context.Context, key string, ruleName string) time.Duration {
	rule, exists := rl.rules[ruleName]
	if !exists {
		rule = rl.rules["guest_api"]
	}

	now := time.Now()
	windowStart := now.Truncate(rule.Window)
	return windowStart.Add(rule.Window).Sub(now)
}

func (rl *RateLimiter) RecordFailure(ctx context.Context, key string) error {
	failKey := fmt.Sprintf("ratelimit:fails:%s", key)

	pipe := rl.redis.Pipeline()
	pipe.Incr(ctx, failKey)
	pipe.Expire(ctx, failKey, time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func (rl *RateLimiter) GetFailureCount(ctx context.Context, key string) (int64, error) {
	failKey := fmt.Sprintf("ratelimit:fails:%s", key)
	count, err := rl.redis.Get(ctx, failKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

func (rl *RateLimiter) ClearFailures(ctx context.Context, key string) error {
	failKey := fmt.Sprintf("ratelimit:fails:%s", key)
	return rl.redis.Del(ctx, failKey).Err()
}

func RateLimitMiddleware(limiter *RateLimiter, ruleName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()

		if userID, exists := c.Get("user_id"); exists {
			key = fmt.Sprintf("%v", userID)
		}

		allowed, remaining, resetTime, err := limiter.Allow(c.Request.Context(), key, ruleName)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.rules[ruleName].Requests))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(resetTime).Unix(), 10))

		if !allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(resetTime.Seconds()), 10))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": int64(resetTime.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func AdaptiveRateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()

		fails, _ := limiter.GetFailureCount(c.Request.Context(), key)

		var ruleName string
		switch {
		case fails > 50:
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many failed requests. Please try again later.",
				"retry_after": 3600,
			})
			c.Abort()
			return
		case fails > 20:
			ruleName = "strict"
			if _, exists := limiter.rules["strict"]; !exists {
				limiter.rules["strict"] = RateLimitRule{
					Requests:  5,
					Window:    time.Minute,
					BurstSize: 1,
				}
			}
		case fails > 10:
			ruleName = "moderate"
			if _, exists := limiter.rules["moderate"]; !exists {
				limiter.rules["moderate"] = RateLimitRule{
					Requests:  15,
					Window:    time.Minute,
					BurstSize: 3,
				}
			}
		default:
			ruleName = "guest_api"
		}

		allowed, remaining, resetTime, err := limiter.Allow(c.Request.Context(), key, ruleName)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		if !allowed {
			limiter.RecordFailure(c.Request.Context(), key)
			c.Header("Retry-After", strconv.FormatInt(int64(resetTime.Seconds()), 10))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": int64(resetTime.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func IPBasedRateLimitMiddleware(limiter *RateLimiter, ruleName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()

		if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
			key = xff
		}

		allowed, remaining, resetTime, err := limiter.Allow(c.Request.Context(), key, ruleName)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		if !allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(resetTime.Seconds()), 10))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": int64(resetTime.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func ConcurrencyLimitMiddleware(limiter *RateLimiter, maxConcurrent int) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		concurrencyKey := fmt.Sprintf("concurrent:%s", key)

		ctx := c.Request.Context()
		current, err := limiter.redis.Incr(ctx, concurrencyKey).Result()
		if err != nil {
			c.Next()
			return
		}

		limiter.redis.Expire(ctx, concurrencyKey, time.Minute)

		if current > int64(maxConcurrent) {
			limiter.redis.Decr(ctx, concurrencyKey)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many concurrent requests",
			})
			c.Abort()
			return
		}

		defer limiter.redis.Decr(ctx, concurrencyKey)

		c.Next()
	}
}
