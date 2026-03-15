package admin

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid_credentials")
	ErrTooManyAttempts   = errors.New("too_many_attempts")
	ErrUserLocked        = errors.New("user_locked")
	ErrInvalidToken      = errors.New("invalid_token")
	ErrUserNotFound      = errors.New("user_not_found")
	ErrUserExists        = errors.New("user_already_exists")
	ErrInvalidRole       = errors.New("invalid_role")
	ErrInsufficientPermission = errors.New("insufficient_permission")
)

// UserRole 用户角色
type UserRole string

const (
	RoleSuperAdmin UserRole = "super_admin" // 超级管理员：所有权限
	RoleAdmin      UserRole = "admin"       // 管理员：管理模型、API Key、查看日志
	RoleOperator   UserRole = "operator"    // 操作员：查看状态和日志
	RoleViewer     UserRole = "viewer"      // 查看者：只读访问
)

// ValidRoles 有效角色列表
var ValidRoles = map[UserRole]bool{
	RoleSuperAdmin: true,
	RoleAdmin:      true,
	RoleOperator:   true,
	RoleViewer:     true,
}

// IsValidRole 检查角色是否有效
func IsValidRole(role UserRole) bool {
	return ValidRoles[role]
}

// RolePermissions 角色权限定义
var RolePermissions = map[UserRole][]string{
	RoleSuperAdmin: {"*"}, // 所有权限
	RoleAdmin:      {
		"models.read", "models.write", "models.delete",
		"keys.read", "keys.write", "keys.delete",
		"logs.read", "logs.export",
		"users.read", "users.write",
		"stats.read",
		"plugins.read", "plugins.write",
		"billing.read", "billing.write",
	},
	RoleOperator: {
		"models.read",
		"keys.read",
		"logs.read", "logs.export",
		"stats.read",
		"billing.read",
	},
	RoleViewer: {
		"models.read",
		"logs.read",
		"stats.read",
		"billing.read",
	},
}

// HasPermission 检查角色是否有指定权限
func (r UserRole) HasPermission(permission string) bool {
	permissions, ok := RolePermissions[r]
	if !ok {
		return false
	}

	for _, p := range permissions {
		if p == "*" || p == permission {
			return true
		}
	}
	return false
}

// AdminUser 管理员用户
type AdminUser struct {
	ID                 string     `json:"id" db:"id"`
	Username           string     `json:"username" db:"username"`
	PasswordHash       string     `json:"-" db:"password_hash"`
	Email              string     `json:"email,omitempty" db:"email"`
	Role               UserRole   `json:"role" db:"role"`
	IsActive           bool       `json:"is_active" db:"is_active"`
	CreatedBy          *string    `json:"created_by,omitempty" db:"created_by"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	FailedLoginAttempts int        `json:"failed_login_attempts" db:"failed_login_attempts"`
	LockedUntil        *time.Time `json:"locked_until,omitempty" db:"locked_until"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// HasPermission 检查用户是否有指定权限
func (u *AdminUser) HasPermission(permission string) bool {
	if !u.IsActive {
		return false
	}
	return u.Role.HasPermission(permission)
}

// CanManageUser 检查是否可以管理指定用户
func (u *AdminUser) CanManageUser(targetRole UserRole) bool {
	// super_admin 可以管理所有人
	if u.Role == RoleSuperAdmin {
		return true
	}
	// admin 不能管理 super_admin 或其他 admin
	if u.Role == RoleAdmin {
		return targetRole != RoleSuperAdmin && targetRole != RoleAdmin
	}
	// 其他角色不能管理用户
	return false
}

// CanEditModel 检查是否可以编辑模型
func (u *AdminUser) CanEditModel() bool {
	return u.IsActive && (u.Role == RoleSuperAdmin || u.Role == RoleAdmin)
}

// CanDeleteModel 检查是否可以删除模型
func (u *AdminUser) CanDeleteModel() bool {
	return u.IsActive && (u.Role == RoleSuperAdmin || u.Role == RoleAdmin)
}

// CanManageKeys 检查是否可以管理 API Keys
func (u *AdminUser) CanManageKeys() bool {
	return u.IsActive && (u.Role == RoleSuperAdmin || u.Role == RoleAdmin)
}

// CanViewLogs 检查是否可以查看日志
func (u *AdminUser) CanViewLogs() bool {
	return u.IsActive && u.HasPermission("logs.read")
}

// CanExportLogs 检查是否可以导出日志
func (u *AdminUser) CanExportLogs() bool {
	return u.IsActive && u.HasPermission("logs.export")
}

// IsLocked 检查用户是否被锁定
func (u *AdminUser) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// LoginAttempt 登录尝试记录
type LoginAttempt struct {
	ID          int64      `json:"-" db:"id"`
	IPAddress  string     `json:"ip_address" db:"ip_address"`
	Username   string     `json:"username,omitempty" db:"username"`
	Success    bool       `json:"success" db:"success"`
	AttemptedAt time.Time `json:"attempted_at" db:"attempted_at"`
}

// JWT 管理器
type JWTManager struct {
	secretKey       []byte
	issuer          string
	tokenExpiration time.Duration
}

// NewJWTManager 创建 JWT 管理器
func NewJWTManager(secret string, expiration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:       []byte(secret),
		issuer:          "api2openclaw",
		tokenExpiration: expiration,
	}
}

