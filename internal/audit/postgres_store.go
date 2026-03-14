package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// PostgreSQLStore PostgreSQL 审计日志存储
type PostgreSQLStore struct {
	db *sqlx.DB
}

// NewPostgreSQLStore 创建 PostgreSQL 存储
func NewPostgreSQLStore(dsn string) (*PostgreSQLStore, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	// 设置连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgreSQLStore{db: db}, nil
}

// Log 记录审计日志
func (s *PostgreSQLStore) Log(ctx context.Context, entry *Log) error {
	query := `
		INSERT INTO audit_logs (
			tenant_id, api_key_id, actor_id, actor_type,
			action, resource_type, resource_id, details,
			ip_address, user_agent, status, error_code, error_message
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`

	var detailsJSON []byte
	if entry.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(entry.Details)
		if err != nil {
			return fmt.Errorf("marshal details: %w", err)
		}
	}

	_, err := s.db.ExecContext(ctx, query,
		entry.TenantID,
		entry.APIKeyID,
		nullString(entry.ActorID),
		nullString(entry.ActorType),
		entry.Action,
		entry.ResourceType,
		nullString(entry.ResourceID),
		detailsJSON,
		nullString(entry.IPAddress),
		nullString(entry.UserAgent),
		entry.Status,
		nullString(entry.ErrorCode),
		nullString(entry.ErrorMessage),
	)

	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return nil
}

// Query 查询审计日志
func (s *PostgreSQLStore) Query(ctx context.Context, filter *Filter) ([]*Log, error) {
	query := `SELECT id, tenant_id, api_key_id, actor_id, actor_type,
		action, resource_type, resource_id, details, ip_address, user_agent,
		status, error_code, error_message, created_at
		FROM audit_logs`

	where, args := s.buildWhereClause(filter)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	query += " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*Log
	for rows.Next() {
		log := &Log{}
		var detailsJSON []byte
		var actorID, actorType, resourceID, ipAddress, userAgent, errorCode, errorMessage sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.TenantID,
			&log.APIKeyID,
			&actorID,
			&actorType,
			&log.Action,
			&log.ResourceType,
			&resourceID,
			&detailsJSON,
			&ipAddress,
			&userAgent,
			&log.Status,
			&errorCode,
			&errorMessage,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}

		if actorID.Valid {
			log.ActorID = actorID.String
		}
		if actorType.Valid {
			log.ActorType = actorType.String
		}
		if resourceID.Valid {
			log.ResourceID = resourceID.String
		}
		if ipAddress.Valid {
			log.IPAddress = ipAddress.String
		}
		if userAgent.Valid {
			log.UserAgent = userAgent.String
		}
		if errorCode.Valid {
			log.ErrorCode = errorCode.String
		}
		if errorMessage.Valid {
			log.ErrorMessage = errorMessage.String
		}
		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
				return nil, fmt.Errorf("unmarshal details: %w", err)
			}
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

// Count 统计审计日志数量
func (s *PostgreSQLStore) Count(ctx context.Context, filter *Filter) (int64, error) {
	query := `SELECT COUNT(*) FROM audit_logs`

	where, args := s.buildWhereClause(filter)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	var count int64
	err := s.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("count audit logs: %w", err)
	}

	return count, nil
}

// GetByID 根据 ID 获取日志
func (s *PostgreSQLStore) GetByID(ctx context.Context, id int64) (*Log, error) {
	query := `SELECT id, tenant_id, api_key_id, actor_id, actor_type,
		action, resource_type, resource_id, details, ip_address, user_agent,
		status, error_code, error_message, created_at
		FROM audit_logs WHERE id = $1`

	log := &Log{}
	var detailsJSON []byte
	var actorID, actorType, resourceID, ipAddress, userAgent, errorCode, errorMessage sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&log.ID,
		&log.TenantID,
		&log.APIKeyID,
		&actorID,
		&actorType,
		&log.Action,
		&log.ResourceType,
		&resourceID,
		&detailsJSON,
		&ipAddress,
		&userAgent,
		&log.Status,
		&errorCode,
		&errorMessage,
		&log.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("audit log not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}

	if actorID.Valid {
		log.ActorID = actorID.String
	}
	if actorType.Valid {
		log.ActorType = actorType.String
	}
	if resourceID.Valid {
		log.ResourceID = resourceID.String
	}
	if ipAddress.Valid {
		log.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		log.UserAgent = userAgent.String
	}
	if errorCode.Valid {
		log.ErrorCode = errorCode.String
	}
	if errorMessage.Valid {
		log.ErrorMessage = errorMessage.String
	}
	if len(detailsJSON) > 0 {
		if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
			return nil, fmt.Errorf("unmarshal details: %w", err)
		}
	}

	return log, nil
}

// DeleteOld 删除旧日志
func (s *PostgreSQLStore) DeleteOld(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM audit_logs WHERE created_at < $1`

	result, err := s.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("delete old audit logs: %w", err)
	}

	return result.RowsAffected()
}

// buildWhereClause 构建查询条件
func (s *PostgreSQLStore) buildWhereClause(filter *Filter) ([]string, []interface{}) {
	var where []string
	var args []interface{}
	argNum := 1

	if filter.TenantID != "" {
		where = append(where, fmt.Sprintf("tenant_id = $%d", argNum))
		args = append(args, filter.TenantID)
		argNum++
	}

	if filter.APIKeyID != "" {
		where = append(where, fmt.Sprintf("api_key_id = $%d", argNum))
		args = append(args, filter.APIKeyID)
		argNum++
	}

	if filter.ActorID != "" {
		where = append(where, fmt.Sprintf("actor_id = $%d", argNum))
		args = append(args, filter.ActorID)
		argNum++
	}

	if filter.Action != "" {
		where = append(where, fmt.Sprintf("action = $%d", argNum))
		args = append(args, filter.Action)
		argNum++
	}

	if filter.ResourceType != "" {
		where = append(where, fmt.Sprintf("resource_type = $%d", argNum))
		args = append(args, filter.ResourceType)
		argNum++
	}

	if filter.ResourceID != "" {
		where = append(where, fmt.Sprintf("resource_id = $%d", argNum))
		args = append(args, filter.ResourceID)
		argNum++
	}

	if filter.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argNum))
		args = append(args, filter.Status)
		argNum++
	}

	if filter.StartTime != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", argNum))
		args = append(args, filter.StartTime)
		argNum++
	}

	if filter.EndTime != nil {
		where = append(where, fmt.Sprintf("created_at <= $%d", argNum))
		args = append(args, filter.EndTime)
		argNum++
	}

	return where, args
}

// Close 关闭数据库连接
func (s *PostgreSQLStore) Close() error {
	return s.db.Close()
}

// nullString 返回 sql.NullString
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
