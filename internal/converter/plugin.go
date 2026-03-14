package converter

import (
	"encoding/json"
	"fmt"
	"io"
	"plugin"
	"sync"
)

// ConverterPlugin 转换器插件接口
type ConverterPlugin interface {
	// Name 插件名称
	Name() string

	// Version 插件版本
	Version() string

	// Init 初始化插件
	Init(config map[string]interface{}) error

	// Convert 转换内容
	Convert(data []byte) ([]byte, error)

	// ConvertStream 转换流式内容
	ConvertStream(r io.Reader, w io.Writer) error

	// Supports 检查是否支持指定格式
	Supports(inputFormat, outputFormat string) bool

	// Cleanup 清理资源
	Cleanup() error
}

// PluginRegistry 插件注册表
type PluginRegistry struct {
	mu      sync.RWMutex
	plugins map[string]ConverterPlugin
	configs map[string]map[string]interface{}
}

// NewPluginRegistry 创建插件注册表
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]ConverterPlugin),
		configs: make(map[string]map[string]interface{}),
	}
}

// Register 注册插件
func (r *PluginRegistry) Register(plugin ConverterPlugin, config map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := plugin.Name()
	if err := plugin.Init(config); err != nil {
		return fmt.Errorf("plugin init failed: %w", err)
	}

	r.plugins[name] = plugin
	r.configs[name] = config

	return nil
}

// Unregister 注销插件
func (r *PluginRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, ok := r.plugins[name]
	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	if err := plugin.Cleanup(); err != nil {
		return err
	}

	delete(r.plugins, name)
	delete(r.configs, name)

	return nil
}

// Get 获取插件
func (r *PluginRegistry) Get(name string) (ConverterPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.plugins[name]
	return plugin, ok
}

// List 列出所有插件
func (r *PluginRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}

// Find 查找支持指定格式的插件
func (r *PluginRegistry) Find(inputFormat, outputFormat string) (ConverterPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, plugin := range r.plugins {
		if plugin.Supports(inputFormat, outputFormat) {
			return plugin, nil
		}
	}

	return nil, fmt.Errorf("no plugin found for format: %s -> %s", inputFormat, outputFormat)
}

// PluginLoader 插件加载器
type PluginLoader struct {
	registry *PluginRegistry
}

// NewPluginLoader 创建插件加载器
func NewPluginLoader(registry *PluginRegistry) *PluginLoader {
	return &PluginLoader{registry: registry}
}

// LoadFromSoFile 从 .so 文件加载插件（Go plugin）
func (l *PluginLoader) LoadFromSoFile(soPath, symbolName string, config map[string]interface{}) error {
	// 加载插件
	p, err := plugin.Open(soPath)
	if err != nil {
		return fmt.Errorf("load plugin: %w", err)
	}

	// 查找符号
	sym, err := p.Lookup(symbolName)
	if err != nil {
		return fmt.Errorf("lookup symbol: %w", err)
	}

	// 类型断言
	pluginFunc, ok := sym.(func() ConverterPlugin)
	if !ok {
		return fmt.Errorf("unexpected type from module symbol")
	}

	// 创建插件实例
	plugin := pluginFunc()

	// 注册到注册表
	return l.registry.Register(plugin, config)
}

// LoadFromConfig 从配置加载插件
func (l *PluginLoader) LoadFromConfig(config *PluginConfig) error {
	switch config.Type {
	case "builtin":
		return l.loadBuiltinPlugin(config)
	case "so":
		return l.LoadFromSoFile(config.Path, config.Symbol, config.Config)
	default:
		return fmt.Errorf("unknown plugin type: %s", config.Type)
	}
}

// loadBuiltinPlugin 加载内置插件
func (l *PluginLoader) loadBuiltinPlugin(config *PluginConfig) error {
	var plugin ConverterPlugin

	switch config.Name {
	case "deepseek":
		plugin = NewBuiltinDeepSeekPlugin()
	case "openai":
		plugin = NewBuiltinOpenAIPlugin()
	case "openclaw":
		plugin = NewBuiltinOpenClawPlugin()
	default:
		return fmt.Errorf("unknown builtin plugin: %s", config.Name)
	}

	return l.registry.Register(plugin, config.Config)
}

