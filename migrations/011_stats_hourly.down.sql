-- 回滚 stats_hourly 物化视图

-- 删除物化视图
DROP MATERIALIZED VIEW IF EXISTS stats_hourly;

-- 删除索引
DROP INDEX IF EXISTS idx_stats_hourly_date_hour_model;
DROP INDEX IF EXISTS idx_stats_hourly_date;
DROP INDEX IF EXISTS idx_stats_hourly_model;

-- 删除刷新函数
DROP FUNCTION IF EXISTS refresh_stats_hourly();

-- 删除系统配置表（如果其他功能未使用）
-- DROP TABLE IF EXISTS system_configs;
