package admin

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// PostgreSQLStore PostgreSQL 实现
type PostgreSQLStore struct {
	db *sqlx.DB
}

// NewPostgreSQLStore 创建 PostgreSQL 存储
func NewPostgreSQLStore(db *sqlx.DB) *PostgreSQLStore {
	return &PostgreSQLStore{db: db}
}

// FindByUsername 根据用户名查找用户
func (s *PostgreSQLStore) FindByUsername(ctx context.Context, username string) (*AdminUser, error) {
	var user AdminUser
	query := `
		SELECT id, username, password_hash, last_login_at,
	       failed_login_attempts, locked_until, created_at, updated_at
		FROM admin_users
		WHERE username = $1
	`

	err := s.db.GetContext(ctx, &user, query, username)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Update 更新用户
func (s *PostgreSQLStore) Update(ctx context.Context, user *AdminUser) error {
	query := `
		UPDATE admin_users
		SET last_login_at = $2,
		    failed_login_attempts = $3,
		    locked_until = $4,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.LastLoginAt,
		user.FailedLoginAttempts,
		user.LockedUntil,
	)

	return err
}

// RecordLoginAttempt 记录登录尝试
func (s *PostgreSQLStore) RecordLoginAttempt(ctx context.Context, attempt *LoginAttempt) error {
	query := `
		INSERT INTO login_attempts (ip_address, username, success)
		VALUES ($1, $2, $3)
	`

	_, err := s.db.ExecContext(ctx, query, attempt.IPAddress, attempt.Username, attempt.Success)
	return err
}

// CountFailedAttempts 统计失败的登录尝试
func (s *PostgreSQLStore) CountFailedAttempts(ctx context.Context, ipAddress string, since time.Time) (int64, error) {
	var count int64
	query := `
		SELECT COUNT(*)
		FROM login_attempts
		WHERE ip_address = $1
		  AND attempted_at > $2
		  AND success = false
	`

	err := s.db.GetContext(ctx, &count, query, ipAddress, since)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CleanupOldAttempts 清理旧的登录尝试记录
func (s *PostgreSQLStore) CleanupOldAttempts(ctx context.Context, before time.Time) error {
	query := `DELETE FROM login_attempts WHERE attempted_at < $1`
	_, err := s.db.ExecContext(ctx, query, before)
	return err
}

// AdminUserStoreAdmin 扩展的管理功能
type AdminUserStoreAdmin interface {
	Create(ctx context.Context, user *AdminUser) error
	List(ctx context.Context) ([]*AdminUser, error)
	Delete(ctx context.Context, id string) error
	ChangePassword(ctx context.Context, userID, newHash string) error
}

// Create 创建管理员用户
func (s *PostgreSQLStore) Create(ctx context.Context, user *AdminUser) error {
	query := `
		INSERT INTO admin_users (id, username, password_hash)
		VALUES ($1, $2, $3)
	`

	_, err := s.db.ExecContext(ctx, query, user.ID, user.Username, user.PasswordHash)
	return err
}

// List 列出所有管理员用户
func (s *PostgreSQLStore) List(ctx context.Context) ([]*AdminUser, error) {
	var users []*AdminUser
	query := `
		SELECT id, username, last_login_at, failed_login_attempts,
		       locked_until, created_at, updated_at
		FROM admin_users
		ORDER BY created_at DESC
	`

	err := s.db.SelectContext(ctx, &users, query)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// Delete 删除管理员用户
func (s *PostgreSQLStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM admin_users WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// ChangePassword 修改密码
func (s *PostgreSQLStore) ChangePassword(ctx context.Context, userID, newHash string) error {
	query := `
		UPDATE admin_users
		SET password_hash = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, userID, newHash)
	return err
}
