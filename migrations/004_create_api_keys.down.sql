-- 回滚 API Key 管理表

DROP TRIGGER IF EXISTS trigger_update_api_keys_updated_at ON api_keys;
DROP FUNCTION IF EXISTS update_api_keys_updated_at();

DROP TABLE IF EXISTS api_key_usage;
DROP TABLE IF EXISTS api_keys;
