package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/openclaw/api2openclaw/internal/models"
)

// Manager 认证管理器
type Manager struct {
	store Store
}

// Store 存储接口
type Store interface {
	GetTenant(ctx context.Context, id string) (*models.Tenant, error)
	CreateTenant(ctx context.Context, tenant *models.Tenant) error
	ListTenants(ctx context.Context) ([]*models.Tenant, error)

	GetAPIKey(ctx context.Context, id string) (*models.APIKey, error)
	CreateAPIKey(ctx context.Context, key *models.APIKey) error
	UpdateAPIKeyLastUsed(ctx context.Context, id string) error
	ListAPIKeys(ctx context.Context, tenantID string) ([]*models.APIKey, error)
	DeleteAPIKey(ctx context.Context, id string) error
	RevokeAPIKey(ctx context.Context, id string) error
}

// NewManager 创建认证管理器
func NewManager(store Store) *Manager {
	return &Manager{store: store}
}

// GenerateAPIKey 生成新的 API Key
func (m *Manager) GenerateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	// 验证租户存在
	tenant, err := m.store.GetTenant(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}

	// 生成 key ID 和密钥
	keyID, secret, err := generateKeyID()
	if err != nil {
		return nil, fmt.Errorf("generate key id: %w", err)
	}

	// 哈希密钥
	keyHash, err := hashSecret(secret)
	if err != nil {
		return nil, fmt.Errorf("hash secret: %w", err)
	}

	// 默认值
	if len(req.Permissions) == 0 {
		req.Permissions = []string{"*"}
	}
	if len(req.AllowedModels) == 0 {
		req.AllowedModels = []string{"*"}
	}
	if req.RateLimit == nil {
		req.RateLimit = &models.RateLimit{
			RequestsPerMinute: 100,
			RequestsPerHour:   1000,
			RequestsPerDay:    10000,
		}
	}

	// 创建 API Key
	now := time.Now()
	key := &models.APIKey{
		ID:                keyID,
		TenantID:          req.TenantID,
		KeyHash:           keyHash,
		Permissions:       req.Permissions,
		RequestsPerMinute: req.RateLimit.RequestsPerMinute,
		RequestsPerHour:   req.RateLimit.RequestsPerHour,
		RequestsPerDay:    req.RateLimit.RequestsPerDay,
		AllowedModels:     req.AllowedModels,
		PinnedBackends:    req.PinnedBackends,
		ExpiresAt:         req.ExpiresAt,
		ExpiresInDays:     req.ExpiresInDays,
		Status:            models.KeyStatusActive,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// 如果设置了相对天数，计算绝对过期时间
	if req.ExpiresInDays > 0 && req.ExpiresAt == nil {
		expiresAt := now.AddDate(0, 0, req.ExpiresInDays)
		key.ExpiresAt = &expiresAt
	}

	if err := m.store.CreateAPIKey(ctx, key); err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	return &CreateAPIKeyResponse{
		KeyID:       keyID,
		KeySecret:   secret,
		TenantID:    req.TenantID,
		TenantName:  tenant.Name,
		Permissions: req.Permissions,
		ExpiresAt:   req.ExpiresAt,
	}, nil
}

// ValidateAPIKey 验证 API Key
func (m *Manager) ValidateAPIKey(ctx context.Context, keyID, keySecret string) (*models.APIKey, error) {
	// 获取 API Key
	key, err := m.store.GetAPIKey(ctx, keyID)
	if err != nil {
		if err == ErrKeyNotFound {
			return nil, ErrInvalidKey
		}
		return nil, err
	}

	// 检查状态
	if !key.IsActive() {
		if key.Status != models.KeyStatusActive {
			return nil, ErrKeyInactive
		}
		return nil, ErrKeyExpired
	}

	// 验证密钥（如果提供了明文密钥）
	if keySecret != "" {
		if err := verifySecret(keySecret, key.KeyHash); err != nil {
			return nil, ErrInvalidKey
		}
	}

	// 更新最后使用时间
	_ = m.store.UpdateAPIKeyLastUsed(ctx, keyID)

	return key, nil
}

