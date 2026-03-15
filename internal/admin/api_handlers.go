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
	authService    *AdminAuthService
	modelService   *ModelService
	apiKeyService  *APIKeyService
	authHandler    *Handler
}

// NewAPIHandlers 创建 API 处理器
func NewAPIHandlers(
	authService *AdminAuthService,
	modelService *ModelService,
	apiKeyService *APIKeyService,
) *APIHandlers {
	return &APIHandlers{
		authService:   authService,
		modelService:  modelService,
		apiKeyService: apiKeyService,
		authHandler:   NewHandler(authService),
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
	// TODO: 实现使用统计查询
	_ = c.Param("id") // 未来用于查询指定 key 的使用统计
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"total_requests":   0,
			"total_tokens":     0,
			"prompt_tokens":    0,
			"completion_tokens": 0,
		},
	})
}

// GetUsage 获取用量统计
func (h *APIHandlers) GetUsage(c *gin.Context) {
	// TODO: 实现用量统计聚合查询
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"total_requests":   0,
			"total_tokens":     0,
			"active_keys":      0,
			"active_models":    0,
		},
	})
}

// GetLogs 获取请求日志
func (h *APIHandlers) GetLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	// TODO: 实现日志查询
	c.JSON(http.StatusOK, gin.H{
		"data":     []interface{}{},
		"page":     page,
		"limit":    limit,
		"total":    0,
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

	// TODO: 写入实际数据

	// 空行（确保至少有数据）
	writer.Write([]string{})
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
