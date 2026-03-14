package models

import "time"

// BackendStatus 后端状态
type BackendStatus string

const (
	BackendStatusHealthy   BackendStatus = "healthy"
	BackendStatusUnhealthy BackendStatus = "unhealthy"
	BackendStatusDraining  BackendStatus = "draining"
)

// Backend 模型后端实例
type Backend struct {
	ID              string              `json:"id" db:"id"`
	Name            string              `json:"name" db:"name"`
	Type            string              `json:"type" db:"type"`
	BaseURL         string              `json:"base_url" db:"base_url"`
	APIKey          string              `json:"api_key,omitempty" db:"api_key"`
	Headers         map[string]string   `json:"headers,omitempty" db:"headers"`
	Weight          int                 `json:"weight" db:"weight"`
	HealthCheck     HealthCheckConfig   `json:"health_check" db:"health_check"`
	Status          BackendStatus       `json:"status" db:"status"`
	LastCheckAt     *time.Time          `json:"last_check_at" db:"last_check_at"`
	CreatedAt       time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at" db:"updated_at"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Enabled  bool          `json:"enabled" db:"enabled"`
	Interval time.Duration `json:"interval" db:"interval"`
	Endpoint string        `json:"endpoint" db:"endpoint"`
	Timeout  time.Duration `json:"timeout" db:"timeout"`
}

// IsHealthy 检查后端是否健康
func (b *Backend) IsHealthy() bool {
	return b.Status == BackendStatusHealthy
}
