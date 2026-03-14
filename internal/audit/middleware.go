package audit

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Middleware 审计日志中间件
type Middleware struct {
	logger *Logger
}

// NewMiddleware 创建审计日志中间件
func NewMiddleware(logger *Logger) *Middleware {
	return &Middleware{logger: logger}
}

// Handler 返回 Gin 中间件处理器
func (m *Middleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只记录管理接口的审计日志
		if !strings.HasPrefix(c.Request.URL.Path, "/v1/admin") {
			c.Next()
			return
		}

		startTime := time.Now()

		// 获取认证信息
		var tenantID, apiKeyID string
		if apiKey := c.Request.Context().Value("api_key"); apiKey != nil {
			if ak, ok := apiKey.(interface{ GetTenantID() string; GetID() string }); ok {
				tenantID = ak.GetTenantID()
				apiKeyID = ak.GetID()
			}
		}

		// 构建日志条目
		builder := NewEntryBuilder().
			WithTenant(tenantID).
			WithAPIKey(apiKeyID).
			WithActor(apiKeyID, ActorTypeAPIKey).
			WithRequest(c.ClientIP(), c.Request.UserAgent()).
			WithAction(determineAction(c.Request.Method)).
			WithResource(determineResourceType(c.Request.URL.Path), extractResourceID(c))

		// 执行请求
		c.Next()

		// 根据状态码设置结果
		statusCode := c.Writer.Status()
		if statusCode >= 400 {
			// 请求失败
			builder.WithError(string(rune(statusCode)), extractErrorMessage(c))
		}

		// 添加详细信息
		details := map[string]any{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"query":      c.Request.URL.RawQuery,
			"status":     statusCode,
			"latency_ms": time.Since(startTime).Milliseconds(),
		}

		builder.WithDetails(details)

		// 记录日志
		entry := builder.Build()
		if err := m.logger.Log(c.Request.Context(), entry); err != nil {
			// 记录失败不影响请求
			// 可以选择输出到标准错误日志
		}
	}
}

// determineAction 根据 HTTP 方法确定操作类型
func determineAction(method string) string {
	switch method {
	case http.MethodGet:
		return ActionRead
	case http.MethodPost:
		return ActionCreate
	case http.MethodPut, http.MethodPatch:
		return ActionUpdate
	case http.MethodDelete:
		return ActionDelete
	default:
		return "unknown"
	}
}

// determineResourceType 根据 URL 路径确定资源类型
func determineResourceType(path string) string {
	switch {
	case strings.Contains(path, "/api-keys"):
		return ResourceTypeAPIKey
	case strings.Contains(path, "/tenants"):
		return ResourceTypeTenant
	case strings.Contains(path, "/backends"):
		return ResourceTypeBackend
	case strings.Contains(path, "/models"):
		return ResourceTypeModel
	case strings.Contains(path, "/stats"):
		return "stats"
	default:
		return "unknown"
	}
}

// extractResourceID 从 URL 中提取资源 ID
func extractResourceID(c *gin.Context) string {
	// 尝试从路径参数获取 ID
	if id := c.Param("id"); id != "" {
		return id
	}
	return ""
}

// extractErrorMessage 从上下文中提取错误消息
func extractErrorMessage(c *gin.Context) string {
	// 尝试从 gin.Context 获取错误
	if len(c.Errors) > 0 {
		return c.Errors.String()
	}
	return http.StatusText(c.Writer.Status())
}

// AuditMiddlewareConfig 审计中间件配置
type AuditMiddlewareConfig struct {
	// SkipPaths 跳过审计的路径
	SkipPaths []string
	// LogRequestHeaders 是否记录请求头
	LogRequestHeaders bool
	// LogRequestBody 是否记录请求体
	LogRequestBody bool
	// LogResponseBody 是否记录响应体
	LogResponseBody bool
}

// DetailedMiddleware 详细审计中间件（可配置）
type DetailedMiddleware struct {
	logger *Logger
	config AuditMiddlewareConfig
}

// NewDetailedMiddleware 创建详细审计中间件
func NewDetailedMiddleware(logger *Logger, config AuditMiddlewareConfig) *DetailedMiddleware {
	if config.SkipPaths == nil {
		config.SkipPaths = []string{"/health", "/metrics"}
	}
	return &DetailedMiddleware{
		logger: logger,
		config: config,
	}
}

// Handler 返回 Gin 中间件处理器
func (m *DetailedMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否跳过
		for _, skipPath := range m.config.SkipPaths {
			if strings.HasPrefix(c.Request.URL.Path, skipPath) {
				c.Next()
				return
			}
		}

		startTime := time.Now()

		// 获取认证信息
		var tenantID, apiKeyID string
		if apiKey := c.Request.Context().Value("api_key"); apiKey != nil {
			if ak, ok := apiKey.(interface{ GetTenantID() string; GetID() string }); ok {
				tenantID = ak.GetTenantID()
				apiKeyID = ak.GetID()
			}
		}

		// 构建详细信息
		details := make(map[string]any)
		details["method"] = c.Request.Method
		details["path"] = c.Request.URL.Path
		details["query"] = c.Request.URL.RawQuery

		if m.config.LogRequestHeaders {
			headers := make(map[string]string)
			for k, v := range c.Request.Header {
				if len(v) == 1 {
					headers[k] = v[0]
				} else {
					headers[k] = strings.Join(v, ", ")
				}
			}
			details["headers"] = headers
		}

		// 执行请求
		c.Next()

		details["status"] = c.Writer.Status()
		details["latency_ms"] = time.Since(startTime).Milliseconds()

		// 构建日志条目
		builder := NewEntryBuilder().
			WithTenant(tenantID).
			WithAPIKey(apiKeyID).
			WithActor(apiKeyID, ActorTypeAPIKey).
			WithRequest(c.ClientIP(), c.Request.UserAgent()).
			WithAction(determineAction(c.Request.Method)).
			WithResource(determineResourceType(c.Request.URL.Path), extractResourceID(c)).
			WithDetails(details)

		// 根据状态码设置结果
		statusCode := c.Writer.Status()
		if statusCode >= 400 {
			builder.WithError(string(rune(statusCode)), extractErrorMessage(c))
		}

		// 记录日志
		entry := builder.Build()
		if err := m.logger.Log(c.Request.Context(), entry); err != nil {
			// 记录失败不影响请求
		}
	}
}

// AsyncLogger 异步审计日志记录器
type AsyncLogger struct {
	logger *Logger
	ch     chan *Log
}

// NewAsyncLogger 创建异步审计日志记录器
func NewAsyncLogger(logger *Logger, bufferSize int) *AsyncLogger {
	al := &AsyncLogger{
		logger: logger,
		ch:     make(chan *Log, bufferSize),
	}
	go al.process()
	return al
}

// Log 异步记录日志
func (al *AsyncLogger) Log(ctx context.Context, entry *Log) error {
	select {
	case al.ch <- entry:
		return nil
	default:
		// 通道满，同步记录
		return al.logger.Log(ctx, entry)
	}
}

// process 处理异步日志
func (al *AsyncLogger) process() {
	for entry := range al.ch {
		// 使用 background context 避免 context 取消影响日志记录
		ctx := context.Background()
		if err := al.logger.Log(ctx, entry); err != nil {
			// 记录失败，可以考虑重试或写入备用存储
		}
	}
}

// Close 关闭异步日志记录器
func (al *AsyncLogger) Close() {
	close(al.ch)
}
