package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// traceIDKey 是 context 中 trace ID 的 key
type traceIDKey struct{}

// TraceIDHeader 响应头中 trace ID 的 key
const TraceIDHeader = "X-Trace-Id"

// GenerateTraceID 生成新的 trace ID
func GenerateTraceID() string {
	return uuid.New().String()
}

// GetTraceIDFromContext 从 context 获取 trace ID
func GetTraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey{}).(string); ok {
		return traceID
	}
	return ""
}

// ContextWithTraceID 创建包含 trace ID 的 context
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// TraceIDMiddleware Trace ID 中间件
// 为每个请求生成唯一 trace ID，注入 context 并通过响应头返回
func TraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查是否已有 trace ID（从上游传递）
		traceID := r.Header.Get(TraceIDHeader)
		if traceID == "" {
			// 生成新的 trace ID
			traceID = GenerateTraceID()
		}

		// 注入 context
		ctx := ContextWithTraceID(r.Context(), traceID)

		// 使用响应头包装器，确保 trace ID 被写入
		wrapped := &traceIDResponseWriter{
			ResponseWriter: w,
			traceID:        traceID,
			written:        false,
		}

		// 调用下一个处理器
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// 确保在响应结束时写入 trace ID（如果还未写入）
		wrapped.ensureTraceID()
	})
}

// traceIDResponseWriter 响应写入器包装器
type traceIDResponseWriter struct {
	http.ResponseWriter
	traceID string
	written bool
}

// WriteHeader 实现 http.ResponseWriter 接口
func (w *traceIDResponseWriter) WriteHeader(statusCode int) {
	w.ensureTraceID()
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write 实现 http.ResponseWriter 接口
func (w *traceIDResponseWriter) Write(b []byte) (int, error) {
	w.ensureTraceID()
	return w.ResponseWriter.Write(b)
}

// ensureTraceID 确保 trace ID 被写入响应头
func (w *traceIDResponseWriter) ensureTraceID() {
	if !w.written {
		w.Header().Set(TraceIDHeader, w.traceID)
		w.written = true
	}
}

// Flush 实现 http.Flusher 接口（用于流式响应）
func (w *traceIDResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack 实现 http.Hijacker 接口（用于 WebSocket）
func (w *traceIDResponseWriter) Hijack() (c interface{}, rw interface{}, err error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("response writer does not support hijacking")
}

// WithTraceID 为已有请求添加 trace ID 的辅助函数
// 用于在中间件之外的地方手动设置 trace ID
func WithTraceID(r *http.Request, traceID string) *http.Request {
	ctx := ContextWithTraceID(r.Context(), traceID)
	return r.WithContext(ctx)
}

// MustGetTraceID 从 context 获取 trace ID，不存在时返回空字符串
// 比 GetTraceIDFromContext 更简洁的调用方式
func MustGetTraceID(ctx context.Context) string {
	return GetTraceIDFromContext(ctx)
}
