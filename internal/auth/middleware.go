package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/openclaw/api2openclaw/internal/models"
)

// contextKey 上下文键类型
type contextKey string

const (
	// APIKeyContextKey API Key 上下文键
	APIKeyContextKey contextKey = "api_key"
	// TenantContextKey 租户上下文键
	TenantContextKey contextKey = "tenant"
)

// Middleware 认证中间件
type Middleware struct {
	manager *Manager
}

// NewMiddleware 创建认证中间件
func NewMiddleware(manager *Manager) *Middleware {
	return &Middleware{manager: manager}
}

// GetManager 获取管理器
func (m *Middleware) GetManager() *Manager {
	return m.manager
}

// Auth 认证中间件
func (m *Middleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 提取 Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.respondError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		// 解析 Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			m.respondError(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		keyID := parts[1]

		// 验证 API Key（不验证密钥，仅验证 ID）
		key, err := m.manager.ValidateAPIKey(r.Context(), keyID, "")
		if err != nil {
			m.respondError(w, http.StatusUnauthorized, "invalid api key")
			return
		}

		// 获取租户信息
		tenant, err := m.manager.GetTenant(r.Context(), key.TenantID)
		if err != nil {
			m.respondError(w, http.StatusInternalServerError, "failed to get tenant")
			return
		}

		// 注入上下文
		ctx := r.Context()
		ctx = context.WithValue(ctx, APIKeyContextKey, key)
		ctx = context.WithValue(ctx, TenantContextKey, tenant)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth 可选认证中间件（未认证时继续，但标记为匿名）
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 尝试认证
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				keyID := parts[1]
				if key, err := m.manager.ValidateAPIKey(r.Context(), keyID, ""); err == nil {
					ctx = context.WithValue(ctx, APIKeyContextKey, key)
					if tenant, err := m.manager.GetTenant(r.Context(), key.TenantID); err == nil {
						ctx = context.WithValue(ctx, TenantContextKey, tenant)
					}
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission 权限检查中间件
func (m *Middleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := GetAPIKey(r)
			if key == nil {
				m.respondError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			if err := m.manager.CheckPermission(key, permission); err != nil {
				m.respondError(w, http.StatusForbidden, "permission denied")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// respondError 响应错误
func (m *Middleware) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":{"message":"` + message + `","type":"auth_error"}}`))
}

// --- 辅助函数 ---

// GetAPIKey 从请求中获取 API Key
func GetAPIKey(r *http.Request) *models.APIKey {
	key, _ := r.Context().Value(APIKeyContextKey).(*models.APIKey)
	return key
}

// GetTenant 从请求中获取租户
func GetTenant(r *http.Request) *models.Tenant {
	tenant, _ := r.Context().Value(TenantContextKey).(*models.Tenant)
	return tenant
}

// GetAPIKeyID 获取 API Key ID
func GetAPIKeyID(r *http.Request) string {
	if key := GetAPIKey(r); key != nil {
		return key.ID
	}
	return "anonymous"
}

// GetTenantID 获取租户 ID
func GetTenantID(r *http.Request) string {
	if tenant := GetTenant(r); tenant != nil {
		return tenant.ID
	}
	return "anonymous"
}
