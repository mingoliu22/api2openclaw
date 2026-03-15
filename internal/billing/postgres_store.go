package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// PostgreSQLStore PostgreSQL 计费存储实现
type PostgreSQLStore struct {
	db *sqlx.DB
}

// NewPostgreSQLStore 创建 PostgreSQL 存储
func NewPostgreSQLStore(db *sqlx.DB) *PostgreSQLStore {
	return &PostgreSQLStore{db: db}
}

// ============== 规则管理 ==============

// ListRules 列出计费规则
func (s *PostgreSQLStore) ListRules(ctx context.Context, activeOnly bool) ([]*BillingRule, error) {
	query := `
		SELECT id, name, description, rule_type, model_alias, key_id,
		       unit_price, currency, free_quota, tier_threshold, tier_price,
		       is_active, valid_from, valid_until, created_at, updated_at
		FROM billing_rules
	`
	if activeOnly {
		query += " WHERE is_active = true"
	}
	query += " ORDER BY created_at DESC"

	var rules []*BillingRule
	err := s.db.SelectContext(ctx, &rules, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list billing rules: %w", err)
	}
	return rules, nil
}

// GetRule 获取单个规则
func (s *PostgreSQLStore) GetRule(ctx context.Context, id int64) (*BillingRule, error) {
	query := `
		SELECT id, name, description, rule_type, model_alias, key_id,
		       unit_price, currency, free_quota, tier_threshold, tier_price,
		       is_active, valid_from, valid_until, created_at, updated_at
		FROM billing_rules
		WHERE id = $1
	`

	var rule BillingRule
	err := s.db.GetContext(ctx, &rule, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get billing rule: %w", err)
	}
	return &rule, nil
}

// CreateRule 创建规则
func (s *PostgreSQLStore) CreateRule(ctx context.Context, rule *BillingRule) error {
	query := `
		INSERT INTO billing_rules (
			name, description, rule_type, model_alias, key_id,
			unit_price, currency, free_quota, tier_threshold, tier_price,
			is_active, valid_from, valid_until
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		rule.Name, rule.Description, rule.RuleType, rule.ModelAlias, rule.KeyID,
		rule.UnitPrice, rule.Currency, rule.FreeQuota, rule.TierThreshold, rule.TierPrice,
		rule.IsActive, rule.ValidFrom, rule.ValidUntil,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create billing rule: %w", err)
	}
	return nil
}

// UpdateRule 更新规则
func (s *PostgreSQLStore) UpdateRule(ctx context.Context, rule *BillingRule) error {
	query := `
		UPDATE billing_rules SET
			name = $2, description = $3, rule_type = $4, model_alias = $5,
			key_id = $6, unit_price = $7, currency = $8, free_quota = $9,
			tier_threshold = $10, tier_price = $11, is_active = $12,
			valid_from = $13, valid_until = $14
		WHERE id = $1
		RETURNING updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		rule.ID, rule.Name, rule.Description, rule.RuleType, rule.ModelAlias,
		rule.KeyID, rule.UnitPrice, rule.Currency, rule.FreeQuota,
		rule.TierThreshold, rule.TierPrice, rule.IsActive,
		rule.ValidFrom, rule.ValidUntil,
	).Scan(&rule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update billing rule: %w", err)
	}
	return nil
}

// DeleteRule 删除规则
func (s *PostgreSQLStore) DeleteRule(ctx context.Context, id int64) error {
	query := `DELETE FROM billing_rules WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete billing rule: %w", err)
	}
	return nil
}

// ============== 账单管理 ==============

// ListInvoices 列出账单
func (s *PostgreSQLStore) ListInvoices(ctx context.Context, keyID *string, status *string, page, limit int) ([]*Invoice, int64, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if keyID != nil {
		whereClause += fmt.Sprintf(" AND key_id = $%d", argIdx)
		args = append(args, *keyID)
		argIdx++
	}

	if status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}

	// 获取总数
	countQuery := "SELECT COUNT(*) FROM invoices " + whereClause
	var total int64
	err := s.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count invoices: %w", err)
	}

	// 获取分页数据
	offset := (page - 1) * limit
	query := `
		SELECT id, invoice_number, key_id, billing_period_start, billing_period_end,
		       currency, subtotal, tax, discount, total, status, due_date, paid_date,
		       notes, created_at, updated_at
		FROM invoices
	` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprint(argIdx) + ` OFFSET $` + fmt.Sprint(argIdx+1)

	args = append(args, limit, offset)

	var invoices []*Invoice
	err = s.db.SelectContext(ctx, &invoices, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list invoices: %w", err)
	}

	return invoices, total, nil
}

// GetInvoice 获取单个账单
func (s *PostgreSQLStore) GetInvoice(ctx context.Context, id int64) (*Invoice, error) {
	query := `
		SELECT id, invoice_number, key_id, billing_period_start, billing_period_end,
		       currency, subtotal, tax, discount, total, status, due_date, paid_date,
		       notes, created_at, updated_at
		FROM invoices
		WHERE id = $1
	`

	var invoice Invoice
	err := s.db.GetContext(ctx, &invoice, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}
	return &invoice, nil
}

// CreateInvoice 创建账单
func (s *PostgreSQLStore) CreateInvoice(ctx context.Context, invoice *Invoice) error {
	query := `
		INSERT INTO invoices (
			invoice_number, key_id, billing_period_start, billing_period_end,
			currency, subtotal, tax, discount, total, status, due_date, notes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		invoice.InvoiceNumber, invoice.KeyID, invoice.BillingPeriodStart, invoice.BillingPeriodEnd,
		invoice.Currency, invoice.Subtotal, invoice.Tax, invoice.Discount, invoice.Total,
		invoice.Status, invoice.DueDate, invoice.Notes,
	).Scan(&invoice.ID, &invoice.CreatedAt, &invoice.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create invoice: %w", err)
	}
	return nil
}

