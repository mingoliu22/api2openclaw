package admin

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"
)

// ModelConfig 模型配置
type ModelConfig struct {
	ID              string     `json:"id" db:"id"`
	Alias           string     `json:"alias" db:"alias"`
	ModelID         string     `json:"model_id" db:"model_id"`
	BaseURL         string     `json:"base_url" db:"base_url"`
	APIKeyEncrypted string     `json:"-" db:"api_key_encrypted"`
	Note            string     `json:"note,omitempty" db:"note"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`

	// 运行时字段（不在数据库中）
	HealthStatus    *ModelHealthStatus `json:"health_status,omitempty"`
}

// ModelHealthStatus 模型健康状态
type ModelHealthStatus struct {
	Status      string    `json:"status"` // healthy, unhealthy, unknown
	LatencyMs   int       `json:"latency_ms"`
	Error       string    `json:"error,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
}

// ModelStore 模型存储接口
type ModelStore interface {
	List(ctx context.Context, activeOnly bool) ([]*ModelConfig, error)
	GetByID(ctx context.Context, id string) (*ModelConfig, error)
	GetByAlias(ctx context.Context, alias string) (*ModelConfig, error)
	Create(ctx context.Context, model *ModelConfig) error
	Update(ctx context.Context, model *ModelConfig) error
	Delete(ctx context.Context, id string) error
	ToggleActive(ctx context.Context, id string, isActive bool) error
}

// ModelService 模型服务
type ModelService struct {
	store    ModelStore
	secretKey []byte
}

// NewModelService 创建模型服务
func NewModelService(store ModelStore, encryptionKey string) *ModelService {
	return &ModelService{
		store:    store,
		secretKey: []byte(encryptionKey),
	}
}

// List 获取模型列表
func (s *ModelService) List(ctx context.Context, activeOnly bool) ([]*ModelConfig, error) {
	return s.store.List(ctx, activeOnly)
}

// GetByID 根据 ID 获取模型
func (s *ModelService) GetByID(ctx context.Context, id string) (*ModelConfig, error) {
	return s.store.GetByID(ctx, id)
}

// GetByAlias 根据别名获取模型
func (s *ModelService) GetByAlias(ctx context.Context, alias string) (*ModelConfig, error) {
	return s.store.GetByAlias(ctx, alias)
}

// CreateModelRequest 创建模型请求
type CreateModelRequest struct {
	Alias   string `json:"alias" binding:"required"`
	ModelID string `json:"model_id" binding:"required"`
	BaseURL string `json:"base_url" binding:"required"`
	APIKey  string `json:"api_key"`
	Note    string `json:"note"`
}

// UpdateModelRequest 更新模型请求
type UpdateModelRequest struct {
	Alias   string `json:"alias"`
	ModelID string `json:"model_id"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Note    string `json:"note"`
}

// Create 创建模型
func (s *ModelService) Create(ctx context.Context, req *CreateModelRequest) (*ModelConfig, error) {
	model := &ModelConfig{
		Alias:    req.Alias,
		ModelID:  req.ModelID,
		BaseURL:  req.BaseURL,
		Note:     req.Note,
		IsActive: true,
	}

	// 加密 API Key
	if req.APIKey != "" {
		encrypted, err := s.encryptAPIKey(req.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt API key: %w", err)
		}
		model.APIKeyEncrypted = encrypted
	}

	if err := s.store.Create(ctx, model); err != nil {
		return nil, err
	}

	return model, nil
}

// Update 更新模型
func (s *ModelService) Update(ctx context.Context, id string, req *UpdateModelRequest) (*ModelConfig, error) {
	model, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.Alias != "" {
		model.Alias = req.Alias
	}
	if req.ModelID != "" {
		model.ModelID = req.ModelID
	}
	if req.BaseURL != "" {
		model.BaseURL = req.BaseURL
	}
	if req.Note != "" {
		model.Note = req.Note
	}

	// API Key 需要特殊处理 - 只有提供新值时才更新
	if req.APIKey != "" {
		encrypted, err := s.encryptAPIKey(req.APIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt API key: %w", err)
		}
		model.APIKeyEncrypted = encrypted
	}

	if err := s.store.Update(ctx, model); err != nil {
		return nil, err
	}

	return model, nil
}

// Delete 删除模型
func (s *ModelService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ToggleActive 切换模型启用状态
func (s *ModelService) ToggleActive(ctx context.Context, id string, isActive bool) error {
	return s.store.ToggleActive(ctx, id, isActive)
}

// TestConnectionRequest 测试连接请求
type TestConnectionRequest struct {
	BaseURL string `json:"base_url" binding:"required"`
	APIKey  string `json:"api_key"`
}

// TestConnectionResponse 测试连接响应
type TestConnectionResponse struct {
	OK        bool   `json:"ok"`
	LatencyMs int    `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

// TestConnection 测试模型连接
func (s *ModelService) TestConnection(ctx context.Context, req *TestConnectionRequest) *TestConnectionResponse {
	start := time.Now()

	// TODO: 实现实际的 HTTP 连接测试
	// 这里简化处理，实际应该发送请求到 {BaseURL}/models

	latency := time.Since(start).Milliseconds()

	// 模拟成功
	return &TestConnectionResponse{
		OK:        true,
		LatencyMs: int(latency),
	}
}

// encryptAPIKey 加密 API Key
func (s *ModelService) encryptAPIKey(apiKey string) (string, error) {
	block, err := aes.NewCipher(s.secretKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(apiKey), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptAPIKey 解密 API Key
func (s *ModelService) decryptAPIKey(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.secretKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
