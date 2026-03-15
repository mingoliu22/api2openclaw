-- api2openclaw 计费模块
-- PostgreSQL 14+

-- 计费规则表
CREATE TABLE IF NOT EXISTS billing_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    rule_type VARCHAR(32) NOT NULL, -- 'token_based', 'request_based', 'tier'
    model_alias VARCHAR(64),          -- 空表示适用于所有模型
    key_id VARCHAR(36),                 -- 空表示适用于所有 Key
    unit_price DECIMAL(10, 4) NOT NULL DEFAULT 0.0001,  -- 单价（每 token 或每次请求）
    currency VARCHAR(8) NOT NULL DEFAULT 'CNY',
    free_quota INT NOT NULL DEFAULT 0,     -- 免费配额
    tier_threshold INT,                   -- 阶梯阈值（tokens 或请求数）
    tier_price DECIMAL(10, 4),            -- 阶梯价格
    is_active BOOLEAN NOT NULL DEFAULT true,
    valid_from TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uniq_rule UNIQUE (name, model_alias, key_id, valid_from)
);

CREATE INDEX idx_billing_rules_model ON billing_rules(model_alias);
CREATE INDEX idx_billing_rules_key ON billing_rules(key_id);
CREATE INDEX idx_billing_rules_active ON billing_rules(is_active, valid_from, valid_until);

-- 账单表
CREATE TABLE IF NOT EXISTS invoices (
    id BIGSERIAL PRIMARY KEY,
    invoice_number VARCHAR(64) NOT NULL UNIQUE,
    key_id VARCHAR(36) REFERENCES api_keys(id) ON DELETE SET NULL,
    billing_period_start DATE NOT NULL,
    billing_period_end DATE NOT NULL,
    currency VARCHAR(8) NOT NULL DEFAULT 'CNY',
    subtotal DECIMAL(12, 2) NOT NULL DEFAULT 0,
    tax DECIMAL(12, 2) NOT NULL DEFAULT 0,
    discount DECIMAL(12, 2) NOT NULL DEFAULT 0,
    total DECIMAL(12, 2) NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending, paid, overdue, cancelled
    due_date DATE,
    paid_date DATE,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invoices_key_id ON invoices(key_id);
CREATE INDEX idx_invoices_period ON invoices(billing_period_start, billing_period_end);
CREATE INDEX idx_invoices_status ON invoices(status);

-- 账单明细表
CREATE TABLE IF NOT EXISTS invoice_items (
    id BIGSERIAL PRIMARY KEY,
    invoice_id BIGINT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    model_alias VARCHAR(64) NOT NULL,
    request_count INT NOT NULL DEFAULT 0,
    token_count INT NOT NULL DEFAULT 0,
    unit_price DECIMAL(10, 4) NOT NULL,
    tier_applied BOOLEAN NOT NULL DEFAULT false,
    line_total DECIMAL(12, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invoice_items_invoice ON invoice_items(invoice_id);
CREATE INDEX idx_invoice_items_model ON invoice_items(model_alias);

-- 付款记录表
CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,
    invoice_id BIGINT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    amount DECIMAL(12, 2) NOT NULL,
    payment_method VARCHAR(64),     -- 'wechat', 'alipay', 'bank_transfer', etc.
    payment_reference VARCHAR(256),
    status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending, completed, failed, refunded
    paid_at TIMESTAMP,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_payments_invoice ON payments(invoice_id);
CREATE INDEX idx_payments_status ON payments(status);

-- 更新时间触发器
CREATE OR REPLACE FUNCTION update_billing_tables_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_billing_rules_updated_at
    BEFORE UPDATE ON billing_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_billing_tables_updated_at();

CREATE TRIGGER trigger_update_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION update_billing_tables_updated_at();

-- 清理旧数据（保留 2 年）
DELETE FROM invoice_items WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '2 years';
DELETE FROM payments WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '2 years';
DELETE FROM invoices WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '2 years';
