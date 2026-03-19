-- 回滚模型能力字段迁移

-- 删除索引
DROP INDEX IF EXISTS idx_models_family;
DROP INDEX IF EXISTS idx_models_streaming;
DROP INDEX IF EXISTS idx_models_tool_use;
DROP INDEX IF EXISTS idx_models_json_mode;

-- 删除列
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'models' AND column_name = 'supports_streaming'
    ) THEN
        ALTER TABLE models DROP COLUMN supports_streaming;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'models' AND column_name = 'supports_tool_use'
    ) THEN
        ALTER TABLE models DROP COLUMN supports_tool_use;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'models' AND column_name = 'supports_json_mode'
    ) THEN
        ALTER TABLE models DROP COLUMN supports_json_mode;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'models' AND column_name = 'context_window'
    ) THEN
        ALTER TABLE models DROP COLUMN context_window;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'models' AND column_name = 'model_family'
    ) THEN
        ALTER TABLE models DROP COLUMN model_family;
    END IF;
END $$;
