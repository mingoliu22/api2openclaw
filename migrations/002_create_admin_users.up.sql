-- api2openclaw 管理员用户表
-- PostgreSQL 14+

-- 管理员用户表
CREATE TABLE IF NOT EXISTS admin_users (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(128) NOT NULL,
    last_login_at TIMESTAMP,
    failed_login_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_admin_users_username ON admin_users(username);
CREATE INDEX idx_admin_users_locked ON admin_users(locked_until) WHERE locked_until IS NOT NULL;

-- 登录失败记录表（用于 IP 锁定）
CREATE TABLE IF NOT EXISTS login_attempts (
    id BIGSERIAL PRIMARY KEY,
    ip_address VARCHAR(45) NOT NULL, -- 支持 IPv6
    username VARCHAR(64),
    success BOOLEAN NOT NULL DEFAULT false,
    attempted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_login_attempts_ip ON login_attempts(ip_address, attempted_at);

-- 插入默认管理员账号（密码: admin123，请在生产环境立即修改）
-- bcrypt hash (cost 10) for "admin123"
INSERT INTO admin_users (username, password_hash)
VALUES ('admin', '$2a$10$Bv9srk3XmUiZNNnIesouZuVHqjK/FoMQ4X9rZzFgb0F.S8FPvSZdS')
ON CONFLICT (username) DO NOTHING;

-- 更新时间触发器
CREATE OR REPLACE FUNCTION update_admin_users_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_admin_users_updated_at
    BEFORE UPDATE ON admin_users
    FOR EACH ROW
    EXECUTE FUNCTION update_admin_users_updated_at();

-- 清理旧登录失败记录（90 天）
DELETE FROM login_attempts WHERE attempted_at < CURRENT_TIMESTAMP - INTERVAL '90 days';
