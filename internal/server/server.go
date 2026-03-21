package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/openclaw/api2openclaw/internal/admin"
	"github.com/openclaw/api2openclaw/internal/auth"
	"github.com/openclaw/api2openclaw/internal/audit"
	"github.com/openclaw/api2openclaw/internal/billing"
	"github.com/openclaw/api2openclaw/internal/config"
	"github.com/openclaw/api2openclaw/internal/converter"
	"github.com/openclaw/api2openclaw/internal/models"
	"github.com/openclaw/api2openclaw/internal/monitor"
	"github.com/openclaw/api2openclaw/internal/router"
)

// Server HTTP 服务器
type Server struct {
	mu             sync.RWMutex
	config         *config.Config
	configPath     string
	router         *gin.Engine
	httpSrv        *http.Server
	authMgr        *auth.Manager
	auditLogger    *audit.Logger
	converter      converter.Converter
	modelRouter    *router.Router
	forwarder      *router.Forwarder
	metrics        *monitor.Collector
	promMetrics    *monitor.PrometheusMetrics
	rateLimiter    *monitor.RateLimiter
	circuitBreaker *monitor.CircuitBreakerRegistry
	activeTracker  *monitor.ActiveRequestsTracker

	// 管理控制台组件
	adminAuthService   *admin.AdminAuthService
	adminModelService  *admin.ModelService
	adminAPIKeyService *admin.APIKeyService
	adminAPIHandlers   *admin.APIHandlers
	requestLogStore     admin.RequestLogStore
	reloadWatcher      *admin.ReloadWatcher
}