// PluginConfig 插件配置
type PluginConfig struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`    // "builtin", "so"
	Path    string                 `json:"path"`    // .so 文件路径
	Symbol  string                 `json:"symbol"`  // 插件符号名
	Config  map[string]interface{} `json:"config"`
	Enabled bool                   `json:"enabled"`
}

// PluginManager 插件管理器
type PluginManager struct {
	registry *PluginRegistry
	loader   *PluginLoader
	configs  []PluginConfig
}

// NewPluginManager 创建插件管理器
func NewPluginManager() *PluginManager {
	registry := NewPluginRegistry()
	return &PluginManager{
		registry: registry,
		loader:   NewPluginLoader(registry),
	}
}

// LoadConfigs 从配置加载插件
func (m *PluginManager) LoadConfigs(configs []PluginConfig) error {
	m.configs = configs

	for _, config := range configs {
		if !config.Enabled {
			continue
		}

		if err := m.loader.LoadFromConfig(&config); err != nil {
			return fmt.Errorf("load plugin %s: %w", config.Name, err)
		}
	}

	return nil
}

// Reload 重新加载所有插件
func (m *PluginManager) Reload() error {
	// 清理现有插件
	for _, name := range m.registry.List() {
		if err := m.registry.Unregister(name); err != nil {
			return err
		}
	}

	// 重新加载
	return m.LoadConfigs(m.configs)
}

// GetRegistry 获取插件注册表
func (m *PluginManager) GetRegistry() *PluginRegistry {
	return m.registry
}

// PluginConverter 使用插件的转换器
type PluginConverter struct {
	registry *PluginRegistry
	fallback Converter
}

// NewPluginConverter 创建插件转换器
func NewPluginConverter(registry *PluginRegistry, fallback Converter) *PluginConverter {
	return &PluginConverter{
		registry: registry,
		fallback: fallback,
	}
}

// Convert 使用插件转换
func (c *PluginConverter) Convert(inputFormat, outputFormat string, data []byte) ([]byte, error) {
	// 查找支持该格式的插件
	plugin, err := c.registry.Find(inputFormat, outputFormat)
	if err != nil {
		// 没有找到插件，使用备用转换器
		if c.fallback != nil {
			return c.fallback.Convert(data)
		}
		return nil, err
	}

	return plugin.Convert(data)
}

// ConvertStream 使用插件流式转换
func (c *PluginConverter) ConvertStream(inputFormat, outputFormat string, r io.Reader, w io.Writer) error {
	// 查找支持该格式的插件
	plugin, err := c.registry.Find(inputFormat, outputFormat)
	if err != nil {
		// 没有找到插件，使用备用转换器
		if c.fallback != nil {
			return c.fallback.ConvertStream(r, w)
		}
		return err
	}

	return plugin.ConvertStream(r, w)
}

// --- 内置插件实现 ---

// BuiltinDeepSeekPlugin 内置 DeepSeek 插件
type BuiltinDeepSeekPlugin struct {
	config map[string]interface{}
	conv   *DeepSeekConverter
}

func NewBuiltinDeepSeekPlugin() *BuiltinDeepSeekPlugin {
	return &BuiltinDeepSeekPlugin{}
}

func (p *BuiltinDeepSeekPlugin) Name() string { return "deepseek" }
func (p *BuiltinDeepSeekPlugin) Version() string { return "1.0.0" }

func (p *BuiltinDeepSeekPlugin) Init(config map[string]interface{}) error {
	p.config = config

	convConfig := &ConverterConfig{
		InputFormat:  "deepseek",
		OutputFormat: "openclaw",
		IncludeUsage: false,
	}

	if inputFmt, ok := config["input_format"].(string); ok {
		convConfig.InputFormat = inputFmt
	}
	if outputFmt, ok := config["output_format"].(string); ok {
		convConfig.OutputFormat = outputFmt
	}

	p.conv = NewDeepSeekConverter(convConfig)
	return nil
}

func (p *BuiltinDeepSeekPlugin) Convert(data []byte) ([]byte, error) {
	return p.conv.Convert(data)
}

func (p *BuiltinDeepSeekPlugin) ConvertStream(r io.Reader, w io.Writer) error {
	return p.conv.ConvertStream(r, w)
}

func (p *BuiltinDeepSeekPlugin) Supports(inputFormat, outputFormat string) bool {
	return inputFormat == "deepseek" && (outputFormat == "openclaw" || outputFormat == "json")
}

func (p *BuiltinDeepSeekPlugin) Cleanup() error {
	return nil
}

// BuiltinOpenAIPlugin 内置 OpenAI 插件
type BuiltinOpenAIPlugin struct {
	config map[string]interface{}
	conv   *OpenAIConverter
}

func NewBuiltinOpenAIPlugin() *BuiltinOpenAIPlugin {
	return &BuiltinOpenAIPlugin{}
}

func (p *BuiltinOpenAIPlugin) Name() string { return "openai" }
func (p *BuiltinOpenAIPlugin) Version() string { return "1.0.0" }

func (p *BuiltinOpenAIPlugin) Init(config map[string]interface{}) error {
	p.config = config

	convConfig := &ConverterConfig{
		InputFormat:  "openai-json",
		OutputFormat: "openclaw",
		IncludeUsage: false,
	}

	p.conv = NewOpenAIConverter(convConfig)
	return nil
}

func (p *BuiltinOpenAIPlugin) Convert(data []byte) ([]byte, error) {
	return p.conv.Convert(data)
}

func (p *BuiltinOpenAIPlugin) ConvertStream(r io.Reader, w io.Writer) error {
	return p.conv.ConvertStream(r, w)
}

func (p *BuiltinOpenAIPlugin) Supports(inputFormat, outputFormat string) bool {
	return inputFormat == "openai-json" && (outputFormat == "openclaw" || outputFormat == "json")
}

func (p *BuiltinOpenAIPlugin) Cleanup() error {
	return nil
}

// BuiltinOpenClawPlugin 内置 OpenClaw 插件（直接透传）
type BuiltinOpenClawPlugin struct {
	config map[string]interface{}
}

func NewBuiltinOpenClawPlugin() *BuiltinOpenClawPlugin {
	return &BuiltinOpenClawPlugin{}
}

func (p *BuiltinOpenClawPlugin) Name() string { return "openclaw" }
func (p *BuiltinOpenClawPlugin) Version() string { return "1.0.0" }

func (p *BuiltinOpenClawPlugin) Init(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *BuiltinOpenClawPlugin) Convert(data []byte) ([]byte, error) {
	return data, nil // 直接透传
}

func (p *BuiltinOpenClawPlugin) ConvertStream(r io.Reader, w io.Writer) error {
	_, err := io.Copy(w, r)
	return err
}

func (p *BuiltinOpenClawPlugin) Supports(inputFormat, outputFormat string) bool {
	return inputFormat == "openclaw" && outputFormat == "openclaw"
}

func (p *BuiltinOpenClawPlugin) Cleanup() error {
	return nil
}

// PluginConfigFile 插件配置文件
type PluginConfigFile struct {
	Plugins []PluginConfig `json:"plugins"`
}

// LoadPluginConfigFile 加载插件配置文件
func LoadPluginConfigFile(data []byte) (*PluginConfigFile, error) {
	var config PluginConfigFile
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
