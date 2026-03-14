package security

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessControl 访问控制
type AccessControl struct {
	mu              sync.RWMutex
	ipWhitelist     []string
	ipBlacklist     []string
	allowedCIDRs    []*net.IPNet
	authRequired    bool
	adminTokens     map[string]bool
	rateLimitPerIP  map[string]*rateLimitEntry
	rateLimitConfig *RateLimitConfig
}

// rateLimitEntry 速率限制条目
type rateLimitEntry struct {
	requests  []time.Time
	blockUntil *time.Time
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	MaxRequestsPerMinute int
	BlockDuration        time.Duration
}

// NewAccessControl 创建访问控制
func NewAccessControl() *AccessControl {
	return &AccessControl{
		ipWhitelist:    make([]string, 0),
		ipBlacklist:    make([]string, 0),
		allowedCIDRs:   make([]*net.IPNet, 0),
		adminTokens:     make(map[string]bool),
		rateLimitPerIP:  make(map[string]*rateLimitEntry),
		rateLimitConfig: &RateLimitConfig{
			MaxRequestsPerMinute: 60,
			BlockDuration:        5 * time.Minute,
		},
	}
}

// Middleware 返回 Gin 中间件
func (ac *AccessControl) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查 IP 黑名单
		if ac.isIPBlacklisted(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Access denied: IP blocked",
			})
			return
		}

		// 检查 IP 白名单（如果启用）
		if ac.isIPWhitelistEnabled() && !ac.isIPWhitelisted(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Access denied: IP not allowed",
			})
			return
		}

		// 检查 IP 速率限制
		if ac.isRateLimited(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests from this IP",
			})
			return
		}

		c.Next()
	}
}

// AdminAuth 管理员认证中间件
func (ac *AccessControl) AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取管理员 Token
		token := c.GetHeader("X-Admin-Token")
		if token == "" {
			token = c.GetHeader("Authorization")
			if strings.HasPrefix(token, "Bearer ") {
				token = token[7:]
			}
		}

		// 验证 Token
		if !ac.isValidAdminToken(token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized: invalid admin token",
			})
			return
		}

		c.Next()
	}
}

// IPWhitelist IP 白名单中间件
func (ac *AccessControl) IPWhitelist(whitelist []string) gin.HandlerFunc {
	ac.mu.Lock()
	ac.ipWhitelist = whitelist
	ac.mu.Unlock()

	return func(c *gin.Context) {
		if !ac.isIPWhitelisted(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Access denied: IP not in whitelist",
			})
			return
		}
		c.Next()
	}
}

// CIDRRestriction CIDR 限制中间件
func (ac *AccessControl) CIDRRestriction(cidrs []string) gin.HandlerFunc {
	ac.mu.Lock()
	for _, cidr := range cidrs {
		_, parsedIPNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		ac.allowedCIDRs = append(ac.allowedCIDRs, parsedIPNet)
	}
	ac.mu.Unlock()

	return func(c *gin.Context) {
		if !ac.isIPInCIDR(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Access denied: IP not in allowed range",
			})
			return
		}
		c.Next()
	}
}

// AddAdminToken 添加管理员 Token
func (ac *AccessControl) AddAdminToken(token string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.adminTokens[token] = true
}

// RemoveAdminToken 移除管理员 Token
func (ac *AccessControl) RemoveAdminToken(token string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	delete(ac.adminTokens, token)
}

// AddToIPWhitelist 添加 IP 到白名单
func (ac *AccessControl) AddToIPWhitelist(ip string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.ipWhitelist = append(ac.ipWhitelist, ip)
}

// RemoveFromIPWhitelist 从白名单移除 IP
func (ac *AccessControl) RemoveFromIPWhitelist(ip string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	for i, existing := range ac.ipWhitelist {
		if existing == ip {
			ac.ipWhitelist = append(ac.ipWhitelist[:i], ac.ipWhitelist[i+1:]...)
			break
		}
	}
}

// AddToIPBlacklist 添加 IP 到黑名单
func (ac *AccessControl) AddToIPBlacklist(ip string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.ipBlacklist = append(ac.ipBlacklist, ip)
}

// RemoveFromIPBlacklist 从黑名单移除 IP
func (ac *AccessControl) RemoveFromIPBlacklist(ip string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	for i, existing := range ac.ipBlacklist {
		if existing == ip {
			ac.ipBlacklist = append(ac.ipBlacklist[:i], ac.ipBlacklist[i+1:]...)
			break
		}
	}
}