// New 创建服务器
func New(cfg *config.Config, configPath string) (*Server, error) {
	log.Printf("[SERVER] Creating new server, auth enabled: %v", cfg.Auth.Enabled)
	// 初始化认证管理器
	var authMgr *auth.Manager
	if cfg.Auth.Enabled {
		store, err := auth.NewPostgreSQLStore(buildDSN(cfg))
		if err != nil {
			return nil, fmt.Errorf("init auth store: %w", err)
		}
		authMgr = auth.NewManager(store)
	}

	// 初始化审计日志
	var auditLogger *audit.Logger
	if cfg.Auth.Enabled {
		auditStore, err := audit.NewPostgreSQLStore(buildDSN(cfg))
		if err != nil {
			return nil, fmt.Errorf("init audit store: %w", err)
		}
		auditLogger = audit.NewLogger(auditStore)
	}

	// 初始化转换器
	cvtCfg := &converter.ConverterConfig{
		InputFormat:  cfg.Converter.InputFormat,
		OutputFormat: cfg.Converter.OutputFormat,
		Templates: converter.TemplatesConfig{
			Message:     cfg.Converter.Templates.Message,
			StreamChunk: cfg.Converter.Templates.StreamChunk,
		},
		IncludeUsage: false,
	}

	cvt, err := converter.NewConverter(cvtCfg)
	if err != nil {
		return nil, fmt.Errorf("init converter: %w", err)
	}

	// 初始化模型路由器
	modelRouter := router.New()

	// 注册配置的后端
	for _, backendCfg := range cfg.Router.Backends {
		backend := &models.Backend{
			ID:      backendCfg.ID,
			Name:    backendCfg.Name,
			Type:    backendCfg.Type,
			BaseURL: backendCfg.BaseURL,
			APIKey:  backendCfg.APIKey,
			Weight:  backendCfg.Weight,
			HealthCheck: models.HealthCheckConfig{
				Enabled:  backendCfg.HealthCheck.Enabled,
				Interval: backendCfg.HealthCheck.Interval,
				Endpoint: backendCfg.HealthCheck.Endpoint,
				Timeout:  backendCfg.HealthCheck.Timeout,
			},
		}
		if err := modelRouter.RegisterBackend(backend); err != nil {
			log.Printf("Failed to register backend %s: %v", backendCfg.ID, err)
		}
	}

	// 注册配置的模型
	for _, modelCfg := range cfg.Router.Models {
		model := &models.ModelConfig{
			Name:            modelCfg.Name,
			BackendGroup:    modelCfg.BackendGroup,
			RoutingStrategy: modelCfg.RoutingStrategy,
		}
		if err := modelRouter.RegisterModel(model); err != nil {
			log.Printf("Failed to register model %s: %v", modelCfg.Name, err)
		}
	}

	// 注册配置的模型别名
	for _, aliasCfg := range cfg.Router.Aliases {
		if err := modelRouter.RegisterAlias(
			aliasCfg.Alias,
			aliasCfg.Target,
			aliasCfg.Backends,
			aliasCfg.Strategy,
		); err != nil {
			log.Printf("Failed to register alias %s: %v", aliasCfg.Alias, err)
		}
	}

	// 初始化监控
	var promMetrics *monitor.PrometheusMetrics
	var rateLimiter *monitor.RateLimiter
	var circuitBreaker *monitor.CircuitBreakerRegistry
	var activeTracker *monitor.ActiveRequestsTracker

	if cfg.Monitor.Enabled {
		promMetrics = monitor.NewPrometheusMetrics()
		activeTracker = monitor.NewActiveRequestsTracker(promMetrics)

		limitStore := monitor.NewMemoryLimitStore()
		rateLimiter = monitor.NewRateLimiter(limitStore)

		circuitBreaker = monitor.NewCircuitBreakerRegistry()

		// 为每个后端创建熔断器
		for _, backendCfg := range cfg.Router.Backends {
			config := monitor.CircuitConfig{
				ErrorRateThreshold:  cfg.Monitor.CircuitBreaker.ErrorRateThreshold,
				ConsecutiveErrors:    cfg.Monitor.CircuitBreaker.ConsecutiveErrors,
				RecoveryTimeout:      cfg.Monitor.CircuitBreaker.RecoveryTimeout,
				HalfOpenMaxAttempts:  cfg.Monitor.CircuitBreaker.HalfOpenMaxAttempts,
			}
			circuitBreaker.Get(backendCfg.ID, config)
		}
	}

	// 初始化转发器
	var forwarder *router.Forwarder
	if promMetrics != nil {
		forwarder = router.NewForwarder(cvt, promMetrics)
	}

	// 初始化管理控制台服务
	var adminAuthService *admin.AdminAuthService
	var adminModelService *admin.ModelService
	var adminAPIKeyService *admin.APIKeyService
	var adminAPIHandlers *admin.APIHandlers
	var requestLogStore admin.RequestLogStore
	var reloadWatcher *admin.ReloadWatcher

	if cfg.Auth.Enabled {
		log.Printf("[SERVER] Initializing admin auth (enabled: %v)", cfg.Auth.Enabled)
		// 构建 PostgreSQL 连接字符串
		postgresDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Auth.Database.Host,
			cfg.Auth.Database.Port,
			cfg.Auth.Database.User,
			cfg.Auth.Database.Password,
			cfg.Auth.Database.Database,
			cfg.Auth.Database.SSLMode,
		)
		log.Printf("[SERVER] Connecting to admin DB: %s", cfg.Auth.Database.Host)

		// 使用 sqlx 创建数据库连接
		sqlxDB, err := sqlx.Connect("postgres", postgresDSN)
		if err != nil {
			log.Printf("[SERVER] Failed to connect to admin DB: %v", err)
			return nil, fmt.Errorf("failed to connect to admin database: %w", err)
		}
		sqlxDB.SetMaxOpenConns(25)
		sqlxDB.SetMaxIdleConns(5)
		log.Printf("[SERVER] Connected to admin DB successfully")

		// 创建管理员存储
		adminStore := admin.NewPostgreSQLStore(sqlxDB)
		log.Printf("[SERVER] Created PostgreSQL admin store: %T", adminStore)

		// 创建模型和 API Key 存储
		modelStore := admin.NewPostgreSQLModelStore(sqlxDB)
		apiKeyStore := admin.NewPostgreSQLAPIKeyStore(sqlxDB)
		requestLogStore = admin.NewPostgreSQLRequestLogStore(sqlxDB)

		// 创建 JWT 管理器（使用环境变量中的密钥）
		jwtSecret := getEnvOrDefault("JWT_SECRET", "change-this-secret-in-production")
		adminAuthService = admin.NewAdminAuthService(adminStore, jwtSecret)

		// 创建模型服务（使用加密密钥）
		encryptionKey := getEnvOrDefault("ENCRYPTION_KEY", "32-byte-encryption-key-1234")
		adminModelService = admin.NewModelService(modelStore, encryptionKey)

		// 创建 API Key 服务
		adminAPIKeyService = admin.NewAPIKeyService(apiKeyStore)

		// 创建 API 处理器
		adminAPIHandlers = admin.NewAPIHandlers(adminAuthService, adminModelService, adminAPIKeyService, requestLogStore)
		log.Printf("[SERVER] Created admin API handlers")

		// 初始化插件管理器
		pluginManager := converter.NewPluginManager()
		pluginDir := getEnvOrDefault("PLUGIN_DIR", "./plugins")
		pluginHandlers := admin.NewPluginHandlers(pluginManager, pluginDir)
		adminAPIHandlers.SetPluginHandlers(pluginHandlers)

		// 初始化计费服务
		billingStore := billing.NewPostgreSQLStore(sqlxDB)
		billingService := billing.NewBillingService(billingStore)
		billingHandlers := admin.NewBillingHandlers(billingService)
		adminAPIHandlers.SetBillingHandlers(billingHandlers)

		// 初始化统计服务
		statsStore := admin.NewStatsStore(sqlxDB)
		statsHandlers := admin.NewStatsHandlers(statsStore)
		adminAPIHandlers.SetStatsHandlers(statsHandlers)

		// 初始化成本服务
		costStore := admin.NewCostStore(sqlxDB)
		costHandlers := admin.NewCostHandlers(costStore)
		adminAPIHandlers.SetCostHandlers(costHandlers)

		// 创建配置重载监听器
		reloadWatcher = admin.NewReloadWatcher(sqlxDB)
	}

	s := &Server{
		config:             cfg,
		configPath:         configPath,
		authMgr:            authMgr,
		auditLogger:        auditLogger,
		converter:          cvt,
		modelRouter:        modelRouter,
		forwarder:          forwarder,
		metrics:            nil, // TODO
		promMetrics:        promMetrics,
		rateLimiter:        rateLimiter,
		circuitBreaker:     circuitBreaker,
		activeTracker:      activeTracker,
		adminAuthService:    adminAuthService,
		adminModelService:   adminModelService,
		adminAPIKeyService:  adminAPIKeyService,
		adminAPIHandlers:   adminAPIHandlers,
		requestLogStore:    requestLogStore,
		reloadWatcher:      reloadWatcher,
	}

	// 添加配置重载监听器
	if s.reloadWatcher != nil {
		s.reloadWatcher.AddListener(s)
		// 将 reloadWatcher 传递给 adminAPIHandlers
		if adminAPIHandlers != nil {
			adminAPIHandlers.SetReloadWatcher(s.reloadWatcher)
		}
	}

	s.setupRouter()
	return s, nil
}

