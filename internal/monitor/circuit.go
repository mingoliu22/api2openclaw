package monitor

import (
	"sync"
	"time"
)

// CircuitState 熔断器状态
type CircuitState string

const (
	StateClosed    CircuitState = "closed"
	StateOpen      CircuitState = "open"
	StateHalfOpen  CircuitState = "half-open"
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	mu              sync.Mutex
	state           CircuitState
	lastFailureTime time.Time
	failureCount    int
	successCount    int
	lastStateChange time.Time

	config CircuitConfig
}

// CircuitConfig 熔断器配置
type CircuitConfig struct {
	ErrorRateThreshold  float64       // 错误率阈值
	ConsecutiveErrors   int           // 连续错误阈值
	RecoveryTimeout     time.Duration // 恢复超时
	HalfOpenMaxAttempts int           // 半开状态最大尝试次数
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(config CircuitConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state: StateClosed,
		config: config,
	}
}

// Allow 请求是否允许通过
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// 检查是否可以尝试恢复
		if now.Sub(cb.lastFailureTime) >= cb.config.RecoveryTimeout {
			cb.setState(StateHalfOpen)
			cb.successCount = 0
			return true
		}
		return false

	case StateHalfOpen:
		return cb.successCount < cb.config.HalfOpenMaxAttempts
	}

	return false
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.config.HalfOpenMaxAttempts {
			cb.setState(StateClosed)
		}
	}
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	// 检查是否需要熔断
	if cb.failureCount >= cb.config.ConsecutiveErrors {
		cb.setState(StateOpen)
	}
}

// GetState 获取当前状态
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// setState 设置状态
func (cb *CircuitBreaker) setState(state CircuitState) {
	if cb.state != state {
		cb.state = state
		cb.lastStateChange = time.Now()
	}
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
}

// CircuitBreakerRegistry 熔断器注册表
type CircuitBreakerRegistry struct {
	mu    sync.RWMutex
	items map[string]*CircuitBreaker
}

// NewCircuitBreakerRegistry 创建熔断器注册表
func NewCircuitBreakerRegistry() *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		items: make(map[string]*CircuitBreaker),
	}
}

// Get 获取熔断器
func (r *CircuitBreakerRegistry) Get(name string, config CircuitConfig) *CircuitBreaker {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cb, exists := r.items[name]; exists {
		return cb
	}

	cb := NewCircuitBreaker(config)
	r.items[name] = cb
	return cb
}

// Reset 重置指定熔断器
func (r *CircuitBreakerRegistry) Reset(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cb, exists := r.items[name]; exists {
		cb.Reset()
	}
}
