-- api2openclaw 模型能力字段迁移
-- v0.3.0 新增模型能力声明字段

-- 为 models 表添加能力字段
ALTER TABLE models
ADD COLUMN IF NOT EXISTS supports_streaming BOOLEAN NOT NULL DEFAULT true,

-- 是否支持 Tool Use / 函数调用
ADD COLUMN IF NOT EXISTS supports_tool_use BOOLEAN NOT NULL DEFAULT false,

-- 是否原生支持 response_format json
ADD COLUMN IF NOT EXISTS supports_json_mode BOOLEAN NOT NULL DEFAULT false,

-- 上下文窗口大小（tokens）
ADD COLUMN IF NOT EXISTS context_window INT NOT NULL DEFAULT 4096,

-- 模型家族：qwen / deepseek / llama / other
ADD COLUMN IF NOT EXISTS model_family VARCHAR(32) NOT NULL DEFAULT 'other';

-- 添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_models_family ON models(model_family);
CREATE INDEX IF NOT EXISTS idx_models_streaming ON models(supports_streaming) WHERE supports_streaming = true;
CREATE INDEX IF NOT EXISTS idx_models_tool_use ON models(supports_tool_use) WHERE supports_tool_use = true;
CREATE INDEX IF NOT EXISTS idx_models_json_mode ON models(supports_json_mode) WHERE supports_json_mode = true;

-- 添加注释
COMMENT ON COLUMN models.supports_streaming IS '该模型是否支持 SSE 流式输出';
COMMENT ON COLUMN models.supports_tool_use IS '是否支持 Tool Use / 函数调用';
COMMENT ON COLUMN models.supports_json_mode IS '是否原生支持 response_format json';
COMMENT ON COLUMN models.context_window IS '上下文窗口大小（tokens）';
COMMENT ON COLUMN models.model_family IS '模型家族：qwen / deepseek / llama / other';

-- 更新现有模型的能力值（示例）
-- 假设默认模型是 Qwen，支持流式和 tool_use
UPDATE models
SET
    supports_streaming = true,
    supports_tool_use = false,  -- Tool Use 暂未实现
    supports_json_mode = false,
    context_window = 32768,
    model_family = 'qwen'
WHERE model_id LIKE '%qwen%';

UPDATE models
SET
    supports_streaming = true,
    supports_tool_use = false,
    supports_json_mode = false,
    context_window = 32768,
    model_family = 'deepseek'
WHERE model_id LIKE '%deepseek%';
