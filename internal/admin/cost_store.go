package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// CostStore 成本存储实现
type CostStore struct {
	db *sqlx.DB
}

// NewCostStore 创建成本存储
func NewCostStore(db *sqlx.DB) *CostStore {
	return &CostStore{db: db}
}

// ModelCostConfig 模型成本配置
type ModelCostConfig struct {
	ID                      string    `json:"id" db:"id"`
	ModelID                 string    `json:"model_id" db:"model_id"`
	ModelAlias              string    `json:"model_alias" db:"model_alias"`
	GPUCount                int       `json:"gpu_count" db:"gpu_count"`
	PowerPerGPUW            int       `json:"power_per_gpu_w" db:"power_per_gpu_w"`
	ElectricityPricePerKWh  float64   `json:"electricity_price_per_kwh" db:"electricity_price_per_kwh"`
	DepreciationPerGPUMonth int       `json:"depreciation_per_gpu_month" db:"depreciation_per_gpu_month"`
	PUE                     float64   `json:"pue" db:"pue"`
	EffectiveFrom           time.Time `json:"effective_from" db:"effective_from"`
	CreatedAt               time.Time `json:"created_at" db:"created_at"`
}

// DailyCostStats 每日成本统计
type DailyCostStats struct {
	StatDate          string  `json:"stat_date" db:"stat_date"`
	ModelAlias        string  `json:"model_alias" db:"model_alias"`
	OnlineHours       float64 `json:"online_hours" db:"online_hours"`
	TotalTokens       int64   `json:"total_tokens" db:"total_tokens"`
	CostElectricity   float64 `json:"cost_electricity" db:"cost_electricity"`
	CostDepreciation  float64 `json:"cost_depreciation" db:"cost_depreciation"`
	CostTotal         float64 `json:"cost_total" db:"cost_total"`
	CostPer1kTokens   float64 `json:"cost_per_1k_tokens" db:"cost_per_1k_tokens"`
}

// ListModelCostConfigs 获取指定模型的成本配置列表
func (s *CostStore) ListModelCostConfigs(ctx context.Context, modelID string) ([]ModelCostConfig, error) {
	query := `
		SELECT
			c.id,
			c.model_id,
			m.alias as model_alias,
			c.gpu_count,
			c.power_per_gpu_w,
			c.electricity_price_per_kwh,
			c.depreciation_per_gpu_month,
			c.pue,
			c.effective_from,
			c.created_at
		FROM model_cost_configs c
		JOIN models m ON m.id = c.model_id
		WHERE c.model_id = $1
		ORDER BY c.effective_from DESC
	`

	var configs []ModelCostConfig
	err := s.db.SelectContext(ctx, &configs, query, modelID)
	if err != nil {
		return nil, fmt.Errorf("list cost configs: %w", err)
	}

	return configs, nil
}

// GetActiveCostConfig 获取模型当前生效的成本配置
func (s *CostStore) GetActiveCostConfig(ctx context.Context, modelID string) (*ModelCostConfig, error) {
	query := `
		SELECT
			c.id,
			c.model_id,
			m.alias as model_alias,
			c.gpu_count,
			c.power_per_gpu_w,
			c.electricity_price_per_kwh,
			c.depreciation_per_gpu_month,
			c.pue,
			c.effective_from,
			c.created_at
		FROM model_cost_configs c
		JOIN models m ON m.id = c.model_id
		WHERE c.model_id = $1
		AND c.effective_from <= NOW()
		ORDER BY c.effective_from DESC
		LIMIT 1
	`

	var config ModelCostConfig
	err := s.db.GetContext(ctx, &config, query, modelID)
	if err != nil {
		return nil, fmt.Errorf("get active cost config: %w", err)
	}

	return &config, nil
}

// CreateModelCostConfig 创建成本配置
func (s *CostStore) CreateModelCostConfig(ctx context.Context, config *ModelCostConfig) error {
	query := `
		INSERT INTO model_cost_configs (
			model_id, gpu_count, power_per_gpu_w,
			electricity_price_per_kwh, depreciation_per_gpu_month,
			pue, effective_from
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	err := s.db.QueryRowContext(ctx, query,
		config.ModelID,
		config.GPUCount,
		config.PowerPerGPUW,
		config.ElectricityPricePerKWh,
		config.DepreciationPerGPUMonth,
		config.PUE,
		config.EffectiveFrom,
	).Scan(&config.ID, &config.CreatedAt)

	if err != nil {
		return fmt.Errorf("create cost config: %w", err)
	}

	return nil
}

// UpdateModelCostConfig 更新成本配置
func (s *CostStore) UpdateModelCostConfig(ctx context.Context, id string, config *ModelCostConfig) error {
	query := `
		UPDATE model_cost_configs
		SET
			gpu_count = $2,
			power_per_gpu_w = $3,
			electricity_price_per_kwh = $4,
			depreciation_per_gpu_month = $5,
			pue = $6
		WHERE id = $1
	`

	result, err := s.db.ExecContext(ctx, query,
		id,
		config.GPUCount,
		config.PowerPerGPUW,
		config.ElectricityPricePerKWh,
		config.DepreciationPerGPUMonth,
		config.PUE,
	)

	if err != nil {
		return fmt.Errorf("update cost config: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("cost config not found")
	}

	return nil
}

// DeleteModelCostConfig 删除成本配置
func (s *CostStore) DeleteModelCostConfig(ctx context.Context, id string) error {
	query := `DELETE FROM model_cost_configs WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete cost config: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("cost config not found")
	}

	return nil
}

