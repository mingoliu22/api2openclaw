package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openclaw/api2openclaw/internal/models"
	_ "github.com/lib/pq"
)

// PostgreSQLStore PostgreSQL 存储实现
type PostgreSQLStore struct {
	db *sql.DB
}

// NewPostgreSQLStore 创建 PostgreSQL 存储
func NewPostgreSQLStore(dataSource string) (*PostgreSQLStore, error) {
	db, err := sql.Open("postgres", dataSource)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 验证连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &PostgreSQLStore{db: db}, nil
}

// Close 关闭数据库连接
func (s *PostgreSQLStore) Close() error {
	return s.db.Close()
}

// GetTenant 获取租户
func (s *PostgreSQLStore) GetTenant(ctx context.Context, id string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, tier, requests_per_day, tokens_per_month, created_at, updated_at
		FROM tenants WHERE id = $1
	`, id).Scan(
		&tenant.ID, &tenant.Name, &tenant.Tier,
		&tenant.RequestsPerDay, &tenant.TokensPerMonth,
		&tenant.CreatedAt, &tenant.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrTenantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query tenant: %w", err)
	}
	return &tenant, nil
}

// CreateTenant 创建租户
func (s *PostgreSQLStore) CreateTenant(ctx context.Context, tenant *models.Tenant) error {
	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tenants (id, name, tier, requests_per_day, tokens_per_month)
		VALUES ($1, $2, $3, $4, $5)
	`, tenant.ID, tenant.Name, tenant.Tier,
		tenant.RequestsPerDay, tenant.TokensPerMonth)
	if err != nil {
		return fmt.Errorf("insert tenant: %w", err)
	}
	return nil
}

// ListTenants 列出租户
func (s *PostgreSQLStore) ListTenants(ctx context.Context) ([]*models.Tenant, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, tier, requests_per_day, tokens_per_month, created_at, updated_at
		FROM tenants ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		var t models.Tenant
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Tier,
			&t.RequestsPerDay, &t.TokensPerMonth,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, &t)
	}
	return tenants, rows.Err()
}

// GetAPIKey 获取 API Key
func (s *PostgreSQLStore) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	var key models.APIKey
	var permissionsJSON, modelsJSON []byte
	var expiresAt, lastUsedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, key_hash, permissions,
		       requests_per_minute, requests_per_hour, requests_per_day,
		       allowed_models, expires_at, status, created_at, last_used_at, updated_at
		FROM api_keys WHERE id = $1
	`, id).Scan(
		&key.ID, &key.TenantID, &key.KeyHash, &permissionsJSON,
		&key.RequestsPerMinute, &key.RequestsPerHour, &key.RequestsPerDay,
		&modelsJSON, &expiresAt, &key.Status,
		&key.CreatedAt, &lastUsedAt, &key.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query api key: %w", err)
	}

	// 解析 JSON 字段
	if err := json.Unmarshal(permissionsJSON, &key.Permissions); err != nil {
		return nil, fmt.Errorf("parse permissions: %w", err)
	}
	if err := json.Unmarshal(modelsJSON, &key.AllowedModels); err != nil {
		return nil, fmt.Errorf("parse allowed_models: %w", err)
	}

	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}

	return &key, nil
}

// CreateAPIKey 创建 API Key
func (s *PostgreSQLStore) CreateAPIKey(ctx context.Context, key *models.APIKey) error {
	key.CreatedAt = time.Now()
	key.UpdatedAt = time.Now()

	permissionsJSON, err := json.Marshal(key.Permissions)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}

	modelsJSON, err := json.Marshal(key.AllowedModels)
	if err != nil {
		return fmt.Errorf("marshal allowed_models: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO api_keys (id, tenant_id, key_hash, permissions,
		                     requests_per_minute, requests_per_hour, requests_per_day,
		                     allowed_models, expires_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, key.ID, key.TenantID, key.KeyHash, permissionsJSON,
		key.RequestsPerMinute, key.RequestsPerHour, key.RequestsPerDay,
		modelsJSON, key.ExpiresAt, key.Status)
	if err != nil {
		return fmt.Errorf("insert api key: %w", err)
	}
	return nil
}

// UpdateAPIKeyLastUsed 更新最后使用时间
func (s *PostgreSQLStore) UpdateAPIKeyLastUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("update last_used_at: %w", err)
	}
	return nil
}

// ListAPIKeys 列出 API Keys
func (s *PostgreSQLStore) ListAPIKeys(ctx context.Context, tenantID string) ([]*models.APIKey, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tenant_id, key_hash, permissions,
		       requests_per_minute, requests_per_hour, requests_per_day,
		       allowed_models, expires_at, status, created_at, last_used_at, updated_at
		FROM api_keys WHERE tenant_id = $1 ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query api keys: %w", err)
	}
	defer rows.Close()

	var keys []*models.APIKey
	for rows.Next() {
		var key models.APIKey
		var permissionsJSON, modelsJSON []byte
		var expiresAt, lastUsedAt sql.NullTime

		if err := rows.Scan(
			&key.ID, &key.TenantID, &key.KeyHash, &permissionsJSON,
			&key.RequestsPerMinute, &key.RequestsPerHour, &key.RequestsPerDay,
			&modelsJSON, &expiresAt, &key.Status,
			&key.CreatedAt, &lastUsedAt, &key.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}

		if err := json.Unmarshal(permissionsJSON, &key.Permissions); err != nil {
			return nil, fmt.Errorf("parse permissions: %w", err)
		}
		if err := json.Unmarshal(modelsJSON, &key.AllowedModels); err != nil {
			return nil, fmt.Errorf("parse allowed_models: %w", err)
		}

		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}

		keys = append(keys, &key)
	}
	return keys, rows.Err()
}

// DeleteAPIKey 删除 API Key（实际标记为 revoked）
func (s *PostgreSQLStore) DeleteAPIKey(ctx context.Context, id string) error {
	return s.RevokeAPIKey(ctx, id)
}

// RevokeAPIKey 吊销 API Key
func (s *PostgreSQLStore) RevokeAPIKey(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE api_keys
		SET status = 'revoked', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	return nil
}
