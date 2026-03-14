package monitor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to get metric value from registry
func getMetricValue(m prometheus.Collector, labels ...string) (float64, error) {
	ch := make(chan prometheus.Metric, 100)
	go func() {
		m.Collect(ch)
		close(ch)
	}()

	for metric := range ch {
		var m dto.Metric
		if err := metric.Write(&m); err != nil {
			return 0, err
		}

		// Check label match if specified
		if len(labels) > 0 {
			labelPairs := m.GetLabel()
			matched := true
			for i, label := range labels {
				if i >= len(labelPairs) {
					matched = false
					break
				}
				if labelPairs[i].GetValue() != label {
					matched = false
					break
				}
			}
			if matched {
				return m.Counter.GetValue() + m.Gauge.GetValue() + m.Histogram.GetSampleSum(), nil
			}
			continue
		}

		if m.Counter != nil {
			return m.Counter.GetValue(), nil
		}
		if m.Gauge != nil {
			return m.Gauge.GetValue(), nil
		}
		if m.Histogram != nil {
			return m.Histogram.GetSampleSum(), nil
		}
	}

	return 0, nil
}

// TestNewPrometheusMetrics tests creating new Prometheus metrics
func TestNewPrometheusMetrics(t *testing.T) {
	m := NewPrometheusMetrics()
	require.NotNil(t, m)

	// Verify all metrics are initialized
	assert.NotNil(t, m.requestsTotal)
	assert.NotNil(t, m.requestDuration)
	assert.NotNil(t, m.requestSize)
	assert.NotNil(t, m.responseSize)
	assert.NotNil(t, m.modelRequests)
	assert.NotNil(t, m.modelErrors)
	assert.NotNil(t, m.modelLatency)
	assert.NotNil(t, m.authTotal)
	assert.NotNil(t, m.authErrors)
	assert.NotNil(t, m.backendHealth)
	assert.NotNil(t, m.rateLimitTotal)
	assert.NotNil(t, m.circuitState)
	assert.NotNil(t, m.circuitErrors)
	assert.NotNil(t, m.tokensTotal)
	assert.NotNil(t, m.activeRequests)
}

// TestRecordHTTPRequest tests HTTP request recording
func TestRecordHTTPRequest(t *testing.T) {
	m := NewPrometheusMetrics()

	// Record some HTTP requests
	m.RecordHTTPRequest("GET", "/v1/chat/completions", 200, 100*time.Millisecond)
	m.RecordHTTPRequest("POST", "/v1/chat/completions", 200, 150*time.Millisecond)
	m.RecordHTTPRequest("GET", "/v1/models", 401, 50*time.Millisecond)
	m.RecordHTTPRequest("POST", "/v1/chat/completions", 500, 200*time.Millisecond)

	// Test handler output
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler := m.Handler()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	// Verify metrics are present
	assert.Contains(t, body, `api2openclaw_http_requests_total`)
	assert.Contains(t, body, `method="GET"`)
	assert.Contains(t, body, `endpoint="/v1/chat/completions"`)
	assert.Contains(t, body, `status="2xx"`)
	assert.Contains(t, body, `api2openclaw_http_request_duration_seconds`)
}

// TestRecordModelRequest tests model request recording
func TestRecordModelRequest(t *testing.T) {
	m := NewPrometheusMetrics()

	// Record model requests
	m.RecordModelRequest("gpt-4", "backend-1", 200, 500*time.Millisecond)
	m.RecordModelRequest("gpt-4", "backend-1", 200, 300*time.Millisecond)
	m.RecordModelRequest("gpt-3.5", "backend-2", 200, 150*time.Millisecond)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(t, body, `api2openclaw_model_requests_total`)
	assert.Contains(t, body, `model="gpt-4"`)
	assert.Contains(t, body, `backend="backend-1"`)
	assert.Contains(t, body, `api2openclaw_model_latency_seconds`)
}

