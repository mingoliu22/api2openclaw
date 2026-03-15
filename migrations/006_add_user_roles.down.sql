-- 回滚多管理员角色支持

-- 删除审计日志表
DROP INDEX IF EXISTS idx_admin_audit_logs_action;
DROP INDEX IF EXISTS idx_admin_audit_logs_admin_id;
DROP TABLE IF EXISTS admin_audit_logs;

-- 删除索引
DROP INDEX IF EXISTS idx_admin_users_is_active;
DROP INDEX IF EXISTS idx_admin_users_role;

-- 删除约束
ALTER TABLE admin_users DROP CONSTRAINT IF EXISTS fk_created_by;
ALTER TABLE admin_users DROP CONSTRAINT IF EXISTS check_role;

-- 删除新增的列
ALTER TABLE admin_users DROP COLUMN IF EXISTS created_by;
ALTER TABLE admin_users DROP COLUMN IF EXISTS is_active;
ALTER TABLE admin_users DROP COLUMN IF EXISTS email;
ALTER TABLE admin_users DROP COLUMN IF EXISTS role;

-- 删除枚举类型（如果使用了）
-- DROP TYPE IF EXISTS user_role;
