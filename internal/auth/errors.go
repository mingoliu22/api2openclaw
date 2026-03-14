package auth

import "errors"

var (
	// ErrTenantNotFound 租户不存在
	ErrTenantNotFound = errors.New("tenant not found")

	// ErrKeyNotFound API Key 不存在
	ErrKeyNotFound = errors.New("api key not found")

	// ErrInvalidKey API Key 无效
	ErrInvalidKey = errors.New("invalid api key")

	// ErrKeyInactive API Key 未激活
	ErrKeyInactive = errors.New("api key is inactive")

	// ErrKeyExpired API Key 已过期
	ErrKeyExpired = errors.New("api key has expired")

	// ErrPermissionDenied 权限不足
	ErrPermissionDenied = errors.New("permission denied")

	// ErrModelNotAllowed 不允许使用该模型
	ErrModelNotAllowed = errors.New("model not allowed for this key")

	// ErrRateLimitExceeded 超过速率限制
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrQuotaExceeded 超过配额
	ErrQuotaExceeded = errors.New("quota exceeded")
)
