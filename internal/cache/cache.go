package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrCacheMiss reports that a key is absent. It is a normal outcome, not a
// failure: callers should fall through to the origin, not surface an error.
var ErrCacheMiss = errors.New("cache miss")

type CacheOptions struct {
	TTL        time.Duration
	LocalCache bool
}

type CacheService struct {
	redis      *redis.Client
	local      *LocalCache
	defaultTTL time.Duration
}

// LocalCache is the in-process L1 tier in front of Redis.
//
// Occupancy is len(items) rather than a hand-maintained counter. The previous
// version incremented a size field on every Set, including overwrites of an
// existing key, so the count drifted permanently above the real item count and
// the cache began evicting live entries while nominally under its limit.
type LocalCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
	max   int

	stop chan struct{}
	once sync.Once
}

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

func (i *cacheItem) expired(now time.Time) bool {
	return now.After(i.expiresAt)
}

func NewCacheService(redisClient *redis.Client, localCacheSize int) *CacheService {
	local := &LocalCache{
		items: make(map[string]*cacheItem, localCacheSize),
		max:   localCacheSize,
		stop:  make(chan struct{}),
	}

	go local.cleanup()

	return &CacheService{
		redis:      redisClient,
		local:      local,
		defaultTTL: 5 * time.Minute,
	}
}

// Close stops the background eviction goroutine. Without it the goroutine
// outlives the cache for the life of the process.
func (c *CacheService) Close() {
	c.local.close()
}

func (l *LocalCache) close() {
	l.once.Do(func() { close(l.stop) })
}

func (l *LocalCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-l.stop:
			return
		case now := <-ticker.C:
			l.mu.Lock()
			for key, item := range l.items {
				if item.expired(now) {
					delete(l.items, key)
				}
			}
			l.mu.Unlock()
		}
	}
}

func (l *LocalCache) Get(key string) ([]byte, bool) {
	l.mu.RLock()
	item, exists := l.items[key]
	l.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if item.expired(time.Now()) {
		// Drop it now rather than waiting for the sweep; otherwise an expired
		// entry keeps occupying a slot and can be chosen as the eviction victim
		// ahead of live ones.
		l.Delete(key)
		return nil, false
	}

	return item.value, true
}

func (l *LocalCache) Set(key string, value []byte, ttl time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Only an insert can push the cache over its limit; overwriting an existing
	// key does not grow it.
	if _, exists := l.items[key]; !exists && len(l.items) >= l.max {
		l.evictOldestLocked()
	}

	l.items[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// evictOldestLocked removes the entry closest to expiry. The caller must hold
// the write lock.
//
// "Found nothing yet" is tracked with a bool rather than by treating the empty
// string as a sentinel: "" is a legal map key, and overloading it would both
// skip evicting such an entry and let it be re-selected spuriously.
func (l *LocalCache) evictOldestLocked() {
	var (
		victim   string
		earliest time.Time
		found    bool
	)
	for key, item := range l.items {
		if !found || item.expiresAt.Before(earliest) {
			victim, earliest, found = key, item.expiresAt, true
		}
	}
	if found {
		delete(l.items, victim)
	}
}

func (l *LocalCache) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.items, key)
}

// Len reports the number of entries currently held.
func (l *LocalCache) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.items)
}

func (c *CacheService) Get(ctx context.Context, key string) ([]byte, error) {
	if data, found := c.local.Get(key); found {
		return data, nil
	}

	data, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	c.local.Set(key, data, time.Minute)

	return data, nil
}

func (c *CacheService) Set(ctx context.Context, key string, value []byte, opts CacheOptions) error {
	ttl := opts.TTL
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	if err := c.redis.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	if opts.LocalCache {
		localTTL := ttl
		if localTTL > time.Minute {
			localTTL = time.Minute
		}
		c.local.Set(key, value, localTTL)
	}

	return nil
}

