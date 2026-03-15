-- api2openclaw API Key 管理表
-- PostgreSQL 14+

-- API Key 表
CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid(),
    label VARCHAR(64) NOT NULL,
    key_hash VARCHAR(128) NOT NULL UNIQUE,
    key_prefix VARCHAR(20) NOT NULL,
    model_alias VARCHAR(64),
    expires_at TIMESTAMP NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active', -- 'active', 'revoked', 'expired'
    note VARCHAR(200),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP NULL
);

CREATE INDEX idx_api_keys_status ON api_keys(status);
CREATE INDEX idx_api_keys_model_alias ON api_keys(model_alias);
CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at);

-- API Key 使用统计表
CREATE TABLE IF NOT EXISTS api_key_usage (
    id BIGSERIAL PRIMARY KEY,
    key_id VARCHAR(36) NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    model_alias VARCHAR(64) NOT NULL,
    request_count INT NOT NULL DEFAULT 0,
    prompt_tokens BIGINT NOT NULL DEFAULT 0,
    completion_tokens BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_api_key_usage_unique ON api_key_usage(key_id, date, model_alias);
CREATE INDEX idx_api_key_usage_date ON api_key_usage(date DESC);

-- 更新时间触发器
CREATE OR REPLACE FUNCTION update_api_keys_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_api_keys_updated_at
    BEFORE UPDATE ON api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_api_keys_updated_at();

-- 清理过期使用统计（保留 1 年）
DELETE FROM api_key_usage WHERE date < CURRENT_DATE - INTERVAL '1 year';
