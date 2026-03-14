package monitor

import (
	"context"
	"time"
)

// Collector 指标采集器
type Collector struct {
	storage Storage
}

// Storage 指标存储接口
type Storage interface {
	RecordMetric(ctx context.Context, metric *Metric) error
	QueryMetrics(ctx context.Context, filter *MetricFilter) ([]*Metric, error)
	GetUsageStats(ctx context.Context, apiKeyID string, from, to time.Time) (*UsageStats, error)
}

// Metric 指标数据
type Metric struct {
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
	APIKeyID         string    `json:"api_key_id" db:"api_key_id"`
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	Model            string    `json:"model" db:"model"`
	RequestID        string    `json:"request_id" db:"request_id"`
	StatusCode       int       `json:"status_code" db:"status_code"`
	LatencyMs        int64     `json:"latency_ms" db:"latency_ms"`
	PromptTokens     int       `json:"prompt_tokens" db:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens" db:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens" db:"total_tokens"`
	Error            string    `json:"error,omitempty" db:"error"`
}

// MetricFilter 指标查询过滤器
type MetricFilter struct {
	APIKeyID  string
	TenantID  string
	Model     string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

// UsageStats 使用量统计
type UsageStats struct {
	TotalRequests      int64 `json:"total_requests"`
	SuccessRequests    int64 `json:"success_requests"`
	ErrorRequests      int64 `json:"error_requests"`
	AvgLatencyMs       float64 `json:"avg_latency_ms"`
	TotalPromptTokens  int64 `json:"total_prompt_tokens"`
	TotalCompletionTokens int64 `json:"total_completion_tokens"`
	TotalTokens        int64 `json:"total_tokens"`
}

// NewCollector 创建指标采集器
func NewCollector(storage Storage) *Collector {
	return &Collector{storage: storage}
}

// RecordRequest 记录请求指标
func (c *Collector) RecordRequest(ctx context.Context, req *RequestContext) error {
	metric := &Metric{
		Timestamp:        time.Now(),
		APIKeyID:         req.APIKeyID,
		TenantID:         req.TenantID,
		Model:            req.Model,
		RequestID:        req.RequestID,
		StatusCode:       req.StatusCode,
		LatencyMs:        time.Since(req.StartTime).Milliseconds(),
		PromptTokens:     req.PromptTokens,
		CompletionTokens: req.CompletionTokens,
		TotalTokens:      req.TotalTokens,
		Error:            req.Error,
	}

	return c.storage.RecordMetric(ctx, metric)
}

// GetUsage 获取使用量统计
func (c *Collector) GetUsage(ctx context.Context, apiKeyID string, days int) (*UsageStats, error) {
	to := time.Now()
	from := to.AddDate(0, 0, -days)

	return c.storage.GetUsageStats(ctx, apiKeyID, from, to)
}

// RequestContext 请求上下文
type RequestContext struct {
	APIKeyID         string
	TenantID         string
	Model            string
	RequestID        string
	StartTime        time.Time
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	StatusCode       int
	Error            string
}

// NewRequestContext 创建请求上下文
func NewRequestContext(apiKeyID, tenantID, model, requestID string) *RequestContext {
	return &RequestContext{
		APIKeyID:  apiKeyID,
		TenantID:  tenantID,
		Model:     model,
		RequestID: requestID,
		StartTime: time.Now(),
	}
}

// Complete 完成请求
func (r *RequestContext) Complete(statusCode int, promptTokens, completionTokens int) {
	r.StatusCode = statusCode
	r.PromptTokens = promptTokens
	r.CompletionTokens = completionTokens
	r.TotalTokens = promptTokens + completionTokens
}

// Fail 请求失败
func (r *RequestContext) Fail(err error) {
	r.StatusCode = 500
	r.Error = err.Error()
}
