-- api2openclaw Token 工厂 - 配额使用统计
-- v0.3.1 新增：quota_usage_daily 表

-- 创建配额使用统计表
CREATE TABLE IF NOT EXISTS quota_usage_daily (
    key_id VARCHAR(36) NOT NULL,
    usage_date DATE NOT NULL,

    -- 统计字段
    tokens_used BIGINT NOT NULL DEFAULT 0,
    requests_count INT NOT NULL DEFAULT 0,

    -- 阈值跟踪
    soft_exceeded_at TIMESTAMP,
    hard_exceeded_count INT NOT NULL DEFAULT 0,

    -- 时间戳
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- 主键
    PRIMARY KEY (key_id, usage_date)
);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_quota_usage_date ON quota_usage_daily(usage_date);
CREATE INDEX IF NOT EXISTS idx_quota_usage_soft_exceeded ON quota_usage_daily(soft_exceeded_at)
    WHERE soft_exceeded_at IS NOT NULL;

-- 添加注释
COMMENT ON COLUMN quota_usage_daily.key_id IS '关联 api_keys.id';
COMMENT ON COLUMN quota_usage_daily.usage_date IS '统计日期（时区 UTC+8）';
COMMENT ON COLUMN quota_usage_daily.tokens_used IS '该 Key 当日已消耗 token 总数（原子递增）';
COMMENT ON COLUMN quota_usage_daily.requests_count IS '当日请求次数';
COMMENT ON COLUMN quota_usage_daily.soft_exceeded_at IS '首次超软上限的时间戳';
COMMENT ON COLUMN quota_usage_daily.hard_exceeded_count IS '当日触发硬上限的次数';

-- 创建每日记录创建/重置函数
-- 每日 00:00 UTC+8 执行一次，为所有活跃 Key 创建新记录
CREATE OR REPLACE FUNCTION create_daily_quota_records()
RETURNS void AS $$
BEGIN
    -- 为所有活跃 Key 创建今日记录
    INSERT INTO quota_usage_daily (key_id, usage_date, tokens_used, requests_count)
    SELECT
        id,
        CURRENT_DATE,
        0,
        0
    FROM api_keys
    WHERE status = 'active'
    ON CONFLICT (key_id, usage_date) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

-- 创建原子递增函数（用于每次请求完成后更新配额）
CREATE OR REPLACE FUNCTION increment_quota_usage(
    p_key_id VARCHAR(36),
    p_tokens BIGINT,
    p_soft_limit BIGINT,
    p_hard_limit BIGINT
) RETURNS JSONB AS $$
DECLARE
    v_now TIMESTAMP := NOW();
    v_soft_exceeded_at TIMESTAMP;
BEGIN
    -- 更新 tokens_used 和 requests_count
    UPDATE quota_usage_daily
    SET
        tokens_used = tokens_used + p_tokens,
        requests_count = requests_count + 1,
        updated_at = v_now,
        -- 首次超软上限时记录时间
        soft_exceeded_at = CASE
            WHEN soft_exceeded_at IS NULL
                AND (tokens_used + p_tokens) >= p_soft_limit
                AND p_soft_limit > 0
            THEN v_now
            ELSE soft_exceeded_at
        END,
        -- 硬上限触发计数
        hard_exceeded_count = hard_exceeded_count +
            CASE
                WHEN (tokens_used + p_tokens) >= p_hard_limit
                AND p_hard_limit > 0
                THEN 1
                ELSE 0
            END
    WHERE key_id = p_key_id AND usage_date = CURRENT_DATE
    RETURNING
        tokens_used,
        requests_count,
        soft_exceeded_at,
        hard_exceeded_count;
END;
$$ LANGUAGE plpgsql;

-- 注：该函数在请求完成后由中间件调用
-- 使用示例（在 Go 代码中）：
-- result, _ := db.Exec(`SELECT increment_quota_usage($1, $2, $3, $4)`, keyID, tokens, softLimit, hardLimit)

COMMENT ON FUNCTION increment_quota_usage IS
'原子递增配额使用量，首次超软上限时记录时间戳，超硬上限时计数。返回更新后的记录。';
