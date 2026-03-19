-- 回滚部署指南配置表

-- 删除触发器
DROP TRIGGER IF EXISTS trigger_update_deploy_guides_updated_at ON deploy_guides;
DROP FUNCTION IF EXISTS update_deploy_guides_updated_at();

-- 删除表
DROP TABLE IF EXISTS deploy_guides;