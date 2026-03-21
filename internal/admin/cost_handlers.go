package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CostHandlers 成本管理处理器
type CostHandlers struct {
	store *CostStore
}

// NewCostHandlers 创建成本管理处理器
func NewCostHandlers(store *CostStore) *CostHandlers {
	return &CostHandlers{store: store}
}

// ListCostConfigs 列出所有成本配置
// GET /admin/cost/configs
func (h *CostHandlers) ListCostConfigs(c *gin.Context) {
	configs, err := h.store.GetAllCostConfigs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to list cost configs",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": configs,
	})
}

// GetModelCostConfigs 获取指定模型的成本配置
// GET /admin/cost/configs/model/:model_id
func (h *CostHandlers) GetModelCostConfigs(c *gin.Context) {
	modelID := c.Param("model_id")

	configs, err := h.store.ListModelCostConfigs(c.Request.Context(), modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get model cost configs",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": configs,
	})
}

// GetActiveCostConfig 获取模型当前生效的成本配置
// GET /admin/cost/configs/model/:model_id/active
func (h *CostHandlers) GetActiveCostConfig(c *gin.Context) {
	modelID := c.Param("model_id")

	config, err := h.store.GetActiveCostConfig(c.Request.Context(), modelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
			"code":    "not_found",
			"message": "No active cost config found",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": config,
	})
}

// CreateCostConfigRequest 创建成本配置请求
type CreateCostConfigRequest struct {
	ModelID                string    `json:"model_id" binding:"required"`
	GPUCount               int       `json:"gpu_count" binding:"required,min=1"`
	PowerPerGPUW           int       `json:"power_per_gpu_w" binding:"required,min=1"`
	ElectricityPricePerKWh float64   `json:"electricity_price_per_kwh" binding:"required,min=0"`
	DepreciationPerGPUMonth int      `json:"depreciation_per_gpu_month" binding:"required,min=0"`
	PUE                    float64   `json:"pue" binding:"required,min=1"`
	EffectiveFrom          string    `json:"effective_from"`
}

// CreateCostConfig 创建成本配置
// POST /admin/cost/configs
func (h *CostHandlers) CreateCostConfig(c *gin.Context) {
	var req CreateCostConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "invalid_request",
			"message": err.Error(),
		}})
		return
	}

	config := &ModelCostConfig{
		ModelID:                req.ModelID,
		GPUCount:               req.GPUCount,
		PowerPerGPUW:           req.PowerPerGPUW,
		ElectricityPricePerKWh: req.ElectricityPricePerKWh,
		DepreciationPerGPUMonth: req.DepreciationPerGPUMonth,
		PUE:                    req.PUE,
	}

	// 解析生效时间
	if req.EffectiveFrom != "" {
		t, err := time.Parse(time.RFC3339, req.EffectiveFrom)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"code":    "invalid_format",
				"message": "Invalid effective_from format, use RFC3339",
			}})
			return
		}
		config.EffectiveFrom = t
	} else {
		config.EffectiveFrom = time.Now()
	}

	if err := h.store.CreateModelCostConfig(c.Request.Context(), config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to create cost config",
		}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    config,
		"message": "成本配置已创建",
	})
}

// UpdateCostConfigRequest 更新成本配置请求
type UpdateCostConfigRequest struct {
	GPUCount               *int     `json:"gpu_count"`
	PowerPerGPUW           *int     `json:"power_per_gpu_w"`
	ElectricityPricePerKWh *float64 `json:"electricity_price_per_kwh"`
	DepreciationPerGPUMonth *int    `json:"depreciation_per_gpu_month"`
	PUE                    *float64 `json:"pue"`
}

