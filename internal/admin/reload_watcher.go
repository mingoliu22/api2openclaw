package admin

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// ReloadListener 配置重载监听器接口
type ReloadListener interface {
	OnModelsChanged()
}

// ReloadWatcher 配置重载监听器
type ReloadWatcher struct {
	db        *sqlx.DB
	listeners []ReloadListener
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
}

// NewReloadWatcher 创建配置重载监听器
func NewReloadWatcher(db *sqlx.DB) *ReloadWatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &ReloadWatcher{
		db:     db,
		ctx:    ctx,
		cancel: cancel,
	}
}

// AddListener 添加监听器
func (w *ReloadWatcher) AddListener(listener ReloadListener) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.listeners = append(w.listeners, listener)
}

// notifyListeners 通知所有监听器
func (w *ReloadWatcher) notifyListeners() {
	w.mu.RLock()
	listeners := make([]ReloadListener, len(w.listeners))
	copy(listeners, w.listeners)
	w.mu.RUnlock()

	for _, listener := range listeners {
		listener.OnModelsChanged()
	}
}

// Start 启动监听
func (w *ReloadWatcher) Start() {
	w.wg.Add(1)
	go w.listen()
}

// Stop 停止监听
func (w *ReloadWatcher) Stop() {
	w.cancel()
	w.wg.Wait()
}

// listen 监听 PostgreSQL NOTIFY 事件
func (w *ReloadWatcher) listen() {
	defer w.wg.Done()

	// 使用轮询方式检查配置变更（更简单可靠）
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			// 检查是否有配置变更（通过查询最后更新时间）
			// 这里简化实现：定期触发检查
			w.checkAndNotify()
		}
	}
}

var lastModelUpdate time.Time

// checkAndNotify 检查配置变更并通知
func (w *ReloadWatcher) checkAndNotify() {
	// 查询最新的模型配置更新时间
	var updatedAt time.Time
	err := w.db.Get(&updatedAt, "SELECT MAX(updated_at) FROM models WHERE updated_at > $1", lastModelUpdate)
	if err == nil && !updatedAt.IsZero() {
		// 有新的更新
		log.Println("[ReloadWatcher] detected model config change")
		lastModelUpdate = updatedAt
		w.notifyListeners()
	}
}

// NotifyModelsChanged 手动触发模型配置变更通知（用于测试或其他触发方式）
func (w *ReloadWatcher) NotifyModelsChanged() {
	_, err := w.db.Exec("NOTIFY models_config_changed")
	if err != nil {
		log.Printf("[ReloadWatcher] failed to send notification: %v", err)
		// 降级为直接通知
		w.notifyListeners()
	}
}
