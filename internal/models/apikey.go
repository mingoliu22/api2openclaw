package models

import "time"

// KeyStatus API Key 状态
type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusDisabled KeyStatus = "disabled"
	KeyStatusRevoked  KeyStatus = "revoked"
)

// APIKey API 密钥
type APIKey struct {
	ID                string     `json:"id" db:"id"`
	TenantID          string     `json:"tenant_id" db:"tenant_id"`
	KeyHash           string     `json:"-" db:"key_hash"`           // 哈希存储，不输出
	Permissions       []string   `json:"permissions" db:"permissions"`
	RequestsPerMinute int        `json:"requests_per_minute" db:"requests_per_minute"`
	RequestsPerHour   int        `json:"requests_per_hour" db:"requests_per_hour"`
	RequestsPerDay    int        `json:"requests_per_day" db:"requests_per_day"`
	AllowedModels     []string   `json:"allowed_models" db:"allowed_models"`
	PinnedBackends    []string   `json:"pinned_backends,omitempty" db:"pinned_backends"` // 固定后端列表（路由隔离）
	ExpiresAt         *time.Time `json:"expires_at" db:"expires_at"`
	ExpiresInDays     int        `json:"expires_in_days,omitempty" db:"-"` // 相对天数，仅用于创建
	Status            KeyStatus  `json:"status" db:"status"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	LastUsedAt        *time.Time `json:"last_used_at" db:"last_used_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// IsActive 检查 Key 是否激活
func (k *APIKey) IsActive() bool {
	if k.Status != KeyStatusActive {
		return false
	}

	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return false
	}

	return true
}

// HasPermission 检查是否有指定权限
func (k *APIKey) HasPermission(permission string) bool {
	for _, p := range k.Permissions {
		if p == "*" || p == permission {
			return true
		}
	}
	return false
}

// CanUseModel 检查是否可以使用指定模型
func (k *APIKey) CanUseModel(model string) bool {
	for _, m := range k.AllowedModels {
		if m == "*" || m == model {
			return true
		}
	}
	return false
}

// IsExpired 检查是否已过期
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false // 没有设置过期时间表示永久有效
	}
	return time.Now().After(*k.ExpiresAt)
}

// GetExpiresAt 获取过期时间，处理相对天数
func (k *APIKey) GetExpiresAt() *time.Time {
	if k.ExpiresAt != nil {
		return k.ExpiresAt
	}
	if k.ExpiresInDays > 0 {
		expiresAt := k.CreatedAt.AddDate(0, 0, k.ExpiresInDays)
		return &expiresAt
	}
	return nil // 永久有效
}

// GetTTL 获取剩余有效时间（秒），-1 表示永久有效
func (k *APIKey) GetTTL() int64 {
	expiresAt := k.GetExpiresAt()
	if expiresAt == nil {
		return -1
	}
	ttl := int64(expiresAt.Sub(time.Now()).Seconds())
	if ttl < 0 {
		return 0
	}
	return ttl
}

// HasPinnedBackends 检查是否固定了后端
func (k *APIKey) HasPinnedBackends() bool {
	return len(k.PinnedBackends) > 0
}

// GetPinnedBackends 获取固定的后端列表
func (k *APIKey) GetPinnedBackends() []string {
	return k.PinnedBackends
}

// IsPinnedToBackend 检查是否固定到指定后端
func (k *APIKey) IsPinnedToBackend(backendID string) bool {
	if !k.HasPinnedBackends() {
		return false
	}
	for _, id := range k.PinnedBackends {
		if id == backendID {
			return true
		}
	}
	return false
}

// AddPinnedBackend 添加固定后端
func (k *APIKey) AddPinnedBackend(backendID string) {
	for _, id := range k.PinnedBackends {
		if id == backendID {
			return // 已存在
		}
	}
	k.PinnedBackends = append(k.PinnedBackends, backendID)
}

// RemovePinnedBackend 移除固定后端
func (k *APIKey) RemovePinnedBackend(backendID string) {
	for i, id := range k.PinnedBackends {
		if id == backendID {
			k.PinnedBackends = append(k.PinnedBackends[:i], k.PinnedBackends[i+1:]...)
			return
		}
	}
}

// RateLimit 速率限制配置
type RateLimit struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	RequestsPerHour   int `json:"requests_per_hour"`
	RequestsPerDay    int `json:"requests_per_day"`
}