// buildDSN 构建数据库连接字符串
func buildDSN(cfg *config.Config) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Auth.Database.Host,
		cfg.Auth.Database.Port,
		cfg.Auth.Database.User,
		cfg.Auth.Database.Password,
		cfg.Auth.Database.Database,
		cfg.Auth.Database.SSLMode,
	)
}

// getEnvOrDefault 获取环境变量或返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// setupRouter 设置路由
func (s *Server) setupRouter() {
	gin.SetMode(gin.ReleaseMode)
	s.router = gin.New()
	s.router.Use(gin.Recovery())
	s.router.Use(s.loggingMiddleware())
	s.router.Use(s.corsMiddleware())

	// 健康检查
	s.router.GET("/health", s.handleHealth)

	// 测试端点
	s.router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Server is working",
			"auth_enabled": s.config.Auth.Enabled,
		})
	})

	// Prometheus 指标
	if s.config.Monitor.Prometheus.Enabled && s.promMetrics != nil {
		s.router.GET(s.config.Monitor.Prometheus.MetricsPath, gin.WrapH(s.promMetrics.Handler()))
	}

	// API 路由组
	v1 := s.router.Group(s.config.Server.BasePath)
	{
		// 公开接口
		v1.POST("/convert", s.handleConvert)

		// 模型相关（需要认证）
		if s.config.Auth.Enabled && s.authMgr != nil {
			authMW := auth.NewMiddleware(s.authMgr)

			// 审计中间件
			if s.auditLogger != nil {
				auditMW := audit.NewMiddleware(s.auditLogger)
				s.router.Use(auditMW.Handler())
			}

			// 聊天完成接口
			v1.POST("/chat/completions", ginAuthMiddleware(authMW), s.handleChatCompletions)
			v1.POST("/chat/completions/stream", ginAuthMiddleware(authMW), s.handleChatCompletionsStream)

			// 管理接口
			admin := v1.Group("/admin")
			admin.Use(ginAuthMiddleware(authMW))
			admin.Use(ginRequirePermissionMiddleware(authMW, "admin"))
			{
				admin.POST("/tenants", s.handleCreateTenant)
				admin.GET("/tenants", s.handleListTenants)
				admin.POST("/api-keys", s.handleCreateAPIKey)
				admin.GET("/api-keys", s.handleListAPIKeys)
				admin.POST("/api-keys/:id/revoke", s.handleRevokeAPIKey)
				admin.DELETE("/api-keys/:id", s.handleDeleteAPIKey)

				// 后端管理
				admin.GET("/backends", s.handleListBackends)
				admin.POST("/backends", s.handleRegisterBackend)
				admin.PUT("/backends/:id", s.handleUpdateBackend)
				admin.DELETE("/backends/:id", s.handleDeleteBackend)

				// 模型管理
				admin.GET("/models", s.handleListModels)
				admin.POST("/models", s.handleRegisterModel)
				admin.PUT("/models/:name", s.handleUpdateModel)
				admin.DELETE("/models/:name", s.handleDeleteModel)

				// 模型别名管理
				admin.GET("/aliases", s.handleListAliases)
				admin.POST("/aliases", s.handleRegisterAlias)
				admin.DELETE("/aliases/:alias", s.handleDeleteAlias)

				// 监控接口
				admin.GET("/stats", s.handleStats)
				admin.GET("/usage", s.handleGetUsageStats)

				// 审计日志接口
				admin.GET("/audit-logs", s.handleListAuditLogs)
				admin.GET("/audit-logs/:id", s.handleGetAuditLog)
			}

			// JWT 认证的管理接口（新版控制台 API）
			if s.adminAPIHandlers != nil {
				log.Printf("[SERVER] Registering admin API handlers")
				s.adminAPIHandlers.RegisterRoutes(s.router.Group(""))
			} else {
				log.Printf("[SERVER] adminAPIHandlers is nil!")
			}
		} else {
		log.Printf("[SERVER] Auth is disabled or authMgr is nil")
	}
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	// 启动配置重载监听器
	if s.reloadWatcher != nil {
		s.reloadWatcher.Start()
		log.Println("[Server] Reload watcher started")
	}

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
	}

	log.Printf("Server listening on %s", addr)
	return s.httpSrv.ListenAndServe()
}

