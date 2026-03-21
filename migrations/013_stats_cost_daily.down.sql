-- 回滚每日成本统计视图

-- 删除函数
DROP FUNCTION IF EXISTS update_model_cost;
DROP FUNCTION IF EXISTS calculate_daily_costs;
DROP FUNCTION IF EXISTS refresh_stats_cost_daily;

-- 删除物化视图
DROP MATERIALIZED VIEW IF EXISTS stats_cost_daily;

-- 删除索引
DROP INDEX IF EXISTS idx_stats_cost_daily_date_model;
DROP INDEX IF EXISTS idx_stats_cost_daily_date;
