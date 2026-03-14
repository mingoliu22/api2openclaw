package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// QuotaManager 配额管理器
type QuotaManager struct {
	mu         sync.RWMutex
	quotas     map[string]*QuotaState
	limiter    *CompositeLimiter
	resetTimes map[string]time.Time
}

// QuotaState 配额状态
type QuotaState struct {
	APIKeyID        string
	TenantID        string

	// 请求数配额
	RPMUsed         int
	RPMReset        time.Time
	RPHUsed         int
	RPHReset        time.Time
	RPDUsed         int
	RPDReset        time.Time

	// Token 配额
	TPDUsed         int
	TPDReset        time.Time

	// 总配额
	RPMLimit        int
	RPHLimit        int
	RPDLimit        int
	TPDLimit        int
}

// QuotaCheckResult 配额检查结果
type QuotaCheckResult struct {
	Allowed     bool
	Reason      string
	LimitType   string
	Limit       int
	Used        int
	Remaining   int
	ResetTime   time.Time
	RetryAfter  int64
}

// NewQuotaManager 创建配额管理器
func NewQuotaManager() *QuotaManager {
	return &QuotaManager{
		quotas:     make(map[string]*QuotaState),
		limiter:    NewCompositeLimiter(),
		resetTimes: make(map[string]time.Time),
	}
}

// CheckQuota 检查配额
func (m *QuotaManager) CheckQuota(ctx context.Context, apiKeyID string, limits *RateLimitConfig) *QuotaCheckResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	state, ok := m.quotas[apiKeyID]
	if !ok {
		// 初始化配额状态
		state = &QuotaState{
			APIKeyID:  apiKeyID,
			RPMReset:  now.Truncate(time.Minute).Add(time.Minute),
			RPHReset:  now.Truncate(time.Hour).Add(time.Hour),
			RPDReset:  now.Truncate(24 * time.Hour).Add(24 * time.Hour),
			TPDReset:  now.Truncate(24 * time.Hour).Add(24 * time.Hour),
			RPMLimit:  limits.RequestsPerMinute,
			RPHLimit:  limits.RequestsPerHour,
			RPDLimit:  limits.RequestsPerDay,
			TPDLimit:  limits.TokensPerDay,
		}
		m.quotas[apiKeyID] = state
	}

	// 检查是否需要重置计数器
	m.resetCountersIfNeeded(state, now)

	// 检查 RPM
	if limits.RequestsPerMinute > 0 && state.RPMUsed >= limits.RequestsPerMinute {
		retryAfter := int64(state.RPMReset.Sub(now).Seconds()) + 1
		return &QuotaCheckResult{
			Allowed:    false,
			Reason:     "RPM limit exceeded",
			LimitType:  "rpm",
			Limit:      limits.RequestsPerMinute,
			Used:       state.RPMUsed,
			Remaining:  0,
			ResetTime:  state.RPMReset,
			RetryAfter: retryAfter,
		}
	}

	// 检查 RPH
	if limits.RequestsPerHour > 0 && state.RPHUsed >= limits.RequestsPerHour {
		retryAfter := int64(state.RPHReset.Sub(now).Seconds()) + 1
		return &QuotaCheckResult{
			Allowed:    false,
			Reason:     "RPH limit exceeded",
			LimitType:  "rph",
			Limit:      limits.RequestsPerHour,
			Used:       state.RPHUsed,
			Remaining:  0,
			ResetTime:  state.RPHReset,
			RetryAfter: retryAfter,
		}
	}

	// 检查 RPD
	if limits.RequestsPerDay > 0 && state.RPDUsed >= limits.RequestsPerDay {
		retryAfter := int64(state.RPDReset.Sub(now).Seconds()) + 1
		return &QuotaCheckResult{
			Allowed:    false,
			Reason:     "RPD limit exceeded",
			LimitType:  "rpd",
			Limit:      limits.RequestsPerDay,
			Used:       state.RPDUsed,
			Remaining:  0,
			ResetTime:  state.RPDReset,
			RetryAfter: retryAfter,
		}
	}

	// 检查 TPD
	if limits.TokensPerDay > 0 && state.TPDUsed >= limits.TokensPerDay {
		retryAfter := int64(state.TPDReset.Sub(now).Seconds()) + 1
		return &QuotaCheckResult{
			Allowed:    false,
			Reason:     "TPD limit exceeded",
			LimitType:  "tpd",
			Limit:      limits.TokensPerDay,
			Used:       state.TPDUsed,
			Remaining:  0,
			ResetTime:  state.TPDReset,
			RetryAfter: retryAfter,
		}
	}

	// 配额充足
	return &QuotaCheckResult{
		Allowed: true,
	}
}

// RecordUsage 记录使用量
func (m *QuotaManager) RecordUsage(apiKeyID string, tokens int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.quotas[apiKeyID]
	if !ok {
		return fmt.Errorf("quota state not initialized for %s", apiKeyID)
	}

	now := time.Now()
	m.resetCountersIfNeeded(state, now)

	// 增加计数
	state.RPMUsed++
	state.RPHUsed++
	state.RPDUsed++
	state.TPDUsed += tokens

	return nil
}

