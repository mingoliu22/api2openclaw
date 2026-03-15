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
		admin.DELETE("/keys/:id", h.RevokeAPIKey)
		admin.GET("/keys/:id", h.GetAPIKey)
		admin.GET("/keys/:id/usage", h.GetAPIKeyUsage)

		// 用量与日志
		admin.GET("/usage", h.GetUsage)
		admin.GET("/logs", h.GetLogs)
		admin.GET("/logs/export", h.ExportLogs)

		// 系统状态
		admin.GET("/health", h.GetHealth)
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