// GetDailyCostStats 获取每日成本统计
func (s *CostStore) GetDailyCostStats(ctx context.Context, days int) ([]DailyCostStats, error) {
	query := `
		SELECT
			stat_date,
			model_alias,
			online_hours,
			total_tokens,
			cost_electricity,
			cost_depreciation,
			cost_total,
			cost_per_1k_tokens
		FROM stats_cost_daily
		WHERE stat_date >= CURRENT_DATE - INTERVAL '%d days'
		ORDER BY stat_date DESC, model_alias
	`

	var stats []DailyCostStats
	err := s.db.SelectContext(ctx, &stats, query, days)
	if err != nil {
		return nil, fmt.Errorf("get daily cost stats: %w", err)
	}

	return stats, nil
}

// GetDailyCostStatsByModel 获取指定模型的每日成本统计
func (s *CostStore) GetDailyCostStatsByModel(ctx context.Context, modelAlias string, days int) ([]DailyCostStats, error) {
	query := `
		SELECT
			stat_date,
			model_alias,
			online_hours,
			total_tokens,
			cost_electricity,
			cost_depreciation,
			cost_total,
			cost_per_1k_tokens
		FROM stats_cost_daily
		WHERE model_alias = $1
		AND stat_date >= CURRENT_DATE - INTERVAL '%d days'
		ORDER BY stat_date DESC
	`

	var stats []DailyCostStats
	err := s.db.SelectContext(ctx, &stats, query, modelAlias, days)
	if err != nil {
		return nil, fmt.Errorf("get daily cost stats by model: %w", err)
	}

	return stats, nil
}

// RefreshCostStats 触发成本统计刷新
func (s *CostStore) RefreshCostStats(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `SELECT refresh_stats_cost_daily()`)
	if err != nil {
		return fmt.Errorf("refresh cost stats: %w", err)
	}
	return nil
}

// CalculateDailyCosts 触发指定日期的成本计算
func (s *CostStore) CalculateDailyCosts(ctx context.Context, statDate string) error {
	_, err := s.db.ExecContext(ctx, `SELECT calculate_daily_costs($1::DATE)`, statDate)
	if err != nil {
		return fmt.Errorf("calculate daily costs: %w", err)
	}
	return nil
}

// GetAllCostConfigs 获取所有成本配置（用于管理页面）
func (s *CostStore) GetAllCostConfigs(ctx context.Context) ([]ModelCostConfig, error) {
	query := `
		SELECT
			c.id,
			c.model_id,
			m.alias as model_alias,
			c.gpu_count,
			c.power_per_gpu_w,
			c.electricity_price_per_kwh,
			c.depreciation_per_gpu_month,
			c.pue,
			c.effective_from,
			c.created_at
		FROM model_cost_configs c
		JOIN models m ON m.id = c.model_id
		ORDER BY m.alias, c.effective_from DESC
	`

	var configs []ModelCostConfig
	err := s.db.SelectContext(ctx, &configs, query)
	if err != nil {
		return nil, fmt.Errorf("get all cost configs: %w", err)
	}

	return configs, nil
}

// GetCostSummary 获取成本汇总数据
func (s *CostStore) GetCostSummary(ctx context.Context, days int) (map[string]interface{}, error) {
	query := `
		SELECT
			COALESCE(SUM(cost_electricity), 0) as total_electricity,
			COALESCE(SUM(cost_depreciation), 0) as total_depreciation,
			COALESCE(SUM(cost_total), 0) as total_cost,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COUNT(DISTINCT model_alias) as active_models
		FROM stats_cost_daily
		WHERE stat_date >= CURRENT_DATE - INTERVAL '%d days'
		AND stat_date < CURRENT_DATE
	`

	var summary struct {
		TotalElectricity float64 `db:"total_electricity"`
		TotalDepreciation float64 `db:"total_depreciation"`
		TotalCost         float64 `db:"total_cost"`
		TotalTokens       int64   `db:"total_tokens"`
		ActiveModels      int     `db:"active_models"`
	}

	err := s.db.GetContext(ctx, &summary, query, days)
	if err != nil {
		return nil, fmt.Errorf("get cost summary: %w", err)
	}

	costPer1k := 0.0
	if summary.TotalTokens > 0 {
		costPer1k = summary.TotalCost / float64(summary.TotalTokens) * 1000.0
	}

	return map[string]interface{}{
		"total_electricity":    summary.TotalElectricity,
		"total_depreciation":   summary.TotalDepreciation,
		"total_cost":           summary.TotalCost,
		"total_tokens":         summary.TotalTokens,
		"active_models":        summary.ActiveModels,
		"cost_per_1k_tokens":   costPer1k,
		"period_days":          days,
	}, nil
}
