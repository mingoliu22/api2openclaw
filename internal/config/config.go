package config

import (
	"time"
)

// Config 是主配置结构
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Auth      AuthConfig      `yaml:"auth"`
	Router    RouterConfig    `yaml:"router"`
	Converter ConverterConfig `yaml:"converter"`
	Monitor   MonitorConfig   `yaml:"monitor"`
	Logging   LoggingConfig   `yaml:"logging"`
	HotReload HotReloadConfig `yaml:"hot_reload"`
	TLS       TLSConfig       `yaml:"tls"`
	HSTS      HSTSConfig      `yaml:"hsts"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	BasePath        string        `yaml:"base_path"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	Enabled      bool           `yaml:"enabled"`
	StoreType    string         `yaml:"store_type"`
	Database     DatabaseConfig `yaml:"database"`
	DefaultQuota Quota         `yaml:"default_quota"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"sslmode"`
}

// Quota 配额配置
type Quota struct {
	RequestsPerDay  int64 `yaml:"requests_per_day"`
	TokensPerMonth  int64 `yaml:"tokens_per_month"`
}

// RouterConfig 路由器配置
type RouterConfig struct {
	Backends []BackendConfig      `yaml:"backends"`
	Models   []ModelConfig        `yaml:"models"`
	Aliases  []ModelAliasConfig   `yaml:"aliases"`
}

// ModelAliasConfig 模型别名配置
type ModelAliasConfig struct {
	Alias      string   `yaml:"alias"`      // 别名，如 "gpt-4"
	Target     string   `yaml:"target"`     // 目标模型，如 "Qwen-72B"
	Backends   []string `yaml:"backends,omitempty"` // 可选：直接指定后端组
	Strategy   string   `yaml:"strategy,omitempty"`  // 可选：覆盖路由策略
}

// BackendConfig 后端实例配置
type BackendConfig struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Type        string            `yaml:"type"`
	BaseURL     string            `yaml:"base_url"`
	APIKey      string            `yaml:"api_key"`
	Headers     map[string]string `yaml:"headers"`
	Weight      int               `yaml:"weight"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
	Endpoint string        `yaml:"endpoint"`
	Timeout  time.Duration `yaml:"timeout"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	Name            string   `yaml:"name"`
	BackendGroup    []string `yaml:"backend_group"`
	RoutingStrategy string   `yaml:"routing_strategy"`
}

// ConverterConfig 格式转换配置
type ConverterConfig struct {
	InputFormat  string            `yaml:"input_format"`
	OutputFormat string            `yaml:"output_format"`
	Templates    TemplatesConfig   `yaml:"templates"`
}

// TemplatesConfig 模板配置
type TemplatesConfig struct {
	Message     string `yaml:"message"`
	StreamChunk string `yaml:"stream_chunk"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Enabled          bool              `yaml:"enabled"`
	Metrics          MetricsConfig     `yaml:"metrics"`
	RateLimiting     RateLimitConfig   `yaml:"rate_limiting"`
	CircuitBreaker   CircuitConfig     `yaml:"circuit_breaker"`
	Prometheus       PrometheusConfig  `yaml:"prometheus"`
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	Enabled       bool `yaml:"enabled"`
	RetentionDays int  `yaml:"retention_days"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Storage string `yaml:"storage"`
}

// CircuitConfig 熔断器配置
type CircuitConfig struct {
	Enabled             bool          `yaml:"enabled"`
	ErrorRateThreshold  float64       `yaml:"error_rate_threshold"`
	ConsecutiveErrors   int           `yaml:"consecutive_errors"`
	RecoveryTimeout     time.Duration `yaml:"recovery_timeout"`
	HalfOpenMaxAttempts int           `yaml:"half_open_max_attempts"`
}

// PrometheusConfig Prometheus 配置
type PrometheusConfig struct {
	Enabled        bool   `yaml:"enabled"`
	ListenAddress  string `yaml:"listen_address"`
	MetricsPath    string `yaml:"metrics_path"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// HotReloadConfig 热重载配置
type HotReloadConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
	Debounce time.Duration `yaml:"debounce"`
}

// TLSConfig TLS 配置
type TLSConfig struct {
	Enabled        bool     `yaml:"enabled"`
	CertFile       string   `yaml:"cert_file"`
	KeyFile        string   `yaml:"key_file"`
	AutoCert       bool     `yaml:"auto_cert"`
	CertDir        string   `yaml:"cert_dir"`
	HostWhitelist  []string `yaml:"host_whitelist"`
	MinVersion     string   `yaml:"min_version"`
	CipherSuites    []string `yaml:"cipher_suites"`
	ClientAuth      bool     `yaml:"client_auth"`
	ClientCAFile    string   `yaml:"client_ca_file"`
}

// HSTSConfig HSTS 配置
type HSTSConfig struct {
	Enabled          bool `yaml:"enabled"`
	MaxAge           int  `yaml:"max_age"`
	IncludeSubDomains bool `yaml:"include_sub_domains"`
	Preload          bool `yaml:"preload"`
}
