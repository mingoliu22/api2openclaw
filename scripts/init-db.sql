-- api2openclaw 数据库初始化脚本
-- 此脚本在 PostgreSQL 容器首次启动时自动执行

\echo '正在初始化 api2openclaw 数据库...'

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 输出初始化完成信息
SELECT 'api2openclaw 数据库初始化完成！' AS status;
