package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
)

// RuleGuest is the fallback rule applied when a caller asks for a rule that was
// never registered.
const RuleGuest = "guest_api"

// RateLimitRule is a fixed-window allowance: Requests per Window, plus a burst
// allowance on top of it.
type RateLimitRule struct {
	Requests  int
	Window    time.Duration
	BurstSize int
}

// Limit is the effective ceiling for the rule, burst included.
func (r RateLimitRule) Limit() int64 {
	return int64(r.Requests + r.BurstSize)
}

// defaultRateLimits is the built-in rule set. It is unexported and copied into
// each RateLimiter: it used to be an exported map assigned directly into the
// limiter, so any SetRule call mutated the defaults for every limiter in the
// process.
var defaultRateLimits = map[string]RateLimitRule{
	RuleGuest:     {Requests: 30, Window: time.Minute, BurstSize: 10},
	"user_api":    {Requests: 60, Window: time.Minute, BurstSize: 20},
	"premium_api": {Requests: 300, Window: time.Minute, BurstSize: 50},
	"upload":      {Requests: 5, Window: time.Hour, BurstSize: 2},
	"search":      {Requests: 30, Window: time.Minute, BurstSize: 10},
	"streaming":   {Requests: 300, Window: time.Minute, BurstSize: 100},
	"auth":        {Requests: 10, Window: time.Minute, BurstSize: 5},
}

// RateLimiter enforces fixed-window request limits backed by Redis.
//
// The rule set is guarded by a mutex. The previous implementation registered new
// rules lazily from inside request handlers, writing to a plain map from many
// goroutines at once — a concurrent map write, which panics the process rather
// than merely racing.
type RateLimiter struct {
	redis *redis.Client

	mu    sync.RWMutex
	rules map[string]RateLimitRule
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	rules := make(map[string]RateLimitRule, len(defaultRateLimits))
	for name, rule := range defaultRateLimits {
		rules[name] = rule
	}
	return &RateLimiter{redis: redisClient, rules: rules}
}

// SetRule registers or replaces a rule.
func (rl *RateLimiter) SetRule(name string, rule RateLimitRule) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.rules[name] = rule
}

// Rule returns the named rule, falling back to RuleGuest when it is unknown.
func (rl *RateLimiter) Rule(name string) RateLimitRule {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if rule, ok := rl.rules[name]; ok {
		return rule
	}
	return rl.rules[RuleGuest]
}

// Decision is the outcome of a limit check.
type Decision struct {
	Allowed   bool
	Limit     int64
	Remaining int64
	ResetIn   time.Duration
}

// Allow records one request against key and reports whether it is permitted.
//
// A Redis failure is returned to the caller rather than swallowed. The previous
// implementation returned (allowed=true, err=nil) on every Redis error, which
// silently disabled rate limiting and made the error checks at all four call
// sites unreachable.
func (rl *RateLimiter) Allow(ctx context.Context, key, ruleName string) (Decision, error) {
	rule := rl.Rule(ruleName)

	now := time.Now()
	windowStart := now.Truncate(rule.Window)
	windowKey := fmt.Sprintf("ratelimit:%s:%s:%d", ruleName, key, windowStart.Unix())

	pipe := rl.redis.Pipeline()
	incr := pipe.Incr(ctx, windowKey)
	// Expiry is set on every request rather than only on creation. Setting it
	// only when the counter is new would leave the key immortal whenever the
	// EXPIRE leg of the pipeline failed.
	pipe.Expire(ctx, windowKey, rule.Window)

	if _, err := pipe.Exec(ctx); err != nil {
		return Decision{}, fmt.Errorf("rate limit check: %w", err)
	}

	count := incr.Val()
	limit := rule.Limit()

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	return Decision{
		Allowed:   count <= limit,
		Limit:     limit,
		Remaining: remaining,
		ResetIn:   windowStart.Add(rule.Window).Sub(now),
	}, nil
}

// RateLimitMiddleware enforces ruleName, keyed by authenticated user when there
// is one and by client IP otherwise.
//
// On a Redis outage the request is allowed through and the failure is logged.
// That is a deliberate choice — availability over enforcement — but it is now an
// explicit, visible one rather than an accident.
func RateLimitMiddleware(limiter *RateLimiter, ruleName string) gin.HandlerFunc {
	return rateLimitWith(limiter, ruleName, nil)
}

// RateLimitWithLogger is RateLimitMiddleware that reports Redis failures.
func RateLimitWithLogger(limiter *RateLimiter, ruleName string, log *logger.Logger) gin.HandlerFunc {
	return rateLimitWith(limiter, ruleName, log)
}

func rateLimitWith(limiter *RateLimiter, ruleName string, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		key := rateLimitKey(c)

		decision, err := limiter.Allow(ctx, key, ruleName)
		if err != nil {
			if log != nil {
				log.Error(ctx, "rate limiter unavailable; allowing request", err, map[string]interface{}{
					"rule": ruleName,
				})
			}
			c.Next()
			return
		}

		header := c.Writer.Header()
		header.Set("X-RateLimit-Limit", strconv.FormatInt(decision.Limit, 10))
		header.Set("X-RateLimit-Remaining", strconv.FormatInt(decision.Remaining, 10))
		header.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(decision.ResetIn).Unix(), 10))

		if !decision.Allowed {
			retryAfter := int64(decision.ResetIn.Seconds()) + 1
			header.Set("Retry-After", strconv.FormatInt(retryAfter, 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "Rate limit exceeded",
				},
				"retry_after": retryAfter,
			})
			return
		}

		c.Next()
	}
}

// rateLimitKey identifies the caller to limit.
//
// An authenticated user is limited by user ID, so rotating IPs does not reset
// their budget. Everyone else is limited by client IP as gin resolves it, which
// honours the configured trusted-proxy list. The old implementation keyed off a
// raw X-Forwarded-For header, which any client can set to an arbitrary value —
// making the limit trivially bypassable by varying the header per request.
func rateLimitKey(c *gin.Context) string {
	if principal, ok := appctx.PrincipalFrom(c.Request.Context()); ok {
		return "user:" + principal.UserID.String()
	}
	return "ip:" + c.ClientIP()
}

// ConcurrencyLimitMiddleware caps how many requests one caller may have in
// flight at once.
func ConcurrencyLimitMiddleware(limiter *RateLimiter, maxConcurrent int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		key := "concurrent:" + rateLimitKey(c)

		current, err := limiter.redis.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}
		// Bound the counter's lifetime so a crashed request cannot hold a slot
		// forever.
		limiter.redis.Expire(ctx, key, time.Minute)

		// The release must not run on the request context: by the time the
		// handler returns, that context is cancelled, so the DECR would fail
		// and the caller's in-flight count would ratchet up permanently until
		// the key expired.
		defer func() {
			releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			defer cancel()
			limiter.redis.Decr(releaseCtx, key)
		}()

		if current > int64(maxConcurrent) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TOO_MANY_CONCURRENT",
					"message": "Too many concurrent requests",
				},
			})
			return
		}

		c.Next()
	}
}
