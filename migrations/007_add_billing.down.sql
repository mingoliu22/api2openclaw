-- 回滚计费模块

-- 删除触发器
DROP TRIGGER IF EXISTS trigger_update_invoices_updated_at ON invoices;
DROP TRIGGER IF EXISTS trigger_update_billing_rules_updated_at ON billing_rules;
DROP FUNCTION IF EXISTS update_billing_tables_updated_at();

-- 删除表
DROP INDEX IF EXISTS idx_payments_status ON payments;
DROP INDEX IF EXISTS idx_payments_invoice ON payments;
DROP TABLE IF EXISTS payments;

DROP INDEX IF EXISTS idx_invoice_items_model ON invoice_items;
DROP INDEX IF EXISTS idx_invoice_items_invoice ON invoice_items;
DROP TABLE IF EXISTS invoice_items;

DROP INDEX IF EXISTS idx_invoices_status ON invoices;
DROP INDEX IF EXISTS idx_invoices_period ON invoices;
DROP INDEX IF EXISTS idx_invoices_key_id ON invoices;
DROP TABLE IF EXISTS invoices;

DROP INDEX IF EXISTS idx_billing_rules_active ON billing_rules;
DROP INDEX IF EXISTS idx_billing_rules_key ON billing_rules;
DROP INDEX IF EXISTS idx_billing_rules_model ON billing_rules(model_alias);
DROP TABLE IF EXISTS billing_rules;

-- 删除函数
DROP FUNCTION IF EXISTS update_billing_tables_updated_at();