// UpdateInvoiceStatus 更新账单状态
func (s *PostgreSQLStore) UpdateInvoiceStatus(ctx context.Context, id int64, status string) error {
	query := `
		UPDATE invoices SET
			status = $2
			, paid_date = CASE WHEN $2 = 'paid' THEN CURRENT_DATE ELSE paid_date END
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update invoice status: %w", err)
	}
	return nil
}

// ============== 账单明细 ==============

// CreateInvoiceItems 批量创建账单明细
func (s *PostgreSQLStore) CreateInvoiceItems(ctx context.Context, items []*InvoiceItem) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO invoice_items (
			invoice_id, model_alias, request_count, token_count,
			unit_price, tier_applied, line_total
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) RETURNING id, created_at
	`

	for _, item := range items {
		err := tx.QueryRow(query,
			item.InvoiceID, item.ModelAlias, item.RequestCount, item.TokenCount,
			item.UnitPrice, item.TierApplied, item.LineTotal,
		).Scan(&item.ID, &item.CreatedAt)

		if err != nil {
			return fmt.Errorf("failed to create invoice item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// GetInvoiceItems 获取账单明细
func (s *PostgreSQLStore) GetInvoiceItems(ctx context.Context, invoiceID int64) ([]*InvoiceItem, error) {
	query := `
		SELECT id, invoice_id, model_alias, request_count, token_count,
		       unit_price, tier_applied, line_total, created_at
		FROM invoice_items
		WHERE invoice_id = $1
		ORDER BY id
	`

	var items []*InvoiceItem
	err := s.db.SelectContext(ctx, &items, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice items: %w", err)
	}
	return items, nil
}

// ============== 付款记录 ==============

// CreatePayment 创建付款记录
func (s *PostgreSQLStore) CreatePayment(ctx context.Context, payment *Payment) error {
	query := `
		INSERT INTO payments (
			invoice_id, amount, payment_method, payment_reference, status, notes
		) VALUES (
			$1, $2, $3, $4, $5, $6
		) RETURNING id, created_at
	`

	err := s.db.QueryRowContext(ctx, query,
		payment.InvoiceID, payment.Amount, payment.PaymentMethod,
		payment.PaymentReference, payment.Status, payment.Notes,
	).Scan(&payment.ID, &payment.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	return nil
}

// ListPayments 列出付款记录
func (s *PostgreSQLStore) ListPayments(ctx context.Context, invoiceID int64) ([]*Payment, error) {
	query := `
		SELECT id, invoice_id, amount, payment_method, payment_reference,
		       status, paid_at, notes, created_at
		FROM payments
		WHERE invoice_id = $1
		ORDER BY created_at DESC
	`

	var payments []*Payment
	err := s.db.SelectContext(ctx, &payments, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}
	return payments, nil
}

// ============== 用量统计 ==============

// GetUsageStats 获取用量统计
func (s *PostgreSQLStore) GetUsageStats(ctx context.Context, keyID string, startDate, endDate time.Time) (*UsageStats, error) {
	// 确保使用 DATE 类型进行日期比较
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, time.UTC)

	// 获取总体统计
	totalQuery := `
		SELECT
			COALESCE(COUNT(*), 0) as total_requests,
			COALESCE(SUM(total_tokens), 0) as total_tokens
		FROM request_logs
		WHERE key_id = $1
		  AND created_at >= $2
		  AND created_at <= $3
	`

	var totalRequests, totalTokens int64
	err := s.db.QueryRowContext(ctx, totalQuery, keyID, startDate, endDate).Scan(&totalRequests, &totalTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to get total usage stats: %w", err)
	}

	// 按模型分组统计
	modelQuery := `
		SELECT
			model_alias,
			COUNT(*) as request_count,
			COALESCE(SUM(total_tokens), 0) as token_count
		FROM request_logs
		WHERE key_id = $1
		  AND created_at >= $2
		  AND created_at <= $3
		GROUP BY model_alias
		ORDER BY model_alias
	`

	rows, err := s.db.QueryContext(ctx, modelQuery, keyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get model usage stats: %w", err)
	}
	defer rows.Close()

	costByModel := make(map[string]CostInfo)
	for rows.Next() {
		var model string
		var reqCount, tokenCount int64
		if err := rows.Scan(&model, &reqCount, &tokenCount); err != nil {
			return nil, fmt.Errorf("failed to scan model stats: %w", err)
		}
		costByModel[model] = CostInfo{
			RequestCount: int(reqCount),
			TokenCount:   int(tokenCount),
		}
	}

	stats := &UsageStats{
		KeyID:          keyID,
		StartDate:      startDate,
		EndDate:        endDate,
		TotalRequests:  totalRequests,
		TotalTokens:    totalTokens,
		CostByModel:    costByModel,
	}

	return stats, nil
}
