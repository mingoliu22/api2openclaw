package admin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/openclaw/api2openclaw/internal/converter"
)

// PluginHandlers 插件管理处理器
type PluginHandlers struct {
	pluginManager *converter.PluginManager
	pluginDir     string
}

// NewPluginHandlers 创建插件管理处理器
func NewPluginHandlers(pluginManager *converter.PluginManager, pluginDir string) *PluginHandlers {
	return &PluginHandlers{
		pluginManager: pluginManager,
		pluginDir:     pluginDir,
	}
}

// PluginInfo 插件信息
type PluginInfo struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Path     string                 `json:"path,omitempty"`
	Enabled  bool                   `json:"enabled"`
	Config   map[string]interface{} `json:"config"`
	Version  string                 `json:"version,omitempty"`
}

// ListPlugins 列出所有插件
func (h *PluginHandlers) ListPlugins(c *gin.Context) {
	registry := h.pluginManager.GetRegistry()
	loadedPlugins := registry.List()

	// 获取所有已配置的插件信息
	plugins := make([]PluginInfo, 0)

	for _, name := range loadedPlugins {
		plugin, _ := registry.Get(name)
		if plugin != nil {
			plugins = append(plugins, PluginInfo{
				Name:    plugin.Name(),
				Type:    "builtin",
				Enabled: true,
				Version: plugin.Version(),
			})
		}
	}

	// TODO: 从配置文件或数据库获取已配置但未启用的插件
	// 当前仅返回已加载的插件

	c.JSON(http.StatusOK, gin.H{
		"data": plugins,
	})
}

// GetPlugin 获取单个插件详情
func (h *PluginHandlers) GetPlugin(c *gin.Context) {
	name := c.Param("name")

	registry := h.pluginManager.GetRegistry()
	plugin, exists := registry.Get(name)

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	info := PluginInfo{
		Name:    plugin.Name(),
		Type:    "builtin",
		Enabled: true,
		Version: plugin.Version(),
	}

	c.JSON(http.StatusOK, gin.H{"data": info})
}

// UploadPlugin 上传插件文件
func (h *PluginHandlers) UploadPlugin(c *gin.Context) {
	file, err := c.FormFile("plugin")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// 检查文件扩展名
	if !strings.HasSuffix(file.Filename, ".so") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only .so files are supported"})
		return
	}

	// 确保插件目录存在
	if err := os.MkdirAll(h.pluginDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plugin directory"})
		return
	}

	// 保存文件
	destPath := filepath.Join(h.pluginDir, file.Filename)
	if err := c.SaveUploadedFile(file, destPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// 获取插件配置参数
	pluginName := c.PostForm("name")
	if pluginName == "" {
		pluginName = strings.TrimSuffix(file.Filename, ".so")
	}

	symbol := c.PostForm("symbol")
	if symbol == "" {
		symbol = "New" + strings.Title(pluginName) + "Plugin"
	}

	// 创建插件配置
	config := &converter.PluginConfig{
		Name:    pluginName,
		Type:    "so",
		Path:    destPath,
		Symbol:  symbol,
		Enabled: false,
		Config:  make(map[string]interface{}),
	}

	// 解析额外配置参数
	if configStr := c.PostForm("config"); configStr != "" {
		if err := json.Unmarshal([]byte(configStr), &config.Config); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config JSON"})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"name":     config.Name,
			"type":     config.Type,
			"path":     config.Path,
			"symbol":   symbol,
			"enabled":  config.Enabled,
			"config":   config.Config,
			"filename": file.Filename,
		},
	})
}

// EnablePlugin 启用插件
func (h *PluginHandlers) EnablePlugin(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		Config map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: 从数据库/配置文件加载插件配置
	// 当前简化实现：直接加载内置插件

	if name == "deepseek" || name == "openai" || name == "openclaw" {
		config := &converter.PluginConfig{
			Name:    name,
			Type:    "builtin",
			Enabled: true,
			Config:  req.Config,
		}

		if err := h.pluginManager.GetRegistry().Register(
			converter.NewBuiltinDeepSeekPlugin(),
			req.Config,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Plugin enabled"})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
}

// DisablePlugin 禁用插件
func (h *PluginHandlers) DisablePlugin(c *gin.Context) {
	name := c.Param("name")

	registry := h.pluginManager.GetRegistry()
	if err := registry.Unregister(name); err != nil {
		if err.Error() == "plugin not found: "+name {
			c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Plugin disabled"})
}

// UpdatePluginConfig 更新插件配置
func (h *PluginHandlers) UpdatePluginConfig(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		Config map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: 更新插件配置并重新加载
	// 当前简化实现：仅返回成功
	c.JSON(http.StatusOK, gin.H{"message": "Plugin config updated"})
}

// DownloadPlugin 下载插件文件
func (h *PluginHandlers) DownloadPlugin(c *gin.Context) {
	name := c.Param("name")
	filePath := filepath.Join(h.pluginDir, name+".so")

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin file not found"})
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+name+".so")
	c.Header("Content-Type", "application/octet-stream")

	c.File(filePath)
}

// GetPluginLogs 获取插件日志
func (h *PluginHandlers) GetPluginLogs(c *gin.Context) {
	name := c.Param("name")

	// TODO: 实现插件日志收集
	// 当前简化实现：返回空日志
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"plugin": name,
			"logs":   []string{},
		},
	})
}

// TestPlugin 测试插件
func (h *PluginHandlers) TestPlugin(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		InputFormat  string                 `json:"input_format"`
		OutputFormat string                 `json:"output_format"`
		TestData     string                 `json:"test_data"`
		Config       map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	registry := h.pluginManager.GetRegistry()
	plugin, exists := registry.Get(name)

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	// 测试转换
	output, err := plugin.Convert([]byte(req.TestData))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"success": false,
				"error":   err.Error(),
				"output":  nil,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"success": true,
			"output":  string(output),
		},
	})
}

// ListBuiltinPlugins 列出内置插件
func (h *PluginHandlers) ListBuiltinPlugins(c *gin.Context) {
	builtinPlugins := []gin.H{
		{
			"name":        "deepseek",
			"type":        "builtin",
			"description": "DeepSeek 格式转换器",
			"version":     "1.0.0",
			"author":      "api2openclaw",
			"input_formats": []string{"deepseek"},
			"output_formats": []string{"openclaw", "json"},
		},
		{
			"name":        "openai",
			"type":        "builtin",
			"description": "OpenAI 格式转换器",
			"version":     "1.0.0",
			"author":      "api2openclaw",
			"input_formats": []string{"openai-json"},
			"output_formats": []string{"openclaw", "json"},
		},
		{
			"name":        "openclaw",
			"type":        "builtin",
			"description": "OpenClaw 透传转换器",
			"version":     "1.0.0",
			"author":      "api2openclaw",
			"input_formats": []string{"openclaw"},
			"output_formats": []string{"openclaw"},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"data": builtinPlugins,
	})
}

// SaveUploadedFile 保存上传的文件（辅助方法）
func (h *PluginHandlers) SaveUploadedFile(file *gin.Context.FileHeader, dest string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}
