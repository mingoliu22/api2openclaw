-- api2openclaw 数据库初始化脚本
-- PostgreSQL 14+

-- 租户表
CREATE TABLE IF NOT EXISTS tenants (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    tier VARCHAR(32) NOT NULL DEFAULT 'free',
    requests_per_day BIGINT NOT NULL DEFAULT 1000,
    tokens_per_month BIGINT NOT NULL DEFAULT 1000000,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tenants_tier ON tenants(tier);

-- API Key 表
CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    permissions JSONB NOT NULL DEFAULT '["*"]',
    requests_per_minute INT NOT NULL DEFAULT 100,
    requests_per_hour INT NOT NULL DEFAULT 1000,
    requests_per_day INT NOT NULL DEFAULT 10000,
    allowed_models JSONB NOT NULL DEFAULT '["*"]',
    expires_at TIMESTAMP,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_status ON api_keys(status);
CREATE INDEX idx_api_keys_last_used ON api_keys(last_used_at);

-- 模型后端表
CREATE TABLE IF NOT EXISTS backends (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(32) NOT NULL,
    base_url VARCHAR(512) NOT NULL,
    api_key VARCHAR(255),
    headers JSONB NOT NULL DEFAULT '{}',
    health_check_enabled BOOLEAN NOT NULL DEFAULT true,
    health_check_interval INT NOT NULL DEFAULT 30,
    health_check_endpoint VARCHAR(255) NOT NULL DEFAULT '/models',
    health_check_timeout INT NOT NULL DEFAULT 5,
    status VARCHAR(32) NOT NULL DEFAULT 'healthy',
    last_check_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_backends_status ON backends(status);

-- 模型映射表
CREATE TABLE IF NOT EXISTS models (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    backend_group JSONB NOT NULL DEFAULT '[]',
    routing_strategy VARCHAR(32) NOT NULL DEFAULT 'direct',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_models_name ON models(name);

-- 使用量指标表
CREATE TABLE IF NOT EXISTS metrics (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    api_key_id VARCHAR(64) REFERENCES api_keys(id) ON DELETE SET NULL,
    tenant_id VARCHAR(64) REFERENCES tenants(id) ON DELETE SET NULL,
    model VARCHAR(255) NOT NULL,
    request_id VARCHAR(255),
    status_code INT NOT NULL,
    latency_ms BIGINT NOT NULL,
    prompt_tokens INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,
    error TEXT
);

CREATE INDEX idx_metrics_timestamp ON metrics(timestamp);
CREATE INDEX idx_metrics_api_key ON metrics(api_key_id);
CREATE INDEX idx_metrics_tenant ON metrics(tenant_id);
CREATE INDEX idx_metrics_model ON metrics(model);
CREATE INDEX idx_metrics_timestamp_tenant ON metrics(timestamp, tenant_id);

-- 审计日志表
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    level VARCHAR(16) NOT NULL DEFAULT 'info',
    api_key_id VARCHAR(64) REFERENCES api_keys(id) ON DELETE SET NULL,
    tenant_id VARCHAR(64) REFERENCES tenants(id) ON DELETE SET NULL,
    request_id VARCHAR(255),
    model VARCHAR(255),
    method VARCHAR(16),
    path VARCHAR(512),
    status_code INT,
    latency_ms BIGINT,
    tokens_used INT,
    error TEXT,
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX idx_audit_logs_api_key ON audit_logs(api_key_id);
CREATE INDEX idx_audit_logs_tenant ON audit_logs(tenant_id);

-- 限流计数表（内存模式使用 Redis 时此表可不用）
CREATE TABLE IF NOT EXISTS rate_limits (
    key VARCHAR(512) PRIMARY KEY,
    count BIGINT NOT NULL DEFAULT 0,
    window_start TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_rate_limits_expires ON rate_limits(expires_at);

-- 更新时间触发器
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_backends_updated_at BEFORE UPDATE ON backends
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_models_updated_at BEFORE UPDATE ON models
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 插入默认租户
INSERT INTO tenants (id, name, tier, requests_per_day, tokens_per_month)
VALUES ('default', 'Default Tenant', 'free', 1000, 1000000)
ON CONFLICT (id) DO NOTHING;

-- 插入默认 API Key (密钥: sk-dev-default-key-12345)
-- 实际使用时应该通过管理接口创建，这里的 hash 仅用于演示
INSERT INTO api_keys (id, tenant_id, key_hash, status)
VALUES ('sk-dev-default-key-12345', 'default',
        '$2a$10$abcdefghijklmnopqrstuv', 'active')
ON CONFLICT (id) DO NOTHING;
