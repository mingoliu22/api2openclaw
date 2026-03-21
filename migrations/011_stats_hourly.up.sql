-- api2openclaw Token 工厂 - 统计数据
-- v0.3.1 新增：stats_hourly 物化视图
-- 用于高效查询小时级别的 token 产量统计数据

-- 创建系统配置表（如果不存在）
CREATE TABLE IF NOT EXISTS system_configs (
    key VARCHAR(64) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 插入默认配置
INSERT INTO system_configs (key, value) VALUES
    ('stats_threshold', '500'),
    ('quota_webhook_url', ''),
    ('quota_silence_minutes', '10')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- 创建 stats_hourly 物化视图
-- 按 model_alias + 小时聚合 token 产量、请求数、平均延迟
CREATE MATERIALIZED VIEW IF NOT EXISTS stats_hourly AS
SELECT
    -- 统计日期和小时
    DATE(r.created_at) as stat_date,
    EXTRACT(HOUR FROM r.created_at) as stat_hour,

    -- 模型维度
    r.model_alias,

    -- 聚合指标
    SUM(r.completion_tokens) as total_tokens,
    SUM(r.prompt_tokens) as prompt_tokens,
    COUNT(*) as request_count,
    AVG(r.latency_ms) as avg_latency_ms,

    -- 计算每秒 token 生产率（基于该小时总 token / 3600）
    -- 这是一个近似值，实际生产中应使用滑动窗口实时计算
    COALESCE(SUM(r.completion_tokens) / 3600.0, 0) as tokens_per_sec,

    -- 时间戳
    MAX(r.created_at) as last_updated
FROM request_logs r
GROUP BY
    DATE(r.created_at),
    EXTRACT(HOUR FROM r.created_at),
    r.model_alias;

-- 创建索引以提高物化视图刷新性能
CREATE UNIQUE INDEX IF NOT EXISTS idx_stats_hourly_date_hour_model
    ON stats_hourly(stat_date, stat_hour, model_alias);

-- 创建普通索引用于常用查询
CREATE INDEX IF NOT EXISTS idx_stats_hourly_date
    ON stats_hourly(stat_date);

CREATE INDEX IF NOT EXISTS idx_stats_hourly_model
    ON stats_hourly(model_alias);

-- 创建定时刷新函数
-- 注意：物化视图需要手动或通过定时任务刷新
CREATE OR REPLACE FUNCTION refresh_stats_hourly()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY stats_hourly;
END;
$$ LANGUAGE plpgsql;

-- 注：物化视图刷新逻辑
-- 生产环境建议使用 cron 或 pg_cron 每 15 分钟刷新一次：
-- */15 * * * * psql -d api2openclaw -c 'SELECT refresh_stats_hourly()'
-- 或使用 pg_cron:
-- SELECT cron.schedule('refresh-stats-hourly', '*/15 * * * *', 'SELECT refresh_stats_hourly()');

COMMENT ON MATERIALIZED VIEW stats_hourly IS
'按小时聚合的 token 产量统计数据，用于产能仪表盘和报表查询。每 15 分钟刷新一次。';

COMMENT ON COLUMN stats_hourly.stat_date IS '统计日期（自然日）';
COMMENT ON COLUMN stats_hourly.stat_hour IS '统计小时（0-23）';
COMMENT ON COLUMN stats_hourly.model_alias IS '模型别名';
COMMENT ON COLUMN stats_hourly.total_tokens IS '该小时完成的 token 总数';
COMMENT ON COLUMN stats_hourly.prompt_tokens IS '该小时提示的 token 总数';
COMMENT ON COLUMN stats_hourly.request_count IS '该小时的请求数量';
COMMENT ON COLUMN stats_hourly.avg_latency_ms IS '该小时的平均延迟（毫秒）';
COMMENT ON COLUMN stats_hourly.tokens_per_sec IS '近似每秒 token 生产率（total_tokens / 3600）';
