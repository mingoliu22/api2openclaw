package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	mu              sync.RWMutex
	windows         map[string]*slidingWindow
	maxSize         int
	cleanupInterval time.Duration
}

// slidingWindow 滑动窗口
type slidingWindow struct {
	requests []time.Time
	maxCount int
	duration time.Duration
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(maxSize int, cleanupInterval time.Duration) *SlidingWindowLimiter {
	if maxSize <= 0 {
		maxSize = 10000
	}
	if cleanupInterval <= 0 {
		cleanupInterval = 5 * time.Minute
	}

	limiter := &SlidingWindowLimiter{
		windows:         make(map[string]*slidingWindow),
		maxSize:         maxSize,
		cleanupInterval: cleanupInterval,
	}

	// 启动清理协程
	go limiter.cleanupLoop()

	return limiter
}

// cleanupLoop 定期清理过期窗口
func (l *SlidingWindowLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanup()
	}
}

// cleanup 清理过期窗口
func (l *SlidingWindowLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for key, window := range l.windows {
		// 检查窗口是否过期（超过最大时间间隔的 2 倍）
		if len(window.requests) > 0 {
			lastReq := window.requests[len(window.requests)-1]
			if now.Sub(lastReq) > l.cleanupInterval*2 {
				delete(l.windows, key)
			}
		}
	}
}

// checkLimit 检查滑动窗口限制
func (l *SlidingWindowLimiter) checkLimit(key string, maxCount int, duration time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	window, ok := l.windows[key]

	if !ok {
		// 创建新窗口
		if len(l.windows) >= l.maxSize {
			// 窗口数量过多，清理最旧的
			l.evictOldest()
		}

		l.windows[key] = &slidingWindow{
			requests:  []time.Time{now},
			maxCount:  maxCount,
			duration: duration,
		}
		return true
	}

	// 移除窗口外的请求
	cutoff := now.Add(-duration)
	validIdx := 0
	for _, req := range window.requests {
		if req.After(cutoff) {
			window.requests[validIdx] = req
			validIdx++
		}
	}
	window.requests = window.requests[:validIdx]

	// 检查是否超限
	if len(window.requests) >= maxCount {
		return false
	}

	// 添加当前请求
	window.requests = append(window.requests, now)
	return true
}

// evictOldest 淘汰最旧的窗口
func (l *SlidingWindowLimiter) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, window := range l.windows {
		if len(window.requests) > 0 {
			lastReq := window.requests[len(window.requests)-1]
			if oldestTime.IsZero() || lastReq.Before(oldestTime) {
				oldestTime = lastReq
				oldestKey = key
			}
		}
	}

	if oldestKey != "" {
		delete(l.windows, oldestKey)
	}
}

// --- 组合限流器 ---

// CompositeLimiter 组合限流器（支持多种限流规则）
type CompositeLimiter struct {
	rpmLimiter *SlidingWindowLimiter
	rphLimiter *SlidingWindowLimiter
	rpdLimiter *SlidingWindowLimiter
	tpdLimiter *SlidingWindowLimiter
}

// NewCompositeLimiter 创建组合限流器
func NewCompositeLimiter() *CompositeLimiter {
	return &CompositeLimiter{
		rpmLimiter: NewSlidingWindowLimiter(5000, time.Minute),
		rphLimiter: NewSlidingWindowLimiter(5000, time.Hour),
		rpdLimiter: NewSlidingWindowLimiter(5000, 24*time.Hour),
		tpdLimiter: NewSlidingWindowLimiter(5000, 24*time.Hour),
	}
}

// CheckLimit 检查所有限流规则
func (c *CompositeLimiter) CheckLimit(apiKeyID string, limits *RateLimitConfig) error {
	now := time.Now()

	// 检查 RPM（每分钟请求数）
	if limits.RequestsPerMinute > 0 {
		if !c.rpmLimiter.checkLimit(apiKeyID+":rpm", limits.RequestsPerMinute, time.Minute) {
			return &RateLimitError{
				LimitType:     "rpm",
				Limit:         limits.RequestsPerMinute,
				ResetTime:     now.Add(time.Minute),
				RetryAfter:    60,
			}
		}
	}

	// 检查 RPH（每小时请求数）
	if limits.RequestsPerHour > 0 {
		if !c.rphLimiter.checkLimit(apiKeyID+":rph", limits.RequestsPerHour, time.Hour) {
			return &RateLimitError{
				LimitType:     "rph",
				Limit:         limits.RequestsPerHour,
				ResetTime:     now.Add(time.Hour),
				RetryAfter:    3600,
			}
		}
	}

	// 检查 RPD（每天请求数）
	if limits.RequestsPerDay > 0 {
		if !c.rpdLimiter.checkLimit(apiKeyID+":rpd", limits.RequestsPerDay, 24*time.Hour) {
			return &RateLimitError{
				LimitType:     "rpd",
				Limit:         limits.RequestsPerDay,
				ResetTime:     now.Add(24 * time.Hour),
				RetryAfter:    86400,
			}
		}
	}

	// 检查 TPD（每天 Token 数）
	if limits.TokensPerDay > 0 {
		if !c.tpdLimiter.checkLimit(apiKeyID+":tpd", limits.TokensPerDay, 24*time.Hour) {
			return &RateLimitError{
				LimitType:     "tpd",
				Limit:         limits.TokensPerDay,
				ResetTime:     now.Add(24 * time.Hour),
				RetryAfter:    86400,
			}
		}
	}

	return nil
}

