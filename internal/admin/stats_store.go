package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// StatsStore 统计数据存储
type StatsStore struct {
	db *sqlx.DB
}

// NewStatsStore 创建统计数据存储
func NewStatsStore(db *sqlx.DB) *StatsStore {
	return &StatsStore{db: db}
}

// RealtimeStats 实时统计数据
type RealtimeStats struct {
	TokensPerSecond    float64 `json:"tokens_per_sec"`
	TokensToday        int64   `json:"tokens_today"`
	TokensYesterday     int64   `json:"tokens_yesterday"`
	OnlineModels       int     `json:"online_models"`
	TotalModels        int     `json:"total_models"`
	ActiveKeys1Hour    int     `json:"active_keys_1h"`
	Threshold          int     `json:"threshold"`
	ThresholdStatus    string  `json:"threshold_status"` // normal | warning | alert
}

// DailyStats 每日统计数据（按小时）
type DailyStats struct {
	Date      string  `json:"date"`
	Hour      int     `json:"hour"`
	Tokens    int64   `json:"tokens"`
	Yesterday int64   `json:"yesterday"` // 昨日同期对比
}

// ModelStats 模型统计数据
type ModelStats struct {
	ModelAlias      string  `json:"model_alias"`
	TokensToday     int64   `json:"tokens_today"`
	TokensPerSec    float64 `json:"tokens_per_sec"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	Online          bool    `json:"online"`
	OnlineHours     float64 `json:"online_hours"`
}

// GetRealtimeStats 获取实时统计数据
func (s *StatsStore) GetRealtimeStats(ctx context.Context, threshold int) (*RealtimeStats, error) {
	// 从 stats_hourly 物化视图查询近 60 秒的 tokens/s
	var tokensPerSec float64
	err := s.db.GetContext(ctx, &tokensPerSec, `
		SELECT COALESCE(SUM(tokens_per_sec), 0)
		FROM stats_hourly
		WHERE stat_date = CURRENT_DATE
		AND stat_hour = EXTRACT(HOUR FROM NOW() - INTERVAL '60 seconds')
	`)
	if err != nil {
		return nil, fmt.Errorf("get tokens per sec: %w", err)
	}

	// 今日总产量
	var tokensToday int64
	err = s.db.GetContext(ctx, &tokensToday, `
		SELECT COALESCE(SUM(total_tokens), 0)
		FROM stats_hourly
		WHERE stat_date = CURRENT_DATE
	`)
	if err != nil {
		return nil, fmt.Errorf("get tokens today: %w", err)
	}

	// 昨日总产量
	var tokensYesterday int64
	err = s.db.GetContext(ctx, &tokensYesterday, `
		SELECT COALESCE(SUM(total_tokens), 0)
		FROM stats_hourly
		WHERE stat_date = CURRENT_DATE - INTERVAL '1 day'
	`)
	if err != nil {
		return nil, fmt.Errorf("get tokens yesterday: %w", err)
	}

	// 在线模型数（从 models 表 + 健康检查结果）
	var onlineModels, totalModels int
	err = s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (is_active = true) as online,
			COUNT(*) as total
		FROM models
	`).Scan(&onlineModels, &totalModels)
	if err != nil {
		return nil, fmt.Errorf("get online models: %w", err)
	}

	// 过去 1 小时有请求的 key 数量
	var activeKeys1Hour int
	err = s.db.GetContext(ctx, &activeKeys1Hour, `
		SELECT COUNT(DISTINCT key_id)
		FROM request_logs
		WHERE created_at >= NOW() - INTERVAL '1 hour'
	`)
	if err != nil {
		return nil, fmt.Errorf("get active keys: %w", err)
	}

	// 计算阈值状态
	var thresholdStatus string
	if tokensPerSec >= float64(threshold) {
		thresholdStatus = "alert"
	} else if tokensPerSec >= float64(threshold)*0.8 {
		thresholdStatus = "warning"
	} else {
		thresholdStatus = "normal"
	}

	return &RealtimeStats{
		TokensPerSecond: tokensPerSec,
		TokensToday:     tokensToday,
		TokensYesterday:  tokensYesterday,
		OnlineModels:    onlineModels,
		TotalModels:     totalModels,
		ActiveKeys1Hour: activeKeys1Hour,
		Threshold:       threshold,
		ThresholdStatus:  thresholdStatus,
	}, nil
}

