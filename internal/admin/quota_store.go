package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// QuotaStore 配额存储实现
type QuotaStoreImpl struct {
	db *sqlx.DB
}

// NewQuotaStore 创建配额存储
func NewQuotaStore(db *sqlx.DB) QuotaStore {
	return &QuotaStoreImpl{db: db}
}

// CheckAndIncrement 检查并递增配额（原子操作）
func (s *QuotaStoreImpl) CheckAndIncrement(ctx context.Context, keyID string, tokens int64) (*QuotaCheckResult, error) {
	// 获取 Key 的配额配置
	var softLimit, hardLimit int64
	var priority string

	err := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(daily_token_soft_limit, 0),
			COALESCE(daily_token_hard_limit, 0),
			COALESCE(priority, 'normal')
		FROM api_keys
		WHERE id = $1
	`, keyID).Scan(&softLimit, &hardLimit, &priority)

	if err != nil {
		return nil, fmt.Errorf("get quota limits: %w", err)
	}

	// 调用原子递增函数
	var result struct {
		TokensUsed       int64      `db:"tokens_used"`
		RequestsCount    int        `db:"requests_count"`
		SoftExceededAt   *time.Time `db:"soft_exceeded_at"`
		HardExceededCount int        `db:"hard_exceeded_count"`
	}

	err = s.db.QueryRowContext(ctx, `
		SELECT increment_quota_usage($1, $2, $3, $4)
	`, keyID, tokens, softLimit, hardLimit).Scan(
		&result.TokensUsed,
		&result.RequestsCount,
		&result.SoftExceededAt,
		&result.HardExceededCount,
	)

	if err != nil {
		return nil, fmt.Errorf("increment quota: %w", err)
	}

	// 计算重置时间（明日 00:00 UTC+8）
	now := time.Now()
	resetAt := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if now.Hour() >= 0 { // 如果还未过今日 00:00，使用今天
		resetAt = resetAt.AddDate(0, 0, 1)
	}

	return &QuotaCheckResult{
		KeyID:         keyID,
		TokensUsed:    result.TokensUsed,
		SoftLimit:     softLimit,
		HardLimit:     hardLimit,
		Remaining:     hardLimit - result.TokensUsed,
		SoftExceeded:  result.SoftExceededAt != nil,
		HardExceeded:  result.HardExceededCount > 0,
		ResetAt:       resetAt,
	}, nil
}

// GetQuotaStatus 获取配额状态
func (s *QuotaStoreImpl) GetQuotaStatus(ctx context.Context, keyID string) (*QuotaStatus, error) {
	// 获取配额配置
	var softLimit, hardLimit int64

	err := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(daily_token_soft_limit, 0),
			COALESCE(daily_token_hard_limit, 0)
		FROM api_keys
		WHERE id = $1
	`, keyID).Scan(&softLimit, &hardLimit)

	if err != nil {
		return nil, fmt.Errorf("get quota limits: %w", err)
	}

	// 获取今日使用量
	var tokensUsed int
	var softExceededAt *time.Time
	var hardExceededCount int

	err = s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(tokens_used, 0),
			soft_exceeded_at,
			COALESCE(hard_exceeded_count, 0)
		FROM quota_usage_daily
		WHERE key_id = $1 AND usage_date = CURRENT_DATE
	`, keyID).Scan(&tokensUsed, &softExceededAt, &hardExceededCount)

	// 如果记录不存在，返回默认状态
	if err != nil {
		return &QuotaStatus{
			TokensUsed:       0,
			SoftLimit:        softLimit,
			HardLimit:        hardLimit,
			SoftExceededAt:   softExceededAt,
			HardExceededCount: hardExceededCount,
			ResetAt:          getNextResetTime(),
		}, nil
	}

	return &QuotaStatus{
		TokensUsed:       int64(tokensUsed),
		SoftLimit:        softLimit,
		HardLimit:        hardLimit,
		SoftExceededAt:   softExceededAt,
		HardExceededCount: hardExceededCount,
		ResetAt:          getNextResetTime(),
	}, nil
}

// GetQuotaOverview 获取所有 Key 的配额概览（用于列表页）
func (s *QuotaStoreImpl) GetQuotaOverview(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			k.id,
			k.label,
			COALESCE(daily_token_soft_limit, 0),
			COALESCE(daily_token_hard_limit, 0),
			COALESCE(q.tokens_used, 0) as tokens_used,
			k.status,
			k.priority
		FROM api_keys k
		LEFT JOIN quota_usage_daily q ON q.key_id = k.id AND q.usage_date = CURRENT_DATE
		ORDER BY k.created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("get quota overview: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}

	for rows.Next() {
		var id, label, status, priority string
		var softLimit, hardLimit, tokensUsed int64

		if err := rows.Scan(&id, &label, &softLimit, &hardLimit, &tokensUsed, &status, &priority); err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"id":                     id,
			"label":                  label,
			"soft_limit":            softLimit,
			"hard_limit":            hardLimit,
			"tokens_used":           tokensUsed,
			"status":                 status,
			"priority":               priority,
			"usage_percent":         calcUsagePercent(tokensUsed, hardLimit),
			"is_soft_exceeded":      tokensUsed >= softLimit && softLimit > 0,
			"is_hard_exceeded":      tokensUsed >= hardLimit && hardLimit > 0,
		})
	}

	return results, nil
}

// GetQuotaHistory 获取配额使用历史（近 30 日）
func (s *QuotaStoreImpl) GetQuotaHistory(ctx context.Context, keyID string, days int) ([]map[string]interface{}, error) {
	query := `
		SELECT
			usage_date,
			tokens_used,
			requests_count,
			soft_exceeded_at,
			hard_exceeded_count
		FROM quota_usage_daily
		WHERE key_id = $1
		AND usage_date >= CURRENT_DATE - INTERVAL '%d days'
		ORDER BY usage_date DESC
	`

	rows, err := s.db.QueryContext(ctx, query, keyID, days)
	if err != nil {
		return nil, fmt.Errorf("get quota history: %w", err)
	}
	defer rows.Close()

	var history []map[string]interface{}

	for rows.Next() {
		var usageDate time.Time
		var tokensUsed int64
		var requestsCount int
		var softExceededAt *time.Time
		var hardExceededCount int

		if err := rows.Scan(&usageDate, &tokensUsed, &requestsCount, &softExceededAt, &hardExceededCount); err != nil {
			continue
		}

		history = append(history, map[string]interface{}{
			"date":                usageDate.Format("2006-01-02"),
			"tokens_used":         tokensUsed,
			"requests_count":      requestsCount,
			"soft_exceeded_at":    softExceededAt,
			"hard_exceeded_count": hardExceededCount,
		})
	}

	return history, nil
}

// getNextResetTime 获取下一个重置时间（明日 00:00 UTC+8）
func getNextResetTime() time.Time {
	now := time.Now()
	resetAt := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if resetAt.Before(now) {
		resetAt = resetAt.AddDate(0, 0, 1)
	}
	return resetAt
}

// calcUsagePercent 计算使用百分比
func calcUsagePercent(used, limit int64) float64 {
	if limit == 0 {
		return 0
	}
	return float64(used) / float64(limit) * 100
}
