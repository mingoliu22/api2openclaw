package router

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/openclaw/api2openclaw/internal/models"
)

// Router 模型路由器
type Router struct {
	mu           sync.RWMutex
	backends     map[string]*models.Backend
	models       map[string]*models.ModelConfig
	aliases      map[string]*ModelAlias
	strategies   map[string]Strategy
	healthChecker *HealthChecker
	httpClient   *http.Client
}

// ModelAlias 模型别名
type ModelAlias struct {
	Alias      string
	Target     string
	Backends   []string
	Strategy   string
}

// Strategy 路由策略接口
type Strategy interface {
	Select(backends []*models.Backend) (*models.Backend, error)
	Name() string
}

// New 创建路由器
func New() *Router {
	r := &Router{
		backends:   make(map[string]*models.Backend),
		models:     make(map[string]*models.ModelConfig),
		aliases:    make(map[string]*ModelAlias),
		strategies: make(map[string]Strategy),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// 注册默认策略
	r.RegisterStrategy(NewDirectStrategy())
	r.RegisterStrategy(NewRoundRobinStrategy())
	r.RegisterStrategy(NewLeastConnectionsStrategy())
	r.RegisterStrategy(NewRandomStrategy())
	r.RegisterStrategy(NewWeightedRoundRobinStrategy())

	// 启动健康检查
	r.healthChecker = NewHealthChecker(r, r.httpClient)
	go r.healthChecker.Run()

	return r
}

// RegisterStrategy 注册路由策略
func (r *Router) RegisterStrategy(strategy Strategy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies[strategy.Name()] = strategy
}

// RegisterBackend 注册后端实例
func (r *Router) RegisterBackend(backend *models.Backend) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.backends[backend.ID]; exists {
		return fmt.Errorf("backend %s already exists", backend.ID)
	}

	// 默认健康状态
	if backend.Status == "" {
		backend.Status = models.BackendStatusHealthy
	}

	now := time.Now()
	backend.CreatedAt = now
	backend.UpdatedAt = now

	r.backends[backend.ID] = backend
	log.Printf("[Router] Backend registered: %s (%s)", backend.ID, backend.Name)

	return nil
}

// GetBackend 获取后端实例
func (r *Router) GetBackend(id string) (*models.Backend, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.backends[id]
	return b, ok
}

// ListBackends 列出所有后端
func (r *Router) ListBackends() []*models.Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*models.Backend, 0, len(r.backends))
	for _, b := range r.backends {
		list = append(list, b)
	}
	return list
}

// RegisterModel 注册模型配置
func (r *Router) RegisterModel(cfg *models.ModelConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 验证后端存在
	for _, bid := range cfg.BackendGroup {
		if _, exists := r.backends[bid]; !exists {
			return fmt.Errorf("backend %s not found", bid)
		}
	}

	// 检查策略是否存在
	if _, exists := r.strategies[cfg.RoutingStrategy]; !exists {
		return fmt.Errorf("strategy %s not found", cfg.RoutingStrategy)
	}

	now := time.Now()
	cfg.CreatedAt = now
	cfg.UpdatedAt = now

	r.models[cfg.Name] = cfg
	log.Printf("[Router] Model registered: %s -> %v", cfg.Name, cfg.BackendGroup)

	return nil
}

// RegisterAlias 注册模型别名
func (r *Router) RegisterAlias(alias, target string, backends []string, strategy string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 验证目标模型存在
	if _, exists := r.models[target]; !exists {
		return fmt.Errorf("target model %s not found", target)
	}

	// 如果指定了后端组，验证后端存在
	for _, bid := range backends {
		if _, exists := r.backends[bid]; !exists {
			return fmt.Errorf("backend %s not found", bid)
		}
	}

	r.aliases[alias] = &ModelAlias{
		Alias:    alias,
		Target:   target,
		Backends: backends,
		Strategy: strategy,
	}

	log.Printf("[Router] Alias registered: %s -> %s", alias, target)
	return nil
}

// ResolveAlias 解析模型别名，返回实际模型名称
func (r *Router) ResolveAlias(model string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if alias, ok := r.aliases[model]; ok {
		return alias.Target
	}
	return model
}

// GetAlias 获取别名配置
func (r *Router) GetAlias(alias string) (*ModelAlias, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.aliases[alias]
	return a, ok
}

// ListAliases 列出所有别名
func (r *Router) ListAliases() map[string]*ModelAlias {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*ModelAlias, len(r.aliases))
	for k, v := range r.aliases {
		result[k] = v
	}
	return result
}

