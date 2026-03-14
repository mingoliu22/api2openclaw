package router

import (
	"fmt"
	"sync"
	"time"

	"github.com/openclaw/api2openclaw/internal/models"
)

// DynamicRouteManager 动态路由管理器
type DynamicRouteManager struct {
	router *Router
	mu     sync.RWMutex
}

// NewDynamicRouteManager 创建动态路由管理器
func NewDynamicRouteManager(router *Router) *DynamicRouteManager {
	return &DynamicRouteManager{router: router}
}

// UpdateModelRouting 更新模型路由配置
func (m *DynamicRouteManager) UpdateModelRouting(modelName string, config *ModelRoutingUpdate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取现有模型配置
	model, exists := m.router.models[modelName]
	if !exists {
		return fmt.Errorf("model %s not found", modelName)
	}

	// 更新配置
	if config.BackendGroup != nil {
		// 验证后端存在
		for _, bid := range config.BackendGroup {
			if _, ok := m.router.backends[bid]; !ok {
				return fmt.Errorf("backend %s not found", bid)
			}
		}
		model.BackendGroup = config.BackendGroup
	}

	if config.RoutingStrategy != "" {
		if _, ok := m.router.strategies[config.RoutingStrategy]; !ok {
			return fmt.Errorf("strategy %s not found", config.RoutingStrategy)
		}
		model.RoutingStrategy = config.RoutingStrategy
	}

	model.UpdatedAt = time.Now()

	// 记录变更
	return nil
}

// UpdateBackendWeight 更新后端权重
func (m *DynamicRouteManager) UpdateBackendWeight(backendID string, weight int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	backend, ok := m.router.backends[backendID]
	if !ok {
		return fmt.Errorf("backend %s not found", backendID)
	}

	backend.Weight = weight
	backend.UpdatedAt = time.Now()

	// 通知加权轮询策略更新
	if strategy, ok := m.router.strategies["weighted-round-robin"].(*WeightedRoundRobinStrategy); ok {
		strategy.SetWeight(backendID, weight)
	}

	return nil
}

// UpdateAlias 更新模型别名
func (m *DynamicRouteManager) UpdateAlias(alias string, config *AliasUpdate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.router.aliases[alias]
	if !exists {
		return fmt.Errorf("alias %s not found", alias)
	}

	// 更新配置
	if config.Target != "" {
		if _, ok := m.router.models[config.Target]; !ok {
			return fmt.Errorf("target model %s not found", config.Target)
		}
		existing.Target = config.Target
	}

	if config.Backends != nil {
		for _, bid := range config.Backends {
			if _, ok := m.router.backends[bid]; !ok {
				return fmt.Errorf("backend %s not found", bid)
			}
		}
		existing.Backends = config.Backends
	}

	if config.Strategy != "" {
		if _, ok := m.router.strategies[config.Strategy]; !ok {
			return fmt.Errorf("strategy %s not found", config.Strategy)
		}
		existing.Strategy = config.Strategy
	}

	return nil
}

// RemoveModel 移除模型（运行时）
func (m *DynamicRouteManager) RemoveModel(modelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.router.models[modelName]; !ok {
		return fmt.Errorf("model %s not found", modelName)
	}

	delete(m.router.models, modelName)

	// 同时移除相关别名
	for alias, cfg := range m.router.aliases {
		if cfg.Target == modelName {
			delete(m.router.aliases, alias)
		}
	}

	return nil
}

// RemoveAlias 移除别名
func (m *DynamicRouteManager) RemoveAlias(alias string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.router.aliases[alias]; !ok {
		return fmt.Errorf("alias %s not found", alias)
	}

	delete(m.router.aliases, alias)
	return nil
}

// DisableBackend 禁用后端
func (m *DynamicRouteManager) DisableBackend(backendID string) error {
	return m.SetBackendHealth(backendID, models.BackendStatusUnhealthy)
}

// EnableBackend 启用后端
func (m *DynamicRouteManager) EnableBackend(backendID string) error {
	return m.SetBackendHealth(backendID, models.BackendStatusHealthy)
}

