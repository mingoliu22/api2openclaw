package admin

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// APIHandlers 管理 API 处理器
type APIHandlers struct {
	authService      *AdminAuthService
	modelService     *ModelService
	apiKeyService    *APIKeyService
	requestLogStore  RequestLogStore
	reloadWatcher    *ReloadWatcher
	authHandler      *Handler
	pluginHandlers   *PluginHandlers
	billingHandlers  *BillingHandlers
	statsHandlers    *StatsHandlers
	costHandlers     *CostHandlers
}

// NewAPIHandlers 创建 API 处理器
func NewAPIHandlers(
	authService *AdminAuthService,
	modelService *ModelService,
	apiKeyService *APIKeyService,
	requestLogStore RequestLogStore,
) *APIHandlers {
	return &APIHandlers{
		authService:     authService,
		modelService:    modelService,
		apiKeyService:   apiKeyService,
		requestLogStore: requestLogStore,
		authHandler:     NewHandler(authService),
	}
}

// SetReloadWatcher 设置配置重载监听器
func (h *APIHandlers) SetReloadWatcher(watcher *ReloadWatcher) {
	h.reloadWatcher = watcher
}

// SetPluginHandlers 设置插件管理器
func (h *APIHandlers) SetPluginHandlers(handlers *PluginHandlers) {
	h.pluginHandlers = handlers
}

// SetBillingHandlers 设置计费处理器
func (h *APIHandlers) SetBillingHandlers(handlers *BillingHandlers) {
	h.billingHandlers = handlers
}

// SetStatsHandlers 设置统计处理器
func (h *APIHandlers) SetStatsHandlers(handlers *StatsHandlers) {
	h.statsHandlers = handlers
}

// SetCostHandlers 设置成本处理器
func (h *APIHandlers) SetCostHandlers(handlers *CostHandlers) {
	h.costHandlers = handlers
}

// notifyModelChanged 通知模型配置变更
func (h *APIHandlers) notifyModelChanged() {
	if h.reloadWatcher != nil {
		h.reloadWatcher.NotifyModelsChanged()
	}
}