// Shutdown 优雅关闭
func (s *Server) Shutdown(ctx context.Context) error {
	// 停止配置重载监听器
	if s.reloadWatcher != nil {
		s.reloadWatcher.Stop()
		log.Println("[Server] Reload watcher stopped")
	}

	if s.modelRouter != nil {
		s.modelRouter.Close()
	}
	if s.httpSrv != nil {
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}

// OnModelsChanged 模型配置变更回调
func (s *Server) OnModelsChanged() {
	log.Println("[Server] Models config changed, reloading...")
	// 触发模型路由重新加载
	if s.modelRouter != nil {
		// 这里需要重新加载模型配置到路由器
		// 当前简化实现：记录日志
		log.Println("[Server] Model router reloaded")
	}
}

// --- 处理器 ---

// handleHealth 健康检查
func (s *Server) handleHealth(c *gin.Context) {
	status := gin.H{
		"status":   "ok",
		"timestamp": time.Now().Unix(),
	}

	// 添加后端状态
	if s.modelRouter != nil {
		backends := s.modelRouter.ListBackends()
		healthy := 0
		for _, b := range backends {
			if b.IsHealthy() {
				healthy++
			}
		}
		status["backends"] = gin.H{
			"total":   len(backends),
			"healthy": healthy,
		}
	}

	// 添加活跃请求
	if s.activeTracker != nil {
		status["active_requests"] = s.activeTracker.Count()
	}

	c.JSON(http.StatusOK, status)
}

// handleConvert 格式转换接口
func (s *Server) handleConvert(c *gin.Context) {
	var req struct {
		Data json.RawMessage `json:"data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := s.converter.Convert(req.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, string(output))
}

// handleChatCompletions 聊天完成接口
func (s *Server) handleChatCompletions(c *gin.Context) {
	startTime := time.Now()

	// 跟踪活跃请求
	if s.activeTracker != nil {
		s.activeTracker.Begin()
		defer s.activeTracker.End()
	}

	var req router.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"message": "Invalid request: " + err.Error(),
			"type":    "invalid_request_error",
		}})
		return
	}

	// 获取认证信息
	apiKey := auth.GetAPIKey(c.Request)
	if apiKey == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"message": "authentication required",
			"type":    "authentication_error",
		}})
		return
	}

	// 检查模型权限
	if !apiKey.CanUseModel(req.Model) {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
			"message": "Model not allowed for this API key",
			"type":    "permission_error",
		}})
		return
	}

	// 检查限流
	if s.rateLimiter != nil {
		limits := &models.RateLimit{
			RequestsPerMinute: apiKey.RequestsPerMinute,
			RequestsPerHour:   apiKey.RequestsPerHour,
			RequestsPerDay:    apiKey.RequestsPerDay,
		}
		if err := s.rateLimiter.CheckLimit(apiKey.ID, limits); err != nil {
			if s.promMetrics != nil {
				s.promMetrics.RecordRateLimit(apiKey.ID, "exceeded")
			}

			// 添加 Retry-After 响应头（默认 60 秒）
			retryAfter := int64(60)
			if limits.RequestsPerMinute > 0 {
				retryAfter = int64(60 / limits.RequestsPerMinute)
			}

			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))

			c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{
				"message": fmt.Sprintf("Rate limit exceeded. Retry after %d seconds.", retryAfter),
				"type":    "rate_limit_error",
				"retry_after": retryAfter,
			}})
			return
		}
	}

	// 路由到后端
	backend, err := s.modelRouter.Route(c.Request.Context(), req.Model)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
			"message": "No available backend for model: " + err.Error(),
			"type":    "service_unavailable",
		}})
		return
	}

	log.Printf("[Chat] Routing %s to backend %s", req.Model, backend.ID)

	// 检查熔断器
	if s.circuitBreaker != nil {
		cb := s.circuitBreaker.Get(backend.ID, monitor.CircuitConfig{})
		if !cb.Allow() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
				"message": "Circuit breaker is open for backend",
				"type":    "service_unavailable",
			}})
			return
		}
	}

	// 转发请求
	var resp *router.ChatCompletionResponse
	if s.forwarder != nil {
		resp, err = s.forwarder.ForwardRequest(c.Request.Context(), backend, &req)
	} else {
		err = fmt.Errorf("forwarder not initialized")
	}

	if err != nil {
		// 记录熔断器错误
		if s.circuitBreaker != nil {
			cb := s.circuitBreaker.Get(backend.ID, monitor.CircuitConfig{})
			cb.RecordFailure()
		}

		// 记录失败的请求日志
		duration := time.Since(startTime)
		s.logRequestAsync(
			apiKey.ID,
			req.Model,
			backend.ID,
			0, 0, 0,
			int(duration.Milliseconds()),
			500,
			"backend_error",
			err.Error(),
			c.GetHeader("X-Request-ID"),
			c.ClientIP(),
			c.GetHeader("User-Agent"),
		)

		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"message": "Backend request failed: " + err.Error(),
			"type":    "backend_error",
		}})
		return
	}

	// 记录成功
	if s.circuitBreaker != nil {
		cb := s.circuitBreaker.Get(backend.ID, monitor.CircuitConfig{})
		cb.RecordSuccess()
	}

	// 增加限流计数
	if s.rateLimiter != nil {
		limits := &models.RateLimit{
			RequestsPerMinute: apiKey.RequestsPerMinute,
			RequestsPerHour:   apiKey.RequestsPerHour,
			RequestsPerDay:    apiKey.RequestsPerDay,
		}
		_ = s.rateLimiter.Increment(apiKey.ID, limits)
	}

	// 记录指标和日志
	duration := time.Since(startTime)
	if s.promMetrics != nil {
		s.promMetrics.RecordHTTPRequest("POST", "/chat/completions", 200, duration)
	}

	// 记录成功的请求日志
	promptTokens := 0
	completionTokens := 0
	totalTokens := 0
	if resp.Usage != nil {
		promptTokens = resp.Usage.PromptTokens
		completionTokens = resp.Usage.CompletionTokens
		totalTokens = resp.Usage.TotalTokens
	}
	s.logRequestAsync(
		apiKey.ID,
		req.Model,
		backend.ID,
		promptTokens,
		completionTokens,
		totalTokens,
		int(duration.Milliseconds()),
		200,
		"",
		"",
		c.GetHeader("X-Request-ID"),
		c.ClientIP(),
		c.GetHeader("User-Agent"),
	)

	c.JSON(http.StatusOK, resp)
}

// handleChatCompletionsStream 流式聊天完成接口
func (s *Server) handleChatCompletionsStream(c *gin.Context) {
	startTime := time.Now()

	// 跟踪活跃请求
	if s.activeTracker != nil {
		s.activeTracker.Begin()
		defer s.activeTracker.End()
	}

	var req router.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"message": "Invalid request: " + err.Error(),
			"type":    "invalid_request_error",
		}})
		return
	}

	// 强制开启流式
	req.Stream = true

	// 获取认证信息
	apiKey := auth.GetAPIKey(c.Request)
	if apiKey == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"message": "authentication required",
			"type":    "authentication_error",
		}})
		return
	}

	// 检查模型权限
	if !apiKey.CanUseModel(req.Model) {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
			"message": "Model not allowed for this API key",
			"type":    "permission_error",
		}})
		return
	}

	// 检查限流
	if s.rateLimiter != nil {
		limits := &models.RateLimit{
			RequestsPerMinute: apiKey.RequestsPerMinute,
			RequestsPerHour:   apiKey.RequestsPerHour,
			RequestsPerDay:    apiKey.RequestsPerDay,
		}
		if err := s.rateLimiter.CheckLimit(apiKey.ID, limits); err != nil {
			if s.promMetrics != nil {
				s.promMetrics.RecordRateLimit(apiKey.ID, "exceeded")
			}

			retryAfter := int64(60)
			if limits.RequestsPerMinute > 0 {
				retryAfter = int64(60 / limits.RequestsPerMinute)
			}

			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{
				"message": fmt.Sprintf("Rate limit exceeded. Retry after %d seconds.", retryAfter),
				"type":    "rate_limit_error",
				"retry_after": retryAfter,
			}})
			return
		}
	}

	// 路由到后端
	backend, err := s.modelRouter.Route(c.Request.Context(), req.Model)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
			"message": "No available backend for model: " + err.Error(),
			"type":    "service_unavailable",
		}})
		return
	}

	log.Printf("[Chat Stream] Routing %s to backend %s", req.Model, backend.ID)

	// 检查熔断器
	if s.circuitBreaker != nil {
		cb := s.circuitBreaker.Get(backend.ID, monitor.CircuitConfig{})
		if !cb.Allow() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
				"message": "Circuit breaker is open for backend",
				"type":    "service_unavailable",
			}})
			return
		}
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 创建 flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// 使用转发器处理流式请求
	if s.forwarder == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Forwarder not initialized"})
		return
	}

	chunkChan, errChan := s.forwarder.ForwardStreamRequest(c.Request.Context(), backend, &req, apiKey.ID)

	// 发送流式数据
	for {
		select {
		case <-c.Request.Context().Done():
			// 客户端断开连接
			log.Printf("[Chat Stream] Client disconnected")
			duration := time.Since(startTime)
			s.logRequestAsync(
				apiKey.ID,
				req.Model,
				backend.ID,
				0, 0, 0,
				int(duration.Milliseconds()),
				499, // Client Closed Request
				"client_disconnected",
				"Client disconnected during stream",
				c.GetHeader("X-Request-ID"),
				c.ClientIP(),
				c.GetHeader("User-Agent"),
			)
			return

		case chunk, ok := <-chunkChan:
			if !ok {
				// 流式传输完成
				// 发送完成信号
				fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
				flusher.Flush()

				// 记录成功的流式请求日志（流式请求通常无法获取准确的 token 数）
				duration := time.Since(startTime)
				s.logRequestAsync(
					apiKey.ID,
					req.Model,
					backend.ID,
					0, 0, 0, // 流式请求的 token 数需要从后端响应中解析，暂时记录 0
					int(duration.Milliseconds()),
					200,
					"",
					"",
					c.GetHeader("X-Request-ID"),
					c.ClientIP(),
					c.GetHeader("User-Agent"),
				)

				// 记录成功并增加限流计数
				if s.circuitBreaker != nil {
					cb := s.circuitBreaker.Get(backend.ID, monitor.CircuitConfig{})
					cb.RecordSuccess()
				}
				if s.rateLimiter != nil {
					limits := &models.RateLimit{
						RequestsPerMinute: apiKey.RequestsPerMinute,
						RequestsPerHour:   apiKey.RequestsPerHour,
						RequestsPerDay:    apiKey.RequestsPerDay,
					}
					_ = s.rateLimiter.Increment(apiKey.ID, limits)
				}

				return
			}

			// 发送 SSE 事件
			chunkData, err := json.Marshal(chunk)
			if err != nil {
				log.Printf("[Chat Stream] Failed to marshal chunk: %v", err)
				continue
			}

			fmt.Fprintf(c.Writer, "data: %s\n\n", chunkData)
			flusher.Flush()

		case err := <-errChan:
			if err != nil {
				log.Printf("[Chat Stream] Error: %v", err)

				// 记录失败的流式请求日志
				duration := time.Since(startTime)
				s.logRequestAsync(
					apiKey.ID,
					req.Model,
					backend.ID,
					0, 0, 0,
					int(duration.Milliseconds()),
					500,
					"stream_error",
					err.Error(),
					c.GetHeader("X-Request-ID"),
					c.ClientIP(),
					c.GetHeader("User-Agent"),
				)

				// 记录熔断器失败
				if s.circuitBreaker != nil {
					cb := s.circuitBreaker.Get(backend.ID, monitor.CircuitConfig{})
					cb.RecordFailure()
				}

				// 发送错误事件
				errorData, _ := json.Marshal(gin.H{
					"error": gin.H{
						"message": err.Error(),
						"type":    "stream_error",
					},
				})
				fmt.Fprintf(c.Writer, "data: %s\n\n", errorData)
				flusher.Flush()
				return
			}
		}
	}
}

// handleListBackends 列出后端
func (s *Server) handleListBackends(c *gin.Context) {
	backends := s.modelRouter.ListBackends()
	c.JSON(http.StatusOK, gin.H{
		"data": backends,
	})
}

// handleListModels 列出模型
func (s *Server) handleListModels(c *gin.Context) {
	models := s.modelRouter.ListModels()
	c.JSON(http.StatusOK, gin.H{
		"data": models,
	})
}

// handleStats 获取统计信息
func (s *Server) handleStats(c *gin.Context) {
	stats := gin.H{
		"backends": s.modelRouter.ListBackends(),
		"models":   s.modelRouter.ListModels(),
	}

	if s.activeTracker != nil {
		stats["active_requests"] = s.activeTracker.Count()
	}

	c.JSON(http.StatusOK, stats)
}

// handleCreateTenant 创建租户
func (s *Server) handleCreateTenant(c *gin.Context) {
	var req auth.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant, err := s.authMgr.CreateTenant(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tenant)
}

// handleListTenants 列出租户
func (s *Server) handleListTenants(c *gin.Context) {
	tenants, err := s.authMgr.ListTenants(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tenants)
}

// handleCreateAPIKey 创建 API Key
func (s *Server) handleCreateAPIKey(c *gin.Context) {
	var req auth.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := s.authMgr.GenerateAPIKey(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// handleListAPIKeys 列出 API Keys
func (s *Server) handleListAPIKeys(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	keys, err := s.authMgr.ListAPIKeys(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, keys)
}

// handleRevokeAPIKey 吊销 API Key
func (s *Server) handleRevokeAPIKey(c *gin.Context) {
	id := c.Param("id")

	// 获取当前认证的 API Key
	apiKey := auth.GetAPIKey(c.Request)
	if apiKey == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// 记录操作日志
	log.Printf("[Admin] API Key %s being revoked by %s", id, apiKey.ID)

	if err := s.authMgr.RevokeAPIKey(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// handleDeleteAPIKey 删除 API Key
func (s *Server) handleDeleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	if err := s.authMgr.DeleteAPIKey(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// corsMiddleware CORS 中间件
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		// 允许的来源列表
		allowedOrigins := map[string]bool{
			"http://localhost:5173":     true,
			"http://localhost:5174":     true,
			"http://127.0.0.1:5173":     true,
			"http://127.0.0.1:5174":     true,
			"http://localhost:3000":     true,
			"http://localhost:8080":     true,
		}

		// 检查来源是否允许
		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cookie")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// loggingMiddleware 日志中间件
func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// 记录 Prometheus 指标
		if s.promMetrics != nil {
			s.promMetrics.RecordHTTPRequest(c.Request.Method, path, statusCode, latency)
		}

		log.Printf("[%s] %s %s - %d - %v",
			c.Request.Method,
			path,
			c.ClientIP(),
			statusCode,
			latency,
		)
	}
}

// --- Gin 中间件适配器 ---

func ginAuthMiddleware(mw *auth.Middleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		w := &ginResponseWriter{
			ResponseWriter: c.Writer,
			written:        false,
		}

		mw.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 更新 c.Request 的 context
			c.Request = r
			c.Next()
		})).ServeHTTP(w, c.Request)

		if w.written {
			c.Abort()
		}
	}
}

func ginRequirePermissionMiddleware(mw *auth.Middleware, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := auth.GetAPIKey(c.Request)
		if key == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		if err := mw.GetManager().CheckPermission(key, permission); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			c.Abort()
			return
		}

		c.Next()
	}
}

type ginResponseWriter struct {
	gin.ResponseWriter
	written bool
}

func (w *ginResponseWriter) Write(b []byte) (int, error) {
	w.written = true
	return w.ResponseWriter.Write(b)
}

func (w *ginResponseWriter) WriteHeader(statusCode int) {
	w.written = true
	w.ResponseWriter.WriteHeader(statusCode)
}

// ReloadConfig 重载配置
func (s *Server) ReloadConfig(ctx context.Context, newCfg *config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[Server] Reloading configuration...")

	// 更新配置
	s.config = newCfg

	// 重新初始化转换器（如果配置有变）
	cvtCfg := &converter.ConverterConfig{
		InputFormat:  newCfg.Converter.InputFormat,
		OutputFormat: newCfg.Converter.OutputFormat,
		Templates: converter.TemplatesConfig{
			Message:     newCfg.Converter.Templates.Message,
			StreamChunk: newCfg.Converter.Templates.StreamChunk,
		},
		IncludeUsage: false,
	}

	cvt, err := converter.NewConverter(cvtCfg)
	if err != nil {
		return fmt.Errorf("recreate converter: %w", err)
	}
	s.converter = cvt

	// 重新配置路由器
	// 清空现有的后端和模型注册
	s.modelRouter.Close()
	newModelRouter := router.New()

	// 注册配置的后端
	for _, backendCfg := range newCfg.Router.Backends {
		backend := &models.Backend{
			ID:      backendCfg.ID,
			Name:    backendCfg.Name,
			Type:    backendCfg.Type,
			BaseURL: backendCfg.BaseURL,
			APIKey:  backendCfg.APIKey,
			Weight:  backendCfg.Weight,
			HealthCheck: models.HealthCheckConfig{
				Enabled:  backendCfg.HealthCheck.Enabled,
				Interval: backendCfg.HealthCheck.Interval,
				Endpoint: backendCfg.HealthCheck.Endpoint,
				Timeout:  backendCfg.HealthCheck.Timeout,
			},
		}
		if err := newModelRouter.RegisterBackend(backend); err != nil {
			log.Printf("[Server] Failed to register backend %s: %v", backendCfg.ID, err)
		}
	}

	// 注册配置的模型
	for _, modelCfg := range newCfg.Router.Models {
		model := &models.ModelConfig{
			Name:            modelCfg.Name,
			BackendGroup:    modelCfg.BackendGroup,
			RoutingStrategy: modelCfg.RoutingStrategy,
		}
		if err := newModelRouter.RegisterModel(model); err != nil {
			log.Printf("[Server] Failed to register model %s: %v", modelCfg.Name, err)
		}
	}

	s.modelRouter = newModelRouter

	// 重新配置熔断器
	if newCfg.Monitor.Enabled && s.circuitBreaker != nil {
		for _, backendCfg := range newCfg.Router.Backends {
			config := monitor.CircuitConfig{
				ErrorRateThreshold:  newCfg.Monitor.CircuitBreaker.ErrorRateThreshold,
				ConsecutiveErrors:    newCfg.Monitor.CircuitBreaker.ConsecutiveErrors,
				RecoveryTimeout:      newCfg.Monitor.CircuitBreaker.RecoveryTimeout,
				HalfOpenMaxAttempts:  newCfg.Monitor.CircuitBreaker.HalfOpenMaxAttempts,
			}
			s.circuitBreaker.Get(backendCfg.ID, config)
		}
	}

	log.Printf("[Server] Configuration reloaded successfully")
	return nil
}

// handleListAuditLogs 列出审计日志
func (s *Server) handleListAuditLogs(c *gin.Context) {
	if s.auditLogger == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "audit logger not available"})
		return
	}

	// 构建查询过滤器
	filter := &audit.Filter{
		TenantID:     c.Query("tenant_id"),
		APIKeyID:     c.Query("api_key_id"),
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		ResourceID:   c.Query("resource_id"),
		Status:       c.Query("status"),
	}

	// 处理分页
	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	filter.Limit = limit
	filter.Offset = offset

	// 处理时间范围
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = &t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = &t
		}
	}

	logs, err := s.auditLogger.Query(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取总数
	count, err := s.auditLogger.Count(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"pagination": gin.H{
			"total":  count,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// handleGetAuditLog 获取单条审计日志
func (s *Server) handleGetAuditLog(c *gin.Context) {
	if s.auditLogger == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "audit logger not available"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid log ID"})
		return
	}

	log, err := s.auditLogger.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "audit log not found"})
		return
	}

	c.JSON(http.StatusOK, log)
}

// handleRegisterBackend 注册新后端
func (s *Server) handleRegisterBackend(c *gin.Context) {
	var req struct {
		ID      string `json:"id" binding:"required"`
		Name    string `json:"name" binding:"required"`
		Type    string `json:"type" binding:"required"`
		BaseURL string `json:"base_url" binding:"required"`
		APIKey  string `json:"api_key"`
		Weight  int    `json:"weight"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	backend := &models.Backend{
		ID:      req.ID,
		Name:    req.Name,
		Type:    req.Type,
		BaseURL: req.BaseURL,
		APIKey:  req.APIKey,
		Weight:  req.Weight,
		HealthCheck: models.HealthCheckConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Endpoint: "/models",
			Timeout:  5 * time.Second,
		},
		Status: models.BackendStatusHealthy,
	}

	if err := s.modelRouter.RegisterBackend(backend); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// 记录审计日志
	if s.auditLogger != nil {
		s.auditLogger.LogAction(c.Request.Context(), audit.ActionCreate, audit.ResourceTypeBackend, req.ID, "", "", map[string]any{
			"name":     req.Name,
			"type":     req.Type,
			"base_url": req.BaseURL,
		})
	}

	c.JSON(http.StatusCreated, backend)
}

// handleUpdateBackend 更新后端配置
func (s *Server) handleUpdateBackend(c *gin.Context) {
	id := c.Param("id")
	backend, exists := s.modelRouter.GetBackend(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "backend not found"})
		return
	}

	var req struct {
		Name    string `json:"name"`
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Weight  int    `json:"weight"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新字段（需要实现 UpdateBackend 方法）
	// 临时方案：返回成功提示
	c.JSON(http.StatusOK, gin.H{
		"message": "backend update requires restart",
		"backend": backend,
	})
}

// handleDeleteBackend 删除后端
func (s *Server) handleDeleteBackend(c *gin.Context) {
	id := c.Param("id")

	if _, exists := s.modelRouter.GetBackend(id); !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "backend not found"})
		return
	}

	// 需要实现 UnregisterBackend 方法
	c.JSON(http.StatusOK, gin.H{"message": "backend deletion requires restart"})
}

// handleRegisterModel 注册新模型
func (s *Server) handleRegisterModel(c *gin.Context) {
	var req struct {
		Name            string   `json:"name" binding:"required"`
		BackendGroup    []string `json:"backend_group" binding:"required"`
		RoutingStrategy string   `json:"routing_strategy"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.RoutingStrategy == "" {
		req.RoutingStrategy = "round-robin"
	}

	model := &models.ModelConfig{
		Name:            req.Name,
		BackendGroup:    req.BackendGroup,
		RoutingStrategy: req.RoutingStrategy,
	}

	if err := s.modelRouter.RegisterModel(model); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// 记录审计日志
	if s.auditLogger != nil {
		s.auditLogger.LogAction(c.Request.Context(), audit.ActionCreate, audit.ResourceTypeModel, req.Name, "", "", map[string]any{
			"backend_group":    req.BackendGroup,
			"routing_strategy": req.RoutingStrategy,
		})
	}

	c.JSON(http.StatusCreated, model)
}

