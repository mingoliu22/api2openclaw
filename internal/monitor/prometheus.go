package monitor

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// PrometheusMetrics Prometheus 指标
type PrometheusMetrics struct {
	registry *prometheus.Registry

	// HTTP 请求指标
	requestsTotal *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestSize *prometheus.HistogramVec
	responseSize *prometheus.HistogramVec

	// 模型相关指标
	modelRequests *prometheus.CounterVec
	modelErrors *prometheus.CounterVec
	modelLatency *prometheus.HistogramVec

	// 认证相关指标
	authTotal *prometheus.CounterVec
	authErrors *prometheus.CounterVec

	// 后端状态指标
	backendHealth *prometheus.GaugeVec

	// 限流指标
	rateLimitTotal *prometheus.CounterVec

	// 熔断器指标
	circuitState *prometheus.GaugeVec
	circuitErrors *prometheus.CounterVec

	// Token 指标
	tokensTotal *prometheus.CounterVec

	// 活跃请求
	activeRequests prometheus.Gauge

	// v0.3.0: AI 友好型指标
	// 端到端请求延迟（含模型时间）
	api2ocRequestDuration *prometheus.HistogramVec

	// 网关自身处理延迟（不含模型时间）
	api2ocMiddlewareDuration *prometheus.HistogramVec

	// Token 消耗（按模型、key、类型分类）
	api2ocTokensTotal *prometheus.CounterVec

	// 错误计数（按错误码、模型分类）
	api2ocErrorsTotal *prometheus.CounterVec

	// SSE chunk 推送计数
	api2ocStreamChunksTotal *prometheus.CounterVec
}

// NewPrometheusMetrics 创建 Prometheus 指标
func NewPrometheusMetrics() *PrometheusMetrics {
	m := &PrometheusMetrics{
		// HTTP 请求指标
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api2openclaw_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),

		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api2openclaw_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),

		requestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api2openclaw_http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "endpoint"},
		),

		responseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api2openclaw_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "endpoint"},
		),

		// 模型相关指标
		modelRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api2openclaw_model_requests_total",
				Help: "Total number of model requests",
			},
			[]string{"model", "backend", "status"},
		),

		modelErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api2openclaw_model_errors_total",
				Help: "Total number of model errors",
			},
			[]string{"model", "backend", "error_type"},
		),

		modelLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api2openclaw_model_latency_seconds",
				Help:    "Model request latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"model", "backend"},
		),

		// 认证相关指标
		authTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api2openclaw_auth_requests_total",
				Help: "Total number of authentication requests",
			},
			[]string{"method", "status"},
		),

		authErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api2openclaw_auth_errors_total",
				Help: "Total number of authentication errors",
			},
			[]string{"error_type"},
		),

		// 后端状态指标
		backendHealth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api2openclaw_backend_health",
				Help: "Backend health status (1=healthy, 0=unhealthy)",
			},
			[]string{"backend_id", "backend_name"},
		),

		// 限流指标
		rateLimitTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api2openclaw_rate_limit_total",
				Help: "Total number of rate limit rejections",
			},
			[]string{"api_key_id", "limit_type"},
		),

		// 熔断器指标
		circuitState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api2openclaw_circuit_state",
				Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
			},
			[]string{"name"},
		),

		circuitErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api2openclaw_circuit_errors_total",
				Help: "Total number of circuit breaker errors",
			},
			[]string{"name"},
		),

		// Token 指标
		tokensTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api2openclaw_tokens_total",
				Help: "Total number of tokens processed",
			},
			[]string{"api_key_id", "model", "token_type"},
		),

		// 活跃请求
		activeRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "api2openclaw_active_requests",
				Help: "Number of active requests",
			},
		),

		// v0.3.0: AI 友好型指标
		api2ocRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "api2oc_request_duration_ms",
				Help: "End-to-end request duration in milliseconds (including model time)",
				Buckets: []float64{10, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 30000},
			},
			[]string{"model", "key_id"},
		),

		api2ocMiddlewareDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "api2oc_middleware_duration_ms",
				Help: "Gateway-only processing duration in milliseconds (excluding model time)",
				Buckets: []float64{1, 5, 10, 20, 50, 100, 200, 500},
			},
			[]string{"model"},
		),

		api2ocTokensTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api2oc_tokens_total",
				Help: "Total tokens consumed, by model, key_id, and type (prompt/completion)",
			},
			[]string{"model", "key_id", "type"},
		),

		api2ocErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api2oc_errors_total",
				Help: "Total errors, by error_code and model",
			},
			[]string{"error_code", "model"},
		),

		api2ocStreamChunksTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api2oc_stream_chunks_total",
				Help: "Total SSE chunks pushed (for streaming performance analysis)",
			},
			[]string{"model", "key_id"},
		),
	}

	// 注册指标
	m.registry = prometheus.NewRegistry()
	m.registry.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.requestSize,
		m.responseSize,
		m.modelRequests,
		m.modelErrors,
		m.modelLatency,
		m.authTotal,
		m.authErrors,
		m.backendHealth,
		m.rateLimitTotal,
		m.circuitState,
		m.circuitErrors,
		m.tokensTotal,
		m.activeRequests,
		// v0.3.0: 注册 AI 友好型指标
		m.api2ocRequestDuration,
		m.api2ocMiddlewareDuration,
		m.api2ocTokensTotal,
		m.api2ocErrorsTotal,
		m.api2ocStreamChunksTotal,
	)
	m.registry.MustRegister(collectors.NewGoCollector())
	m.registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	return m
}

