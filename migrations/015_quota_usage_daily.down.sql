-- 回滚配额使用统计表

-- 删除函数
DROP FUNCTION IF EXISTS increment_quota_usage;
DROP FUNCTION IF EXISTS create_daily_quota_records;

-- 删除索引
DROP INDEX IF EXISTS idx_quota_usage_date;
DROP INDEX IF EXISTS idx_quota_usage_soft_exceeded;

-- 删除表
DROP TABLE IF EXISTS quota_usage_daily;
