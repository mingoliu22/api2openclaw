-- 回滚模型成本配置表

-- 删除索引
DROP INDEX IF EXISTS idx_cost_config_model_effective;
DROP INDEX IF EXISTS idx_cost_config_effective_from;

-- 删除表
DROP TABLE IF EXISTS model_cost_configs;