// TestRecordModelError tests model error recording
func TestRecordModelError(t *testing.T) {
	m := NewPrometheusMetrics()

	// Record model errors
	m.RecordModelError("gpt-4", "backend-1", "timeout")
	m.RecordModelError("gpt-4", "backend-1", "rate_limit")
	m.RecordModelError("gpt-3.5", "backend-2", "connection_error")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(t, body, `api2openclaw_model_errors_total`)
	assert.Contains(t, body, `error_type="timeout"`)
	assert.Contains(t, body, `error_type="rate_limit"`)
}

// TestRecordAuth tests authentication recording
func TestRecordAuth(t *testing.T) {
	m := NewPrometheusMetrics()

	// Record auth attempts
	m.RecordAuth("api_key", "success")
	m.RecordAuth("api_key", "success")
	m.RecordAuth("api_key", "failed")
	m.RecordAuth("jwt", "success")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(t, body, `api2openclaw_auth_requests_total`)
	assert.Contains(t, body, `method="api_key"`)
	assert.Contains(t, body, `status="success"`)
	assert.Contains(t, body, `status="failed"`)
}

// TestRecordAuthError tests authentication error recording
func TestRecordAuthError(t *testing.T) {
	m := NewPrometheusMetrics()

	// Record auth errors
	m.RecordAuthError("invalid_key")
	m.RecordAuthError("expired_key")
	m.RecordAuthError("missing_key")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(t, body, `api2openclaw_auth_errors_total`)
	assert.Contains(t, body, `error_type="invalid_key"`)
	assert.Contains(t, body, `error_type="expired_key"`)
}

// TestSetBackendHealth tests backend health status
func TestSetBackendHealth(t *testing.T) {
	m := NewPrometheusMetrics()

	// Set backend health
	m.SetBackendHealth("backend-1", "Primary Backend", true)
	m.SetBackendHealth("backend-2", "Secondary Backend", false)
	m.SetBackendHealth("backend-3", "Tertiary Backend", true)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(t, body, `api2openclaw_backend_health`)
	assert.Contains(t, body, `backend_id="backend-1"`)
	assert.Contains(t, body, `backend_name="Primary Backend"`)
	assert.Contains(t, body, `api2openclaw_backend_health{backend_id="backend-1",backend_name="Primary Backend"} 1`)
	assert.Contains(t, body, `api2openclaw_backend_health{backend_id="backend-2",backend_name="Secondary Backend"} 0`)
}

// TestRecordRateLimit tests rate limit recording
func TestRecordRateLimit(t *testing.T) {
	m := NewPrometheusMetrics()

	// Record rate limits
	m.RecordRateLimit("key-1", "rpm")
	m.RecordRateLimit("key-1", "rpm")
	m.RecordRateLimit("key-2", "rph")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(t, body, `api2openclaw_rate_limit_total`)
	assert.Contains(t, body, `api_key_id="key-1"`)
	assert.Contains(t, body, `limit_type="rpm"`)
}

// TestSetCircuitState tests circuit breaker state
func TestSetCircuitState(t *testing.T) {
	m := NewPrometheusMetrics()

	// Set circuit states
	m.SetCircuitState("circuit-1", StateClosed)
	m.SetCircuitState("circuit-2", StateOpen)
	m.SetCircuitState("circuit-3", StateHalfOpen)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(t, body, `api2openclaw_circuit_state`)
	// StateClosed = 0, StateOpen = 1, StateHalfOpen = 2
	assert.Contains(t, body, `name="circuit-1"} 0`)
	assert.Contains(t, body, `name="circuit-2"} 1`)
	assert.Contains(t, body, `name="circuit-3"} 2`)
}

// TestRecordCircuitError tests circuit breaker error recording
func TestRecordCircuitError(t *testing.T) {
	m := NewPrometheusMetrics()

	// Record circuit errors
	m.RecordCircuitError("circuit-1")
	m.RecordCircuitError("circuit-1")
	m.RecordCircuitError("circuit-2")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(t, body, `api2openclaw_circuit_errors_total`)
	assert.Contains(t, body, `name="circuit-1"`)
}

