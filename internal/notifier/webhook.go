package notifier

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// WebhookManager Webhook 管理器
type WebhookManager struct {
	mu          sync.RWMutex
	webhooks    map[string]*WebhookConfig
	history     map[string][]*WebhookEvent
	historySize int
	client      *http.Client
}

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	URL           string            `json:"url"`
	Secret        string            `json:"secret,omitempty"`
	Events        []string          `json:"events"`
	Headers       map[string]string `json:"headers,omitempty"`
	Enabled       bool              `json:"enabled"`
	Timeout       time.Duration     `json:"timeout"`
	RetryPolicy   *RetryPolicy      `json:"retry_policy"`
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	RetryDelay    time.Duration `json:"retry_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// WebhookEvent Webhook 事件
type WebhookEvent struct {
	EventID      string                 `json:"event_id"`
	EventType    string                 `json:"event_type"`
	Timestamp    time.Time              `json:"timestamp"`
	Source       string                 `json:"source"`
	Data         map[string]interface{} `json:"data"`
	TriggeredBy  string                 `json:"triggered_by"`
	RetryCount   int                    `json:"retry_count"`
}

// WebhookPayload Webhook 负载
type WebhookPayload struct {
	EventID   string                 `json:"event_id"`
	Event     string                 `json:"event"`
	Timestamp string                 `json:"timestamp"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Signature string                 `json:"signature,omitempty"`
}

// NewWebhookManager 创建 Webhook 管理器
func NewWebhookManager() *WebhookManager {
	return &WebhookManager{
		webhooks:    make(map[string]*WebhookConfig),
		history:     make(map[string][]*WebhookEvent),
		historySize: 1000,
		client:      &http.Client{Timeout: 30 * time.Second},
	}
}

// Register 注册 Webhook
func (m *WebhookManager) Register(config *WebhookConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.ID == "" {
		return fmt.Errorf("webhook ID is required")
	}

	if config.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	// 默认重试策略
	if config.RetryPolicy == nil {
		config.RetryPolicy = &RetryPolicy{
			MaxRetries:    3,
			RetryDelay:    1 * time.Second,
			BackoffFactor: 2.0,
		}
	}

	m.webhooks[config.ID] = config

	return nil
}

// Unregister 注销 Webhook
func (m *WebhookManager) Unregister(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.webhooks[id]; !ok {
		return fmt.Errorf("webhook not found: %s", id)
	}

	delete(m.webhooks, id)
	return nil
}

// List 列出所有 Webhook
func (m *WebhookManager) List() []*WebhookConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*WebhookConfig, 0, len(m.webhooks))
	for _, config := range m.webhooks {
		result = append(result, config)
	}

	return result
}

// Get 获取 Webhook 配置
func (m *WebhookManager) Get(id string) (*WebhookConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, ok := m.webhooks[id]
	if !ok {
		return nil, fmt.Errorf("webhook not found: %s", id)
	}

	// 返回副本
	copy := *config
	return &copy, nil
}

// Trigger 触发事件
func (m *WebhookManager) Trigger(ctx context.Context, eventType string, data map[string]interface{}) error {
	return m.TriggerWithSource(ctx, eventType, data, "system")
}

// TriggerWithSource 带来源地触发事件
func (m *WebhookManager) TriggerWithSource(ctx context.Context, eventType string, data map[string]interface{}, source string) error {
	m.mu.RLock()
	webhooks := make([]*WebhookConfig, 0, len(m.webhooks))
	for _, config := range m.webhooks {
		if config.Enabled && m.shouldNotify(config, eventType) {
			webhooks = append(webhooks, config)
		}
	}
	m.mu.RUnlock()

	// 并发发送 Webhook
	var wg sync.WaitGroup
	errChan := make(chan error, len(webhooks))

	for _, config := range webhooks {
		wg.Add(1)
		go func(cfg *WebhookConfig) {
			defer wg.Done()

			event := &WebhookEvent{
				EventID:     generateEventID(),
				EventType:   eventType,
				Timestamp:   time.Now(),
				Source:      source,
				Data:        data,
				TriggeredBy:  source,
				RetryCount:  0,
			}

			err := m.sendWebhook(ctx, cfg, event)
			if err != nil {
				errChan <- err
			}

			// 记录历史
			m.recordHistory(event)
		}(config)
	}

	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("webhook delivery failed: %d errors", len(errors))
	}

	return nil
}

