-- 回滚 AI 友好型改造字段

-- 删除索引
DROP INDEX IF EXISTS idx_request_logs_trace_id;
DROP INDEX IF EXISTS idx_request_logs_is_streaming;
DROP INDEX IF EXISTS idx_request_logs_json_enforced;

-- 删除列（PostgreSQL 不支持直接 DROP COLUMN IF NOT EXISTS，使用子查询）
DO $$
BEGIN
    -- 检查并删除 trace_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'request_logs' AND column_name = 'trace_id'
    ) THEN
        ALTER TABLE request_logs DROP COLUMN trace_id;
    END IF;

    -- 检查并删除 is_streaming
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'request_logs' AND column_name = 'is_streaming'
    ) THEN
        ALTER TABLE request_logs DROP COLUMN is_streaming;
    END IF;

    -- 检查并删除 tool_calls_count
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'request_logs' AND column_name = 'tool_calls_count'
    ) THEN
        ALTER TABLE request_logs DROP COLUMN tool_calls_count;
    END IF;

    -- 检查并删除 middleware_latency_ms
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'request_logs' AND column_name = 'middleware_latency_ms'
    ) THEN
        ALTER TABLE request_logs DROP COLUMN middleware_latency_ms;
    END IF;

    -- 检查并删除 json_enforced
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'request_logs' AND column_name = 'json_enforced'
    ) THEN
        ALTER TABLE request_logs DROP COLUMN json_enforced;
    END IF;
END $$;
