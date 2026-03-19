-- api2openclaw 部署指南配置表
-- v0.3.1 新增：支持管理控制台动态配置部署指南内容

-- 部署指南框架/模型配置表
CREATE TABLE IF NOT EXISTS deploy_guides (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 框架标识
    framework_id VARCHAR(32) NOT NULL,        -- vllm / sglang / xinference / ollama
    model_id VARCHAR(128) NOT NULL,           -- 模型唯一标识

    -- 显示信息
    name VARCHAR(256) NOT NULL,               -- 模型显示名称
    alias VARCHAR(64),                         -- 接入网关时建议使用的别名

    -- 安装与启动
    install_cmd TEXT,                          -- 安装命令（框架级别，相同框架共享）
    start_cmd TEXT,                            -- 启动命令

    -- JSON 配置字段
    params JSONB,                              -- 启动参数列表 [{name, default, desc}]
    features JSONB,                            -- 能力标签列表 ["PagedAttention", "连续批处理"]
    requirements JSONB,                        -- 环境要求 {os, python, cuda, gpu}

    -- 框架配置（框架级别）
    api_port INT,                              -- 推理服务默认端口
    tagline VARCHAR(128),                      -- 一句话定位描述
    description TEXT,                          -- 详细描述
    badge VARCHAR(64),                         -- 定位标签：生产首选/Agent场景/多模型管理/最易上手
    badge_color VARCHAR(16),                   -- 标签颜色：blue/purple/green/orange
    accent_color VARCHAR(16),                  -- 主题色：#3B82F6/#8B5CF6/#10B981/#F97316
    icon VARCHAR(8),                           -- 框架图标（emoji）：⚡/🧠/🎛/🦙

    -- 模型配置（模型级别）
    model_family VARCHAR(32),                  -- 模型家族：Qwen/DeepSeek/Llama
    vram_requirement VARCHAR(32),              -- 最低显存要求：≥48GB
    precision VARCHAR(16),                     -- 精度：BF16/FP8/Q4_K_M
    hf_id VARCHAR(256),                        -- HuggingFace 模型 ID

    -- 部署步骤
    steps JSONB,                               -- 部署步骤 [{title, code, note}]

    -- 管理字段
    display_order INT DEFAULT 0,               -- 显示顺序（同框架内）
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_deploy_guides_framework ON deploy_guides(framework_id);
CREATE INDEX idx_deploy_guides_active ON deploy_guides(is_active) WHERE is_active = true;
CREATE INDEX idx_deploy_guides_order ON deploy_guides(framework_id, display_order);

-- 唯一约束：同一框架下模型 ID 唯一
CREATE UNIQUE INDEX idx_deploy_guides_framework_model ON deploy_guides(framework_id, model_id);

-- 更新时间触发器
CREATE OR REPLACE FUNCTION update_deploy_guides_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_deploy_guides_updated_at
    BEFORE UPDATE ON deploy_guides
    FOR EACH ROW
    EXECUTE FUNCTION update_deploy_guides_updated_at();

-- 插入默认数据（vLLM 框架）
INSERT INTO deploy_guides (framework_id, model_id, name, alias, install_cmd, start_cmd, api_port,
    tagline, description, badge, badge_color, accent_color, icon, features, requirements,
    model_family, vram_requirement, precision, hf_id, display_order)
VALUES
    -- vLLM 框架配置（框架级别数据，model_id 用于区分）
    ('vllm', '_framework', 'vLLM', NULL,
        'pip install vllm',
        NULL,
        8000,
        '生产首选',
        '高吞吐推理引擎，专为多用户并发生产环境优化。PagedAttention 技术减少 50%+ 显存碎片。',
        '生产首选',
        'blue',
        '#3B82F6',
        '⚡',
        '["PagedAttention", "连续批处理", "Speculative Decoding"]'::JSONB,
        '{"os": "Linux", "python": ">=3.8", "cuda": ">=11.8", "gpu": "NVIDIA (A100/H100/RTX 40xx)"}'::JSONB,
        NULL, NULL, NULL, NULL, 0),

    -- Qwen2.5-72B-Instruct
    ('vllm', 'qwen2.5-72b-instruct', 'Qwen2.5-72B-Instruct', 'qwen-72b',
        NULL,
        'vllm serve Qwen/Qwen2.5-72B-Instruct --tensor-parallel-size 4 --host 0.0.0.0 --port 8000',
        8000, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        'Qwen', '≥48GB', 'BF16', 'Qwen/Qwen2.5-72B-Instruct', 1),

    -- DeepSeek-V3
    ('vllm', 'deepseek-v3', 'DeepSeek-V3', 'deepseek-v3',
        NULL,
        'vllm serve deepseek-ai/DeepSeek-V3 --tensor-parallel-size 8 --host 0.0.0.0 --port 8000 --max-model-len 8192',
        8000, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        'DeepSeek', '≥8×80GB', 'BF16', 'deepseek-ai/DeepSeek-V3', 2),

    -- Llama-3.3-70B-Instruct
    ('vllm', 'llama-3.3-70b-instruct', 'Llama-3.3-70B-Instruct', 'llama-70b',
        NULL,
        'vllm serve meta-llama/Llama-3.3-70B-Instruct --tensor-parallel-size 4 --host 0.0.0.0 --port 8000',
        8000, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        'Llama', '≥40GB', 'BF16', 'meta-llama/Llama-3.3-70B-Instruct', 3)
ON CONFLICT (framework_id, model_id) DO NOTHING;

-- 插入 SGLang 框架数据
INSERT INTO deploy_guides (framework_id, model_id, name, alias, install_cmd, start_cmd, api_port,
    tagline, description, badge, badge_color, accent_color, icon, features, requirements,
    model_family, vram_requirement, precision, hf_id, display_order)
VALUES
    -- SGLang 框架
    ('sglang', '_framework', 'SGLang', NULL,
        'pip install --upgrade pip && pip install uv && uv pip install "sglang[all]"',
        NULL,
        30000,
        'Agent 场景',
        '专为多轮对话、RAG、Agent 推理优化。RadixAttention 实现自动 prefix cache 复用，首 token 延迟极低。',
        'Agent场景',
        'purple',
        '#8B5CF6',
        '🧠',
        '["RadixAttention", "Prefix Cache", "Low Latency"]'::JSONB,
        '{"os": "Linux", "python": ">=3.10", "cuda": ">=11.8", "gpu": "NVIDIA (A100/H100)"}'::JSONB,
        NULL, NULL, NULL, NULL, 0),

    -- Qwen3-32B
    ('sglang', 'qwen3-32b', 'Qwen3-32B', 'qwen-32b',
        NULL,
        'python -m sglang.launch_server --model-path Qwen/Qwen2.5-32B-Instruct --host 0.0.0.0 --port 30000',
        30000, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        'Qwen', '≥24GB', 'BF16', 'Qwen/Qwen2.5-32B-Instruct', 1),

    -- DeepSeek-V3 FP8
    ('sglang', 'deepseek-v3-fp8', 'DeepSeek-V3 (FP8)', 'deepseek-fp8',
        NULL,
        'python -m sglang.launch_server --model-path deepseek-ai/DeepSeek-V3 --host 0.0.0.0 --port 30000 --tp 8 --quantization fp8',
        30000, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        'DeepSeek', '≥8×80GB', 'FP8', 'deepseek-ai/DeepSeek-V3', 2),

    -- Llama-3.1-8B-Instruct
    ('sglang', 'llama-3.1-8b-instruct', 'Llama-3.1-8B-Instruct', 'llama-8b',
        NULL,
        'python -m sglang.launch_server --model-path meta-llama/Llama-3.1-8B-Instruct --host 0.0.0.0 --port 30000',
        30000, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        'Llama', '≥16GB', 'BF16', 'meta-llama/Llama-3.1-8B-Instruct', 3)
ON CONFLICT (framework_id, model_id) DO NOTHING;

-- 插入 Xinference 框架数据
INSERT INTO deploy_guides (framework_id, model_id, name, alias, install_cmd, start_cmd, api_port,
    tagline, description, badge, badge_color, accent_color, icon, features, requirements,
    model_family, vram_requirement, precision, hf_id, display_order)
VALUES
    -- Xinference 框架
    ('xinference', '_framework', 'Xinference', NULL,
        'pip install "xinference[all]"',
        NULL,
        9997,
        '多模型管理',
        '统一管理多个模型和推理框架。支持 vLLM/SGLang/llama.cpp/Transformers/MLX 多后端，自带 Web UI。',
        '多模型管理',
        'green',
        '#10B981',
        '🎛',
        '["Web UI", "Multi Backend", "RESTful API"]'::JSONB,
        '{"os": "Linux/Mac/Windows", "python": ">=3.9", "cuda": ">=11.8 (可选)", "gpu": "NVIDIA (可选)"}'::JSONB,
        NULL, NULL, NULL, NULL, 0),

    -- Qwen2.5-Instruct (vLLM 后端)
    ('xinference', 'qwen2.5-instruct-vllm', 'Qwen2.5-Instruct (vLLM)', 'qwen-vllm',
        NULL,
        'xinference launch --model-name qwen2.5-instruct --model-format pytorch --size-in-billions 7 --backend vllm',
        9997, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        'Qwen', '≥8GB (7B)', 'BF16/AWQ', 'Qwen/Qwen2.5-7B-Instruct', 1),

    -- DeepSeek-R1-Distill-14B (SGLang 后端)
    ('xinference', 'deepseek-r1-14b-sglang', 'DeepSeek-R1-Distill-14B (SGLang)', 'deepseek-sglang',
        NULL,
        'xinference launch --model-name deepseek-r1 --model-format pytorch --size-in-billions 14 --backend sglang',
        9997, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        'DeepSeek', '≥16GB', 'BF16', 'deepseek-ai/DeepSeek-R1-Distill-Qwen-14B', 2),

    -- Llama-3.1-8B (llama.cpp 后端)
    ('xinference', 'llama-3.1-8b-gguf', 'Llama-3.1-8B (GGUF)', 'llama-gguf',
        NULL,
        'xinference launch --model-name llama-3.1-8b-instruct --model-format gguf --backend llama.cpp',
        9997, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        'Llama', '≥6GB (Q4)', 'GGUF Q4_K_M', 'meta-llama/Llama-3.1-8B-Instruct-GGUF', 3)
ON CONFLICT (framework_id, model_id) DO NOTHING;

-- 插入 Ollama 框架数据
INSERT INTO deploy_guides (framework_id, model_id, name, alias, install_cmd, start_cmd, api_port,
    tagline, description, badge, badge_color, accent_color, icon, features, requirements,
    model_family, vram_requirement, precision, hf_id, display_order)
VALUES
    -- Ollama 框架
    ('ollama', '_framework', 'Ollama', NULL,
        'curl -fsSL https://ollama.com/install.sh | sh',
        NULL,
        11434,
        '最易上手',
        '零配置、一行命令安装运行。支持 Mac/Linux/Windows，个人开发和快速原型验证首选。',
        '最易上手',
        'orange',
        '#F97316',
        '🦙',
        '["Zero Config", "Cross Platform", "RESTful API"]'::JSONB,
        '{"os": "Linux/Mac/Windows", "python": "无需", "cuda": "内置(可选)", "gpu": "NVIDIA/Apple Silicon/AMD"}'::JSONB,
        NULL, NULL, NULL, NULL, 0),

    -- qwen2.5:72b
    ('ollama', 'qwen2.5-72b', 'qwen2.5:72b', 'qwen-72b',
        NULL,
        'ollama run qwen2.5:72b',
        11434, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        'Qwen', '≥48GB', 'Q4_K_M', NULL, 1),

    -- deepseek-r1:14b
    ('ollama', 'deepseek-r1-14b', 'deepseek-r1:14b', 'deepseek-14b',
        NULL,
        'ollama run deepseek-r1:14b',
        11434, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        'DeepSeek', '≥10GB', 'Q4_K_M', NULL, 2),

    -- llama3.3:70b
    ('ollama', 'llama-3.3-70b', 'llama3.3:70b', 'llama-70b',
        NULL,
        'ollama run llama3.3:70b',
        11434, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL,
        'Llama', '≥40GB', 'Q4_K_M', NULL, 3)
ON CONFLICT (framework_id, model_id) DO NOTHING;

-- 清理旧数据（90 天）
DELETE FROM deploy_guides WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '90 days';