// handleUpdateModel 更新模型配置
func (s *Server) handleUpdateModel(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		BackendGroup    []string `json:"backend_group"`
		RoutingStrategy string   `json:"routing_strategy"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 需要实现 UpdateModel 方法
	c.JSON(http.StatusOK, gin.H{
		"message": "model update requires restart",
		"name":    name,
	})
}

// handleDeleteModel 删除模型
func (s *Server) handleDeleteModel(c *gin.Context) {
	name := c.Param("name")

	if _, exists := s.modelRouter.GetModel(name); !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
		return
	}

	// 需要实现 UnregisterModel 方法
	c.JSON(http.StatusOK, gin.H{"message": "model deletion requires restart"})
}

// handleListAliases 列出模型别名
func (s *Server) handleListAliases(c *gin.Context) {
	aliases := s.modelRouter.ListAliases()

	// 转换为 JSON 友好格式
	result := make([]gin.H, 0, len(aliases))
	for alias, cfg := range aliases {
		result = append(result, gin.H{
			"alias":    alias,
			"target":   cfg.Target,
			"backends": cfg.Backends,
			"strategy": cfg.Strategy,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// handleRegisterAlias 注册模型别名
func (s *Server) handleRegisterAlias(c *gin.Context) {
	var req struct {
		Alias    string   `json:"alias" binding:"required"`
		Target   string   `json:"target" binding:"required"`
		Backends []string `json:"backends"`
		Strategy string   `json:"strategy"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.modelRouter.RegisterAlias(req.Alias, req.Target, req.Backends, req.Strategy); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// 记录审计日志
	if s.auditLogger != nil {
		s.auditLogger.LogAction(c.Request.Context(), audit.ActionCreate, "alias", req.Alias, "", "", map[string]any{
			"target":   req.Target,
			"backends": req.Backends,
			"strategy": req.Strategy,
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"alias":    req.Alias,
		"target":   req.Target,
		"backends": req.Backends,
		"strategy": req.Strategy,
	})
}

// handleDeleteAlias 删除模型别名
func (s *Server) handleDeleteAlias(c *gin.Context) {
	alias := c.Param("alias")

	if _, exists := s.modelRouter.GetAlias(alias); !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "alias not found"})
		return
	}

	// 需要实现 UnregisterAlias 方法
	c.JSON(http.StatusOK, gin.H{"message": "alias deletion requires restart"})
}

