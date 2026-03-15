package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// RequestLogStore 请求日志存储接口
type RequestLogStore interface {
	List(ctx context.Context, filter *RequestLogFilter) ([]*RequestLog, int64, error)
	Create(ctx context.Context, log *RequestLog) error
	GetUsageStats(ctx context.Context, filter *UsageStatsFilter) (*UsageStats, error)
	GetKeyUsageStats(ctx context.Context, keyID string) (*UsageStats, error)
}

// PostgreSQLRequestLogStore PostgreSQL 请求日志存储
type PostgreSQLRequestLogStore struct {
	db *sqlx.DB
}

// NewPostgreSQLRequestLogStore 创建 PostgreSQL 请求日志存储
func NewPostgreSQLRequestLogStore(db *sqlx.DB) RequestLogStore {
	return &PostgreSQLRequestLogStore{db: db}
}

// RequestLogFilter 请求日志过滤器
type RequestLogFilter struct {
	KeyID       *string
	ModelAlias  *string
	StatusCode  *int
	From        *time.Time
	To          *time.Time
	Limit       int
	Offset      int
}

// RequestLog 请求日志
type RequestLog struct {
	ID                string     `json:"id" db:"id"`
	KeyID             *string    `json:"key_id,omitempty" db:"key_id"`
	ModelAlias        string     `json:"model_alias" db:"model_alias"`
	ModelActual       *string    `json:"model_actual,omitempty" db:"model_actual"`
	PromptTokens      int        `json:"prompt_tokens" db:"prompt_tokens"`
	CompletionTokens  int        `json:"completion_tokens" db:"completion_tokens"`
	TotalTokens       int        `json:"total_tokens" db:"total_tokens"`
	LatencyMs         int        `json:"latency_ms" db:"latency_ms"`
	StatusCode        int        `json:"status_code" db:"status_code"`
	ErrorCode         *string    `json:"error_code,omitempty" db:"error_code"`
	ErrorMessage      *string    `json:"error_message,omitempty" db:"error_message"`
	RequestID         *string    `json:"request_id,omitempty" db:"request_id"`
	IPAddress         *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent         *string    `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// List 获取请求日志列表
func (s *PostgreSQLRequestLogStore) List(ctx context.Context, filter *RequestLogFilter) ([]*RequestLog, int64, error) {
	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 1

	if filter.KeyID != nil {
		whereClause += fmt.Sprintf(" AND key_id = $%d", argCount)
		args = append(args, *filter.KeyID)
		argCount++
	}

	if filter.ModelAlias != nil {
		whereClause += fmt.Sprintf(" AND model_alias = $%d", argCount)
		args = append(args, *filter.ModelAlias)
		argCount++
	}

	if filter.StatusCode != nil {
		whereClause += fmt.Sprintf(" AND status_code = $%d", argCount)
		args = append(args, *filter.StatusCode)
		argCount++
	}

	if filter.From != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filter.From)
		argCount++
	}

	if filter.To != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filter.To)
		argCount++
	}

	// 查询总数
	countQuery := "SELECT COUNT(*) FROM request_logs " + whereClause
	var total int64
	err := s.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count request logs: %w", err)
	}

	// 查询数据
	query := `SELECT id, key_id, model_alias, model_actual, prompt_tokens, completion_tokens, total_tokens,
	          latency_ms, status_code, error_code, error_message, request_id, ip_address, user_agent, created_at
	          FROM request_logs ` + whereClause + ` ORDER BY created_at DESC`

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, filter.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query request logs: %w", err)
	}
	defer rows.Close()

	var logs []*RequestLog
	for rows.Next() {
		log := &RequestLog{}
		err := rows.Scan(&log.ID, &log.KeyID, &log.ModelAlias, &log.ModelActual, &log.PromptTokens,
			&log.CompletionTokens, &log.TotalTokens, &log.LatencyMs, &log.StatusCode,
			&log.ErrorCode, &log.ErrorMessage, &log.RequestID, &log.IPAddress, &log.UserAgent, &log.CreatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan request log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}

// Create 创建请求日志
func (s *PostgreSQLRequestLogStore) Create(ctx context.Context, log *RequestLog) error {
	query := `INSERT INTO request_logs (id, key_id, model_alias, model_actual, prompt_tokens, completion_tokens,
	          total_tokens, latency_ms, status_code, error_code, error_message, request_id, ip_address, user_agent)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := s.db.ExecContext(ctx, query,
		log.ID, log.KeyID, log.ModelAlias, log.ModelActual, log.PromptTokens,
		log.CompletionTokens, log.TotalTokens, log.LatencyMs, log.StatusCode,
		log.ErrorCode, log.ErrorMessage, log.RequestID, log.IPAddress, log.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("failed to create request log: %w", err)
	}

	return nil
}

