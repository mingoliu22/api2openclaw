package monitor

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// AnomalyDetector 异常检测器（检测 API Key 泄露）
type AnomalyDetector struct {
	mu              sync.RWMutex
	keyStates       map[string]*keyAnomalyState
	alertThresholds AnomalyThresholds
	alertChannel    chan *AnomalyAlert
}

// keyAnomalyState Key 异常状态
type keyAnomalyState struct {
	apiKeyID          string
	requestHistory    []requestRecord
	windowSize        int
	avgRequestsPerMin float64
	stdDev            float64
	alertCount        int
	lastAlertTime     time.Time
}

// requestRecord 请求记录
type requestRecord struct {
	timestamp   time.Time
	ipAddress   string
	userAgent   string
	success     bool
}

// AnomalyThresholds 异常检测阈值
type AnomalyThresholds struct {
	MinSampleSize      int           // 最小样本数
	Multiplier         float64       // 标准差倍数
	ConsecutiveErrors  int           // 连续错误数阈值
	BurstThreshold     int           // 突发请求阈值
	BurstWindowSeconds time.Duration // 突发检测窗口
	AlertCooldown      time.Duration // 告警冷却时间
}

// AnomalyAlert 异常告警
type AnomalyAlert struct {
	AlertID       string              `json:"alert_id"`
	APIKeyID      string              `json:"api_key_id"`
	TenantID      string              `json:"tenant_id"`
	AlertType     string              `json:"alert_type"`
	Severity      string              `json:"severity"`
	Message       string              `json:"message"`
	Metadata      map[string]interface{} `json:"metadata"`
	Timestamp     time.Time           `json:"timestamp"`
}

// NewAnomalyDetector 创建异常检测器
func NewAnomalyDetector(alertChannelSize int) *AnomalyDetector {
	if alertChannelSize <= 0 {
		alertChannelSize = 1000
	}

	return &AnomalyDetector{
		keyStates:    make(map[string]*keyAnomalyState),
		alertChannel: make(chan *AnomalyAlert, alertChannelSize),
		alertThresholds: AnomalyThresholds{
			MinSampleSize:      20,
			Multiplier:         3.0,
			ConsecutiveErrors:  10,
			BurstThreshold:     100,
			BurstWindowSeconds: 60 * time.Second,
			AlertCooldown:      5 * time.Minute,
		},
	}
}

// RecordRequest 记录请求并检测异常
func (d *AnomalyDetector) RecordRequest(apiKeyID, ipAddress, userAgent string, success bool) *AnomalyAlert {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	state, ok := d.keyStates[apiKeyID]
	if !ok {
		state = &keyAnomalyState{
			apiKeyID:       apiKeyID,
			requestHistory: make([]requestRecord, 0, 100),
			windowSize:     100,
		}
		d.keyStates[apiKeyID] = state
	}

	// 添加请求记录
	state.requestHistory = append(state.requestHistory, requestRecord{
		timestamp: now,
		ipAddress: ipAddress,
		userAgent: userAgent,
		success:   success,
	})

	// 保持窗口大小
	if len(state.requestHistory) > state.windowSize {
		state.requestHistory = state.requestHistory[1:]
	}

	// 检测各种异常
	return d.detectAnomalies(state, now)
}

// detectAnomalies 检测异常
func (d *AnomalyDetector) detectAnomalies(state *keyAnomalyState, now time.Time) *AnomalyAlert {
	// 检查冷却时间
	if now.Sub(state.lastAlertTime) < d.alertThresholds.AlertCooldown {
		return nil
	}

	// 1. 检测突发流量异常
	if alert := d.detectBurstAnomaly(state, now); alert != nil {
		state.lastAlertTime = now
		return alert
	}

	// 2. 检测连续失败异常
	if alert := d.detectConsecutiveErrors(state, now); alert != nil {
		state.lastAlertTime = now
		return alert
	}

	// 3. 检测流量异常（基于标准差）
	if alert := d.detectVolumeAnomaly(state, now); alert != nil {
		state.lastAlertTime = now
		return alert
	}

	// 4. 检测 IP 地址异常
	if alert := d.detectIPAnomaly(state, now); alert != nil {
		state.lastAlertTime = now
		return alert
	}

	return nil
}

