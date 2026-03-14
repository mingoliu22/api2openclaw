package monitor

import (
	"fmt"
	"sync"
	"time"

	"github.com/openclaw/api2openclaw/internal/auth"
	"github.com/openclaw/api2openclaw/internal/models"
)

// RateLimiter 限流器
type RateLimiter struct {
	mu    sync.RWMutex
	store LimitStore
}

// LimitStore 限流存储接口
type LimitStore interface {
	GetCounter(key string, window time.Duration) (int64, error)
	IncrementCounter(key string, window time.Duration) (int64, error)
	ResetCounter(key string, window time.Duration) error
}

// MemoryLimitStore 内存限流存储
type MemoryLimitStore struct {
	mu     sync.Mutex
	counters map[string]*counterEntry
}

type counterEntry struct {
	count      int64
	windowStart time.Time
	expiresAt  time.Time
}

// NewMemoryLimitStore 创建内存限流存储
func NewMemoryLimitStore() *MemoryLimitStore {
	store := &MemoryLimitStore{
		counters: make(map[string]*counterEntry),
	}

	// 启动清理协程
	go store.cleanup()

	return store
}

// cleanup 清理过期计数器
func (s *MemoryLimitStore) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, entry := range s.counters {
			if now.After(entry.expiresAt) {
				delete(s.counters, key)
			}
		}
		s.mu.Unlock()
	}
}

// GetCounter 获取计数器
func (s *MemoryLimitStore) GetCounter(key string, window time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.counters[key]
	if !exists {
		return 0, nil
	}

	// 检查窗口是否过期
	if time.Now().After(entry.expiresAt) {
		delete(s.counters, key)
		return 0, nil
	}

	return entry.count, nil
}

// IncrementCounter 增加计数器
func (s *MemoryLimitStore) IncrementCounter(key string, window time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	entry, exists := s.counters[key]

	if !exists || now.After(entry.expiresAt) {
		// 创建新窗口
		entry = &counterEntry{
			windowStart: now,
			expiresAt:  now.Add(window),
		}
		s.counters[key] = entry
	}

	entry.count++
	return entry.count, nil
}

// ResetCounter 重置计数器
func (s *MemoryLimitStore) ResetCounter(key string, window time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.counters, key)
	return nil
}

// NewRateLimiter 创建限流器
func NewRateLimiter(store LimitStore) *RateLimiter {
	return &RateLimiter{store: store}
}

// CheckLimit 检查是否超过限流
func (r *RateLimiter) CheckLimit(apiKeyID string, limits *models.RateLimit) error {
	if limits == nil {
		return nil
	}

	now := time.Now()

	// 按分钟检查
	if limits.RequestsPerMinute > 0 {
		key := r.makeKey(apiKeyID, "minute", now)
		count, _ := r.store.GetCounter(key, time.Minute)
		if count >= int64(limits.RequestsPerMinute) {
			return auth.ErrRateLimitExceeded
		}
	}

	// 按小时检查
	if limits.RequestsPerHour > 0 {
		key := r.makeKey(apiKeyID, "hour", now)
		count, _ := r.store.GetCounter(key, time.Hour)
		if count >= int64(limits.RequestsPerHour) {
			return auth.ErrRateLimitExceeded
		}
	}

	// 按天检查
	if limits.RequestsPerDay > 0 {
		key := r.makeKey(apiKeyID, "day", now)
		count, _ := r.store.GetCounter(key, 24*time.Hour)
		if count >= int64(limits.RequestsPerDay) {
			return auth.ErrRateLimitExceeded
		}
	}

	return nil
}

// Increment 增加计数
func (r *RateLimiter) Increment(apiKeyID string, limits *models.RateLimit) error {
	if limits == nil {
		return nil
	}

	now := time.Now()

	if limits.RequestsPerMinute > 0 {
		key := r.makeKey(apiKeyID, "minute", now)
		r.store.IncrementCounter(key, time.Minute)
	}

	if limits.RequestsPerHour > 0 {
		key := r.makeKey(apiKeyID, "hour", now)
		r.store.IncrementCounter(key, time.Hour)
	}

	if limits.RequestsPerDay > 0 {
		key := r.makeKey(apiKeyID, "day", now)
		r.store.IncrementCounter(key, 24*time.Hour)
	}

	return nil
}

// makeKey 生成限流键
func (r *RateLimiter) makeKey(apiKeyID, window string, t time.Time) string {
	windowSeconds := int64(60)
	switch window {
	case "hour":
		windowSeconds = 3600
	case "day":
		windowSeconds = 86400
	}

	windowIndex := t.Unix() / windowSeconds
	return fmt.Sprintf("ratelimit:%s:%s:%d", apiKeyID, window, windowIndex)
}
