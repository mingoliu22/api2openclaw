package admin

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Handler 管理员处理器
type Handler struct {
	authService *AdminAuthService
}

// NewHandler 创建管理员处理器
func NewHandler(authService *AdminAuthService) *Handler {
	return &Handler{
		authService: authService,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Username    string    `json:"username"`
	LastLoginAt *string   `json:"last_login_at,omitempty"`
	ExpiresAt   string    `json:"expires_at"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// LoginHandler 登录处理器
func (h *Handler) LoginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}{
				Code:    "invalid_request",
				Message: "请求格式错误",
			},
		})
		return
	}

	// 获取客户端 IP
	ipAddress := c.ClientIP()
	if ipAddress == "" {
		ipAddress = c.Request.RemoteAddr
	}

	// 调用登录服务
	token, expiresAt, err := h.authService.Login(c.Request.Context(), req.Username, req.Password, ipAddress)
	if err != nil {
		code := "invalid_credentials"
		message := "用户名或密码错误"
		statusCode := http.StatusUnauthorized

		switch err {
		case ErrTooManyAttempts:
			code = "too_many_attempts"
			message = "登录尝试过多，请 15 分钟后再试"
			statusCode = http.StatusTooManyRequests
		case ErrUserLocked:
			code = "user_locked"
			message = "账号已被锁定，请稍后重试"
			statusCode = http.StatusLocked
		}

		c.JSON(statusCode, ErrorResponse{
			Error: struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}{
				Code:    code,
				Message: message,
			},
		})
		return
	}

	// 设置 HttpOnly Cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "admin_token",
		Value:    token,
		Path:     "/admin",
		MaxAge:   int(8 * time.Hour.Seconds()),
		HttpOnly: true,
		Secure:   c.Request.TLS != nil, // 生产环境需要 HTTPS
		SameSite: http.SameSiteStrictMode,
	})

	// 返回成功响应
	c.JSON(http.StatusOK, LoginResponse{
		Username:    req.Username,
		LastLoginAt: nil, // 首次登录不返回
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	})
}

// LogoutHandler 登出处理器
func (h *Handler) LogoutHandler(c *gin.Context) {
	// 清除 Cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "admin_token",
		Value:    "",
		Path:     "/admin",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

// MeHandler 获取当前用户信息
func (h *Handler) MeHandler(c *gin.Context) {
	claims, exists := c.Get("admin_claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}{
				Code:    "not_authenticated",
				Message: "未登录",
			},
		})
		return
	}

	adminClaims := claims.(*Claims)
	c.JSON(http.StatusOK, gin.H{
		"username": adminClaims.Username,
		"user_id":  adminClaims.UserID,
	})
}

// JWTMiddleware JWT 认证中间件
func JWTMiddleware(jwtManager *JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Cookie 获取 Token
		cookie, err := c.Cookie("admin_token")
		if err != nil {
			// 尝试从 Authorization Header 获取
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				cookie = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error: struct {
						Code    string `json:"code"`
						Message string `json:"message"`
					}{
						Code:    "not_authenticated",
						Message: "未登录",
					},
				})
				c.Abort()
				return
			}
		}

		// 验证 Token
		claims, err := jwtManager.ValidateToken(cookie)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				}{
					Code:    "invalid_token",
					Message: "登录已过期，请重新登录",
				},
			})
			c.Abort()
			return
		}

		// 将 claims 存入上下文
		c.Set("admin_claims", claims)
		c.Next()
	}
}

// MockAdminAuth 模拟管理员认证（用于开发测试）
func MockAdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置模拟 claims
		claims := &Claims{
			UserID:   "mock-admin-id",
			Username: "admin",
		}
		c.Set("admin_claims", claims)
		c.Next()
	}
}

// InitMockAdminDB 初始化模拟管理员数据库（用于开发测试）
func InitMockAdminDB(db *gorm.DB) error {
	// 自动迁移
	err := db.AutoMigrate(&AdminUser{}, &LoginAttempt{})
	if err != nil {
		return err
	}

	// 创建模拟管理员
	var count int64
	db.Model(&AdminUser{}).Count(&count)
	if count == 0 {
		// 生成密码 hash
		hash, err := HashPassword("admin123")
		if err != nil {
			return err
		}

		admin := &AdminUser{
			Username:     "admin",
			PasswordHash: hash,
		}
		if err := db.Create(admin).Error; err != nil {
			return err
		}
	}

	return nil
}

// InitMockAdminSQLite 初始化 SQLite 模拟数据库
func InitMockAdminSQLite(dbPath string) (*gorm.DB, error) {
	gormDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, err
	}

	if err := InitMockAdminDB(gormDB); err != nil {
		return nil, err
	}

	return gormDB, nil
}