// GetModel 获取模型配置
func (r *Router) GetModel(name string) (*models.ModelConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[name]
	return m, ok
}

// ListModels 列出所有模型
func (r *Router) ListModels() []*models.ModelConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*models.ModelConfig, 0, len(r.models))
	for _, m := range r.models {
		list = append(list, m)
	}
	return list
}

// Route 路由请求到后端
func (r *Router) Route(ctx context.Context, modelName string) (*models.Backend, error) {
	// 解析别名
	actualModel := r.ResolveAlias(modelName)

	// 检查是否有别名覆盖的后端配置
	if alias, ok := r.GetAlias(modelName); ok && len(alias.Backends) > 0 {
		// 使用别名指定的后端组
		healthyBackends := r.getHealthyBackends(alias.Backends)
		if len(healthyBackends) == 0 {
			return nil, fmt.Errorf("no healthy backends for alias %s", modelName)
		}

		// 使用别名指定的策略或默认策略
		strategyName := alias.Strategy
		if strategyName == "" {
			strategyName = "round-robin"
		}

		r.mu.RLock()
		strategy, ok := r.strategies[strategyName]
		r.mu.RUnlock()

		if !ok {
			strategy = r.strategies["direct"]
		}

		return strategy.Select(healthyBackends)
	}

	// 使用标准模型路由
	r.mu.RLock()
	model, ok := r.models[actualModel]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("model %s not found", actualModel)
	}

	// 获取健康的后端列表
	healthyBackends := r.getHealthyBackends(model.BackendGroup)
	if len(healthyBackends) == 0 {
		return nil, fmt.Errorf("no healthy backends for model %s", actualModel)
	}

	// 应用策略选择
	r.mu.RLock()
	strategy, ok := r.strategies[model.RoutingStrategy]
	r.mu.RUnlock()

	if !ok {
		strategy = r.strategies["direct"]
	}

	return strategy.Select(healthyBackends)
}

// RouteWithKeyRestriction 使用 API Key 限制的路由（支持路由隔离）
func (r *Router) RouteWithKeyRestriction(ctx context.Context, modelName string, apiKey *models.APIKey) (*models.Backend, error) {
	// 如果 API Key 固定了后端，使用固定后端
	if apiKey.HasPinnedBackends() {
		pinnedBackends := apiKey.GetPinnedBackends()

		// 检查固定后端是否健康
		healthyBackends := r.getHealthyBackends(pinnedBackends)
		if len(healthyBackends) == 0 {
			return nil, fmt.Errorf("no healthy backends for pinned configuration")
		}

		// 使用固定后端的第一个（或应用策略）
		r.mu.RLock()
		strategy := r.strategies["direct"]
		r.mu.RUnlock()

		return strategy.Select(healthyBackends)
	}

	// 否则使用标准路由
	return r.Route(ctx, modelName)
}

// RouteToBackend 直接路由到指定后端（用于隔离场景）
func (r *Router) RouteToBackend(backendID string) (*models.Backend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	backend, ok := r.backends[backendID]
	if !ok {
		return nil, fmt.Errorf("backend %s not found", backendID)
	}

	if !backend.IsHealthy() {
		return nil, fmt.Errorf("backend %s is not healthy", backendID)
	}

	return backend, nil
}

// ForwardRequest 转发请求到后端
func (r *Router) ForwardRequest(ctx context.Context, backend *models.Backend, body []byte) (*http.Response, error) {
	url := backend.BaseURL + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if backend.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+backend.APIKey)
	}
	for k, v := range backend.Headers {
		req.Header.Set(k, v)
	}

	// TODO: 设置请求体
	// req.Body = ...

	return r.httpClient.Do(req)
}

// getHealthyBackends 获取健康的后端列表
func (r *Router) getHealthyBackends(ids []string) []*models.Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*models.Backend
	for _, id := range ids {
		if b, ok := r.backends[id]; ok && b.IsHealthy() {
			result = append(result, b)
		}
	}
	return result
}

// UpdateBackendStatus 更新后端状态
func (r *Router) UpdateBackendStatus(backendID string, status models.BackendStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if backend, ok := r.backends[backendID]; ok {
		oldStatus := backend.Status
		backend.Status = status
		now := time.Now()
		backend.LastCheckAt = &now
		backend.UpdatedAt = now

		if oldStatus != status {
			log.Printf("[Router] Backend %s status: %s -> %s", backendID, oldStatus, status)
		}
	}
}

// Close 关闭路由器
func (r *Router) Close() error {
	if r.healthChecker != nil {
		r.healthChecker.Stop()
	}
	return nil
}
