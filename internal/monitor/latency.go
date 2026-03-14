package monitor

import (
	"context"
	"sync"
	"time"
)

// LatencyTracker 延迟跟踪器，支持分位数计算
type LatencyTracker struct {
	mu     sync.RWMutex
	values []time.Duration
	maxLen int
}

// NewLatencyTracker 创建延迟跟踪器
func NewLatencyTracker(maxLen int) *LatencyTracker {
	if maxLen <= 0 {
		maxLen = 10000 // 默认保留 10000 个样本
	}
	return &LatencyTracker{
		values: make([]time.Duration, 0, maxLen),
		maxLen: maxLen,
	}
}

// Record 记录延迟
func (t *LatencyTracker) Record(latency time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 如果满了，移除最旧的值
	if len(t.values) >= t.maxLen {
		t.values = t.values[1:]
	}

	t.values = append(t.values, latency)
}

// GetPercentiles 获取分位数延迟
func (t *LatencyTracker) GetPercentiles() (p50, p90, p95, p99 time.Duration) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.values) == 0 {
		return 0, 0, 0, 0
	}

	// 复制并排序
	sorted := make([]time.Duration, len(t.values))
	copy(sorted, t.values)

	// 使用快速选择算法排序
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	p50 = sorted[len(sorted)*50/100]
	p90 = sorted[len(sorted)*90/100]
	p95 = sorted[len(sorted)*95/100]
	p99 = sorted[len(sorted)*99/100]

	return
}

// GetAverage 获取平均延迟
func (t *LatencyTracker) GetAverage() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.values) == 0 {
		return 0
	}

	var sum time.Duration
	for _, v := range t.values {
		sum += v
	}

	return sum / time.Duration(len(t.values))
}

// Clear 清空记录
func (t *LatencyTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.values = t.values[:0]
}

// Count 获取样本数量
func (t *LatencyTracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.values)
}

// UsageStatsExtended 扩展的用量统计（包含延迟分位数）
type UsageStatsExtended struct {
	TotalRequests        int64     `json:"total_requests"`
	SuccessRequests      int64     `json:"success_requests"`
	ErrorRequests        int64     `json:"error_requests"`
	AvgLatencyMs         float64   `json:"avg_latency_ms"`
	P50LatencyMs         float64   `json:"p50_latency_ms"`
	P90LatencyMs         float64   `json:"p90_latency_ms"`
	P95LatencyMs         float64   `json:"p95_latency_ms"`
	P99LatencyMs         float64   `json:"p99_latency_ms"`
	TotalPromptTokens    int64     `json:"total_prompt_tokens"`
	TotalCompletionTokens int64    `json:"total_completion_tokens"`
	TotalTokens          int64     `json:"total_tokens"`
	StartTime            time.Time `json:"start_time"`
	EndTime              time.Time `json:"end_time"`
}

// ExtendedUsageStatsStore 扩展用量统计存储接口
type ExtendedUsageStatsStore interface {
	GetExtendedUsageStats(ctx context.Context, filter *UsageStatsFilter) (*UsageStatsExtended, error)
	GetHourlyStats(ctx context.Context, filter *UsageStatsFilter) ([]*HourlyStats, error)
	GetDailyStats(ctx context.Context, filter *UsageStatsFilter) ([]*DailyStats, error)
}

// UsageStatsFilter 用量统计过滤器
type UsageStatsFilter struct {
	APIKeyID  string     `json:"api_key_id,omitempty"`
	TenantID  string     `json:"tenant_id,omitempty"`
	Model     string     `json:"model,omitempty"`
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time"`
}

// HourlyStats 小时统计
type HourlyStats struct {
	Hour          string  `json:"hour"`           // YYYY-MM-DDTHH
	TotalRequests int64   `json:"total_requests"`
	ErrorRequests int64   `json:"error_requests"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	P99LatencyMs  float64 `json:"p99_latency_ms"`
	TotalTokens   int64   `json:"total_tokens"`
}

// DailyStats 日统计
type DailyStats struct {
	Date          string  `json:"date"`           // YYYY-MM-DD
	TotalRequests int64   `json:"total_requests"`
	ErrorRequests int64   `json:"error_requests"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	P99LatencyMs  float64 `json:"p99_latency_ms"`
	TotalTokens   int64   `json:"total_tokens"`
}

// TimeSeriesAggregator 时间序列聚合器
type TimeSeriesAggregator struct {
	mu       sync.RWMutex
	trackers map[string]*LatencyTracker // key: api_key_id:model
	counters map[string]*RequestCounter // key: api_key_id:model
}

// RequestCounter 请求计数器
type RequestCounter struct {
	Total     int64
	Success   int64
	Error     int64
	Tokens    int64
	Timestamp time.Time
}

// NewTimeSeriesAggregator 创建时间序列聚合器
func NewTimeSeriesAggregator() *TimeSeriesAggregator {
	return &TimeSeriesAggregator{
		trackers: make(map[string]*LatencyTracker),
		counters: make(map[string]*RequestCounter),
	}
}

// RecordRequest 记录请求
func (a *TimeSeriesAggregator) RecordRequest(apiKeyID, model string, latency time.Duration, success bool, tokens int) {
	key := apiKeyID + ":" + model

	a.mu.Lock()
	defer a.mu.Unlock()

	// 记录延迟
	if _, ok := a.trackers[key]; !ok {
		a.trackers[key] = NewLatencyTracker(10000)
	}
	a.trackers[key].Record(latency)

	// 更新计数器
	if _, ok := a.counters[key]; !ok {
		a.counters[key] = &RequestCounter{Timestamp: time.Now()}
	}

	a.counters[key].Total++
	if success {
		a.counters[key].Success++
	} else {
		a.counters[key].Error++
	}
	a.counters[key].Tokens += int64(tokens)
}

// GetStats 获取统计信息
func (a *TimeSeriesAggregator) GetStats(apiKeyID, model string) *UsageStatsExtended {
	key := apiKeyID + ":" + model

	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := &UsageStatsExtended{
		StartTime: time.Now().Add(-24 * time.Hour),
		EndTime:   time.Now(),
	}

	if counter, ok := a.counters[key]; ok {
		stats.TotalRequests = counter.Total
		stats.SuccessRequests = counter.Success
		stats.ErrorRequests = counter.Error
		stats.TotalTokens = counter.Tokens
	}

	if tracker, ok := a.trackers[key]; ok {
		stats.AvgLatencyMs = float64(tracker.GetAverage().Milliseconds())
		p50, p90, p95, p99 := tracker.GetPercentiles()
		stats.P50LatencyMs = float64(p50.Milliseconds())
		stats.P90LatencyMs = float64(p90.Milliseconds())
		stats.P95LatencyMs = float64(p95.Milliseconds())
		stats.P99LatencyMs = float64(p99.Milliseconds())
	}

	return stats
}

// Clear 清空所有数据
func (a *TimeSeriesAggregator) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.trackers = make(map[string]*LatencyTracker)
	a.counters = make(map[string]*RequestCounter)
}

// GetAggregationKeys 获取所有聚合键
func (a *TimeSeriesAggregator) GetAggregationKeys() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	keys := make([]string, 0, len(a.trackers))
	for key := range a.trackers {
		keys = append(keys, key)
	}
	return keys
}