// RegisterRoutes 注册路由
func (h *APIHandlers) RegisterRoutes(r *gin.RouterGroup) {
	// 认证相关（无需 JWT）
	auth := r.Group("/admin/auth")
	{
		auth.POST("/login", h.LoginHandler)
		auth.POST("/logout", h.LogoutHandler)
	}

	// 需要认证的管理接口
	admin := r.Group("/admin")
	admin.Use(JWTMiddleware(h.authService.jwtManager))
	{
		// 用户信息
		admin.GET("/auth/me", h.MeHandler)

		// 用户管理（需要权限）
		users := admin.Group("/users")
		users.Use(RequirePermissionMiddleware(h.authService.jwtManager, "users.read"))
		{
			users.GET("", h.ListUsers)
			users.GET("/:id", h.GetUser)
			// 创建/更新/删除需要更高权限
			users.POST("", RequirePermissionMiddleware(h.authService.jwtManager, "users.write"), h.CreateUser)
			users.PUT("/:id", RequirePermissionMiddleware(h.authService.jwtManager, "users.write"), h.UpdateUser)
			users.DELETE("/:id", RequirePermissionMiddleware(h.authService.jwtManager, "users.write"), h.DeleteUser)
		}

		// 模型管理
		admin.GET("/models", h.GetModels)
		admin.POST("/models", h.CreateModel)
		admin.PUT("/models/:id", h.UpdateModel)
		admin.DELETE("/models/:id", h.DeleteModel)
		admin.POST("/models/test", h.TestConnection)
		admin.POST("/models/:id/toggle", h.ToggleModel)

		// API Key 管理
		admin.GET("/keys", h.GetAPIKeys)
		admin.POST("/keys", h.CreateAPIKey)
		admin.PUT("/keys/:id", h.UpdateAPIKey)
		admin.DELETE("/keys/:id", h.RevokeAPIKey)
		admin.GET("/keys/:id", h.GetAPIKey)
		admin.GET("/keys/:id/usage", h.GetAPIKeyUsage)
		admin.GET("/keys/:id/quota", h.GetAPIKeyQuota)

		// 用量与日志
		admin.GET("/usage", h.GetUsage)
		admin.GET("/logs", h.GetLogs)
		admin.GET("/logs/export", h.ExportLogs)

		// 插件管理（需要管理员权限）
		if h.pluginHandlers != nil {
			plugins := admin.Group("/plugins")
			plugins.Use(RequirePermissionMiddleware(h.authService.jwtManager, "plugins.read"))
			{
				plugins.GET("", h.ListPlugins)
				plugins.GET("/builtin", h.ListBuiltinPlugins)
				plugins.GET("/:name", h.GetPlugin)
				plugins.POST("", RequirePermissionMiddleware(h.authService.jwtManager, "plugins.write"), h.UploadPlugin)
				plugins.PUT("/:name/enable", RequirePermissionMiddleware(h.authService.jwtManager, "plugins.write"), h.EnablePlugin)
				plugins.PUT("/:name/disable", RequirePermissionMiddleware(h.authService.jwtManager, "plugins.write"), h.DisablePlugin)
				plugins.PUT("/:name/config", RequirePermissionMiddleware(h.authService.jwtManager, "plugins.write"), h.UpdatePluginConfig)
				plugins.GET("/:name/download", h.DownloadPlugin)
				plugins.GET("/:name/logs", h.GetPluginLogs)
				plugins.POST("/:name/test", h.TestPlugin)
			}
		}

		// 计费管理（需要管理员权限）
		if h.billingHandlers != nil {
			billing := admin.Group("/billing")
			billing.Use(RequirePermissionMiddleware(h.authService.jwtManager, "billing.read"))
			{
				// 用量查询
				billing.GET("/usage", h.GetBillingUsage)

				// 计费规则管理
				billing.GET("/rules", h.ListBillingRules)
				billing.GET("/rules/:id", h.GetBillingRule)
				billing.POST("/rules", RequirePermissionMiddleware(h.authService.jwtManager, "billing.write"), h.CreateBillingRule)
				billing.PUT("/rules/:id", RequirePermissionMiddleware(h.authService.jwtManager, "billing.write"), h.UpdateBillingRule)
				billing.DELETE("/rules/:id", RequirePermissionMiddleware(h.authService.jwtManager, "billing.write"), h.DeleteBillingRule)

				// 账单管理
				billing.GET("/invoices", h.ListInvoices)
				billing.GET("/invoices/export", h.StreamInvoicesCSV)
				billing.POST("/invoices/generate", RequirePermissionMiddleware(h.authService.jwtManager, "billing.write"), h.GenerateInvoice)
				billing.GET("/invoices/:id", h.GetInvoice)
				billing.GET("/invoices/:id/export", h.ExportInvoice)
				billing.PUT("/invoices/:id/status", RequirePermissionMiddleware(h.authService.jwtManager, "billing.write"), h.UpdateInvoiceStatus)

				// 付款管理
				billing.POST("/invoices/:id/payments", RequirePermissionMiddleware(h.authService.jwtManager, "billing.write"), h.CreatePayment)
			}
		}

		// 统计管理（需要管理员权限）
		if h.statsHandlers != nil {
			stats := admin.Group("/stats")
			stats.Use(RequirePermissionMiddleware(h.authService.jwtManager, "admin"))
			{
				stats.GET("/realtime", h.statsHandlers.GetRealtimeStats)
				stats.GET("/daily", h.statsHandlers.GetDailyStats)
				stats.GET("/models", h.statsHandlers.GetModelStats)
				stats.GET("/threshold", h.statsHandlers.GetThreshold)
				stats.PUT("/threshold", h.statsHandlers.UpdateThreshold)
			}
		}

		// 成本管理（需要管理员权限）
		if h.costHandlers != nil {
			cost := admin.Group("/cost")
			cost.Use(RequirePermissionMiddleware(h.authService.jwtManager, "admin"))
			{
				// 成本配置
				cost.GET("/configs", h.costHandlers.ListCostConfigs)
				cost.GET("/configs/model/:model_id", h.costHandlers.GetModelCostConfigs)
				cost.GET("/configs/model/:model_id/active", h.costHandlers.GetActiveCostConfig)
				cost.POST("/configs", RequirePermissionMiddleware(h.authService.jwtManager, "admin"), h.costHandlers.CreateCostConfig)
				cost.PUT("/configs/:id", RequirePermissionMiddleware(h.authService.jwtManager, "admin"), h.costHandlers.UpdateCostConfig)
				cost.DELETE("/configs/:id", RequirePermissionMiddleware(h.authService.jwtManager, "admin"), h.costHandlers.DeleteCostConfig)

				// 成本统计
				cost.GET("/stats/daily", h.costHandlers.GetDailyCostStats)
				cost.GET("/stats/daily/model/:model_alias", h.costHandlers.GetDailyCostStatsByModel)
				cost.GET("/stats/summary", h.costHandlers.GetCostSummary)
				cost.POST("/stats/refresh", RequirePermissionMiddleware(h.authService.jwtManager, "admin"), h.costHandlers.RefreshCostStats)
				cost.POST("/stats/calculate", RequirePermissionMiddleware(h.authService.jwtManager, "admin"), h.costHandlers.CalculateDailyCosts)
			}
		}

		// 系统状态
		admin.GET("/health", h.GetHealth)
	}

	// 公开 API（无需认证，供前端仪表盘使用）
	api := r.Group("/api")
	{
		// 统计数据（公开）
		if h.statsHandlers != nil {
			api.GET("/stats/overview", h.statsHandlers.GetPublicStats)
			api.GET("/stats/daily-chart", h.statsHandlers.GetDailyChart)
		}

		// 成本统计（公开）
		if h.costHandlers != nil {
			api.GET("/cost/stats", h.costHandlers.GetPublicCostStats)
		}
	}
}

