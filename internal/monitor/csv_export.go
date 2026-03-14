package monitor

import (
	"context"
	"encoding/csv"
	"io"
	"strconv"
	"time"
)

// CSVExporter CSV 导出器
type CSVExporter struct {
	store UsageStatsStore
}

// NewCSVExporter 创建 CSV 导出器
func NewCSVExporter(store UsageStatsStore) *CSVExporter {
	return &CSVExporter{store: store}
}

// UsageStatsStore 用量统计存储接口
type UsageStatsStore interface {
	QueryMetrics(ctx context.Context, filter *MetricFilter) ([]*Metric, error)
}

// ExportUsageReport 导出用量报告
func (e *CSVExporter) ExportUsageReport(ctx context.Context, filter *UsageReportFilter) ([]byte, error) {
	// 查询指标数据
	metricFilter := &MetricFilter{
		APIKeyID: filter.APIKeyID,
		TenantID: filter.TenantID,
		Model:    filter.Model,
		Limit:    filter.Limit,
	}
	if filter.StartTime != nil {
		metricFilter.StartTime = *filter.StartTime
	}
	if filter.EndTime != nil {
		metricFilter.EndTime = *filter.EndTime
	}

	metrics, err := e.store.QueryMetrics(ctx, metricFilter)
	if err != nil {
		return nil, err
	}

	// 生成 CSV
	var output []byte
	buffer := &buffer{&output}
	writer := csv.NewWriter(buffer)

	// 写入表头
	headers := []string{
		"Timestamp",
		"API Key ID",
		"Tenant ID",
		"Model",
		"Request ID",
		"Status Code",
		"Latency (ms)",
		"Prompt Tokens",
		"Completion Tokens",
		"Total Tokens",
		"Error",
	}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	// 写入数据行
	for _, metric := range metrics {
		row := []string{
			metric.Timestamp.Format(time.RFC3339),
			metric.APIKeyID,
			metric.TenantID,
			metric.Model,
			metric.RequestID,
			strconv.Itoa(metric.StatusCode),
			strconv.FormatInt(metric.LatencyMs, 10),
			strconv.Itoa(metric.PromptTokens),
			strconv.Itoa(metric.CompletionTokens),
			strconv.Itoa(metric.TotalTokens),
			metric.Error,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return output, nil
}

// ExportAggregatedReport 导出聚合报告
func (e *CSVExporter) ExportAggregatedReport(ctx context.Context, filter *UsageReportFilter) ([]byte, error) {
	// 查询指标数据
	metricFilter := &MetricFilter{
		APIKeyID: filter.APIKeyID,
		TenantID: filter.TenantID,
		Model:    filter.Model,
		Limit:    filter.Limit,
	}
	if filter.StartTime != nil {
		metricFilter.StartTime = *filter.StartTime
	}
	if filter.EndTime != nil {
		metricFilter.EndTime = *filter.EndTime
	}

	metrics, err := e.store.QueryMetrics(ctx, metricFilter)
	if err != nil {
		return nil, err
	}

	// 按时间聚合
	aggregated := e.aggregateByTime(metrics, filter.GroupBy)

	// 生成 CSV
	var output []byte
	buffer := &buffer{&output}
	writer := csv.NewWriter(buffer)

	// 写入表头
	headers := []string{"Period", "Total Requests", "Success Requests", "Error Requests", "Avg Latency (ms)", "Total Tokens"}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	// 写入聚合数据
	for _, agg := range aggregated {
		row := []string{
			agg.Period,
			strconv.FormatInt(agg.TotalRequests, 10),
			strconv.FormatInt(agg.SuccessRequests, 10),
			strconv.FormatInt(agg.ErrorRequests, 10),
			strconv.FormatFloat(agg.AvgLatencyMs, 'f', 2, 64),
			strconv.FormatInt(agg.TotalTokens, 10),
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return output, nil
}

// aggregateByTime 按时间聚合
func (e *CSVExporter) aggregateByTime(metrics []*Metric, groupBy string) []*AggregatedMetric {
	aggregatedMap := make(map[string]*AggregatedMetric)

	for _, metric := range metrics {
		var period string
		switch groupBy {
		case "hour":
			period = metric.Timestamp.Format("2006-01-02T15:00")
		case "day":
			period = metric.Timestamp.Format("2006-01-02")
		case "month":
			period = metric.Timestamp.Format("2006-01")
		default:
			period = metric.Timestamp.Format("2006-01-02")
		}

		if _, ok := aggregatedMap[period]; !ok {
			aggregatedMap[period] = &AggregatedMetric{
				Period: period,
			}
		}

		agg := aggregatedMap[period]
		agg.TotalRequests++
		if metric.StatusCode >= 200 && metric.StatusCode < 400 {
			agg.SuccessRequests++
		} else {
			agg.ErrorRequests++
		}
		agg.TotalTokens += int64(metric.TotalTokens)
		agg.AvgLatencyMs += float64(metric.LatencyMs)
	}

	// 计算平均延迟
	result := make([]*AggregatedMetric, 0, len(aggregatedMap))
	for _, agg := range aggregatedMap {
		if agg.TotalRequests > 0 {
			agg.AvgLatencyMs /= float64(agg.TotalRequests)
		}
		result = append(result, agg)
	}

	return result
}

// AggregatedMetric 聚合指标
type AggregatedMetric struct {
	Period          string
	TotalRequests   int64
	SuccessRequests int64
	ErrorRequests   int64
	AvgLatencyMs    float64
	TotalTokens     int64
}

// UsageReportFilter 用量报告过滤器
type UsageReportFilter struct {
	APIKeyID  string     `json:"api_key_id,omitempty"`
	TenantID  string     `json:"tenant_id,omitempty"`
	Model     string     `json:"model,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	GroupBy   string     `json:"group_by,omitempty"` // hour, day, month
	Limit     int        `json:"limit,omitempty"`
}

// buffer 用于写入 []byte
type buffer struct {
	b *[]byte
}

func (b *buffer) Write(p []byte) (int, error) {
	*b.b = append(*b.b, p...)
	return len(p), nil
}

// ReportGenerator 报告生成器
type ReportGenerator struct {
	exporter *CSVExporter
}

// NewReportGenerator 创建报告生成器
func NewReportGenerator(store UsageStatsStore) *ReportGenerator {
	return &ReportGenerator{
		exporter: NewCSVExporter(store),
	}
}

// GenerateDailyReport 生成日报
func (g *ReportGenerator) GenerateDailyReport(ctx context.Context, date time.Time, apiKeyID string) ([]byte, error) {
	startTime := date.Truncate(24 * time.Hour)
	endTime := startTime.Add(24 * time.Hour)

	filter := &UsageReportFilter{
		APIKeyID:  apiKeyID,
		StartTime: &startTime,
		EndTime:   &endTime,
		GroupBy:   "hour",
	}

	return g.exporter.ExportAggregatedReport(ctx, filter)
}

// GenerateMonthlyReport 生成月报
func (g *ReportGenerator) GenerateMonthlyReport(ctx context.Context, year, month int, apiKeyID string) ([]byte, error) {
	startTime := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.AddDate(0, 1, 0)

	filter := &UsageReportFilter{
		APIKeyID:  apiKeyID,
		StartTime: &startTime,
		EndTime:   &endTime,
		GroupBy:   "day",
	}

	return g.exporter.ExportAggregatedReport(ctx, filter)
}

// GenerateCustomReport 生成自定义报告
func (g *ReportGenerator) GenerateCustomReport(ctx context.Context, filter *UsageReportFilter) ([]byte, error) {
	if filter.GroupBy == "" {
		filter.GroupBy = "day"
	}

	return g.exporter.ExportAggregatedReport(ctx, filter)
}

// CSVWriter CSV 写入器
type CSVWriter struct {
	csvWriter *csv.Writer
}

// NewCSVWriter 创建 CSV 写入器
func NewCSVWriter(w io.Writer) *CSVWriter {
	return &CSVWriter{
		csvWriter: csv.NewWriter(w),
	}
}

// WriteReport 写入报告
func (w *CSVWriter) WriteReport(report interface{}) error {
	// 使用反射写入报告
	// 这里简化实现，实际应该更灵活
	return nil
}

// Flush 刷新缓冲区
func (w *CSVWriter) Flush() {
	w.csvWriter.Flush()
}

// ExportConfig 导出配置
type ExportConfig struct {
	IncludeHeaders bool   `json:"include_headers"`
	Delimiter       string `json:"delimiter"` // comma, tab, semicolon
	Encoding        string `json:"encoding"`
	Compression     string `json:"compression"` // none, gzip
}

// DefaultExportConfig 默认导出配置
func DefaultExportConfig() *ExportConfig {
	return &ExportConfig{
		IncludeHeaders: true,
		Delimiter:       ",",
		Encoding:        "utf-8",
		Compression:     "none",
	}
}
