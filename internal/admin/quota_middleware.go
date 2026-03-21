package admin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// QuotaMiddleware 配额检查中间件
type QuotaMiddleware struct {
	store *QuotaStore
}

// NewQuotaMiddleware 创建配额中间件
func NewQuotaMiddleware(store *QuotaStore) *QuotaMiddleware {
	return &QuotaMiddleware{store: store}
}

// QuotaStore 配额存储接口
type QuotaStore interface {
	// CheckAndIncrement 检查并递增配额使用
	CheckAndIncrement(ctx context.Context, keyID string, tokens int64) (*QuotaCheckResult, error)
	// GetQuotaStatus 获取配额状态
	GetQuotaStatus(ctx context.Context, keyID string) (*QuotaStatus, error)
}

// QuotaCheckResult 配额检查结果
type QuotaCheckResult struct {
	KeyID            string    `json:"key_id"`
	TokensUsed       int64     `json:"tokens_used"`
	SoftLimit        int64     `json:"soft_limit"`
	HardLimit        int64     `json:"hard_limit"`
	Remaining        int64     `json:"remaining"`
	SoftExceeded     bool      `json:"soft_exceeded"`
	HardExceeded     bool      `json:"hard_exceeded"`
	ResetAt          time.Time `json:"reset_at"`
}

// QuotaStatus 配额状态
type QuotaStatus struct {
	TokensUsed       int64     `json:"tokens_used"`
	SoftLimit        int64     `json:"soft_limit"`
	HardLimit        int64     `json:"hard_limit"`
	SoftExceededAt   *time.Time `json:"soft_exceeded_at,omitempty"`
	HardExceededCount int       `json:"hard_exceeded_count"`
	ResetAt          time.Time `json:"reset_at"`
}

// CheckAndIncrement 检查配额并递增（请求完成后调用）
func (m *QuotaMiddleware) CheckAndIncrement(c *gin.Context, keyID string, tokens int64) (*QuotaCheckResult, error) {
	ctx := c.Request.Context()

	result, err := m.store.CheckAndIncrement(ctx, keyID, tokens)
	if err != nil {
		return nil, fmt.Errorf("check quota: %w", err)
	}

	// 触发软上限告警
	if result.SoftExceeded {
		m.sendSoftLimitAlert(c, result)
	}

	// 设置响应头
	c.Header("X-RateLimit-Limit-Tokens-Day", fmt.Sprintf("%d", result.HardLimit))
	c.Header("X-RateLimit-Remaining-Tokens-Day", fmt.Sprintf("%d", result.Remaining))

	// 计算重置时间（明日 00:00 UTC+8）
	resetAt := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
	c.Header("X-RateLimit-Reset-Tokens-Day", fmt.Sprintf("%d", resetAt.Unix()))

	return result, nil
}

// CheckQuota 检查配额（请求前调用）
func (m *QuotaMiddleware) CheckQuota(c *gin.Context, keyID string) error {
	ctx := c.Request.Context()

	status, err := m.store.GetQuotaStatus(ctx, keyID)
	if err != nil {
		return fmt.Errorf("get quota status: %w", err)
	}

	// 检查硬上限
	if status.HardLimit > 0 && status.TokensUsed >= status.HardLimit {
		return m.quotaExceededError(c, status)
	}

	return nil
}

// sendSoftLimitAlert 发送软上限告警
func (m *QuotaMiddleware) sendSoftLimitAlert(c *gin.Context, result *QuotaCheckResult) {
	// TODO: 实现告警逻辑
	// 1. 检查静默期（10 分钟内不重复发送）
	// 2. 调用 Webhook 发送告警
	// 3. 记录告警日志
	fmt.Printf("[QuotaMiddleware] Soft limit exceeded for key %s\n", result.KeyID)
}

// quotaExceededError 返回配额超限错误
func (m *QuotaMiddleware) quotaExceededError(c *gin.Context, status *QuotaStatus) error {
	resetAt := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Header("Retry-After", fmt.Sprintf("%d", int(time.Until(resetAt.Sub(time.Now()).Seconds())))

	// 返回 429 响应
	c.JSON(http.StatusTooManyRequests, gin.H{
		"error": gin.H{
			"code":    "quota_exceeded",
			"message": "今日 token 配额已用完",
			"details": gin.H{
				"quota_type":      "daily_tokens",
				"limit":           status.HardLimit,
				"used":             status.TokensUsed,
				"reset_at":         resetAt.Format(time.RFC3339),
				"retry_after_seconds": int(time.Until(resetAt.Sub(time.Now()).Seconds()),
				"suggestion":       "请联系管理员提升配额，或等待明日 00:00 重置",
			},
		},
	})

	return fmt.Errorf("quota exceeded")
}

// QuotaCheckItem 请求完成后配额检查项
type QuotaCheckItem struct {
	KeyID    string `json:"key_id"`
	Tokens   int64  `json:"tokens"`
}

// ProcessQuota 处理配额检查（在响应发送器中调用）
func (m *QuotaMiddleware) ProcessQuota(ctx context.Context, items []QuotaCheckItem) error {
	for _, item := range items {
		if item.Tokens <= 0 {
			continue
		}

		// 原子递增配额使用
		_, err := m.store.CheckAndIncrement(ctx, item.KeyID, item.Tokens)
		if err != nil {
			return fmt.Errorf("process quota for key %s: %w", item.KeyID, err)
		}
	}
	return nil
}