// LoginHandler 登录处理器（委托给 authHandler）
func (h *APIHandlers) LoginHandler(c *gin.Context) {
	h.authHandler.LoginHandler(c)
}

// LogoutHandler 登出处理器（委托给 authHandler）
func (h *APIHandlers) LogoutHandler(c *gin.Context) {
	h.authHandler.LogoutHandler(c)
}

// MeHandler 获取当前用户信息（委托给 authHandler）
func (h *APIHandlers) MeHandler(c *gin.Context) {
	h.authHandler.MeHandler(c)
}

// GetModels 获取模型列表
func (h *APIHandlers) GetModels(c *gin.Context) {
	activeOnly := c.DefaultQuery("active", "true") == "true"

	models, err := h.modelService.List(c.Request.Context(), activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": models,
	})
}

// CreateModel 创建模型
func (h *APIHandlers) CreateModel(c *gin.Context) {
	var req CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	model, err := h.modelService.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 通知模型配置变更
	h.notifyModelChanged()

	c.JSON(http.StatusCreated, gin.H{
		"data": model,
		"message": "模型配置已保存并生效",
	})
}

// UpdateModel 更新模型
func (h *APIHandlers) UpdateModel(c *gin.Context) {
	id := c.Param("id")

	var req UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	model, err := h.modelService.Update(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 通知模型配置变更
	h.notifyModelChanged()

	c.JSON(http.StatusOK, gin.H{
		"data": model,
		"message": "模型配置已更新并生效",
	})
}

// DeleteModel 删除模型
func (h *APIHandlers) DeleteModel(c *gin.Context) {
	id := c.Param("id")

	// TODO: 检查是否有 API Key 绑定此模型

	if err := h.modelService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 通知模型配置变更
	h.notifyModelChanged()

	c.JSON(http.StatusOK, gin.H{
		"message": "模型已删除",
	})
}

// TestConnection 测试模型连接
func (h *APIHandlers) TestConnection(c *gin.Context) {
	var req TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.modelService.TestConnection(c.Request.Context(), &req)

	if !result.OK {
		c.JSON(http.StatusOK, gin.H{
			"ok":    false,
			"error": result.Error,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":         true,
		"latency_ms": result.LatencyMs,
	})
}

// ToggleModel 切换模型启用状态
func (h *APIHandlers) ToggleModel(c *gin.Context) {
	id := c.Param("id")
	isActive := c.DefaultQuery("active", "true") == "true"

	if err := h.modelService.ToggleActive(c.Request.Context(), id, isActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 通知模型配置变更
	h.notifyModelChanged()

	status := "已禁用"
	if isActive {
		status = "已启用"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": status,
	})
}

// GetAPIKeys 获取 API Key 列表
func (h *APIHandlers) GetAPIKeys(c *gin.Context) {
	filter := &APIKeyFilter{
		Status:    c.DefaultQuery("status", ""),
		Limit:     100,
		Offset:    0,
	}

	keys, err := h.apiKeyService.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": keys,
	})
}

