package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// APIKey API 密钥
type APIKey struct {
	ID          string     `json:"id" db:"id"`
	Label       string     `json:"label" db:"label"`
	KeyHash     string     `json:"-" db:"key_hash"`
	KeyPrefix   string     `json:"key_prefix" db:"key_prefix"`
	ModelAlias  *string    `json:"model_alias" db:"model_alias"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	Status      string     `json:"status" db:"status"` // active, revoked, expired
	Note        string     `json:"note,omitempty" db:"note"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`

	// 运行时字段
	KeyValue   string     `json:"key,omitempty"` // 仅在创建时返回明文
}

// APIKeyStore API Key 存储接口
type APIKeyStore interface {
	List(ctx context.Context, filter *APIKeyFilter) ([]*APIKey, error)
	GetByID(ctx context.Context, id string) (*APIKey, error)
	GetByKeyHash(ctx context.Context, keyHash string) (*APIKey, error)
	Create(ctx context.Context, key *APIKey) error
	Revoke(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, status string) error
}

// APIKeyFilter API Key 过滤器
type APIKeyFilter struct {
	Status    string
	ModelAlias *string
	Limit      int
	Offset     int
}

// APIKeyService API Key 服务
type APIKeyService struct {
	store APIKeyStore
}

// NewAPIKeyService 创建 API Key 服务
func NewAPIKeyService(store APIKeyStore) *APIKeyService {
	return &APIKeyService{
		store: store,
	}
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Label      string     `json:"label" binding:"required"`
	ModelAlias *string    `json:"model_alias"`
	ExpiresAt  *time.Time `json:"expires_at"`
	Note       string     `json:"note"`
}

// List 获取 API Key 列表
func (s *APIKeyService) List(ctx context.Context, filter *APIKeyFilter) ([]*APIKey, error) {
	if filter == nil {
		filter = &APIKeyFilter{}
	}
	if filter.Limit == 0 {
		filter.Limit = 100
	}
	return s.store.List(ctx, filter)
}

// GetByID 获取 API Key 详情
func (s *APIKeyService) GetByID(ctx context.Context, id string) (*APIKey, error) {
	return s.store.GetByID(ctx, id)
}

// Create 创建 API Key
func (s *APIKeyService) Create(ctx context.Context, req *CreateAPIKeyRequest) (*APIKey, error) {
	// 生成随机 Key
	keyValue, err := s.generateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// 计算 hash 和前缀
	keyHash, err := s.hashKey(keyValue)
	if err != nil {
		return nil, fmt.Errorf("failed to hash key: %w", err)
	}
	keyPrefix := s.getKeyPrefix(keyValue)

	apiKey := &APIKey{
		Label:      req.Label,
		KeyHash:    keyHash,
		KeyPrefix:  keyPrefix,
		ModelAlias: req.ModelAlias,
		ExpiresAt:  req.ExpiresAt,
		Status:     "active",
		Note:       req.Note,
		KeyValue:   keyValue, // 仅在内存中，不存储到数据库
	}

	if err := s.store.Create(ctx, apiKey); err != nil {
		return nil, err
	}

	return apiKey, nil
}

// Revoke 吊销 API Key
func (s *APIKeyService) Revoke(ctx context.Context, id string) error {
	return s.store.Revoke(ctx, id)
}

// Delete 删除 API Key
func (s *APIKeyService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ValidateKey 验证 API Key
func (s *APIKeyService) ValidateKey(ctx context.Context, keyValue string) (*APIKey, error) {
	keyHash, err := s.hashKey(keyValue)
	if err != nil {
		return nil, err
	}

	apiKey, err := s.store.GetByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("invalid key: %w", err)
	}

	// 检查状态
	if apiKey.Status != "active" {
		return nil, fmt.Errorf("key is %s", apiKey.Status)
	}

	// 检查是否过期
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		// 更新状态为过期
		_ = s.store.UpdateStatus(ctx, apiKey.ID, "expired")
		return nil, fmt.Errorf("key has expired")
	}

	return apiKey, nil
}

// generateKey 生成随机 API Key
func (s *APIKeyService) generateKey() (string, error) {
	// 生成 32 字节随机数据
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	// 转换为 hex 字符串
	key := hex.EncodeToString(randomBytes)

	// 添加前缀
	return fmt.Sprintf("sk-a2oc-%s", key), nil
}

// hashKey 计算 Key 的 hash
func (s *APIKeyService) hashKey(keyValue string) (string, error) {
	// 使用 SHA-256 hash
	hash, err := HashPassword(keyValue)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// getKeyPrefix 获取 Key 前缀（用于脱敏展示）
func (s *APIKeyService) getKeyPrefix(keyValue string) string {
	// 返回前 20 个字符
	if len(keyValue) > 20 {
		return keyValue[:20]
	}
	return keyValue
}

// CheckModelPermission 检查 Key 是否有权限访问指定模型
func (s *APIKeyService) CheckModelPermission(ctx context.Context, apiKey *APIKey, modelAlias string) bool {
	// 如果 Key 没有绑定模型，表示可以访问所有模型
	if apiKey.ModelAlias == nil {
		return true
	}

	// 检查是否匹配
	return *apiKey.ModelAlias == modelAlias
}
