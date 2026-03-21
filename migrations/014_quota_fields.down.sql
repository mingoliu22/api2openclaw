-- 回滚双层配额字段

-- 删除索引
DROP INDEX IF EXISTS idx_api_keys_soft_limit;
DROP INDEX IF EXISTS idx_api_keys_hard_limit;
DROP INDEX IF EXISTS idx_api_keys_priority;

-- 删除列（PostgreSQL 不支持 DROP COLUMN IF EXISTS）
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'api_keys' AND column_name = 'daily_token_soft_limit'
    ) THEN
        ALTER TABLE api_keys DROP COLUMN daily_token_soft_limit;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'api_keys' AND column_name = 'daily_token_hard_limit'
    ) THEN
        ALTER TABLE api_keys DROP COLUMN daily_token_hard_limit;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'api_keys' AND column_name = 'priority'
    ) THEN
        ALTER TABLE api_keys DROP COLUMN priority;
    END IF;
END $$;