// Increment 增加计数
func (c *CompositeLimiter) Increment(apiKeyID string, limits *RateLimitConfig) error {
	// 滑动窗口已经在 checkLimit 时记录了请求
	// 这里只需要记录 token 使用
	if limits.TokensPerDay > 0 {
		// Token 在请求完成后记录，这里需要单独处理
		// 实际使用时应该调用 RecordTokens
	}
	return nil
}

// RecordTokens 记录 token 使用
func (c *CompositeLimiter) RecordTokens(apiKeyID string, tokens int) {
	// Token 计数由独立的 TPD 限流器处理
	// 这里简化处理，实际应该更精确
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	RequestsPerHour   int `json:"requests_per_hour"`
	RequestsPerDay    int `json:"requests_per_day"`
	TokensPerDay      int `json:"tokens_per_day"`
}

// RateLimitError 限流错误
type RateLimitError struct {
	LimitType  string        `json:"limit_type"`
	Limit      int           `json:"limit"`
	ResetTime  time.Time     `json:"reset_time"`
	RetryAfter int64         `json:"retry_after"`
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded: %s (limit: %d, retry after: %d seconds)",
		e.LimitType, e.Limit, e.RetryAfter)
}

// --- 令牌桶限流器 ---

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*tokenBucket
}

// tokenBucket 令牌桶
type tokenBucket struct {
	capacity   int
	tokens     float64
	refillRate float64 // 每秒填充的令牌数
	lastRefill time.Time
}

// NewTokenBucketLimiter 创建令牌桶限流器
func NewTokenBucketLimiter() *TokenBucketLimiter {
	return &TokenBucketLimiter{
		buckets: make(map[string]*tokenBucket),
	}
}

// getOrCreateBucket 获取或创建令牌桶
func (l *TokenBucketLimiter) getOrCreateBucket(key string, capacity int, refillRate float64) *tokenBucket {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[key]
	if !ok {
		bucket = &tokenBucket{
			capacity:   capacity,
			tokens:     float64(capacity),
			refillRate: refillRate,
			lastRefill: time.Now(),
		}
		l.buckets[key] = bucket
	}

	return bucket
}

// refill 填充令牌
func (b *tokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()

	if elapsed > 0 {
		b.tokens += elapsed * b.refillRate
		if b.tokens > float64(b.capacity) {
			b.tokens = float64(b.capacity)
		}
		b.lastRefill = now
	}
}

// tryConsume 尝试消费令牌
func (b *tokenBucket) tryConsume(count int) bool {
	b.refill()

	if b.tokens >= float64(count) {
		b.tokens -= float64(count)
		return true
	}

	return false
}

// CheckLimit 检查令牌桶限流
func (l *TokenBucketLimiter) CheckLimit(key string, capacity int, refillRate float64, tokens int) bool {
	bucket := l.getOrCreateBucket(key, capacity, refillRate)
	return bucket.tryConsume(tokens)
}

// --- 分布式限流器接口（Redis） ---

// DistributedLimiter 分布式限流器接口
type DistributedLimiter interface {
	CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
	Increment(ctx context.Context, key string) error
	GetUsage(ctx context.Context, key string) (int, error)
	Reset(ctx context.Context, key string) error
}

// RedisLimiter Redis 限流器（待实现）
type RedisLimiter struct {
	client interface{} // redis.Client
	prefix string
}

// NewRedisLimiter 创建 Redis 限流器
func NewRedisLimiter(client interface{}, prefix string) *RedisLimiter {
	return &RedisLimiter{
		client: client,
		prefix: prefix,
	}
}

// CheckLimit 检查限流（使用 Redis）
func (r *RedisLimiter) CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	// TODO: 实现 Redis 限流逻辑
	// 使用 Redis INCR + EXPIRE 或 Redis 4.3+ 的 COUNT-MIN SKETCH
	return true, nil
}
