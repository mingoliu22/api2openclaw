package audit

import (
	"context"
	"encoding/json"
	"time"
)

// Log 审计日志条目
type Log struct {
	ID           int64              `json:"id" db:"id"`
	TenantID     string             `json:"tenant_id" db:"tenant_id"`
	APIKeyID     string             `json:"api_key_id" db:"api_key_id"`
	ActorID      string             `json:"actor_id,omitempty" db:"actor_id"`
	ActorType    string             `json:"actor_type,omitempty" db:"actor_type"`
	Action       string             `json:"action" db:"action"`
	ResourceType string             `json:"resource_type" db:"resource_type"`
	ResourceID   string             `json:"resource_id,omitempty" db:"resource_id"`
	Details      map[string]any     `json:"details,omitempty" db:"details"`
	IPAddress    string             `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent    string             `json:"user_agent,omitempty" db:"user_agent"`
	Status       string             `json:"status" db:"status"`
	ErrorCode    string             `json:"error_code,omitempty" db:"error_code"`
	ErrorMessage string             `json:"error_message,omitempty" db:"error_message"`
	CreatedAt    time.Time          `json:"created_at" db:"created_at"`
}

// ActorType 操作者类型
const (
	ActorTypeAPIKey   = "api_key"
	ActorTypeAdmin    = "admin_user"
	ActorTypeSystem   = "system"
	ActorTypeService  = "service"
)

// Action 操作类型
const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionRevoke = "revoke"
	ActionList   = "list"
)

// ResourceType 资源类型
const (
	ResourceTypeAPIKey  = "api_key"
	ResourceTypeTenant  = "tenant"
	ResourceTypeBackend = "backend"
	ResourceTypeModel   = "model"
	ResourceTypeConfig  = "config"
)

// Status 状态
const (
	StatusSuccess = "success"
	StatusFailure = "failure"
)

// Filter 查询过滤器
type Filter struct {
	TenantID     string     `json:"tenant_id,omitempty"`
	APIKeyID     string     `json:"api_key_id,omitempty"`
	ActorID      string     `json:"actor_id,omitempty"`
	Action       string     `json:"action,omitempty"`
	ResourceType string     `json:"resource_type,omitempty"`
	ResourceID   string     `json:"resource_id,omitempty"`
	Status       string     `json:"status,omitempty"`
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	Limit        int        `json:"limit,omitempty"`
	Offset       int        `json:"offset,omitempty"`
}

// Store 审计日志存储接口
type Store interface {
	// Log 记录审计日志
	Log(ctx context.Context, entry *Log) error

	// Query 查询审计日志
	Query(ctx context.Context, filter *Filter) ([]*Log, error)

	// Count 统计审计日志数量
	Count(ctx context.Context, filter *Filter) (int64, error)

	// GetByID 根据 ID 获取日志
	GetByID(ctx context.Context, id int64) (*Log, error)

	// DeleteOld 删除旧日志（用于定期清理）
	DeleteOld(ctx context.Context, before time.Time) (int64, error)
}

// Logger 审计日志记录器
type Logger struct {
	store Store
}

// NewLogger 创建审计日志记录器
func NewLogger(store Store) *Logger {
	return &Logger{store: store}
}

// Log 记录审计日志
func (l *Logger) Log(ctx context.Context, entry *Log) error {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	return l.store.Log(ctx, entry)
}

// LogAction 记录操作
func (l *Logger) LogAction(ctx context.Context, action, resourceType, resourceID, tenantID, apiKeyID string, details map[string]any) error {
	entry := &Log{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		TenantID:     tenantID,
		APIKeyID:     apiKeyID,
		Details:      details,
		Status:       StatusSuccess,
	}
	return l.Log(ctx, entry)
}

// LogActionWithError 记录带错误操作
func (l *Logger) LogActionWithError(ctx context.Context, action, resourceType, resourceID, tenantID, apiKeyID string, err error, details map[string]any) error {
	entry := &Log{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		TenantID:     tenantID,
		APIKeyID:     apiKeyID,
		Details:      details,
		Status:       StatusFailure,
	}
	if err != nil {
		entry.ErrorMessage = err.Error()
	}
	return l.Log(ctx, entry)
}

// Query 查询审计日志
func (l *Logger) Query(ctx context.Context, filter *Filter) ([]*Log, error) {
	if filter == nil {
		filter = &Filter{}
	}
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}
	return l.store.Query(ctx, filter)
}

// Count 统计日志数量
func (l *Logger) Count(ctx context.Context, filter *Filter) (int64, error) {
	if filter == nil {
		filter = &Filter{}
	}
	return l.store.Count(ctx, filter)
}

// GetByID 根据 ID 获取日志
func (l *Logger) GetByID(ctx context.Context, id int64) (*Log, error) {
	return l.store.GetByID(ctx, id)
}

// EntryBuilder 审计日志条目构建器
type EntryBuilder struct {
	entry *Log
}

// NewEntryBuilder 创建审计日志构建器
func NewEntryBuilder() *EntryBuilder {
	return &EntryBuilder{
		entry: &Log{
			Status: StatusSuccess,
		},
	}
}

// WithTenant 设置租户
func (b *EntryBuilder) WithTenant(tenantID string) *EntryBuilder {
	b.entry.TenantID = tenantID
	return b
}

// WithAPIKey 设置 API Key
func (b *EntryBuilder) WithAPIKey(apiKeyID string) *EntryBuilder {
	b.entry.APIKeyID = apiKeyID
	return b
}

// WithActor 设置操作者
func (b *EntryBuilder) WithActor(actorID, actorType string) *EntryBuilder {
	b.entry.ActorID = actorID
	b.entry.ActorType = actorType
	return b
}

// WithAction 设置操作
func (b *EntryBuilder) WithAction(action string) *EntryBuilder {
	b.entry.Action = action
	return b
}

// WithResource 设置资源
func (b *EntryBuilder) WithResource(resourceType, resourceID string) *EntryBuilder {
	b.entry.ResourceType = resourceType
	b.entry.ResourceID = resourceID
	return b
}

// WithDetails 设置详细信息
func (b *EntryBuilder) WithDetails(details map[string]any) *EntryBuilder {
	b.entry.Details = details
	return b
}

// WithRequest 设置请求信息
func (b *EntryBuilder) WithRequest(ip, userAgent string) *EntryBuilder {
	b.entry.IPAddress = ip
	b.entry.UserAgent = userAgent
	return b
}

// WithError 设置错误
func (b *EntryBuilder) WithError(code, message string) *EntryBuilder {
	b.entry.Status = StatusFailure
	b.entry.ErrorCode = code
	b.entry.ErrorMessage = message
	return b
}

// Build 构建日志条目
func (b *EntryBuilder) Build() *Log {
	if b.entry.CreatedAt.IsZero() {
		b.entry.CreatedAt = time.Now()
	}
	return b.entry
}

// MarshalJSON 实现 JSON 序列化
func (l *Log) MarshalJSON() ([]byte, error) {
	type Alias Log
	return json.Marshal(&struct {
		*Alias
		CreatedAt string `json:"created_at"`
	}{
		Alias:     (*Alias)(l),
		CreatedAt: l.CreatedAt.Format(time.RFC3339),
	})
}