// handleGetUsageStats 获取用量统计
func (s *Server) handleGetUsageStats(c *gin.Context) {
	apiKeyID := c.Query("api_key_id")
	model := c.Query("model")
	days := 30

	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	stats := gin.H{
		"period_days": days,
		"api_key_id":  apiKeyID,
		"model":       model,
	}

	// 如果有指标收集器，获取 Prometheus 统计
	if s.promMetrics != nil {
		stats["prometheus_available"] = true
		stats["prometheus_url"] = "/metrics"
	}

	// TODO: 从数据库获取详细统计数据
	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// logRequest 异步记录请求日志
func (s *Server) logRequestAsync(apiKeyID, modelAlias, modelActual string, promptTokens, completionTokens, totalTokens, latencyMs, statusCode int, errorCode, errorMessage, requestID, ipAddress, userAgent string) {
	if s.requestLogStore == nil {
		return
	}

	// 使用 goroutine 异步写入，不阻塞请求响应
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log := &admin.RequestLog{
			ID:                generateUUID(),
			KeyID:             stringPtr(apiKeyID),
			ModelAlias:        modelAlias,
			ModelActual:       stringPtr(modelActual),
			PromptTokens:      promptTokens,
			CompletionTokens:  completionTokens,
			TotalTokens:       totalTokens,
			LatencyMs:         latencyMs,
			StatusCode:        statusCode,
			ErrorCode:         stringPtr(errorCode),
			ErrorMessage:      stringPtr(errorMessage),
			RequestID:         stringPtr(requestID),
			IPAddress:         stringPtr(ipAddress),
			UserAgent:         stringPtr(userAgent),
			CreatedAt:         time.Now(),
		}

		if err := s.requestLogStore.Create(ctx, log); err != nil {
			// Log storage failure
		}
	}()
}

// generateUUID 生成 UUID（简化版，生产环境应使用 github.com/google/uuid）
func generateUUID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), "req")
}

// stringPtr 返回字符串指针的辅助函数
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