// GetQuotaStatus 获取配额状态
func (m *QuotaManager) GetQuotaStatus(apiKeyID string) *QuotaState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if state, ok := m.quotas[apiKeyID]; ok {
		// 返回副本
		copy := *state
		return &copy
	}

	return nil
}

// ResetQuota 重置配额
func (m *QuotaManager) ResetQuota(apiKeyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.quotas, apiKeyID)
	delete(m.resetTimes, apiKeyID)

	return nil
}

// resetCountersIfNeeded 如果需要则重置计数器
func (m *QuotaManager) resetCountersIfNeeded(state *QuotaState, now time.Time) {
	// 重置 RPM
	if now.After(state.RPMReset) {
		state.RPMUsed = 0
		state.RPMReset = now.Truncate(time.Minute).Add(time.Minute)
	}

	// 重置 RPH
	if now.After(state.RPHReset) {
		state.RPHUsed = 0
		state.RPHReset = now.Truncate(time.Hour).Add(time.Hour)
	}

	// 重置 RPD 和 TPD
	if now.After(state.RPDReset) {
		state.RPDUsed = 0
		state.RPDReset = now.Truncate(24 * time.Hour).Add(24 * time.Hour)
		state.TPDUsed = 0
		state.TPDReset = state.RPDReset
	}
}

// CalculateRetryAfter 计算重试时间
func CalculateRetryAfter(resetTime time.Time) int64 {
	now := time.Now()
	retryAfter := int64(resetTime.Sub(now).Seconds())

	if retryAfter < 1 {
		return 1
	}

	return retryAfter
}

// BuildRetryAfterHeader 构建 Retry-After 响应头值
func BuildRetryAfterHeader(resetTime time.Time) string {
	retryAfter := CalculateRetryAfter(resetTime)
	return fmt.Sprintf("%d", retryAfter)
}

// GetRateLimitHeaders 获取限流响应头
func GetRateLimitHeaders(result *QuotaCheckResult) map[string]string {
	headers := make(map[string]string)

	if result == nil {
		return headers
	}

	headers["X-RateLimit-Limit"] = fmt.Sprintf("%d", result.Limit)
	headers["X-RateLimit-Used"] = fmt.Sprintf("%d", result.Used)
	headers["X-RateLimit-Remaining"] = fmt.Sprintf("%d", result.Remaining)
	headers["X-RateLimit-Reset"] = fmt.Sprintf("%d", result.ResetTime.Unix())

	if !result.Allowed {
		headers["Retry-After"] = fmt.Sprintf("%d", result.RetryAfter)
		headers["X-RateLimit-Scope"] = result.LimitType
	}

	return headers
}

// QuotaExceededError 配额超限错误
type QuotaExceededError struct {
	Result *QuotaCheckResult
}

func (e *QuotaExceededError) Error() string {
	if e.Result != nil {
		return fmt.Sprintf("quota exceeded: %s (limit: %d, retry after: %d seconds)",
			e.Result.Reason, e.Result.Limit, e.Result.RetryAfter)
	}
	return "quota exceeded"
}

// HTTPStatus 返回 HTTP 状态码
func (e *QuotaExceededError) HTTPStatus() int {
	return 429
}

// QuotaStore 配额持久化存储接口
type QuotaStore interface {
	SaveQuotaState(ctx context.Context, state *QuotaState) error
	LoadQuotaState(ctx context.Context, apiKeyID string) (*QuotaState, error)
	DeleteQuotaState(ctx context.Context, apiKeyID string) error
}

// PersistentQuotaManager 持久化配额管理器
type PersistentQuotaManager struct {
	*QuotaManager
	store QuotaStore
}

// NewPersistentQuotaManager 创建持久化配额管理器
func NewPersistentQuotaManager(store QuotaStore) *PersistentQuotaManager {
	return &PersistentQuotaManager{
		QuotaManager: NewQuotaManager(),
		store:        store,
	}
}

// RecordUsage 记录使用量（带持久化）
func (m *PersistentQuotaManager) RecordUsage(apiKeyID string, tokens int) error {
	err := m.QuotaManager.RecordUsage(apiKeyID, tokens)
	if err != nil {
		return err
	}

	// 异步保存状态
	state := m.QuotaManager.GetQuotaStatus(apiKeyID)
	if state != nil {
		go func() {
			_ = m.store.SaveQuotaState(context.Background(), state)
		}()
	}

	return nil
}

// RestoreState 恢复配额状态
func (m *PersistentQuotaManager) RestoreState(ctx context.Context, apiKeyID string) error {
	state, err := m.store.LoadQuotaState(ctx, apiKeyID)
	if err != nil {
		return err
	}

	m.QuotaManager.mu.Lock()
	defer m.QuotaManager.mu.Unlock()

	m.QuotaManager.quotas[apiKeyID] = state
	return nil
}
