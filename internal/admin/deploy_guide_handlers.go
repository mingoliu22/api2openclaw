package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// DeployGuideHandlers 部署指南处理器
type DeployGuideHandlers struct {
	store *DeployGuideStore
}

// NewDeployGuideHandlers 创建部署指南处理器
func NewDeployGuideHandlers(store *DeployGuideStore) *DeployGuideHandlers {
	return &DeployGuideHandlers{store: store}
}

// ListDeployGuides 列出所有部署指南
// GET /admin/deploy-guides
func (h *DeployGuideHandlers) ListDeployGuides(c *gin.Context) {
	activeOnly := c.DefaultQuery("active", "true") == "true"

	guides, err := h.store.List(c.Request.Context(), activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to list deploy guides",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": guides,
	})
}

// GetDeployGuide 获取单个部署指南
// GET /admin/deploy-guides/:id
func (h *DeployGuideHandlers) GetDeployGuide(c *gin.Context) {
	id := c.Param("id")

	guide, err := h.store.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
			"code":    "not_found",
			"message": "Deploy guide not found",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": guide,
	})
}

// GetFrameworks 获取所有框架列表
// GET /admin/deploy-guides/frameworks
func (h *DeployGuideHandlers) GetFrameworks(c *gin.Context) {
	frameworks, err := h.store.GetFrameworks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get frameworks",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": frameworks,
	})
}

// GetFrameworkModels 获取指定框架的模型列表
// GET /admin/deploy-guides/frameworks/:framework_id/models
func (h *DeployGuideHandlers) GetFrameworkModels(c *gin.Context) {
	frameworkID := c.Param("framework_id")

	models, err := h.store.GetByFramework(c.Request.Context(), frameworkID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get framework models",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": models,
	})
}

// CreateDeployGuideRequest 创建部署指南请求
type CreateDeployGuideRequest struct {
	FrameworkID  string                 `json:"framework_id" binding:"required"`
	ModelID      string                 `json:"model_id" binding:"required"`
	Name         string                 `json:"name" binding:"required"`
	Alias        string                 `json:"alias"`
	InstallCmd   string                 `json:"install_cmd"`
	StartCmd     string                 `json:"start_cmd"`
	Params       map[string]interface{} `json:"params"`
	Features     []string               `json:"features"`
	Requirements map[string]interface{} `json:"requirements"`
	APIPort      int                    `json:"api_port"`
	Tagline      string                 `json:"tagline"`
	Description  string                 `json:"description"`
	Badge        string                 `json:"badge"`
	BadgeColor   string                 `json:"badge_color"`
	AccentColor  string                 `json:"accent_color"`
	Icon         string                 `json:"icon"`
	ModelFamily  string                 `json:"model_family"`
	VRAMReq      string                 `json:"vram_requirement"`
	Precision    string                 `json:"precision"`
	HFID         string                 `json:"hf_id"`
	Steps        []map[string]interface{} `json:"steps"`
	DisplayOrder int                    `json:"display_order"`
	IsActive     *bool                  `json:"is_active"`
}

// CreateDeployGuide 创建部署指南
// POST /admin/deploy-guides
func (h *DeployGuideHandlers) CreateDeployGuide(c *gin.Context) {
	var req CreateDeployGuideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "invalid_request",
			"message": err.Error(),
		}})
		return
	}

	guide := &DeployGuide{
		FrameworkID:      req.FrameworkID,
		ModelID:          req.ModelID,
		Name:             req.Name,
		Alias:            req.Alias,
		InstallCmd:       req.InstallCmd,
		StartCmd:         req.StartCmd,
		Params:           req.Params,
		Features:         req.Features,
		Requirements:     req.Requirements,
		APIPort:          req.APIPort,
		Tagline:          req.Tagline,
		Description:      req.Description,
		Badge:            req.Badge,
		BadgeColor:       req.BadgeColor,
		AccentColor:      req.AccentColor,
		Icon:             req.Icon,
		ModelFamily:      req.ModelFamily,
		VRAMRequirement:  req.VRAMReq,
		Precision:        req.Precision,
		HFID:             req.HFID,
		Steps:            req.Steps,
		DisplayOrder:     req.DisplayOrder,
		IsActive:         true,
	}
	if req.IsActive != nil {
		guide.IsActive = *req.IsActive
	}

	if err := h.store.Create(c.Request.Context(), guide); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to create deploy guide",
		}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": guide,
	})
}

// UpdateDeployGuideRequest 更新部署指南请求
type UpdateDeployGuideRequest struct {
	Name         *string                 `json:"name"`
	Alias        *string                 `json:"alias"`
	InstallCmd   *string                 `json:"install_cmd"`
	StartCmd     *string                 `json:"start_cmd"`
	Params       map[string]interface{} `json:"params"`
	Features     []string               `json:"features"`
	Requirements map[string]interface{} `json:"requirements"`
	APIPort      *int                    `json:"api_port"`
	Tagline      *string                 `json:"tagline"`
	Description  *string                 `json:"description"`
	Badge        *string                 `json:"badge"`
	BadgeColor   *string                 `json:"badge_color"`
	AccentColor  *string                 `json:"accent_color"`
	Icon         *string                 `json:"icon"`
	ModelFamily  *string                 `json:"model_family"`
	VRAMReq      *string                 `json:"vram_requirement"`
	Precision    *string                 `json:"precision"`
	HFID         *string                 `json:"hf_id"`
	Steps        []map[string]interface{} `json:"steps"`
	DisplayOrder *int                    `json:"display_order"`
	IsActive     *bool                   `json:"is_active"`
}

