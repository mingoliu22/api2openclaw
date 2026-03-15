-- 回滚模型配置表

DROP TRIGGER IF EXISTS trigger_update_models_updated_at ON models;
DROP FUNCTION IF EXISTS update_models_updated_at();

DROP TABLE IF EXISTS model_config_events;
DROP TABLE IF EXISTS model_health;
DROP TABLE IF EXISTS models;