// detectBurstAnomaly 检测突发流量异常
func (d *AnomalyDetector) detectBurstAnomaly(state *keyAnomalyState, now time.Time) *AnomalyAlert {
	// 统计最近时间窗口内的请求数
	windowStart := now.Add(-d.alertThresholds.BurstWindowSeconds)
	recentCount := 0

	for _, req := range state.requestHistory {
		if req.timestamp.After(windowStart) {
			recentCount++
		}
	}

	if recentCount > d.alertThresholds.BurstThreshold {
		return d.createAlert(state.apiKeyID, "burst_traffic", "high",
			fmt.Sprintf("Detected burst traffic: %d requests in %v (threshold: %d)",
				recentCount, d.alertThresholds.BurstWindowSeconds, d.alertThresholds.BurstThreshold),
			map[string]interface{}{
				"request_count": recentCount,
				"window": d.alertThresholds.BurstWindowSeconds.String(),
				"threshold": d.alertThresholds.BurstThreshold,
			})
	}

	return nil
}

// detectConsecutiveErrors 检测连续失败异常
func (d *AnomalyDetector) detectConsecutiveErrors(state *keyAnomalyState, now time.Time) *AnomalyAlert {
	consecutiveErrors := 0

	// 从最近的请求向前计数
	for i := len(state.requestHistory) - 1; i >= 0; i-- {
		if !state.requestHistory[i].success {
			consecutiveErrors++
		} else {
			break
		}
	}

	if consecutiveErrors >= d.alertThresholds.ConsecutiveErrors {
		return d.createAlert(state.apiKeyID, "consecutive_errors", "high",
			fmt.Sprintf("Detected %d consecutive errors (threshold: %d)",
				consecutiveErrors, d.alertThresholds.ConsecutiveErrors),
			map[string]interface{}{
				"error_count": consecutiveErrors,
				"threshold": d.alertThresholds.ConsecutiveErrors,
			})
	}

	return nil
}

// detectVolumeAnomaly 检测流量异常（基于标准差）
func (d *AnomalyDetector) detectVolumeAnomaly(state *keyAnomalyState, now time.Time) *AnomalyAlert {
	if len(state.requestHistory) < d.alertThresholds.MinSampleSize {
		return nil
	}

	// 计算每分钟请求数
	requestsPerMinute := d.calculateRequestsPerMinute(state)

	// 计算平均值和标准差
	mean, stdDev := d.calculateMeanAndStdDev(requestsPerMinute)

	if stdDev == 0 {
		return nil
	}

	// 检查最近的请求率是否异常
	if len(requestsPerMinute) > 0 {
		recentRPM := requestsPerMinute[len(requestsPerMinute)-1]
		zScore := math.Abs((recentRPM - mean) / stdDev)

		if zScore > d.alertThresholds.Multiplier {
			return d.createAlert(state.apiKeyID, "volume_anomaly", "medium",
				fmt.Sprintf("Detected volume anomaly: %.2f requests/min (avg: %.2f, std: %.2f, z-score: %.2f)",
					recentRPM, mean, stdDev, zScore),
				map[string]interface{}{
					"current_rpm": recentRPM,
					"avg_rpm": mean,
					"std_dev": stdDev,
					"z_score": zScore,
				})
		}
	}

	return nil
}

// detectIPAnomaly 检测 IP 地址异常
func (d *AnomalyDetector) detectIPAnomaly(state *keyAnomalyState, now time.Time) *AnomalyAlert {
	if len(state.requestHistory) < 20 {
		return nil
	}

	// 统计 IP 地址
	ipCount := make(map[string]int)
	for _, req := range state.requestHistory {
		ipCount[req.ipAddress]++
	}

	// 检查是否有异常数量的不同 IP
	if len(ipCount) > 10 {
		return d.createAlert(state.apiKeyID, "ip_anomaly", "medium",
			fmt.Sprintf("Detected %d different IP addresses (may indicate key leak)", len(ipCount)),
			map[string]interface{}{
				"unique_ips": len(ipCount),
				"ip_distribution": ipCount,
			})
	}

	return nil
}

