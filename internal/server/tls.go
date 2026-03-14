package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/openclaw/api2openclaw/internal/config"
)

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

// HTTPSServer HTTPS 服务器
type HTTPSServer struct {
	server     *Server
	tlsConfig  *TLSConfig
	certCache  *sync.Map
	autoCertMgr *AutoCertManager
}

// NewHTTPSServer 创建 HTTPS 服务器
func NewHTTPSServer(cfg *config.Config, configPath string) (*HTTPSServer, error) {
	// 创建基础服务器
	srv, err := New(cfg, configPath)
	if err != nil {
		return nil, err
	}

	httpsServer := &HTTPSServer{
		server:    srv,
		certCache: &sync.Map{},
	}

	// 解析 TLS 配置
	if cfg.TLS.Enabled {
		httpsServer.tlsConfig = &TLSConfig{
			Enabled:        cfg.TLS.Enabled,
			CertFile:       cfg.TLS.CertFile,
			KeyFile:        cfg.TLS.KeyFile,
			AutoCert:       cfg.TLS.AutoCert,
			CertDir:        cfg.TLS.CertDir,
			HostWhitelist:  cfg.TLS.HostWhitelist,
			MinVersion:     cfg.TLS.MinVersion,
			CipherSuites:    cfg.TLS.CipherSuites,
			ClientAuth:      cfg.TLS.ClientAuth,
			ClientCAFile:    cfg.TLS.ClientCAFile,
		}
	}

	// 如果启用了自动证书，启动自动证书管理器
	if httpsServer.tlsConfig != nil && httpsServer.tlsConfig.AutoCert {
		mgr, err := NewAutoCertManager(httpsServer.tlsConfig.CertDir, httpsServer.tlsConfig.HostWhitelist)
		if err != nil {
			return nil, fmt.Errorf("create auto cert manager: %w", err)
		}
		httpsServer.autoCertMgr = mgr
	}

	return httpsServer, nil
}

// Start 启动 HTTPS 服务器
func (s *HTTPSServer) Start() error {
	if s.tlsConfig == nil || !s.tlsConfig.Enabled {
		log.Println("[HTTPS] TLS is disabled, starting HTTP server...")
		return s.server.Start()
	}

	var tlsConfig *tls.Config

	if s.tlsConfig.AutoCert && s.autoCertMgr != nil {
		// 使用自动证书
		cert := s.autoCertMgr.GetCertificate()
		tlsConfig = &tls.Config{
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return cert, nil
			},
			MinVersion: parseTLSVersion(s.tlsConfig.MinVersion),
		}
	} else {
		// 加载证书文件
		cert, err := tls.LoadX509KeyPair(s.tlsConfig.CertFile, s.tlsConfig.KeyFile)
		if err != nil {
			return fmt.Errorf("load certificate: %w", err)
		}

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion: parseTLSVersion(s.tlsConfig.MinVersion),
		}
	}

	// 配置密码套件
	if len(s.tlsConfig.CipherSuites) > 0 {
		tlsConfig.CipherSuites = parseCipherSuites(s.tlsConfig.CipherSuites)
	}

	// 配置客户端认证
	if s.tlsConfig.ClientAuth {
		caCertPool, err := createCertPool(s.tlsConfig.ClientCAFile)
		if err != nil {
			return fmt.Errorf("create cert pool: %w", err)
		}

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	// 更新服务器配置
	s.server.httpSrv.TLSConfig = tlsConfig

	// 添加 HTTP 到 HTTPS 重定向
	if err := s.setupHTTPRedirect(); err != nil {
		return fmt.Errorf("setup HTTP redirect: %w", err)
	}

	log.Printf("[HTTPS] Starting HTTPS server on %s:%d",
		s.server.config.Server.Host, s.server.config.Server.Port)

	return s.server.Start()
}

// setupHTTPRedirect 设置 HTTP 到 HTTPS 重定向
func (s *HTTPSServer) setupHTTPRedirect() error {
	// 启动 HTTP 服务器处理重定向
	httpPort := 80 // HTTP 默认端口
	if httpPort == s.server.config.Server.Port {
		httpPort = 8080
	}

	go func() {
		httpServer := &http.Server{
			Addr: fmt.Sprintf("%s:%d", s.server.config.Server.Host, httpPort),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 重定向到 HTTPS
				targetURL := fmt.Sprintf("https://%s%s",
					r.Host,
					r.URL.RequestURI())
				http.Redirect(w, r, targetURL, http.StatusMovedPermanently)
			}),
		}

		log.Printf("[HTTPS] HTTP redirect server listening on :%d", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[HTTPS] HTTP redirect server error: %v", err)
		}
	}()

	return nil
}