// CreateAPIKey 创建 API Key
func (h *APIHandlers) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey, err := h.apiKeyService.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": apiKey,
		"message": "API Key 已创建，请复制保存",
	})
}

// RevokeAPIKey 吊销 API Key
func (h *APIHandlers) RevokeAPIKey(c *gin.Context) {
	id := c.Param("id")

	if err := h.apiKeyService.Revoke(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API Key 已吊销",
	})
}

// GetAPIKey 获取 API Key 详情
func (h *APIHandlers) GetAPIKey(c *gin.Context) {
	id := c.Param("id")

	apiKey, err := h.apiKeyService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API Key not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": apiKey,
	})
}

// UpdateAPIKey 更新 API Key（仅配额字段）
func (h *APIHandlers) UpdateAPIKey(c *gin.Context) {
	id := c.Param("id")

	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey, err := h.apiKeyService.Update(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    apiKey,
		"message": "API Key 已更新",
	})
}

// GetAPIKeyQuota 获取 API Key 配额状态
func (h *APIHandlers) GetAPIKeyQuota(c *gin.Context) {
	id := c.Param("id")

	// 先获取 API Key 信息（包含配额限制）
	apiKey, err := h.apiKeyService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API Key not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"key_id":                 apiKey.ID,
			"label":                  apiKey.Label,
			"daily_token_soft_limit": apiKey.DailyTokenSoftLimit,
			"daily_token_hard_limit": apiKey.DailyTokenHardLimit,
			"priority":               apiKey.Priority,
		},
	})
}

// GetAPIKeyUsage 获取 API Key 使用统计
func (h *APIHandlers) GetAPIKeyUsage(c *gin.Context) {
	keyID := c.Param("id")

	if h.requestLogStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "log service not available"})
		return
	}

	stats, err := h.requestLogStore.GetKeyUsageStats(c.Request.Context(), keyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// GetUsage 获取用量统计
func (h *APIHandlers) GetUsage(c *gin.Context) {
	if h.requestLogStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "log service not available"})
		return
	}

	// 构建筛选条件
	filter := &UsageStatsFilter{}

	// 按 key_id 筛选
	if keyID := c.Query("key_id"); keyID != "" {
		filter.KeyID = &keyID
	}

	// 按模型别名筛选
	if modelAlias := c.Query("model_alias"); modelAlias != "" {
		filter.ModelAlias = &modelAlias
	}

	// 按时间范围筛选
	if fromStr := c.Query("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.From = &from
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.To = &to
		}
	}

	stats, err := h.requestLogStore.GetUsageStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// GetLogs 获取请求日志
func (h *APIHandlers) GetLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}
	if page < 1 {
		page = 1
	}

	// 构建筛选条件
	filter := &RequestLogFilter{
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	// 按 key_id 筛选
	if keyID := c.Query("key_id"); keyID != "" {
		filter.KeyID = &keyID
	}

	// 按模型别名筛选
	if modelAlias := c.Query("model_alias"); modelAlias != "" {
		filter.ModelAlias = &modelAlias
	}

	// 按状态码筛选
	if statusCodeStr := c.Query("status_code"); statusCodeStr != "" {
		if statusCode, err := strconv.Atoi(statusCodeStr); err == nil {
			filter.StatusCode = &statusCode
		}
	}

	// 按时间范围筛选
	if fromStr := c.Query("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.From = &from
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.To = &to
		}
	}

	if h.requestLogStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "log service not available"})
		return
	}

	logs, total, err := h.requestLogStore.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// ExportLogs 导出日志为 CSV
