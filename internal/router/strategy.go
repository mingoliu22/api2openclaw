package router

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/openclaw/api2openclaw/internal/models"
)

// DirectStrategy 直接策略（总是选择第一个）
type DirectStrategy struct{}

func NewDirectStrategy() *DirectStrategy {
	return &DirectStrategy{}
}

func (s *DirectStrategy) Name() string {
	return "direct"
}

func (s *DirectStrategy) Select(backends []*models.Backend) (*models.Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}
	return backends[0], nil
}

// RoundRobinStrategy 轮询策略
type RoundRobinStrategy struct {
	counter uint64
}

func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{}
}

func (s *RoundRobinStrategy) Name() string {
	return "round-robin"
}

func (s *RoundRobinStrategy) Select(backends []*models.Backend) (*models.Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	idx := atomic.AddUint64(&s.counter, 1) - 1
	return backends[idx%uint64(len(backends))], nil
}

// LeastConnectionsStrategy 最少连接策略
type LeastConnectionsStrategy struct {
	counters map[string]*uint64
	mu       sync.Mutex
}

func NewLeastConnectionsStrategy() *LeastConnectionsStrategy {
	return &LeastConnectionsStrategy{
		counters: make(map[string]*uint64),
	}
}

func (s *LeastConnectionsStrategy) Name() string {
	return "least-connections"
}

func (s *LeastConnectionsStrategy) Select(backends []*models.Backend) (*models.Backend, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	var selected *models.Backend
	minConns := uint64(1<<63 - 1) // MaxUint64

	for _, b := range backends {
		if _, exists := s.counters[b.ID]; !exists {
			var counter uint64
			s.counters[b.ID] = &counter
		}

		conn := atomic.LoadUint64(s.counters[b.ID])
		if conn < minConns {
			minConns = conn
			selected = b
		}
	}

	if selected == nil {
		selected = backends[0]
	}

	// 增加连接计数
	atomic.AddUint64(s.counters[selected.ID], 1)

	return selected, nil
}

// Release 释放连接（请求完成后调用）
func (s *LeastConnectionsStrategy) Release(backendID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if counter, exists := s.counters[backendID]; exists {
		atomic.AddUint64(counter, ^uint64(0)) // 减1
	}
}

// RandomStrategy 随机策略
type RandomStrategy struct {
	rand *rand.Rand
}

func NewRandomStrategy() *RandomStrategy {
	return &RandomStrategy{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *RandomStrategy) Name() string {
	return "random"
}

func (s *RandomStrategy) Select(backends []*models.Backend) (*models.Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	idx := s.rand.Intn(len(backends))
	return backends[idx], nil
}

// WeightedRoundRobinStrategy 加权轮询策略（平滑权重算法）
type WeightedRoundRobinStrategy struct {
	mu       sync.Mutex
	backends map[string]*weightedBackend
}

type weightedBackend struct {
	backend         *models.Backend
	currentWeight   int
	effectiveWeight int
}

// NewWeightedRoundRobinStrategy 创建加权轮询策略
func NewWeightedRoundRobinStrategy() *WeightedRoundRobinStrategy {
	return &WeightedRoundRobinStrategy{
		backends: make(map[string]*weightedBackend),
	}
}

func (s *WeightedRoundRobinStrategy) Name() string {
	return "weighted-round-robin"
}

func (s *WeightedRoundRobinStrategy) Select(backends []*models.Backend) (*models.Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 初始化或更新后端权重信息
	total := 0
	var best *weightedBackend

	for _, b := range backends {
		// 设置默认权重
		weight := b.Weight
		if weight <= 0 {
			weight = 1 // 默认权重为 1
		}

		wb, exists := s.backends[b.ID]
		if !exists {
			wb = &weightedBackend{
				backend:         b,
				currentWeight:   0,
				effectiveWeight: weight,
			}
			s.backends[b.ID] = wb
		} else {
			// 更新后端引用和权重
			wb.backend = b
			wb.effectiveWeight = weight
		}

		// 使用平滑加权算法
		wb.currentWeight += wb.effectiveWeight
		total += wb.effectiveWeight

		if best == nil || wb.currentWeight > best.currentWeight {
			best = wb
		}
	}

	if best == nil {
		// 回退到第一个后端
		return backends[0], nil
	}

	// 减少选中后端的当前权重
	best.currentWeight -= total

	return best.backend, nil
}

// SetWeight 动态设置后端权重
func (s *WeightedRoundRobinStrategy) SetWeight(backendID string, weight int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if weight <= 0 {
		weight = 1
	}

	if wb, exists := s.backends[backendID]; exists {
		wb.effectiveWeight = weight
	}
}

// RemoveBackend 移除后端
func (s *WeightedRoundRobinStrategy) RemoveBackend(backendID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.backends, backendID)
}

// Reset 重置所有权重
func (s *WeightedRoundRobinStrategy) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, wb := range s.backends {
		wb.currentWeight = 0
	}
}

