package billing

import (
	"context"
	"fmt"
	"time"
)

// BillingRule 计费规则
type BillingRule struct {
	ID           int64      `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Description  string     `json:"description" db:"description"`
	RuleType     string     `json:"rule_type" db:"rule_type"` // token_based, request_based, tier
	ModelAlias   *string    `json:"model_alias,omitempty" db:"model_alias"`
	KeyID        *string    `json:"key_id,omitempty" db:"key_id"`
	UnitPrice    float64    `json:"unit_price" db:"unit_price"`
	Currency     string     `json:"currency" db:"currency"`
	FreeQuota    int        `json:"free_quota" db:"free_quota"`
	TierThreshold *int       `json:"tier_threshold,omitempty" db:"tier_threshold"`
	TierPrice    *float64   `json:"tier_price,omitempty" db:"tier_price"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	ValidFrom    time.Time  `json:"valid_from" db:"valid_from"`
	ValidUntil    *time.Time `json:"valid_until,omitempty" db:"valid_until"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// Invoice 账单
type Invoice struct {
	ID                    int64     `json:"id" db:"id"`
	InvoiceNumber        string    `json:"invoice_number" db:"invoice_number"`
	KeyID                 *string   `json:"key_id,omitempty" db:"key_id"`
	BillingPeriodStart   time.Time `json:"billing_period_start" db:"billing_period_start"`
	BillingPeriodEnd     time.Time `json:"billing_period_end" db:"billing_period_end"`
	Currency              string    `json:"currency" db:"currency"`
	Subtotal              float64   `json:"subtotal" db:"subtotal"`
	Tax                   float64   `json:"tax" db:"tax"`
	Discount              float64   `json:"discount" db:"discount"`
	Total                 float64   `json:"total" db:"total"`
	Status                string    `json:"status" db:"status"` // pending, paid, overdue, cancelled
	DueDate               *time.Time `json:"due_date,omitempty" db:"due_date"`
	PaidDate              *time.Time `json:"paid_date,omitempty" db:"paid_date"`
	Notes                 string    `json:"notes,omitempty" db:"notes"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

// InvoiceItem 账单明细
type InvoiceItem struct {
	ID           int64   `json:"id" db:"id"`
	InvoiceID    int64   `json:"invoice_id" db:"invoice_id"`
	ModelAlias   string  `json:"model_alias" db:"model_alias"`
	RequestCount int    `json:"request_count" db:"request_count"`
	TokenCount   int    `json:"token_count" db:"token_count"`
	UnitPrice    float64 `json:"unit_price" db:"unit_price"`
	TierApplied  bool    `json:"tier_applied" db:"tier_applied"`
	LineTotal    float64 `json:"line_total" db:"line_total"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Payment 付款记录
type Payment struct {
	ID               int64     `json:"id" db:"id"`
	InvoiceID        int64     `json:"invoice_id" db:"invoice_id"`
	Amount           float64   `json:"amount" db:"amount"`
	PaymentMethod    string    `json:"payment_method" db:"payment_method"`
	PaymentReference string   `json:"payment_reference,omitempty" db:"payment_reference"`
	Status           string    `json:"status" db:"status"` // pending, completed, failed, refunded
	PaidAt           *time.Time `json:"paid_at,omitempty" db:"paid_at"`
	Notes            string    `json:"notes,omitempty" db:"notes"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// BillingStore 计费数据存储接口
type BillingStore interface {
	// 规则管理
	ListRules(ctx context.Context, activeOnly bool) ([]*BillingRule, error)
	GetRule(ctx context.Context, id int64) (*BillingRule, error)
	CreateRule(ctx context.Context, rule *BillingRule) error
	UpdateRule(ctx context.Context, rule *BillingRule) error
	DeleteRule(ctx context.Context, id int64) error

	// 账单管理
	ListInvoices(ctx context.Context, keyID *string, status *string, page, limit int) ([]*Invoice, int64, error)
	GetInvoice(ctx context.Context, id int64) (*Invoice, error)
	CreateInvoice(ctx context.Context, invoice *Invoice) error
	UpdateInvoiceStatus(ctx context.Context, id int64, status string) error

	// 账单明细
	CreateInvoiceItems(ctx context.Context, items []*InvoiceItem) error
	GetInvoiceItems(ctx context.Context, invoiceID int64) ([]*InvoiceItem, error)

	// 付款记录
	CreatePayment(ctx context.Context, payment *Payment) error
	ListPayments(ctx context.Context, invoiceID int64) ([]*Payment, error)

	// 用量统计
	GetUsageStats(ctx context.Context, keyID string, startDate, endDate time.Time) (*UsageStats, error)
}

// UsageStats 用量统计
type UsageStats struct {
	KeyID               string            `json:"key_id"`
	StartDate           time.Time         `json:"start_date"`
	EndDate             time.Time         `json:"end_date"`
	TotalRequests      int64             `json:"total_requests"`
	TotalTokens        int64             `json:"total_tokens"`
	TotalCost           float64           `json:"total_cost"`
	CostByModel         map[string]CostInfo `json:"cost_by_model"`
}

// CostInfo 成本信息
type CostInfo struct {
	RequestCount int     `json:"request_count"`
	TokenCount   int     `json:"token_count"`
	Cost         float64 `json:"cost"`
}

// BillingService 计费服务
type BillingService struct {
	store BillingStore
}

// NewBillingService 创建计费服务
func NewBillingService(store BillingStore) *BillingService {
	return &BillingService{store: store}
}

// CalculateBilling 计算账单
func (s *BillingService) CalculateBilling(ctx context.Context, keyID string, startDate, endDate time.Time) (*Invoice, []*InvoiceItem, error) {
	// 获取用量统计
	stats, err := s.store.GetUsageStats(ctx, keyID, startDate, endDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	// 获取适用的计费规则
	rules, err := s.store.ListRules(ctx, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get billing rules: %w", err)
	}

	// 按模型分组计算费用
	costByModel := make(map[string]*InvoiceItem)
	for model, costInfo := range stats.CostByModel {
		item := &InvoiceItem{
			ModelAlias:   model,
			RequestCount: costInfo.RequestCount,
			TokenCount:   costInfo.TokenCount,
		}

		// 查找适用的计费规则
		for _, rule := range rules {
			if !s.isRuleApplicable(rule, keyID, model) {
				continue
			}

			unitPrice := rule.UnitPrice
			if rule.RuleType == "tier" && rule.TierThreshold != nil && rule.TierPrice != nil {
				if costInfo.TokenCount > *rule.TierThreshold {
					unitPrice = *rule.TierPrice
					item.TierApplied = true
				}
			}

			item.UnitPrice = unitPrice

			// 计算行总计
			if rule.RuleType == "token_based" {
				item.LineTotal = float64(costInfo.TokenCount) * unitPrice
			} else if rule.RuleType == "request_based" {
				item.LineTotal = float64(costInfo.RequestCount) * unitPrice
			}

			costByModel[model] = item
			break
		}
	}

	// 生成账单
	subtotal := 0.0
	for _, item := range costByModel {
		subtotal += item.LineTotal
	}

	tax := subtotal * 0.0 // TODO: 配置税率
	discount := 0.0   // TODO: 配置折扣
	total := subtotal + tax - discount

	// 生成账单号码
	invoiceNumber := fmt.Sprintf("INV-%s-%s", startDate.Format("200601"), endDate.Format("200601"))

	invoice := &Invoice{
		InvoiceNumber:      invoiceNumber,
		KeyID:              &keyID,
		BillingPeriodStart: startDate,
		BillingPeriodEnd:   endDate,
		Currency:           "CNY",
		Subtotal:           subtotal,
		Tax:                tax,
		Discount:           discount,
		Total:              total,
		Status:             "pending",
	}

	// 准备明细项
	items := make([]*InvoiceItem, 0, len(costByModel))
	for _, item := range costByModel {
		items = append(items, item)
	}

	return invoice, items, nil
}

// isRuleApplicable 检查规则是否适用
func (s *BillingService) isRuleApplicable(rule *BillingRule, keyID string, modelAlias string) bool {
	if !rule.IsActive {
		return false
	}

	now := time.Now()
	if now.Before(rule.ValidFrom) {
		return false
	}

	if rule.ValidUntil != nil && now.After(*rule.ValidUntil) {
		return false
	}

	// 检查模型匹配
	if rule.ModelAlias != nil && *rule.ModelAlias != modelAlias {
		return false
	}

	// 检查 Key 匹配
	if rule.KeyID != nil && *rule.KeyID != keyID {
		return false
	}

	return true
}

// GenerateInvoice 生成账单
func (s *BillingService) GenerateInvoice(ctx context.Context, keyID string, startDate, endDate time.Time) (*Invoice, error) {
	invoice, items, err := s.CalculateBilling(ctx, keyID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 保存账单
	if err := s.store.CreateInvoice(ctx, invoice); err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// 保存明细
	for _, item := range items {
		item.InvoiceID = invoice.ID
	}
	if err := s.store.CreateInvoiceItems(ctx, items); err != nil {
		return nil, fmt.Errorf("failed to create invoice items: %w", err)
	}

	return invoice, nil
}

// ============== 规则管理代理方法 ==============

// ListRules 列出计费规则
func (s *BillingService) ListRules(ctx context.Context, activeOnly bool) ([]*BillingRule, error) {
	return s.store.ListRules(ctx, activeOnly)
}

// GetRule 获取单个规则
func (s *BillingService) GetRule(ctx context.Context, id int64) (*BillingRule, error) {
	return s.store.GetRule(ctx, id)
}

// CreateRule 创建规则
func (s *BillingService) CreateRule(ctx context.Context, rule *BillingRule) error {
	return s.store.CreateRule(ctx, rule)
}

// UpdateRule 更新规则
func (s *BillingService) UpdateRule(ctx context.Context, rule *BillingRule) error {
	return s.store.UpdateRule(ctx, rule)
}

// DeleteRule 删除规则
func (s *BillingService) DeleteRule(ctx context.Context, id int64) error {
	return s.store.DeleteRule(ctx, id)
}

// ============== 账单管理代理方法 ==============

// ListInvoices 列出账单
func (s *BillingService) ListInvoices(ctx context.Context, keyID *string, status *string, page, limit int) ([]*Invoice, int64, error) {
	return s.store.ListInvoices(ctx, keyID, status, page, limit)
}

// GetInvoice 获取单个账单
func (s *BillingService) GetInvoice(ctx context.Context, id int64) (*Invoice, error) {
	return s.store.GetInvoice(ctx, id)
}

// GetInvoiceItems 获取账单明细
func (s *BillingService) GetInvoiceItems(ctx context.Context, invoiceID int64) ([]*InvoiceItem, error) {
	return s.store.GetInvoiceItems(ctx, invoiceID)
}

// UpdateInvoiceStatus 更新账单状态
func (s *BillingService) UpdateInvoiceStatus(ctx context.Context, id int64, status string) error {
	return s.store.UpdateInvoiceStatus(ctx, id, status)
}

// ============== 付款记录代理方法 ==============

// CreatePayment 创建付款记录
func (s *BillingService) CreatePayment(ctx context.Context, payment *Payment) error {
	return s.store.CreatePayment(ctx, payment)
}

// ListPayments 列出付款记录
func (s *BillingService) ListPayments(ctx context.Context, invoiceID int64) ([]*Payment, error) {
	return s.store.ListPayments(ctx, invoiceID)
}

// ============== 用量统计代理方法 ==============

// GetUsageStats 获取用量统计
func (s *BillingService) GetUsageStats(ctx context.Context, keyID string, startDate, endDate time.Time) (*UsageStats, error) {
	return s.store.GetUsageStats(ctx, keyID, startDate, endDate)
}
