package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgreSQLModelStore PostgreSQL 模型存储
type PostgreSQLModelStore struct {
	db *sqlx.DB
}

// NewPostgreSQLModelStore 创建 PostgreSQL 模型存储
func NewPostgreSQLModelStore(db *sqlx.DB) ModelStore {
	return &PostgreSQLModelStore{db: db}
}

// List 获取模型列表
func (s *PostgreSQLModelStore) List(ctx context.Context, activeOnly bool) ([]*ModelConfig, error) {
	query := `SELECT id, alias, model_id, base_url, api_key_encrypted, note, is_active, created_at, updated_at
	          FROM models`
	if activeOnly {
		query += " WHERE is_active = true"
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query models: %w", err)
	}
	defer rows.Close()

	var models []*ModelConfig
	for rows.Next() {
		m := &ModelConfig{}
		err := rows.Scan(&m.ID, &m.Alias, &m.ModelID, &m.BaseURL, &m.APIKeyEncrypted, &m.Note, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model: %w", err)
		}
		models = append(models, m)
	}

	return models, nil
}

// GetByID 获取模型详情
func (s *PostgreSQLModelStore) GetByID(ctx context.Context, id string) (*ModelConfig, error) {
	query := `SELECT id, alias, model_id, base_url, api_key_encrypted, note, is_active, created_at, updated_at
	          FROM models WHERE id = $1`

	m := &ModelConfig{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID, &m.Alias, &m.ModelID, &m.BaseURL, &m.APIKeyEncrypted, &m.Note, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	return m, nil
}

// GetByAlias 获取模型详情
func (s *PostgreSQLModelStore) GetByAlias(ctx context.Context, alias string) (*ModelConfig, error) {
	query := `SELECT id, alias, model_id, base_url, api_key_encrypted, note, is_active, created_at, updated_at
	          FROM models WHERE alias = $1`

	m := &ModelConfig{}
	err := s.db.QueryRowContext(ctx, query, alias).Scan(
		&m.ID, &m.Alias, &m.ModelID, &m.BaseURL, &m.APIKeyEncrypted, &m.Note, &m.IsActive, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	return m, nil
}

// Create 创建模型
func (s *PostgreSQLModelStore) Create(ctx context.Context, model *ModelConfig) error {
	model.ID = uuid.New().String()
	model.CreatedAt = time.Now()
	model.UpdatedAt = time.Now()

	query := `INSERT INTO models (id, alias, model_id, base_url, api_key_encrypted, note, is_active, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := s.db.ExecContext(ctx, query,
		model.ID, model.Alias, model.ModelID, model.BaseURL, model.APIKeyEncrypted,
		model.Note, model.IsActive, model.CreatedAt, model.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create model: %w", err)
	}

	return nil
}

// Update 更新模型
func (s *PostgreSQLModelStore) Update(ctx context.Context, model *ModelConfig) error {
	model.UpdatedAt = time.Now()

	query := `UPDATE models SET
	          alias = $2, model_id = $3, base_url = $4, api_key_encrypted = $5,
	          note = $6, is_active = $7, updated_at = $8
	          WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query,
		model.ID, model.Alias, model.ModelID, model.BaseURL, model.APIKeyEncrypted,
		model.Note, model.IsActive, model.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("model not found")
	}

	return nil
}

// Delete 删除模型
func (s *PostgreSQLModelStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM models WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("model not found")
	}

	return nil
}

// UpdateActiveStatus 更新激活状态
func (s *PostgreSQLModelStore) UpdateActiveStatus(ctx context.Context, id string, isActive bool) error {
	query := `UPDATE models SET is_active = $2, updated_at = $3 WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id, isActive, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update model status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("model not found")
	}

	return nil
}

// ToggleActive 切换模型启用状态
func (s *PostgreSQLModelStore) ToggleActive(ctx context.Context, id string, isActive bool) error {
	return s.UpdateActiveStatus(ctx, id, isActive)
}

// HealthCheckStatus 健康检查状态
type HealthCheckStatus struct {
	IsHealthy   bool      `json:"is_healthy"`
	LastChecked time.Time `json:"last_checked"`
	Error       string    `json:"error,omitempty"`
}

// GetHealthStatus 获取健康状态
func (s *PostgreSQLModelStore) GetHealthStatus(ctx context.Context, modelID string) (*HealthCheckStatus, error) {
	query := `SELECT is_healthy, last_checked, error FROM model_health WHERE model_id = $1 ORDER BY last_checked DESC LIMIT 1`

	status := &HealthCheckStatus{}
	err := s.db.QueryRowContext(ctx, query, modelID).Scan(&status.IsHealthy, &status.LastChecked, &status.Error)
	if err != nil {
		return nil, fmt.Errorf("failed to get health status: %w", err)
	}

	return status, nil
}

// SaveHealthStatus 保存健康状态
func (s *PostgreSQLModelStore) SaveHealthStatus(ctx context.Context, modelID string, status *HealthCheckStatus) error {
	query := `INSERT INTO model_health (model_id, is_healthy, last_checked, error)
	          VALUES ($1, $2, $3, $4)`

	_, err := s.db.ExecContext(ctx, query, modelID, status.IsHealthy, status.LastChecked, status.Error)
	if err != nil {
		return fmt.Errorf("failed to save health status: %w", err)
	}

	return nil
}
