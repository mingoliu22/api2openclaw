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
)

// AdminUser 管理员用户
type AdminUser struct {
	ID                 string     `json:"id" db:"id"`
	Username           string     `json:"username" db:"username"`
	PasswordHash       string     `json:"-" db:"password_hash"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	FailedLoginAttempts int        `json:"failed_login_attempts" db:"failed_login_attempts"`
	LockedUntil        *time.Time `json:"locked_until,omitempty" db:"locked_until"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
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
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token
func (j *JWTManager) GenerateToken(user *AdminUser) (string, time.Time, error) {
	expiresAt := time.Now().Add(j.tokenExpiration)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
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
	Update(ctx context.Context, user *AdminUser) error
	RecordLoginAttempt(ctx context.Context, attempt *LoginAttempt) error
	CountFailedAttempts(ctx context.Context, ipAddress string, since time.Time) (int64, error)
	CleanupOldAttempts(ctx context.Context, before time.Time) error
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
