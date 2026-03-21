package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// StatsHandlers 统计数据处理器
type StatsHandlers struct {
	store *StatsStore
}

// NewStatsHandlers 创建统计数据处理器
func NewStatsHandlers(store *StatsStore) *StatsHandlers {
	return &StatsHandlers{store: store}
}

// GetRealtimeStats 获取实时统计数据
// GET /admin/stats/realtime
func (h *StatsHandlers) GetRealtimeStats(c *gin.Context) {
	threshold, err := h.store.GetThreshold(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get threshold",
		}})
		return
	}

	stats, err := h.store.GetRealtimeStats(c.Request.Context(), threshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get realtime stats",
		}})
		return
	}

	// 设置 10s 缓存头
	c.Header("Cache-Control", "max-age=10")
	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// GetDailyStats 获取每日统计数据
// GET /admin/stats/daily?date=YYYY-MM-DD
func (h *StatsHandlers) GetDailyStats(c *gin.Context) {
	date := c.DefaultQuery("date", "")
	if date == "" {
		date = "today"
	}

	// 支持 "today" 转换为当前日期
	if date == "today" {
		date = "CURRENT_DATE"
	}

	stats, err := h.store.GetDailyStats(c.Request.Context(), date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get daily stats",
		}})
		return
	}

	// 设置 5 分钟缓存
	c.Header("Cache-Control", "max-age=300")
	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// GetModelStats 获取模型统计数据
// GET /admin/stats/models
func (h *StatsHandlers) GetModelStats(c *gin.Context) {
	stats, err := h.store.GetModelStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get model stats",
		}})
		return
	}

	c.Header("Cache-Control", "max-age=30")
	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// GetThreshold 获取预警阈值
// GET /admin/stats/threshold
func (h *StatsHandlers) GetThreshold(c *gin.Context) {
	threshold, err := h.store.GetThreshold(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get threshold",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"threshold": threshold,
		},
	})
}

// UpdateThreshold 更新预警阈值
// PUT /admin/stats/threshold
func (h *StatsHandlers) UpdateThreshold(c *gin.Context) {
	var req struct {
		Threshold int `json:"threshold" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "invalid_request",
			"message": err.Error(),
		}})
		return
	}

	if err := h.store.UpdateThreshold(c.Request.Context(), req.Threshold); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to update threshold",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"threshold": req.Threshold,
			"message": "阈值已更新",
		},
	})
}

// GetPublicStats 公开 API：获取统计数据（供前端仪表盘使用）
// GET /api/stats/overview
func (h *StatsHandlers) GetPublicStats(c *gin.Context) {
	threshold, _ := h.store.GetThreshold(c.Request.Context())

	stats, err := h.store.GetRealtimeStats(c.Request.Context(), threshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get stats",
		}})
		return
	}

	// 同时获取模型分布
	modelStats, err := h.store.GetModelStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get model stats",
		}})
		return
	}

	c.Header("Cache-Control", "max-age=10")
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"realtime": stats,
			"models":   modelStats,
		},
	})
}

// GetDailyChart 获取趋势图数据（公开 API）
// GET /api/stats/daily-chart?days=7
func (h *StatsHandlers) GetDailyChart(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 30 {
		days = 7
	}

	// 根据天数查询日期范围
	query := `
		SELECT
			TO_CHAR(stat_date, 'YYYY-MM-DD') as date,
			SUM(total_tokens) as tokens
		FROM stats_hourly
		WHERE stat_date >= CURRENT_DATE - INTERVAL '%d days'
		GROUP BY stat_date
		ORDER BY stat_date
	`

	rows, err := h.store.db.QueryContext(c.Request.Context(), query, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get chart data",
		}})
		return
	}
	defer rows.Close()

	type ChartData struct {
		Date   string `json:"date"`
		Tokens int64  `json:"tokens"`
	}

	var chartData []ChartData
	for rows.Next() {
		var d ChartData
		if err := rows.Scan(&d.Date, &d.Tokens); err != nil {
			continue
		}
		chartData = append(chartData, d)
	}

	c.Header("Cache-Control", "max-age=300")
	c.JSON(http.StatusOK, gin.H{
		"data": chartData,
	})
}