func (h *APIHandlers) ExportLogs(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=request_logs_%s.csv", time.Now().Format("20060102_150405")))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// 写入 CSV 表头
	headers := []string{
		"Timestamp", "Model Alias", "Model Actual",
		"Prompt Tokens", "Completion Tokens", "Total Tokens",
		"Latency (ms)", "Status Code", "Error Code",
		"Request ID", "IP Address",
	}
	writer.Write(headers)

	if h.requestLogStore == nil {
		// 无数据时只返回表头
		return
	}

	// 构建筛选条件（与 GetLogs 相同）
	filter := &RequestLogFilter{
		Limit: 10000, // 导出时限制最多 10000 条
	}

	if keyID := c.Query("key_id"); keyID != "" {
		filter.KeyID = &keyID
	}
	if modelAlias := c.Query("model_alias"); modelAlias != "" {
		filter.ModelAlias = &modelAlias
	}
	if statusCodeStr := c.Query("status_code"); statusCodeStr != "" {
		if statusCode, err := strconv.Atoi(statusCodeStr); err == nil {
			filter.StatusCode = &statusCode
		}
	}
	if fromStr := c.Query("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.From = &from
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.To = &to
		}
	}

	// 查询日志数据
	logs, _, err := h.requestLogStore.List(c.Request.Context(), filter)
	if err != nil {
		// 出错时至少返回表头
		return
	}

	// 写入数据行
	for _, log := range logs {
		errorCode := ""
		if log.ErrorCode != nil {
			errorCode = *log.ErrorCode
		}
		modelActual := ""
		if log.ModelActual != nil {
			modelActual = *log.ModelActual
		}
		requestID := ""
		if log.RequestID != nil {
			requestID = *log.RequestID
		}
		ipAddress := ""
		if log.IPAddress != nil {
			ipAddress = *log.IPAddress
		}

		row := []string{
			log.CreatedAt.Format(time.RFC3339),
			log.ModelAlias,
			modelActual,
			strconv.Itoa(log.PromptTokens),
			strconv.Itoa(log.CompletionTokens),
			strconv.Itoa(log.TotalTokens),
			strconv.Itoa(log.LatencyMs),
			strconv.Itoa(log.StatusCode),
			errorCode,
			requestID,
			ipAddress,
		}
		writer.Write(row)
	}
}

// GetHealth 获取系统健康状态
func (h *APIHandlers) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"timestamp": time.Now().Unix(),
		"services": gin.H{
			"database": "ok",
			"router":   "ok",
		},
		"models": []interface{}{}, // TODO: 获取实际模型状态
	})
}

// --- 用户管理处理函数 ---

// ListUsers 列出用户
func (h *APIHandlers) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * limit
	users, total, err := h.authService.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  users,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// GetUser 获取用户详情
func (h *APIHandlers) GetUser(c *gin.Context) {
	id := c.Param("id")
	user, err := h.authService.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}

// CreateUser 创建用户
func (h *APIHandlers) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从 JWT 获取当前用户 ID
	claims, _ := GetClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := h.authService.CreateUser(c.Request.Context(), &req, claims.UserID)
	if err != nil {
		if err == ErrUserExists {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		if err == ErrInvalidRole {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": user})
}

// UpdateUser 更新用户
func (h *APIHandlers) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.UpdateUser(c.Request.Context(), id, &req)
	if err != nil {
		if err == ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if err == ErrInvalidRole {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}

// DeleteUser 删除用户
func (h *APIHandlers) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// 不允许删除自己
	claims, _ := GetClaimsFromContext(c)
	if claims != nil && claims.UserID == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete yourself"})
		return
	}

	if err := h.authService.DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// --- 插件管理处理函数 ---

// ListPlugins 列出所有插件
func (h *APIHandlers) ListPlugins(c *gin.Context) {
	if h.pluginHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Plugin service not available"})
		return
	}
	h.pluginHandlers.ListPlugins(c)
}

// ListBuiltinPlugins 列出内置插件
func (h *APIHandlers) ListBuiltinPlugins(c *gin.Context) {
	if h.pluginHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Plugin service not available"})
		return
	}
	h.pluginHandlers.ListBuiltinPlugins(c)
}