// TestRecordTokens tests token usage recording
func TestRecordTokens(t *testing.T) {
	m := NewPrometheusMetrics()

	// Record tokens
	m.RecordTokens("key-1", "gpt-4", 100, 50)
	m.RecordTokens("key-1", "gpt-4", 200, 100)
	m.RecordTokens("key-2", "gpt-3.5", 50, 25)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(t, body, `api2openclaw_tokens_total`)
	assert.Contains(t, body, `api_key_id="key-1"`)
	assert.Contains(t, body, `model="gpt-4"`)
	assert.Contains(t, body, `token_type="prompt"`)
	assert.Contains(t, body, `token_type="completion"`)
}

// TestActiveRequests tests active requests tracking
func TestActiveRequests(t *testing.T) {
	m := NewPrometheusMetrics()
	tracker := NewActiveRequestsTracker(m)

	// Initially no active requests
	assert.Equal(t, int64(0), tracker.Count())

	// Add some active requests
	tracker.Begin()
	tracker.Begin()
	tracker.Begin()

	assert.Equal(t, int64(3), tracker.Count())

	// Complete some requests
	tracker.End()
	tracker.End()

	assert.Equal(t, int64(1), tracker.Count())

	// Complete remaining
	tracker.End()

	assert.Equal(t, int64(0), tracker.Count())

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	assert.Contains(t, body, `api2openclaw_active_requests`)
}

// TestStatusLabel tests status label conversion
func TestStatusLabel(t *testing.T) {
	tests := []struct {
		code    int
		expects string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{299, "2xx"},
		{301, "3xx"},
		{302, "3xx"},
		{400, "4xx"},
		{401, "4xx"},
		{404, "4xx"},
		{500, "5xx"},
		{502, "5xx"},
		{503, "5xx"},
		{100, "other"},
		{600, "other"},
	}

	for _, tt := range tests {
		t.Run(tt.expects, func(t *testing.T) {
			result := statusLabel(tt.code)
			assert.Equal(t, tt.expects, result)
		})
	}
}

// TestPrometheusMetricsIntegration tests the full integration
func TestPrometheusMetricsIntegration(t *testing.T) {
	m := NewPrometheusMetrics()
	tracker := NewActiveRequestsTracker(m)

	// Simulate a complete request flow
	tracker.Begin()

	m.RecordAuth("api_key", "success")
	m.RecordHTTPRequest("POST", "/v1/chat/completions", 200, 250*time.Millisecond)
	m.RecordModelRequest("gpt-4", "backend-1", 200, 200*time.Millisecond)
	m.RecordTokens("key-1", "gpt-4", 150, 75)
	m.SetBackendHealth("backend-1", "Primary Backend", true)

	tracker.End()

	// Verify all metrics are exported
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	m.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	// Check all metric types are present
	metricNames := []string{
		"api2openclaw_http_requests_total",
		"api2openclaw_http_request_duration_seconds",
		"api2openclaw_model_requests_total",
		"api2openclaw_model_latency_seconds",
		"api2openclaw_tokens_total",
		"api2openclaw_backend_health",
		"api2openclaw_auth_requests_total",
		"api2openclaw_active_requests",
	}

	for _, name := range metricNames {
		if !strings.Contains(body, name) {
			t.Errorf("Expected metric %s not found in output", name)
		}
	}
}

// TestPrometheusHandler tests the HTTP handler
func TestPrometheusHandler(t *testing.T) {
	m := NewPrometheusMetrics()

	// Record some metrics
	m.RecordHTTPRequest("GET", "/test", 200, 100*time.Millisecond)

	// Test the handler
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler := m.Handler()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "api2openclaw_http_requests_total")
}