// Claims JWT 声明
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Role     UserRole `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token
func (j *JWTManager) GenerateToken(user *AdminUser) (string, time.Time, error) {
	expiresAt := time.Now().Add(j.tokenExpiration)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// ValidateToken 验证 JWT Token
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// AdminAuthService 管理员认证服务
type AdminAuthService struct {
	store           AdminUserStore
	jwtManager      *JWTManager
	lockoutDuration time.Duration
	maxAttempts     int
	lockoutWindow   time.Duration
}

// AdminUserStore 管理员用户存储接口
type AdminUserStore interface {
	FindByUsername(ctx context.Context, username string) (*AdminUser, error)
	FindByID(ctx context.Context, id string) (*AdminUser, error)
	List(ctx context.Context, limit, offset int) ([]*AdminUser, int64, error)
	Create(ctx context.Context, user *AdminUser) error
	Update(ctx context.Context, user *AdminUser) error
	Delete(ctx context.Context, id string) error
	RecordLoginAttempt(ctx context.Context, attempt *LoginAttempt) error
	CountFailedAttempts(ctx context.Context, ipAddress string, since time.Time) (int64, error)
	CleanupOldAttempts(ctx context.Context, before time.Time) error
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string   `json:"username" binding:"required,min=3,max=64"`
	Password string   `json:"password" binding:"required,min=8"`
	Email    string   `json:"email" binding:"omitempty,email"`
	Role     UserRole `json:"role" binding:"required"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email    *string  `json:"email" binding:"omitempty,email"`
	Role     *UserRole `json:"role"`
	Password string   `json:"password" binding:"omitempty,min=8"`
	IsActive *bool    `json:"is_active"`
}

// NewAdminAuthService 创建管理员认证服务
func NewAdminAuthService(store AdminUserStore, jwtSecret string) *AdminAuthService {
	return &AdminAuthService{
		store:           store,
		jwtManager:      NewJWTManager(jwtSecret, 8*time.Hour),
		lockoutDuration: 15 * time.Minute,
		maxAttempts:     5,
		lockoutWindow:   15 * time.Minute,
	}
}