// GetPlugin 获取插件详情
func (h *APIHandlers) GetPlugin(c *gin.Context) {
	if h.pluginHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Plugin service not available"})
		return
	}
	h.pluginHandlers.GetPlugin(c)
}

// UploadPlugin 上传插件
func (h *APIHandlers) UploadPlugin(c *gin.Context) {
	if h.pluginHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Plugin service not available"})
		return
	}
	h.pluginHandlers.UploadPlugin(c)
}

// EnablePlugin 启用插件
func (h *APIHandlers) EnablePlugin(c *gin.Context) {
	if h.pluginHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Plugin service not available"})
		return
	}
	h.pluginHandlers.EnablePlugin(c)
}

// DisablePlugin 禁用插件
func (h *APIHandlers) DisablePlugin(c *gin.Context) {
	if h.pluginHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Plugin service not available"})
		return
	}
	h.pluginHandlers.DisablePlugin(c)
}

// UpdatePluginConfig 更新插件配置
func (h *APIHandlers) UpdatePluginConfig(c *gin.Context) {
	if h.pluginHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Plugin service not available"})
		return
	}
	h.pluginHandlers.UpdatePluginConfig(c)
}

// DownloadPlugin 下载插件文件
func (h *APIHandlers) DownloadPlugin(c *gin.Context) {
	if h.pluginHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Plugin service not available"})
		return
	}
	h.pluginHandlers.DownloadPlugin(c)
}

// GetPluginLogs 获取插件日志
func (h *APIHandlers) GetPluginLogs(c *gin.Context) {
	if h.pluginHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Plugin service not available"})
		return
	}
	h.pluginHandlers.GetPluginLogs(c)
}

// TestPlugin 测试插件
func (h *APIHandlers) TestPlugin(c *gin.Context) {
	if h.pluginHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Plugin service not available"})
		return
	}
	h.pluginHandlers.TestPlugin(c)
}

// --- 计费管理处理函数 ---

// GetBillingUsage 获取计费用量统计
func (h *APIHandlers) GetBillingUsage(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.GetUsageStats(c)
}

// ListBillingRules 列出计费规则
func (h *APIHandlers) ListBillingRules(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.ListRules(c)
}

// GetBillingRule 获取单个计费规则
func (h *APIHandlers) GetBillingRule(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.GetRule(c)
}

// CreateBillingRule 创建计费规则
func (h *APIHandlers) CreateBillingRule(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.CreateRule(c)
}

// UpdateBillingRule 更新计费规则
func (h *APIHandlers) UpdateBillingRule(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.UpdateRule(c)
}

// DeleteBillingRule 删除计费规则
func (h *APIHandlers) DeleteBillingRule(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.DeleteRule(c)
}

// ListInvoices 列出账单
func (h *APIHandlers) ListInvoices(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.ListInvoices(c)
}

// GetInvoice 获取账单详情
func (h *APIHandlers) GetInvoice(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.GetInvoice(c)
}

// GenerateInvoice 生成账单
func (h *APIHandlers) GenerateInvoice(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.GenerateInvoice(c)
}

// UpdateInvoiceStatus 更新账单状态
func (h *APIHandlers) UpdateInvoiceStatus(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.UpdateInvoiceStatus(c)
}

// CreatePayment 创建付款记录
func (h *APIHandlers) CreatePayment(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.CreatePayment(c)
}

// ExportInvoice 导出账单为 CSV
func (h *APIHandlers) ExportInvoice(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.ExportInvoice(c)
}

// StreamInvoicesCSV 流式导出账单列表
func (h *APIHandlers) StreamInvoicesCSV(c *gin.Context) {
	if h.billingHandlers == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Billing service not available"})
		return
	}
	h.billingHandlers.StreamInvoiceCSV(c)
}
