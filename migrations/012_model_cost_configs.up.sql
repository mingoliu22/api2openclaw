-- api2openclaw Token 工厂 - 成本核算引擎
-- v0.3.1 新增：model_cost_configs 成本配置表

-- 创建模型成本配置表（支持版本管理）
CREATE TABLE IF NOT EXISTS model_cost_configs (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 关联模型
    model_id VARCHAR(36) NOT NULL,

    -- 成本参数
    gpu_count INT NOT NULL CHECK (gpu_count > 0),
    power_per_gpu_w INT NOT NULL CHECK (power_per_gpu_w > 0),

    -- 电费单价（元/度）
    electricity_price_per_kwh DECIMAL(10,4) NOT NULL CHECK (electricity_price_per_kwh >= 0),

    -- 硬件折旧（元/张/月）
    depreciation_per_gpu_month INT NOT NULL DEFAULT 0 CHECK (depreciation_per_gpu_month >= 0),

    -- PUE 系数
    pue DECIMAL(4,2) NOT NULL DEFAULT 1.30 CHECK (pue >= 1.0),

    -- 版本管理
    effective_from TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- 外键约束
    CONSTRAINT fk_cost_config_model
        FOREIGN KEY (model_id)
        REFERENCES models(id)
        ON DELETE CASCADE
);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_cost_config_model_effective
    ON model_cost_configs(model_id, effective_from DESC);

CREATE INDEX IF NOT EXISTS idx_cost_config_effective_from
    ON model_cost_configs(effective_from);

-- 添加注释
COMMENT ON COLUMN model_cost_configs.model_id IS '关联 models.id';
COMMENT ON COLUMN model_cost_configs.gpu_count IS 'GPU 数量（张）';
COMMENT ON COLUMN model_cost_configs.power_per_gpu_w IS '单卡满载功耗（瓦特）';
COMMENT ON COLUMN model_cost_configs.electricity_price_per_kwh IS '电费单价（元/度），典型值 0.6~1.2 元/度';
COMMENT ON COLUMN model_cost_configs.depreciation_per_gpu_month IS '单卡每月折旧（元），建议按 36 个月摊销，H100 约 5000 元/月';
COMMENT ON COLUMN model_cost_configs.pue IS 'PUE 系数，数据中心电力使用效率，典型值 1.2~1.5';
COMMENT ON COLUMN model_cost_configs.effective_from IS '该版本参数生效时间';

-- 插入示例配置（可选）
-- Qwen-72B 配置示例：4 张 A100 (400W)，电费 0.8 元/度，折旧 5000 元/月/张
-- INSERT INTO model_cost_configs (model_id, gpu_count, power_per_gpu_w, electricity_price_per_kwh, depreciation_per_gpu_month, pue)
-- SELECT id, 4, 400, 0.8, 5000, 1.3 FROM models WHERE alias LIKE '%qwen%72b%' LIMIT 1;
