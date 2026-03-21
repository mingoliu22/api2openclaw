-- api2openclaw Token 工厂 - 成本核算引擎
-- v0.3.1 新增：stats_cost_daily 每日成本汇总视图

-- 创建每日成本统计视图
CREATE MATERIALIZED VIEW IF NOT EXISTS stats_cost_daily AS
SELECT
    -- 统计日期
    DATE(r.created_at) as stat_date,
    r.model_alias,

    -- 在线小时数（从第一个请求到最后一个请求的时间差）
    EXTRACT(EPOCH FROM (
        MAX(r.created_at) - MIN(r.created_at)
    )::int / 3600.0 as online_hours,

    -- Token 产量
    SUM(r.completion_tokens) as total_tokens,

    -- 电费成本
    -- 公式：gpu_count × power_per_gpu_w × pue × electricity_price / 1000 × online_hours
    -- 注意：这里需要从 model_cost_configs 获取成本参数
    -- 暂时使用固定计算方式，后续可通过 JOIN model_cost_configs 动态计算
    -- 这里先放置 0 作为占位符，由定时 Job 更新
    0::DECIMAL(12,4) as cost_electricity,

    -- 折旧成本
    -- 公式：gpu_count × depreciation_per_gpu_month / (30 × 24) × online_hours
    0::DECIMAL(12,4) as cost_depreciation,

    -- 总成本
    0::DECIMAL(12,4) as cost_total,

    -- 每千 token 成本
    -- 公式：cost_total / (total_tokens / 1000)
    0::DECIMAL(10,6) as cost_per_1k_tokens
FROM request_logs r
WHERE DATE(r.created_at) >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY
    DATE(r.created_at),
    r.model_alias
HAVING SUM(r.completion_tokens) > 0;

-- 创建索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_stats_cost_daily_date_model
    ON stats_cost_daily(stat_date, model_alias);

CREATE INDEX IF NOT EXISTS idx_stats_cost_daily_date
    ON stats_cost_daily(stat_date);

COMMENT ON MATERIALIZED VIEW stats_cost_daily IS
'每日成本统计视图：记录每个模型的在线时长、token 产量和成本。由后台定时 Job 每小时计算并更新成本字段。';

-- 创建成本计算函数（由定时 Job 调用）
CREATE OR REPLACE FUNCTION calculate_daily_costs(stat_date DATE)
RETURNS void AS $$
BEGIN
    -- 遍历每个有产量的模型
    FOR model_rec IN
        SELECT DISTINCT model_alias
        FROM request_logs
        WHERE DATE(created_at) = stat_date
        AND completion_tokens > 0
    LOOP
        -- 更新该模型的成本
        PERFORM update_model_cost(stat_date, model_rec.model_alias);
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- 更新单个模型的成本
CREATE OR REPLACE FUNCTION update_model_cost(stat_date DATE, model_alias VARCHAR)
RETURNS void AS $$
DECLARE
    v_online_hours NUMERIC;
    v_total_tokens BIGINT;
    v_gpu_count INT;
    v_power_per_gpu_w INT;
    v_electricity_price DECIMAL(10,4);
    v_depreciation_per_month INT;
    v_pue DECIMAL(4,2);
    v_cost_electricity DECIMAL(12,4);
    v_cost_depreciation DECIMAL(12,4);
    v_cost_total DECIMAL(12,4);
    v_cost_per_1k DECIMAL(10,6);
BEGIN
    -- 获取在线小时数
    SELECT EXTRACT(EPOCH FROM (
        MAX(created_at) - MIN(created_at)
    )::int / 3600.0
    INTO v_online_hours
    FROM request_logs
    WHERE DATE(created_at) = stat_date
    AND model_alias = model_alias;

    IF v_online_hours IS NULL OR v_online_hours = 0 THEN
        RETURN;
    END IF;

    -- 获取 token 产量
    SELECT SUM(completion_tokens)
    INTO v_total_tokens
    FROM request_logs
    WHERE DATE(created_at) = stat_date
    AND model_alias = model_alias;

    -- 获取成本参数（使用最新生效的配置）
    SELECT
        c.gpu_count,
        c.power_per_gpu_w,
        c.electricity_price_per_kwh,
        c.depreciation_per_gpu_month,
        c.pue
    INTO v_gpu_count, v_power_per_gpu_w, v_electricity_price, v_depreciation_per_month, v_pue
    FROM model_cost_configs c
    JOIN models m ON m.id = c.model_id
    WHERE m.alias = model_alias
    AND c.effective_from <= (
        SELECT stat_date + INTERVAL '24 hours'
    )
    ORDER BY c.effective_from DESC
    LIMIT 1;

    -- 如果没有配置成本参数，使用默认值
    IF v_gpu_count IS NULL THEN
        v_gpu_count := 1;
        v_power_per_gpu_w := 400;
        v_electricity_price := 0.8;
        v_depreciation_month := 5000;
        v_pue := 1.3;
    END IF;

    -- 计算电费成本
    -- 公式：gpu_count × power_per_gpu_w × pue × electricity_price / 1000 × online_hours
    v_cost_electricity := v_gpu_count * v_power_per_gpu_w * v_pue * v_electricity_price / 1000.0 * v_online_hours;

    -- 计算折旧成本
    -- 公式：gpu_count × depreciation_per_month / (30 × 24) × online_hours
    v_cost_depreciation := v_gpu_count * v_depreciation_per_month / 720.0 * v_online_hours;

    -- 总成本
    v_cost_total := v_cost_electricity + v_cost_depreciation;

    -- 每千 token 成本
    IF v_total_tokens > 0 THEN
        v_cost_per_1k := v_cost_total / (v_total_tokens / 1000.0);
    END IF;

    -- 更新物化视图（使用 DELETE + INSERT）
    DELETE FROM stats_cost_daily
    WHERE stat_date = stat_date
    AND model_alias = model_alias;

    INSERT INTO stats_cost_daily (
        stat_date, model_alias, online_hours, total_tokens,
        cost_electricity, cost_depreciation, cost_total, cost_per_1k_tokens
    ) VALUES (
        stat_date, model_alias, v_online_hours, v_total_tokens,
        v_cost_electricity, v_cost_depreciation, v_cost_total, v_cost_per_1k_tokens
    );
END;
$$ LANGUAGE plpgsql;

-- 刷新物化视图函数
CREATE OR REPLACE FUNCTION refresh_stats_cost_daily()
RETURNS void AS $$
BEGIN
    -- 刷新近 90 天的成本数据
    REFRESH MATERIALIZED VIEW CONCURRENTLY stats_cost_daily;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION calculate_daily_costs IS
'计算指定日期的所有模型成本，由定时 Job 每小时调用。';