// Login 登录
func (s *AdminAuthService) Login(ctx context.Context, username, password, ipAddress string) (string, time.Time, error) {
	// 检查 IP 是否被锁定
	failedCount, err := s.store.CountFailedAttempts(ctx, ipAddress, time.Now().Add(-s.lockoutWindow))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to check login attempts: %w", err)
	}

	if failedCount >= int64(s.maxAttempts) {
		// 记录本次失败尝试
		s.store.RecordLoginAttempt(ctx, &LoginAttempt{
			IPAddress: ipAddress,
			Username:  username,
			Success:   false,
		})
		return "", time.Time{}, ErrTooManyAttempts
	}

	// 查找用户
	user, err := s.store.FindByUsername(ctx, username)
	if err != nil {
		// 记录失败尝试
		s.store.RecordLoginAttempt(ctx, &LoginAttempt{
			IPAddress: ipAddress,
			Username:  username,
			Success:   false,
		})
		return "", time.Time{}, ErrInvalidCredentials
	}

	// 检查用户是否被锁定
	if user.IsLocked() {
		return "", time.Time{}, ErrUserLocked
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// 密码错误，增加失败计数
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= s.maxAttempts {
			lockedUntil := time.Now().Add(s.lockoutDuration)
			user.LockedUntil = &lockedUntil
		}
		s.store.Update(ctx, user)

		// 记录失败尝试
		s.store.RecordLoginAttempt(ctx, &LoginAttempt{
			IPAddress: ipAddress,
			Username:  username,
			Success:   false,
		})

		return "", time.Time{}, ErrInvalidCredentials
	}

	// 登录成功，重置失败计数
	now := time.Now()
	user.LastLoginAt = &now
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	if err := s.store.Update(ctx, user); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to update user: %w", err)
	}

	// 记录成功尝试
	s.store.RecordLoginAttempt(ctx, &LoginAttempt{
		IPAddress: ipAddress,
		Username:  username,
		Success:   true,
	})

	// 生成 JWT
	token, expiresAt, err := s.jwtManager.GenerateToken(user)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, expiresAt, nil
}

// ValidateToken 验证 Token
func (s *AdminAuthService) ValidateToken(tokenString string) (*Claims, error) {
	return s.jwtManager.ValidateToken(tokenString)
}

// HashPassword 哈希密码
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// GenerateSecureKey 生成安全密钥
func GenerateSecureKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// ConstantTimeStringCompare 常量时间字符串比较
func ConstantTimeStringCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// --- 用户管理方法 ---

// CreateUser 创建新用户
func (s *AdminAuthService) CreateUser(ctx context.Context, req *CreateUserRequest, createdBy string) (*AdminUser, error) {
	// 验证角色
	if !IsValidRole(req.Role) {
		return nil, ErrInvalidRole
	}

	// 检查用户名是否已存在
	if _, err := s.store.FindByUsername(ctx, req.Username); err == nil {
		return nil, ErrUserExists
	}

	// 哈希密码
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &AdminUser{
		ID:           generateUUID(),
		Username:     req.Username,
		PasswordHash: passwordHash,
		Email:        req.Email,
		Role:         req.Role,
		IsActive:     true,
		CreatedBy:    &createdBy,
	}

	if err := s.store.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// ListUsers 列出用户
func (s *AdminAuthService) ListUsers(ctx context.Context, limit, offset int) ([]*AdminUser, int64, error) {
	return s.store.List(ctx, limit, offset)
}

// GetUser 获取用户
func (s *AdminAuthService) GetUser(ctx context.Context, id string) (*AdminUser, error) {
	return s.store.FindByID(ctx, id)
}

// UpdateUser 更新用户
func (s *AdminAuthService) UpdateUser(ctx context.Context, id string, req *UpdateUserRequest) (*AdminUser, error) {
	user, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 更新字段
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Role != nil {
		if !IsValidRole(*req.Role) {
			return nil, ErrInvalidRole
		}
		user.Role = *req.Role
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.Password != "" {
		passwordHash, err := HashPassword(req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = passwordHash
	}

	if err := s.store.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// DeleteUser 删除用户
func (s *AdminAuthService) DeleteUser(ctx context.Context, id string) error {
	// 不允许删除自己
	// TODO: 从 context 获取当前用户 ID 进行检查
	return s.store.Delete(ctx, id)
}

// generateUUID 生成 UUID
func generateUUID() string {
	// 简化版实现，生产环境应使用 github.com/google/uuid
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
