package router

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/openclaw/api2openclaw/internal/models"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	router     *Router
	httpClient *http.Client
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(router *Router, httpClient *http.Client) *HealthChecker {
	return &HealthChecker{
		router:     router,
		httpClient: httpClient,
		stopCh:     make(chan struct{}),
	}
}

// Run 启动健康检查
func (h *HealthChecker) Run() {
	h.wg.Add(1)
	go h.checkLoop()

	log.Printf("[HealthChecker] Started")
}

// Stop 停止健康检查
func (h *HealthChecker) Stop() {
	close(h.stopCh)
	h.wg.Wait()
	log.Printf("[HealthChecker] Stopped")
}

// checkLoop 健康检查循环
func (h *HealthChecker) checkLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 首次检查
	h.checkAll()

	for {
		select {
		case <-ticker.C:
			h.checkAll()
		case <-h.stopCh:
			return
		}
	}
}

// checkAll 检查所有后端
func (h *HealthChecker) checkAll() {
	backends := h.router.ListBackends()

	for _, backend := range backends {
		if !backend.HealthCheck.Enabled {
			continue
		}

		go h.checkBackend(backend)
	}
}

// checkBackend 检查单个后端
func (h *HealthChecker) checkBackend(backend *models.Backend) {
	// 获取健康检查配置
	timeout := backend.HealthCheck.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	url := backend.BaseURL + backend.HealthCheck.Endpoint
	if url == "" {
		url = backend.BaseURL + "/models"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		h.markUnhealthy(backend.ID)
		log.Printf("[HealthChecker] %s: create request failed: %v", backend.ID, err)
		return
	}

	// 设置认证头
	if backend.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+backend.APIKey)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.markUnhealthy(backend.ID)
		log.Printf("[HealthChecker] %s: request failed: %v", backend.ID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		h.markHealthy(backend.ID)
		log.Printf("[HealthChecker] %s: healthy (%d)", backend.ID, resp.StatusCode)
	} else {
		h.markUnhealthy(backend.ID)
		log.Printf("[HealthChecker] %s: unhealthy (%d)", backend.ID, resp.StatusCode)
	}
}

// markHealthy 标记为健康
func (h *HealthChecker) markHealthy(backendID string) {
	h.router.UpdateBackendStatus(backendID, models.BackendStatusHealthy)
}

// markUnhealthy 标记为不健康
func (h *HealthChecker) markUnhealthy(backendID string) {
	h.router.UpdateBackendStatus(backendID, models.BackendStatusUnhealthy)
}

// CheckNow 立即检查指定后端
func (h *HealthChecker) CheckNow(backendID string) {
	if backend, ok := h.router.GetBackend(backendID); ok {
		h.checkBackend(backend)
	}
}
