-- api2openclaw Token 工厂 - 双层配额体系
-- v0.3.1 新增：软上限/硬上限 + 优先级

-- api_keys 表新增字段
ALTER TABLE api_keys
ADD COLUMN IF NOT EXISTS daily_token_soft_limit BIGINT NOT NULL DEFAULT 0,

-- 每日 token 软上限，0 表示不设置软上限预警
ADD COLUMN IF NOT EXISTS daily_token_hard_limit BIGINT NOT NULL DEFAULT 0,

-- 每日 token 硬上限，0 表示不限制
ADD COLUMN IF NOT EXISTS priority VARCHAR(16) NOT NULL DEFAULT 'normal'
    CHECK (priority IN ('high', 'normal', 'low'));

-- 添加注释
COMMENT ON COLUMN api_keys.daily_token_soft_limit IS '每日 token 软上限（预警线），0 表示不设置；单位：token';
COMMENT ON COLUMN api_keys.daily_token_hard_limit IS '每日 token 硬上限（强制限制），0 表示不限制；单位：token；必须 ≥ 软上限';
COMMENT ON COLUMN api_keys.priority IS 'API Key 优先级：high（高优先级）/normal（普通）/low（低优先级）';

-- 添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_api_keys_soft_limit ON api_keys(daily_token_soft_limit)
    WHERE daily_token_soft_limit > 0;

CREATE INDEX IF NOT EXISTS idx_api_keys_hard_limit ON api_keys(daily_token_hard_limit)
    WHERE daily_token_hard_limit > 0;

CREATE INDEX IF NOT EXISTS idx_api_keys_priority ON api_keys(priority);