// SetBackendHealth 设置后端健康状态
func (m *DynamicRouteManager) SetBackendHealth(backendID string, status models.BackendStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	backend, ok := m.router.backends[backendID]
	if !ok {
		return fmt.Errorf("backend %s not found", backendID)
	}

	backend.Status = status
	now := time.Now()
	backend.LastCheckAt = &now
	backend.UpdatedAt = now

	return nil
}

// GetRoutingInfo 获取路由信息
func (m *DynamicRouteManager) GetRoutingInfo(modelName string) (*RoutingInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := &RoutingInfo{
		ModelName: modelName,
	}

	// 检查是否是别名
	if alias, ok := m.router.aliases[modelName]; ok {
		info.IsAlias = true
		info.AliasTarget = alias.Target
		info.BackendGroup = alias.Backends
		info.Strategy = alias.Strategy
		info.ResolvedModel = alias.Target
	} else if model, ok := m.router.models[modelName]; ok {
		info.BackendGroup = model.BackendGroup
		info.Strategy = model.RoutingStrategy
	}

	// 获取后端状态
	for _, bid := range info.BackendGroup {
		if backend, ok := m.router.backends[bid]; ok {
			info.Backends = append(info.Backends, BackendInfo{
				ID:     backend.ID,
				Name:   backend.Name,
				Health: string(backend.Status),
				Weight: backend.Weight,
			})
		}
	}

	return info, nil
}

// GetAllRoutingInfo 获取所有路由信息
func (m *DynamicRouteManager) GetAllRoutingInfo() map[string]*RoutingInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*RoutingInfo)

	// 添加所有模型
	for name := range m.router.models {
		if info, err := m.GetRoutingInfo(name); err == nil {
			result[name] = info
		}
	}

	// 添加所有别名
	for alias := range m.router.aliases {
		if info, err := m.GetRoutingInfo(alias); err == nil {
			result[alias] = info
		}
	}

	return result
}

// --- 数据结构 ---

// ModelRoutingUpdate 模型路由更新
type ModelRoutingUpdate struct {
	BackendGroup    []string `json:"backend_group,omitempty"`
	RoutingStrategy string   `json:"routing_strategy,omitempty"`
}

// AliasUpdate 别名更新
type AliasUpdate struct {
	Target   string   `json:"target,omitempty"`
	Backends []string `json:"backends,omitempty"`
	Strategy string   `json:"strategy,omitempty"`
}

// RoutingInfo 路由信息
type RoutingInfo struct {
	ModelName    string         `json:"model_name"`
	IsAlias      bool           `json:"is_alias"`
	AliasTarget  string         `json:"alias_target,omitempty"`
	ResolvedModel string         `json:"resolved_model,omitempty"`
	BackendGroup []string       `json:"backend_group,omitempty"`
	Strategy     string         `json:"strategy,omitempty"`
	Backends     []BackendInfo  `json:"backends,omitempty"`
}

// BackendInfo 后端信息
type BackendInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Health string `json:"health"`
	Weight int    `json:"weight"`
}

// RouteChange 路由变更记录
type RouteChange struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"` // "model_update", "backend_update", "alias_update", etc.
	EntityID    string    `json:"entity_id"`
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	ChangedBy   string    `json:"changed_by"`
}

// RouteHistory 路由变更历史
type RouteHistory struct {
	mu      sync.RWMutex
	changes []RouteChange
	maxSize int
}

// NewRouteHistory 创建路由历史
func NewRouteHistory(maxSize int) *RouteHistory {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &RouteHistory{
		changes: make([]RouteChange, 0, maxSize),
		maxSize: maxSize,
	}
}

// Record 记录变更
func (h *RouteHistory) Record(change RouteChange) {
	h.mu.Lock()
	defer h.mu.Unlock()

	change.Timestamp = time.Now()

	if len(h.changes) >= h.maxSize {
		// 移除最旧的记录
		h.changes = h.changes[1:]
	}

	h.changes = append(h.changes, change)
}

// GetChanges 获取变更历史
func (h *RouteHistory) GetChanges(limit int) []RouteChange {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit <= 0 || limit > len(h.changes) {
		limit = len(h.changes)
	}

	// 返回最近的变更
	start := len(h.changes) - limit
	if start < 0 {
		start = 0
	}

	result := make([]RouteChange, limit)
	copy(result, h.changes[start:])

	return result
}
