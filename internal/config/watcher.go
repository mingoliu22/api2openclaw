package config

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ReloadCallback 配置重载回调函数
type ReloadCallback func(ctx context.Context, oldCfg, newCfg *Config) error

// Watcher 配置文件监听器
type Watcher struct {
	mu         sync.RWMutex
	configPath string
	config     *Config
	callbacks  []ReloadCallback
	watcher    *fsnotify.Watcher
	debounce   time.Duration
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewWatcher 创建配置监听器
func NewWatcher(configPath string, cfg *Config) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		configPath: configPath,
		config:     cfg,
		callbacks:  make([]ReloadCallback, 0),
		watcher:    w,
		debounce:   cfg.HotReload.Debounce,
	}, nil
}

// OnReload 注册重载回调
func (w *Watcher) OnReload(cb ReloadCallback) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, cb)
}

// Start 启动监听
func (w *Watcher) Start(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)

	// 监听配置文件
	if err := w.watcher.Add(w.configPath); err != nil {
		return err
	}

	w.wg.Add(1)
	go w.watchLoop(ctx)

	log.Printf("[Config] Hot reload enabled, watching: %s", w.configPath)
	return nil
}

// watchLoop 监听循环
func (w *Watcher) watchLoop(ctx context.Context) {
	defer w.wg.Done()

	var timer *time.Timer
	var debounceCh <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// 只处理写入和创建事件
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			log.Printf("[Config] Config file changed: %s", event.Op)

			// 防抖处理
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(w.debounce)
			debounceCh = timer.C

		case <-debounceCh:
			if err := w.reloadConfig(ctx); err != nil {
				log.Printf("[Config] Failed to reload config: %v", err)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[Config] Watcher error: %v", err)
		}
	}
}

// reloadConfig 重载配置
func (w *Watcher) reloadConfig(ctx context.Context) error {
	log.Println("[Config] Reloading configuration...")

	// 加载新配置
	newCfg, err := Load(w.configPath)
	if err != nil {
		return err
	}

	w.mu.Lock()
	oldCfg := w.config
	w.mu.Unlock()

	// 调用所有回调
	for _, cb := range w.callbacks {
		if err := cb(ctx, oldCfg, newCfg); err != nil {
			log.Printf("[Config] Reload callback failed: %v", err)
			return err
		}
	}

	// 更新配置
	w.mu.Lock()
	w.config = newCfg
	w.mu.Unlock()

	log.Println("[Config] Configuration reloaded successfully")
	return nil
}

// GetConfig 获取当前配置
func (w *Watcher) GetConfig() *Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.config
}

// Close 关闭监听器
func (w *Watcher) Close() error {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	return w.watcher.Close()
}

// Trigger 手动触发重载
func (w *Watcher) Trigger(ctx context.Context) error {
	return w.reloadConfig(ctx)
}

// ReloadChan 创建一个重载通道
func (w *Watcher) ReloadChan() <-chan struct{} {
	ch := make(chan struct{}, 1)

	// 注册回调
	w.OnReload(func(ctx context.Context, oldCfg, newCfg *Config) error {
		select {
		case ch <- struct{}{}:
		default:
		}
		return nil
	})

	return ch
}