// calculateRequestsPerMinute 计算每分钟请求数
func (d *AnomalyDetector) calculateRequestsPerMinute(state *keyAnomalyState) []float64 {
	if len(state.requestHistory) == 0 {
		return nil
	}

	// 按分钟分组
	minuteBuckets := make(map[int]int)
	for _, req := range state.requestHistory {
		minute := req.timestamp.Truncate(time.Minute).Unix()
		minuteBuckets[int(minute)]++
	}

	// 转换为数组
	var rpms []float64
	for _, count := range minuteBuckets {
		rpms = append(rpms, float64(count))
	}

	return rpms
}

// calculateMeanAndStdDev 计算平均值和标准差
func (d *AnomalyDetector) calculateMeanAndStdDev(values []float64) (mean, stdDev float64) {
	n := float64(len(values))
	if n == 0 {
		return 0, 0
	}

	// 计算平均值
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / n

	// 计算标准差
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	variance := sumSquares / n
	stdDev = math.Sqrt(variance)

	return mean, stdDev
}

// createAlert 创建告警
func (d *AnomalyDetector) createAlert(apiKeyID, alertType, severity, message string, metadata map[string]interface{}) *AnomalyAlert {
	alert := &AnomalyAlert{
		AlertID:   generateAlertID(),
		APIKeyID:  apiKeyID,
		AlertType: alertType,
		Severity:  severity,
		Message:   message,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	// 发送到告警通道
	select {
	case d.alertChannel <- alert:
	default:
		log.Printf("[AnomalyDetector] Alert channel full, dropping alert")
	}

	return alert
}

// generateAlertID 生成告警 ID
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

// GetAlertChannel 获取告警通道
func (d *AnomalyDetector) GetAlertChannel() <-chan *AnomalyAlert {
	return d.alertChannel
}

// GetKeyState 获取 Key 状态
func (d *AnomalyDetector) GetKeyState(apiKeyID string) *keyAnomalyState {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if state, ok := d.keyStates[apiKeyID]; ok {
		// 返回副本
		copy := *state
		copy.requestHistory = append([]requestRecord(nil), state.requestHistory...)
		return &copy
	}

	return nil
}

// ClearKeyState 清除 Key 状态
func (d *AnomalyDetector) ClearKeyState(apiKeyID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.keyStates, apiKeyID)
}

// AlertHandler 告警处理器接口
type AlertHandler interface {
	HandleAlert(ctx context.Context, alert *AnomalyAlert) error
}

// WebhookAlertHandler Webhook 告警处理器
type WebhookAlertHandler struct {
	webhookURL string
	client     interface{} // http.Client
}

// NewWebhookAlertHandler 创建 Webhook 告警处理器
func NewWebhookAlertHandler(webhookURL string) *WebhookAlertHandler {
	return &WebhookAlertHandler{
		webhookURL: webhookURL,
	}
}

// HandleAlert 处理告警
func (h *WebhookAlertHandler) HandleAlert(ctx context.Context, alert *AnomalyAlert) error {
	// TODO: 实现 HTTP POST 发送到 Webhook URL
	log.Printf("[WebhookAlertHandler] Sending alert to %s: %s", h.webhookURL, alert.Message)
	return nil
}

// AlertRouter 告警路由器（根据严重程度路由到不同处理器）
type AlertRouter struct {
	handlers map[string]AlertHandler // severity -> handler
}

// NewAlertRouter 创建告警路由器
func NewAlertRouter() *AlertRouter {
	return &AlertRouter{
		handlers: make(map[string]AlertHandler),
	}
}

// RegisterHandler 注册处理器
func (r *AlertRouter) RegisterHandler(severity string, handler AlertHandler) {
	r.handlers[severity] = handler
}

// Route 路由告警
func (r *AlertRouter) Route(ctx context.Context, alert *AnomalyAlert) error {
	handler, ok := r.handlers[alert.Severity]
	if !ok {
		// 尝试默认处理器
		handler, ok = r.handlers["default"]
		if !ok {
			return fmt.Errorf("no handler for severity: %s", alert.Severity)
		}
	}

	return handler.HandleAlert(ctx, alert)
}
