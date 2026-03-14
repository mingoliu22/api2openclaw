package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/openclaw/api2openclaw/internal/config"
	"github.com/openclaw/api2openclaw/internal/server"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "api2openclaw",
	Short: "API gateway and format converter for local LLM models",
	Long: `api2openclaw is a gateway service that provides:
- Unified API authentication with API Keys
- Format conversion from JSON to OpenClaw plain text
- Model routing and load balancing
- Usage monitoring and rate limiting`,
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "configs/config.yaml", "config file path")
}

func initConfig() {
	if cfgFile != "" {
		// 配置文件路径已在 flag 中设置
	}
}

func run() {
	// 加载配置
	cfg, err := config.Load(cfgFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建服务器
	srv, err := server.New(cfg, cfgFile)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// 创建配置监听器
	var watcher *config.Watcher
	if cfg.HotReload.Enabled {
		watcher, err = config.NewWatcher(cfgFile, cfg)
		if err != nil {
			log.Fatalf("Failed to create config watcher: %v", err)
		}

		// 注册重载回调
		watcher.OnReload(func(ctx context.Context, oldCfg, newCfg *config.Config) error {
			return srv.ReloadConfig(ctx, newCfg)
		})

		// 启动监听
		if err := watcher.Start(context.Background()); err != nil {
			log.Fatalf("Failed to start config watcher: %v", err)
		}
	}

	// 打印启动信息
	printStartupInfo(cfg)

	// 启动服务器
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 优雅关闭
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	log.Printf("Received signal %v, shutting down gracefully...", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// 关闭 watcher
	if watcher != nil {
		if err := watcher.Close(); err != nil {
			log.Printf("Watcher close error: %v", err)
		}
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func printStartupInfo(cfg *config.Config) {
	fmt.Println(`
╔══════════════════════════════════════════════════════════════════╗
║                     api2openclaw                                 ║
║              Local LLM Gateway & Format Converter                 ║
╠══════════════════════════════════════════════════════════════════╣`)
	fmt.Printf("║ Version: %-20s Build: %-28s ║\n", Version, BuildTime)
	fmt.Printf("║ Server:  %s:%d                                              ║\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
