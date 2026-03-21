package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgreSQLAPIKeyStore PostgreSQL API Key 存储
type PostgreSQLAPIKeyStore struct {
	db *sqlx.DB
}

// NewPostgreSQLAPIKeyStore 创建 PostgreSQL API Key 存储
func NewPostgreSQLAPIKeyStore(db *sqlx.DB) APIKeyStore {
	return &PostgreSQLAPIKeyStore{db: db}
}

// List 获取 API Key 列表
func (s *PostgreSQLAPIKeyStore) List(ctx context.Context, filter *APIKeyFilter) ([]*APIKey, error) {
	query := `SELECT id, label, key_hash, key_prefix, model_alias, expires_at, status, note,
	          daily_token_soft_limit, daily_token_hard_limit, priority, created_at, revoked_at
	          FROM api_keys WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}

	if filter.ModelAlias != nil {
		query += fmt.Sprintf(" AND model_alias = $%d", argCount)
		args = append(args, *filter.ModelAlias)
		argCount++
	}

	query += " ORDER BY created_at DESC"
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
		return nil, fmt.Errorf("failed to query api keys: %w", err)
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		k := &APIKey{}
		err := rows.Scan(&k.ID, &k.Label, &k.KeyHash, &k.KeyPrefix, &k.ModelAlias, &k.ExpiresAt,
			&k.Status, &k.Note, &k.DailyTokenSoftLimit, &k.DailyTokenHardLimit, &k.Priority,
			&k.CreatedAt, &k.RevokedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan api key: %w", err)
		}
		keys = append(keys, k)
	}

	return keys, nil
}

// GetByID 获取 API Key 详情
func (s *PostgreSQLAPIKeyStore) GetByID(ctx context.Context, id string) (*APIKey, error) {
	query := `SELECT id, label, key_hash, key_prefix, model_alias, expires_at, status, note,
	          daily_token_soft_limit, daily_token_hard_limit, priority, created_at, revoked_at
	          FROM api_keys WHERE id = $1`

	k := &APIKey{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&k.ID, &k.Label, &k.KeyHash, &k.KeyPrefix, &k.ModelAlias, &k.ExpiresAt,
		&k.Status, &k.Note, &k.DailyTokenSoftLimit, &k.DailyTokenHardLimit, &k.Priority,
		&k.CreatedAt, &k.RevokedAt)
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}

	return k, nil
}

// GetByKeyHash 获取 API Key 详情
func (s *PostgreSQLAPIKeyStore) GetByKeyHash(ctx context.Context, keyHash string) (*APIKey, error) {
	query := `SELECT id, label, key_hash, key_prefix, model_alias, expires_at, status, note,
	          daily_token_soft_limit, daily_token_hard_limit, priority, created_at, revoked_at
	          FROM api_keys WHERE key_hash = $1`

	k := &APIKey{}
	err := s.db.QueryRowContext(ctx, query, keyHash).Scan(
		&k.ID, &k.Label, &k.KeyHash, &k.KeyPrefix, &k.ModelAlias, &k.ExpiresAt,
		&k.Status, &k.Note, &k.DailyTokenSoftLimit, &k.DailyTokenHardLimit, &k.Priority,
		&k.CreatedAt, &k.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}

	return k, nil
}

// Create 创建 API Key
func (s *PostgreSQLAPIKeyStore) Create(ctx context.Context, key *APIKey) error {
	key.ID = uuid.New().String()
	key.CreatedAt = time.Now()

	query := `INSERT INTO api_keys (id, label, key_hash, key_prefix, model_alias, expires_at, status, note,
	          daily_token_soft_limit, daily_token_hard_limit, priority, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := s.db.ExecContext(ctx, query,
		key.ID, key.Label, key.KeyHash, key.KeyPrefix, key.ModelAlias,
		key.ExpiresAt, key.Status, key.Note, key.DailyTokenSoftLimit, key.DailyTokenHardLimit,
		key.Priority, key.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create api key: %w", err)
	}

	return nil
}

// Revoke 吊销 API Key
func (s *PostgreSQLAPIKeyStore) Revoke(ctx context.Context, id string) error {
	now := time.Now()
	query := `UPDATE api_keys SET status = 'revoked', revoked_at = $1 WHERE id = $2`

	result, err := s.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to revoke api key: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("api key not found")
	}

	return nil
}

// Delete 删除 API Key
func (s *PostgreSQLAPIKeyStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete api key: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("api key not found")
	}

	return nil
}

// Update 更新 API Key（仅配额字段和标签、备注）
func (s *PostgreSQLAPIKeyStore) Update(ctx context.Context, id string, req *UpdateAPIKeyRequest) (*APIKey, error) {
	// 构建动态更新语句
	updates := []string{}
	args := []interface{}{}
	argCount := 1

	if req.Label != nil {
		updates = append(updates, fmt.Sprintf("label = $%d", argCount))
		args = append(args, *req.Label)
		argCount++
	}
	if req.Note != nil {
		updates = append(updates, fmt.Sprintf("note = $%d", argCount))
		args = append(args, *req.Note)
		argCount++
	}
	if req.DailyTokenSoftLimit != nil {
		updates = append(updates, fmt.Sprintf("daily_token_soft_limit = $%d", argCount))
		args = append(args, *req.DailyTokenSoftLimit)
		argCount++
	}
	if req.DailyTokenHardLimit != nil {
		updates = append(updates, fmt.Sprintf("daily_token_hard_limit = $%d", argCount))
		args = append(args, *req.DailyTokenHardLimit)
		argCount++
	}
	if req.Priority != nil {
		updates = append(updates, fmt.Sprintf("priority = $%d", argCount))
		args = append(args, *req.Priority)
		argCount++
	}

	if len(updates) == 0 {
		return s.GetByID(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE api_keys SET %s WHERE id = $%d", fmt.Join(updates, ", "), argCount)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update api key: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("api key not found")
	}

	return s.GetByID(ctx, id)
}

// UpdateStatus 更新状态
func (s *PostgreSQLAPIKeyStore) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE api_keys SET status = $1 WHERE id = $2`

	result, err := s.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update api key status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("api key not found")
	}

	return nil
}