// GetTLSConfig 获取 TLS 配置
func (s *HTTPSServer) GetTLSConfig() *TLSConfig {
	return s.tlsConfig
}

// AutoCertManager 自动证书管理器（Let's Encrypt）
type AutoCertManager struct {
	certDir   string
	hosts     []string
	certCache *sync.Map
}

// NewAutoCertManager 创建自动证书管理器
func NewAutoCertManager(certDir string, hosts []string) (*AutoCertManager, error) {
	mgr := &AutoCertManager{
		certDir:   certDir,
		hosts:     hosts,
		certCache: &sync.Map{},
	}

	// TODO: 实现真实的 Let's Encrypt 集成
	// 这里提供一个简化实现

	return mgr, nil
}

// GetCertificate 获取证书
func (m *AutoCertManager) GetCertificate() *tls.Certificate {
	// 生成自签名证书（用于开发）
	cert, err := m.generateSelfSignedCert()
	if err != nil {
		log.Printf("[AutoCert] Failed to generate cert: %v", err)
		return nil
	}

	return cert
}

// generateSelfSignedCert 生成自签名证书
func (m *AutoCertManager) generateSelfSignedCert() (*tls.Certificate, error) {
	// TODO: 实现自签名证书生成
	// 这里需要 crypto/x509 包
	return nil, fmt.Errorf("self-signed cert not implemented")
}

// parseTLSVersion 解析 TLS 版本
func parseTLSVersion(version string) uint16 {
	switch version {
	case "1.0", "TLSv1.0":
		return tls.VersionTLS10
	case "1.1", "TLSv1.1":
		return tls.VersionTLS11
	case "1.2", "TLSv1.2":
		return tls.VersionTLS12
	case "1.3", "TLSv1.3":
		return tls.VersionTLS13
	default:
		return tls.VersionTLS12
	}
}

// parseCipherSuites 解析密码套件
func parseCipherSuites(suites []string) []uint16 {
	cipherMap := map[string]uint16{
		"TLS_RSA_WITH_AES_128_CBC_SHA":                     tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":                     tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":                   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":             tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":            tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_AES_128_GCM_SHA256":                          tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":                          tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256":                     tls.TLS_CHACHA20_POLY1305_SHA256,
	}

	result := make([]uint16, 0, len(suites))
	for _, suite := range suites {
		if id, ok := cipherMap[suite]; ok {
			result = append(result, id)
		}
	}

	return result
}

// createCertPool 创建证书池
func createCertPool(caFile string) (*x509.CertPool, error) {
	caCertPool := x509.NewCertPool()
	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}
	return caCertPool, nil
}

// RedirectMiddleware HTTPS 重定向中间件
func RedirectMiddleware(httpsPort int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果已经是 HTTPS，跳过
		if c.Request.TLS != nil {
			c.Next()
			return
		}

		// 检查是否是本地请求
		host := c.Request.Host
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			c.Next()
			return
		}

		// 重定向到 HTTPS
		url := fmt.Sprintf("https://%s%s", host, c.Request.URL.RequestURI())
		c.Redirect(http.StatusMovedPermanently, url)
	}
}

// HSTSConfig HSTS 配置
type HSTSConfig struct {
	Enabled         bool          `yaml:"enabled"`
	MaxAge          int           `yaml:"max_age"`
	IncludeSubDomains bool         `yaml:"include_sub_domains"`
	Preload         bool          `yaml:"preload"`
}

// HSTSMiddleware HSTS 中间件
func HSTSMiddleware(config *HSTSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Enabled {
			maxAge := config.MaxAge
			if maxAge == 0 {
				maxAge = 31536000 // 1 年
			}

			header := fmt.Sprintf("max-age=%d", maxAge)
			if config.IncludeSubDomains {
				header += "; includeSubDomains"
			}
			if config.Preload {
				header += "; preload"
			}

			c.Header("Strict-Transport-Security", header)
		}
		c.Next()
	}
}