// RecordHTTPRequest 记录 HTTP 请求
func (m *PrometheusMetrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	status := statusLabel(statusCode)
	m.requestsTotal.WithLabelValues(method, endpoint, status).Inc()
	m.requestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordModelRequest 记录模型请求
func (m *PrometheusMetrics) RecordModelRequest(model, backend string, statusCode int, duration time.Duration) {
	status := statusLabel(statusCode)
	m.modelRequests.WithLabelValues(model, backend, status).Inc()
	m.modelLatency.WithLabelValues(model, backend).Observe(duration.Seconds())
}

// RecordModelError 记录模型错误
func (m *PrometheusMetrics) RecordModelError(model, backend, errorType string) {
	m.modelErrors.WithLabelValues(model, backend, errorType).Inc()
}

// RecordAuth 记录认证请求
func (m *PrometheusMetrics) RecordAuth(method, status string) {
	m.authTotal.WithLabelValues(method, status).Inc()
}

// RecordAuthError 记录认证错误
func (m *PrometheusMetrics) RecordAuthError(errorType string) {
	m.authErrors.WithLabelValues(errorType).Inc()
}

// SetBackendHealth 设置后端健康状态
func (m *PrometheusMetrics) SetBackendHealth(backendID, backendName string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.backendHealth.WithLabelValues(backendID, backendName).Set(value)
}

// RecordRateLimit 记录限流
func (m *PrometheusMetrics) RecordRateLimit(apiKeyID, limitType string) {
	m.rateLimitTotal.WithLabelValues(apiKeyID, limitType).Inc()
}

// SetCircuitState 设置熔断器状态
func (m *PrometheusMetrics) SetCircuitState(name string, state CircuitState) {
	value := 0.0
	switch state {
	case StateOpen:
		value = 1.0
	case StateHalfOpen:
		value = 2.0
	}
	m.circuitState.WithLabelValues(name).Set(value)
}

// RecordCircuitError 记录熔断器错误
func (m *PrometheusMetrics) RecordCircuitError(name string) {
	m.circuitErrors.WithLabelValues(name).Inc()
}

// RecordTokens 记录 Token 使用
func (m *PrometheusMetrics) RecordTokens(apiKeyID, model string, promptTokens, completionTokens int) {
	m.tokensTotal.WithLabelValues(apiKeyID, model, "prompt").Add(float64(promptTokens))
	m.tokensTotal.WithLabelValues(apiKeyID, model, "completion").Add(float64(completionTokens))
}

// IncActiveRequests 增加活跃请求
func (m *PrometheusMetrics) IncActiveRequests() {
	m.activeRequests.Inc()
}

// DecActiveRequests 减少活跃请求
func (m *PrometheusMetrics) DecActiveRequests() {
	m.activeRequests.Dec()
}

// Handler 返回 Prometheus 处理器
func (m *PrometheusMetrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// statusLabel 将状态码转换为标签
func statusLabel(code int) string {
	if code >= 200 && code < 300 {
		return "2xx"
	} else if code >= 300 && code < 400 {
		return "3xx"
	} else if code >= 400 && code < 500 {
		return "4xx"
	} else if code >= 500 && code < 600 {
		return "5xx"
	}
	return "other"
}

// ActiveRequestsTracker 活跃请求跟踪器
type ActiveRequestsTracker struct {
	metrics *PrometheusMetrics
	counter int64
}

// NewActiveRequestsTracker 创建活跃请求跟踪器
func NewActiveRequestsTracker(metrics *PrometheusMetrics) *ActiveRequestsTracker {
	return &ActiveRequestsTracker{metrics: metrics}
}

// Begin 开始请求
func (t *ActiveRequestsTracker) Begin() {
	atomic.AddInt64(&t.counter, 1)
	t.metrics.IncActiveRequests()
}

// End 结束请求
func (t *ActiveRequestsTracker) End() {
	atomic.AddInt64(&t.counter, -1)
	t.metrics.DecActiveRequests()
}

// Count 获取当前活跃请求数
func (t *ActiveRequestsTracker) Count() int64 {
	return atomic.LoadInt64(&t.counter)
}

// === v0.3.0: AI 友好型指标方法 ===

// RecordRequestDuration 记录端到端请求延迟（含模型时间）
func (m *PrometheusMetrics) RecordRequestDuration(model, keyID string, durationMs float64) {
	m.api2ocRequestDuration.WithLabelValues(model, keyID).Observe(durationMs)
}

// RecordMiddlewareDuration 记录网关自身处理延迟（不含模型时间）
func (m *PrometheusMetrics) RecordMiddlewareDuration(model string, durationMs float64) {
	m.api2ocMiddlewareDuration.WithLabelValues(model).Observe(durationMs)
}

// RecordAPI2Tokens 记录 Token 消耗（v0.3.0 版本）
func (m *PrometheusMetrics) RecordAPI2Tokens(model, keyID string, promptTokens, completionTokens int) {
	m.api2ocTokensTotal.WithLabelValues(model, keyID, "prompt").Add(float64(promptTokens))
	m.api2ocTokensTotal.WithLabelValues(model, keyID, "completion").Add(float64(completionTokens))
}

// RecordAPI2Error 记录错误（v0.3.0 版本）
func (m *PrometheusMetrics) RecordAPI2Error(errorCode, model string) {
	m.api2ocErrorsTotal.WithLabelValues(errorCode, model).Inc()
}

// RecordStreamChunk 记录 SSE chunk 推送
func (m *PrometheusMetrics) RecordStreamChunk(model, keyID string) {
	m.api2ocStreamChunksTotal.WithLabelValues(model, keyID).Inc()
}