// GetDailyStats 获取每日统计数据（按小时）
func (s *StatsStore) GetDailyStats(ctx context.Context, date string) ([]DailyStats, error) {
	var stats []DailyStats

	// 查询今日每小时数据
	query := `
		SELECT
			EXTRACT(EPOCH FROM stat_date)::int as date,
			EXTRACT(HOUR FROM stat_date)::int as hour,
			SUM(total_tokens) as tokens,
			0 as yesterday
		FROM stats_hourly
		WHERE stat_date = $1::date
		GROUP BY stat_date, EXTRACT(HOUR FROM stat_date)
		ORDER BY stat_date, EXTRACT(HOUR FROM stat_date)
	`

	err := s.db.SelectContext(ctx, &stats, query, date)
	if err != nil {
		return nil, fmt.Errorf("get daily stats: %w", err)
	}

	// 填充昨日对比数据（简化处理）
	for i := range stats {
		var yesterdayTokens int64
		yesterdayQuery := `
			SELECT COALESCE(SUM(total_tokens), 0)
			FROM stats_hourly
			WHERE stat_date = $1::date - INTERVAL '1 day'
			AND EXTRACT(HOUR FROM stat_date) = $2
		`
		_ = s.db.GetContext(ctx, &yesterdayTokens, yesterdayQuery, date, stats[i].Hour)
		stats[i].Yesterday = yesterdayTokens
	}

	return stats, nil
}

// GetModelStats 获取模型统计数据
func (s *StatsStore) GetModelStats(ctx context.Context) ([]ModelStats, error) {
	var stats []ModelStats

	// 从 stats_hourly 聚合今日各模型数据
	query := `
		SELECT
			sh.model_alias,
			COALESCE(SUM(sh.total_tokens), 0) as tokens_today,
			COALESCE(sh.tokens_per_sec, 0) as tokens_per_sec,
			COALESCE(sh.avg_latency_ms, 0) as avg_latency_ms,
			COALESCE(m.is_active, false) as online
		FROM stats_hourly sh
		LEFT JOIN models m ON sh.model_alias = m.alias
		WHERE sh.stat_date = CURRENT_DATE
		GROUP BY sh.model_alias, m.is_active
		ORDER BY tokens_per_sec DESC
	`

	err := s.db.SelectContext(ctx, &stats, query)
	if err != nil {
		return nil, fmt.Errorf("get model stats: %w", err)
	}

	// 填充在线小时数（简化处理：从今日第一个请求到最后一个请求的时间差）
	for i := range stats {
		var onlineHours float64
		hoursQuery := `
			SELECT EXTRACT(EPOCH FROM (
				MAX(created_at) - MIN(created_at)
			)::int / 3600.0 as hours
			FROM request_logs
			WHERE model_alias = $1
			AND DATE(created_at) = CURRENT_DATE
		`
		_ = s.db.GetContext(ctx, &onlineHours, hoursQuery, stats[i].ModelAlias)
		stats[i].OnlineHours = onlineHours
	}

	return stats, nil
}

// GetThreshold 获取预警阈值配置
func (s *StatsStore) GetThreshold(ctx context.Context) (int, error) {
	var threshold int
	err := s.db.GetContext(ctx, &threshold, `
		SELECT COALESCE(value::int, 500)
		FROM system_configs
		WHERE key = 'stats_threshold'
	`)
	if err != nil {
		// 返回默认值
		return 500, nil
	}
	return threshold, nil
}

// UpdateThreshold 更新预警阈值
func (s *StatsStore) UpdateThreshold(ctx context.Context, threshold int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO system_configs (key, value, updated_at)
		VALUES ('stats_threshold', $1, NOW())
		ON CONFLICT (key) DO UPDATE
		SET value = $1, updated_at = NOW()
	`, threshold)
	if err != nil {
		return fmt.Errorf("update threshold: %w", err)
	}
	return nil
}