// CheckPermission 检查权限
func (m *Manager) CheckPermission(key *models.APIKey, permission string) error {
	if !key.HasPermission(permission) {
		return ErrPermissionDenied
	}
	return nil
}

// CheckModel 检查模型权限
func (m *Manager) CheckModel(key *models.APIKey, model string) error {
	if !key.CanUseModel(model) {
		return ErrModelNotAllowed
	}
	return nil
}

// GetTenant 获取租户信息
func (m *Manager) GetTenant(ctx context.Context, id string) (*models.Tenant, error) {
	return m.store.GetTenant(ctx, id)
}

// CreateTenant 创建租户
func (m *Manager) CreateTenant(ctx context.Context, req *CreateTenantRequest) (*models.Tenant, error) {
	// 生成租户 ID
	tenantID := generateTenantID(req.Name)

	// 默认值
	if req.Quota == nil {
		req.Quota = &models.Quota{
			RequestsPerDay: 1000,
			TokensPerMonth: 1000000,
		}
	}

	tenant := &models.Tenant{
		ID:             tenantID,
		Name:           req.Name,
		Tier:           req.Tier,
		RequestsPerDay: req.Quota.RequestsPerDay,
		TokensPerMonth: req.Quota.TokensPerMonth,
	}

	if err := m.store.CreateTenant(ctx, tenant); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	return tenant, nil
}

// ListTenants 列出租户
func (m *Manager) ListTenants(ctx context.Context) ([]*models.Tenant, error) {
	return m.store.ListTenants(ctx)
}

// ListAPIKeys 列出 API Keys
func (m *Manager) ListAPIKeys(ctx context.Context, tenantID string) ([]*models.APIKey, error) {
	return m.store.ListAPIKeys(ctx, tenantID)
}

// DeleteAPIKey 删除 API Key（实际为吊销）
func (m *Manager) DeleteAPIKey(ctx context.Context, keyID string) error {
	return m.RevokeAPIKey(ctx, keyID)
}

// RevokeAPIKey 吊销 API Key
func (m *Manager) RevokeAPIKey(ctx context.Context, keyID string) error {
	return m.store.RevokeAPIKey(ctx, keyID)
}

// --- 请求/响应结构 ---

// CreateTenantRequest 创建租户请求
type CreateTenantRequest struct {
	Name string       `json:"name"`
	Tier models.TenantTier `json:"tier"`
	Quota *models.Quota `json:"quota,omitempty"`
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	TenantID      string            `json:"tenant_id"`
	Permissions   []string          `json:"permissions,omitempty"`
	AllowedModels []string          `json:"allowed_models,omitempty"`
	PinnedBackends []string         `json:"pinned_backends,omitempty"` // 固定后端列表（路由隔离）
	RateLimit     *models.RateLimit `json:"rate_limit,omitempty"`
	ExpiresAt     *time.Time        `json:"expires_at,omitempty"`      // 绝对过期时间
	ExpiresInDays int               `json:"expires_in_days,omitempty"` // 相对天数（0 表示永久）
}

// CreateAPIKeyResponse 创建 API Key 响应
type CreateAPIKeyResponse struct {
	KeyID       string     `json:"key_id"`
	KeySecret   string     `json:"key_secret"`
	TenantID    string     `json:"tenant_id"`
	TenantName  string     `json:"tenant_name"`
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// --- 工具函数 ---

// generateKeyID 生成 API Key ID 和密钥
func generateKeyID() (keyID, secret string, err error) {
	// 生成随机字节
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}

	// key_id: sk-xxx
	keyID = "sk-" + base64.URLEncoding.EncodeToString(b[:12])
	secret = base64.URLEncoding.EncodeToString(b[12:])

	return keyID, secret, nil
}

// generateTenantID 生成租户 ID
func generateTenantID(name string) string {
	// 使用时间戳 + 名称哈希生成唯一 ID
	h := sha256.New()
	h.Write([]byte(name + time.Now().String()))
	return "tenant_" + base64.URLEncoding.EncodeToString(h.Sum(nil))[:16]
}

// hashSecret 哈希密钥
func hashSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// verifySecret 验证密钥
func verifySecret(secret, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret))
}
