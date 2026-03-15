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
		SELECT id, username, password_hash, email, role, is_active, created_by,
		       last_login_at, failed_login_attempts, locked_until, created_at, updated_at
		FROM admin_users
		WHERE username = $1
	`

	err := s.db.GetContext(ctx, &user, query, username)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// FindByID 根据 ID 查找用户
func (s *PostgreSQLStore) FindByID(ctx context.Context, id string) (*AdminUser, error) {
	var user AdminUser
	query := `
		SELECT id, username, password_hash, email, role, is_active, created_by,
		       last_login_at, failed_login_attempts, locked_until, created_at, updated_at
		FROM admin_users
		WHERE id = $1
	`

	err := s.db.GetContext(ctx, &user, query, id)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// List 列出用户（支持分页）
func (s *PostgreSQLStore) List(ctx context.Context, limit, offset int) ([]*AdminUser, int64, error) {
	// 查询总数
	var total int64
	countQuery := `SELECT COUNT(*) FROM admin_users`
	if err := s.db.GetContext(ctx, &total, countQuery); err != nil {
		return nil, 0, err
	}

	// 查询数据
	var users []*AdminUser
	query := `
		SELECT id, username, password_hash, email, role, is_active, created_by,
		       last_login_at, failed_login_attempts, locked_until, created_at, updated_at
		FROM admin_users
		ORDER BY created_at DESC
	`

	if limit > 0 {
		query += ` LIMIT $1`
		if offset > 0 {
			query += ` OFFSET $2`
		}
	}

	var err error
	if limit > 0 && offset > 0 {
		err = s.db.SelectContext(ctx, &users, query, limit, offset)
	} else if limit > 0 {
		err = s.db.SelectContext(ctx, &users, query, limit)
	} else {
		err = s.db.SelectContext(ctx, &users, query)
	}

	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Create 创建用户
func (s *PostgreSQLStore) Create(ctx context.Context, user *AdminUser) error {
	query := `
		INSERT INTO admin_users (id, username, password_hash, email, role, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.ExecContext(ctx, query,
		user.ID, user.Username, user.PasswordHash, user.Email, user.Role, user.IsActive, user.CreatedBy,
	)

	return err
}

// Update 更新用户
func (s *PostgreSQLStore) Update(ctx context.Context, user *AdminUser) error {
	query := `
		UPDATE admin_users
		SET email = $2,
		    role = $3,
		    is_active = $4,
		    last_login_at = $5,
		    failed_login_attempts = $6,
		    locked_until = $7,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Role,
		user.IsActive,
		user.LastLoginAt,
		user.FailedLoginAttempts,
		user.LockedUntil,
	)

	return err
}

// Delete 删除用户
func (s *PostgreSQLStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM admin_users WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
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
