-- api2openclaw 多管理员角色支持
-- PostgreSQL 14+

-- 添加角色字段到 admin_users 表
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS role VARCHAR(32) NOT NULL DEFAULT 'admin';
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS email VARCHAR(255);
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS created_by VARCHAR(36);

-- 添加约束：角色只能是预定义的值
ALTER TABLE admin_users ADD CONSTRAINT check_role
    CHECK (role IN ('super_admin', 'admin', 'operator', 'viewer'));

-- 添加外键约束：created_by 关联到 admin_users
ALTER TABLE admin_users ADD CONSTRAINT fk_created_by
    FOREIGN KEY (created_by) REFERENCES admin_users(id) ON DELETE SET NULL;

-- 创建角色枚举类型（可选，使用约束更灵活）
-- DO $$ BEGIN
--     CREATE TYPE user_role AS ENUM ('super_admin', 'admin', 'operator', 'viewer');
-- EXCEPTION
--     WHEN duplicate_object THEN null;
-- END $$;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_admin_users_role ON admin_users(role);
CREATE INDEX IF NOT EXISTS idx_admin_users_is_active ON admin_users(is_active);

-- 用户操作日志表（审计管理员操作）
CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    admin_id VARCHAR(36) REFERENCES admin_users(id) ON DELETE SET NULL,
    action VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64),
    resource_id VARCHAR(128),
    details JSONB,
    ip_address VARCHAR(45),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_admin_id ON admin_audit_logs(admin_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_action ON admin_audit_logs(action, created_at DESC);

-- 更新现有 admin 用户的角色
UPDATE admin_users SET role = 'super_admin' WHERE username = 'admin' AND role = 'admin';

-- 清理旧的审计日志（180 天）
DELETE FROM admin_audit_logs WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '180 days';
