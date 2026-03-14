package models

import "time"

// TenantTier 租户级别
type TenantTier string

const (
	TierFree      TenantTier = "free"
	TierPro       TenantTier = "pro"
	TierEnterprise TenantTier = "enterprise"
)

// Tenant 租户
type Tenant struct {
	ID             string     `json:"id" db:"id"`
	Name           string     `json:"name" db:"name"`
	Tier           TenantTier `json:"tier" db:"tier"`
	RequestsPerDay int64      `json:"requests_per_day" db:"requests_per_day"`
	TokensPerMonth int64      `json:"tokens_per_month" db:"tokens_per_month"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// Quota 配额
type Quota struct {
	RequestsPerDay int64 `json:"requests_per_day"`
	TokensPerMonth int64 `json:"tokens_per_month"`
}

// CheckQuota 检查配额是否充足
func (t *Tenant) CheckQuota(requestsUsed int64) bool {
	return requestsUsed < t.RequestsPerDay
}
