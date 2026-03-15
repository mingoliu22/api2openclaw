-- api2openclaw 模型配置表
-- PostgreSQL 14+

-- 模型配置表
CREATE TABLE IF NOT EXISTS models (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid(),
    alias VARCHAR(64) NOT NULL UNIQUE,
    model_id VARCHAR(128) NOT NULL,
    base_url VARCHAR(256) NOT NULL,
    api_key_encrypted TEXT,
    note VARCHAR(200),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_models_alias ON models(alias);
CREATE INDEX idx_models_active ON models(is_active) WHERE is_active = true;

-- 模型健康状态表
CREATE TABLE IF NOT EXISTS model_health (
    id BIGSERIAL PRIMARY KEY,
    model_id VARCHAR(36) NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    status VARCHAR(16) NOT NULL, -- 'healthy', 'unhealthy', 'unknown'
    latency_ms INT,
    error_message TEXT,
    checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_model_health_model_checked ON model_health(model_id, checked_at DESC);

-- 模型配置变更事件表（用于热重载）
CREATE TABLE IF NOT EXISTS model_config_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(16) NOT NULL, -- 'created', 'updated', 'deleted', 'toggled'
    model_id VARCHAR(36),
    old_value JSONB,
    new_value JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_model_config_events_created ON model_config_events(created_at DESC);

-- 插入默认模型配置（示例）
INSERT INTO models (alias, model_id, base_url, api_key_encrypted, note)
VALUES ('gpt-4', 'qwen2.5-72b-instruct', 'http://localhost:11434/v1', NULL, '默认主力模型')
ON CONFLICT (alias) DO NOTHING;

-- 更新时间触发器
CREATE OR REPLACE FUNCTION update_models_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_models_updated_at
    BEFORE UPDATE ON models
    FOR EACH ROW
    EXECUTE FUNCTION update_models_updated_at();

-- 清理旧健康检查记录（7 天）
DELETE FROM model_health WHERE checked_at < CURRENT_TIMESTAMP - INTERVAL '7 days';

-- 清理旧配置事件（30 天）
DELETE FROM model_config_events WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '30 days';