// isIPBlacklisted 检查 IP 是否在黑名单
func (ac *AccessControl) isIPBlacklisted(clientIP string) bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	for _, ip := range ac.ipBlacklist {
		if ip == clientIP || ip == "*" {
			return true
		}
	}
	return false
}

// isIPWhitelisted 检查 IP 是否在白名单
func (ac *AccessControl) isIPWhitelisted(clientIP string) bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	// 如果白名单为空，允许所有 IP
	if len(ac.ipWhitelist) == 0 {
		return true
	}

	for _, ip := range ac.ipWhitelist {
		if ip == clientIP || ip == "*" {
			return true
		}
	}

	// 检查 CIDR
	return ac.isIPInCIDR(clientIP)
}

// isIPWhitelistEnabled 检查是否启用白名单
func (ac *AccessControl) isIPWhitelistEnabled() bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return len(ac.ipWhitelist) > 0
}

// isIPInCIDR 检查 IP 是否在 CIDR 范围内
func (ac *AccessControl) isIPInCIDR(clientIP string) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	for _, cidr := range ac.allowedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}

	return true // 如果没有配置 CIDR 限制，允许所有
}

// isValidAdminToken 验证管理员 Token
func (ac *AccessControl) isValidAdminToken(token string) bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	return ac.adminTokens[token]
}

// isRateLimited 检查 IP 是否被限流
func (ac *AccessControl) isRateLimited(clientIP string) bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	entry, exists := ac.rateLimitPerIP[clientIP]
	if !exists {
		entry = &rateLimitEntry{
			requests: make([]time.Time, 0, ac.rateLimitConfig.MaxRequestsPerMinute),
		}
		ac.rateLimitPerIP[clientIP] = entry
	}

	now := time.Now()

	// 检查是否在阻止期
	if entry.blockUntil != nil && now.Before(*entry.blockUntil) {
		return true
	}

	// 清除阻止期
	if entry.blockUntil != nil && now.After(*entry.blockUntil) {
		entry.blockUntil = nil
	}

	// 清理过期请求记录（1分钟前）
	cutoff := now.Add(-time.Minute)
	validIdx := 0
	for _, reqTime := range entry.requests {
		if reqTime.After(cutoff) {
			entry.requests[validIdx] = reqTime
			validIdx++
		}
	}
	entry.requests = entry.requests[:validIdx]

	// 检查请求数
	if len(entry.requests) >= ac.rateLimitConfig.MaxRequestsPerMinute {
		// 触发阻止
		blockUntil := now.Add(ac.rateLimitConfig.BlockDuration)
		entry.blockUntil = &blockUntil
		return true
	}

	// 记录请求
	entry.requests = append(entry.requests, now)
	return false
}

// GenerateAdminToken 生成管理员 Token
func (ac *AccessControl) GenerateAdminToken() string {
	token := generateSecureToken()
	ac.AddAdminToken(token)
	return token
}

// generateSecureToken 生成安全 Token
func generateSecureToken() string {
	// TODO: 使用更安全的方式生成 Token
	return "admin_token_" + time.Now().Format("20060102150405")
}

// SecurityHeaders 安全响应头中间件
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止点击劫持
		c.Header("X-Frame-Options", "DENY")

		// 防止 MIME 类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// 启用 XSS 保护
		c.Header("X-XSS-Protection", "1; mode=block")

		// 严格传输安全（仅 HTTPS）
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// 内容安全策略
		c.Header("Content-Security-Policy", "default-src 'self'")

		// 隐藏服务器信息
		c.Header("Server", "api2openclaw")

		c.Next()
	}
}

// CORSMiddleware CORS 中间件
type CORSMiddleware struct {
	allowedOrigins []string
	allowedMethods []string
	allowedHeaders []string
}

// NewCORSMiddleware 创建 CORS 中间件
func NewCORSMiddleware() *CORSMiddleware {
	return &CORSMiddleware{
		allowedOrigins: []string{"*"},
		allowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		allowedHeaders: []string{"Content-Type", "Authorization"},
	}
}

// Middleware 返回 CORS 中间件
func (m *CORSMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查源
		allowed := false
		for _, allowedOrigin := range m.allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", strings.Join(m.allowedMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(m.allowedHeaders, ", "))
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SetAllowedOrigins 设置允许的源
func (m *CORSMiddleware) SetAllowedOrigins(origins []string) {
	m.allowedOrigins = origins
}

// SetAllowedMethods 设置允许的方法
func (m *CORSMiddleware) SetAllowedMethods(methods []string) {
	m.allowedMethods = methods
}

// SetAllowedHeaders 设置允许的头
func (m *CORSMiddleware) SetAllowedHeaders(headers []string) {
	m.allowedHeaders = headers
}
