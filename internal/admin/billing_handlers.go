package admin

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liu/api2openclaw/internal/billing"
)

// BillingHandlers 计费处理器
type BillingHandlers struct {
	billingService *billing.BillingService
}

// NewBillingHandlers 创建计费处理器
func NewBillingHandlers(billingService *billing.BillingService) *BillingHandlers {
	return &BillingHandlers{
		billingService: billingService,
	}
}

// ============== 计费规则管理 ==============

// ListRules 列出计费规则
func (h *BillingHandlers) ListRules(c *gin.Context) {
	ctx := c.Request.Context()
	activeOnly := c.DefaultQuery("active_only", "false") == "true"

	rules, err := h.billingService.ListRules(ctx, activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// GetRule 获取单个规则
func (h *BillingHandlers) GetRule(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	rule, err := h.billingService.GetRule(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	RuleType     string   `json:"rule_type" binding:"required,oneof=token_based request_based tier"`
	ModelAlias   *string  `json:"model_alias"`
	KeyID        *string  `json:"key_id"`
	UnitPrice    float64  `json:"unit_price" binding:"required,min=0"`
	Currency     string   `json:"currency" binding:"required"`
	FreeQuota    int      `json:"free_quota"`
	TierThreshold *int    `json:"tier_threshold"`
	TierPrice    *float64 `json:"tier_price"`
	IsActive     bool     `json:"is_active"`
	ValidFrom    string   `json:"valid_from" binding:"required"`
	ValidUntil   *string  `json:"valid_until"`
}

// CreateRule 创建规则
func (h *BillingHandlers) CreateRule(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validFrom, err := time.Parse(time.RFC3339, req.ValidFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid valid_from format"})
		return
	}

	var validUntil *time.Time
	if req.ValidUntil != nil && *req.ValidUntil != "" {
		t, err := time.Parse(time.RFC3339, *req.ValidUntil)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid valid_until format"})
			return
		}
		validUntil = &t
	}

	rule := &billing.BillingRule{
		Name:         req.Name,
		Description:  req.Description,
		RuleType:     req.RuleType,
		ModelAlias:   req.ModelAlias,
		KeyID:        req.KeyID,
		UnitPrice:    req.UnitPrice,
		Currency:     req.Currency,
		FreeQuota:    req.FreeQuota,
		TierThreshold: req.TierThreshold,
		TierPrice:    req.TierPrice,
		IsActive:     req.IsActive,
		ValidFrom:    validFrom,
		ValidUntil:   validUntil,
	}

	if err := h.billingService.CreateRule(ctx, rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name         *string  `json:"name"`
	Description  *string  `json:"description"`
	RuleType     *string  `json:"rule_type" binding:"omitempty,oneof=token_based request_based tier"`
	ModelAlias   *string  `json:"model_alias"`
	KeyID        *string  `json:"key_id"`
	UnitPrice    *float64 `json:"unit_price" binding:"omitempty,min=0"`
	Currency     *string  `json:"currency"`
	FreeQuota    *int     `json:"free_quota"`
	TierThreshold *int    `json:"tier_threshold"`
	TierPrice    *float64 `json:"tier_price"`
	IsActive     *bool    `json:"is_active"`
	ValidFrom    *string  `json:"valid_from"`
	ValidUntil   *string  `json:"valid_until"`
}

// UpdateRule 更新规则
func (h *BillingHandlers) UpdateRule(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	rule, err := h.billingService.GetRule(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.RuleType != nil {
		rule.RuleType = *req.RuleType
	}
	if req.ModelAlias != nil {
		rule.ModelAlias = req.ModelAlias
	}
	if req.KeyID != nil {
		rule.KeyID = req.KeyID
	}
	if req.UnitPrice != nil {
		rule.UnitPrice = *req.UnitPrice
	}
	if req.Currency != nil {
		rule.Currency = *req.Currency
	}
	if req.FreeQuota != nil {
		rule.FreeQuota = *req.FreeQuota
	}
	if req.TierThreshold != nil {
		rule.TierThreshold = req.TierThreshold
	}
	if req.TierPrice != nil {
		rule.TierPrice = req.TierPrice
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}
	if req.ValidFrom != nil {
		t, err := time.Parse(time.RFC3339, *req.ValidFrom)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid valid_from format"})
			return
		}
		rule.ValidFrom = t
	}
	if req.ValidUntil != nil {
		if *req.ValidUntil == "" {
			rule.ValidUntil = nil
		} else {
			t, err := time.Parse(time.RFC3339, *req.ValidUntil)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid valid_until format"})
				return
			}
			rule.ValidUntil = &t
		}
	}

	if err := h.billingService.UpdateRule(ctx, rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteRule 删除规则
func (h *BillingHandlers) DeleteRule(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	if err := h.billingService.DeleteRule(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rule deleted"})
}

// ============== 账单管理 ==============

// ListInvoices 列出账单
func (h *BillingHandlers) ListInvoices(c *gin.Context) {
	ctx := c.Request.Context()

	keyID := c.Query("key_id")
	var keyIDPtr *string
	if keyID != "" {
		keyIDPtr = &keyID
	}

	status := c.Query("status")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	invoices, total, err := h.billingService.ListInvoices(ctx, keyIDPtr, statusPtr, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invoices": invoices,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// GetInvoice 获取单个账单
func (h *BillingHandlers) GetInvoice(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice id"})
		return
	}

	invoice, err := h.billingService.GetInvoice(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	// 获取账单明细
	items, err := h.billingService.GetInvoiceItems(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取付款记录
	payments, err := h.billingService.ListPayments(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invoice":  invoice,
		"items":    items,
		"payments": payments,
	})
}

// GenerateInvoiceRequest 生成账单请求
type GenerateInvoiceRequest struct {
	KeyID     string `json:"key_id" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

// GenerateInvoice 生成账单
func (h *BillingHandlers) GenerateInvoice(c *gin.Context) {
	ctx := c.Request.Context()

	var req GenerateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format"})
		return
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format"})
		return
	}

	invoice, err := h.billingService.GenerateInvoice(ctx, req.KeyID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, invoice)
}

// UpdateInvoiceStatusRequest 更新账单状态请求
type UpdateInvoiceStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending paid overdue cancelled"`
}

// UpdateInvoiceStatus 更新账单状态
func (h *BillingHandlers) UpdateInvoiceStatus(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice id"})
		return
	}

	var req UpdateInvoiceStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.billingService.UpdateInvoiceStatus(ctx, id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "invoice status updated"})
}

// ============== 付款管理 ==============

// CreatePaymentRequest 创建付款请求
type CreatePaymentRequest struct {
	Amount           float64 `json:"amount" binding:"required,min=0"`
	PaymentMethod    string  `json:"payment_method" binding:"required"`
	PaymentReference string  `json:"payment_reference"`
	Notes            string  `json:"notes"`
}

// CreatePayment 创建付款记录
func (h *BillingHandlers) CreatePayment(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	invoiceID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice id"})
		return
	}

	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment := &billing.Payment{
		InvoiceID:        invoiceID,
		Amount:           req.Amount,
		PaymentMethod:    req.PaymentMethod,
		PaymentReference: &req.PaymentReference,
		Status:           "pending",
		Notes:            req.Notes,
	}

	if err := h.billingService.CreatePayment(ctx, payment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 如果付款金额足够，更新账单状态为已支付
	if req.Amount > 0 {
		invoice, err := h.billingService.GetInvoice(ctx, invoiceID)
		if err == nil && payment.Status == "completed" {
			// 计算已付金额
			payments, _ := h.billingService.ListPayments(ctx, invoiceID)
			totalPaid := 0.0
			for _, p := range payments {
				if p.Status == "completed" {
					totalPaid += p.Amount
				}
			}
			if totalPaid >= invoice.Total {
				h.billingService.UpdateInvoiceStatus(ctx, invoiceID, "paid")
			}
		}
	}

	c.JSON(http.StatusCreated, payment)
}

// ============== 用量查询 ==============

// GetUsageStats 获取用量统计
func (h *BillingHandlers) GetUsageStats(c *gin.Context) {
	ctx := c.Request.Context()

	keyID := c.Query("key_id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key_id is required"})
		return
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format"})
			return
		}
	} else {
		// 默认本月第一天
		now := time.Now()
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	if endDateStr != "" {
		endDate, err = time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format"})
			return
		}
	} else {
		// 默认今天
		endDate = time.Now()
	}

	stats, err := h.billingService.GetUsageStats(ctx, keyID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 计算预估费用（使用当前规则）
	rules, err := h.billingService.ListRules(ctx, true)
	if err == nil {
		stats.TotalCost = 0
		for model, costInfo := range stats.CostByModel {
			for _, rule := range rules {
				if rule.RuleType == "token_based" &&
					(rule.ModelAlias == nil || *rule.ModelAlias == model) &&
					(rule.KeyID == nil || *rule.KeyID == keyID) {
					stats.TotalCost += float64(costInfo.TokenCount) * rule.UnitPrice
					break
				}
			}
		}
	}

	c.JSON(http.StatusOK, stats)
}

// ExportInvoice 导出账单为 CSV
func (h *BillingHandlers) ExportInvoice(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invoice id"})
		return
	}

	invoice, err := h.billingService.GetInvoice(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	items, err := h.billingService.GetInvoiceItems(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", invoice.InvoiceNumber))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// 写入表头
	writer.Write([]string{"模型", "请求数", "Token数", "单价", "阶梯价", "行总计"})
	for _, item := range items {
		tierPrice := "否"
		if item.TierApplied {
			tierPrice = "是"
		}
		writer.Write([]string{
			item.ModelAlias,
			strconv.Itoa(item.RequestCount),
			strconv.Itoa(item.TokenCount),
			strconv.FormatFloat(item.UnitPrice, 'f', 4, 64),
			tierPrice,
			strconv.FormatFloat(item.LineTotal, 'f', 2, 64),
		})
	}

	// 写入汇总
	writer.Write([]string{})
	writer.Write([]string{"", "", "", "小计", strconv.FormatFloat(invoice.Subtotal, 'f', 2, 64)})
	writer.Write([]string{"", "", "", "税", strconv.FormatFloat(invoice.Tax, 'f', 2, 64)})
	writer.Write([]string{"", "", "", "折扣", strconv.FormatFloat(invoice.Discount, 'f', 2, 64)})
	writer.Write([]string{"", "", "", "总计", strconv.FormatFloat(invoice.Total, 'f', 2, 64)})
}

// StreamInvoiceCSV 流式导出账单列表
func (h *BillingHandlers) StreamInvoiceCSV(c *gin.Context) {
	ctx := c.Request.Context()

	keyID := c.Query("key_id")
	var keyIDPtr *string
	if keyID != "" {
		keyIDPtr = &keyID
	}

	status := c.Query("status")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=invoices.csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// 写入表头
	writer.Write([]string{"账单号", "Key ID", "账期开始", "账期结束", "币种", "小计", "税", "折扣", "总计", "状态", "到期日", "支付日", "创建日"})

	// 分批查询并写入
	page := 1
	limit := 100
	for {
		invoices, _, err := h.billingService.ListInvoices(ctx, keyIDPtr, statusPtr, page, limit)
		if err != nil {
			break
		}

		if len(invoices) == 0 {
			break
		}

		for _, inv := range invoices {
			var dueDate, paidDate string
			if inv.DueDate != nil {
				dueDate = inv.DueDate.Format("2006-01-02")
			}
			if inv.PaidDate != nil {
				paidDate = inv.PaidDate.Format("2006-01-02")
			}
			var keyIDVal string
			if inv.KeyID != nil {
				keyIDVal = *inv.KeyID
			}

			writer.Write([]string{
				inv.InvoiceNumber,
				keyIDVal,
				inv.BillingPeriodStart.Format("2006-01-02"),
				inv.BillingPeriodEnd.Format("2006-01-02"),
				inv.Currency,
				strconv.FormatFloat(inv.Subtotal, 'f', 2, 64),
				strconv.FormatFloat(inv.Tax, 'f', 2, 64),
				strconv.FormatFloat(inv.Discount, 'f', 2, 64),
				strconv.FormatFloat(inv.Total, 'f', 2, 64),
				inv.Status,
				dueDate,
				paidDate,
				inv.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}

		if len(invoices) < limit {
			break
		}
		page++
	}
}

// Helper function for streaming large CSV exports
func (h *BillingHandlers) streamCSVResponse(c *gin.Context, filename string, writeRows func(*csv.Writer)) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Transfer-Encoding", "chunked")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// 设置 bufio 以提高性能
	flusher := c.Writer.(http.Flusher)
	bufWriter := bufio.NewWriterSize(c.Writer, 4096)
	writer = csv.NewWriter(bufWriter)

	writeRows(writer)

	bufWriter.Flush()
	flusher.Flush()
}
