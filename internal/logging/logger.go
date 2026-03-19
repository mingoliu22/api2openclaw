package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// LogFormat 日志格式
type LogFormat string

const (
	FormatJSON  LogFormat = "json"
	FormatConsole LogFormat = "console"
)

// Logger 结构化日志器
type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	format   LogFormat
	level    LogLevel
	minLevel int
}

// NewLogger 创建日志器
func NewLogger(out io.Writer, format LogFormat, level LogLevel) *Logger {
	l := &Logger{
		out:    out,
		format: format,
		level:  level,
	}
	l.minLevel = levelToInt(level)
	return l
}

// NewDefault 创建默认日志器（输出到 stdout，JSON 格式）
func NewDefault() *Logger {
	return NewLogger(os.Stdout, FormatJSON, LevelInfo)
}

// levelToInt 将日志级别转换为数值
func levelToInt(level LogLevel) int {
	switch level {
	case LevelDebug:
		return 0
	case LevelInfo:
		return 1
	case LevelWarn:
		return 2
	case LevelError:
		return 3
	default:
		return 1
	}
}

// shouldLog 检查是否应该记录此级别的日志
func (l *Logger) shouldLog(level LogLevel) bool {
	return levelToInt(level) >= l.minLevel
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	TraceID   string                 `json:"trace_id,omitempty"`
	Model     string                 `json:"model,omitempty"`
	KeyID     string                 `json:"key_id,omitempty"`
	LatencyMs int64                  `json:"latency_ms,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	// 调用信息
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Function string `json:"function,omitempty"`
}

// log 内部日志方法
func (l *Logger) log(ctx context.Context, level LogLevel, msg string, fields map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	// 获取调用信息
	_, file, line, ok := runtime.Caller(2)
	if ok {
		// 简化文件名
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				file = file[i+1:]
				break
			}
		}
	}

	entry := &LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     string(level),
		Message:   msg,
		Fields:    fields,
		File:      file,
		Line:      line,
	}

	// 从 context 提取 trace_id
	if traceID := getTraceIDFromContext(ctx); traceID != "" {
		entry.TraceID = traceID
	}

	// 从 fields 提取常用字段
	if v, ok := fields["model"]; ok {
		if s, ok := v.(string); ok {
			entry.Model = s
			delete(fields, "model")
		}
	}
	if v, ok := fields["key_id"]; ok {
		if s, ok := v.(string); ok {
			entry.KeyID = s
			delete(fields, "key_id")
		}
	}
	if v, ok := fields["latency_ms"]; ok {
		if i, ok := v.(int64); ok {
			entry.LatencyMs = i
			delete(fields, "latency_ms")
		}
	}
	if v, ok := fields["status"]; ok {
		if s, ok := v.(string); ok {
			entry.Status = s
			delete(fields, "status")
		}
	}
	if v, ok := fields["error"]; ok {
		if s, ok := v.(string); ok {
			entry.Error = s
			delete(fields, "error")
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	switch l.format {
	case FormatJSON:
		l.logJSON(entry)
	case FormatConsole:
		l.logConsole(entry)
	}
}

// logJSON 输出 JSON 格式日志
func (l *Logger) logJSON(entry *LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(l.out, "{\"error\":\"marshal log error\",\"message\":\"%s\"}\n", err.Error())
		return
	}
	l.out.Write(data)
	l.out.Write([]byte("\n"))
}

// logConsole 输出控制台格式日志
func (l *Logger) logConsole(entry *LogEntry) {
	timestamp := entry.Timestamp[:19] // 截取到秒
	color := levelColor(entry.Level)

	// 基础日志
	fmt.Fprintf(l.out, "%s [%s] %s%s\033[0m %s",
		timestamp,
		entry.Level,
		color,
		entry.Level,
		entry.Message,
	)

	// 附加信息
	if entry.TraceID != "" {
		fmt.Fprintf(l.out, " trace_id=%s", entry.TraceID)
	}
	if entry.Model != "" {
		fmt.Fprintf(l.out, " model=%s", entry.Model)
	}
	if entry.KeyID != "" {
		fmt.Fprintf(l.out, " key_id=%s", entry.KeyID)
	}
	if entry.LatencyMs > 0 {
		fmt.Fprintf(l.out, " latency_ms=%d", entry.LatencyMs)
	}
	if entry.Status != "" {
		fmt.Fprintf(l.out, " status=%s", entry.Status)
	}
	if entry.Error != "" {
		fmt.Fprintf(l.out, " error=%s", entry.Error)
	}

	// 额外字段
	for k, v := range entry.Fields {
		fmt.Fprintf(l.out, " %s=%v", k, v)
	}

	fmt.Fprintln(l.out)
}

// levelColor 获取日志级别对应的颜色
func levelColor(level string) string {
	switch level {
	case "DEBUG":
		return "\033[36m" // 青色
	case "INFO":
		return "\033[32m" // 绿色
	case "WARN":
		return "\033[33m" // 黄色
	case "ERROR":
		return "\033[31m" // 红色
	default:
		return "\033[0m"
	}
}

// Debug 记录 DEBUG 级别日志
func (l *Logger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log(ctx, LevelDebug, msg, fields)
}

// Info 记录 INFO 级别日志
func (l *Logger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log(ctx, LevelInfo, msg, fields)
}

// Warn 记录 WARN 级别日志
func (l *Logger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log(ctx, LevelWarn, msg, fields)
}

// Error 记录 ERROR 级别日志
func (l *Logger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	l.log(ctx, LevelError, msg, fields)
}

// WithFields 创建带字段的日志上下文
func (l *Logger) WithFields(fields map[string]interface{}) *FieldLogger {
	return &FieldLogger{
		logger:  l,
		fields:  fields,
		context: context.Background(),
	}
}

// WithContext 创建带 context 的日志器
func (l *Logger) WithContext(ctx context.Context) *FieldLogger {
	return &FieldLogger{
		logger:  l,
		fields:  make(map[string]interface{}),
		context: ctx,
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
	l.minLevel = levelToInt(level)
}

// SetFormat 设置日志格式
func (l *Logger) SetFormat(format LogFormat) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.format = format
}

// === FieldLogger 带字段的日志器 ===

// FieldLogger 带预设字段的日志器
type FieldLogger struct {
	logger  *Logger
	fields  map[string]interface{}
	context context.Context
}

// Debug 记录 DEBUG 级别日志
func (fl *FieldLogger) Debug(msg string, fields map[string]interface{}) {
	merged := fl.mergeFields(fields)
	fl.logger.Debug(fl.context, msg, merged)
}

// Info 记录 INFO 级别日志
func (fl *FieldLogger) Info(msg string, fields map[string]interface{}) {
	merged := fl.mergeFields(fields)
	fl.logger.Info(fl.context, msg, merged)
}

// Warn 记录 WARN 级别日志
func (fl *FieldLogger) Warn(msg string, fields map[string]interface{}) {
	merged := fl.mergeFields(fields)
	fl.logger.Warn(fl.context, msg, merged)
}

// Error 记录 ERROR 级别日志
func (fl *FieldLogger) Error(msg string, fields map[string]interface{}) {
	merged := fl.mergeFields(fields)
	fl.logger.Error(fl.context, msg, merged)
}

// WithFields 添加更多字段
func (fl *FieldLogger) WithFields(fields map[string]interface{})) *FieldLogger {
	merged := fl.mergeFields(fields)
	return &FieldLogger{
		logger:  fl.logger,
		fields:  merged,
		context: fl.context,
	}
}

// WithContext 设置 context
func (fl *FieldLogger) WithContext(ctx context.Context) *FieldLogger {
	return &FieldLogger{
		logger:  fl.logger,
		fields:  fl.fields,
		context: ctx,
	}
}

// mergeFields 合并字段
func (fl *FieldLogger) mergeFields(fields map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range fl.fields {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	return merged
}

// === 全局日志器 ===

var defaultLogger = NewDefault()

// SetDefaultLogger 设置默认日志器
func SetDefaultLogger(l *Logger) {
	defaultLogger = l
}

// Debug 记录 DEBUG 日志
func Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	defaultLogger.Debug(ctx, msg, fields)
}

// Info 记录 INFO 日志
func Info(ctx context.Context, msg string, fields map[string]interface{}) {
	defaultLogger.Info(ctx, msg, fields)
}

// Warn 记录 WARN 日志
func Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	defaultLogger.Warn(ctx, msg, fields)
}

// Error 记录 ERROR 日志
func Error(ctx context.Context, msg string, fields map[string]interface{}) {
	defaultLogger.Error(ctx, msg, fields)
}

// WithFields 创建带字段的日志器
func WithFields(fields map[string]interface{}) *FieldLogger {
	return defaultLogger.WithFields(fields)
}

// WithContext 创建带 context 的日志器
func WithContext(ctx context.Context) *FieldLogger {
	return defaultLogger.WithContext(ctx)
}
