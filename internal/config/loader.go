package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Load 加载配置文件
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// 设置默认值
	setDefaults(&cfg)

	// 验证配置
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults(cfg *Config) {
	// 服务器默认值
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30 * time.Second
	}
	if cfg.Server.ShutdownTimeout == 0 {
		cfg.Server.ShutdownTimeout = 10 * time.Second
	}
	if cfg.Server.BasePath == "" {
		cfg.Server.BasePath = "/v1"
	}

	// 认证默认值
	if cfg.Auth.StoreType == "" {
		cfg.Auth.StoreType = "postgres"
	}
	if cfg.Auth.Database.Host == "" {
		cfg.Auth.Database.Host = "localhost"
	}
	if cfg.Auth.Database.Port == 0 {
		cfg.Auth.Database.Port = 5432
	}
	if cfg.Auth.Database.SSLMode == "" {
		cfg.Auth.Database.SSLMode = "disable"
	}

	// 格式转换默认值
	if cfg.Converter.InputFormat == "" {
		cfg.Converter.InputFormat = "deepseek"
	}
	if cfg.Converter.OutputFormat == "" {
		cfg.Converter.OutputFormat = "openclaw"
	}
	if cfg.Converter.Templates.Message == "" {
		cfg.Converter.Templates.Message = "%s"
	}
	if cfg.Converter.Templates.StreamChunk == "" {
		cfg.Converter.Templates.StreamChunk = "%s"
	}

	// 监控默认值
	if cfg.Monitor.Prometheus.Enabled && cfg.Monitor.Prometheus.ListenAddress == "" {
		cfg.Monitor.Prometheus.ListenAddress = ":9090"
	}
	if cfg.Monitor.Prometheus.MetricsPath == "" {
		cfg.Monitor.Prometheus.MetricsPath = "/metrics"
	}

	// 日志默认值
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}

	// 热重载默认值
	if cfg.HotReload.Interval == 0 {
		cfg.HotReload.Interval = 1 * time.Second
	}
	if cfg.HotReload.Debounce == 0 {
		cfg.HotReload.Debounce = 500 * time.Millisecond
	}
}

// Validate 验证配置
func Validate(cfg *Config) error {
	// 验证服务器配置
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	// 验证认证数据库配置
	if cfg.Auth.Enabled && cfg.Auth.StoreType == "postgres" {
		if cfg.Auth.Database.Host == "" {
			return fmt.Errorf("auth.database.host is required when using postgres")
		}
		if cfg.Auth.Database.User == "" {
			return fmt.Errorf("auth.database.user is required when using postgres")
		}
		if cfg.Auth.Database.Database == "" {
			return fmt.Errorf("auth.database.name is required when using postgres")
		}
	}

	return nil
}

// Save 保存配置到文件
func Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
