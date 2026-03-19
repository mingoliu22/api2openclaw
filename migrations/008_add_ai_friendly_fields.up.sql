-- api2openclaw AI 友好型改造字段迁移
-- v0.3.0 新增字段

-- 为 request_logs 表添加 AI 友好型字段
ALTER TABLE request_logs
ADD COLUMN IF NOT EXISTS trace_id VARCHAR(36),

-- 是否为流式请求
ADD COLUMN IF NOT EXISTS is_streaming BOOLEAN DEFAULT false,

-- 本次请求中工具调用次数，0 表示无工具调用
ADD COLUMN IF NOT EXISTS tool_calls_count INT DEFAULT 0,

-- 网关自身处理延迟（不含模型推理时间）
ADD COLUMN IF NOT EXISTS middleware_latency_ms INT DEFAULT 0,

-- 是否触发了 JSON 强制输出逻辑
ADD COLUMN IF NOT EXISTS json_enforced BOOLEAN DEFAULT false;

-- 添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_request_logs_trace_id ON request_logs(trace_id);
CREATE INDEX IF NOT EXISTS idx_request_logs_is_streaming ON request_logs(is_streaming);
CREATE INDEX IF NOT EXISTS idx_request_logs_json_enforced ON request_logs(json_enforced);

-- 添加注释
COMMENT ON COLUMN request_logs.trace_id IS 'UUID，请求唯一追踪标识，写入日志并通过响应头返回';
COMMENT ON COLUMN request_logs.is_streaming IS '是否为流式请求';
COMMENT ON COLUMN request_logs.tool_calls_count IS '本次请求中工具调用次数，0 表示无工具调用';
COMMENT ON COLUMN request_logs.middleware_latency_ms IS '网关自身处理延迟（不含模型推理时间）';
COMMENT ON COLUMN request_logs.json_enforced IS '是否触发了 JSON 强制输出逻辑';
