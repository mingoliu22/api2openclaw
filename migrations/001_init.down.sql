-- api2openclaw 数据库回滚脚本

-- 删除触发器
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TRIGGER IF EXISTS update_api_keys_updated_at ON api_keys;
DROP TRIGGER IF EXISTS update_backends_updated_at ON backends;
DROP TRIGGER IF EXISTS update_models_updated_at ON models;

-- 删除函数
DROP FUNCTION IF EXISTS update_updated_at();

-- 删除表（按依赖关系逆序）
DROP TABLE IF EXISTS rate_limits;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS metrics;
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS backends;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS tenants;