// UpdateCostConfig 更新成本配置
// PUT /admin/cost/configs/:id
func (h *CostHandlers) UpdateCostConfig(c *gin.Context) {
	id := c.Param("id")

	// 获取现有配置
	configs, err := h.store.GetAllCostConfigs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get cost config",
		}})
		return
	}

	var config *ModelCostConfig
	for _, cfg := range configs {
		if cfg.ID == id {
			config = &cfg
			break
		}
	}

	if config == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
			"code":    "not_found",
			"message": "Cost config not found",
		}})
		return
	}

	var req UpdateCostConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "invalid_request",
			"message": err.Error(),
		}})
		return
	}

	// 更新非空字段
	if req.GPUCount != nil {
		config.GPUCount = *req.GPUCount
	}
	if req.PowerPerGPUW != nil {
		config.PowerPerGPUW = *req.PowerPerGPUW
	}
	if req.ElectricityPricePerKWh != nil {
		config.ElectricityPricePerKWh = *req.ElectricityPricePerKWh
	}
	if req.DepreciationPerGPUMonth != nil {
		config.DepreciationPerGPUMonth = *req.DepreciationPerGPUMonth
	}
	if req.PUE != nil {
		config.PUE = *req.PUE
	}

	if err := h.store.UpdateModelCostConfig(c.Request.Context(), id, config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to update cost config",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    config,
		"message": "成本配置已更新",
	})
}

// DeleteCostConfig 删除成本配置
// DELETE /admin/cost/configs/:id
func (h *CostHandlers) DeleteCostConfig(c *gin.Context) {
	id := c.Param("id")

	if err := h.store.DeleteModelCostConfig(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to delete cost config",
		}})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetDailyCostStats 获取每日成本统计
// GET /admin/cost/stats/daily?days=30
func (h *CostHandlers) GetDailyCostStats(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 90 {
		days = 30
	}

	stats, err := h.store.GetDailyCostStats(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get daily cost stats",
		}})
		return
	}

	// 设置 5 分钟缓存
	c.Header("Cache-Control", "max-age=300")
	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// GetDailyCostStatsByModel 获取指定模型的每日成本统计
// GET /admin/cost/stats/daily/model/:model_alias?days=30
func (h *CostHandlers) GetDailyCostStatsByModel(c *gin.Context) {
	modelAlias := c.Param("model_alias")
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 90 {
		days = 30
	}

	stats, err := h.store.GetDailyCostStatsByModel(c.Request.Context(), modelAlias, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get daily cost stats",
		}})
		return
	}

	c.Header("Cache-Control", "max-age=300")
	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// GetCostSummary 获取成本汇总数据
// GET /admin/cost/stats/summary?days=30
func (h *CostHandlers) GetCostSummary(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 365 {
		days = 30
	}

	summary, err := h.store.GetCostSummary(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get cost summary",
		}})
		return
	}

	c.Header("Cache-Control", "max-age=60")
	c.JSON(http.StatusOK, gin.H{
		"data": summary,
	})
}

// RefreshCostStats 手动触发成本统计刷新
// POST /admin/cost/stats/refresh
func (h *CostHandlers) RefreshCostStats(c *gin.Context) {
	if err := h.store.RefreshCostStats(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to refresh cost stats",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "成本统计已刷新",
	})
}

// CalculateDailyCosts 触发指定日期的成本计算
// POST /admin/cost/stats/calculate
func (h *CostHandlers) CalculateDailyCosts(c *gin.Context) {
	var req struct {
		StatDate string `json:"stat_date" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "invalid_request",
			"message": err.Error(),
		}})
		return
	}

	// 验证日期格式
	_, err := time.Parse("2006-01-02", req.StatDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "invalid_format",
			"message": "Invalid date format, use YYYY-MM-DD",
		}})
		return
	}

	if err := h.store.CalculateDailyCosts(c.Request.Context(), req.StatDate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to calculate daily costs",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "成本计算已完成",
		"data": gin.H{
			"stat_date": req.StatDate,
		},
	})
}

// GetPublicCostStats 公开 API：获取成本统计（供前端仪表盘使用）
// GET /api/cost/stats
func (h *CostHandlers) GetPublicCostStats(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 30 {
		days = 7
	}

	// 获取成本汇总
	summary, err := h.store.GetCostSummary(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get cost summary",
		}})
		return
	}

	// 获取每日成本统计
	dailyStats, err := h.store.GetDailyCostStats(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get daily stats",
		}})
		return
	}

	c.Header("Cache-Control", "max-age=60")
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"summary": summary,
			"daily":   dailyStats,
		},
	})
}