// sendWebhook 发送 Webhook
func (m *WebhookManager) sendWebhook(ctx context.Context, config *WebhookConfig, event *WebhookEvent) error {
	// 构建负载
	payload := &WebhookPayload{
		EventID:   event.EventID,
		Event:     event.EventType,
		Timestamp: event.Timestamp.Format(time.RFC3339),
		Source:    event.Source,
		Data:      event.Data,
	}

	// 添加签名
	if config.Secret != "" {
		payload.Signature = m.signPayload(payload, config.Secret)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", config.URL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "api2openclaw-webhook/1.0")

	// 添加自定义头
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// 如果有签名，添加到头
	if payload.Signature != "" {
		req.Header.Set("X-Webhook-Signature", payload.Signature)
		req.Header.Set("X-Webhook-ID", event.EventID)
		req.Header.Set("X-Webhook-Event", event.EventType)
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()
	req = req.WithContext(ctx)

	// 发送请求（带重试）
	var lastErr error
	for attempt := 0; attempt <= config.RetryPolicy.MaxRetries; attempt++ {
		resp, err := m.client.Do(req)
		if err != nil {
			lastErr = err
			m.retryRequest(req, attempt, config.RetryPolicy)
			continue
		}

		// 检查响应
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return lastErr
}

// shouldNotify 检查是否应该通知
func (m *WebhookManager) shouldNotify(config *WebhookConfig, eventType string) bool {
	if len(config.Events) == 0 {
		return true // 订阅所有事件
	}

	for _, event := range config.Events {
		if event == eventType || event == "*" {
			return true
		}
	}

	return false
}

// signPayload 签名负载
func (m *WebhookManager) signPayload(payload *WebhookPayload, secret string) string {
	jsonData, _ := json.Marshal(payload)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(jsonData)
	return hex.EncodeToString(h.Sum(nil))
}

// retryRequest 重试请求
func (m *WebhookManager) retryRequest(req *http.Request, attempt int, policy *RetryPolicy) {
	if attempt >= policy.MaxRetries {
		return
	}

	// 计算延迟（指数退避）
	delay := policy.RetryDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * policy.BackoffFactor)
	}

	time.Sleep(delay)

	// 重置 Body
	body := bytes.NewBuffer(nil)
	// 需要重新读取原始 body
	// 这里简化处理
	req.Body = io.NopCloser(body)
}

// recordHistory 记录历史
func (m *WebhookManager) recordHistory(event *WebhookEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.history[event.EventType] == nil {
		m.history[event.EventType] = make([]*WebhookEvent, 0, m.historySize)
	}

	history := m.history[event.EventType]
	if len(history) >= m.historySize {
		// 移除最旧的记录
		m.history[event.EventType] = history[1:]
	}

	m.history[event.EventType] = append(history, event)
}

// GetHistory 获取历史记录
func (m *WebhookManager) GetHistory(eventType string, limit int) []*WebhookEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history, ok := m.history[eventType]
	if !ok {
		return nil
	}

	if limit <= 0 || limit > len(history) {
		limit = len(history)
	}

	// 返回最近的记录
	start := len(history) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*WebhookEvent, limit)
	copy(result, history[start:])
	return result
}

// generateEventID 生成事件 ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// VerifyWebhookSignature 验证 Webhook 签名
func VerifyWebhookSignature(payload []byte, signature, secret string) bool {
	expectedSig := signPayloadBytes(payload, secret)
	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// signPayloadBytes 签名负载字节
func signPayloadBytes(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// EventTypes 事件类型常量
const (
	EventTypeRateLimitExceeded    = "rate_limit.exceeded"
	EventTypeCircuitBreakerOpen   = "circuit_breaker.open"
	EventTypeBackendDown         = "backend.down"
	EventTypeBackendUp           = "backend.up"
	EventTypeAPICreated          = "api_key.created"
	EventTypeAPIRevoked          = "api_key.revoked"
	EventTypeQuotaExceeded       = "quota.exceeded"
	EventTypeAnomalyDetected    = "anomaly.detected"
	EventTypeConfigReloaded      = "config.reloaded"
	EventTypeSystemAlert         = "system.alert"
)

// AlertData 告警数据
type AlertData struct {
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// TriggerAlert 触发告警
func (m *WebhookManager) TriggerAlert(ctx context.Context, severity, title, description string, metadata map[string]interface{}) error {
	data := map[string]interface{}{
		"severity":    severity,
		"title":       title,
		"description": description,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	for k, v := range metadata {
		data[k] = v
	}

	return m.TriggerWithSource(ctx, EventTypeSystemAlert, data, "alert_system")
}

// WebhookTestResult Webhook 测试结果
type WebhookTestResult struct {
	WebhookID string    `json:"webhook_id"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
	Latency   int64     `json:"latency_ms"`
	Timestamp time.Time `json:"timestamp"`
}

// Test 测试 Webhook
func (m *WebhookManager) Test(ctx context.Context, webhookID string) (*WebhookTestResult, error) {
	m.mu.RLock()
	config, ok := m.webhooks[webhookID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("webhook not found: %s", webhookID)
	}

	startTime := time.Now()

	// 发送测试事件
	event := &WebhookEvent{
		EventID:     "test_" + generateEventID(),
		EventType:   "webhook.test",
		Timestamp:   startTime,
		Source:      "system",
		Data: map[string]interface{}{
			"message": "This is a test webhook event",
		},
	}

	err := m.sendWebhook(ctx, config, event)

	latency := time.Since(startTime).Milliseconds()

	return &WebhookTestResult{
		WebhookID: webhookID,
		Success:   err == nil,
		Error:     func() string { if err != nil { return err.Error() }; return "" }(),
		Latency:   latency,
		Timestamp: startTime,
	}, nil
}