// UsageStatsFilter 用量统计过滤器
type UsageStatsFilter struct {
	KeyID      *string
	ModelAlias *string
	From       *time.Time
	To         *time.Time
}

// UsageStats 用量统计
type UsageStats struct {
	TotalRequests     int64 `json:"total_requests"`
	TotalTokens       int64 `json:"total_tokens"`
	PromptTokens      int64 `json:"prompt_tokens"`
	CompletionTokens  int64 `json:"completion_tokens"`
	ActiveKeys        int64 `json:"active_keys"`
	ActiveModels      int64 `json:"active_models"`
}

// GetUsageStats 获取用量统计
func (s *PostgreSQLRequestLogStore) GetUsageStats(ctx context.Context, filter *UsageStatsFilter) (*UsageStats, error) {
	stats := &UsageStats{}

	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 1

	if filter.KeyID != nil {
		whereClause += fmt.Sprintf(" AND key_id = $%d", argCount)
		args = append(args, *filter.KeyID)
		argCount++
	}

	if filter.ModelAlias != nil {
		whereClause += fmt.Sprintf(" AND model_alias = $%d", argCount)
		args = append(args, *filter.ModelAlias)
		argCount++
	}

	if filter.From != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filter.From)
		argCount++
	}

	if filter.To != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filter.To)
		argCount++
	}

	// 查询总请求数和 Token 统计
	query := `SELECT
	          COUNT(*) as total_requests,
	          COALESCE(SUM(total_tokens), 0) as total_tokens,
	          COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
	          COALESCE(SUM(completion_tokens), 0) as completion_tokens
	          FROM request_logs ` + whereClause

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalRequests, &stats.TotalTokens, &stats.PromptTokens, &stats.CompletionTokens,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	// 查询活跃 Key 数（最近 24 小时有请求的 Key）
	activeQuery := `SELECT COUNT(DISTINCT key_id) FROM request_logs WHERE key_id IS NOT NULL AND created_at > $1`
	err = s.db.QueryRowContext(ctx, activeQuery, time.Now().Add(-24*time.Hour)).Scan(&stats.ActiveKeys)
	if err != nil {
		stats.ActiveKeys = 0
	}

	// 查询活跃模型数（从 models 表查询 is_active = true 的数量）
	modelQuery := `SELECT COUNT(*) FROM models WHERE is_active = true`
	err = s.db.QueryRowContext(ctx, modelQuery).Scan(&stats.ActiveModels)
	if err != nil {
		stats.ActiveModels = 0
	}

	return stats, nil
}

// GetKeyUsageStats 获取指定 API Key 的使用统计
func (s *PostgreSQLRequestLogStore) GetKeyUsageStats(ctx context.Context, keyID string) (*UsageStats, error) {
	stats := &UsageStats{}

	// 查询该 Key 的总请求数和 Token 统计
	query := `SELECT
	          COUNT(*) as total_requests,
	          COALESCE(SUM(total_tokens), 0) as total_tokens,
	          COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
	          COALESCE(SUM(completion_tokens), 0) as completion_tokens
	          FROM request_logs
	          WHERE key_id = $1`

	err := s.db.QueryRowContext(ctx, query, keyID).Scan(
		&stats.TotalRequests, &stats.TotalTokens, &stats.PromptTokens, &stats.CompletionTokens,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get key usage stats: %w", err)
	}

	// 活跃 Key 数和活跃模型数不适用单 Key 统计，设为 0
	stats.ActiveKeys = 0
	stats.ActiveModels = 0

	return stats, nil
}

