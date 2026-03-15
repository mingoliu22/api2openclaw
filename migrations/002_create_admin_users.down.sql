-- 回滚管理员用户表

DROP TRIGGER IF EXISTS trigger_update_admin_users_updated_at ON admin_users;
DROP FUNCTION IF EXISTS update_admin_users_updated_at();

DROP TABLE IF EXISTS login_attempts;
DROP TABLE IF EXISTS admin_users;
