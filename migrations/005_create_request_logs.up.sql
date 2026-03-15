-- api2openclaw 请求日志表
-- PostgreSQL 14+

-- 请求日志表
CREATE TABLE IF NOT EXISTS request_logs (
    id BIGSERIAL PRIMARY KEY,
    key_id VARCHAR(36) REFERENCES api_keys(id) ON DELETE SET NULL,
    model_alias VARCHAR(64) NOT NULL,
    model_actual VARCHAR(128),
    prompt_tokens INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,
    latency_ms INT NOT NULL,
    status_code INT NOT NULL,
    error_code VARCHAR(64),
    error_message TEXT,
    request_id VARCHAR(64),
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_request_logs_created_at ON request_logs(created_at DESC);
CREATE INDEX idx_request_logs_key_id ON request_logs(key_id, created_at DESC);
CREATE INDEX idx_request_logs_model_alias ON request_logs(model_alias, created_at DESC);
CREATE INDEX idx_request_logs_status_code ON request_logs(status_code);

-- 清理旧日志（保留 90 天）
DELETE FROM request_logs WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '90 days';