// UpdateDeployGuide 更新部署指南
// PUT /admin/deploy-guides/:id
func (h *DeployGuideHandlers) UpdateDeployGuide(c *gin.Context) {
	id := c.Param("id")

	guide, err := h.store.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
			"code":    "not_found",
			"message": "Deploy guide not found",
		}})
		return
	}

	var req UpdateDeployGuideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "invalid_request",
			"message": err.Error(),
		}})
		return
	}

	// 更新非空字段
	if req.Name != nil {
		guide.Name = *req.Name
	}
	if req.Alias != nil {
		guide.Alias = *req.Alias
	}
	if req.InstallCmd != nil {
		guide.InstallCmd = *req.InstallCmd
	}
	if req.StartCmd != nil {
		guide.StartCmd = *req.StartCmd
	}
	if req.Params != nil {
		guide.Params = req.Params
	}
	if req.Features != nil {
		guide.Features = req.Features
	}
	if req.Requirements != nil {
		guide.Requirements = req.Requirements
	}
	if req.APIPort != nil {
		guide.APIPort = *req.APIPort
	}
	if req.Tagline != nil {
		guide.Tagline = *req.Tagline
	}
	if req.Description != nil {
		guide.Description = *req.Description
	}
	if req.Badge != nil {
		guide.Badge = *req.Badge
	}
	if req.BadgeColor != nil {
		guide.BadgeColor = *req.BadgeColor
	}
	if req.AccentColor != nil {
		guide.AccentColor = *req.AccentColor
	}
	if req.Icon != nil {
		guide.Icon = *req.Icon
	}
	if req.ModelFamily != nil {
		guide.ModelFamily = *req.ModelFamily
	}
	if req.VRAMReq != nil {
		guide.VRAMRequirement = *req.VRAMReq
	}
	if req.Precision != nil {
		guide.Precision = *req.Precision
	}
	if req.HFID != nil {
		guide.HFID = *req.HFID
	}
	if req.Steps != nil {
		guide.Steps = req.Steps
	}
	if req.DisplayOrder != nil {
		guide.DisplayOrder = *req.DisplayOrder
	}
	if req.IsActive != nil {
		guide.IsActive = *req.IsActive
	}

	if err := h.store.Update(c.Request.Context(), guide); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to update deploy guide",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": guide,
	})
}

// DeleteDeployGuide 删除部署指南
// DELETE /admin/deploy-guides/:id
func (h *DeployGuideHandlers) DeleteDeployGuide(c *gin.Context) {
	id := c.Param("id")

	if err := h.store.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to delete deploy guide",
		}})
		return
	}

	c.Status(http.StatusNoContent)
}

// QueryXinferenceUID 查询 Xinference 模型 UID
// GET /admin/deploy-guides/xinference/uid
func (h *DeployGuideHandlers) QueryXinferenceUID(c *gin.Context) {
	host := c.DefaultQuery("host", "localhost:9997")

	// 调用 Xinference API
	resp, err := http.Get("http://" + host + "/v1/models")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "connection_failed",
			"message": "Failed to connect to Xinference: " + err.Error(),
		}})
		return
	}
	defer resp.Body.Close()

	var result struct {
		Object string `json:"object"`
		Data   []struct {
			ID   string `json:"id"`
			Object string `json:"object"`
			ModelName string `json:"model_name"`
			ModelType string `json:"model_type"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "parse_failed",
			"message": "Failed to parse Xinference response",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result.Data,
	})
}

// GetPublicDeployGuides 公开 API：获取部署指南（供前端使用）
// GET /api/deploy-guides
func (h *DeployGuideHandlers) GetPublicDeployGuides(c *gin.Context) {
	guides, err := h.store.List(c.Request.Context(), true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "internal_error",
			"message": "Failed to get deploy guides",
		}})
		return
	}

	// 按框架组织数据
	frameworkMap := make(map[string]interface{})
	for _, guide := range guides {
		if guide.ModelID == "_framework" {
			// 框架记录
			frameworkMap[guide.FrameworkID] = map[string]interface{}{
				"id":           guide.ID,
				"framework_id": guide.FrameworkID,
				"name":         guide.Name,
				"tagline":      guide.Tagline,
				"description":  guide.Description,
				"badge":        guide.Badge,
				"badge_color":  guide.BadgeColor,
				"accent_color": guide.AccentColor,
				"icon":         guide.Icon,
				"features":     guide.Features,
				"requirements": guide.Requirements,
				"api_port":     guide.APIPort,
				"install": map[string]interface{}{
					"code": guide.InstallCmd,
				},
				"models": []interface{}{},
			}
		}
	}

	// 添加模型到框架
	for _, guide := range guides {
		if guide.ModelID != "_framework" {
			if fw, ok := frameworkMap[guide.FrameworkID].(map[string]interface{}); ok {
				models := fw["models"].([]interface{})
				models = append(models, map[string]interface{}{
					"id":                guide.ID,
					"model_id":          guide.ModelID,
					"name":              guide.Name,
					"alias":             guide.Alias,
					"model_family":      guide.ModelFamily,
					"vram_requirement":  guide.VRAMRequirement,
					"precision":         guide.Precision,
					"hf_id":             guide.HFID,
					"start_cmd":         guide.StartCmd,
					"steps":             guide.Steps,
					"display_order":     guide.DisplayOrder,
				})
				fw["models"] = models
			}
		}
	}

	// 转换为数组
	frameworks := make([]interface{}, 0, len(frameworkMap))
	order := []string{"vllm", "sglang", "xinference", "ollama"}
	for _, id := range order {
		if fw, ok := frameworkMap[id]; ok {
			frameworks = append(frameworks, fw)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": frameworks,
	})
}