func (c *CacheService) Delete(ctx context.Context, key string) error {
	c.local.Delete(key)

	if err := c.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}

	return nil
}

func (c *CacheService) DeletePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var keys []string

	for {
		var err error
		var batch []string
		batch, cursor, err = c.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("redis scan error: %w", err)
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	if len(keys) > 0 {
		if err := c.redis.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("redis delete error: %w", err)
		}
	}

	return nil
}

// GetJSON decodes the cached value for key into dest.
//
// A miss returns ErrCacheMiss. It previously returned a nil error without
// touching dest, so callers could not tell a hit from a miss and would treat the
// zero value as though it had been cached.
func (c *CacheService) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	if data == nil {
		return ErrCacheMiss
	}

	return json.Unmarshal(data, dest)
}

func (c *CacheService) SetJSON(ctx context.Context, key string, value interface{}, opts CacheOptions) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	return c.Set(ctx, key, data, opts)
}

func (c *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	if _, found := c.local.Get(key); found {
		return true, nil
	}

	result, err := c.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists error: %w", err)
	}

	return result > 0, nil
}

func (c *CacheService) Incr(ctx context.Context, key string) (int64, error) {
	return c.redis.Incr(ctx, key).Result()
}

func (c *CacheService) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.redis.IncrBy(ctx, key, value).Result()
}

func (c *CacheService) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.redis.Expire(ctx, key, ttl).Err()
}

func (c *CacheService) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.redis.TTL(ctx, key).Result()
}

func (c *CacheService) ZAdd(ctx context.Context, key string, score float64, member string) error {
	return c.redis.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

func (c *CacheService) ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error) {
	return c.redis.ZIncrBy(ctx, key, increment, member).Result()
}

func (c *CacheService) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.redis.ZRevRange(ctx, key, start, stop).Result()
}

func (c *CacheService) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return c.redis.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

func (c *CacheService) ZRemRangeByScore(ctx context.Context, key string, min, max string) error {
	return c.redis.ZRemRangeByScore(ctx, key, min, max).Err()
}

func (c *CacheService) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.redis.SAdd(ctx, key, members...).Err()
}

func (c *CacheService) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.redis.SMembers(ctx, key).Result()
}

func (c *CacheService) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return c.redis.SIsMember(ctx, key, member).Result()
}

func (c *CacheService) SRem(ctx context.Context, key string, members ...interface{}) error {
	return c.redis.SRem(ctx, key, members...).Err()
}

func (c *CacheService) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.redis.HSet(ctx, key, values...).Err()
}

func (c *CacheService) HGet(ctx context.Context, key, field string) (string, error) {
	result, err := c.redis.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return "", nil
	}
	return result, err
}

func (c *CacheService) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.redis.HGetAll(ctx, key).Result()
}

func (c *CacheService) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return c.redis.HIncrBy(ctx, key, field, incr).Result()
}

func GenerateCacheKey(prefix string, params ...interface{}) string {
	h := sha256.New()
	data := fmt.Sprintf("%s|%v", prefix, params)
	h.Write([]byte(data))
	return fmt.Sprintf("%s:%x", prefix, h.Sum(nil)[:8])
}

func UserCacheKey(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

func VideoCacheKey(videoID string) string {
	return fmt.Sprintf("video:%s", videoID)
}

func VideoListCacheKey(page, limit int, filters string) string {
	return GenerateCacheKey("video_list", page, limit, filters)
}

func SearchCacheKey(query string, filters interface{}, page, limit int) string {
	return GenerateCacheKey("search", query, filters, page, limit)
}

func RecommendationCacheKey(userID string) string {
	return fmt.Sprintf("recommendations:%s", userID)
}

func TrendingCacheKey() string {
	return "trending:videos"
}

func PopularSearchesCacheKey() string {
	return "popular_searches"
}

func AnalyticsCacheKey(videoID, dateRange string) string {
	return fmt.Sprintf("analytics:%s:%s", videoID, dateRange)
}
